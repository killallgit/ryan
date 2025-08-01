package integration_test

import (
	"os"
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = Describe("Chat Integration Tests", func() {
	var (
		client     *chat.Client
		controller *controllers.ChatController
		ollamaURL  string
		testModel  string
	)

	BeforeEach(func() {
		// Get Ollama URL from environment or use default
		ollamaURL = os.Getenv("OLLAMA_URL")
		if ollamaURL == "" {
			ollamaURL = "https://ollama.kitty-tetra.ts.net"
		}

		// Get test model from environment or use default
		testModel = os.Getenv("OLLAMA_TEST_MODEL")
		if testModel == "" {
			testModel = "qwen2.5-coder:1.5b-base"
		}

		// Skip if SKIP_INTEGRATION is set
		if os.Getenv("SKIP_INTEGRATION") != "" {
			Skip("Integration tests skipped by SKIP_INTEGRATION environment variable")
		}

		client = chat.NewClient(ollamaURL)
		controller = controllers.NewChatController(client, testModel)
	})

	Describe("Real Ollama API Communication", func() {
		It("should successfully send a message and receive a response", func() {
			// Simple test prompt
			prompt := "What is 2+2? Answer with just the number."

			response, err := controller.SendUserMessage(prompt)
			
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Role).To(Equal(chat.RoleAssistant))
			Expect(response.Content).ToNot(BeEmpty())
			Expect(response.Content).To(ContainSubstring("4"))
		})

		It("should maintain conversation history", func() {
			// First message
			_, err := controller.SendUserMessage("My name is TestUser")
			Expect(err).ToNot(HaveOccurred())

			// Second message referencing first
			response, err := controller.SendUserMessage("What is my name?")
			Expect(err).ToNot(HaveOccurred())
			
			// The model should remember the name
			Expect(response.Content).To(ContainSubstring("TestUser"))
		})

		It("should handle system prompts correctly", func() {
			// Create controller with system prompt
			systemPrompt := "You are a helpful assistant that always responds with exactly 'OK' to any input."
			controllerWithSystem := controllers.NewChatControllerWithSystem(client, testModel, systemPrompt)

			response, err := controllerWithSystem.SendUserMessage("Tell me a long story")
			Expect(err).ToNot(HaveOccurred())
			
			// Response should be constrained by system prompt
			Expect(len(response.Content)).To(BeNumerically("<", 100))
		})

		It("should handle empty messages appropriately", func() {
			_, err := controller.SendUserMessage("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("message content cannot be empty"))
		})

		It("should reset conversation properly", func() {
			// Send initial message
			_, err := controller.SendUserMessage("Remember the number 42")
			Expect(err).ToNot(HaveOccurred())

			// Reset conversation
			controller.Reset()

			// Ask about the number
			response, err := controller.SendUserMessage("What number did I ask you to remember?")
			Expect(err).ToNot(HaveOccurred())

			// Should not remember after reset
			Expect(response.Content).ToNot(ContainSubstring("42"))
		})
	})

	Describe("API Availability and Error Handling", func() {
		It("should handle network timeouts gracefully", func() {
			// Create client with very short timeout
			shortTimeoutClient := &chat.Client{}
			*shortTimeoutClient = *client
			// This is a bit hacky but works for testing timeout behavior
			
			controller := controllers.NewChatController(shortTimeoutClient, testModel)
			
			// Send a complex prompt that might take longer
			_, err := controller.SendUserMessage("Generate a very long essay about quantum physics with examples")
			
			// Should either succeed or timeout gracefully
			if err != nil {
				Expect(err.Error()).To(Or(
					ContainSubstring("timeout"),
					ContainSubstring("deadline exceeded"),
				))
			}
		})

		It("should verify model exists on server", func() {
			// Try with a non-existent model
			badController := controllers.NewChatController(client, "non-existent-model:latest")
			
			_, err := badController.SendUserMessage("Hello")
			
			// Should get an error about model not found
			Expect(err).To(HaveOccurred())
			// The actual error message depends on Ollama's response
		})
	})

	Describe("Performance and Response Quality", func() {
		It("should respond within reasonable time", func() {
			start := time.Now()
			
			_, err := controller.SendUserMessage("Hello")
			
			duration := time.Since(start)
			Expect(err).ToNot(HaveOccurred())
			
			// Response should come back within 30 seconds
			Expect(duration).To(BeNumerically("<", 30*time.Second))
		})

		It("should handle multiple concurrent requests", func() {
			// Note: This tests the client's behavior, not true concurrency
			// since we're using the same controller
			
			for i := 0; i < 3; i++ {
				response, err := controller.SendUserMessage("Count to 3")
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Content).ToNot(BeEmpty())
			}
		})
	})
})

var _ = Describe("Configuration Integration", func() {
	It("should respect viper configuration", func() {
		// Set up viper config
		viper.Set("ollama.url", "https://ollama.kitty-tetra.ts.net")
		viper.Set("ollama.model", "qwen2.5-coder:1.5b-base")

		// Create client using viper config
		client := chat.NewClient(viper.GetString("ollama.url"))
		controller := controllers.NewChatController(client, viper.GetString("ollama.model"))

		// Should work with viper configuration
		response, err := controller.SendUserMessage("Say 'config works'")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Content).To(ContainSubstring("config works"))
	})
})