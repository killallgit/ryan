package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExecutorPool manages a pool of workers for concurrent tool execution
type ExecutorPool struct {
	workerCount    int
	jobQueue       chan ToolJob
	workers        []*Worker
	resultChannels map[string]chan ToolResult
	resultsMu      sync.RWMutex
	quit           chan bool
	isRunning      bool
	mu             sync.RWMutex
}

// Worker represents a single worker in the pool
type Worker struct {
	ID         int
	pool       *ExecutorPool
	jobChannel chan ToolJob
	quit       chan bool
}

// ToolJob represents a tool execution job
type ToolJob struct {
	ID      string
	Request ToolRequest
	Tool    Tool
	Result  chan ToolResult
}

// NewExecutorPool creates a new executor pool with the specified number of workers
func NewExecutorPool(workerCount int) *ExecutorPool {
	if workerCount <= 0 {
		workerCount = 4 // Default to 4 workers
	}

	return &ExecutorPool{
		workerCount:    workerCount,
		jobQueue:       make(chan ToolJob, workerCount*2), // Buffered queue
		workers:        make([]*Worker, 0, workerCount),
		resultChannels: make(map[string]chan ToolResult),
		quit:           make(chan bool),
		isRunning:      false,
	}
}

// Start initializes and starts all workers in the pool
func (p *ExecutorPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("executor pool is already running")
	}

	// Create and start workers
	for i := 0; i < p.workerCount; i++ {
		worker := &Worker{
			ID:         i + 1,
			pool:       p,
			jobChannel: make(chan ToolJob),
			quit:       make(chan bool),
		}
		p.workers = append(p.workers, worker)
		go worker.start()
	}

	// Start the job dispatcher
	go p.dispatch()

	p.isRunning = true
	return nil
}

// Stop gracefully shuts down the executor pool
func (p *ExecutorPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("executor pool is not running")
	}

	// Signal all workers to quit
	for _, worker := range p.workers {
		close(worker.quit)
	}

	// Signal dispatcher to quit
	close(p.quit)

	// Close all result channels
	p.resultsMu.Lock()
	for _, ch := range p.resultChannels {
		close(ch)
	}
	p.resultChannels = make(map[string]chan ToolResult)
	p.resultsMu.Unlock()

	p.isRunning = false
	return nil
}

// Submit submits a tool for execution and returns a channel for the result
func (p *ExecutorPool) Submit(id string, tool Tool, request ToolRequest) (<-chan ToolResult, error) {
	p.mu.RLock()
	if !p.isRunning {
		p.mu.RUnlock()
		return nil, fmt.Errorf("executor pool is not running")
	}
	p.mu.RUnlock()

	// Create result channel for this job
	resultChan := make(chan ToolResult, 1)
	
	p.resultsMu.Lock()
	p.resultChannels[id] = resultChan
	p.resultsMu.Unlock()

	job := ToolJob{
		ID:      id,
		Request: request,
		Tool:    tool,
		Result:  resultChan,
	}

	// Try to submit job with timeout to prevent blocking
	select {
	case p.jobQueue <- job:
		return resultChan, nil
	case <-time.After(5 * time.Second):
		// Clean up if we can't submit
		p.resultsMu.Lock()
		if _, exists := p.resultChannels[id]; exists {
			delete(p.resultChannels, id)
			close(resultChan)
		}
		p.resultsMu.Unlock()
		return nil, fmt.Errorf("failed to submit job: queue full")
	}
}

// GetStats returns current pool statistics
func (p *ExecutorPool) GetStats() ExecutorStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.resultsMu.RLock()
	activeJobs := len(p.resultChannels)
	p.resultsMu.RUnlock()

	return ExecutorStats{
		WorkerCount:      p.workerCount,
		QueuedJobs:       len(p.jobQueue),
		ActiveJobs:       activeJobs,
		IsRunning:        p.isRunning,
		TotalWorkers:     len(p.workers),
	}
}

// ExecutorStats provides statistics about the executor pool
type ExecutorStats struct {
	WorkerCount  int  `json:"worker_count"`
	QueuedJobs   int  `json:"queued_jobs"`
	ActiveJobs   int  `json:"active_jobs"`
	IsRunning    bool `json:"is_running"`
	TotalWorkers int  `json:"total_workers"`
}

// dispatch distributes jobs to available workers
func (p *ExecutorPool) dispatch() {
	for {
		select {
		case job := <-p.jobQueue:
			// Try to send job to any available worker
			go func(job ToolJob) {
				// Try each worker until one accepts the job
				for _, worker := range p.workers {
					select {
					case worker.jobChannel <- job:
						return // Job dispatched successfully
					default:
						continue // Worker busy, try next one
					}
				}
				// If no worker is immediately available, block on the first worker
				// This ensures the job is eventually processed
				if len(p.workers) > 0 {
					p.workers[0].jobChannel <- job
				}
			}(job)
		case <-p.quit:
			return
		}
	}
}

// start begins the worker's job processing loop
func (w *Worker) start() {
	for {
		select {
		case job := <-w.jobChannel:
			w.executeJob(job)
		case <-w.quit:
			return
		}
	}
}

// executeJob executes a single tool job
func (w *Worker) executeJob(job ToolJob) {
	startTime := time.Now()
	var result ToolResult

	defer func() {
		// Recover from panics
		if r := recover(); r != nil {
			result = ToolResult{
				Success: false,
				Error:   fmt.Sprintf("tool execution panicked: %v", r),
				Metadata: ToolMetadata{
					StartTime:     startTime,
					EndTime:       time.Now(),
					ExecutionTime: time.Since(startTime),
					ToolName:      job.Tool.Name(),
					Parameters:    job.Request.Parameters,
				},
			}
		}

		// Send result (even if it's from a panic)
		select {
		case job.Result <- result:
			// Result sent successfully
		case <-time.After(1 * time.Second):
			// Result channel might be closed or blocked
			// This prevents the worker from hanging
		}

		// Clean up result channel from pool and close it
		w.pool.resultsMu.Lock()
		if _, exists := w.pool.resultChannels[job.ID]; exists {
			delete(w.pool.resultChannels, job.ID)
			close(job.Result)
		}
		w.pool.resultsMu.Unlock()
	}()

	// Execute the tool
	ctx := job.Request.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	result, err = job.Tool.Execute(ctx, job.Request.Parameters)
	
	// Update metadata
	endTime := time.Now()
	result.Metadata.StartTime = startTime
	result.Metadata.EndTime = endTime
	result.Metadata.ExecutionTime = endTime.Sub(startTime)
	result.Metadata.ToolName = job.Tool.Name()
	result.Metadata.Parameters = job.Request.Parameters

	// Handle execution error
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	}
}