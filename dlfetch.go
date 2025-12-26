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

// New creates a new Fetcher instance with the provided options.
func New(options ...FetcherOption) *Fetcher {
	// Default values
	fetcher := &Fetcher{
		requestClient: http.DefaultClient,
		maxWorkers:    defaultWorkers,
		targetDir:     defaultTargetDir,
		queue:         make(chan DownloadRequest, defaultQueueSize),
		stopChan:      make(chan struct{}),
	}

	// Apply provided options
	for _, option := range options {
		option(fetcher)
	}

	return fetcher
}

// Enqueue adds a download request to the Fetcher's queue.
func (f *Fetcher) Enqueue(request DownloadRequest) {
	f.queue <- request
}

// EnqueueMany adds multiple download requests to the Fetcher's queue.
func (f *Fetcher) EnqueueMany(requests []DownloadRequest) {
	for _, request := range requests {
		f.queue <- request
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
	// Ensure the request has a valid FileName
	EnsureFileName(&req)

	fullPath := filepath.Join(f.targetDir, req.Path, req.FileName)

	// Ensure directory exists
	err := EnsureDir(fullPath)
	if err != nil {
		return DownloadResult{}, err
	}

	// Perform the download
	resp, err := f.requestClient.Get(req.URL)
	if err != nil {
		return DownloadResult{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DownloadResult{}, fmt.Errorf("failed to download file: %s, status code: %d", req.URL, resp.StatusCode)
	}

	// Write to a tmp file first
	// To prevent incomplete files in case of failure
	tmpPath := fullPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return DownloadResult{}, err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return DownloadResult{}, err
	}

	if err := out.Close(); err != nil {
		return DownloadResult{}, err
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		return DownloadResult{}, err
	}

	return DownloadResult{
		ID:       req.ID,
		FileName: req.FileName,
		Path:     fullPath,
		MimeType: resp.Header.Get("Content-Type"),
	}, nil
}
