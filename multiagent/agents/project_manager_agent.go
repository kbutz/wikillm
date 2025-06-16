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

// ProjectManagerAgent specializes in project planning, tracking, and management
type ProjectManagerAgent struct {
	*BaseAgent
	activeProjects map[string]*Project
	projectMutex   sync.RWMutex
}

// Project represents a managed project with tasks, milestones, and tracking
type Project struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Status         ProjectStatus          `json:"status"`
	Priority       multiagent.Priority    `json:"priority"`
	Owner          string                 `json:"owner"`
	CreatedAt      time.Time              `json:"created_at"`
	StartDate      *time.Time             `json:"start_date,omitempty"`
	DueDate        *time.Time             `json:"due_date,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Tasks          []ProjectTask          `json:"tasks"`
	Milestones     []Milestone            `json:"milestones"`
	Resources      []Resource             `json:"resources"`
	Dependencies   []string               `json:"dependencies"`
	Progress       float64                `json:"progress"`
	EstimatedHours float64                `json:"estimated_hours"`
	ActualHours    float64                `json:"actual_hours"`
	Budget         *Budget                `json:"budget,omitempty"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ProjectTask represents a task within a project
type ProjectTask struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Status         TaskStatus          `json:"status"`
	Priority       multiagent.Priority `json:"priority"`
	Assignee       string              `json:"assignee"`
	CreatedAt      time.Time           `json:"created_at"`
	StartDate      *time.Time          `json:"start_date,omitempty"`
	DueDate        *time.Time          `json:"due_date,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	Dependencies   []string            `json:"dependencies"`
	Progress       float64             `json:"progress"`
	EstimatedHours float64             `json:"estimated_hours"`
	ActualHours    float64             `json:"actual_hours"`
	Tags           []string            `json:"tags"`
	Comments       []TaskComment       `json:"comments"`
}

// TaskStatus represents the status of a project task
type TaskStatus string

const (
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusOnHold     TaskStatus = "on_hold"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusPlanning  ProjectStatus = "planning"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusOnHold    ProjectStatus = "on_hold"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
	ProjectStatusArchived  ProjectStatus = "archived"
)

// Milestone represents a significant checkpoint in a project
type Milestone struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     time.Time  `json:"due_date"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string     `json:"status"`
	Tasks       []string   `json:"tasks"`
}

// Resource represents a resource needed for a project
type Resource struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	CostPerUnit  float64 `json:"cost_per_unit"`
	TotalCost    float64 `json:"total_cost"`
	Availability string  `json:"availability"`
}

// Budget represents project budget information
type Budget struct {
	TotalBudget     float64            `json:"total_budget"`
	SpentAmount     float64            `json:"spent_amount"`
	RemainingBudget float64            `json:"remaining_budget"`
	Categories      map[string]float64 `json:"categories"`
	Currency        string             `json:"currency"`
}

// TaskComment represents a comment on a task
type TaskComment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// NewProjectManagerAgent creates a new project manager agent
func NewProjectManagerAgent(config BaseAgentConfig) *ProjectManagerAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeProjectManager

	// Add project management capabilities
	config.Capabilities = append(config.Capabilities,
		"project_planning",
		"task_management",
		"milestone_tracking",
		"resource_allocation",
		"progress_monitoring",
		"budget_tracking",
		"timeline_management",
		"dependency_analysis",
		"status_reporting",
		"project_coordination",
	)

	return &ProjectManagerAgent{
		BaseAgent:      NewBaseAgent(config),
		activeProjects: make(map[string]*Project),
	}
}

// HandleMessage processes incoming project management requests
func (a *ProjectManagerAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Managing projects"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("project_manager:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	// Process based on message content
	content := strings.ToLower(msg.Content)

	if strings.Contains(content, "create project") || strings.Contains(content, "new project") {
		return a.handleCreateProject(ctx, msg)
	} else if strings.Contains(content, "list projects") || strings.Contains(content, "show projects") {
		return a.handleListProjects(ctx, msg)
	} else if strings.Contains(content, "project status") || strings.Contains(content, "project progress") {
		return a.handleProjectStatus(ctx, msg)
	} else if strings.Contains(content, "add task") || strings.Contains(content, "create task") {
		return a.handleAddTask(ctx, msg)
	} else if strings.Contains(content, "update task") || strings.Contains(content, "complete task") {
		return a.handleUpdateTask(ctx, msg)
	} else if strings.Contains(content, "project timeline") || strings.Contains(content, "project schedule") {
		return a.handleProjectTimeline(ctx, msg)
	} else if strings.Contains(content, "project budget") || strings.Contains(content, "budget") {
		return a.handleProjectBudget(ctx, msg)
	} else if strings.Contains(content, "milestone") {
		return a.handleMilestone(ctx, msg)
	} else {
		// Use LLM for general project management queries
		return a.handleGeneralQuery(ctx, msg)
	}
}

// handleCreateProject creates a new project
func (a *ProjectManagerAgent) handleCreateProject(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context for LLM to extract project details
	contextPrompt := fmt.Sprintf(`
You are a project manager extracting project details from user input.
Extract the following information from this request: "%s"

Please provide the response in JSON format with these fields:
{
  "name": "project name",
  "description": "project description", 
  "priority": "low|medium|high|critical",
  "due_date": "YYYY-MM-DD format if mentioned, otherwise null",
  "estimated_hours": number if mentioned, otherwise 0,
  "tags": ["tag1", "tag2"] if any categories mentioned
}

If information is missing, make reasonable assumptions based on context.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project details: %w", err)
	}

	// Parse the JSON response
	var projectData struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Priority       string   `json:"priority"`
		DueDate        string   `json:"due_date"`
		EstimatedHours float64  `json:"estimated_hours"`
		Tags           []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(response), &projectData); err != nil {
		// If JSON parsing fails, create project with basic info
		projectData.Name = "New Project"
		projectData.Description = msg.Content
		projectData.Priority = "medium"
	}

	// Create project
	project := &Project{
		ID:             fmt.Sprintf("proj_%d", time.Now().UnixNano()),
		Name:           projectData.Name,
		Description:    projectData.Description,
		Status:         ProjectStatusPlanning,
		Priority:       a.parsePriority(projectData.Priority),
		Owner:          string(msg.From),
		CreatedAt:      time.Now(),
		Tasks:          []ProjectTask{},
		Milestones:     []Milestone{},
		Resources:      []Resource{},
		Dependencies:   []string{},
		Progress:       0.0,
		EstimatedHours: projectData.EstimatedHours,
		ActualHours:    0.0,
		Tags:           projectData.Tags,
		Metadata:       make(map[string]interface{}),
	}

	// Set due date if provided
	if projectData.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", projectData.DueDate); err == nil {
			project.DueDate = &dueDate
		}
	}

	// Store project
	a.projectMutex.Lock()
	a.activeProjects[project.ID] = project
	a.projectMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		projectKey := fmt.Sprintf("project:%s", project.ID)
		a.memoryStore.Store(ctx, projectKey, project)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ Project '%s' created successfully!\n\nProject ID: %s\nStatus: %s\nPriority: %s\n\nYou can now add tasks, set milestones, and track progress.", project.Name, project.ID, project.Status, project.Priority),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"project_id": project.ID,
			"action":     "project_created",
		},
	}, nil
}

// handleListProjects lists all active projects
func (a *ProjectManagerAgent) handleListProjects(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Load projects from memory if not in cache
	a.loadProjectsFromMemory(ctx)

	a.projectMutex.RLock()
	defer a.projectMutex.RUnlock()

	if len(a.activeProjects) == 0 {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "📋 No projects found. Use 'create project' to start your first project!",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	var responseBuilder strings.Builder
	responseBuilder.WriteString("📋 **Active Projects**\n\n")

	// Sort projects by priority and due date
	projects := make([]*Project, 0, len(a.activeProjects))
	for _, project := range a.activeProjects {
		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Priority != projects[j].Priority {
			return projects[i].Priority > projects[j].Priority
		}
		if projects[i].DueDate != nil && projects[j].DueDate != nil {
			return projects[i].DueDate.Before(*projects[j].DueDate)
		}
		return projects[i].CreatedAt.Before(projects[j].CreatedAt)
	})

	for i, project := range projects {
		responseBuilder.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, project.Name, project.Status))
		responseBuilder.WriteString(fmt.Sprintf("   📅 Created: %s\n", project.CreatedAt.Format("2006-01-02")))

		if project.DueDate != nil {
			responseBuilder.WriteString(fmt.Sprintf("   ⏰ Due: %s\n", project.DueDate.Format("2006-01-02")))
		}

		responseBuilder.WriteString(fmt.Sprintf("   📊 Progress: %.1f%%\n", project.Progress))
		responseBuilder.WriteString(fmt.Sprintf("   🎯 Priority: %s\n", project.Priority))
		responseBuilder.WriteString(fmt.Sprintf("   📝 Tasks: %d\n", len(project.Tasks)))

		if len(project.Tags) > 0 {
			responseBuilder.WriteString(fmt.Sprintf("   🏷️ Tags: %s\n", strings.Join(project.Tags, ", ")))
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

// handleProjectStatus provides detailed project status
func (a *ProjectManagerAgent) handleProjectStatus(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Extract project identifier from message
	projectID := a.extractProjectID(msg.Content)

	a.projectMutex.RLock()
	project, exists := a.activeProjects[projectID]
	a.projectMutex.RUnlock()

	if !exists {
		// Try to find project by name
		project = a.findProjectByName(msg.Content)
		if project == nil {
			return &multiagent.Message{
				ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
				From:      a.id,
				To:        []multiagent.AgentID{msg.From},
				Type:      multiagent.MessageTypeResponse,
				Content:   "❌ Project not found. Use 'list projects' to see available projects.",
				ReplyTo:   msg.ID,
				Timestamp: time.Now(),
			}, nil
		}
	}

	// Calculate project statistics
	totalTasks := len(project.Tasks)
	completedTasks := 0
	overdueTasks := 0
	now := time.Now()

	for _, task := range project.Tasks {
		if task.Status == TaskStatusCompleted {
			completedTasks++
		}
		if task.DueDate != nil && task.DueDate.Before(now) && task.Status != TaskStatusCompleted {
			overdueTasks++
		}
	}

	// Build status report
	var statusBuilder strings.Builder
	statusBuilder.WriteString(fmt.Sprintf("📊 **Project Status: %s**\n\n", project.Name))
	statusBuilder.WriteString(fmt.Sprintf("🔍 **Overview**\n"))
	statusBuilder.WriteString(fmt.Sprintf("• Status: %s\n", project.Status))
	statusBuilder.WriteString(fmt.Sprintf("• Priority: %s\n", project.Priority))
	statusBuilder.WriteString(fmt.Sprintf("• Progress: %.1f%%\n", project.Progress))
	statusBuilder.WriteString(fmt.Sprintf("• Owner: %s\n", project.Owner))

	if project.DueDate != nil {
		daysUntilDue := int(time.Until(*project.DueDate).Hours() / 24)
		statusBuilder.WriteString(fmt.Sprintf("• Due Date: %s (%d days)\n", project.DueDate.Format("2006-01-02"), daysUntilDue))
	}

	statusBuilder.WriteString(fmt.Sprintf("\n📋 **Tasks Summary**\n"))
	statusBuilder.WriteString(fmt.Sprintf("• Total Tasks: %d\n", totalTasks))
	statusBuilder.WriteString(fmt.Sprintf("• Completed: %d\n", completedTasks))
	statusBuilder.WriteString(fmt.Sprintf("• Remaining: %d\n", totalTasks-completedTasks))

	if overdueTasks > 0 {
		statusBuilder.WriteString(fmt.Sprintf("• ⚠️ Overdue: %d\n", overdueTasks))
	}

	if project.EstimatedHours > 0 {
		statusBuilder.WriteString(fmt.Sprintf("\n⏱️ **Time Tracking**\n"))
		statusBuilder.WriteString(fmt.Sprintf("• Estimated: %.1f hours\n", project.EstimatedHours))
		statusBuilder.WriteString(fmt.Sprintf("• Actual: %.1f hours\n", project.ActualHours))
		if project.EstimatedHours > 0 {
			variance := ((project.ActualHours - project.EstimatedHours) / project.EstimatedHours) * 100
			statusBuilder.WriteString(fmt.Sprintf("• Variance: %.1f%%\n", variance))
		}
	}

	if len(project.Milestones) > 0 {
		statusBuilder.WriteString(fmt.Sprintf("\n🎯 **Milestones**\n"))
		for _, milestone := range project.Milestones {
			status := "Pending"
			if milestone.CompletedAt != nil {
				status = "✅ Completed"
			}
			statusBuilder.WriteString(fmt.Sprintf("• %s - %s (%s)\n", milestone.Title, milestone.DueDate.Format("2006-01-02"), status))
		}
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   statusBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"project_id": project.ID,
			"action":     "status_report",
		},
	}, nil
}

// handleAddTask adds a new task to a project
func (a *ProjectManagerAgent) handleAddTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Extract project and task information using LLM
	contextPrompt := fmt.Sprintf(`
Extract task information from this request: "%s"

Provide response in JSON format:
{
  "project_name": "name of project if mentioned",
  "task_title": "task title",
  "task_description": "detailed description",
  "priority": "low|medium|high|critical",
  "due_date": "YYYY-MM-DD if mentioned, otherwise null",
  "estimated_hours": number if mentioned, otherwise 0,
  "assignee": "person if mentioned, otherwise null"
}`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task details: %w", err)
	}

	var taskData struct {
		ProjectName     string  `json:"project_name"`
		TaskTitle       string  `json:"task_title"`
		TaskDescription string  `json:"task_description"`
		Priority        string  `json:"priority"`
		DueDate         string  `json:"due_date"`
		EstimatedHours  float64 `json:"estimated_hours"`
		Assignee        string  `json:"assignee"`
	}

	if err := json.Unmarshal([]byte(response), &taskData); err != nil {
		return nil, fmt.Errorf("failed to parse task JSON: %w", err)
	}

	// Find the project
	var project *Project
	if taskData.ProjectName != "" {
		project = a.findProjectByName(taskData.ProjectName)
	}

	if project == nil {
		// Use the most recent project or create a default one
		a.projectMutex.RLock()
		for _, p := range a.activeProjects {
			if project == nil || p.CreatedAt.After(project.CreatedAt) {
				project = p
			}
		}
		a.projectMutex.RUnlock()
	}

	if project == nil {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "❌ No project found. Please create a project first or specify which project to add the task to.",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Create new task
	task := ProjectTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:          taskData.TaskTitle,
		Description:    taskData.TaskDescription,
		Status:         TaskStatusNotStarted,
		Priority:       a.parsePriority(taskData.Priority),
		Assignee:       taskData.Assignee,
		CreatedAt:      time.Now(),
		Dependencies:   []string{},
		Progress:       0.0,
		EstimatedHours: taskData.EstimatedHours,
		ActualHours:    0.0,
		Tags:           []string{},
		Comments:       []TaskComment{},
	}

	// Set due date if provided
	if taskData.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02", taskData.DueDate); err == nil {
			task.DueDate = &dueDate
		}
	}

	// Add task to project
	a.projectMutex.Lock()
	project.Tasks = append(project.Tasks, task)
	a.projectMutex.Unlock()

	// Save project
	if a.memoryStore != nil {
		projectKey := fmt.Sprintf("project:%s", project.ID)
		a.memoryStore.Store(ctx, projectKey, project)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ Task '%s' added to project '%s'!\n\nTask ID: %s\nPriority: %s\nStatus: %s", task.Title, project.Name, task.ID, task.Priority, task.Status),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"project_id": project.ID,
			"task_id":    task.ID,
			"action":     "task_added",
		},
	}, nil
}

// handleUpdateTask updates an existing task
func (a *ProjectManagerAgent) handleUpdateTask(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract update information
	contextPrompt := fmt.Sprintf(`
Extract task update information from: "%s"

Provide response in JSON format:
{
  "task_identifier": "task name or ID mentioned",
  "status": "not_started|in_progress|on_hold|completed|cancelled if mentioned",
  "progress": number between 0-100 if mentioned, otherwise null,
  "actual_hours": number if mentioned, otherwise null,
  "comment": "any comment or note to add"
}`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse update details: %w", err)
	}

	var updateData struct {
		TaskIdentifier string   `json:"task_identifier"`
		Status         string   `json:"status"`
		Progress       *float64 `json:"progress"`
		ActualHours    *float64 `json:"actual_hours"`
		Comment        string   `json:"comment"`
	}

	if err := json.Unmarshal([]byte(response), &updateData); err != nil {
		return nil, fmt.Errorf("failed to parse update JSON: %w", err)
	}

	// Find the task
	var project *Project
	var task *ProjectTask
	a.projectMutex.Lock()
	defer a.projectMutex.Unlock()

	for _, p := range a.activeProjects {
		for i := range p.Tasks {
			if strings.Contains(strings.ToLower(p.Tasks[i].Title), strings.ToLower(updateData.TaskIdentifier)) ||
				p.Tasks[i].ID == updateData.TaskIdentifier {
				project = p
				task = &p.Tasks[i]
				break
			}
		}
		if task != nil {
			break
		}
	}

	if task == nil {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("❌ Task '%s' not found.", updateData.TaskIdentifier),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Apply updates
	var changes []string

	if updateData.Status != "" {
		oldStatus := task.Status
		task.Status = TaskStatus(updateData.Status)
		changes = append(changes, fmt.Sprintf("Status: %s → %s", oldStatus, task.Status))

		if task.Status == TaskStatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
			task.Progress = 100.0
		}
	}

	if updateData.Progress != nil {
		oldProgress := task.Progress
		task.Progress = *updateData.Progress
		changes = append(changes, fmt.Sprintf("Progress: %.1f%% → %.1f%%", oldProgress, task.Progress))
	}

	if updateData.ActualHours != nil {
		oldHours := task.ActualHours
		task.ActualHours = *updateData.ActualHours
		changes = append(changes, fmt.Sprintf("Hours: %.1f → %.1f", oldHours, task.ActualHours))

		// Update project actual hours
		project.ActualHours += (*updateData.ActualHours - oldHours)
	}

	if updateData.Comment != "" {
		comment := TaskComment{
			ID:        fmt.Sprintf("comment_%d", time.Now().UnixNano()),
			Author:    string(msg.From),
			Content:   updateData.Comment,
			Timestamp: time.Now(),
		}
		task.Comments = append(task.Comments, comment)
		changes = append(changes, "Added comment")
	}

	// Recalculate project progress
	a.recalculateProjectProgress(project)

	// Save project
	if a.memoryStore != nil {
		projectKey := fmt.Sprintf("project:%s", project.ID)
		a.memoryStore.Store(ctx, projectKey, project)
	}

	changesText := "No changes made"
	if len(changes) > 0 {
		changesText = strings.Join(changes, "\n• ")
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ Task '%s' updated successfully!\n\n**Changes:**\n• %s\n\n**Current Status:** %s (%.1f%%)", task.Title, changesText, task.Status, task.Progress),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"project_id": project.ID,
			"task_id":    task.ID,
			"action":     "task_updated",
		},
	}, nil
}

// handleProjectTimeline generates project timeline information
func (a *ProjectManagerAgent) handleProjectTimeline(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Find the project
	projectID := a.extractProjectID(msg.Content)
	project := a.getProject(ctx, projectID)

	if project == nil {
		project = a.findProjectByName(msg.Content)
	}

	if project == nil {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "❌ Project not found. Please specify a valid project name or ID.",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Build timeline
	var timelineBuilder strings.Builder
	timelineBuilder.WriteString(fmt.Sprintf("📅 **Project Timeline: %s**\n\n", project.Name))

	// Project overview
	timelineBuilder.WriteString("🎯 **Project Overview**\n")
	timelineBuilder.WriteString(fmt.Sprintf("• Start: %s\n", project.CreatedAt.Format("2006-01-02")))
	if project.DueDate != nil {
		timelineBuilder.WriteString(fmt.Sprintf("• Due: %s\n", project.DueDate.Format("2006-01-02")))
		duration := project.DueDate.Sub(project.CreatedAt).Hours() / 24
		timelineBuilder.WriteString(fmt.Sprintf("• Duration: %.0f days\n", duration))
	}

	// Milestones
	if len(project.Milestones) > 0 {
		timelineBuilder.WriteString("\n🎯 **Milestones**\n")
		for _, milestone := range project.Milestones {
			status := "📅"
			if milestone.CompletedAt != nil {
				status = "✅"
			}
			timelineBuilder.WriteString(fmt.Sprintf("• %s %s - %s\n", status, milestone.Title, milestone.DueDate.Format("2006-01-02")))
		}
	}

	// Upcoming tasks
	timelineBuilder.WriteString("\n📋 **Upcoming Tasks**\n")
	upcomingTasks := a.getUpcomingTasks(project, 7) // Next 7 days
	if len(upcomingTasks) == 0 {
		timelineBuilder.WriteString("• No upcoming tasks with due dates\n")
	} else {
		for _, task := range upcomingTasks {
			status := "📅"
			if task.Status == TaskStatusCompleted {
				status = "✅"
			} else if task.Status == TaskStatusInProgress {
				status = "⏳"
			}
			timelineBuilder.WriteString(fmt.Sprintf("• %s %s - %s (%.1f%%)\n", status, task.Title, task.DueDate.Format("2006-01-02"), task.Progress))
		}
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   timelineBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"project_id": project.ID,
			"action":     "timeline_report",
		},
	}, nil
}

// handleProjectBudget manages project budget information
func (a *ProjectManagerAgent) handleProjectBudget(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "💰 Budget tracking is available but requires additional configuration. Would you like help setting up budget tracking for your project?",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleMilestone manages project milestones
func (a *ProjectManagerAgent) handleMilestone(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🎯 Milestone management is available. You can create milestones, track progress, and set deadlines. What would you like to do with milestones?",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleGeneralQuery handles general project management questions
func (a *ProjectManagerAgent) handleGeneralQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context with project information
	contextPrompt := a.buildProjectContext(ctx, msg)

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

// Helper methods

func (a *ProjectManagerAgent) parsePriority(priority string) multiagent.Priority {
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

func (a *ProjectManagerAgent) extractProjectID(content string) string {
	// Simple extraction - could be enhanced with regex
	words := strings.Fields(content)
	for _, word := range words {
		if strings.HasPrefix(word, "proj_") {
			return word
		}
	}
	return ""
}

func (a *ProjectManagerAgent) findProjectByName(content string) *Project {
	contentLower := strings.ToLower(content)

	a.projectMutex.RLock()
	defer a.projectMutex.RUnlock()

	for _, project := range a.activeProjects {
		if strings.Contains(contentLower, strings.ToLower(project.Name)) {
			return project
		}
	}
	return nil
}

func (a *ProjectManagerAgent) getProject(ctx context.Context, projectID string) *Project {
	if projectID == "" {
		return nil
	}

	a.projectMutex.RLock()
	project, exists := a.activeProjects[projectID]
	a.projectMutex.RUnlock()

	if exists {
		return project
	}

	// Try loading from memory
	if a.memoryStore != nil {
		projectKey := fmt.Sprintf("project:%s", projectID)
		if projectInterface, err := a.memoryStore.Get(ctx, projectKey); err == nil {
			var project Project
			if projectData, err := json.Marshal(projectInterface); err == nil {
				if err := json.Unmarshal(projectData, &project); err == nil {
					a.projectMutex.Lock()
					a.activeProjects[projectID] = &project
					a.projectMutex.Unlock()
					return &project
				}
			}
		}
	}

	return nil
}

func (a *ProjectManagerAgent) loadProjectsFromMemory(ctx context.Context) {
	if a.memoryStore == nil {
		return
	}

	// List all project keys
	keys, err := a.memoryStore.List(ctx, "project:", 100)
	if err != nil {
		return
	}

	// Load projects
	projects, err := a.memoryStore.GetMultiple(ctx, keys)
	if err != nil {
		return
	}

	a.projectMutex.Lock()
	defer a.projectMutex.Unlock()

	for _, projectInterface := range projects {
		var project Project
		if projectData, err := json.Marshal(projectInterface); err == nil {
			if err := json.Unmarshal(projectData, &project); err == nil {
				a.activeProjects[project.ID] = &project
			}
		}
	}
}

func (a *ProjectManagerAgent) recalculateProjectProgress(project *Project) {
	if len(project.Tasks) == 0 {
		project.Progress = 0.0
		return
	}

	totalProgress := 0.0
	for _, task := range project.Tasks {
		totalProgress += task.Progress
	}

	project.Progress = totalProgress / float64(len(project.Tasks))
}

func (a *ProjectManagerAgent) getUpcomingTasks(project *Project, days int) []ProjectTask {
	var upcoming []ProjectTask
	cutoff := time.Now().AddDate(0, 0, days)

	for _, task := range project.Tasks {
		if task.DueDate != nil && task.DueDate.Before(cutoff) && task.Status != TaskStatusCompleted {
			upcoming = append(upcoming, task)
		}
	}

	// Sort by due date
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].DueDate.Before(*upcoming[j].DueDate)
	})

	return upcoming
}

func (a *ProjectManagerAgent) buildProjectContext(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(fmt.Sprintf("You are %s, a project management specialist.\n\n", a.name))
	contextBuilder.WriteString("You help users manage projects, tasks, timelines, and resources effectively.\n\n")

	// Add current projects summary
	a.projectMutex.RLock()
	if len(a.activeProjects) > 0 {
		contextBuilder.WriteString("Current Projects:\n")
		for _, project := range a.activeProjects {
			contextBuilder.WriteString(fmt.Sprintf("- %s (%s) - %.1f%% complete, %d tasks\n",
				project.Name, project.Status, project.Progress, len(project.Tasks)))
		}
		contextBuilder.WriteString("\n")
	}
	a.projectMutex.RUnlock()

	contextBuilder.WriteString(fmt.Sprintf("User request: %s\n\n", msg.Content))
	contextBuilder.WriteString("Please provide helpful project management advice, suggestions, or execute the requested action.")

	return contextBuilder.String()
}
