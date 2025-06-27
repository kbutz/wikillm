@echo off
REM MCP Integration Setup and Test Script for Windows

echo 🚀 WikiLLM Assistant MCP Integration Setup
echo ========================================

REM Check if we're in the right directory
if not exist "main.py" (
    echo ❌ Error: Please run this script from the assistant directory
    pause
    exit /b 1
)

echo 🔍 Checking prerequisites...

REM Check Python
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Python not found
    pause
    exit /b 1
) else (
    echo ✅ Python found
)

REM Check Node.js
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠️  Node.js not found - download from https://nodejs.org/
) else (
    echo ✅ Node.js found
)

REM Check npx
npx --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠️  npx not found - needed for MCP servers
) else (
    echo ✅ npx found
)

REM Install Python dependencies
echo 📦 Installing Python dependencies...
if exist "requirements.txt" (
    pip install -r requirements.txt
    echo ✅ Python dependencies installed
) else (
    echo ❌ requirements.txt not found
    pause
    exit /b 1
)

REM Test MCP basic functionality
echo 🧪 Testing MCP integration...
python test_mcp.py

REM Ask user if they want to start the assistant
echo.
set /p choice="🚀 Start the assistant now? (y/n): "
if /i "%choice%"=="y" (
    echo 🎯 Starting WikiLLM Assistant with MCP integration...
    echo    - API will be available at: http://localhost:8000
    echo    - Frontend (if running): http://localhost:3000
    echo    - Use the debug panel in the frontend to manage MCP servers
    echo.
    python main.py
) else (
    echo 📋 To start the assistant later, run: python main.py
    echo 📋 To test MCP integration, run: python debug_mcp.py
    echo 📋 To configure MCP servers, edit: mcp_servers.json
    pause
)
