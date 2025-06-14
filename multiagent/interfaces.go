// Package multiagent provides a comprehensive multi-agent system for orchestrating
// specialized AI agents with persistent memory and coordinated workflows.
package multiagent

import (
	"context"
	"time"
)

// AgentID uniquely identifies an agent in the system
type AgentID string

// AgentType represents different types of specialized agents
type AgentType string

const (
	// Core agent types
	AgentTypeCoordinator  AgentType = "coordinator"   // Main orchestrator agent
	AgentTypeMemory       AgentType = "memory"        // Memory management specialist
	AgentTypeTask         AgentType = "task"          // Task management specialist
	AgentTypeResearch     AgentType = "research"      // Research and information gathering
	AgentTypeAnalyst      AgentType = "analyst"       // Data analysis and insights
	AgentTypeWriter       AgentType = "writer"        // Content creation and documentation
	AgentTypeCoder        AgentType = "coder"         // Code generation and review
	AgentTypeConversation AgentType = "conversation"  // Natural conversation handler
)

// Priority levels for agent messages and tasks
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Message represents communication between agents
type Message struct {
	ID          string                 `json:"id"`
	From        AgentID                `json:"from"`
	To          []AgentID              `json:"to"`          // Multiple recipients supported
	Type        MessageType            `json:"type"`
	Content     string                 `json:"content"`
	Context     map[string]interface{} `json:"context"`
	Priority    Priority               `json:"priority"`
	ReplyTo     string                 `json:"reply_to,omitempty"`    // Reference to parent message
	Timestamp   time.Time              `json:"timestamp"`
	RequiresACK bool                   `json:"requires_ack"`          // Whether acknowledgment is required
}

// MessageType defines different types of messages between agents
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"       // Request for action
	MessageTypeResponse     MessageType = "response"      // Response to request
	MessageTypeNotification MessageType = "notification"  // Information broadcast
	MessageTypeQuery        MessageType = "query"         // Information query
	MessageTypeCommand      MessageType = "command"       // Direct command
	MessageTypeReport       MessageType = "report"        // Status or result report
	MessageTypeError        MessageType = "error"         // Error notification
)

// AgentState represents the current state of an agent
type AgentState struct {
	Status        AgentStatus            `json:"status"`
	CurrentTask   string                 `json:"current_task,omitempty"`
	LastActivity  time.Time              `json:"last_activity"`
	Capabilities  []string               `json:"capabilities"`
	Workload      int                    `json:"workload"`      // 0-100 scale
	Metadata      map[string]interface{} `json:"metadata"`
}

// AgentStatus represents the operational status of an agent
type AgentStatus string

const (
	AgentStatusIdle     AgentStatus = "idle"
	AgentStatusBusy     AgentStatus = "busy"
	AgentStatusError    AgentStatus = "error"
	AgentStatusOffline  AgentStatus = "offline"
	AgentStatusStarting AgentStatus = "starting"
)

// Agent defines the interface that all agents must implement
type Agent interface {
	// Core identification
	ID() AgentID
	Type() AgentType
	Name() string
	Description() string

	// Lifecycle management
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetState() AgentState

	// Message handling
	SendMessage(ctx context.Context, msg *Message) error
	ReceiveMessage(ctx context.Context) (*Message, error)
	HandleMessage(ctx context.Context, msg *Message) (*Message, error)

	// Capabilities
	GetCapabilities() []string
	CanHandle(messageType MessageType) bool
}

// Tool defines the interface for tools that agents can use
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args string) (string, error)
}

// MemoryStore defines the interface for persistent memory storage
type MemoryStore interface {
	// Store operations
	Store(ctx context.Context, key string, value interface{}) error
	StoreWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Retrieval operations
	Get(ctx context.Context, key string) (interface{}, error)
	GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error)
	
	// Search operations
	Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
	SearchByTags(ctx context.Context, tags []string, limit int) ([]MemoryEntry, error)
	
	// Management operations
	Delete(ctx context.Context, key string) error
	Update(ctx context.Context, key string, updater func(interface{}) (interface{}, error)) error
	List(ctx context.Context, prefix string, limit int) ([]string, error)
	
	// Cleanup
	Cleanup(ctx context.Context) error
}

// MemoryEntry represents a single memory item
type MemoryEntry struct {
	Key          string                 `json:"key"`
	Value        interface{}            `json:"value"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	AccessedAt   time.Time              `json:"accessed_at"`
	AccessCount  int                    `json:"access_count"`
	TTL          *time.Duration         `json:"ttl,omitempty"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
}

// Orchestrator manages multiple agents and coordinates their activities
type Orchestrator interface {
	// Agent management
	RegisterAgent(agent Agent) error
	UnregisterAgent(agentID AgentID) error
	GetAgent(agentID AgentID) (Agent, error)
	ListAgents() []Agent
	
	// Message routing
	RouteMessage(ctx context.Context, msg *Message) error
	BroadcastMessage(ctx context.Context, msg *Message) error
	
	// Task coordination
	AssignTask(ctx context.Context, task Task) (AgentID, error)
	GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)
	
	// System management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetSystemHealth() SystemHealth
}

// Task represents a unit of work that can be assigned to agents
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Priority    Priority               `json:"priority"`
	Requester   AgentID                `json:"requester"`
	Assignee    AgentID                `json:"assignee,omitempty"`
	Status      TaskStatus             `json:"status"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// SystemHealth represents the overall health of the multi-agent system
type SystemHealth struct {
	Status        SystemStatus           `json:"status"`
	ActiveAgents  int                    `json:"active_agents"`
	TotalAgents   int                    `json:"total_agents"`
	PendingTasks  int                    `json:"pending_tasks"`
	ActiveTasks   int                    `json:"active_tasks"`
	MessageQueue  int                    `json:"message_queue"`
	MemoryUsage   float64                `json:"memory_usage_percent"`
	Uptime        time.Duration          `json:"uptime"`
	LastCheck     time.Time              `json:"last_check"`
	AgentHealth   map[AgentID]AgentState `json:"agent_health"`
}

// SystemStatus represents the overall system status
type SystemStatus string

const (
	SystemStatusHealthy  SystemStatus = "healthy"
	SystemStatusDegraded SystemStatus = "degraded"
	SystemStatusCritical SystemStatus = "critical"
	SystemStatusOffline  SystemStatus = "offline"
)

// EventType represents different types of system events
type EventType string

const (
	EventAgentRegistered   EventType = "agent_registered"
	EventAgentUnregistered EventType = "agent_unregistered"
	EventAgentStateChange  EventType = "agent_state_change"
	EventTaskCreated       EventType = "task_created"
	EventTaskAssigned      EventType = "task_assigned"
	EventTaskCompleted     EventType = "task_completed"
	EventTaskFailed        EventType = "task_failed"
	EventMessageSent       EventType = "message_sent"
	EventMessageReceived   EventType = "message_received"
	EventSystemError       EventType = "system_error"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// EventHandler processes system events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *Event) error
}

// LLMProvider defines the interface for language model providers
type LLMProvider interface {
	Name() string
	Query(ctx context.Context, prompt string) (string, error)
	QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error)
}

// ConversationContext maintains context for ongoing conversations
type ConversationContext struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	StartTime    time.Time              `json:"start_time"`
	LastActivity time.Time              `json:"last_activity"`
	Messages     []ConversationMessage  `json:"messages"`
	Context      map[string]interface{} `json:"context"`
	ActiveAgents []AgentID              `json:"active_agents"`
}

// ConversationMessage represents a single message in a conversation
type ConversationMessage struct {
	Role      string    `json:"role"`      // "user", "assistant", "system"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	AgentID   AgentID   `json:"agent_id,omitempty"`
}
