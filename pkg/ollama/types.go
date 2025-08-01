package ollama

import "time"

type Model struct {
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	Details   Details   `json:"details"`
	ExpiresAt time.Time `json:"expires_at"`
	SizeVram  int64     `json:"size_vram"`
}

type Details struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

type TagsResponse struct {
	Models []Model `json:"models"`
}

type PsResponse struct {
	Models []Model `json:"models"`
}