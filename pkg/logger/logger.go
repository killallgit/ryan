package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/killallgit/ryan/pkg/config"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides a unified logging interface
type Logger struct {
	level       LogLevel
	logger      *log.Logger
	file        *os.File
	initialized bool
}

var defaultLogger *Logger

// Init initializes the logger with configuration from global config
func Init() error {
	if defaultLogger != nil && defaultLogger.initialized {
		return nil // Already initialized
	}

	settings := config.Get()
	level := parseLevel(settings.Logging.Level)
	logFile := settings.Logging.LogFile
	persist := settings.Logging.Persist

	logger, err := New(level, logFile, persist)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	defaultLogger = logger
	return nil
}

// New creates a new Logger instance
func New(level LogLevel, logFile string, persist bool) (*Logger, error) {
	// Handle log file path resolution
	logPath := logFile
	if !filepath.IsAbs(logPath) {
		// If path is relative, make it relative to settings directory
		logFilename := filepath.Base(logPath)
		logPath = config.BuildSettingsPath(logFilename)
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Handle file clearing/creation based on persist flag
	var file *os.File
	var err error
	if persist {
		// Append to existing file
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		// Truncate existing file (clear it)
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create the Go logger with our file as output
	goLogger := log.New(file, "", log.LstdFlags)

	return &Logger{
		level:       level,
		logger:      goLogger,
		file:        file,
		initialized: true,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// parseLevel converts a string level to LogLevel
func parseLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// log writes a log message if the level is appropriate
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s", level.String(), message)

	// Also write to stderr for errors and fatal messages
	if level >= LevelError {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", level.String(), message)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// Package-level convenience functions using the default logger

// Debug logs a debug message using the default logger
func Debug(format string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Debug(format, args...)
}

// Info logs an info message using the default logger
func Info(format string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger
func Warn(format string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger
func Error(format string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Error(format, args...)
}

// Fatal logs a fatal message and exits using the default logger
func Fatal(format string, args ...interface{}) {
	if defaultLogger == nil {
		fmt.Fprintf(os.Stderr, "[FATAL] "+format+"\n", args...)
		os.Exit(1)
	}
	defaultLogger.Fatal(format, args...)
}

// SetOutput sets the output writer for the logger (useful for testing)
func SetOutput(w io.Writer) {
	if defaultLogger != nil && defaultLogger.logger != nil {
		defaultLogger.logger.SetOutput(w)
	}
}

// Close closes the default logger
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}
