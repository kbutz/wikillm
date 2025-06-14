package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kbutz/wikillm/agents/models"
	"github.com/kbutz/wikillm/agents/tools"
)

// MemoryEnabledAgent is an enhanced agent that proactively uses memory
type MemoryEnabledAgent struct {
	agent      *Agent
	memoryTool *tools.EnhancedMemoryTool
}

// NewMemoryEnabledAgent creates a new memory-enabled agent
func NewMemoryEnabledAgent(model models.LLMModel, tools []models.Tool, memoryTool *tools.EnhancedMemoryTool) *MemoryEnabledAgent {
	return &MemoryEnabledAgent{
		agent:      NewAgent(model, tools),
		memoryTool: memoryTool,
	}
}

// ProcessQuery processes a query with automatic memory management
func (m *MemoryEnabledAgent) ProcessQuery(ctx context.Context, query string) (string, error) {
	// 1. Auto-store any relevant information from the query
	if m.shouldStoreMemory(query) {
		_, err := m.memoryTool.Execute(ctx, fmt.Sprintf(`{"command": "auto_store", "content": "%s"}`, 
			strings.ReplaceAll(query, `"`, `\"`)))
		if err != nil {
			// Log error but don't fail the query
			fmt.Printf("Warning: Failed to auto-store memory: %v\n", err)
		}
	}
	
	// 2. Search for relevant memories
	relevantContext := m.searchRelevantMemories(ctx, query)
	
	// 3. Enhance the query with context
	enhancedQuery := query
	if relevantContext != "" {
		enhancedQuery = fmt.Sprintf(`Based on the following context from memory:
%s

Current query: %s`, relevantContext, query)
	}
	
	// 4. Process the enhanced query
	response, err := m.agent.ProcessQuery(ctx, enhancedQuery)
	if err != nil {
		return "", err
	}
	
	// 5. Store significant exchanges
	if m.isSignificantExchange(query, response) {
		exchange := fmt.Sprintf("User: %s\nAssistant: %s", query, response)
		_, storeErr := m.memoryTool.Execute(ctx, fmt.Sprintf(`{
			"command": "store",
			"category": "conversations",
			"content": "%s",
			"tags": ["significant", "exchange"]
		}`, strings.ReplaceAll(exchange, `"`, `\"`)))
		
		if storeErr != nil {
			fmt.Printf("Warning: Failed to store conversation: %v\n", storeErr)
		}
	}
	
	return response, nil
}

// InitializeContext loads relevant memories at startup
func (m *MemoryEnabledAgent) InitializeContext(ctx context.Context) error {
	// Get context summary
	contextSummary, err := m.memoryTool.Execute(ctx, `{"command": "context"}`)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	
	if contextSummary != "" && contextSummary != "Memory Context Summary:\n\n" {
		fmt.Println("\n=== Loaded Memory Context ===")
		fmt.Println(contextSummary)
		fmt.Println("=============================\n")
	}
	
	return nil
}

// shouldStoreMemory determines if the query contains information worth storing
func (m *MemoryEnabledAgent) shouldStoreMemory(query string) bool {
	queryLower := strings.ToLower(query)
	
	// Keywords that indicate information worth storing
	storeKeywords := []string{
		// Projects
		"project", "working on", "building", "developing",
		// Preferences  
		"prefer", "i like", "always", "never", "please always",
		// Tasks
		"need to", "have to", "deadline", "due", "by tomorrow", "by next",
		// Decisions
		"decided", "going with", "chose", "will use", "using",
		// Personal info
		"my name", "i work", "my role", "my team", "i am",
		// Technical
		"stack", "using", "implemented", "architecture",
	}
	
	for _, keyword := range storeKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}
	
	return false
}

// searchRelevantMemories searches for memories relevant to the current query
func (m *MemoryEnabledAgent) searchRelevantMemories(ctx context.Context, query string) string {
	// Search for relevant memories
	searchResult, err := m.memoryTool.Execute(ctx, fmt.Sprintf(`{
		"command": "search",
		"content": "%s"
	}`, strings.ReplaceAll(query, `"`, `\"`)))
	
	if err != nil || strings.Contains(searchResult, "No memories found") {
		return ""
	}
	
	return searchResult
}

// isSignificantExchange determines if an exchange is worth storing
func (m *MemoryEnabledAgent) isSignificantExchange(query, response string) bool {
	combined := strings.ToLower(query + " " + response)
	
	significantIndicators := []string{
		"decision", "decided", "important", "remember",
		"don't forget", "key point", "conclusion", "will do",
		"noted", "understood", "confirmed", "agreed",
	}
	
	for _, indicator := range significantIndicators {
		if strings.Contains(combined, indicator) {
			return true
		}
	}
	
	return false
}
