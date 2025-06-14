package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// FileMemoryTool implements the Tool interface for reading and writing to a file
// to store conversation history and other information
type FileMemoryTool struct {
	filePath string
}

// NewFileMemoryTool creates a new FileMemoryTool
func NewFileMemoryTool(filePath string) *FileMemoryTool {
	return &FileMemoryTool{
		filePath: filePath,
	}
}

// Name returns the name of the tool
func (t *FileMemoryTool) Name() string {
	return "file_memory"
}

// Description returns a description of what the tool does
func (t *FileMemoryTool) Description() string {
	return `Reads from and writes to a file to store conversation history and other information.
Commands:
- read - Read the entire contents of the file
- write <content> - Write content to the file (replaces existing content)
- append <content> - Append content to the file
- clear - Clear the file contents

Examples:
- read
- write This is some new content
- append And this is additional content
- clear`
}

// Parameters returns the parameter schema for the tool
func (t *FileMemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute (read, write, append, clear)",
				"enum":        []string{"read", "write", "append", "clear"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write or append to the file",
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

// Execute runs the tool with the given arguments and returns the result
func (t *FileMemoryTool) Execute(ctx context.Context, args string) (string, error) {
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
			case "read":
				return t.readFile()
			case "write", "append":
				// Extract content
				contentInterface, hasContent := params["content"]
				if !hasContent {
					return "", fmt.Errorf("content parameter is required for %s command", command)
				}

				content, ok := contentInterface.(string)
				if !ok {
					return "", fmt.Errorf("content must be a string")
				}

				if command == "write" {
					return t.writeFile(content)
				}
				return t.appendFile(content)
			case "clear":
				return t.clearFile()
			default:
				return "", fmt.Errorf("unknown command: %s", command)
			}
		}
	}

	// Fall back to parsing command from string
	parts := strings.SplitN(args, " ", 2)
	if len(parts) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "read":
		return t.readFile()
	case "write":
		if len(parts) < 2 {
			return "", fmt.Errorf("write command requires content")
		}
		return t.writeFile(parts[1])
	case "append":
		if len(parts) < 2 {
			return "", fmt.Errorf("append command requires content")
		}
		return t.appendFile(parts[1])
	case "clear":
		return t.clearFile()
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// readFile reads the entire contents of the file
func (t *FileMemoryTool) readFile() (string, error) {
	// Check if file exists
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		return "File does not exist or is empty.", nil
	}

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(data) == 0 {
		return "File is empty.", nil
	}

	return string(data), nil
}

// writeFile writes content to the file (replaces existing content)
func (t *FileMemoryTool) writeFile(content string) (string, error) {
	// Add timestamp to the content
	timestampedContent := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), content)
	
	if err := os.WriteFile(t.filePath, []byte(timestampedContent), 0644); err != nil {
		return "", fmt.Errorf("error writing to file: %w", err)
	}

	return "✓ Content written to file.", nil
}

// appendFile appends content to the file
func (t *FileMemoryTool) appendFile(content string) (string, error) {
	// Add timestamp to the content
	timestampedContent := fmt.Sprintf("\n[%s] %s", time.Now().Format(time.RFC3339), content)
	
	// Check if file exists
	fileExists := true
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		fileExists = false
	}

	// If file doesn't exist, create it without the newline prefix
	if !fileExists {
		timestampedContent = strings.TrimPrefix(timestampedContent, "\n")
	}

	// Open file in append mode
	file, err := os.OpenFile(t.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(timestampedContent); err != nil {
		return "", fmt.Errorf("error appending to file: %w", err)
	}

	return "✓ Content appended to file.", nil
}

// clearFile clears the file contents
func (t *FileMemoryTool) clearFile() (string, error) {
	if err := os.WriteFile(t.filePath, []byte(""), 0644); err != nil {
		return "", fmt.Errorf("error clearing file: %w", err)
	}

	return "✓ File contents cleared.", nil
}