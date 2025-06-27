# 🧪 MCP Integration Testing Checklist

## ✅ Pre-Testing Setup

1. **Make scripts executable:**
   ```bash
   python make_executable.py
   ```

2. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

3. **Install Node.js** (if not already installed):
   - macOS: `brew install node`
   - Ubuntu: `sudo apt install nodejs npm`
   - Windows: Download from https://nodejs.org/

## 🔍 Step-by-Step Testing

### 1. Basic Integration Test
```bash
python test_mcp.py
```
**Expected:** ✅ All imports work, configuration loads, basic functionality confirmed

### 2. Comprehensive Debug Test  
```bash
python debug_mcp.py
```
**Expected:** 📊 Detailed system analysis, server status, recommendations

### 3. Start the Assistant
```bash
python main.py
```
**Expected:** 🚀 Server starts without errors on http://localhost:8000

### 4. Frontend Compilation
```bash
cd frontend
npm start
```
**Expected:** ✅ No compilation errors, frontend loads on http://localhost:3000

### 5. Access Debug Panel
1. Open frontend (http://localhost:3000)
2. Click settings icon in sidebar
3. Navigate through MCP debug tabs
**Expected:** 🎛️ All tabs load, server status visible

### 6. Test Server Configuration
1. In debug panel, go to "Add Server" tab
2. Add a test filesystem server:
   ```json
   Server ID: filesystem-test
   Name: Test Filesystem
   Type: stdio
   Command: npx
   Args: -y, @modelcontextprotocol/server-filesystem, /tmp
   Enable: true
   ```
3. Click "Add Server"
**Expected:** ✅ Server added successfully

### 7. Test Server Connection
1. Go to "Servers" tab
2. Click "Test Connection" on the test server
**Expected:** 🔌 Connection test completes (may fail if npx/node not available)

### 8. Check Available Tools
1. Go to "Tools" tab
2. View available MCP tools
**Expected:** 🔧 Tools list shows (empty if no servers connected)

### 9. Test Chat Integration
1. Start a new conversation
2. Ask: "What MCP tools do you have available?"
**Expected:** 🤖 Assistant responds with available tools info

## 🐛 Common Issues & Solutions

### "Import Error" in test_mcp.py
**Solution:** Ensure all MCP files are in the assistant directory

### "Node.js not found"
**Solution:** Install Node.js, then test with `node --version`

### "npx not found" 
**Solution:** Install npm: `npm install -g npm`

### "MCP servers not connecting"
**Solution:** 
1. Check server configuration in mcp_servers.json
2. Test MCP server manually: `npx -y @modelcontextprotocol/server-filesystem /tmp`
3. Check assistant.log for errors

### "Frontend won't compile"
**Solution:** Check that api.ts doesn't use reserved words like `arguments`

### "Debug panel blank"
**Solution:** Check browser console for errors, ensure API is running

## 📊 Success Indicators

- [ ] ✅ test_mcp.py runs without errors
- [ ] 📊 debug_mcp.py shows system status  
- [ ] 🚀 python main.py starts successfully
- [ ] 🎛️ Frontend debug panel loads
- [ ] 🔧 Can add/configure MCP servers
- [ ] 🤖 Assistant knows about MCP tools
- [ ] 📈 Tool usage appears in analytics

## 🎯 Next Steps After Testing

1. **Configure Production Servers:**
   - Edit mcp_servers.json with real server configurations
   - Add API keys to .env file
   - Enable servers by setting "enabled": true

2. **Test Real Tools:**
   - Try filesystem operations: "List files in my directory"
   - Test web search: "Search for recent AI news"
   - Use database tools: "Query my database"

3. **Monitor Performance:**
   - Check tool usage analytics
   - Monitor assistant.log for errors
   - Use debug panel for real-time status

4. **Scale Up:**
   - Add more MCP servers
   - Build custom tools
   - Integrate with your workflow

## 🆘 Getting Help

If tests fail:
1. Check assistant.log for detailed errors
2. Run debug_mcp.py for comprehensive analysis  
3. Use frontend debug panel for real-time status
4. Review MCP_SETUP_GUIDE.md for detailed setup

The system is designed to gracefully handle missing servers and provide clear error messages to guide troubleshooting.
