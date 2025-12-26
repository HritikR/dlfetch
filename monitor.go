package dlfetch

import "sync"

type Monitor struct {
	mu    sync.RWMutex
	tasks map[int]*DownloadTask
}

// Creates a Monitor
func NewMonitor() *Monitor {
	return &Monitor{
		tasks: make(map[int]*DownloadTask),
	}
}

// Add downloadRequest to track its progress
func (m *Monitor) add(req DownloadRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[req.ID] = &DownloadTask{
		ID:       req.ID,
		FileName: req.FileName,
		Status:   StatusPending,
	}
}

// Update the progress and status of a download task
func (m *Monitor) update(id int, done int64, total int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.DoneBytes = done
		t.TotalBytes = total
		t.Status = StatusInProgress
	}
}

// Mark task as completed
func (m *Monitor) makeAsCompleted(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.DoneBytes = t.TotalBytes
		t.Status = StatusCompleted
	}
}

// Mark task as failed
func (m *Monitor) markAsFailed(id int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusFailed
		t.Error = err.Error()
	}
}
