package integration_test

import (
	"context"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming Tool Calling", func() {
	var (
		controller   *controllers.LangChainController
		toolRegistry *tools.Registry
		toolExecuted bool
		executedTool string
		executedCmd  string
	)

	BeforeEach(func() {
		// Initialize config
		err := config.InitializeDefaults()
		Expect(err).ToNot(HaveOccurred())
		
		_, err = config.Load("")
		Expect(err).ToNot(HaveOccurred())

		// Initialize tool registry
		toolRegistry = tools.NewRegistry()
		err = toolRegistry.RegisterBuiltinTools()
		Expect(err).ToNot(HaveOccurred())

		// Create controller with qwen3 model
		ollamaURL := "https://ollama.kitty-tetra.ts.net"
		controller, err = controllers.NewLangChainController(ollamaURL, "qwen3:latest", toolRegistry)
		Expect(err).ToNot(HaveOccurred())

		// Reset tool execution tracking
		toolExecuted = false
		executedTool = ""
		executedCmd = ""

		// Set up progress callback to track tool execution
		client := controller.GetClient()
		Expect(client).ToNot(BeNil())
		
		client.SetProgressCallback(func(toolName, command string) {
			toolExecuted = true
			executedTool = toolName
			executedCmd = command
			GinkgoWriter.Printf("Tool executed during streaming: %s -> %s\n", toolName, command)
		})
	})

	It("should execute tools during streaming when asked to list files", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Use streaming interface like the TUI does
		updates, err := controller.StartStreaming(ctx, "list the files in this dir")
		Expect(err).ToNot(HaveOccurred())

		// Collect all streaming updates
		var allContent strings.Builder
		var streamCompleted bool
		
		for update := range updates {
			switch update.Type {
			case controllers.ChunkReceived:
				allContent.WriteString(update.Content)
				GinkgoWriter.Printf("Chunk: %s\n", update.Content)
			case controllers.MessageComplete:
				streamCompleted = true
				GinkgoWriter.Printf("Stream completed\n")
			case controllers.StreamError:
				Fail("Streaming failed: " + update.Error.Error())
			case controllers.ToolExecutionStarted:
				GinkgoWriter.Printf("Tool execution started\n")
			case controllers.ToolExecutionComplete:
				GinkgoWriter.Printf("Tool execution completed\n")
			}
		}

		// Verify streaming completed
		Expect(streamCompleted).To(BeTrue(), "Expected streaming to complete")

		// The critical test: tool should have been executed during streaming
		Expect(toolExecuted).To(BeTrue(), "Expected tool to be executed during streaming")
		
		// Should have used execute_bash tool
		Expect(executedTool).To(Equal("execute_bash"), "Expected execute_bash tool to be used")
		
		// Command should contain ls
		Expect(strings.ToLower(executedCmd)).To(ContainSubstring("ls"), "Expected ls command to be executed")

		// Response should contain actual file listing results
		finalContent := allContent.String()
		GinkgoWriter.Printf("Final streaming content: %s\n", finalContent)
		
		// Should contain file listing results (not just thinking or descriptions)
		Expect(finalContent).To(Or(
			ContainSubstring(".go"),    // Go files likely in directory
			ContainSubstring("total"),  // ls -l output
			ContainSubstring("drwx"),   // Directory permissions
			MatchRegexp(`\d+`),         // File sizes or counts
		), "Expected actual file listing results in streamed response")
	})

	It("should execute tools during streaming when asked to count files", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Use streaming interface
		updates, err := controller.StartStreaming(ctx, "how many files are in this directory")
		Expect(err).ToNot(HaveOccurred())

		// Collect streaming results
		var allContent strings.Builder
		var streamCompleted bool
		
		for update := range updates {
			switch update.Type {
			case controllers.ChunkReceived:
				allContent.WriteString(update.Content)
			case controllers.MessageComplete:
				streamCompleted = true
			case controllers.StreamError:
				Fail("Streaming failed: " + update.Error.Error())
			case controllers.ToolExecutionStarted:
				GinkgoWriter.Printf("Tool execution started for count\n")
			case controllers.ToolExecutionComplete:
				GinkgoWriter.Printf("Tool execution completed for count\n")
			}
		}

		// Verify streaming completed and tool executed
		Expect(streamCompleted).To(BeTrue())
		Expect(toolExecuted).To(BeTrue(), "Expected tool to be executed during streaming")
		Expect(executedTool).To(Equal("execute_bash"))
		
		finalContent := allContent.String()
		
		// Should contain a number (file count)
		Expect(finalContent).To(MatchRegexp(`\d+`), "Expected file count in response")
	})
})