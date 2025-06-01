package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ToolCall represents a request to use a tool
type ToolCall struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
}

// ExtractToolCall attempts to extract a tool call from the LLM response
func ExtractToolCall(response string) (ToolCall, bool) {
	// Look for JSON blocks in the response
	// Try to find JSON within code blocks first
	var jsonStr string

	// Check for ```json blocks
	jsonBlockRegex := regexp.MustCompile("```json\\s*\\n([\\s\\S]*?)\\n```")
	if matches := jsonBlockRegex.FindStringSubmatch(response); len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		// Fall back to finding raw JSON
		jsonStart := strings.Index(response, "{")
		jsonEnd := strings.LastIndex(response, "}")

		if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
			return ToolCall{}, false
		}

		jsonStr = response[jsonStart : jsonEnd+1]
	}

	// First try to unmarshal into a map to handle different args formats
	var rawMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &rawMap)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\nJSON string: %s\n", err, jsonStr)
		return ToolCall{}, false
	}

	// Extract the tool name
	toolName, ok := rawMap["tool"].(string)
	if !ok || toolName == "" {
		return ToolCall{}, false
	}

	// Handle different formats of args
	var argsStr string
	rawArgs, hasArgs := rawMap["args"]

	if !hasArgs {
		// Some LLMs might use "arguments" instead of "args"
		rawArgs, hasArgs = rawMap["arguments"]
	}

	if !hasArgs {
		argsStr = "" // No args provided
	} else {
		switch args := rawArgs.(type) {
		case string:
			// If args is already a string, use it directly
			argsStr = strings.TrimSpace(args)
		case []interface{}:
			// If args is an array, join the elements
			var parts []string
			for _, arg := range args {
				if str, ok := arg.(string); ok {
					parts = append(parts, str)
				} else {
					parts = append(parts, fmt.Sprintf("%v", arg))
				}
			}
			argsStr = strings.Join(parts, " ")
		case map[string]interface{}:
			// If args is an object, try to extract meaningful content

			// First check for specific fields that indicate the command
			if command, ok := args["command"].(string); ok {
				argsStr = command

				// Add task/content if present
				if task, ok := args["task"].(string); ok && task != "" {
					argsStr += " " + task
				} else if content, ok := args["content"].(string); ok && content != "" {
					argsStr += " " + content
				} else if desc, ok := args["description"].(string); ok && desc != "" {
					argsStr += " " + desc
				}

				// Add priority if present
				if priority, ok := args["priority"].(string); ok && priority != "" {
					argsStr += " priority:" + priority
				}

				// Add time if present
				if timeStr, ok := args["time"].(string); ok && timeStr != "" {
					argsStr += " time:" + timeStr
				} else if timeMin, ok := args["time_minutes"].(float64); ok {
					argsStr += fmt.Sprintf(" time:%dm", int(timeMin))
				}
			} else if task, ok := args["task"].(string); ok && task != "" {
				// Just a task field - assume it's an add command
				argsStr = "add " + task
			} else {
				// Fall back to concatenating all string values
				var parts []string

				// Try to maintain some order: command-like fields first
				commandFields := []string{"action", "command", "operation"}
				for _, field := range commandFields {
					if val, ok := args[field].(string); ok {
						parts = append(parts, val)
						delete(args, field)
					}
				}

				// Then add remaining fields
				for _, v := range args {
					if str, ok := v.(string); ok && str != "" {
						parts = append(parts, str)
					} else if num, ok := v.(float64); ok {
						parts = append(parts, fmt.Sprintf("%v", num))
					}
				}
				argsStr = strings.Join(parts, " ")
			}
		default:
			// For any other type, convert to string
			argsStr = fmt.Sprintf("%v", rawArgs)
		}
	}

	return ToolCall{
		Tool: toolName,
		Args: strings.TrimSpace(argsStr),
	}, true
}
