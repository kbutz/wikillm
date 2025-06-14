package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// CoordinatorAgent orchestrates the work of specialist agents
type CoordinatorAgent struct {
	*BaseAgent
	activeCoordinations map[string]*coordination
	mu                  sync.RWMutex
}

// coordination tracks the state of a multi-agent coordination
type coordination struct {
	ID              string
	TaskID          string
	UserMessage     string
	ConversationID  string
	Specialists     []multiagent.AgentType
	SpecialistIDs   []multiagent.AgentID
	Responses       map[multiagent.AgentID]string
	Status          string
	StartTime       time.Time
	CompletionTime  *time.Time
	RequesterID     multiagent.AgentID
	FinalResponse   string
}

// NewCoordinatorAgent creates a new coordinator agent
func NewCoordinatorAgent(config BaseAgentConfig) *CoordinatorAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeCoordinator
	
	// Add coordinator-specific capabilities
	config.Capabilities = append(config.Capabilities,
		"task_delegation",
		"response_synthesis",
		"agent_coordination",
		"workflow_management",
	)
	
	return &CoordinatorAgent{
		BaseAgent:           NewBaseAgent(config),
		activeCoordinations: make(map[string]*coordination),
	}
}

// HandleMessage processes an incoming message
func (a *CoordinatorAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Coordinating agents"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("coordinator:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	// Process based on message type
	switch msg.Type {
	case multiagent.MessageTypeRequest:
		return a.handleRequest(ctx, msg)
	case multiagent.MessageTypeReport:
		return a.handleReport(ctx, msg)
	default:
		// For other message types, use the base implementation
		return a.BaseAgent.HandleMessage(ctx, msg)
	}
}

// handleRequest processes a request to coordinate specialist agents
func (a *CoordinatorAgent) handleRequest(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Check if this is a task assignment
	taskID, isTask := msg.Context["task_id"].(string)
	if !isTask {
		// Not a task assignment, use default handling
		return a.BaseAgent.HandleMessage(ctx, msg)
	}

	// Get task details
	taskInterface, err := a.memoryStore.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Convert to Task
	var task multiagent.Task
	taskData, err := json.Marshal(taskInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}
	
	if err := json.Unmarshal(taskData, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Extract coordination details from task input
	userMessage, _ := task.Input["user_message"].(string)
	conversationID, _ := task.Input["conversation_id"].(string)
	
	// Extract specialists
	var specialists []multiagent.AgentType
	if specialistsInterface, ok := task.Input["specialists"]; ok {
		if specialistsSlice, ok := specialistsInterface.([]multiagent.AgentType); ok {
			specialists = specialistsSlice
		} else if specialistsData, err := json.Marshal(specialistsInterface); err == nil {
			if err := json.Unmarshal(specialistsData, &specialists); err != nil {
				// If unmarshaling fails, use default specialists
				specialists = []multiagent.AgentType{
					multiagent.AgentTypeResearch,
					multiagent.AgentTypeTask,
				}
			}
		}
	}

	// Create a new coordination
	coordID := fmt.Sprintf("coord_%s", taskID)
	coord := &coordination{
		ID:             coordID,
		TaskID:         taskID,
		UserMessage:    userMessage,
		ConversationID: conversationID,
		Specialists:    specialists,
		SpecialistIDs:  []multiagent.AgentID{},
		Responses:      make(map[multiagent.AgentID]string),
		Status:         "in_progress",
		StartTime:      time.Now(),
		RequesterID:    task.Requester,
	}

	// Store coordination
	a.mu.Lock()
	a.activeCoordinations[coordID] = coord
	a.mu.Unlock()

	// Delegate to specialists
	if err := a.delegateToSpecialists(ctx, coord); err != nil {
		return nil, fmt.Errorf("failed to delegate to specialists: %w", err)
	}

	// Return acknowledgment
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("Coordination %s started with %d specialists", coordID, len(specialists)),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"coordination_id": coordID,
			"task_id":         taskID,
		},
	}, nil
}

// handleReport processes a report from a specialist agent
func (a *CoordinatorAgent) handleReport(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Check if this is a response to a coordination
	coordID, isCoord := msg.Context["coordination_id"].(string)
	if !isCoord {
		// Not a coordination response, use default handling
		return a.BaseAgent.HandleMessage(ctx, msg)
	}

	// Get coordination
	a.mu.Lock()
	coord, exists := a.activeCoordinations[coordID]
	if !exists {
		a.mu.Unlock()
		return nil, fmt.Errorf("coordination not found: %s", coordID)
	}

	// Store specialist response
	coord.Responses[msg.From] = msg.Content
	
	// Check if all specialists have responded
	allResponded := true
	for _, specialistID := range coord.SpecialistIDs {
		if _, responded := coord.Responses[specialistID]; !responded {
			allResponded = false
			break
		}
	}
	
	a.mu.Unlock()

	// If all specialists have responded, synthesize final response
	if allResponded {
		if err := a.finalizeCoordination(ctx, coord); err != nil {
			return nil, fmt.Errorf("failed to finalize coordination: %w", err)
		}
	}

	// Return acknowledgment
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "Response received and processed",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// delegateToSpecialists sends requests to specialist agents
func (a *CoordinatorAgent) delegateToSpecialists(ctx context.Context, coord *coordination) error {
	if a.orchestrator == nil {
		return fmt.Errorf("no orchestrator configured")
	}

	// Get available agents for each specialist type
	for _, specialistType := range coord.Specialists {
		agents := a.getAgentsByType(ctx, specialistType)
		if len(agents) == 0 {
			continue
		}

		// Use the first available agent of each type
		specialistID := agents[0]
		coord.SpecialistIDs = append(coord.SpecialistIDs, specialistID)

		// Create message for specialist
		message := &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{specialistID},
			Type:      multiagent.MessageTypeRequest,
			Content:   coord.UserMessage,
			Priority:  multiagent.PriorityHigh,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"coordination_id": coord.ID,
				"conversation_id": coord.ConversationID,
				"role":            string(specialistType),
			},
		}

		// Send message
		if err := a.orchestrator.RouteMessage(ctx, message); err != nil {
			return fmt.Errorf("failed to send message to specialist %s: %w", specialistID, err)
		}
	}

	return nil
}

// finalizeCoordination synthesizes specialist responses and sends final response
func (a *CoordinatorAgent) finalizeCoordination(ctx context.Context, coord *coordination) error {
	// Mark coordination as completed
	a.mu.Lock()
	coord.Status = "completed"
	now := time.Now()
	coord.CompletionTime = &now
	a.mu.Unlock()

	// Build context for LLM
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("You are %s, a coordinator agent. You need to synthesize responses from specialist agents into a coherent, helpful response for the user.\n\n", a.name))
	promptBuilder.WriteString(fmt.Sprintf("User message: %s\n\n", coord.UserMessage))
	promptBuilder.WriteString("Specialist responses:\n")
	
	for specialistID, response := range coord.Responses {
		promptBuilder.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", specialistID, response))
	}
	
	promptBuilder.WriteString("Please synthesize these responses into a single, coherent response that addresses the user's request comprehensively. Be concise but thorough, and ensure all relevant information is included.")

	// Query LLM for synthesized response
	synthesizedResponse, err := a.llmProvider.Query(ctx, promptBuilder.String())
	if err != nil {
		return fmt.Errorf("failed to synthesize response: %w", err)
	}

	// Store final response
	coord.FinalResponse = synthesizedResponse

	// Update task with final response
	if err := a.updateTask(ctx, coord); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Send final response to requester
	finalMessage := &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{coord.RequesterID},
		Type:      multiagent.MessageTypeResponse,
		Content:   synthesizedResponse,
		Priority:  multiagent.PriorityHigh,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"coordination_id": coord.ID,
			"conversation_id": coord.ConversationID,
			"task_id":         coord.TaskID,
		},
	}

	if err := a.orchestrator.RouteMessage(ctx, finalMessage); err != nil {
		return fmt.Errorf("failed to send final response: %w", err)
	}

	return nil
}

// updateTask updates the task with the final response
func (a *CoordinatorAgent) updateTask(ctx context.Context, coord *coordination) error {
	// Get task
	taskInterface, err := a.memoryStore.Get(ctx, coord.TaskID)
	if err != nil {
		return fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Convert to Task
	var task multiagent.Task
	taskData, err := json.Marshal(taskInterface)
	if err != nil {
		return fmt.Errorf("failed to marshal task data: %w", err)
	}
	
	if err := json.Unmarshal(taskData, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Update task
	task.Status = multiagent.TaskStatusCompleted
	now := time.Now()
	task.CompletedAt = &now
	task.Output["final_response"] = coord.FinalResponse
	task.Output["specialist_responses"] = coord.Responses

	// Store updated task
	if err := a.memoryStore.Store(ctx, coord.TaskID, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// getAgentsByType returns available agents of a specific type
func (a *CoordinatorAgent) getAgentsByType(ctx context.Context, agentType multiagent.AgentType) []multiagent.AgentID {
	if a.orchestrator == nil {
		return nil
	}

	// Get all agents
	allAgents := a.orchestrator.ListAgents()
	
	// Filter by type
	var matchingAgents []multiagent.AgentID
	for _, agent := range allAgents {
		if agent.Type() == agentType {
			matchingAgents = append(matchingAgents, agent.ID())
		}
	}

	return matchingAgents
}