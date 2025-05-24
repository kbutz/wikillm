# WikiLLM RAG

A Retrieval-Augmented Generation (RAG) system for querying Wikipedia content using multiple LLM providers locally.

## Features

### Multi-Provider Support
- **Ollama**: Local LLM inference with models like Llama2, Mistral, etc.
- **OpenAI**: GPT models via OpenAI API
- **LM Studio**: Local OpenAI-compatible server

### Advanced Embedding Options
- Separate embedding providers from LLM providers
- Support for specialized embedding models
- Automatic embedding generation for vector storage

### Modern langchaingo Integration
- Uses latest v0.1.24+ APIs
- Improved error handling and context management
- Schema-based document processing
- Enhanced vector store operations

## Quick Start

### Prerequisites

1. **Go 1.21+** installed
2. **Qdrant** vector database running:
   ```bash
   docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
   ```

3. **Choose your LLM provider**:

   **Option A: Ollama (Recommended for local use)**
   ```bash
   # Install Ollama
   curl -fsSL https://ollama.com/install.sh | sh

   # Pull models
   ollama pull llama2
   ollama pull nomic-embed-text
   ```

   **Option B: OpenAI**
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

   **Option C: LM Studio**
   ```bash
   # Download and start LM Studio server on localhost:1234
   # Set a dummy API key for compatibility
   export OPENAI_API_KEY="lm-studio"
   ```

### Getting Wikipedia Data via Dump

You'll need a Wikipedia dump file to create the index. You can download one from:
https://dumps.wikimedia.org/

For testing, you might want to start with a smaller Wikipedia dump, such as Simple English Wikipedia:
https://dumps.wikimedia.org/simplewiki/latest/simplewiki-latest-pages-articles.xml.bz2

After downloading, extract the bz2 file:

```bash
bunzip2 simplewiki-latest-pages-articles.xml.bz2
```

### Getting Wikipedia Data via Embeddings

https://huggingface.co/datasets/Supabase/wikipedia-en-embeddings/blob/main/wiki_gte.ndjson.gz

### Build and Run

```bash
# Build the application
go mod download
go mod tidy
go build -o wikillm-rag .

# For a clean build (optional)
go clean -cache
rm -f go.sum
go mod download
go mod tidy
go build -o wikillm-rag .

# Run with Ollama (default - uses llama3.2)
./wikillm-rag

# Run with specific Ollama model
./wikillm-rag -provider ollama -model llama3.1

# Run with OpenAI
./wikillm-rag -provider openai -model gpt-3.5-turbo

# Index Wikipedia data (make sure you've pulled the embedding model first)
# For Ollama, ensure you've run: ollama pull nomic-embed-text
./wikillm-rag -wikipedia ./path/to/simplewiki.xml

# Load pre-indexed embeddings from a file with the following. The file is hard coded right now.
./wikillm-rag -load

# If you loaded the minilm embeddings from the example here and want to use lmstudio, you also need to specify the embedding provider:
./wikillm-rag -provider lmstudio -embedding-provider all-minlm
```

### Running with LM Studio

TODO: LM Studio doesn't work with the all-minilm embedding model right now, so need to figure that out still

1. Download and install [LM Studio](https://lmstudio.ai/) for your platform
2. Launch LM Studio and download a model of your choice
3. Start the local server in LM Studio:
   - Click on "Local Server" in the sidebar
   - Select your model from the dropdown
   - Click "Start Server"
   - The server will run on http://localhost:1234 by default

4. Build the application:
   ```bash
   go mod download
   go mod tidy
   go build -o wikillm-rag .
   ```
5. Run with LM Studio as the provider:
```bash
# Set a dummy API key for compatibility (can be any value)
export OPENAI_API_KEY="lm-studio"

# Run the application with LM Studio provider
./wikillm-rag -provider lmstudio -model default
```

Note: The `-model default` parameter is used because the model selection is handled within LM Studio itself.

## Configuration Options

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-provider` | LLM provider (ollama/openai/lmstudio) | ollama |
| `-model` | Model name | llama2 |
| `-embedding-provider` | Separate embedding provider | (same as provider) |
| `-embedding-model` | Embedding model name | nomic-embed-text |
| `-wikipedia` | Path to Wikipedia XML dump | |
| `-qdrant-url` | Qdrant server URL | http://localhost:6333 |
| `-qdrant-collection` | Collection name | wikipedia |
| `-limit` | Search result limit | 5 |
| `-openai-key` | OpenAI API key | (from env) |
| `-ollama-url` | Ollama server URL | http://localhost:11434 |

### Environment Variables

```bash
# OpenAI API Key
export OPENAI_API_KEY="your-key-here"

# Optional: Override default URLs
export QDRANT_URL="http://localhost:6333"
export OLLAMA_URL="http://localhost:11434"
```

## Advanced Usage Examples

### Mixed Providers
Use different providers for LLM and embeddings:
```bash
# Use OpenAI for chat, Ollama for embeddings
./wikillm-rag \
    -provider openai \
    -model gpt-4 \
    -embedding-provider ollama \
    -embedding-model nomic-embed-text
```

### High-Performance Setup
```bash
# Use optimized settings for production
./wikillm-rag \
    -provider openai \
    -model gpt-3.5-turbo \
    -embedding-model text-embedding-ada-002 \
    -limit 10 \
    -qdrant-url http://your-qdrant-cluster:6333
```

## Troubleshooting

### Common Issues

1. **Import Errors**: Run `go mod tidy` to resolve dependencies
2. **Connection Errors**: Verify Qdrant and LLM services are running
3. **API Key Issues**: Check environment variables and permissions
4. **Memory Issues**: Reduce batch size for large Wikipedia dumps
5. **Model Not Found Error**: If you see `model "nomic-embed-text" not found`, run `ollama pull nomic-embed-text` to download the embedding model
6. **Collection Doesn't Exist Error**: If you see `Collection 'wikipedia' doesn't exist`, make sure Qdrant is running and accessible at the URL specified by `-qdrant-url` (default: http://localhost:6333). The application will attempt to create the collection automatically when indexing Wikipedia data.
7. **Undefined Symbol Errors**: If you see errors like `undefined: GetProvider` or `undefined: RAGPipeline`, make sure you're building the entire package, not just main.go. Use `go build -o wikillm-rag .` or `go run .` instead of `go build -o wikillm-rag ./main.go` or `go run main.go`.
