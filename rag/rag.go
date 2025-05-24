package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// RAGPipeline manages the RAG (Retrieval-Augmented Generation) pipeline
type RAGPipeline struct {
	embedder       embeddings.Embedder
	vectorStore    vectorstores.VectorStore
	collectionName string
	vectorSize     int
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
	vectorSize, err := GetEmbeddingDimensions(embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to determine embedding dimensions: %w", err)
	}
	log.Printf("📏 Detected embedding dimensions: %d", vectorSize)

	// Parse Qdrant URL
	qdrantURL, err := url.Parse(config.QdrantURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Qdrant URL: %w", err)
	}

	// Create the collection if it doesn't exist with the correct dimensions
	if err := CreateQdrantCollection(qdrantURL, config.QdrantCollectionName, vectorSize, config.ForceRecreate); err != nil {
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
		vectorSize:     vectorSize,
	}, nil
}

// ProcessBatch adds a batch of documents to the vector store
func (r *RAGPipeline) ProcessBatch(ctx context.Context, documents []schema.Document) error {
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

// Close closes the RAG pipeline
func (r *RAGPipeline) Close() error {
	// Nothing specific to close for now
	return nil
}