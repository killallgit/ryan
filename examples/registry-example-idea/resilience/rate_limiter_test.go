package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_BasicOperation(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   10,
		RefillRate: 5, // 5 tokens per second
	}
	
	tb := NewTokenBucket(config)
	
	// Should start with full capacity
	if !tb.Allow() {
		t.Error("Expected first request to be allowed")
	}
	
	available := tb.AvailableTokens()
	if available != 9 { // Should have 9 tokens left
		t.Errorf("Expected 9 tokens available, got %f", available)
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   10,
		RefillRate: 5,
	}
	
	tb := NewTokenBucket(config)
	
	// Should allow taking 5 tokens
	if !tb.AllowN(5) {
		t.Error("Expected AllowN(5) to succeed")
	}
	
	// Should have 5 tokens left
	available := tb.AvailableTokens()
	if available != 5 {
		t.Errorf("Expected 5 tokens available, got %f", available)
	}
	
	// Should not allow taking 6 tokens (only 5 available)
	if tb.AllowN(6) {
		t.Error("Expected AllowN(6) to fail when only 5 tokens available")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   10,
		RefillRate: 10, // 10 tokens per second
	}
	
	tb := NewTokenBucket(config)
	
	// Consume all tokens
	tb.AllowN(10)
	
	if tb.AvailableTokens() != 0 {
		t.Error("Expected 0 tokens after consuming all")
	}
	
	// Wait for refill (100ms should add 1 token at 10 tokens/second)
	time.Sleep(110 * time.Millisecond)
	
	available := tb.AvailableTokens()
	if available < 0.9 || available > 1.1 { // Allow some precision error
		t.Errorf("Expected ~1 token after 100ms, got %f", available)
	}
}

func TestTokenBucket_WaitN(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   5,
		RefillRate: 10, // 10 tokens per second
	}
	
	tb := NewTokenBucket(config)
	
	// Consume all tokens
	tb.AllowN(5)
	
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	
	// Wait for 1 token (should take ~100ms at 10 tokens/second)
	err := tb.WaitN(ctx, 1)
	
	elapsed := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if elapsed < 90*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Errorf("Expected wait time ~100ms, got %v", elapsed)
	}
}

func TestTokenBucket_WaitN_ContextTimeout(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   5,
		RefillRate: 1, // Very slow refill
	}
	
	tb := NewTokenBucket(config)
	
	// Consume all tokens
	tb.AllowN(5)
	
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	// Try to wait for tokens with short timeout
	err := tb.WaitN(ctx, 1)
	
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	config := TokenBucketConfig{
		Capacity:   1000,
		RefillRate: 500, // 500 tokens per second
	}
	
	tb := NewTokenBucket(config)
	
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64
	
	numGoroutines := 10
	numOperations := 200
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				if tb.Allow() {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failureCount, 1)
				}
			}
		}()
	}
	
	wg.Wait()
	
	totalRequests := successCount + failureCount
	expectedRequests := int64(numGoroutines * numOperations)
	
	if totalRequests != expectedRequests {
		t.Errorf("Expected %d total requests, got %d", expectedRequests, totalRequests)
	}
	
	// Should have some successes (started with full bucket)
	if successCount == 0 {
		t.Error("Expected some successful requests")
	}
	
	t.Logf("Success: %d, Failures: %d", successCount, failureCount)
}

func TestRateLimiter_GlobalLimit(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   5,
			RefillRate: 10,
		},
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("Expected request %d to succeed", i)
		}
	}
	
	// 6th request should fail
	if rl.Allow() {
		t.Error("Expected 6th request to fail")
	}
}

func TestRateLimiter_UserLimit(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   100, // High global limit
			RefillRate: 100,
		},
		PerUser: TokenBucketConfig{
			Capacity:   3, // Low per-user limit
			RefillRate: 1,
		},
		EnableCleanup: false, // Disable cleanup for predictable test
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// User1 should be limited to 3 requests
	for i := 0; i < 3; i++ {
		if !rl.AllowUser("user1") {
			t.Errorf("Expected user1 request %d to succeed", i)
		}
	}
	
	// 4th request for user1 should fail
	if rl.AllowUser("user1") {
		t.Error("Expected user1 4th request to fail")
	}
	
	// But user2 should still be able to make requests
	if !rl.AllowUser("user2") {
		t.Error("Expected user2 request to succeed")
	}
}

func TestRateLimiter_ResourceLimit(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   100,
			RefillRate: 100,
		},
		PerResource: TokenBucketConfig{
			Capacity:   2,
			RefillRate: 1,
		},
		EnableCleanup: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// Resource1 should be limited to 2 requests
	for i := 0; i < 2; i++ {
		if !rl.AllowResource("resource1") {
			t.Errorf("Expected resource1 request %d to succeed", i)
		}
	}
	
	// 3rd request for resource1 should fail
	if rl.AllowResource("resource1") {
		t.Error("Expected resource1 3rd request to fail")
	}
	
	// But resource2 should still be able to handle requests
	if !rl.AllowResource("resource2") {
		t.Error("Expected resource2 request to succeed")
	}
}

func TestRateLimiter_WaitUser(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   100,
			RefillRate: 100,
		},
		PerUser: TokenBucketConfig{
			Capacity:   1,
			RefillRate: 10, // 10 tokens per second
		},
		EnableCleanup: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// Consume the user's token
	if !rl.AllowUser("user1") {
		t.Error("Expected first request to succeed")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	
	// Wait for user rate limit
	err := rl.WaitUser(ctx, "user1")
	
	elapsed := time.Since(start)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should have waited approximately 100ms (1 token at 10 tokens/second)
	if elapsed < 90*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Errorf("Expected wait time ~100ms, got %v", elapsed)
	}
}

func TestRateLimiter_Statistics(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   5,
			RefillRate: 10,
		},
		PerUser: TokenBucketConfig{
			Capacity:   2,
			RefillRate: 5,
		},
		EnableCleanup: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// Make some global requests
	for i := 0; i < 3; i++ {
		rl.Allow()
	}
	
	// Make some user requests
	for i := 0; i < 2; i++ {
		rl.AllowUser("user1")
	}
	
	// This should fail (user limit exceeded)
	rl.AllowUser("user1")
	
	stats := rl.GetStats()
	
	if stats.GlobalAllowed != 5 { // 3 global + 2 user (both consume global)
		t.Errorf("Expected 5 global allowed, got %d", stats.GlobalAllowed)
	}
	
	if stats.UserAllowed != 2 {
		t.Errorf("Expected 2 user allowed, got %d", stats.UserAllowed)
	}
	
	if stats.UserDenied != 1 {
		t.Errorf("Expected 1 user denied, got %d", stats.UserDenied)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   100,
			RefillRate: 100,
		},
		PerUser: TokenBucketConfig{
			Capacity:   10,
			RefillRate: 10,
		},
		EnableCleanup:   true,
		CleanupInterval: 50 * time.Millisecond,
		MaxIdleTime:     100 * time.Millisecond,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	// Create user buckets
	rl.AllowUser("user1")
	rl.AllowUser("user2")
	
	stats := rl.GetStats()
	if stats.ActiveUserBuckets != 2 {
		t.Errorf("Expected 2 active user buckets, got %d", stats.ActiveUserBuckets)
	}
	
	// Wait for cleanup to occur
	time.Sleep(200 * time.Millisecond)
	
	stats = rl.GetStats()
	if stats.ActiveUserBuckets != 0 {
		t.Errorf("Expected 0 active user buckets after cleanup, got %d", stats.ActiveUserBuckets)
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	config := TokenBucketConfig{
		Capacity:   1000000,
		RefillRate: 1000000,
	}
	
	tb := NewTokenBucket(config)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tb.Allow()
		}
	})
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   1000000,
			RefillRate: 1000000,
		},
		EnableCleanup: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow()
		}
	})
}

func BenchmarkRateLimiter_AllowUser(b *testing.B) {
	config := RateLimiterConfig{
		Global: TokenBucketConfig{
			Capacity:   1000000,
			RefillRate: 1000000,
		},
		PerUser: TokenBucketConfig{
			Capacity:   1000000,
			RefillRate: 1000000,
		},
		EnableCleanup: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user_%d", i%100) // Simulate 100 different users
			rl.AllowUser(userID)
			i++
		}
	})
}