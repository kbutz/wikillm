package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
		promptBuilder.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	promptBuilder.WriteString("\nTo use a tool, respond with a JSON object in the following format:\n")
	promptBuilder.WriteString("```json\n{\"tool\": \"tool_name\", \"args\": \"tool arguments\"}\n```\n")
	promptBuilder.WriteString("If you don't need to use a tool, just respond normally.\n\n")
	promptBuilder.WriteString("User query: " + query + "\n\n")
	promptBuilder.WriteString("Your response:")

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
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return ToolCall{}, false
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// First try to unmarshal into a map to handle different args formats
	var rawMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &rawMap)
	if err != nil || rawMap["tool"] == "" {
		return ToolCall{}, false
	}

	// Extract the tool name
	toolName, _ := rawMap["tool"].(string)
	if toolName == "" {
		return ToolCall{}, false
	}

	// Handle different formats of args
	var argsStr string
	switch args := rawMap["args"].(type) {
	case string:
		// If args is already a string, use it directly
		argsStr = args
	case []interface{}:
		// If args is an array, join the elements
		if len(args) > 0 {
			// First element is usually the command
			command, _ := args[0].(string)
			argsStr = command

			// If there are more elements, they are the arguments
			if len(args) > 1 {
				// Join the rest of the arguments
				for i := 1; i < len(args); i++ {
					if arg, ok := args[i].(string); ok {
						argsStr += " " + arg
					}
				}
			}
		}
	case map[string]interface{}:
		// If args is an object, extract the task
		if task, ok := args["task"].(string); ok && task != "" {
			argsStr = "add " + task
		} else {
			// If there's no task field or it's empty, try to construct from other fields
			var parts []string
			for _, v := range args {
				if str, ok := v.(string); ok {
					parts = append(parts, str)
				} else if str, ok := v.(float64); ok {
					parts = append(parts, fmt.Sprintf("%v", str))
				}
			}
			argsStr = strings.Join(parts, " ")
		}
	default:
		// For any other type, convert to string
		argsStr = fmt.Sprintf("%v", args)
	}

	return ToolCall{
		Tool: toolName,
		Args: argsStr,
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
