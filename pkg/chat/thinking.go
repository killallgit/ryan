package chat

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ParsedMessage represents a message content that has been parsed for thinking blocks
type ParsedMessage struct {
	ThinkingContent string
	ResponseContent string
	HasThinking     bool
}

// ParseMessageThinking parses a message content to separate <think> blocks from regular content
func ParseMessageThinking(content string) ParsedMessage {
	thinkRegex := regexp.MustCompile(`(?ims)<think(?:ing)?>(.*?)</think(?:ing)?>`)
	matches := thinkRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return ParsedMessage{
			ThinkingContent: "",
			ResponseContent: strings.TrimSpace(content),
			HasThinking:     false,
		}
	}

	// Extract thinking content (combine all THINK blocks if multiple)
	var thinkingParts []string
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			thinkingParts = append(thinkingParts, strings.TrimSpace(m[1]))
		}
	}

	// Remove THINK blocks from content to get response
	response := strings.TrimSpace(thinkRegex.ReplaceAllString(content, ""))

	return ParsedMessage{
		ThinkingContent: strings.Join(thinkingParts, "\n\n"),
		ResponseContent: response,
		HasThinking:     len(thinkingParts) > 0,
	}
}

// ExtractResponseContent extracts only the response content from a message, removing thinking blocks
func ExtractResponseContent(content string) string {
	parsed := ParseMessageThinking(content)
	return parsed.ResponseContent
}

// Enhanced parsing functions for the new message architecture

// ParseMessageWithThinking takes a message content and returns a Message with separated thinking
func ParseMessageWithThinking(content string, role string, showThinking bool) Message {
	parsed := ParseMessageThinking(content)

	msg := Message{
		Role:      role,
		Content:   parsed.ResponseContent,
		Timestamp: time.Now(),
		Metadata: &MessageMetadata{
			Source: MessageSourceFinal,
		},
	}

	if parsed.HasThinking {
		msg.Thinking = &ThinkingBlock{
			Content: parsed.ThinkingContent,
			Visible: showThinking,
		}
	}

	return msg
}

// ParseAssistantMessageWithThinking creates an assistant message with separated thinking
func ParseAssistantMessageWithThinking(content string, showThinking bool) Message {
	return ParseMessageWithThinking(content, RoleAssistant, showThinking)
}

// UpdateMessageWithThinking updates an existing message by parsing and separating thinking blocks
func UpdateMessageWithThinking(msg Message, showThinking bool) Message {
	if msg.Content == "" {
		return msg
	}

	parsed := ParseMessageThinking(msg.Content)

	updated := msg
	updated.Content = parsed.ResponseContent

	if parsed.HasThinking {
		updated.Thinking = &ThinkingBlock{
			Content: parsed.ThinkingContent,
			Visible: showThinking,
		}
	}

	return updated
}

// GetEffectiveContent returns the content to display based on thinking visibility
func GetEffectiveContent(msg Message) string {
	if msg.HasThinking() && msg.IsThinkingVisible() {
		// If thinking is visible, return both thinking and content
		if msg.Content != "" {
			return fmt.Sprintf("<think>%s</think>\n\n%s", msg.Thinking.Content, msg.Content)
		}
		return fmt.Sprintf("<think>%s</think>", msg.Thinking.Content)
	}
	return msg.Content
}
