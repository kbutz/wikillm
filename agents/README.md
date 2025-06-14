# WikiLLM Agents

WikiLLM Agents is a playground repo to help me experiment with building interactive tool using agents using large language models (LLMs). This project provides a simple command-line interface for managing a to-do list with an advanced memory system, allowing you to add, remove, and analyze tasks using natural language commands while maintaining context across conversations.

## Prerequisites

Before you can use WikiLLM Agents, you'll need to have the following installed:

1. **Go** (version 1.16 or later) - [Download Go](https://golang.org/dl/)
2. **One of the following LLM providers**:
   - **LM Studio** - [Download LM Studio](https://lmstudio.ai/)
     - I've only tested with LM Studio most recently, so not sure if the ollama path works right now
   - **Ollama** - [Download Ollama](https://ollama.ai/)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/kbutz/wikillm.git
   cd wikillm
   ```

2. Navigate to the agents directory:
   ```
   cd agents
   ```

3. Build the application:
   ```
   go build
   ```

## Setting Up Your LLM Provider

### Option 1: LM Studio (Recommended for beginners)

1. Download and install LM Studio from [https://lmstudio.ai/](https://lmstudio.ai/)
2. Launch LM Studio
3. Download a model (recommended: Mistral 7B or Llama 2)
4. Start the local server by clicking on "Local Server" in the sidebar and then "Start Server"
5. The server will run on http://localhost:1234 by default

### Option 2: Ollama

1. Download and install Ollama from [https://ollama.ai/](https://ollama.ai/)
2. Pull a model using the command line:
   ```
   ollama pull mistral
   ```
   (You can replace "mistral" with any other model you prefer)
3. Ollama will automatically start a server on http://localhost:11434

## Usage

### Starting the Application

Run the application with the following command:

```
./agents
```

By default, this will use LM Studio as the provider, look for a to-do list file named "todo.txt" in the current directory, and enable the enhanced memory system.

### Command-Line Options

You can customize the behavior with these command-line flags:

- `--model`: Name of the LLM model to use (default: "default")
- `--provider`: Model provider to use, either "lmstudio" or "ollama" (default: "lmstudio")
- `--todo-file`: Path to the to-do list file (default: "todo.txt")
- `--memory-file`: Path to the simple memory file (default: "memory.txt")
- `--memory-dir`: Directory for enhanced memory storage (default: "memory")
- `--enhanced-memory`: Use enhanced memory system (default: true)
- `--debug`: Enable debug mode for more detailed logs (default: false)

Example:
```
./agents --model mistral --provider ollama --todo-file my_tasks.txt --enhanced-memory=true --debug
```

### Interacting with the Agent

Once the application is running, you'll see a prompt where you can type your commands:

```
LLM Agent To-Do List
Using model: default
Type 'exit' to quit
>
```

You can now interact with the agent using natural language. Here are some examples:

- "Add a new task to buy groceries with high priority and a time estimate of 30 minutes"
- "Show me all my tasks"
- "Show me my high priority tasks"
- "Mark task 2 as completed"
- "Remove task 3"
- "Give me a summary of my tasks"
- "How much time do I need to complete all my tasks?"

Type `exit` to quit the application.

## Enhanced Memory System

The agent now includes an advanced memory system that automatically:
- **Remembers** project details, preferences, and decisions
- **Retrieves** relevant context for your queries
- **Maintains** conversation history across sessions
- **Learns** your work patterns and preferences

For detailed information about the memory system, see [MEMORY_GUIDE.md](MEMORY_GUIDE.md).

## To-Do List Commands

The agent understands the following commands for managing your to-do list:

- **Add a task**: Add a new task with optional priority and time estimate
  - Example: "Add buy groceries with high priority and 30 minutes"

- **List tasks**: Show all active tasks or filter by criteria
  - Example: "Show me all my tasks"
  - Example: "List my high priority tasks"

- **Complete a task**: Mark a task as completed
  - Example: "Mark task 2 as complete"

- **Remove a task**: Delete a task from the list
  - Example: "Remove task 3"

- **Clear tasks**: Remove all tasks or just completed ones
  - Example: "Clear all my tasks"
  - Example: "Clear completed tasks"

- **Analyze tasks**: Get insights about your tasks
  - Example: "Analyze my tasks by priority"
  - Example: "Give me a summary of my tasks"
  - Example: "How much time do I need for all my tasks?"

## Troubleshooting

### Connection Issues

If you're having trouble connecting to your LLM provider:

1. Make sure the LLM server is running
   - For LM Studio, check that the local server is started in the application
   - For Ollama, ensure the service is running (`ollama list` should work)

2. Check that you're using the correct provider flag
   - Use `--provider lmstudio` for LM Studio
   - Use `--provider ollama` for Ollama

3. Verify that the model you specified exists
   - For LM Studio, check that the model is loaded in the application
   - For Ollama, run `ollama list` to see available models

### Performance Issues

If the agent is responding slowly:

1. Try a smaller model if available
2. Ensure your computer meets the minimum requirements for running LLMs
3. Close other resource-intensive applications

## Examples

### Example 1: Managing a Project

```
> Add create project proposal with critical priority and 2 hours time estimate
Processing your request...

Response:
Task added: [ ] create project proposal [Critical] (~2h0m)

Response generated in 0.52 seconds.

> Add research competitors with high priority and 1 hour time estimate
Processing your request...

Response:
Task added: [ ] research competitors [High] (~1h0m)

Response generated in 0.48 seconds.

> Show me my tasks by priority
Processing your request...

Response:
Tasks by Priority:
1. [ ] create project proposal [Critical] (~2h0m)
2. [ ] research competitors [High] (~1h0m)

Response generated in 0.50 seconds.
```

### Example 2: Task Analysis

```
> Analyze my time estimates
Processing your request...

Response:
Time Estimate Analysis:
- Total estimated time for all tasks: 3 hours 0 minutes
- Total estimated time for active tasks: 3 hours 0 minutes
- Average time per task: 1 hour 30 minutes
- Tasks with highest time estimates:
  1. create project proposal: 2 hours 0 minutes
  2. research competitors: 1 hour 0 minutes

Response generated in 0.55 seconds.
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.