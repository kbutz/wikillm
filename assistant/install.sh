#!/bin/bash

# AI Assistant Installation Script
# This script sets up the complete AI Assistant environment

set -e

echo "🤖 AI Assistant Installation"
echo "============================"

# Check system requirements
echo "🔍 Checking system requirements..."

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3.11+ is required but not installed."
    echo "   Please install Python 3.11 or later from https://python.org"
    exit 1
fi

PYTHON_VERSION=$(python3 -c 'import sys; print(".".join(map(str, sys.version_info[:2])))')
echo "✅ Python $PYTHON_VERSION found"

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is required but not installed."
    echo "   Please install Node.js 18+ from https://nodejs.org"
    exit 1
fi

NODE_VERSION=$(node --version)
echo "✅ Node.js $NODE_VERSION found"

# Check npm
if ! command -v npm &> /dev/null; then
    echo "❌ npm is required but not installed."
    exit 1
fi

echo "✅ npm $(npm --version) found"

# Setup backend
echo ""
echo "🐍 Setting up Python backend..."

# Create virtual environment
if [ ! -d "venv" ]; then
    echo "📦 Creating Python virtual environment..."
    python3 -m venv venv
else
    echo "✅ Virtual environment already exists"
fi

# Activate virtual environment
echo "🔄 Activating virtual environment..."
source venv/bin/activate

# Install Python dependencies
echo "📦 Installing Python dependencies..."
pip install --upgrade pip
pip install -r requirements.txt

# Copy environment file
if [ ! -f ".env" ]; then
    echo "📝 Creating environment configuration..."
    cp .env.example .env
    echo "✅ Created .env file - please review and customize as needed"
else
    echo "✅ Environment file already exists"
fi

# Initialize database
echo "🗄️  Initializing database..."
python -c "from database import init_database; init_database()"
echo "✅ Database initialized"

# Setup frontend
echo ""
echo "🎨 Setting up React frontend..."
cd frontend

# Install Node.js dependencies
echo "📦 Installing Node.js dependencies..."
npm install

echo "✅ Frontend dependencies installed"

# Return to root directory
cd ..

# Create sample data
echo ""
echo "📊 Creating sample data..."
python dev_utils.py sample_data

# Make startup script executable
chmod +x start.sh

echo ""
echo "🎉 Installation complete!"
echo "========================"
echo ""
echo "📋 Next steps:"
echo "1. Review and customize .env file if needed"
echo "2. Install and start LMStudio with a model loaded on port 1234"
echo "3. Run ./start.sh to start both backend and frontend"
echo ""
echo "📚 Documentation:"
echo "- API docs will be available at: http://localhost:8000/docs"
echo "- Frontend will be available at: http://localhost:3000"
echo "- See README.md for detailed usage instructions"
echo ""
echo "🤝 Need help?"
echo "- Check the README.md file"
echo "- Review the API documentation"
echo "- Ensure LMStudio is running before starting the application"
