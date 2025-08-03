package chat_test

import (
	"github.com/killallgit/ryan/pkg/chat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client *chat.Client
	)

	Describe("NewClient", func() {
		It("should create a new client successfully", func() {
			var err error
			client, err = chat.NewClient("http://localhost:11434", "llama3.1:8b")
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())
		})

		It("should handle invalid configuration gracefully", func() {
			var err error
			client, err = chat.NewClient("", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())
		})
	})

	Describe("Error handling", func() {
		BeforeEach(func() {
			var err error
			client, err = chat.NewClient("http://invalid-url:11434", "llama3.1:8b")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle connection errors", func() {
			req := chat.ChatRequest{
				Model: "llama3.1:8b",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			_, err := client.SendMessage(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("LangChain content generation failed"))
		})
	})
})

var _ = Describe("CreateChatRequest", func() {
	It("should create request with user message added to conversation", func() {
		conv := chat.NewConversation("llama3.1:8b")
		conv = chat.AddMessage(conv, chat.NewSystemMessage("You are helpful"))

		req := chat.CreateChatRequest(conv, "Hello there")

		Expect(req.Model).To(Equal("llama3.1:8b"))
		Expect(req.Stream).To(BeFalse())
		Expect(req.Messages).To(HaveLen(2))

		Expect(req.Messages[0].Role).To(Equal(chat.RoleSystem))
		Expect(req.Messages[0].Content).To(Equal("You are helpful"))

		Expect(req.Messages[1].Role).To(Equal(chat.RoleUser))
		Expect(req.Messages[1].Content).To(Equal("Hello there"))
	})

	It("should handle empty conversation", func() {
		conv := chat.NewConversation("gpt-4")

		req := chat.CreateChatRequest(conv, "First message")

		Expect(req.Model).To(Equal("gpt-4"))
		Expect(req.Messages).To(HaveLen(1))
		Expect(req.Messages[0].Role).To(Equal(chat.RoleUser))
		Expect(req.Messages[0].Content).To(Equal("First message"))
	})
})
