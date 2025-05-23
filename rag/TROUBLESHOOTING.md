# Vector Dimension Mismatch - Troubleshooting Guide

This guide helps resolve the common "Vector dimension error" when using WikiLLM RAG with different embedding models.

## The Problem

```
{"status":{"error":"Wrong input: Vector dimension error: expected dim: 1536, got 768"}}
```

This error occurs when:
1. Qdrant collection was created with one embedding model (e.g., OpenAI's 1536 dimensions)
2. You're now using a different embedding model (e.g., nomic-embed-text with 768 dimensions)

## Common Embedding Model Dimensions

| Model | Provider | Dimensions | Use Case |
|-------|----------|------------|----------|
| `text-embedding-ada-002` | OpenAI | 1536 | High quality, expensive |
| `text-embedding-3-small` | OpenAI | 1536 | Newer OpenAI model |
| `nomic-embed-text` | Ollama | 768 | **Default**, good quality |
| `all-minilm` | Ollama | 384 | Fast, lightweight |
| `mxbai-embed-large` | Ollama | 1024 | High quality |

## Quick Solutions

### Solution 1: Use the Fix Script (Recommended)

```bash
# Make script executable
chmod +x fix-dimensions.sh

# Run the fix
./fix-dimensions.sh

# Then run your application
./wikillm-rag
```

### Solution 2: Manual Qdrant Collection Reset

```bash
# Delete the problematic collection
curl -X DELETE http://localhost:6333/collections/wikipedia

# Verify deletion
curl http://localhost:6333/collections

# Run your application (it will recreate with correct dimensions)
./wikillm-rag
```

### Solution 3: Use the Enhanced Version with Auto-Fix

```bash
# Use the updated main.go with better error handling
cp main-fixed.go main.go

# Build and run with auto-fix flag
go build -o wikillm-rag ./main.go
./wikillm-rag --force-recreate
```

### Solution 4: Use a Different Collection Name

```bash
# Use a new collection name
./wikillm-rag -qdrant-collection wikipedia-nomic

# Or with different embedding model
./wikillm-rag -qdrant-collection wikipedia-openai -embedding-provider openai -embedding-model text-embedding-ada-002
```

## Detailed Fix Steps

### Step 1: Check Current Collection

```bash
# Check what collections exist
curl http://localhost:6333/collections

# Get details of your collection
curl http://localhost:6333/collections/wikipedia
```

### Step 2: Identify Your Embedding Model Dimensions

```bash
# Check what model you're trying to use
./wikillm-rag --help

# Test embedding dimensions
go run -c 'package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/ollama"
)

func main() {
    llm, _ := ollama.New(ollama.WithModel("nomic-embed-text"))
    embedder, _ := embeddings.NewEmbedder(llm)
    
    vecs, _ := embedder.EmbedDocuments(context.Background(), []string{"test"})
    fmt.Printf("Dimensions: %d\n", len(vecs[0]))
}'
```

### Step 3: Choose Your Fix Strategy

#### Option A: Keep Current Embedding Model, Reset Collection
```bash
# Delete collection and let app recreate
curl -X DELETE http://localhost:6333/collections/wikipedia
./wikillm-rag
```

#### Option B: Change to Match Existing Collection
If your collection is 1536 dimensions, use OpenAI embeddings:
```bash
./wikillm-rag -embedding-provider openai -embedding-model text-embedding-ada-002
```

If your collection is 768 dimensions, use nomic-embed-text:
```bash
./wikillm-rag -embedding-model nomic-embed-text
```

## Prevention Strategies

### 1. Use Descriptive Collection Names
```bash
# Include embedding model in collection name
./wikillm-rag -qdrant-collection wikipedia-nomic-768
./wikillm-rag -qdrant-collection wikipedia-openai-1536
```

### 2. Document Your Configuration
Create a config file:
```bash
# config.env
EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_DIMENSIONS=768
COLLECTION_NAME=wikipedia-nomic-768
```

### 3. Use the Enhanced Version
The `main-fixed.go` version automatically:
- Detects embedding dimensions
- Checks for mismatches
- Offers to recreate collections
- Provides helpful error messages

## Advanced Troubleshooting

### Check Qdrant Logs
```bash
# If using Docker
docker logs <qdrant-container-id>
```

### Verify Embedding Model is Available
```bash
# For Ollama
ollama list
ollama pull nomic-embed-text

# Test embedding generation
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "test"
}'
```

### Debug Vector Store Connection
```bash
# Test Qdrant connection
curl http://localhost:6333/health

# Check collection details
curl http://localhost:6333/collections/wikipedia | jq .
```

## Common Error Patterns

### Error: "model not found"
```bash
# Solution: Pull the embedding model
ollama pull nomic-embed-text
```

### Error: "connection refused"
```bash
# Solution: Start Qdrant
docker run -p 6333:6333 qdrant/qdrant
```

### Error: "unauthorized"
```bash
# Solution: Check OpenAI API key
export OPENAI_API_KEY="your-key-here"
```

## Best Practices

1. **Consistent Embedding Models**: Stick with one embedding model per collection
2. **Clear Naming**: Use descriptive collection names that include the model
3. **Documentation**: Keep track of which models you're using
4. **Testing**: Always test with small datasets first
5. **Backup**: Export your collections before making changes

## Recovery Procedures

### If You Have Important Data
```bash
# Export collection before deleting
curl "http://localhost:6333/collections/wikipedia/points/scroll" \
  -H "Content-Type: application/json" \
  -d '{"limit": 10000}' > backup.json

# Delete and recreate
curl -X DELETE http://localhost:6333/collections/wikipedia

# Re-index your data
./wikillm-rag -wikipedia ./path/to/wiki.xml
```

### If You Need to Switch Models
```bash
# Create new collection with different model
./wikillm-rag -embedding-model mxbai-embed-large -qdrant-collection wikipedia-mxbai

# Keep old collection as backup
# Test new collection before deleting old one
```

## Getting Help

If you're still having issues:

1. Check the full error message in the logs
2. Verify all services are running (Ollama, Qdrant)
3. Test with a fresh collection name
4. Use the enhanced version with `--force-recreate`
5. Check the GitHub issues for similar problems

## Quick Reference Commands

```bash
# Reset everything
curl -X DELETE http://localhost:6333/collections/wikipedia
./wikillm-rag --force-recreate

# Check status
curl http://localhost:6333/health
curl http://localhost:6333/collections
ollama list

# Test different models
./wikillm-rag -embedding-model nomic-embed-text -qdrant-collection test-nomic
./wikillm-rag -embedding-model all-minilm -qdrant-collection test-minilm
```
