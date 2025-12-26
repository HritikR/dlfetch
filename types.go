package dlfetch

type DownloadRequest struct {
	ID       int
	URL      string
	FileName string
	Path     string // Path will be optional; if empty, use only FileName and targetDir
	MimeType string
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
	ID         int            `json:"id"`
	FileName   string         `json:"fileName"`
	TotalBytes int64          `json:"totalBytes"`
	DoneBytes  int64          `json:"doneBytes"`
	Status     DownloadStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
}

type TaskStatusCount struct {
	Total      int `json:"total,omitempty"`
	Pending    int `json:"pending,omitempty"`
	InProgress int `json:"in_progress,omitempty"`
	Completed  int `json:"completed,omitempty"`
	Failed     int `json:"failed,omitempty"`
}

type MonitorSnapshot struct {
	Tasks []DownloadTask  `json:"tasks"`
	Count TaskStatusCount `json:"count"`
}
