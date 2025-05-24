package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	pb "github.com/qdrant/go-client/qdrant"
	"log"
	"os"
)

type record struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
}

// --------------------------------------------------------------------------

func load() {
	const (
		modelFile  = "wiki_minilm.ndjson.gz"
		dim        = 384
		collection = "wiki_minilm"
		batchSize  = 2048
		qdrantHost = "localhost"
		qdrantPort = 6333
	)

	// Set environment variables to disable HTTP/2
	os.Setenv("GRPC_GO_REQUIRE_HANDSHAKE", "off")
	os.Setenv("GRPC_GO_HTTP2_DISABLE", "1")

	ctx := context.Background()

	// Configure gRPC client with HTTP/1.1 compatibility
	//client, err := pb.NewClient(&pb.Config{
	//	Host: qdrantHost,
	//	Port: qdrantPort,
	//	GrpcOptions: []grpc.DialOption{
	//		grpc.WithTransportCredentials(insecure.NewCredentials()),
	//		grpc.WithDefaultCallOptions(
	//			grpc.MaxCallRecvMsgSize(100 * 1024 * 1024), // 100 MB
	//			grpc.MaxCallSendMsgSize(100 * 1024 * 1024), // 100 MB
	//		),
	//		grpc.WithKeepaliveParams(keepalive.ClientParameters{
	//			Time:                10 * time.Second,
	//			Timeout:             5 * time.Second,
	//			PermitWithoutStream: true,
	//		}),
	//		// Force HTTP/1.1 by disabling HTTP/2
	//		grpc.WithDisableServiceConfig(),
	//	},
	//})
	client, err := pb.NewClient(&pb.Config{
		Host: "localhost",
		Port: 6334,
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

	for dec.More() {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			log.Fatalf("Failed to decode record at position %d: %v", id, err)
		}

		pl := map[string]interface{}{
			"title": rec.Title,
			"url":   rec.URL,
		}

		// Convert map[string]interface{} to map[string]*pb.Value using the helper function
		payload := pb.NewValueMap(pl)

		points = append(points, &pb.PointStruct{
			Id: &pb.PointId{PointIdOptions: &pb.PointId_Num{Num: id}},
			Vectors: &pb.Vectors{
				VectorsOptions: &pb.Vectors_Vector{
					Vector: &pb.Vector{Data: rec.Embedding},
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
}
