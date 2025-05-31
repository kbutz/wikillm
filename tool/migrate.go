package main

import (
	"fmt"
	"os"
)

// MigrateTodoFile converts an old text-based todo file to the new JSON format
func MigrateTodoFile(oldPath, newPath string) error {
	// Check if old file exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("old todo file not found: %s", oldPath)
	}

	// Check if new file already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("new todo file already exists: %s", newPath)
	}

	// Create temporary tool to load from old format
	tempTool := NewImprovedTodoListTool(oldPath)
	tasks, err := tempTool.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks from old format: %w", err)
	}

	// Save to new location
	finalTool := NewImprovedTodoListTool(newPath)
	if err := finalTool.saveTasks(tasks); err != nil {
		return fmt.Errorf("failed to save tasks in new format: %w", err)
	}

	fmt.Printf("Successfully migrated %d tasks from %s to %s\n",
		len(tasks.Tasks), oldPath, newPath)

	return nil
}

// MigrateInPlace migrates a todo file from old format to new format in place
func MigrateInPlace(todoPath string) error {
	// Create backup of original file
	backupPath := todoPath + ".backup"

	// Read original content
	originalContent, err := os.ReadFile(todoPath)
	if err != nil {
		return fmt.Errorf("failed to read original file: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, originalContent, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Load tasks using the tool (will auto-migrate)
	tool := NewImprovedTodoListTool(todoPath)
	tasks, err := tool.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Save in new format
	if err := tool.saveTasks(tasks); err != nil {
		// Restore backup on failure
		os.WriteFile(todoPath, originalContent, 0644)
		return fmt.Errorf("failed to save in new format: %w", err)
	}

	fmt.Printf("Successfully migrated %d tasks. Backup saved to %s\n",
		len(tasks.Tasks), backupPath)

	return nil
}

// Example migration program
func ExampleMigration() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate <todo-file-path>")
		fmt.Println("This will convert your old text-based todo file to the new JSON format")
		fmt.Println("A backup will be created with .backup extension")
		return
	}

	todoPath := os.Args[1]

	if err := MigrateInPlace(todoPath); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}
}
