package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// TodoListTool implements the Tool interface with enhanced functionality
type TodoListTool struct {
	filePath string
}

// NewTodoListTool creates a new ImprovedTodoListTool
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
	return `Manages a to-do list with priorities and time estimates. 
Commands:
- add <task> [priority:low/medium/high/critical] [time:XXm/XXh] - Add a task with optional priority and time estimate
- list - Show all active tasks
- list all - Show all tasks including completed
- list priority - Show tasks sorted by priority
- complete <number> - Mark a task as completed
- remove <number> - Remove a task
- clear - Clear all tasks
- clear completed - Clear only completed tasks

Examples:
- add Buy groceries priority:high time:30m
- add Call dentist
- complete 1
- list priority`
}

// Parameters returns the parameter schema for the tool
func (t *TodoListTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute (add, list, complete, remove, clear)",
				"enum":        []string{"add", "list", "complete", "remove", "clear"},
			},
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The task description for add command",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"description": "The priority level for add command (low, medium, high, critical)",
				"enum":        []string{"low", "medium", "high", "critical"},
			},
			"time": map[string]interface{}{
				"type":        "string",
				"description": "The time estimate for add command (e.g., 30m, 2h)",
				"pattern":     "^\\d+[mh]$",
			},
			"number": map[string]interface{}{
				"type":        "integer",
				"description": "The task number for complete or remove commands",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "The type for list, clear, or analyze commands (all, priority, completed, summary, time)",
				"enum":        []string{"all", "priority", "completed", "summary", "time"},
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

// Execute runs the tool with the given arguments and returns the result
func (t *TodoListTool) Execute(ctx context.Context, args string) (string, error) {
	// Check if args is a JSON string
	var params map[string]interface{}
	if strings.HasPrefix(strings.TrimSpace(args), "{") {
		if err := json.Unmarshal([]byte(args), &params); err == nil {
			// Successfully parsed JSON, extract command and arguments
			cmdInterface, ok := params["command"]
			if !ok {
				return "", fmt.Errorf("command parameter is required")
			}

			command, ok := cmdInterface.(string)
			if !ok {
				return "", fmt.Errorf("command must be a string")
			}

			command = strings.ToLower(command)

			switch command {
			case "add":
				// Extract task, priority, and time
				taskInterface, hasTask := params["task"]
				if !hasTask {
					return "", fmt.Errorf("task parameter is required for add command")
				}

				task, ok := taskInterface.(string)
				if !ok {
					return "", fmt.Errorf("task must be a string")
				}

				// Build add command
				addCmd := "add " + task

				// Add priority if present
				if priorityInterface, hasProperty := params["priority"]; hasProperty {
					if priority, ok := priorityInterface.(string); ok {
						addCmd += " priority:" + priority
					}
				}

				// Add time if present
				if timeInterface, hasProperty := params["time"]; hasProperty {
					if timeStr, ok := timeInterface.(string); ok {
						addCmd += " time:" + timeStr
					}
				}

				return t.addTask(addCmd)
			case "list":
				// Extract list type
				listType := "active"
				if typeInterface, hasProperty := params["type"]; hasProperty {
					if typeStr, ok := typeInterface.(string); ok {
						listType = typeStr
					}
				}

				return t.listTasks(listType)
			case "complete":
				// Extract task number
				numberInterface, hasNumber := params["number"]
				if !hasNumber {
					return "", fmt.Errorf("number parameter is required for complete command")
				}

				var numberStr string
				switch num := numberInterface.(type) {
				case float64:
					numberStr = fmt.Sprintf("%d", int(num))
				case int:
					numberStr = fmt.Sprintf("%d", num)
				case string:
					numberStr = num
				default:
					return "", fmt.Errorf("number must be an integer")
				}

				return t.completeTask(numberStr)
			case "remove":
				// Extract task number
				numberInterface, hasNumber := params["number"]
				if !hasNumber {
					return "", fmt.Errorf("number parameter is required for remove command")
				}

				var numberStr string
				switch num := numberInterface.(type) {
				case float64:
					numberStr = fmt.Sprintf("%d", int(num))
				case int:
					numberStr = fmt.Sprintf("%d", num)
				case string:
					numberStr = num
				default:
					return "", fmt.Errorf("number must be an integer")
				}

				return t.removeTask(numberStr)
			case "clear":
				// Extract clear type
				clearType := "all"
				if typeInterface, hasProperty := params["type"]; hasProperty {
					if typeStr, ok := typeInterface.(string); ok {
						clearType = typeStr
					}
				}

				return t.clearTasks(clearType)
			default:
				return "", fmt.Errorf("unknown command: %s", command)
			}
		}
	}

	// Fall back to parsing command from string
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "add":
		return t.addTask(args)
	case "list":
		listType := "active"
		if len(parts) > 1 {
			listType = strings.ToLower(parts[1])
		}
		return t.listTasks(listType)
	case "complete":
		if len(parts) < 2 {
			return "", fmt.Errorf("complete command requires a task number")
		}
		return t.completeTask(parts[1])
	case "remove":
		if len(parts) < 2 {
			return "", fmt.Errorf("remove command requires a task number")
		}
		return t.removeTask(parts[1])
	case "clear":
		clearType := "all"
		if len(parts) > 1 {
			clearType = strings.ToLower(parts[1])
		}
		return t.clearTasks(clearType)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// parseAddCommand extracts task description, priority, and time estimate from add command
func (t *TodoListTool) parseAddCommand(args string) (description string, priority TaskPriority, timeEstimate int) {
	// Remove the "add" command
	args = strings.TrimPrefix(args, "add ")
	args = strings.TrimSpace(args)

	// Default values
	priority = PriorityMedium
	timeEstimate = 0

	// Extract priority if present
	priorityRegex := regexp.MustCompile(`\bpriority:(\w+)\b`)
	if matches := priorityRegex.FindStringSubmatch(args); len(matches) > 1 {
		priority = ParsePriority(matches[1])
		args = priorityRegex.ReplaceAllString(args, "")
	}

	// Extract time estimate if present
	timeRegex := regexp.MustCompile(`\btime:(\d+)([hm])\b`)
	if matches := timeRegex.FindStringSubmatch(args); len(matches) > 2 {
		amount, _ := strconv.Atoi(matches[1])
		unit := matches[2]
		if unit == "h" {
			timeEstimate = amount * 60 // Convert hours to minutes
		} else {
			timeEstimate = amount
		}
		args = timeRegex.ReplaceAllString(args, "")
	}

	// Clean up the description
	description = strings.TrimSpace(args)
	return
}

// addTask adds a new task with priority and time estimate
func (t *TodoListTool) addTask(args string) (string, error) {
	description, priority, timeEstimate := t.parseAddCommand(args)

	if description == "" {
		return "", fmt.Errorf("task description cannot be empty")
	}

	// Load existing tasks
	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	// Create new task
	task := NewTask(description)
	task.Priority = priority
	task.TimeEstimate = timeEstimate

	// Add to list
	taskList.Add(*task)

	// Save tasks
	if err := t.saveTasks(taskList); err != nil {
		return "", err
	}

	result := fmt.Sprintf("✓ Added task: %s", task.String())
	return result, nil
}

// listTasks returns a formatted list of tasks
func (t *TodoListTool) listTasks(listType string) (string, error) {
	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	var tasks []Task
	var title string

	switch listType {
	case "all":
		tasks = taskList.Tasks
		title = "All Tasks"
	case "priority":
		tasks = taskList.GetTasksByPriority()
		title = "Tasks by Priority"
	default: // "active" or anything else
		tasks = taskList.GetActiveTasks()
		title = "Active Tasks"
	}

	if len(tasks) == 0 {
		return fmt.Sprintf("No %s tasks found.", strings.ToLower(title)), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("%s:\n", title))
	result.WriteString(strings.Repeat("-", 50) + "\n")

	for i, task := range tasks {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, task.String()))
	}

	// Add summary
	activeTasks := taskList.GetActiveTasks()
	totalTime := 0
	for _, task := range activeTasks {
		totalTime += task.TimeEstimate
	}

	result.WriteString(strings.Repeat("-", 50) + "\n")
	result.WriteString(fmt.Sprintf("Total: %d active tasks", len(activeTasks)))
	if totalTime > 0 {
		hours := totalTime / 60
		minutes := totalTime % 60
		if hours > 0 {
			result.WriteString(fmt.Sprintf(" (~%dh %dm total time)", hours, minutes))
		} else {
			result.WriteString(fmt.Sprintf(" (~%dm total time)", minutes))
		}
	}

	return result.String(), nil
}

// completeTask marks a task as completed
func (t *TodoListTool) completeTask(indexStr string) (string, error) {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", fmt.Errorf("invalid task number: %s", indexStr)
	}

	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	// Get active tasks for indexing
	activeTasks := taskList.GetActiveTasks()
	if index < 1 || index > len(activeTasks) {
		return "", fmt.Errorf("task number %d out of range (1-%d)", index, len(activeTasks))
	}

	// Find the actual index in the full task list
	targetTask := activeTasks[index-1]
	for i, task := range taskList.Tasks {
		if task.ID == targetTask.ID {
			if err := taskList.Complete(i + 1); err != nil {
				return "", err
			}
			break
		}
	}

	// Save tasks
	if err := t.saveTasks(taskList); err != nil {
		return "", err
	}

	return fmt.Sprintf("✓ Completed task: %s", targetTask.Description), nil
}

// removeTask removes a task from the list
func (t *TodoListTool) removeTask(indexStr string) (string, error) {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", fmt.Errorf("invalid task number: %s", indexStr)
	}

	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	// Get active tasks for indexing
	activeTasks := taskList.GetActiveTasks()
	if index < 1 || index > len(activeTasks) {
		return "", fmt.Errorf("task number %d out of range (1-%d)", index, len(activeTasks))
	}

	// Find the actual index in the full task list
	targetTask := activeTasks[index-1]
	for i, task := range taskList.Tasks {
		if task.ID == targetTask.ID {
			if err := taskList.Remove(i + 1); err != nil {
				return "", err
			}
			break
		}
	}

	// Save tasks
	if err := t.saveTasks(taskList); err != nil {
		return "", err
	}

	return fmt.Sprintf("✓ Removed task: %s", targetTask.Description), nil
}

// clearTasks clears tasks based on type
func (t *TodoListTool) clearTasks(clearType string) (string, error) {
	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	if clearType == "completed" {
		// Remove only completed tasks
		var activeTasks []Task
		for _, task := range taskList.Tasks {
			if !task.Completed {
				activeTasks = append(activeTasks, task)
			}
		}
		taskList.Tasks = activeTasks

		if err := t.saveTasks(taskList); err != nil {
			return "", err
		}

		return "✓ Cleared all completed tasks.", nil
	}

	// Clear all tasks
	taskList.Tasks = []Task{}
	if err := t.saveTasks(taskList); err != nil {
		return "", err
	}

	return "✓ Cleared all tasks from the to-do list.", nil
}

// loadTasks loads tasks from the JSON file
func (t *TodoListTool) loadTasks() (*TaskList, error) {
	taskList := &TaskList{Tasks: []Task{}}

	// Check if file exists
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		// Return empty task list
		return taskList, nil
	}

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading tasks file: %w", err)
	}

	if len(data) == 0 {
		return taskList, nil
	}

	if err := taskList.Unmarshal(data); err != nil {
		return nil, fmt.Errorf("error parsing tasks: %w", err)
	}

	return taskList, nil
}

// saveTasks saves tasks to the JSON file
func (t *TodoListTool) saveTasks(taskList *TaskList) error {
	data, err := taskList.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling tasks: %w", err)
	}

	if err := os.WriteFile(t.filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing tasks file: %w", err)
	}

	return nil
}
