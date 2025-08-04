package controllers_test

import (
	"context"
	"time"

	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/testutil"
	"github.com/killallgit/ryan/pkg/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Streaming with Fake Client", func() {
	var (
		fakeClient *testutil.FakeStreamingChatClient
		controller *controllers.ChatController
	)

	BeforeEach(func() {
		fakeClient = testutil.NewFakeStreamingChatClient(
			"test-model",
			"Hello! I'm streaming this response to you.",
			"This is a longer response that will be streamed in multiple chunks for testing purposes.",
		)
		controller = controllers.NewChatController(fakeClient, "test-model", nil)
	})

	Describe("Basic streaming", func() {
		It("should stream response in chunks", func() {
			ctx := context.Background()
			fakeClient.SetChunkSize(5)
			fakeClient.SetChunkDelay(10 * time.Millisecond)

			updateChan, err := controller.StartStreaming(ctx, "Hi there!")
			Expect(err).ToNot(HaveOccurred())

			var updates []controllers.StreamingUpdate
			for update := range updateChan {
				updates = append(updates, update)
			}

			// Should have multiple chunk updates plus start and complete
			Expect(len(updates)).To(BeNumerically(">", 3))

			// First update should be start
			Expect(updates[0].Type).To(Equal(controllers.StreamStarted))

			// Last update should be complete
			lastUpdate := updates[len(updates)-1]
			Expect(lastUpdate.Type).To(Equal(controllers.MessageComplete))
			Expect(lastUpdate.Message.Content).To(Equal("Hello! I'm streaming this response to you."))

			// Should have chunk updates in between
			hasChunks := false
			for _, update := range updates {
				if update.Type == controllers.ChunkReceived {
					hasChunks = true
					break
				}
			}
			Expect(hasChunks).To(BeTrue())
		})

		It("should handle streaming cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel() // Ensure cancel is called on all paths

			fakeClient.SetChunkSize(2)
			fakeClient.SetChunkDelay(50 * time.Millisecond)

			updateChan, err := controller.StartStreaming(ctx, "Start streaming")
			Expect(err).ToNot(HaveOccurred())

			// Cancel after receiving first chunk update
			var updates []controllers.StreamingUpdate
			for update := range updateChan {
				updates = append(updates, update)
				if update.Type == controllers.ChunkReceived {
					cancel()
				}
			}

			// Should have received start and at least one chunk
			Expect(len(updates)).To(BeNumerically(">=", 2))

			// Should not have received complete update
			hasComplete := false
			for _, update := range updates {
				if update.Type == controllers.MessageComplete {
					hasComplete = true
					break
				}
			}
			Expect(hasComplete).To(BeFalse())
		})
	})

	Describe("Error handling", func() {
		It("should handle streaming errors", func() {
			ctx := context.Background()
			fakeClient.SetChunkSize(10)
			fakeClient.SetChunkDelay(10 * time.Millisecond)
			fakeClient.SetFailAfter(2, "simulated network error")

			updateChan, err := controller.StartStreaming(ctx, "This will fail")
			Expect(err).ToNot(HaveOccurred())

			var updates []controllers.StreamingUpdate
			var errorUpdate *controllers.StreamingUpdate
			for update := range updateChan {
				updates = append(updates, update)
				if update.Type == controllers.StreamError {
					errorUpdate = &update
				}
			}

			// Should have received error update
			Expect(errorUpdate).ToNot(BeNil())
			Expect(errorUpdate.Error).ToNot(BeNil())
			Expect(errorUpdate.Error.Error()).To(ContainSubstring("simulated network error"))
		})
	})

	Describe("Conversation management", func() {
		It("should update conversation after streaming", func() {
			ctx := context.Background()
			fakeClient.SetChunkSize(20) // Larger chunks for faster test

			// Initial message count
			Expect(controller.GetMessageCount()).To(Equal(0))

			updateChan, err := controller.StartStreaming(ctx, "Stream this")
			Expect(err).ToNot(HaveOccurred())

			// Consume all updates
			var lastUpdate controllers.StreamingUpdate
			for update := range updateChan {
				lastUpdate = update
			}

			// Should have added both user and assistant messages
			Expect(controller.GetMessageCount()).To(Equal(2))

			// Last assistant message should match streamed content
			msg, exists := controller.GetLastAssistantMessage()
			Expect(exists).To(BeTrue())
			Expect(msg.Content).To(Equal(lastUpdate.Message.Content))
		})

		It("should handle multiple streaming sessions", func() {
			ctx := context.Background()
			fakeClient.SetChunkSize(50) // Large chunks for speed

			// First streaming session
			updateChan1, err := controller.StartStreaming(ctx, "First message")
			Expect(err).ToNot(HaveOccurred())
			for range updateChan1 {
				// Consume all updates
			}

			// Second streaming session
			updateChan2, err := controller.StartStreaming(ctx, "Second message")
			Expect(err).ToNot(HaveOccurred())

			var finalContent string
			for update := range updateChan2 {
				if update.Type == controllers.MessageComplete {
					finalContent = update.Message.Content
				}
			}

			// Should have cycled to second response
			Expect(finalContent).To(Equal("This is a longer response that will be streamed in multiple chunks for testing purposes."))

			// Should have 4 messages total (2 user + 2 assistant)
			Expect(controller.GetMessageCount()).To(Equal(4))
		})
	})

	Describe("Tool support during streaming", func() {
		BeforeEach(func() {
			// Create streaming client with tool response
			fakeClient = testutil.NewFakeStreamingChatClient(
				"test-model",
				`{"tool_calls": [{"name": "weather", "arguments": {"location": "NYC"}}]}`,
				"The weather in NYC is sunny and 72°F",
			)

			// Create controller with weather tool
			toolRegistry := tools.NewRegistry()
			weatherTool := &testWeatherTool{}
			err := toolRegistry.Register(weatherTool)
			Expect(err).ToNot(HaveOccurred())

			controller = controllers.NewChatController(fakeClient, "test-model", toolRegistry)
		})

		It("should handle tool calls during streaming", func() {
			ctx := context.Background()

			updateChan, err := controller.StartStreaming(ctx, "What's the weather in NYC?")
			Expect(err).ToNot(HaveOccurred())

			var updates []controllers.StreamingUpdate
			for update := range updateChan {
				updates = append(updates, update)
			}

			// Should have tool execution updates
			hasToolUpdate := false
			for _, update := range updates {
				if update.Type == controllers.ToolExecutionStarted {
					hasToolUpdate = true
					break
				}
			}
			Expect(hasToolUpdate).To(BeTrue())

			// Final message should be the weather response
			lastUpdate := updates[len(updates)-1]
			Expect(lastUpdate.Type).To(Equal(controllers.MessageComplete))
			Expect(lastUpdate.Message.Content).To(Equal("The weather in NYC is sunny and 72°F"))

			// Conversation should include tool result
			messages := controller.GetHistory()
			hasToolResult := false
			for _, msg := range messages {
				if msg.Role == "tool" && msg.ToolName == "weather" {
					hasToolResult = true
					break
				}
			}
			Expect(hasToolResult).To(BeTrue())
		})
	})
})

// testWeatherTool implements a simple weather tool for testing
type testWeatherTool struct{}

func (t *testWeatherTool) Name() string {
	return "weather"
}

func (t *testWeatherTool) Description() string {
	return "Get weather information for a location"
}

func (t *testWeatherTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": "The city to get weather for",
			},
		},
		"required": []string{"location"},
	}
}

func (t *testWeatherTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	location, _ := params["location"].(string)
	return tools.ToolResult{
		Success: true,
		Content: "Sunny and 72°F in " + location,
		Metadata: tools.ToolMetadata{
			ToolName:   t.Name(),
			Parameters: params,
		},
	}, nil
}
