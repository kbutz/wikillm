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
)

// Configuration options for the application
type Config struct {
	ModelName      string // Name of the LLM model to use
	ModelProvider  string // Provider to use (lmstudio or ollama)
	WikipediaPath  string // Path to the Wikipedia dump file
	IndexDirectory string // Directory to store the search index
	SearchLimit    int    // Maximum number of search results to return
}

func main() {
	// Parse command line arguments
	config := parseFlags()

	// Initialize the LLM model
	var model LLMModel
	var err error

	// Initialize the model based on the selected provider
	switch strings.ToLower(config.ModelProvider) {
	case "lmstudio":
		model, err = NewLMStudioModel(config.ModelName)
		if err != nil {
			log.Fatalf("Failed to initialize LM Studio model: %v", err)
		}
	case "ollama":
		model, err = NewOllamaModel(config.ModelName)
		if err != nil {
			log.Fatalf("Failed to initialize Ollama model: %v", err)
		}
	default:
		log.Printf("Unknown model provider: %s. Defaulting to LM Studio.", config.ModelProvider)
		model, err = NewLMStudioModel(config.ModelName)
		if err != nil {
			log.Fatalf("Failed to initialize LM Studio model: %v", err)
		}
	}

	// Initialize the Wikipedia index
	wikiIndex, err := NewWikipediaIndex(config.IndexDirectory)
	if err != nil {
		log.Fatalf("Failed to initialize Wikipedia index: %v", err)
	}
	defer wikiIndex.Close()

	// Check if we need to create the index
	if config.WikipediaPath != "" {
		log.Println("Creating new index from Wikipedia dump...")
		err = wikiIndex.IndexWikipediaDump(config.WikipediaPath)
		if err != nil {
			log.Fatalf("Failed to create index: %v", err)
		}
		log.Println("Index created successfully.")
	}

	// Start interactive session
	startInteractiveSession(model, wikiIndex, config.SearchLimit)
}

// Parse command line flags
func parseFlags() Config {
	modelName := flag.String("model", "default", "Name of the LLM model to use")
	modelProvider := flag.String("provider", "lmstudio", "Model provider to use (lmstudio or ollama)")
	wikipediaPath := flag.String("wikipedia", "", "Path to the Wikipedia dump file (only needed for initial indexing)")
	indexDirectory := flag.String("index", "./wikipedia_index", "Directory to store the search index")
	searchLimit := flag.Int("limit", 5, "Maximum number of search results to return")

	flag.Parse()

	return Config{
		ModelName:      *modelName,
		ModelProvider:  *modelProvider,
		WikipediaPath:  *wikipediaPath,
		IndexDirectory: *indexDirectory,
		SearchLimit:    *searchLimit,
	}
}

// Start an interactive session with the user
func startInteractiveSession(model LLMModel, wikiIndex *WikipediaIndex, searchLimit int) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("WikiLLM Interactive Session")
	fmt.Printf("Using model: %s\n", model.Name())
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
		fmt.Println("Searching Wikipedia and generating response...")
		startTime := time.Now()

		response, err := processQuery(context.Background(), model, wikiIndex, query, searchLimit)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\nResponse (generated in %.2f seconds):\n%s\n", elapsed.Seconds(), response)
	}
}

// Process a user query
func processQuery(ctx context.Context, model LLMModel, wikiIndex *WikipediaIndex, query string, limit int) (string, error) {
	// Search Wikipedia for relevant content
	results, err := wikiIndex.Search(query, limit)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}

	if len(results) == 0 {
		// If no results found, ask the model directly
		return model.Query(ctx, query)
	}

	// Format the search results for the model
	var promptBuilder strings.Builder

	promptBuilder.WriteString("I want you to answer the following question based on the Wikipedia information provided below.\n\n")
	promptBuilder.WriteString("Question: " + query + "\n\n")
	promptBuilder.WriteString("Wikipedia Information:\n")

	for i, result := range results {
		title, _ := result["title"].(string)
		content, _ := result["content"].(string)

		// Truncate content if it's too long
		if len(content) > 1000 {
			content = content[:1000] + "..."
		}

		promptBuilder.WriteString(fmt.Sprintf("%d. %s\n%s\n\n", i+1, title, content))
	}

	promptBuilder.WriteString("Please provide a comprehensive answer to the question based on the information above.")

	// Send the prompt to the model
	return model.Query(ctx, promptBuilder.String())
}
