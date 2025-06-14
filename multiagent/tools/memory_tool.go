package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// MemoryTool provides agents with access to the memory store
type MemoryTool struct {
	name        string
	description string
	memoryStore multiagent.MemoryStore
}

// NewMemoryTool creates a new memory tool
func NewMemoryTool(memoryStore multiagent.MemoryStore) *MemoryTool {
	return &MemoryTool{
		name:        "memory",
		description: "Access and manage agent memory",
		memoryStore: memoryStore,
	}
}

// Name returns the name of the tool
func (t *MemoryTool) Name() string {
	return t.name
}

// Description returns a description of what the tool does
func (t *MemoryTool) Description() string {
	return `Memory tool for storing and retrieving information.
Commands:
- store <content> [tags...] - Store new information
- retrieve <key> - Retrieve specific memory by key
- search <query> [limit] - Search memories by content
- list <prefix> [limit] - List keys with a prefix
- context - Get a summary of relevant context

Examples:
- store "User prefers dark mode" preferences,ui
- retrieve conversation:20230615:1
- search "project goals" 5
- list user: 10
- context`
}

// Parameters returns the parameter schema for the tool
func (t *MemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
				"enum":        []string{"store", "retrieve", "search", "list", "context"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to store or search for",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "Memory key for retrieval",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Category for organizing memories",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Tags for categorization",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return",
			},
			"prefix": map[string]interface{}{
				"type":        "string",
				"description": "Key prefix for listing",
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the tool with the given arguments and returns the result
func (t *MemoryTool) Execute(ctx context.Context, args string) (string, error) {
	var params map[string]interface{}

	// Try to parse as JSON
	if strings.HasPrefix(strings.TrimSpace(args), "{") {
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", fmt.Errorf("failed to parse JSON arguments: %w", err)
		}
	} else {
		// Parse simple command format
		params = t.parseSimpleCommand(args)
	}

	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}

	switch command {
	case "store":
		return t.executeStore(ctx, params)
	case "retrieve":
		return t.executeRetrieve(ctx, params)
	case "search":
		return t.executeSearch(ctx, params)
	case "list":
		return t.executeList(ctx, params)
	case "context":
		return t.executeContext(ctx)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// parseSimpleCommand parses simple command format
func (t *MemoryTool) parseSimpleCommand(args string) map[string]interface{} {
	parts := strings.Fields(args)
	params := make(map[string]interface{})

	if len(parts) > 0 {
		params["command"] = parts[0]
	}

	if len(parts) > 1 {
		switch parts[0] {
		case "store":
			// Find content in quotes
			contentStart := strings.Index(args, "\"")
			contentEnd := strings.LastIndex(args, "\"")
			if contentStart != -1 && contentEnd != -1 && contentEnd > contentStart {
				params["content"] = args[contentStart+1 : contentEnd]
				// Parse tags after content
				remainder := args[contentEnd+1:]
				tagParts := strings.Fields(remainder)
				if len(tagParts) > 0 {
					params["tags"] = tagParts
				}
			} else {
				// No quotes, assume the rest is content
				params["content"] = strings.Join(parts[1:], " ")
			}
		case "retrieve":
			params["key"] = parts[1]
		case "search":
			params["content"] = strings.Join(parts[1:], " ")
			// Check if last part is a number (limit)
			if len(parts) > 2 {
				var limit int
				if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &limit); err == nil {
					params["limit"] = limit
					params["content"] = strings.Join(parts[1:len(parts)-1], " ")
				}
			}
		case "list":
			params["prefix"] = parts[1]
			if len(parts) > 2 {
				var limit int
				if _, err := fmt.Sscanf(parts[2], "%d", &limit); err == nil {
					params["limit"] = limit
				}
			}
		}
	}

	return params
}

// executeStore handles storing new information
func (t *MemoryTool) executeStore(ctx context.Context, params map[string]interface{}) (string, error) {
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required for store command")
	}

	// Extract category
	category := "general"
	if cat, ok := params["category"].(string); ok {
		category = cat
	}

	// Extract tags
	var tags []string
	if tagsInterface, ok := params["tags"]; ok {
		switch v := tagsInterface.(type) {
		case []string:
			tags = v
		case []interface{}:
			for _, tag := range v {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		}
	}

	// Create memory entry
	entry := multiagent.MemoryEntry{
		Key:       fmt.Sprintf("%s:%d", category, time.Now().UnixNano()),
		Value:     content,
		Category:  category,
		Tags:      tags,
		Metadata:  map[string]interface{}{"source": "memory_tool"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store in memory
	if err := t.memoryStore.Store(ctx, entry.Key, entry); err != nil {
		return "", fmt.Errorf("failed to store memory: %w", err)
	}

	return fmt.Sprintf("✓ Memory stored with key: %s", entry.Key), nil
}

// executeRetrieve handles retrieving specific memory
func (t *MemoryTool) executeRetrieve(ctx context.Context, params map[string]interface{}) (string, error) {
	key, ok := params["key"].(string)
	if !ok {
		return "", fmt.Errorf("key is required for retrieve command")
	}

	value, err := t.memoryStore.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve memory: %w", err)
	}

	// Format output based on type
	switch v := value.(type) {
	case multiagent.MemoryEntry:
		return fmt.Sprintf("Memory [%s]:\n%v\nTags: %s\nCreated: %s", 
			v.Key, v.Value, strings.Join(v.Tags, ", "), v.CreatedAt.Format("2006-01-02 15:04:05")), nil
	case string:
		return fmt.Sprintf("Memory [%s]:\n%s", key, v), nil
	default:
		// Convert to JSON for structured display
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("Memory [%s]:\n%v", key, value), nil
		}
		return fmt.Sprintf("Memory [%s]:\n%s", key, string(data)), nil
	}
}

// executeSearch handles searching memories
func (t *MemoryTool) executeSearch(ctx context.Context, params map[string]interface{}) (string, error) {
	query, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content (search query) is required for search command")
	}

	limit := 10
	if limitVal, ok := params["limit"]; ok {
		switch v := limitVal.(type) {
		case int:
			limit = v
		case float64:
			limit = int(v)
		}
	}

	results, err := t.memoryStore.Search(ctx, query, limit)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No memories found matching: %s", query), nil
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for '%s' (found %d):\n\n", query, len(results)))
	
	for i, entry := range results {
		output.WriteString(fmt.Sprintf("%d. [%s] ", i+1, entry.Category))
		
		switch v := entry.Value.(type) {
		case string:
			output.WriteString(v)
		default:
			// Try to convert to string
			output.WriteString(fmt.Sprintf("%v", v))
		}
		
		output.WriteString(fmt.Sprintf("\n   Key: %s\n", entry.Key))
		if len(entry.Tags) > 0 {
			output.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(entry.Tags, ", ")))
		}
	}

	return output.String(), nil
}

// executeList handles listing keys with a prefix
func (t *MemoryTool) executeList(ctx context.Context, params map[string]interface{}) (string, error) {
	prefix, ok := params["prefix"].(string)
	if !ok {
		return "", fmt.Errorf("prefix is required for list command")
	}

	limit := 20
	if limitVal, ok := params["limit"]; ok {
		switch v := limitVal.(type) {
		case int:
			limit = v
		case float64:
			limit = int(v)
		}
	}

	keys, err := t.memoryStore.List(ctx, prefix, limit)
	if err != nil {
		return "", fmt.Errorf("listing failed: %w", err)
	}

	if len(keys) == 0 {
		return fmt.Sprintf("No keys found with prefix: %s", prefix), nil
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Keys with prefix '%s' (found %d):\n\n", prefix, len(keys)))
	
	for i, key := range keys {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, key))
	}

	return output.String(), nil
}

// executeContext provides a summary of relevant context
func (t *MemoryTool) executeContext(ctx context.Context) (string, error) {
	// Get user preferences
	userPrefs, _ := t.memoryStore.Search(ctx, "preference", 5)
	
	// Get recent conversations
	convKeys, _ := t.memoryStore.List(ctx, "conversation:", 5)
	conversations, _ := t.memoryStore.GetMultiple(ctx, convKeys)
	
	// Get active tasks
	tasks, _ := t.memoryStore.SearchByTags(ctx, []string{"task", "active"}, 5)
	
	// Format output
	var output strings.Builder
	output.WriteString("Context Summary:\n\n")
	
	if len(userPrefs) > 0 {
		output.WriteString("### User Preferences:\n")
		for _, pref := range userPrefs {
			output.WriteString(fmt.Sprintf("- %v\n", pref.Value))
		}
		output.WriteString("\n")
	}
	
	if len(tasks) > 0 {
		output.WriteString("### Active Tasks:\n")
		for _, task := range tasks {
			output.WriteString(fmt.Sprintf("- %v\n", task.Value))
		}
		output.WriteString("\n")
	}
	
	if len(conversations) > 0 {
		output.WriteString("### Recent Conversations:\n")
		i := 0
		for _, conv := range conversations {
			output.WriteString(fmt.Sprintf("- %v\n", conv))
			i++
			if i >= 3 {
				break
			}
		}
	}
	
	return output.String(), nil
}