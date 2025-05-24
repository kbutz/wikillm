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

	"github.com/tmc/langchaingo/llms"
)

// Config holds configuration options for the application
type Config struct {
	ModelName            string // Name of the LLM model to use
	ModelProvider        string // Provider to use (lmstudio, ollama, openai)
	EmbeddingModel       string // Name of the embedding model to use
	EmbeddingProvider    string // Provider for embeddings (ollama, openai)
	WikipediaPath        string // Path to the Wikipedia dump file
	QdrantURL            string // URL for the Qdrant vector database
	QdrantCollectionName string // Collection name for the Qdrant vector database
	SearchLimit          int    // Maximum number of search results to return
	OpenAIAPIKey         string // OpenAI API key for LM Studio compatibility
	OllamaURL            string // Ollama server URL
	ForceRecreate        bool   // Force recreate collection if dimensions mismatch
}

func main() {
	// Parse configuration
	config := parseFlags()

	// Get provider and create model
	provider := GetProvider(config)

	log.Printf("Initializing %s model: %s", provider.Name(), config.ModelName)
	model, err := provider.CreateLLM(config)
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
	}

	// Initialize RAG pipeline
	log.Println("Initializing RAG pipeline...")
	ragPipeline, err := NewRAGPipeline(config)
	if err != nil {
		log.Fatalf("Failed to initialize RAG pipeline: %v", err)
	}
	defer func() {
		err = ragPipeline.Close()
		if err != nil {
			log.Printf("Closing RAG pipeline: %v", err)
		}
	}()

	// Index Wikipedia if a path is provided
	if config.WikipediaPath != "" {
		//log.Printf("Indexing Wikipedia dump: %s", config.WikipediaPath)
		//if err := ragPipeline.IndexWikipediaDump(config.WikipediaPath); err != nil {
		//	log.Fatalf("Failed to index Wikipedia: %v", err)
		//}
		//log.Println("✅ Indexing complete")
		Load()
	}

	// Start an interactive session
	startInteractiveSession(model, ragPipeline, config)
}

// parseFlags parses command line flags and returns a Config struct
func parseFlags() Config {
	modelName := flag.String("model", "llama3.2", "Name of the LLM model to use")
	modelProvider := flag.String("provider", "ollama", "Model provider to use (ollama, openai, lmstudio)")
	// Previously nomic-embed-text, trying all-minilm
	embeddingModel := flag.String("embedding-model", "all-minilm", "Name of the embedding model to use")
	embeddingProvider := flag.String("embedding-provider", "", "Provider for embeddings (defaults to model provider)")
	wikipediaPath := flag.String("wikipedia", "", "Path to the Wikipedia dump file")
	qdrantURL := flag.String("qdrant-url", "http://localhost:6333", "URL for the Qdrant vector database")
	// value from load() is wiki_minilm, value from the original langchain embedder was wikipedia
	qdrantCollection := flag.String("qdrant-collection", "wiki_minilm", "Collection name for Qdrant")
	searchLimit := flag.Int("limit", 5, "Maximum number of search results")
	openaiKey := flag.String("openai-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama server URL")
	forceRecreate := flag.Bool("force-recreate", false, "Force recreate collection if dimensions mismatch")
	testConnection := flag.Bool("test-connection", false, "Test Qdrant connection and exit")
	testLoad := flag.Bool("test-load", false, "Test loading the wiki_minilm.ndjson.gz file and exit")

	flag.Parse()

	// Get API key from environment if not provided
	apiKey := *openaiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	config := Config{
		ModelName:            *modelName,
		ModelProvider:        *modelProvider,
		EmbeddingModel:       *embeddingModel,
		EmbeddingProvider:    *embeddingProvider,
		WikipediaPath:        *wikipediaPath,
		QdrantURL:            *qdrantURL,
		QdrantCollectionName: *qdrantCollection,
		SearchLimit:          *searchLimit,
		OpenAIAPIKey:         apiKey,
		OllamaURL:            *ollamaURL,
		ForceRecreate:        *forceRecreate,
	}

	// Test connection if requested
	if *testConnection {
		log.Println("=== Qdrant Connection Test ===")
		if err := TestQdrantConnection(); err != nil {
			log.Fatalf("❌ Connection test failed: %v", err)
		}
		os.Exit(0)
	}

	// Test loading if requested
	if *testLoad {
		log.Println("=== Testing Loading wiki_minilm.ndjson.gz ===")
		Load()
		os.Exit(0)
	}

	return config
}

// startInteractiveSession provides an interactive chat interface
func startInteractiveSession(model llms.Model, ragPipeline *RAGPipeline, config Config) {
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	fmt.Println("=== WikiLLM RAG Interactive Session ===")
	fmt.Printf("Model: %s (%s)\n", config.ModelName, config.ModelProvider)
	fmt.Printf("Embedding: %s (%d dimensions)\n", config.EmbeddingModel, ragPipeline.vectorSize)
	fmt.Printf("Vector Store: %s\n", config.QdrantURL)
	fmt.Println("Type 'exit' to quit, 'help' for commands")
	fmt.Println(strings.Repeat("=", 50))

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "help":
			fmt.Println("Commands:")
			fmt.Println("  exit/quit - Exit the session")
			fmt.Println("  help      - Show this help")
			fmt.Println("  Or ask any question about Wikipedia content")
			continue
		}

		// Process the query
		fmt.Println("🔍 Searching and generating response...")
		startTime := time.Now()

		response, err := processQuery(ctx, model, ragPipeline, input, config.SearchLimit)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\n📝 Response (%.2fs):\n%s\n", elapsed.Seconds(), response)
	}
}

// ProcessQuery handles a user query with improved context formatting
func processQuery(ctx context.Context, model llms.Model, ragPipeline *RAGPipeline, query string, limit int) (string, error) {
	// Search for relevant documents
	docs, err := ragPipeline.Search(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}

	if len(docs) == 0 {
		log.Println("Debug: No results found from vector store, querying model directly...")
		// If no results found, ask the model directly
		return llms.GenerateFromSinglePrompt(ctx, model, query)
	}

	// Build context from search results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Answer the following question based on the provided Wikipedia context.\n\n")
	contextBuilder.WriteString("Question: " + query + "\n\n")
	contextBuilder.WriteString("Context:\n")

	for i, doc := range docs {
		title, _ := doc.Metadata["title"].(string)
		content := doc.PageContent

		// Truncate content if too long
		if len(content) > 800 {
			content = content[:800] + "..."
		}

		contextBuilder.WriteString(fmt.Sprintf("%d. %s\n%s\n\n", i+1, title, content))
		log.Printf("Debug: Context %d: %s\n content: %s\n", i+1, title, content)
	}

	contextBuilder.WriteString("Please provide a comprehensive answer based on the context above. If the context doesn't contain enough information, mention that.")

	// Generate response using the new API
	return llms.GenerateFromSinglePrompt(ctx, model, contextBuilder.String(),
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(1000),
	)
}
