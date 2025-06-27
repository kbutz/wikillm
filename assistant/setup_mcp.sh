#!/bin/bash

# MCP Integration Setup and Test Script

echo "🚀 WikiLLM Assistant MCP Integration Setup"
echo "========================================"

# Check if we're in the right directory
if [ ! -f "main.py" ]; then
    echo "❌ Error: Please run this script from the assistant directory"
    exit 1
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check Python
if command_exists python3; then
    echo "✅ Python3 found: $(python3 --version)"
else
    echo "❌ Python3 not found"
    exit 1
fi

# Check Node.js
if command_exists node; then
    echo "✅ Node.js found: $(node --version)"
else
    echo "⚠️  Node.js not found - installing via package manager recommended"
    echo "   macOS: brew install node"
    echo "   Ubuntu/Debian: sudo apt install nodejs npm"
    echo "   Windows: Download from https://nodejs.org/"
fi

# Check npm/npx
if command_exists npx; then
    echo "✅ npx found"
else
    echo "⚠️  npx not found - needed for MCP servers"
fi

# Install Python dependencies
echo "📦 Installing Python dependencies..."
if [ -f "requirements.txt" ]; then
    pip install -r requirements.txt
    echo "✅ Python dependencies installed"
else
    echo "❌ requirements.txt not found"
    exit 1
fi

# Test MCP basic functionality
echo "🧪 Testing MCP integration..."
python3 test_mcp.py

# Ask user if they want to start the assistant
echo ""
read -p "🚀 Start the assistant now? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🎯 Starting WikiLLM Assistant with MCP integration..."
    echo "   - API will be available at: http://localhost:8000"
    echo "   - Frontend (if running): http://localhost:3000"
    echo "   - Use the debug panel in the frontend to manage MCP servers"
    echo ""
    python3 main.py
else
    echo "📋 To start the assistant later, run: python3 main.py"
    echo "📋 To test MCP integration, run: python3 debug_mcp.py"
    echo "📋 To configure MCP servers, edit: mcp_servers.json"
fi
