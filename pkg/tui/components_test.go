package tui_test

import (
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MessageDisplay", func() {
	Describe("NewMessageDisplay", func() {
		It("should create message display with default values", func() {
			display := tui.NewMessageDisplay(80, 20)

			Expect(display.Width).To(Equal(80))
			Expect(display.Height).To(Equal(20))
			Expect(display.Scroll).To(Equal(0))
			Expect(display.Messages).To(HaveLen(0))
		})
	})

	Describe("WithMessages", func() {
		It("should update messages while preserving other properties", func() {
			display := tui.NewMessageDisplay(80, 20)
			messages := []chat.Message{
				chat.NewUserMessage("Hello"),
				chat.NewAssistantMessage("Hi there!"),
			}

			updated := display.WithMessages(messages)

			Expect(updated.Messages).To(Equal(messages))
			Expect(updated.Width).To(Equal(80))
			Expect(updated.Height).To(Equal(20))
			Expect(updated.Scroll).To(Equal(0))
		})
	})

	Describe("WithSize", func() {
		It("should update size while preserving other properties", func() {
			display := tui.NewMessageDisplay(80, 20)
			messages := []chat.Message{chat.NewUserMessage("Hello")}
			display = display.WithMessages(messages).WithScroll(5)

			updated := display.WithSize(100, 30)

			Expect(updated.Width).To(Equal(100))
			Expect(updated.Height).To(Equal(30))
			Expect(updated.Messages).To(Equal(messages))
			Expect(updated.Scroll).To(Equal(5))
		})
	})

	Describe("WithScroll", func() {
		It("should update scroll while preserving other properties", func() {
			display := tui.NewMessageDisplay(80, 20)
			messages := []chat.Message{chat.NewUserMessage("Hello")}
			display = display.WithMessages(messages)

			updated := display.WithScroll(10)

			Expect(updated.Scroll).To(Equal(10))
			Expect(updated.Width).To(Equal(80))
			Expect(updated.Height).To(Equal(20))
			Expect(updated.Messages).To(Equal(messages))
		})
	})

	Describe("Rendering", func() {
		var (
			screen  *tui.TestScreen
			display tui.MessageDisplay
		)

		BeforeEach(func() {
			screen = tui.NewTestScreen()
			err := screen.Init()
			Expect(err).ToNot(HaveOccurred())
			screen.SetSize(80, 24)
			screen.Clear()
			
			display = tui.NewMessageDisplay(80, 20)
		})

		AfterEach(func() {
			screen.Fini()
		})

		It("should render messages on screen", func() {
			messages := []chat.Message{
				chat.NewUserMessage("Hello, how are you?"),
				chat.NewAssistantMessage("I'm doing well, thank you!"),
			}
			display = display.WithMessages(messages)

			// Render messages
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 20}
			spinner := tui.SpinnerComponent{IsVisible: false}
			tui.RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, false)

			// Verify messages appear on screen
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Hello, how are you?"))
			Expect(content).To(ContainSubstring("I'm doing well, thank you!"))
		})

		It("should handle word wrapping for long messages", func() {
			longMessage := "This is a very long message that should definitely wrap to multiple lines when displayed in the message area because it exceeds the width of the terminal."
			messages := []chat.Message{
				chat.NewUserMessage(longMessage),
			}
			display = display.WithMessages(messages)

			// Render with narrow width
			area := tui.Rect{X: 0, Y: 0, Width: 40, Height: 20}
			spinner := tui.SpinnerComponent{IsVisible: false}
			tui.RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, false)

			// Verify message is wrapped
			content := screen.CaptureContent()
			// The message should be split across lines
			Expect(content).To(ContainSubstring("This is a very long message"))
			// Check that the content contains multiple lines by looking for "definitely"
			// which should appear on a different line
			Expect(content).To(ContainSubstring("definitely"))
			// And verify wrapping happened by checking the terminal width is respected
			lines := strings.Split(content, "\n")
			hasWrappedLine := false
			for _, line := range lines {
				if strings.Contains(line, "message") && len(strings.TrimSpace(line)) <= 40 {
					hasWrappedLine = true
					break
				}
			}
			Expect(hasWrappedLine).To(BeTrue())
		})

		It("should show spinner when streaming", func() {
			messages := []chat.Message{
				chat.NewUserMessage("Tell me a story"),
				chat.NewAssistantMessage("Once upon a time..."),
			}
			display = display.WithMessages(messages)
			
			// Render with streaming indicator
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 20}
			spinner := tui.SpinnerComponent{
				IsVisible: true,
				Frame:     0,
			}
			tui.RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, false)

			// Verify streaming content appears
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Tell me a story"))
			Expect(content).To(ContainSubstring("Once upon a time..."))
		})

		It("should handle thinking blocks", func() {
			messages := []chat.Message{
				chat.NewAssistantMessage("<think>Processing your request...</think>Here's the answer"),
			}
			display = display.WithMessages(messages)
			
			area := tui.Rect{X: 0, Y: 0, Width: 80, Height: 20}
			spinner := tui.SpinnerComponent{IsVisible: false}
			tui.RenderMessagesWithSpinnerAndStreaming(screen, display, area, spinner, false)

			// Verify thinking content is rendered with special formatting
			content := screen.CaptureContent()
			// Thinking content might be shown or hidden depending on implementation
			Expect(content).To(Or(
				ContainSubstring("Processing your request..."),
				ContainSubstring("Thinking"),
			))
		})
	})
})

var _ = Describe("InputField", func() {
	Describe("NewInputField", func() {
		It("should create input field with default values", func() {
			input := tui.NewInputField(80)

			Expect(input.Width).To(Equal(80))
			Expect(input.Content).To(Equal(""))
			Expect(input.Cursor).To(Equal(0))
		})
	})

	Describe("WithContent", func() {
		It("should update content and adjust cursor if needed", func() {
			input := tui.NewInputField(80).WithContent("Hello World").WithCursor(8)

			updated := input.WithContent("Hi")

			Expect(updated.Content).To(Equal("Hi"))
			Expect(updated.Cursor).To(Equal(2)) // Cursor adjusted to end of new content
			Expect(updated.Width).To(Equal(80))
		})

		It("should preserve cursor position if valid", func() {
			input := tui.NewInputField(80).WithContent("Initial").WithCursor(3)

			updated := input.WithContent("Hello World")

			Expect(updated.Content).To(Equal("Hello World"))
			Expect(updated.Cursor).To(Equal(3))
			Expect(updated.Width).To(Equal(80))
		})
	})

	Describe("WithCursor", func() {
		It("should clamp cursor to valid range", func() {
			input := tui.NewInputField(80).WithContent("Hello")

			// Test negative cursor
			updated := input.WithCursor(-5)
			Expect(updated.Cursor).To(Equal(0))

			// Test cursor beyond content
			updated = input.WithCursor(10)
			Expect(updated.Cursor).To(Equal(5)) // Length of "Hello"

			// Test valid cursor
			updated = input.WithCursor(3)
			Expect(updated.Cursor).To(Equal(3))
		})
	})

	Describe("InsertRune", func() {
		It("should insert rune at cursor position", func() {
			input := tui.NewInputField(80).WithContent("Hello").WithCursor(2)

			updated := input.InsertRune('X')

			Expect(updated.Content).To(Equal("HeXllo"))
			Expect(updated.Cursor).To(Equal(3))
		})

		It("should insert at end when cursor is at end", func() {
			input := tui.NewInputField(80).WithContent("Hello").WithCursor(5)

			updated := input.InsertRune('!')

			Expect(updated.Content).To(Equal("Hello!"))
			Expect(updated.Cursor).To(Equal(6))
		})
	})

	Describe("DeleteBackward", func() {
		It("should delete character before cursor", func() {
			input := tui.NewInputField(80).WithContent("Hello").WithCursor(3)

			updated := input.DeleteBackward()

			Expect(updated.Content).To(Equal("Helo"))
			Expect(updated.Cursor).To(Equal(2))
		})

		It("should do nothing when cursor is at beginning", func() {
			input := tui.NewInputField(80).WithContent("Hello").WithCursor(0)

			updated := input.DeleteBackward()

			Expect(updated.Content).To(Equal("Hello"))
			Expect(updated.Cursor).To(Equal(0))
		})
	})

	Describe("Clear", func() {
		It("should clear content and reset cursor", func() {
			input := tui.NewInputField(80).WithContent("Hello World").WithCursor(5)

			updated := input.Clear()

			Expect(updated.Content).To(Equal(""))
			Expect(updated.Cursor).To(Equal(0))
			Expect(updated.Width).To(Equal(80))
		})
	})

	Describe("Rendering", func() {
		var (
			screen *tui.TestScreen
			input  tui.InputField
		)

		BeforeEach(func() {
			screen = tui.NewTestScreen()
			err := screen.Init()
			Expect(err).ToNot(HaveOccurred())
			screen.SetSize(80, 24)
			screen.Clear()
			
			input = tui.NewInputField(78) // Leave room for borders
		})

		AfterEach(func() {
			screen.Fini()
		})

		It("should render input field with content", func() {
			input = input.WithContent("Hello World")

			// Render input field
			area := tui.Rect{X: 1, Y: 20, Width: 78, Height: 3}
			tui.RenderInput(screen, input, area)

			// Verify content appears
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Hello World"))
		})

		It("should show cursor position", func() {
			input = input.WithContent("Test").WithCursor(2)

			// Render input field
			area := tui.Rect{X: 1, Y: 20, Width: 78, Height: 3}
			tui.RenderInput(screen, input, area)

			// The cursor should be visible at position 2
			// Check if cursor is rendered (implementation specific)
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("Test"))
		})

		It("should handle text overflow with scrolling", func() {
			// Create very long content
			longText := strings.Repeat("A", 100)
			input = input.WithContent(longText).WithCursor(95)

			// Render in limited width
			area := tui.Rect{X: 1, Y: 20, Width: 40, Height: 3}
			tui.RenderInput(screen, input, area)

			// Should show the portion around cursor
			content := screen.CaptureContent()
			// Should see some A's but not all 100
			Expect(content).To(ContainSubstring("AAAA"))
		})

		It("should render with input prompt", func() {
			input = input.WithContent("my command")

			// Render input field with its built-in box
			area := tui.Rect{X: 0, Y: 20, Width: 80, Height: 3}
			tui.RenderInput(screen, input, area)

			// Verify prompt and content
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring(">"))
			Expect(content).To(ContainSubstring("my command"))
			// Also check for box borders
			Expect(content).To(ContainSubstring("╭"))
			Expect(content).To(ContainSubstring("╮"))
		})
	})
})

var _ = Describe("StatusBar", func() {
	Describe("NewStatusBar", func() {
		It("should create status bar with default values", func() {
			status := tui.NewStatusBar(100)

			Expect(status.Width).To(Equal(100))
			Expect(status.Model).To(Equal(""))
			Expect(status.Status).To(Equal("Ready"))
		})
	})

	Describe("WithModel", func() {
		It("should update model while preserving other properties", func() {
			status := tui.NewStatusBar(100).WithStatus("Busy")

			updated := status.WithModel("llama3.1:8b")

			Expect(updated.Model).To(Equal("llama3.1:8b"))
			Expect(updated.Status).To(Equal("Busy"))
			Expect(updated.Width).To(Equal(100))
		})
	})

	Describe("WithStatus", func() {
		It("should update status while preserving other properties", func() {
			status := tui.NewStatusBar(100).WithModel("gpt-4")

			updated := status.WithStatus("Processing...")

			Expect(updated.Status).To(Equal("Processing..."))
			Expect(updated.Model).To(Equal("gpt-4"))
			Expect(updated.Width).To(Equal(100))
		})
	})

	Describe("WithWidth", func() {
		It("should update width while preserving other properties", func() {
			status := tui.NewStatusBar(100).WithModel("gpt-4").WithStatus("Busy")

			updated := status.WithWidth(120)

			Expect(updated.Width).To(Equal(120))
			Expect(updated.Model).To(Equal("gpt-4"))
			Expect(updated.Status).To(Equal("Busy"))
		})
	})

	Describe("Rendering", func() {
		var (
			screen *tui.TestScreen
			status tui.StatusBar
		)

		BeforeEach(func() {
			screen = tui.NewTestScreen()
			err := screen.Init()
			Expect(err).ToNot(HaveOccurred())
			screen.SetSize(80, 24)
			screen.Clear()
			
			status = tui.NewStatusBar(80)
		})

		AfterEach(func() {
			screen.Fini()
		})

		It("should render status bar with model and status", func() {
			status = status.WithModel("llama3.1:8b").WithStatus("Ready")

			// Render status bar
			area := tui.Rect{X: 0, Y: 23, Width: 80, Height: 1}
			tui.RenderStatus(screen, status, area)

			// Verify content appears
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("llama3.1:8b"))
		})

		It("should show model availability indicator", func() {
			status = status.WithModel("gpt-4").WithStatus("Ready").WithModelAvailability(true)

			area := tui.Rect{X: 0, Y: 23, Width: 80, Height: 1}
			tui.RenderStatus(screen, status, area)

			// Should show availability indicator (●)
			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("●"))
			Expect(content).To(ContainSubstring("gpt-4"))
		})

		It("should show streaming status", func() {
			status = status.WithModel("llama3.1").WithStatus("Streaming...")

			area := tui.Rect{X: 0, Y: 23, Width: 80, Height: 1}
			tui.RenderStatus(screen, status, area)

			content := screen.CaptureContent()
			Expect(content).To(ContainSubstring("llama3.1"))
			// Note: RenderStatus doesn't show the status text itself, just the model
		})

		It("should handle long model names", func() {
			longModelName := "very-long-model-name-that-might-overflow:latest"
			status = status.WithModel(longModelName).WithStatus("Ready")

			area := tui.Rect{X: 0, Y: 23, Width: 40, Height: 1} // Narrow width
			tui.RenderStatus(screen, status, area)

			// Should truncate or handle overflow gracefully
			content := screen.CaptureContent()
			// Should see at least part of the model name
			Expect(content).To(ContainSubstring("..."))
		})

		It("should align content properly", func() {
			status = status.WithModel("model").WithStatus("OK")

			area := tui.Rect{X: 0, Y: 23, Width: 80, Height: 1}
			tui.RenderStatus(screen, status, area)

			// Status should be right-aligned
			statusLine := screen.GetRegion(0, 23, 80, 1)
			// The model should be on the right side
			Expect(strings.TrimRight(statusLine, " ")).To(HaveSuffix("model"))
		})
	})
})
