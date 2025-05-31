package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

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

	// Initialize the LLM model
	var model LLMModel
	var err error

	// Initialize the model based on the selected provider
	switch strings.ToLower(config.ModelProvider) {
	case "ollama":
		model, err = NewOllamaModel(config.ModelName)
		if err != nil {
			log.Fatalf("Failed to initialize Ollama model: %v", err)
		}
	default:
		log.Printf("Using LM Studio as the model provider.")
		model, err = NewLMStudioModel(config.ModelName)
		if err != nil {
			log.Fatalf("Failed to initialize LM Studio model: %v", err)
		}
	}

	// Initialize the to-do list tool
	todoTool := NewTodoListTool(config.TodoFilePath)

	// Initialize the agent with the to-do list tool
	agent := NewAgent(model, []Tool{todoTool})

	// Start HTTP server if port is specified
	if config.Port > 0 {
		go startHTTPServer(config.Port, agent)
	}

	// Start interactive session
	startInteractiveSession(agent)
}

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

// Start an interactive session with the user
func startInteractiveSession(agent *Agent) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("LLM Agent To-Do List")
	fmt.Printf("Using model: %s\n", agent.model.Name())
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

		// Process the query
		fmt.Println("Processing your request...")
		startTime := time.Now()

		response, err := agent.ProcessQuery(context.Background(), query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\nResponse (generated in %.2f seconds):\n%s\n", elapsed.Seconds(), response)
	}
}

// Start HTTP server
func startHTTPServer(port int, agent *Agent) {
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Query string `json:"query"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		response, err := agent.ProcessQuery(r.Context(), requestBody.Query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error processing query: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": response})
	})

	log.Printf("Starting HTTP server on port %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
