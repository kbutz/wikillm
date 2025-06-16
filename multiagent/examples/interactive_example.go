// This example demonstrates how to create an interactive command-line interface
// for the multiagent service with ALL specialist agents. It accepts user input 
// from the command line, intelligently routes to appropriate agents, and displays responses.
//
// To run this example:
//
//	cd multiagent/examples
//	go run interactive_example.go
//
// This example uses LMStudio integration for local LLM processing and includes
// all personal assistant specialist agents.
//
// Then type your messages and press Enter to interact with the agents.
// Type 'exit' to quit the application.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

// Import the DefaultOrchestrator type directly to avoid import issues
type DefaultOrchestrator = orchestrator.DefaultOrchestrator

// LMStudioProvider implements the LLMProvider interface for LMStudio
type LMStudioProvider struct {
	ServerURL   string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Debug       bool
}

// NewLMStudioProvider creates a new LMStudio provider
func NewLMStudioProvider(serverURL string, options ...func(*LMStudioProvider)) *LMStudioProvider {
	provider := &LMStudioProvider{
		ServerURL:   serverURL,
		Model:       "default", // LMStudio typically uses the loaded model
		MaxTokens:   2048,      // Increased for more comprehensive responses
		Temperature: 0.7,
		Debug:       false,
	}

	// Apply options
	for _, option := range options {
		option(provider)
	}

	return provider
}

// WithAPIKey sets the API key for the provider
func WithAPIKey(apiKey string) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.APIKey = apiKey
	}
}

// WithModel sets the model for the provider
func WithModel(model string) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.Model = model
	}
}

// WithMaxTokens sets the max tokens for the provider
func WithMaxTokens(maxTokens int) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for the provider
func WithTemperature(temperature float64) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.Temperature = temperature
	}
}

// WithDebug enables or disables debug mode
func WithDebug(debug bool) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.Debug = debug
	}
}

// Name returns the name of the provider
func (p *LMStudioProvider) Name() string {
	return "lmstudio"
}

// Query sends a prompt to the LMStudio server and returns the response
func (p *LMStudioProvider) Query(ctx context.Context, prompt string) (string, error) {
	// Create request payload
	payload := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"model":       p.Model,
		"temperature": p.Temperature,
		"max_tokens":  p.MaxTokens,
		"stream":      false,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Print request payload in debug mode
	if p.Debug {
		log.Printf("Request payload: %s", string(jsonData))
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.ServerURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	// Send request
	if p.Debug {
		log.Printf("Sending request to LMStudio at %s", p.ServerURL+"/chat/completions")
	}
	client := &http.Client{
		Timeout: 300 * time.Second, // Increased timeout for model loading/processing
	}
	resp, err := client.Do(req)
	if err != nil {
		if p.Debug {
			log.Printf("Error sending request to LMStudio: %v", err)
		}
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if p.Debug {
			log.Printf("Error reading response from LMStudio: %v", err)
		}
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Print response body in debug mode
	if p.Debug {
		log.Printf("Response body: %s", string(body))
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		if p.Debug {
			log.Printf("LMStudio returned error status %d: %s", resp.StatusCode, body)
		}
		return "", fmt.Errorf("server returned error (status %d): %s", resp.StatusCode, body)
	}

	if p.Debug {
		log.Printf("Received response from LMStudio (status %d)", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		if p.Debug {
			log.Printf("Error parsing JSON response: %v. Response body: %s", err, body)
		}
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		if p.Debug {
			log.Printf("Invalid response format - missing or empty 'choices' array: %+v", result)
		}
		return "", fmt.Errorf("invalid response format - missing or empty 'choices' array")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		if p.Debug {
			log.Printf("Invalid choice format - expected map: %+v", choices[0])
		}
		return "", fmt.Errorf("invalid choice format - expected map")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		if p.Debug {
			log.Printf("Invalid message format - expected map: %+v", choice)
		}
		return "", fmt.Errorf("invalid message format - expected map")
	}

	content, ok := message["content"].(string)
	if !ok {
		if p.Debug {
			log.Printf("Invalid content format - expected string: %+v", message)
		}
		return "", fmt.Errorf("invalid content format - expected string")
	}

	if p.Debug {
		log.Printf("Successfully extracted content from LMStudio response")
	}

	return content, nil
}

// QueryWithTools sends a prompt with tools to the LMStudio server
func (p *LMStudioProvider) QueryWithTools(ctx context.Context, prompt string, tools []multiagent.Tool) (string, error) {
	// For LMStudio, we'll use a simplified approach since it may not support OpenAI-style function calling
	// We'll include tool descriptions in the prompt

	var toolsPrompt string
	if len(tools) > 0 {
		toolsPrompt = "\n\nYou have access to the following tools:\n"
		for _, tool := range tools {
			toolsPrompt += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
		}
		toolsPrompt += "\nTo use a tool, respond with: [TOOL] tool_name {\"param1\": \"value1\", ...} [/TOOL]"
	}

	// Combine prompt with tools description
	fullPrompt := prompt + toolsPrompt

	// Send the query
	response, err := p.Query(ctx, fullPrompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// Enhanced MultiAgentService with all specialist agents
type MultiAgentService struct {
	memoryStore     multiagent.MemoryStore
	orchestrator    multiagent.Orchestrator
	agents          map[multiagent.AgentID]multiagent.Agent
	tools           map[string]multiagent.Tool
	llmProvider     multiagent.LLMProvider
	baseDir         string
	pendingRequests map[string]chan string
	requestsMutex   sync.RWMutex
}

// ServiceConfig holds configuration for creating a MultiAgentService
type ServiceConfig struct {
	BaseDir     string
	LLMProvider multiagent.LLMProvider
}

// NewMultiAgentService creates a new multi-agent service with all specialist agents
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

	// Initialize all agents (including new specialist agents)
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
	// Use a consistent conversation ID based on user ID
	conversationID := fmt.Sprintf("conv_%s", userID)
	log.Printf("Service: Using consistent conversation ID: %s", conversationID)

	// Create a response channel to capture the agent's response
	responseChannel := make(chan string, 1)
	responseKey := fmt.Sprintf("user_response_%s_%d", userID, time.Now().UnixNano())
	
	// Create a handler function that sends to our response channel
	handler := func(response string) {
		select {
		case responseChannel <- response:
		default:
			// Channel full, ignore
		}
	}
	
	// Register the handler with the orchestrator
	if orch, ok := s.orchestrator.(*DefaultOrchestrator); ok {
		orch.RegisterUserResponseHandler(responseKey, handler)
		defer orch.UnregisterUserResponseHandler(responseKey)
	} else {
		return "", fmt.Errorf("orchestrator does not support user response handlers")
	}

	// Create a message for the conversation agent (which will route to specialists as needed)
	msg := &multiagent.Message{
		ID:        fmt.Sprintf("msg_user_%d", time.Now().UnixNano()),
		From:      multiagent.AgentID(responseKey), // Use response key as sender so responses can be routed back
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

	// Route the message through the orchestrator
	if err := s.orchestrator.RouteMessage(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to route message: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChannel:
		return response, nil
	case <-time.After(60 * time.Second): // Increased timeout for complex agent processing
		return "I'm still processing your request. The specialist agents are working on it. Please check back in a moment.", nil
	case <-ctx.Done():
		return "", ctx.Err()
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

// GetSystemHealth returns the current health of the system
func (s *MultiAgentService) GetSystemHealth() multiagent.SystemHealth {
	return s.orchestrator.GetSystemHealth()
}

// ListAgents returns information about all available agents
func (s *MultiAgentService) ListAgents() map[string]AgentInfo {
	agentInfos := make(map[string]AgentInfo)
	
	for id, agent := range s.agents {
		state := agent.GetState()
		agentInfos[string(id)] = AgentInfo{
			ID:           string(id),
			Name:         agent.Name(),
			Type:         string(agent.Type()),
			Description:  agent.Description(),
			Status:       string(state.Status),
			Capabilities: agent.GetCapabilities(),
		}
	}
	
	return agentInfos
}

// AgentInfo contains information about an agent
type AgentInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Capabilities []string `json:"capabilities"`
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

func printWelcomeMessage() {
	fmt.Println("\n🤖 ===============================================")
	fmt.Println("   WikiLLM Personal Assistant Multi-Agent System")
	fmt.Println("   ===============================================")
	fmt.Println()
	fmt.Println("🎯 Your AI-powered personal assistant is ready with:")
	fmt.Println("   📋 Project Manager   - Project planning & tracking")
	fmt.Println("   ✅ Task Manager      - Personal productivity & GTD")
	fmt.Println("   🔍 Research Assistant - Information gathering & analysis") 
	fmt.Println("   📅 Scheduler         - Calendar & appointment management")
	fmt.Println("   📞 Communication Mgr - Contact & message management")
	fmt.Println("   💬 Conversation Agent - Natural language interface")
	fmt.Println("   🎯 Coordinator       - Multi-agent workflow management")
	fmt.Println()
	fmt.Println("💡 Example requests:")
	fmt.Println("   • \"Create a project for website redesign\"")
	fmt.Println("   • \"Add a task to review quarterly reports\"")
	fmt.Println("   • \"Research AI trends for 2024\"")
	fmt.Println("   • \"Schedule a meeting tomorrow at 2 PM\"")
	fmt.Println("   • \"Add John Smith as a client contact\"")
	fmt.Println("   • \"What's my schedule for this week?\"")
	fmt.Println()
	fmt.Println("🛠️  Special commands:")
	fmt.Println("   • 'agents' - List all available agents")
	fmt.Println("   • 'health' - Show system health status")
	fmt.Println("   • 'clear-memory' - Clear conversation history")
	fmt.Println("   • 'exit' - Quit the application")
	fmt.Println()
	fmt.Println("===============================================\n")
}

func main() {
	// Create memory directory within examples folder for easy access
	examplesDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}
	
	baseDir := filepath.Join(examplesDir, "wikillm_memory")
	
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}
	
	log.Printf("Using memory directory: %s", baseDir)

	// Test LMStudio connectivity first
	log.Println("🔌 Testing LMStudio connection...")
	llmProvider := NewLMStudioProvider("http://localhost:1234/v1",
		WithTemperature(0.7),
		WithMaxTokens(2048),
		WithDebug(false), // Reduced debug output for cleaner interaction
	)

	// Test a simple query to ensure LMStudio is working
	log.Println("⏳ First request may take longer if model is loading...")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	testResponse, err := llmProvider.Query(ctx, "Say hello in one word.")
	cancel()

	if err != nil {
		log.Printf("❌ LMStudio connection test failed: %v", err)
		log.Println("Please ensure:")
		log.Println("1. LMStudio is running")
		log.Println("2. A model is loaded")
		log.Println("3. The server is accessible at http://localhost:1234")
		log.Fatalf("Cannot proceed without LMStudio connection")
	}

	log.Printf("✅ LMStudio connection successful! Test response: %s", testResponse)

	// Create the multi-agent service with all specialist agents
	log.Println("🏗️  Creating personal assistant service with all specialist agents...")
	svc, err := NewMultiAgentService(ServiceConfig{
		BaseDir:     baseDir,
		LLMProvider: llmProvider,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-agent service: %v", err)
	}

	// Start the service
	ctx = context.Background()
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	// Defer cleanup
	defer func() {
		log.Println("🔄 Shutting down personal assistant...")
		if err := svc.Stop(ctx); err != nil {
			log.Printf("Warning: Failed to stop service cleanly: %v", err)
		}
	}()

	// Print welcome message
	printWelcomeMessage()

	// Generate a unique user ID
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create a scanner for user input
	scanner := bufio.NewScanner(os.Stdin)

	// Main interaction loop
	for {
		fmt.Print("🤖 > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Skip empty input
		if input == "" {
			continue
		}

		// Check for special commands
		switch strings.ToLower(input) {
		case "exit":
			fmt.Println("👋 Goodbye! Thanks for using the Personal Assistant!")
			return

		case "agents":
			fmt.Println("\n📋 Available Agents:")
			agentInfos := svc.ListAgents()
			for _, info := range agentInfos {
				fmt.Printf("   🤖 %s (%s)\n", info.Name, info.ID)
				fmt.Printf("      📝 %s\n", info.Description)
				fmt.Printf("      📊 Status: %s\n", info.Status)
				fmt.Printf("      🛠️  Capabilities: %s\n", strings.Join(info.Capabilities, ", "))
				fmt.Println()
			}
			continue

		case "health":
			fmt.Println("\n📊 System Health Status:")
			health := svc.GetSystemHealth()
			fmt.Printf("   🟢 Status: %s\n", health.Status)
			fmt.Printf("   🤖 Active Agents: %d/%d\n", health.ActiveAgents, health.TotalAgents)
			fmt.Printf("   📋 Pending Tasks: %d\n", health.PendingTasks)
			fmt.Printf("   📬 Message Queue: %d\n", health.MessageQueue)
			fmt.Printf("   ⏱️  Uptime: %s\n", health.Uptime)
			fmt.Printf("   💾 Memory Directory: %s\n", baseDir)
			fmt.Println()
			continue

		case "clear-memory":
			memoryDir := filepath.Join(baseDir, "memory")
			fmt.Printf("🧹 Clearing memory directory: %s\n", memoryDir)
			if err := os.RemoveAll(memoryDir); err != nil {
				fmt.Printf("❌ Error clearing memory: %v\n", err)
			} else {
				fmt.Println("✅ Memory cleared. You can start a fresh conversation.")
			}
			continue
		}

		// Show processing message
		fmt.Println("⏳ Processing your request with specialist agents...")

		// Create a timeout context for the request
		requestCtx, cancel := context.WithTimeout(ctx, 300*time.Second)

		// Process the user message
		response, err := svc.ProcessUserMessage(requestCtx, userID, input)
		cancel()

		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Print the response with nice formatting
		fmt.Printf("\n🤖 Personal Assistant: %s\n\n", response)
	}

	log.Println("✅ Interactive personal assistant session completed successfully")
}
