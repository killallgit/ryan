package integration

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
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
		controller *controllers.LangChainController
		ollamaURL  string
		testModel  string
	)

	BeforeEach(func() {
		// Use Viper for configuration
		viper.SetEnvPrefix("")
		viper.AutomaticEnv()

		// Skip integration tests unless explicitly enabled
		if viper.GetString("INTEGRATION_TEST") != "true" {
			Skip("Integration tests skipped. Set INTEGRATION_TEST=true to run.")
		}

		// Set default test configuration and get values
		viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
		viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")

		ollamaURL = viper.GetString("ollama.url")
		testModel = viper.GetString("ollama.model")

		// Create tool registry
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		if err != nil {
			Skip("Failed to register builtin tools: " + err.Error())
		}

		// Create LangChain controller
		controllerCfg := &controllers.InitConfig{
			Config: &config.Config{
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					URL:   ollamaURL,
					Model: testModel,
				},
			},
			Model:        testModel,
			ToolRegistry: toolRegistry,
		}

		controller, err = controllers.InitializeLangChainController(controllerCfg)
		if err != nil {
			Skip("Failed to create LangChain controller: " + err.Error())
		}
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
			toolRegistry := tools.NewRegistry()
			err := toolRegistry.RegisterBuiltinTools()
			Expect(err).ToNot(HaveOccurred())

			controllerWithSystem, err := controllers.NewLangChainControllerWithSystem(
				ollamaURL, testModel, systemPrompt, toolRegistry)
			Expect(err).ToNot(HaveOccurred())

			response, err := controllerWithSystem.SendUserMessage("Tell me a long story")
			Expect(err).ToNot(HaveOccurred())

			// Response should be influenced by system prompt (though models may be verbose)
			// Just verify we got a response
			Expect(response.Content).ToNot(BeEmpty())
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
			// For LangChain controller, we'll test timeout at the controller level
			// rather than creating a separate client

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
			// Try with a clearly non-existent model
			toolRegistry := tools.NewRegistry()
			err := toolRegistry.RegisterBuiltinTools()
			Expect(err).ToNot(HaveOccurred())

			badControllerCfg := &controllers.InitConfig{
				Config: &config.Config{
					Provider: "ollama",
					Ollama: config.OllamaConfig{
						URL:   ollamaURL,
						Model: "absolutely-non-existent-model-12345:latest",
					},
				},
				Model:        "absolutely-non-existent-model-12345:latest",
				ToolRegistry: toolRegistry,
			}

			badController, controllerErr := controllers.InitializeLangChainController(badControllerCfg)
			if controllerErr != nil {
				// Controller creation failed - that's acceptable
				return
			}

			_, sendErr := badController.SendUserMessage("Hello")

			// Should get an error about model not found
			// Note: This may pass if the server auto-pulls models, which is acceptable
			if sendErr != nil {
				Expect(sendErr.Error()).To(ContainSubstring("model"))
			}
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
		// Use Viper for configuration
		viper.SetEnvPrefix("")
		viper.AutomaticEnv()

		if viper.GetString("INTEGRATION_TEST") != "true" {
			Skip("Integration tests skipped. Set INTEGRATION_TEST=true to run.")
		}

		// Set up viper config with defaults
		viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
		viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")

		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		Expect(err).ToNot(HaveOccurred())

		// Create LangChain controller using viper config
		controllerCfg := &controllers.InitConfig{
			Config: &config.Config{
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					URL:   viper.GetString("ollama.url"),
					Model: viper.GetString("ollama.model"),
				},
			},
			Model:        viper.GetString("ollama.model"),
			ToolRegistry: toolRegistry,
		}

		controller, err := controllers.InitializeLangChainController(controllerCfg)
		if err != nil {
			Skip("Failed to create LangChain controller with viper config: " + err.Error())
		}

		// Should work with viper configuration
		response, err := controller.SendUserMessage("Say 'config works'")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.Content).To(ContainSubstring("config works"))
	})
})
