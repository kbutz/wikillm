package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// DefaultOrchestrator implements the Orchestrator interface
type DefaultOrchestrator struct {
	agents       map[multiagent.AgentID]multiagent.Agent
	agentsByType map[multiagent.AgentType][]multiagent.Agent
	tasks        map[string]*multiagent.Task
	messageQueue chan *multiagent.Message
	eventQueue   chan *multiagent.Event
	memoryStore  multiagent.MemoryStore
	mu           sync.RWMutex
	startTime    time.Time
	stopChan     chan struct{}
	wg           sync.WaitGroup
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
		agents:       make(map[multiagent.AgentID]multiagent.Agent),
		agentsByType: make(map[multiagent.AgentType][]multiagent.Agent),
		tasks:        make(map[string]*multiagent.Task),
		messageQueue: make(chan *multiagent.Message, config.MessageQueueSize),
		eventQueue:   make(chan *multiagent.Event, config.EventQueueSize),
		memoryStore:  config.MemoryStore,
		stopChan:     make(chan struct{}),
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

	// Emit registration event
	event := &multiagent.Event{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Type:      multiagent.EventAgentRegistered,
		Source:    string(agentID),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"agent_id":   agentID,
			"agent_type": agentType,
			"agent_name": agent.Name(),
		},
	}
	
	select {
	case o.eventQueue <- event:
	default:
		// Event queue full, log but don't block
		fmt.Printf("Warning: Event queue full, dropping event %s\n", event.ID)
	}

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

	// Emit unregistration event
	event := &multiagent.Event{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Type:      multiagent.EventAgentUnregistered,
		Source:    string(agentID),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"agent_id": agentID,
		},
	}
	
	select {
	case o.eventQueue <- event:
	default:
		fmt.Printf("Warning: Event queue full, dropping event %s\n", event.ID)
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

	// Add to message queue
	select {
	case o.messageQueue <- msg:
		// Emit message sent event
		event := &multiagent.Event{
			ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
			Type:      multiagent.EventMessageSent,
			Source:    string(msg.From),
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message_id": msg.ID,
				"from":       msg.From,
				"to":         msg.To,
				"type":       msg.Type,
			},
		}
		
		select {
		case o.eventQueue <- event:
		default:
			// Event queue full
		}
		
		return nil
		
	case <-ctx.Done():
		return ctx.Err()
		
	default:
		return fmt.Errorf("message queue full")
	}
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

	// Set initial status
	task.Status = multiagent.TaskStatusPending
	task.CreatedAt = time.Now()

	// Find best agent for the task
	agent, err := o.findBestAgent(task)
	if err != nil {
		return "", err
	}

	// Assign task
	task.Assignee = agent.ID()
	task.Status = multiagent.TaskStatusAssigned
	
	// Store task
	o.tasks[task.ID] = &task

	// Store in memory
	if o.memoryStore != nil {
		taskKey := fmt.Sprintf("orchestrator:task:%s", task.ID)
		o.memoryStore.Store(ctx, taskKey, task)
	}

	// Send task to agent
	taskMsg := &multiagent.Message{
		ID:       fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		From:     multiagent.AgentID("orchestrator"),
		To:       []multiagent.AgentID{agent.ID()},
		Type:     multiagent.MessageTypeCommand,
		Content:  fmt.Sprintf("Execute task %s: %s", task.ID, task.Description),
		Context:  map[string]interface{}{"task": task},
		Priority: task.Priority,
	}

	if err := o.RouteMessage(ctx, taskMsg); err != nil {
		task.Status = multiagent.TaskStatusFailed
		task.Error = fmt.Sprintf("Failed to send task to agent: %v", err)
		return "", err
	}

	// Emit task assigned event
	event := &multiagent.Event{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Type:      multiagent.EventTaskAssigned,
		Source:    "orchestrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task_id":  task.ID,
			"assignee": task.Assignee,
		},
	}
	
	select {
	case o.eventQueue <- event:
	default:
		// Event queue full
	}

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
	if o.startTime.IsZero() {
		o.startTime = time.Now()
	}
	o.mu.Unlock()

	// Start all registered agents
	for _, agent := range o.agents {
		if err := agent.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize agent %s: %w", agent.ID(), err)
		}
		
		if err := agent.Start(ctx); err != nil {
			return fmt.Errorf("failed to start agent %s: %w", agent.ID(), err)
		}
	}

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
	// Signal stop
	close(o.stopChan)

	// Stop all agents
	var errors []error
	for _, agent := range o.agents {
		if err := agent.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop agent %s: %w", agent.ID(), err))
		}
	}

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

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// GetSystemHealth returns the current system health
func (o *DefaultOrchestrator) GetSystemHealth() multiagent.SystemHealth {
	o.mu.RLock()
	defer o.mu.RUnlock()

	health := multiagent.SystemHealth{
		Status:       multiagent.SystemStatusHealthy,
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
	if errorCount > len(o.agents)/2 {
		health.Status = multiagent.SystemStatusCritical
	} else if errorCount > 0 || health.MessageQueue > 800 {
		health.Status = multiagent.SystemStatusDegraded
	}

	return health
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

func (o *DefaultOrchestrator) routeMessageToAgents(ctx context.Context, msg *multiagent.Message) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Route to each recipient
	for _, recipientID := range msg.To {
		agent, exists := o.agents[recipientID]
		if !exists {
			// Log error but continue with other recipients
			fmt.Printf("Warning: Agent %s not found for message %s\n", recipientID, msg.ID)
			continue
		}

		// Send message to agent (non-blocking)
		go func(a multiagent.Agent, m *multiagent.Message) {
			if err := a.SendMessage(ctx, m); err != nil {
				fmt.Printf("Error sending message %s to agent %s: %v\n", m.ID, a.ID(), err)
			}

			// Emit message received event
			event := &multiagent.Event{
				ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
				Type:      multiagent.EventMessageReceived,
				Source:    string(a.ID()),
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"message_id": m.ID,
					"from":       m.From,
				},
			}
			
			select {
			case o.eventQueue <- event:
			default:
				// Event queue full
			}
		}(agent, msg)
	}
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
