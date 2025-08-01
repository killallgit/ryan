package controllers

import (
	"fmt"
	"strings"

	"github.com/killallgit/ryan/pkg/chat"
)

type ChatController struct {
	client       chat.ChatClient
	conversation chat.Conversation
}

func NewChatController(client chat.ChatClient, model string) *ChatController {
	return &ChatController{
		client:       client,
		conversation: chat.NewConversation(model),
	}
}

func NewChatControllerWithSystem(client chat.ChatClient, model, systemPrompt string) *ChatController {
	return &ChatController{
		client:       client,
		conversation: chat.NewConversationWithSystem(model, systemPrompt),
	}
}

func (cc *ChatController) SendUserMessage(content string) (chat.Message, error) {
	if strings.TrimSpace(content) == "" {
		return chat.Message{}, fmt.Errorf("message content cannot be empty")
	}

	req := chat.CreateChatRequest(cc.conversation, content)
	
	assistantMessage, err := cc.client.SendMessage(req)
	if err != nil {
		return chat.Message{}, fmt.Errorf("failed to send message: %w", err)
	}

	cc.conversation = chat.AddMessage(cc.conversation, chat.NewUserMessage(content))
	cc.conversation = chat.AddMessage(cc.conversation, assistantMessage)

	return assistantMessage, nil
}

func (cc *ChatController) GetHistory() []chat.Message {
	return chat.GetMessages(cc.conversation)
}

func (cc *ChatController) GetConversation() chat.Conversation {
	return cc.conversation
}

func (cc *ChatController) GetMessageCount() int {
	return chat.GetMessageCount(cc.conversation)
}

func (cc *ChatController) GetLastAssistantMessage() (chat.Message, bool) {
	return chat.GetLastAssistantMessage(cc.conversation)
}

func (cc *ChatController) GetLastUserMessage() (chat.Message, bool) {
	return chat.GetLastUserMessage(cc.conversation)
}

func (cc *ChatController) HasSystemMessage() bool {
	return chat.HasSystemMessage(cc.conversation)
}

func (cc *ChatController) GetModel() string {
	return cc.conversation.Model
}

func (cc *ChatController) SetModel(model string) {
	cc.conversation = chat.WithModel(cc.conversation, model)
}

func (cc *ChatController) Reset() {
	systemPrompt := ""
	if chat.HasSystemMessage(cc.conversation) {
		messages := chat.GetMessagesByRole(cc.conversation, chat.RoleSystem)
		if len(messages) > 0 {
			systemPrompt = messages[0].Content
		}
	}
	
	cc.conversation = chat.NewConversationWithSystem(cc.conversation.Model, systemPrompt)
}