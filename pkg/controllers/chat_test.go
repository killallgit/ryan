package controllers_test

import (
	"errors"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)


type MockChatClient struct {
	mock.Mock
}

func (m *MockChatClient) SendMessage(req chat.ChatRequest) (chat.Message, error) {
	args := m.Called(req)
	return args.Get(0).(chat.Message), args.Error(1)
}

var _ = Describe("ChatController", func() {
	var (
		mockClient *MockChatClient
		controller *controllers.ChatController
	)

	BeforeEach(func() {
		mockClient = &MockChatClient{}
		controller = controllers.NewChatController(mockClient, "llama3.1:8b")
	})

	AfterEach(func() {
		mockClient.AssertExpectations(GinkgoT())
	})

	Describe("NewChatController", func() {
		It("should create controller with empty conversation", func() {
			Expect(controller.GetModel()).To(Equal("llama3.1:8b"))
			Expect(controller.GetMessageCount()).To(Equal(0))
			Expect(controller.HasSystemMessage()).To(BeFalse())
		})
	})

	Describe("NewChatControllerWithSystem", func() {
		It("should create controller with system message", func() {
			systemPrompt := "You are a helpful assistant"
			controller = controllers.NewChatControllerWithSystem(mockClient, "gpt-4", systemPrompt)
			
			Expect(controller.GetModel()).To(Equal("gpt-4"))
			Expect(controller.GetMessageCount()).To(Equal(1))
			Expect(controller.HasSystemMessage()).To(BeTrue())
		})

		It("should create controller without system message when empty", func() {
			controller = controllers.NewChatControllerWithSystem(mockClient, "gpt-4", "")
			
			Expect(controller.GetModel()).To(Equal("gpt-4"))
			Expect(controller.GetMessageCount()).To(Equal(0))
			Expect(controller.HasSystemMessage()).To(BeFalse())
		})
	})

	Describe("SendUserMessage", func() {
		It("should send message and update conversation", func() {
			assistantResponse := chat.NewAssistantMessage("Hello there!")
			
			mockClient.On("SendMessage", mock.MatchedBy(func(req chat.ChatRequest) bool {
				return req.Model == "llama3.1:8b" &&
					len(req.Messages) == 1 &&
					req.Messages[0].Role == chat.RoleUser &&
					req.Messages[0].Content == "Hello" &&
					req.Stream == false
			})).Return(assistantResponse, nil)

			response, err := controller.SendUserMessage("Hello")

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Content).To(Equal("Hello there!"))
			Expect(response.Role).To(Equal(chat.RoleAssistant))
			
			// Verify conversation state
			Expect(controller.GetMessageCount()).To(Equal(2))
			
			userMsg, found := controller.GetLastUserMessage()
			Expect(found).To(BeTrue())
			Expect(userMsg.Content).To(Equal("Hello"))
			
			assistantMsg, found := controller.GetLastAssistantMessage()
			Expect(found).To(BeTrue())
			Expect(assistantMsg.Content).To(Equal("Hello there!"))
		})

		It("should handle empty message content", func() {
			_, err := controller.SendUserMessage("")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("message content cannot be empty"))
			Expect(controller.GetMessageCount()).To(Equal(0))
		})

		It("should handle whitespace-only message content", func() {
			_, err := controller.SendUserMessage("   ")
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("message content cannot be empty"))
			Expect(controller.GetMessageCount()).To(Equal(0))
		})

		It("should handle client errors", func() {
			mockClient.On("SendMessage", mock.Anything).Return(chat.Message{}, errors.New("network error"))

			_, err := controller.SendUserMessage("Hello")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to send message"))
			Expect(err.Error()).To(ContainSubstring("network error"))
			
			// Conversation should not be updated on error
			Expect(controller.GetMessageCount()).To(Equal(0))
		})
	})

	Describe("Conversation management", func() {
		BeforeEach(func() {
			assistantResponse := chat.NewAssistantMessage("Response")
			mockClient.On("SendMessage", mock.Anything).Return(assistantResponse, nil)
			
			_, err := controller.SendUserMessage("Test message")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should provide access to conversation history", func() {
			history := controller.GetHistory()
			
			Expect(history).To(HaveLen(2))
			Expect(history[0].Role).To(Equal(chat.RoleUser))
			Expect(history[0].Content).To(Equal("Test message"))
			Expect(history[1].Role).To(Equal(chat.RoleAssistant))
			Expect(history[1].Content).To(Equal("Response"))
		})

		It("should provide access to full conversation", func() {
			conv := controller.GetConversation()
			
			Expect(conv.Model).To(Equal("llama3.1:8b"))
			Expect(chat.GetMessageCount(conv)).To(Equal(2))
		})
	})

	Describe("Model management", func() {
		It("should allow model changes", func() {
			controller.SetModel("gpt-4")
			
			Expect(controller.GetModel()).To(Equal("gpt-4"))
		})
	})

	Describe("Reset", func() {
		It("should reset conversation without system message", func() {
			assistantResponse := chat.NewAssistantMessage("Response")
			mockClient.On("SendMessage", mock.Anything).Return(assistantResponse, nil)
			
			_, err := controller.SendUserMessage("Test")
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.GetMessageCount()).To(Equal(2))
			
			controller.Reset()
			
			Expect(controller.GetMessageCount()).To(Equal(0))
			Expect(controller.HasSystemMessage()).To(BeFalse())
			Expect(controller.GetModel()).To(Equal("llama3.1:8b"))
		})

		It("should reset conversation but preserve system message", func() {
			systemPrompt := "You are helpful"
			controller = controllers.NewChatControllerWithSystem(mockClient, "gpt-4", systemPrompt)
			
			assistantResponse := chat.NewAssistantMessage("Response")
			mockClient.On("SendMessage", mock.Anything).Return(assistantResponse, nil)
			
			_, err := controller.SendUserMessage("Test")
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.GetMessageCount()).To(Equal(3)) // system + user + assistant
			
			controller.Reset()
			
			Expect(controller.GetMessageCount()).To(Equal(1)) // just system
			Expect(controller.HasSystemMessage()).To(BeTrue())
			Expect(controller.GetModel()).To(Equal("gpt-4"))
			
			// Verify system message is preserved
			history := controller.GetHistory()
			Expect(history[0].Role).To(Equal(chat.RoleSystem))
			Expect(history[0].Content).To(Equal(systemPrompt))
		})
	})

	Describe("With system message", func() {
		BeforeEach(func() {
			controller = controllers.NewChatControllerWithSystem(mockClient, "gpt-4", "You are helpful")
		})

		It("should include system message in requests", func() {
			assistantResponse := chat.NewAssistantMessage("Hello!")
			
			mockClient.On("SendMessage", mock.MatchedBy(func(req chat.ChatRequest) bool {
				return len(req.Messages) == 2 &&
					req.Messages[0].Role == chat.RoleSystem &&
					req.Messages[0].Content == "You are helpful" &&
					req.Messages[1].Role == chat.RoleUser &&
					req.Messages[1].Content == "Hi"
			})).Return(assistantResponse, nil)

			_, err := controller.SendUserMessage("Hi")
			
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.GetMessageCount()).To(Equal(3)) // system + user + assistant
		})
	})
})