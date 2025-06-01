package main

import (
	"context"
	"fmt"
	"github.com/kbutz/wikillm/tool/tools"
	"strings"
)

//go:generate mockgen -source=agent.go -package=main -destination=./mocks/agent_mock.go

// Tool defines the interface for tools that the agent can use
type Tool interface {
	// Name returns the name of the tool
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, args string) (string, error)
}

// Agent represents an LLM agent with access to tools
type Agent struct {
	model LLMModel
	tools []Tool
}

// NewAgent creates a new Agent with the given model and tools
func NewAgent(model LLMModel, tools []Tool) *Agent {
	return &Agent{
		model: model,
		tools: tools,
	}
}

// ProcessQuery processes a user query and returns a response
func (a *Agent) ProcessQuery(ctx context.Context, query string) (string, error) {
	// Create a prompt for the LLM that includes information about available tools
	var promptBuilder strings.Builder

	promptBuilder.WriteString("You are an AI assistant with access to the following tools:\n\n")

	for _, tool := range a.tools {
		promptBuilder.WriteString(fmt.Sprintf("### Tool: %s\n%s\n\n", tool.Name(), tool.Description()))
	}

	promptBuilder.WriteString("\n## How to Use Tools\n\n")
	promptBuilder.WriteString("To use a tool, respond with a JSON object in a code block:\n\n")
	promptBuilder.WriteString("```json\n{\n  \"tool\": \"tool_name\",\n  \"args\": \"command and arguments\"\n}\n```\n\n")
	promptBuilder.WriteString("### Important Notes:\n")
	promptBuilder.WriteString("- Always use the exact tool name as shown above\n")
	promptBuilder.WriteString("- The 'args' field should contain the complete command with all parameters\n")
	promptBuilder.WriteString("- For the todo_list tool, include priority and time in the args string (e.g., \"add Buy milk priority:high time:30m\")\n")
	promptBuilder.WriteString("- If you don't need to use a tool, just respond normally without JSON\n\n")
	promptBuilder.WriteString("User query: " + query + "\n\n")
	promptBuilder.WriteString("Your response:")

	// Add examples for complex queries
	if strings.Contains(strings.ToLower(query), "todo") || strings.Contains(strings.ToLower(query), "task") {
		promptBuilder.WriteString("\n\n(Hint: The user is asking about tasks. Use the todo_list tool if appropriate.)")
	}

	// Send the prompt to the LLM
	response, err := a.model.Query(ctx, promptBuilder.String())
	if err != nil {
		return "", fmt.Errorf("error querying LLM: %w", err)
	}

	// Check if the response contains a tool call
	toolCall, found := tools.ExtractToolCall(response)
	if !found {
		return response, nil
	}

	// Find the requested tool
	var selectedTool Tool
	for _, tool := range a.tools {
		if strings.EqualFold(tool.Name(), toolCall.Tool) {
			selectedTool = tool
			break
		}
	}

	if selectedTool == nil {
		return fmt.Sprintf("I tried to use the %s tool, but it's not available. Here's what I know:\n\n%s",
			toolCall.Tool, response), nil
	}

	// Execute the tool
	toolResult, err := selectedTool.Execute(ctx, toolCall.Args)
	if err != nil {
		return "", fmt.Errorf("error executing tool %s: %w", toolCall.Tool, err)
	}
	fmt.Printf("Tool %s executed with args: %s\nResult: %s\n", toolCall.Tool, toolCall.Args, toolResult)

	// For all todo_list tool commands, return the result directly
	// to avoid the LLM potentially replacing actual task content with placeholders
	if strings.EqualFold(toolCall.Tool, "todo_list") {
		return toolResult, nil
	}

	// Create a follow-up prompt with the tool result
	followUpPrompt := fmt.Sprintf(
		"You previously tried to answer this query: %s\n\n"+
			"You used the %s tool with args: %s\n\n"+
			"The tool returned this result: %s\n\n"+
			"Please provide your final response to the user based on this information:",
		query, toolCall.Tool, toolCall.Args, toolResult)

	// Send the follow-up prompt to the LLM
	finalResponse, err := a.model.Query(ctx, followUpPrompt)
	if err != nil {
		return "", fmt.Errorf("error querying LLM for final response: %w", err)
	}

	fmt.Println("Final response from LLM:", finalResponse)

	return finalResponse, nil
}
