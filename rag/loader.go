package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"os"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
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

	ctx := context.Background()
	client, err := pb.NewClient(&pb.Config{
		Host: qdrantHost,
		Port: qdrantPort,
		GrpcOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(100 * 1024 * 1024), // 100 MB
				grpc.MaxCallSendMsgSize(100 * 1024 * 1024), // 100 MB
			),
		},
	})
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}

	// create collection once
	err = client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: collection,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     dim,
			Distance: pb.Distance_Cosine,
		}),
	})
	if err != nil {
		if err.Error() != "collection already exists" {
			log.Fatalf("create collection: %v", err)
		} else {
			log.Printf("Collection %q already exists", collection)
		}
	}

	// open dataset
	f, _ := os.Open(modelFile)
	gzr, _ := gzip.NewReader(f)
	dec := json.NewDecoder(bufio.NewReader(gzr))

	points := make([]*pb.PointStruct, 0, batchSize)
	var id uint64

	flush := func() {
		if len(points) == 0 {
			return
		}
		if _, err := client.Upsert(ctx, &pb.UpsertPoints{
			CollectionName: collection,
			Points:         points,
		}); err != nil {
			log.Fatalf("upsert: %v", err)
		}
		points = points[:0]
	}

	for dec.More() {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			log.Fatalf("decode: %v", err)
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
	log.Printf("Loaded %d vectors into Qdrant collection %q", id, collection)
}
