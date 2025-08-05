package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/killallgit/ryan/pkg/tools"
	"github.com/killallgit/ryan/pkg/vectorstore"
)

type App struct {
	screen          tcell.Screen
	controller      ControllerInterface
	input           InputField
	messages        MessageDisplay
	status          StatusBar
	layout          Layout
	quit            bool
	sending         bool          // Track if we're currently sending a message
	sendStartTime   time.Time     // Track when message sending started
	timeout         time.Duration // Request timeout duration
	cancelSend      chan bool     // Channel to cancel current send operation
	viewManager     *ViewManager
	chatView        *ChatView
	contextTreeView *ContextTreeStandaloneView
	spinnerTicker   *time.Ticker
	spinnerStop     chan bool
	modal           ModalDialog

	// Streaming state
	streaming         bool                     // Track if we're currently streaming
	currentStreamID   string                   // Current stream identifier
	streamingContent  string                   // Accumulating streaming content
	streamAccumulator *chat.MessageAccumulator // Message accumulator for streaming
}

func checkOllamaConnectivity(baseURL string) error {
	log := logger.WithComponent("tui_app")
	log.Debug("Checking Ollama connectivity", "url", baseURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%s/api/tags", baseURL))
	if err != nil {
		return fmt.Errorf("Cannot connect to Ollama at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d - is it running?", resp.StatusCode)
	}

	log.Debug("Ollama connectivity check successful")
	return nil
}

// ControllerInterface defines the interface that both ChatController and LangChainController implement
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

func NewApp(controller ControllerInterface) (*App, error) {
	log := logger.WithComponent("tui_app")
	log.Debug("Creating new TUI application")

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Error("Failed to create tcell screen", "error", err)
		return nil, err
	}

	if err := screen.Init(); err != nil {
		log.Error("Failed to initialize tcell screen", "error", err)
		return nil, err
	}

	width, height := screen.Size()
	log.Debug("Screen initialized", "width", width, "height", height)

	cfg := config.Get()
	log.Debug("Creating ollama client for models", "url", cfg.Ollama.URL)

	// Check Ollama connectivity before proceeding
	var connectivityError error
	if err := checkOllamaConnectivity(cfg.Ollama.URL); err != nil {
		log.Error("Ollama connectivity check failed", "error", err)
		connectivityError = err
	}

	ollamaClient := ollama.NewClientWithTimeout(cfg.Ollama.URL, cfg.Ollama.Timeout)
	modelsController := controllers.NewModelsController(ollamaClient)

	// Connect ollama client to chat controller for model validation
	controller.SetOllamaClient(ollamaClient)

	// Initialize vector store if enabled
	if cfg.VectorStore.Enabled {
		log.Debug("Initializing vector store")
		manager, err := vectorstore.GetGlobalManager()
		if err != nil {
			log.Warn("Failed to initialize vector store", "error", err)
		} else if manager != nil {
			log.Info("Vector store initialized successfully",
				"provider", cfg.VectorStore.Provider,
				"persistence", cfg.VectorStore.EnablePersistence,
				"embedder", cfg.VectorStore.Embedder.Provider+"/"+cfg.VectorStore.Embedder.Model)
		}
	} else {
		log.Info("Vector store is disabled")
	}

	viewManager := NewViewManager()
	chatView := NewChatView(controller, modelsController, screen)
	modelView := NewModelView(modelsController, controller, screen)
	vectorStoreView := NewVectorStoreView(screen)
	contextTreeView := NewContextTreeStandaloneView()

	toolsView := NewToolsView(controller.GetToolRegistry())
	toolsView.SetCurrentModel(controller.GetModel())

	viewManager.RegisterView("chat", chatView)
	viewManager.RegisterView("models", modelView)
	viewManager.RegisterView("vectorstore", vectorStoreView)
	viewManager.RegisterView("context-tree", contextTreeView)
	viewManager.RegisterView("tools", toolsView)
	log.Debug("Registered views with view manager", "views", []string{"chat", "models", "vectorstore", "context-tree", "tools"})

	app := &App{
		screen:          screen,
		controller:      controller,
		input:           NewInputField(width),
		messages:        NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:          NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready"),
		layout:          NewLayout(width, height),
		quit:            false,
		sending:         false,
		timeout:         cfg.Ollama.Timeout,
		cancelSend:      make(chan bool, 1), // Buffered channel for cancellation
		viewManager:     viewManager,
		chatView:        chatView,
		contextTreeView: contextTreeView,
		spinnerTicker:   time.NewTicker(100 * time.Millisecond), // Faster animation for smoother spinner
		spinnerStop:     make(chan bool),
		modal:           NewModalDialog(),

		// Initialize streaming state
		streaming:         false,
		currentStreamID:   "",
		streamingContent:  "",
		streamAccumulator: chat.NewMessageAccumulator(),
	}

	app.updateMessages()

	// Show connectivity error modal if there was an issue
	if connectivityError != nil {
		app.modal = app.modal.WithError("Ollama Connection Error",
			fmt.Sprintf("Cannot connect to Ollama at %s\n\n%v\n\nPress any key to continue with limited functionality.",
				cfg.Ollama.URL, connectivityError))
	}

	// Start spinner animation timer
	go app.runSpinnerTimer()

	log.Debug("TUI application created successfully")

	return app, nil
}

func (app *App) Run() error {
	defer app.screen.Fini()
	defer app.cleanup()

	app.render()

	for !app.quit {
		event := app.screen.PollEvent()
		app.handleEvent(event)
		app.render()
	}

	return nil
}

func (app *App) cleanup() {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting app cleanup")

	// Stop spinner timer
	if app.spinnerTicker != nil {
		app.spinnerTicker.Stop()
	}
	if app.spinnerStop != nil {
		close(app.spinnerStop)
	}

	log.Debug("App cleanup complete")
}

func (app *App) runSpinnerTimer() {
	for {
		select {
		case <-app.spinnerTicker.C:
			// Only animate when sending
			if app.sending {
				app.screen.PostEvent(NewSpinnerAnimationEvent())
			}
		case <-app.spinnerStop:
			return
		}
	}
}

func (app *App) handleEvent(event tcell.Event) {
	switch ev := event.(type) {
	case *tcell.EventKey:
		app.handleKeyEvent(ev)
	case *tcell.EventMouse:
		app.handleMouseEvent(ev)
	case *tcell.EventResize:
		app.handleResize(ev)
	case *MessageResponseEvent:
		app.handleMessageResponse(ev)
	case *MessageErrorEvent:
		app.handleMessageError(ev)
	case *ViewChangeEvent:
		app.handleViewChange(ev)
	case *MenuToggleEvent:
		app.handleMenuToggle(ev)
	case *ModelListUpdateEvent:
		app.handleModelListUpdate(ev)
	case *ModelStatsUpdateEvent:
		app.handleModelStatsUpdate(ev)
	case *ModelErrorEvent:
		app.handleModelError(ev)
	case *ModelDeletedEvent:
		app.handleModelDeleted(ev)
	case *ChatMessageSendEvent:
		app.handleChatMessageSend(ev)
	case *SpinnerAnimationEvent:
		app.handleSpinnerAnimation(ev)
	case *ModelDownloadProgressEvent:
		app.handleModelDownloadProgress(ev)
	case *ModelDownloadCompleteEvent:
		app.handleModelDownloadComplete(ev)
	case *ModelDownloadErrorEvent:
		app.handleModelDownloadError(ev)
	case *ModelNotFoundEvent:
		app.handleModelNotFound(ev)
	case *ModelChangeEvent:
		app.handleModelChange(ev)
	case *MessageChunkEvent:
		app.handleMessageChunk(ev)
	case *StreamStartEvent:
		app.handleStreamStart(ev)
	case *StreamCompleteEvent:
		app.handleStreamComplete(ev)
	case *StreamErrorEvent:
		app.handleStreamError(ev)
	case *StreamProgressEvent:
		app.handleStreamProgress(ev)
	}
}

func (app *App) handleKeyEvent(ev *tcell.EventKey) {
	log := logger.WithComponent("tui_app")

	// Handle modal first
	if app.modal.Visible {
		app.modal = app.modal.Hide()
		log.Debug("Modal dismissed")
		return
	}

	if app.viewManager.HandleMenuKeyEvent(ev) {
		return
	}

	// Handle critical app-level shortcuts first (before views)
	switch ev.Key() {
	case tcell.KeyCtrlP:
		app.viewManager.ToggleMenu()
		log.Debug("Menu toggled via Ctrl+P")
		return
	case tcell.KeyCtrlC:
		// Handle Ctrl-C for canceling or quitting
		if app.viewManager.IsMenuVisible() {
			app.viewManager.HideMenu()
			app.viewManager.SyncViewState(app.sending)
			log.Debug("Menu hidden via Ctrl+C, state synced")
		} else if app.sending {
			// Cancel the current send operation
			select {
			case app.cancelSend <- true:
				log.Debug("Cancellation signal sent")
			default:
				log.Debug("Cancellation channel full, already cancelling")
			}
		} else {
			app.quit = true
			log.Debug("Application quit triggered via Ctrl+C")
		}
		return
	}

	// Let the current view handle the key event
	currentView := app.viewManager.GetCurrentView()
	if currentView != nil {
		if currentView.HandleKeyEvent(ev, app.sending) {
			// Event was consumed by the view, don't handle it at app level
			return
		}
	}

	// Handle remaining app-level key events only if not consumed by the current view
	switch ev.Key() {
	case tcell.KeyEscape:
		if app.viewManager.IsMenuVisible() {
			app.viewManager.HideMenu()
			app.viewManager.SyncViewState(app.sending)
			log.Debug("Menu hidden via Escape, state synced")
		} else if app.sending {
			// Cancel the current send operation
			select {
			case app.cancelSend <- true:
				log.Debug("Cancellation signal sent")
			default:
				log.Debug("Cancellation channel full, already cancelling")
			}
		} else if app.viewManager.GetCurrentViewName() != "chat" {
			// Switch back to chat view if not already there
			app.viewManager.SetCurrentView("chat")
			app.viewManager.SyncViewState(app.sending)
			log.Debug("Switched to chat view via Escape")
		} else {
			app.quit = true
			log.Debug("Application quit triggered")
		}
	}
}

func (app *App) handleMouseEvent(ev *tcell.EventMouse) {
	log := logger.WithComponent("tui_app")

	// Get mouse coordinates
	x, y := ev.Position()
	buttons := ev.Buttons()

	log.Debug("Mouse event received", "x", x, "y", y, "buttons", buttons)

	// Handle modal first - if modal is visible, consume the click
	if app.modal.Visible {
		app.modal = app.modal.Hide()
		log.Debug("Modal dismissed by mouse click")
		return
	}

	// Handle menu clicks
	if app.viewManager.IsMenuVisible() {
		// Let the menu handle the mouse event
		if app.viewManager.HandleMenuMouseEvent(ev) {
			return
		}
	}

	// Only handle left mouse button clicks for now
	if buttons&tcell.ButtonPrimary != 0 {
		// Let the current view handle the mouse event
		currentView := app.viewManager.GetCurrentView()
		if currentView != nil {
			// For chat view, we need to check if it supports mouse handling
			if chatView, ok := currentView.(*ChatView); ok {
				if chatView.HandleMouseEvent(ev) {
					return
				}
			}
		}
	}
}

func (app *App) handleResize(ev *tcell.EventResize) {
	app.screen.Sync()
	width, height := ev.Size()

	app.layout = NewLayout(width, height)
	app.input = app.input.WithWidth(width)
	app.messages = app.messages.WithSize(width, height-5) // -5 for status, input, and alert areas
	app.status = app.status.WithWidth(width)

	app.viewManager.HandleResize(width, height)
}

func (app *App) sendMessageWithContent(content string) {
	log := logger.WithComponent("tui_app")

	// Optimistically add user message to conversation and update UI
	app.controller.AddUserMessage(content)
	if app.chatView != nil {
		app.chatView.updateMessages()
		app.chatView.scrollToBottom()
	}
	// Render immediately to show the message before processing starts
	app.render()

	log.Debug("STATE TRANSITION: Starting message send",
		"content", content,
		"length", len(content),
		"previous_sending", app.sending)

	app.sending = true
	app.sendStartTime = time.Now()
	app.status = app.status.WithStatus("Sending...")
	log.Debug("STATE TRANSITION: Set sending=true, status=Sending")

	// Sync view state to show spinner immediately
	app.viewManager.SyncViewState(app.sending)
	log.Debug("STATE TRANSITION: Synced view state with sending=true")

	// Force immediate render to show spinner
	app.render()

	// Send the message using streaming in a goroutine to avoid blocking the UI
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("PANIC: Goroutine panic in sendMessageWithContent", "panic", r)
				app.screen.PostEvent(NewMessageErrorEvent(fmt.Errorf("message sending panic: %v", r)))
			}
		}()

		log.Debug("STREAMING: Starting streaming for message", "content", content)

		// Create context with timeout
		// Get timeout from config or use default
		timeout := 90 * time.Second
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Config not initialized, use default
				}
			}()
			if cfg := config.Get(); cfg != nil {
				timeout = cfg.Ollama.Timeout
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Start streaming
		updates, err := app.controller.StartStreaming(ctx, content)
		if err != nil {
			log.Error("STREAMING: Failed to start streaming", "error", err)

			// Provide more specific error messages
			var displayError error
			if strings.Contains(err.Error(), "connection refused") {
				displayError = fmt.Errorf("Cannot connect to Ollama. Is it running? Try: ollama serve")
			} else if strings.Contains(err.Error(), "timeout") {
				displayError = fmt.Errorf("Request timed out. The model might be loading or processing a complex request")
			} else if strings.Contains(err.Error(), "404") {
				displayError = fmt.Errorf("Model not found. Check if the model is pulled: ollama pull <model>")
			} else {
				displayError = err
			}

			app.screen.PostEvent(NewMessageErrorEvent(displayError))
			return
		}

		log.Debug("STREAMING: Successfully started streaming, processing updates")

		// Process streaming updates
		for update := range updates {
			select {
			case <-app.cancelSend:
				log.Debug("STREAMING: Cancelled by user")
				cancel() // Cancel the context to stop streaming
				return
			default:
			}

			switch update.Type {
			case controllers.StreamStarted:
				log.Debug("STREAMING: Stream started", "stream_id", update.StreamID, "model", update.Metadata.Model)
				app.screen.PostEvent(NewStreamStartEvent(update.StreamID, update.Metadata.Model))

			case controllers.ChunkReceived:
				log.Debug("STREAMING: Chunk received", "stream_id", update.StreamID, "content_length", len(update.Content))
				app.screen.PostEvent(NewMessageChunkEvent(update.StreamID, update.Content, false, update.Metadata.ChunkCount))

			case controllers.MessageComplete:
				log.Debug("STREAMING: Message complete", "stream_id", update.StreamID, "final_length", len(update.Message.Content))
				app.screen.PostEvent(NewStreamCompleteEvent(update.StreamID, update.Message, update.Metadata.ChunkCount, update.Metadata.Duration))

			case controllers.StreamError:
				log.Error("STREAMING: Stream error", "stream_id", update.StreamID, "error", update.Error)
				app.screen.PostEvent(NewStreamErrorEvent(update.StreamID, update.Error))

			case controllers.ToolExecutionStarted:
				log.Debug("STREAMING: Tool execution started", "stream_id", update.StreamID)
				// Could add tool execution UI indicators here

			case controllers.ToolExecutionComplete:
				log.Debug("STREAMING: Tool execution complete", "stream_id", update.StreamID)
				// Could add tool execution completion indicators here
			}
		}

		log.Debug("STREAMING: Streaming completed")
	}()
}

func (app *App) handleChatMessageSend(ev *ChatMessageSendEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("EVENT: Handling ChatMessageSendEvent",
		"content", ev.Content,
		"current_sending", app.sending)

	if !app.sending {
		log.Debug("STATE CHECK: Not currently sending, proceeding with message")
		app.sendMessageWithContent(ev.Content)
	} else {
		log.Warn("STATE CHECK: Already sending, ignoring new message request",
			"ignored_content", ev.Content)
	}
}

func (app *App) handleMessageResponse(ev *MessageResponseEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("EVENT: Handling MessageResponseEvent",
		"previous_sending", app.sending,
		"response_content_length", len(ev.Message.Content))

	app.sending = false
	log.Debug("STATE TRANSITION: Set sending=false after successful response")

	if app.chatView != nil {
		app.chatView.HandleMessageResponse(*ev)
		log.Debug("EVENT: Forwarded response to ChatView")
	} else {
		log.Warn("EVENT: No ChatView available to handle response")
	}

	// Sync view state after state change
	app.viewManager.SyncViewState(app.sending)
	log.Debug("STATE TRANSITION: Synced view state with sending=false")

	// Force render to update UI immediately
	app.render()
	log.Debug("STATE TRANSITION: Forced render after response")
}

func (app *App) handleMessageError(ev *MessageErrorEvent) {
	log := logger.WithComponent("tui_app")
	log.Error("EVENT: Handling MessageErrorEvent",
		"previous_sending", app.sending,
		"error", ev.Error)

	app.sending = false
	log.Debug("STATE TRANSITION: Set sending=false after error")

	// Add error message to conversation so it appears as a red chat message
	errorMsg := "Error: " + ev.Error.Error()
	app.controller.AddErrorMessage(errorMsg)
	log.Debug("Added error message to conversation", "full_error", errorMsg, "length", len(errorMsg))

	if app.chatView != nil {
		app.chatView.HandleMessageError(*ev)
		log.Debug("EVENT: Forwarded error to ChatView")
	} else {
		log.Warn("EVENT: No ChatView available to handle error")
	}

	// Sync view state after state change
	app.viewManager.SyncViewState(app.sending)
	log.Debug("STATE TRANSITION: Synced view state with sending=false")

	// Force render to update UI immediately
	app.render()
	log.Debug("STATE TRANSITION: Forced render after error")
}

func (app *App) handleSpinnerAnimation(_ *SpinnerAnimationEvent) {
	// Update spinner animation in ChatView
	if app.chatView != nil && app.sending {
		app.chatView.UpdateSpinnerFrame()

		// Update spinner text with elapsed time
		if !app.sendStartTime.IsZero() {
			// Simplified spinner - no extra text as per TODO
			spinnerText := ""

			// Update alert display with new text
			if app.chatView.alert.IsSpinnerVisible {
				app.chatView.alert = app.chatView.alert.WithSpinner(true, spinnerText)
			}
		}

	}
}

func (app *App) updateMessages() {
	history := app.controller.GetHistory()
	// Filter out system messages - they should not be displayed to the user
	var filteredHistory []chat.Message
	for _, msg := range history {
		if msg.Role != chat.RoleSystem {
			filteredHistory = append(filteredHistory, msg)
		}
	}
	app.messages = app.messages.WithMessages(filteredHistory)
}

func (app *App) handleViewChange(ev *ViewChangeEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ViewChangeEvent", "view", ev.ViewName)

	app.viewManager.SetCurrentView(ev.ViewName)
	app.viewManager.SyncViewState(app.sending)
}

func (app *App) handleMenuToggle(ev *MenuToggleEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling MenuToggleEvent", "show", ev.Show)

	if ev.Show {
		app.viewManager.ToggleMenu()
	} else {
		app.viewManager.HideMenu()
		// Sync state when menu closes in case view switched
		app.viewManager.SyncViewState(app.sending)
	}
}

func (app *App) handleModelListUpdate(ev *ModelListUpdateEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelListUpdateEvent", "model_count", len(ev.Models))

	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelListUpdate(*ev)
		log.Debug("Forwarded ModelListUpdateEvent to ModelView")
	} else {
		log.Debug("Current view is not ModelView, ignoring ModelListUpdateEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelStatsUpdate(ev *ModelStatsUpdateEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelStatsUpdateEvent", "total_models", ev.Stats.TotalModels)

	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelStatsUpdate(*ev)
		log.Debug("Forwarded ModelStatsUpdateEvent to ModelView")
	} else {
		log.Debug("Current view is not ModelView, ignoring ModelStatsUpdateEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelError(ev *ModelErrorEvent) {
	log := logger.WithComponent("tui_app")
	log.Error("Handling ModelErrorEvent", "error", ev.Error)

	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelError(*ev)
		log.Debug("Forwarded ModelErrorEvent to ModelView")
	} else {
		log.Debug("Current view is not ModelView, ignoring ModelErrorEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelDeleted(ev *ModelDeletedEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelDeletedEvent", "model_name", ev.ModelName)
	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelDeleted(*ev)
		log.Debug("Forwarded ModelDeletedEvent to ModelView")
	} else {
		log.Debug("Current view is not ModelView, ignoring ModelDeletedEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelDownloadProgress(ev *ModelDownloadProgressEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelDownloadProgressEvent", "model_name", ev.ModelName, "progress", ev.Progress)
	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelDownloadProgress(*ev)
		log.Debug("Forwarded ModelDownloadProgressEvent to ModelView")
	} else if chatView, ok := currentView.(*ChatView); ok {
		chatView.HandleModelDownloadProgress(*ev)
		log.Debug("Forwarded ModelDownloadProgressEvent to ChatView")
	} else {
		log.Debug("Current view does not support download progress, ignoring ModelDownloadProgressEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelDownloadComplete(ev *ModelDownloadCompleteEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelDownloadCompleteEvent", "model_name", ev.ModelName)
	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelDownloadComplete(*ev)
		log.Debug("Forwarded ModelDownloadCompleteEvent to ModelView")
	} else if chatView, ok := currentView.(*ChatView); ok {
		chatView.HandleModelDownloadComplete(*ev)
		log.Debug("Forwarded ModelDownloadCompleteEvent to ChatView")
	} else {
		log.Debug("Current view does not support download complete, ignoring ModelDownloadCompleteEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelDownloadError(ev *ModelDownloadErrorEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelDownloadErrorEvent", "model_name", ev.ModelName, "error", ev.Error)
	currentView := app.viewManager.GetCurrentView()
	if modelView, ok := currentView.(*ModelView); ok {
		modelView.HandleModelDownloadError(*ev)
		log.Debug("Forwarded ModelDownloadErrorEvent to ModelView")
	} else if chatView, ok := currentView.(*ChatView); ok {
		chatView.HandleModelDownloadError(*ev)
		log.Debug("Forwarded ModelDownloadErrorEvent to ChatView")
	} else {
		log.Debug("Current view does not support download error, ignoring ModelDownloadErrorEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelNotFound(ev *ModelNotFoundEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelNotFoundEvent", "model_name", ev.ModelName)
	currentView := app.viewManager.GetCurrentView()
	if _, ok := currentView.(*ModelView); ok {
		// For now, just log - this event could be used for other integrations
		log.Debug("Model not found event received", "model_name", ev.ModelName)
	} else {
		log.Debug("Current view is not ModelView, ignoring ModelNotFoundEvent", "current_view_type", currentView)
	}
}

func (app *App) handleModelChange(ev *ModelChangeEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling ModelChangeEvent", "model_name", ev.ModelName)

	// Forward the event to ChatView to update its status bar
	if app.chatView != nil {
		app.chatView.HandleModelChange(*ev)
		log.Debug("Forwarded ModelChangeEvent to ChatView")
	} else {
		log.Debug("No ChatView available to handle model change")
	}

	// Update tools view with new model
	if toolsView, ok := app.viewManager.views["tools"].(*ToolsView); ok {
		toolsView.SetCurrentModel(ev.ModelName)
		log.Debug("Updated ToolsView current model")
	}

}

// Streaming Event Handlers

func (app *App) handleMessageChunk(ev *MessageChunkEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling MessageChunkEvent",
		"stream_id", ev.StreamID,
		"content_length", len(ev.Content),
		"is_complete", ev.IsComplete,
		"chunk_index", ev.ChunkIndex)

	// Update streaming state
	if app.currentStreamID == "" {
		app.currentStreamID = ev.StreamID
		app.streaming = true
	}

	// Accumulate content
	if ev.Content != "" {
		app.streamingContent += ev.Content
	}

	// Add chunk to accumulator
	chunk := chat.MessageChunk{
		ID:        fmt.Sprintf("%s-%d", ev.StreamID, ev.ChunkIndex),
		Content:   ev.Content,
		Done:      ev.IsComplete,
		Timestamp: ev.Timestamp,
		StreamID:  ev.StreamID,
	}
	app.streamAccumulator.AddChunk(chunk)

	// Update ChatView with streaming content
	if app.chatView != nil {
		app.chatView.UpdateStreamingContent(ev.StreamID, app.streamingContent, ev.IsComplete)
	}

	log.Debug("Updated streaming content",
		"total_length", len(app.streamingContent),
		"chunk_count", ev.ChunkIndex+1)
}

func (app *App) handleStreamStart(ev *StreamStartEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling StreamStartEvent",
		"stream_id", ev.StreamID,
		"model", ev.Model)

	// Initialize streaming state
	app.streaming = true
	app.currentStreamID = ev.StreamID
	app.streamingContent = ""

	// Update status to show streaming
	app.status = app.status.WithStatus("Streaming...")

	// Sync view state to show streaming indicators
	app.viewManager.SyncViewState(app.sending)

	// Notify ChatView about stream start
	if app.chatView != nil {
		app.chatView.HandleStreamStart(ev.StreamID, ev.Model)
	}

	log.Debug("Started streaming session", "stream_id", ev.StreamID)
}

func (app *App) handleStreamComplete(ev *StreamCompleteEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling StreamCompleteEvent",
		"stream_id", ev.StreamID,
		"total_chunks", ev.TotalChunks,
		"duration", ev.Duration,
		"final_message_length", len(ev.FinalMessage.Content))

	// Clear streaming state
	app.streaming = false
	app.sending = false
	app.currentStreamID = ""
	app.streamingContent = ""

	// Update status
	app.status = app.status.WithStatus("Ready")

	// Finalize message in accumulator
	if app.streamAccumulator != nil {
		app.streamAccumulator.FinalizeMessage(ev.StreamID)
	}

	// Notify ChatView about completion
	if app.chatView != nil {
		app.chatView.HandleStreamComplete(ev.StreamID, ev.FinalMessage, ev.TotalChunks, ev.Duration)
		app.chatView.updateMessages()
		app.chatView.scrollToBottom()
	}

	// Sync view state after state change
	app.viewManager.SyncViewState(app.sending)

	// Force render to update UI
	app.render()

	log.Debug("Completed streaming session",
		"stream_id", ev.StreamID,
		"final_content_length", len(ev.FinalMessage.Content),
		"duration", ev.Duration)
}

func (app *App) handleStreamError(ev *StreamErrorEvent) {
	log := logger.WithComponent("tui_app")
	log.Error("Handling StreamErrorEvent",
		"stream_id", ev.StreamID,
		"error", ev.Error)

	// Clear streaming state
	app.streaming = false
	app.sending = false
	app.currentStreamID = ""
	app.streamingContent = ""

	// Update status
	app.status = app.status.WithStatus("Ready")

	// Add error message to conversation
	errorMsg := "Streaming Error: " + ev.Error.Error()
	app.controller.AddErrorMessage(errorMsg)

	// Clean up accumulator
	if app.streamAccumulator != nil {
		app.streamAccumulator.CleanupStream(ev.StreamID)
	}

	// Notify ChatView about error
	if app.chatView != nil {
		app.chatView.HandleStreamError(ev.StreamID, ev.Error)
		app.chatView.updateMessages()
	}

	// Sync view state after state change
	app.viewManager.SyncViewState(app.sending)

	// Force render to update UI
	app.render()

	log.Debug("Handled streaming error", "stream_id", ev.StreamID)
}

func (app *App) handleStreamProgress(ev *StreamProgressEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling StreamProgressEvent",
		"stream_id", ev.StreamID,
		"content_length", ev.ContentLength,
		"chunk_count", ev.ChunkCount,
		"duration", ev.Duration)

	// Update progress indicators in ChatView
	if app.chatView != nil {
		app.chatView.UpdateStreamProgress(ev.StreamID, ev.ContentLength, ev.ChunkCount, ev.Duration)
	}

	// Optional: Update status with progress info for very long streams
	if ev.Duration > 5*time.Second {
		statusText := fmt.Sprintf("Streaming... %d chars (%s)",
			ev.ContentLength,
			ev.Duration.Round(time.Second))
		app.status = app.status.WithStatus(statusText)
	}
}

func (app *App) render() {
	app.screen.Clear()

	width, height := app.screen.Size()
	area := Rect{X: 0, Y: 0, Width: width, Height: height}

	// Update context tree view if it's the current view
	if app.viewManager.GetCurrentViewName() == "context-tree" {
		app.updateContextTreeView()
	}

	app.viewManager.Render(app.screen, area)

	// Render modal on top of everything else
	app.modal.Render(app.screen, area)

	app.screen.Show()
}

// updateContextTreeView updates the context tree view with current conversation data
func (app *App) updateContextTreeView() {
	if app.contextTreeView == nil {
		return
	}

	// Get messages from controller to build a context tree
	messages := app.controller.GetHistory()

	// Create a context tree with the current conversation
	tree := &chat.ContextTree{
		RootContextID: "main",
		Contexts: map[string]*chat.Context{
			"main": {
				ID:          "main",
				ParentID:    nil,
				BranchPoint: nil,
				Title:       "Main Conversation",
				Created:     time.Now(),
				MessageIDs:  []string{},
				IsActive:    true,
			},
		},
		Messages:      make(map[string]*chat.Message),
		ParentIndex:   make(map[string][]string),
		ChildIndex:    make(map[string]string),
		ActiveContext: "main",
	}

	// Add existing messages to the tree
	for i, msg := range messages {
		msgCopy := msg // Create a copy
		msgID := fmt.Sprintf("msg-%d", i)
		msgCopy.ID = msgID
		msgCopy.ContextID = "main"
		tree.Messages[msgID] = &msgCopy
		tree.Contexts["main"].MessageIDs = append(tree.Contexts["main"].MessageIDs, msgID)
	}

	// Create some example branches if we have messages
	if len(messages) > 2 {
		// Create a branch from the second message
		branchID := "branch1"
		branchMsgID := "msg-1"
		tree.Contexts[branchID] = &chat.Context{
			ID:          branchID,
			ParentID:    &[]string{"main"}[0],
			BranchPoint: &branchMsgID,
			Title:       "Alternative Response",
			Created:     time.Now().Add(-5 * time.Minute),
			MessageIDs:  []string{},
			IsActive:    false,
		}
		tree.ParentIndex["main"] = append(tree.ParentIndex["main"], branchID)
		tree.ChildIndex[branchID] = "main"
	}

	// Update the context tree view with the new tree
	app.contextTreeView.UpdateTree(tree)
}
