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

// Download Monitoring
type DownloadStatus string

const (
	StatusPending    DownloadStatus = "pending"
	StatusInProgress DownloadStatus = "in_progress"
	StatusCompleted  DownloadStatus = "completed"
	StatusFailed     DownloadStatus = "failed"
)

type DownloadTask struct {
	ID         int
	FileName   string
	TotalBytes int64
	DoneBytes  int64
	Status     DownloadStatus
	Error      string
}
