package execution

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"your-project/pkg/registry/errors"
	"your-project/pkg/registry/resilience"
)

// ExecutionEngine manages tool execution with worker pools and backpressure
type ExecutionEngine struct {
	// Configuration
	config *EngineConfig
	
	// Worker management
	workers      []*Worker
	workerPool   chan *Worker
	taskQueue    chan *ExecutionTask
	
	// Resource management
	activeWorkers  int32
	totalTasks     int64
	completedTasks int64
	failedTasks    int64
	
	// Circuit breaker and rate limiter
	circuitBreaker *resilience.CircuitBreaker
	rateLimiter    *resilience.RateLimiter
	
	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	stopping   bool
	mu         sync.RWMutex
	
	// Statistics and monitoring
	stats      ExecutionStats
	statsMu    sync.RWMutex
	
	// Resource monitoring
	memMonitor *MemoryMonitor
	
	// Cleanup and maintenance
	cleanupTicker *time.Ticker
}

// EngineConfig holds configuration for ExecutionEngine
type EngineConfig struct {
	// Worker pool settings
	MinWorkers    int           `yaml:"min_workers" json:"min_workers"`
	MaxWorkers    int           `yaml:"max_workers" json:"max_workers"`
	QueueSize     int           `yaml:"queue_size" json:"queue_size"`
	WorkerTimeout time.Duration `yaml:"worker_timeout" json:"worker_timeout"`
	
	// Resource limits
	MaxMemoryPerTask   int64         `yaml:"max_memory_per_task" json:"max_memory_per_task"`
	MaxCPUPerTask      time.Duration `yaml:"max_cpu_per_task" json:"max_cpu_per_task"`
	GlobalMemoryLimit  int64         `yaml:"global_memory_limit" json:"global_memory_limit"`
	
	// Backpressure settings
	EnableBackpressure bool          `yaml:"enable_backpressure" json:"enable_backpressure"`
	BackpressureThreshold float64    `yaml:"backpressure_threshold" json:"backpressure_threshold"`
	
	// Circuit breaker
	CircuitBreakerConfig resilience.CircuitBreakerConfig `yaml:"circuit_breaker" json:"circuit_breaker"`
	
	// Rate limiting
	RateLimiterConfig resilience.RateLimiterConfig `yaml:"rate_limiter" json:"rate_limiter"`
	
	// Monitoring
	EnableMonitoring   bool          `yaml:"enable_monitoring" json:"enable_monitoring"`
	MonitoringInterval time.Duration `yaml:"monitoring_interval" json:"monitoring_interval"`
	
	// Cleanup
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// ExecutionTask represents a task to be executed
type ExecutionTask struct {
	ID          string                 `json:"id"`
	ToolName    string                 `json:"tool_name"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     context.Context        `json:"-"`
	ResultChan  chan *ExecutionResult  `json:"-"`
	
	// Timing and limits
	SubmittedAt time.Time     `json:"submitted_at"`
	Timeout     time.Duration `json:"timeout"`
	Deadline    time.Time     `json:"deadline"`
	
	// Resource limits
	MemoryLimit int64         `json:"memory_limit"`
	CPULimit    time.Duration `json:"cpu_limit"`
	
	// Priority and retry
	Priority    int           `json:"priority"`
	RetryCount  int           `json:"retry_count"`
	MaxRetries  int           `json:"max_retries"`
	
	// Callback for tool execution
	ExecuteFunc func(ctx context.Context) (*ExecutionResult, error) `json:"-"`
}

// ExecutionResult represents the result of task execution
type ExecutionResult struct {
	TaskID      string                 `json:"task_id"`
	ToolName    string                 `json:"tool_name"`
	Success     bool                   `json:"success"`
	Result      interface{}            `json:"result"`
	Error       error                  `json:"error"`
	
	// Timing information
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	QueueTime     time.Duration `json:"queue_time"`
	
	// Resource usage
	MemoryUsed    int64         `json:"memory_used"`
	CPUTime       time.Duration `json:"cpu_time"`
	
	// Worker information
	WorkerID      int           `json:"worker_id"`
	
	// Retry information
	AttemptNumber int           `json:"attempt_number"`
	Retried       bool          `json:"retried"`
}

// ExecutionStats contains execution engine statistics
type ExecutionStats struct {
	// Task statistics
	TotalTasks      int64         `json:"total_tasks"`
	CompletedTasks  int64         `json:"completed_tasks"`
	FailedTasks     int64         `json:"failed_tasks"`
	QueuedTasks     int64         `json:"queued_tasks"`
	
	// Worker statistics
	ActiveWorkers   int32         `json:"active_workers"`
	IdleWorkers     int32         `json:"idle_workers"`
	TotalWorkers    int32         `json:"total_workers"`
	
	// Timing statistics
	AverageQueueTime   time.Duration `json:"average_queue_time"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	
	// Resource statistics
	TotalMemoryUsed    int64         `json:"total_memory_used"`
	PeakMemoryUsed     int64         `json:"peak_memory_used"`
	TotalCPUTime       time.Duration `json:"total_cpu_time"`
	
	// Backpressure statistics
	BackpressureEvents int64         `json:"backpressure_events"`
	ThrottledTasks     int64         `json:"throttled_tasks"`
	
	// Circuit breaker statistics
	CircuitBreakerOpened int64       `json:"circuit_breaker_opened"`
	CircuitBreakerClosed int64       `json:"circuit_breaker_closed"`
	
	// Rate limiting statistics
	RateLimitedTasks   int64         `json:"rate_limited_tasks"`
	
	// Last update
	LastUpdated        time.Time     `json:"last_updated"`
}

// Worker represents a worker in the execution engine
type Worker struct {
	ID           int
	engine       *ExecutionEngine
	taskChan     chan *ExecutionTask
	quit         chan struct{}
	
	// Worker state
	busy         bool
	currentTask  *ExecutionTask
	startTime    time.Time
	lastActivity time.Time
	
	// Worker statistics
	tasksCompleted int64
	tasksErrored   int64
	totalCPUTime   time.Duration
	totalMemoryUsed int64
	
	// Resource monitoring
	memoryBefore int64
	cpuBefore    time.Time
	
	mu sync.RWMutex
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(config *EngineConfig) *ExecutionEngine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	
	// Validate and set defaults
	if config.MinWorkers <= 0 {
		config.MinWorkers = 1
	}
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU() * 2
	}
	if config.QueueSize <= 0 {
		config.QueueSize = config.MaxWorkers * 10
	}
	if config.WorkerTimeout <= 0 {
		config.WorkerTimeout = 30 * time.Second
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	engine := &ExecutionEngine{
		config:    config,
		workers:   make([]*Worker, 0, config.MaxWorkers),
		workerPool: make(chan *Worker, config.MaxWorkers),
		taskQueue: make(chan *ExecutionTask, config.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
	
	// Initialize circuit breaker if configured
	if config.CircuitBreakerConfig.MaxFailures > 0 {
		engine.circuitBreaker = resilience.NewCircuitBreaker(config.CircuitBreakerConfig)
	}
	
	// Initialize rate limiter if configured
	if config.RateLimiterConfig.Global.Capacity > 0 {
		engine.rateLimiter = resilience.NewRateLimiter(config.RateLimiterConfig)
	}
	
	// Initialize memory monitor if configured
	if config.EnableMonitoring {
		engine.memMonitor = NewMemoryMonitor(config.GlobalMemoryLimit)
	}
	
	return engine
}

// Start initializes and starts the execution engine
func (ee *ExecutionEngine) Start(ctx context.Context) error {
	ee.mu.Lock()
	defer ee.mu.Unlock()
	
	if ee.started {
		return errors.NewError(errors.ErrInternalError, "execution engine already started").Build()
	}
	
	// Start minimum number of workers
	for i := 0; i < ee.config.MinWorkers; i++ {
		worker := ee.createWorker(i)
		ee.workers = append(ee.workers, worker)
		go worker.start()
		ee.workerPool <- worker
	}
	
	// Start task dispatcher
	go ee.taskDispatcher()
	
	// Start monitoring if enabled
	if ee.config.EnableMonitoring {
		go ee.monitoringLoop()
	}
	
	// Start cleanup routine
	if ee.config.CleanupInterval > 0 {
		ee.cleanupTicker = time.NewTicker(ee.config.CleanupInterval)
		go ee.cleanupLoop()
	}
	
	// Start memory monitor if enabled
	if ee.memMonitor != nil {
		go ee.memMonitor.Start(ctx)
	}
	
	ee.started = true
	return nil
}

// Stop gracefully shuts down the execution engine
func (ee *ExecutionEngine) Stop(timeout time.Duration) error {
	ee.mu.Lock()
	if !ee.started || ee.stopping {
		ee.mu.Unlock()
		return nil
	}
	ee.stopping = true
	ee.mu.Unlock()
	
	// Stop accepting new tasks
	ee.cancel()
	
	// Close task queue
	close(ee.taskQueue)
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		ee.waitForWorkers()
		close(done)
	}()
	
	select {
	case <-done:
		// All workers finished gracefully
	case <-time.After(timeout):
		// Timeout reached, force shutdown
		ee.forceShutdown()
	}
	
	// Stop cleanup ticker
	if ee.cleanupTicker != nil {
		ee.cleanupTicker.Stop()
	}
	
	// Stop memory monitor
	if ee.memMonitor != nil {
		ee.memMonitor.Stop()
	}
	
	// Close rate limiter
	if ee.rateLimiter != nil {
		ee.rateLimiter.Close()
	}
	
	ee.mu.Lock()
	ee.started = false
	ee.stopping = false
	ee.mu.Unlock()
	
	return nil
}

// Execute submits a task for execution
func (ee *ExecutionEngine) Execute(ctx context.Context, task *ExecutionTask) (*ExecutionResult, error) {
	if !ee.started || ee.stopping {
		return nil, errors.NewError(errors.ErrRegistryNotStarted, "execution engine not started").Build()
	}
	
	// Check rate limits
	if ee.rateLimiter != nil {
		if !ee.rateLimiter.Allow() {
			atomic.AddInt64(&ee.stats.RateLimitedTasks, 1)
			return nil, errors.NewError(errors.ErrSystemOverload, "rate limit exceeded").
				WithRetry(1 * time.Second).
				Build()
		}
	}
	
	// Check circuit breaker
	if ee.circuitBreaker != nil && ee.circuitBreaker.IsOpen() {
		return nil, errors.NewError(errors.ErrSystemOverload, "circuit breaker is open").
			WithRetry(30 * time.Second).
			Build()
	}
	
	// Check backpressure
	if ee.config.EnableBackpressure {
		if ee.isBackpressureActive() {
			atomic.AddInt64(&ee.stats.BackpressureEvents, 1)
			atomic.AddInt64(&ee.stats.ThrottledTasks, 1)
			return nil, errors.NewError(errors.ErrSystemOverload, "system under high load").
				WithRetry(5 * time.Second).
				Build()
		}
	}
	
	// Check memory pressure
	if ee.memMonitor != nil && ee.memMonitor.IsUnderPressure() {
		return nil, errors.NewError(errors.ErrResourceLimit, "memory pressure detected").
			WithRetry(10 * time.Second).
			Build()
	}
	
	// Set task defaults
	if task.ID == "" {
		task.ID = generateTaskID()
	}
	if task.SubmittedAt.IsZero() {
		task.SubmittedAt = time.Now()
	}
	if task.Timeout <= 0 {
		task.Timeout = ee.config.WorkerTimeout
	}
	if task.Deadline.IsZero() {
		task.Deadline = task.SubmittedAt.Add(task.Timeout)
	}
	if task.MemoryLimit <= 0 {
		task.MemoryLimit = ee.config.MaxMemoryPerTask
	}
	if task.CPULimit <= 0 {
		task.CPULimit = ee.config.MaxCPUPerTask
	}
	
	task.Context = ctx
	task.ResultChan = make(chan *ExecutionResult, 1)
	
	// Submit task to queue
	select {
	case ee.taskQueue <- task:
		atomic.AddInt64(&ee.totalTasks, 1)
		atomic.AddInt64(&ee.stats.QueuedTasks, 1)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Second): // Submission timeout
		return nil, errors.NewError(errors.ErrSystemOverload, "task submission timeout").Build()
	}
	
	// Wait for result
	select {
	case result := <-task.ResultChan:
		atomic.AddInt64(&ee.stats.QueuedTasks, -1)
		
		if result.Success {
			atomic.AddInt64(&ee.completedTasks, 1)
			atomic.AddInt64(&ee.stats.CompletedTasks, 1)
		} else {
			atomic.AddInt64(&ee.failedTasks, 1)
			atomic.AddInt64(&ee.stats.FailedTasks, 1)
		}
		
		return result, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetStats returns current execution statistics
func (ee *ExecutionEngine) GetStats() ExecutionStats {
	ee.statsMu.RLock()
	defer ee.statsMu.RUnlock()
	
	stats := ee.stats
	stats.TotalTasks = atomic.LoadInt64(&ee.totalTasks)
	stats.CompletedTasks = atomic.LoadInt64(&ee.completedTasks)
	stats.FailedTasks = atomic.LoadInt64(&ee.failedTasks)
	stats.ActiveWorkers = atomic.LoadInt32(&ee.activeWorkers)
	stats.TotalWorkers = int32(len(ee.workers))
	stats.IdleWorkers = stats.TotalWorkers - stats.ActiveWorkers
	stats.LastUpdated = time.Now()
	
	return stats
}

// Private methods

func (ee *ExecutionEngine) createWorker(id int) *Worker {
	return &Worker{
		ID:           id,
		engine:       ee,
		taskChan:     make(chan *ExecutionTask, 1),
		quit:         make(chan struct{}),
		startTime:    time.Now(),
		lastActivity: time.Now(),
	}
}

func (ee *ExecutionEngine) taskDispatcher() {
	for {
		select {
		case task := <-ee.taskQueue:
			if task == nil {
				return // Channel closed
			}
			
			// Get available worker or create new one
			worker := ee.getOrCreateWorker()
			if worker == nil {
				// No workers available and can't create more
				result := &ExecutionResult{
					TaskID:   task.ID,
					ToolName: task.ToolName,
					Success:  false,
					Error:    errors.NewError(errors.ErrSystemOverload, "no workers available").Build(),
					EndTime:  time.Now(),
				}
				task.ResultChan <- result
				continue
			}
			
			// Assign task to worker
			select {
			case worker.taskChan <- task:
				// Task assigned successfully
			default:
				// Worker busy, put back in pool
				ee.workerPool <- worker
				// Retry task assignment
				go func() {
					ee.taskQueue <- task
				}()
			}
			
		case <-ee.ctx.Done():
			return
		}
	}
}

func (ee *ExecutionEngine) getOrCreateWorker() *Worker {
	select {
	case worker := <-ee.workerPool:
		return worker
	default:
		// No idle workers, try to create new one
		ee.mu.Lock()
		defer ee.mu.Unlock()
		
		if len(ee.workers) < ee.config.MaxWorkers {
			worker := ee.createWorker(len(ee.workers))
			ee.workers = append(ee.workers, worker)
			go worker.start()
			return worker
		}
		
		return nil // Can't create more workers
	}
}

func (ee *ExecutionEngine) isBackpressureActive() bool {
	queuedTasks := atomic.LoadInt64(&ee.stats.QueuedTasks)
	queueUtilization := float64(queuedTasks) / float64(ee.config.QueueSize)
	
	return queueUtilization > ee.config.BackpressureThreshold
}

func (ee *ExecutionEngine) monitoringLoop() {
	ticker := time.NewTicker(ee.config.MonitoringInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ee.updateStats()
		case <-ee.ctx.Done():
			return
		}
	}
}

func (ee *ExecutionEngine) updateStats() {
	ee.statsMu.Lock()
	defer ee.statsMu.Unlock()
	
	// Update timing statistics
	// This would collect timing data from completed tasks
	// Implementation would maintain running averages
	
	// Update resource statistics
	if ee.memMonitor != nil {
		ee.stats.TotalMemoryUsed = ee.memMonitor.GetCurrentUsage()
		if ee.stats.TotalMemoryUsed > ee.stats.PeakMemoryUsed {
			ee.stats.PeakMemoryUsed = ee.stats.TotalMemoryUsed
		}
	}
	
	ee.stats.LastUpdated = time.Now()
}

func (ee *ExecutionEngine) cleanupLoop() {
	for {
		select {
		case <-ee.cleanupTicker.C:
			ee.cleanup()
		case <-ee.ctx.Done():
			return
		}
	}
}

func (ee *ExecutionEngine) cleanup() {
	// Clean up idle workers if we have more than minimum
	ee.mu.Lock()
	defer ee.mu.Unlock()
	
	if len(ee.workers) <= ee.config.MinWorkers {
		return
	}
	
	now := time.Now()
	idleThreshold := 10 * time.Minute
	
	// Find idle workers to remove
	var activeWorkers []*Worker
	for _, worker := range ee.workers {
		worker.mu.RLock()
		idle := !worker.busy && now.Sub(worker.lastActivity) > idleThreshold
		worker.mu.RUnlock()
		
		if idle && len(activeWorkers) >= ee.config.MinWorkers {
			// Stop idle worker
			close(worker.quit)
		} else {
			activeWorkers = append(activeWorkers, worker)
		}
	}
	
	ee.workers = activeWorkers
}

func (ee *ExecutionEngine) waitForWorkers() {
	var wg sync.WaitGroup
	
	for _, worker := range ee.workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			close(w.quit)
		}(worker)
	}
	
	wg.Wait()
}

func (ee *ExecutionEngine) forceShutdown() {
	for _, worker := range ee.workers {
		close(worker.quit)
	}
}

// Worker methods

func (w *Worker) start() {
	defer func() {
		if r := recover(); r != nil {
			// Log panic and restart worker
			fmt.Printf("Worker %d panicked: %v\n", w.ID, r)
		}
	}()
	
	for {
		select {
		case task := <-w.taskChan:
			if task != nil {
				w.executeTask(task)
			}
			
			// Return worker to pool
			select {
			case w.engine.workerPool <- w:
			default:
				// Pool full, worker will be garbage collected
				return
			}
			
		case <-w.quit:
			return
		}
	}
}

func (w *Worker) executeTask(task *ExecutionTask) {
	w.mu.Lock()
	w.busy = true
	w.currentTask = task
	w.lastActivity = time.Now()
	w.mu.Unlock()
	
	defer func() {
		w.mu.Lock()
		w.busy = false
		w.currentTask = nil
		w.lastActivity = time.Now()
		w.mu.Unlock()
	}()
	
	atomic.AddInt32(&w.engine.activeWorkers, 1)
	defer atomic.AddInt32(&w.engine.activeWorkers, -1)
	
	startTime := time.Now()
	queueTime := startTime.Sub(task.SubmittedAt)
	
	// Record memory before execution
	if w.engine.memMonitor != nil {
		w.memoryBefore = w.engine.memMonitor.GetCurrentUsage()
	}
	w.cpuBefore = time.Now()
	
	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(task.Context, task.Timeout)
	defer cancel()
	
	// Execute the task
	var result *ExecutionResult
	var err error
	
	// Execute with circuit breaker protection if available
	if w.engine.circuitBreaker != nil {
		err = w.engine.circuitBreaker.ExecuteWithContext(ctx, func() error {
			result, err = task.ExecuteFunc(ctx)
			return err
		})
	} else {
		result, err = task.ExecuteFunc(ctx)
	}
	
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	cpuTime := endTime.Sub(w.cpuBefore)
	
	// Calculate memory usage
	var memoryUsed int64
	if w.engine.memMonitor != nil {
		currentMemory := w.engine.memMonitor.GetCurrentUsage()
		if currentMemory > w.memoryBefore {
			memoryUsed = currentMemory - w.memoryBefore
		}
	}
	
	// Create execution result
	if result == nil {
		result = &ExecutionResult{}
	}
	
	result.TaskID = task.ID
	result.ToolName = task.ToolName
	result.Success = err == nil
	result.Error = err
	result.StartTime = startTime
	result.EndTime = endTime
	result.Duration = duration
	result.QueueTime = queueTime
	result.MemoryUsed = memoryUsed
	result.CPUTime = cpuTime
	result.WorkerID = w.ID
	result.AttemptNumber = task.RetryCount + 1
	
	// Update worker statistics
	w.mu.Lock()
	if result.Success {
		w.tasksCompleted++
	} else {
		w.tasksErrored++
	}
	w.totalCPUTime += cpuTime
	w.totalMemoryUsed += memoryUsed
	w.mu.Unlock()
	
	// Send result back
	select {
	case task.ResultChan <- result:
	default:
		// Channel might be closed, ignore
	}
}

// Utility functions

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// DefaultEngineConfig returns a default configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MinWorkers:            1,
		MaxWorkers:            runtime.NumCPU() * 2,
		QueueSize:             1000,
		WorkerTimeout:         30 * time.Second,
		MaxMemoryPerTask:      100 * 1024 * 1024, // 100MB
		MaxCPUPerTask:         10 * time.Second,
		GlobalMemoryLimit:     1024 * 1024 * 1024, // 1GB
		EnableBackpressure:    true,
		BackpressureThreshold: 0.8,
		EnableMonitoring:      true,
		MonitoringInterval:    30 * time.Second,
		CleanupInterval:       5 * time.Minute,
	}
}