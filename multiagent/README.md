# WikiLLM MultiAgent System

Mostly unusable, not worth hacking on top of, though.

This was an example repo to try to wire up multi agents in a "Conversation of Agents" style to run a "personal assistant" locally. 
The end goal was really to have an OpenAI style system that can run locally, primarily to keep data private and to avoid the costs of using a hosted LLM service at scale.
Ultimately, I can't get it to work well enough to be useful, so I'm leaving it here as an example of how I tried to do it.

Some findings:
* I think the architecture is more complicated than it needs to be, which leads to more debugging time and increased debugging difficulty
* I think the "specialist agents" are too rigid and not flexible enough to handle the variety of tasks I want to throw at them. 
* I'm still learning here, but think the specialist agents in this form limit the LLM more than enable it, giving it reduced Agency to fulfill user requests.
* It *almost* works, but I'm going to try again with something simpler.

## Overview

The WikiLLM MultiAgent System provides a framework for creating and managing multiple specialized agents that can work together to handle complex tasks. The system includes:

- **Memory Management**: Persistent storage for agent memory, conversations, and task history
- **Agent Orchestration**: Coordination of multiple agents working together
- **Tool Integration**: Extensible tool system for agents to interact with external systems
- **Message Routing**: Efficient message passing between agents
- **Task Management**: Creation, assignment, and tracking of tasks

## Architecture

The system is built around the following core components:

### Agents

- **Conversation Agent**: Handles natural language interactions with users
- **Coordinator Agent**: Orchestrates the work of specialist agents
- **Specialist Agents**:
  - **Project Manager Agent**: Project planning and lifecycle management
  - **Task Manager Agent**: Personal task management using GTD methodology
  - **Research Assistant Agent**: Information gathering and source evaluation
  - **Scheduler Agent**: Calendar management and appointment scheduling
  - **Communication Manager Agent**: Contact management and communication tracking

### Memory

- **File-based Memory Store**: Persistent storage for agent memory
- **Memory Tool**: Interface for agents to store and retrieve information

### Tools

- **Memory Tool**: Access and manage agent memory
- **Task Tool**: Create and manage tasks

### Orchestration

- **Orchestrator**: Manages agent registration, message routing, and task assignment
- **Service**: High-level API for using the multi-agent system

## Getting Started

### Prerequisites

- Go 1.18 or higher
- An LLM provider implementation

### Basic Usage

```go
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/kbutz/wikillm/multiagent/service"
)

func main() {
	// Create a base directory for the service
	baseDir := filepath.Join(os.TempDir(), "wikillm_multiagent")

	// Create your LLM provider implementation
	llmProvider := YourLLMProviderImplementation{}

	// Create the multi-agent service
	svc, err := service.NewMultiAgentService(service.ServiceConfig{
		BaseDir:     baseDir,
		LLMProvider: llmProvider,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-agent service: %v", err)
	}

	// Start the service
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Process a user message
	response, err := svc.ProcessUserMessage(ctx, "user123", "Hello! Can you help me with a task?")
	if err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}

	log.Printf("Response: %s", response)

	// Stop the service when done
	if err := svc.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}
}
```

### Implementing an LLM Provider

To use the system, you need to implement the `LLMProvider` interface:

```go
type LLMProvider interface {
	Name() string
	Query(ctx context.Context, prompt string) (string, error)
	QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error)
}
```

This interface allows the system to interact with any language model provider (OpenAI, Anthropic, local models, etc.).

## Extending the System

### Adding New Agents

You can create new specialist agents by extending the `BaseAgent`:

```go
// Create a new agent
researchAgent := agents.NewResearchAgent(agents.BaseAgentConfig{
	ID:           "research_agent",
	Type:         multiagent.AgentTypeResearch,
	Name:         "Research Agent",
	Description:  "Specializes in information gathering and research",
	Tools:        yourTools,
	LLMProvider:  yourLLMProvider,
	MemoryStore:  yourMemoryStore,
	Orchestrator: yourOrchestrator,
})

// Add it to the service
svc.AddAgent(researchAgent)
```

### Adding New Tools

You can create new tools by implementing the `Tool` interface:

```go
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args string) (string, error)
}
```

Then add them to the service:

```go
// Create a new tool
searchTool := NewSearchTool()

// Add it to the service
svc.AddTool(searchTool)
```

## Examples

### Basic Example
See the `examples/multiagent_example.go` file for a basic example of how to use the system.

### Interactive Example with LMStudio Integration
The `examples/interactive_example.go` file demonstrates how to connect the multiagent service to a local LMStudio server and provides an interactive command-line interface for testing the system.

To run the interactive example:

1. Download and install [LMStudio](https://lmstudio.ai/)
2. Start LMStudio and load a model
3. Start the local server in LMStudio (default URL: http://localhost:1234)
4. Run the example:

```bash
cd multiagent/examples
go run interactive_example.go
```

The example connects to LMStudio's API endpoint and uses it as the LLM provider for the multiagent service.

### Personal Assistant Demo
The `examples/personal_assistant_demo.go` file demonstrates a comprehensive personal assistant system built on top of the multiagent framework. It includes specialized agents for project management, task management, research, scheduling, and communication.

To run the personal assistant demo:

```bash
cd multiagent/examples
go run personal_assistant_demo.go
```

For more detailed information about the personal assistant functionality, see the [PERSONAL_ASSISTANT.md](PERSONAL_ASSISTANT.md) file.

## Features

- **Memory Persistence**: Agents remember past interactions and context
- **Task Delegation**: Complex tasks are broken down and assigned to specialist agents
- **Coordinated Responses**: Multiple agents can collaborate on a single user request
- **Extensible Architecture**: Easy to add new agent types and tools
- **Conversation Management**: Tracking and management of ongoing conversations
- **Personal Assistant Capabilities**: Specialized agents for project management, task management, scheduling, research, and communication
- **LMStudio Integration**: Support for local LLM processing using LMStudio

## Additional Documentation

For more detailed information about the personal assistant functionality, see the [PERSONAL_ASSISTANT.md](PERSONAL_ASSISTANT.md) file.