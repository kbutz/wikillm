package tools

import "testing"

// TestExtractToolCall tests the ExtractToolCall function with different JSON structures
func TestExtractToolCall(t *testing.T) {
	// Test with args as a string
	t.Run("ArgsAsString", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"todo_list\", \"args\": \"add Buy milk\"}\n\nThis will add \"Buy milk\" to your to-do list."

		toolCall, found := ExtractToolCall(response)
		if !found {
			t.Fatal("Failed to extract tool call")
		}

		if toolCall.Tool != "todo_list" {
			t.Errorf("Expected tool 'todo_list', got '%s'", toolCall.Tool)
		}

		if toolCall.Args != "add Buy milk" {
			t.Errorf("Expected args 'add Buy milk', got '%s'", toolCall.Args)
		}
	})

	// Test with args as an array
	t.Run("ArgsAsArray", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"todo_list\", \"args\": [\"add\", \"Schedule dentist appointment\"]}\n\nThis will add a reminder to schedule a dentist appointment."

		toolCall, found := ExtractToolCall(response)
		if !found {
			t.Fatal("Failed to extract tool call")
		}

		if toolCall.Tool != "todo_list" {
			t.Errorf("Expected tool 'todo_list', got '%s'", toolCall.Tool)
		}

		if toolCall.Args != "add Schedule dentist appointment" {
			t.Errorf("Expected args 'add Schedule dentist appointment', got '%s'", toolCall.Args)
		}
	})

	// Test with args as an object with task field
	t.Run("ArgsAsObjectWithTask", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"todo_list\", \"args\": {\"task\": \"Call mom on her birthday\"}}\n\nThis will add a reminder to call your mom on her birthday."

		toolCall, found := ExtractToolCall(response)
		if !found {
			t.Fatal("Failed to extract tool call")
		}

		if toolCall.Tool != "todo_list" {
			t.Errorf("Expected tool 'todo_list', got '%s'", toolCall.Tool)
		}

		if toolCall.Args != "add Call mom on her birthday" {
			t.Errorf("Expected args 'add Call mom on her birthday', got '%s'", toolCall.Args)
		}
	})

	// Test with args as an object without task field
	t.Run("ArgsAsObjectWithoutTask", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"todo_list\", \"args\": {\"command\": \"add\", \"content\": \"Buy dog food\"}}\n\nThis will add \"Buy dog food\" to your to-do list."

		toolCall, found := ExtractToolCall(response)
		if !found {
			t.Fatal("Failed to extract tool call")
		}

		if toolCall.Tool != "todo_list" {
			t.Errorf("Expected tool 'todo_list', got '%s'", toolCall.Tool)
		}

		// The exact order might vary, but both values should be in the args
		if toolCall.Args != "add Buy dog food" && toolCall.Args != "Buy dog food add" {
			t.Errorf("Expected args to contain 'add' and 'Buy dog food', got '%s'", toolCall.Args)
		}
	})

	// Test with invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"todo_list\", \"args\": \"add Buy milk\"\n\nThis will add \"Buy milk\" to your to-do list."

		_, found := ExtractToolCall(response)
		if found {
			t.Error("Expected not to find tool call with invalid JSON, but one was found")
		}
	})

	// Test with missing tool field
	t.Run("MissingToolField", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"args\": \"add Buy milk\"}\n\nThis will add \"Buy milk\" to your to-do list."

		_, found := ExtractToolCall(response)
		if found {
			t.Error("Expected not to find tool call with missing tool field, but one was found")
		}
	})

	// Test with empty tool field
	t.Run("EmptyToolField", func(t *testing.T) {
		response := "I'll help you with that. Let me use the todo_list tool.\n\n{\"tool\": \"\", \"args\": \"add Buy milk\"}\n\nThis will add \"Buy milk\" to your to-do list."

		_, found := ExtractToolCall(response)
		if found {
			t.Error("Expected not to find tool call with empty tool field, but one was found")
		}
	})

	// Test with no JSON in response
	t.Run("NoJSON", func(t *testing.T) {
		response := "I'll help you with that. Let me add \"Buy milk\" to your to-do list."

		_, found := ExtractToolCall(response)
		if found {
			t.Error("Expected not to find tool call with no JSON, but one was found")
		}
	})
}
