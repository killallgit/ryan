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

var _ = Describe("Basic Tool Calling", func() {
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
			GinkgoWriter.Printf("Tool executed: %s -> %s\n", toolName, command)
		})
	})

	It("should execute ls command when asked to list files", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Send the query
		response, err := controller.SendUserMessageWithContext(ctx, "list the files in this dir")

		// Log the response for debugging
		GinkgoWriter.Printf("Response: %s\n", response.Content)
		GinkgoWriter.Printf("Tool executed: %v\n", toolExecuted)
		if toolExecuted {
			GinkgoWriter.Printf("Tool: %s, Command: %s\n", executedTool, executedCmd)
		}

		// The test should not error
		Expect(err).ToNot(HaveOccurred())

		// Tool should have been executed
		Expect(toolExecuted).To(BeTrue(), "Expected tool to be executed")

		// Should have used execute_bash tool
		Expect(executedTool).To(Equal("execute_bash"))

		// Command should contain ls
		Expect(strings.ToLower(executedCmd)).To(ContainSubstring("ls"))
	})

	It("should execute command when asked to count files", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Send the query
		response, err := controller.SendUserMessageWithContext(ctx, "how many files are in this directory")

		// Log the response for debugging
		GinkgoWriter.Printf("Response: %s\n", response.Content)
		GinkgoWriter.Printf("Tool executed: %v\n", toolExecuted)
		if toolExecuted {
			GinkgoWriter.Printf("Tool: %s, Command: %s\n", executedTool, executedCmd)
		}

		// The test should not error
		Expect(err).ToNot(HaveOccurred())

		// Tool should have been executed
		Expect(toolExecuted).To(BeTrue(), "Expected tool to be executed")

		// Should have used execute_bash tool
		Expect(executedTool).To(Equal("execute_bash"))

		// Command should be for counting files
		Expect(executedCmd).To(Or(
			ContainSubstring("ls"),
			ContainSubstring("find"),
			ContainSubstring("wc"),
		))

		// Response should contain a number
		Expect(response.Content).To(MatchRegexp(`\d+`))
	})
})
