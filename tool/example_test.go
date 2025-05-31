package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// ExampleImprovedAgent demonstrates how to use the improved TODO agent
func ExampleImprovedAgent() {
	// Mock model for demonstration
	model := &MockModel{}
	
	// Create the improved TODO tool
	todoTool := NewImprovedTodoListTool("example-todos.json")
	
	// Create agent with the improved tool
	agent := NewAgent(model, []Tool{todoTool})
	
	// Example queries
	queries := []string{
		"Add 'Buy groceries' to my todo list with high priority and estimate 30 minutes",
		"I need to call the dentist urgently",
		"Show me my tasks sorted by priority",
		"Mark the first task as complete",
		"What's on my todo list?",
		"Clear all completed tasks",
	}
	
	ctx := context.Background()
	
	for _, query := range queries {
		fmt.Printf("\n>>> User: %s\n", query)
		
		response, err := agent.ProcessQuery(ctx, query)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Printf("<<< Agent: %s\n", response)
	}
}

// MockModel is a simple mock for testing
type MockModel struct{}

func (m *MockModel) Name() string {
	return "mock-model"
}

func (m *MockModel) Query(ctx context.Context, prompt string) (string, error) {
	// Simulate LLM responses based on the prompt
	if strings.Contains(prompt, "Buy groceries") && strings.Contains(prompt, "high priority") {
		return `I'll add that task to your todo list with high priority and a 30-minute time estimate.

` + "```json\n{\n  \"tool\": \"todo_list\",\n  \"args\": \"add Buy groceries priority:high time:30m\"\n}\n```", nil
	}
	
	if strings.Contains(prompt, "dentist urgently") {
		return `I'll add calling the dentist as a critical priority task.

` + "```json\n{\n  \"tool\": \"todo_list\",\n  \"args\": \"add Call the dentist priority:critical\"\n}\n```", nil
	}
	
	if strings.Contains(prompt, "sorted by priority") {
		return `I'll show you your tasks sorted by priority.

{"tool": "todo_list", "args": "list priority"}`, nil
	}
	
	if strings.Contains(prompt, "Mark the first task as complete") {
		return `{"tool": "todo_list", "args": "complete 1"}`, nil
	}
	
	if strings.Contains(prompt, "What's on my todo list") {
		return `Let me show you your current tasks.

{"tool": "todo_list", "args": "list"}`, nil
	}
	
	if strings.Contains(prompt, "Clear all completed") {
		return `{"tool": "todo_list", "args": "clear completed"}`, nil
	}
	
	// For follow-up responses after tool execution
	if strings.Contains(prompt, "The tool returned this result:") {
		if strings.Contains(prompt, "Added task:") {
			return "I've successfully added that task to your todo list.", nil
		}
		if strings.Contains(prompt, "Completed task:") {
			return "I've marked that task as complete.", nil
		}
		if strings.Contains(prompt, "Cleared all completed tasks") {
			return "I've removed all completed tasks from your list.", nil
		}
	}
	
	return "I understand your request but I'm not sure how to help with that.", nil
}
