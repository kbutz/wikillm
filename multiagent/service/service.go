package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
	"github.com/kbutz/wikillm/multiagent/agents"
	"github.com/kbutz/wikillm/multiagent/memory"
	"github.com/kbutz/wikillm/multiagent/orchestrator"
	"github.com/kbutz/wikillm/multiagent/tools"
)

// MultiAgentService provides a complete multi-agent system with memory, tools, and orchestration
type MultiAgentService struct {
	memoryStore     multiagent.MemoryStore
	orchestrator    multiagent.Orchestrator
	agents          map[multiagent.AgentID]multiagent.Agent
	tools           map[string]multiagent.Tool
	llmProvider     multiagent.LLMProvider
	baseDir         string
	pendingRequests map[string]chan string // Track pending user requests
	requestsMutex   sync.RWMutex
}

// ServiceConfig holds configuration for creating a MultiAgentService
type ServiceConfig struct {
	BaseDir     string
	LLMProvider multiagent.LLMProvider
}

// NewMultiAgentService creates a new multi-agent service
func NewMultiAgentService(config ServiceConfig) (*MultiAgentService, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Initialize memory store
	memoryDir := filepath.Join(config.BaseDir, "memory")
	memoryStore, err := memory.NewFileMemoryStore(memoryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize memory store: %w", err)
	}

	// Initialize orchestrator
	orch := orchestrator.NewOrchestrator(orchestrator.OrchestratorConfig{
		MemoryStore:      memoryStore,
		MessageQueueSize: 1000,
		EventQueueSize:   500,
	})

	service := &MultiAgentService{
		memoryStore:     memoryStore,
		orchestrator:    orch,
		agents:          make(map[multiagent.AgentID]multiagent.Agent),
		tools:           make(map[string]multiagent.Tool),
		llmProvider:     config.LLMProvider,
		baseDir:         config.BaseDir,
		pendingRequests: make(map[string]chan string),
	}

	// Initialize tools
	if err := service.initializeTools(); err != nil {
		return nil, fmt.Errorf("failed to initialize tools: %w", err)
	}

	// Initialize agents
	if err := service.initializeAgents(); err != nil {
		return nil, fmt.Errorf("failed to initialize agents: %w", err)
	}

	return service, nil
}

// Start starts the multi-agent service
func (s *MultiAgentService) Start(ctx context.Context) error {
	// Start orchestrator
	if err := s.orchestrator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start orchestrator: %w", err)
	}

	// Start all agents
	for id, agent := range s.agents {
		// Initialize agent first
		if err := agent.Initialize(ctx); err != nil {
			log.Printf("Warning: Failed to initialize agent %s: %v", id, err)
			continue
		}

		// Then start agent
		if err := agent.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start agent %s: %v", id, err)
		} else {
			log.Printf("✅ Started agent: %s (%s)", agent.Name(), id)
		}
	}

	log.Println("🚀 MultiAgentService started successfully with all specialist agents")
	return nil
}

// Stop stops the multi-agent service
func (s *MultiAgentService) Stop(ctx context.Context) error {
	// Stop all agents
	for id, agent := range s.agents {
		if err := agent.Stop(ctx); err != nil {
			log.Printf("Warning: Failed to stop agent %s: %v", id, err)
		}
	}

	// Stop orchestrator
	if err := s.orchestrator.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop orchestrator: %w", err)
	}

	// Close any pending request channels
	s.requestsMutex.Lock()
	for _, ch := range s.pendingRequests {
		close(ch)
	}
	s.pendingRequests = make(map[string]chan string)
	s.requestsMutex.Unlock()

	log.Println("🛑 MultiAgentService stopped successfully")
	return nil
}

// ProcessUserMessage processes a user message and returns a response
func (s *MultiAgentService) ProcessUserMessage(ctx context.Context, userID string, message string) (string, error) {
	conversationID := fmt.Sprintf("conv_%s", userID)
	log.Printf("Service: Using consistent conversation ID: %s", conversationID)

	responseKey := fmt.Sprintf("user_response_%s_%d", userID, time.Now().UnixNano())
	responseChannel := make(chan string, 10) // Increased buffer

	// Handler state tracking
	var handlerState struct {
		mutex      sync.RWMutex
		registered bool
		called     bool
		response   string
		timestamp  time.Time
	}

	// Create persistent handler
	handler := func(response string) {
		handlerState.mutex.Lock()
		defer handlerState.mutex.Unlock()

		// Always store the response, even if it's an acknowledgment
		handlerState.response = response
		handlerState.timestamp = time.Now()

		log.Printf("Service: [HANDLER] ✅ Called for key %s at %v", responseKey, handlerState.timestamp)
		log.Printf("Service: [HANDLER] Response length: %d", len(response))

		// Check if this is an acknowledgment message from the conversation agent
		if strings.Contains(response, "I'm working on your request and consulting with specialists") {
			log.Printf("Service: [HANDLER] 🔄 Received acknowledgment message, waiting for final response")
			// Don't mark as called yet and don't send to channel
			// This ensures we'll still process the final response from the coordinator
			return
		}

		// If we already processed a response (that wasn't an acknowledgment), ignore duplicates
		if handlerState.called {
			log.Printf("Service: [HANDLER] Already called for key %s, ignoring duplicate", responseKey)
			return
		}

		// Mark as called for non-acknowledgment responses
		handlerState.called = true

		// Send to channel with timeout
		select {
		case responseChannel <- response:
			log.Printf("Service: [HANDLER] ✅ Sent to channel for key: %s", responseKey)
		case <-time.After(10 * time.Second):
			log.Printf("Service: [HANDLER] ⚠️ Channel send timeout for key: %s", responseKey)
			// Still store the response for polling
		}
	}

	// Register handler with orchestrator
	if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
		orch.RegisterUserResponseHandler(responseKey, handler)
		handlerState.mutex.Lock()
		handlerState.registered = true
		handlerState.mutex.Unlock()
		log.Printf("Service: [REGISTER] ✅ Handler registered for key: %s", responseKey)
	} else {
		return "", fmt.Errorf("orchestrator does not support user response handlers")
	}

	// *** CRITICAL: DO NOT SCHEDULE CLEANUP HERE ***
	// The handler will remain registered until explicitly cleaned up after success

	// Create message
	msg := &multiagent.Message{
		ID:        fmt.Sprintf("msg_user_%d", time.Now().UnixNano()),
		From:      multiagent.AgentID(responseKey),
		To:        []multiagent.AgentID{multiagent.AgentID("conversation_agent")},
		Type:      multiagent.MessageTypeRequest,
		Content:   message,
		Priority:  multiagent.PriorityMedium,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"conversation_id": conversationID,
			"source":          "user",
			"user_id":         userID,
			"response_key":    responseKey,
		},
	}

	// Route message
	if err := s.orchestrator.RouteMessage(ctx, msg); err != nil {
		// Only cleanup on immediate routing failure
		log.Printf("Service: [ERROR] Message routing failed, cleaning up handler")
		if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
			orch.UnregisterUserResponseHandler(responseKey)
		}
		return "", fmt.Errorf("failed to route message: %w", err)
	}

	log.Printf("Service: [ROUTE] ✅ Message routed successfully")

	// Wait with enhanced monitoring and orphan recovery
	startTime := time.Now()

	// Status check every 2 seconds for more responsive recovery
	statusTicker := time.NewTicker(2 * time.Second)
	defer statusTicker.Stop()

	// Main response wait loop
	for {
		select {
		case response := <-responseChannel:
			elapsed := time.Since(startTime)
			log.Printf("Service: [SUCCESS] ✅ Response received via channel after %v", elapsed)

			// Wait a short time before unregistering to ensure any in-flight responses are processed
			// This helps prevent race conditions where the handler is unregistered too early
			time.Sleep(500 * time.Millisecond)

			// NOW we can cleanup since we got the response
			if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
				orch.UnregisterUserResponseHandler(responseKey)
				log.Printf("Service: [CLEANUP] Handler unregistered after successful response")
			}
			return response, nil

		case <-statusTicker.C:
			elapsed := time.Since(startTime)

			// FIRST: Check for orphaned responses immediately
			if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
				if orphanResponse, found := orch.GetOrphanedResponse(ctx, responseKey); found {
					log.Printf("Service: [RECOVERY] ✅ Orphaned response recovered after %v", elapsed)
					// Wait a short time before unregistering
					time.Sleep(500 * time.Millisecond)
					orch.UnregisterUserResponseHandler(responseKey)
					return orphanResponse, nil
				}
			}

			// Check handler state second
			handlerState.mutex.RLock()
			called := handlerState.called
			response := handlerState.response
			handlerState.mutex.RUnlock()

			// If handler was called but response wasn't delivered via channel
			if called && len(response) > 0 {
				log.Printf("Service: [RECOVERY] ✅ Found response in handler state after %v", elapsed)
				// Wait a short time before unregistering
				time.Sleep(500 * time.Millisecond)
				if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
					orch.UnregisterUserResponseHandler(responseKey)
				}
				return response, nil
			}

			// Verify handler still exists and re-register if missing
			if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
				keys := orch.GetUserResponseHandlerKeys()
				handlerExists := false
				for _, key := range keys {
					if key == responseKey {
						handlerExists = true
						break
					}
				}
				totalHandlers := orch.GetUserResponseHandlerCount()

				if !handlerExists {
					log.Printf("Service: [ERROR] ❌ Handler disappeared after %v! Re-registering...", elapsed)
					orch.RegisterUserResponseHandler(responseKey, handler)
					handlerState.mutex.Lock()
					handlerState.registered = true
					handlerState.mutex.Unlock()
				} else {
					// Only log every 10 seconds to reduce noise
					if int(elapsed.Seconds())%10 == 0 {
						log.Printf("Service: [STATUS] ⏳ Waiting %v, handler exists, total: %d", elapsed, totalHandlers)
					}
				}
			}

			// Ultimate timeout - be more aggressive about recovery
			if elapsed > 10*time.Minute {
				log.Printf("Service: [TIMEOUT] ❌ Timeout reached after %v", elapsed)

				// Final comprehensive check
				handlerState.mutex.RLock()
				finalResponse := handlerState.response
				handlerState.mutex.RUnlock()

				if len(finalResponse) > 0 {
					log.Printf("Service: [SUCCESS] ✅ Final response found in handler state")
					// Wait a short time before unregistering
					time.Sleep(500 * time.Millisecond)
					if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
						orch.UnregisterUserResponseHandler(responseKey)
					}
					return finalResponse, nil
				}

				// Final orphan check
				if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
					if orphanResponse, found := orch.GetOrphanedResponse(ctx, responseKey); found {
						log.Printf("Service: [SUCCESS] ✅ Final orphaned response recovered")
						// Wait a short time before unregistering
						time.Sleep(500 * time.Millisecond)
						orch.UnregisterUserResponseHandler(responseKey)
						return orphanResponse, nil
					}
				}

				// Give up - but keep the handler registered for a bit longer
				// This helps with race conditions where the response arrives just after the timeout
				log.Printf("Service: [TIMEOUT] ⚠️ Keeping handler registered for potential late responses")

				// Return timeout message but don't unregister the handler yet
				// The handler will be garbage collected eventually
				return fmt.Sprintf("Request timed out after %v. The system may still be processing your request. Please try again.", elapsed.Round(time.Second)), nil
			}

		case <-ctx.Done():
			log.Printf("Service: [CANCELLED] Context cancelled")
			// Wait a short time before unregistering
			time.Sleep(500 * time.Millisecond)
			if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
				orch.UnregisterUserResponseHandler(responseKey)
			}
			return "", ctx.Err()
		}
	}
}

// GetAgent returns an agent by ID
func (s *MultiAgentService) GetAgent(id multiagent.AgentID) (multiagent.Agent, error) {
	agent, exists := s.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

// GetOrchestrator returns the orchestrator
func (s *MultiAgentService) GetOrchestrator() multiagent.Orchestrator {
	return s.orchestrator
}

// GetMemoryStore returns the memory store
func (s *MultiAgentService) GetMemoryStore() multiagent.MemoryStore {
	return s.memoryStore
}

// AgentInfo provides information about an agent for display purposes
type AgentInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
}

// SystemHealthInfo provides extended health information for display purposes
type SystemHealthInfo struct {
	Status            string        `json:"status"`
	ActiveAgents      int           `json:"active_agents"`
	TotalAgents       int           `json:"total_agents"`
	MessagesProcessed int           `json:"messages_processed"`
	MessageQueueSize  int           `json:"message_queue_size"`
	EventsProcessed   int           `json:"events_processed"`
	EventQueueSize    int           `json:"event_queue_size"`
	Uptime            time.Duration `json:"uptime"`
}

// ListAgents returns information about all registered agents
func (s *MultiAgentService) ListAgents() []AgentInfo {
	agents := s.orchestrator.ListAgents()
	agentInfos := make([]AgentInfo, 0, len(agents))

	for _, agent := range agents {
		state := agent.GetState()
		agentInfos = append(agentInfos, AgentInfo{
			ID:           string(agent.ID()),
			Name:         agent.Name(),
			Description:  agent.Description(),
			Status:       string(state.Status),
			Capabilities: state.Capabilities,
		})
	}

	return agentInfos
}

// GetSystemHealth returns the current health of the system
func (s *MultiAgentService) GetSystemHealth() SystemHealthInfo {
	health := s.orchestrator.GetSystemHealth()

	// Get message and event queue stats from orchestrator if available
	messagesProcessed := 0
	eventsProcessed := 0
	messageQueueSize := health.MessageQueue
	eventQueueSize := 0

	if orch, ok := s.orchestrator.(*orchestrator.DefaultOrchestrator); ok {
		// These methods might not exist, but if they do, we'll use them
		if statsProvider, ok := interface{}(orch).(interface{
			GetMessageQueueSize() int
			GetEventQueueSize() int
			GetMessagesProcessed() int
			GetEventsProcessed() int
		}); ok {
			messageQueueSize = statsProvider.GetMessageQueueSize()
			eventQueueSize = statsProvider.GetEventQueueSize()
			messagesProcessed = statsProvider.GetMessagesProcessed()
			eventsProcessed = statsProvider.GetEventsProcessed()
		}
	}

	return SystemHealthInfo{
		Status:            string(health.Status),
		ActiveAgents:      health.ActiveAgents,
		TotalAgents:       health.TotalAgents,
		MessagesProcessed: messagesProcessed,
		MessageQueueSize:  messageQueueSize,
		EventsProcessed:   eventsProcessed,
		EventQueueSize:    eventQueueSize,
		Uptime:            health.Uptime,
	}
}

// initializeTools initializes all tools
func (s *MultiAgentService) initializeTools() error {
	// Create memory tool
	memoryTool := tools.NewMemoryTool(s.memoryStore)
	s.tools[memoryTool.Name()] = memoryTool

	// Create task tool
	taskTool := tools.NewTaskTool(s.memoryStore, s.orchestrator)
	s.tools[taskTool.Name()] = taskTool

	log.Printf("📚 Initialized %d tools", len(s.tools))
	return nil
}

// initializeAgents initializes ALL agents including new specialist agents
func (s *MultiAgentService) initializeAgents() error {
	// Create a list of tools for agents
	agentTools := make([]multiagent.Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		agentTools = append(agentTools, tool)
	}

	log.Println("🤖 Initializing all specialist agents...")

	// 1. Create Project Manager Agent
	projectManagerAgent := agents.NewProjectManagerAgent(agents.BaseAgentConfig{
		ID:           "project_manager_agent",
		Name:         "Project Manager",
		Description:  "Specialized in project planning, task management, and progress tracking",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[projectManagerAgent.ID()] = projectManagerAgent

	// 2. Create Task Manager Agent
	taskManagerAgent := agents.NewTaskManagerAgent(agents.BaseAgentConfig{
		ID:           "task_manager_agent",
		Name:         "Task Manager",
		Description:  "Personal productivity specialist using GTD methodology",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[taskManagerAgent.ID()] = taskManagerAgent

	// 3. Create Research Assistant Agent
	researchAssistantAgent := agents.NewResearchAssistantAgent(agents.BaseAgentConfig{
		ID:           "research_assistant_agent",
		Name:         "Research Assistant",
		Description:  "Information gathering, fact-checking, and knowledge synthesis specialist",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[researchAssistantAgent.ID()] = researchAssistantAgent

	// 4. Create Scheduler Agent
	schedulerAgent := agents.NewSchedulerAgent(agents.BaseAgentConfig{
		ID:           "scheduler_agent",
		Name:         "Scheduler",
		Description:  "Calendar management and appointment scheduling specialist",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[schedulerAgent.ID()] = schedulerAgent

	// 5. Create Communication Manager Agent
	communicationManagerAgent := agents.NewCommunicationManagerAgent(agents.BaseAgentConfig{
		ID:           "communication_manager_agent",
		Name:         "Communication Manager",
		Description:  "Contact management and communication coordination specialist",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[communicationManagerAgent.ID()] = communicationManagerAgent

	// 6. Create Conversation Agent (handles routing to specialists)
	conversationAgent := agents.NewConversationAgent(agents.BaseAgentConfig{
		ID:           "conversation_agent",
		Type:         multiagent.AgentTypeConversation,
		Name:         "Conversation Agent",
		Description:  "Natural language interface that routes requests to appropriate specialists",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[conversationAgent.ID()] = conversationAgent

	// 7. Create Coordinator Agent (manages multi-agent workflows)
	coordinatorAgent := agents.NewCoordinatorAgent(agents.BaseAgentConfig{
		ID:           "coordinator_agent",
		Type:         multiagent.AgentTypeCoordinator,
		Name:         "Coordinator Agent",
		Description:  "Coordinates specialist agents to handle complex multi-step tasks",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[coordinatorAgent.ID()] = coordinatorAgent

	// Register all agents with orchestrator
	for _, agent := range s.agents {
		if err := s.orchestrator.RegisterAgent(agent); err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agent.ID(), err)
		}
	}

	log.Printf("📋 Initialized %d specialist agents", len(s.agents))
	return nil
}

// AddAgent adds a new agent to the service
func (s *MultiAgentService) AddAgent(agent multiagent.Agent) error {
	// Check if agent already exists
	if _, exists := s.agents[agent.ID()]; exists {
		return fmt.Errorf("agent with ID %s already exists", agent.ID())
	}

	// Register with orchestrator
	if err := s.orchestrator.RegisterAgent(agent); err != nil {
		return fmt.Errorf("failed to register agent with orchestrator: %w", err)
	}

	// Add to agents map
	s.agents[agent.ID()] = agent

	// Initialize and start agent if service is running
	if s.orchestrator.GetSystemHealth().Status != multiagent.SystemStatusOffline {
		ctx := context.Background()
		if err := agent.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize agent: %w", err)
		}
		if err := agent.Start(ctx); err != nil {
			return fmt.Errorf("failed to start agent: %w", err)
		}
	}

	return nil
}

// AddTool adds a new tool to the service
func (s *MultiAgentService) AddTool(tool multiagent.Tool) error {
	// Check if tool already exists
	if _, exists := s.tools[tool.Name()]; exists {
		return fmt.Errorf("tool with name %s already exists", tool.Name())
	}

	// Add to tools map
	s.tools[tool.Name()] = tool

	return nil
}
