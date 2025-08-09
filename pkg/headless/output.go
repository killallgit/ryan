package headless

import (
	"github.com/killallgit/ryan/pkg/logger"
)

// Output handles console output for headless mode
type Output struct{}

// NewOutput creates a new output handler
func NewOutput() *Output {
	return &Output{}
}

// Error prints an error message using the logger
func (o *Output) Error(msg string) {
	logger.Error(msg)
}
