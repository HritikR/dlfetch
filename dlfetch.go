package dlfetch

import "net/http"

type Fetcher struct {
	requestClient *http.Client // HTTP client to make requests
	maxWorkers    int          // Maximum number of concurrent workers
	targetDir     string       // Directory to save downloaded files
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

// New creates a new Fetcher instance with the provided options.
func New(options ...FetcherOption) *Fetcher {
	// Default values
	fetcher := &Fetcher{
		requestClient: http.DefaultClient,
		maxWorkers:    4,
		targetDir:     "./downloads",
	}

	// Apply provided options
	for _, option := range options {
		option(fetcher)
	}

	return fetcher
}
