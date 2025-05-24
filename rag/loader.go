package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	pb "github.com/qdrant/go-client/qdrant"
	"log"
	"os"
	"strings"
)

// Constants for the loader
const (
	DefaultModelFile  = "wiki_minilm.ndjson.gz"
	DefaultDimension  = 384
	DefaultCollection = "wiki_minilm"
	DefaultBatchSize  = 2048
	DefaultQdrantHost = "localhost"
	DefaultQdrantPort = 6334

	// String constants
	TitlePrefix      = "Title: "
	ContentPrefix    = " Content: "
	TitlePrefixLen   = len(TitlePrefix)
	ContentPrefixLen = len(ContentPrefix)
)

// Record represents a single record in the embeddings file
type record struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Embedding []float32 `json:"embedding,omitempty"`
	AllMiniLM []float32 `json:"all-MiniLM-L6-v2,omitempty"`
}

// LoaderConfig holds configuration for the loader
type LoaderConfig struct {
	ModelFile  string
	Dimension  int
	Collection string
	BatchSize  int
	QdrantHost string
	QdrantPort int
}

// NewDefaultLoaderConfig creates a new loader configuration with default values
func NewDefaultLoaderConfig() LoaderConfig {
	return LoaderConfig{
		ModelFile:  DefaultModelFile,
		Dimension:  DefaultDimension,
		Collection: DefaultCollection,
		BatchSize:  DefaultBatchSize,
		QdrantHost: DefaultQdrantHost,
		QdrantPort: DefaultQdrantPort,
	}
}

// Load loads the embeddings from the wiki_minilm.ndjson.gz file into Qdrant
// If a Config is provided, it will use the configuration from it
func loadFromEmbeddings() {
	loadWithConfig(NewDefaultLoaderConfig())
}

// loadWithConfig loads the embeddings with the specified configuration
func loadWithConfig(config LoaderConfig) {
	// Set environment variables to disable HTTP/2
	//os.Setenv("GRPC_GO_REQUIRE_HANDSHAKE", "off")
	//os.Setenv("GRPC_GO_HTTP2_DISABLE", "1")

	ctx := context.Background()

	// Initialize Qdrant client
	client := initQdrantClient(ctx, config)

	// Create or verify collection
	createOrVerifyCollection(ctx, client, config)

	// Process the embeddings file
	processEmbeddingsFile(ctx, client, config)
}

// initQdrantClient initializes the Qdrant client and tests the connection
func initQdrantClient(ctx context.Context, config LoaderConfig) *pb.Client {
	client, err := pb.NewClient(&pb.Config{
		Host: config.QdrantHost,
		Port: config.QdrantPort,
	})
	if err != nil {
		log.Fatalf("Failed to create Qdrant client: %v", err)
	}

	// Test connection before proceeding
	log.Printf("Testing connection to Qdrant at %s:%d...", config.QdrantHost, config.QdrantPort)
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v. Make sure Qdrant is running and accessible.", err)
	}
	log.Printf("✅ Connected to Qdrant. Found %d existing collections", len(collections))

	return client
}

// createOrVerifyCollection creates a collection if it doesn't exist or verifies an existing one
func createOrVerifyCollection(ctx context.Context, client *pb.Client, config LoaderConfig) {
	log.Printf("📁 Creating/verifying collection %q with %d dimensions...", config.Collection, config.Dimension)
	err := client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: config.Collection,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     uint64(config.Dimension),
			Distance: pb.Distance_Cosine,
		}),
	})
	if err != nil {
		// Check if collection already exists
		if collections, listErr := client.ListCollections(ctx); listErr == nil {
			for _, existingCollection := range collections {
				if existingCollection == config.Collection {
					log.Printf("ℹ️ Collection %q already exists", config.Collection)
					return
				}
			}
		}
		log.Fatalf("Failed to create collection: CreateCollection() failed: %s: %v", config.Collection, err)
	}
	log.Printf("✅ Created collection %q successfully", config.Collection)
}

// processEmbeddingsFile processes the embeddings file and loads the data into Qdrant
func processEmbeddingsFile(ctx context.Context, client *pb.Client, config LoaderConfig) {
	// Open dataset
	log.Printf("📂 Opening dataset file: %s", config.ModelFile)
	f, err := os.Open(config.ModelFile)
	if err != nil {
		log.Fatalf("Failed to open dataset file %s: %v", config.ModelFile, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	// Create a decoder to read the JSON data
	dec := json.NewDecoder(bufio.NewReader(gzr))
	log.Printf("🚀 Starting to process embeddings in batches of %d...", config.BatchSize)

	points := make([]*pb.PointStruct, 0, config.BatchSize)
	var id uint64
	var skippedCount uint64
	batchCount := 0

	// Process records
	for dec.More() {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			log.Fatalf("Failed to decode record at position %d: %v", id, err)
		}

		// Extract title and content
		title, content := extractTitleAndContent(rec.Body)

		// Create payload
		payload := createPayload(rec.ID, title, content)

		// Get embedding
		embedding := getEmbedding(rec)

		// Validate embedding
		if !isValidEmbedding(embedding, config.Dimension, id, &skippedCount) {
			continue
		}

		// Create point
		point := createPoint(id, embedding, payload)
		points = append(points, point)
		id++

		// Flush batch if full
		if len(points) == config.BatchSize {
			flushBatch(ctx, client, config.Collection, points, &batchCount, id)
			points = points[:0]
		}
	}

	// Flush any remaining points
	if len(points) > 0 {
		flushBatch(ctx, client, config.Collection, points, &batchCount, id)
	}

	// Log summary
	log.Printf("🎉 Successfully loaded %d vectors into Qdrant collection %q across %d batches", id, config.Collection, batchCount)
	if skippedCount > 0 {
		log.Printf("⚠️ Skipped %d records due to empty or incorrectly dimensioned embeddings", skippedCount)
	}
}

// extractTitleAndContent extracts the title and content from the body text
func extractTitleAndContent(body string) (string, string) {
	title := ""
	content := ""

	if len(body) > 0 {
		if titleStart := strings.Index(body, TitlePrefix); titleStart >= 0 {
			titleStart += TitlePrefixLen
			titleEnd := strings.Index(body[titleStart:], ContentPrefix)
			if titleEnd > 0 {
				title = body[titleStart : titleStart+titleEnd]
				contentStart := titleStart + titleEnd + ContentPrefixLen
				content = body[contentStart:]
			}
		}
	}

	return title, content
}

// createPayload creates a payload map for the record
func createPayload(id, title, content string) map[string]*pb.Value {
	pl := map[string]interface{}{
		"title":   title,
		"content": content,
		"id":      id,
	}

	return pb.NewValueMap(pl)
}

// getEmbedding gets the embedding from the record, preferring AllMiniLM if available
func getEmbedding(rec record) []float32 {
	if len(rec.AllMiniLM) > 0 {
		return rec.AllMiniLM
	} else if len(rec.Embedding) > 0 {
		return rec.Embedding
	}
	return nil
}

// isValidEmbedding checks if the embedding is valid
func isValidEmbedding(embedding []float32, dimension int, id uint64, skippedCount *uint64) bool {
	// Check if embedding is empty
	if len(embedding) == 0 {
		log.Printf("⚠️ Warning: Record %d has empty embedding, skipping", id)
		*skippedCount++
		return false
	}

	// Check if embedding has the expected dimension
	if len(embedding) != dimension {
		log.Printf("⚠️ Warning: Record %d has unexpected embedding dimension (expected %d, got %d), skipping",
			id, dimension, len(embedding))
		*skippedCount++
		return false
	}

	return true
}

// createPoint creates a point struct for Qdrant
func createPoint(id uint64, embedding []float32, payload map[string]*pb.Value) *pb.PointStruct {
	return &pb.PointStruct{
		Id: &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: id}},
		Vectors: &pb.Vectors{
			VectorsOptions: &pb.Vectors_Vector{
				Vector: &pb.Vector{Data: embedding},
			},
		},
		Payload: payload,
	}
}

// flushBatch flushes a batch of points to Qdrant
func flushBatch(ctx context.Context, client *pb.Client, collection string, points []*pb.PointStruct, batchCount *int, totalProcessed uint64) {
	log.Printf("💾 Upserting batch %d with %d points...", *batchCount+1, len(points))
	if _, err := client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collection,
		Points:         points,
	}); err != nil {
		log.Fatalf("Failed to upsert batch %d: %v", *batchCount+1, err)
	}
	*batchCount++
	log.Printf("✅ Successfully upserted batch %d (total processed: %d)", *batchCount, totalProcessed)
}
