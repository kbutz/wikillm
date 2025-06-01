package main

import (
	"flag"
	"github.com/kbutz/wikillm/agents/models"
	"github.com/kbutz/wikillm/agents/tools"
	"log"
)

// Parse command line flags
func parseFlags() Config {
	modelName := flag.String("model", "default", "Name of the LLM model to use")
	modelProvider := flag.String("provider", "lmstudio", "Model provider to use (lmstudio or ollama)")
	port := flag.Int("port", 0, "HTTP server port (0 to disable)")
	todoFilePath := flag.String("todo-file", "todo.txt", "Path to the to-do list file")

	flag.Parse()

	return Config{
		ModelName:     *modelName,
		ModelProvider: *modelProvider,
		Port:          *port,
		TodoFilePath:  *todoFilePath,
	}
}

// Config Configuration options for the application
type Config struct {
	ModelName     string // Name of the LLM model to use
	ModelProvider string // Provider to use (lmstudio or ollama)
	Port          int    // HTTP server port
	TodoFilePath  string // Path to the to-do list file
}

func main() {
	// Parse command line arguments
	config := parseFlags()

	model, err := models.New(config.ModelName, config.ModelProvider)
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
		return
	}

	// Initialize the tools
	todoTool := tools.NewTodoListTool(config.TodoFilePath)
	// Create the agent
	agent := NewAgent(model, []Tool{todoTool})

	// Start the interactive session
	agent.Run()
}
