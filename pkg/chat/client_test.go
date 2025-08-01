package chat_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		client *chat.Client
		server *httptest.Server
	)

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal("POST"))
			Expect(r.URL.Path).To(Equal("/api/chat"))
			Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

			var req chat.ChatRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			Expect(err).ToNot(HaveOccurred())

			Expect(req.Stream).To(BeFalse())
			Expect(req.Model).To(Equal("llama3.1:8b"))
			Expect(req.Messages).To(HaveLen(1))
			Expect(req.Messages[0].Role).To(Equal(chat.RoleUser))
			Expect(req.Messages[0].Content).To(Equal("Hello"))

			response := chat.ChatResponse{
				Model:     "llama3.1:8b",
				CreatedAt: time.Now(),
				Message: chat.Message{
					Role:      chat.RoleAssistant,
					Content:   "Hi there!",
					Timestamp: time.Now(),
				},
				Done: true,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))

		client = chat.NewClient(server.URL)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("SendMessage", func() {
		It("should send request and receive response", func() {
			req := chat.ChatRequest{
				Model: "llama3.1:8b",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
				Stream: true, // Should be overridden to false
			}

			message, err := client.SendMessage(req)

			Expect(err).ToNot(HaveOccurred())
			Expect(message.Role).To(Equal(chat.RoleAssistant))
			Expect(message.Content).To(Equal("Hi there!"))
		})
	})

	Describe("Error handling", func() {
		BeforeEach(func() {
			client = chat.NewClient("http://invalid-url")
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
			Expect(err.Error()).To(ContainSubstring("request failed"))
		})
	})

	Describe("HTTP error responses", func() {
		BeforeEach(func() {
			server.Close()
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			client = chat.NewClient(server.URL)
		})

		It("should handle HTTP error status codes", func() {
			req := chat.ChatRequest{
				Model: "llama3.1:8b",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			_, err := client.SendMessage(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("request failed with status 500"))
		})
	})

	Describe("Invalid JSON response", func() {
		BeforeEach(func() {
			server.Close()
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			}))
			client = chat.NewClient(server.URL)
		})

		It("should handle invalid JSON responses", func() {
			req := chat.ChatRequest{
				Model: "llama3.1:8b",
				Messages: []chat.Message{
					chat.NewUserMessage("Hello"),
				},
			}

			_, err := client.SendMessage(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to decode response"))
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
