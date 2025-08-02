package tools

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressManager_NewProgressManager(t *testing.T) {
	tests := []struct {
		name       string
		updateRate time.Duration
		expected   time.Duration
	}{
		{"custom update rate", 50 * time.Millisecond, 50 * time.Millisecond},
		{"zero rate defaults", 0, 100 * time.Millisecond},
		{"negative rate defaults", -1 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewProgressManager(tt.updateRate)
			assert.Equal(t, tt.expected, pm.updateRate)
			assert.False(t, pm.isRunning)
			assert.NotNil(t, pm.trackers)
			assert.NotNil(t, pm.subscribers)
		})
	}
}

func TestProgressManager_StartStop(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)

	// Test starting
	pm.Start()
	assert.True(t, pm.isRunning)

	// Test starting again (should be idempotent)
	pm.Start()
	assert.True(t, pm.isRunning)

	// Test stopping
	pm.Stop()
	assert.False(t, pm.isRunning)

	// Test stopping again (should be idempotent)
	pm.Stop()
	assert.False(t, pm.isRunning)
}

func TestProgressManager_CreateTracker(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)

	tracker := pm.CreateTracker("test-tracker", 5)
	assert.NotNil(t, tracker)
	assert.Equal(t, "test-tracker", tracker.ID)
	assert.Equal(t, 5, tracker.TotalTools)
	assert.Equal(t, 0, tracker.CompletedTools)
	assert.Equal(t, StatusPending, tracker.Status)
	assert.NotNil(t, tracker.ActiveTools)
	assert.NotNil(t, tracker.CompletedResults)
	assert.NotNil(t, tracker.Errors)

	// Verify tracker was added to manager
	retrieved, exists := pm.GetTracker("test-tracker")
	assert.True(t, exists)
	assert.Equal(t, tracker, retrieved)
}

func TestProgressManager_RemoveTracker(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)

	// Create tracker
	pm.CreateTracker("test-tracker", 3)

	// Add subscriber
	var receivedUpdates []ProgressUpdate
	pm.Subscribe("test-tracker", func(update ProgressUpdate) {
		receivedUpdates = append(receivedUpdates, update)
	})

	// Remove tracker
	pm.RemoveTracker("test-tracker")

	// Verify removal
	_, exists := pm.GetTracker("test-tracker")
	assert.False(t, exists)
}

func TestProgressManager_Subscribe(t *testing.T) {
	pm := NewProgressManager(50 * time.Millisecond)
	pm.Start()
	defer pm.Stop()

	pm.CreateTracker("test-tracker", 2)

	var updates []ProgressUpdate
	var mu sync.Mutex

	// Subscribe to specific tracker
	pm.Subscribe("test-tracker", func(update ProgressUpdate) {
		mu.Lock()
		defer mu.Unlock()
		updates = append(updates, update)
	})

	// Update tool status to trigger updates
	pm.UpdateToolStatus("test-tracker", "tool1", "TestTool1", ToolStatusRunning, 0.5, "Processing")

	// Wait for updates
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, len(updates), 0)
	
	// Verify update content
	lastUpdate := updates[len(updates)-1]
	assert.Equal(t, "test-tracker", lastUpdate.TrackerID)
	assert.Equal(t, StatusRunning, lastUpdate.OverallStatus)
	assert.Len(t, lastUpdate.ToolUpdates, 1)
	assert.Equal(t, "tool1", lastUpdate.ToolUpdates[0].ToolID)
	assert.Equal(t, ToolStatusRunning, lastUpdate.ToolUpdates[0].Status)
}

func TestProgressManager_SubscribeAll(t *testing.T) {
	pm := NewProgressManager(20 * time.Millisecond)
	pm.Start()
	defer pm.Stop()

	// Create multiple trackers
	pm.CreateTracker("tracker1", 1)
	pm.CreateTracker("tracker2", 1)

	var allUpdates []ProgressUpdate
	var mu sync.Mutex

	// Subscribe to all trackers
	pm.SubscribeAll(func(update ProgressUpdate) {
		mu.Lock()
		defer mu.Unlock()
		allUpdates = append(allUpdates, update)
	})

	// Update both trackers
	pm.UpdateToolStatus("tracker1", "tool1", "Tool1", ToolStatusRunning, 0.5, "Working")
	pm.UpdateToolStatus("tracker2", "tool2", "Tool2", ToolStatusRunning, 0.3, "Processing")

	// Wait for updates
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, len(allUpdates), 0)

	// Verify we received updates from both trackers
	tracker1Updates := 0
	tracker2Updates := 0
	for _, update := range allUpdates {
		switch update.TrackerID {
		case "tracker1":
			tracker1Updates++
		case "tracker2":
			tracker2Updates++
		}
	}
	assert.Greater(t, tracker1Updates, 0)
	assert.Greater(t, tracker2Updates, 0)
}

func TestExecutionTracker_UpdateToolProgress(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 3)

	// Initial state
	assert.Equal(t, StatusPending, tracker.Status)
	assert.Len(t, tracker.ActiveTools, 0)

	// Add first tool
	tracker.UpdateToolProgress("tool1", "TestTool1", ToolStatusRunning, 0.5, "Processing data")

	assert.Equal(t, StatusRunning, tracker.Status)
	assert.Len(t, tracker.ActiveTools, 1)
	
	toolProgress := tracker.ActiveTools["tool1"]
	assert.Equal(t, "TestTool1", toolProgress.ToolName)
	assert.Equal(t, ToolStatusRunning, toolProgress.Status)
	assert.Equal(t, 0.5, toolProgress.Progress)
	assert.Equal(t, "Processing data", toolProgress.Message)

	// Update same tool
	tracker.UpdateToolProgress("tool1", "TestTool1", ToolStatusRunning, 0.8, "Almost done")
	
	assert.Len(t, tracker.ActiveTools, 1)
	toolProgress = tracker.ActiveTools["tool1"]
	assert.Equal(t, 0.8, toolProgress.Progress)
	assert.Equal(t, "Almost done", toolProgress.Message)
}

func TestExecutionTracker_CompleteToolExecution(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 2)

	// Start tool execution
	tracker.UpdateToolProgress("tool1", "TestTool1", ToolStatusRunning, 0.5, "Working")
	assert.Len(t, tracker.ActiveTools, 1)
	assert.Equal(t, 0, tracker.CompletedTools)

	// Complete successfully
	result := ToolResult{
		Success: true,
		Content: "Tool completed",
		Metadata: ToolMetadata{
			ToolName: "TestTool1",
			StartTime: time.Now().Add(-1 * time.Second),
			EndTime: time.Now(),
			ExecutionTime: 1 * time.Second,
		},
	}

	tracker.CompleteToolExecution("tool1", result, nil)

	assert.Len(t, tracker.ActiveTools, 0)
	assert.Equal(t, 1, tracker.CompletedTools)
	assert.Len(t, tracker.CompletedResults, 1)
	assert.Len(t, tracker.Errors, 0)
	assert.Equal(t, StatusRunning, tracker.Status) // Still running until all tools complete

	// Complete second tool with error
	tracker.UpdateToolProgress("tool2", "TestTool2", ToolStatusRunning, 1.0, "Finishing")
	errorMsg := "Tool failed"
	tracker.CompleteToolExecution("tool2", ToolResult{Success: false, Error: errorMsg}, assert.AnError)

	assert.Equal(t, 2, tracker.CompletedTools)
	assert.Len(t, tracker.Errors, 1)
	assert.Equal(t, StatusFailed, tracker.Status) // Failed because of error
}

func TestExecutionTracker_GetProgressSnapshot(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 4)

	// Initial snapshot
	snapshot := tracker.GetProgressSnapshot()
	assert.Equal(t, 4, snapshot.TotalTools)
	assert.Equal(t, 0, snapshot.CompletedTools)
	assert.Equal(t, 0, snapshot.ActiveTools)
	assert.Equal(t, 0, snapshot.FailedTools)
	assert.Equal(t, 0.0, snapshot.Progress)

	// Add active tools
	tracker.UpdateToolProgress("tool1", "Tool1", ToolStatusRunning, 0.5, "Working")
	tracker.UpdateToolProgress("tool2", "Tool2", ToolStatusRunning, 0.3, "Processing")

	snapshot = tracker.GetProgressSnapshot()
	assert.Equal(t, 2, snapshot.ActiveTools)
	assert.Equal(t, 0.0, snapshot.Progress) // No completed tools yet

	// Complete one tool
	result := ToolResult{Success: true, Content: "Done"}
	tracker.CompleteToolExecution("tool1", result, nil)

	snapshot = tracker.GetProgressSnapshot()
	assert.Equal(t, 1, snapshot.CompletedTools)
	assert.Equal(t, 1, snapshot.ActiveTools)
	assert.Equal(t, 0.25, snapshot.Progress) // 1 out of 4 completed
	assert.Greater(t, snapshot.ElapsedTime, time.Duration(0))
}

func TestExecutionTracker_Cancel(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 3)

	tracker.UpdateToolProgress("tool1", "Tool1", ToolStatusRunning, 0.5, "Working")
	assert.Equal(t, StatusRunning, tracker.Status)

	tracker.Cancel()
	assert.Equal(t, StatusCancelled, tracker.Status)
	assert.True(t, tracker.IsComplete())
}

func TestExecutionTracker_IsComplete(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 2)

	// Initially not complete
	assert.False(t, tracker.IsComplete())

	// Running - not complete
	tracker.UpdateToolProgress("tool1", "Tool1", ToolStatusRunning, 0.5, "Working")
	assert.False(t, tracker.IsComplete())

	// Complete all tools successfully
	result := ToolResult{Success: true}
	tracker.CompleteToolExecution("tool1", result, nil)
	tracker.CompleteToolExecution("tool2", result, nil)
	assert.True(t, tracker.IsComplete())
	assert.Equal(t, StatusCompleted, tracker.Status)
}

func TestProgressManager_UpdateToolStatus(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 1)

	pm.UpdateToolStatus("test-tracker", "tool1", "TestTool", ToolStatusRunning, 0.7, "Nearly done")

	assert.Len(t, tracker.ActiveTools, 1)
	toolProgress := tracker.ActiveTools["tool1"]
	assert.Equal(t, "TestTool", toolProgress.ToolName)
	assert.Equal(t, ToolStatusRunning, toolProgress.Status)
	assert.Equal(t, 0.7, toolProgress.Progress)
	assert.Equal(t, "Nearly done", toolProgress.Message)
}

func TestProgressManager_CompleteToolExecution(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 1)

	// Start tool
	pm.UpdateToolStatus("test-tracker", "tool1", "TestTool", ToolStatusRunning, 0.5, "Working")

	// Complete tool
	result := ToolResult{
		Success: true,
		Content: "Completed successfully",
		Metadata: ToolMetadata{ToolName: "TestTool"},
	}
	pm.CompleteToolExecution("test-tracker", "tool1", result, nil)

	assert.Len(t, tracker.ActiveTools, 0)
	assert.Equal(t, 1, tracker.CompletedTools)
	assert.Equal(t, StatusCompleted, tracker.Status)
}

func TestProgressManager_GetStats(t *testing.T) {
	pm := NewProgressManager(25 * time.Millisecond)

	// Initial stats
	stats := pm.GetStats()
	assert.Equal(t, 0, stats.ActiveTrackers)
	assert.Equal(t, 0, stats.TotalSubscribers)
	assert.False(t, stats.IsRunning)
	assert.Equal(t, 25*time.Millisecond, stats.UpdateRate)

	// Add trackers and subscribers
	pm.CreateTracker("tracker1", 2)
	pm.CreateTracker("tracker2", 3)
	
	pm.Subscribe("tracker1", func(update ProgressUpdate) {})
	pm.Subscribe("tracker2", func(update ProgressUpdate) {})
	pm.SubscribeAll(func(update ProgressUpdate) {})

	pm.Start()

	stats = pm.GetStats()
	assert.Equal(t, 2, stats.ActiveTrackers)
	assert.Equal(t, 3, stats.TotalSubscribers) // 2 specific + 1 global
	assert.True(t, stats.IsRunning)

	pm.Stop()
}

func TestProgressManager_ConcurrentAccess(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	pm.Start()
	defer pm.Stop()

	// Create multiple trackers concurrently
	var wg sync.WaitGroup
	trackerCount := 10
	toolsPerTracker := 5

	for i := 0; i < trackerCount; i++ {
		wg.Add(1)
		go func(trackerID int) {
			defer wg.Done()
			
			id := fmt.Sprintf("tracker_%d", trackerID)
			pm.CreateTracker(id, toolsPerTracker)
			
			// Update tools concurrently
			for j := 0; j < toolsPerTracker; j++ {
				toolID := fmt.Sprintf("tool_%d_%d", trackerID, j)
				pm.UpdateToolStatus(id, toolID, "ConcurrentTool", ToolStatusRunning, 0.5, "Working")
				
				result := ToolResult{Success: true, Content: "Done"}
				pm.CompleteToolExecution(id, toolID, result, nil)
			}
		}(i)
	}

	wg.Wait()

	// Verify all trackers were created and completed
	stats := pm.GetStats()
	assert.Equal(t, trackerCount, stats.ActiveTrackers)
}

func TestProgressManager_EdgeCases(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)

	t.Run("update nonexistent tracker", func(t *testing.T) {
		// Should not panic or error
		pm.UpdateToolStatus("nonexistent", "tool1", "Tool", ToolStatusRunning, 0.5, "Working")
		pm.CompleteToolExecution("nonexistent", "tool1", ToolResult{}, nil)
	})

	t.Run("zero total tools", func(t *testing.T) {
		tracker := pm.CreateTracker("zero-tools", 0)
		snapshot := tracker.GetProgressSnapshot()
		assert.Equal(t, 1.0, snapshot.Progress) // Should be 100% when no tools expected
		assert.True(t, tracker.IsComplete())
	})

	t.Run("more completions than expected", func(t *testing.T) {
		tracker := pm.CreateTracker("over-complete", 1)
		
		result := ToolResult{Success: true}
		tracker.CompleteToolExecution("tool1", result, nil)
		tracker.CompleteToolExecution("tool2", result, nil) // Extra completion
		
		assert.Equal(t, 2, tracker.CompletedTools)
		assert.True(t, tracker.IsComplete())
	})
}

func TestProgressUpdate_Generation(t *testing.T) {
	pm := NewProgressManager(10 * time.Millisecond)
	tracker := pm.CreateTracker("test-tracker", 3)

	// Add some active and completed tools
	tracker.UpdateToolProgress("active1", "ActiveTool1", ToolStatusRunning, 0.5, "Working")
	tracker.UpdateToolProgress("active2", "ActiveTool2", ToolStatusQueued, 0.0, "Waiting")
	
	result := ToolResult{
		Success: true,
		Content: "Completed",
		Metadata: ToolMetadata{
			ToolName: "CompletedTool",
			StartTime: time.Now().Add(-2 * time.Second),
			EndTime: time.Now().Add(-1 * time.Second),
			ExecutionTime: 1 * time.Second,
		},
	}
	tracker.CompleteToolExecution("completed1", result, nil)

	// Generate update
	update := pm.generateUpdate(tracker)

	assert.Equal(t, "test-tracker", update.TrackerID)
	assert.Equal(t, StatusRunning, update.OverallStatus)
	
	// Check progress info
	progress := update.Progress
	assert.Equal(t, 3, progress.TotalTools)
	assert.Equal(t, 1, progress.CompletedTools)
	assert.Equal(t, 2, progress.ActiveTools)
	assert.Equal(t, 0, progress.FailedTools)
	assert.InDelta(t, 0.33, progress.Progress, 0.01)
	assert.Greater(t, progress.ElapsedTime, time.Duration(0))

	// Check tool updates
	assert.Len(t, update.ToolUpdates, 3) // 2 active + 1 completed
	
	// Find updates by status
	activeUpdates := 0
	completedUpdates := 0
	for _, toolUpdate := range update.ToolUpdates {
		switch toolUpdate.Status {
		case ToolStatusRunning, ToolStatusQueued:
			activeUpdates++
		case ToolStatusCompleted:
			completedUpdates++
			assert.NotNil(t, toolUpdate.Result)
		}
	}
	assert.Equal(t, 2, activeUpdates)
	assert.Equal(t, 1, completedUpdates)
}