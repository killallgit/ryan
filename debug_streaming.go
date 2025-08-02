package main

import (
	"fmt"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/viper"
)

func main() {
	// Initialize viper and logger
	viper.Set("logging.file", "./.ryan/debug.log")
	viper.Set("logging.preserve", false)
	viper.Set("logging.level", "debug")
	
	err := logger.InitLogger()
	if err != nil {
		panic(err)
	}

	fmt.Println("Testing streaming flow with thinking content...")
	fmt.Println("Check ./.ryan/debug.log for detailed debug output")
	fmt.Println()

	// Create a mock chat controller (no real client needed for this test)
	controller := controllers.NewChatController(nil, "test-model", nil)

	// Add a user message to start the conversation
	controller.AddUserMessage("Tell me about AI")

	// Simulate what happens during streaming completion
	// This simulates the final message from the accumulator
	finalMessageContent := `<think>The user is asking about AI. I should provide a comprehensive but concise overview covering what AI is, its applications, and current state.</think>

Artificial Intelligence (AI) refers to computer systems that can perform tasks typically requiring human intelligence. This includes learning, reasoning, problem-solving, and understanding language.

Key applications include:
- Machine learning and data analysis
- Natural language processing
- Computer vision
- Robotics and automation

AI has made significant advances recently, particularly in large language models like GPT and Claude, which can engage in human-like conversations and assist with various tasks.`

	// Create the final message as it would come from the accumulator
	finalMessage := chat.Message{
		Role:      chat.RoleAssistant,
		Content:   finalMessageContent,
		Timestamp: time.Now(),
	}

	fmt.Printf("=== Simulating Stream Completion ===\n")
	fmt.Printf("Final message content length: %d\n", len(finalMessage.Content))
	fmt.Printf("Contains thinking tags: %t\n", len(finalMessage.Content) > 0)
	fmt.Println()

	// Test 1: What does the controller store?
	fmt.Printf("=== Before Adding Final Message ===\n")
	historyBefore := controller.GetHistory()
	fmt.Printf("Messages in conversation: %d\n", len(historyBefore))
	for i, msg := range historyBefore {
		fmt.Printf("  [%d] %s: %q\n", i, msg.Role, msg.Content)
	}
	fmt.Println()

	// Manually add the final message as the controller would during streaming
	// We need to access the internal conversation, but since it's private, 
	// let's simulate what happens by testing the components directly
	fmt.Printf("=== Simulating Controller Message Addition ===\n")
	fmt.Printf("In real streaming, controller would do:\n")
	fmt.Printf("  cc.conversation = chat.AddMessage(cc.conversation, assistantMessage)\n")
	fmt.Println()

	// Skip the actual addition since we can't access private fields
	// Focus on testing the parsing which is where the issue likely is

	// Test 2: What does ParseThinkingBlock do with this content?
	fmt.Printf("=== Testing ParseThinkingBlock on Final Message ===\n")
	parsed := tui.ParseThinkingBlock(finalMessage.Content)
	fmt.Printf("HasThinking: %t\n", parsed.HasThinking)
	fmt.Printf("ThinkingBlock length: %d\n", len(parsed.ThinkingBlock))
	fmt.Printf("ResponseContent length: %d\n", len(parsed.ResponseContent))
	fmt.Printf("ResponseContent: %q\n", parsed.ResponseContent)
	fmt.Println()

	// Test 3: What would the UI render?
	fmt.Printf("=== What Would UI Render? ===\n")
	if parsed.HasThinking {
		fmt.Printf("Would render thinking: %q\n", parsed.ThinkingBlock[:min(50, len(parsed.ThinkingBlock))]+"...")
		if parsed.ResponseContent != "" {
			fmt.Printf("Would render response: %q\n", parsed.ResponseContent)
		} else {
			fmt.Printf("⚠️  NO RESPONSE CONTENT TO RENDER!\n")
		}
	} else {
		fmt.Printf("Would render full content: %q\n", finalMessage.Content[:min(100, len(finalMessage.Content))]+"...")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}