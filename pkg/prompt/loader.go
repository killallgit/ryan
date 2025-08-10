package prompt

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/prompts"
	"gopkg.in/yaml.v3"
)

// FileLoader loads templates from files
type FileLoader struct {
	baseDir string
	config  *Config
}

// NewFileLoader creates a new file-based template loader
func NewFileLoader(baseDir string) *FileLoader {
	return &FileLoader{
		baseDir: baseDir,
		config:  &Config{},
	}
}

// NewFileLoaderWithConfig creates a new file loader with configuration
func NewFileLoaderWithConfig(baseDir string, config *Config) *FileLoader {
	return &FileLoader{
		baseDir: baseDir,
		config:  config,
	}
}

// Load loads a template by name/path
func (f *FileLoader) Load(name string) (Template, error) {
	path := f.resolvePath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Check if it's a structured template file (JSON/YAML)
	if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return f.loadStructuredTemplate(data, path)
	}

	// Otherwise treat as raw template string
	return f.loadRawTemplate(string(data))
}

// LoadChat loads a chat template by name/path
func (f *FileLoader) LoadChat(name string) (ChatTemplate, error) {
	path := f.resolvePath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Chat templates must be structured (JSON/YAML)
	if !strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		return nil, fmt.Errorf("chat templates must be in JSON or YAML format")
	}

	return f.loadStructuredChatTemplate(data, path)
}

// resolvePath resolves the template path
func (f *FileLoader) resolvePath(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(f.baseDir, name)
}

// loadRawTemplate loads a raw text template
func (f *FileLoader) loadRawTemplate(content string) (Template, error) {
	// Extract variables from template (simple {{.var}} pattern)
	vars := extractVariables(content)

	return NewPromptTemplate(content, vars), nil
}

// loadStructuredTemplate loads a structured template from JSON/YAML
func (f *FileLoader) loadStructuredTemplate(data []byte, path string) (Template, error) {
	var spec TemplateSpec

	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON template: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML template: %w", err)
		}
	}

	// Create template with metadata
	template := NewPromptTemplate(spec.Template, spec.Variables)

	// Set variable metadata
	for _, v := range spec.Metadata {
		template.SetVariableMetadata(v)
	}

	// Set partial variables
	if len(spec.Partials) > 0 {
		template = template.WithPartialVariables(spec.Partials).(*PromptTemplate)
	}

	return template, nil
}

// loadStructuredChatTemplate loads a structured chat template
func (f *FileLoader) loadStructuredChatTemplate(data []byte, path string) (ChatTemplate, error) {
	var spec ChatTemplateSpec

	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON chat template: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML chat template: %w", err)
		}
	}

	return NewChatTemplateFromMessages(spec.Messages)
}

// EmbedLoader loads templates from embedded files
type EmbedLoader struct {
	fs     embed.FS
	prefix string
}

// NewEmbedLoader creates a new embedded template loader
func NewEmbedLoader(fs embed.FS, prefix string) *EmbedLoader {
	return &EmbedLoader{
		fs:     fs,
		prefix: prefix,
	}
}

// Load loads a template from embedded files
func (e *EmbedLoader) Load(name string) (Template, error) {
	path := filepath.Join(e.prefix, name)

	file, err := e.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded template: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template: %w", err)
	}

	// Simple template - just extract variables
	content := string(data)
	vars := extractVariables(content)

	return NewPromptTemplate(content, vars), nil
}

// LoadChat loads a chat template from embedded files
func (e *EmbedLoader) LoadChat(name string) (ChatTemplate, error) {
	path := filepath.Join(e.prefix, name)

	data, err := e.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded chat template: %w", err)
	}

	var spec ChatTemplateSpec
	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON chat template: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML chat template: %w", err)
		}
	}

	return NewChatTemplateFromMessages(spec.Messages)
}

// StringLoader loads templates from strings
type StringLoader struct {
	templates map[string]string
	chats     map[string][]MessageDefinition
}

// NewStringLoader creates a new string-based template loader
func NewStringLoader() *StringLoader {
	return &StringLoader{
		templates: make(map[string]string),
		chats:     make(map[string][]MessageDefinition),
	}
}

// AddTemplate adds a string template
func (s *StringLoader) AddTemplate(name string, template string, vars []string) {
	s.templates[name] = template
}

// AddChatTemplate adds a chat template
func (s *StringLoader) AddChatTemplate(name string, messages []MessageDefinition) {
	s.chats[name] = messages
}

// Load loads a template by name
func (s *StringLoader) Load(name string) (Template, error) {
	template, exists := s.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}

	vars := extractVariables(template)
	return NewPromptTemplate(template, vars), nil
}

// LoadChat loads a chat template by name
func (s *StringLoader) LoadChat(name string) (ChatTemplate, error) {
	messages, exists := s.chats[name]
	if !exists {
		return nil, fmt.Errorf("chat template %s not found", name)
	}

	return NewChatTemplateFromMessages(messages)
}

// TemplateSpec defines the structure of a template file
type TemplateSpec struct {
	Name      string         `json:"name" yaml:"name"`
	Template  string         `json:"template" yaml:"template"`
	Variables []string       `json:"variables" yaml:"variables"`
	Metadata  []*Variable    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Partials  map[string]any `json:"partials,omitempty" yaml:"partials,omitempty"`
}

// ChatTemplateSpec defines the structure of a chat template file
type ChatTemplateSpec struct {
	Name     string              `json:"name" yaml:"name"`
	Messages []MessageDefinition `json:"messages" yaml:"messages"`
}

// extractVariables extracts variable names from a template string
func extractVariables(template string) []string {
	varMap := make(map[string]bool)

	// Simple extraction for {{.var}} pattern
	start := 0
	for {
		idx := strings.Index(template[start:], "{{.")
		if idx == -1 {
			break
		}
		start += idx + 3

		end := strings.Index(template[start:], "}}")
		if end == -1 {
			break
		}

		varName := strings.TrimSpace(template[start : start+end])
		if varName != "" {
			varMap[varName] = true
		}
		start += end + 2
	}

	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}

	return vars
}

// QuickTemplate creates a simple template from a string
func QuickTemplate(template string) Template {
	vars := extractVariables(template)
	return NewPromptTemplate(template, vars)
}

// QuickChatTemplate creates a simple chat template
func QuickChatTemplate(systemPrompt, humanPrompt string) ChatTemplate {
	messages := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(systemPrompt, extractVariables(systemPrompt)),
		prompts.NewHumanMessagePromptTemplate(humanPrompt, extractVariables(humanPrompt)),
	}

	return NewChatTemplate(messages)
}
