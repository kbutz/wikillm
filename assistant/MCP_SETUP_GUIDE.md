# MCP Integration Setup Guide for WikiLLM Assistant

## Overview

This guide explains how to set up and configure Model Context Protocol (MCP) integration with the WikiLLM Assistant. MCP allows the assistant to connect to external tools, services, and data sources through standardized protocols.

## Installation

### 1. Update Python Dependencies

```bash
# Navigate to your assistant directory
cd /path/to/wikillm/assistant

# Install new requirements
pip install -r requirements.txt

# Or install individual MCP-related packages
pip install aiofiles jsonschema
```

### 2. Install Node.js (Required for MCP Servers)

Most MCP servers are built in Node.js. Install Node.js and npm:

```bash
# On macOS
brew install node

# On Ubuntu/Debian
sudo apt install nodejs npm

# On Windows
# Download from https://nodejs.org/
```

### 3. Verify Installation

```bash
# Check Node.js and npm versions
node --version
npm --version

# Test MCP server installation
npx -y @modelcontextprotocol/server-filesystem --help
```

## Configuration

### 1. MCP Server Configuration

The system automatically creates `mcp_servers.json` with example configurations. Edit this file to configure your servers:

```json
{
  "version": "1.0.0",
  "servers": [
    {
      "server_id": "filesystem",
      "name": "Filesystem Access",
      "description": "Local file operations",
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/safe/directory/path"],
      "enabled": true,
      "timeout": 30
    }
  ]
}
```

### 2. Environment Variables

Add any required API keys to your `.env` file:

```bash
# Example API keys for various MCP servers
BRAVE_API_KEY=your_brave_search_api_key
GITHUB_PERSONAL_ACCESS_TOKEN=your_github_token
NOTION_API_KEY=your_notion_api_key
SLACK_BOT_TOKEN=your_slack_bot_token
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

## Quick Start Examples

### 1. Enable Filesystem Access

Edit `mcp_servers.json`:

```json
{
  "server_id": "filesystem",
  "name": "File System",
  "type": "stdio",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users/yourname/Documents"],
  "enabled": true
}
```

### 2. Enable Web Search

```json
{
  "server_id": "web-search",
  "name": "Web Search",
  "type": "stdio", 
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-brave-search"],
  "enabled": true,
  "env": {
    "BRAVE_API_KEY": "your-api-key-here"
  }
}
```

### 3. Start the Assistant

```bash
python main.py
```

The assistant will now have access to the configured MCP tools!

## Available MCP Servers

### File & Data Access
- **Filesystem**: Local file operations
- **SQLite**: Database queries
- **PostgreSQL**: Database operations
- **Google Drive**: Cloud file access
- **Memory**: Persistent storage

### Web & APIs
- **Brave Search**: Web search
- **GitHub**: Repository access
- **Fetch**: HTTP requests
- **Puppeteer**: Web scraping

### Communication
- **Slack**: Team messaging
- **Notion**: Documentation

## API Endpoints

### Server Management
- `GET /mcp/status` - Server status
- `POST /mcp/servers` - Add server
- `PUT /mcp/servers/{id}` - Update server
- `DELETE /mcp/servers/{id}` - Remove server

### Tools & Resources
- `GET /mcp/tools` - List available tools
- `POST /mcp/tools/call` - Call a tool
- `GET /mcp/resources` - List resources
- `POST /mcp/resources/read` - Read resource

### Chat Integration
- `/chat` - Enhanced chat with MCP tools
- `/chat/stream` - Streaming chat with tools

## Usage Examples

### 1. Chat with File Access

```bash
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "message": "Please read the file /Users/me/Documents/report.txt and summarize it"
  }'
```

### 2. Web Search Query

```bash
curl -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "message": "Search the web for the latest AI news and give me a summary"
  }'
```

### 3. Direct Tool Call

```bash
curl -X POST http://localhost:8000/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "read_file",
    "arguments": {"path": "/Users/me/Documents/data.txt"}
  }'
```

## Security Best Practices

### 1. Filesystem Security
- Only allow access to safe directories
- Use absolute paths in configuration
- Regularly audit allowed directories

```json
{
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/safe/sandbox/directory"]
}
```

### 2. API Key Management
- Store keys in `.env` files
- Never commit keys to version control
- Use environment-specific keys
- Implement key rotation

### 3. Network Security
- Configure firewalls for HTTP servers
- Use HTTPS where possible
- Implement rate limiting
- Monitor connection attempts

## Troubleshooting

### Common Issues

**Server won't connect:**
```bash
# Check if Node.js is installed
node --version

# Test server manually
npx -y @modelcontextprotocol/server-filesystem /tmp

# Check logs
tail -f assistant.log
```

**Permission errors:**
```bash
# Check directory permissions
ls -la /path/to/directory

# Fix permissions if needed
chmod 755 /path/to/directory
```

**Tool call failures:**
- Verify tool parameters match schema
- Check server logs for errors
- Ensure environment variables are set
- Test tools individually via API

### Debug Mode

Enable debug logging by setting:

```bash
export PYTHONPATH="."
export LOG_LEVEL="DEBUG"
python main.py
```

## Advanced Configuration

### Custom MCP Server

Create your own MCP server:

```python
# custom_server.py
from mcp.server import Server
from mcp.server.stdio import stdio_server

server = Server("my-custom-server")

@server.tool("my_tool")
def my_tool(param: str) -> str:
    return f"Processed: {param}"

if __name__ == "__main__":
    stdio_server(server)
```

Add to configuration:

```json
{
  "server_id": "custom",
  "name": "My Custom Server",
  "type": "stdio",
  "command": "python",
  "args": ["custom_server.py"],
  "enabled": true
}
```

### Load Balancing

For high-traffic scenarios:

```json
{
  "global_settings": {
    "max_concurrent_connections": 20,
    "connection_retry_attempts": 5,
    "health_check_interval": 60
  }
}
```

## Support

- **MCP Specification**: https://spec.modelcontextprotocol.io/
- **Official Servers**: https://github.com/modelcontextprotocol/servers
- **Documentation**: https://docs.modelcontextprotocol.io/
- **Community**: https://github.com/modelcontextprotocol/

## Next Steps

1. **Start Simple**: Enable filesystem access first
2. **Add Web Search**: Configure Brave API for web capabilities  
3. **Database Integration**: Connect to your databases
4. **Custom Tools**: Build domain-specific MCP servers
5. **Monitor Usage**: Track tool performance and usage patterns

The MCP integration transforms your WikiLLM Assistant into a powerful, extensible platform that can connect to virtually any external service while maintaining security and performance.
