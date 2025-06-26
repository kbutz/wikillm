#!/bin/bash

# AI Assistant Startup Script
# This script starts both the backend API and frontend development server

set -e

echo "🤖 AI Assistant Startup Script"
echo "================================"

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required but not installed."
    exit 1
fi

# Check if Node.js is available
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is required but not installed."
    exit 1
fi

# Check if LMStudio is running
echo "🔍 Checking LMStudio connection..."
if curl -s http://localhost:1234/v1/models &> /dev/null; then
    echo "✅ LMStudio is running"
else
    echo "⚠️  LMStudio not detected on localhost:1234"
    echo "   Please ensure LMStudio is running with a model loaded"
    echo "   Continuing anyway - you can start LMStudio later"
fi

# Function to cleanup background processes
cleanup() {
    echo "🧹 Cleaning up..."
    jobs -p | xargs -r kill
    exit
}
trap cleanup SIGINT SIGTERM

# Start backend
echo "🚀 Starting backend server..."
cd "$(dirname "$0")"

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "📦 Creating Python virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install Python dependencies
if [ ! -f "venv/.dependencies_installed" ]; then
    echo "📦 Installing Python dependencies..."
    pip install -r requirements.txt
    touch venv/.dependencies_installed
fi

# Initialize database if it doesn't exist
if [ ! -f "assistant.db" ]; then
    echo "🗄️  Initializing database..."
    python -c "from database import init_database; init_database()"
fi

# Start backend in background
echo "🖥️  Starting FastAPI server on http://localhost:8000"
python main.py &
BACKEND_PID=$!

# Wait for backend to start
echo "⏳ Waiting for backend to start..."
sleep 3

# Check if backend started successfully
if kill -0 $BACKEND_PID 2>/dev/null; then
    echo "✅ Backend server started successfully"
else
    echo "❌ Backend server failed to start"
    exit 1
fi

# Start frontend
echo "🎨 Starting frontend..."
cd frontend

# Install Node.js dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing Node.js dependencies..."
    npm install
fi

# Start frontend in background
echo "🌐 Starting React development server on http://localhost:3000"
npm start &
FRONTEND_PID=$!

# Wait for frontend to start
sleep 3

echo ""
echo "🎉 AI Assistant is now running!"
echo "================================"
echo "Frontend: http://localhost:3000"
echo "Backend API: http://localhost:8000"
echo "API Docs: http://localhost:8000/docs"
echo ""
echo "Press Ctrl+C to stop all services"

# Wait for user to stop
wait
