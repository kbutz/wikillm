package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImprovedTodoListTool tests the enhanced functionality
func TestImprovedTodoListTool(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "todo-improved-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary todo file
	todoFilePath := filepath.Join(tempDir, "todo-test.json")

	// Create a new ImprovedTodoListTool with the temporary file
	tool := NewImprovedTodoListTool(todoFilePath)

	// Test adding tasks with priority and time
	t.Run("AddTaskWithMetadata", func(t *testing.T) {
		ctx := context.Background()
		
		// Add task with high priority and time estimate
		result, err := tool.Execute(ctx, "add Buy groceries priority:high time:30m")
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
		if !strings.Contains(result, "Buy groceries") {
			t.Errorf("Result doesn't contain task description: %s", result)
		}
		
		// Add task with just priority
		result, err = tool.Execute(ctx, "add Call dentist priority:critical")
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
		
		// Add task with time in hours
		result, err = tool.Execute(ctx, "add Study for exam time:2h")
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
		
		// Verify tasks were saved correctly
		tasks, err := tool.loadTasks()
		if err != nil {
			t.Fatalf("Failed to load tasks: %v", err)
		}
		if len(tasks.Tasks) != 3 {
			t.Errorf("Expected 3 tasks, got %d", len(tasks.Tasks))
		}
		
		// Check first task
		if tasks.Tasks[0].Description != "Buy groceries" {
			t.Errorf("Expected 'Buy groceries', got '%s'", tasks.Tasks[0].Description)
		}
		if tasks.Tasks[0].Priority != PriorityHigh {
			t.Errorf("Expected High priority, got %s", tasks.Tasks[0].Priority.String())
		}
		if tasks.Tasks[0].TimeEstimate != 30 {
			t.Errorf("Expected 30 minutes, got %d", tasks.Tasks[0].TimeEstimate)
		}
		
		// Check second task
		if tasks.Tasks[1].Priority != PriorityCritical {
			t.Errorf("Expected Critical priority, got %s", tasks.Tasks[1].Priority.String())
		}
		
		// Check third task
		if tasks.Tasks[2].TimeEstimate != 120 { // 2 hours = 120 minutes
			t.Errorf("Expected 120 minutes, got %d", tasks.Tasks[2].TimeEstimate)
		}
	})

	// Test listing by priority
	t.Run("ListByPriority", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := tool.Execute(ctx, "list priority")
		if err != nil {
			t.Fatalf("Failed to list tasks by priority: %v", err)
		}
		
		// Should show critical task first
		lines := strings.Split(result, "\n")
		foundCritical := false
		for i, line := range lines {
			if strings.Contains(line, "Call dentist") && strings.Contains(line, "[Critical]") {
				foundCritical = true
				// Make sure it's in the first few lines (accounting for header)
				if i > 5 {
					t.Error("Critical task should appear near the top")
				}
				break
			}
		}
		if !foundCritical {
			t.Error("Critical task not found in priority listing")
		}
	})

	// Test completing tasks
	t.Run("CompleteTask", func(t *testing.T) {
		ctx := context.Background()
		
		// Complete the first active task
		result, err := tool.Execute(ctx, "complete 1")
		if err != nil {
			t.Fatalf("Failed to complete task: %v", err)
		}
		if !strings.Contains(result, "Completed task") {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Verify task was marked as completed
		tasks, err := tool.loadTasks()
		if err != nil {
			t.Fatalf("Failed to load tasks: %v", err)
		}
		
		completedCount := 0
		for _, task := range tasks.Tasks {
			if task.Completed {
				completedCount++
			}
		}
		if completedCount != 1 {
			t.Errorf("Expected 1 completed task, got %d", completedCount)
		}
		
		// List should now show only 2 active tasks
		result, err = tool.Execute(ctx, "list")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		if !strings.Contains(result, "2 active tasks") {
			t.Errorf("Expected 2 active tasks in summary, got: %s", result)
		}
	})

	// Test clearing completed tasks
	t.Run("ClearCompleted", func(t *testing.T) {
		ctx := context.Background()
		
		// Clear only completed tasks
		result, err := tool.Execute(ctx, "clear completed")
		if err != nil {
			t.Fatalf("Failed to clear completed tasks: %v", err)
		}
		if !strings.Contains(result, "Cleared all completed tasks") {
			t.Errorf("Unexpected result: %s", result)
		}
		
		// Verify only active tasks remain
		tasks, err := tool.loadTasks()
		if err != nil {
			t.Fatalf("Failed to load tasks: %v", err)
		}
		
		if len(tasks.Tasks) != 2 {
			t.Errorf("Expected 2 tasks after clearing completed, got %d", len(tasks.Tasks))
		}
		
		for _, task := range tasks.Tasks {
			if task.Completed {
				t.Error("Found completed task after clearing completed tasks")
			}
		}
	})

	// Test time summary
	t.Run("TimeSummary", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := tool.Execute(ctx, "list")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}
		
		// Should show total time estimate
		if !strings.Contains(result, "total time") {
			t.Error("Expected time summary in listing")
		}
	})

	// Test migration from old format
	t.Run("MigrateOldFormat", func(t *testing.T) {
		// Create a new tool with a different file
		oldFormatFile := filepath.Join(tempDir, "old-format.txt")
		
		// Write old format data
		oldData := "Task 1\nTask 2\nTask 3"
		err := os.WriteFile(oldFormatFile, []byte(oldData), 0644)
		if err != nil {
			t.Fatalf("Failed to write old format file: %v", err)
		}
		
		// Create tool and load tasks
		oldTool := NewImprovedTodoListTool(oldFormatFile)
		tasks, err := oldTool.loadTasks()
		if err != nil {
			t.Fatalf("Failed to migrate old format: %v", err)
		}
		
		if len(tasks.Tasks) != 3 {
			t.Errorf("Expected 3 migrated tasks, got %d", len(tasks.Tasks))
		}
		
		// Verify tasks have default values
		for i, task := range tasks.Tasks {
			expectedDesc := fmt.Sprintf("Task %d", i+1)
			if task.Description != expectedDesc {
				t.Errorf("Expected '%s', got '%s'", expectedDesc, task.Description)
			}
			if task.Priority != PriorityMedium {
				t.Error("Migrated tasks should have medium priority")
			}
			if task.TimeEstimate != 0 {
				t.Error("Migrated tasks should have no time estimate")
			}
		}
	})
}

// TestExtractToolCallImproved tests the improved JSON extraction
func TestExtractToolCallImproved(t *testing.T) {
	testCases := []struct {
		name     string
		response string
		expected ToolCall
		found    bool
	}{
		{
			name: "JSON in code block",
			response: `I'll add that task for you.

` + "```json\n{\n  \"tool\": \"todo_list\",\n  \"args\": \"add Buy milk priority:high time:15m\"\n}\n```",
			expected: ToolCall{Tool: "todo_list", Args: "add Buy milk priority:high time:15m"},
			found:    true,
		},
		{
			name:     "Raw JSON",
			response: `I'll help with that. {"tool": "todo_list", "args": "list priority"}`,
			expected: ToolCall{Tool: "todo_list", Args: "list priority"},
			found:    true,
		},
		{
			name: "Args as object with command",
			response: `{"tool": "todo_list", "args": {"command": "add", "task": "Call mom", "priority": "high"}}`,
			expected: ToolCall{Tool: "todo_list", Args: "add Call mom priority:high"},
			found:    true,
		},
		{
			name:     "Args with task field only",
			response: `{"tool": "todo_list", "args": {"task": "Water plants"}}`,
			expected: ToolCall{Tool: "todo_list", Args: "add Water plants"},
			found:    true,
		},
		{
			name:     "Arguments instead of args",
			response: `{"tool": "todo_list", "arguments": "list all"}`,
			expected: ToolCall{Tool: "todo_list", Args: "list all"},
			found:    true,
		},
		{
			name:     "Empty args",
			response: `{"tool": "todo_list", "args": ""}`,
			expected: ToolCall{Tool: "todo_list", Args: ""},
			found:    true,
		},
		{
			name:     "No JSON",
			response: `I don't think you need to add anything to your todo list right now.`,
			expected: ToolCall{},
			found:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, found := extractToolCall(tc.response)
			
			if found != tc.found {
				t.Errorf("Expected found=%v, got %v", tc.found, found)
			}
			
			if found && (result.Tool != tc.expected.Tool || result.Args != tc.expected.Args) {
				t.Errorf("Expected %+v, got %+v", tc.expected, result)
			}
		})
	}
}

// TestTaskPriority tests priority parsing
func TestTaskPriority(t *testing.T) {
	testCases := []struct {
		input    string
		expected TaskPriority
	}{
		{"low", PriorityLow},
		{"Low", PriorityLow},
		{"LOW", PriorityLow},
		{"1", PriorityLow},
		{"medium", PriorityMedium},
		{"med", PriorityMedium},
		{"2", PriorityMedium},
		{"high", PriorityHigh},
		{"HIGH", PriorityHigh},
		{"3", PriorityHigh},
		{"critical", PriorityCritical},
		{"crit", PriorityCritical},
		{"4", PriorityCritical},
		{"unknown", PriorityMedium}, // Default
		{"", PriorityMedium},         // Default
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ParsePriority(tc.input)
			if result != tc.expected {
				t.Errorf("ParsePriority(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestParseAddCommand tests the command parsing
func TestParseAddCommand(t *testing.T) {
	tool := NewImprovedTodoListTool("test.json")
	
	testCases := []struct {
		name         string
		args         string
		expDesc      string
		expPriority  TaskPriority
		expTime      int
	}{
		{
			name:        "Simple task",
			args:        "add Buy groceries",
			expDesc:     "Buy groceries",
			expPriority: PriorityMedium,
			expTime:     0,
		},
		{
			name:        "Task with priority",
			args:        "add Call doctor priority:high",
			expDesc:     "Call doctor",
			expPriority: PriorityHigh,
			expTime:     0,
		},
		{
			name:        "Task with time in minutes",
			args:        "add Clean house time:45m",
			expDesc:     "Clean house",
			expPriority: PriorityMedium,
			expTime:     45,
		},
		{
			name:        "Task with time in hours",
			args:        "add Write report time:3h",
			expDesc:     "Write report",
			expPriority: PriorityMedium,
			expTime:     180,
		},
		{
			name:        "Task with priority and time",
			args:        "add Prepare presentation priority:critical time:2h",
			expDesc:     "Prepare presentation",
			expPriority: PriorityCritical,
			expTime:     120,
		},
		{
			name:        "Task with metadata in different order",
			args:        "add time:30m Study for test priority:high",
			expDesc:     "Study for test",
			expPriority: PriorityHigh,
			expTime:     30,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			desc, priority, time := tool.parseAddCommand(tc.args)
			
			if desc != tc.expDesc {
				t.Errorf("Description: got %q, want %q", desc, tc.expDesc)
			}
			if priority != tc.expPriority {
				t.Errorf("Priority: got %v, want %v", priority, tc.expPriority)
			}
			if time != tc.expTime {
				t.Errorf("Time: got %d, want %d", time, tc.expTime)
			}
		})
	}
}
