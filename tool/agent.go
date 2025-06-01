package main

import (
	"context"
	"fmt"
	toolsdir "github.com/kbutz/wikillm/tool/tools"
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
	model          LLMModel
	tools          []Tool
	queryAnalyzer  *toolsdir.QueryAnalyzer
	responseFilter *ResponseFilter
}

// NewAgent creates a new Agent with the given model and tools
func NewAgent(model LLMModel, tools []Tool) *Agent {
	return &Agent{
		model:          model,
		tools:          tools,
		queryAnalyzer:  toolsdir.NewQueryAnalyzer(),
		responseFilter: NewResponseFilter(),
	}
}

// ProcessQuery processes a user query and returns a response
func (a *Agent) ProcessQuery(ctx context.Context, query string) (string, error) {
	// First, check if this is a direct query about tasks that we can handle more efficiently
	if a.isTaskQuery(query) {
		return a.handleTaskQuery(ctx, query)
	}
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

	// Add examples for complex queries
	if strings.Contains(strings.ToLower(query), "todo") || strings.Contains(strings.ToLower(query), "task") ||
		strings.Contains(strings.ToLower(query), "summarize") || strings.Contains(strings.ToLower(query), "summary") {
		promptBuilder.WriteString("\n\n(Hint: For summaries use 'analyze summary', for priorities use 'analyze priority')")
	}

	// Send the prompt to the LLM
	response, err := a.model.Query(ctx, promptBuilder.String())
	if err != nil {
		return "", fmt.Errorf("error querying LLM: %w", err)
	}

	// Check if the response contains a tool call
	toolCall, found := toolsdir.ExtractToolCall(response)
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
	// fmt.Printf("Tool %s executed with args: %s\nResult: %s\n", toolCall.Tool, toolCall.Args, toolResult)

	// For todo_list tool commands, check if it's a direct command or analytical query
	// Direct commands (add, complete, remove, clear) should return results directly
	// Analytical queries (export, analyze) should be processed by the LLM
	if strings.EqualFold(toolCall.Tool, "todo_list") {
		// Parse the command from args
		commandParts := strings.Fields(toolCall.Args)
		if len(commandParts) > 0 {
			command := strings.ToLower(commandParts[0])
			// Direct commands that modify the list - return immediately
			if command == "add" || command == "complete" || command == "remove" || command == "clear" {
				return toolResult, nil
			}
			// List commands that are informational - return immediately
			if command == "list" && (len(commandParts) == 1 ||
				(len(commandParts) > 1 && (commandParts[1] == "all" || commandParts[1] == "priority"))) {
				return toolResult, nil
			}
		}
		// For export/analyze commands, let the LLM process the data
	}

	// Create a follow-up prompt with the tool result
	// For analytical commands, provide clear instructions to avoid verbose output
	followUpPrompt := fmt.Sprintf(
		"Based on the following task analysis data, provide a clear and concise response to the user's query.\n\n"+
			"User's original query: %s\n\n"+
			"Task analysis result:\n%s\n\n"+
			"CRITICAL Instructions:\n"+
			"- Provide ONLY the final answer to the user\n"+
			"- Do NOT include your thought process, reasoning, or any meta-commentary\n"+
			"- Do NOT repeat the query or explain what you did\n"+
			"- Do NOT say things like 'Based on the analysis' or 'I've analyzed'\n"+
			"- Format the response in a user-friendly way\n"+
			"- Be direct and helpful\n"+
			"- Start your response immediately with the answer\n\n"+
			"Your response:",
		query, toolResult)

	// Send the follow-up prompt to the LLM
	finalResponse, err := a.model.Query(ctx, followUpPrompt)
	if err != nil {
		return "", fmt.Errorf("error querying LLM for final response: %w", err)
	}

	// Clean up the response - remove any duplicate content or thinking process
	finalResponse = a.responseFilter.FilterResponse(finalResponse)

	return finalResponse, nil
}

// isTaskQuery checks if the query is about tasks/todos
func (a *Agent) isTaskQuery(query string) bool {
	lowerQuery := strings.ToLower(query)
	return strings.Contains(lowerQuery, "task") ||
		strings.Contains(lowerQuery, "todo") ||
		strings.Contains(lowerQuery, "important") ||
		strings.Contains(lowerQuery, "priority") ||
		strings.Contains(lowerQuery, "easiest") ||
		strings.Contains(lowerQuery, "difficult")
}

// handleTaskQuery handles task queries more efficiently
func (a *Agent) handleTaskQuery(ctx context.Context, query string) (string, error) {
	// Analyze the query type
	queryType := a.queryAnalyzer.AnalyzeQuery(query)

	// Get the appropriate tool command
	toolCommand := a.queryAnalyzer.GetToolCommand(queryType)

	// Find the todo tool
	var todoTool Tool
	for _, tool := range a.tools {
		if tool.Name() == "todo_list" {
			todoTool = tool
			break
		}
	}

	if todoTool == nil {
		return "I don't have access to a todo list tool.", nil
	}

	// Execute the tool directly
	toolResult, err := todoTool.Execute(ctx, toolCommand)
	if err != nil {
		return "", fmt.Errorf("error executing todo tool: %w", err)
	}

	// For direct queries that need formatting, format the response
	if queryType == toolsdir.QueryTypeMostImportant || queryType == toolsdir.QueryTypeDifficulty {
		// Load the task list for special formatting
		if _, ok := todoTool.(*toolsdir.ImprovedTodoListTool); ok {
			// This is a bit of a hack, but we need access to the task list
			// In a real implementation, we'd refactor to expose this properly
			return a.formatDirectResponse(ctx, queryType, toolResult, query)
		}
	}

	// For analytical queries, let the LLM process the result
	if strings.Contains(toolCommand, "analyze") || strings.Contains(toolCommand, "export") {
		return a.processAnalyticalResult(ctx, query, toolResult)
	}
	
	// For direct commands, return the result as-is
	return toolResult, nil
}

// formatDirectResponse formats direct query responses
func (a *Agent) formatDirectResponse(ctx context.Context, queryType toolsdir.QueryType, toolResult, originalQuery string) (string, error) {
	// For now, use the LLM to format the response concisely
	prompt := fmt.Sprintf(
		"Based on this task data, answer the user's question directly and concisely.\n\n"+
			"User's question: %s\n\n"+
			"Task data:\n%s\n\n"+
			"Instructions:\n"+
			"- Give ONLY the direct answer\n"+
			"- Do NOT include any reasoning or explanation\n"+
			"- Be specific and helpful\n"+
			"- Start immediately with the answer\n\n"+
			"Your response:",
		originalQuery, toolResult)

	response, err := a.model.Query(ctx, prompt)
	if err != nil {
		return "", err
	}

	return a.responseFilter.FilterResponse(response), nil
}

// processAnalyticalResult processes analytical command results
func (a *Agent) processAnalyticalResult(ctx context.Context, query, toolResult string) (string, error) {
	prompt := fmt.Sprintf(
		"Based on this analysis, provide a clear and direct response to the user's query.\n\n"+
			"User's query: %s\n\n"+
			"Analysis result:\n%s\n\n"+
			"Instructions:\n"+
			"- Provide ONLY the answer\n"+
			"- Do NOT include meta-commentary\n"+
			"- Be direct and helpful\n"+
			"- Format nicely if appropriate\n\n"+
			"Your response:",
		query, toolResult)

	response, err := a.model.Query(ctx, prompt)
	if err != nil {
		return "", err
	}

	return a.responseFilter.FilterResponse(response), nil
}
