package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kbutz/wikillm/multiagent"
	"github.com/kbutz/wikillm/multiagent/agents"
	"github.com/kbutz/wikillm/multiagent/memory"
	"github.com/kbutz/wikillm/multiagent/orchestrator"
	"github.com/kbutz/wikillm/multiagent/tools"
)

// MultiAgentService provides a complete multi-agent system with memory, tools, and orchestration
type MultiAgentService struct {
	memoryStore  multiagent.MemoryStore
	orchestrator multiagent.Orchestrator
	agents       map[multiagent.AgentID]multiagent.Agent
	tools        map[string]multiagent.Tool
	llmProvider  multiagent.LLMProvider
	baseDir      string
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
		memoryStore:  memoryStore,
		orchestrator: orch,
		agents:       make(map[multiagent.AgentID]multiagent.Agent),
		tools:        make(map[string]multiagent.Tool),
		llmProvider:  config.LLMProvider,
		baseDir:      config.BaseDir,
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
		if err := agent.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start agent %s: %v", id, err)
		}
	}

	log.Println("MultiAgentService started successfully")
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

	log.Println("MultiAgentService stopped successfully")
	return nil
}

// ProcessUserMessage processes a user message and returns a response
func (s *MultiAgentService) ProcessUserMessage(ctx context.Context, userID string, message string) (string, error) {
	// Create a conversation ID if not provided
	conversationID := fmt.Sprintf("conv_%s_%d", userID, time.Now().UnixNano())

	// Create a message for the conversation agent
	msg := &multiagent.Message{
		ID:        fmt.Sprintf("msg_user_%d", time.Now().UnixNano()),
		From:      multiagent.AgentID(userID),
		To:        []multiagent.AgentID{multiagent.AgentID("conversation_agent")},
		Type:      multiagent.MessageTypeRequest,
		Content:   message,
		Priority:  multiagent.PriorityMedium,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"conversation_id": conversationID,
			"source":          "user",
		},
	}

	// Route the message through the orchestrator
	if err := s.orchestrator.RouteMessage(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to route message: %w", err)
	}

	// For synchronous API, we need to wait for a response
	// In a real implementation, this would be handled asynchronously
	// For now, we'll wait a reasonable amount of time for the LLM to respond
	time.Sleep(1 * time.Second)

	// Get the conversation from memory
	convKey := fmt.Sprintf("conversation:%s", conversationID)
	convInterface, err := s.memoryStore.Get(ctx, convKey)
	if err != nil {
		return "I'm processing your request. Please check back in a moment.", nil
	}

	// Try to extract the latest assistant response
	var conversation multiagent.ConversationContext
	convData, err := json.Marshal(convInterface)
	if err != nil {
		return "I'm working on your request. Please wait a moment.", nil
	}

	if err := json.Unmarshal(convData, &conversation); err != nil {
		return "I'm analyzing your message. I'll respond shortly.", nil
	}

	// Find the latest assistant response
	for i := len(conversation.Messages) - 1; i >= 0; i-- {
		msg := conversation.Messages[i]
		if msg.Role == "assistant" {
			return msg.Content, nil
		}
	}

	return "I've received your message and I'm working on a response.", nil
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

// GetSystemHealth returns the current health of the system
func (s *MultiAgentService) GetSystemHealth() multiagent.SystemHealth {
	return s.orchestrator.GetSystemHealth()
}

// initializeTools initializes all tools
func (s *MultiAgentService) initializeTools() error {
	// Create memory tool
	memoryTool := tools.NewMemoryTool(s.memoryStore)
	s.tools[memoryTool.Name()] = memoryTool

	// Create task tool
	taskTool := tools.NewTaskTool(s.memoryStore, s.orchestrator)
	s.tools[taskTool.Name()] = taskTool

	return nil
}

// initializeAgents initializes all agents
func (s *MultiAgentService) initializeAgents() error {
	// Create a list of tools for agents
	agentTools := make([]multiagent.Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		agentTools = append(agentTools, tool)
	}

	// Create conversation agent
	conversationAgent := agents.NewConversationAgent(agents.BaseAgentConfig{
		ID:           "conversation_agent",
		Type:         multiagent.AgentTypeConversation,
		Name:         "Conversation Agent",
		Description:  "Handles natural language conversations with users",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[conversationAgent.ID()] = conversationAgent

	// Create coordinator agent
	coordinatorAgent := agents.NewCoordinatorAgent(agents.BaseAgentConfig{
		ID:           "coordinator_agent",
		Type:         multiagent.AgentTypeCoordinator,
		Name:         "Coordinator Agent",
		Description:  "Coordinates specialist agents to handle complex tasks",
		Tools:        agentTools,
		LLMProvider:  s.llmProvider,
		MemoryStore:  s.memoryStore,
		Orchestrator: s.orchestrator,
	})
	s.agents[coordinatorAgent.ID()] = coordinatorAgent

	// Register agents with orchestrator
	for _, agent := range s.agents {
		if err := s.orchestrator.RegisterAgent(agent); err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agent.ID(), err)
		}
	}

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
