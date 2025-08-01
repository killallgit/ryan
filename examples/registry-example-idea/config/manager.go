package config

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	
	"your-project/pkg/registry/collections"
	"your-project/pkg/registry/errors"
)

// ConfigManager handles atomic configuration updates with versioning and rollback
type ConfigManager struct {
	// Current configuration (atomic access)
	current atomic.Value // *ConfigVersion
	
	// Version history with bounded storage
	versions *collections.RingBuffer[*ConfigVersion]
	
	// Validation and watching
	validators []ConfigValidator
	watchers   []ConfigWatcher
	
	// File watching for hot reload
	fileWatcher *fsnotify.Watcher
	debouncer   *Debouncer
	watchedFile string
	
	// Lifecycle management
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	mu      sync.RWMutex
	
	// Statistics
	stats ConfigStats
	statsMu sync.RWMutex
}

// ConfigVersion represents a versioned configuration with metadata
type ConfigVersion struct {
	Config     *RegistryConfig `json:"config"`
	Version    int64           `json:"version"`
	Timestamp  time.Time       `json:"timestamp"`
	Checksum   string          `json:"checksum"`
	Source     ConfigSource    `json:"source"`
	Valid      bool            `json:"valid"`
	
	// Rollback information
	RollbackReason string        `json:"rollback_reason,omitempty"`
	PreviousVersion int64        `json:"previous_version,omitempty"`
}

// ConfigSource indicates where the configuration came from
type ConfigSource string

const (
	SourceFile     ConfigSource = "file"
	SourceAPI      ConfigSource = "api"
	SourceDefault  ConfigSource = "default"
	SourceRollback ConfigSource = "rollback"
)

// RegistryConfig holds all registry configuration
type RegistryConfig struct {
	// Core settings
	MaxConcurrentTools int           `yaml:"max_concurrent_tools" json:"max_concurrent_tools"`
	DefaultTimeout     time.Duration `yaml:"default_timeout" json:"default_timeout"`
	EnableAnalytics    bool          `yaml:"enable_analytics" json:"enable_analytics"`
	
	// Plugin settings
	PluginDirectories []string      `yaml:"plugin_directories" json:"plugin_directories"`
	AutoLoadPlugins   bool          `yaml:"auto_load_plugins" json:"auto_load_plugins"`
	HotReload         bool          `yaml:"hot_reload" json:"hot_reload"`
	
	// Security settings
	StrictValidation   bool          `yaml:"strict_validation" json:"strict_validation"`
	RequirePermissions bool          `yaml:"require_permissions" json:"require_permissions"`
	AuditLogging       bool          `yaml:"audit_logging" json:"audit_logging"`
	
	// Performance settings
	CacheResults      bool          `yaml:"cache_results" json:"cache_results"`
	CacheTTL          time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	MetricsInterval   time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
	
	// Memory management
	MaxMemoryUsage    int64         `yaml:"max_memory_usage" json:"max_memory_usage"`
	EvictionPolicy    string        `yaml:"eviction_policy" json:"eviction_policy"`
	
	// Concurrency settings
	WorkerPoolSize    int           `yaml:"worker_pool_size" json:"worker_pool_size"`
	QueueSize         int           `yaml:"queue_size" json:"queue_size"`
	
	// Network settings
	ListenAddress     string        `yaml:"listen_address" json:"listen_address"`
	TLSEnabled        bool          `yaml:"tls_enabled" json:"tls_enabled"`
	CertFile          string        `yaml:"cert_file" json:"cert_file"`
	KeyFile           string        `yaml:"key_file" json:"key_file"`
}

// ConfigValidator validates configuration changes
type ConfigValidator interface {
	ValidateConfig(config *RegistryConfig) error
	GetValidatorName() string
}

// ConfigWatcher is notified of configuration changes
type ConfigWatcher interface {
	OnConfigChanged(oldConfig, newConfig *RegistryConfig) error
	GetWatcherName() string
}

// ConfigStats tracks configuration management statistics
type ConfigStats struct {
	Updates          int64     `json:"updates"`
	Rollbacks        int64     `json:"rollbacks"`
	ValidationErrors int64     `json:"validation_errors"`
	LastUpdate       time.Time `json:"last_update"`
	LastRollback     time.Time `json:"last_rollback"`
	WatcherErrors    int64     `json:"watcher_errors"`
}

// Debouncer prevents rapid-fire configuration reloads
type Debouncer struct {
	delay    time.Duration
	timer    *time.Timer
	callback func()
	mu       sync.Mutex
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(ctx context.Context) *ConfigManager {
	ctx, cancel := context.WithCancel(ctx)
	
	// Create version history buffer (keep last 10 versions)
	versions := collections.NewRingBuffer[*ConfigVersion](collections.RingBufferConfig[*ConfigVersion]{
		Capacity: 10,
	})
	
	cm := &ConfigManager{
		versions:   versions,
		validators: make([]ConfigValidator, 0),
		watchers:   make([]ConfigWatcher, 0),
		ctx:        ctx,
		cancel:     cancel,
		debouncer:  NewDebouncer(500*time.Millisecond), // 500ms debounce
	}
	
	// Set default configuration
	defaultConfig := DefaultRegistryConfig()
	cm.setCurrentConfig(defaultConfig, SourceDefault)
	
	return cm
}

// Start initializes the configuration manager
func (cm *ConfigManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.started {
		return errors.NewError(errors.ErrConfigInvalid, "config manager already started").Build()
	}
	
	// Initialize file watcher if needed
	if cm.watchedFile != "" {
		if err := cm.initFileWatcher(); err != nil {
			return errors.NewError(errors.ErrConfigInvalid, "failed to initialize file watcher").
				Cause(err).
				Context("file", cm.watchedFile).
				Build()
		}
	}
	
	cm.started = true
	return nil
}

// Stop shuts down the configuration manager
func (cm *ConfigManager) Stop() error {
	cm.cancel()
	
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.fileWatcher != nil {
		cm.fileWatcher.Close()
	}
	
	cm.started = false
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *RegistryConfig {
	version := cm.current.Load().(*ConfigVersion)
	
	// Return a deep copy to prevent external modification
	configCopy := *version.Config
	return &configCopy
}

// GetConfigVersion returns the current configuration version
func (cm *ConfigManager) GetConfigVersion() *ConfigVersion {
	version := cm.current.Load().(*ConfigVersion)
	
	// Return a copy
	versionCopy := *version
	configCopy := *version.Config
	versionCopy.Config = &configCopy
	
	return &versionCopy
}

// UpdateConfig atomically updates the configuration
func (cm *ConfigManager) UpdateConfig(newConfig *RegistryConfig, source ConfigSource) error {
	// Validate the new configuration
	if err := cm.validateConfig(newConfig); err != nil {
		cm.statsMu.Lock()
		cm.stats.ValidationErrors++
		cm.statsMu.Unlock()
		
		return errors.NewError(errors.ErrConfigInvalid, "configuration validation failed").
			Cause(err).
			Context("source", source).
			Build()
	}
	
	// Get current version for rollback
	currentVersion := cm.current.Load().(*ConfigVersion)
	
	// Create new version
	newVersion := &ConfigVersion{
		Config:    newConfig,
		Version:   time.Now().UnixNano(),
		Timestamp: time.Now(),
		Checksum:  cm.calculateChecksum(newConfig),
		Source:    source,
		Valid:     true,
	}
	
	// Store the new configuration atomically
	cm.current.Store(newVersion)
	
	// Add to version history
	cm.versions.Push(newVersion)
	
	// Notify watchers (this might fail and trigger rollback)
	if err := cm.notifyWatchers(currentVersion.Config, newConfig); err != nil {
		// Rollback on watcher failure
		cm.rollbackToVersion(currentVersion, fmt.Sprintf("watcher failure: %v", err))
		
		return errors.NewError(errors.ErrConfigInvalid, "configuration update failed, rolled back").
			Cause(err).
			Context("source", source).
			Context("rolled_back_to_version", currentVersion.Version).
			Build()
	}
	
	// Update statistics
	cm.statsMu.Lock()
	cm.stats.Updates++
	cm.stats.LastUpdate = time.Now()
	cm.statsMu.Unlock()
	
	return nil
}

// LoadFromFile loads configuration from a file
func (cm *ConfigManager) LoadFromFile(filename string) error {
	data, err := fs.ReadFile(nil, filename) // Using os.ReadFile through fs interface
	if err != nil {
		return errors.NewError(errors.ErrConfigInvalid, "failed to read config file").
			Cause(err).
			Context("filename", filename).
			Build()
	}
	
	var config RegistryConfig
	
	// Determine file format by extension
	switch filepath.Ext(filename) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return errors.NewError(errors.ErrConfigInvalid, "failed to parse YAML config").
				Cause(err).
				Context("filename", filename).
				Build()
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return errors.NewError(errors.ErrConfigInvalid, "failed to parse JSON config").
				Cause(err).
				Context("filename", filename).
				Build()
		}
	default:
		return errors.NewError(errors.ErrConfigInvalid, "unsupported config file format").
			Context("filename", filename).
			Context("supported_formats", []string{".yaml", ".yml", ".json"}).
			Build()
	}
	
	return cm.UpdateConfig(&config, SourceFile)
}

// WatchFile sets up file watching for automatic reloading
func (cm *ConfigManager) WatchFile(filename string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.watchedFile = filename
	
	if cm.started {
		return cm.initFileWatcher()
	}
	
	return nil
}

// RollbackToPrevious rolls back to the previous configuration version
func (cm *ConfigManager) RollbackToPrevious(reason string) error {
	// Get version history
	versions := cm.versions.GetAll()
	if len(versions) < 2 {
		return errors.NewError(errors.ErrConfigInvalid, "no previous version available for rollback").
			Context("reason", reason).
			Build()
	}
	
	// Get the previous version (second most recent)
	previousVersion := versions[len(versions)-2]
	
	return cm.rollbackToVersion(previousVersion, reason)
}

// RollbackToVersion rolls back to a specific configuration version
func (cm *ConfigManager) RollbackToVersion(version int64, reason string) error {
	// Find the target version
	versions := cm.versions.GetAll()
	var targetVersion *ConfigVersion
	
	for _, v := range versions {
		if v.Version == version {
			targetVersion = v
			break
		}
	}
	
	if targetVersion == nil {
		return errors.NewError(errors.ErrConfigInvalid, "target version not found").
			Context("target_version", version).
			Context("reason", reason).
			Build()
	}
	
	return cm.rollbackToVersion(targetVersion, reason)
}

// GetVersionHistory returns the configuration version history
func (cm *ConfigManager) GetVersionHistory() []*ConfigVersion {
	versions := cm.versions.GetAll()
	
	// Return copies to prevent external modification
	result := make([]*ConfigVersion, len(versions))
	for i, v := range versions {
		versionCopy := *v
		configCopy := *v.Config
		versionCopy.Config = &configCopy
		result[i] = &versionCopy
	}
	
	return result
}

// AddValidator adds a configuration validator
func (cm *ConfigManager) AddValidator(validator ConfigValidator) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.validators = append(cm.validators, validator)
}

// AddWatcher adds a configuration watcher
func (cm *ConfigManager) AddWatcher(watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.watchers = append(cm.watchers, watcher)
}

// GetStats returns configuration management statistics
func (cm *ConfigManager) GetStats() ConfigStats {
	cm.statsMu.RLock()
	defer cm.statsMu.RUnlock()
	
	return cm.stats
}

// Private methods

func (cm *ConfigManager) setCurrentConfig(config *RegistryConfig, source ConfigSource) {
	version := &ConfigVersion{
		Config:    config,
		Version:   time.Now().UnixNano(),
		Timestamp: time.Now(),
		Checksum:  cm.calculateChecksum(config),
		Source:    source,
		Valid:     true,
	}
	
	cm.current.Store(version)
	cm.versions.Push(version)
}

func (cm *ConfigManager) validateConfig(config *RegistryConfig) error {
	// Basic validation
	if config.MaxConcurrentTools <= 0 {
		return fmt.Errorf("max_concurrent_tools must be positive")
	}
	
	if config.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive")
	}
	
	if config.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker_pool_size must be positive")
	}
	
	// Run custom validators
	for _, validator := range cm.validators {
		if err := validator.ValidateConfig(config); err != nil {
			return fmt.Errorf("validator %s failed: %w", validator.GetValidatorName(), err)
		}
	}
	
	return nil
}

func (cm *ConfigManager) notifyWatchers(oldConfig, newConfig *RegistryConfig) error {
	var watcherErrors []error
	
	for _, watcher := range cm.watchers {
		if err := watcher.OnConfigChanged(oldConfig, newConfig); err != nil {
			watcherErrors = append(watcherErrors, fmt.Errorf("watcher %s failed: %w", watcher.GetWatcherName(), err))
			
			cm.statsMu.Lock()
			cm.stats.WatcherErrors++
			cm.statsMu.Unlock()
		}
	}
	
	if len(watcherErrors) > 0 {
		return fmt.Errorf("watcher errors: %v", watcherErrors)
	}
	
	return nil
}

func (cm *ConfigManager) rollbackToVersion(targetVersion *ConfigVersion, reason string) error {
	currentVersion := cm.current.Load().(*ConfigVersion)
	
	// Create rollback version
	rollbackVersion := &ConfigVersion{
		Config:          targetVersion.Config,
		Version:         time.Now().UnixNano(),
		Timestamp:       time.Now(),
		Checksum:        targetVersion.Checksum,
		Source:          SourceRollback,
		Valid:           true,
		RollbackReason:  reason,
		PreviousVersion: currentVersion.Version,
	}
	
	// Store the rollback configuration
	cm.current.Store(rollbackVersion)
	cm.versions.Push(rollbackVersion)
	
	// Notify watchers of rollback
	if err := cm.notifyWatchers(currentVersion.Config, targetVersion.Config); err != nil {
		// If rollback notification fails, we're in a bad state
		// Mark the configuration as invalid but keep it
		rollbackVersion.Valid = false
		return fmt.Errorf("rollback notification failed: %w", err)
	}
	
	// Update statistics
	cm.statsMu.Lock()
	cm.stats.Rollbacks++
	cm.stats.LastRollback = time.Now()
	cm.statsMu.Unlock()
	
	return nil
}

func (cm *ConfigManager) calculateChecksum(config *RegistryConfig) string {
	data, _ := json.Marshal(config)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (cm *ConfigManager) initFileWatcher() error {
	var err error
	cm.fileWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	
	// Add file to watcher
	if err := cm.fileWatcher.Add(cm.watchedFile); err != nil {
		cm.fileWatcher.Close()
		return err
	}
	
	// Start watching goroutine
	go cm.watchFileChanges()
	
	return nil
}

func (cm *ConfigManager) watchFileChanges() {
	for {
		select {
		case event, ok := <-cm.fileWatcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Debounce file changes to prevent rapid reloads
				cm.debouncer.Trigger(func() {
					if err := cm.LoadFromFile(cm.watchedFile); err != nil {
						// Log error but don't fail - keep current config
						// In a real implementation, you'd use a proper logger
						fmt.Printf("Failed to reload config from file: %v\n", err)
					}
				})
			}
			
		case err, ok := <-cm.fileWatcher.Errors:
			if !ok {
				return
			}
			
			fmt.Printf("File watcher error: %v\n", err)
			
		case <-cm.ctx.Done():
			return
		}
	}
}

// Debouncer implementation

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
	}
}

func (d *Debouncer) Trigger(callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.callback = callback
	
	if d.timer != nil {
		d.timer.Stop()
	}
	
	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		
		if d.callback != nil {
			d.callback()
		}
	})
}

// DefaultRegistryConfig returns a default configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		MaxConcurrentTools:  10,
		DefaultTimeout:      30 * time.Second,
		EnableAnalytics:     true,
		PluginDirectories:   []string{"./plugins"},
		AutoLoadPlugins:     true,
		HotReload:           false,
		StrictValidation:    true,
		RequirePermissions:  true,
		AuditLogging:        true,
		CacheResults:        false,
		CacheTTL:            5 * time.Minute,
		MetricsInterval:     1 * time.Minute,
		MaxMemoryUsage:      1024 * 1024 * 1024, // 1GB
		EvictionPolicy:      "lru",
		WorkerPoolSize:      5,
		QueueSize:           1000,
		ListenAddress:       ":8080",
		TLSEnabled:          false,
	}
}