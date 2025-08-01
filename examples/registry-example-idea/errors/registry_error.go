package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ErrorCode represents a structured error code for the registry system
type ErrorCode string

const (
	// Tool management errors
	ErrToolNotFound      ErrorCode = "TOOL_NOT_FOUND"
	ErrToolAlreadyExists ErrorCode = "TOOL_ALREADY_EXISTS"
	ErrToolInvalid       ErrorCode = "TOOL_INVALID"
	ErrToolDisabled      ErrorCode = "TOOL_DISABLED"
	
	// Execution errors
	ErrExecutionTimeout  ErrorCode = "EXECUTION_TIMEOUT"
	ErrExecutionFailed   ErrorCode = "EXECUTION_FAILED"
	ErrInvalidParams     ErrorCode = "INVALID_PARAMETERS"
	ErrResourceLimit     ErrorCode = "RESOURCE_LIMIT_EXCEEDED"
	
	// Permission errors
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrInsufficientPerms ErrorCode = "INSUFFICIENT_PERMISSIONS"
	ErrAccessRestricted  ErrorCode = "ACCESS_RESTRICTED"
	
	// System errors
	ErrSystemOverload    ErrorCode = "SYSTEM_OVERLOAD"
	ErrConfigInvalid     ErrorCode = "CONFIG_INVALID"
	ErrPluginFailure     ErrorCode = "PLUGIN_FAILURE"
	ErrInternalError     ErrorCode = "INTERNAL_ERROR"
	
	// Registry errors
	ErrRegistryNotStarted ErrorCode = "REGISTRY_NOT_STARTED"
	ErrRegistryShutdown   ErrorCode = "REGISTRY_SHUTDOWN"
)

// RegistryError is a comprehensive error type with context and tracing
type RegistryError struct {
	Code        ErrorCode         `json:"code"`
	Message     string            `json:"message"`
	Cause       error             `json:"cause,omitempty"`
	Context     map[string]any    `json:"context,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	StackTrace  string            `json:"stack_trace,omitempty"`
	
	// Distributed tracing support
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	
	// Error chain for nested errors
	Chain       []*RegistryError  `json:"chain,omitempty"`
	
	// Retry information
	Retryable   bool              `json:"retryable"`
	RetryAfter  time.Duration     `json:"retry_after,omitempty"`
	
	// User-facing information
	UserMessage string            `json:"user_message,omitempty"`
	HelpURL     string            `json:"help_url,omitempty"`
}

// Error implements the error interface
func (re *RegistryError) Error() string {
	if re.UserMessage != "" {
		return re.UserMessage
	}
	
	if re.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", re.Code, re.Message, re.Cause)
	}
	
	return fmt.Sprintf("%s: %s", re.Code, re.Message)
}

// Unwrap supports error unwrapping for Go 1.13+
func (re *RegistryError) Unwrap() error {
	return re.Cause
}

// Is supports error comparison for Go 1.13+
func (re *RegistryError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if registryErr, ok := target.(*RegistryError); ok {
		return re.Code == registryErr.Code
	}
	
	return false
}

// WithContext adds context information to the error
func (re *RegistryError) WithContext(key string, value any) *RegistryError {
	if re.Context == nil {
		re.Context = make(map[string]any)
	}
	re.Context[key] = value
	return re
}

// WithTrace adds tracing information
func (re *RegistryError) WithTrace(traceID, spanID string) *RegistryError {
	re.TraceID = traceID
	re.SpanID = spanID
	return re
}

// WithCause wraps another error as the cause
func (re *RegistryError) WithCause(cause error) *RegistryError {
	re.Cause = cause
	return re
}

// WithRetry marks the error as retryable
func (re *RegistryError) WithRetry(retryAfter time.Duration) *RegistryError {
	re.Retryable = true
	re.RetryAfter = retryAfter
	return re
}

// WithUserMessage sets a user-friendly message
func (re *RegistryError) WithUserMessage(message string) *RegistryError {
	re.UserMessage = message
	return re
}

// WithHelpURL adds a help URL for more information
func (re *RegistryError) WithHelpURL(url string) *RegistryError {
	re.HelpURL = url
	return re
}

// MarshalJSON provides custom JSON marshaling
func (re *RegistryError) MarshalJSON() ([]byte, error) {
	type Alias RegistryError
	aux := &struct {
		*Alias
		CauseString string `json:"cause_string,omitempty"`
	}{
		Alias: (*Alias)(re),
	}
	
	if re.Cause != nil {
		aux.CauseString = re.Cause.Error()
	}
	
	return json.Marshal(aux)
}

// ErrorBuilder provides a fluent interface for building registry errors
type ErrorBuilder struct {
	err *RegistryError
}

// NewError creates a new error builder
func NewError(code ErrorCode, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &RegistryError{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
		},
	}
}

// NewErrorFromCode creates an error with predefined message
func NewErrorFromCode(code ErrorCode) *ErrorBuilder {
	message := getDefaultMessage(code)
	return NewError(code, message)
}

// Context adds context information
func (eb *ErrorBuilder) Context(key string, value any) *ErrorBuilder {
	if eb.err.Context == nil {
		eb.err.Context = make(map[string]any)
	}
	eb.err.Context[key] = value
	return eb
}

// Cause sets the underlying cause
func (eb *ErrorBuilder) Cause(cause error) *ErrorBuilder {
	eb.err.Cause = cause
	return eb
}

// Trace adds tracing information
func (eb *ErrorBuilder) Trace(traceID, spanID string) *ErrorBuilder {
	eb.err.TraceID = traceID
	eb.err.SpanID = spanID
	return eb
}

// Stack captures the current stack trace
func (eb *ErrorBuilder) Stack() *ErrorBuilder {
	eb.err.StackTrace = captureStackTrace(2) // Skip this function and the caller
	return eb
}

// Retryable marks the error as retryable
func (eb *ErrorBuilder) Retryable(after time.Duration) *ErrorBuilder {
	eb.err.Retryable = true
	eb.err.RetryAfter = after
	return eb
}

// UserMessage sets a user-friendly message
func (eb *ErrorBuilder) UserMessage(message string) *ErrorBuilder {
	eb.err.UserMessage = message
	return eb
}

// HelpURL adds a help URL
func (eb *ErrorBuilder) HelpURL(url string) *ErrorBuilder {
	eb.err.HelpURL = url
	return eb
}

// Build returns the constructed error
func (eb *ErrorBuilder) Build() *RegistryError {
	return eb.err
}

// ErrorAggregator collects multiple errors from concurrent operations
type ErrorAggregator struct {
	errors []error
	mu     sync.RWMutex
	limit  int
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator(limit int) *ErrorAggregator {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	
	return &ErrorAggregator{
		errors: make([]error, 0),
		limit:  limit,
	}
}

// Add adds an error to the aggregator
func (ea *ErrorAggregator) Add(err error) {
	if err == nil {
		return
	}
	
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	// Prevent unbounded growth
	if len(ea.errors) >= ea.limit {
		// Remove oldest error (FIFO)
		ea.errors = ea.errors[1:]
	}
	
	ea.errors = append(ea.errors, err)
}

// HasErrors returns true if any errors were collected
func (ea *ErrorAggregator) HasErrors() bool {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	return len(ea.errors) > 0
}

// Count returns the number of errors
func (ea *ErrorAggregator) Count() int {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	return len(ea.errors)
}

// Errors returns a copy of all collected errors
func (ea *ErrorAggregator) Errors() []error {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	
	result := make([]error, len(ea.errors))
	copy(result, ea.errors)
	return result
}

// Aggregate returns a single error containing all collected errors
func (ea *ErrorAggregator) Aggregate() error {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	
	if len(ea.errors) == 0 {
		return nil
	}
	
	if len(ea.errors) == 1 {
		return ea.errors[0]
	}
	
	// Create an aggregate error
	messages := make([]string, len(ea.errors))
	for i, err := range ea.errors {
		messages[i] = err.Error()
	}
	
	return NewError(ErrInternalError, fmt.Sprintf("multiple errors occurred (%d total)", len(ea.errors))).
		Context("error_count", len(ea.errors)).
		Context("errors", messages).
		UserMessage(fmt.Sprintf("%d errors occurred during operation", len(ea.errors))).
		Build()
}

// Clear removes all collected errors
func (ea *ErrorAggregator) Clear() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.errors = ea.errors[:0] // Keep underlying capacity
}

// Helper functions

func getDefaultMessage(code ErrorCode) string {
	switch code {
	case ErrToolNotFound:
		return "The requested tool was not found"
	case ErrToolAlreadyExists:
		return "A tool with this name already exists"
	case ErrToolInvalid:
		return "The tool configuration is invalid"
	case ErrToolDisabled:
		return "The tool is currently disabled"
	case ErrExecutionTimeout:
		return "Tool execution timed out"
	case ErrExecutionFailed:
		return "Tool execution failed"
	case ErrInvalidParams:
		return "Invalid parameters provided"
	case ErrResourceLimit:
		return "Resource limit exceeded"
	case ErrPermissionDenied:
		return "Permission denied"
	case ErrInsufficientPerms:
		return "Insufficient permissions"
	case ErrAccessRestricted:
		return "Access is restricted"
	case ErrSystemOverload:
		return "System is currently overloaded"
	case ErrConfigInvalid:
		return "Configuration is invalid"
	case ErrPluginFailure:
		return "Plugin operation failed"
	case ErrInternalError:
		return "An internal error occurred"
	case ErrRegistryNotStarted:
		return "Registry has not been started"
	case ErrRegistryShutdown:
		return "Registry is shutting down"
	default:
		return "An unknown error occurred"
	}
}

func captureStackTrace(skip int) string {
	const maxFrames = 32
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip+1, pcs) // +1 to skip this function
	
	if n == 0 {
		return "no stack trace available"
	}
	
	frames := runtime.CallersFrames(pcs[:n])
	var stackLines []string
	
	for {
		frame, more := frames.Next()
		stackLines = append(stackLines, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		
		if !more {
			break
		}
	}
	
	return strings.Join(stackLines, "\n")
}

// Predefined errors for common cases
var (
	ErrRegistryShuttingDown = NewErrorFromCode(ErrRegistryShutdown).
		UserMessage("The registry is currently shutting down").
		Build()
	
	ErrRegistryNotInitialized = NewErrorFromCode(ErrRegistryNotStarted).
		UserMessage("The registry has not been initialized").
		Build()
)

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if registryErr, ok := err.(*RegistryError); ok {
		return registryErr.Retryable
	}
	return false
}

// GetRetryAfter extracts retry duration from an error
func GetRetryAfter(err error) time.Duration {
	if registryErr, ok := err.(*RegistryError); ok {
		return registryErr.RetryAfter
	}
	return 0
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if registryErr, ok := err.(*RegistryError); ok {
		return registryErr.Code
	}
	return ErrInternalError
}