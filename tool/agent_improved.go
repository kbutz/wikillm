package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ImprovedTodoListTool implements the Tool interface with enhanced functionality
type ImprovedTodoListTool struct {
	filePath string
}

// NewImprovedTodoListTool creates a new ImprovedTodoListTool
func NewImprovedTodoListTool(filePath string) *ImprovedTodoListTool {
	return &ImprovedTodoListTool{
		filePath: filePath,
	}
}

// Name returns the name of the tool
func (t *ImprovedTodoListTool) Name() string {
	return "todo_list"
}

// Description returns a description of what the tool does
func (t *ImprovedTodoListTool) Description() string {
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

// Execute runs the tool with the given arguments and returns the result
func (t *ImprovedTodoListTool) Execute(ctx context.Context, args string) (string, error) {
	// Parse the command
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
func (t *ImprovedTodoListTool) parseAddCommand(args string) (description string, priority TaskPriority, timeEstimate int) {
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
func (t *ImprovedTodoListTool) addTask(args string) (string, error) {
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
func (t *ImprovedTodoListTool) listTasks(listType string) (string, error) {
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
func (t *ImprovedTodoListTool) completeTask(indexStr string) (string, error) {
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
func (t *ImprovedTodoListTool) removeTask(indexStr string) (string, error) {
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
func (t *ImprovedTodoListTool) clearTasks(clearType string) (string, error) {
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
func (t *ImprovedTodoListTool) loadTasks() (*TaskList, error) {
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
		// Try to migrate from old format
		if migratedList := t.migrateOldFormat(data); migratedList != nil {
			return migratedList, nil
		}
		return nil, fmt.Errorf("error parsing tasks: %w", err)
	}

	return taskList, nil
}

// migrateOldFormat attempts to migrate from the old text format
func (t *ImprovedTodoListTool) migrateOldFormat(data []byte) *TaskList {
	lines := strings.Split(string(data), "\n")
	taskList := &TaskList{Tasks: []Task{}}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			task := NewTask(line)
			taskList.Add(*task)
		}
	}

	return taskList
}

// saveTasks saves tasks to the JSON file
func (t *ImprovedTodoListTool) saveTasks(taskList *TaskList) error {
	data, err := taskList.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling tasks: %w", err)
	}

	if err := os.WriteFile(t.filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing tasks file: %w", err)
	}

	return nil
}
