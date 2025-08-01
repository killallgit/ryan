package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	
	"your-project/pkg/registry/errors"
)

// State represents the current state of the circuit breaker
type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker prevents cascading failures by monitoring operation success/failure rates
type CircuitBreaker struct {
	// Configuration
	maxFailures     int64
	resetTimeout    time.Duration
	halfOpenMaxCalls int64
	
	// State management (atomic access)
	state           int32 // State
	failures        int64
	successes       int64
	lastFailureTime int64 // Unix timestamp
	lastResetTime   int64 // Unix timestamp
	
	// Half-open state tracking
	halfOpenCalls   int64
	halfOpenFailures int64
	
	// Statistics
	totalRequests   int64
	totalFailures   int64
	totalSuccesses  int64
	totalTimeouts   int64
	stateChanges    int64
	
	// Timing
	createdAt       time.Time
	lastStateChange time.Time
	
	// Callbacks
	onStateChange   func(from, to State)
	onOpen          func()
	onClose         func()
	onHalfOpen      func()
	
	// Thread safety for callbacks and stats
	mu sync.RWMutex
}

// CircuitBreakerConfig holds configuration for CircuitBreaker
type CircuitBreakerConfig struct {
	MaxFailures      int64
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int64
	OnStateChange    func(from, to State)
	OnOpen           func()
	OnClose          func()
	OnHalfOpen       func()
}

// CircuitBreakerStats contains usage statistics
type CircuitBreakerStats struct {
	State              string        `json:"state"`
	Failures           int64         `json:"failures"`
	Successes          int64         `json:"successes"`
	TotalRequests      int64         `json:"total_requests"`
	TotalFailures      int64         `json:"total_failures"`
	TotalSuccesses     int64         `json:"total_successes"`
	TotalTimeouts      int64         `json:"total_timeouts"`
	StateChanges       int64         `json:"state_changes"`
	FailureRate        float64       `json:"failure_rate"`
	SuccessRate        float64       `json:"success_rate"`
	LastFailureTime    time.Time     `json:"last_failure_time,omitempty"`
	LastResetTime      time.Time     `json:"last_reset_time,omitempty"`
	LastStateChange    time.Time     `json:"last_state_change"`
	CreatedAt          time.Time     `json:"created_at"`
	Age                time.Duration `json:"age"`
	TimeSinceLastReset time.Duration `json:"time_since_last_reset"`
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5 // Default threshold
	}
	
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second // Default timeout
	}
	
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 3 // Default half-open calls
	}
	
	now := time.Now()
	
	return &CircuitBreaker{
		maxFailures:       config.MaxFailures,
		resetTimeout:      config.ResetTimeout,
		halfOpenMaxCalls:  config.HalfOpenMaxCalls,
		onStateChange:     config.OnStateChange,
		onOpen:            config.OnOpen,
		onClose:           config.OnClose,
		onHalfOpen:        config.OnHalfOpen,
		createdAt:         now,
		lastStateChange:   now,
		lastResetTime:     now.Unix(),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.ExecuteWithContext(context.Background(), fn)
}

// ExecuteWithContext runs the given function with circuit breaker protection and context
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() error) error {
	// Check if request is allowed
	if !cb.allowRequest() {
		atomic.AddInt64(&cb.totalRequests, 1)
		return errors.NewError(errors.ErrSystemOverload, "circuit breaker is open").
			Context("state", cb.GetState().String()).
			Context("failures", atomic.LoadInt64(&cb.failures)).
			Context("time_since_last_failure", time.Since(cb.getLastFailureTime())).
			WithRetry(cb.getRetryAfter()).
			Build()
	}
	
	// Execute with timeout if context has deadline
	var err error
	done := make(chan struct{})
	
	go func() {
		defer close(done)
		err = fn()
	}()
	
	select {
	case <-done:
		// Function completed
		cb.recordResult(err == nil)
		return err
	case <-ctx.Done():
		// Context timeout or cancellation
		atomic.AddInt64(&cb.totalTimeouts, 1)
		cb.recordResult(false)
		return errors.NewError(errors.ErrExecutionTimeout, "operation timed out").
			Cause(ctx.Err()).
			Context("state", cb.GetState().String()).
			Build()
	}
}

// Call is an alias for Execute for backward compatibility
func (cb *CircuitBreaker) Call(fn func() error) error {
	return cb.Execute(fn)
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}

// IsClosed returns true if the circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.GetState() == StateClosed
}

// IsHalfOpen returns true if the circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.GetState() == StateHalfOpen
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.GetState()
	
	// Reset counters
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
	atomic.StoreInt64(&cb.halfOpenCalls, 0)
	atomic.StoreInt64(&cb.halfOpenFailures, 0)
	atomic.StoreInt64(&cb.lastResetTime, time.Now().Unix())
	
	// Set to closed state
	cb.setState(StateClosed)
	
	// Notify of state change
	if oldState != StateClosed {
		cb.notifyStateChange(oldState, StateClosed)
	}
}

// ForceOpen manually opens the circuit breaker
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.GetState()
	cb.setState(StateOpen)
	
	if oldState != StateOpen {
		cb.notifyStateChange(oldState, StateOpen)
	}
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	totalRequests := atomic.LoadInt64(&cb.totalRequests)
	totalFailures := atomic.LoadInt64(&cb.totalFailures)
	totalSuccesses := atomic.LoadInt64(&cb.totalSuccesses)
	
	var failureRate, successRate float64
	if totalRequests > 0 {
		failureRate = float64(totalFailures) / float64(totalRequests)
		successRate = float64(totalSuccesses) / float64(totalRequests)
	}
	
	var lastFailureTime, lastResetTime time.Time
	if lft := atomic.LoadInt64(&cb.lastFailureTime); lft > 0 {
		lastFailureTime = time.Unix(lft, 0)
	}
	if lrt := atomic.LoadInt64(&cb.lastResetTime); lrt > 0 {
		lastResetTime = time.Unix(lrt, 0)
	}
	
	now := time.Now()
	var timeSinceLastReset time.Duration
	if !lastResetTime.IsZero() {
		timeSinceLastReset = now.Sub(lastResetTime)
	}
	
	return CircuitBreakerStats{
		State:              cb.GetState().String(),
		Failures:           atomic.LoadInt64(&cb.failures),
		Successes:          atomic.LoadInt64(&cb.successes),
		TotalRequests:      totalRequests,
		TotalFailures:      totalFailures,
		TotalSuccesses:     totalSuccesses,
		TotalTimeouts:      atomic.LoadInt64(&cb.totalTimeouts),
		StateChanges:       atomic.LoadInt64(&cb.stateChanges),
		FailureRate:        failureRate,
		SuccessRate:        successRate,
		LastFailureTime:    lastFailureTime,
		LastResetTime:      lastResetTime,
		LastStateChange:    cb.lastStateChange,
		CreatedAt:          cb.createdAt,
		Age:                now.Sub(cb.createdAt),
		TimeSinceLastReset: timeSinceLastReset,
	}
}

// Private methods

func (cb *CircuitBreaker) allowRequest() bool {
	state := cb.GetState()
	
	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if it's time to try half-open
		lastFailureTime := time.Unix(atomic.LoadInt64(&cb.lastFailureTime), 0)
		if time.Since(lastFailureTime) > cb.resetTimeout {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				atomic.StoreInt64(&cb.halfOpenCalls, 0)
				atomic.StoreInt64(&cb.halfOpenFailures, 0)
				cb.mu.Unlock()
				
				cb.notifyStateChange(StateOpen, StateHalfOpen)
				
				if cb.onHalfOpen != nil {
					go cb.onHalfOpen()
				}
				
				return true
			}
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		halfOpenCalls := atomic.LoadInt64(&cb.halfOpenCalls)
		return halfOpenCalls < cb.halfOpenMaxCalls
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(success bool) {
	atomic.AddInt64(&cb.totalRequests, 1)
	
	if success {
		cb.recordSuccess()
	} else {
		cb.recordFailure()
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	atomic.AddInt64(&cb.totalSuccesses, 1)
	atomic.AddInt64(&cb.successes, 1)
	
	state := cb.GetState()
	
	if state == StateHalfOpen {
		halfOpenCalls := atomic.AddInt64(&cb.halfOpenCalls, 1)
		
		// If we've had enough successful calls in half-open, close the circuit
		if halfOpenCalls >= cb.halfOpenMaxCalls {
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateClosed)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				atomic.StoreInt64(&cb.failures, 0)
				atomic.StoreInt64(&cb.successes, 0)
				atomic.StoreInt64(&cb.lastResetTime, time.Now().Unix())
				cb.mu.Unlock()
				
				cb.notifyStateChange(StateHalfOpen, StateClosed)
				
				if cb.onClose != nil {
					go cb.onClose()
				}
			}
		}
	}
}

func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt64(&cb.totalFailures, 1)
	atomic.AddInt64(&cb.failures, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().Unix())
	
	state := cb.GetState()
	
	if state == StateHalfOpen {
		// Any failure in half-open state should open the circuit
		atomic.AddInt64(&cb.halfOpenFailures, 1)
		
		if atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateOpen)) {
			cb.mu.Lock()
			cb.lastStateChange = time.Now()
			cb.mu.Unlock()
			
			cb.notifyStateChange(StateHalfOpen, StateOpen)
			
			if cb.onOpen != nil {
				go cb.onOpen()
			}
		}
	} else if state == StateClosed {
		// Check if we should open the circuit due to too many failures
		failures := atomic.LoadInt64(&cb.failures)
		
		if failures >= cb.maxFailures {
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateClosed), int32(StateOpen)) {
				cb.mu.Lock()
				cb.lastStateChange = time.Now()
				cb.mu.Unlock()
				
				cb.notifyStateChange(StateClosed, StateOpen)
				
				if cb.onOpen != nil {
					go cb.onOpen()
				}
			}
		}
	}
}

func (cb *CircuitBreaker) setState(newState State) {
	atomic.StoreInt32(&cb.state, int32(newState))
	atomic.AddInt64(&cb.stateChanges, 1)
	cb.lastStateChange = time.Now()
}

func (cb *CircuitBreaker) notifyStateChange(from, to State) {
	if cb.onStateChange != nil {
		go cb.onStateChange(from, to)
	}
}

func (cb *CircuitBreaker) getLastFailureTime() time.Time {
	timestamp := atomic.LoadInt64(&cb.lastFailureTime)
	if timestamp == 0 {
		return time.Time{}
	}
	return time.Unix(timestamp, 0)
}

func (cb *CircuitBreaker) getRetryAfter() time.Duration {
	if cb.GetState() == StateOpen {
		lastFailureTime := cb.getLastFailureTime()
		if !lastFailureTime.IsZero() {
			elapsed := time.Since(lastFailureTime)
			if elapsed < cb.resetTimeout {
				return cb.resetTimeout - elapsed
			}
		}
	}
	return 0
}

// CircuitBreakerGroup manages multiple circuit breakers
type CircuitBreakerGroup struct {
	breakers sync.Map // string -> *CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerGroup creates a new group of circuit breakers
func NewCircuitBreakerGroup(config CircuitBreakerConfig) *CircuitBreakerGroup {
	return &CircuitBreakerGroup{
		config: config,
	}
}

// GetBreaker returns a circuit breaker for the given name, creating it if needed
func (cbg *CircuitBreakerGroup) GetBreaker(name string) *CircuitBreaker {
	if breaker, ok := cbg.breakers.Load(name); ok {
		return breaker.(*CircuitBreaker)
	}
	
	// Create new circuit breaker
	newBreaker := NewCircuitBreaker(cbg.config)
	
	if actual, loaded := cbg.breakers.LoadOrStore(name, newBreaker); loaded {
		return actual.(*CircuitBreaker)
	}
	
	return newBreaker
}

// Execute runs a function with the named circuit breaker
func (cbg *CircuitBreakerGroup) Execute(name string, fn func() error) error {
	breaker := cbg.GetBreaker(name)
	return breaker.Execute(fn)
}

// ExecuteWithContext runs a function with the named circuit breaker and context
func (cbg *CircuitBreakerGroup) ExecuteWithContext(ctx context.Context, name string, fn func() error) error {
	breaker := cbg.GetBreaker(name)
	return breaker.ExecuteWithContext(ctx, fn)
}

// GetAllStats returns stats for all circuit breakers
func (cbg *CircuitBreakerGroup) GetAllStats() map[string]CircuitBreakerStats {
	stats := make(map[string]CircuitBreakerStats)
	
	cbg.breakers.Range(func(key, value any) bool {
		name := key.(string)
		breaker := value.(*CircuitBreaker)
		stats[name] = breaker.GetStats()
		return true
	})
	
	return stats
}

// ResetAll resets all circuit breakers in the group
func (cbg *CircuitBreakerGroup) ResetAll() {
	cbg.breakers.Range(func(key, value any) bool {
		breaker := value.(*CircuitBreaker)
		breaker.Reset()
		return true
	})
}

// Remove removes a circuit breaker from the group
func (cbg *CircuitBreakerGroup) Remove(name string) {
	cbg.breakers.Delete(name)
}

// List returns all circuit breaker names
func (cbg *CircuitBreakerGroup) List() []string {
	var names []string
	
	cbg.breakers.Range(func(key, value any) bool {
		names = append(names, key.(string))
		return true
	})
	
	return names
}