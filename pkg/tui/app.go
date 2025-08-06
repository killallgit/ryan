package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/config"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/ollama"
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
	AddAssistantMessage(content string)
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
	app           *tview.Application
	pages         *tview.Pages
	controller    ControllerInterface
	renderManager *RenderManager

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
	previousView string

	// Channels
	cancelSend chan bool

	// Config
	config *config.Config

	// Input handling
	originalInputCapture func(event *tcell.EventKey) *tcell.EventKey
}

// NewApp creates a new TUI application
func NewApp(controller ControllerInterface) (*App, error) {
	log := logger.WithComponent("tui_app")
	log.Debug("Creating new TUI application")

	tviewApp := tview.NewApplication()
	cfg := config.Get()

	// Apply theme
	theme := DefaultTheme()
	ApplyTheme(theme)

	// Initialize render manager
	renderManager, err := NewRenderManager(theme, 80) // Default width, will be updated
	if err != nil {
		return nil, fmt.Errorf("failed to create render manager: %w", err)
	}

	app := &App{
		app:           tviewApp,
		pages:         tview.NewPages(),
		controller:    controller,
		renderManager: renderManager,
		sending:       false,
		streaming:     false,
		currentView:   "chat",
		previousView:  "chat",
		cancelSend:    make(chan bool, 1),
		config:        cfg,
	}

	// Set background color for pages container
	app.pages.SetBackgroundColor(tcell.GetColor(ColorBase00))

	// Initialize views
	if err := app.initializeViews(); err != nil {
		return nil, fmt.Errorf("failed to initialize views: %w", err)
	}

	// Setup global key bindings
	app.setupGlobalKeyBindings()

	// Set initial focus
	tviewApp.SetRoot(app.pages, true).SetFocus(app.pages)

	// Initialize with chat view and ensure proper focus
	app.currentView = "chat"
	app.previousView = "chat"
	app.pages.SwitchToPage("chat")

	// Explicitly set focus to chat view input
	if app.chatView != nil {
		tviewApp.SetFocus(app.chatView)
	}

	log.Debug("TUI application created successfully")
	return app, nil
}

// initializeViews creates and registers all application views
func (a *App) initializeViews() error {
	log := logger.WithComponent("tview_app")

	// Create chat view (already has its own render manager)
	a.chatView = NewChatView(a.controller, a.app)
	a.chatView.SetSendMessageHandler(func(content string) {
		a.SendMessage(content)
	})
	a.pages.AddPage("chat", a.chatView, true, true)
	log.Debug("Created chat view")

	// Create model view with Ollama client
	ollamaURL := a.config.Ollama.URL
	if ollamaURL == "" {
		ollamaURL = "https://ollama.kitty-tetra.ts.net" // fallback
	}
	// Use shorter timeout for UI operations to avoid blocking
	ollamaClient := ollama.NewClientWithTimeout(ollamaURL, 5*time.Second)
	modelsController := controllers.NewModelsController(ollamaClient)
	a.modelView = NewModelView(modelsController, a.controller, a.app, a.renderManager)
	a.pages.AddPage("models", a.modelView, true, false)
	log.Debug("Created model view")

	// Create tools view
	a.toolsView = NewToolsView(a.controller.GetToolRegistry(), a.renderManager)
	a.pages.AddPage("tools", a.toolsView, true, false)
	log.Debug("Created tools view")

	// Create vector store view
	a.vectorStoreView = NewVectorStoreView(a.renderManager)
	a.pages.AddPage("vectorstore", a.vectorStoreView, true, false)
	log.Debug("Created vector store view")

	// Create context tree view
	a.contextTreeView = NewContextTreeView(a.renderManager)
	a.pages.AddPage("context-tree", a.contextTreeView, true, false)
	log.Debug("Created context tree view")

	return nil
}

// setupGlobalKeyBindings configures application-wide keyboard shortcuts
func (a *App) setupGlobalKeyBindings() {
	a.originalInputCapture = func(event *tcell.EventKey) *tcell.EventKey {
		log := logger.WithComponent("global_input")
		log.Debug("Global input capture", "key", event.Key(), "rune", string(event.Rune()))

		// Let Enter key pass through to input field when in chat view
		if event.Key() == tcell.KeyEnter && a.currentView == "chat" {
			log.Debug("Enter key - passing through to chat input field")
			return event // Let the input field handle Enter
		}

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

		// Escape: Return to previous view or chat if nowhere else to go
		if event.Key() == tcell.KeyEscape {
			if a.currentView != a.previousView {
				a.switchToView(a.previousView)
				return nil
			} else if a.currentView != "chat" {
				// If current and previous are the same but not chat, go to chat
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
	}
}

// Run starts the tview application
func (a *App) Run() error {
	// Remove all global input capture - let components handle events naturally
	return a.app.Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.app.Stop()
}

// switchToView switches to the specified view
func (a *App) switchToView(viewName string) {
	if a.currentView != viewName {
		a.previousView = a.currentView
		a.currentView = viewName
	}
	a.pages.SwitchToPage(viewName)

	// Update current model in tools view if switching to it
	if viewName == "tools" && a.toolsView != nil {
		a.toolsView.SetCurrentModel(a.controller.GetModel())
	}
}

// showViewSwitcher displays a modal for switching between views
func (a *App) showViewSwitcher() {
	// Store the previous view
	previousView := a.currentView

	// Create a simple list of available views
	list := tview.NewList().
		AddItem("Chat", "", 0, func() {
			a.switchToView("chat")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Models", "", 0, func() {
			a.switchToView("models")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Tools", "", 0, func() {
			a.switchToView("tools")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Vector Store", "", 0, func() {
			a.switchToView("vectorstore")
			a.pages.RemovePage("view-switcher")
		}).
		AddItem("Context Tree", "", 0, func() {
			a.switchToView("context-tree")
			a.pages.RemovePage("view-switcher")
		})

	list.SetBorder(false).
		SetBackgroundColor(tcell.GetColor(ColorBase01))
	list.ShowSecondaryText(false)

	// Setup key bindings for j/k navigation
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			a.switchToView(previousView)
			a.pages.RemovePage("view-switcher")
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j', 'J':
				current := list.GetCurrentItem()
				if current < list.GetItemCount()-1 {
					list.SetCurrentItem(current + 1)
				}
				return nil
			case 'k', 'K':
				current := list.GetCurrentItem()
				if current > 0 {
					list.SetCurrentItem(current - 1)
				}
				return nil
			}
		}
		return event
	})

	// Create outer container with background
	outerContainer := tview.NewBox().
		SetBackgroundColor(tcell.GetColor(ColorBase01))

	// Create inner flex for horizontal padding
	innerFlex := tview.NewFlex().SetDirection(tview.FlexColumn)
	innerFlex.SetBackgroundColor(tcell.GetColor(ColorBase01))
	innerFlex.AddItem(nil, 3, 0, false) // Left padding
	innerFlex.AddItem(list, 0, 1, true) // List content
	innerFlex.AddItem(nil, 3, 0, false) // Right padding

	// Create a wrapper that adds padding between the outer container and list
	paddedWrapper := tview.NewFlex().SetDirection(tview.FlexRow)
	paddedWrapper.SetBackgroundColor(tcell.GetColor(ColorBase01))
	paddedWrapper.AddItem(nil, 2, 0, false)      // Top padding
	paddedWrapper.AddItem(innerFlex, 0, 1, true) // Content with horizontal padding
	paddedWrapper.AddItem(nil, 2, 0, false)      // Bottom padding

	// Stack the background and padded content
	modalContent := tview.NewPages().
		AddPage("bg", outerContainer, true, true).
		AddPage("content", paddedWrapper, true, true)

	// Create modal with height to show all 5 items plus padding
	modal := createModal(modalContent, 30, 9)

	// Add as overlay
	a.pages.AddPage("view-switcher", modal, true, true)
}

// createModal creates a centered modal primitive
func createModal(p tview.Primitive, width, height int) tview.Primitive {
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
	modal.SetBackgroundColor(tcell.ColorDefault) // Transparent to show the base background
	return modal
}

// SendMessage sends a message through the controller
func (a *App) SendMessage(content string) {
	if a.sending {
		return
	}

	a.sending = true

	// Send message in goroutine to avoid UI thread deadlock
	go func() {
		log := logger.WithComponent("send_message")
		log.Info("Starting send message goroutine", "content_length", len(content))
		// Note: Don't set sending=false here, it will be done when streaming completes

		// Add user message to controller
		a.controller.AddUserMessage(content)
		log.Debug("Added user message to controller")

		// Update UI to show sending state
		a.app.QueueUpdateDraw(func() {
			log.Debug("Setting sending state to true in UI thread")
			a.chatView.SetSending(true)
			a.chatView.UpdateMessages()
		})

		// Create context with cancel
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Monitor for cancellation
		go func() {
			select {
			case <-a.cancelSend:
				log.Info("Send cancelled by user")
				cancel()
			case <-ctx.Done():
			}
		}()

		// Start streaming
		log.Info("Calling StartStreaming on controller")
		updates, err := a.controller.StartStreaming(ctx, content)
		if err != nil {
			log.Error("StartStreaming failed", "error", err)
			a.app.QueueUpdateDraw(func() {
				a.sending = false            // Clear sending state on error
				a.chatView.SetSending(false) // Stop the spinner
				a.controller.AddErrorMessage(fmt.Sprintf("Error: %v", err))
				a.chatView.UpdateMessages()
			})
			return
		}
		log.Info("StartStreaming returned successfully, processing updates")

		// Process updates
		a.processStreamingUpdates(updates)
		log.Info("processStreamingUpdates completed")
	}()
}

// processStreamingUpdates handles streaming updates from the controller
func (a *App) processStreamingUpdates(updates <-chan controllers.StreamingUpdate) {
	log := logger.WithComponent("tview_app")
	streamingContent := ""
	updateCount := 0

	log.Info("Starting to process streaming updates")

	for update := range updates {
		updateCount++
		log.Debug("Received streaming update",
			"type", update.Type,
			"update_number", updateCount,
			"content_length", len(update.Content))

		switch update.Type {
		case controllers.StreamStarted:
			log.Info("Stream started", "id", update.StreamID)
			a.app.QueueUpdateDraw(func() {
				a.streaming = true
				a.chatView.StartStreaming(update.StreamID)
				log.Debug("Called StartStreaming on chat view")
			})

		case controllers.ChunkReceived:
			// Don't accumulate special markers in the streaming content
			if update.Content != "<<<TOOL_MODE>>>" {
				streamingContent += update.Content
			}
			// But still pass the marker to the chat view for state updates
			if update.Content == "<<<TOOL_MODE>>>" {
				a.app.QueueUpdateDraw(func() {
					a.chatView.UpdateStreamingContent(update.StreamID, update.Content)
				})
			} else {
				content := streamingContent // Capture for closure
				a.app.QueueUpdateDraw(func() {
					a.chatView.UpdateStreamingContent(update.StreamID, content)
				})
			}

		case controllers.MessageComplete:
			log.Info("Message complete received",
				"id", update.StreamID,
				"accumulated_content_length", len(streamingContent),
				"accumulated_content_preview", truncateString(streamingContent, 100))
			// Use the accumulated streaming content as the final message
			finalContent := streamingContent
			a.app.QueueUpdateDraw(func() {
				log.Debug("Processing MessageComplete in UI thread",
					"was_streaming", a.streaming,
					"was_sending", a.sending,
					"final_content_length", len(finalContent))

				a.streaming = false
				a.sending = false            // Clear sending state when complete
				a.chatView.SetSending(false) // Stop the spinner

				// Add the accumulated content as the final assistant message
				if finalContent != "" {
					log.Info("Adding assistant message", "content_length", len(finalContent))
					a.controller.AddAssistantMessage(finalContent)
				} else {
					log.Warn("No content to add as assistant message")
				}

				a.chatView.CompleteStreaming(update.StreamID, chat.Message{})
				a.chatView.UpdateMessages()
				log.Debug("MessageComplete processing done")
			})

		case controllers.StreamError:
			log.Error("Stream error", "error", update.Error)
			a.app.QueueUpdateDraw(func() {
				a.streaming = false
				a.sending = false            // Clear sending state on error
				a.chatView.SetSending(false) // Stop the spinner
				a.controller.AddErrorMessage(fmt.Sprintf("Stream error: %v", update.Error))
				a.chatView.UpdateMessages()
			})

		case controllers.AgentActivityUpdate:
			// Update the activity tree display with formatted output
			activityTreeStr := update.Metadata.ActivityTree
			log.Debug("Received activity update", "tree_length", len(activityTreeStr))
			a.app.QueueUpdateDraw(func() {
				if a.chatView != nil {
					// The activity tree string is already formatted by the controller
					// Just pass it to the chat view for display
					a.chatView.UpdateActivityTree(activityTreeStr)
					log.Debug("Updated chat view with activity tree")
				}
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

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// checkOllamaHealth performs an initial health check for Ollama
func (a *App) checkOllamaHealth() {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting Ollama health check")

	// Get Ollama URL from config
	ollamaURL := a.config.Ollama.URL
	if ollamaURL == "" {
		ollamaURL = "https://ollama.kitty-tetra.ts.net" // fallback
	}

	// Create Ollama client for health check
	client := ollama.NewClientWithTimeout(ollamaURL, 5*time.Second)

	// Check Ollama health with a short timeout
	health, err := client.CheckHealthWithTimeout(3 * time.Second)
	if err != nil || !health.Available {
		log.Error("Ollama health check failed", "error", err)
		// Show URL input modal for connection issues
		a.showOllamaURLModal()
		return
	}

	// Check if we have any models
	if len(health.Models) == 0 {
		log.Info("No Ollama models available")
		a.showModelDownloadModal()
		return
	}

	// Check if the configured model is available
	configuredModel := a.config.Ollama.Model
	if configuredModel != "" {
		hasModel, err := client.CheckModelWithTimeout(configuredModel, 2*time.Second)
		if err != nil {
			log.Error("Failed to check configured model", "model", configuredModel, "error", err)
		} else if !hasModel {
			log.Info("Configured model not available", "model", configuredModel)
			a.showOllamaModal("no_model")
			return
		}
	}

	log.Debug("Ollama health check passed", "model_count", len(health.Models))
}

// showOllamaModal displays a modal for Ollama setup issues
func (a *App) showOllamaModal(issue string) {
	log := logger.WithComponent("tui_app")
	log.Debug("Showing Ollama modal", "issue", issue)

	var modal *Modal
	modal = OllamaSetupModal(a.app, issue, func(result ModalResult) {
		log.Debug("Modal result", "result", result)

		// Close the modal first
		modal.Close(a.pages)

		switch result {
		case ModalResultConfirm:
			// User wants to proceed with setup
			if issue == "no_model" {
				a.handleModelDownload()
			}
		case ModalResultCancel:
			// User cancelled - continue without setup
			log.Debug("User cancelled Ollama setup")
		}
	})

	// Show the modal on the UI thread
	a.app.QueueUpdateDraw(func() {
		modal.Show(a.pages)
	})
}

// showOllamaURLModal shows a modal for entering Ollama URL when service is unavailable
func (a *App) showOllamaURLModal() {
	log := logger.WithComponent("tui_app")
	log.Debug("Showing Ollama URL input modal")

	// Get current URL from config
	currentURL := a.config.Ollama.URL
	if currentURL == "" {
		currentURL = "https://ollama.kitty-tetra.ts.net" // fallback
	}

	var modal *Modal
	modal = OllamaURLInputModal(a.app, currentURL, func(result ModalResult, url string) {
		log.Debug("URL modal result", "result", result, "url", url)

		// Close the modal first
		modal.Close(a.pages)

		switch result {
		case ModalResultConfirm:
			if url != "" {
				// Update session configuration with new URL
				a.handleURLUpdate(url)
			}
		case ModalResultCancel:
			// User cancelled - exit the application
			log.Debug("User cancelled URL input, exiting application")
			a.app.Stop()
		}
	})

	// Show the modal on the UI thread
	a.app.QueueUpdateDraw(func() {
		modal.Show(a.pages)
	})
}

// showModelDownloadModal shows a modal for downloading Ollama models
func (a *App) showModelDownloadModal() {
	log := logger.WithComponent("tui_app")
	log.Debug("Showing model download modal")

	// Get default model from config
	defaultModel := a.config.Ollama.Model
	if defaultModel == "" {
		defaultModel = "qwen3:latest" // fallback to config default
	}

	var downloadModal *DownloadModal
	downloadModal = NewDownloadModal(a.app, defaultModel, func(result ModalResult, modelName string) {
		log.Debug("Model download modal result", "result", result, "model_from_input", modelName)

		switch result {
		case ModalResultConfirm:
			if modelName != "" {
				// Start the download process
				a.startModelDownloadWithNewModal(downloadModal, modelName)
			} else {
				// Close modal if no model specified
				downloadModal.Close(a.pages)
			}
		case ModalResultCancel:
			// User cancelled - exit the application
			log.Debug("User cancelled model download, exiting application")
			downloadModal.Close(a.pages)
			a.app.Stop()
		}
	})

	// Show the modal on the UI thread
	a.app.QueueUpdateDraw(func() {
		downloadModal.Show(a.pages, 60, 18) // Standard download modal size
	})
}

// startModelDownloadWithNewModal starts download with the new modal system
func (a *App) startModelDownloadWithNewModal(downloadModal *DownloadModal, modelName string) {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting model download with new modal", "model_to_download", modelName)

	// Convert to progress mode
	downloadModal.ShowProgress()
	downloadModal.SetProgress("Initializing...", 0, 1)

	// Get Ollama URL from config
	ollamaURL := a.config.Ollama.URL
	if ollamaURL == "" {
		ollamaURL = "https://ollama.kitty-tetra.ts.net" // fallback
	}

	// Create Ollama client for download
	client := ollama.NewClientWithTimeout(ollamaURL, 30*time.Minute) // Long timeout for downloads

	// Start download in a goroutine to not block the UI
	go func() {
		ctx := context.Background()

		err := client.PullWithProgress(ctx, modelName, func(status string, completed, total int64) {
			// Update progress on the UI thread
			a.app.QueueUpdateDraw(func() {
				downloadModal.SetProgress(status, completed, total)
			})
		})

		// Handle download completion on the UI thread
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				log.Error("Model download failed", "model", modelName, "error", err)

				// Close download modal and show error modal
				downloadModal.Close(a.pages)

				// Create and show error modal
				errorMessage := fmt.Sprintf("Failed to download model: %s\n\n%s", modelName, err.Error())
				errorModal := NewErrorModal(a.app, "Download Error", errorMessage, func() {
					a.app.Stop()
				})
				errorModal.Show(a.pages, 80, 20) // Larger modal for error display

			} else {
				log.Info("Model download completed", "model", modelName)
				downloadModal.SetProgress("Complete", 100, 100)

				// Auto-close after a brief pause
				go func() {
					time.Sleep(2 * time.Second)
					a.app.QueueUpdateDraw(func() {
						downloadModal.Close(a.pages)
					})
				}()
			}
		})
	}()
}

// startModelDownload starts the actual model download with progress tracking
func (a *App) startModelDownload(modal *Modal, modelName string) {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting model download", "model_to_download", modelName)

	// Update the modal to show progress
	modal.SetMessage(fmt.Sprintf("Downloading model: %s\n\nThis may take several minutes...", modelName))
	modal.ShowProgress()
	modal.SetProgressLabel("Initializing...")
	modal.HideButtons()

	// Get Ollama URL from config
	ollamaURL := a.config.Ollama.URL
	if ollamaURL == "" {
		ollamaURL = "https://ollama.kitty-tetra.ts.net" // fallback
	}

	// Create Ollama client for download
	client := ollama.NewClientWithTimeout(ollamaURL, 30*time.Minute) // Long timeout for downloads

	// Start download in a goroutine to not block the UI
	go func() {
		ctx := context.Background()

		err := client.PullWithProgress(ctx, modelName, func(status string, completed, total int64) {
			// Update progress on the UI thread
			a.app.QueueUpdateDraw(func() {
				modal.SetProgressLabel(status)
				modal.SetProgress(completed, total)

				// Update message with progress info
				if total > 0 {
					percentage := int(completed * 100 / total)
					modal.SetMessage(fmt.Sprintf("Downloading model: %s\n\n%s (%d%%)", modelName, status, percentage))
				} else {
					modal.SetMessage(fmt.Sprintf("Downloading model: %s\n\n%s", modelName, status))
				}
			})
		})

		// Handle download completion on the UI thread
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				log.Error("Model download failed", "model", modelName, "error", err)

				// Extract and categorize the error for better user feedback
				errorMsg := err.Error()
				var userMessage string

				if strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "network") {
					// Ollama is not available - close this modal and show URL input modal
					modal.Close(a.pages)
					a.showOllamaURLModal()
					return
				} else {
					// Show only the error message, let it take up full modal space
					userMessage = fmt.Sprintf("Download Failed: %s\n\n%s\n\nPress any key to exit", modelName, errorMsg)
				}

				// Convert modal to error-only display
				modal.SetMessage(userMessage)
				modal.HideProgress()
				modal.HideButtons()
				modal.SetTitle("Error")

				// Make the message area flexible to accommodate the full error
				modal.ResizeForError()

				// Re-show the modal with larger size to accommodate error text
				// This will automatically remove the old modal first
				modal.ShowWithSize(a.pages, 80, 20) // Larger modal for error display

				// Set up any-key-to-quit behavior
				modal.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					a.app.Stop()
					return nil
				})
			} else {
				log.Info("Model download completed", "model", modelName)
				modal.SetMessage(fmt.Sprintf("Successfully downloaded: %s\n\nModel is ready to use!", modelName))
				modal.SetProgressLabel("Complete")
				modal.SetProgress(100, 100)

				// Auto-close after a brief pause
				go func() {
					time.Sleep(2 * time.Second)
					a.app.QueueUpdateDraw(func() {
						modal.Close(a.pages)
					})
				}()
			}
		})
	}()
}

// handleModelDownload handles the model download process
func (a *App) handleModelDownload() {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting model download")

	// For now, just show a message that download would happen
	// TODO: Implement actual download in next iteration

	// Show download modal
	var downloadModal *Modal
	downloadModal = OllamaSetupModal(a.app, "download_model", func(result ModalResult) {
		log.Debug("Download modal result", "result", result)

		// Close the modal
		downloadModal.Close(a.pages)

		// For now, just close the modal
		// TODO: Implement actual download progress and completion
	})

	downloadModal.Show(a.pages)
}

// handleURLUpdate updates the session configuration with a new Ollama URL
func (a *App) handleURLUpdate(url string) {
	log := logger.WithComponent("tui_app")
	log.Debug("Updating Ollama URL for session", "url", url)

	// Update the config for this session
	a.config.Ollama.URL = url

	// Re-run health check with the new URL
	go func() {
		a.checkOllamaHealth()
	}()
}

// StartWithHealthCheck starts the app with an initial Ollama health check
func (a *App) StartWithHealthCheck() error {
	log := logger.WithComponent("tui_app")
	log.Debug("Starting TUI app with health check")

	// Perform health check in a goroutine to not block the UI
	go func() {
		// Small delay to let the UI initialize
		time.Sleep(100 * time.Millisecond)
		a.checkOllamaHealth()
	}()

	// Start the normal TUI app
	return a.app.Run()
}
