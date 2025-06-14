package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kbutz/wikillm/multiagent"
	"github.com/kbutz/wikillm/multiagent/service"
)

// SimpleLLMProvider is a basic implementation of the LLMProvider interface for demonstration
type SimpleLLMProvider struct{}

func (p *SimpleLLMProvider) Name() string {
	return "simple_llm"
}

func (p *SimpleLLMProvider) Query(ctx context.Context, prompt string) (string, error) {
	// In a real implementation, this would call an actual LLM API
	// For this example, we'll just return a simple response
	return fmt.Sprintf("Response to: %s\n\nI'm a simple LLM provider for demonstration purposes. In a real implementation, this would be a response from an actual language model.", prompt), nil
}

func (p *SimpleLLMProvider) QueryWithTools(ctx context.Context, prompt string, tools []multiagent.Tool) (string, error) {
	// In a real implementation, this would call an actual LLM API with tool calling capabilities
	// For this example, we'll just return a simple response
	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Name()
	}
	
	return fmt.Sprintf("Response to: %s\n\nI'm a simple LLM provider with access to these tools: %v. In a real implementation, I would use these tools to help answer your question.", prompt, toolNames), nil
}

func main() {
	// Create a base directory for the service
	baseDir := filepath.Join(os.TempDir(), "wikillm_multiagent_example")
	os.RemoveAll(baseDir) // Clean up any previous run
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}
	
	// Create a simple LLM provider
	llmProvider := &SimpleLLMProvider{}
	
	// Create the multi-agent service
	svc, err := service.NewMultiAgentService(service.ServiceConfig{
		BaseDir:     baseDir,
		LLMProvider: llmProvider,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-agent service: %v", err)
	}
	
	// Start the service
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
	
	// Process some example user messages
	processExampleMessages(ctx, svc)
	
	// Stop the service
	if err := svc.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}
	
	log.Println("Example completed successfully")
}

func processExampleMessages(ctx context.Context, svc *service.MultiAgentService) {
	// Example 1: Simple greeting
	processMessage(ctx, svc, "user123", "Hello! How can you help me today?")
	
	// Example 2: Task creation request
	processMessage(ctx, svc, "user123", "I need to remember to buy groceries tomorrow")
	
	// Example 3: Complex question that requires research
	processMessage(ctx, svc, "user123", "Can you explain how neural networks work?")
	
	// Example 4: Request for code
	processMessage(ctx, svc, "user123", "Write a simple function in Go to calculate fibonacci numbers")
	
	// Example 5: Follow-up question
	processMessage(ctx, svc, "user123", "Can you explain the time complexity of that algorithm?")
}

func processMessage(ctx context.Context, svc *service.MultiAgentService, userID, message string) {
	fmt.Printf("\n--- User: %s ---\n", message)
	
	// Process the message
	response, err := svc.ProcessUserMessage(ctx, userID, message)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Print the response
	fmt.Printf("--- Assistant: ---\n%s\n", response)
	
	// Add a small delay between messages to allow for processing
	time.Sleep(500 * time.Millisecond)
}