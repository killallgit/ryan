package ollama

import (
	"os"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/spf13/viper"
	lcollama "github.com/tmc/langchaingo/llms/ollama"
)

type OllamaClient struct {
	*lcollama.LLM
}

func NewClient() *OllamaClient {
	// Use OLLAMA_HOST environment variable, no default
	ollamaUrl := os.Getenv("OLLAMA_HOST")
	if ollamaUrl == "" {
		logger.Fatal("OLLAMA_HOST environment variable is not set")
	}
	ollamaModel := viper.GetString("ollama.default_model")

	logger.Info("Creating Ollama client - URL: %s, Model: %s", ollamaUrl, ollamaModel)

	ollamaLLM, err := lcollama.New(lcollama.WithModel(ollamaModel), lcollama.WithServerURL(ollamaUrl))
	if err != nil {
		logger.Fatal("Failed to create Ollama client: %v", err)
	}

	logger.Debug("Ollama client created successfully")

	return &OllamaClient{
		LLM: ollamaLLM,
	}
}
