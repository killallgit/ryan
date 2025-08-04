package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	*slog.Logger
}

var defaultLogger *Logger
var logFile *os.File

func init() {
	defaultLogger = &Logger{slog.Default()}
}

func InitLogger() error {
	// For now, use defaults until we refactor to pass config
	return InitLoggerWithConfig(".ryan/app.log", false, "info")
}

func InitLoggerWithConfig(logFilePath string, preserve bool, level string) error {
	if logFilePath == "" {
		logFilePath = ".ryan/app.log"
	}

	// Ensure directory exists
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open single consolidated log file
	flag := os.O_CREATE | os.O_WRONLY
	if preserve {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	var err error
	logFile, err = os.OpenFile(logFilePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
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

	// Create handler for single log file
	handler := slog.NewTextHandler(logFile, opts)
	logger := slog.New(handler)

	defaultLogger = &Logger{logger}

	// Log initialization
	defaultLogger.Info("Logger initialized",
		"file", logFilePath,
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
	}
}

func (l *Logger) WithContext(key string, value any) *Logger {
	return &Logger{
		l.Logger.With(key, value),
	}
}

// Error method - now just writes to the single log file
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
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
	defaultLogger.Logger.Error(msg, args...)
}

func WithComponent(component string) *Logger {
	return defaultLogger.WithComponent(component)
}

func WithContext(key string, value any) *Logger {
	return defaultLogger.WithContext(key, value)
}

// Close closes the log file
func Close() error {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}
	return nil
}
