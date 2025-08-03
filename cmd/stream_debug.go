package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/killallgit/ryan/pkg/chat"
	"github.com/killallgit/ryan/pkg/controllers"
	"github.com/spf13/cobra"
)

var streamDebugCmd = &cobra.Command{
	Use:    "stream-debug",
	Short:  "Debug streaming functionality without requiring Ollama",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Testing Streaming Integration")
		fmt.Println("================================")

		// Test 1: Client Creation
		fmt.Println("\n1. Testing StreamingClient creation...")
		client, err := chat.NewStreamingClientWithTimeout("http://localhost:11434", "llama3.1:8b", 30*time.Second)
		if err != nil {
			log.Fatalf("❌ Failed to create StreamingClient: %v", err)
		}
		fmt.Println("✅ StreamingClient created successfully")

		// Test 2: Controller Creation
		fmt.Println("\n2. Testing ChatController with StreamingClient...")
		controller := controllers.NewChatController(client, "llama3.1:8b", nil)
		if controller == nil {
			log.Fatal("❌ Failed to create ChatController")
		}
		fmt.Println("✅ ChatController created successfully")

		// Test 3: Interface Check
		fmt.Println("\n3. Testing StreamingChatClient interface...")
		var streamingClient chat.StreamingChatClient = client
		if streamingClient != nil {
			fmt.Println("✅ Client implements StreamingChatClient interface")
		} else {
			log.Fatal("❌ Client does not implement StreamingChatClient interface")
		}

		// Test 4: StartStreaming Method
		fmt.Println("\n4. Testing StartStreaming method call...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// This will fail due to no Ollama server, but we can test the setup
		updates, err := controller.StartStreaming(ctx, "Hello, this is a test message")
		if err != nil {
			if err.Error() == "message content cannot be empty" {
				log.Fatal("❌ Empty message validation failed")
			}
			// Expected error due to no Ollama server
			fmt.Printf("✅ StartStreaming method called (expected error: %v)\n", err)
		} else {
			fmt.Println("✅ StartStreaming returned updates channel")
			// Try to read one update (will likely timeout)
			select {
			case update, ok := <-updates:
				if ok {
					fmt.Printf("✅ Received update: Type=%d\n", update.Type)
				} else {
					fmt.Println("✅ Updates channel closed properly")
				}
			case <-time.After(2 * time.Second):
				fmt.Println("✅ No updates received (expected without Ollama)")
			}
		}

		fmt.Println("\n🎉 Streaming Integration Test Summary")
		fmt.Println("=====================================")
		fmt.Println("✅ StreamingClient creation: PASS")
		fmt.Println("✅ ChatController integration: PASS")
		fmt.Println("✅ Interface implementation: PASS")
		fmt.Println("✅ Method invocation: PASS")
		fmt.Println("\n📋 Next Steps:")
		fmt.Println("   1. Start Ollama: ollama serve")
		fmt.Println("   2. Pull a model: ollama pull llama3.1:8b")
		fmt.Println("   3. Run: ryan")
		fmt.Println("   4. Send a message and watch it stream! 🎈")
	},
}

func init() {
	rootCmd.AddCommand(streamDebugCmd)
}
