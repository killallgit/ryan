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
var historyFile *os.File

func init() {
	defaultLogger = &Logger{slog.Default(), slog.Default()}
}

func InitLogger() error {
	// For now, use defaults until we refactor to pass config
	return InitLoggerWithConfig(".ryan/logs/debug.log", false, "info")
}

func InitLoggerWithConfig(logFile string, preserve bool, level string) error {
	if logFile == "" {
		logFile = ".ryan/logs/debug.log"
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

// InitHistoryFile initializes the chat history file (always overwrites)
func InitHistoryFile(historyPath string) error {
	if historyPath == "" {
		historyPath = ".ryan/logs/debug.history"
	}

	// Ensure directory exists
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Always truncate the history file on startup
	var err error
	historyFile, err = os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}

	// Write session start marker
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := fmt.Fprintf(historyFile, "=== Ryan Chat Session Started: %s ===\n\n", timestamp); err != nil {
		return fmt.Errorf("failed to write session start: %w", err)
	}

	return nil
}

// LogChatHistory logs a chat interaction to the history file
func LogChatHistory(role, content string) error {
	if historyFile == nil {
		return fmt.Errorf("history file not initialized")
	}

	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s: %s\n\n", timestamp, role, content)
	
	if _, err := historyFile.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to history: %w", err)
	}

	// Flush to ensure it's written immediately
	historyFile.Sync()

	return nil
}

// LogChatEvent logs a general chat event (like tool execution, errors, etc.)
func LogChatEvent(event, details string) error {
	if historyFile == nil {
		return fmt.Errorf("history file not initialized")
	}

	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] EVENT: %s - %s\n\n", timestamp, event, details)
	
	if _, err := historyFile.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write event to history: %w", err)
	}

	// Flush to ensure it's written immediately
	historyFile.Sync()

	return nil
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

	if historyFile != nil {
		// Write session end marker before closing
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(historyFile, "=== Ryan Chat Session Ended: %s ===\n", timestamp)
		
		if err := historyFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close history file: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}

	return nil
}
