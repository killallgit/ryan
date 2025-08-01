package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
	"github.com/spf13/viper"
)

type App struct {
	screen      tcell.Screen
	controller  *controllers.ChatController
	input       InputField
	messages    MessageDisplay
	status      StatusBar
	layout      Layout
	quit        bool
	sending     bool  // Track if we're currently sending a message
	viewManager *ViewManager
	chatView    *ChatView
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
	
	viewManager := NewViewManager()
	chatView := NewChatView(controller, screen)
	
	ollamaURL := viper.GetString("ollama.url")
	log.Debug("Creating ollama client for models", "url", ollamaURL)
	ollamaClient := ollama.NewClient(ollamaURL)
	modelsController := controllers.NewModelsController(ollamaClient)
	modelView := NewModelView(modelsController, screen)
	
	viewManager.RegisterView("chat", chatView)
	viewManager.RegisterView("models", modelView)
	log.Debug("Registered views with view manager", "views", []string{"chat", "models"})
	
	app := &App{
		screen:      screen,
		controller:  controller,
		input:       NewInputField(width),
		messages:    NewMessageDisplay(width, height-4),
		status:      NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready"),
		layout:      NewLayout(width, height),
		quit:        false,
		sending:     false,
		viewManager: viewManager,
		chatView:    chatView,
	}
	
	app.updateMessages()
	log.Debug("TUI application created successfully")
	
	return app, nil
}

func (app *App) Run() error {
	defer app.screen.Fini()
	
	app.render()
	
	for !app.quit {
		event := app.screen.PollEvent()
		app.handleEvent(event)
		app.render()
	}
	
	return nil
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
	case *ChatMessageSendEvent:
		app.handleChatMessageSend(ev)
	}
}

func (app *App) handleKeyEvent(ev *tcell.EventKey) {
	log := logger.WithComponent("tui_app")
	log.Debug("Handling key event", "key", ev.Key(), "rune", ev.Rune(), "modifiers", ev.Modifiers())
	
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
		} else {
			app.quit = true
			log.Debug("Application quit triggered")
		}
		
	case tcell.KeyF1:  // Changed from KeyCtrlM to avoid conflict with Enter
		app.viewManager.ToggleMenu()
		log.Debug("Menu toggled via F1")
		
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
	app.messages = app.messages.WithSize(width, height-4)
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
	app.status = app.status.WithStatus("Sending...")
	log.Debug("STATE TRANSITION: Set sending=true, status=Sending")
	
	// Send the message in a goroutine to avoid blocking the UI
	go func() {
		log.Debug("API CALL: Calling controller.SendUserMessage", "content", content)
		response, err := app.controller.SendUserMessage(content)
		
		// Post the result back to the main event loop
		if err != nil {
			log.Error("API CALL: Message send failed", "error", err)
			app.screen.PostEvent(NewMessageErrorEvent(err))
		} else {
			log.Debug("API CALL: Message send succeeded", "response_content_length", len(response.Content))
			app.screen.PostEvent(NewMessageResponseEvent(response))
		}
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
}

func (app *App) handleMessageError(ev *MessageErrorEvent) {
	log := logger.WithComponent("tui_app")
	log.Error("EVENT: Handling MessageErrorEvent", 
		"previous_sending", app.sending,
		"error", ev.Error)
	
	app.sending = false
	log.Debug("STATE TRANSITION: Set sending=false after error")
	
	if app.chatView != nil {
		app.chatView.HandleMessageError(*ev)
		log.Debug("EVENT: Forwarded error to ChatView")
	} else {
		log.Warn("EVENT: No ChatView available to handle error")
	}
	
	// Sync view state after state change
	app.viewManager.SyncViewState(app.sending)
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

func (app *App) render() {
	app.screen.Clear()
	
	width, height := app.screen.Size()
	area := Rect{X: 0, Y: 0, Width: width, Height: height}
	
	app.viewManager.Render(app.screen, area)
	
	app.screen.Show()
}