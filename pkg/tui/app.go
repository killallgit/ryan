package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/rivo/tview"
)

// ControllerInterface defines the interface that controllers must implement
type ControllerInterface interface {
	SendUserMessage(content string) (chat.Message, error)
	GetHistory() []chat.Message
	GetModel() string
	SetModel(model string)
	AddUserMessage(content string)
	AddErrorMessage(errorMsg string)
	Reset()
	StartStreaming(ctx context.Context, content string) (<-chan controllers.StreamingUpdate, error)
	SetOllamaClient(client any)
	ValidateModel(model string) error
	GetToolRegistry() *tools.Registry
	GetTokenUsage() (promptTokens, responseTokens int)
	CleanThinkingBlocks()
}

// App represents the TUI application
type App struct {
	app        *tview.Application
	pages      *tview.Pages
	controller ControllerInterface
	
	// Views
	chatView        *ChatView
	modelView       *ModelView
	toolsView       *ToolsView
	vectorStoreView *VectorStoreView
	contextTreeView *ContextTreeView
	
	// State
	sending      bool
	streaming    bool
	currentView  string
	
	// Channels
	cancelSend   chan bool
	
	// Config
	config *config.Config
}

// NewApp creates a new TUI application
func NewApp(controller ControllerInterface) (*App, error) {
	log := logger.WithComponent("tui_app")
	log.Debug("Creating new TUI application")
	
	tviewApp := tview.NewApplication()
	cfg := config.Get()
	
	app := &App{
		app:         tviewApp,
		pages:       tview.NewPages(),
		controller:  controller,
		sending:     false,
		streaming:   false,
		currentView: "chat",
		cancelSend:  make(chan bool, 1),
		config:      cfg,
	}
	
	// Initialize views
	if err := app.initializeViews(); err != nil {
		return nil, fmt.Errorf("failed to initialize views: %w", err)
	}
	
	// Setup global key bindings
	app.setupGlobalKeyBindings()
	
	// Set initial focus
	tviewApp.SetRoot(app.pages, true).SetFocus(app.pages)
	
	log.Debug("TUI application created successfully")
	return app, nil
}

// initializeViews creates and registers all application views
func (a *App) initializeViews() error {
	log := logger.WithComponent("tview_app")
	
	// Create chat view
	a.chatView = NewChatView(a.controller, a.app)
	a.chatView.SetSendMessageHandler(func(content string) {
		a.SendMessage(content)
	})
	a.pages.AddPage("chat", a.chatView, true, true)
	log.Debug("Created chat view")
	
	// Create model view
	modelsController := controllers.NewModelsController(nil) // Will be set later
	a.modelView = NewModelView(modelsController, a.controller, a.app)
	a.pages.AddPage("models", a.modelView, true, false)
	log.Debug("Created model view")
	
	// Create tools view
	a.toolsView = NewToolsView(a.controller.GetToolRegistry())
	a.pages.AddPage("tools", a.toolsView, true, false)
	log.Debug("Created tools view")
	
	// Create vector store view
	a.vectorStoreView = NewVectorStoreView()
	a.pages.AddPage("vectorstore", a.vectorStoreView, true, false)
	log.Debug("Created vector store view")
	
	// Create context tree view
	a.contextTreeView = NewContextTreeView()
	a.pages.AddPage("context-tree", a.contextTreeView, true, false)
	log.Debug("Created context tree view")
	
	return nil
}

// setupGlobalKeyBindings configures application-wide keyboard shortcuts
func (a *App) setupGlobalKeyBindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl-P: Toggle command palette/view switcher
		if event.Key() == tcell.KeyCtrlP {
			a.showViewSwitcher()
			return nil
		}
		
		// Ctrl-C: Cancel operation or quit
		if event.Key() == tcell.KeyCtrlC {
			if a.sending {
				// Cancel current operation
				select {
				case a.cancelSend <- true:
				default:
				}
			} else {
				// Quit application
				a.app.Stop()
			}
			return nil
		}
		
		// Escape: Return to chat view or quit
		if event.Key() == tcell.KeyEscape {
			if a.currentView != "chat" {
				a.switchToView("chat")
				return nil
			}
		}
		
		// Ctrl-1 through Ctrl-5: Quick view switching
		if event.Key() >= tcell.KeyCtrlA && event.Key() <= tcell.KeyCtrlE {
			views := []string{"chat", "models", "tools", "vectorstore", "context-tree"}
			index := int(event.Key() - tcell.KeyCtrlA)
			if index < len(views) {
				a.switchToView(views[index])
				return nil
			}
		}
		
		return event
	})
}

// Run starts the tview application
func (a *App) Run() error {
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.app.Stop()
}

// switchToView switches to the specified view
func (a *App) switchToView(viewName string) {
	a.currentView = viewName
	a.pages.SwitchToPage(viewName)
	
	// Update current model in tools view if switching to it
	if viewName == "tools" && a.toolsView != nil {
		a.toolsView.SetCurrentModel(a.controller.GetModel())
	}
}

// showViewSwitcher displays a modal for switching between views
func (a *App) showViewSwitcher() {
	// Create a list of available views
	list := tview.NewList().
		AddItem("Chat", "Main chat interface", '1', func() {
			a.switchToView("chat")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Models", "Manage Ollama models", '2', func() {
			a.switchToView("models")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Tools", "View available tools", '3', func() {
			a.switchToView("tools")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Vector Store", "Manage vector store", '4', func() {
			a.switchToView("vectorstore")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Context Tree", "View conversation tree", '5', func() {
			a.switchToView("context-tree")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Cancel", "Close this menu", 'q', func() {
			a.pages.RemovePage("view-switcher")
		})
	
	list.SetBorder(true).SetTitle("Switch View (Ctrl-P)")
	
	// Create a modal layout for the list
	modal := createModal(list, 40, 15)
	
	// Add as overlay
	a.pages.AddPage("view-switcher", modal, true, true)
}

// createModal creates a centered modal primitive
func createModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

// SendMessage sends a message through the controller
func (a *App) SendMessage(content string) {
	if a.sending {
		return
	}
	
	log := logger.WithComponent("tview_app")
	log.Debug("Sending message", "content", content)
	
	a.sending = true
	a.chatView.SetSending(true)
	
	// Add user message
	a.controller.AddUserMessage(content)
	a.chatView.UpdateMessages()
	
	// Send message in goroutine
	go func() {
		defer func() {
			a.app.QueueUpdateDraw(func() {
				a.sending = false
				a.chatView.SetSending(false)
			})
		}()
		
		// Create context with cancel
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		// Monitor for cancellation
		go func() {
			select {
			case <-a.cancelSend:
				cancel()
			case <-ctx.Done():
			}
		}()
		
		// Start streaming
		updates, err := a.controller.StartStreaming(ctx, content)
		if err != nil {
			log.Error("Failed to start streaming", "error", err)
			a.app.QueueUpdateDraw(func() {
				a.controller.AddErrorMessage(fmt.Sprintf("Error: %v", err))
				a.chatView.UpdateMessages()
			})
			return
		}
		
		// Process updates
		a.processStreamingUpdates(updates)
	}()
}

// processStreamingUpdates handles streaming updates from the controller
func (a *App) processStreamingUpdates(updates <-chan controllers.StreamingUpdate) {
	log := logger.WithComponent("tview_app")
	streamingContent := ""
	
	for update := range updates {
		switch update.Type {
		case controllers.StreamStarted:
			log.Debug("Stream started", "id", update.StreamID)
			a.app.QueueUpdateDraw(func() {
				a.streaming = true
				a.chatView.StartStreaming(update.StreamID)
			})
			
		case controllers.ChunkReceived:
			streamingContent += update.Content
			content := streamingContent // Capture for closure
			a.app.QueueUpdateDraw(func() {
				a.chatView.UpdateStreamingContent(update.StreamID, content)
			})
			
		case controllers.MessageComplete:
			log.Debug("Message complete", "id", update.StreamID)
			finalMsg := update.Message
			a.app.QueueUpdateDraw(func() {
				a.streaming = false
				a.chatView.CompleteStreaming(update.StreamID, finalMsg)
				a.chatView.UpdateMessages()
			})
			
		case controllers.StreamError:
			log.Error("Stream error", "error", update.Error)
			a.app.QueueUpdateDraw(func() {
				a.streaming = false
				a.controller.AddErrorMessage(fmt.Sprintf("Stream error: %v", update.Error))
				a.chatView.UpdateMessages()
			})
		}
	}
}

// UpdateMessages updates the chat view messages
func (a *App) UpdateMessages() {
	if a.chatView != nil {
		a.app.QueueUpdateDraw(func() {
			a.chatView.UpdateMessages()
		})
	}
}

// GetCurrentView returns the name of the current view
func (a *App) GetCurrentView() string {
	return a.currentView
}