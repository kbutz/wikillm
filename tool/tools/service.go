package tools

import (
	"context"
	"fmt"
)

// TodoListService provides high-level access to TODO list functionality
type TodoListService struct {
	tool *ImprovedTodoListTool
}

// NewTodoListService creates a new TODO list service
func NewTodoListService(filePath string) *TodoListService {
	return &TodoListService{
		tool: NewImprovedTodoListTool(filePath),
	}
}

// GetTaskList returns the current task list
func (s *TodoListService) GetTaskList() (*TaskList, error) {
	return s.tool.loadTasks()
}

// ExecuteCommand executes a tool command
func (s *TodoListService) ExecuteCommand(ctx context.Context, command string) (string, error) {
	return s.tool.Execute(ctx, command)
}

// GetMostImportantTask returns the most important task with context
func (s *TodoListService) GetMostImportantTask() (string, error) {
	taskList, err := s.GetTaskList()
	if err != nil {
		return "", err
	}

	tasks := taskList.GetTasksByPriority()
	if len(tasks) == 0 {
		return "You have no active tasks.", nil
	}

	mostImportant := tasks[0]

	// Count tasks by priority
	priorityCounts := make(map[TaskPriority]int)
	totalActive := len(tasks)
	for _, task := range tasks {
		priorityCounts[task.Priority]++
	}

	response := fmt.Sprintf("The most important task is \"%s\" (Priority: %s).",
		mostImportant.Description, mostImportant.Priority.String())

	// Add context about critical tasks if any
	if criticalCount := priorityCounts[PriorityCritical]; criticalCount > 0 {
		percentage := float64(criticalCount) / float64(totalActive) * 100
		response += fmt.Sprintf(" You have %d critical task(s), which is %.0f%% of your active tasks.",
			criticalCount, percentage)
	}

	// Add time estimate if available
	if mostImportant.TimeEstimate > 0 {
		hours := mostImportant.TimeEstimate / 60
		minutes := mostImportant.TimeEstimate % 60
		if hours > 0 {
			response += fmt.Sprintf(" Estimated time: %dh %dm.", hours, minutes)
		} else {
			response += fmt.Sprintf(" Estimated time: %d minutes.", minutes)
		}
	}

	return response, nil
}

// GetTasksByDifficulty returns tasks sorted by difficulty (using time as proxy)
func (s *TodoListService) GetTasksByDifficulty(ascending bool) (string, error) {
	taskList, err := s.GetTaskList()
	if err != nil {
		return "", err
	}

	activeTasks := taskList.GetActiveTasks()

	// Separate tasks with and without time estimates
	var tasksWithTime []Task
	var tasksWithoutTime []Task

	for _, task := range activeTasks {
		if task.TimeEstimate > 0 {
			tasksWithTime = append(tasksWithTime, task)
		} else {
			tasksWithoutTime = append(tasksWithoutTime, task)
		}
	}

	if len(tasksWithTime) == 0 {
		return "No tasks have time estimates to determine difficulty. Add time estimates to tasks using 'add <task> time:XXm'.", nil
	}

	// Sort by time
	for i := 0; i < len(tasksWithTime)-1; i++ {
		for j := i + 1; j < len(tasksWithTime); j++ {
			shouldSwap := false
			if ascending && tasksWithTime[j].TimeEstimate < tasksWithTime[i].TimeEstimate {
				shouldSwap = true
			} else if !ascending && tasksWithTime[j].TimeEstimate > tasksWithTime[i].TimeEstimate {
				shouldSwap = true
			}
			if shouldSwap {
				tasksWithTime[i], tasksWithTime[j] = tasksWithTime[j], tasksWithTime[i]
			}
		}
	}

	// Format response
	order := "easiest"
	if !ascending {
		order = "most difficult"
	}

	response := fmt.Sprintf("Tasks ranked by difficulty (%s first):\n\n", order)

	for i, task := range tasksWithTime {
		timeStr := fmt.Sprintf("%d minutes", task.TimeEstimate)
		if task.TimeEstimate >= 60 {
			hours := task.TimeEstimate / 60
			minutes := task.TimeEstimate % 60
			if minutes > 0 {
				timeStr = fmt.Sprintf("%dh %dm", hours, minutes)
			} else {
				timeStr = fmt.Sprintf("%d hour(s)", hours)
			}
		}
		response += fmt.Sprintf("%d. %s (%s) [%s]\n",
			i+1, task.Description, timeStr, task.Priority.String())
	}

	if len(tasksWithoutTime) > 0 {
		response += fmt.Sprintf("\n%d task(s) without time estimates not included in difficulty ranking.",
			len(tasksWithoutTime))
	}

	return response, nil
}

// GetTaskSummary returns a comprehensive task summary
func (s *TodoListService) GetTaskSummary() (string, error) {
	result, err := s.tool.Execute(context.Background(), "analyze summary")
	return result, err
}

// StructuredQuery represents a parsed query with intent
type StructuredQuery struct {
	Intent     string                 `json:"intent"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ParseQuery attempts to parse a natural language query into a structured format
func (s *TodoListService) ParseQuery(query string) (*StructuredQuery, error) {
	// This would ideally use NLP, but for now we'll use simple pattern matching
	// This is where you could integrate with the QueryAnalyzer

	analyzer := NewQueryAnalyzer()
	queryType := analyzer.AnalyzeQuery(query)

	structured := &StructuredQuery{
		Parameters: make(map[string]interface{}),
	}

	switch queryType {
	case QueryTypeMostImportant:
		structured.Intent = "get_most_important"
	case QueryTypeDifficulty:
		structured.Intent = "get_by_difficulty"
		structured.Parameters["ascending"] = true // Default to easiest first
	case QueryTypeSummary:
		structured.Intent = "get_summary"
	case QueryTypeList:
		structured.Intent = "list_tasks"
	default:
		structured.Intent = "unknown"
	}

	return structured, nil
}
