package execution

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
)

// MemoryMonitor tracks memory usage and pressure
type MemoryMonitor struct {
	// Configuration
	memoryLimit     int64
	pressureThreshold float64
	
	// Current state (atomic access)
	currentUsage    int64
	peakUsage       int64
	underPressure   int32 // boolean
	
	// Monitoring
	monitorInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Statistics
	samples         int64
	lastGC          time.Time
	gcCount         uint32
	lastSample      time.Time
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(memoryLimit int64) *MemoryMonitor {
	if memoryLimit <= 0 {
		memoryLimit = 1024 * 1024 * 1024 // Default 1GB
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MemoryMonitor{
		memoryLimit:       memoryLimit,
		pressureThreshold: 0.8, // 80% threshold
		monitorInterval:   1 * time.Second,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start begins memory monitoring
func (mm *MemoryMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(mm.monitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mm.sample()
		case <-mm.ctx.Done():
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops memory monitoring
func (mm *MemoryMonitor) Stop() {
	mm.cancel()
}

// GetCurrentUsage returns current memory usage in bytes
func (mm *MemoryMonitor) GetCurrentUsage() int64 {
	return atomic.LoadInt64(&mm.currentUsage)
}

// GetPeakUsage returns peak memory usage in bytes
func (mm *MemoryMonitor) GetPeakUsage() int64 {
	return atomic.LoadInt64(&mm.peakUsage)
}

// IsUnderPressure returns true if memory pressure is detected
func (mm *MemoryMonitor) IsUnderPressure() bool {
	return atomic.LoadInt32(&mm.underPressure) == 1
}

// GetMemoryLimit returns the configured memory limit
func (mm *MemoryMonitor) GetMemoryLimit() int64 {
	return mm.memoryLimit
}

// GetUtilization returns memory utilization as a percentage
func (mm *MemoryMonitor) GetUtilization() float64 {
	current := float64(atomic.LoadInt64(&mm.currentUsage))
	limit := float64(mm.memoryLimit)
	return current / limit
}

// Private methods

func (mm *MemoryMonitor) sample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Update current usage (using heap in use as approximation)
	currentUsage := int64(m.HeapInuse + m.StackInuse)
	atomic.StoreInt64(&mm.currentUsage, currentUsage)
	
	// Update peak usage
	for {
		peak := atomic.LoadInt64(&mm.peakUsage)
		if currentUsage <= peak {
			break
		}
		if atomic.CompareAndSwapInt64(&mm.peakUsage, peak, currentUsage) {
			break
		}
	}
	
	// Check pressure
	utilization := float64(currentUsage) / float64(mm.memoryLimit)
	if utilization > mm.pressureThreshold {
		atomic.StoreInt32(&mm.underPressure, 1)
		
		// Trigger GC if memory pressure is high
		if utilization > 0.9 {
			mm.triggerGC()
		}
	} else {
		atomic.StoreInt32(&mm.underPressure, 0)
	}
	
	// Update statistics
	atomic.AddInt64(&mm.samples, 1)
	mm.lastSample = time.Now()
	
	// Check if GC occurred
	if m.NumGC > mm.gcCount {
		mm.gcCount = m.NumGC
		mm.lastGC = time.Now()
	}
}

func (mm *MemoryMonitor) triggerGC() {
	runtime.GC()
	runtime.GC() // Run twice for better cleanup
}