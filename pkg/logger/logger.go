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
var historyFile *os.File

func init() {
	defaultLogger = &Logger{slog.Default()}
}

func InitLogger() error {
	// For now, use defaults until we refactor to pass config
	return InitLoggerWithConfig(".ryan/logs/debug.log", false, "info")
}

func InitLoggerWithConfig(logPath string, preserve bool, level string) error {
	if logPath == "" {
		logPath = ".ryan/logs/debug.log"
	}

	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open single log file (respects preserve setting)
	flag := os.O_CREATE | os.O_WRONLY
	if preserve {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	var err error
	logFile, err = os.OpenFile(logPath, flag, 0644)
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
		"file", logPath,
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

// InitHistoryFile initializes the chat history file
func InitHistoryFile(historyPath string, continueHistory bool) error {
	if historyPath == "" {
		historyPath = ".ryan/logs/debug.history"
	}

	// Ensure directory exists
	dir := filepath.Dir(historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Determine file open flags based on continueHistory
	var flags int
	if continueHistory {
		// Append to existing file
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	} else {
		// Truncate the history file on startup (default behavior)
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	var err error
	historyFile, err = os.OpenFile(historyPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}

	// Write session start marker
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	sessionMarker := fmt.Sprintf("\n=== Ryan Chat Session %s: %s ===\n\n",
		map[bool]string{true: "Continued", false: "Started"}[continueHistory],
		timestamp)

	if _, err := fmt.Fprint(historyFile, sessionMarker); err != nil {
		return fmt.Errorf("failed to write session marker: %w", err)
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

	if logFile != nil {
		if err := logFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close log file: %w", err))
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
