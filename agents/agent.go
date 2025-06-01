package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kbutz/wikillm/agents/models"
	"github.com/kbutz/wikillm/agents/tools"
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
	model models.LLMModel
	tools []Tool
}

// NewAgent creates a new Agent with the given model and tools
func NewAgent(model models.LLMModel, tools []Tool) *Agent {
	return &Agent{
		model: model,
		tools: tools,
	}
}

// Run Start an interactive session with the user
func (a *Agent) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("LLM Agent To-Do List")
	fmt.Printf("Using model: %s\n", a.model.Name())
	fmt.Println("Type 'exit' to quit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		query := scanner.Text()
		if strings.ToLower(query) == "exit" {
			break
		}

		// Process the query
		fmt.Println("Processing your request...")
		startTime := time.Now()

		response, err := a.ProcessQuery(context.Background(), query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\nResponse:\n%s\n", response)
		fmt.Printf("\nResponse generated in %.2f seconds.\n", elapsed.Seconds())
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
	promptBuilder.WriteString("- If you don't need to use a tool, just respond normally without JSON\n")
	promptBuilder.WriteString("- For analytical queries about tasks (most important, summary, analysis), use 'export' or 'analyze' commands\n\n")
	promptBuilder.WriteString("### Query Analysis Guidelines:\n")
	promptBuilder.WriteString("- If asked about 'most important', 'highest priority', or 'urgent' tasks, use: todo_list with 'analyze priority'\n")
	promptBuilder.WriteString("- If asked for a summary or overview of tasks, use: todo_list with 'analyze summary'\n")
	promptBuilder.WriteString("- If asked complex questions about tasks, use: todo_list with 'export' to get full data\n\n")
	promptBuilder.WriteString("User query: " + query + "\n\n")
	promptBuilder.WriteString("IMPORTANT: Respond with ONLY the tool call JSON or your direct response. Do not include any thinking process or explanations.\n\n")
	promptBuilder.WriteString("Your response:")

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
	fmt.Printf("Tool Result: %s executed with args: %s\nResult: %s\n", toolCall.Tool, toolCall.Args, toolResult)

	// For todo_list tool commands, check if it's a direct command or analytical query
	// Direct commands (add, complete, remove, clear) should return results directly
	// Analytical queries (export, analyze) should be processed by the LLM
	//if strings.EqualFold(toolCall.Tool, "todo_list") {
	//	// Parse the command from args
	//	commandParts := strings.Fields(toolCall.Args)
	//	if len(commandParts) > 0 {
	//		command := strings.ToLower(commandParts[0])
	//		// Direct commands that modify the list - return immediately
	//		if command == "add" || command == "complete" || command == "remove" || command == "clear" {
	//			return toolResult, nil
	//		}
	//		// List commands that are informational - return immediately
	//		if command == "list" && (len(commandParts) == 1 ||
	//			(len(commandParts) > 1 && (commandParts[1] == "all" || commandParts[1] == "priority"))) {
	//			return toolResult, nil
	//		}
	//	}
	//	// For export/analyze commands, let the LLM process the data
	//}

	// Create a follow-up prompt with the tool result
	// For analytical commands, provide clear instructions to avoid verbose output
	followUpPrompt := fmt.Sprintf(
		"Based on the following tool result, provide a clear and concise response to the user's query.\n\n"+
			"If the tool result itself is a satisfactory answer, simply summarize the tool result.\n\n"+
			"User's original query: %s\n\n"+
			"Tool result:\n%s\n\n",
		query, toolResult)

	// Send the follow-up prompt to the LLM
	finalResponse, err := a.model.Query(ctx, followUpPrompt)
	if err != nil {
		return "", fmt.Errorf("error querying LLM for final response: %w", err)
	}

	return finalResponse, nil
}
