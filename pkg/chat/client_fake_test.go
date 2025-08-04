package chat_test

import (
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client with Fake LLM", func() {
	var (
		fakeClient *testutil.FakeChatClient
	)

	Describe("SendMessage with fake responses", func() {
		BeforeEach(func() {
			fakeClient = testutil.NewFakeChatClient(
				"test-model",
				"Hello! I'm a helpful assistant.",
				"I can answer your questions.",
				"Let me help you with that.",
			)
		})

		It("should send message and get response", func() {
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			msg, err := fakeClient.SendMessage(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(msg.Role).To(Equal("assistant"))
			Expect(msg.Content).To(Equal("Hello! I'm a helpful assistant."))
		})

		It("should handle multiple messages in conversation", func() {
			// First message
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			resp1, err := fakeClient.SendMessage(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp1.Content).To(Equal("Hello! I'm a helpful assistant."))

			// Second message
			req.Messages = append(req.Messages, resp1, chat.NewUserMessage("What can you do?"))

			resp2, err := fakeClient.SendMessage(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.Content).To(Equal("I can answer your questions."))
		})

		It("should track call history", func() {
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewSystemMessage("You are helpful"),
					chat.NewUserMessage("Test prompt"),
				},
			}

			_, err := fakeClient.SendMessage(req)
			Expect(err).ToNot(HaveOccurred())

			fakeLLM := fakeClient.GetFakeLLM()
			Expect(fakeLLM.GetCallCount()).To(Equal(1))
			Expect(fakeLLM.GetLastPrompt()).To(ContainSubstring("You are helpful"))
			Expect(fakeLLM.GetLastPrompt()).To(ContainSubstring("Test prompt"))
		})
	})

	Describe("Error simulation", func() {
		BeforeEach(func() {
			fakeClient = testutil.NewFakeChatClient("test-model", "success response")
			fakeClient.GetFakeLLM().SetErrorOnCall(1, "simulated network error")
		})

		It("should handle simulated errors", func() {
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			_, err := fakeClient.SendMessage(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("simulated network error"))
		})
	})

	Describe("Tool calling", func() {
		BeforeEach(func() {
			toolResponse := `{"tool_calls": [{"name": "get_weather", "arguments": {"location": "San Francisco"}}]}`
			fakeClient = testutil.NewFakeChatClient("test-model", toolResponse)
		})

		It("should handle tool calls", func() {
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewUserMessage("What's the weather in San Francisco?"),
				},
				Tools: []map[string]any{
					{
						"name":        "get_weather",
						"description": "Get the weather for a location",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location": map[string]any{
									"type":        "string",
									"description": "The city name",
								},
							},
						},
					},
				},
			}

			resp, err := fakeClient.SendMessageWithResponse(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Message.ToolCalls).To(HaveLen(1))
			Expect(resp.Message.ToolCalls[0].Function.Name).To(Equal("get_weather"))
			Expect(resp.Message.ToolCalls[0].Function.Arguments["location"]).To(Equal("San Francisco"))
		})
	})

	Describe("Response metadata", func() {
		BeforeEach(func() {
			fakeClient = testutil.NewFakeChatClient("test-model", "Test response")
		})

		It("should return complete response with metadata", func() {
			req := chat.ChatRequest{
				Model: "test-model",
				Messages: []chat.Message{
					chat.NewUserMessage("Test"),
				},
			}

			resp, err := fakeClient.SendMessageWithResponse(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Model).To(Equal("test-model"))
			Expect(resp.Done).To(BeTrue())
			Expect(resp.DoneReason).To(Equal("complete"))
			Expect(resp.PromptEvalCount).To(BeNumerically(">", 0))
			Expect(resp.EvalCount).To(BeNumerically(">", 0))
			Expect(resp.CreatedAt).ToNot(BeZero())
		})
	})
})
