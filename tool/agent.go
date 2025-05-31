package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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
	toolCall, found := extractToolCall(response)
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

// ToolCall represents a request to use a tool
type ToolCall struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
}

// extractToolCall attempts to extract a tool call from the LLM response
func extractToolCall(response string) (ToolCall, bool) {
	// Look for JSON blocks in the response
	// Try to find JSON within code blocks first
	var jsonStr string
	
	// Check for ```json blocks
	jsonBlockRegex := regexp.MustCompile("```json\\s*\\n([\\s\\S]*?)\\n```")
	if matches := jsonBlockRegex.FindStringSubmatch(response); len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		// Fall back to finding raw JSON
		jsonStart := strings.Index(response, "{")
		jsonEnd := strings.LastIndex(response, "}")
		
		if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
			return ToolCall{}, false
		}
		
		jsonStr = response[jsonStart : jsonEnd+1]
	}

	// First try to unmarshal into a map to handle different args formats
	var rawMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &rawMap)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\nJSON string: %s\n", err, jsonStr)
		return ToolCall{}, false
	}

	// Extract the tool name
	toolName, ok := rawMap["tool"].(string)
	if !ok || toolName == "" {
		return ToolCall{}, false
	}

	// Handle different formats of args
	var argsStr string
	rawArgs, hasArgs := rawMap["args"]
	
	if !hasArgs {
		// Some LLMs might use "arguments" instead of "args"
		rawArgs, hasArgs = rawMap["arguments"]
	}
	
	if !hasArgs {
		argsStr = "" // No args provided
	} else {
		switch args := rawArgs.(type) {
		case string:
			// If args is already a string, use it directly
			argsStr = strings.TrimSpace(args)
		case []interface{}:
			// If args is an array, join the elements
			var parts []string
			for _, arg := range args {
				if str, ok := arg.(string); ok {
					parts = append(parts, str)
				} else {
					parts = append(parts, fmt.Sprintf("%v", arg))
				}
			}
			argsStr = strings.Join(parts, " ")
		case map[string]interface{}:
			// If args is an object, try to extract meaningful content
			
			// First check for specific fields that indicate the command
			if command, ok := args["command"].(string); ok {
				argsStr = command
				
				// Add task/content if present
				if task, ok := args["task"].(string); ok && task != "" {
					argsStr += " " + task
				} else if content, ok := args["content"].(string); ok && content != "" {
					argsStr += " " + content
				} else if desc, ok := args["description"].(string); ok && desc != "" {
					argsStr += " " + desc
				}
				
				// Add priority if present
				if priority, ok := args["priority"].(string); ok && priority != "" {
					argsStr += " priority:" + priority
				}
				
				// Add time if present
				if timeStr, ok := args["time"].(string); ok && timeStr != "" {
					argsStr += " time:" + timeStr
				} else if timeMin, ok := args["time_minutes"].(float64); ok {
					argsStr += fmt.Sprintf(" time:%dm", int(timeMin))
				}
			} else if task, ok := args["task"].(string); ok && task != "" {
				// Just a task field - assume it's an add command
				argsStr = "add " + task
			} else {
				// Fall back to concatenating all string values
				var parts []string
				
				// Try to maintain some order: command-like fields first
				commandFields := []string{"action", "command", "operation"}
				for _, field := range commandFields {
					if val, ok := args[field].(string); ok {
						parts = append(parts, val)
						delete(args, field)
					}
				}
				
				// Then add remaining fields
				for _, v := range args {
					if str, ok := v.(string); ok && str != "" {
						parts = append(parts, str)
					} else if num, ok := v.(float64); ok {
						parts = append(parts, fmt.Sprintf("%v", num))
					}
				}
				argsStr = strings.Join(parts, " ")
			}
		default:
			// For any other type, convert to string
			argsStr = fmt.Sprintf("%v", rawArgs)
		}
	}

	return ToolCall{
		Tool: toolName,
		Args: strings.TrimSpace(argsStr),
	}, true
}

// TodoListTool implements the Tool interface for managing a to-do list
type TodoListTool struct {
	filePath string
}

// NewTodoListTool creates a new TodoListTool
func NewTodoListTool(filePath string) *TodoListTool {
	return &TodoListTool{
		filePath: filePath,
	}
}

// Name returns the name of the tool
func (t *TodoListTool) Name() string {
	return "todo_list"
}

// Description returns a description of what the tool does
func (t *TodoListTool) Description() string {
	return "Manages a to-do list. Commands: add <task>, list, remove <number>, clear"
}

// Execute runs the tool with the given arguments and returns the result
func (t *TodoListTool) Execute(ctx context.Context, args string) (string, error) {
	parts := strings.SplitN(args, " ", 2)
	command := strings.ToLower(parts[0])

	switch command {
	case "add":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("add command requires a task")
		}
		return t.addTask(parts[1])

	case "list":
		return t.listTasks()

	case "remove":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("remove command requires a task number")
		}
		return t.removeTask(parts[1])

	case "clear":
		return t.clearTasks()

	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// addTask adds a task to the to-do list
func (t *TodoListTool) addTask(task string) (string, error) {
	tasks, err := t.readTasks()
	if err != nil {
		return "", err
	}

	tasks = append(tasks, strings.TrimSpace(task))

	if err := t.writeTasks(tasks); err != nil {
		return "", err
	}

	return fmt.Sprintf("Added task: %s", task), nil
}

// listTasks returns a list of all tasks
func (t *TodoListTool) listTasks() (string, error) {
	tasks, err := t.readTasks()
	if err != nil {
		return "", err
	}

	if len(tasks) == 0 {
		return "No tasks in the to-do list.", nil
	}

	var result strings.Builder
	result.WriteString("To-Do List:\n")

	for i, task := range tasks {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, task))
	}

	return result.String(), nil
}

// removeTask removes a task from the to-do list
func (t *TodoListTool) removeTask(indexStr string) (string, error) {
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		return "", fmt.Errorf("invalid task number: %s", indexStr)
	}

	tasks, err := t.readTasks()
	if err != nil {
		return "", err
	}

	if index < 1 || index > len(tasks) {
		return "", fmt.Errorf("task number out of range: %d", index)
	}

	removedTask := tasks[index-1]
	tasks = append(tasks[:index-1], tasks[index:]...)

	if err := t.writeTasks(tasks); err != nil {
		return "", err
	}

	return fmt.Sprintf("Removed task: %s", removedTask), nil
}

// clearTasks removes all tasks from the to-do list
func (t *TodoListTool) clearTasks() (string, error) {
	if err := t.writeTasks([]string{}); err != nil {
		return "", err
	}

	return "Cleared all tasks from the to-do list.", nil
}

// readTasks reads the tasks from the file
func (t *TodoListTool) readTasks() ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		// Create empty file
		return []string{}, nil
	}

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading to-do list file: %w", err)
	}

	if len(data) == 0 {
		return []string{}, nil
	}

	tasks := strings.Split(string(data), "\n")

	// Filter out empty lines
	var filteredTasks []string
	for _, task := range tasks {
		if strings.TrimSpace(task) != "" {
			filteredTasks = append(filteredTasks, task)
		}
	}

	return filteredTasks, nil
}

// writeTasks writes the tasks to the file
func (t *TodoListTool) writeTasks(tasks []string) error {
	data := strings.Join(tasks, "\n")

	err := os.WriteFile(t.filePath, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("error writing to-do list file: %w", err)
	}

	return nil
}
