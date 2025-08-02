package main

import (
	"fmt"

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

	// Test cases with thinking content
	testCases := []string{
		// Case 1: Thinking with response
		`<think>This is my thinking process about the user's question. I need to consider multiple factors.</think>

This is my response to the user after thinking about it.`,

		// Case 2: Thinking only
		`<think>This is just thinking with no response.</think>`,

		// Case 3: Response only
		`This is just a response with no thinking.`,

		// Case 4: Empty thinking
		`<think></think>

This is a response after empty thinking.`,
	}

	fmt.Println("Testing ParseThinkingBlock with debug logging...")
	fmt.Println("Check ./.ryan/debug.log for detailed debug output")
	fmt.Println()

	for i, testCase := range testCases {
		fmt.Printf("=== Test Case %d ===\n", i+1)
		fmt.Printf("Input: %q\n", testCase)
		
		result := tui.ParseThinkingBlock(testCase)
		
		fmt.Printf("HasThinking: %t\n", result.HasThinking)
		fmt.Printf("ThinkingBlock: %q\n", result.ThinkingBlock)
		fmt.Printf("ResponseContent: %q\n", result.ResponseContent)
		fmt.Println()
	}
}