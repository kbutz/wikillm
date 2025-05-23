package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Configuration options for the application
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
}

// WikipediaPage represents a page in the Wikipedia dump
type WikipediaPage struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Content string `xml:"revision>text"`
}

// RAGPipeline manages the RAG (Retrieval-Augmented Generation) pipeline
type RAGPipeline struct {
	embedder       embeddings.Embedder
	vectorStore    vectorstores.VectorStore
	collectionName string
}

// LLMProvider interface for different model providers
type LLMProvider interface {
	CreateLLM(config Config) (llms.Model, error)
	CreateEmbedder(config Config) (embeddings.Embedder, error)
	Name() string
}

// OllamaProvider implements LLMProvider for Ollama
type OllamaProvider struct{}

func (p *OllamaProvider) CreateLLM(config Config) (llms.Model, error) {
	return ollama.New(
		ollama.WithModel(config.ModelName),
		ollama.WithServerURL(config.OllamaURL),
	)
}

func (p *OllamaProvider) CreateEmbedder(config Config) (embeddings.Embedder, error) {
	llm, err := ollama.New(
		ollama.WithModel(config.EmbeddingModel),
		ollama.WithServerURL(config.OllamaURL),
	)
	if err != nil {
		if strings.Contains(err.Error(), "model not found") {
			return nil, fmt.Errorf("embedding model %q not found. Please pull it first with: ollama pull %s",
				config.EmbeddingModel, config.EmbeddingModel)
		}
		return nil, fmt.Errorf("failed to create Ollama LLM for embeddings: %w", err)
	}

	return embeddings.NewEmbedder(llm)
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

// OpenAIProvider implements LLMProvider for OpenAI (including LM Studio compatibility)
type OpenAIProvider struct{}

func (p *OpenAIProvider) CreateLLM(config Config) (llms.Model, error) {
	options := []openai.Option{
		openai.WithModel(config.ModelName),
	}

	// For LM Studio compatibility
	if config.ModelProvider == "lmstudio" {
		options = append(options,
			openai.WithBaseURL("http://localhost:1234/v1"),
			openai.WithToken(config.OpenAIAPIKey),
		)
	} else {
		options = append(options, openai.WithToken(config.OpenAIAPIKey))
	}

	return openai.New(options...)
}

func (p *OpenAIProvider) CreateEmbedder(config Config) (embeddings.Embedder, error) {
	options := []openai.Option{
		openai.WithModel(config.EmbeddingModel),
		openai.WithToken(config.OpenAIAPIKey),
	}

	// For LM Studio compatibility
	if config.ModelProvider == "lmstudio" {
		options = append(options, openai.WithBaseURL("http://localhost:1234/v1"))
	}

	client, err := openai.New(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client for embeddings: %w", err)
	}

	return embeddings.NewEmbedder(client)
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

// GetProvider returns the appropriate provider based on configuration
func GetProvider(config Config) LLMProvider {
	switch strings.ToLower(config.ModelProvider) {
	case "ollama":
		return &OllamaProvider{}
	case "openai", "lmstudio":
		return &OpenAIProvider{}
	default:
		log.Printf("Unknown provider %s, defaulting to Ollama", config.ModelProvider)
		return &OllamaProvider{}
	}
}

// createQdrantCollection creates a collection in Qdrant if it doesn't exist
func createQdrantCollection(qdrantURL *url.URL, collectionName string, vectorSize int) error {
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/%s", qdrantURL.String(), collectionName)
	resp, err := http.Get(checkURL)
	if err != nil {
		return fmt.Errorf("failed to check if collection exists: %w", err)
	}
	defer resp.Body.Close()

	// If collection exists, return
	if resp.StatusCode == http.StatusOK {
		log.Printf("Collection %s already exists", collectionName)
		return nil
	}

	// Collection doesn't exist, create it
	createURL := fmt.Sprintf("%s/collections/%s", qdrantURL.String(), collectionName)

	// Prepare request body
	requestBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create collection
	req, err := http.NewRequest("PUT", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("Created collection %s", collectionName)
	return nil
}

// NewRAGPipeline creates a new RAG pipeline with the latest APIs
func NewRAGPipeline(config Config) (*RAGPipeline, error) {
	// Get the appropriate provider
	provider := GetProvider(config)

	// Create embedder based on provider
	var embedder embeddings.Embedder
	var err error

	if config.EmbeddingProvider != "" {
		// Use specific provider for embeddings if specified
		embeddingConfig := config
		embeddingConfig.ModelProvider = config.EmbeddingProvider
		embeddingConfig.ModelName = config.EmbeddingModel

		embeddingProvider := GetProvider(embeddingConfig)
		embedder, err = embeddingProvider.CreateEmbedder(embeddingConfig)
	} else {
		// Use same provider for embeddings
		embedder, err = provider.CreateEmbedder(config)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Determine vector dimensions by making a test embedding
	vectorSize, err := getEmbeddingDimensions(embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to determine embedding dimensions: %w", err)
	}
	log.Printf("Detected embedding dimensions: %d", vectorSize)

	// Parse Qdrant URL
	qdrantURL, err := url.Parse(config.QdrantURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Qdrant URL: %w", err)
	}

	// Create the collection if it doesn't exist with the correct dimensions
	if err := createQdrantCollection(qdrantURL, config.QdrantCollectionName, vectorSize); err != nil {
		return nil, fmt.Errorf("failed to create Qdrant collection: %w", err)
	}

	// Create Qdrant vector store using the new API
	store, err := qdrant.New(
		qdrant.WithURL(*qdrantURL),
		qdrant.WithCollectionName(config.QdrantCollectionName),
		qdrant.WithEmbedder(embedder),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant store: %w", err)
	}

	return &RAGPipeline{
		embedder:       embedder,
		vectorStore:    store,
		collectionName: config.QdrantCollectionName,
	}, nil
}

// cleanWikiMarkup removes wiki markup from the content
func cleanWikiMarkup(content string) string {
	// Basic cleanup - a real implementation would be more sophisticated
	content = strings.ReplaceAll(content, "[[", "")
	content = strings.ReplaceAll(content, "]]", "")
	content = strings.ReplaceAll(content, "{{", "")
	content = strings.ReplaceAll(content, "}}", "")
	content = strings.ReplaceAll(content, "'''", "")
	content = strings.ReplaceAll(content, "''", "")
	content = strings.ReplaceAll(content, "<ref>", "")
	content = strings.ReplaceAll(content, "</ref>", "")

	// Remove common Wikipedia templates and formatting
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#REDIRECT") ||
			strings.HasPrefix(line, "{{") || strings.HasPrefix(line, "|}") ||
			strings.HasPrefix(line, "|") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, " ")
}

// IndexWikipediaDump indexes a Wikipedia XML dump file using the new API
func (r *RAGPipeline) IndexWikipediaDump(dumpPath string) error {
	ctx := context.Background()

	file, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to open dump file: %w", err)
	}
	defer file.Close()

	batchSize := 50
	var documents []schema.Document
	totalIndexed := 0

	decoder := xml.NewDecoder(file)
	var inPage bool
	var currentPage WikipediaPage

	log.Println("Starting Wikipedia indexing...")

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error decoding XML: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "page" {
				inPage = true
				currentPage = WikipediaPage{}
			} else if inPage {
				switch se.Name.Local {
				case "title":
					var title string
					if err := decoder.DecodeElement(&title, &se); err != nil {
						log.Printf("Error decoding title: %v", err)
						continue
					}
					currentPage.Title = title
				case "id":
					if currentPage.ID == "" { // Only take the first ID (page ID, not revision ID)
						var id string
						if err := decoder.DecodeElement(&id, &se); err != nil {
							log.Printf("Error decoding ID: %v", err)
							continue
						}
						currentPage.ID = id
					}
				case "text":
					var content string
					if err := decoder.DecodeElement(&content, &se); err != nil {
						log.Printf("Error decoding content: %v", err)
						continue
					}
					currentPage.Content = content
				}
			}
		case xml.EndElement:
			if se.Name.Local == "page" && inPage {
				if currentPage.Title != "" && currentPage.ID != "" && currentPage.Content != "" {
					// Clean up the content
					cleanContent := cleanWikiMarkup(currentPage.Content)

					// Skip empty or very short content
					if len(cleanContent) < 100 {
						inPage = false
						continue
					}

					// Create document using the new schema
					doc := schema.Document{
						PageContent: cleanContent,
						Metadata: map[string]any{
							"id":     currentPage.ID,
							"title":  currentPage.Title,
							"source": "wikipedia",
						},
					}

					documents = append(documents, doc)
					totalIndexed++

					// Process batch when full
					if len(documents) >= batchSize {
						if err := r.processBatch(ctx, documents); err != nil {
							return fmt.Errorf("error processing batch: %w", err)
						} else {
							log.Printf("Indexed %d pages", totalIndexed)
						}
						documents = documents[:0] // Reset slice
					}
				}
				inPage = false
			}
		}
	}

	// Process remaining documents
	if len(documents) > 0 {
		if err := r.processBatch(ctx, documents); err != nil {
			return fmt.Errorf("error processing final batch: %w", err)
		}
	}

	log.Printf("Indexing complete. Total pages indexed: %d", totalIndexed)
	return nil
}

// processBatch adds a batch of documents to the vector store
func (r *RAGPipeline) processBatch(ctx context.Context, documents []schema.Document) error {
	_, err := r.vectorStore.AddDocuments(ctx, documents)
	return err
}

// Search searches for documents similar to the query using the new API
func (r *RAGPipeline) Search(ctx context.Context, query string, limit int) ([]schema.Document, error) {
	// Use the new SimilaritySearch method
	docs, err := r.vectorStore.SimilaritySearch(ctx, query, limit,
		vectorstores.WithScoreThreshold(0.7), // Adjust threshold as needed
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return docs, nil
}

// getEmbeddingDimensions determines the vector dimensions by making a test embedding
func getEmbeddingDimensions(embedder embeddings.Embedder) (int, error) {
	ctx := context.Background()

	// Make a test embedding with a simple text
	embeddings, err := embedder.EmbedDocuments(ctx, []string{"test"})
	if err != nil {
		return 0, fmt.Errorf("failed to create test embedding: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return 0, fmt.Errorf("embedding model returned empty vector")
	}

	// Return the dimensions of the first vector
	return len(embeddings[0]), nil
}

// Close closes the RAG pipeline
func (r *RAGPipeline) Close() error {
	// Nothing specific to close for now
	return nil
}

// parseFlags parses command line flags with updated options
func parseFlags() Config {
	modelName := flag.String("model", "llama3.2", "Name of the LLM model to use")
	modelProvider := flag.String("provider", "ollama", "Model provider to use (ollama, openai, lmstudio)")
	embeddingModel := flag.String("embedding-model", "nomic-embed-text", "Name of the embedding model to use")
	embeddingProvider := flag.String("embedding-provider", "", "Provider for embeddings (defaults to model provider)")
	wikipediaPath := flag.String("wikipedia", "", "Path to the Wikipedia dump file")
	qdrantURL := flag.String("qdrant-url", "http://localhost:6333", "URL for the Qdrant vector database")
	qdrantCollection := flag.String("qdrant-collection", "wikipedia", "Collection name for Qdrant")
	searchLimit := flag.Int("limit", 5, "Maximum number of search results")
	openaiKey := flag.String("openai-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
	ollamaURL := flag.String("ollama-url", "http://localhost:11434", "Ollama server URL")

	flag.Parse()

	// Get API key from environment if not provided
	apiKey := *openaiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	return Config{
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
	}
}

// processQuery handles a user query with improved context formatting
func processQuery(ctx context.Context, model llms.Model, ragPipeline *RAGPipeline, query string, limit int) (string, error) {
	// Search for relevant documents
	docs, err := ragPipeline.Search(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}

	if len(docs) == 0 {
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
	}

	contextBuilder.WriteString("Please provide a comprehensive answer based on the context above. If the context doesn't contain enough information, mention that.")

	// Generate response using the new API
	return llms.GenerateFromSinglePrompt(ctx, model, contextBuilder.String(),
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(1000),
	)
}

// startInteractiveSession provides an interactive chat interface
func startInteractiveSession(model llms.Model, ragPipeline *RAGPipeline, config Config) {
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	fmt.Println("=== WikiLLM RAG Interactive Session ===")
	fmt.Printf("Model: %s (%s)\n", config.ModelName, config.ModelProvider)
	fmt.Printf("Embedding: %s\n", config.EmbeddingModel)
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
	defer ragPipeline.Close()

	// Index Wikipedia if path is provided
	if config.WikipediaPath != "" {
		log.Printf("Indexing Wikipedia dump: %s", config.WikipediaPath)
		if err := ragPipeline.IndexWikipediaDump(config.WikipediaPath); err != nil {
			log.Fatalf("Failed to index Wikipedia: %v", err)
		}
		log.Println("✅ Indexing complete")
	}

	// Start interactive session
	startInteractiveSession(model, ragPipeline, config)
}
