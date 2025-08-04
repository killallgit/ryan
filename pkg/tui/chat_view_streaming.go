package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
)

// Streaming and message processing methods for ChatView
// This file contains all streaming-related logic including content buffering and thinking detection

// detectThinkingStart checks if content begins with <think> or <thinking> tags
func (cv *ChatView) detectThinkingStart(content string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(content))
	return strings.HasPrefix(trimmed, "<think>") || strings.HasPrefix(trimmed, "<thinking>")
}

// detectContentTypeFromBuffer analyzes the buffer to determine content type
// Returns true if content type has been determined, false if more buffering needed
func (cv *ChatView) detectContentTypeFromBuffer() bool {
	const minBufferSize = 10 // Need at least 10 chars to detect "<thinking>"

	if cv.bufferSize < minBufferSize && cv.bufferSize < len(cv.streamingContent) {
		// Still need more characters for reliable detection
		return false
	}

	// Check if it starts with thinking tags
	if cv.detectThinkingStart(cv.contentBuffer) {
		cv.isStreamingThinking = true
		// Extract content after the opening tag
		thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
		cv.thinkingContent = strings.TrimSpace(thinkStartRegex.ReplaceAllString(cv.contentBuffer, ""))
	} else {
		cv.isStreamingThinking = false
		// Not thinking content, treat as regular response
		cv.responseContent = cv.contentBuffer
	}

	cv.contentTypeDetected = true
	return true
}

// processStreamingContent processes the full streaming content and separates thinking from response
func (cv *ChatView) processStreamingContent() {
	fullContent := cv.streamingContent

	// If we haven't detected thinking yet, check for thinking tags at the start
	if !cv.isStreamingThinking && len(cv.thinkingContent) == 0 && len(cv.responseContent) == 0 {
		if cv.detectThinkingStart(fullContent) {
			cv.isStreamingThinking = true
		}
	}

	// Process the content based on current state
	if cv.isStreamingThinking {
		// Check if thinking block ends
		thinkEndRegex := regexp.MustCompile(`(?i)</think(?:ing)?>`)
		if thinkEndRegex.MatchString(fullContent) {
			// Split at the end of thinking block
			parts := thinkEndRegex.Split(fullContent, 2)
			if len(parts) == 2 {
				// Extract thinking content (remove opening tags)
				thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
				thinkingRaw := thinkStartRegex.ReplaceAllString(parts[0], "")
				cv.thinkingContent = strings.TrimSpace(thinkingRaw)

				// Start response content
				cv.responseContent = strings.TrimSpace(parts[1])
				cv.isStreamingThinking = false
			}
		} else {
			// Still in thinking block, accumulate thinking content
			thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
			cv.thinkingContent = strings.TrimSpace(thinkStartRegex.ReplaceAllString(fullContent, ""))
		}
	} else {
		// In response mode or no thinking detected
		if len(cv.thinkingContent) == 0 {
			// No thinking content detected, treat as regular response
			cv.responseContent = fullContent
		} else {
			// Already have thinking content, extract response part from full content
			thinkEndRegex := regexp.MustCompile(`(?i)</think(?:ing)?>`)
			if thinkEndRegex.MatchString(fullContent) {
				parts := thinkEndRegex.Split(fullContent, 2)
				if len(parts) == 2 {
					cv.responseContent = strings.TrimSpace(parts[1])
				}
			}
		}
	}
}

// createStreamingMessage creates a properly formatted message for streaming display
func (cv *ChatView) createStreamingMessage() chat.Message {
	var content string

	// If we haven't detected content type yet, don't show anything
	if !cv.contentTypeDetected && cv.isStreaming {
		return chat.Message{
			Role:    chat.RoleAssistant,
			Content: "", // Show nothing while buffering
		}
	}

	if cv.thinkingContent != "" {
		// Format thinking content with proper tags so ParseThinkingBlock can style it correctly
		thinkingWithTags := "<think>" + cv.thinkingContent

		if cv.isStreamingThinking {
			// Still streaming thinking content, add cursor before closing tag
			content = thinkingWithTags + " ▌"
		} else {
			// Thinking complete, close tag and add response if any
			content = thinkingWithTags + "</think>"

			if cv.responseContent != "" {
				// Add response content with cursor if still streaming
				responseContent := cv.responseContent
				if cv.isStreaming {
					responseContent += " ▌"
				}
				content += "\n\n" + responseContent
			}
		}
	} else if cv.responseContent != "" {
		// Only response content (no thinking detected)
		content = cv.responseContent
		if cv.isStreaming {
			content += " ▌"
		}
	} else if cv.isStreamingThinking {
		// Currently streaming thinking content from the beginning
		thinkingRaw := cv.streamingContent
		// Remove any <think> tags that might be in the raw content
		thinkStartRegex := regexp.MustCompile(`(?i)<think(?:ing)?>`)
		thinkingRaw = thinkStartRegex.ReplaceAllString(thinkingRaw, "")
		content = "<think>" + strings.TrimSpace(thinkingRaw) + " ▌"
	} else {
		// Regular content without thinking
		content = cv.streamingContent
		if cv.isStreaming {
			content += " ▌"
		}
	}

	return chat.Message{
		Role:    chat.RoleAssistant,
		Content: content,
	}
}

func (cv *ChatView) HandleStreamStart(streamID, model string) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling stream start in chat view", "stream_id", streamID, "model", model)

	// Initialize streaming state
	cv.isStreaming = true
	cv.currentStreamID = streamID
	cv.streamingContent = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Initialize buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Update status to show streaming
	cv.status = cv.status.WithStatus("Streaming response...")

	// Initialize status row with current token count and streaming spinner
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	totalTokens := promptTokens + responseTokens
	cv.statusRow = cv.statusRow.WithSpinner(true, "Streaming...").WithTokens(totalTokens)

	// Show alert spinner
	cv.alert = cv.alert.WithSpinner(true, "Streaming...")
}

func (cv *ChatView) UpdateStreamingContent(streamID, content string, isComplete bool) {
	log := logger.WithComponent("chat_view")
	log.Debug("Updating streaming content in chat view",
		"stream_id", streamID,
		"content_length", len(content),
		"is_complete", isComplete,
		"content_type_detected", cv.contentTypeDetected,
		"buffer_size", cv.bufferSize)

	// Update basic streaming state
	cv.currentStreamID = streamID
	cv.streamingContent = content
	cv.isStreaming = !isComplete

	// Early detection buffering logic
	if !cv.contentTypeDetected && !isComplete {
		// Still buffering to detect content type
		cv.contentBuffer = content
		cv.bufferSize = len(content)

		// Try to detect content type from buffer
		if cv.detectContentTypeFromBuffer() {
			log.Debug("Content type detected",
				"is_thinking", cv.isStreamingThinking,
				"thinking_content", cv.thinkingContent,
				"response_content", cv.responseContent)
		} else {
			// Still need more content for detection, don't display anything yet
			log.Debug("Still buffering for content type detection", "buffer_size", cv.bufferSize)
			return
		}
	}

	// Content type already detected or stream is complete, process normally
	if cv.contentTypeDetected || isComplete {
		cv.processStreamingContent()

		// Update the message display to show streaming content with proper formatting
		cv.updateMessagesWithStreamingThinking()
	}

	if !isComplete {
		// Update spinner text based on current mode
		spinnerText := "Streaming..."
		if cv.isStreamingThinking {
			spinnerText = "Thinking..."
		}
		cv.alert = cv.alert.WithSpinner(true, spinnerText).NextSpinnerFrame()
		cv.statusRow = cv.statusRow.WithSpinner(true, spinnerText).NextSpinnerFrame()
	} else {
		// Clear streaming state when complete
		cv.isStreaming = false
		cv.streamingContent = ""
		cv.currentStreamID = ""
		cv.isStreamingThinking = false
		cv.thinkingContent = ""
		cv.responseContent = ""

		// Clear buffering state
		cv.contentBuffer = ""
		cv.contentTypeDetected = false
		cv.bufferSize = 0

		cv.alert = cv.alert.WithSpinner(false, "")
		cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count
	}
}

func (cv *ChatView) HandleStreamComplete(streamID string, finalMessage chat.Message, totalChunks int, duration time.Duration) {
	log := logger.WithComponent("chat_view")
	log.Debug("Handling stream complete in chat view",
		"stream_id", streamID,
		"total_chunks", totalChunks,
		"duration", duration.String(),
		"final_message_length", len(finalMessage.Content))

	// DEBUG: Log the exact final message content
	log.Debug("Final message details",
		"role", finalMessage.Role,
		"content_length", len(finalMessage.Content),
		"content_preview", func() string {
			if len(finalMessage.Content) > 200 {
				return finalMessage.Content[:200] + "..."
			}
			return finalMessage.Content
		}(),
		"has_thinking_tags", strings.Contains(finalMessage.Content, "<think"),
		"has_response_after_thinking", strings.Contains(finalMessage.Content, "</think>"))

	// Clear streaming state
	cv.isStreaming = false
	cv.streamingContent = ""
	cv.currentStreamID = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Clear buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")
	cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count

	// Update status
	cv.status = cv.status.WithStatus("Ready")

	// Update token information
	promptTokens, responseTokens := cv.controller.GetTokenUsage()
	cv.status = cv.status.WithTokens(promptTokens, responseTokens)
	// Note: Token counts are currently 0 due to LangChain Go not exposing usage info

	// Update messages display with final content (no streaming)
	cv.updateMessages()
	cv.scrollToBottom()
}

func (cv *ChatView) HandleStreamError(streamID string, err error) {
	log := logger.WithComponent("chat_view")
	log.Error("Handling stream error in chat view", "stream_id", streamID, "error", err)

	// Clear streaming state
	cv.isStreaming = false
	cv.streamingContent = ""
	cv.currentStreamID = ""
	cv.isStreamingThinking = false
	cv.thinkingContent = ""
	cv.responseContent = ""

	// Clear buffering state
	cv.contentBuffer = ""
	cv.contentTypeDetected = false
	cv.bufferSize = 0

	// Hide spinner
	cv.alert = cv.alert.WithSpinner(false, "")
	cv.statusRow = cv.statusRow.ClearSpinnerOnly() // Preserve token count

	// Update status with error
	cv.status = cv.status.WithStatus("Streaming failed: " + err.Error())

	// Update messages display to show error
	cv.updateMessages()
}

func (cv *ChatView) UpdateStreamProgress(streamID string, contentLength, chunkCount int, duration time.Duration) {
	log := logger.WithComponent("chat_view")
	log.Debug("Updating stream progress in chat view",
		"stream_id", streamID,
		"content_length", contentLength,
		"chunk_count", chunkCount,
		"duration", duration.String())

	// Update spinner with progress info for long streams
	if duration > 3*time.Second {
		progressText := fmt.Sprintf("Streaming... %d chars", contentLength)
		cv.alert = cv.alert.WithSpinner(true, progressText).NextSpinnerFrame()
		cv.statusRow = cv.statusRow.WithSpinner(true, progressText).WithDuration(duration).NextSpinnerFrame()
	}
}

func (cv *ChatView) updateMessagesWithStreamingThinking() {
	history := cv.controller.GetHistory()
	// Filter out system messages - they should not be displayed to the user
	var filteredHistory []chat.Message
	for _, msg := range history {
		if msg.Role != chat.RoleSystem {
			filteredHistory = append(filteredHistory, msg)
		}
	}
	history = filteredHistory

	// If we're streaming and have detected content type, show streaming content
	if cv.isStreaming && cv.contentTypeDetected {
		// Create a copy of history to avoid modifying the original
		messagesWithStreaming := make([]chat.Message, len(history))
		copy(messagesWithStreaming, history)

		// Create properly formatted streaming message with thinking detection
		streamingMessage := cv.createStreamingMessage()

		// Only add the streaming message if it has content
		if streamingMessage.Content != "" {
			messagesWithStreaming = append(messagesWithStreaming, streamingMessage)
		}

		cv.messages = cv.messages.WithMessages(messagesWithStreaming)

		// Auto-scroll to bottom during streaming
		cv.scrollToBottom()
	} else {
		// No streaming or content type not detected yet, show regular messages
		cv.messages = cv.messages.WithMessages(history)
	}
}
