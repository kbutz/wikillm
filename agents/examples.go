package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kbutz/wikillm/agents/models"
	"github.com/kbutz/wikillm/agents/tools"
)

// Example demonstrates the enhanced memory system
func Example() {
	// Create a mock LLM model for demonstration
	model, err := models.New("default", "lmstudio", false)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Initialize enhanced memory tool
	memoryTool := tools.NewEnhancedMemoryTool("./example_memory")
	
	// Create memory-enabled agent
	memAgent := NewMemoryEnabledAgent(
		model,
		[]models.Tool{memoryTool},
		memoryTool,
	)
	
	ctx := context.Background()
	
	// Example 1: Store project information
	fmt.Println("=== Example 1: Storing Project Information ===")
	response1, _ := memAgent.ProcessQuery(ctx, "I'm working on a project called WikiLLM using Go and it's about building AI agents")
	fmt.Printf("Response: %s\n\n", response1)
	
	// Example 2: Store user preference
	fmt.Println("=== Example 2: Storing User Preference ===")
	response2, _ := memAgent.ProcessQuery(ctx, "I prefer detailed technical documentation with examples")
	fmt.Printf("Response: %s\n\n", response2)
	
	// Example 3: Query with context
	fmt.Println("=== Example 3: Query Using Stored Context ===")
	response3, _ := memAgent.ProcessQuery(ctx, "What project am I working on?")
	fmt.Printf("Response: %s\n\n", response3)
	
	// Example 4: Store a task
	fmt.Println("=== Example 4: Storing a Task ===")
	response4, _ := memAgent.ProcessQuery(ctx, "I need to implement the memory search function by 2024-06-20")
	fmt.Printf("Response: %s\n\n", response4)
	
	// Example 5: Get context summary
	fmt.Println("=== Example 5: Context Summary ===")
	contextSummary, _ := memoryTool.Execute(ctx, `{"command": "context"}`)
	fmt.Printf("Current Memory Context:\n%s\n", contextSummary)
}

// ExampleMemoryOperations demonstrates direct memory tool operations
func ExampleMemoryOperations() {
	ctx := context.Background()
	memoryTool := tools.NewEnhancedMemoryTool("./example_memory")
	
	// Store a memory
	fmt.Println("=== Storing Memory ===")
	storeResult, err := memoryTool.Execute(ctx, `{
		"command": "store",
		"category": "projects",
		"content": "WikiLLM Agent Development - Phase 2",
		"tags": ["development", "ai", "agents"],
		"metadata": {"status": "active", "language": "Go"}
	}`)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Println(storeResult)
	}
	
	// Retrieve memories
	fmt.Println("\n=== Retrieving Project Memories ===")
	retrieveResult, _ := memoryTool.Execute(ctx, `{
		"command": "retrieve",
		"category": "projects",
		"limit": 5
	}`)
	fmt.Println(retrieveResult)
	
	// Search memories
	fmt.Println("\n=== Searching Memories ===")
	searchResult, _ := memoryTool.Execute(ctx, `{
		"command": "search",
		"content": "WikiLLM"
	}`)
	fmt.Println(searchResult)
	
	// Auto-store from content
	fmt.Println("\n=== Auto-Store Example ===")
	autoStoreResult, _ := memoryTool.Execute(ctx, `{
		"command": "auto_store",
		"content": "I decided to use the enhanced memory system for better context retention"
	}`)
	fmt.Println(autoStoreResult)
}
