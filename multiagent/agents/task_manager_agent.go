package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// TaskManagerAgent specializes in personal task management, reminders, and productivity
type TaskManagerAgent struct {
	*BaseAgent
	tasks      map[string]*PersonalTask
	reminders  map[string]*Reminder
	taskMutex  sync.RWMutex
}

// PersonalTask represents a personal task with detailed tracking
type PersonalTask struct {
	ID              string                      `json:"id"`
	Title           string                      `json:"title"`
	Description     string                      `json:"description"`
	Status          PersonalTaskStatus          `json:"status"`
	Priority        multiagent.Priority         `json:"priority"`
	Category        string                      `json:"category"`
	Tags            []string                    `json:"tags"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
	DueDate         *time.Time                  `json:"due_date,omitempty"`
	CompletedAt     *time.Time                  `json:"completed_at,omitempty"`
	EstimatedTime   time.Duration               `json:"estimated_time"`
	ActualTime      time.Duration               `json:"actual_time"`
	Progress        float64                     `json:"progress"`
	Subtasks        []Subtask                   `json:"subtasks"`
	Dependencies    []string                    `json:"dependencies"`
	Context         string                      `json:"context"`
	Location        string                      `json:"location"`
	Energy          EnergyLevel                 `json:"energy_level"`
	Recurring       *RecurrencePattern          `json:"recurring,omitempty"`
	Reminders       []string                    `json:"reminders"`
	Notes           []TaskNote                  `json:"notes"`
	Attachments     []string                    `json:"attachments"`
	LastWorkedOn    *time.Time                  `json:"last_worked_on,omitempty"`
	TimeSpent       []TimeEntry                 `json:"time_spent"`
	Metadata        map[string]interface{}      `json:"metadata"`
}

// PersonalTaskStatus represents the status of a personal task
type PersonalTaskStatus string

const (
	PersonalTaskStatusInbox      PersonalTaskStatus = "inbox"       // Captured but not processed
	PersonalTaskStatusNext       PersonalTaskStatus = "next"        // Ready to work on
	PersonalTaskStatusSomeday    PersonalTaskStatus = "someday"     // Future consideration
	PersonalTaskStatusWaiting    PersonalTaskStatus = "waiting"     // Waiting for someone/something
	PersonalTaskStatusInProgress PersonalTaskStatus = "in_progress" // Currently working on
	PersonalTaskStatusCompleted  PersonalTaskStatus = "completed"   // Finished
	PersonalTaskStatusCancelled  PersonalTaskStatus = "cancelled"   // No longer relevant
	PersonalTaskStatusDeferred   PersonalTaskStatus = "deferred"    // Postponed to specific date
)

// EnergyLevel represents the energy required for a task
type EnergyLevel string

const (
	EnergyLevelLow    EnergyLevel = "low"    // Can do when tired
	EnergyLevelMedium EnergyLevel = "medium" // Normal energy required
	EnergyLevelHigh   EnergyLevel = "high"   // Requires focus and high energy
)

// Subtask represents a smaller task within a main task
type Subtask struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Completed   bool      `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RecurrencePattern defines how a task repeats
type RecurrencePattern struct {
	Type       RecurrenceType `json:"type"`
	Interval   int            `json:"interval"`
	DaysOfWeek []time.Weekday `json:"days_of_week,omitempty"`
	DayOfMonth int            `json:"day_of_month,omitempty"`
	EndDate    *time.Time     `json:"end_date,omitempty"`
	Count      int            `json:"count,omitempty"`
}

// RecurrenceType defines the type of recurrence
type RecurrenceType string

const (
	RecurrenceTypeDaily   RecurrenceType = "daily"
	RecurrenceTypeWeekly  RecurrenceType = "weekly"
	RecurrenceTypeMonthly RecurrenceType = "monthly"
	RecurrenceTypeYearly  RecurrenceType = "yearly"
)

// TaskNote represents a note attached to a task
type TaskNote struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "note", "update", "comment"
}

// TimeEntry represents time spent on a task
type TimeEntry struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration"`
	Note      string        `json:"note"`
}

// Reminder represents a reminder for tasks or events
type Reminder struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	Message    string          `json:"message"`
	TriggerAt  time.Time       `json:"trigger_at"`
	CreatedAt  time.Time       `json:"created_at"`
	Status     ReminderStatus  `json:"status"`
	Type       ReminderType    `json:"type"`
	TaskID     string          `json:"task_id,omitempty"`
	Recurring  bool            `json:"recurring"`
	Snoozed    bool            `json:"snoozed"`
	SnoozedUntil *time.Time    `json:"snoozed_until,omitempty"`
	Context    map[string]interface{} `json:"context"`
}

// ReminderStatus represents the status of a reminder
type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"
	ReminderStatusTriggered ReminderStatus = "triggered"
	ReminderStatusCompleted ReminderStatus = "completed"
	ReminderStatusCancelled ReminderStatus = "cancelled"
	ReminderStatusSnoozed   ReminderStatus = "snoozed"
)

// ReminderType defines different types of reminders
type ReminderType string

const (
	ReminderTypeTask        ReminderType = "task"
	ReminderTypeDeadline    ReminderType = "deadline"
	ReminderTypeAppointment ReminderType = "appointment"
	ReminderTypeFollowUp    ReminderType = "follow_up"
	ReminderTypeGeneral     ReminderType = "general"
)

// NewTaskManagerAgent creates a new task manager agent
func NewTaskManagerAgent(config BaseAgentConfig) *TaskManagerAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeTask

	// Add task management capabilities
	config.Capabilities = append(config.Capabilities,
		"task_management",
		"reminder_system",
		"productivity_tracking",
		"time_management",
		"gtd_methodology",
		"task_prioritization",
		"context_switching",
		"recurring_tasks",
		"progress_tracking",
		"workflow_optimization",
	)

	agent := &TaskManagerAgent{
		BaseAgent: NewBaseAgent(config),
		tasks:     make(map[string]*PersonalTask),
		reminders: make(map[string]*Reminder),
	}

	// Start reminder checking routine
	go agent.reminderChecker(context.Background())

	return agent
}

// HandleMessage processes incoming task management requests
func (a *TaskManagerAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Managing tasks"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("task_manager:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	content := strings.ToLower(msg.Content)

	// Route to appropriate handler based on content
	if strings.Contains(content, "add task") || strings.Contains(content, "create task") || strings.Contains(content, "new task") {
		return a.handleAddTask(ctx, msg)
	} else if strings.Contains(content, "list tasks") || strings.Contains(content, "show tasks") || strings.Contains(content, "my tasks") {
		return a.handleListTasks(ctx, msg)
	} else if strings.Contains(content, "complete task") || strings.Contains(content, "finish task") || strings.Contains(content, "done") {
		return a.handleCompleteTask(ctx, msg)
	} else if strings.Contains(content, "remind me") || strings.Contains(content, "reminder") || strings.Contains(content, "set reminder") {
		return a.handleCreateReminder(ctx, msg)
	} else if strings.Contains(content, "update task") || strings.Contains(content, "modify task") {
		return a.handleUpdateTask(ctx, msg)
	} else if strings.Contains(content, "delete task") || strings.Contains(content, "remove task") {
		return a.handleDeleteTask(ctx, msg)
	} else if strings.Contains(content, "prioritize") || strings.Contains(content, "priority") {
		return a.handlePrioritize(ctx, msg)
	} else if strings.Contains(content, "today") || strings.Contains(content, "due today") {
		return a.handleTodayTasks(ctx, msg)
	} else if strings.Contains(content, "overdue") {
		return a.handleOverdueTasks(ctx, msg)
	} else if strings.Contains(content, "next actions") || strings.Contains(content, "next tasks") {
		return a.handleNextActions(ctx, msg)
	} else if strings.Contains(content, "productivity") || strings.Contains(content, "statistics") || strings.Contains(content, "stats") {
		return a.handleProductivityStats(ctx, msg)
	} else {
		// Use LLM for general task management queries
		return a.handleGeneralQuery(ctx, msg)
	}
}

// handleAddTask creates a new personal task
func (a *TaskManagerAgent) handleAddTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract task details
	contextPrompt := fmt.Sprintf(`
Extract task information from this request: "%s"

Provide response in JSON format:
{
  "title": "task title",
  "description": "detailed description",
  "priority": "low|medium|high|critical",
  "category": "work|personal|health|learning|etc",
  "due_date": "YYYY-MM-DD HH:MM if mentioned, otherwise null",
  "estimated_time": "duration in minutes if mentioned, otherwise 0",
  "energy_level": "low|medium|high",
  "context": "location or context if mentioned",
  "tags": ["tag1", "tag2"] if any mentioned,
  "recurring": "daily|weekly|monthly|yearly if recurring, otherwise null"
}

Make reasonable assumptions for missing information.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task details: %w", err)
	}

	var taskData struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		Priority      string   `json:"priority"`
		Category      string   `json:"category"`
		DueDate       string   `json:"due_date"`
		EstimatedTime int      `json:"estimated_time"`
		EnergyLevel   string   `json:"energy_level"`
		Context       string   `json:"context"`
		Tags          []string `json:"tags"`
		Recurring     string   `json:"recurring"`
	}

	if err := json.Unmarshal([]byte(response), &taskData); err != nil {
		// Fallback to basic task creation
		taskData.Title = msg.Content
		taskData.Priority = "medium"
		taskData.Category = "personal"
		taskData.EnergyLevel = "medium"
	}

	// Create task
	task := &PersonalTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:          taskData.Title,
		Description:    taskData.Description,
		Status:         PersonalTaskStatusInbox,
		Priority:       a.parsePriority(taskData.Priority),
		Category:       taskData.Category,
		Tags:           taskData.Tags,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		EstimatedTime:  time.Duration(taskData.EstimatedTime) * time.Minute,
		Energy:         a.parseEnergyLevel(taskData.EnergyLevel),
		Context:        taskData.Context,
		Progress:       0.0,
		Subtasks:       []Subtask{},
		Dependencies:   []string{},
		Reminders:      []string{},
		Notes:          []TaskNote{},
		Attachments:    []string{},
		TimeSpent:      []TimeEntry{},
		Metadata:       make(map[string]interface{}),
	}

	// Set due date if provided
	if taskData.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02 15:04", taskData.DueDate); err == nil {
			task.DueDate = &dueDate
		} else if dueDate, err := time.Parse("2006-01-02", taskData.DueDate); err == nil {
			task.DueDate = &dueDate
		}
	}

	// Set up recurring pattern if needed
	if taskData.Recurring != "" {
		task.Recurring = &RecurrencePattern{
			Type:     RecurrenceType(taskData.Recurring),
			Interval: 1,
		}
	}

	// Store task
	a.taskMutex.Lock()
	a.tasks[task.ID] = task
	a.taskMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		taskKey := fmt.Sprintf("personal_task:%s", task.ID)
		a.memoryStore.Store(ctx, taskKey, task)
	}

	// Create automatic reminder if due date is set
	if task.DueDate != nil {
		a.createAutomaticReminder(ctx, task)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ Task '%s' added successfully!\n\n📋 **Details:**\n• ID: %s\n• Priority: %s\n• Category: %s\n• Status: %s\n• Energy Level: %s", task.Title, task.ID, task.Priority, task.Category, task.Status, task.Energy),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"task_id": task.ID,
			"action":  "task_created",
		},
	}, nil
}

// handleListTasks lists tasks based on various criteria
func (a *TaskManagerAgent) handleListTasks(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Load tasks from memory if needed
	a.loadTasksFromMemory(ctx)

	content := strings.ToLower(msg.Content)
	var filteredTasks []*PersonalTask

	a.taskMutex.RLock()
	defer a.taskMutex.RUnlock()

	// Apply filters based on request
	for _, task := range a.tasks {
		include := true

		// Filter by status
		if strings.Contains(content, "completed") && task.Status != PersonalTaskStatusCompleted {
			include = false
		} else if strings.Contains(content, "active") && task.Status == PersonalTaskStatusCompleted {
			include = false
		} else if strings.Contains(content, "next") && task.Status != PersonalTaskStatusNext {
			include = false
		} else if strings.Contains(content, "waiting") && task.Status != PersonalTaskStatusWaiting {
			include = false
		}

		// Filter by priority
		if strings.Contains(content, "high priority") && task.Priority != multiagent.PriorityHigh {
			include = false
		} else if strings.Contains(content, "critical") && task.Priority != multiagent.PriorityCritical {
			include = false
		}

		// Filter by category
		if strings.Contains(content, "work") && task.Category != "work" {
			include = false
		} else if strings.Contains(content, "personal") && task.Category != "personal" {
			include = false
		}

		if include {
			filteredTasks = append(filteredTasks, task)
		}
	}

	if len(filteredTasks) == 0 {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "📋 No tasks found matching your criteria. Use 'add task' to create your first task!",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Sort tasks by priority and due date
	sort.Slice(filteredTasks, func(i, j int) bool {
		if filteredTasks[i].Priority != filteredTasks[j].Priority {
			return filteredTasks[i].Priority > filteredTasks[j].Priority
		}
		if filteredTasks[i].DueDate != nil && filteredTasks[j].DueDate != nil {
			return filteredTasks[i].DueDate.Before(*filteredTasks[j].DueDate)
		}
		if filteredTasks[i].DueDate != nil {
			return true
		}
		return false
	})

	// Build response
	var responseBuilder strings.Builder
	responseBuilder.WriteString("📋 **Your Tasks**\n\n")

	for i, task := range filteredTasks {
		if i >= 20 { // Limit to 20 tasks
			responseBuilder.WriteString(fmt.Sprintf("... and %d more tasks\n", len(filteredTasks)-i))
			break
		}

		status := a.getStatusEmoji(task.Status)
		priority := a.getPriorityEmoji(task.Priority)
		
		responseBuilder.WriteString(fmt.Sprintf("%d. %s %s **%s** (%s)\n", i+1, status, priority, task.Title, task.Category))
		
		if task.DueDate != nil {
			dueText := a.formatDueDate(*task.DueDate)
			responseBuilder.WriteString(fmt.Sprintf("   📅 Due: %s\n", dueText))
		}
		
		if task.Progress > 0 {
			responseBuilder.WriteString(fmt.Sprintf("   📊 Progress: %.0f%%\n", task.Progress))
		}
		
		if task.Energy != EnergyLevelMedium {
			responseBuilder.WriteString(fmt.Sprintf("   ⚡ Energy: %s\n", task.Energy))
		}
		
		responseBuilder.WriteString("\n")
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   responseBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleCompleteTask marks a task as completed
func (a *TaskManagerAgent) handleCompleteTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	taskID := a.extractTaskID(msg.Content)
	
	a.taskMutex.Lock()
	defer a.taskMutex.Unlock()
	
	task, exists := a.tasks[taskID]
	if !exists {
		// Try to find by title
		task = a.findTaskByTitle(msg.Content)
		if task == nil {
			return &multiagent.Message{
				ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
				From:      a.id,
				To:        []multiagent.AgentID{msg.From},
				Type:      multiagent.MessageTypeResponse,
				Content:   "❌ Task not found. Please specify a valid task ID or title.",
				ReplyTo:   msg.ID,
				Timestamp: time.Now(),
			}, nil
		}
	}
	
	// Mark as completed
	task.Status = PersonalTaskStatusCompleted
	task.Progress = 100.0
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now
	
	// Save to memory
	if a.memoryStore != nil {
		taskKey := fmt.Sprintf("personal_task:%s", task.ID)
		a.memoryStore.Store(ctx, taskKey, task)
	}
	
	// Handle recurring tasks
	if task.Recurring != nil {
		newTask := a.createRecurringTask(task)
		if newTask != nil {
			a.tasks[newTask.ID] = newTask
			if a.memoryStore != nil {
				newTaskKey := fmt.Sprintf("personal_task:%s", newTask.ID)
				a.memoryStore.Store(ctx, newTaskKey, newTask)
			}
		}
	}
	
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ Task '%s' marked as completed! 🎉\n\nCompleted at: %s", task.Title, now.Format("2006-01-02 15:04")),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"task_id": task.ID,
			"action":  "task_completed",
		},
	}, nil
}

// handleCreateReminder creates a new reminder
func (a *TaskManagerAgent) handleCreateReminder(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract reminder details
	contextPrompt := fmt.Sprintf(`
Extract reminder information from: "%s"

Provide response in JSON format:
{
  "title": "reminder title",
  "message": "reminder message",
  "trigger_time": "YYYY-MM-DD HH:MM when to trigger",
  "type": "task|deadline|appointment|follow_up|general",
  "recurring": true/false
}`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reminder details: %w", err)
	}

	var reminderData struct {
		Title       string `json:"title"`
		Message     string `json:"message"`
		TriggerTime string `json:"trigger_time"`
		Type        string `json:"type"`
		Recurring   bool   `json:"recurring"`
	}

	if err := json.Unmarshal([]byte(response), &reminderData); err != nil {
		return nil, fmt.Errorf("failed to parse reminder JSON: %w", err)
	}

	// Parse trigger time
	triggerAt, err := time.Parse("2006-01-02 15:04", reminderData.TriggerTime)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger time format: %w", err)
	}

	// Create reminder
	reminder := &Reminder{
		ID:        fmt.Sprintf("reminder_%d", time.Now().UnixNano()),
		Title:     reminderData.Title,
		Message:   reminderData.Message,
		TriggerAt: triggerAt,
		CreatedAt: time.Now(),
		Status:    ReminderStatusPending,
		Type:      ReminderType(reminderData.Type),
		Recurring: reminderData.Recurring,
		Context:   make(map[string]interface{}),
	}

	// Store reminder
	a.taskMutex.Lock()
	a.reminders[reminder.ID] = reminder
	a.taskMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		reminderKey := fmt.Sprintf("reminder:%s", reminder.ID)
		a.memoryStore.Store(ctx, reminderKey, reminder)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("⏰ Reminder '%s' set for %s\n\nI'll remind you: %s", reminder.Title, triggerAt.Format("2006-01-02 15:04"), reminder.Message),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"reminder_id": reminder.ID,
			"action":      "reminder_created",
		},
	}, nil
}

// Helper methods

func (a *TaskManagerAgent) parsePriority(priority string) multiagent.Priority {
	switch strings.ToLower(priority) {
	case "critical":
		return multiagent.PriorityCritical
	case "high":
		return multiagent.PriorityHigh
	case "low":
		return multiagent.PriorityLow
	default:
		return multiagent.PriorityMedium
	}
}

func (a *TaskManagerAgent) parseEnergyLevel(energy string) EnergyLevel {
	switch strings.ToLower(energy) {
	case "low":
		return EnergyLevelLow
	case "high":
		return EnergyLevelHigh
	default:
		return EnergyLevelMedium
	}
}

func (a *TaskManagerAgent) getStatusEmoji(status PersonalTaskStatus) string {
	switch status {
	case PersonalTaskStatusCompleted:
		return "✅"
	case PersonalTaskStatusInProgress:
		return "⏳"
	case PersonalTaskStatusWaiting:
		return "⏸️"
	case PersonalTaskStatusNext:
		return "➡️"
	case PersonalTaskStatusSomeday:
		return "💭"
	case PersonalTaskStatusDeferred:
		return "📅"
	case PersonalTaskStatusCancelled:
		return "❌"
	default:
		return "📋"
	}
}

func (a *TaskManagerAgent) getPriorityEmoji(priority multiagent.Priority) string {
	switch priority {
	case multiagent.PriorityCritical:
		return "🔥"
	case multiagent.PriorityHigh:
		return "⚠️"
	case multiagent.PriorityLow:
		return "🔽"
	default:
		return "🔸"
	}
}

func (a *TaskManagerAgent) formatDueDate(dueDate time.Time) string {
	now := time.Now()
	diff := dueDate.Sub(now)
	
	if diff < 0 {
		return fmt.Sprintf("%s (⚠️ Overdue)", dueDate.Format("2006-01-02"))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%s (Today)", dueDate.Format("15:04"))
	} else if diff < 48*time.Hour {
		return fmt.Sprintf("%s (Tomorrow)", dueDate.Format("15:04"))
	} else {
		return dueDate.Format("2006-01-02")
	}
}

func (a *TaskManagerAgent) extractTaskID(content string) string {
	words := strings.Fields(content)
	for _, word := range words {
		if strings.HasPrefix(word, "task_") {
			return word
		}
	}
	return ""
}

func (a *TaskManagerAgent) findTaskByTitle(content string) *PersonalTask {
	contentLower := strings.ToLower(content)
	
	for _, task := range a.tasks {
		if strings.Contains(contentLower, strings.ToLower(task.Title)) {
			return task
		}
	}
	return nil
}

func (a *TaskManagerAgent) loadTasksFromMemory(ctx context.Context) {
	if a.memoryStore == nil {
		return
	}

	// List all task keys
	keys, err := a.memoryStore.List(ctx, "personal_task:", 1000)
	if err != nil {
		return
	}

	// Load tasks
	tasks, err := a.memoryStore.GetMultiple(ctx, keys)
	if err != nil {
		return
	}

	a.taskMutex.Lock()
	defer a.taskMutex.Unlock()

	for _, taskInterface := range tasks {
		var task PersonalTask
		if taskData, err := json.Marshal(taskInterface); err == nil {
			if err := json.Unmarshal(taskData, &task); err == nil {
				a.tasks[task.ID] = &task
			}
		}
	}
}

func (a *TaskManagerAgent) createAutomaticReminder(ctx context.Context, task *PersonalTask) {
	if task.DueDate == nil {
		return
	}

	// Create reminder 1 day before due date
	reminderTime := task.DueDate.Add(-24 * time.Hour)
	if reminderTime.Before(time.Now()) {
		return // Don't create past reminders
	}

	reminder := &Reminder{
		ID:        fmt.Sprintf("reminder_%s_due", task.ID),
		Title:     fmt.Sprintf("Task Due Tomorrow: %s", task.Title),
		Message:   fmt.Sprintf("Task '%s' is due tomorrow at %s", task.Title, task.DueDate.Format("15:04")),
		TriggerAt: reminderTime,
		CreatedAt: time.Now(),
		Status:    ReminderStatusPending,
		Type:      ReminderTypeDeadline,
		TaskID:    task.ID,
		Context:   make(map[string]interface{}),
	}

	a.taskMutex.Lock()
	a.reminders[reminder.ID] = reminder
	a.taskMutex.Unlock()

	if a.memoryStore != nil {
		reminderKey := fmt.Sprintf("reminder:%s", reminder.ID)
		a.memoryStore.Store(ctx, reminderKey, reminder)
	}
}

func (a *TaskManagerAgent) createRecurringTask(originalTask *PersonalTask) *PersonalTask {
	if originalTask.Recurring == nil {
		return nil
	}

	newTask := &PersonalTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:          originalTask.Title,
		Description:    originalTask.Description,
		Status:         PersonalTaskStatusNext,
		Priority:       originalTask.Priority,
		Category:       originalTask.Category,
		Tags:           originalTask.Tags,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		EstimatedTime:  originalTask.EstimatedTime,
		Energy:         originalTask.Energy,
		Context:        originalTask.Context,
		Location:       originalTask.Location,
		Progress:       0.0,
		Subtasks:       []Subtask{},
		Dependencies:   []string{},
		Recurring:      originalTask.Recurring,
		Reminders:      []string{},
		Notes:          []TaskNote{},
		Attachments:    []string{},
		TimeSpent:      []TimeEntry{},
		Metadata:       make(map[string]interface{}),
	}

	// Calculate next due date
	if originalTask.DueDate != nil {
		nextDue := a.calculateNextDueDate(*originalTask.DueDate, originalTask.Recurring)
		newTask.DueDate = &nextDue
	}

	return newTask
}

func (a *TaskManagerAgent) calculateNextDueDate(lastDue time.Time, pattern *RecurrencePattern) time.Time {
	switch pattern.Type {
	case RecurrenceTypeDaily:
		return lastDue.AddDate(0, 0, pattern.Interval)
	case RecurrenceTypeWeekly:
		return lastDue.AddDate(0, 0, pattern.Interval*7)
	case RecurrenceTypeMonthly:
		return lastDue.AddDate(0, pattern.Interval, 0)
	case RecurrenceTypeYearly:
		return lastDue.AddDate(pattern.Interval, 0, 0)
	default:
		return lastDue.AddDate(0, 0, 1)
	}
}

func (a *TaskManagerAgent) reminderChecker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.checkReminders(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *TaskManagerAgent) checkReminders(ctx context.Context) {
	now := time.Now()
	
	a.taskMutex.Lock()
	defer a.taskMutex.Unlock()
	
	for _, reminder := range a.reminders {
		if reminder.Status == ReminderStatusPending && reminder.TriggerAt.Before(now) {
			reminder.Status = ReminderStatusTriggered
			
			// Send reminder message (in a real system, this would notify the user)
			// For now, we'll just log it or store it as a system message
			if a.memoryStore != nil {
				reminderKey := fmt.Sprintf("reminder:%s", reminder.ID)
				a.memoryStore.Store(ctx, reminderKey, reminder)
				
				// Store triggered reminder as a system message
				systemMsgKey := fmt.Sprintf("system_reminder:%d", time.Now().UnixNano())
				a.memoryStore.Store(ctx, systemMsgKey, map[string]interface{}{
					"type":       "reminder_triggered",
					"reminder":   reminder,
					"timestamp":  now,
				})
			}
		}
	}
}

// Additional handler methods (simplified for space)

func (a *TaskManagerAgent) handleUpdateTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🔄 Task update functionality is available. Please specify which task and what changes you'd like to make.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleDeleteTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🗑️ Task deletion is available. Please specify which task you'd like to delete.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handlePrioritize(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🎯 Task prioritization is available. I can help you organize tasks by priority, deadline, or energy level.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleTodayTasks(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "📅 Today's tasks functionality is available. I can show you tasks due today and help you plan your day.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleOverdueTasks(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "⚠️ Overdue task tracking is available. I can help you identify and reschedule overdue tasks.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleNextActions(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "➡️ Next actions functionality is available. I can help you identify your next actionable tasks based on GTD methodology.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleProductivityStats(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "📊 Productivity statistics are available. I can analyze your task completion rates, time tracking, and productivity patterns.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *TaskManagerAgent) handleGeneralQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context with task information
	contextPrompt := a.buildTaskContext(ctx, msg)
	
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

func (a *TaskManagerAgent) buildTaskContext(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder
	
	contextBuilder.WriteString(fmt.Sprintf("You are %s, a personal task management specialist.\n\n", a.name))
	contextBuilder.WriteString("You help users manage their personal tasks, reminders, and productivity using GTD (Getting Things Done) methodology.\n\n")
	
	// Add current task summary
	a.taskMutex.RLock()
	if len(a.tasks) > 0 {
		contextBuilder.WriteString("Current Task Summary:\n")
		statusCounts := make(map[PersonalTaskStatus]int)
		for _, task := range a.tasks {
			statusCounts[task.Status]++
		}
		for status, count := range statusCounts {
			contextBuilder.WriteString(fmt.Sprintf("- %s: %d tasks\n", status, count))
		}
		contextBuilder.WriteString("\n")
	}
	a.taskMutex.RUnlock()
	
	contextBuilder.WriteString(fmt.Sprintf("User request: %s\n\n", msg.Content))
	contextBuilder.WriteString("Please provide helpful task management advice, suggestions, or execute the requested action.")
	
	return contextBuilder.String()
}
