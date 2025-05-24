#!/bin/bash

# Build and test the RAG application

echo "🔨 Building wikillm-rag..."
go build -o wikillm-rag .

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

echo ""
echo "🧪 Testing Qdrant connection..."
./wikillm-rag -test-connection

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉 All tests passed! You can now run:"
    echo "   ./wikillm-rag -wikipedia simplewiki-latest-pages-articles.xml"
else
    echo "❌ Connection test failed. Please check your Qdrant setup."
    exit 1
fi
