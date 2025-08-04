package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
)

type VectorStoreView struct {
	controller   *controllers.VectorStoreController
	listDisplay  VectorStoreListDisplay
	statsDisplay VectorStoreStatsDisplay
	status       StatusBar
	layout       Layout
	loading      bool
	screen       tcell.Screen
	showDetails  bool
	selectedInfo *controllers.CollectionInfo
}

func NewVectorStoreView(screen tcell.Screen) *VectorStoreView {
	width, height := screen.Size()

	log := logger.WithComponent("vectorstore_view")
	log.Debug("Creating new VectorStoreView", "width", width, "height", height)

	view := &VectorStoreView{
		controller:   controllers.NewVectorStoreController(),
		listDisplay:  NewVectorStoreListDisplay(width, height-6),
		statsDisplay: NewVectorStoreStatsDisplay(width, 4),
		status:       NewStatusBar(width).WithStatus("Ready"),
		layout:       NewLayout(width, height),
		loading:      false,
		screen:       screen,
		showDetails:  false,
	}

	// Load initial data
	view.refreshData()

	return view
}

func (vv *VectorStoreView) Name() string {
	return "vectorstore"
}

func (vv *VectorStoreView) Description() string {
	return "Vector Store Debug View"
}

func (vv *VectorStoreView) refreshData() {
	log := logger.WithComponent("vectorstore_view")
	log.Debug("Starting vector store data refresh")
	vv.loading = true

	// Get collections
	collections, err := vv.controller.GetCollections()
	if err != nil {
		log.Error("Failed to get collections", "error", err)
		vv.status = vv.status.WithStatus("Error: " + err.Error())
		vv.loading = false
		return
	}
	log.Debug("Retrieved collections", "count", len(collections))

	// Get stats
	stats, err := vv.controller.GetStoreMetadata()
	if err != nil {
		log.Error("Failed to get stats", "error", err)
		vv.status = vv.status.WithStatus("Error: " + err.Error())
		vv.loading = false
		return
	}
	log.Info("Vector store stats", 
		"enabled", stats.IsEnabled, 
		"provider", stats.Provider,
		"collections", stats.TotalCollections,
		"documents", stats.TotalDocuments)

	vv.listDisplay = vv.listDisplay.WithCollections(collections)
	vv.statsDisplay = vv.statsDisplay.WithStats(*stats)
	vv.status = vv.status.WithStatus("Ready")
	vv.loading = false

	log.Debug("Refreshed vector store data", "collections", len(collections), "total_docs", stats.TotalDocuments)
}

func (vv *VectorStoreView) Render(screen tcell.Screen, area Rect) {
	if vv.showDetails && vv.selectedInfo != nil {
		vv.renderDetailsView(screen, area)
		return
	}

	helpHeight := 1
	statsHeight := 4

	listArea := Rect{
		X:      area.X,
		Y:      area.Y,
		Width:  area.Width,
		Height: area.Height - helpHeight - statsHeight - 1,
	}

	statsArea := Rect{
		X:      area.X,
		Y:      listArea.Y + listArea.Height,
		Width:  area.Width,
		Height: statsHeight,
	}

	helpArea := Rect{
		X:      area.X,
		Y:      statsArea.Y + statsArea.Height,
		Width:  area.Width,
		Height: helpHeight,
	}

	// Render components
	RenderVectorStoreList(screen, vv.listDisplay, listArea)
	RenderVectorStoreStats(screen, vv.statsDisplay, statsArea)
	vv.renderHelp(screen, helpArea)
}

func (vv *VectorStoreView) renderDetailsView(screen tcell.Screen, area Rect) {
	clearArea(screen, area)
	drawBorder(screen, area, StyleBorder)

	// Add title
	title := "Collection Details - " + vv.selectedInfo.Name
	titleX := area.X + (area.Width-len(title))/2
	renderText(screen, titleX, area.Y, title, StyleHighlight)

	// Calculate content area
	contentArea := Rect{
		X:      area.X + 2,
		Y:      area.Y + 2,
		Width:  area.Width - 4,
		Height: area.Height - 4,
	}

	normalStyle := StyleMenuNormal
	y := contentArea.Y

	// Collection information
	renderText(screen, contentArea.X, y, "Collection: "+vv.selectedInfo.Name, normalStyle)
	y++

	renderText(screen, contentArea.X, y, fmt.Sprintf("Documents: %d", vv.selectedInfo.DocumentCount), normalStyle)
	y++

	renderText(screen, contentArea.X, y, "Embedder: "+vv.selectedInfo.EmbedderModel, normalStyle)
	y++

	renderText(screen, contentArea.X, y, "Last Updated: "+formatTimeAgo(vv.selectedInfo.LastUpdated), normalStyle)
	y += 2

	// Instructions
	renderText(screen, contentArea.X, y, "Press ESC to return to list", StyleDimText)
}

func (vv *VectorStoreView) renderHelp(screen tcell.Screen, area Rect) {
	if area.Height < 1 {
		return
	}

	helpText := "↑↓: Navigate  Enter: Details  r: Refresh  ESC: Back"
	if vv.loading {
		helpText = "Loading..."
	}

	// Center the help text
	helpX := area.X + (area.Width-len(helpText))/2
	if helpX < area.X {
		helpX = area.X
	}

	renderText(screen, helpX, area.Y, helpText, StyleDimText)
}

func (vv *VectorStoreView) HandleKeyEvent(ev *tcell.EventKey, sending bool) bool {
	log := logger.WithComponent("vectorstore_view")

	if vv.showDetails {
		return vv.handleDetailsKeyEvent(ev)
	}

	switch ev.Key() {
	case tcell.KeyUp, tcell.KeyCtrlP:
		vv.listDisplay = vv.listDisplay.SelectPrevious()
		return true

	case tcell.KeyDown, tcell.KeyCtrlN:
		vv.listDisplay = vv.listDisplay.SelectNext()
		return true

	case tcell.KeyPgUp:
		vv.listDisplay = vv.listDisplay.PageUp()
		return true

	case tcell.KeyPgDn:
		vv.listDisplay = vv.listDisplay.PageDown()
		return true

	case tcell.KeyEnter:
		selected := vv.listDisplay.GetSelectedCollection()
		if selected != nil {
			vv.selectedInfo = selected
			vv.showDetails = true
			log.Debug("Showing details for collection", "name", selected.Name)
		}
		return true

	case tcell.KeyRune:
		switch ev.Rune() {
		case 'r', 'R':
			log.Debug("Refreshing vector store data")
			vv.refreshData()
			return true
		case 'j', 'J':
			vv.listDisplay = vv.listDisplay.SelectNext()
			return true
		case 'k', 'K':
			vv.listDisplay = vv.listDisplay.SelectPrevious()
			return true
		}
	}

	return false
}

func (vv *VectorStoreView) handleDetailsKeyEvent(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		vv.showDetails = false
		vv.selectedInfo = nil
		return true
	}
	return false
}

func (vv *VectorStoreView) HandleResize(width, height int) {
	vv.layout = NewLayout(width, height)
	helpHeight := 1
	statsHeight := 4
	vv.listDisplay = vv.listDisplay.WithSize(width, height-helpHeight-statsHeight-1)
	vv.statsDisplay = vv.statsDisplay.WithSize(width, statsHeight)
	vv.status = vv.status.WithWidth(width)
}
