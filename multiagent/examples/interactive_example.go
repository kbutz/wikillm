// Interactive Example for WikiLLM Multi-Agent System
//
// To run this example:
//
//	cd multiagent/examples
//	go run interactive_example.go
//
// This example uses LMStudio integration for local LLM processing and includes
// all personal assistant specialist agents.
//
// Then type your messages and press Enter to interact with the agents.
// Type 'exit' to quit the application.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kbutz/wikillm/multiagent/llmprovider"
	"github.com/kbutz/wikillm/multiagent/service"
)



func printWelcomeMessage() {
	fmt.Println("\n🤖 ===============================================")
	fmt.Println("   WikiLLM Personal Assistant Multi-Agent System")
	fmt.Println("   ===============================================")
	fmt.Println()
	fmt.Println("💬 Type your messages and press Enter to interact with the agents.")
	fmt.Println()
	fmt.Println("🛠️  Special commands:")
	fmt.Println("   • 'agents' - List all available agents")
	fmt.Println("   • 'health' - Show system health status")
	fmt.Println("   • 'clear-memory' - Clear conversation history")
	fmt.Println("   • 'debug-handlers' - Show active response handlers")
	fmt.Println("   • 'exit' - Quit the application")
	fmt.Println()
	fmt.Println("===============================================\n")
}

func main() {
	// Create memory directory within examples folder for easy access
	examplesDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	baseDir := filepath.Join(examplesDir, "wikillm_memory")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	log.Printf("Using memory directory: %s", baseDir)

	// Test LMStudio connectivity first
	log.Println("🔌 Testing LMStudio connection...")
	llmProvider := llmprovider.NewLMStudioProvider("http://localhost:1234/v1",
		llmprovider.WithTemperature(0.7),
		llmprovider.WithMaxTokens(2048),
		llmprovider.WithDebug(false), // Reduced debug output for cleaner interaction
	)

	// Test a simple query to ensure LMStudio is working
	log.Println("⏳ First request may take longer if model is loading...")
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	testResponse, err := llmProvider.Query(ctx, "Say hello in one word.")
	cancel()

	if err != nil {
		log.Printf("❌ LMStudio connection test failed: %v", err)
		log.Println("Please ensure:")
		log.Println("1. LMStudio is running")
		log.Println("2. A model is loaded")
		log.Println("3. The server is accessible at http://localhost:1234")
		log.Fatalf("Cannot proceed without LMStudio connection")
	}

	log.Printf("✅ LMStudio connection successful! Test response: %s", testResponse)

	// Create the multi-agent service with all specialist agents
	log.Println("🏗️  Creating personal assistant service with all specialist agents...")
	svc, err := service.NewMultiAgentService(service.ServiceConfig{
		BaseDir:     baseDir,
		LLMProvider: llmProvider,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-agent service: %v", err)
	}

	// Start the service
	ctx = context.Background()
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Defer cleanup
	defer func() {
		log.Println("🔄 Shutting down personal assistant...")
		if err := svc.Stop(ctx); err != nil {
			log.Printf("Warning: Failed to stop service cleanly: %v", err)
		}
	}()

	// Print welcome message
	printWelcomeMessage()

	// Generate a unique user ID
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create a scanner for user input
	scanner := bufio.NewScanner(os.Stdin)

	// Main interaction loop
	for {
		fmt.Print("🤖 > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Skip empty input
		if input == "" {
			continue
		}

		// Check for special commands
		switch strings.ToLower(input) {
		case "exit":
			fmt.Println("👋 Goodbye! Thanks for using the Personal Assistant!")
			return

		case "agents":
			fmt.Println("\n📋 Available Agents:")
			agentInfos := svc.ListAgents()
			for _, info := range agentInfos {
				fmt.Printf("   🤖 %s (%s)\n", info.Name, info.ID)
				fmt.Printf("      📝 %s\n", info.Description)
				fmt.Printf("      📊 Status: %s\n", info.Status)
				fmt.Printf("      🛠️  Capabilities: %s\n", strings.Join(info.Capabilities, ", "))
				fmt.Println()
			}
			continue

		case "health":
			fmt.Println("\n📊 System Health Status:")
			health := svc.GetSystemHealth()
			fmt.Printf("   🟢 Status: %s\n", health.Status)
			fmt.Printf("   🤖 Active Agents: %d/%d\n", health.ActiveAgents, health.TotalAgents)
			fmt.Printf("   📨 Messages Processed: %d\n", health.MessagesProcessed)
			fmt.Printf("   📊 Message Queue Size: %d\n", health.MessageQueueSize)
			fmt.Printf("   📈 Events Processed: %d\n", health.EventsProcessed)
			fmt.Printf("   📊 Event Queue Size: %d\n", health.EventQueueSize)
			fmt.Printf("   ⏱️  Uptime: %v\n", health.Uptime)
			fmt.Println()
			continue

		case "clear-memory":
			fmt.Println("\n🧹 Clearing conversation memory...")
			// This is just a simple way to reset the conversation
			userID = fmt.Sprintf("user_%d", time.Now().UnixNano())
			fmt.Println("✅ Memory cleared! Starting fresh conversation.")
			continue

		case "debug-handlers":
			fmt.Println("\n🔍 Active Response Handlers:")
			orch := svc.GetOrchestrator()
			if debugOrch, ok := orch.(interface {
				GetUserResponseHandlerCount() int
				GetUserResponseHandlerKeys() []string
			}); ok {
				count := debugOrch.GetUserResponseHandlerCount()
				keys := debugOrch.GetUserResponseHandlerKeys()
				fmt.Printf("   Total handlers: %d\n", count)
				for i, key := range keys {
					fmt.Printf("   %d. %s\n", i+1, key)
				}
			} else {
				fmt.Println("   Orchestrator doesn't support handler debugging")
			}
			fmt.Println()
			continue
		}

		// Process regular user input
		fmt.Println("\n⏳ Processing your request...")
		startTime := time.Now()

		// Send the message to the service
		response, err := svc.ProcessUserMessage(ctx, userID, input)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}

		// Calculate and display processing time
		elapsed := time.Since(startTime)
		fmt.Printf("\n%s\n\n", response)
		fmt.Printf("(Processed in %v)\n\n", elapsed.Round(time.Millisecond))
	}
}
