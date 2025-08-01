package tui

import (
	"fmt"
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
	borderStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	drawBorder(screen, area, borderStyle)

	// Calculate content area with padding
	contentArea := Rect{
		X:      area.X + 2,
		Y:      area.Y + 1,
		Width:  area.Width - 4,
		Height: area.Height - 2,
	}

	headerStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	selectedStyle := tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite)
	currentModelStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	header := fmt.Sprintf("%-32s %10s %12s %15s", "NAME", "SIZE", "PARAMETERS", "QUANTIZATION")
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
		if i == display.Selected {
			style = selectedStyle
		} else if model.Name == currentModel {
			style = currentModelStyle
		}

		sizeGB := float64(model.Size) / (1024 * 1024 * 1024)

		// Add status indicator
		statusIcon := "⚪" // stopped
		if model.IsRunning {
			statusIcon = "🟢" // running
		}

		// Add tool compatibility indicator
		toolIcon := ""
		if models.IsRecommendedForTools(model.Name) {
			modelInfo := models.GetModelInfo(model.Name)
			switch modelInfo.ToolCompatibility {
			case models.ToolCompatibilityExcellent:
				toolIcon = "🔧" // Excellent tool support
			case models.ToolCompatibilityGood:
				toolIcon = "⚙️" // Good tool support
			case models.ToolCompatibilityBasic:
				toolIcon = "🔩" // Basic tool support
			}
		}

		nameWithIndicators := statusIcon + toolIcon + " " + truncateString(model.Name, 26)
		line := fmt.Sprintf("%-30s %8.1fGB %12s %15s",
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

	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	normalStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	runningStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)

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

func convertOllamaModelsToModelInfo(models []ollama.Model) []ModelInfo {
	return convertOllamaModelsToModelInfoWithStatus(models, make(map[string]bool))
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
