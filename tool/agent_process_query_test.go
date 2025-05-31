package main

import (
	"context"
	"strings"
	"testing"
)

// MockLLMModel is a mock implementation of the LLMModel interface for testing
type MockLLMModel struct {
	name      string
	responses map[string]string
}

// NewMockLLMModel creates a new MockLLMModel with predefined responses
func NewMockLLMModel(responses map[string]string) *MockLLMModel {
	return &MockLLMModel{
		name:      "mock-model",
		responses: responses,
	}
}

// Query returns a predefined response based on the prompt
func (m *MockLLMModel) Query(ctx context.Context, prompt string) (string, error) {
	// Check if we have an exact match for the prompt
	if response, ok := m.responses[prompt]; ok {
		return response, nil
	}

	// If no exact match, try to find a partial match
	for key, response := range m.responses {
		if strings.Contains(prompt, key) {
			return response, nil
		}
	}

	// Default response if no match is found
	return "I don't know how to respond to that.", nil
}

// Name returns the name of the model
func (m *MockLLMModel) Name() string {
	return m.name
}

// MockTool is a mock implementation of the Tool interface for testing
type MockTool struct {
	name        string
	description string
	responses   map[string]string
	executions  []string // Track the arguments passed to Execute
}

// NewMockTool creates a new MockTool with predefined responses
func NewMockTool(name, description string, responses map[string]string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
		responses:   responses,
		executions:  []string{},
	}
}

// Name returns the name of the tool
func (t *MockTool) Name() string {
	return t.name
}

// Description returns a description of what the tool does
func (t *MockTool) Description() string {
	return t.description
}

// Execute returns a predefined response based on the args
func (t *MockTool) Execute(ctx context.Context, args string) (string, error) {
	// Track the execution
	t.executions = append(t.executions, args)

	// Check if we have an exact match for the args
	if response, ok := t.responses[args]; ok {
		return response, nil
	}

	// If no exact match, try to find a partial match
	for key, response := range t.responses {
		if strings.Contains(args, key) {
			return response, nil
		}
	}

	// Default response if no match is found
	return "Tool executed with args: " + args, nil
}

// TestAgentProcessQuery tests the ProcessQuery method of the Agent
func TestAgentProcessQuery(t *testing.T) {
	// Test direct response (no tool call)
	t.Run("DirectResponse", func(t *testing.T) {
		// Create a mock model that returns a direct response
		mockModel := NewMockLLMModel(map[string]string{
			"query": "This is a direct response with no tool call.",
		})

		// Create a mock tool
		mockTool := NewMockTool("test_tool", "A test tool", map[string]string{
			"test": "Test tool executed",
		})

		// Create an agent with the mock model and tool
		agent := NewAgent(mockModel, []Tool{mockTool})

		// Process a query
		response, err := agent.ProcessQuery(context.Background(), "What is the weather today?")
		if err != nil {
			t.Fatalf("ProcessQuery failed: %v", err)
		}

		// Verify the response
		if response != "This is a direct response with no tool call." {
			t.Errorf("Expected direct response, got: %s", response)
		}

		// Verify the tool was not executed
		if len(mockTool.executions) > 0 {
			t.Errorf("Expected no tool executions, got: %v", mockTool.executions)
		}
	})

	// Test tool call
	t.Run("ToolCall", func(t *testing.T) {
		// Create a mock model that returns a tool call and then a final response
		mockModel := NewMockLLMModel(map[string]string{
			"query": "{\"tool\": \"todo_list\", \"args\": \"add Buy milk\"}",
			"tool":  "I've added 'Buy milk' to your to-do list.",
		})

		// Create a mock todo_list tool
		mockTool := NewMockTool("todo_list", "Manages a to-do list", map[string]string{
			"add Buy milk": "Added task: Buy milk",
		})

		// Create an agent with the mock model and tool
		agent := NewAgent(mockModel, []Tool{mockTool})

		// Process a query
		response, err := agent.ProcessQuery(context.Background(), "Add buy milk to my to-do list")
		if err != nil {
			t.Fatalf("ProcessQuery failed: %v", err)
		}

		// Verify the response
		if response != "Added task: Buy milk" {
			t.Errorf("Expected 'Added task: Buy milk', got: %s", response)
		}

		// Verify the tool was executed with the correct args
		if len(mockTool.executions) != 1 {
			t.Errorf("Expected 1 tool execution, got: %d", len(mockTool.executions))
		} else if mockTool.executions[0] != "add Buy milk" {
			t.Errorf("Expected tool execution with args 'add Buy milk', got: %s", mockTool.executions[0])
		}
	})

	// Test tool call with unknown tool
	t.Run("UnknownTool", func(t *testing.T) {
		// Create a mock model that returns a tool call for an unknown tool
		mockModel := NewMockLLMModel(map[string]string{
			"query": "{\"tool\": \"unknown_tool\", \"args\": \"some args\"}",
		})

		// Create a mock tool with a different name
		mockTool := NewMockTool("todo_list", "Manages a to-do list", map[string]string{
			"add": "Added task",
		})

		// Create an agent with the mock model and tool
		agent := NewAgent(mockModel, []Tool{mockTool})

		// Process a query
		response, err := agent.ProcessQuery(context.Background(), "Use the unknown tool")
		if err != nil {
			t.Fatalf("ProcessQuery failed: %v", err)
		}

		// Verify the response contains the expected error message
		if !strings.Contains(response, "I tried to use the unknown_tool tool, but it's not available") {
			t.Errorf("Expected error message about unknown tool, got: %s", response)
		}

		// Verify the tool was not executed
		if len(mockTool.executions) > 0 {
			t.Errorf("Expected no tool executions, got: %v", mockTool.executions)
		}
	})

	// Test different JSON formats for args
	t.Run("DifferentArgsFormats", func(t *testing.T) {
		// Test cases with different JSON formats for args
		testCases := []struct {
			name          string
			toolResponse  string
			expectedArgs  string
			expectedFinal string
		}{
			{
				name:          "ArgsAsString",
				toolResponse:  "{\"tool\": \"todo_list\", \"args\": \"list\"}",
				expectedArgs:  "list",
				expectedFinal: "To-Do List:\n1. Task 1\n2. Task 2\n",
			},
			{
				name:          "ArgsAsArray",
				toolResponse:  "{\"tool\": \"todo_list\", \"args\": [\"add\", \"New task\"]}",
				expectedArgs:  "add New task",
				expectedFinal: "Added task: New task",
			},
			{
				name:          "ArgsAsObject",
				toolResponse:  "{\"tool\": \"todo_list\", \"args\": {\"task\": \"Important task\"}}",
				expectedArgs:  "add Important task",
				expectedFinal: "Added task: Important task",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a mock model that returns the test case's tool response
				mockModel := NewMockLLMModel(map[string]string{
					"query": tc.toolResponse,
				})

				// Create a mock tool that expects the test case's args
				mockTool := NewMockTool("todo_list", "Manages a to-do list", map[string]string{
					tc.expectedArgs: tc.expectedFinal,
				})

				// Create an agent with the mock model and tool
				agent := NewAgent(mockModel, []Tool{mockTool})

				// Process a query
				response, err := agent.ProcessQuery(context.Background(), "Test query")
				if err != nil {
					t.Fatalf("ProcessQuery failed: %v", err)
				}

				// Verify the response
				if response != tc.expectedFinal {
					t.Errorf("Expected '%s', got: '%s'", tc.expectedFinal, response)
				}

				// Verify the tool was executed with the correct args
				if len(mockTool.executions) != 1 {
					t.Errorf("Expected 1 tool execution, got: %d", len(mockTool.executions))
				} else if mockTool.executions[0] != tc.expectedArgs {
					t.Errorf("Expected tool execution with args '%s', got: '%s'", tc.expectedArgs, mockTool.executions[0])
				}
			})
		}
	})
}