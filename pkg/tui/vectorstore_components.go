package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
)

type VectorStoreListDisplay struct {
	Collections []controllers.CollectionInfo
	Selected    int
	Scroll      int
	Width       int
	Height      int
}

func NewVectorStoreListDisplay(width, height int) VectorStoreListDisplay {
	return VectorStoreListDisplay{
		Collections: []controllers.CollectionInfo{},
		Selected:    0,
		Scroll:      0,
		Width:       width,
		Height:      height,
	}
}

func (vld VectorStoreListDisplay) WithCollections(collections []controllers.CollectionInfo) VectorStoreListDisplay {
	return VectorStoreListDisplay{
		Collections: collections,
		Selected:    vld.Selected,
		Scroll:      vld.Scroll,
		Width:       vld.Width,
		Height:      vld.Height,
	}
}

func (vld VectorStoreListDisplay) WithSelection(selected int) VectorStoreListDisplay {
	if selected < 0 {
		selected = 0
	}
	if selected >= len(vld.Collections) && len(vld.Collections) > 0 {
		selected = len(vld.Collections) - 1
	}

	return VectorStoreListDisplay{
		Collections: vld.Collections,
		Selected:    selected,
		Scroll:      vld.Scroll,
		Width:       vld.Width,
		Height:      vld.Height,
	}
}

func (vld VectorStoreListDisplay) WithScroll(scroll int) VectorStoreListDisplay {
	if scroll < 0 {
		scroll = 0
	}

	return VectorStoreListDisplay{
		Collections: vld.Collections,
		Selected:    vld.Selected,
		Scroll:      scroll,
		Width:       vld.Width,
		Height:      vld.Height,
	}
}

func (vld VectorStoreListDisplay) WithSize(width, height int) VectorStoreListDisplay {
	return VectorStoreListDisplay{
		Collections: vld.Collections,
		Selected:    vld.Selected,
		Scroll:      vld.Scroll,
		Width:       width,
		Height:      height,
	}
}

func (vld VectorStoreListDisplay) GetSelectedCollection() *controllers.CollectionInfo {
	if vld.Selected >= 0 && vld.Selected < len(vld.Collections) {
		return &vld.Collections[vld.Selected]
	}
	return nil
}

func (vld VectorStoreListDisplay) SelectNext() VectorStoreListDisplay {
	if len(vld.Collections) == 0 {
		return vld
	}

	newSelected := vld.Selected + 1
	if newSelected >= len(vld.Collections) {
		newSelected = len(vld.Collections) - 1
	}

	// Adjust scroll if needed
	visibleHeight := vld.Height - 4 // Account for header and borders
	if newSelected >= vld.Scroll+visibleHeight {
		vld = vld.WithScroll(newSelected - visibleHeight + 1)
	}

	return vld.WithSelection(newSelected)
}

func (vld VectorStoreListDisplay) SelectPrevious() VectorStoreListDisplay {
	if len(vld.Collections) == 0 {
		return vld
	}

	newSelected := vld.Selected - 1
	if newSelected < 0 {
		newSelected = 0
	}

	// Adjust scroll if needed
	if newSelected < vld.Scroll {
		vld = vld.WithScroll(newSelected)
	}

	return vld.WithSelection(newSelected)
}

func (vld VectorStoreListDisplay) PageDown() VectorStoreListDisplay {
	if len(vld.Collections) == 0 {
		return vld
	}

	pageSize := vld.Height - 4
	newSelected := vld.Selected + pageSize
	if newSelected >= len(vld.Collections) {
		newSelected = len(vld.Collections) - 1
	}

	// Adjust scroll
	if newSelected >= vld.Scroll+pageSize {
		vld = vld.WithScroll(newSelected - pageSize + 1)
	}

	return vld.WithSelection(newSelected)
}

func (vld VectorStoreListDisplay) PageUp() VectorStoreListDisplay {
	if len(vld.Collections) == 0 {
		return vld
	}

	pageSize := vld.Height - 4
	newSelected := vld.Selected - pageSize
	if newSelected < 0 {
		newSelected = 0
	}

	// Adjust scroll
	if newSelected < vld.Scroll {
		vld = vld.WithScroll(newSelected)
	}

	return vld.WithSelection(newSelected)
}

func RenderVectorStoreList(screen tcell.Screen, display VectorStoreListDisplay, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)
	drawBorder(screen, area, StyleBorder)

	// Add title
	title := "Vector Store Collections"
	titleX := area.X + (area.Width-len(title))/2
	renderText(screen, titleX, area.Y, title, StyleHighlight)

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
	dimStyle := StyleDimText

	// Fixed header with consistent column widths
	header := fmt.Sprintf("%-30s %10s %-25s %15s", "COLLECTION", "DOCUMENTS", "EMBEDDER", "LAST UPDATED")
	if len(header) > contentArea.Width {
		header = header[:contentArea.Width]
	}
	renderText(screen, contentArea.X, contentArea.Y, header, headerStyle)

	startY := contentArea.Y + 2
	visibleHeight := contentArea.Height - 2

	for i := display.Scroll; i < len(display.Collections) && i-display.Scroll < visibleHeight; i++ {
		collection := display.Collections[i]
		y := startY + (i - display.Scroll)

		style := normalStyle
		if i == display.Selected {
			style = selectedStyle
		}

		// Format last updated time
		lastUpdated := formatTimeAgo(collection.LastUpdated)

		// Format document count
		docCount := fmt.Sprintf("%d", collection.DocumentCount)
		if collection.DocumentCount < 0 {
			docCount = "?"
		}

		line := fmt.Sprintf("%-30s %10s %-25s %15s",
			truncateString(collection.Name, 30),
			docCount,
			truncateString(collection.EmbedderModel, 25),
			lastUpdated,
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

	// Show empty state if no collections
	if len(display.Collections) == 0 {
		emptyMsg := "No collections found"
		renderText(screen, contentArea.X+(contentArea.Width-len(emptyMsg))/2, startY+2, emptyMsg, dimStyle)
	}
}

type VectorStoreStatsDisplay struct {
	Stats  controllers.VectorStoreStats
	Width  int
	Height int
}

func NewVectorStoreStatsDisplay(width, height int) VectorStoreStatsDisplay {
	return VectorStoreStatsDisplay{
		Stats:  controllers.VectorStoreStats{},
		Width:  width,
		Height: height,
	}
}

func (vsd VectorStoreStatsDisplay) WithStats(stats controllers.VectorStoreStats) VectorStoreStatsDisplay {
	return VectorStoreStatsDisplay{
		Stats:  stats,
		Width:  vsd.Width,
		Height: vsd.Height,
	}
}

func (vsd VectorStoreStatsDisplay) WithSize(width, height int) VectorStoreStatsDisplay {
	return VectorStoreStatsDisplay{
		Stats:  vsd.Stats,
		Width:  width,
		Height: height,
	}
}

func RenderVectorStoreStats(screen tcell.Screen, display VectorStoreStatsDisplay, area Rect) {
	if area.Width <= 0 || area.Height <= 0 {
		return
	}

	clearArea(screen, area)

	titleStyle := StyleHighlight
	normalStyle := StyleMenuNormal

	// Status line
	status := "Disabled"
	if display.Stats.IsEnabled {
		status = "Enabled"
	}
	statusText := fmt.Sprintf("Vector Store: %s", status)
	renderText(screen, area.X, area.Y, statusText, titleStyle)

	if display.Stats.IsEnabled {
		// Provider and persistence info
		providerText := fmt.Sprintf("Provider: %s | Collections: %d | Documents: %d",
			display.Stats.Provider,
			display.Stats.TotalCollections,
			display.Stats.TotalDocuments)
		renderText(screen, area.X, area.Y+1, providerText, normalStyle)

		// Persistence directory
		if display.Stats.PersistenceDir != "" {
			persistText := fmt.Sprintf("Persistence: %s", truncateString(display.Stats.PersistenceDir, area.Width-13))
			renderText(screen, area.X, area.Y+2, persistText, normalStyle)
		}
	}
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}
