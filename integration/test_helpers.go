package integration

import "os"

// isLangChainCompatibleModel checks if the configured model is known to work well with LangChain agents
func isLangChainCompatibleModel() bool {
	model := os.Getenv("OLLAMA_DEFAULT_MODEL")
	if model == "" {
		model = "qwen3:latest" // default
	}

	// Small models that are known to have issues with LangChain agent parsing
	incompatibleModels := []string{
		"smollm2:135m",
		"smollm2:360m",
		"tinyllama:1.1b",
		"qwen2.5:0.5b",
		"qwen2.5:1.5b",
		"qwen2.5:3b", // Has issues with agent output formatting
	}

	for _, incompatible := range incompatibleModels {
		if model == incompatible {
			return false
		}
	}

	return true
}
