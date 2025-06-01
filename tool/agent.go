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
	finalResponse = cleanupResponse(finalResponse)

	return finalResponse, nil
}

// cleanupResponse removes duplicate content and thinking process from LLM responses
func cleanupResponse(response string) string {
	// First, check if the response contains code blocks and extract content after them
	if strings.Contains(response, "```") {
		// Find the last occurrence of closing code block
		parts := strings.Split(response, "```")
		if len(parts) >= 3 {
			// Take the content after the last code block
			response = parts[len(parts)-1]
		}
	}
	
	// Remove common patterns of thinking or duplicate responses
	lines := strings.Split(response, "\n")
	var cleanedLines []string
	seenContent := make(map[string]bool)
	var foundMainContent bool
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Skip empty lines before main content
		if trimmedLine == "" && !foundMainContent {
			continue
		}
		
		// Skip lines that look like thinking process
		lowerLine := strings.ToLower(trimmedLine)
		if strings.Contains(lowerLine, "i should") ||
			strings.Contains(lowerLine, "i need to") ||
			strings.Contains(lowerLine, "let me") ||
			strings.Contains(lowerLine, "i think") ||
			strings.Contains(lowerLine, "now that i've used") ||
			strings.Contains(lowerLine, "based on the information") ||
			strings.Contains(lowerLine, "that's all i have to say") ||
			strings.Contains(lowerLine, "i've analyzed") ||
			strings.Contains(lowerLine, "based on the analysis") {
			continue
		}
		
		// Mark that we've found actual content
		if trimmedLine != "" {
			foundMainContent = true
		}
		
		// Skip duplicate content (but keep the first occurrence)
		// Create a normalized version for comparison
		normalizedLine := strings.ToLower(strings.ReplaceAll(trimmedLine, " ", ""))
		if !seenContent[normalizedLine] || trimmedLine == "" {
			if trimmedLine != "" {
				seenContent[normalizedLine] = true
			}
			cleanedLines = append(cleanedLines, line)
		}
	}
	
	// Join the cleaned lines
	cleanedResponse := strings.Join(cleanedLines, "\n")
	
	// Trim any trailing whitespace
	cleanedResponse = strings.TrimSpace(cleanedResponse)
	
	// If the response became empty after cleaning, return a default message
	if cleanedResponse == "" {
		return "I've completed the analysis of your tasks."
	}
	
	return cleanedResponse
}
