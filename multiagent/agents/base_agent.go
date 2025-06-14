package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	id           multiagent.AgentID
	agentType    multiagent.AgentType
	name         string
	description  string
	state        multiagent.AgentState
	capabilities []string
	messageChan  chan *multiagent.Message
	stopChan     chan struct{}
	mu           sync.RWMutex
	tools        []multiagent.Tool
	llmProvider  multiagent.LLMProvider
	memoryStore  multiagent.MemoryStore
	orchestrator multiagent.Orchestrator
}

// BaseAgentConfig holds configuration for creating a base agent
type BaseAgentConfig struct {
	ID           multiagent.AgentID
	Type         multiagent.AgentType
	Name         string
	Description  string
	Capabilities []string
	Tools        []multiagent.Tool
	LLMProvider  multiagent.LLMProvider
	MemoryStore  multiagent.MemoryStore
	Orchestrator multiagent.Orchestrator
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(config BaseAgentConfig) *BaseAgent {
	return &BaseAgent{
		id:           config.ID,
		agentType:    config.Type,
		name:         config.Name,
		description:  config.Description,
		capabilities: config.Capabilities,
		tools:        config.Tools,
		llmProvider:  config.LLMProvider,
		memoryStore:  config.MemoryStore,
		orchestrator: config.Orchestrator,
		messageChan:  make(chan *multiagent.Message, 100),
		stopChan:     make(chan struct{}),
		state: multiagent.AgentState{
			Status:       multiagent.AgentStatusOffline,
			Capabilities: config.Capabilities,
			Workload:     0,
			Metadata:     make(map[string]interface{}),
		},
	}
}

// ID returns the agent's unique identifier
func (a *BaseAgent) ID() multiagent.AgentID {
	return a.id
}

// Type returns the agent type
func (a *BaseAgent) Type() multiagent.AgentType {
	return a.agentType
}

// Name returns the agent's name
func (a *BaseAgent) Name() string {
	return a.name
}

// Description returns the agent's description
func (a *BaseAgent) Description() string {
	return a.description
}

// Initialize prepares the agent for operation
func (a *BaseAgent) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update state
	a.state.Status = multiagent.AgentStatusStarting
	a.state.LastActivity = time.Now()

	// Store agent initialization in memory
	if a.memoryStore != nil {
		initData := map[string]interface{}{
			"agent_id":     a.id,
			"agent_type":   a.agentType,
			"initialized":  time.Now(),
			"capabilities": a.capabilities,
		}
		
		key := fmt.Sprintf("agent:%s:init", a.id)
		if err := a.memoryStore.Store(ctx, key, initData); err != nil {
			return fmt.Errorf("failed to store initialization data: %w", err)
		}
	}

	return nil
}

// Start begins the agent's operation
func (a *BaseAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.state.Status != multiagent.AgentStatusStarting && 
	   a.state.Status != multiagent.AgentStatusOffline {
		a.mu.Unlock()
		return fmt.Errorf("agent %s is already running", a.id)
	}
	
	a.state.Status = multiagent.AgentStatusIdle
	a.state.LastActivity = time.Now()
	a.mu.Unlock()

	// Start message processing loop
	go a.messageLoop(ctx)

	// Announce agent availability
	announcement := &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{multiagent.AgentID("coordinator")},
		Type:      multiagent.MessageTypeNotification,
		Content:   fmt.Sprintf("Agent %s (%s) is now online", a.name, a.id),
		Priority:  multiagent.PriorityMedium,
		Timestamp: time.Now(),
	}

	if a.orchestrator != nil {
		if err := a.orchestrator.RouteMessage(ctx, announcement); err != nil {
			return fmt.Errorf("failed to announce agent availability: %w", err)
		}
	}

	return nil
}

// Stop halts the agent's operation
func (a *BaseAgent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state.Status == multiagent.AgentStatusOffline {
		return nil
	}

	// Update state
	a.state.Status = multiagent.AgentStatusOffline
	a.state.LastActivity = time.Now()

	// Close stop channel to signal message loop to exit
	close(a.stopChan)

	// Store shutdown event
	if a.memoryStore != nil {
		shutdownData := map[string]interface{}{
			"agent_id":  a.id,
			"shutdown":  time.Now(),
			"workload":  a.state.Workload,
			"last_task": a.state.CurrentTask,
		}
		
		key := fmt.Sprintf("agent:%s:shutdown:%d", a.id, time.Now().Unix())
		if err := a.memoryStore.Store(ctx, key, shutdownData); err != nil {
			// Log error but don't fail shutdown
			fmt.Printf("Failed to store shutdown data: %v\n", err)
		}
	}

	return nil
}

// GetState returns the current state of the agent
func (a *BaseAgent) GetState() multiagent.AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	// Create a copy to avoid external modifications
	stateCopy := a.state
	stateCopy.Metadata = make(map[string]interface{})
	for k, v := range a.state.Metadata {
		stateCopy.Metadata[k] = v
	}
	
	return stateCopy
}

// SendMessage sends a message through the orchestrator
func (a *BaseAgent) SendMessage(ctx context.Context, msg *multiagent.Message) error {
	if a.orchestrator == nil {
		return fmt.Errorf("no orchestrator configured")
	}
	
	// Set sender if not already set
	if msg.From == "" {
		msg.From = a.id
	}
	
	// Set timestamp if not already set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	
	// Route through orchestrator
	return a.orchestrator.RouteMessage(ctx, msg)
}

// ReceiveMessage receives a message from the agent's message channel
func (a *BaseAgent) ReceiveMessage(ctx context.Context) (*multiagent.Message, error) {
	select {
	case msg := <-a.messageChan:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleMessage processes an incoming message
func (a *BaseAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	a.mu.Lock()
	a.state.LastActivity = time.Now()
	currentWorkload := a.state.Workload
	a.state.Workload = min(currentWorkload+10, 100)
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Workload = max(a.state.Workload-10, 0)
		a.mu.Unlock()
	}()

	// Store message in memory for context
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("msg:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	// Process based on message type
	var response *multiagent.Message
	var err error

	switch msg.Type {
	case multiagent.MessageTypeRequest:
		response, err = a.handleRequest(ctx, msg)
	case multiagent.MessageTypeQuery:
		response, err = a.handleQuery(ctx, msg)
	case multiagent.MessageTypeCommand:
		response, err = a.handleCommand(ctx, msg)
	default:
		// For other message types, acknowledge receipt
		response = a.createAcknowledgment(msg)
	}

	return response, err
}

// GetCapabilities returns the agent's capabilities
func (a *BaseAgent) GetCapabilities() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	capabilities := make([]string, len(a.capabilities))
	copy(capabilities, a.capabilities)
	return capabilities
}

// CanHandle checks if the agent can handle a specific message type
func (a *BaseAgent) CanHandle(messageType multiagent.MessageType) bool {
	switch messageType {
	case multiagent.MessageTypeRequest, 
	     multiagent.MessageTypeQuery,
	     multiagent.MessageTypeCommand:
		return true
	default:
		return false
	}
}

// Internal helper methods

func (a *BaseAgent) messageLoop(ctx context.Context) {
	for {
		select {
		case msg := <-a.messageChan:
			response, err := a.HandleMessage(ctx, msg)
			if err != nil {
				// Send error response
				errorResponse := &multiagent.Message{
					ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
					From:      a.id,
					To:        []multiagent.AgentID{msg.From},
					Type:      multiagent.MessageTypeError,
					Content:   fmt.Sprintf("Error processing message: %v", err),
					ReplyTo:   msg.ID,
					Timestamp: time.Now(),
				}
				a.SendMessage(ctx, errorResponse)
			} else if response != nil && msg.RequiresACK {
				// Send response if acknowledgment was required
				a.SendMessage(ctx, response)
			}
			
		case <-a.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

func (a *BaseAgent) handleRequest(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context for LLM
	contextPrompt := a.buildContextPrompt(ctx, msg)
	
	// Query LLM with available tools
	response, err := a.llmProvider.QueryWithTools(ctx, contextPrompt, a.tools)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Create response message
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   response,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context:   msg.Context,
	}, nil
}

func (a *BaseAgent) handleQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Search memory for relevant information
	results, err := a.searchRelevantMemory(ctx, msg.Content)
	if err != nil {
		return nil, fmt.Errorf("memory search failed: %w", err)
	}

	// Build response with memory context
	contextPrompt := fmt.Sprintf("Based on the following context and query, provide a helpful response.\n\nContext:\n%s\n\nQuery: %s", results, msg.Content)
	
	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   response,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *BaseAgent) handleCommand(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = msg.Content
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Execute command
	result, err := a.executeCommand(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeReport,
		Content:   result,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *BaseAgent) createAcknowledgment(msg *multiagent.Message) *multiagent.Message {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("Message %s received and acknowledged", msg.ID),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}
}

func (a *BaseAgent) buildContextPrompt(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder
	
	contextBuilder.WriteString(fmt.Sprintf("You are %s, a %s agent.\n", a.name, a.agentType))
	contextBuilder.WriteString(fmt.Sprintf("Description: %s\n\n", a.description))
	
	// Add message context
	contextBuilder.WriteString(fmt.Sprintf("Request from %s: %s\n", msg.From, msg.Content))
	
	// Add any additional context from the message
	if len(msg.Context) > 0 {
		contextBuilder.WriteString("\nAdditional Context:\n")
		for k, v := range msg.Context {
			contextBuilder.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}
	
	// Add recent memory context
	if memories := a.getRecentMemories(ctx, 5); len(memories) > 0 {
		contextBuilder.WriteString("\nRecent Memory:\n")
		for _, mem := range memories {
			contextBuilder.WriteString(fmt.Sprintf("- %v\n", mem.Value))
		}
	}
	
	return contextBuilder.String()
}

func (a *BaseAgent) searchRelevantMemory(ctx context.Context, query string) (string, error) {
	if a.memoryStore == nil {
		return "", nil
	}

	results, err := a.memoryStore.Search(ctx, query, 10)
	if err != nil {
		return "", err
	}

	var contextBuilder strings.Builder
	for i, entry := range results {
		contextBuilder.WriteString(fmt.Sprintf("%d. %v\n", i+1, entry.Value))
	}

	return contextBuilder.String(), nil
}

func (a *BaseAgent) getRecentMemories(ctx context.Context, limit int) []multiagent.MemoryEntry {
	if a.memoryStore == nil {
		return nil
	}

	// Get recent messages for this agent
	prefix := fmt.Sprintf("msg:%s:", a.id)
	keys, err := a.memoryStore.List(ctx, prefix, limit)
	if err != nil {
		return nil
	}

	entries := make([]multiagent.MemoryEntry, 0, len(keys))
	values, err := a.memoryStore.GetMultiple(ctx, keys)
	if err != nil {
		return nil
	}

	for key, value := range values {
		entries = append(entries, multiagent.MemoryEntry{
			Key:   key,
			Value: value,
		})
	}

	return entries
}

func (a *BaseAgent) executeCommand(ctx context.Context, msg *multiagent.Message) (string, error) {
	// This is a placeholder - specific agents will override this method
	return fmt.Sprintf("Command '%s' executed successfully by %s", msg.Content, a.name), nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
