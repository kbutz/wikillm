package tools

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskPriority represents the importance level of a task
type TaskPriority int

const (
	PriorityLow TaskPriority = iota + 1
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of a TaskPriority
func (p TaskPriority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Medium"
	case PriorityHigh:
		return "High"
	case PriorityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// ParsePriority converts a string to TaskPriority
func ParsePriority(s string) TaskPriority {
	switch s {
	case "low", "Low", "LOW", "1":
		return PriorityLow
	case "medium", "Medium", "MEDIUM", "2", "med":
		return PriorityMedium
	case "high", "High", "HIGH", "3":
		return PriorityHigh
	case "critical", "Critical", "CRITICAL", "4", "crit":
		return PriorityCritical
	default:
		return PriorityMedium // Default to medium if not specified
	}
}

// Task represents a TODO item with metadata
type Task struct {
	ID           string       `json:"id"`
	Description  string       `json:"description"`
	Priority     TaskPriority `json:"priority"`
	TimeEstimate int          `json:"time_estimate_minutes,omitempty"` // Time estimate in minutes
	CreatedAt    time.Time    `json:"created_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	Completed    bool         `json:"completed"`
}

// NewTask creates a new task with the given description
func NewTask(description string) *Task {
	return &Task{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Description: description,
		Priority:    PriorityMedium,
		CreatedAt:   time.Now(),
		Completed:   false,
	}
}

// String returns a formatted string representation of the task
func (t *Task) String() string {
	status := "[ ]"
	if t.Completed {
		status = "[✓]"
	}

	timeStr := ""
	if t.TimeEstimate > 0 {
		hours := t.TimeEstimate / 60
		minutes := t.TimeEstimate % 60
		if hours > 0 {
			timeStr = fmt.Sprintf(" (~%dh%dm)", hours, minutes)
		} else {
			timeStr = fmt.Sprintf(" (~%dm)", minutes)
		}
	}

	return fmt.Sprintf("%s %s [%s]%s", status, t.Description, t.Priority.String(), timeStr)
}

// TaskList represents a collection of tasks
type TaskList struct {
	Tasks []Task `json:"tasks"`
}

// Add adds a new task to the list
func (tl *TaskList) Add(task Task) {
	tl.Tasks = append(tl.Tasks, task)
}

// Remove removes a task by index (1-based)
func (tl *TaskList) Remove(index int) error {
	if index < 1 || index > len(tl.Tasks) {
		return fmt.Errorf("task index %d out of range (1-%d)", index, len(tl.Tasks))
	}

	// Remove the task
	tl.Tasks = append(tl.Tasks[:index-1], tl.Tasks[index:]...)
	return nil
}

// Complete marks a task as completed by index (1-based)
func (tl *TaskList) Complete(index int) error {
	if index < 1 || index > len(tl.Tasks) {
		return fmt.Errorf("task index %d out of range (1-%d)", index, len(tl.Tasks))
	}

	now := time.Now()
	tl.Tasks[index-1].Completed = true
	tl.Tasks[index-1].CompletedAt = &now
	return nil
}

// GetActiveTasks returns all non-completed tasks
func (tl *TaskList) GetActiveTasks() []Task {
	var active []Task
	for _, task := range tl.Tasks {
		if !task.Completed {
			active = append(active, task)
		}
	}
	return active
}

// GetTasksByPriority returns active tasks sorted by priority (highest first)
func (tl *TaskList) GetTasksByPriority() []Task {
	active := tl.GetActiveTasks()

	// Simple bubble sort by priority (descending)
	for i := 0; i < len(active); i++ {
		for j := i + 1; j < len(active); j++ {
			if active[j].Priority > active[i].Priority {
				active[i], active[j] = active[j], active[i]
			}
		}
	}

	return active
}

// Marshal converts the TaskList to JSON
func (tl *TaskList) Marshal() ([]byte, error) {
	return json.MarshalIndent(tl, "", "  ")
}

// Unmarshal loads a TaskList from JSON
func (tl *TaskList) Unmarshal(data []byte) error {
	return json.Unmarshal(data, tl)
}
