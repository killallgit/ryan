package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
)

// OllamaVersion represents version information from Ollama server
type OllamaVersion struct {
	Version string `json:"version"`
}

// CheckOllamaVersion checks if the Ollama server supports tool calling
func CheckOllamaVersion(ollamaURL string) (string, bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/version", ollamaURL))
	if err != nil {
		return "", false, fmt.Errorf("failed to check Ollama version: %w", err)
	}
	defer resp.Body.Close()

	var version OllamaVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return "", false, fmt.Errorf("failed to decode version response: %w", err)
	}

	// Tool calling was introduced in Ollama 0.4.x, became more stable in 1.0+
	supported := VersionSupportsTools(version.Version)
	return version.Version, supported, nil
}

// VersionSupportsTools checks if a version string indicates tool calling support
func VersionSupportsTools(version string) bool {
	// Extract major and minor version numbers
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 3 {
		return false // Can't parse version, assume no support
	}

	major, err1 := strconv.Atoi(matches[1])
	minor, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return false
	}

	// Tool calling support introduced in 0.4.x, more stable in 1.0+
	if major > 1 {
		return true
	}
	if major == 1 {
		return true
	}
	if major == 0 && minor >= 4 {
		return true
	}

	return false
}
