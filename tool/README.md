# Enhanced LLM Agent To-Do List

This is a command-line and HTTP-based to-do list application powered by an LLM Agent. The agent can interact with a JSON file to manage your to-do list with advanced analytical capabilities, enabling complex queries about task priorities, time estimates, and comprehensive summaries.

## Features

- Command-line interface for interactive use
- HTTP API for integration with other applications
- Works offline using local LLM models (LM Studio or Ollama)
- Extensible tool framework for adding new capabilities
- **Advanced task analysis**: Priority analysis, time estimates, and comprehensive summaries
- **Intelligent query processing**: Distinguishes between direct commands and analytical queries
- **Structured data export**: JSON export for complex LLM analysis

## Prerequisites

- Go 1.16 or higher
- One of the following local LLM servers:
  - [LM Studio](https://lmstudio.ai/) (default)
  - [Ollama](https://ollama.ai/)

## Installation

```bash
go build -o todo-agent
```

## Usage

### Command-line Interface

```bash
# Using default settings (LM Studio, default model, no HTTP server)
./todo-agent

# Using Ollama with a specific model
./todo-agent -provider ollama -model llama2

# Start with HTTP server on port 8080
./todo-agent -port 8080

# Specify a custom to-do list file
./todo-agent -todo-file /path/to/my-todos.txt
```

### Command-line Options

- `-model`: Name of the LLM model to use (default: "default")
- `-provider`: Model provider to use (lmstudio or ollama) (default: "lmstudio")
- `-port`: HTTP server port (0 to disable) (default: 0)
- `-todo-file`: Path to the to-do list file (default: "todo.txt")

### HTTP API

When the HTTP server is enabled with the `-port` flag, you can interact with the agent using HTTP requests:

```bash
# Example: Send a query to the agent
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "Add buy milk to my to-do list"}'
```

## Interacting with the To-Do List

The agent understands natural language commands for managing your to-do list. Here are some examples:

### Basic Task Management
- "Add buy groceries to my to-do list with high priority and 30 minutes"
- "Show me my to-do list"
- "Complete task 3"
- "Remove item 2 from my to-do list"
- "Clear my completed tasks"

### Advanced Analytical Queries
- **"What is my most important task today?"** - Uses priority analysis to identify critical tasks
- **"Give me a summary of all my tasks"** - Provides comprehensive overview with statistics
- **"What tasks can I complete quickly?"** - Analyzes time estimates to find quick wins
- **"Which tasks should I prioritize?"** - Exports data for complex multi-factor analysis

### Behind the Scenes Commands

The `todo_list` tool supports these commands:

#### Task Management
- `add <task> [priority:low/medium/high/critical] [time:XXm/XXh]`: Adds a task with optional priority and time
- `list`: Shows all active tasks
- `list all`: Shows all tasks including completed
- `list priority`: Shows tasks sorted by priority
- `complete <number>`: Marks a task as completed
- `remove <number>`: Removes a task
- `clear`: Removes all tasks
- `clear completed`: Removes only completed tasks

#### Analytical Commands (NEW)
- `export`: Exports all task data as JSON for LLM analysis
- `analyze priority`: Provides detailed priority analysis with most important tasks
- `analyze summary`: Generates comprehensive task summary with statistics
- `analyze time`: Analyzes tasks by time estimates to identify quick wins

## Extending with New Tools

The agent is designed to be extensible. To add a new tool:

1. Implement the `Tool` interface defined in `agent.go`
2. Add your tool to the agent in `main.go`

Example:

```go
// Create a new tool
myTool := NewMyCustomTool()

// Initialize the agent with multiple tools
agent := NewAgent(model, []Tool{todoTool, myTool})
```

## License

This project is open source and available under the [MIT License](LICENSE).