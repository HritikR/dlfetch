package dlfetch

type DownloadRequest struct {
	ID       int
	URL      string
	FileName string
	Path     string // Path will be optional; if empty, use only FileName and targetDir
}

type DownloadResult struct {
	ID       int
	FileName string
	Path     string
	MimeType string
}
