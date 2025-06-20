package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// ConversationAgent specializes in natural language interactions with users
type ConversationAgent struct {
	*BaseAgent
	conversations map[string]*multiagent.ConversationContext
}

// NewConversationAgent creates a new conversation agent
func NewConversationAgent(config BaseAgentConfig) *ConversationAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeConversation

	// Add conversation-specific capabilities
	config.Capabilities = append(config.Capabilities,
		"natural_language_understanding",
		"conversation_management",
		"user_interaction",
		"context_tracking",
	)

	return &ConversationAgent{
		BaseAgent:     NewBaseAgent(config),
		conversations: make(map[string]*multiagent.ConversationContext),
	}
}

// HandleMessage processes an incoming message
func (a *ConversationAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Processing conversation"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("conversation:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	// Process based on message type
	switch msg.Type {
	case multiagent.MessageTypeRequest:
		return a.handleConversation(ctx, msg)
	case multiagent.MessageTypeQuery:
		return a.handleQuery(ctx, msg)
	default:
		// For other message types, use the base implementation
		return a.BaseAgent.HandleMessage(ctx, msg)
	}
}

// handleConversation processes a conversation request
func (a *ConversationAgent) handleConversation(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Get or create conversation context
	conversationID := a.getConversationID(msg)
	conversation := a.getOrCreateConversation(ctx, conversationID, msg)

	// Add user message to conversation
	conversation.Messages = append(conversation.Messages, multiagent.ConversationMessage{
		Role:      "user",
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	})
	conversation.LastActivity = time.Now()

	// Update conversation in memory
	a.updateConversation(ctx, conversation)

	// Check if we need to delegate to other agents
	if a.shouldDelegate(msg.Content) {
		log.Printf("ConversationAgent: Delegating message to specialists: %s", msg.Content[:min(50, len(msg.Content))])
		return a.delegateToSpecialists(ctx, msg, conversation)
	}

	log.Printf("ConversationAgent: Handling message directly with LLM: %s", msg.Content[:min(50, len(msg.Content))])

	// Build context for LLM
	contextPrompt := a.buildConversationPrompt(conversation)

	// Query LLM
	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Add assistant response to conversation
	conversation.Messages = append(conversation.Messages, multiagent.ConversationMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
		AgentID:   a.id,
	})
	conversation.LastActivity = time.Now()

	// Update conversation in memory
	a.updateConversation(ctx, conversation)

	// Create response message
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   response,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"conversation_id": conversationID,
		},
	}, nil
}

// handleQuery processes a query about conversation history
func (a *ConversationAgent) handleQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	query := strings.ToLower(msg.Content)

	if strings.Contains(query, "conversation") || strings.Contains(query, "history") {
		// Get conversation ID from context or use default
		conversationID := "default"
		if ctxID, ok := msg.Context["conversation_id"].(string); ok {
			conversationID = ctxID
		}

		// Get conversation
		conversation, exists := a.conversations[conversationID]
		if !exists {
			// Try to load from memory
			convInterface, err := a.memoryStore.Get(ctx, fmt.Sprintf("conversation:%s", conversationID))
			if err != nil {
				return nil, fmt.Errorf("conversation not found: %s", conversationID)
			}

			// Convert to ConversationContext
			var conv multiagent.ConversationContext
			convData, err := json.Marshal(convInterface)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal conversation data: %w", err)
			}

			if err := json.Unmarshal(convData, &conv); err != nil {
				return nil, fmt.Errorf("failed to unmarshal conversation data: %w", err)
			}
			conversation = &conv
		}

		// Format conversation history
		var history strings.Builder
		history.WriteString(fmt.Sprintf("Conversation History (ID: %s):\n\n", conversationID))

		for i, msg := range conversation.Messages {
			history.WriteString(fmt.Sprintf("[%d] %s: %s\n", i+1, msg.Role, msg.Content))
		}

		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   history.String(),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// For other queries, use the base implementation
	return a.BaseAgent.HandleMessage(ctx, msg)
}

// getConversationID extracts or generates a conversation ID
func (a *ConversationAgent) getConversationID(msg *multiagent.Message) string {
	// First, check if conversation ID is explicitly provided in the message context
	if ctxID, ok := msg.Context["conversation_id"].(string); ok {
		log.Printf("ConversationAgent: Using provided conversation ID: %s", ctxID)
		return ctxID
	}

	// Next, try to get the actual user ID from context and create consistent conversation ID
	if userID, ok := msg.Context["user_id"].(string); ok {
		// Use a consistent conversation ID based on the actual user ID
		conversationID := fmt.Sprintf("conv_%s", userID)
		log.Printf("ConversationAgent: Using user-based conversation ID: %s", conversationID)
		return conversationID
	}

	// Check if this is a reply to an existing conversation
	if msg.ReplyTo != "" {
		// Try to find the conversation ID from the original message
		for id, conv := range a.conversations {
			for _, m := range conv.Messages {
				if strings.Contains(m.Content, msg.ReplyTo) {
					log.Printf("ConversationAgent: Found conversation ID from reply: %s", id)
					return id
				}
			}
		}
	}

	// Fallback: Generate a new conversation ID based on the sender
	// But clean up the sender ID to remove response key prefixes
	senderID := string(msg.From)
	if strings.HasPrefix(senderID, "user_response_") {
		// Extract the actual user ID from the response key
		parts := strings.Split(senderID, "_")
		if len(parts) >= 3 {
			// user_response_user_123_456 -> user_123
			senderID = strings.Join(parts[2:len(parts)-1], "_")
		}
	}
	conversationID := fmt.Sprintf("conv_%s", senderID)
	log.Printf("ConversationAgent: Generated new conversation ID: %s", conversationID)
	return conversationID
}

// getOrCreateConversation retrieves an existing conversation or creates a new one
func (a *ConversationAgent) getOrCreateConversation(ctx context.Context, conversationID string, msg *multiagent.Message) *multiagent.ConversationContext {
	// Check if conversation exists in memory
	if conv, exists := a.conversations[conversationID]; exists {
		log.Printf("ConversationAgent: Found conversation in memory: %s (has %d messages)", conversationID, len(conv.Messages))
		return conv
	}

	// Try to load from persistent storage
	if a.memoryStore != nil {
		convKey := fmt.Sprintf("conversation:%s", conversationID)
		log.Printf("ConversationAgent: Attempting to load conversation from storage: %s", convKey)
		convInterface, err := a.memoryStore.Get(ctx, convKey)
		if err == nil {
			log.Printf("ConversationAgent: Successfully loaded conversation from storage: %s", convKey)
			// Convert to ConversationContext
			var conv multiagent.ConversationContext
			convData, err := json.Marshal(convInterface)
			if err == nil {
				if err := json.Unmarshal(convData, &conv); err == nil {
					log.Printf("ConversationAgent: Restored conversation with %d messages", len(conv.Messages))
					a.conversations[conversationID] = &conv
					return &conv
				} else {
					log.Printf("ConversationAgent: Failed to unmarshal conversation: %v", err)
				}
			} else {
				log.Printf("ConversationAgent: Failed to marshal conversation: %v", err)
			}
		} else {
			log.Printf("ConversationAgent: Could not load conversation from storage: %v", err)
		}
	}

	// Create new conversation
	log.Printf("ConversationAgent: Creating new conversation: %s", conversationID)
	conv := &multiagent.ConversationContext{
		ID:           conversationID,
		UserID:       string(msg.From),
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Messages:     []multiagent.ConversationMessage{},
		Context:      make(map[string]interface{}),
		ActiveAgents: []multiagent.AgentID{a.id},
	}

	// Add system message
	conv.Messages = append(conv.Messages, multiagent.ConversationMessage{
		Role:      "system",
		Content:   fmt.Sprintf("Conversation started with %s. I'm here to help you with any questions or tasks.", a.name),
		Timestamp: time.Now(),
		AgentID:   a.id,
	})

	a.conversations[conversationID] = conv
	return conv
}

// updateConversation persists the conversation to memory
func (a *ConversationAgent) updateConversation(ctx context.Context, conversation *multiagent.ConversationContext) {
	if a.memoryStore != nil {
		convKey := fmt.Sprintf("conversation:%s", conversation.ID)
		log.Printf("ConversationAgent: Saving conversation to storage: %s (has %d messages)", convKey, len(conversation.Messages))
		if err := a.memoryStore.Store(ctx, convKey, conversation); err != nil {
			log.Printf("ConversationAgent: Failed to save conversation: %v", err)
		} else {
			log.Printf("ConversationAgent: Successfully saved conversation: %s", convKey)
		}
	}
}

// shouldDelegate determines if the request should be delegated to specialist agents
func (a *ConversationAgent) shouldDelegate(content string) bool {
	// First check if we have an orchestrator and if specialist agents exist
	if a.orchestrator == nil {
		return false
	}

	// Get all available agents
	allAgents := a.orchestrator.ListAgents()
	hasSpecialists := false

	// Check if any specialist agents are available (other than conversation and coordinator)
	for _, agent := range allAgents {
		agentType := agent.Type()
		if agentType != multiagent.AgentTypeConversation && agentType != multiagent.AgentTypeCoordinator {
			hasSpecialists = true
			break
		}
	}

	// If no specialists are available, don't delegate
	if !hasSpecialists {
		return false
	}

	contentLower := strings.ToLower(content)

	// Check for specialized topics that specifically require delegation
	specialistKeywords := map[string][]string{
		"research":     {"research", "find information", "look up", "search for", "information about", "investigate", "analyze data", "fact check", "verify"},
		"task":        {"create task", "add task", "task", "todo", "to-do", "to do", "remind me", "reminder", "productivity"},
		"project":     {"create project", "new project", "project", "plan", "planning", "milestone", "timeline", "manage", "track progress"},
		"schedule":    {"schedule", "calendar", "appointment", "meeting", "book", "available", "free time", "time slot"},
		"communication": {"email", "message", "contact", "send", "compose", "draft", "write email", "communication", "follow up"},
		"coder":       {"write code", "programming", "function", "algorithm", "write a program", "debug", "script", "software"},
		"analyst":     {"analyze", "data analysis", "statistics", "trends", "patterns", "insights", "metrics", "performance"},
		"writer":      {"write article", "draft", "compose", "blog post", "document", "report", "essay", "outline"},
	}

	// Only delegate if there are strong indicators for specialist work
	for _, keywords := range specialistKeywords {
		for _, keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				return true
			}
		}
	}

	// Don't delegate based on length alone - most conversations should be handled directly
	return false
}

// delegateToSpecialists routes the request to appropriate specialist agents
func (a *ConversationAgent) delegateToSpecialists(ctx context.Context, msg *multiagent.Message, conversation *multiagent.ConversationContext) (*multiagent.Message, error) {
	contentLower := strings.ToLower(msg.Content)

	// Determine which specialists to involve
	specialists := []multiagent.AgentType{}
	log.Printf("ConversationAgent: Analyzing content for delegation: %s", contentLower)

	if containsAny(contentLower, []string{"research", "find information", "look up", "search for", "investigate", "fact check", "verify"}) {
		specialists = append(specialists, multiagent.AgentTypeResearch)
		log.Printf("ConversationAgent: Added research specialist")
	}

	if containsAny(contentLower, []string{"create task", "add task", "task", "todo", "to-do", "to do", "remind me", "reminder", "productivity"}) {
		specialists = append(specialists, multiagent.AgentTypeTask)
	}

	if containsAny(contentLower, []string{"create project", "new project", "project", "plan", "planning", "milestone", "timeline", "manage", "track progress"}) {
		specialists = append(specialists, multiagent.AgentTypeProjectManager)
	}

	if containsAny(contentLower, []string{"schedule", "calendar", "appointment", "meeting", "book", "available", "free time", "time slot"}) {
		specialists = append(specialists, multiagent.AgentTypeScheduler)
	}

	if containsAny(contentLower, []string{"email", "message", "contact", "send", "compose", "draft", "write email", "communication", "follow up"}) {
		specialists = append(specialists, multiagent.AgentTypeCommunicationManager)
		log.Printf("ConversationAgent: Added communication manager specialist")
	}

	if containsAny(contentLower, []string{"write code", "programming", "function", "algorithm", "debug", "script", "software"}) {
		specialists = append(specialists, multiagent.AgentTypeCoder)
	}

	if containsAny(contentLower, []string{"analyze", "data analysis", "statistics", "trends", "patterns", "insights", "metrics"}) {
		specialists = append(specialists, multiagent.AgentTypeAnalyst)
	}

	if containsAny(contentLower, []string{"write article", "draft", "compose", "blog post", "document", "report", "essay", "outline"}) {
		specialists = append(specialists, multiagent.AgentTypeWriter)
	}

	log.Printf("ConversationAgent: Selected specialists: %v", specialists)

	// If no specialists matched, use the coordinator
	if len(specialists) == 0 {
		specialists = append(specialists, multiagent.AgentTypeCoordinator)
		log.Printf("ConversationAgent: No specific specialists matched, using coordinator")
	}

	// Create a task for the coordinator to handle
	if a.orchestrator != nil {
	// Extract the response key from the original message sender
	responseKey := string(msg.From)
	log.Printf("ConversationAgent: Extracted response key: %s", responseKey)
	
	task := multiagent.Task{
	ID:          fmt.Sprintf("task_%s_%d", a.id, time.Now().UnixNano()),
	Type:        "user_request",
	Description: fmt.Sprintf("Handle user request: %s", msg.Content),
	Priority:    msg.Priority,
	Requester:   a.id,
	Assignee:    multiagent.AgentID("coordinator_agent"), // Explicitly assign to coordinator_agent
	Status:      multiagent.TaskStatusPending,
	CreatedAt:   time.Now(),
	Input: map[string]interface{}{
	"user_message":    msg.Content,
	"conversation_id": conversation.ID,
	"specialists":     specialists,
	"response_key":    responseKey, // Add the response key for final response routing
	},
	Output: make(map[string]interface{}), // Ensure Output is properly initialized
	}

		// Assign task to coordinator
		_, err := a.orchestrator.AssignTask(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("failed to assign task to coordinator: %w", err)
		}

		// Return immediate acknowledgment
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "I'm working on your request and consulting with specialists. I'll get back to you shortly.",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"conversation_id": conversation.ID,
				"task_id":         task.ID,
			},
		}, nil
	}

	// If no orchestrator, handle directly
	return a.handleConversation(ctx, msg)
}

// buildConversationPrompt creates a prompt with conversation history
func (a *ConversationAgent) buildConversationPrompt(conversation *multiagent.ConversationContext) string {
	var prompt strings.Builder

	// Add agent identity
	prompt.WriteString(fmt.Sprintf("You are %s, a conversation agent designed to help users.\n\n", a.name))

	// Add conversation history
	prompt.WriteString("Conversation history:\n")

	// Get the last 10 messages or all if fewer
	startIdx := 0
	if len(conversation.Messages) > 10 {
		startIdx = len(conversation.Messages) - 10
	}

	for _, msg := range conversation.Messages[startIdx:] {
		prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	// Add context information if available
	if len(conversation.Context) > 0 {
		prompt.WriteString("\nAdditional context:\n")
		for k, v := range conversation.Context {
			prompt.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}

	// Add instruction
	prompt.WriteString("\nPlease provide a helpful, accurate, and concise response to the user's latest message.")

	return prompt.String()
}

// Helper function to check if a string contains any of the keywords
func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
