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
- export - Export all task data as JSON for analysis
- analyze priority - Get detailed priority analysis
- analyze summary - Get a comprehensive task summary
- analyze time - Get time estimate analysis

Examples:
- add Buy groceries priority:high time:30m
- add Call dentist
- complete 1
- list priority
- analyze priority
- export`
}

// Parameters returns the parameter schema for the tool
func (t *TodoListTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute (add, list, complete, remove, clear, export, analyze)",
				"enum":        []string{"add", "list", "complete", "remove", "clear", "export", "analyze"},
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

			case "export":
				return t.exportTasks()

			case "analyze":
				// Extract analyze type
				typeInterface, hasType := params["type"]
				if !hasType {
					return "", fmt.Errorf("type parameter is required for analyze command")
				}

				typeStr, ok := typeInterface.(string)
				if !ok {
					return "", fmt.Errorf("type must be a string")
				}

				return t.analyzeTasks(typeStr)

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
	case "export":
		return t.exportTasks()
	case "analyze":
		if len(parts) < 2 {
			return "", fmt.Errorf("analyze command requires a type (priority, summary, or time)")
		}
		return t.analyzeTasks(parts[1])
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

// exportTasks exports all tasks as JSON for LLM analysis
func (t *TodoListTool) exportTasks() (string, error) {
	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	// Create a structured export format
	type ExportFormat struct {
		TotalTasks     int     `json:"total_tasks"`
		ActiveTasks    int     `json:"active_tasks"`
		CompletedTasks int     `json:"completed_tasks"`
		TotalTimeHours float64 `json:"total_time_hours"`
		Tasks          []Task  `json:"tasks"`
	}

	activeTasks := taskList.GetActiveTasks()
	totalTime := 0
	for _, task := range activeTasks {
		totalTime += task.TimeEstimate
	}

	export := ExportFormat{
		TotalTasks:     len(taskList.Tasks),
		ActiveTasks:    len(activeTasks),
		CompletedTasks: len(taskList.Tasks) - len(activeTasks),
		TotalTimeHours: float64(totalTime) / 60.0,
		Tasks:          taskList.Tasks,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling export data: %w", err)
	}

	return string(data), nil
}

// analyzeTasks provides various analyses of the task list
func (t *TodoListTool) analyzeTasks(analysisType string) (string, error) {
	taskList, err := t.loadTasks()
	if err != nil {
		return "", err
	}

	switch strings.ToLower(analysisType) {
	case "priority":
		return t.analyzePriority(taskList)
	case "summary":
		return t.analyzeSummary(taskList)
	case "time":
		return t.analyzeTime(taskList)
	default:
		return "", fmt.Errorf("unknown analysis type: %s (use priority, summary, or time)", analysisType)
	}
}

// analyzePriority provides detailed priority analysis
func (t *TodoListTool) analyzePriority(taskList *TaskList) (string, error) {
	activeTasks := taskList.GetActiveTasks()
	tasksByPriority := taskList.GetTasksByPriority()

	// Count tasks by priority
	priorityCounts := make(map[TaskPriority]int)
	for _, task := range activeTasks {
		priorityCounts[task.Priority]++
	}

	var result strings.Builder
	result.WriteString("## Priority Analysis\n\n")

	// Most important tasks
	if len(tasksByPriority) > 0 {
		result.WriteString("### Most Important Tasks:\n")
		criticalCount := 0
		for _, task := range tasksByPriority {
			if task.Priority == PriorityCritical {
				criticalCount++
				result.WriteString(fmt.Sprintf("- **%s** (Critical)\n", task.Description))
			}
		}
		if criticalCount == 0 {
			// Show top 3 highest priority tasks
			limit := 3
			if len(tasksByPriority) < limit {
				limit = len(tasksByPriority)
			}
			for i := 0; i < limit; i++ {
				task := tasksByPriority[i]
				result.WriteString(fmt.Sprintf("- **%s** (%s)\n", task.Description, task.Priority.String()))
			}
		}
		result.WriteString("\n")
	}

	// Priority distribution
	result.WriteString("### Priority Distribution:\n")
	for _, priority := range []TaskPriority{PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow} {
		count := priorityCounts[priority]
		if count > 0 {
			percentage := float64(count) / float64(len(activeTasks)) * 100
			result.WriteString(fmt.Sprintf("- %s: %d tasks (%.1f%%)\n", priority.String(), count, percentage))
		}
	}

	return result.String(), nil
}

// analyzeSummary provides a comprehensive task summary
func (t *TodoListTool) analyzeSummary(taskList *TaskList) (string, error) {
	activeTasks := taskList.GetActiveTasks()
	tasksByPriority := taskList.GetTasksByPriority()

	// Calculate statistics
	totalTime := 0
	tasksWithTime := 0
	for _, task := range activeTasks {
		if task.TimeEstimate > 0 {
			tasksWithTime++
			totalTime += task.TimeEstimate
		}
	}

	var result strings.Builder
	result.WriteString("## Task Summary\n\n")

	// Overview
	result.WriteString("### Overview:\n")
	result.WriteString(fmt.Sprintf("- Total Tasks: %d (%d active, %d completed)\n",
		len(taskList.Tasks), len(activeTasks), len(taskList.Tasks)-len(activeTasks)))

	if totalTime > 0 {
		hours := totalTime / 60
		minutes := totalTime % 60
		result.WriteString(fmt.Sprintf("- Estimated Time: %dh %dm for %d tasks\n", hours, minutes, tasksWithTime))
		if tasksWithTime < len(activeTasks) {
			result.WriteString(fmt.Sprintf("- Tasks without time estimates: %d\n", len(activeTasks)-tasksWithTime))
		}
	}
	result.WriteString("\n")

	// Top priorities
	if len(tasksByPriority) > 0 {
		result.WriteString("### Current Focus (Top 3 Priorities):\n")
		limit := 3
		if len(tasksByPriority) < limit {
			limit = len(tasksByPriority)
		}
		for i := 0; i < limit; i++ {
			task := tasksByPriority[i]
			timeStr := ""
			if task.TimeEstimate > 0 {
				timeStr = fmt.Sprintf(" - %d minutes", task.TimeEstimate)
			}
			result.WriteString(fmt.Sprintf("%d. %s [%s]%s\n", i+1, task.Description, task.Priority.String(), timeStr))
		}
		result.WriteString("\n")
	}

	// Recent additions
	if len(activeTasks) > 0 {
		result.WriteString("### Recently Added:\n")
		// Sort by creation time (newest first)
		recentTasks := make([]Task, len(activeTasks))
		copy(recentTasks, activeTasks)
		for i := 0; i < len(recentTasks)-1; i++ {
			for j := i + 1; j < len(recentTasks); j++ {
				if recentTasks[j].CreatedAt.After(recentTasks[i].CreatedAt) {
					recentTasks[i], recentTasks[j] = recentTasks[j], recentTasks[i]
				}
			}
		}

		limit := 3
		if len(recentTasks) < limit {
			limit = len(recentTasks)
		}
		for i := 0; i < limit; i++ {
			task := recentTasks[i]
			result.WriteString(fmt.Sprintf("- %s (added %s)\n", task.Description,
				task.CreatedAt.Format("Jan 2, 3:04 PM")))
		}
	}

	return result.String(), nil
}

// analyzeTime provides time-based analysis
func (t *TodoListTool) analyzeTime(taskList *TaskList) (string, error) {
	activeTasks := taskList.GetActiveTasks()

	// Group tasks by time estimates
	quickTasks := []Task{}      // < 30 minutes
	mediumTasks := []Task{}     // 30-60 minutes
	longTasks := []Task{}       // > 60 minutes
	noEstimateTasks := []Task{} // No time estimate

	totalTime := 0
	for _, task := range activeTasks {
		if task.TimeEstimate == 0 {
			noEstimateTasks = append(noEstimateTasks, task)
		} else if task.TimeEstimate < 30 {
			quickTasks = append(quickTasks, task)
			totalTime += task.TimeEstimate
		} else if task.TimeEstimate <= 60 {
			mediumTasks = append(mediumTasks, task)
			totalTime += task.TimeEstimate
		} else {
			longTasks = append(longTasks, task)
			totalTime += task.TimeEstimate
		}
	}

	var result strings.Builder
	result.WriteString("## Time Analysis\n\n")

	// Total time summary
	if totalTime > 0 {
		hours := totalTime / 60
		minutes := totalTime % 60
		result.WriteString(fmt.Sprintf("### Total Estimated Time: %dh %dm\n\n", hours, minutes))
	}

	// Quick wins
	if len(quickTasks) > 0 {
		result.WriteString("### Quick Wins (< 30 minutes):\n")
		for _, task := range quickTasks {
			result.WriteString(fmt.Sprintf("- %s (%d min) [%s]\n",
				task.Description, task.TimeEstimate, task.Priority.String()))
		}
		result.WriteString("\n")
	}

	// Medium tasks
	if len(mediumTasks) > 0 {
		result.WriteString("### Medium Tasks (30-60 minutes):\n")
		for _, task := range mediumTasks {
			result.WriteString(fmt.Sprintf("- %s (%d min) [%s]\n",
				task.Description, task.TimeEstimate, task.Priority.String()))
		}
		result.WriteString("\n")
	}

	// Long tasks
	if len(longTasks) > 0 {
		result.WriteString("### Long Tasks (> 1 hour):\n")
		for _, task := range longTasks {
			hours := task.TimeEstimate / 60
			minutes := task.TimeEstimate % 60
			result.WriteString(fmt.Sprintf("- %s (%dh %dm) [%s]\n",
				task.Description, hours, minutes, task.Priority.String()))
		}
		result.WriteString("\n")
	}

	// Tasks without estimates
	if len(noEstimateTasks) > 0 {
		result.WriteString("### Tasks Without Time Estimates:\n")
		for _, task := range noEstimateTasks {
			result.WriteString(fmt.Sprintf("- %s [%s]\n", task.Description, task.Priority.String()))
		}
	}

	return result.String(), nil
}
