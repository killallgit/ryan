package tui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/viper"
)

type App struct {
	screen        tcell.Screen
	controller    *controllers.ChatController
	input         InputField
	messages      MessageDisplay
	status        StatusBar
	layout        Layout
	quit          bool
	sending       bool      // Track if we're currently sending a message
	sendStartTime time.Time // Track when message sending started
	cancelSend    chan bool // Channel to cancel current send operation
	viewManager   *ViewManager
	chatView      *ChatView
	spinnerTicker *time.Ticker
	spinnerStop   chan bool
	modal         ModalDialog
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

func NewApp(controller *controllers.ChatController) (*App, error) {
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

	ollamaURL := viper.GetString("ollama.url")
	log.Debug("Creating ollama client for models", "url", ollamaURL)

	// Check Ollama connectivity before proceeding
	var connectivityError error
	if err := checkOllamaConnectivity(ollamaURL); err != nil {
		log.Error("Ollama connectivity check failed", "error", err)
		connectivityError = err
	}

	ollamaClient := ollama.NewClient(ollamaURL)
	modelsController := controllers.NewModelsController(ollamaClient)

	// Connect ollama client to chat controller for model validation
	controller.SetOllamaClient(ollamaClient)

	viewManager := NewViewManager()
	chatView := NewChatView(controller, modelsController, screen)
	modelView := NewModelView(modelsController, controller, screen)

	viewManager.RegisterView("chat", chatView)
	viewManager.RegisterView("models", modelView)
	log.Debug("Registered views with view manager", "views", []string{"chat", "models"})

	app := &App{
		screen:        screen,
		controller:    controller,
		input:         NewInputField(width),
		messages:      NewMessageDisplay(width, height-5), // -5 for status, input, and alert areas
		status:        NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready"),
		layout:        NewLayout(width, height),
		quit:          false,
		sending:       false,
		cancelSend:    make(chan bool, 1), // Buffered channel for cancellation
		viewManager:   viewManager,
		chatView:      chatView,
		spinnerTicker: time.NewTicker(100 * time.Millisecond), // Faster animation for smoother spinner
		spinnerStop:   make(chan bool),
		modal:         NewModalDialog(),
	}

	app.updateMessages()

	// Show connectivity error modal if there was an issue
	if connectivityError != nil {
		app.modal = app.modal.WithError("Ollama Connection Error",
			fmt.Sprintf("Cannot connect to Ollama at %s\n\n%v\n\nPress any key to continue with limited functionality.",
				ollamaURL, connectivityError))
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
	// Stop spinner timer
	if app.spinnerTicker != nil {
		app.spinnerTicker.Stop()
	}
	if app.spinnerStop != nil {
		close(app.spinnerStop)
	}
}

func (app *App) runSpinnerTimer() {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting spinner animation timer")

	for {
		select {
		case <-app.spinnerTicker.C:
			// Only animate when sending
			if app.sending {
				app.screen.PostEvent(NewSpinnerAnimationEvent())
			}
		case <-app.spinnerStop:
			log.Debug("Stopping spinner animation timer")
			return
		}
	}
}

func (app *App) handleEvent(event tcell.Event) {
	switch ev := event.(type) {
	case *tcell.EventKey:
		app.handleKeyEvent(ev)
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
	}
}

func (app *App) handleKeyEvent(ev *tcell.EventKey) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling key event", "key", ev.Key(), "rune", ev.Rune(), "modifiers", ev.Modifiers())

	// Handle modal first
	if app.modal.Visible {
		app.modal = app.modal.Hide()
		log.Debug("Modal dismissed")
		return
	}

	if app.viewManager.HandleMenuKeyEvent(ev) {
		log.Debug("Key event handled by menu")
		return
	}

	switch ev.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		if app.viewManager.IsMenuVisible() {
			app.viewManager.HideMenu()
			app.viewManager.SyncViewState(app.sending)
			log.Debug("Menu hidden via Escape/Ctrl+C, state synced")
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

	case tcell.KeyCtrlP:
		app.viewManager.ToggleMenu()
		log.Debug("Menu toggled via Ctrl+P")

	default:
		currentView := app.viewManager.GetCurrentView()
		if currentView != nil {
			handled := currentView.HandleKeyEvent(ev, app.sending)
			log.Debug("Key event forwarded to current view", "view", currentView.Name(), "handled", handled)
		} else {
			log.Debug("No current view to handle key event")
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

func (app *App) sendMessage() {
	content := strings.TrimSpace(app.input.Content)
	if content == "" {
		return
	}

	// Clear input immediately and set sending state
	app.input = app.input.Clear()
	app.sendMessageWithContent(content)
}

func (app *App) sendMessageWithContent(content string) {
	log := logger.WithComponent("tui_app")
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
	log.Debug("STATE TRANSITION: Forced render to show spinner")

	// Send the message in a goroutine to avoid blocking the UI
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("PANIC: Goroutine panic in sendMessageWithContent", "panic", r)
				app.screen.PostEvent(NewMessageErrorEvent(fmt.Errorf("message sending panic: %v", r)))
			}
		}()

		log.Debug("API CALL: Starting SendUserMessage", "content", content, "goroutine_id", fmt.Sprintf("%p", &content))

		// Add timeout channel for debugging
		done := make(chan bool, 1)
		var response chat.Message
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("PANIC: Controller panic", "panic", r)
					err = fmt.Errorf("controller panic: %v", r)
				}
				done <- true
			}()

			log.Debug("API CALL: Calling controller.SendUserMessage", "content", content)
			response, err = app.controller.SendUserMessage(content)
			log.Debug("API CALL: Controller call completed", "error", err, "has_response", response.Content != "")
		}()

		select {
		case <-done:
			log.Debug("API CALL: Controller completed normally")
		case <-time.After(30 * time.Second):
			log.Error("API CALL: Timeout after 30 seconds", "content", content)
			err = fmt.Errorf("API call timeout after 30 seconds")
		case <-app.cancelSend:
			log.Debug("API CALL: Cancelled by user")
			err = fmt.Errorf("Message sending cancelled by user")
			return // Exit the goroutine early
		}

		// Post the result back to the main event loop
		if err != nil {
			log.Error("API CALL: Message send failed", "error", err)

			// Provide more specific error messages
			var displayError error
			if strings.Contains(err.Error(), "connection refused") {
				displayError = fmt.Errorf("Cannot connect to Ollama. Is it running? Try: ollama serve")
			} else if strings.Contains(err.Error(), "timeout") {
				displayError = fmt.Errorf("Request timed out after %v. The model might be loading or processing a complex request",
					time.Since(app.sendStartTime).Round(time.Second))
			} else if strings.Contains(err.Error(), "404") {
				displayError = fmt.Errorf("Model not found. Check if the model is pulled: ollama pull <model>")
			} else {
				displayError = err
			}

			app.screen.PostEvent(NewMessageErrorEvent(displayError))
		} else {
			log.Debug("API CALL: Message send succeeded", "response_content_length", len(response.Content))
			app.screen.PostEvent(NewMessageResponseEvent(response))
		}

		log.Debug("API CALL: Goroutine completing", "content", content)
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
	app.controller.AddErrorMessage("Error: " + ev.Error.Error())
	log.Debug("Added error message to conversation")

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

func (app *App) handleSpinnerAnimation(ev *SpinnerAnimationEvent) {
	log := logger.WithComponent("tui_app")
	log.Debug("EVENT: Handling SpinnerAnimationEvent")

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

		log.Debug("EVENT: Updated spinner frame in ChatView")
	}
}

func (app *App) updateMessages() {
	history := app.controller.GetHistory()
	app.messages = app.messages.WithMessages(history)
}

func (app *App) scrollUp() {
	if app.messages.Scroll > 0 {
		app.messages = app.messages.WithScroll(app.messages.Scroll - 1)
	}
}

func (app *App) scrollDown() {
	app.messages = app.messages.WithScroll(app.messages.Scroll + 1)
}

func (app *App) pageUp() {
	newScroll := app.messages.Scroll - app.messages.Height
	if newScroll < 0 {
		newScroll = 0
	}
	app.messages = app.messages.WithScroll(newScroll)
}

func (app *App) pageDown() {
	newScroll := app.messages.Scroll + app.messages.Height
	app.messages = app.messages.WithScroll(newScroll)
}

func (app *App) scrollToBottom() {
	var totalLines int
	for _, msg := range app.messages.Messages {
		lines := WrapText(msg.Content, app.messages.Width)
		totalLines += len(lines) + 2
	}

	if totalLines > app.messages.Height {
		app.messages = app.messages.WithScroll(totalLines - app.messages.Height)
	}
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

func (app *App) render() {
	app.screen.Clear()

	width, height := app.screen.Size()
	area := Rect{X: 0, Y: 0, Width: width, Height: height}

	app.viewManager.Render(app.screen, area)

	// Render modal on top of everything else
	app.modal.Render(app.screen, area)

	app.screen.Show()
}
