package chat_test

import (
	"testing"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chat Suite")
}

var _ = Describe("Messages", func() {
	var testTime time.Time

	BeforeEach(func() {
		testTime = time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	})

	Describe("NewUserMessage", func() {
		It("should create a user message with trimmed content", func() {
			msg := chat.NewUserMessage("  Hello World  ")

			Expect(msg.Role).To(Equal(chat.RoleUser))
			Expect(msg.Content).To(Equal("Hello World"))
			Expect(msg.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})

		It("should handle empty content", func() {
			msg := chat.NewUserMessage("   ")

			Expect(msg.Role).To(Equal(chat.RoleUser))
			Expect(msg.Content).To(Equal(""))
			Expect(msg.IsEmpty()).To(BeTrue())
		})
	})

	Describe("NewAssistantMessage", func() {
		It("should create an assistant message", func() {
			msg := chat.NewAssistantMessage("Hello there!")

			Expect(msg.Role).To(Equal(chat.RoleAssistant))
			Expect(msg.Content).To(Equal("Hello there!"))
			Expect(msg.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})
	})

	Describe("NewSystemMessage", func() {
		It("should create a system message", func() {
			msg := chat.NewSystemMessage("You are a helpful assistant")

			Expect(msg.Role).To(Equal(chat.RoleSystem))
			Expect(msg.Content).To(Equal("You are a helpful assistant"))
			Expect(msg.Timestamp).To(BeTemporally("~", time.Now(), time.Second))
		})
	})

	Describe("Message methods", func() {
		var userMsg, assistantMsg, systemMsg chat.Message

		BeforeEach(func() {
			userMsg = chat.NewUserMessage("User message")
			assistantMsg = chat.NewAssistantMessage("Assistant message")
			systemMsg = chat.NewSystemMessage("System message")
		})

		It("should correctly identify user messages", func() {
			Expect(userMsg.IsUser()).To(BeTrue())
			Expect(userMsg.IsAssistant()).To(BeFalse())
			Expect(userMsg.IsSystem()).To(BeFalse())
		})

		It("should correctly identify assistant messages", func() {
			Expect(assistantMsg.IsUser()).To(BeFalse())
			Expect(assistantMsg.IsAssistant()).To(BeTrue())
			Expect(assistantMsg.IsSystem()).To(BeFalse())
		})

		It("should correctly identify system messages", func() {
			Expect(systemMsg.IsUser()).To(BeFalse())
			Expect(systemMsg.IsAssistant()).To(BeFalse())
			Expect(systemMsg.IsSystem()).To(BeTrue())
		})

		It("should detect empty messages", func() {
			emptyMsg := chat.NewUserMessage("")
			nonEmptyMsg := chat.NewUserMessage("Hello")

			Expect(emptyMsg.IsEmpty()).To(BeTrue())
			Expect(nonEmptyMsg.IsEmpty()).To(BeFalse())
		})

		It("should detect whitespace-only messages as empty", func() {
			whitespaceMsg := chat.Message{
				Role:    chat.RoleUser,
				Content: "   \t\n  ",
			}

			Expect(whitespaceMsg.IsEmpty()).To(BeTrue())
		})
	})

	Describe("WithTimestamp", func() {
		It("should create a new message with specified timestamp", func() {
			original := chat.NewUserMessage("Hello")
			updated := original.WithTimestamp(testTime)

			Expect(updated.Role).To(Equal(original.Role))
			Expect(updated.Content).To(Equal(original.Content))
			Expect(updated.Timestamp).To(Equal(testTime))

			// Original should be unchanged
			Expect(original.Timestamp).ToNot(Equal(testTime))
		})
	})

	Describe("Role constants", func() {
		It("should have correct role constants", func() {
			Expect(chat.RoleUser).To(Equal("user"))
			Expect(chat.RoleAssistant).To(Equal("assistant"))
			Expect(chat.RoleSystem).To(Equal("system"))
		})
	})

	// New tests for enhanced message architecture
	
	Describe("ThinkingBlock", func() {
		It("should create messages with thinking blocks", func() {
			msg := chat.NewAssistantMessageWithThinking("Response", "My thinking", true)
			
			Expect(msg.HasThinking()).To(BeTrue())
			Expect(msg.IsThinkingVisible()).To(BeTrue())
			Expect(msg.Thinking.Content).To(Equal("My thinking"))
			Expect(msg.Thinking.Visible).To(BeTrue())
			Expect(msg.Content).To(Equal("Response"))
		})
		
		It("should handle invisible thinking blocks", func() {
			msg := chat.NewAssistantMessageWithThinking("Response", "Hidden thinking", false)
			
			Expect(msg.HasThinking()).To(BeTrue())
			Expect(msg.IsThinkingVisible()).To(BeFalse())
			Expect(msg.Thinking.Visible).To(BeFalse())
		})
		
		It("should handle messages without thinking blocks", func() {
			msg := chat.NewAssistantMessage("Simple response")
			
			Expect(msg.HasThinking()).To(BeFalse())
			Expect(msg.IsThinkingVisible()).To(BeFalse())
		})
	})
	
	Describe("MessageMetadata", func() {
		It("should create optimistic messages", func() {
			msg := chat.NewOptimisticUserMessage("Test message")
			
			Expect(msg.IsOptimistic()).To(BeTrue())
			Expect(msg.GetSource()).To(Equal(chat.MessageSourceOptimistic))
			Expect(msg.HasMetadata()).To(BeTrue())
		})
		
		It("should create streaming messages", func() {
			msg := chat.NewStreamingMessage(chat.RoleAssistant, "Chunk", "stream-123", 5)
			
			Expect(msg.IsStreaming()).To(BeTrue())
			Expect(msg.GetStreamID()).To(Equal("stream-123"))
			Expect(msg.GetSource()).To(Equal(chat.MessageSourceStreaming))
			Expect(msg.Metadata.ChunkIndex).To(Equal(5))
		})
		
		It("should handle final messages", func() {
			msg := chat.NewUserMessage("Final message")
			
			Expect(msg.IsOptimistic()).To(BeFalse())
			Expect(msg.IsStreaming()).To(BeFalse())
			Expect(msg.GetSource()).To(Equal(chat.MessageSourceFinal))
		})
	})
	
	Describe("Message helpers", func() {
		It("should add thinking to existing messages", func() {
			msg := chat.NewAssistantMessage("Response")
			updated := msg.WithThinking("Added thinking", true)
			
			Expect(updated.HasThinking()).To(BeTrue())
			Expect(updated.Thinking.Content).To(Equal("Added thinking"))
			Expect(updated.Thinking.Visible).To(BeTrue())
			
			// Original should be unchanged
			Expect(msg.HasThinking()).To(BeFalse())
		})
		
		It("should update message source", func() {
			msg := chat.NewUserMessage("Test")
			updated := msg.WithSource(chat.MessageSourceOptimistic)
			
			Expect(updated.GetSource()).To(Equal(chat.MessageSourceOptimistic))
			Expect(updated.IsOptimistic()).To(BeTrue())
			
			// Original should be unchanged
			Expect(msg.GetSource()).To(Equal(chat.MessageSourceFinal))
		})
	})
})
