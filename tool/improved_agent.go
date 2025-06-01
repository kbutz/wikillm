package main

import (
	"context"
	"fmt"
	"github.com/kbutz/wikillm/tool/tools"
)

// ImprovedAgent represents an enhanced LLM agent with better response handling
type ImprovedAgent struct {
	*Agent
	todoService *tools.TodoListService
}

// NewImprovedAgent creates a new improved agent with direct TODO service access
func NewImprovedAgent(model LLMModel, todoFilePath string) *ImprovedAgent {
	// Create the todo tool
	todoTool := tools.NewImprovedTodoListTool(todoFilePath)

	// Create base agent
	baseAgent := NewAgent(model, []Tool{todoTool})

	// Create the service
	todoService := tools.NewTodoListService(todoFilePath)

	return &ImprovedAgent{
		Agent:       baseAgent,
		todoService: todoService,
	}
}

// ProcessQuery processes queries with improved handling for TODO queries
func (ia *ImprovedAgent) ProcessQuery(ctx context.Context, query string) (string, error) {
	// Parse the query to understand intent
	structured, err := ia.todoService.ParseQuery(query)
	if err != nil {
		fmt.Println(err)
		// Fall back to base agent processing
		return ia.Agent.ProcessQuery(ctx, query)
	}

	// Handle known intents directly
	switch structured.Intent {
	case "get_most_important":
		return ia.todoService.GetMostImportantTask()

	case "get_by_difficulty":
		ascending := true
		if val, ok := structured.Parameters["ascending"].(bool); ok {
			ascending = val
		}
		return ia.todoService.GetTasksByDifficulty(ascending)

	case "get_summary":
		return ia.todoService.GetTaskSummary()

	case "list_tasks":
		return ia.todoService.ExecuteCommand(ctx, "list")

	default:
		// Fall back to regular agent processing for unknown intents
		return ia.Agent.ProcessQuery(ctx, query)
	}
}

// QueryWithContext provides a way to query with additional context
func (ia *ImprovedAgent) QueryWithContext(ctx context.Context, query string, context map[string]interface{}) (string, error) {
	// This could be extended to provide context-aware responses
	// For now, just use regular processing
	return ia.ProcessQuery(ctx, query)
}
