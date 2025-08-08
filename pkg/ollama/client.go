package ollama

import (
	"net/http"

	"github.com/spf13/viper"
)

type OllamaClient struct {
	Url          string
	DefaultModel string
	httpClient   *http.Client
}

func NewClient() *OllamaClient {
	ollamaUrl := viper.GetString("ollama.url")
	defaultModel := viper.GetString("ollama.default_model")
	return &OllamaClient{
		Url:          ollamaUrl,
		DefaultModel: defaultModel,
		httpClient:   &http.Client{},
	}
}
