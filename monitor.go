package dlfetch

import (
	"sync"
)

type Monitor interface {
	add(DownloadRequest)
	update(id int, done, total int64)
	markAsCompleted(id int)
	markAsFailed(id int, err error)
	GetSnapshot() MonitorSnapshot
}

type TaskMonitor struct {
	mu    sync.RWMutex
	tasks map[int]*DownloadTask
}

// Creates a TaskMonitor
func NewMonitor() *TaskMonitor {
	return &TaskMonitor{
		tasks: make(map[int]*DownloadTask),
	}
}

// Add downloadRequest to track its progress
func (m *TaskMonitor) add(req DownloadRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[req.ID] = &DownloadTask{
		ID:       req.ID,
		FileName: req.FileName,
		Status:   StatusPending,
	}
}

// Update the progress and status of a download task
func (m *TaskMonitor) update(id int, done int64, total int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.DoneBytes = done
		t.TotalBytes = total
		t.Status = StatusInProgress
	}
}

// Mark task as completed
func (m *TaskMonitor) markAsCompleted(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.DoneBytes = t.TotalBytes
		t.Status = StatusCompleted
	}
}

// Mark task as failed
func (m *TaskMonitor) markAsFailed(id int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusFailed
		t.Error = err.Error()
	}
}

// GetSnapshot returns a copy of the current state of all download
func (m *TaskMonitor) GetSnapshot() MonitorSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MonitorSnapshot{}
	for _, t := range m.tasks {
		snapshot.Tasks = append(snapshot.Tasks, *t)
		snapshot.Count.Total++
		switch t.Status {
		case StatusCompleted:
			snapshot.Count.Completed++
		case StatusFailed:
			snapshot.Count.Failed++
		case StatusInProgress:
			snapshot.Count.InProgress++
		case StatusPending:
			snapshot.Count.Pending++
		}
	}
	return snapshot
}

// Monitor Writer
// This is a custom writer that reports progress to the monitor
type monitorWriter struct {
	id      int
	total   int64
	written int64
	monitor Monitor
}

func (mw *monitorWriter) Write(p []byte) (int, error) {
	n := len(p)
	mw.written += int64(n)
	mw.monitor.update(mw.id, mw.written, mw.total)
	return n, nil
}

// No-Op Monitor
// This is default monitor that does nothing

type noopMonitor struct{}

func (n *noopMonitor) add(DownloadRequest)          {}
func (n *noopMonitor) update(int, int64, int64)     {}
func (n *noopMonitor) markAsCompleted(int)          {}
func (n *noopMonitor) markAsFailed(int, error)      {}
func (n *noopMonitor) GetSnapshot() MonitorSnapshot { return MonitorSnapshot{} }
