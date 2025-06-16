package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// DefaultOrchestrator implements the Orchestrator interface
type DefaultOrchestrator struct {
	agents               map[multiagent.AgentID]multiagent.Agent
	agentsByType         map[multiagent.AgentType][]multiagent.Agent
	tasks                map[string]*multiagent.Task
	messageQueue         chan *multiagent.Message
	eventQueue           chan *multiagent.Event
	memoryStore          multiagent.MemoryStore
	mu                   sync.RWMutex
	startTime            time.Time
	stopChan             chan struct{}
	wg                   sync.WaitGroup
	running              bool
	userResponseHandlers map[string]func(string) // Map of response key to handler function
	handlersMutex        sync.RWMutex
}

// OrchestratorConfig holds configuration for creating an orchestrator
type OrchestratorConfig struct {
	MemoryStore      multiagent.MemoryStore
	MessageQueueSize int
	EventQueueSize   int
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(config OrchestratorConfig) *DefaultOrchestrator {
	if config.MessageQueueSize == 0 {
		config.MessageQueueSize = 1000
	}
	if config.EventQueueSize == 0 {
		config.EventQueueSize = 500
	}

	return &DefaultOrchestrator{
		agents:               make(map[multiagent.AgentID]multiagent.Agent),
		agentsByType:         make(map[multiagent.AgentType][]multiagent.Agent),
		tasks:                make(map[string]*multiagent.Task),
		messageQueue:         make(chan *multiagent.Message, config.MessageQueueSize),
		eventQueue:           make(chan *multiagent.Event, config.EventQueueSize),
		memoryStore:          config.MemoryStore,
		stopChan:             make(chan struct{}),
		running:              false,
		userResponseHandlers: make(map[string]func(string)),
	}
}

// RegisterAgent registers a new agent with the orchestrator
func (o *DefaultOrchestrator) RegisterAgent(agent multiagent.Agent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	agentID := agent.ID()

	// Check if agent already registered
	if _, exists := o.agents[agentID]; exists {
		return fmt.Errorf("agent %s already registered", agentID)
	}

	// Add to agent maps
	o.agents[agentID] = agent

	agentType := agent.Type()
	if o.agentsByType[agentType] == nil {
		o.agentsByType[agentType] = []multiagent.Agent{}
	}
	o.agentsByType[agentType] = append(o.agentsByType[agentType], agent)

	log.Printf("Orchestrator: Registered agent %s (%s) of type %s", agent.Name(), agentID, agentType)

	// Store registration in memory
	if o.memoryStore != nil {
		regKey := fmt.Sprintf("orchestrator:agent_registered:%s", agentID)
		o.memoryStore.Store(context.Background(), regKey, map[string]interface{}{
			"agent_id":     agentID,
			"agent_type":   agentType,
			"registered":   time.Now(),
			"capabilities": agent.GetCapabilities(),
		})
	}

	return nil
}

// UnregisterAgent removes an agent from the orchestrator
func (o *DefaultOrchestrator) UnregisterAgent(agentID multiagent.AgentID) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	agent, exists := o.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Stop the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := agent.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent %s: %w", agentID, err)
	}

	// Remove from maps
	delete(o.agents, agentID)

	// Remove from type map
	agentType := agent.Type()
	if agents, exists := o.agentsByType[agentType]; exists {
		newAgents := []multiagent.Agent{}
		for _, a := range agents {
			if a.ID() != agentID {
				newAgents = append(newAgents, a)
			}
		}

		if len(newAgents) > 0 {
			o.agentsByType[agentType] = newAgents
		} else {
			delete(o.agentsByType, agentType)
		}
	}

	return nil
}

// GetAgent retrieves an agent by ID
func (o *DefaultOrchestrator) GetAgent(agentID multiagent.AgentID) (multiagent.Agent, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agent, exists := o.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// ListAgents returns all registered agents
func (o *DefaultOrchestrator) ListAgents() []multiagent.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]multiagent.Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}

	return agents
}

// RouteMessage routes a message to appropriate agents
func (o *DefaultOrchestrator) RouteMessage(ctx context.Context, msg *multiagent.Message) error {
	// Validate message
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Store message in memory
	if o.memoryStore != nil {
		msgKey := fmt.Sprintf("orchestrator:message:%s", msg.ID)
		o.memoryStore.Store(ctx, msgKey, msg)
	}

	// If orchestrator is running, add to message queue
	if o.running {
		select {
		case o.messageQueue <- msg:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("message queue full")
		}
	}

	// If not running, route directly
	return o.routeMessageToAgents(ctx, msg)
}

// BroadcastMessage sends a message to all agents
func (o *DefaultOrchestrator) BroadcastMessage(ctx context.Context, msg *multiagent.Message) error {
	o.mu.RLock()
	agentIDs := make([]multiagent.AgentID, 0, len(o.agents))
	for id := range o.agents {
		agentIDs = append(agentIDs, id)
	}
	o.mu.RUnlock()

	// Set recipients to all agents
	msg.To = agentIDs

	return o.RouteMessage(ctx, msg)
}

// AssignTask assigns a task to an appropriate agent
func (o *DefaultOrchestrator) AssignTask(ctx context.Context, task multiagent.Task) (multiagent.AgentID, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Generate task ID if not set
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	log.Printf("Orchestrator: AssignTask called for task %s, assignee: %s", task.ID, task.Assignee)

	// Set initial status
	task.Status = multiagent.TaskStatusPending
	task.CreatedAt = time.Now()

	// Ensure Output map is initialized if nil
	if task.Output == nil {
		task.Output = make(map[string]interface{})
		log.Printf("Orchestrator: Initialized nil Output map for task %s", task.ID)
	}

	var agent multiagent.Agent
	var err error

	// Check if task already has an assignee
	if task.Assignee != "" {
		// Use the specified assignee
		var exists bool
		agent, exists = o.agents[task.Assignee]
		if !exists {
			return "", fmt.Errorf("specified assignee %s not found", task.Assignee)
		}
	} else {
		// Find best agent for the task
		agent, err = o.findBestAgent(task)
		if err != nil {
			return "", err
		}
		// Assign task
		task.Assignee = agent.ID()
	}

	task.Status = multiagent.TaskStatusAssigned

	// Store task
	o.tasks[task.ID] = &task

	// Store in memory
	if o.memoryStore != nil {
		taskKey := task.ID // Use task ID directly as key
		o.memoryStore.Store(ctx, taskKey, task)
	}

	// Send task to agent
	taskMsg := &multiagent.Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		From:      multiagent.AgentID("orchestrator"),
		To:        []multiagent.AgentID{agent.ID()},
		Type:      multiagent.MessageTypeRequest,
		Content:   fmt.Sprintf("Execute task %s: %s", task.ID, task.Description),
		Context:   map[string]interface{}{"task_id": task.ID},
		Priority:  task.Priority,
		Timestamp: time.Now(),
	}

	log.Printf("Orchestrator: Sending task message to agent %s", agent.ID())
	if err := o.RouteMessage(ctx, taskMsg); err != nil {
		task.Status = multiagent.TaskStatusFailed
		task.Error = fmt.Sprintf("Failed to send task to agent: %v", err)
		log.Printf("Orchestrator: Failed to send task to agent %s: %v", agent.ID(), err)
		return "", err
	}

	log.Printf("Orchestrator: Successfully assigned task %s to agent %s", task.ID, agent.ID())

	return agent.ID(), nil
}

// GetTaskStatus retrieves the status of a task
func (o *DefaultOrchestrator) GetTaskStatus(ctx context.Context, taskID string) (multiagent.TaskStatus, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	task, exists := o.tasks[taskID]
	if !exists {
		// Try to load from memory
		if o.memoryStore != nil {
			taskKey := fmt.Sprintf("orchestrator:task:%s", taskID)
			if value, err := o.memoryStore.Get(ctx, taskKey); err == nil {
				if task, ok := value.(multiagent.Task); ok {
					return task.Status, nil
				}
			}
		}
		return "", fmt.Errorf("task %s not found", taskID)
	}

	return task.Status, nil
}

// Start begins the orchestrator's operation
func (o *DefaultOrchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return fmt.Errorf("orchestrator is already running")
	}

	o.startTime = time.Now()
	o.running = true

	// Start message router
	o.wg.Add(1)
	go o.messageRouter(ctx)

	// Start event processor
	o.wg.Add(1)
	go o.eventProcessor(ctx)

	// Start health monitor
	o.wg.Add(1)
	go o.healthMonitor(ctx)

	return nil
}

// Stop halts the orchestrator's operation
func (o *DefaultOrchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return nil
	}
	o.running = false
	o.mu.Unlock()

	// Signal stop
	close(o.stopChan)

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// GetSystemHealth returns the current system health
func (o *DefaultOrchestrator) GetSystemHealth() multiagent.SystemHealth {
	o.mu.RLock()
	defer o.mu.RUnlock()

	status := multiagent.SystemStatusOffline
	if o.running {
		status = multiagent.SystemStatusHealthy
	}

	health := multiagent.SystemHealth{
		Status:       status,
		TotalAgents:  len(o.agents),
		ActiveAgents: 0,
		PendingTasks: 0,
		ActiveTasks:  0,
		MessageQueue: len(o.messageQueue),
		Uptime:       time.Since(o.startTime),
		LastCheck:    time.Now(),
		AgentHealth:  make(map[multiagent.AgentID]multiagent.AgentState),
	}

	// Check agent states
	errorCount := 0
	for id, agent := range o.agents {
		state := agent.GetState()
		health.AgentHealth[id] = state

		switch state.Status {
		case multiagent.AgentStatusIdle, multiagent.AgentStatusBusy:
			health.ActiveAgents++
		case multiagent.AgentStatusError:
			errorCount++
		}
	}

	// Count tasks
	for _, task := range o.tasks {
		switch task.Status {
		case multiagent.TaskStatusPending:
			health.PendingTasks++
		case multiagent.TaskStatusAssigned, multiagent.TaskStatusInProgress:
			health.ActiveTasks++
		}
	}

	// Determine overall system status
	if o.running {
		if errorCount > len(o.agents)/2 {
			health.Status = multiagent.SystemStatusCritical
		} else if errorCount > 0 || health.MessageQueue > 800 {
			health.Status = multiagent.SystemStatusDegraded
		}
	}

	return health
}

// RegisterUserResponseHandler registers a handler for user responses
func (o *DefaultOrchestrator) RegisterUserResponseHandler(responseKey string, handler func(string)) {
	o.handlersMutex.Lock()
	defer o.handlersMutex.Unlock()
	
	if existing, exists := o.userResponseHandlers[responseKey]; exists {
		log.Printf("Orchestrator: [REGISTER] ⚠️ Replacing existing handler for key: %s", responseKey)
		_ = existing
	}
	
	o.userResponseHandlers[responseKey] = handler
	log.Printf("Orchestrator: [REGISTER] ✅ Handler registered for key: %s (total: %d)", 
		responseKey, len(o.userResponseHandlers))
}

// UnregisterUserResponseHandler removes a user response handler
func (o *DefaultOrchestrator) UnregisterUserResponseHandler(responseKey string) {
	o.handlersMutex.Lock()
	defer o.handlersMutex.Unlock()
	
	if _, exists := o.userResponseHandlers[responseKey]; exists {
		delete(o.userResponseHandlers, responseKey)
		log.Printf("Orchestrator: [UNREGISTER] ✅ Handler unregistered for key: %s (total: %d)", 
			responseKey, len(o.userResponseHandlers))
	} else {
		log.Printf("Orchestrator: [UNREGISTER] ⚠️ Attempted to unregister non-existent handler: %s", responseKey)
	}
}

// GetOrphanedResponse attempts to retrieve an orphaned response for recovery
func (o *DefaultOrchestrator) GetOrphanedResponse(ctx context.Context, responseKey string) (string, bool) {
	if o.memoryStore == nil {
		return "", false
	}
	
	orphanKey := fmt.Sprintf("orchestrator:orphaned_response:%s", responseKey)
	if value, err := o.memoryStore.Get(ctx, orphanKey); err == nil {
		if orphanData, ok := value.(map[string]interface{}); ok {
			if content, ok := orphanData["content"].(string); ok {
				log.Printf("Orchestrator: [RECOVERY] ✅ Retrieved orphaned response for key: %s", responseKey)
				// Clean up the orphaned response after retrieval
				o.memoryStore.Delete(ctx, orphanKey)
				return content, true
			}
		}
	}
	
	return "", false
}

// GetUserResponseHandlerCount returns the current number of registered handlers
func (o *DefaultOrchestrator) GetUserResponseHandlerCount() int {
	o.handlersMutex.RLock()
	defer o.handlersMutex.RUnlock()
	return len(o.userResponseHandlers)
}

// GetUserResponseHandlerKeys returns the keys of all registered handlers
func (o *DefaultOrchestrator) GetUserResponseHandlerKeys() []string {
	o.handlersMutex.RLock()
	defer o.handlersMutex.RUnlock()
	
	keys := make([]string, 0, len(o.userResponseHandlers))
	for key := range o.userResponseHandlers {
		keys = append(keys, key)
	}
	return keys
}

// handleUserResponse handles responses meant for users with enhanced diagnostics
func (o *DefaultOrchestrator) handleUserResponse(ctx context.Context, response *multiagent.Message) {
	if len(response.To) == 0 {
		log.Printf("Orchestrator: [USER_RESPONSE] ❌ No recipients in message")
		return
	}
	
	responseKey := string(response.To[0])
	log.Printf("Orchestrator: [USER_RESPONSE] Processing response for key: %s", responseKey)
	log.Printf("Orchestrator: [USER_RESPONSE] Response content length: %d", len(response.Content))
	
	// Get handler with detailed logging
	o.handlersMutex.RLock()
	handler, exists := o.userResponseHandlers[responseKey]
	totalHandlers := len(o.userResponseHandlers)
	
	// Log all available handlers for debugging
	availableKeys := make([]string, 0, len(o.userResponseHandlers))
	for key := range o.userResponseHandlers {
		availableKeys = append(availableKeys, key)
	}
	o.handlersMutex.RUnlock()
	
	log.Printf("Orchestrator: [USER_RESPONSE] Handler exists: %v, total handlers: %d", exists, totalHandlers)
	log.Printf("Orchestrator: [USER_RESPONSE] Available handler keys: %v", availableKeys)
	
	if exists && handler != nil {
		log.Printf("Orchestrator: [USER_RESPONSE] ✅ Executing handler for key: %s", responseKey)
		
		// Execute handler with panic recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Orchestrator: [USER_RESPONSE] ❌ Handler panic for %s: %v", responseKey, r)
				}
			}()
			
			handler(response.Content)
			log.Printf("Orchestrator: [USER_RESPONSE] ✅ Handler executed successfully for %s", responseKey)
		}()
		
		return
	}
	
	// Handler not found - this should not happen with our new approach
	log.Printf("Orchestrator: [USER_RESPONSE] ❌ CRITICAL: No handler found for %s", responseKey)
	log.Printf("Orchestrator: [USER_RESPONSE] This indicates a cleanup bug - handler was unregistered prematurely")
	
	// Store as orphaned response for recovery
	if o.memoryStore != nil {
		orphanKey := fmt.Sprintf("orchestrator:orphaned_response:%s", responseKey)
		orphanData := map[string]interface{}{
			"response_key": responseKey,
			"content":      response.Content,
			"timestamp":    time.Now(),
			"from_agent":   response.From,
		}
		
		if err := o.memoryStore.StoreWithTTL(ctx, orphanKey, orphanData, 2*time.Hour); err != nil {
			log.Printf("Orchestrator: [USER_RESPONSE] ❌ Failed to store orphaned response: %v", err)
		} else {
			log.Printf("Orchestrator: [USER_RESPONSE] 💾 Stored orphaned response with 2-hour TTL")
		}
	}
}

// Internal helper methods

func (o *DefaultOrchestrator) findBestAgent(task multiagent.Task) (multiagent.Agent, error) {
	// Simple algorithm: find least loaded agent that can handle the task type
	var bestAgent multiagent.Agent
	lowestWorkload := 101

	for _, agent := range o.agents {
		state := agent.GetState()

		// Skip unavailable agents
		if state.Status != multiagent.AgentStatusIdle && state.Status != multiagent.AgentStatusBusy {
			continue
		}

		// Check if agent can handle this task type
		canHandle := false
		for _, capability := range agent.GetCapabilities() {
			if capability == task.Type {
				canHandle = true
				break
			}
		}

		if canHandle && state.Workload < lowestWorkload {
			bestAgent = agent
			lowestWorkload = state.Workload
		}
	}

	if bestAgent == nil {
		return nil, fmt.Errorf("no suitable agent found for task type: %s", task.Type)
	}

	return bestAgent, nil
}

func (o *DefaultOrchestrator) messageRouter(ctx context.Context) {
	defer o.wg.Done()

	for {
		select {
		case msg := <-o.messageQueue:
			o.routeMessageToAgents(ctx, msg)

		case <-o.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

func (o *DefaultOrchestrator) routeMessageToAgents(ctx context.Context, msg *multiagent.Message) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	log.Printf("Orchestrator: Routing message %s from %s to %v (type: %s)", msg.ID, msg.From, msg.To, msg.Type)

	// Route to each recipient
	for _, recipientID := range msg.To {
		// Special handling for user response keys
		if strings.HasPrefix(string(recipientID), "user_response_") {
			log.Printf("Orchestrator: Routing message %s to user response handler for %s", msg.ID, recipientID)
			o.handleUserResponse(ctx, msg)
			continue
		}

		// Special handling for messages directed to the orchestrator itself
		if recipientID == "orchestrator" {
			log.Printf("Orchestrator: Processing message %s directed to orchestrator", msg.ID)

			// Handle orchestrator-directed messages
			go func(m *multiagent.Message) {
				response := o.handleOrchestratorMessage(ctx, m)
				if response != nil {
					log.Printf("Orchestrator: Routing orchestrator response back")
					if err := o.RouteMessage(ctx, response); err != nil {
						log.Printf("Error routing orchestrator response: %v", err)
					}
				}
			}(msg)
			continue
		}

		agent, exists := o.agents[recipientID]
		if !exists {
			// Log error but continue with other recipients
			log.Printf("Warning: Agent %s not found for message %s", recipientID, msg.ID)
			continue
		}

		log.Printf("Orchestrator: Sending message %s to agent %s (%s)", msg.ID, recipientID, agent.Name())

		// Handle the message directly with the agent
		go func(a multiagent.Agent, m *multiagent.Message) {
			log.Printf("Orchestrator: Processing message %s with agent %s", m.ID, a.ID())
			// Process the message with the agent
			response, err := a.HandleMessage(ctx, m)
			if err != nil {
				log.Printf("Error handling message %s with agent %s: %v", m.ID, a.ID(), err)
				return
			}

			log.Printf("Orchestrator: Agent %s processed message %s, response: %v", a.ID(), m.ID, response != nil)

			// If we got a response, handle it appropriately
			if response != nil {
				log.Printf("Orchestrator: Handling response from agent %s to %v (type: %s)", a.ID(), response.To, response.Type)

				// Check if the response is meant for a user (starts with "user_response_")
				if len(response.To) > 0 && strings.HasPrefix(string(response.To[0]), "user_response_") {
					// This is a response to a user request - handle it via callback
					log.Printf("Orchestrator: Routing response to user callback")
					o.handleUserResponse(ctx, response)
				} else if o.shouldRouteResponse(m, response) {
					// Route the response back through the orchestrator for agent-to-agent communication
					log.Printf("Orchestrator: Routing response back through orchestrator")
					if err := o.RouteMessage(ctx, response); err != nil {
						log.Printf("Error routing response from agent %s: %v", a.ID(), err)
					}
				} else {
					log.Printf("Orchestrator: Terminating message chain to prevent loop")
				}
			}
		}(agent, msg)
	}

	return nil
}

func (o *DefaultOrchestrator) eventProcessor(ctx context.Context) {
	defer o.wg.Done()

	for {
		select {
		case event := <-o.eventQueue:
			o.processEvent(ctx, event)

		case <-o.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

func (o *DefaultOrchestrator) processEvent(ctx context.Context, event *multiagent.Event) {
	// Store event in memory
	if o.memoryStore != nil {
		eventKey := fmt.Sprintf("orchestrator:event:%s", event.ID)
		o.memoryStore.StoreWithTTL(ctx, eventKey, event, 24*time.Hour)
	}

	// Process based on event type
	switch event.Type {
	case multiagent.EventTaskCompleted:
		// Update task status
		if taskID, ok := event.Data["task_id"].(string); ok {
			o.mu.Lock()
			if task, exists := o.tasks[taskID]; exists {
				task.Status = multiagent.TaskStatusCompleted
				task.CompletedAt = &event.Timestamp
			}
			o.mu.Unlock()
		}

	case multiagent.EventTaskFailed:
		// Update task status
		if taskID, ok := event.Data["task_id"].(string); ok {
			o.mu.Lock()
			if task, exists := o.tasks[taskID]; exists {
				task.Status = multiagent.TaskStatusFailed
				if errorMsg, ok := event.Data["error"].(string); ok {
					task.Error = errorMsg
				}
			}
			o.mu.Unlock()
		}
	}
}

func (o *DefaultOrchestrator) healthMonitor(ctx context.Context) {
	defer o.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			health := o.GetSystemHealth()

			// Store health snapshot
			if o.memoryStore != nil {
				healthKey := fmt.Sprintf("orchestrator:health:%d", time.Now().Unix())
				o.memoryStore.StoreWithTTL(ctx, healthKey, health, 7*24*time.Hour)
			}

			// Log if system is degraded
			if health.Status != multiagent.SystemStatusHealthy {
				fmt.Printf("System health warning: %s\n", health.Status)
			}

		case <-o.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// shouldRouteResponse determines if a response should be routed to prevent infinite loops
func (o *DefaultOrchestrator) shouldRouteResponse(originalMsg *multiagent.Message, response *multiagent.Message) bool {
	// Don't route if it's the same agent responding to itself
	if response.From == originalMsg.From {
		return false
	}

	// Always route messages intended for users (user_response_ prefix)
	if len(response.To) > 0 && strings.HasPrefix(string(response.To[0]), "user_response_") {
		log.Printf("Orchestrator: Allowing user-directed response")
		return true
	}

	// Always route final responses from coordination
	if finalResp, ok := response.Context["final_response"].(bool); ok && finalResp {
		log.Printf("Orchestrator: Allowing final coordination response")
		return true
	}

	// Don't route simple acknowledgment messages between agents
	if response.Type == multiagent.MessageTypeResponse {
		// Check for coordination acknowledgments that are just status updates
		if _, hasCoordID := response.Context["coordination_id"]; hasCoordID {
			if ack, hasAck := response.Context["acknowledged"].(bool); hasAck && ack {
				log.Printf("Orchestrator: Skipping routing of coordination acknowledgment")
				return false
			}

			// Allow all coordination responses regardless of content - coordinator will handle them
			log.Printf("Orchestrator: Allowing coordination response")
			return true
		}

		// Check for simple acknowledgment content patterns (only for non-coordination messages)
		contentLower := strings.ToLower(response.Content)
		if (strings.Contains(contentLower, "response received") && len(response.Content) < 50) ||
			(strings.Contains(contentLower, "processed") && len(response.Content) < 50) {
			log.Printf("Orchestrator: Skipping routing of simple acknowledgment")
			return false
		}

		// Don't route generic help messages that create loops (only for non-coordination messages)
		if strings.Contains(contentLower, "thank you for confirming") ||
			(strings.Contains(contentLower, "as your") && strings.Contains(contentLower, "manager") && len(response.Content) < 200) ||
			(strings.Contains(contentLower, "would you like to:") && len(response.Content) < 300) {
			log.Printf("Orchestrator: Skipping routing of generic help message")
			return false
		}
	}

	// Check for reply chains that are getting too long
	// Only block if we're seeing the same two agents repeatedly exchanging messages
	if response.ReplyTo != "" && originalMsg.ReplyTo != "" {
		// Allow coordinator final responses even in reply chains
		if response.From == "coordinator_agent" && strings.HasPrefix(string(response.To[0]), "user_response_") {
			log.Printf("Orchestrator: Allowing coordinator final response despite reply chain")
			return true
		}

		// Only block if it's the same agents talking back and forth
		if response.From == originalMsg.To[0] && response.To[0] == originalMsg.From {
			log.Printf("Orchestrator: Terminating deep reply chain between same agents")
			return false
		}
	}

	// Route if it's a legitimate new message from a different agent
	return response.From != originalMsg.From
}

// handleOrchestratorMessage handles messages directed to the orchestrator itself
func (o *DefaultOrchestrator) handleOrchestratorMessage(ctx context.Context, msg *multiagent.Message) *multiagent.Message {
	log.Printf("Orchestrator: Handling message %s of type %s", msg.ID, msg.Type)

	switch msg.Type {
	case multiagent.MessageTypeResponse:
		// Handle coordination status updates
		if coordinationID, ok := msg.Context["coordination_id"].(string); ok {
			log.Printf("Orchestrator: Received coordination status update for %s", coordinationID)

			// Store coordination status in memory
			if o.memoryStore != nil {
				statusKey := fmt.Sprintf("orchestrator:coordination:%s", coordinationID)
				o.memoryStore.Store(ctx, statusKey, map[string]interface{}{
					"coordination_id": coordinationID,
					"status":          msg.Content,
					"timestamp":       msg.Timestamp,
					"from_agent":      msg.From,
				})
			}

			// Acknowledge the coordination message
			return &multiagent.Message{
				ID:      fmt.Sprintf("msg_orchestrator_%d", time.Now().UnixNano()),
				From:    multiagent.AgentID("orchestrator"),
				To:      []multiagent.AgentID{msg.From},
				Type:    multiagent.MessageTypeResponse,
				Content: fmt.Sprintf("Coordination %s status acknowledged", coordinationID),
				Context: map[string]interface{}{
					"coordination_id": coordinationID,
					"acknowledged":    true,
				},
				Priority:  multiagent.PriorityLow,
				ReplyTo:   msg.ID,
				Timestamp: time.Now(),
			}
		}

	case multiagent.MessageTypeRequest:
		// Handle direct requests to orchestrator
		log.Printf("Orchestrator: Processing direct request: %s", msg.Content)

		// Respond with orchestrator status or capabilities
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_orchestrator_%d", time.Now().UnixNano()),
			From:      multiagent.AgentID("orchestrator"),
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("Orchestrator received request: %s", msg.Content),
			Priority:  multiagent.PriorityMedium,
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}
	}

	// No response needed for other message types
	return nil
}
