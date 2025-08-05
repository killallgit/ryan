package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLockConfig(t *testing.T) {
	config := DefaultLockConfig()
	
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 100*time.Millisecond, config.RetryDelay)
}

func TestNewFileLock(t *testing.T) {
	testPath := "/tmp/test-file.txt"
	lock := NewFileLock(testPath)
	
	assert.Equal(t, testPath, lock.path)
	assert.Equal(t, testPath+".lock", lock.lockPath)
	assert.False(t, lock.locked)
	assert.Nil(t, lock.file)
}

func TestFileLock_BasicLockUnlock(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	lock := NewFileLock(testPath)
	config := DefaultLockConfig()
	config.Timeout = 1 * time.Second // Shorter timeout for tests
	
	// Initially not locked
	assert.False(t, lock.IsLocked())
	
	// Lock should succeed
	err := lock.Lock(config)
	require.NoError(t, err)
	assert.True(t, lock.IsLocked())
	
	// Lock file should exist
	_, err = os.Stat(lock.lockPath)
	assert.NoError(t, err)
	
	// Unlock should succeed
	err = lock.Unlock()
	require.NoError(t, err)
	assert.False(t, lock.IsLocked())
	
	// Lock file should be removed
	_, err = os.Stat(lock.lockPath)
	assert.True(t, os.IsNotExist(err))
}

func TestFileLock_DoubleLock(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	lock := NewFileLock(testPath)
	config := DefaultLockConfig()
	config.Timeout = 100 * time.Millisecond
	
	// First lock should succeed
	err := lock.Lock(config)
	require.NoError(t, err)
	
	// Second lock on same instance should fail
	err = lock.Lock(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already locked")
	
	// Clean up
	lock.Unlock()
}

func TestFileLock_ConcurrentLock(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	lock1 := NewFileLock(testPath)
	lock2 := NewFileLock(testPath)
	
	config := DefaultLockConfig()
	config.Timeout = 500 * time.Millisecond
	
	// First lock should succeed
	err := lock1.Lock(config)
	require.NoError(t, err)
	
	// Second lock should timeout
	start := time.Now()
	err = lock2.Lock(config)
	duration := time.Since(start)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.GreaterOrEqual(t, duration, config.Timeout)
	
	// Clean up
	lock1.Unlock()
}

func TestFileLock_IsProcessRunning(t *testing.T) {
	lock := &FileLock{}
	
	// Test with current process (should be running)
	currentPID := os.Getpid()
	assert.True(t, lock.isProcessRunning(currentPID))
	
	// Test with non-existent process (high PID unlikely to exist)
	assert.False(t, lock.isProcessRunning(999999))
}

func TestCopyFileBytes(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")
	
	testData := []byte("test file content")
	
	// Create source file
	err := os.WriteFile(srcPath, testData, 0600)
	require.NoError(t, err)
	
	// Copy file
	err = copyFileBytes(srcPath, dstPath)
	require.NoError(t, err)
	
	// Verify copy
	copiedData, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, testData, copiedData)
}

func TestCopyFileBytes_NonExistentSource(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")
	
	err := copyFileBytes(srcPath, dstPath)
	assert.Error(t, err)
}

func TestWithLock(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	config := DefaultLockConfig()
	config.Timeout = 1 * time.Second
	
	executed := false
	
	err := WithLock(testPath, config, func() error {
		executed = true
		// Verify lock file exists during execution
		_, err := os.Stat(testPath + ".lock")
		assert.NoError(t, err)
		return nil
	})
	
	require.NoError(t, err)
	assert.True(t, executed)
	
	// Verify lock file is cleaned up
	_, err = os.Stat(testPath + ".lock")
	assert.True(t, os.IsNotExist(err))
}

func TestWithLock_FunctionError(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	config := DefaultLockConfig()
	config.Timeout = 1 * time.Second
	
	testError := fmt.Errorf("test function error")
	
	err := WithLock(testPath, config, func() error {
		return testError
	})
	
	assert.Equal(t, testError, err)
	
	// Verify lock file is still cleaned up
	_, err = os.Stat(testPath + ".lock")
	assert.True(t, os.IsNotExist(err))
}

func TestAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "atomic-test.txt")
	
	testData := []byte("atomic write test data")
	
	err := AtomicWrite(testPath, testData, 0644)
	require.NoError(t, err)
	
	// Verify file was written
	writtenData, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, testData, writtenData)
	
	// Verify backup was created (should exist temporarily during write)
	// The backup should be cleaned up automatically
}

func TestAtomicWrite_WithExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "existing-file.txt")
	
	// Create existing file
	originalData := []byte("original data")
	err := os.WriteFile(testPath, originalData, 0644)
	require.NoError(t, err)
	
	// Write new data atomically
	newData := []byte("new atomic data")
	err = AtomicWrite(testPath, newData, 0644)
	require.NoError(t, err)
	
	// Verify new data was written
	writtenData, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, newData, writtenData)
	
	// Backup file may or may not exist depending on cleanup timing
	// The important thing is that the operation succeeded
}

func TestRecoverFromBackup(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	backupPath := testPath + ".backup"
	
	// Create backup file
	backupData := []byte("backup file content")
	err := os.WriteFile(backupPath, backupData, 0600)
	require.NoError(t, err)
	
	// Recover from backup
	err = RecoverFromBackup(testPath)
	require.NoError(t, err)
	
	// Verify file was restored
	restoredData, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, backupData, restoredData)
}

func TestRecoverFromBackup_NoBackup(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "nonexistent-file.txt")
	
	err := RecoverFromBackup(testPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup file found")
}

func TestFileLock_UnlockWhenNotLocked(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	lock := NewFileLock(testPath)
	
	// Unlock when not locked should not error
	err := lock.Unlock()
	assert.NoError(t, err)
}

func TestLockConfig_CustomValues(t *testing.T) {
	config := LockConfig{
		Timeout:    5 * time.Second,
		RetryDelay: 50 * time.Millisecond,
	}
	
	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.Equal(t, 50*time.Millisecond, config.RetryDelay)
}

func TestFileLock_LockInfo(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test-file.txt")
	
	lock := NewFileLock(testPath)
	config := DefaultLockConfig()
	config.Timeout = 1 * time.Second
	
	err := lock.Lock(config)
	require.NoError(t, err)
	defer lock.Unlock()
	
	// Read lock file content
	lockData, err := os.ReadFile(lock.lockPath)
	require.NoError(t, err)
	
	lockContent := string(lockData)
	assert.Contains(t, lockContent, fmt.Sprintf("pid:%d", os.Getpid()))
	assert.Contains(t, lockContent, "time:")
	assert.Contains(t, lockContent, fmt.Sprintf("path:%s", testPath))
}

func TestFileLock_StaleDetection(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "stale-test.txt")
	
	lock := NewFileLock(testPath)
	
	// Create a stale lock file with non-existent PID
	staleLockData := fmt.Sprintf("pid:%d\ntime:%s\npath:%s\n", 
		999999, // Non-existent PID
		time.Now().Add(-10*time.Minute).Format(time.RFC3339), // Old timestamp
		testPath)
	
	err := os.MkdirAll(filepath.Dir(lock.lockPath), 0700)
	require.NoError(t, err)
	
	err = os.WriteFile(lock.lockPath, []byte(staleLockData), 0600)
	require.NoError(t, err)
	
	// Set modification time to make it old
	oldTime := time.Now().Add(-10 * time.Minute)
	err = os.Chtimes(lock.lockPath, oldTime, oldTime)
	require.NoError(t, err)
	
	// Should detect as stale
	assert.True(t, lock.isLockStale())
}