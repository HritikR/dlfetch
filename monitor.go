package dlfetch

import (
	"math"
	"sort"
	"sync"
	"time"
)

type Monitor interface {
	add(DownloadRequest)
	update(id int, done, total int64, ds float64, eta string)
	close()
	markAsCompleted(id int)
	markAsFailed(id int, err error)
	GetSnapshot() MonitorSnapshot
	EventSignal() <-chan struct{}
}

type TaskMonitor struct {
	mu          sync.RWMutex
	tasks       map[int]*DownloadTask
	eventSignal chan struct{}
}

// Creates a TaskMonitor
func NewMonitor() *TaskMonitor {
	return &TaskMonitor{
		tasks:       make(map[int]*DownloadTask),
		eventSignal: make(chan struct{}, 1),
	}
}

// EventSignal returns a read-only channel that signals
// whenever the TaskMonitor's state changes.
func (m *TaskMonitor) EventSignal() <-chan struct{} {
	return m.eventSignal
}

// signalEvent sends a signal on the eventSignal channel
// to notify listeners that the TaskMonitor has changed.
// If the channel already has a pending signal, it does nothing
// to avoid blocking or sending duplicate notifications.
func (m *TaskMonitor) signalEvent() {
	select {
	case m.eventSignal <- struct{}{}:
	default:
	}
}

func (m *TaskMonitor) close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case <-m.eventSignal:
		// already closed or drained
	default:
		close(m.eventSignal)
	}
}

// Add downloadRequest to track its progress
func (m *TaskMonitor) add(req DownloadRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[req.ID] = &DownloadTask{
		ID:         req.ID,
		FileName:   req.FileName,
		FilePath:   req.FullPath,
		Status:     StatusPending,
		EnqueuedAt: time.Now(),
	}
	m.signalEvent()
}

// Update the progress and status of a download task
func (m *TaskMonitor) update(id int, done int64, total int64, ds float64, eta string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.DoneBytes = done
		t.TotalBytes = total
		t.Status = StatusInProgress
		t.DownloadSpeed = ds
		t.ETA = eta
	}
	m.signalEvent()
}

// Mark task as completed
func (m *TaskMonitor) markAsCompleted(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		if t.TotalBytes > 0 {
			t.DoneBytes = t.TotalBytes
		}
		t.Status = StatusCompleted
		now := time.Now()
		t.CompletedAt = &now
	}
	m.signalEvent()
}

// Mark task as failed
func (m *TaskMonitor) markAsFailed(id int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		t.Status = StatusFailed
		t.Error = err.Error()
	}
	m.signalEvent()
}

// GetSnapshot returns a copy of the current state of all download
func (m *TaskMonitor) GetSnapshot() MonitorSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MonitorSnapshot{}

	var pendingTasks []pendingTask

	for _, t := range m.tasks {
		snapshot.Tasks = append(snapshot.Tasks, *t)
		snapshot.Count.Total++
		switch t.Status {
		case StatusPending:
			snapshot.Count.Pending++
			pendingTasks = append(pendingTasks, pendingTask{
				id:         t.ID,
				enqueuedAt: t.EnqueuedAt,
			})
		case StatusCompleted:
			snapshot.Count.Completed++
		case StatusFailed:
			snapshot.Count.Failed++
		case StatusInProgress:
			snapshot.Count.InProgress++
		}
	}

	// Sort pending tasks by enqueue time (FIFO order)
	sort.Slice(pendingTasks, func(i, j int) bool {
		return pendingTasks[i].enqueuedAt.Before(pendingTasks[j].enqueuedAt)
	})

	queuePositions := make(map[int]int)
	for pos, pt := range pendingTasks {
		queuePositions[pt.id] = pos + 1
	}

	for i := range snapshot.Tasks {
		if snapshot.Tasks[i].Status == StatusPending {
			snapshot.Tasks[i].QueuePosition = queuePositions[snapshot.Tasks[i].ID]
		} else {
			snapshot.Tasks[i].QueuePosition = 0 // Not in pending state
		}
	}

	return snapshot
}

// Monitor Writer
// This is a custom writer that reports progress to the monitor
type monitorWriter struct {
	id        int
	total     int64
	written   int64
	monitor   Monitor
	startTime time.Time
}

func (mw *monitorWriter) Write(p []byte) (int, error) {
	n := len(p)
	mw.written += int64(n)

	// Set startTime, when we start the download
	if mw.startTime.IsZero() {
		mw.startTime = time.Now()
	}

	elapsed := time.Since(mw.startTime).Seconds()
	speedMBs := (float64(mw.written) / (1024 * 1024)) / elapsed
	speedMBs = math.Round(speedMBs*10) / 10

	var eta string
	if mw.total > 0 {

		remainingBytes := mw.total - mw.written
		if speedMBs > 0 {
			etaSec := float64(remainingBytes) / (speedMBs * 1024 * 1024)
			eta = time.Duration(etaSec * float64(time.Second)).Truncate(time.Second).String()
		} else {
			eta = "calculating..."
		}
	} else {
		eta = "unknown"
	}

	mw.monitor.update(mw.id, mw.written, mw.total, speedMBs, eta)
	return n, nil
}

// No-Op Monitor
// This is default monitor that does nothing

type noopMonitor struct{}

func (n *noopMonitor) add(DownloadRequest)                       {}
func (n *noopMonitor) update(int, int64, int64, float64, string) {}
func (n *noopMonitor) close()                                    {}
func (n *noopMonitor) markAsCompleted(int)                       {}
func (n *noopMonitor) markAsFailed(int, error)                   {}
func (n *noopMonitor) GetSnapshot() MonitorSnapshot              { return MonitorSnapshot{} }
func (n *noopMonitor) EventSignal() <-chan struct{}              { return nil }
