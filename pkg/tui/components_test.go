package tui_test

import (
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
})