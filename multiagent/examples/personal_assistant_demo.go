package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kbutz/wikillm/multiagent"
	"github.com/kbutz/wikillm/multiagent/agents"
	"github.com/kbutz/wikillm/multiagent/memory"
	"github.com/kbutz/wikillm/multiagent/orchestrator"
)

// MockLLMProvider implements the LLMProvider interface for demonstration
type MockLLMProvider struct {
	name string
}

func (m *MockLLMProvider) Name() string {
	return m.name
}

func (m *MockLLMProvider) Query(ctx context.Context, prompt string) (string, error) {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Return a mock response based on the prompt content
	if len(prompt) > 100 {
		return "I understand your request and I'm processing it. This is a comprehensive response based on the detailed prompt you provided.", nil
	}
	return "Thank you for your message. I'm here to help with your personal assistant needs.", nil
}

func (m *MockLLMProvider) QueryWithTools(ctx context.Context, prompt string, tools []multiagent.Tool) (string, error) {
	return m.Query(ctx, prompt)
}

func main() {
	fmt.Println("🤖 Initializing Personal Assistant Multi-Agent System...")

	// Create context
	ctx := context.Background()

	// Initialize memory store
	memoryStore, err := memory.NewFileMemoryStore("./assistant_memory")
	if err != nil {
		log.Fatalf("Failed to initialize memory store: %v", err)
	}
	if err := memoryStore.Cleanup(ctx); err != nil {
		log.Printf("Warning: Memory cleanup failed: %v", err)
	}

	// Initialize LLM provider
	llmProvider := NewLMStudioProvider("http://localhost:1234/v1",
		WithTemperature(0.7),
		WithMaxTokens(2048),
		WithDebug(true), // Enable debug mode to help diagnose issues
	)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(orchestrator.OrchestratorConfig{
		MessageQueueSize: 10,
		MemoryStore:      memoryStore,
	})

	// Start orchestrator
	if err := orch.Start(ctx); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}
	defer orch.Stop(ctx)

	// Create base agent config
	baseConfig := agents.BaseAgentConfig{
		LLMProvider:  llmProvider,
		MemoryStore:  memoryStore,
		Orchestrator: orch,
		Tools:        []multiagent.Tool{},
	}

	fmt.Println("📋 Creating specialized agents...")

	// 1. Create Project Manager Agent
	projectManagerConfig := baseConfig
	projectManagerConfig.ID = "project_manager_agent"
	projectManagerConfig.Name = "Project Manager"
	projectManagerConfig.Description = "Specialized in project planning, task management, and progress tracking"
	projectManager := agents.NewProjectManagerAgent(projectManagerConfig)

	// 2. Create Task Manager Agent
	taskManagerConfig := baseConfig
	taskManagerConfig.ID = "task_manager_agent"
	taskManagerConfig.Name = "Task Manager"
	taskManagerConfig.Description = "Personal productivity specialist using GTD methodology"
	taskManager := agents.NewTaskManagerAgent(taskManagerConfig)

	// 3. Create Research Assistant Agent
	researchConfig := baseConfig
	researchConfig.ID = "research_assistant_agent"
	researchConfig.Name = "Research Assistant"
	researchConfig.Description = "Information gathering, fact-checking, and knowledge synthesis specialist"
	researchAssistant := agents.NewResearchAssistantAgent(researchConfig)

	// 4. Create Scheduler Agent
	schedulerConfig := baseConfig
	schedulerConfig.ID = "scheduler_agent"
	schedulerConfig.Name = "Scheduler"
	schedulerConfig.Description = "Calendar management and appointment scheduling specialist"
	scheduler := agents.NewSchedulerAgent(schedulerConfig)

	// 5. Create Communication Manager Agent
	commManagerConfig := baseConfig
	commManagerConfig.ID = "communication_manager_agent"
	commManagerConfig.Name = "Communication Manager"
	commManagerConfig.Description = "Contact management and communication coordination specialist"
	commManager := agents.NewCommunicationManagerAgent(commManagerConfig)

	// 6. Create Conversation Agent (existing)
	conversationConfig := baseConfig
	conversationConfig.ID = "conversation_agent"
	conversationConfig.Name = "Conversation Agent"
	conversationConfig.Description = "Natural language interaction and user interface specialist"
	conversationAgent := agents.NewConversationAgent(conversationConfig)

	// 7. Create Coordinator Agent (existing)
	coordinatorConfig := baseConfig
	coordinatorConfig.ID = "coordinator_agent"
	coordinatorConfig.Name = "Coordinator Agent"
	coordinatorConfig.Description = "Multi-agent coordination and workflow management specialist"
	coordinatorAgent := agents.NewCoordinatorAgent(coordinatorConfig)

	// Register all agents with orchestrator
	agents_list := []multiagent.Agent{
		projectManager,
		taskManager,
		researchAssistant,
		scheduler,
		commManager,
		conversationAgent,
		coordinatorAgent,
	}

	fmt.Println("🔗 Registering agents with orchestrator...")
	for _, agent := range agents_list {
		if err := orch.RegisterAgent(agent); err != nil {
			log.Fatalf("Failed to register agent %s: %v", agent.ID(), err)
		}

		// Initialize and start each agent
		if err := agent.Initialize(ctx); err != nil {
			log.Printf("Warning: Failed to initialize agent %s: %v", agent.ID(), err)
			continue
		}

		if err := agent.Start(ctx); err != nil {
			log.Printf("Warning: Failed to start agent %s: %v", agent.ID(), err)
			continue
		}

		fmt.Printf("✅ %s (%s) - Ready\n", agent.Name(), agent.ID())
	}

	// Demonstrate the personal assistant capabilities
	fmt.Println("\n🎯 Personal Assistant System Ready!")
	fmt.Println("=====================================")

	// Test different agent capabilities
	testScenarios := []struct {
		description string
		message     string
		targetAgent string
	}{
		{
			description: "Project Management",
			message:     "Create a new project called 'Website Redesign' with high priority due next month",
			targetAgent: "project_manager_agent",
		},
		{
			description: "Task Management",
			message:     "Add a task to review quarterly reports, high priority, due this Friday",
			targetAgent: "task_manager_agent",
		},
		{
			description: "Research Request",
			message:     "Research the latest trends in artificial intelligence and machine learning for 2024",
			targetAgent: "research_assistant_agent",
		},
		{
			description: "Calendar Scheduling",
			message:     "Schedule a team meeting for tomorrow at 2 PM for 1 hour in the conference room",
			targetAgent: "scheduler_agent",
		},
		{
			description: "Contact Management",
			message:     "Add John Smith from TechCorp as a new client contact with high priority",
			targetAgent: "communication_manager_agent",
		},
		{
			description: "General Conversation",
			message:     "What's my schedule looking like for the rest of the week?",
			targetAgent: "conversation_agent",
		},
	}

	fmt.Println("🧪 Running demonstration scenarios...")
	for i, scenario := range testScenarios {
		fmt.Printf("\n%d. %s\n", i+1, scenario.description)
		fmt.Printf("   Request: %s\n", scenario.message)

		// Create test message
		testMessage := &multiagent.Message{
			ID:        fmt.Sprintf("demo_msg_%d_%d", i+1, time.Now().UnixNano()),
			From:      "demo_user",
			To:        []multiagent.AgentID{multiagent.AgentID(scenario.targetAgent)},
			Type:      multiagent.MessageTypeRequest,
			Content:   scenario.message,
			Priority:  multiagent.PriorityMedium,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"demo":    true,
				"user_id": "demo_user",
			},
		}

		// Send message through orchestrator
		if err := orch.RouteMessage(ctx, testMessage); err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		// Give time for processing
		time.Sleep(200 * time.Millisecond)
		fmt.Printf("   ✅ Message sent to %s\n", scenario.targetAgent)
	}

	// Display system health
	fmt.Println("\n📊 System Health Status:")
	health := orch.GetSystemHealth()
	fmt.Printf("   Status: %s\n", health.Status)
	fmt.Printf("   Active Agents: %d/%d\n", health.ActiveAgents, health.TotalAgents)
	fmt.Printf("   Pending Tasks: %d\n", health.PendingTasks)
	fmt.Printf("   Message Queue: %d\n", health.MessageQueue)
	fmt.Printf("   Uptime: %s\n", health.Uptime)

	// List available agent capabilities
	fmt.Println("\n🛠️  Available Capabilities:")
	allAgents := orch.ListAgents()
	for _, agent := range allAgents {
		capabilities := agent.GetCapabilities()
		fmt.Printf("   %s (%s):\n", agent.Name(), agent.Type())
		for _, capability := range capabilities {
			fmt.Printf("     • %s\n", capability)
		}
		fmt.Println()
	}

	fmt.Println("🎉 Personal Assistant Multi-Agent System Demo Complete!")
	fmt.Println("\nYour personal assistant is now equipped with:")
	fmt.Println("📋 Project Management - Plan and track complex projects")
	fmt.Println("✅ Task Management - Personal productivity with GTD methodology")
	fmt.Println("🔍 Research Assistant - Information gathering and analysis")
	fmt.Println("📅 Scheduler - Calendar and appointment management")
	fmt.Println("📞 Communication Manager - Contact and message management")
	fmt.Println("💬 Conversation Agent - Natural language interface")
	fmt.Println("🎯 Coordinator - Multi-agent workflow orchestration")

	// Keep the system running for a bit to observe any async operations
	fmt.Println("\n⏱️  System running for 5 seconds to complete background operations...")
	time.Sleep(5 * time.Second)

	// Graceful shutdown
	fmt.Println("🔄 Shutting down agents...")
	for _, agent := range agents_list {
		if err := agent.Stop(ctx); err != nil {
			log.Printf("Warning: Failed to stop agent %s: %v", agent.ID(), err)
		}
	}

	fmt.Println("✅ Personal Assistant System Shutdown Complete")
}
