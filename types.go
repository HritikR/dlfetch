package dlfetch

import "time"

type DownloadRequest struct {
	ID       int
	URL      string
	FileName string
	Path     string // Path will be optional; if empty, use only FileName and targetDir
	MimeType string
	FullPath string // Computed after enqueuing
}

type EnqueueResult struct {
	Request DownloadRequest
	Queued  bool
	Error   error
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
	ID            int            `json:"id"`
	FileName      string         `json:"fileName"`
	TotalBytes    int64          `json:"totalBytes"`
	DoneBytes     int64          `json:"doneBytes"`
	Status        DownloadStatus `json:"status"`
	Error         string         `json:"error,omitempty"`
	StartTime     time.Time      `json:"startTime"`
	CompletedAt   *time.Time     `json:"completedAt,omitempty"`
	DownloadSpeed float64        `json:"downloadSpeed"`
	ETA           string         `json:"eta"`
	QueuePosition int            `json:"queuePosition"`
	EnqueuedAt    time.Time      `json:"enqueuedAt"`
}

type TaskStatusCount struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"inProgress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}

type MonitorSnapshot struct {
	Tasks []DownloadTask  `json:"tasks"`
	Count TaskStatusCount `json:"count"`
}

type pendingTask struct {
	id         int
	enqueuedAt time.Time
}

var pendingTasks []pendingTask
