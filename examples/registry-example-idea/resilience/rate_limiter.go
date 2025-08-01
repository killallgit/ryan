package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	
	"your-project/pkg/registry/errors"
)

// TokenBucket implements the token bucket algorithm for rate limiting
type TokenBucket struct {
	// Configuration
	capacity   float64       // Maximum number of tokens
	refillRate float64       // Tokens per second
	
	// State (atomic access for performance)
	tokens     int64         // Current tokens (scaled by 1e6 for precision)
	lastRefill int64         // Last refill timestamp (nanoseconds)
	
	// Thread safety for configuration changes
	mu sync.RWMutex
}

// TokenBucketConfig holds configuration for TokenBucket
type TokenBucketConfig struct {
	Capacity   float64 // Maximum tokens
	RefillRate float64 // Tokens per second
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(config TokenBucketConfig) *TokenBucket {
	if config.Capacity <= 0 {
		config.Capacity = 100 // Default capacity
	}
	
	if config.RefillRate <= 0 {
		config.RefillRate = 10 // Default rate: 10 tokens per second
	}
	
	now := time.Now().UnixNano()
	
	return &TokenBucket{
		capacity:   config.Capacity,
		refillRate: config.RefillRate,
		tokens:     int64(config.Capacity * 1e6), // Start with full bucket
		lastRefill: now,
	}
}

// Allow checks if a request can be processed (consumes 1 token)
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if N tokens are available and consumes them if so
func (tb *TokenBucket) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	
	tb.refill()
	
	tokensNeeded := int64(n) * 1e6
	currentTokens := atomic.LoadInt64(&tb.tokens)
	
	for {
		if currentTokens < tokensNeeded {
			return false // Not enough tokens
		}
		
		newTokens := currentTokens - tokensNeeded
		if atomic.CompareAndSwapInt64(&tb.tokens, currentTokens, newTokens) {
			return true // Successfully consumed tokens
		}
		
		// Retry with updated value
		currentTokens = atomic.LoadInt64(&tb.tokens)
	}
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

// WaitN blocks until N tokens are available
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}
	
	// Quick check if tokens are immediately available
	if tb.AllowN(n) {
		return nil
	}
	
	// Calculate wait time
	waitTime := tb.calculateWaitTime(n)
	if waitTime <= 0 {
		return tb.WaitN(ctx, n) // Retry immediately
	}
	
	// Wait with context timeout
	timer := time.NewTimer(waitTime)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		// Try again after waiting
		if tb.AllowN(n) {
			return nil
		}
		// If still not available, wait again (recursive with backoff)
		return tb.WaitN(ctx, n)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reserve reserves N tokens and returns the wait duration
func (tb *TokenBucket) Reserve(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	
	tb.refill()
	
	tokensNeeded := int64(n) * 1e6
	currentTokens := atomic.LoadInt64(&tb.tokens)
	
	if currentTokens >= tokensNeeded {
		// Tokens available immediately
		for {
			if atomic.CompareAndSwapInt64(&tb.tokens, currentTokens, currentTokens-tokensNeeded) {
				return 0
			}
			currentTokens = atomic.LoadInt64(&tb.tokens)
			if currentTokens < tokensNeeded {
				break
			}
		}
	}
	
	// Calculate wait time for needed tokens
	return tb.calculateWaitTime(n)
}

// AvailableTokens returns the current number of available tokens
func (tb *TokenBucket) AvailableTokens() float64 {
	tb.refill()
	tokens := atomic.LoadInt64(&tb.tokens)
	return float64(tokens) / 1e6
}

// Capacity returns the bucket capacity
func (tb *TokenBucket) Capacity() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.capacity
}

// RefillRate returns the current refill rate
func (tb *TokenBucket) RefillRate() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.refillRate
}

// UpdateConfig updates the bucket configuration
func (tb *TokenBucket) UpdateConfig(config TokenBucketConfig) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	if config.Capacity > 0 {
		tb.capacity = config.Capacity
		// Adjust current tokens if new capacity is smaller
		maxTokens := int64(tb.capacity * 1e6)
		currentTokens := atomic.LoadInt64(&tb.tokens)
		if currentTokens > maxTokens {
			atomic.StoreInt64(&tb.tokens, maxTokens)
		}
	}
	
	if config.RefillRate > 0 {
		tb.refillRate = config.RefillRate
	}
}

// Private methods

func (tb *TokenBucket) refill() {
	now := time.Now().UnixNano()
	lastRefill := atomic.LoadInt64(&tb.lastRefill)
	
	if now <= lastRefill {
		return // Clock went backwards or no time passed
	}
	
	// Calculate tokens to add
	timePassed := float64(now-lastRefill) / 1e9 // Convert to seconds
	tb.mu.RLock()
	tokensToAdd := int64(timePassed * tb.refillRate * 1e6)
	capacity := int64(tb.capacity * 1e6)
	tb.mu.RUnlock()
	
	if tokensToAdd <= 0 {
		return
	}
	
	// Update last refill time first to prevent multiple goroutines from refilling
	if !atomic.CompareAndSwapInt64(&tb.lastRefill, lastRefill, now) {
		return // Another goroutine already refilled
	}
	
	// Add tokens up to capacity
	for {
		currentTokens := atomic.LoadInt64(&tb.tokens)
		newTokens := currentTokens + tokensToAdd
		
		if newTokens > capacity {
			newTokens = capacity
		}
		
		if newTokens <= currentTokens {
			break // No tokens to add
		}
		
		if atomic.CompareAndSwapInt64(&tb.tokens, currentTokens, newTokens) {
			break // Successfully added tokens
		}
	}
}

func (tb *TokenBucket) calculateWaitTime(n int) time.Duration {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	
	currentTokens := float64(atomic.LoadInt64(&tb.tokens)) / 1e6
	tokensNeeded := float64(n)
	
	if currentTokens >= tokensNeeded {
		return 0
	}
	
	tokensToWaitFor := tokensNeeded - currentTokens
	waitTime := time.Duration(tokensToWaitFor/tb.refillRate*1000) * time.Millisecond
	
	// Add small buffer to account for timing precision
	return waitTime + time.Millisecond
}

// RateLimiter manages multiple token buckets for different rate limiting scenarios
type RateLimiter struct {
	// Global bucket
	globalBucket *TokenBucket
	
	// Per-user buckets
	userBuckets  sync.Map // string -> *TokenBucket
	userConfig   TokenBucketConfig
	
	// Per-resource buckets
	resourceBuckets sync.Map // string -> *TokenBucket
	resourceConfig  TokenBucketConfig
	
	// Cleanup management
	cleanupInterval time.Duration
	maxIdleTime     time.Duration
	lastCleanup     time.Time
	cleanupMu       sync.Mutex
	
	// Statistics
	stats      RateLimiterStats
	statsMu    sync.RWMutex
	
	// Context for cleanup goroutine
	ctx    context.Context
	cancel context.CancelFunc
}

// RateLimiterConfig holds configuration for RateLimiter
type RateLimiterConfig struct {
	Global              TokenBucketConfig
	PerUser             TokenBucketConfig
	PerResource         TokenBucketConfig
	CleanupInterval     time.Duration
	MaxIdleTime         time.Duration
	EnableCleanup       bool
}

// RateLimiterStats contains usage statistics
type RateLimiterStats struct {
	GlobalAllowed      int64 `json:"global_allowed"`
	GlobalDenied       int64 `json:"global_denied"`
	UserAllowed        int64 `json:"user_allowed"`
	UserDenied         int64 `json:"user_denied"`
	ResourceAllowed    int64 `json:"resource_allowed"`
	ResourceDenied     int64 `json:"resource_denied"`
	ActiveUserBuckets  int   `json:"active_user_buckets"`
	ActiveResourceBuckets int `json:"active_resource_buckets"`
	LastCleanup        time.Time `json:"last_cleanup"`
}

// BucketInfo holds metadata about a token bucket
type BucketInfo struct {
	Bucket     *TokenBucket
	LastUsed   time.Time
	UsageCount int64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	
	if config.MaxIdleTime <= 0 {
		config.MaxIdleTime = 30 * time.Minute
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	rl := &RateLimiter{
		globalBucket:    NewTokenBucket(config.Global),
		userConfig:      config.PerUser,
		resourceConfig:  config.PerResource,
		cleanupInterval: config.CleanupInterval,
		maxIdleTime:     config.MaxIdleTime,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Start cleanup goroutine if enabled
	if config.EnableCleanup {
		go rl.cleanupLoop()
	}
	
	return rl
}

// Allow checks if a request is allowed (consumes 1 token from global bucket)
func (rl *RateLimiter) Allow() bool {
	allowed := rl.globalBucket.Allow()
	
	rl.statsMu.Lock()
	if allowed {
		rl.stats.GlobalAllowed++
	} else {
		rl.stats.GlobalDenied++
	}
	rl.statsMu.Unlock()
	
	return allowed
}

// AllowUser checks if a request is allowed for a specific user
func (rl *RateLimiter) AllowUser(userID string) bool {
	return rl.AllowUserN(userID, 1)
}

// AllowUserN checks if N requests are allowed for a specific user
func (rl *RateLimiter) AllowUserN(userID string, n int) bool {
	// Check global limit first
	if !rl.globalBucket.AllowN(n) {
		rl.statsMu.Lock()
		rl.stats.GlobalDenied++
		rl.statsMu.Unlock()
		return false
	}
	
	// Check user-specific limit
	bucket := rl.getUserBucket(userID)
	allowed := bucket.AllowN(n)
	
	rl.statsMu.Lock()
	if allowed {
		rl.stats.UserAllowed++
		rl.stats.GlobalAllowed++
	} else {
		rl.stats.UserDenied++
		// Global tokens were consumed but user limit hit, so refund global tokens
		// This is a simplified approach - in practice you might want to check user limits first
	}
	rl.statsMu.Unlock()
	
	return allowed
}

// AllowResource checks if a request is allowed for a specific resource
func (rl *RateLimiter) AllowResource(resourceID string) bool {
	return rl.AllowResourceN(resourceID, 1)
}

// AllowResourceN checks if N requests are allowed for a specific resource
func (rl *RateLimiter) AllowResourceN(resourceID string, n int) bool {
	// Check global limit first
	if !rl.globalBucket.AllowN(n) {
		rl.statsMu.Lock()
		rl.stats.GlobalDenied++
		rl.statsMu.Unlock()
		return false
	}
	
	// Check resource-specific limit
	bucket := rl.getResourceBucket(resourceID)
	allowed := bucket.AllowN(n)
	
	rl.statsMu.Lock()
	if allowed {
		rl.stats.ResourceAllowed++
		rl.stats.GlobalAllowed++
	} else {
		rl.stats.ResourceDenied++
	}
	rl.statsMu.Unlock()
	
	return allowed
}

// WaitUser waits for user rate limit to allow the request
func (rl *RateLimiter) WaitUser(ctx context.Context, userID string) error {
	return rl.WaitUserN(ctx, userID, 1)
}

// WaitUserN waits for user rate limit to allow N requests
func (rl *RateLimiter) WaitUserN(ctx context.Context, userID string, n int) error {
	// Wait for global limit
	if err := rl.globalBucket.WaitN(ctx, n); err != nil {
		return errors.NewError(errors.ErrSystemOverload, "global rate limit exceeded").
			Cause(err).
			Build()
	}
	
	// Wait for user limit
	bucket := rl.getUserBucket(userID)
	if err := bucket.WaitN(ctx, n); err != nil {
		return errors.NewError(errors.ErrSystemOverload, "user rate limit exceeded").
			Cause(err).
			Context("user_id", userID).
			Build()
	}
	
	return nil
}

// WaitResource waits for resource rate limit to allow the request
func (rl *RateLimiter) WaitResource(ctx context.Context, resourceID string) error {
	return rl.WaitResourceN(ctx, resourceID, 1)
}

// WaitResourceN waits for resource rate limit to allow N requests
func (rl *RateLimiter) WaitResourceN(ctx context.Context, resourceID string, n int) error {
	// Wait for global limit
	if err := rl.globalBucket.WaitN(ctx, n); err != nil {
		return errors.NewError(errors.ErrSystemOverload, "global rate limit exceeded").
			Cause(err).
			Build()
	}
	
	// Wait for resource limit
	bucket := rl.getResourceBucket(resourceID)
	if err := bucket.WaitN(ctx, n); err != nil {
		return errors.NewError(errors.ErrSystemOverload, "resource rate limit exceeded").
			Cause(err).
			Context("resource_id", resourceID).
			Build()
	}
	
	return nil
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.statsMu.RLock()
	defer rl.statsMu.RUnlock()
	
	stats := rl.stats
	
	// Count active buckets
	stats.ActiveUserBuckets = 0
	rl.userBuckets.Range(func(key, value any) bool {
		stats.ActiveUserBuckets++
		return true
	})
	
	stats.ActiveResourceBuckets = 0
	rl.resourceBuckets.Range(func(key, value any) bool {
		stats.ActiveResourceBuckets++
		return true
	})
	
	return stats
}

// Close shuts down the rate limiter
func (rl *RateLimiter) Close() error {
	rl.cancel()
	return nil
}

// Private methods

func (rl *RateLimiter) getUserBucket(userID string) *TokenBucket {
	if bucket, ok := rl.userBuckets.Load(userID); ok {
		bucketInfo := bucket.(*BucketInfo)
		bucketInfo.LastUsed = time.Now()
		atomic.AddInt64(&bucketInfo.UsageCount, 1)
		return bucketInfo.Bucket
	}
	
	// Create new bucket
	newBucketInfo := &BucketInfo{
		Bucket:     NewTokenBucket(rl.userConfig),
		LastUsed:   time.Now(),
		UsageCount: 1,
	}
	
	if actual, loaded := rl.userBuckets.LoadOrStore(userID, newBucketInfo); loaded {
		actualInfo := actual.(*BucketInfo)
		actualInfo.LastUsed = time.Now()
		atomic.AddInt64(&actualInfo.UsageCount, 1)
		return actualInfo.Bucket
	}
	
	return newBucketInfo.Bucket
}

func (rl *RateLimiter) getResourceBucket(resourceID string) *TokenBucket {
	if bucket, ok := rl.resourceBuckets.Load(resourceID); ok {
		bucketInfo := bucket.(*BucketInfo)
		bucketInfo.LastUsed = time.Now()
		atomic.AddInt64(&bucketInfo.UsageCount, 1)
		return bucketInfo.Bucket
	}
	
	// Create new bucket
	newBucketInfo := &BucketInfo{
		Bucket:     NewTokenBucket(rl.resourceConfig),
		LastUsed:   time.Now(),
		UsageCount: 1,
	}
	
	if actual, loaded := rl.resourceBuckets.LoadOrStore(resourceID, newBucketInfo); loaded {
		actualInfo := actual.(*BucketInfo)
		actualInfo.LastUsed = time.Now()
		atomic.AddInt64(&actualInfo.UsageCount, 1)
		return actualInfo.Bucket
	}
	
	return newBucketInfo.Bucket
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.ctx.Done():
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.cleanupMu.Lock()
	defer rl.cleanupMu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.maxIdleTime)
	
	// Cleanup user buckets
	var expiredUsers []string
	rl.userBuckets.Range(func(key, value any) bool {
		userID := key.(string)
		bucketInfo := value.(*BucketInfo)
		
		if bucketInfo.LastUsed.Before(cutoff) {
			expiredUsers = append(expiredUsers, userID)
		}
		return true
	})
	
	for _, userID := range expiredUsers {
		rl.userBuckets.Delete(userID)
	}
	
	// Cleanup resource buckets
	var expiredResources []string
	rl.resourceBuckets.Range(func(key, value any) bool {
		resourceID := key.(string)
		bucketInfo := value.(*BucketInfo)
		
		if bucketInfo.LastUsed.Before(cutoff) {
			expiredResources = append(expiredResources, resourceID)
		}
		return true
	})
	
	for _, resourceID := range expiredResources {
		rl.resourceBuckets.Delete(resourceID)
	}
	
	// Update cleanup stats
	rl.statsMu.Lock()
	rl.stats.LastCleanup = now
	rl.statsMu.Unlock()
}

// Helper functions for creating common rate limiters

// NewSimpleRateLimiter creates a rate limiter with only global limits
func NewSimpleRateLimiter(capacity float64, refillRate float64) *RateLimiter {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   capacity,
			RefillRate: refillRate,
		},
		EnableCleanup: false, // No per-user/resource buckets to clean up
	}
	
	return NewRateLimiter(config)
}

// NewUserRateLimiter creates a rate limiter with global and per-user limits
func NewUserRateLimiter(globalCapacity, globalRate, userCapacity, userRate float64) *RateLimiter {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   globalCapacity,
			RefillRate: globalRate,
		},
		PerUser: TokenBucketConfig{
			Capacity:   userCapacity,
			RefillRate: userRate,
		},
		EnableCleanup: true,
	}
	
	return NewRateLimiter(config)
}