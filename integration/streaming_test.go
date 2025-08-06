package integration

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
)

var _ = Describe("Streaming Integration Tests", func() {
	var (
		client     *chat.StreamingClient
		controller *controllers.ChatController
		ollamaURL  string
		testModel  string
	)

	BeforeEach(func() {
		// Skip integration tests unless explicitly enabled
		if os.Getenv("INTEGRATION_TEST") != "true" {
			Skip("Integration tests skipped. Set INTEGRATION_TEST=true to run.")
		}

		// Configure test settings
		viper.Set("ollama.url", "http://localhost:11434")
		viper.Set("ollama.model", "llama3.1:8b")
		viper.Set("ollama.timeout", "30s")

		ollamaURL = viper.GetString("ollama.url")
		testModel = viper.GetString("ollama.model")

		// Create streaming client
		var err error
		client, err = chat.NewStreamingClient(ollamaURL, testModel)
		if err != nil {
			Skip("Failed to create streaming client: " + err.Error())
		}
		controller = controllers.NewChatController(client, testModel, nil)

		// Basic connectivity check
		basicClient, err := chat.NewClient(ollamaURL, testModel)
		if err != nil {
			Skip("Failed to create basic client: " + err.Error())
		}
		_, err = basicClient.SendMessage(chat.ChatRequest{
			Model:    testModel,
			Messages: []chat.Message{{Role: "user", Content: "test"}},
			Stream:   false,
		})
		if err != nil {
			Skip("Ollama server not available or model not found: " + err.Error())
		}
	})

	Describe("Real Ollama Streaming API", func() {
		It("should stream message chunks progressively", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Start streaming with a prompt that should generate a longer response
			updates, err := controller.StartStreaming(ctx, "Tell me a short story about a robot learning to paint. Keep it to about 100 words.")
			Expect(err).ToNot(HaveOccurred())
			Expect(updates).ToNot(BeNil())

			var (
				streamStarted    = false
				chunksReceived   = 0
				messageCompleted = false
				finalMessage     chat.Message
				totalDuration    time.Duration
				contentLength    int
			)

			// Process all streaming updates
			for update := range updates {
				GinkgoWriter.Printf("Received update: Type=%d, StreamID=%s, Content='%s'\n",
					update.Type, update.StreamID, update.Content)

				switch update.Type {
				case controllers.StreamStarted:
					streamStarted = true
					Expect(update.StreamID).ToNot(BeEmpty())
					Expect(update.Metadata.Model).To(Equal(testModel))

				case controllers.ChunkReceived:
					chunksReceived++
					contentLength += len(update.Content)
					Expect(update.StreamID).ToNot(BeEmpty())
					Expect(update.Content).ToNot(BeEmpty()) // Each chunk should have content
					GinkgoWriter.Printf("Chunk %d: '%s' (length: %d)\n",
						chunksReceived, update.Content, len(update.Content))

				case controllers.MessageComplete:
					messageCompleted = true
					finalMessage = update.Message
					totalDuration = update.Metadata.Duration
					Expect(update.StreamID).ToNot(BeEmpty())
					Expect(finalMessage.Content).ToNot(BeEmpty())
					Expect(finalMessage.Role).To(Equal(chat.RoleAssistant))

				case controllers.StreamError:
					Fail("Received stream error: " + update.Error.Error())

				case controllers.ToolExecutionStarted, controllers.ToolExecutionComplete:
					// These are fine, just log them
					GinkgoWriter.Printf("Tool execution update: %d\n", update.Type)
				}
			}

			// Verify streaming workflow
			Expect(streamStarted).To(BeTrue(), "Stream should have started")
			Expect(chunksReceived).To(BeNumerically(">", 0), "Should have received chunks")
			Expect(messageCompleted).To(BeTrue(), "Message should have completed")
			Expect(finalMessage.Content).ToNot(BeEmpty(), "Final message should have content")

			// Verify the message makes sense
			Expect(len(finalMessage.Content)).To(BeNumerically(">", 20), "Response should be substantial")
			Expect(totalDuration).To(BeNumerically(">", 0), "Duration should be tracked")

			GinkgoWriter.Printf("Streaming completed successfully:\n")
			GinkgoWriter.Printf("  - Chunks received: %d\n", chunksReceived)
			GinkgoWriter.Printf("  - Total content length: %d\n", len(finalMessage.Content))
			GinkgoWriter.Printf("  - Duration: %v\n", totalDuration)
			GinkgoWriter.Printf("  - Final message: %s\n", finalMessage.Content)
		})

		It("should handle streaming with tool calls", func() {
			// This test requires tools to be set up, skip for now
			Skip("Tool integration streaming test requires tool registry setup")
		})

		It("should handle streaming errors gracefully", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create controller with non-existent model to trigger error
			badController := controllers.NewChatController(client, "non-existent-model:latest", nil)

			updates, err := badController.StartStreaming(ctx, "test message")
			Expect(err).ToNot(HaveOccurred()) // StartStreaming should not error immediately
			Expect(updates).ToNot(BeNil())

			errorReceived := false
			for update := range updates {
				if update.Type == controllers.StreamError {
					errorReceived = true
					Expect(update.Error).ToNot(BeNil())
					GinkgoWriter.Printf("Expected error received: %s\n", update.Error.Error())
					break
				}
			}

			Expect(errorReceived).To(BeTrue(), "Should have received a stream error for non-existent model")
		})

		It("should handle streaming cancellation", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			// Start streaming with a long prompt that will take time
			updates, err := controller.StartStreaming(ctx, "Write a very detailed essay about quantum physics, including history, principles, applications, and future prospects. Make it at least 500 words.")
			Expect(err).ToNot(HaveOccurred())
			Expect(updates).ToNot(BeNil())

			chunksReceived := 0

			go func() {
				// Cancel after receiving a few chunks
				time.Sleep(2 * time.Second)
				cancel()
			}()

			for update := range updates {
				switch update.Type {
				case controllers.ChunkReceived:
					chunksReceived++
					GinkgoWriter.Printf("Received chunk %d before cancellation\n", chunksReceived)

				case controllers.StreamError:
					if update.Error == context.Canceled {
						GinkgoWriter.Printf("Stream cancelled as expected\n")
					}
					break

				case controllers.MessageComplete:
					// This might happen if the response was very fast
					GinkgoWriter.Printf("Message completed before cancellation\n")
					break
				}
			}

			// We should have received some chunks before cancellation
			Expect(chunksReceived).To(BeNumerically(">", 0), "Should have received some chunks before cancellation")
			GinkgoWriter.Printf("Cancellation test completed with %d chunks received\n", chunksReceived)
		})

		Measure("streaming performance", func(b Benchmarker) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			startTime := time.Now()

			updates, err := controller.StartStreaming(ctx, "Count from 1 to 10 with brief explanations.")
			Expect(err).ToNot(HaveOccurred())

			var (
				firstChunkTime time.Time
				lastChunkTime  time.Time
				chunksReceived int
			)

			for update := range updates {
				switch update.Type {
				case controllers.ChunkReceived:
					chunksReceived++
					if firstChunkTime.IsZero() {
						firstChunkTime = time.Now()
					}
					lastChunkTime = time.Now()

				case controllers.MessageComplete:
					break
				}
			}

			timeToFirstChunk := firstChunkTime.Sub(startTime)
			totalStreamingTime := lastChunkTime.Sub(firstChunkTime)

			b.RecordValue("Time to first chunk (ms)", float64(timeToFirstChunk.Milliseconds()))
			b.RecordValue("Total streaming time (ms)", float64(totalStreamingTime.Milliseconds()))
			b.RecordValue("Chunks received", float64(chunksReceived))

			if chunksReceived > 0 {
				avgChunkInterval := totalStreamingTime / time.Duration(chunksReceived)
				b.RecordValue("Average chunk interval (ms)", float64(avgChunkInterval.Milliseconds()))
			}

			// Performance expectations
			Expect(timeToFirstChunk).To(BeNumerically("<", 5*time.Second), "First chunk should arrive within 5 seconds")
			Expect(chunksReceived).To(BeNumerically(">", 0), "Should receive multiple chunks")

		}, 3) // Run 3 times for average
	})

	Describe("Streaming Client Direct Tests", func() {
		It("should handle raw streaming chunks", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req := chat.ChatRequest{
				Model:    testModel,
				Messages: []chat.Message{{Role: "user", Content: "Say hello and count to 3"}},
				Stream:   true,
			}

			chunks, err := client.StreamMessage(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(chunks).ToNot(BeNil())

			chunksReceived := 0
			contentReceived := ""
			streamCompleted := false

			for chunk := range chunks {
				GinkgoWriter.Printf("Raw chunk: StreamID=%s, Content='%s', Done=%t, Error=%v\n",
					chunk.StreamID, chunk.Content, chunk.Done, chunk.Error)

				if chunk.Error != nil {
					Fail("Received chunk error: " + chunk.Error.Error())
				}

				chunksReceived++
				contentReceived += chunk.Content

				if chunk.Done {
					streamCompleted = true
					break
				}
			}

			Expect(chunksReceived).To(BeNumerically(">", 0), "Should receive chunks")
			Expect(streamCompleted).To(BeTrue(), "Stream should complete")
			Expect(contentReceived).ToNot(BeEmpty(), "Should receive content")

			GinkgoWriter.Printf("LangChain streaming test completed:\n")
			GinkgoWriter.Printf("  - Updates: %d\n", chunksReceived)
			GinkgoWriter.Printf("  - Content: %s\n", contentReceived)
		})
	})
})
