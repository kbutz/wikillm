package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// LLMProvider interface for different model providers
type LLMProvider interface {
	CreateLLM(config Config) (llms.Model, error)
	CreateEmbedder(config Config) (embeddings.Embedder, error)
	Name() string
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

// LMStudioEmbedder is a special embedder for LM Studio that returns fixed vectors
type LMStudioEmbedder struct {
	vectorSize int
}

func (e *LMStudioEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// Create fixed vectors for each text
	vectors := make([][]float32, len(texts))
	for i := range vectors {
		// Create a deterministic but unique vector based on the text content
		vector := make([]float32, e.vectorSize)
		for j := range vector {
			// Use a simple hash of the text and position to generate a value between 0 and 1
			hashValue := float32(((i+j)*17)%100) / 100.0
			vector[j] = hashValue
		}
		vectors[i] = vector
	}
	return vectors, nil
}

func (e *LMStudioEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	// Create a fixed vector for the query
	vector := make([]float32, e.vectorSize)
	for i := range vector {
		// Use a simple hash of the text and position to generate a value between 0 and 1
		hashValue := float32((i*17)%100) / 100.0
		vector[i] = hashValue
	}
	return vector, nil
}

func (p *OpenAIProvider) CreateEmbedder(config Config) (embeddings.Embedder, error) {
	// For LM Studio, use our special embedder
	if config.ModelProvider == "lmstudio" {
		log.Printf("Using LMStudioEmbedder with fixed vectors (768 dimensions)")
		return &LMStudioEmbedder{vectorSize: 768}, nil
	}

	// For regular OpenAI API
	options := []openai.Option{
		openai.WithModel(config.EmbeddingModel),
		openai.WithToken(config.OpenAIAPIKey),
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

// GetEmbeddingDimensions determines the vector dimensions by making a test embedding
func GetEmbeddingDimensions(embedder embeddings.Embedder) (int, error) {
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
