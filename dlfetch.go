package dlfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
	ctx             context.Context              // Context for cancellation and timeouts
	cancel          context.CancelFunc           // Cancel function to stop the fetcher
	requestClient   *http.Client                 // HTTP client to make requests
	maxWorkers      int                          // Maximum number of concurrent workers
	targetDir       string                       // Directory to save downloaded files
	queue           chan DownloadRequest         // Channel to queue download requests
	wg              sync.WaitGroup               // WaitGroup to manage goroutines
	onComplete      func(DownloadResult)         // Callback function on download completion
	onError         func(DownloadRequest, error) // Callback function on error
	monitor         Monitor                      // Monitor to track download progress and status
	enableOverwrite bool                         // Enable Overwriting when file is already available
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

// WithEnableOverwrite sets the enableOverwrite for the Fetcher
// When enabled the files gets overwritten if they exists
func WithEnableOverwrite(eo bool) FetcherOption {
	return func(f *Fetcher) {
		f.enableOverwrite = eo
	}
}

// WithContext sets the context for the Fetcher, allowing for cancellation and timeouts.
func WithContext(ctx context.Context) FetcherOption {
	return func(f *Fetcher) {
		f.ctx, f.cancel = context.WithCancel(ctx)
	}
}

// New creates a new Fetcher instance with the provided options.
func New(options ...FetcherOption) *Fetcher {
	ctx, cancel := context.WithCancel(context.Background())

	// Default values
	fetcher := &Fetcher{
		ctx:             ctx,
		cancel:          cancel,
		requestClient:   http.DefaultClient,
		maxWorkers:      defaultWorkers,
		targetDir:       defaultTargetDir,
		queue:           make(chan DownloadRequest, defaultQueueSize),
		monitor:         &noopMonitor{},
		enableOverwrite: false,
	}

	// Apply provided options
	for _, option := range options {
		option(fetcher)
	}

	return fetcher
}

// Enqueue adds a download request to the Fetcher's queue.
func (f *Fetcher) Enqueue(req DownloadRequest) EnqueueResult {
	if err := f.validateRequest(&req); err != nil {
		return EnqueueResult{Queued: false, Error: err}
	}

	f.monitor.add(req)
	f.queue <- req
	return EnqueueResult{Queued: true, Error: nil}
}

// EnqueueMany adds multiple download requests to the Fetcher's queue.
func (f *Fetcher) EnqueueMany(reqs []DownloadRequest) []EnqueueResult {
	results := make([]EnqueueResult, 0, len(reqs))

	for _, req := range reqs {
		result := f.Enqueue(req)
		results = append(results, result)
	}

	return results
}

// Start begins processing download requests with the configured number of workers.
func (f *Fetcher) Start() {
	for i := 0; i < f.maxWorkers; i++ {
		f.wg.Add(1)
		go f.worker()
	}
}

// Stop gracefully shuts down the Fetcher.
// It cancels the internal context to stop accepting new work
// and abort any in-flight download requests.
// Then it waits for all worker goroutines to exit
// and finally closes the monitor.
func (f *Fetcher) Stop() {
	f.cancel()
	f.wg.Wait()
	f.monitor.close()
}

func (f *Fetcher) worker() {
	defer f.wg.Done()

	for {
		select {
		case <-f.ctx.Done():
			return
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
		}
	}
}

// processDownload handles the actual downloading of a file based on the DownloadRequest.
// It returns a DownloadResult or an error if the download fails.
func (f *Fetcher) processDownload(req DownloadRequest) (DownloadResult, error) {

	// Check if file already exists
	// To make sure another program / process has not created the file
	if !f.enableOverwrite && checkFileExists(req.FullPath) {
		err := fmt.Errorf("file already exists: id=%d, name=%s, path=%s", req.ID, req.FileName, req.FullPath)
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	// Ensure directory exists
	err := ensureDir(req.FullPath)
	if err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	// Create HTTP request with context for cancellation
	httpReq, err := http.NewRequestWithContext(f.ctx, http.MethodGet, req.URL, nil)

	if err != nil {
		f.monitor.markAsFailed(req.ID, err)
		return DownloadResult{}, err
	}

	// Perform the download
	resp, err := f.requestClient.Do(httpReq)
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
		total:   resolveFileSize(resp),
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
