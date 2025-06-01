package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

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
