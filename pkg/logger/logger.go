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

func init() {
	defaultLogger = &Logger{slog.Default()}
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

	// Open log file
	flag := os.O_CREATE | os.O_WRONLY
	if preserve {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(logFile, flag, 0644)
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

	// Only write to file for TUI applications to avoid interfering with display
	handler := slog.NewTextHandler(file, opts)

	logger := slog.New(handler)
	defaultLogger = &Logger{logger}

	// Log initialization
	defaultLogger.Info("Logger initialized",
		"file", logFile,
		"preserve", preserve,
		"level", level,
	)

	return nil
}

func Get() *Logger {
	return defaultLogger
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{l.Logger.With("component", component)}
}

func (l *Logger) WithContext(key string, value any) *Logger {
	return &Logger{l.Logger.With(key, value)}
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
	defaultLogger.Error(msg, args...)
}

func WithComponent(component string) *Logger {
	return defaultLogger.WithComponent(component)
}

func WithContext(key string, value any) *Logger {
	return defaultLogger.WithContext(key, value)
}
