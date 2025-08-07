package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tui/events"
	"github.com/rivo/tview"
)

// Operation represents an asynchronous operation
type Operation struct {
	ID          string
	Type        string
	Description string
	Task        func(ctx context.Context) (interface{}, error)
	OnSuccess   func(result interface{})
	OnError     func(error)
	OnProgress  func(progress float64, message string)
	Context     context.Context
	Cancel      context.CancelFunc
	StartTime   time.Time
}

// AsyncManager handles all blocking operations off the UI thread
type AsyncManager struct {
	operations map[string]*Operation
	mutex      sync.RWMutex
	app        *tview.Application
	eventBus   *events.EventBus
	log        *logger.Logger
	workers    int
	queue      chan *Operation
	done       chan struct{}
}

// NewAsyncManager creates a new async operations manager
func NewAsyncManager(app *tview.Application, eventBus *events.EventBus, workers int) *AsyncManager {
	if workers <= 0 {
		workers = 5 // Default worker count
	}

	am := &AsyncManager{
		operations: make(map[string]*Operation),
		app:        app,
		eventBus:   eventBus,
		log:        logger.WithComponent("async_manager"),
		workers:    workers,
		queue:      make(chan *Operation, 100), // Buffered queue
		done:       make(chan struct{}),
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go am.worker(i)
	}

	return am
}

// Execute queues an asynchronous operation
func (am *AsyncManager) Execute(op *Operation) string {
	if op.ID == "" {
		op.ID = am.generateOperationID()
	}

	// Create context with cancellation if not provided
	if op.Context == nil {
		op.Context, op.Cancel = context.WithCancel(context.Background())
	}

	op.StartTime = time.Now()

	// Store operation
	am.mutex.Lock()
	am.operations[op.ID] = op
	am.mutex.Unlock()

	// Notify that operation started
	am.eventBus.Publish(events.EventAsyncOpStarted, events.AsyncOpPayload{
		OperationID: op.ID,
		Type:        op.Type,
		Data:        op.Description,
	}, "async_manager")

	// Queue the operation
	select {
	case am.queue <- op:
		am.log.Debug("Operation queued", "id", op.ID, "type", op.Type)
	default:
		am.log.Error("Operation queue full, dropping operation", "id", op.ID, "type", op.Type)
		if op.OnError != nil {
			am.safeCallback(func() {
				op.OnError(fmt.Errorf("operation queue full"))
			})
		}
	}

	return op.ID
}

// Cancel cancels an operation by ID
func (am *AsyncManager) Cancel(operationID string) bool {
	am.mutex.RLock()
	op, exists := am.operations[operationID]
	am.mutex.RUnlock()

	if !exists {
		return false
	}

	if op.Cancel != nil {
		op.Cancel()
	}

	am.log.Debug("Operation cancelled", "id", operationID)
	return true
}

// GetOperation returns operation info
func (am *AsyncManager) GetOperation(operationID string) (*Operation, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	op, exists := am.operations[operationID]
	return op, exists
}

// ListOperations returns all current operations
func (am *AsyncManager) ListOperations() map[string]*Operation {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Return a copy to prevent external modifications
	result := make(map[string]*Operation)
	for id, op := range am.operations {
		result[id] = op
	}
	return result
}

// worker processes operations from the queue
func (am *AsyncManager) worker(workerID int) {
	am.log.Debug("Worker started", "workerID", workerID)

	for {
		select {
		case op := <-am.queue:
			am.processOperation(op, workerID)
		case <-am.done:
			am.log.Debug("Worker stopping", "workerID", workerID)
			return
		}
	}
}

// processOperation executes a single operation
func (am *AsyncManager) processOperation(op *Operation, workerID int) {
	am.log.Debug("Processing operation", "id", op.ID, "type", op.Type, "workerID", workerID)

	// Execute the operation
	result, err := op.Task(op.Context)

	// Handle result
	if err != nil {
		// Check if it was cancelled
		if op.Context.Err() == context.Canceled {
			am.log.Debug("Operation cancelled", "id", op.ID)
		} else {
			am.log.Error("Operation failed", "id", op.ID, "error", err)

			// Notify failure
			am.eventBus.Publish(events.EventAsyncOpFailed, events.AsyncOpPayload{
				OperationID: op.ID,
				Type:        op.Type,
				Data:        err.Error(),
			}, "async_manager")

			// Call error callback safely on UI thread
			if op.OnError != nil {
				am.safeCallback(func() {
					op.OnError(err)
				})
			}
		}
	} else {
		am.log.Debug("Operation completed", "id", op.ID, "duration", time.Since(op.StartTime))

		// Notify success
		am.eventBus.Publish(events.EventAsyncOpCompleted, events.AsyncOpPayload{
			OperationID: op.ID,
			Type:        op.Type,
			Data:        result,
		}, "async_manager")

		// Call success callback safely on UI thread
		if op.OnSuccess != nil {
			am.safeCallback(func() {
				op.OnSuccess(result)
			})
		}
	}

	// Clean up operation
	am.mutex.Lock()
	delete(am.operations, op.ID)
	am.mutex.Unlock()
}

// safeCallback executes a callback on the UI thread safely
func (am *AsyncManager) safeCallback(callback func()) {
	if am.app != nil {
		am.app.QueueUpdateDraw(func() {
			defer func() {
				if r := recover(); r != nil {
					am.log.Error("Callback panic", "error", r)
				}
			}()
			callback()
		})
	} else {
		// Fallback: execute directly (for testing)
		callback()
	}
}

// generateOperationID generates a unique operation ID
func (am *AsyncManager) generateOperationID() string {
	return fmt.Sprintf("op_%d", time.Now().UnixNano())
}

// Close shuts down the async manager
func (am *AsyncManager) Close() {
	am.log.Debug("Shutting down async manager")

	// Cancel all pending operations
	am.mutex.RLock()
	for _, op := range am.operations {
		if op.Cancel != nil {
			op.Cancel()
		}
	}
	am.mutex.RUnlock()

	// Stop workers
	close(am.done)

	am.log.Debug("Async manager shut down")
}

// Helper methods for common operation types

// ExecuteAPICall executes an API call asynchronously
func (am *AsyncManager) ExecuteAPICall(
	description string,
	apiCall func(ctx context.Context) (interface{}, error),
	onSuccess func(result interface{}),
	onError func(error),
) string {
	op := &Operation{
		Type:        "api_call",
		Description: description,
		Task:        apiCall,
		OnSuccess:   onSuccess,
		OnError:     onError,
	}

	return am.Execute(op)
}

// ExecuteModelValidation validates a model asynchronously
func (am *AsyncManager) ExecuteModelValidation(
	modelName string,
	validator func(ctx context.Context, model string) error,
	onValid func(),
	onInvalid func(error),
) string {
	op := &Operation{
		Type:        "model_validation",
		Description: fmt.Sprintf("Validating model: %s", modelName),
		Task: func(ctx context.Context) (interface{}, error) {
			return nil, validator(ctx, modelName)
		},
		OnSuccess: func(result interface{}) {
			if onValid != nil {
				onValid()
			}
		},
		OnError: onInvalid,
	}

	return am.Execute(op)
}

// ExecuteModelRefresh refreshes model list asynchronously
func (am *AsyncManager) ExecuteModelRefresh(
	refresher func(ctx context.Context) ([]string, error),
	onRefreshed func(models []string),
	onError func(error),
) string {
	op := &Operation{
		Type:        "model_refresh",
		Description: "Refreshing model list",
		Task: func(ctx context.Context) (interface{}, error) {
			return refresher(ctx)
		},
		OnSuccess: func(result interface{}) {
			if models, ok := result.([]string); ok && onRefreshed != nil {
				onRefreshed(models)
			}
		},
		OnError: onError,
	}

	return am.Execute(op)
}
