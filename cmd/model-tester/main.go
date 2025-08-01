package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/killallgit/ryan/pkg/testing"
)

// PrimaryTestModels are the core models we want to test first
var PrimaryTestModels = []string{
	"llama3.1:8b",
	"qwen2.5-coder:1.5b",
	"qwen2.5:7b",
}

// SecondaryTestModels are additional models for extended testing
var SecondaryTestModels = []string{
	"llama3.2:3b",
	"mistral:7b",
	"qwen3:8b",
	"llama3.2:1b",
	"qwen2.5:3b",
	"mistral-nemo",
}

// AllRecommendedModels combines primary and secondary for comprehensive testing
var AllRecommendedModels = append(PrimaryTestModels, SecondaryTestModels...)

func main() {
	var (
		ollamaURL = flag.String("url", "http://localhost:11434", "Ollama server URL")
		modelList = flag.String("models", "primary", "Models to test: 'primary', 'secondary', 'all', or comma-separated list")
		verbose   = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Parse model selection
	var modelsToTest []string
	switch strings.ToLower(*modelList) {
	case "primary":
		modelsToTest = PrimaryTestModels
		fmt.Println("🎯 Testing PRIMARY models (core functionality)")
	case "secondary":
		modelsToTest = SecondaryTestModels
		fmt.Println("📋 Testing SECONDARY models (extended compatibility)")
	case "all":
		modelsToTest = AllRecommendedModels
		fmt.Println("🚀 Testing ALL recommended models (comprehensive)")
	default:
		// Custom comma-separated list
		modelsToTest = strings.Split(*modelList, ",")
		for i, model := range modelsToTest {
			modelsToTest[i] = strings.TrimSpace(model)
		}
		fmt.Printf("🔧 Testing CUSTOM model list: %v\n", modelsToTest)
	}

	if len(modelsToTest) == 0 {
		fmt.Println("❌ No models specified for testing")
		os.Exit(1)
	}

	// Create tester
	tester := testing.NewModelCompatibilityTester(*ollamaURL)

	fmt.Printf("🔗 Connecting to Ollama at: %s\n", *ollamaURL)
	fmt.Printf("📊 Testing %d models...\n\n", len(modelsToTest))

	// Run tests
	results := tester.TestMultipleModels(modelsToTest)

	// Print results
	tester.PrintResults(results)

	// Generate recommendations
	generateRecommendations(results)

	// Exit with error code if no models passed
	hasSuccess := false
	for _, result := range results {
		if result.ToolCallSupported && result.PassedTests > 0 {
			hasSuccess = true
			break
		}
	}

	if !hasSuccess {
		fmt.Println("\n❌ No models passed compatibility testing")
		os.Exit(1)
	}

	fmt.Println("\n✅ Model compatibility testing completed successfully")
}

func generateRecommendations(results []testing.ModelTestResult) {
	fmt.Printf("\n" + strings.Repeat("=", 80))
	fmt.Println("🎯 RECOMMENDATIONS")
	fmt.Println(strings.Repeat("=", 80))

	// Find best performing models
	var excellent, good, problematic []testing.ModelTestResult

	for _, result := range results {
		if !result.ToolCallSupported {
			problematic = append(problematic, result)
			continue
		}

		passRate := float64(result.PassedTests) / float64(result.TotalTests)
		if passRate >= 0.75 {
			excellent = append(excellent, result)
		} else if passRate >= 0.5 {
			good = append(good, result)
		} else {
			problematic = append(problematic, result)
		}
	}

	if len(excellent) > 0 {
		fmt.Println("\n🌟 EXCELLENT for production use:")
		for _, result := range excellent {
			passRate := float64(result.PassedTests) / float64(result.TotalTests) * 100
			fmt.Printf("   ✅ %s (%.0f%% pass rate, %v avg response)\n", 
				result.ModelName, passRate, result.AverageResponseTime.Round(100000000))
		}
	}

	if len(good) > 0 {
		fmt.Println("\n👍 GOOD for development/testing:")
		for _, result := range good {
			passRate := float64(result.PassedTests) / float64(result.TotalTests) * 100
			fmt.Printf("   ⚠️  %s (%.0f%% pass rate, %v avg response)\n", 
				result.ModelName, passRate, result.AverageResponseTime.Round(100000000))
		}
	}

	if len(problematic) > 0 {
		fmt.Println("\n⚠️  PROBLEMATIC models:")
		for _, result := range problematic {
			if result.ToolCallSupported {
				passRate := float64(result.PassedTests) / float64(result.TotalTests) * 100
				fmt.Printf("   ❌ %s (%.0f%% pass rate) - %s\n", 
					result.ModelName, passRate, strings.Join(result.Errors, "; "))
			} else {
				fmt.Printf("   ❌ %s (No tool support)\n", result.ModelName)
			}
		}
	}

	// Configuration recommendations
	fmt.Println("\n💡 CONFIGURATION RECOMMENDATIONS:")
	if len(excellent) > 0 {
		fastest := excellent[0]
		for _, result := range excellent {
			if result.AverageResponseTime < fastest.AverageResponseTime {
				fastest = result
			}
		}
		fmt.Printf("   • Default model: %s (best balance of accuracy and speed)\n", fastest.ModelName)
	}

	if len(excellent) > 1 {
		fmt.Println("   • Consider model switching based on task complexity")
		fmt.Println("   • Enable tool compatibility validation in UI")
	}

	if len(problematic) > 0 {
		fmt.Println("   • Add warnings for problematic models in model selection")
		fmt.Println("   • Consider fallback mechanisms for failed tool calls")
	}

	fmt.Println(strings.Repeat("=", 80))
}