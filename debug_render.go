package main

import (
	"fmt"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/logger"
	"github.com/killallgit/ryan/pkg/tui"
	"github.com/spf13/viper"
)

func main() {
	// Initialize viper and logger
	viper.Set("logging.file", "./.ryan/debug.log")
	viper.Set("logging.preserve", false)
	viper.Set("logging.level", "debug")
	viper.Set("show_thinking", true)
	
	err := logger.InitLogger()
	if err != nil {
		panic(err)
	}

	fmt.Println("Testing UI rendering logic...")
	fmt.Println("Check ./.ryan/debug.log for detailed debug output")
	fmt.Println()

	// Test message with thinking content
	finalMessageContent := `<think>The user is asking about AI. I should provide a comprehensive but concise overview covering what AI is, its applications, and current state.</think>

Artificial Intelligence (AI) refers to computer systems that can perform tasks typically requiring human intelligence. This includes learning, reasoning, problem-solving, and understanding language.

Key applications include:
- Machine learning and data analysis
- Natural language processing
- Computer vision
- Robotics and automation

AI has made significant advances recently, particularly in large language models like GPT and Claude, which can engage in human-like conversations and assist with various tasks.`

	message := chat.Message{
		Role:    chat.RoleAssistant,
		Content: finalMessageContent,
	}

	// Test the rendering logic step by step
	fmt.Printf("=== Step 1: Parse Thinking Block ===\n")
	parsed := tui.ParseThinkingBlock(message.Content)
	fmt.Printf("HasThinking: %t\n", parsed.HasThinking)
	fmt.Printf("ThinkingBlock length: %d\n", len(parsed.ThinkingBlock))
	fmt.Printf("ResponseContent length: %d\n", len(parsed.ResponseContent))
	fmt.Printf("ResponseContent empty: %t\n", parsed.ResponseContent == "")
	fmt.Println()

	// Test the content selection logic from render.go
	fmt.Printf("=== Step 2: Content Selection Logic ===\n")
	showThinking := viper.GetBool("show_thinking")
	fmt.Printf("show_thinking config: %t\n", showThinking)
	
	var contentToRender string
	if parsed.HasThinking && showThinking {
		contentToRender = parsed.ResponseContent
		fmt.Printf("Using ResponseContent (has thinking + show_thinking)\n")
	} else {
		contentToRender = message.Content
		fmt.Printf("Using full message content (no thinking or show_thinking=false)\n")
	}
	
	fmt.Printf("contentToRender length: %d\n", len(contentToRender))
	fmt.Printf("contentToRender empty: %t\n", contentToRender == "")
	fmt.Printf("contentToRender preview: %q\n", contentToRender[:min(100, len(contentToRender))]+"...")
	fmt.Println()

	// Test what would be rendered
	fmt.Printf("=== Step 3: What Would Be Rendered ===\n")
	if contentToRender != "" {
		lines := tui.WrapText(contentToRender, 80) // Simulate 80-char width
		fmt.Printf("Would render %d lines:\n", len(lines))
		for i, line := range lines {
			if i < 5 { // Show first 5 lines
				fmt.Printf("  Line %d: %q\n", i+1, line)
			} else if i == 5 {
				fmt.Printf("  ... (%d more lines)\n", len(lines)-5)
				break
			}
		}
	} else {
		fmt.Printf("⚠️  NOTHING WOULD BE RENDERED - contentToRender is empty!\n")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}