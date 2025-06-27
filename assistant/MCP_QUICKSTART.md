# MCP Integration - Quick Start Guide

## ✨ What is MCP?

Model Context Protocol (MCP) allows your WikiLLM Assistant to connect to external tools, services, and data sources. Think of it as giving your AI assistant superpowers! 🦸‍♂️

## 🚀 Quick Setup

### 1. Run the Setup Script

**Linux/macOS:**
```bash
chmod +x setup_mcp.sh
./setup_mcp.sh
```

**Windows:**
```cmd
setup_mcp.bat
```

### 2. Test the Integration

```bash
# Test basic functionality
python test_mcp.py

# Full debug analysis
python debug_mcp.py
```

### 3. Configure Servers

Edit `mcp_servers.json` to add your servers:

```json
{
  "servers": [
    {
      "server_id": "filesystem-safe",
      "name": "Safe Filesystem Access", 
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/you/safe-directory"],
      "enabled": true
    }
  ]
}
```

### 4. Start the Assistant

```bash
python main.py
```

## 🎛️ Using the Debug Panel

1. **Open the frontend** (http://localhost:3000)
2. **Click the settings icon** in the sidebar
3. **View MCP status** - see connected servers and available tools
4. **Test connections** - verify servers are working
5. **Add new servers** - configure additional MCP servers

## 🔧 Available MCP Servers

### File & Data Access
- **Filesystem**: `@modelcontextprotocol/server-filesystem`
- **SQLite**: `@modelcontextprotocol/server-sqlite` 
- **PostgreSQL**: `@modelcontextprotocol/server-postgres`
- **Google Drive**: Custom server for cloud storage

### Web & APIs
- **Brave Search**: `@modelcontextprotocol/server-brave-search`
- **GitHub**: `@modelcontextprotocol/server-github`
- **HTTP Fetch**: `@modelcontextprotocol/server-fetch`
- **Web Scraping**: `@modelcontextprotocol/server-puppeteer`

### Communication & Productivity
- **Slack**: `@modelcontextprotocol/server-slack`
- **Notion**: `@modelcontextprotocol/server-notion`
- **Memory Bank**: `@modelcontextprotocol/server-memory`

## 💡 Quick Examples

### Enable Filesystem Access
```json
{
  "server_id": "filesystem",
  "name": "Local Files",
  "type": "stdio", 
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/you/Documents"],
  "enabled": true
}
```

Then ask your assistant:
- "What files are in my Documents folder?"
- "Read the contents of report.txt"
- "Create a new file called notes.md with my ideas"

### Enable Web Search
```json
{
  "server_id": "web-search",
  "name": "Web Search",
  "type": "stdio",
  "command": "npx", 
  "args": ["-y", "@modelcontextprotocol/server-brave-search"],
  "enabled": true,
  "env": {
    "BRAVE_API_KEY": "your-api-key"
  }
}
```

Then ask:
- "Search the web for the latest AI news"
- "Find information about Python async programming"
- "What's the weather like today?" 

## 🔍 Debugging Issues

### Common Problems

**"MCP servers not connecting"**
- Check Node.js is installed: `node --version`
- Verify npx works: `npx --version`
- Test server manually: `npx -y @modelcontextprotocol/server-filesystem /tmp`

**"No tools available"**
- Ensure at least one server is `"enabled": true`
- Check server status in debug panel
- Verify server configuration is correct

**"Permission denied"**
- Check filesystem paths are accessible
- Ensure API keys are valid
- Review server-specific requirements

### Getting Help

1. **Check logs**: `tail -f assistant.log`
2. **Run diagnostics**: `python debug_mcp.py`
3. **Test basic setup**: `python test_mcp.py`  
4. **Use debug panel**: Settings icon in frontend
5. **Check MCP docs**: https://modelcontextprotocol.io/

## 🛠️ Development

### Adding Custom Servers

Create your own MCP server:

```python
from mcp.server import Server
from mcp.server.stdio import stdio_server

server = Server("my-custom-server")

@server.tool("my_tool")
def my_tool(param: str) -> str:
    return f"Processed: {param}"

if __name__ == "__main__":
    stdio_server(server)
```

Add to config:
```json
{
  "server_id": "custom",
  "name": "My Custom Server", 
  "type": "stdio",
  "command": "python",
  "args": ["my_server.py"],
  "enabled": true
}
```

### API Integration

The MCP system exposes REST APIs:

- `GET /mcp/status` - System status
- `GET /mcp/servers` - List servers
- `POST /mcp/servers` - Add server  
- `GET /mcp/tools` - List available tools
- `POST /mcp/tools/call` - Call a tool

## 🎯 What's Next?

1. **Configure your first server** using the debug panel
2. **Test tool usage** in conversations
3. **Add more servers** for different capabilities
4. **Build custom tools** for your specific needs
5. **Monitor usage** with analytics

Your AI assistant is now ready to interact with the world! 🌍
