package collections

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBoundedMap_BasicOperations(t *testing.T) {
	config := BoundedMapConfig[string, int]{
		MaxSize: 3,
	}
	
	bm := NewBoundedMap(config)
	
	// Test Store and Load
	bm.Store("key1", 100)
	bm.Store("key2", 200)
	bm.Store("key3", 300)
	
	if value, ok := bm.Load("key1"); !ok || value != 100 {
		t.Errorf("Expected key1=100, got %v, %v", value, ok)
	}
	
	if value, ok := bm.Load("key2"); !ok || value != 200 {
		t.Errorf("Expected key2=200, got %v, %v", value, ok)
	}
	
	if bm.Size() != 3 {
		t.Errorf("Expected size 3, got %d", bm.Size())
	}
}

func TestBoundedMap_LRUEviction(t *testing.T) {
	config := BoundedMapConfig[string, int]{
		MaxSize: 2,
	}
	
	bm := NewBoundedMap(config)
	
	// Fill to capacity
	bm.Store("key1", 100)
	bm.Store("key2", 200)
	
	// Access key1 to make it more recently used
	bm.Load("key1")
	
	// Add key3, should evict key2 (least recently used)
	bm.Store("key3", 300)
	
	// key1 should still exist
	if _, ok := bm.Load("key1"); !ok {
		t.Error("Expected key1 to still exist")
	}
	
	// key2 should be evicted
	if _, ok := bm.Load("key2"); ok {
		t.Error("Expected key2 to be evicted")
	}
	
	// key3 should exist
	if value, ok := bm.Load("key3"); !ok || value != 300 {
		t.Errorf("Expected key3=300, got %v, %v", value, ok)
	}
}

func TestBoundedMap_ConcurrentAccess(t *testing.T) {
	config := BoundedMapConfig[string, int]{
		MaxSize: 100,
	}
	
	bm := NewBoundedMap(config)
	
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100
	
	// Concurrent stores
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				bm.Store(key, id*1000+j)
			}
		}(i)
	}
	
	// Concurrent loads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				bm.Load(key) // May or may not exist due to eviction
			}
		}(i)
	}
	
	wg.Wait()
	
	// Map should not exceed max size
	if bm.Size() > 100 {
		t.Errorf("Expected size <= 100, got %d", bm.Size())
	}
}

func TestBoundedMap_EvictionCallback(t *testing.T) {
	evictedKeys := make([]string, 0)
	var mu sync.Mutex
	
	config := BoundedMapConfig[string, int]{
		MaxSize: 2,
		EvictFunc: func(key string, value int) {
			mu.Lock()
			evictedKeys = append(evictedKeys, key)
			mu.Unlock()
		},
	}
	
	bm := NewBoundedMap(config)
	
	bm.Store("key1", 100)
	bm.Store("key2", 200)
	bm.Store("key3", 300) // Should evict key1
	
	// Wait a bit for callback to execute
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	if len(evictedKeys) != 1 || evictedKeys[0] != "key1" {
		t.Errorf("Expected evicted keys [key1], got %v", evictedKeys)
	}
	mu.Unlock()
}

func TestBoundedMap_MemoryTracking(t *testing.T) {
	itemSizeFunc := func(key string, value int) int64 {
		return int64(len(key)) + 8 // 8 bytes for int
	}
	
	config := BoundedMapConfig[string, int]{
		MaxSize:      10,
		MaxMemory:    50, // Very small limit to test memory eviction
		ItemSizeFunc: itemSizeFunc,
	}
	
	bm := NewBoundedMap(config)
	
	// Store items that will exceed memory limit
	bm.Store("key1", 100)  // ~12 bytes
	bm.Store("key2", 200)  // ~12 bytes
	bm.Store("key3", 300)  // ~12 bytes
	bm.Store("key4", 400)  // ~12 bytes - should trigger eviction
	
	stats := bm.GetStats()
	if stats.MemoryUsed > 50 {
		t.Errorf("Expected memory usage <= 50, got %d", stats.MemoryUsed)
	}
}

func BenchmarkBoundedMap_Store(b *testing.B) {
	config := BoundedMapConfig[string, int]{
		MaxSize: 10000,
	}
	
	bm := NewBoundedMap(config)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%1000)
			bm.Store(key, i)
			i++
		}
	})
}

func BenchmarkBoundedMap_Load(b *testing.B) {
	config := BoundedMapConfig[string, int]{
		MaxSize: 10000,
	}
	
	bm := NewBoundedMap(config)
	
	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		bm.Store(key, i)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%1000)
			bm.Load(key)
			i++
		}
	})
}