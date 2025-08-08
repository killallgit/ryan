package headless

import (
	"fmt"
	"os"
)

// Output handles console output for headless mode
type Output struct{}

// NewOutput creates a new output handler
func NewOutput() *Output {
	return &Output{}
}

// Error prints an error message to stderr
func (o *Output) Error(msg string) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
}
