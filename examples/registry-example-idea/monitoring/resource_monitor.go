package monitoring

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"your-project/pkg/registry/collections"
	"your-project/pkg/registry/errors"
)

// ResourceMonitor tracks system resources and provides alerts on pressure
type ResourceMonitor struct {
	// Configuration
	config *ResourceMonitorConfig
	
	// Current resource state (atomic access)
	memoryUsage     int64
	cpuUsage        int64 // percentage * 100 for precision
	goroutineCount  int64
	heapSize        int64
	stackSize       int64
	gcPauseTime     int64 // nanoseconds
	
	// Pressure indicators (atomic flags)
	memoryPressure    int32 // boolean
	cpuPressure       int32 // boolean
	goroutinePressure int32 // boolean
	
	// Historical data
	memoryHistory     *collections.TimeBasedRingBuffer[MemorySample]
	cpuHistory        *collections.TimeBasedRingBuffer[CPUSample]
	goroutineHistory  *collections.TimeBasedRingBuffer[GoroutineSample]
	
	// Alerting
	alertHandlers []AlertHandler
	alertMu       sync.RWMutex
	
	// Statistics
	stats         ResourceStats
	statsMu       sync.RWMutex
	
	// Cleanup handlers prioritized by importance
	cleanupHandlers []CleanupHandler
	cleanupMu       sync.RWMutex
	
	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	runningMu sync.RWMutex
	
	// Internal metrics
	lastGCStats    runtime.MemStats
	lastSampleTime time.Time
	sampleCount    int64
}

// ResourceMonitorConfig holds configuration for ResourceMonitor
type ResourceMonitorConfig struct {
	// Sampling intervals
	SampleInterval    time.Duration `yaml:"sample_interval" json:"sample_interval"`
	HistoryDuration   time.Duration `yaml:"history_duration" json:"history_duration"`
	
	// Memory thresholds
	MemoryThreshold    float64 `yaml:"memory_threshold" json:"memory_threshold"`         // 0.8 = 80%
	MemoryLimit        int64   `yaml:"memory_limit" json:"memory_limit"`                 // bytes
	HeapThreshold      float64 `yaml:"heap_threshold" json:"heap_threshold"`             // 0.7 = 70%
	
	// CPU thresholds
	CPUThreshold       float64 `yaml:"cpu_threshold" json:"cpu_threshold"`               // 0.8 = 80%
	
	// Goroutine thresholds
	GoroutineThreshold int     `yaml:"goroutine_threshold" json:"goroutine_threshold"`   // absolute count
	GoroutineLimit     int     `yaml:"goroutine_limit" json:"goroutine_limit"`           // hard limit
	
	// GC thresholds
	GCPauseThreshold   time.Duration `yaml:"gc_pause_threshold" json:"gc_pause_threshold"` // max acceptable pause
	
	// Alert settings
	EnableAlerts       bool          `yaml:"enable_alerts" json:"enable_alerts"`
	AlertCooldown      time.Duration `yaml:"alert_cooldown" json:"alert_cooldown"`
	
	// Cleanup settings
	EnableAutoCleanup  bool          `yaml:"enable_auto_cleanup" json:"enable_auto_cleanup"`
	CleanupThreshold   float64       `yaml:"cleanup_threshold" json:"cleanup_threshold"`   // trigger cleanup at this pressure
}

// MemorySample represents a memory usage sample
type MemorySample struct {
	Timestamp    time.Time `json:"timestamp"`
	HeapInUse    int64     `json:"heap_in_use"`
	HeapSys      int64     `json:"heap_sys"`
	StackInUse   int64     `json:"stack_in_use"`
	StackSys     int64     `json:"stack_sys"`
	TotalAlloc   int64     `json:"total_alloc"`
	NumGC        uint32    `json:"num_gc"`
	GCPauseTotal int64     `json:"gc_pause_total"`
	Utilization  float64   `json:"utilization"`
}

// CPUSample represents a CPU usage sample
type CPUSample struct {
	Timestamp   time.Time `json:"timestamp"`
	Usage       float64   `json:"usage"`       // percentage
	UserTime    int64     `json:"user_time"`   // nanoseconds
	SystemTime  int64     `json:"system_time"` // nanoseconds
	Utilization float64   `json:"utilization"`
}

// GoroutineSample represents a goroutine count sample
type GoroutineSample struct {
	Timestamp   time.Time `json:"timestamp"`
	Count       int       `json:"count"`
	Utilization float64   `json:"utilization"`
}

// ResourceStats contains resource monitoring statistics
type ResourceStats struct {
	// Current values
	CurrentMemoryUsage   int64     `json:"current_memory_usage"`
	CurrentCPUUsage      float64   `json:"current_cpu_usage"`
	CurrentGoroutines    int       `json:"current_goroutines"`
	CurrentHeapSize      int64     `json:"current_heap_size"`
	
	// Peak values
	PeakMemoryUsage      int64     `json:"peak_memory_usage"`
	PeakCPUUsage         float64   `json:"peak_cpu_usage"`
	PeakGoroutineCount   int       `json:"peak_goroutine_count"`
	PeakHeapSize         int64     `json:"peak_heap_size"`
	
	// Pressure indicators
	MemoryPressure       bool      `json:"memory_pressure"`
	CPUPressure          bool      `json:"cpu_pressure"`
	GoroutinePressure    bool      `json:"goroutine_pressure"`
	
	// GC statistics
	TotalGCPauses        int64     `json:"total_gc_pauses"`
	LastGCPause          time.Duration `json:"last_gc_pause"`
	AverageGCPause       time.Duration `json:"average_gc_pause"`
	
	// Sampling statistics
	SampleCount          int64     `json:"sample_count"`
	SamplingDuration     time.Duration `json:"sampling_duration"`
	LastSample           time.Time `json:"last_sample"`
	
	// Alert statistics
	MemoryAlerts         int64     `json:"memory_alerts"`
	CPUAlerts            int64     `json:"cpu_alerts"`
	GoroutineAlerts      int64     `json:"goroutine_alerts"`
	
	// Cleanup statistics
	CleanupExecutions    int64     `json:"cleanup_executions"`
	LastCleanup          time.Time `json:"last_cleanup"`
}

// AlertType represents different types of resource alerts
type AlertType int

const (
	AlertMemoryPressure AlertType = iota
	AlertCPUPressure
	AlertGoroutinePressure
	AlertGCPause
	AlertMemoryLimit
	AlertGoroutineLimit
)

func (at AlertType) String() string {
	switch at {
	case AlertMemoryPressure:
		return "MEMORY_PRESSURE"
	case AlertCPUPressure:
		return "CPU_PRESSURE"
	case AlertGoroutinePressure:
		return "GOROUTINE_PRESSURE"
	case AlertGCPause:
		return "GC_PAUSE"
	case AlertMemoryLimit:
		return "MEMORY_LIMIT"
	case AlertGoroutineLimit:
		return "GOROUTINE_LIMIT"
	default:
		return "UNKNOWN"
	}
}

// Alert represents a resource alert
type Alert struct {
	Type        AlertType   `json:"type"`
	Message     string      `json:"message"`
	Timestamp   time.Time   `json:"timestamp"`
	Severity    Severity    `json:"severity"`
	CurrentValue interface{} `json:"current_value"`
	ThresholdValue interface{} `json:"threshold_value"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Severity represents alert severity levels
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityCritical
	SeverityEmergency
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityCritical:
		return "CRITICAL"
	case SeverityEmergency:
		return "EMERGENCY"
	default:
		return "UNKNOWN"
	}
}

// AlertHandler handles resource alerts
type AlertHandler interface {
	HandleAlert(alert *Alert) error
	GetName() string
}

// CleanupHandler handles resource cleanup
type CleanupHandler interface {
	HandleMemoryPressure() error
	HandleGoroutinePressure() error
	GetPriority() int // Higher priority handlers run first
	GetName() string
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(config *ResourceMonitorConfig) *ResourceMonitor {
	if config == nil {
		config = DefaultResourceMonitorConfig()
	}
	
	// Set defaults
	if config.SampleInterval <= 0 {
		config.SampleInterval = 5 * time.Second
	}
	if config.HistoryDuration <= 0 {
		config.HistoryDuration = 1 * time.Hour
	}
	if config.MemoryThreshold <= 0 {
		config.MemoryThreshold = 0.8
	}
	if config.CPUThreshold <= 0 {
		config.CPUThreshold = 0.8
	}
	if config.GoroutineThreshold <= 0 {
		config.GoroutineThreshold = 10000
	}
	if config.AlertCooldown <= 0 {
		config.AlertCooldown = 5 * time.Minute
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Calculate history capacity based on duration and sample interval
	historyCapacity := int(config.HistoryDuration / config.SampleInterval)
	if historyCapacity < 100 {
		historyCapacity = 100
	}
	
	rm := &ResourceMonitor{
		config:           config,
		memoryHistory:    collections.NewTimeBasedRingBuffer[MemorySample](historyCapacity, config.HistoryDuration),
		cpuHistory:       collections.NewTimeBasedRingBuffer[CPUSample](historyCapacity, config.HistoryDuration),
		goroutineHistory: collections.NewTimeBasedRingBuffer[GoroutineSample](historyCapacity, config.HistoryDuration),
		alertHandlers:    make([]AlertHandler, 0),
		cleanupHandlers:  make([]CleanupHandler, 0),
		ctx:              ctx,
		cancel:           cancel,
		lastSampleTime:   time.Now(),
	}
	
	return rm
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start(ctx context.Context) error {
	rm.runningMu.Lock()
	defer rm.runningMu.Unlock()
	
	if rm.running {
		return errors.NewError(errors.ErrInternalError, "resource monitor already running").Build()
	}
	
	rm.running = true
	
	// Start monitoring loop
	go rm.monitoringLoop()
	
	return nil
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() error {
	rm.runningMu.Lock()
	defer rm.runningMu.Unlock()
	
	if !rm.running {
		return nil
	}
	
	rm.cancel()
	rm.running = false
	
	return nil
}

// AddAlertHandler adds an alert handler
func (rm *ResourceMonitor) AddAlertHandler(handler AlertHandler) {
	rm.alertMu.Lock()
	defer rm.alertMu.Unlock()
	
	rm.alertHandlers = append(rm.alertHandlers, handler)
}

// AddCleanupHandler adds a cleanup handler
func (rm *ResourceMonitor) AddCleanupHandler(handler CleanupHandler) {
	rm.cleanupMu.Lock()
	defer rm.cleanupMu.Unlock()
	
	// Insert handler in priority order (higher priority first)
	inserted := false
	for i, existing := range rm.cleanupHandlers {
		if handler.GetPriority() > existing.GetPriority() {
			rm.cleanupHandlers = append(rm.cleanupHandlers[:i], append([]CleanupHandler{handler}, rm.cleanupHandlers[i:]...)...)
			inserted = true
			break
		}
	}
	
	if !inserted {
		rm.cleanupHandlers = append(rm.cleanupHandlers, handler)
	}
}

// GetStats returns current resource statistics
func (rm *ResourceMonitor) GetStats() ResourceStats {
	rm.statsMu.RLock()
	defer rm.statsMu.RUnlock()
	
	stats := rm.stats
	stats.CurrentMemoryUsage = atomic.LoadInt64(&rm.memoryUsage)
	stats.CurrentCPUUsage = float64(atomic.LoadInt64(&rm.cpuUsage)) / 100.0
	stats.CurrentGoroutines = int(atomic.LoadInt64(&rm.goroutineCount))
	stats.CurrentHeapSize = atomic.LoadInt64(&rm.heapSize)
	stats.MemoryPressure = atomic.LoadInt32(&rm.memoryPressure) == 1
	stats.CPUPressure = atomic.LoadInt32(&rm.cpuPressure) == 1
	stats.GoroutinePressure = atomic.LoadInt32(&rm.goroutinePressure) == 1
	stats.SampleCount = atomic.LoadInt64(&rm.sampleCount)
	stats.SamplingDuration = time.Since(rm.lastSampleTime)
	stats.LastSample = rm.lastSampleTime
	
	return stats
}

// GetMemoryHistory returns recent memory samples
func (rm *ResourceMonitor) GetMemoryHistory(duration time.Duration) []MemorySample {
	return rm.memoryHistory.GetItemsNewerThan(duration)
}

// GetCPUHistory returns recent CPU samples
func (rm *ResourceMonitor) GetCPUHistory(duration time.Duration) []CPUSample {
	return rm.cpuHistory.GetItemsNewerThan(duration)
}

// GetGoroutineHistory returns recent goroutine samples
func (rm *ResourceMonitor) GetGoroutineHistory(duration time.Duration) []GoroutineSample {
	return rm.goroutineHistory.GetItemsNewerThan(duration)
}

// IsUnderPressure returns true if any resource is under pressure
func (rm *ResourceMonitor) IsUnderPressure() bool {
	return atomic.LoadInt32(&rm.memoryPressure) == 1 ||
		atomic.LoadInt32(&rm.cpuPressure) == 1 ||
		atomic.LoadInt32(&rm.goroutinePressure) == 1
}

// Private methods

func (rm *ResourceMonitor) monitoringLoop() {
	ticker := time.NewTicker(rm.config.SampleInterval)
	defer ticker.Stop()
	
	// Take initial sample
	rm.sample()
	
	for {
		select {
		case <-ticker.C:
			rm.sample()
		case <-rm.ctx.Done():
			return
		}
	}
}

func (rm *ResourceMonitor) sample() {
	now := time.Now()
	atomic.AddInt64(&rm.sampleCount, 1)
	rm.lastSampleTime = now
	
	// Sample memory
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	memorySample := MemorySample{
		Timestamp:    now,
		HeapInUse:    int64(memStats.HeapInuse),
		HeapSys:      int64(memStats.HeapSys),
		StackInUse:   int64(memStats.StackInuse),
		StackSys:     int64(memStats.StackSys),
		TotalAlloc:   int64(memStats.TotalAlloc),
		NumGC:        memStats.NumGC,
		GCPauseTotal: int64(memStats.PauseTotalNs),
	}
	
	totalMemory := memorySample.HeapInUse + memorySample.StackInUse
	if rm.config.MemoryLimit > 0 {
		memorySample.Utilization = float64(totalMemory) / float64(rm.config.MemoryLimit)
	}
	
	rm.memoryHistory.PushWithTime(memorySample)
	atomic.StoreInt64(&rm.memoryUsage, totalMemory)
	atomic.StoreInt64(&rm.heapSize, memorySample.HeapInUse)
	
	// Sample goroutines
	goroutineCount := runtime.NumGoroutine()
	goroutineSample := GoroutineSample{
		Timestamp: now,
		Count:     goroutineCount,
	}
	
	if rm.config.GoroutineLimit > 0 {
		goroutineSample.Utilization = float64(goroutineCount) / float64(rm.config.GoroutineLimit)
	}
	
	rm.goroutineHistory.PushWithTime(goroutineSample)
	atomic.StoreInt64(&rm.goroutineCount, int64(goroutineCount))
	
	// CPU sampling would require platform-specific code
	// For now, we'll use a simplified approach
	cpuSample := CPUSample{
		Timestamp: now,
		Usage:     0.0, // Would be calculated from system metrics
	}
	rm.cpuHistory.PushWithTime(cpuSample)
	
	// Check for pressure and alerts
	rm.checkPressure(memorySample, cpuSample, goroutineSample)
	
	// Update peak values
	rm.updatePeaks(totalMemory, cpuSample.Usage, goroutineCount, memorySample.HeapInUse)
	
	// Handle GC metrics
	if memStats.NumGC > rm.lastGCStats.NumGC {
		rm.handleGCMetrics(&memStats)
	}
	
	rm.lastGCStats = memStats
}

func (rm *ResourceMonitor) checkPressure(memorySample MemorySample, cpuSample CPUSample, goroutineSample GoroutineSample) {
	// Check memory pressure
	memoryPressure := memorySample.Utilization > rm.config.MemoryThreshold
	if memoryPressure != (atomic.LoadInt32(&rm.memoryPressure) == 1) {
		atomic.StoreInt32(&rm.memoryPressure, boolToInt32(memoryPressure))
		
		if memoryPressure {
			rm.handleAlert(&Alert{
				Type:           AlertMemoryPressure,
				Message:        "Memory pressure detected",
				Timestamp:      time.Now(),
				Severity:       SeverityWarning,
				CurrentValue:   memorySample.Utilization,
				ThresholdValue: rm.config.MemoryThreshold,
			})
			
			if rm.config.EnableAutoCleanup {
				rm.executeCleanup(CleanupTypeMemory)
			}
		}
	}
	
	// Check CPU pressure
	cpuPressure := cpuSample.Usage > rm.config.CPUThreshold
	if cpuPressure != (atomic.LoadInt32(&rm.cpuPressure) == 1) {
		atomic.StoreInt32(&rm.cpuPressure, boolToInt32(cpuPressure))
		
		if cpuPressure {
			rm.handleAlert(&Alert{
				Type:           AlertCPUPressure,
				Message:        "CPU pressure detected",
				Timestamp:      time.Now(),
				Severity:       SeverityWarning,
				CurrentValue:   cpuSample.Usage,
				ThresholdValue: rm.config.CPUThreshold,
			})
		}
	}
	
	// Check goroutine pressure
	goroutinePressure := goroutineSample.Count > rm.config.GoroutineThreshold
	if goroutinePressure != (atomic.LoadInt32(&rm.goroutinePressure) == 1) {
		atomic.StoreInt32(&rm.goroutinePressure, boolToInt32(goroutinePressure))
		
		if goroutinePressure {
			rm.handleAlert(&Alert{
				Type:           AlertGoroutinePressure,
				Message:        "Goroutine pressure detected",
				Timestamp:      time.Now(),
				Severity:       SeverityWarning,
				CurrentValue:   goroutineSample.Count,
				ThresholdValue: rm.config.GoroutineThreshold,
			})
			
			if rm.config.EnableAutoCleanup {
				rm.executeCleanup(CleanupTypeGoroutine)
			}
		}
	}
	
	// Check hard limits
	if rm.config.MemoryLimit > 0 && memorySample.HeapInUse+memorySample.StackInUse > rm.config.MemoryLimit {
		rm.handleAlert(&Alert{
			Type:           AlertMemoryLimit,
			Message:        "Memory limit exceeded",
			Timestamp:      time.Now(),
			Severity:       SeverityCritical,
			CurrentValue:   memorySample.HeapInUse + memorySample.StackInUse,
			ThresholdValue: rm.config.MemoryLimit,
		})
	}
	
	if rm.config.GoroutineLimit > 0 && goroutineSample.Count > rm.config.GoroutineLimit {
		rm.handleAlert(&Alert{
			Type:           AlertGoroutineLimit,
			Message:        "Goroutine limit exceeded",
			Timestamp:      time.Now(),
			Severity:       SeverityCritical,
			CurrentValue:   goroutineSample.Count,
			ThresholdValue: rm.config.GoroutineLimit,
		})
	}
}

func (rm *ResourceMonitor) updatePeaks(memoryUsage int64, cpuUsage float64, goroutineCount int, heapSize int64) {
	rm.statsMu.Lock()
	defer rm.statsMu.Unlock()
	
	if memoryUsage > rm.stats.PeakMemoryUsage {
		rm.stats.PeakMemoryUsage = memoryUsage
	}
	
	if cpuUsage > rm.stats.PeakCPUUsage {
		rm.stats.PeakCPUUsage = cpuUsage
	}
	
	if goroutineCount > rm.stats.PeakGoroutineCount {
		rm.stats.PeakGoroutineCount = goroutineCount
	}
	
	if heapSize > rm.stats.PeakHeapSize {
		rm.stats.PeakHeapSize = heapSize
	}
}

func (rm *ResourceMonitor) handleGCMetrics(memStats *runtime.MemStats) {
	rm.statsMu.Lock()
	defer rm.statsMu.Unlock()
	
	// Calculate last GC pause
	var lastGCPause time.Duration
	if memStats.NumGC > 0 {
		lastGCPause = time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
	}
	
	rm.stats.LastGCPause = lastGCPause
	rm.stats.TotalGCPauses = int64(memStats.NumGC)
	
	// Calculate average GC pause
	if memStats.NumGC > 0 {
		rm.stats.AverageGCPause = time.Duration(memStats.PauseTotalNs / uint64(memStats.NumGC))
	}
	
	// Check GC pause threshold
	if rm.config.GCPauseThreshold > 0 && lastGCPause > rm.config.GCPauseThreshold {
		rm.handleAlert(&Alert{
			Type:           AlertGCPause,
			Message:        "GC pause threshold exceeded",
			Timestamp:      time.Now(),
			Severity:       SeverityWarning,
			CurrentValue:   lastGCPause,
			ThresholdValue: rm.config.GCPauseThreshold,
		})
	}
}

func (rm *ResourceMonitor) handleAlert(alert *Alert) {
	if !rm.config.EnableAlerts {
		return
	}
	
	// Update alert statistics
	rm.statsMu.Lock()
	switch alert.Type {
	case AlertMemoryPressure, AlertMemoryLimit:
		rm.stats.MemoryAlerts++
	case AlertCPUPressure:
		rm.stats.CPUAlerts++
	case AlertGoroutinePressure, AlertGoroutineLimit:
		rm.stats.GoroutineAlerts++
	}
	rm.statsMu.Unlock()
	
	// Send to alert handlers
	rm.alertMu.RLock()
	handlers := make([]AlertHandler, len(rm.alertHandlers))
	copy(handlers, rm.alertHandlers)
	rm.alertMu.RUnlock()
	
	for _, handler := range handlers {
		go func(h AlertHandler) {
			if err := h.HandleAlert(alert); err != nil {
				// Log error (in a real implementation)
			}
		}(handler)
	}
}

// CleanupType represents the type of cleanup to perform
type CleanupType int

const (
	CleanupTypeMemory CleanupType = iota
	CleanupTypeGoroutine
)

func (rm *ResourceMonitor) executeCleanup(cleanupType CleanupType) {
	rm.cleanupMu.RLock()
	handlers := make([]CleanupHandler, len(rm.cleanupHandlers))
	copy(handlers, rm.cleanupHandlers)
	rm.cleanupMu.RUnlock()
	
	for _, handler := range handlers {
		var err error
		switch cleanupType {
		case CleanupTypeMemory:
			err = handler.HandleMemoryPressure()
		case CleanupTypeGoroutine:
			err = handler.HandleGoroutinePressure()
		}
		
		if err != nil {
			// Log error (in a real implementation)
		}
	}
	
	// Update cleanup statistics
	rm.statsMu.Lock()
	rm.stats.CleanupExecutions++
	rm.stats.LastCleanup = time.Now()
	rm.statsMu.Unlock()
}

// Utility functions

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// DefaultResourceMonitorConfig returns a default configuration
func DefaultResourceMonitorConfig() *ResourceMonitorConfig {
	return &ResourceMonitorConfig{
		SampleInterval:     5 * time.Second,
		HistoryDuration:    1 * time.Hour,
		MemoryThreshold:    0.8,
		MemoryLimit:        1024 * 1024 * 1024, // 1GB
		HeapThreshold:      0.7,
		CPUThreshold:       0.8,
		GoroutineThreshold: 10000,
		GoroutineLimit:     50000,
		GCPauseThreshold:   100 * time.Millisecond,
		EnableAlerts:       true,
		AlertCooldown:      5 * time.Minute,
		EnableAutoCleanup:  true,
		CleanupThreshold:   0.9,
	}
}