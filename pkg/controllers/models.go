package controllers

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
)

type OllamaClient interface {
	Tags() (*ollama.TagsResponse, error)
	Ps() (*ollama.PsResponse, error)
	Pull(modelName string) error
	Delete(modelName string) error
}

type ModelsController struct {
	client OllamaClient
}

func NewModelsController(client OllamaClient) *ModelsController {
	return &ModelsController{
		client: client,
	}
}

func (mc *ModelsController) Tags() (*ollama.TagsResponse, error) {
	log := logger.WithComponent("models_controller")
	log.Debug("Calling ollama client Tags()")

	response, err := mc.client.Tags()
	if err != nil {
		log.Error("ollama client Tags() failed", "error", err)
		return nil, err
	}

	log.Debug("ollama client Tags() succeeded", "model_count", len(response.Models))
	return response, nil
}

func (mc *ModelsController) Ps() (*ollama.PsResponse, error) {
	log := logger.WithComponent("models_controller")
	log.Debug("Calling ollama client Ps()")

	response, err := mc.client.Ps()
	if err != nil {
		log.Error("ollama client Ps() failed", "error", err)
		return nil, err
	}

	log.Debug("ollama client Ps() succeeded", "running_count", len(response.Models))
	return response, nil
}

func (mc *ModelsController) ListModels(writer io.Writer) error {
	response, err := mc.client.Tags()
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	if len(response.Models) == 0 {
		fmt.Fprintln(writer, "No models found")
		return nil
	}

	w := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSIZE\tPARAMETER SIZE\tQUANTIZATION")

	for _, model := range response.Models {
		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)
		fmt.Fprintf(w, "%s\t%.1fGB\t%s\t%s\n",
			model.Name,
			sizeGB,
			model.Details.ParameterSize,
			model.Details.QuantizationLevel)
	}

	return w.Flush()
}

func (mc *ModelsController) Pull(modelName string) error {
	log := logger.WithComponent("models_controller")
	log.Debug("Calling ollama client Pull()", "model_name", modelName)

	err := mc.client.Pull(modelName)
	if err != nil {
		log.Error("ollama client Pull() failed", "model_name", modelName, "error", err)
		return err
	}

	log.Debug("ollama client Pull() succeeded", "model_name", modelName)
	return nil
}

func (mc *ModelsController) Delete(modelName string) error {
	log := logger.WithComponent("models_controller")
	log.Debug("Calling ollama client Delete()", "model_name", modelName)

	err := mc.client.Delete(modelName)
	if err != nil {
		log.Error("ollama client Delete() failed", "model_name", modelName, "error", err)
		return err
	}

	log.Debug("ollama client Delete() succeeded", "model_name", modelName)
	return nil
}
