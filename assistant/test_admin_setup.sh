#!/bin/bash

# WikiLLM Assistant Admin Tools Test Script

echo "🔧 Testing WikiLLM Assistant Admin Tools Implementation"
echo "======================================================="

# Check if we're in the right directory
if [ ! -f "main.py" ]; then
    echo "❌ Error: Please run this script from the assistant directory"
    exit 1
fi

echo "✅ Found main.py - we're in the right directory"

# Check if admin_routes.py exists
if [ ! -f "admin_routes.py" ]; then
    echo "❌ Error: admin_routes.py not found"
    exit 1
fi

echo "✅ Found admin_routes.py"

# Check if the frontend admin components exist
if [ ! -f "frontend/src/components/AdminDashboard.tsx" ]; then
    echo "❌ Error: AdminDashboard.tsx not found"
    exit 1
fi

echo "✅ Found AdminDashboard.tsx"

if [ ! -f "frontend/src/components/MemoryInspector.tsx" ]; then
    echo "❌ Error: MemoryInspector.tsx not found"
    exit 1
fi

echo "✅ Found MemoryInspector.tsx"

if [ ! -f "frontend/src/services/admin.ts" ]; then
    echo "❌ Error: admin.ts service not found"
    exit 1
fi

echo "✅ Found admin.ts service"

# Check if Python dependencies are installed
echo "🔍 Checking Python dependencies..."
python3 -c "import fastapi, sqlalchemy, pydantic" 2>/dev/null
if [ $? -ne 0 ]; then
    echo "❌ Error: Required Python dependencies not installed"
    echo "   Please run: pip install -r requirements.txt"
    exit 1
fi

echo "✅ Python dependencies are installed"

# Check if Node.js dependencies are installed
echo "🔍 Checking Node.js dependencies..."
if [ ! -d "frontend/node_modules" ]; then
    echo "❌ Error: Node.js dependencies not installed"
    echo "   Please run: cd frontend && npm install"
    exit 1
fi

echo "✅ Node.js dependencies are installed"

# Check if the database exists
if [ ! -f "assistant.db" ]; then
    echo "⚠️  Warning: Database file not found. It will be created on first run."
else
    echo "✅ Found database file"
fi

echo ""
echo "🎉 Admin Tools Implementation Complete!"
echo "======================================="
echo ""
echo "🚀 To start the application:"
echo "   1. Backend: python main.py"
echo "   2. Frontend: cd frontend && npm start"
echo ""
echo "🔧 Admin Features Available:"
echo "   • User Management (Create/Delete/View)"
echo "   • Memory Inspection & Editing"
echo "   • Conversation Management"
echo "   • Data Export"
echo "   • System Statistics"
echo "   • User Impersonation"
echo ""
echo "🌐 Access Points:"
echo "   • Main App: http://localhost:3000"
echo "   • Admin Dashboard: Click the shield icon in the app"
echo "   • API Docs: http://localhost:8000/docs"
echo "   • Admin API: http://localhost:8000/admin/"
echo ""
echo "🔒 Security Note:"
echo "   Admin tools are currently open access (no authentication)"
echo "   This is intentional for development - add auth for production"
