package tools

import (
	"sync"
	"time"
)

// ToolStats tracks usage statistics for a tool
type ToolStats struct {
	Name          string        `json:"name"`
	CallCount     int64         `json:"call_count"`
	SuccessCount  int64         `json:"success_count"`
	ErrorCount    int64         `json:"error_count"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	LastCalled    time.Time     `json:"last_called"`
	IsRunning     bool          `json:"is_running"`
	CurrentCalls  int32         `json:"current_calls"`
}

// ToolStatsTracker manages statistics for all tools
type ToolStatsTracker struct {
	stats map[string]*ToolStats
	mu    sync.RWMutex
}

// NewToolStatsTracker creates a new stats tracker
func NewToolStatsTracker() *ToolStatsTracker {
	return &ToolStatsTracker{
		stats: make(map[string]*ToolStats),
	}
}

// GetStats returns statistics for a specific tool
func (t *ToolStatsTracker) GetStats(toolName string) *ToolStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats, exists := t.stats[toolName]
	if !exists {
		return &ToolStats{
			Name:          toolName,
			CallCount:     0,
			SuccessCount:  0,
			ErrorCount:    0,
			TotalDuration: 0,
			AvgDuration:   0,
			LastCalled:    time.Time{},
			IsRunning:     false,
			CurrentCalls:  0,
		}
	}

	// Return a copy to prevent external modification
	statsCopy := *stats
	return &statsCopy
}

// GetAllStats returns statistics for all tools
func (t *ToolStatsTracker) GetAllStats() map[string]*ToolStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*ToolStats, len(t.stats))
	for name, stats := range t.stats {
		statsCopy := *stats
		result[name] = &statsCopy
	}
	return result
}

// RecordStart records the start of a tool execution
func (t *ToolStatsTracker) RecordStart(toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	stats, exists := t.stats[toolName]
	if !exists {
		stats = &ToolStats{
			Name:          toolName,
			CallCount:     0,
			SuccessCount:  0,
			ErrorCount:    0,
			TotalDuration: 0,
			AvgDuration:   0,
			LastCalled:    time.Time{},
			IsRunning:     false,
			CurrentCalls:  0,
		}
		t.stats[toolName] = stats
	}

	stats.CallCount++
	stats.LastCalled = time.Now()
	stats.CurrentCalls++
	stats.IsRunning = stats.CurrentCalls > 0
}

// RecordEnd records the completion of a tool execution
func (t *ToolStatsTracker) RecordEnd(toolName string, duration time.Duration, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	stats, exists := t.stats[toolName]
	if !exists {
		return // Should not happen, but be defensive
	}

	stats.CurrentCalls--
	stats.IsRunning = stats.CurrentCalls > 0
	stats.TotalDuration += duration

	if success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}

	// Calculate average duration
	if stats.CallCount > 0 {
		stats.AvgDuration = stats.TotalDuration / time.Duration(stats.CallCount)
	}
}

// Reset resets statistics for a specific tool
func (t *ToolStatsTracker) Reset(toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.stats, toolName)
}

// ResetAll resets all statistics
func (t *ToolStatsTracker) ResetAll() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats = make(map[string]*ToolStats)
}