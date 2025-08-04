package controllers_test

import (
	"context"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/testutil"
	"github.com/killallgit/ryan/pkg/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testCalculatorTool implements the Tool interface for testing
type testCalculatorTool struct{}

func (t *testCalculatorTool) Name() string {
	return "calculate"
}

func (t *testCalculatorTool) Description() string {
	return "Perform mathematical calculations"
}

func (t *testCalculatorTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The mathematical expression to evaluate",
			},
		},
		"required": []string{"expression"},
	}
}

func (t *testCalculatorTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	// For testing, always return 4
	return tools.ToolResult{
		Success: true,
		Content: "4",
		Metadata: tools.ToolMetadata{
			ToolName:   t.Name(),
			Parameters: params,
		},
	}, nil
}

var _ = Describe("ChatController with Fake LLM", func() {
	var (
		fakeClient *testutil.FakeChatClient
		controller *controllers.ChatController
	)

	BeforeEach(func() {
		fakeClient = testutil.NewFakeChatClient(
			"test-model",
			testutil.PredefinedResponses.SimpleChat...,
		)
		controller = controllers.NewChatController(fakeClient, "test-model", nil)
	})

	Describe("Conversation flow", func() {
		It("should handle a complete conversation", func() {
			// First user message
			resp1, err := controller.SendUserMessage("Hi there!")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp1.Content).To(Equal("Hello! How can I help you today?"))
			Expect(controller.GetMessageCount()).To(Equal(2)) // user + assistant

			// Second user message
			resp2, err := controller.SendUserMessage("Can you help me?")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.Content).To(Equal("I understand your question. Here's my response."))
			Expect(controller.GetMessageCount()).To(Equal(4)) // 2 user + 2 assistant

			// Third user message
			resp3, err := controller.SendUserMessage("Thanks!")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp3.Content).To(Equal("Thank you for the clarification."))
			Expect(controller.GetMessageCount()).To(Equal(6))
		})

		It("should maintain conversation history", func() {
			// Send first message
			_, err := controller.SendUserMessage("Remember the number 42")
			Expect(err).ToNot(HaveOccurred())

			// Check that conversation includes both messages
			messages := controller.GetHistory()
			Expect(messages).To(HaveLen(2))
			Expect(messages[0].Role).To(Equal(chat.RoleUser))
			Expect(messages[0].Content).To(Equal("Remember the number 42"))
			Expect(messages[1].Role).To(Equal(chat.RoleAssistant))

			// Verify the fake LLM received the full conversation
			fakeLLM := fakeClient.GetFakeLLM()
			lastPrompt := fakeLLM.GetLastPrompt()
			Expect(lastPrompt).To(ContainSubstring("Remember the number 42"))
		})
	})

	Describe("System messages", func() {
		BeforeEach(func() {
			controller = controllers.NewChatControllerWithSystem(
				fakeClient,
				"test-model",
				"You are a helpful coding assistant",
				nil,
			)
		})

		It("should include system message in conversation", func() {
			Expect(controller.HasSystemMessage()).To(BeTrue())

			_, err := controller.SendUserMessage("Hello")
			Expect(err).ToNot(HaveOccurred())

			// Check that the fake LLM received the system message
			fakeLLM := fakeClient.GetFakeLLM()
			lastPrompt := fakeLLM.GetLastPrompt()
			Expect(lastPrompt).To(ContainSubstring("You are a helpful coding assistant"))
			Expect(lastPrompt).To(ContainSubstring("Hello"))
		})
	})

	Describe("Error handling", func() {
		It("should handle errors from LLM", func() {
			// Configure to fail on second call
			fakeClient.GetFakeLLM().SetErrorOnCall(2, "API rate limit exceeded")

			// First message should succeed
			_, err := controller.SendUserMessage("First message")
			Expect(err).ToNot(HaveOccurred())

			// Second message should fail
			_, err = controller.SendUserMessage("Second message")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API rate limit exceeded"))
		})
	})

	Describe("Tool usage", func() {
		BeforeEach(func() {
			// Set up fake client with tool response
			fakeClient = testutil.NewFakeChatClient(
				"test-model",
				`{"tool_calls": [{"name": "calculate", "arguments": {"expression": "2+2"}}]}`,
				"The result of 2+2 is 4",
			)

			// Create controller with tools
			toolRegistry := tools.NewRegistry()
			calculatorTool := &testCalculatorTool{}
			err := toolRegistry.Register(calculatorTool)
			Expect(err).ToNot(HaveOccurred())
			controller = controllers.NewChatController(fakeClient, "test-model", toolRegistry)
		})

		It("should handle tool calls", func() {
			resp, err := controller.SendUserMessage("What is 2+2?")
			Expect(err).ToNot(HaveOccurred())

			// The final response should be after tool execution
			Expect(resp.Content).To(Equal("The result of 2+2 is 4"))

			// The conversation should now include tool result
			messages := controller.GetHistory()
			// Should have: user message, assistant with tool call, tool progress, tool result, final assistant response
			Expect(messages).To(HaveLen(5))

			// Check the assistant message with tool call
			assistantMsg := messages[1]
			Expect(assistantMsg.Role).To(Equal(chat.RoleAssistant))
			Expect(assistantMsg.ToolCalls).To(HaveLen(1))
			Expect(assistantMsg.ToolCalls[0].Function.Name).To(Equal("calculate"))

			// Check the tool progress message
			progressMsg := messages[2]
			Expect(progressMsg.Role).To(Equal(chat.RoleToolProgress))
			Expect(progressMsg.Content).To(Equal("calculate(2+2)"))
			Expect(progressMsg.ToolName).To(Equal("calculate"))

			// Check tool result message
			toolMsg := messages[3]
			Expect(toolMsg.Role).To(Equal(chat.RoleTool))
			Expect(toolMsg.Content).To(Equal("4"))
			Expect(toolMsg.ToolName).To(Equal("calculate"))

			// Check final assistant response
			finalMsg := messages[4]
			Expect(finalMsg.Role).To(Equal(chat.RoleAssistant))
			Expect(finalMsg.Content).To(Equal("The result of 2+2 is 4"))
		})
	})

	Describe("Response timing", func() {
		It("should simulate realistic response times", func() {
			// Set a longer response time
			fakeClient.SetResponseTime(100 * time.Millisecond)

			start := time.Now()
			_, err := controller.SendUserMessage("Test timing")
			elapsed := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			Expect(elapsed).To(BeNumerically(">=", 100*time.Millisecond))
		})
	})

	Describe("Conversation reset", func() {
		It("should clear conversation history", func() {
			// Send some messages
			_, err := controller.SendUserMessage("Message 1")
			Expect(err).ToNot(HaveOccurred())
			_, err = controller.SendUserMessage("Message 2")
			Expect(err).ToNot(HaveOccurred())

			Expect(controller.GetMessageCount()).To(Equal(4))

			// Reset conversation
			controller.Reset()
			Expect(controller.GetMessageCount()).To(Equal(0))

			// Reset the fake LLM too
			fakeClient.GetFakeLLM().Reset()

			// New message should start fresh
			_, err = controller.SendUserMessage("Fresh start")
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.GetMessageCount()).To(Equal(2))
		})
	})
})
