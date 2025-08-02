package tools

import (
	"sync"
	"time"
)

// ProgressManager coordinates real-time progress tracking for tool executions
type ProgressManager struct {
	trackers    map[string]*ExecutionTracker
	subscribers map[string][]ProgressSubscriber
	mu          sync.RWMutex
	tickerDone  chan struct{}
	updateRate  time.Duration
	isRunning   bool
}

// ExecutionTracker tracks progress for a single execution context
type ExecutionTracker struct {
	ID            string
	StartTime     time.Time
	LastUpdate    time.Time
	Status        ExecutionStatus
	TotalTools    int
	CompletedTools int
	ActiveTools   map[string]ToolProgress
	CompletedResults map[string]ToolResult
	Errors        map[string]error
	mu            sync.RWMutex
}

// ToolProgress represents the progress of a single tool execution
type ToolProgress struct {
	ToolName    string
	StartTime   time.Time
	Status      ToolExecutionStatus
	Message     string
	Progress    float64 // 0.0 to 1.0
}

// ExecutionStatus represents the overall status of an execution
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusRunning    ExecutionStatus = "running"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
)

// ToolExecutionStatus represents the status of a single tool
type ToolExecutionStatus string

const (
	ToolStatusQueued    ToolExecutionStatus = "queued"
	ToolStatusRunning   ToolExecutionStatus = "running"
	ToolStatusCompleted ToolExecutionStatus = "completed"
	ToolStatusFailed    ToolExecutionStatus = "failed"
)

// ProgressUpdate contains progress information sent to subscribers
type ProgressUpdate struct {
	TrackerID     string
	Timestamp     time.Time
	OverallStatus ExecutionStatus
	Progress      ExecutionProgressInfo
	ToolUpdates   []ToolProgressUpdate
}

// ExecutionProgressInfo provides high-level execution progress
type ExecutionProgressInfo struct {
	TotalTools      int
	CompletedTools  int
	ActiveTools     int
	FailedTools     int
	Progress        float64
	ElapsedTime     time.Duration
	EstimatedTimeRemaining time.Duration
}

// ToolProgressUpdate represents an update for a specific tool
type ToolProgressUpdate struct {
	ToolID      string
	ToolName    string
	Status      ToolExecutionStatus
	Progress    float64
	Message     string
	StartTime   time.Time
	ElapsedTime time.Duration
	Result      *ToolResult
	Error       error
}

// ProgressSubscriber receives progress updates
type ProgressSubscriber func(update ProgressUpdate)

// NewProgressManager creates a new progress manager
func NewProgressManager(updateRate time.Duration) *ProgressManager {
	if updateRate <= 0 {
		updateRate = 100 * time.Millisecond // Default 10 updates per second
	}

	return &ProgressManager{
		trackers:    make(map[string]*ExecutionTracker),
		subscribers: make(map[string][]ProgressSubscriber),
		updateRate:  updateRate,
		tickerDone:  make(chan struct{}),
	}
}

// Start begins the progress manager's update loop
func (pm *ProgressManager) Start() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isRunning {
		return
	}

	pm.isRunning = true
	go pm.updateLoop()
}

// Stop stops the progress manager
func (pm *ProgressManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isRunning {
		return
	}

	pm.isRunning = false
	close(pm.tickerDone)
}

// CreateTracker creates a new execution tracker
func (pm *ProgressManager) CreateTracker(id string, totalTools int) *ExecutionTracker {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	status := StatusPending
	if totalTools == 0 {
		status = StatusCompleted
	}

	tracker := &ExecutionTracker{
		ID:               id,
		StartTime:        time.Now(),
		LastUpdate:       time.Now(),
		Status:           status,
		TotalTools:       totalTools,
		CompletedTools:   0,
		ActiveTools:      make(map[string]ToolProgress),
		CompletedResults: make(map[string]ToolResult),
		Errors:           make(map[string]error),
	}

	pm.trackers[id] = tracker
	return tracker
}

// GetTracker retrieves a tracker by ID
func (pm *ProgressManager) GetTracker(id string) (*ExecutionTracker, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	tracker, exists := pm.trackers[id]
	return tracker, exists
}

// RemoveTracker removes a tracker
func (pm *ProgressManager) RemoveTracker(id string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.trackers, id)
	delete(pm.subscribers, id)
}

// Subscribe adds a progress subscriber for a specific tracker
func (pm *ProgressManager) Subscribe(trackerID string, subscriber ProgressSubscriber) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.subscribers[trackerID] == nil {
		pm.subscribers[trackerID] = make([]ProgressSubscriber, 0)
	}
	pm.subscribers[trackerID] = append(pm.subscribers[trackerID], subscriber)
}

// SubscribeAll adds a subscriber for all trackers (use "*" as trackerID)
func (pm *ProgressManager) SubscribeAll(subscriber ProgressSubscriber) {
	pm.Subscribe("*", subscriber)
}

// UpdateToolStatus updates the status of a specific tool
func (pm *ProgressManager) UpdateToolStatus(trackerID, toolID, toolName string, status ToolExecutionStatus, progress float64, message string) {
	pm.mu.RLock()
	tracker, exists := pm.trackers[trackerID]
	pm.mu.RUnlock()

	if !exists {
		return
	}

	tracker.UpdateToolProgress(toolID, toolName, status, progress, message)
}

// CompleteToolExecution marks a tool as completed with its result
func (pm *ProgressManager) CompleteToolExecution(trackerID, toolID string, result ToolResult, err error) {
	pm.mu.RLock()
	tracker, exists := pm.trackers[trackerID]
	pm.mu.RUnlock()

	if !exists {
		return
	}

	tracker.CompleteToolExecution(toolID, result, err)
}

// updateLoop runs the periodic update process
func (pm *ProgressManager) updateLoop() {
	ticker := time.NewTicker(pm.updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.broadcastUpdates()
		case <-pm.tickerDone:
			return
		}
	}
}

// broadcastUpdates sends progress updates to all subscribers
func (pm *ProgressManager) broadcastUpdates() {
	pm.mu.RLock()
	trackers := make(map[string]*ExecutionTracker)
	subscribers := make(map[string][]ProgressSubscriber)

	// Copy maps to avoid holding lock during callbacks
	for k, v := range pm.trackers {
		trackers[k] = v
	}
	for k, v := range pm.subscribers {
		subscribers[k] = v
	}
	pm.mu.RUnlock()

	// Generate and send updates
	for trackerID, tracker := range trackers {
		update := pm.generateUpdate(tracker)
		
		// Send to specific subscribers
		if subs, exists := subscribers[trackerID]; exists {
			for _, subscriber := range subs {
				go subscriber(update)
			}
		}

		// Send to global subscribers
		if globalSubs, exists := subscribers["*"]; exists {
			for _, subscriber := range globalSubs {
				go subscriber(update)
			}
		}
	}
}

// generateUpdate creates a progress update for a tracker
func (pm *ProgressManager) generateUpdate(tracker *ExecutionTracker) ProgressUpdate {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	now := time.Now()
	elapsedTime := now.Sub(tracker.StartTime)
	
	// Calculate progress
	progress := float64(tracker.CompletedTools) / float64(tracker.TotalTools)
	if tracker.TotalTools == 0 {
		progress = 1.0
	}

	// Estimate remaining time
	var estimatedRemaining time.Duration
	if progress > 0 && progress < 1.0 {
		totalEstimated := time.Duration(float64(elapsedTime) / progress)
		estimatedRemaining = totalEstimated - elapsedTime
	}

	// Count failed tools
	failedTools := len(tracker.Errors)

	// Build tool updates
	toolUpdates := make([]ToolProgressUpdate, 0, len(tracker.ActiveTools)+len(tracker.CompletedResults))
	
	// Add active tools
	for toolID, toolProgress := range tracker.ActiveTools {
		toolUpdates = append(toolUpdates, ToolProgressUpdate{
			ToolID:      toolID,
			ToolName:    toolProgress.ToolName,
			Status:      toolProgress.Status,
			Progress:    toolProgress.Progress,
			Message:     toolProgress.Message,
			StartTime:   toolProgress.StartTime,
			ElapsedTime: now.Sub(toolProgress.StartTime),
		})
	}

	// Add completed tools
	for toolID, result := range tracker.CompletedResults {
		status := ToolStatusCompleted
		var err error
		if toolErr, hasError := tracker.Errors[toolID]; hasError {
			status = ToolStatusFailed
			err = toolErr
		}

		toolUpdates = append(toolUpdates, ToolProgressUpdate{
			ToolID:      toolID,
			ToolName:    result.Metadata.ToolName,
			Status:      status,
			Progress:    1.0,
			StartTime:   result.Metadata.StartTime,
			ElapsedTime: result.Metadata.ExecutionTime,
			Result:      &result,
			Error:       err,
		})
	}

	return ProgressUpdate{
		TrackerID: tracker.ID,
		Timestamp: now,
		OverallStatus: tracker.Status,
		Progress: ExecutionProgressInfo{
			TotalTools:             tracker.TotalTools,
			CompletedTools:         tracker.CompletedTools,
			ActiveTools:            len(tracker.ActiveTools),
			FailedTools:            failedTools,
			Progress:               progress,
			ElapsedTime:            elapsedTime,
			EstimatedTimeRemaining: estimatedRemaining,
		},
		ToolUpdates: toolUpdates,
	}
}

// UpdateToolProgress updates the progress of a specific tool
func (et *ExecutionTracker) UpdateToolProgress(toolID, toolName string, status ToolExecutionStatus, progress float64, message string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.LastUpdate = time.Now()

	// Update or create tool progress
	toolProgress, exists := et.ActiveTools[toolID]
	if !exists {
		toolProgress = ToolProgress{
			ToolName:  toolName,
			StartTime: time.Now(),
		}
	}

	toolProgress.Status = status
	toolProgress.Progress = progress
	toolProgress.Message = message
	et.ActiveTools[toolID] = toolProgress

	// Update overall status
	if et.Status == StatusPending && len(et.ActiveTools) > 0 {
		et.Status = StatusRunning
	}
}

// CompleteToolExecution marks a tool execution as completed
func (et *ExecutionTracker) CompleteToolExecution(toolID string, result ToolResult, err error) {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.LastUpdate = time.Now()

	// Move from active to completed
	delete(et.ActiveTools, toolID)
	et.CompletedResults[toolID] = result
	et.CompletedTools++

	if err != nil {
		et.Errors[toolID] = err
	}

	// Update overall status
	if et.CompletedTools >= et.TotalTools {
		if len(et.Errors) > 0 {
			et.Status = StatusFailed
		} else {
			et.Status = StatusCompleted
		}
	}
}

// Cancel marks the execution as cancelled
func (et *ExecutionTracker) Cancel() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.Status = StatusCancelled
	et.LastUpdate = time.Now()
}

// GetProgressSnapshot returns a snapshot of current progress
func (et *ExecutionTracker) GetProgressSnapshot() ExecutionProgressInfo {
	et.mu.RLock()
	defer et.mu.RUnlock()

	now := time.Now()
	elapsedTime := now.Sub(et.StartTime)
	progress := float64(et.CompletedTools) / float64(et.TotalTools)
	if et.TotalTools == 0 {
		progress = 1.0
	}

	var estimatedRemaining time.Duration
	if progress > 0 && progress < 1.0 {
		totalEstimated := time.Duration(float64(elapsedTime) / progress)
		estimatedRemaining = totalEstimated - elapsedTime
	}

	return ExecutionProgressInfo{
		TotalTools:             et.TotalTools,
		CompletedTools:         et.CompletedTools,
		ActiveTools:            len(et.ActiveTools),
		FailedTools:            len(et.Errors),
		Progress:               progress,
		ElapsedTime:            elapsedTime,
		EstimatedTimeRemaining: estimatedRemaining,
	}
}

// IsComplete returns true if the execution is complete (success or failure)
func (et *ExecutionTracker) IsComplete() bool {
	et.mu.RLock()
	defer et.mu.RUnlock()

	// If no tools expected, consider it complete from the start
	if et.TotalTools == 0 {
		return true
	}

	return et.Status == StatusCompleted || et.Status == StatusFailed || et.Status == StatusCancelled
}

// GetStats returns statistics about the progress manager
func (pm *ProgressManager) GetStats() ProgressManagerStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := ProgressManagerStats{
		ActiveTrackers:    len(pm.trackers),
		TotalSubscribers:  0,
		IsRunning:         pm.isRunning,
		UpdateRate:        pm.updateRate,
	}

	for _, subs := range pm.subscribers {
		stats.TotalSubscribers += len(subs)
	}

	return stats
}

// ProgressManagerStats provides statistics about the progress manager
type ProgressManagerStats struct {
	ActiveTrackers   int           `json:"active_trackers"`
	TotalSubscribers int           `json:"total_subscribers"`
	IsRunning        bool          `json:"is_running"`
	UpdateRate       time.Duration `json:"update_rate"`
}