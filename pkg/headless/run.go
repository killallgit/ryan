package headless

import (
	"context"
	"fmt"

	"github.com/killallgit/ryan/pkg/agent"
	"github.com/killallgit/ryan/pkg/logger"
)

// RunHeadless executes a single prompt in headless mode
// This is the main entry point for headless/CLI execution
func RunHeadless(agent agent.Agent, prompt string) error {
	return RunHeadlessWithOptions(agent, prompt, false)
}

// RunHeadlessWithOptions executes a single prompt in headless mode with options
func RunHeadlessWithOptions(agent agent.Agent, prompt string, continueHistory bool) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty in headless mode")
	}

	// Create runner (internal implementation detail)
	runner, err := newRunnerWithOptions(agent, continueHistory)
	if err != nil {
		return fmt.Errorf("failed to initialize headless mode: %w", err)
	}

	// Execute the prompt
	ctx := context.Background()
	if err := runner.run(ctx, prompt); err != nil {
		return fmt.Errorf("failed to execute prompt: %w", err)
	}

	// Cleanup
	if err := runner.cleanup(); err != nil {
		// Log warning but don't fail
		logger.Warn("Cleanup error: %v", err)
	}

	return nil
}
