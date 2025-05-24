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

type record struct {
	ID         string    `json:"id"`
	Body       string    `json:"body"`
	Embedding  []float32 `json:"embedding,omitempty"`
	AllMiniLM  []float32 `json:"all-MiniLM-L6-v2,omitempty"`
}

// --------------------------------------------------------------------------

// Load loads the embeddings from the wiki_minilm.ndjson.gz file into Qdrant
func Load() {
	const (
		modelFile  = "wiki_minilm.ndjson.gz"
		dim        = 384
		collection = "wiki_minilm"
		batchSize  = 2048
		qdrantHost = "localhost"
		qdrantPort = 6334
	)

	// Set environment variables to disable HTTP/2
	os.Setenv("GRPC_GO_REQUIRE_HANDSHAKE", "off")
	os.Setenv("GRPC_GO_HTTP2_DISABLE", "1")

	ctx := context.Background()

	client, err := pb.NewClient(&pb.Config{
		Host: qdrantHost,
		Port: qdrantPort,
	})
	if err != nil {
		log.Fatalf("Failed to create Qdrant client: %v", err)
	}

	// Test connection before proceeding
	log.Printf("Testing connection to Qdrant at %s:%d...", qdrantHost, qdrantPort)
	collections, err := client.ListCollections(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v. Make sure Qdrant is running and accessible.", err)
	}
	log.Printf("✅ Connected to Qdrant. Found %d existing collections", len(collections))

	// create collection once
	log.Printf("📁 Creating/verifying collection %q with %d dimensions...", collection, dim)
	err = client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: collection,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     dim,
			Distance: pb.Distance_Cosine,
		}),
	})
	if err != nil {
		// Check if collection already exists
		if collections, listErr := client.ListCollections(ctx); listErr == nil {
			for _, existingCollection := range collections {
				if existingCollection == collection {
					log.Printf("ℹ️ Collection %q already exists", collection)
					goto CollectionReady
				}
			}
		}
		log.Fatalf("Failed to create collection: CreateCollection() failed: %s: %v", collection, err)
	}
	log.Printf("✅ Created collection %q successfully", collection)

CollectionReady:

	// open dataset
	log.Printf("📂 Opening dataset file: %s", modelFile)
	f, err := os.Open(modelFile)
	if err != nil {
		log.Fatalf("Failed to open dataset file %s: %v", modelFile, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	dec := json.NewDecoder(bufio.NewReader(gzr))
	log.Printf("🚀 Starting to process embeddings in batches of %d...", batchSize)

	points := make([]*pb.PointStruct, 0, batchSize)
	var id uint64
	var skippedCount uint64

	batchCount := 0
	flush := func() {
		if len(points) == 0 {
			return
		}
		log.Printf("💾 Upserting batch %d with %d points...", batchCount+1, len(points))
		if _, err := client.Upsert(ctx, &pb.UpsertPoints{
			CollectionName: collection,
			Points:         points,
		}); err != nil {
			log.Fatalf("Failed to upsert batch %d: %v", batchCount+1, err)
		}
		batchCount++
		log.Printf("✅ Successfully upserted batch %d (total processed: %d)", batchCount, id)
		points = points[:0]
	}

	// Create a decoder to read the JSON data
	dec = json.NewDecoder(bufio.NewReader(gzr))

	for dec.More() {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			log.Fatalf("Failed to decode record at position %d: %v", id, err)
		}

		// Extract title from the body (format is typically "Title: XXX Content: YYY")
		title := ""
		content := ""
		if len(rec.Body) > 0 {
			// Try to extract title and content
			if titleStart := strings.Index(rec.Body, "Title: "); titleStart >= 0 {
				titleStart += 7 // Length of "Title: "
				titleEnd := strings.Index(rec.Body[titleStart:], " Content: ")
				if titleEnd > 0 {
					title = rec.Body[titleStart : titleStart+titleEnd]
					contentStart := titleStart + titleEnd + 10 // Length of " Content: "
					content = rec.Body[contentStart:]
				}
			}
		}

		pl := map[string]interface{}{
			"title": title,
			"content": content,
			"id": rec.ID,
		}

		// Convert map[string]interface{} to map[string]*pb.Value using the helper function
		payload := pb.NewValueMap(pl)

		// Use AllMiniLM field if available, otherwise fall back to Embedding field
		var embedding []float32
		if len(rec.AllMiniLM) > 0 {
			embedding = rec.AllMiniLM
		} else if len(rec.Embedding) > 0 {
			embedding = rec.Embedding
		}

		// Check if embedding is empty or has zero length
		if len(embedding) == 0 {
			log.Printf("⚠️ Warning: Record %d has empty embedding, skipping", id)
			skippedCount++
			continue
		}

		// Check if embedding has the expected dimension
		if len(embedding) != dim {
			log.Printf("⚠️ Warning: Record %d has unexpected embedding dimension (expected %d, got %d), skipping",
				id, dim, len(embedding))
			skippedCount++
			continue
		}

		points = append(points, &pb.PointStruct{
			Id: &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: id}},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: embedding},
				},
			},
			Payload: payload,
		})
		id++

		if len(points) == batchSize {
			flush()
		}
	}
	flush()
	log.Printf("🎉 Successfully loaded %d vectors into Qdrant collection %q across %d batches", id, collection, batchCount)
	if skippedCount > 0 {
		log.Printf("⚠️ Skipped %d records due to empty or incorrectly dimensioned embeddings", skippedCount)
	}
}
