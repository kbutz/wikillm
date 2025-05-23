#!/bin/bash

# Quick fix script for Qdrant dimension mismatch
# This script deletes the existing collection and allows the app to recreate it

QDRANT_URL="http://localhost:6333"
COLLECTION_NAME="wikipedia"

echo "🔧 Fixing Qdrant dimension mismatch..."

# Check if Qdrant is running
if ! curl -s "$QDRANT_URL/health" > /dev/null; then
    echo "❌ Qdrant is not running at $QDRANT_URL"
    echo "   Start it with: docker run -p 6333:6333 qdrant/qdrant"
    exit 1
fi

echo "📋 Current collections:"
curl -s "$QDRANT_URL/collections" | jq '.' 2>/dev/null || echo "Install jq for pretty output"

echo ""
echo "🗑️  Deleting collection '$COLLECTION_NAME'..."
response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$QDRANT_URL/collections/$COLLECTION_NAME")

if [ "$response" = "200" ] || [ "$response" = "404" ]; then
    echo "✅ Collection deleted (or didn't exist)"
else
    echo "⚠️  Unexpected response: $response"
fi

echo ""
echo "🎉 Fixed! The application will now create a new collection with correct dimensions."
echo ""
echo "Next steps:"
echo "1. Run your application: ./wikillm-rag"
echo "2. If indexing Wikipedia: ./wikillm-rag -wikipedia ./path/to/wiki.xml"
