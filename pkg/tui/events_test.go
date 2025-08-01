package tui_test

import (
	"errors"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/tui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom Events", func() {
	Describe("MessageResponseEvent", func() {
		It("should create event with message", func() {
			message := chat.Message{
				Role:      chat.RoleAssistant,
				Content:   "Test response",
				Timestamp: time.Now(),
			}
			
			event := tui.NewMessageResponseEvent(message)
			
			Expect(event.Message.Role).To(Equal(chat.RoleAssistant))
			Expect(event.Message.Content).To(Equal("Test response"))
		})
	})

	Describe("MessageErrorEvent", func() {
		It("should create event with error", func() {
			err := errors.New("test error")
			
			event := tui.NewMessageErrorEvent(err)
			
			Expect(event.Error).To(Equal(err))
			Expect(event.Error.Error()).To(Equal("test error"))
		})
	})

	Describe("ChatMessageSendEvent", func() {
		It("should create event with content", func() {
			content := "Hello, world!"
			
			event := tui.NewChatMessageSendEvent(content)
			
			Expect(event.Content).To(Equal(content))
		})

		It("should handle empty content", func() {
			content := ""
			
			event := tui.NewChatMessageSendEvent(content)
			
			Expect(event.Content).To(Equal(""))
		})
	})
})

// Note: Tests are run by the existing TestTUI function in tui_suite_test.go