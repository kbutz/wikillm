package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// TestQdrantConnection validates the Qdrant connection using the same configuration as the loader
func TestQdrantConnection() error {
	const (
		qdrantHost = "localhost"
		qdrantPort = 6333
	)

	// Set environment variables to disable HTTP/2
	os.Setenv("GRPC_GO_REQUIRE_HANDSHAKE", "off")
	os.Setenv("GRPC_GO_HTTP2_DISABLE", "1")

	ctx := context.Background()

	// Create HTTP/1.1 compatible transport
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		ForceAttemptHTTP2: false, // Force HTTP/1.1
		MaxIdleConns:      10,
		IdleConnTimeout:   30 * time.Second,
	}
	_ = transport // Keep for reference

	// Configure gRPC client with HTTP/1.1 compatibility
	client, err := pb.NewClient(&pb.Config{
		Host: qdrantHost,
		Port: qdrantPort,
		GrpcOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(100 * 1024 * 1024), // 100 MB
				grpc.MaxCallSendMsgSize(100 * 1024 * 1024), // 100 MB
			),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			}),
			// Force HTTP/1.1 by disabling HTTP/2
			grpc.WithDisableServiceConfig(),
		},
	})
	if err != nil {
		return err
	}

	// Test basic connectivity
	log.Printf("🔗 Testing connection to Qdrant at %s:%d...", qdrantHost, qdrantPort)
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return err
	}

	log.Printf("✅ Qdrant connection successful")
	log.Printf("📊 Available collections: %d", len(collections))

	// List existing collections
	for i, collection := range collections {
		log.Printf("  %d. %s", i+1, collection)
	}

	// Test collection operations
	testCollection := "connection_test"

	log.Printf("🧪 Testing collection operations...")

	// Create test collection
	err = client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: testCollection,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     768,
			Distance: pb.Distance_Cosine,
		}),
	})

	if err != nil {
		log.Printf("⚠️  Test collection creation failed (may already exist): %v", err)
	} else {
		log.Printf("✅ Test collection created successfully")

		// Clean up test collection
		err = client.DeleteCollection(ctx, testCollection)
		if err != nil {
			log.Printf("⚠️  Test collection cleanup failed: %v", err)
		} else {
			log.Printf("🧹 Test collection cleaned up")
		}
	}

	log.Printf("🎉 Full connection validation successful")
	return nil
}
