package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kbutz/wikillm/agents/models"
	"github.com/kbutz/wikillm/agents/tools"
)

// Parse command line flags
func parseFlags() Config {
	modelName := flag.String("model", "default", "Name of the LLM model to use")
	modelProvider := flag.String("provider", "lmstudio", "Model provider to use (lmstudio or ollama)")
	port := flag.Int("port", 0, "HTTP server port (0 to disable)")
	todoFilePath := flag.String("todo-file", "todo.txt", "Path to the to-do list file")
	memoryFilePath := flag.String("memory-file", "memory.txt", "Path to the memory file")
	memoryDir := flag.String("memory-dir", "memory", "Directory for enhanced memory storage")
	useEnhancedMemory := flag.Bool("enhanced-memory", true, "Use enhanced memory system")
	debug := flag.Bool("debug", false, "Enable debug mode")

	flag.Parse()

	return Config{
		ModelName:         *modelName,
		ModelProvider:     *modelProvider,
		Port:              *port,
		TodoFilePath:      *todoFilePath,
		MemoryFilePath:    *memoryFilePath,
		MemoryDir:         *memoryDir,
		UseEnhancedMemory: *useEnhancedMemory,
		Debug:             *debug,
	}
}

// Config Configuration options for the application
type Config struct {
	ModelName         string // Name of the LLM model to use
	ModelProvider     string // Provider to use (lmstudio or ollama)
	Port              int    // HTTP server port
	TodoFilePath      string // Path to the to-do list file
	MemoryFilePath    string // Path to the memory file
	MemoryDir         string // Directory for enhanced memory storage
	UseEnhancedMemory bool   // Use enhanced memory system
	Debug             bool   // Enable debug mode
}

func main() {
	// Parse command line arguments
	config := parseFlags()

	model, err := models.New(config.ModelName, config.ModelProvider, config.Debug)
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
		return
	}

	// Initialize the tools
	todoTool := tools.NewTodoListTool(config.TodoFilePath)
	
	// Use enhanced memory tool if enabled
	if config.UseEnhancedMemory {
		enhancedMemoryTool := tools.NewEnhancedMemoryTool(config.MemoryDir)
		
		// Create memory-enabled model
		memoryModel, err := models.NewMemoryEnabledModel(config.ModelName, config.ModelProvider, config.Debug)
		if err != nil {
			log.Fatalf("Failed to initialize memory-enabled model: %v", err)
			return
		}
		
		// Create memory-enabled agent
		memAgent := NewMemoryEnabledAgent(
			memoryModel,
			[]models.Tool{todoTool, enhancedMemoryTool},
			enhancedMemoryTool,
		)
		
		// Initialize context
		if err := memAgent.InitializeContext(context.Background()); err != nil {
			log.Printf("Warning: Failed to initialize context: %v", err)
		}
		
		// Run enhanced agent
		RunEnhancedAgent(memAgent)
	} else {
		// Use basic file memory tool
		memoryTool := tools.NewFileMemoryTool(config.MemoryFilePath)
		
		// Create the agent with all tools
		agent := NewAgent(model, []models.Tool{todoTool, memoryTool})
		
		// Start the interactive session
		agent.Run()
	}
}

// RunEnhancedAgent runs the interactive session with the memory-enabled agent
func RunEnhancedAgent(memAgent *MemoryEnabledAgent) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Enhanced LLM Agent with Proactive Memory")
	fmt.Printf("Using model: %s\n", memAgent.agent.model.Name())
	fmt.Println("Available tools:")
	for _, tool := range memAgent.agent.tools {
		fmt.Printf("- %s: %s\n", tool.Name(), strings.Split(tool.Description(), "\n")[0])
	}
	fmt.Println("\nMemory system: Active (automatically storing and retrieving context)")
	fmt.Println("Type 'exit' to quit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		query := scanner.Text()
		if strings.ToLower(query) == "exit" {
			break
		}

		// Process the query with memory enhancement
		fmt.Println("Processing your request...")
		startTime := time.Now()

		response, err := memAgent.ProcessQuery(context.Background(), query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\nResponse:\n%s\n", response)
		fmt.Printf("\nResponse generated in %.2f seconds.\n", elapsed.Seconds())
	}
}
