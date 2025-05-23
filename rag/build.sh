#!/bin/bash

# Updated Build script for WikiLLM RAG application with latest langchaingo
# Handles dependency resolution and build process

set -e  # Exit on any error

echo "🔧 Building WikiLLM RAG Application (Updated)"
echo "============================================="

# Clean any existing builds
echo "📦 Cleaning previous builds..."
go clean -cache
rm -f go.sum

# Download and verify dependencies
echo "⬇️  Downloading dependencies..."
go mod download
go mod tidy

# Verify all dependencies are resolved
echo "✅ Verifying dependencies..."
go mod verify

# Check for test files and run them
if ls *_test.go 1> /dev/null 2>&1; then
    echo "🧪 Running tests..."
    go test -v ./...
fi

# Build the application with version info
echo "🏗️  Building application..."
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

go build -v -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o wikillm-rag ./main.go

# Create build info
echo "📋 Build completed successfully!"
echo "Executable: ./wikillm-rag"
echo "Version: $VERSION"
echo "Build time: $BUILD_TIME" 
echo "Go version: $(go version)"

# Display usage information
echo ""
echo "🚀 Usage examples:"
echo "./wikillm-rag -h                                              # Show help"
echo "./wikillm-rag -provider ollama -model llama2                  # Use Ollama"
echo "./wikillm-rag -provider openai -model gpt-3.5-turbo           # Use OpenAI"
echo "./wikillm-rag -provider lmstudio -model default               # Use LM Studio"
echo "./wikillm-rag -wikipedia ./data/simplewiki.xml                # Index Wikipedia"
echo "./wikillm-rag -embedding-provider openai -embedding-model text-embedding-ada-002 # Mixed providers"
