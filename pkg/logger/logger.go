package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	*slog.Logger
	errorLogger *slog.Logger
}

var defaultLogger *Logger
var errorFile *os.File
var stdoutFile *os.File

func init() {
	defaultLogger = &Logger{slog.Default(), slog.Default()}
}

func InitLogger() error {
	// For now, use defaults until we refactor to pass config
	return InitLoggerWithConfig(".ryan/debug.log", false, "info")
}

func InitLoggerWithConfig(logFile string, preserve bool, level string) error {
	if logFile == "" {
		logFile = ".ryan/debug.log"
	}

	// Ensure directory exists
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open stdout log file (always truncated)
	stdoutPath := filepath.Join(dir, "stdout.log")
	var err error
	stdoutFile, err = os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open stdout log file: %w", err)
	}

	// Open error log file
	errorPath := filepath.Join(dir, "error.log")
	flag := os.O_CREATE | os.O_WRONLY
	if preserve {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	errorFile, err = os.OpenFile(errorPath, flag, 0644)
	if err != nil {
		stdoutFile.Close()
		return fmt.Errorf("failed to open error log file: %w", err)
	}

	// Parse log level
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelDebug
	}

	// Create structured logger with timestamp and component
	opts := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(time.Now().Format("2006-01-02 15:04:05.000")),
				}
			}
			return a
		},
	}

	// Error logger options (only errors)
	errorOpts := &slog.HandlerOptions{
		Level:       slog.LevelError,
		ReplaceAttr: opts.ReplaceAttr,
	}

	// Create multi-writer for stdout log
	multiWriter := io.MultiWriter(stdoutFile)

	// Create handlers
	stdoutHandler := slog.NewTextHandler(multiWriter, opts)
	errorHandler := slog.NewTextHandler(errorFile, errorOpts)

	logger := slog.New(stdoutHandler)
	errLogger := slog.New(errorHandler)

	defaultLogger = &Logger{logger, errLogger}

	// Log initialization
	defaultLogger.Info("Logger initialized",
		"stdout", stdoutPath,
		"error", errorPath,
		"preserve", preserve,
		"level", level,
	)

	return nil
}

func Get() *Logger {
	return defaultLogger
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		l.Logger.With("component", component),
		l.errorLogger.With("component", component),
	}
}

func (l *Logger) WithContext(key string, value any) *Logger {
	return &Logger{
		l.Logger.With(key, value),
		l.errorLogger.With(key, value),
	}
}

// Override Error method to write to both loggers
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	l.errorLogger.Error(msg, args...)
}

// Convenience functions
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	if defaultLogger.errorLogger != nil {
		defaultLogger.errorLogger.Error(msg, args...)
	}
	defaultLogger.Logger.Error(msg, args...)
}

func WithComponent(component string) *Logger {
	return defaultLogger.WithComponent(component)
}

func WithContext(key string, value any) *Logger {
	return defaultLogger.WithContext(key, value)
}

// Close closes the log files
func Close() error {
	var errs []error

	if stdoutFile != nil {
		if err := stdoutFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdout log: %w", err))
		}
	}

	if errorFile != nil {
		if err := errorFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close error log: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}

	return nil
}
