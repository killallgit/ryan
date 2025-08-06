package chat_test

import (
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conversation", func() {
	var testTime time.Time

	BeforeEach(func() {
		testTime = time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	})

	Describe("NewConversation", func() {
		It("should create an empty conversation with model", func() {
			conv := chat.NewConversation("llama3.1:8b")

			Expect(conv.Model).To(Equal("llama3.1:8b"))
			messages := chat.GetMessages(conv)
			Expect(messages).To(BeEmpty())
			Expect(chat.IsEmpty(conv)).To(BeTrue())
		})
	})

	Describe("NewConversationFromTree", func() {
		It("should create conversation from existing context tree", func() {
			tree := chat.NewContextTree()

			// Add messages to tree using the root context
			tree.AddMessage(chat.NewUserMessage("Hello"), tree.RootContextID)
			tree.AddMessage(chat.NewAssistantMessage("Hi"), tree.RootContextID)

			// Create conversation from tree
			conv := chat.NewConversationFromTree(tree, "test-model")
			Expect(conv).ToNot(BeNil())
			Expect(conv.Tree).To(Equal(tree))
			Expect(conv.Model).To(Equal("test-model"))

			// Verify messages are accessible
			messages := chat.GetMessages(conv)
			Expect(messages).To(HaveLen(2))
			Expect(messages[0].Content).To(Equal("Hello"))
			Expect(messages[1].Content).To(Equal("Hi"))
		})
	})

	Describe("NewConversationWithSystem", func() {
		It("should create conversation with system message", func() {
			systemPrompt := "You are a helpful assistant"
			conv := chat.NewConversationWithSystem("llama3.1:8b", systemPrompt)

			Expect(conv.Model).To(Equal("llama3.1:8b"))
			Expect(chat.GetMessageCount(conv)).To(Equal(1))
			Expect(chat.HasSystemMessage(conv)).To(BeTrue())

			msg, found := chat.GetLastMessage(conv)
			Expect(found).To(BeTrue())
			Expect(msg.IsSystem()).To(BeTrue())
			Expect(msg.Content).To(Equal(systemPrompt))
		})

		It("should create empty conversation when system prompt is empty", func() {
			conv := chat.NewConversationWithSystem("llama3.1:8b", "")

			Expect(conv.Model).To(Equal("llama3.1:8b"))
			Expect(chat.IsEmpty(conv)).To(BeTrue())
		})
	})

	Describe("AddMessage", func() {
		It("should add message to conversation immutably", func() {
			original := chat.NewConversation("llama3.1:8b")
			msg := chat.NewUserMessage("Hello")

			updated := chat.AddMessage(original, msg)

			// Original should be unchanged
			Expect(chat.GetMessageCount(original)).To(Equal(0))

			// Updated should have new message
			Expect(chat.GetMessageCount(updated)).To(Equal(1))
			Expect(updated.Model).To(Equal("llama3.1:8b"))

			lastMsg, found := chat.GetLastMessage(updated)
			Expect(found).To(BeTrue())
			Expect(lastMsg.Content).To(Equal("Hello"))
		})

		It("should preserve message order", func() {
			conv := chat.NewConversation("llama3.1:8b")

			msg1 := chat.NewUserMessage("First")
			msg2 := chat.NewAssistantMessage("Second")
			msg3 := chat.NewUserMessage("Third")

			conv = chat.AddMessage(conv, msg1)
			conv = chat.AddMessage(conv, msg2)
			conv = chat.AddMessage(conv, msg3)

			messages := chat.GetMessages(conv)
			Expect(messages).To(HaveLen(3))
			Expect(messages[0].Content).To(Equal("First"))
			Expect(messages[1].Content).To(Equal("Second"))
			Expect(messages[2].Content).To(Equal("Third"))
		})
	})

	Describe("GetMessages", func() {
		It("should return immutable copy of messages", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("Hello"))

			messages1 := chat.GetMessages(conv)
			messages2 := chat.GetMessages(conv)

			// Should be separate slices
			Expect(&messages1[0]).ToNot(BeIdenticalTo(&messages2[0]))
			Expect(messages1[0]).To(Equal(messages2[0]))
		})
	})

	Describe("GetLastMessage", func() {
		It("should return false for empty conversation", func() {
			conv := chat.NewConversation("llama3.1:8b")

			_, found := chat.GetLastMessage(conv)
			Expect(found).To(BeFalse())
		})

		It("should return last message", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("First"))
			conv = chat.AddMessage(conv, chat.NewUserMessage("Last"))

			msg, found := chat.GetLastMessage(conv)
			Expect(found).To(BeTrue())
			Expect(msg.Content).To(Equal("Last"))
		})
	})

	Describe("GetLastAssistantMessage", func() {
		It("should return false when no assistant messages", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("Hello"))

			_, found := chat.GetLastAssistantMessage(conv)
			Expect(found).To(BeFalse())
		})

		It("should return most recent assistant message", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("First"))
			conv = chat.AddMessage(conv, chat.NewUserMessage("User message"))
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Last assistant"))

			msg, found := chat.GetLastAssistantMessage(conv)
			Expect(found).To(BeTrue())
			Expect(msg.Content).To(Equal("Last assistant"))
			Expect(msg.IsAssistant()).To(BeTrue())
		})
	})

	Describe("GetLastUserMessage", func() {
		It("should return false when no user messages", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Hello"))

			_, found := chat.GetLastUserMessage(conv)
			Expect(found).To(BeFalse())
		})

		It("should return most recent user message", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("First user"))
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Assistant message"))
			conv = chat.AddMessage(conv, chat.NewUserMessage("Last user"))

			msg, found := chat.GetLastUserMessage(conv)
			Expect(found).To(BeTrue())
			Expect(msg.Content).To(Equal("Last user"))
			Expect(msg.IsUser()).To(BeTrue())
		})
	})

	Describe("GetMessagesByRole", func() {
		It("should return all messages for specified role", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("User 1"))
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Assistant 1"))
			conv = chat.AddMessage(conv, chat.NewUserMessage("User 2"))
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Assistant 2"))

			userMessages := chat.GetMessagesByRole(conv, chat.RoleUser)
			assistantMessages := chat.GetMessagesByRole(conv, chat.RoleAssistant)

			Expect(userMessages).To(HaveLen(2))
			Expect(assistantMessages).To(HaveLen(2))
			Expect(userMessages[0].Content).To(Equal("User 1"))
			Expect(userMessages[1].Content).To(Equal("User 2"))
		})

		It("should return empty slice for non-existent role", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("Hello"))

			systemMessages := chat.GetMessagesByRole(conv, chat.RoleSystem)
			Expect(systemMessages).To(BeEmpty())
		})
	})

	Describe("GetMessagesAfter", func() {
		It("should return messages after specified time", func() {
			conv := chat.NewConversation("llama3.1:8b")

			beforeTime := testTime.Add(-1 * time.Hour)
			afterTime := testTime.Add(1 * time.Hour)

			msg1 := chat.NewUserMessage("Before").WithTimestamp(beforeTime)
			msg2 := chat.NewUserMessage("After").WithTimestamp(afterTime)

			conv = chat.AddMessage(conv, msg1)
			conv = chat.AddMessage(conv, msg2)

			messages := chat.GetMessagesAfter(conv, testTime)
			Expect(messages).To(HaveLen(1))
			Expect(messages[0].Content).To(Equal("After"))
		})
	})

	Describe("GetMessagesBefore", func() {
		It("should return messages before specified time", func() {
			conv := chat.NewConversation("llama3.1:8b")

			beforeTime := testTime.Add(-1 * time.Hour)
			afterTime := testTime.Add(1 * time.Hour)

			msg1 := chat.NewUserMessage("Before").WithTimestamp(beforeTime)
			msg2 := chat.NewUserMessage("After").WithTimestamp(afterTime)

			conv = chat.AddMessage(conv, msg1)
			conv = chat.AddMessage(conv, msg2)

			messages := chat.GetMessagesBefore(conv, testTime)
			Expect(messages).To(HaveLen(1))
			Expect(messages[0].Content).To(Equal("Before"))
		})
	})

	Describe("HasSystemMessage", func() {
		It("should return false for conversation without system message", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewUserMessage("Hello"))

			Expect(chat.HasSystemMessage(conv)).To(BeFalse())
		})

		It("should return true for conversation with system message", func() {
			conv := chat.NewConversation("llama3.1:8b")
			conv = chat.AddMessage(conv, chat.NewSystemMessage("System prompt"))

			Expect(chat.HasSystemMessage(conv)).To(BeTrue())
		})
	})

	Describe("WithModel", func() {
		It("should return conversation with new model", func() {
			original := chat.NewConversation("llama3.1:8b")
			original = chat.AddMessage(original, chat.NewUserMessage("Hello"))

			updated := chat.WithModel(original, "gpt-4")

			Expect(updated.Model).To(Equal("gpt-4"))
			Expect(chat.GetMessageCount(updated)).To(Equal(1))

			// Original should be unchanged
			Expect(original.Model).To(Equal("llama3.1:8b"))
		})
	})

	Describe("GetMessageCount", func() {
		It("should return correct count", func() {
			conv := chat.NewConversation("llama3.1:8b")

			Expect(chat.GetMessageCount(conv)).To(Equal(0))

			conv = chat.AddMessage(conv, chat.NewUserMessage("Hello"))
			Expect(chat.GetMessageCount(conv)).To(Equal(1))

			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Hi"))
			Expect(chat.GetMessageCount(conv)).To(Equal(2))
		})
	})

	// Tests for conversation management

	Describe("Message deduplication", func() {
		It("should remove optimistic messages when adding final ones", func() {
			conv := chat.NewConversation("test-model")

			// Add optimistic message
			optimisticMsg := chat.NewOptimisticUserMessage("Hello")
			conv = chat.AddMessage(conv, optimisticMsg)
			Expect(chat.GetMessageCount(conv)).To(Equal(1))

			// Add final message with same content - should replace optimistic
			finalMsg := chat.NewUserMessage("Hello")
			conv = chat.AddMessageWithDeduplication(conv, finalMsg)

			Expect(chat.GetMessageCount(conv)).To(Equal(1))
			messages := chat.GetMessages(conv)
			Expect(messages[0].GetSource()).To(Equal(chat.MessageSourceFinal))
			Expect(messages[0].IsOptimistic()).To(BeFalse())
		})

		It("should not remove optimistic messages with different content", func() {
			conv := chat.NewConversation("test-model")

			// Add optimistic message
			optimisticMsg := chat.NewOptimisticUserMessage("Hello")
			conv = chat.AddMessage(conv, optimisticMsg)

			// Add final message with different content
			finalMsg := chat.NewUserMessage("Goodbye")
			conv = chat.AddMessageWithDeduplication(conv, finalMsg)

			Expect(chat.GetMessageCount(conv)).To(Equal(2))
		})

		It("should replace optimistic messages correctly", func() {
			conv := chat.NewConversation("test-model")

			// Add some messages
			conv = chat.AddMessage(conv, chat.NewSystemMessage("System"))
			conv = chat.AddMessage(conv, chat.NewOptimisticUserMessage("Hello"))
			conv = chat.AddMessage(conv, chat.NewAssistantMessage("Hi"))

			// Replace optimistic message
			finalMsg := chat.NewUserMessage("Hello")
			conv = chat.ReplaceOptimisticMessage(conv, "Hello", finalMsg)

			messages := chat.GetMessages(conv)
			Expect(len(messages)).To(Equal(3))
			Expect(messages[1].GetSource()).To(Equal(chat.MessageSourceFinal))
		})
	})

	Describe("Streaming message management", func() {
		It("should remove streaming messages by stream ID", func() {
			conv := chat.NewConversation("test-model")

			// Add regular and streaming messages
			conv = chat.AddMessage(conv, chat.NewUserMessage("User"))
			conv = chat.AddMessage(conv, chat.NewStreamingMessage(chat.RoleAssistant, "Chunk1", "stream-123", 1))
			conv = chat.AddMessage(conv, chat.NewStreamingMessage(chat.RoleAssistant, "Chunk2", "stream-123", 2))
			conv = chat.AddMessage(conv, chat.NewStreamingMessage(chat.RoleAssistant, "Other", "stream-456", 1))

			Expect(chat.GetMessageCount(conv)).To(Equal(4))

			// Remove streaming messages from specific stream
			conv = chat.RemoveStreamingMessages(conv, "stream-123")

			messages := chat.GetMessages(conv)
			Expect(len(messages)).To(Equal(2))
			Expect(messages[0].Content).To(Equal("User"))
			Expect(messages[1].Content).To(Equal("Other"))
		})

		It("should get streaming messages", func() {
			conv := chat.NewConversation("test-model")

			conv = chat.AddMessage(conv, chat.NewUserMessage("User"))
			conv = chat.AddMessage(conv, chat.NewStreamingMessage(chat.RoleAssistant, "Chunk", "stream-123", 1))

			streamingMsgs := chat.GetStreamingMessages(conv)
			Expect(len(streamingMsgs)).To(Equal(1))
			Expect(streamingMsgs[0].Content).To(Equal("Chunk"))
		})

		It("should get optimistic messages", func() {
			conv := chat.NewConversation("test-model")

			conv = chat.AddMessage(conv, chat.NewOptimisticUserMessage("Optimistic"))
			conv = chat.AddMessage(conv, chat.NewUserMessage("Final"))

			optimisticMsgs := chat.GetOptimisticMessages(conv)
			Expect(len(optimisticMsgs)).To(Equal(1))
			Expect(optimisticMsgs[0].Content).To(Equal("Optimistic"))
		})
	})
})
