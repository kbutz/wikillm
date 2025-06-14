package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryCategory represents different types of memories
type MemoryCategory string

const (
	UserProfile   MemoryCategory = "user_profile"
	Projects      MemoryCategory = "projects"
	Tasks         MemoryCategory = "tasks"
	Technical     MemoryCategory = "technical_details"
	Decisions     MemoryCategory = "decisions"
	Conversations MemoryCategory = "conversations"
)

// MemoryEntry represents a single memory item
type MemoryEntry struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Category     MemoryCategory         `json:"category"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	LastModified time.Time              `json:"last_modified"`
}

// MemoryIndex maintains an index of all memories for efficient retrieval
type MemoryIndex struct {
	Categories map[MemoryCategory][]string `json:"categories"`
	Tags       map[string][]string         `json:"tags"`
}

// EnhancedMemoryTool implements an advanced memory system with categorization and search
type EnhancedMemoryTool struct {
	baseDir string
	index   *MemoryIndex
}

// NewEnhancedMemoryTool creates a new enhanced memory tool
func NewEnhancedMemoryTool(baseDir string) *EnhancedMemoryTool {
	tool := &EnhancedMemoryTool{
		baseDir: baseDir,
	}

	// Ensure directory exists
	os.MkdirAll(baseDir, 0755)

	// Load or create index
	tool.loadIndex()

	return tool
}

// Name returns the name of the tool
func (t *EnhancedMemoryTool) Name() string {
	return "enhanced_memory"
}

// Description returns a description of what the tool does
func (t *EnhancedMemoryTool) Description() string {
	return `Advanced memory system with categorization, tagging, and search capabilities.
Commands:
- store <category> <content> [tags...] - Store a new memory
- retrieve <category> [limit] - Retrieve memories by category
- search <query> - Search memories by content
- update <id> <updates> - Update an existing memory
- context - Get a summary of all stored context
- auto_store <content> - Automatically detect and store memory from content

Categories: user_profile, projects, tasks, technical_details, decisions, conversations

Examples:
- store projects "Working on WikiLLM agent development" development,ai
- retrieve projects 5
- search "WikiLLM"
- context
- auto_store "I prefer using Go for backend development"`
}

// Parameters returns the parameter schema for the tool
func (t *EnhancedMemoryTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
				"enum":        []string{"store", "retrieve", "search", "update", "context", "auto_store"},
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Memory category",
				"enum":        []string{"user_profile", "projects", "tasks", "technical_details", "decisions", "conversations"},
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to store or search for",
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
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Memory ID for updates",
			},
			"updates": map[string]interface{}{
				"type":        "object",
				"description": "Updates to apply to memory",
			},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

// Execute runs the tool with the given arguments and returns the result
func (t *EnhancedMemoryTool) Execute(ctx context.Context, args string) (string, error) {
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
		return t.executeStore(params)
	case "retrieve":
		return t.executeRetrieve(params)
	case "search":
		return t.executeSearch(params)
	case "update":
		return t.executeUpdate(params)
	case "context":
		return t.executeContext()
	case "auto_store":
		return t.executeAutoStore(params)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// parseSimpleCommand parses simple command format
func (t *EnhancedMemoryTool) parseSimpleCommand(args string) map[string]interface{} {
	parts := strings.Fields(args)
	params := make(map[string]interface{})

	if len(parts) > 0 {
		params["command"] = parts[0]
	}

	if len(parts) > 1 {
		switch parts[0] {
		case "store":
			if len(parts) > 2 {
				params["category"] = parts[1]
				// Find content in quotes
				contentStart := strings.Index(args, "\"")
				contentEnd := strings.LastIndex(args, "\"")
				if contentStart != -1 && contentEnd != -1 && contentEnd > contentStart {
					params["content"] = args[contentStart+1 : contentEnd]
					// Parse tags after content
					remainder := args[contentEnd+1:]
					tags := strings.Fields(remainder)
					if len(tags) > 0 {
						params["tags"] = tags
					}
				}
			}
		case "retrieve":
			params["category"] = parts[1]
			if len(parts) > 2 {
				params["limit"] = parts[2]
			}
		case "search", "auto_store":
			params["content"] = strings.Join(parts[1:], " ")
		}
	}

	return params
}

// executeStore handles storing a new memory
func (t *EnhancedMemoryTool) executeStore(params map[string]interface{}) (string, error) {
	category, ok := params["category"].(string)
	if !ok {
		return "", fmt.Errorf("category is required for store command")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required for store command")
	}

	// Convert category string to MemoryCategory
	memCategory := MemoryCategory(category)

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

	// Extract metadata
	metadata := make(map[string]interface{})
	if metaInterface, ok := params["metadata"]; ok {
		if meta, ok := metaInterface.(map[string]interface{}); ok {
			metadata = meta
		}
	}

	// Create memory entry
	memory := &MemoryEntry{
		ID:           t.generateID(memCategory),
		Content:      content,
		Category:     memCategory,
		Tags:         tags,
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		LastModified: time.Now(),
	}

	// Save memory
	if err := t.saveMemory(memory); err != nil {
		return "", err
	}

	// Update index
	t.updateIndex(memory)

	return fmt.Sprintf("✓ Memory stored with ID: %s", memory.ID), nil
}

// executeRetrieve handles retrieving memories by category
func (t *EnhancedMemoryTool) executeRetrieve(params map[string]interface{}) (string, error) {
	category, ok := params["category"].(string)
	if !ok {
		return "", fmt.Errorf("category is required for retrieve command")
	}

	limit := 10
	if limitInterface, ok := params["limit"]; ok {
		switch v := limitInterface.(type) {
		case int:
			limit = v
		case float64:
			limit = int(v)
		case string:
			fmt.Sscanf(v, "%d", &limit)
		}
	}

	memories, err := t.retrieveByCategory(MemoryCategory(category), limit)
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return fmt.Sprintf("No memories found in category: %s", category), nil
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Memories in %s (showing %d of %d):\n", category, len(memories), len(memories)))
	for i, memory := range memories {
		output.WriteString(fmt.Sprintf("\n%d. [%s] %s\n", i+1, memory.ID, memory.Content))
		if len(memory.Tags) > 0 {
			output.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(memory.Tags, ", ")))
		}
		output.WriteString(fmt.Sprintf("   Created: %s\n", memory.CreatedAt.Format("2006-01-02 15:04")))
	}

	return output.String(), nil
}

// executeSearch handles searching memories
func (t *EnhancedMemoryTool) executeSearch(params map[string]interface{}) (string, error) {
	query, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content (search query) is required for search command")
	}

	memories := t.searchMemories(query, 5)

	if len(memories) == 0 {
		return fmt.Sprintf("No memories found matching: %s", query), nil
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for '%s':\n", query))
	for i, memory := range memories {
		output.WriteString(fmt.Sprintf("\n%d. [%s] %s\n", i+1, memory.Category, memory.Content))
		output.WriteString(fmt.Sprintf("   ID: %s\n", memory.ID))
	}

	return output.String(), nil
}

// executeUpdate handles updating an existing memory
func (t *EnhancedMemoryTool) executeUpdate(params map[string]interface{}) (string, error) {
	id, ok := params["id"].(string)
	if !ok {
		return "", fmt.Errorf("id is required for update command")
	}

	updates, ok := params["updates"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("updates are required for update command")
	}

	// Load memory
	memory, err := t.loadMemory(id)
	if err != nil {
		return "", err
	}

	// Apply updates
	if content, ok := updates["content"].(string); ok {
		memory.Content = content
	}
	if tags, ok := updates["tags"].([]string); ok {
		memory.Tags = tags
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			memory.Metadata[k] = v
		}
	}

	memory.LastModified = time.Now()

	// Save updated memory
	if err := t.saveMemory(memory); err != nil {
		return "", err
	}

	return fmt.Sprintf("✓ Memory %s updated", id), nil
}

// executeContext returns a summary of all stored context
func (t *EnhancedMemoryTool) executeContext() (string, error) {
	var output strings.Builder
	output.WriteString("Memory Context Summary:\n\n")

	// User Profile
	profiles, _ := t.retrieveByCategory(UserProfile, 10)
	if len(profiles) > 0 {
		output.WriteString("### User Profile:\n")
		for _, p := range profiles {
			output.WriteString(fmt.Sprintf("- %s\n", p.Content))
		}
		output.WriteString("\n")
	}

	// Active Projects
	projects, _ := t.retrieveByCategory(Projects, 10)
	if len(projects) > 0 {
		output.WriteString("### Active Projects:\n")
		for _, p := range projects {
			status := "active"
			if s, ok := p.Metadata["status"].(string); ok {
				status = s
			}
			output.WriteString(fmt.Sprintf("- %s [%s]\n", p.Content, status))
		}
		output.WriteString("\n")
	}

	// Recent Tasks
	tasks, _ := t.retrieveByCategory(Tasks, 5)
	if len(tasks) > 0 {
		output.WriteString("### Recent Tasks:\n")
		for _, t := range tasks {
			output.WriteString(fmt.Sprintf("- %s\n", t.Content))
		}
		output.WriteString("\n")
	}

	// Recent Decisions
	decisions, _ := t.retrieveByCategory(Decisions, 3)
	if len(decisions) > 0 {
		output.WriteString("### Recent Decisions:\n")
		for _, d := range decisions {
			output.WriteString(fmt.Sprintf("- %s\n", d.Content))
		}
	}

	return output.String(), nil
}

// executeAutoStore automatically detects and stores memories from content
func (t *EnhancedMemoryTool) executeAutoStore(params map[string]interface{}) (string, error) {
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required for auto_store command")
	}

	// Detect memory type and store
	stored := []string{}

	// Check for project mentions
	if strings.Contains(strings.ToLower(content), "project") ||
		strings.Contains(strings.ToLower(content), "working on") {
		memory := &MemoryEntry{
			ID:           t.generateID(Projects),
			Content:      content,
			Category:     Projects,
			Tags:         []string{"auto_detected"},
			Metadata:     map[string]interface{}{},
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			LastModified: time.Now(),
		}
		t.saveMemory(memory)
		t.updateIndex(memory)
		stored = append(stored, "project")
	}

	// Check for preferences
	if strings.Contains(strings.ToLower(content), "prefer") ||
		strings.Contains(strings.ToLower(content), "i like") ||
		strings.Contains(strings.ToLower(content), "always") {
		memory := &MemoryEntry{
			ID:           t.generateID(UserProfile),
			Content:      content,
			Category:     UserProfile,
			Tags:         []string{"preference", "auto_detected"},
			Metadata:     map[string]interface{}{},
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			LastModified: time.Now(),
		}
		t.saveMemory(memory)
		t.updateIndex(memory)
		stored = append(stored, "preference")
	}

	// Check for tasks
	if strings.Contains(strings.ToLower(content), "need to") ||
		strings.Contains(strings.ToLower(content), "have to") ||
		strings.Contains(strings.ToLower(content), "deadline") {
		memory := &MemoryEntry{
			ID:           t.generateID(Tasks),
			Content:      content,
			Category:     Tasks,
			Tags:         []string{"auto_detected"},
			Metadata:     map[string]interface{}{},
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			LastModified: time.Now(),
		}
		t.saveMemory(memory)
		t.updateIndex(memory)
		stored = append(stored, "task")
	}

	// Check for decisions
	if strings.Contains(strings.ToLower(content), "decided") ||
		strings.Contains(strings.ToLower(content), "going with") ||
		strings.Contains(strings.ToLower(content), "chose") {
		memory := &MemoryEntry{
			ID:           t.generateID(Decisions),
			Content:      content,
			Category:     Decisions,
			Tags:         []string{"auto_detected"},
			Metadata:     map[string]interface{}{},
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			LastModified: time.Now(),
		}
		t.saveMemory(memory)
		t.updateIndex(memory)
		stored = append(stored, "decision")
	}

	if len(stored) == 0 {
		return "No memories detected in content", nil
	}

	return fmt.Sprintf("✓ Auto-stored memories: %s", strings.Join(stored, ", ")), nil
}

// Helper methods

func (t *EnhancedMemoryTool) generateID(category MemoryCategory) string {
	return fmt.Sprintf("%s_%s", category, time.Now().Format("20060102_150405"))
}

func (t *EnhancedMemoryTool) loadIndex() {
	indexPath := filepath.Join(t.baseDir, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		// Create new index
		t.index = &MemoryIndex{
			Categories: make(map[MemoryCategory][]string),
			Tags:       make(map[string][]string),
		}
		return
	}

	var index MemoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.index = &MemoryIndex{
			Categories: make(map[MemoryCategory][]string),
			Tags:       make(map[string][]string),
		}
		return
	}

	t.index = &index
}

func (t *EnhancedMemoryTool) saveIndex() error {
	indexPath := filepath.Join(t.baseDir, "index.json")

	data, err := json.MarshalIndent(t.index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

func (t *EnhancedMemoryTool) updateIndex(memory *MemoryEntry) {
	// Update category index
	if t.index.Categories[memory.Category] == nil {
		t.index.Categories[memory.Category] = []string{}
	}

	// Check if ID already exists
	exists := false
	for _, id := range t.index.Categories[memory.Category] {
		if id == memory.ID {
			exists = true
			break
		}
	}

	if !exists {
		t.index.Categories[memory.Category] = append(t.index.Categories[memory.Category], memory.ID)
	}

	// Update tag index
	for _, tag := range memory.Tags {
		if t.index.Tags[tag] == nil {
			t.index.Tags[tag] = []string{}
		}

		tagExists := false
		for _, id := range t.index.Tags[tag] {
			if id == memory.ID {
				tagExists = true
				break
			}
		}

		if !tagExists {
			t.index.Tags[tag] = append(t.index.Tags[tag], memory.ID)
		}
	}

	t.saveIndex()
}

func (t *EnhancedMemoryTool) saveMemory(memory *MemoryEntry) error {
	memoryPath := filepath.Join(t.baseDir, memory.ID+".json")

	data, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(memoryPath, data, 0644)
}

func (t *EnhancedMemoryTool) loadMemory(id string) (*MemoryEntry, error) {
	memoryPath := filepath.Join(t.baseDir, id+".json")

	data, err := os.ReadFile(memoryPath)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	var memory MemoryEntry
	if err := json.Unmarshal(data, &memory); err != nil {
		return nil, err
	}

	// Update last accessed time
	memory.LastAccessed = time.Now()
	t.saveMemory(&memory)

	return &memory, nil
}

func (t *EnhancedMemoryTool) retrieveByCategory(category MemoryCategory, limit int) ([]*MemoryEntry, error) {
	ids, ok := t.index.Categories[category]
	if !ok || len(ids) == 0 {
		return []*MemoryEntry{}, nil
	}

	memories := []*MemoryEntry{}

	// Get most recent first
	start := len(ids) - 1
	end := start - limit + 1
	if end < 0 {
		end = 0
	}

	for i := start; i >= end && i >= 0; i-- {
		memory, err := t.loadMemory(ids[i])
		if err == nil {
			memories = append(memories, memory)
		}
	}

	return memories, nil
}

func (t *EnhancedMemoryTool) searchMemories(query string, limit int) []*MemoryEntry {
	queryLower := strings.ToLower(query)
	results := []*MemoryEntry{}
	
	// Search through all categories
	for _, ids := range t.index.Categories {
		for _, id := range ids {
			memory, err := t.loadMemory(id)
			if err != nil {
				continue
			}

			// Simple text search
			if strings.Contains(strings.ToLower(memory.Content), queryLower) {
				results = append(results, memory)
				if len(results) >= limit {
					return results
				}
			}
		}
	}

	return results
}
