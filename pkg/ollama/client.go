package ollama

import (
	"log"

	"github.com/spf13/viper"
	lcollama "github.com/tmc/langchaingo/llms/ollama"
)

type OllamaClient struct {
	*lcollama.LLM
}

func NewClient() *OllamaClient {
	ollamaUrl := viper.GetString("ollama.url")
	ollamaModel := viper.GetString("ollama.default_model")
	ollamaLLM, err := lcollama.New(lcollama.WithModel(ollamaModel), lcollama.WithServerURL(ollamaUrl))
	if err != nil {
		log.Fatal(err)
	}
	return &OllamaClient{
		LLM: ollamaLLM,
	}
}
