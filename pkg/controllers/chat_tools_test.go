package controllers_test

import (
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ChatController Tool Integration", func() {
	var (
		mockClient   *MockChatClient
		toolRegistry *tools.Registry
		controller   *controllers.ChatController
	)

	BeforeEach(func() {
		mockClient = &MockChatClient{}
		toolRegistry = tools.NewRegistry()
		
		// Register built-in tools
		err := toolRegistry.RegisterBuiltinTools()
		Expect(err).ToNot(HaveOccurred())
		
		controller = controllers.NewChatController(mockClient, "test-model", toolRegistry)
	})

	AfterEach(func() {
		mockClient.AssertExpectations(GinkgoT())
	})

	Describe("Tool Registry Integration", func() {
		It("should have tool registry available", func() {
			registry := controller.GetToolRegistry()
			Expect(registry).ToNot(BeNil())
			
			// Check that built-in tools are registered
			toolNames := registry.List()
			Expect(toolNames).To(ContainElement("execute_bash"))
			Expect(toolNames).To(ContainElement("read_file"))
		})

		It("should include tools in chat requests", func() {
			// Mock a response without tool calls (normal chat)
			mockResponse := chat.ChatResponse{
				Model:     "test-model",
				CreatedAt: time.Now(),
				Message: chat.Message{
					Role:    chat.RoleAssistant,
					Content: "Hello, I'm ready to help!",
				},
				Done: true,
			}

			// Capture the request to verify tools are included
			var capturedRequest chat.ChatRequest
			mockClient.On("SendMessageWithResponse", mock.MatchedBy(func(req chat.ChatRequest) bool {
				capturedRequest = req
				return true
			})).Return(mockResponse, nil)

			_, err := controller.SendUserMessage("Hello")

			Expect(err).ToNot(HaveOccurred())
			
			// Verify that tools were included in the request
			Expect(capturedRequest.Tools).ToNot(BeEmpty())
			Expect(len(capturedRequest.Tools)).To(Equal(2)) // bash and file read tools
		})

		It("should handle tool calls in assistant response", func() {
			// Mock a response with tool calls
			toolCallResponse := chat.ChatResponse{
				Model:     "test-model",
				CreatedAt: time.Now(),
				Message: chat.Message{
					Role:    chat.RoleAssistant,
					Content: "",
					ToolCalls: []chat.ToolCall{
						{
							Function: chat.ToolFunction{
								Name: "execute_bash",
								Arguments: map[string]any{
									"command": "echo 'test'",
								},
							},
						},
					},
				},
				Done: true,
			}

			// Mock final response after tool execution
			finalResponse := chat.ChatResponse{
				Model:     "test-model",
				CreatedAt: time.Now(),
				Message: chat.Message{
					Role:    chat.RoleAssistant,
					Content: "I executed the command and got: test",
				},
				Done: true,
			}

			// Set up mock expectations
			mockClient.On("SendMessageWithResponse", mock.Anything).Return(toolCallResponse, nil).Once()
			mockClient.On("SendMessageWithResponse", mock.Anything).Return(finalResponse, nil).Once()

			response, err := controller.SendUserMessage("Run echo test command")

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Content).To(Equal("I executed the command and got: test"))

			// Check that conversation includes tool execution
			history := controller.GetHistory()
			Expect(len(history)).To(BeNumerically(">=", 3)) // user message, assistant with tool calls, tool result, final response
			
			// Find tool result message
			var foundToolResult bool
			for _, msg := range history {
				if msg.IsTool() && msg.ToolName == "execute_bash" {
					foundToolResult = true
					Expect(msg.Content).To(ContainSubstring("test"))
					break
				}
			}
			Expect(foundToolResult).To(BeTrue())
		})
	})

	Describe("Tool Registry Management", func() {
		It("should support setting and getting tool registry", func() {
			newRegistry := tools.NewRegistry()
			controller.SetToolRegistry(newRegistry)
			
			retrievedRegistry := controller.GetToolRegistry()
			Expect(retrievedRegistry).To(Equal(newRegistry))
		})

		It("should work without tool registry", func() {
			controllerWithoutTools := controllers.NewChatController(mockClient, "test-model", nil)
			
			mockResponse := chat.ChatResponse{
				Model:     "test-model",
				CreatedAt: time.Now(),
				Message: chat.Message{
					Role:    chat.RoleAssistant,
					Content: "Hello without tools",
				},
				Done: true,
			}

			var capturedRequest chat.ChatRequest
			mockClient.On("SendMessageWithResponse", mock.MatchedBy(func(req chat.ChatRequest) bool {
				capturedRequest = req
				return true
			})).Return(mockResponse, nil)

			_, err := controllerWithoutTools.SendUserMessage("Hello")

			Expect(err).ToNot(HaveOccurred())
			
			// Verify that no tools were included in the request
			Expect(capturedRequest.Tools).To(BeEmpty())
		})
	})
})