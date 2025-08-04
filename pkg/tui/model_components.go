package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/models"
	"github.com/killallgit/ryan/pkg/ollama"
)

type ModelInfo struct {
	Name              string
	Size              int64
	ParameterSize     string
	QuantizationLevel string
	ModifiedAt        time.Time
	IsRunning         bool
	IsDownloaded      bool
}

type RunningModelInfo struct {
	Name      string
	Size      int64
	SizeVRAM  int64
	UntilTime time.Time
}

type ModelStats struct {
	TotalModels   int
	RunningModels []RunningModelInfo
	TotalSize     int64
}

type ModelListDisplay struct {
	Models   []ModelInfo
	Selected int
	Scroll   int
	Width    int
	Height   int
}

func NewModelListDisplay(width, height int) ModelListDisplay {
	return ModelListDisplay{
		Models:   []ModelInfo{},
		Selected: 0,
		Scroll:   0,
		Width:    width,
		Height:   height,
	}
}

func (mld ModelListDisplay) WithModels(models []ModelInfo) ModelListDisplay {
	return ModelListDisplay{
		Models:   models,
		Selected: mld.Selected,
		Scroll:   mld.Scroll,
		Width:    mld.Width,
		Height:   mld.Height,
	}
}

func (mld ModelListDisplay) WithSelection(selected int) ModelListDisplay {
	if selected < 0 {
		selected = 0
	}
	if selected >= len(mld.Models) && len(mld.Models) > 0 {
		selected = len(mld.Models) - 1
	}

	return ModelListDisplay{
		Models:   mld.Models,
		Selected: selected,
		Scroll:   mld.Scroll,
		Width:    mld.Width,
		Height:   mld.Height,
	}
}

func (mld ModelListDisplay) WithScroll(scroll int) ModelListDisplay {
	if scroll < 0 {
		scroll = 0
	}

	return ModelListDisplay{
		Models:   mld.Models,
		Selected: mld.Selected,
		Scroll:   scroll,
		Width:    mld.Width,
		Height:   mld.Height,
	}
}

func (mld ModelListDisplay) WithSize(width, height int) ModelListDisplay {
	return ModelListDisplay{
		Models:   mld.Models,
		Selected: mld.Selected,
		Scroll:   mld.Scroll,
		Width:    width,
		Height:   height,
	}
}

type ModelStatsDisplay struct {
	Stats  ModelStats
	Width  int
	Height int
}

func NewModelStatsDisplay(width, height int) ModelStatsDisplay {
	return ModelStatsDisplay{
		Stats:  ModelStats{},
		Width:  width,
		Height: height,
	}
}

func (msd ModelStatsDisplay) WithStats(stats ModelStats) ModelStatsDisplay {
	return ModelStatsDisplay{
		Stats:  stats,
		Width:  msd.Width,
		Height: msd.Height,
	}
}

func (msd ModelStatsDisplay) WithSize(width, height int) ModelStatsDisplay {
	return ModelStatsDisplay{
		Stats:  msd.Stats,
		Width:  width,
		Height: height,
	}
}

func RenderModelList(screen tcell.Screen, display ModelListDisplay, area Rect) {
	RenderModelListWithCurrentModel(screen, display, area, "")
}

func RenderModelListWithCurrentModel(screen tcell.Screen, display ModelListDisplay, area Rect, currentModel string) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	// Draw border
	borderStyle := StyleBorder
	drawBorder(screen, area, borderStyle)

	// Calculate content area with padding
	contentArea := Rect{
		X:      area.X + 2,
		Y:      area.Y + 1,
		Width:  area.Width - 4,
		Height: area.Height - 2,
	}

	headerStyle := StyleHeaderText
	normalStyle := StyleMenuNormal
	selectedStyle := tcell.StyleDefault.Background(ColorModelSelected).Foreground(tcell.ColorBlack)
	currentModelStyle := tcell.StyleDefault.Foreground(ColorHighlight)
	availableStyle := StyleDimText

	// Fixed header with consistent column widths
	header := fmt.Sprintf("%-35s %10s %12s %15s", "NAME", "SIZE", "PARAMETERS", "QUANTIZATION")
	if len(header) > contentArea.Width {
		header = header[:contentArea.Width]
	}
	renderText(screen, contentArea.X, contentArea.Y, header, headerStyle)

	startY := contentArea.Y + 2
	visibleHeight := contentArea.Height - 2

	for i := display.Scroll; i < len(display.Models) && i-display.Scroll < visibleHeight; i++ {
		model := display.Models[i]
		y := startY + (i - display.Scroll)

		style := normalStyle
		if !model.IsDownloaded {
			style = availableStyle
		}
		if i == display.Selected {
			style = selectedStyle
		} else if model.Name == currentModel && model.IsDownloaded {
			style = currentModelStyle
		}

		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)

		// Add status indicator (ASCII characters for better compatibility)
		statusIcon := "o" // stopped
		if !model.IsDownloaded {
			statusIcon = "." // available for download
		} else if model.IsRunning {
			statusIcon = "*" // running
		}

		// Add tool compatibility indicator (ASCII characters)
		toolIcon := " "
		if models.IsRecommendedForTools(model.Name) {
			modelInfo := models.GetModelInfo(model.Name)
			switch modelInfo.ToolCompatibility {
			case models.ToolCompatibilityExcellent:
				toolIcon = "+" // Excellent tool support
			case models.ToolCompatibilityGood:
				toolIcon = "~" // Good tool support
			case models.ToolCompatibilityBasic:
				toolIcon = "-" // Basic tool support
			}
		}

		// Build name with fixed-width indicators (2 chars: status + tool)
		nameWithIndicators := statusIcon + toolIcon + " " + truncateString(model.Name, 31)
		line := fmt.Sprintf("%-35s %8.1fGB %12s %15s",
			nameWithIndicators,
			sizeGB,
			model.ParameterSize,
			model.QuantizationLevel,
		)

		if len(line) > contentArea.Width {
			line = line[:contentArea.Width]
		}

		for x := 0; x < contentArea.Width; x++ {
			if x < len(line) {
				screen.SetContent(contentArea.X+x, y, rune(line[x]), nil, style)
			} else {
				screen.SetContent(contentArea.X+x, y, ' ', nil, style)
			}
		}
	}
}

func RenderModelStats(screen tcell.Screen, display ModelStatsDisplay, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	titleStyle := StyleHighlight
	normalStyle := StyleMenuNormal
	runningStyle := tcell.StyleDefault.Foreground(ColorModelRunning)

	title := "System Statistics"
	renderText(screen, area.X, area.Y, title, titleStyle)

	totalSizeGB := float64(display.Stats.TotalSize) / (1024 * 1024 * 1024)
	statsText := fmt.Sprintf("Total Models: %d  |  Total Size: %.1f GB",
		display.Stats.TotalModels, totalSizeGB)
	renderText(screen, area.X, area.Y+2, statsText, normalStyle)

	if len(display.Stats.RunningModels) > 0 {
		runningTitle := "Currently Running:"
		renderText(screen, area.X, area.Y+4, runningTitle, titleStyle)

		for i, running := range display.Stats.RunningModels {
			if area.Y+5+i >= area.Y+area.Height {
				break
			}

			sizeGB := float64(running.Size) / (1024 * 1024 * 1024)
			sizeVRAMGB := float64(running.SizeVRAM) / (1024 * 1024 * 1024)

			runningText := fmt.Sprintf("  %s (%.1fGB, VRAM: %.1fGB)",
				running.Name, sizeGB, sizeVRAMGB)
			renderText(screen, area.X, area.Y+5+i, runningText, runningStyle)
		}
	} else {
		noRunningText := "No models currently running"
		renderText(screen, area.X, area.Y+4, noRunningText, normalStyle)
	}
}

func convertOllamaModelsToModelInfoWithStatus(models []ollama.Model, runningModels map[string]bool) []ModelInfo {
	result := make([]ModelInfo, len(models))
	for i, model := range models {
		result[i] = ModelInfo{
			Name:              model.Name,
			Size:              model.Size,
			ParameterSize:     model.Details.ParameterSize,
			QuantizationLevel: model.Details.QuantizationLevel,
			ModifiedAt:        time.Now(), // Use current time since ModifiedAt isn't available
			IsRunning:         runningModels[model.Name],
			IsDownloaded:      true, // These are models from ollama.tags() so they're downloaded
		}
	}
	return result
}

func convertOllamaPsToRunningModelInfo(models []ollama.Model) []RunningModelInfo {
	result := make([]RunningModelInfo, len(models))
	for i, model := range models {
		result[i] = RunningModelInfo{
			Name:      model.Name,
			Size:      model.Size,
			SizeVRAM:  model.SizeVram,
			UntilTime: model.ExpiresAt,
		}
	}
	return result
}

func createAvailableModelInfo() []ModelInfo {
	// Model information with estimated sizes and compatibility info
	availableModels := []struct {
		name              string
		estimatedSizeGB   float64
		parameterSize     string
		quantizationLevel string
	}{
		{"llama3.2:3b", 2.0, "3B", "Q4_0"},
		{"llama3.2:1b", 0.8, "1B", "Q4_0"},
		{"llama3.2-vision:latest", 7.9, "9.8B", "Q4_K_M"},
		{"qwen2.5:7b", 4.1, "7B", "Q4_0"},
		{"qwen2.5:3b", 1.9, "3B", "Q4_0"},
		{"qwen2.5:1.5b", 0.9, "1.5B", "Q4_0"},
		{"qwen2.5vl:latest", 6.0, "8.3B", "Q4_K_M"},
		{"qwen3:latest", 5.2, "8.2B", "Q4_K_M"},
		{"mistral:7b", 4.1, "7B", "Q4_0"},
		{"deepseek-coder:1.3b", 0.9, "1.3B", "Q4_0"},
		{"deepseek-r1:latest", 5.2, "8.2B", "Q4_K_M"},
		{"codellama:7b", 3.8, "7B", "Q4_0"},
		{"llama3.1:8b", 4.7, "8B", "Q4_0"},
		{"command-r", 20.0, "35B", "Q4_0"},
		{"qwen2.5-coder:7b", 4.1, "7B", "Q4_0"},
		{"granite3.2:8b", 4.7, "8B", "Q4_0"},
	}

	result := make([]ModelInfo, 0, len(availableModels))
	for _, model := range availableModels {
		// Get rich model information from the models package
		modelInfo := models.GetModelInfo(model.name)

		// Only include models that have good tool support
		if modelInfo.RecommendedForTools {
			result = append(result, ModelInfo{
				Name:              model.name,
				Size:              int64(model.estimatedSizeGB * 1024 * 1024 * 1024), // Convert GB to bytes
				ParameterSize:     model.parameterSize,
				QuantizationLevel: model.quantizationLevel,
				ModifiedAt:        time.Time{},
				IsRunning:         false,
				IsDownloaded:      false,
			})
		}
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// normalizeModelNameForComparison normalizes model names for comparison
// This helps match models that might have slight variations in naming
func normalizeModelNameForComparison(name string) string {
	// Convert to lowercase and trim whitespace
	normalized := strings.ToLower(strings.TrimSpace(name))

	// Remove common suffixes that don't affect the core model identity
	suffixes := []string{"-q4_0", "-q4_k_m", "-q8_0", "-fp16", "-instruct", "-chat"}
	for _, suffix := range suffixes {
		normalized = strings.TrimSuffix(normalized, suffix)
	}

	return normalized
}

// isModelAlreadyDownloaded checks if a model is already downloaded using normalized matching
func isModelAlreadyDownloaded(availableModelName string, downloadedModelNames map[string]bool) bool {
	// First try exact match
	if downloadedModelNames[availableModelName] {
		return true
	}

	// Then try normalized matching
	normalizedAvailable := normalizeModelNameForComparison(availableModelName)
	for downloadedName := range downloadedModelNames {
		normalizedDownloaded := normalizeModelNameForComparison(downloadedName)
		if normalizedAvailable == normalizedDownloaded {
			return true
		}
	}

	return false
}

// sortModelsByDownloadStatus sorts models with downloaded models first, then available models
// Within each group, models are sorted alphabetically by name
func sortModelsByDownloadStatus(models []ModelInfo) {
	sort.Slice(models, func(i, j int) bool {
		modelA := models[i]
		modelB := models[j]

		// Primary sort: Downloaded models come first
		if modelA.IsDownloaded != modelB.IsDownloaded {
			return modelA.IsDownloaded // true comes before false
		}

		// Secondary sort: Within each group, sort alphabetically by name
		return strings.ToLower(modelA.Name) < strings.ToLower(modelB.Name)
	})
}
