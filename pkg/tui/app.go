package tui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/controllers"
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
}

func NewApp(controller *controllers.ChatController) (*App, error) {
	screen, err := tcell.NewScreen()	
	if err != nil {
		return nil, err
	}
	
	if err := screen.Init(); err != nil {
		return nil, err
	}
	
	width, height := screen.Size()
	
	app := &App{
		screen:     screen,
		controller: controller,
		input:      NewInputField(width),
		messages:   NewMessageDisplay(width, height-4),
		status:     NewStatusBar(width).WithModel(controller.GetModel()).WithStatus("Ready"),
		layout:     NewLayout(width, height),
		quit:       false,
		sending:    false,
	}
	
	app.updateMessages()
	
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
	}
}

func (app *App) handleKeyEvent(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		app.quit = true
		
	case tcell.KeyEnter:
		if !app.sending {
			app.sendMessage()
		}
		
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		app.input = app.input.DeleteBackward()
		
	case tcell.KeyLeft:
		app.input = app.input.WithCursor(app.input.Cursor - 1)
		
	case tcell.KeyRight:
		app.input = app.input.WithCursor(app.input.Cursor + 1)
		
	case tcell.KeyHome:
		app.input = app.input.WithCursor(0)
		
	case tcell.KeyEnd:
		app.input = app.input.WithCursor(len(app.input.Content))
		
	case tcell.KeyUp:
		app.scrollUp()
		
	case tcell.KeyDown:
		app.scrollDown()
		
	case tcell.KeyPgUp:
		app.pageUp()
		
	case tcell.KeyPgDn:
		app.pageDown()
		
	default:
		if ev.Rune() != 0 {
			app.input = app.input.InsertRune(ev.Rune())
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
}

func (app *App) sendMessage() {
	content := strings.TrimSpace(app.input.Content)
	if content == "" {
		return
	}
	
	// Clear input immediately and set sending state
	app.input = app.input.Clear()
	app.sending = true
	app.status = app.status.WithStatus("Sending...")
	
	// Send the message in a goroutine to avoid blocking the UI
	go func() {
		response, err := app.controller.SendUserMessage(content)
		
		// Post the result back to the main event loop
		if err != nil {
			app.screen.PostEvent(NewMessageErrorEvent(err))
		} else {
			app.screen.PostEvent(NewMessageResponseEvent(response))
		}
	}()
}

func (app *App) handleMessageResponse(ev *MessageResponseEvent) {
	// Reset sending state
	app.sending = false
	app.status = app.status.WithStatus("Ready")
	
	// Update messages and scroll to bottom
	app.updateMessages()
	app.scrollToBottom()
}

func (app *App) handleMessageError(ev *MessageErrorEvent) {
	// Reset sending state and show error
	app.sending = false
	app.status = app.status.WithStatus("Error: " + ev.Error.Error())
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

func (app *App) render() {
	app.screen.Clear()
	
	messageArea, inputArea, statusArea := app.layout.CalculateAreas()
	
	RenderMessages(app.screen, app.messages, messageArea)
	RenderInput(app.screen, app.input, inputArea)
	RenderStatus(app.screen, app.status, statusArea)
	
	app.screen.Show()
}