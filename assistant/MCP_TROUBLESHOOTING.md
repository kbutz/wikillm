# 🔧 MCP Troubleshooting Guide

## Issue: "Server is disabled" Error

### Problem
Getting a 500 error with warning: "Server filesystem-example is disabled"

### Solutions

#### 1. **Check Configuration File**
```bash
# Verify the server is enabled in mcp_servers.json
cat mcp_servers.json | grep -A 10 "filesystem-example"
```

Look for: `"enabled": true`

#### 2. **Reload Configuration**
```bash
# Method 1: Use the API
curl -X POST http://localhost:8000/mcp/reload

# Method 2: Use the debug panel
# Click the "Settings" button (next to refresh) in the debug panel
```

#### 3. **Test the Server Manually**
```bash
python test_filesystem_server.py
```

#### 4. **Restart the Assistant**
```bash
# Stop the assistant (Ctrl+C)
# Start again
python main.py
```

## Issue: "Command not found: npx"

### Problem
MCP servers require Node.js and npx to run

### Solutions

#### Install Node.js
```bash
# macOS
brew install node

# Ubuntu/Debian
sudo apt update
sudo apt install nodejs npm

# Windows
# Download from https://nodejs.org/
```

#### Verify Installation
```bash
node --version
npm --version
npx --version
```

## Issue: "No response from MCP server"

### Problem
Server starts but doesn't respond to requests

### Solutions

#### 1. **Test Server Manually**
```bash
# Test the filesystem server directly
npx -y @modelcontextprotocol/server-filesystem /tmp
```

#### 2. **Check Directory Permissions**
```bash
# Ensure the directory exists and is accessible
ls -la /Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant/tmp
```

#### 3. **Use a Different Directory**
```bash
# Try with /tmp (universally accessible)
npx -y @modelcontextprotocol/server-filesystem /tmp
```

## Issue: "Failed to connect to MCP server"

### Problem
Connection attempt fails with various errors

### Solutions

#### 1. **Check Logs**
```bash
tail -f assistant.log
```

#### 2. **Test Basic Connectivity**
```bash
python debug_mcp.py
```

#### 3. **Verify Server Configuration**
```json
{
  "server_id": "filesystem-test",
  "name": "Test Filesystem",
  "type": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
  "enabled": true,
  "timeout": 30
}
```

## Issue: "Frontend Debug Panel Blank"

### Problem
Debug panel loads but shows no data

### Solutions

#### 1. **Check API Connection**
```bash
# Test the API directly
curl http://localhost:8000/mcp/status
```

#### 2. **Check Browser Console**
- Open Developer Tools (F12)
- Look for JavaScript errors
- Check Network tab for failed requests

#### 3. **Verify CORS Settings**
Ensure the API allows frontend connections

## Issue: "Tools Not Available in Chat"

### Problem
MCP tools don't appear in conversations

### Solutions

#### 1. **Verify Tools are Listed**
```bash
curl http://localhost:8000/mcp/tools
```

#### 2. **Check Server Status**
```bash
curl http://localhost:8000/mcp/servers
```

#### 3. **Test Tool Calling**
```bash
curl -X POST http://localhost:8000/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{"tool_name": "read_file", "arguments": {"path": "/tmp/test.txt"}}'
```

## Issue: "Permission Denied"

### Problem
Server can't access files or directories

### Solutions

#### 1. **Check Directory Permissions**
```bash
# Make directory readable
chmod 755 /path/to/directory

# Make files readable
chmod 644 /path/to/directory/*
```

#### 2. **Use Safe Directories**
```bash
# Use universally accessible directories
/tmp
/Users/username/Documents
/home/username
```

#### 3. **Test with Different User**
```bash
# Create a test directory
mkdir ~/mcp-test
echo "Hello MCP" > ~/mcp-test/test.txt
chmod 755 ~/mcp-test
```

## Quick Diagnostic Commands

### 1. **System Check**
```bash
# Check prerequisites
python test_mcp.py

# Full system analysis
python debug_mcp.py

# Test specific server
python test_filesystem_server.py
```

### 2. **API Tests**
```bash
# System status
curl http://localhost:8000/status

# MCP status
curl http://localhost:8000/mcp/status

# List servers
curl http://localhost:8000/mcp/servers

# List tools
curl http://localhost:8000/mcp/tools
```

### 3. **Configuration Tests**
```bash
# Validate JSON syntax
python -m json.tool mcp_servers.json

# Reload configuration
curl -X POST http://localhost:8000/mcp/reload
```

## Common Fixes

### Fix 1: Enable the Server
```bash
# Edit mcp_servers.json
sed -i 's/"enabled": false/"enabled": true/' mcp_servers.json

# Reload
curl -X POST http://localhost:8000/mcp/reload
```

### Fix 2: Use Safe Directory
```bash
# Update configuration to use /tmp
sed -i 's|/path/to/allowed/directory|/tmp|' mcp_servers.json
```

### Fix 3: Install Dependencies
```bash
# Install Node.js (macOS)
brew install node

# Install Node.js (Ubuntu)
sudo apt install nodejs npm

# Verify MCP server
npx -y @modelcontextprotocol/server-filesystem /tmp
```

## Still Having Issues?

1. **Check the assistant.log** for detailed error messages
2. **Run the diagnostic script**: `python debug_mcp.py`
3. **Test manually**: `python test_filesystem_server.py`
4. **Use the debug panel** for real-time status
5. **Try a minimal configuration** first

The MCP integration is designed to be robust and provide clear error messages to help with troubleshooting.
