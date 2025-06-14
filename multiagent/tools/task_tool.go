package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// TaskTool provides agents with task management capabilities
type TaskTool struct {
	name        string
	description string
	memoryStore multiagent.MemoryStore
	orchestrator multiagent.Orchestrator
}

// NewTaskTool creates a new task management tool
func NewTaskTool(memoryStore multiagent.MemoryStore, orchestrator multiagent.Orchestrator) *TaskTool {
	return &TaskTool{
		name:        "task",
		description: "Create and manage tasks",
		memoryStore: memoryStore,
		orchestrator: orchestrator,
	}
}

// Name returns the name of the tool
func (t *TaskTool) Name() string {
	return t.name
}

// Description returns a description of what the tool does
func (t *TaskTool) Description() string {
	return `Task management tool for creating and tracking tasks.
Commands:
- create <description> [priority] - Create a new task
- assign <task_id> <agent_id> - Assign a task to an agent
- complete <task_id> [output] - Mark a task as completed
- status <task_id> - Check task status
- list [status] - List tasks, optionally filtered by status

Examples:
- create "Research machine learning algorithms" high
- assign task_20230615_1 research_agent
- complete task_20230615_1 "Research completed, findings stored in memory"
- status task_20230615_1
- list pending`
}

// Parameters returns the parameter schema for the tool
func (t *TaskTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command to execute",
				"enum":        []string{"create", "assign", "complete", "status", "list"},
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Task description",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"description": "Task priority (low, medium, high, critical)",
				"enum":        []string{"low", "medium", "high", "critical"},
			},
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the task",
			},
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the agent to assign the task to",
			},
			"output": map[string]interface{}{
				"type":        "string",
				"description": "Output or result of the completed task",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Task status to filter by",
				"enum":        []string{"pending", "assigned", "in_progress", "completed", "failed", "cancelled"},
			},
		},
		"required": []string{"command"},
	}
}

// Execute runs the tool with the given arguments and returns the result
func (t *TaskTool) Execute(ctx context.Context, args string) (string, error) {
	var params map[string]interface{}

	// Try to parse as JSON
	if strings.HasPrefix(strings.TrimSpace(args), "{") {
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return "", fmt.Errorf("failed to parse JSON arguments: %w", err)
		}
	} else {
		// Parse simple command format
		params = t.parseSimpleCommand(args)
	}

	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}

	switch command {
	case "create":
		return t.executeCreate(ctx, params)
	case "assign":
		return t.executeAssign(ctx, params)
	case "complete":
		return t.executeComplete(ctx, params)
	case "status":
		return t.executeStatus(ctx, params)
	case "list":
		return t.executeList(ctx, params)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// parseSimpleCommand parses simple command format
func (t *TaskTool) parseSimpleCommand(args string) map[string]interface{} {
	parts := strings.Fields(args)
	params := make(map[string]interface{})

	if len(parts) == 0 {
		return params
	}

	params["command"] = parts[0]

	switch parts[0] {
	case "create":
		if len(parts) > 1 {
			// Check for quoted description
			descStart := strings.Index(args, "\"")
			descEnd := strings.LastIndex(args, "\"")
			
			if descStart != -1 && descEnd != -1 && descEnd > descStart {
				params["description"] = args[descStart+1 : descEnd]
				
				// Check for priority after the quoted description
				if len(parts) > 2 && descEnd+1 < len(args) {
					remainder := strings.TrimSpace(args[descEnd+1:])
					if remainder != "" {
						params["priority"] = remainder
					}
				}
			} else {
				// No quotes, use all remaining parts as description
				params["description"] = strings.Join(parts[1:], " ")
			}
		}
	case "assign":
		if len(parts) > 2 {
			params["task_id"] = parts[1]
			params["agent_id"] = parts[2]
		}
	case "complete":
		if len(parts) > 1 {
			params["task_id"] = parts[1]
			
			// Check for output
			if len(parts) > 2 {
				outputStart := strings.Index(args, "\"")
				outputEnd := strings.LastIndex(args, "\"")
				
				if outputStart != -1 && outputEnd != -1 && outputEnd > outputStart {
					params["output"] = args[outputStart+1 : outputEnd]
				} else {
					// Join remaining parts as output
					outputParts := parts[2:]
					params["output"] = strings.Join(outputParts, " ")
				}
			}
		}
	case "status":
		if len(parts) > 1 {
			params["task_id"] = parts[1]
		}
	case "list":
		if len(parts) > 1 {
			params["status"] = parts[1]
		}
	}

	return params
}

// executeCreate handles creating a new task
func (t *TaskTool) executeCreate(ctx context.Context, params map[string]interface{}) (string, error) {
	description, ok := params["description"].(string)
	if !ok {
		return "", fmt.Errorf("description is required for create command")
	}

	// Parse priority
	priority := multiagent.PriorityMedium
	if priorityStr, ok := params["priority"].(string); ok {
		switch strings.ToLower(priorityStr) {
		case "low":
			priority = multiagent.PriorityLow
		case "medium":
			priority = multiagent.PriorityMedium
		case "high":
			priority = multiagent.PriorityHigh
		case "critical":
			priority = multiagent.PriorityCritical
		}
	}

	// Create task ID
	taskID := fmt.Sprintf("task_%s", time.Now().Format("20060102_150405"))

	// Create task
	task := multiagent.Task{
		ID:          taskID,
		Type:        "general",
		Description: description,
		Priority:    priority,
		Status:      multiagent.TaskStatusPending,
		CreatedAt:   time.Now(),
		Input:       make(map[string]interface{}),
		Output:      make(map[string]interface{}),
	}

	// Store task in memory
	if err := t.memoryStore.Store(ctx, taskID, task); err != nil {
		return "", fmt.Errorf("failed to store task: %w", err)
	}

	// Add task to index
	t.updateTaskIndex(ctx, task)

	// If orchestrator is available, try to assign the task
	if t.orchestrator != nil {
		go func() {
			// Use a new context to avoid cancellation
			assignCtx := context.Background()
			if _, err := t.orchestrator.AssignTask(assignCtx, task); err != nil {
				// Just log the error, don't fail the task creation
				fmt.Printf("Failed to auto-assign task: %v\n", err)
			}
		}()
	}

	return fmt.Sprintf("✓ Task created with ID: %s", taskID), nil
}

// executeAssign handles assigning a task to an agent
func (t *TaskTool) executeAssign(ctx context.Context, params map[string]interface{}) (string, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required for assign command")
	}

	agentID, ok := params["agent_id"].(string)
	if !ok {
		return "", fmt.Errorf("agent_id is required for assign command")
	}

	// Retrieve task
	taskInterface, err := t.memoryStore.Get(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Convert to Task
	var task multiagent.Task
	taskData, err := json.Marshal(taskInterface)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task data: %w", err)
	}
	
	if err := json.Unmarshal(taskData, &task); err != nil {
		return "", fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Update task
	task.Assignee = multiagent.AgentID(agentID)
	task.Status = multiagent.TaskStatusAssigned
	now := time.Now()
	task.StartedAt = &now

	// Store updated task
	if err := t.memoryStore.Store(ctx, taskID, task); err != nil {
		return "", fmt.Errorf("failed to update task: %w", err)
	}

	// Notify agent about the task
	if t.orchestrator != nil {
		message := &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", taskID, time.Now().UnixNano()),
			From:      multiagent.AgentID("task_tool"),
			To:        []multiagent.AgentID{multiagent.AgentID(agentID)},
			Type:      multiagent.MessageTypeCommand,
			Content:   fmt.Sprintf("You have been assigned task %s: %s", taskID, task.Description),
			Priority:  task.Priority,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"task_id":      taskID,
				"task_type":    task.Type,
				"task_priority": task.Priority,
			},
		}

		if err := t.orchestrator.RouteMessage(ctx, message); err != nil {
			return "", fmt.Errorf("failed to notify agent: %w", err)
		}
	}

	return fmt.Sprintf("✓ Task %s assigned to agent %s", taskID, agentID), nil
}

// executeComplete handles marking a task as completed
func (t *TaskTool) executeComplete(ctx context.Context, params map[string]interface{}) (string, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required for complete command")
	}

	// Retrieve task
	taskInterface, err := t.memoryStore.Get(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Convert to Task
	var task multiagent.Task
	taskData, err := json.Marshal(taskInterface)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task data: %w", err)
	}
	
	if err := json.Unmarshal(taskData, &task); err != nil {
		return "", fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Update task
	task.Status = multiagent.TaskStatusCompleted
	now := time.Now()
	task.CompletedAt = &now

	// Add output if provided
	if output, ok := params["output"].(string); ok && output != "" {
		task.Output["result"] = output
	}

	// Store updated task
	if err := t.memoryStore.Store(ctx, taskID, task); err != nil {
		return "", fmt.Errorf("failed to update task: %w", err)
	}

	// Notify requester about completion
	if t.orchestrator != nil && task.Requester != "" {
		message := &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_complete_%d", taskID, time.Now().UnixNano()),
			From:      task.Assignee,
			To:        []multiagent.AgentID{task.Requester},
			Type:      multiagent.MessageTypeReport,
			Content:   fmt.Sprintf("Task %s has been completed", taskID),
			Priority:  multiagent.PriorityMedium,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"task_id":      taskID,
				"task_status":  string(task.Status),
				"task_output":  task.Output,
			},
		}

		if err := t.orchestrator.RouteMessage(ctx, message); err != nil {
			// Just log the error, don't fail the task completion
			fmt.Printf("Failed to notify requester: %v\n", err)
		}
	}

	return fmt.Sprintf("✓ Task %s marked as completed", taskID), nil
}

// executeStatus handles checking task status
func (t *TaskTool) executeStatus(ctx context.Context, params map[string]interface{}) (string, error) {
	taskID, ok := params["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required for status command")
	}

	// Retrieve task
	taskInterface, err := t.memoryStore.Get(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve task: %w", err)
	}

	// Convert to Task
	var task multiagent.Task
	taskData, err := json.Marshal(taskInterface)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task data: %w", err)
	}
	
	if err := json.Unmarshal(taskData, &task); err != nil {
		return "", fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Task %s Status:\n\n", taskID))
	output.WriteString(fmt.Sprintf("Description: %s\n", task.Description))
	output.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	output.WriteString(fmt.Sprintf("Priority: %d\n", task.Priority))
	
	if task.Requester != "" {
		output.WriteString(fmt.Sprintf("Requester: %s\n", task.Requester))
	}
	
	if task.Assignee != "" {
		output.WriteString(fmt.Sprintf("Assignee: %s\n", task.Assignee))
	}
	
	output.WriteString(fmt.Sprintf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05")))
	
	if task.StartedAt != nil {
		output.WriteString(fmt.Sprintf("Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05")))
	}
	
	if task.CompletedAt != nil {
		output.WriteString(fmt.Sprintf("Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05")))
	}
	
	if len(task.Output) > 0 {
		output.WriteString("\nOutput:\n")
		for k, v := range task.Output {
			output.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
	}

	return output.String(), nil
}

// executeList handles listing tasks
func (t *TaskTool) executeList(ctx context.Context, params map[string]interface{}) (string, error) {
	// Get task index
	taskIndexInterface, err := t.memoryStore.Get(ctx, "task_index")
	if err != nil {
		// No tasks yet
		return "No tasks found", nil
	}

	var taskIndex map[string][]string
	taskIndexData, err := json.Marshal(taskIndexInterface)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task index: %w", err)
	}
	
	if err := json.Unmarshal(taskIndexData, &taskIndex); err != nil {
		return "", fmt.Errorf("failed to unmarshal task index: %w", err)
	}

	// Filter by status if provided
	statusFilter := ""
	if status, ok := params["status"].(string); ok {
		statusFilter = status
	}

	// Get tasks
	var tasks []multiagent.Task
	var taskIDs []string

	if statusFilter != "" && taskIndex[statusFilter] != nil {
		taskIDs = taskIndex[statusFilter]
	} else {
		// Collect all task IDs
		for _, ids := range taskIndex {
			taskIDs = append(taskIDs, ids...)
		}
	}

	// Retrieve tasks
	for _, taskID := range taskIDs {
		taskInterface, err := t.memoryStore.Get(ctx, taskID)
		if err != nil {
			continue
		}

		var task multiagent.Task
		taskData, err := json.Marshal(taskInterface)
		if err != nil {
			continue
		}
		
		if err := json.Unmarshal(taskData, &task); err != nil {
			continue
		}

		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		if statusFilter != "" {
			return fmt.Sprintf("No tasks found with status: %s", statusFilter), nil
		}
		return "No tasks found", nil
	}

	// Format output
	var output strings.Builder
	if statusFilter != "" {
		output.WriteString(fmt.Sprintf("Tasks with status '%s' (found %d):\n\n", statusFilter, len(tasks)))
	} else {
		output.WriteString(fmt.Sprintf("All tasks (found %d):\n\n", len(tasks)))
	}

	for i, task := range tasks {
		output.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, task.ID, task.Description))
		output.WriteString(fmt.Sprintf("   Status: %s, Priority: %d\n", task.Status, task.Priority))
		
		if task.Assignee != "" {
			output.WriteString(fmt.Sprintf("   Assignee: %s\n", task.Assignee))
		}
	}

	return output.String(), nil
}

// updateTaskIndex updates the task index in memory
func (t *TaskTool) updateTaskIndex(ctx context.Context, task multiagent.Task) error {
	// Get current index
	var taskIndex map[string][]string
	
	taskIndexInterface, err := t.memoryStore.Get(ctx, "task_index")
	if err == nil {
		// Convert existing index
		taskIndexData, err := json.Marshal(taskIndexInterface)
		if err != nil {
			return fmt.Errorf("failed to marshal task index: %w", err)
		}
		
		if err := json.Unmarshal(taskIndexData, &taskIndex); err != nil {
			return fmt.Errorf("failed to unmarshal task index: %w", err)
		}
	} else {
		// Create new index
		taskIndex = make(map[string][]string)
	}

	// Update index
	status := string(task.Status)
	if taskIndex[status] == nil {
		taskIndex[status] = []string{}
	}

	// Check if task already exists in this status
	exists := false
	for _, id := range taskIndex[status] {
		if id == task.ID {
			exists = true
			break
		}
	}

	if !exists {
		taskIndex[status] = append(taskIndex[status], task.ID)
	}

	// Store updated index
	return t.memoryStore.Store(ctx, "task_index", taskIndex)
}