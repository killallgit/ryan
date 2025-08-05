package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// FileLock provides cross-platform file locking capabilities
// This implements the file locking mechanism used by Claude CLI
type FileLock struct {
	path     string
	lockPath string
	file     *os.File
	locked   bool
}

// LockConfig holds configuration for file locking behavior
type LockConfig struct {
	Timeout    time.Duration
	RetryDelay time.Duration
}

// DefaultLockConfig returns sensible defaults for file locking
func DefaultLockConfig() LockConfig {
	return LockConfig{
		Timeout:    30 * time.Second,       // Maximum time to wait for lock
		RetryDelay: 100 * time.Millisecond, // Delay between lock attempts
	}
}

// NewFileLock creates a new file lock for the given path
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path:     path,
		lockPath: path + ".lock",
		locked:   false,
	}
}

// Lock acquires an exclusive lock on the file with timeout and retry logic
func (fl *FileLock) Lock(config LockConfig) error {
	if fl.locked {
		return errors.New("file is already locked")
	}

	// Create lock directory if it doesn't exist
	lockDir := filepath.Dir(fl.lockPath)
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	start := time.Now()

	for {
		// Check timeout
		if time.Since(start) > config.Timeout {
			return fmt.Errorf("timeout acquiring lock on %s after %v", fl.path, config.Timeout)
		}

		// Try to acquire lock
		if err := fl.tryLock(); err == nil {
			fl.locked = true
			return nil
		}

		// Wait before retrying
		time.Sleep(config.RetryDelay)
	}
}

// tryLock attempts to acquire the lock without blocking
func (fl *FileLock) tryLock() error {
	// Try to create the lock file exclusively
	file, err := os.OpenFile(fl.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Check if the lock is stale
			if fl.isLockStale() {
				// Remove stale lock and try again
				os.Remove(fl.lockPath)
				return fl.tryLock()
			}
			return fmt.Errorf("lock already exists")
		}
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	// Apply additional file locking at the OS level for extra safety
	if err := fl.applySystemLock(file); err != nil {
		file.Close()
		os.Remove(fl.lockPath)
		return fmt.Errorf("failed to apply system lock: %w", err)
	}

	// Write lock information
	lockInfo := fmt.Sprintf("pid:%d\ntime:%s\npath:%s\n",
		os.Getpid(),
		time.Now().Format(time.RFC3339),
		fl.path)

	if _, err := file.WriteString(lockInfo); err != nil {
		file.Close()
		os.Remove(fl.lockPath)
		return fmt.Errorf("failed to write lock info: %w", err)
	}

	fl.file = file
	return nil
}

// applySystemLock applies OS-level file locking
func (fl *FileLock) applySystemLock(file *os.File) error {
	// Use flock on Unix-like systems
	fd := int(file.Fd())

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("file is locked by another process")
		}
		return fmt.Errorf("failed to acquire system lock: %w", err)
	}

	return nil
}

// isLockStale checks if a lock file is stale (from a dead process)
func (fl *FileLock) isLockStale() bool {
	// Check if lock file is too old (more than 5 minutes)
	info, err := os.Stat(fl.lockPath)
	if err != nil {
		return true // If we can't stat it, consider it stale
	}

	// Consider locks older than 5 minutes as potentially stale
	if time.Since(info.ModTime()) > 5*time.Minute {
		// Try to read the PID from the lock file
		data, err := os.ReadFile(fl.lockPath)
		if err != nil {
			return true
		}

		// Parse PID from lock info
		var pid int
		if _, err := fmt.Sscanf(string(data), "pid:%d", &pid); err != nil {
			return true
		}

		// Check if process is still running
		if !fl.isProcessRunning(pid) {
			return true
		}
	}

	return false
}

// isProcessRunning checks if a process with the given PID is still running
func (fl *FileLock) isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if !fl.locked {
		return nil // Already unlocked
	}

	var lastErr error

	// Release system lock
	if fl.file != nil {
		fd := int(fl.file.Fd())
		if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
			lastErr = fmt.Errorf("failed to release system lock: %w", err)
		}

		// Close file
		if err := fl.file.Close(); err != nil && lastErr == nil {
			lastErr = fmt.Errorf("failed to close lock file: %w", err)
		}
		fl.file = nil
	}

	// Remove lock file
	if err := os.Remove(fl.lockPath); err != nil && lastErr == nil {
		lastErr = fmt.Errorf("failed to remove lock file: %w", err)
	}

	fl.locked = false
	return lastErr
}

// IsLocked returns whether the file is currently locked
func (fl *FileLock) IsLocked() bool {
	return fl.locked
}

// WithLock executes a function while holding a file lock
func WithLock(path string, config LockConfig, fn func() error) error {
	lock := NewFileLock(path)

	if err := lock.Lock(config); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	defer func() {
		if unlockErr := lock.Unlock(); unlockErr != nil {
			// Log the unlock error, but don't override the main error
			fmt.Fprintf(os.Stderr, "Warning: failed to unlock file %s: %v\n", path, unlockErr)
		}
	}()

	return fn()
}

// AtomicWrite performs an atomic write operation with file locking
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	config := DefaultLockConfig()

	return WithLock(path, config, func() error {
		// Create backup if original file exists
		backupPath := path + ".backup"
		if _, err := os.Stat(path); err == nil {
			if err := copyFileBytes(path, backupPath); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}

		// Write to temporary file first
		tempPath := path + ".tmp"
		if err := os.WriteFile(tempPath, data, perm); err != nil {
			return fmt.Errorf("failed to write temporary file: %w", err)
		}

		// Atomic rename
		if err := os.Rename(tempPath, path); err != nil {
			// Clean up temp file on error
			os.Remove(tempPath)
			return fmt.Errorf("failed to rename temporary file: %w", err)
		}

		return nil
	})
}

// copyFileBytes copies file contents from src to dst
func copyFileBytes(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// RecoverFromBackup attempts to recover a file from its backup
func RecoverFromBackup(path string) error {
	backupPath := path + ".backup"

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup file found at %s", backupPath)
	}

	// Use atomic write to restore from backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	return AtomicWrite(path, data, 0600)
}
