package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestTodoListTool tests the functionality of the TodoListTool
func TestTodoListTool(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "todo-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary todo file
	todoFilePath := filepath.Join(tempDir, "todo-test.txt")

	// Create a new TodoListTool with the temporary file
	tool := NewTodoListTool(todoFilePath)

	// Test adding tasks
	t.Run("AddTask", func(t *testing.T) {
		ctx := context.Background()
		
		// Add first task
		result, err := tool.Execute(ctx, "add Buy groceries")
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
		if result != "Added task: Buy groceries" {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Add second task
		result, err = tool.Execute(ctx, "add Walk the dog")
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
		if result != "Added task: Walk the dog" {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Verify tasks were added correctly
		tasks, err := tool.readTasks()
		if err != nil {
			t.Fatalf("Failed to read tasks: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(tasks))
		}
		if tasks[0] != "Buy groceries" {
			t.Errorf("Expected 'Buy groceries', got '%s'", tasks[0])
		}
		if tasks[1] != "Walk the dog" {
			t.Errorf("Expected 'Walk the dog', got '%s'", tasks[1])
		}
	})

	// Test listing tasks
	t.Run("ListTasks", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := tool.Execute(ctx, "list")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		
		expected := "To-Do List:\n1. Buy groceries\n2. Walk the dog\n"
		if result != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
		}
	})

	// Test removing a task
	t.Run("RemoveTask", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := tool.Execute(ctx, "remove 1")
		if err != nil {
			t.Fatalf("Failed to remove task: %v", err)
		}
		if result != "Removed task: Buy groceries" {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Verify task was removed
		tasks, err := tool.readTasks()
		if err != nil {
			t.Fatalf("Failed to read tasks: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
		if tasks[0] != "Walk the dog" {
			t.Errorf("Expected 'Walk the dog', got '%s'", tasks[0])
		}
	})

	// Test clearing tasks
	t.Run("ClearTasks", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := tool.Execute(ctx, "clear")
		if err != nil {
			t.Fatalf("Failed to clear tasks: %v", err)
		}
		if result != "Cleared all tasks from the to-do list." {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Verify tasks were cleared
		tasks, err := tool.readTasks()
		if err != nil {
			t.Fatalf("Failed to read tasks: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("Expected 0 tasks, got %d", len(tasks))
		}
	})

	// Test error cases
	t.Run("ErrorCases", func(t *testing.T) {
		ctx := context.Background()
		
		// Test invalid command
		_, err := tool.Execute(ctx, "invalid")
		if err == nil {
			t.Error("Expected error for invalid command, got nil")
		}
		
		// Test add with no task
		_, err = tool.Execute(ctx, "add")
		if err == nil {
			t.Error("Expected error for add with no task, got nil")
		}
		
		// Test remove with no number
		_, err = tool.Execute(ctx, "remove")
		if err == nil {
			t.Error("Expected error for remove with no number, got nil")
		}
		
		// Test remove with invalid number
		_, err = tool.Execute(ctx, "remove abc")
		if err == nil {
			t.Error("Expected error for remove with invalid number, got nil")
		}
		
		// Test remove with out of range number
		_, err = tool.Execute(ctx, "remove 999")
		if err == nil {
			t.Error("Expected error for remove with out of range number, got nil")
		}
	})
}