package dlfetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Default configuration values
const (
	defaultTargetDir = "./downloads"
	defaultWorkers   = 4
	defaultQueueSize = 100
)

// Fetcher is responsible for managing download requests and processing them.
// It supports configuration through functional options.
type Fetcher struct {
	requestClient *http.Client                 // HTTP client to make requests
	maxWorkers    int                          // Maximum number of concurrent workers
	targetDir     string                       // Directory to save downloaded files
	queue         chan DownloadRequest         // Channel to queue download requests
	wg            sync.WaitGroup               // WaitGroup to manage goroutines
	stopChan      chan struct{}                // Channel to signal stopping of fetcher
	onComplete    func(DownloadResult)         // Callback function on download completion
	onError       func(DownloadRequest, error) // Callback function on error
	monitor       Monitor                      // Monitor to track download progress and status
}

// FetcherOption defines a function type for configuring the Fetcher.
// Each option function modifies the Fetcher's fields.
type FetcherOption func(*Fetcher)

// WithHTTPClient sets a custom HTTP client for the Fetcher.
func WithHTTPClient(client *http.Client) FetcherOption {
	return func(f *Fetcher) {
		f.requestClient = client
	}
}

// WithMaxWorkers sets the maximum number of concurrent workers for the Fetcher.
func WithMaxWorkers(max int) FetcherOption {
	return func(f *Fetcher) {
		f.maxWorkers = max
	}
}

// WithTargetDir sets the target directory for downloaded files.
func WithTargetDir(dir string) FetcherOption {
	return func(f *Fetcher) {
		f.targetDir = dir
	}
}

// WithOnComplete sets the callback function to be called on download completion.
func WithOnComplete(callback func(DownloadResult)) FetcherOption {
	return func(f *Fetcher) {
		f.onComplete = callback
	}
}

// WithOnError sets the callback function to be called on download error.
func WithOnError(callback func(DownloadRequest, error)) FetcherOption {
	return func(f *Fetcher) {
		f.onError = callback
	}
}

// WithMonitor sets the Monitor for the Fetcher.
func WithMonitor(m Monitor) FetcherOption {
	return func(f *Fetcher) {
		f.monitor = m
	}
}

// New creates a new Fetcher instance with the provided options.
func New(options ...FetcherOption) *Fetcher {
	// Default values
	fetcher := &Fetcher{
		requestClient: http.DefaultClient,
		maxWorkers:    defaultWorkers,
		targetDir:     defaultTargetDir,
		queue:         make(chan DownloadRequest, defaultQueueSize),
		stopChan:      make(chan struct{}),
		monitor:       &noopMonitor{},
	}

	// Apply provided options
	for _, option := range options {
		option(fetcher)
	}

	return fetcher
}

// Enqueue adds a download request to the Fetcher's queue.
func (f *Fetcher) Enqueue(req DownloadRequest) {
	if err := f.validateRequest(&req); err != nil {
		if f.onError != nil {
			f.onError(req, err)
		}
		return
	}

	f.monitor.add(req)
	f.queue <- req
}

// EnqueueMany adds multiple download requests to the Fetcher's queue.
func (f *Fetcher) EnqueueMany(reqs []DownloadRequest) {
	for _, req := range reqs {
		f.Enqueue(req)
	}
}

// Start begins processing download requests with the configured number of workers.
func (f *Fetcher) Start() {
	for i := 0; i < f.maxWorkers; i++ {
		f.wg.Add(1)
		go f.worker()
	}
}

// Stop signals the Fetcher to stop processing and waits for all workers to finish.
func (f *Fetcher) Stop() {
	close(f.stopChan)
	f.wg.Wait()
}

func (f *Fetcher) worker() {
	defer f.wg.Done()

	for {
		select {
		case req := <-f.queue:
			result, err := f.processDownload(req)
			if err != nil {
				if f.onError != nil {
					f.onError(req, err)
				}
				continue
			}
			if f.onComplete != nil {
				f.onComplete(result)
			}
		case <-f.stopChan:
			return
		}
	}
}

// processDownload handles the actual downloading of a file based on the DownloadRequest.
// It returns a DownloadResult or an error if the download fails.
func (f *Fetcher) processDownload(req DownloadRequest) (DownloadResult, error) {

	// Check if file already exists
	// To make sure another program / process has not created the file
	if checkFileExists(req.FullPath) {
		err := fmt.Errorf("file already exists: id=%d, name=%s, path=%s", req.ID, req.FileName, req.FullPath)
		f.monitor.markAsFailed(req.ID, err)
	}

	// Ensure directory exists
	err := ensureDir(req.FullPath)
	if err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	// Perform the download
	resp, err := f.requestClient.Get(req.URL)
	if err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to download file: %s, status code: %d", req.URL, resp.StatusCode)
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	// Write to a tmp file first
	// To prevent incomplete files in case of failure
	tmpPath := req.FullPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}
	defer out.Close()

	mw := &monitorWriter{
		id:      req.ID,
		total:   resp.ContentLength,
		monitor: f.monitor,
	}

	reader := io.TeeReader(resp.Body, mw)

	if _, err := io.Copy(out, reader); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	if err := out.Close(); err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	if err := os.Rename(tmpPath, req.FullPath); err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	f.monitor.markAsCompleted(req.ID)

	respContentType := resp.Header.Get("Content-Type")

	return DownloadResult{
		ID:       req.ID,
		FileName: req.FileName,
		Path:     req.FullPath,
		MimeType: determineMimeType(req, respContentType, req.FullPath),
	}, nil
}

// validateRequest checks if the file name is not nil or empty
// also checks if file already exists
func (f *Fetcher) validateRequest(req *DownloadRequest) error {
	ensureFileName(req)

	req.FullPath = filepath.Join(f.targetDir, req.Path, req.FileName)

	if checkFileExists(req.FullPath) {
		return fmt.Errorf("file already exists: %s", req.FullPath)
	}

	return nil
}
