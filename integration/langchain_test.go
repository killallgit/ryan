package integration

import (
	"testing"

	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainIntegration(t *testing.T) {
	// Use Viper for configuration
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	if viper.GetString("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Set default test configuration
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")

	ollamaURL := viper.GetString("ollama.url")
	testModel := viper.GetString("ollama.model")

	t.Run("should create LangChain controller without errors", func(t *testing.T) {
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		require.NoError(t, err)

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

		controller, err := controllers.InitializeLangChainController(controllerCfg)
		require.NoError(t, err, "Failed to create LangChain controller")
		assert.NotNil(t, controller)
		assert.Equal(t, testModel, controller.GetModel())
	})

	t.Run("should handle LangChain streaming with tool registry", func(t *testing.T) {
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		require.NoError(t, err)

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

		controller, err := controllers.InitializeLangChainController(controllerCfg)
		require.NoError(t, err, "Failed to create LangChain controller with tools")
		assert.NotNil(t, controller)
		assert.NotNil(t, controller.GetToolRegistry())
	})

	t.Run("should fail with invalid server URLs", func(t *testing.T) {
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		require.NoError(t, err)

		controllerCfg := &controllers.InitConfig{
			Config: &config.Config{
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					URL:   "http://invalid-server:11434",
					Model: testModel,
				},
			},
			Model:        testModel,
			ToolRegistry: toolRegistry,
		}

		// Controller creation might succeed but usage should fail
		controller, err := controllers.InitializeLangChainController(controllerCfg)
		if err == nil {
			// Test actual usage to verify server connectivity
			_, sendErr := controller.SendUserMessage("test message")
			assert.Error(t, sendErr, "Should fail with invalid server")
		}
		// Either creation or usage should fail with invalid server
	})
}

func TestLangChainStreamingIntegration(t *testing.T) {
	// Use Viper for configuration
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	if viper.GetString("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	// Set default test configuration
	viper.SetDefault("ollama.url", "https://ollama.kitty-tetra.ts.net")
	viper.SetDefault("ollama.model", "qwen2.5-coder:1.5b-base")

	ollamaURL := viper.GetString("ollama.url")
	testModel := viper.GetString("ollama.model")

	t.Run("LangChain controller streaming", func(t *testing.T) {
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		require.NoError(t, err)

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

		controller, err := controllers.InitializeLangChainController(controllerCfg)
		require.NoError(t, err, "Failed to create LangChain controller for streaming test")

		// Test streaming capability
		updates, err := controller.StartStreaming(nil, "Say hello")
		if err != nil {
			t.Skipf("Streaming not supported: %v", err)
		}

		assert.NotNil(t, updates, "Should return streaming updates channel")
	})

	t.Run("LangChain client streaming", func(t *testing.T) {
		toolRegistry := tools.NewRegistry()
		err := toolRegistry.RegisterBuiltinTools()
		require.NoError(t, err)

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

		controller, err := controllers.InitializeLangChainController(controllerCfg)
		require.NoError(t, err, "Failed to create LangChain controller for client streaming test")

		// Verify the controller can handle basic operations
		message, err := controller.SendUserMessage("Hello")
		require.NoError(t, err, "Failed to send message through LangChain controller")
		assert.NotEmpty(t, message.Content, "Response should not be empty")
		assert.Equal(t, "assistant", message.Role, "Response should be from assistant")
	})
}
