package tools

import (
	"fmt"
	"strings"
)

// QueryType represents the type of TODO query
type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeMostImportant
	QueryTypeDifficulty
	QueryTypeSummary
	QueryTypePriority
	QueryTypeTime
	QueryTypeList
)

// QueryAnalyzer analyzes user queries to determine intent
type QueryAnalyzer struct {
	patterns map[QueryType][]string
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer() *QueryAnalyzer {
	return &QueryAnalyzer{
		patterns: map[QueryType][]string{
			QueryTypeMostImportant: {
				"most important",
				"highest priority",
				"urgent",
				"critical",
				"what should i do first",
				"top priority",
				"most urgent",
			},
			QueryTypeDifficulty: {
				"easiest",
				"hardest",
				"most difficult",
				"least difficult",
				"by difficulty",
				"simple tasks",
				"complex tasks",
			},
			QueryTypeSummary: {
				"summary",
				"summarize",
				"overview",
				"status",
				"how many tasks",
				"what's on my list",
			},
			QueryTypePriority: {
				"by priority",
				"priority order",
				"ranked by importance",
				"importance order",
			},
			QueryTypeTime: {
				"how long",
				"time estimate",
				"duration",
				"by time",
				"quick tasks",
				"long tasks",
			},
			QueryTypeList: {
				"show list",
				"show tasks",
				"list all",
				"what tasks",
			},
		},
	}
}

// AnalyzeQuery determines the query type from user input
func (qa *QueryAnalyzer) AnalyzeQuery(query string) QueryType {
	lowerQuery := strings.ToLower(query)
	
	// Check each pattern
	for queryType, patterns := range qa.patterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerQuery, pattern) {
				return queryType
			}
		}
	}
	
	// Default checks
	if strings.Contains(lowerQuery, "todo") || strings.Contains(lowerQuery, "task") {
		return QueryTypeList
	}
	
	return QueryTypeUnknown
}

// GetToolCommand converts a query type to the appropriate tool command
func (qa *QueryAnalyzer) GetToolCommand(queryType QueryType) string {
	switch queryType {
	case QueryTypeMostImportant:
		return "analyze priority"
	case QueryTypeDifficulty:
		return "analyze time" // Time is a proxy for difficulty
	case QueryTypeSummary:
		return "analyze summary"
	case QueryTypePriority:
		return "list priority"
	case QueryTypeTime:
		return "analyze time"
	case QueryTypeList:
		return "list"
	default:
		return "list"
	}
}

// FormatResponseForQuery formats the tool response based on query type
func FormatResponseForQuery(queryType QueryType, toolResponse string, taskList *TaskList) string {
	switch queryType {
	case QueryTypeMostImportant:
		return formatMostImportantResponse(toolResponse, taskList)
	case QueryTypeDifficulty:
		return formatDifficultyResponse(toolResponse, taskList)
	default:
		return toolResponse
	}
}

// formatMostImportantResponse extracts and formats the most important task
func formatMostImportantResponse(toolResponse string, taskList *TaskList) string {
	tasks := taskList.GetTasksByPriority()
	if len(tasks) == 0 {
		return "You have no active tasks."
	}
	
	mostImportant := tasks[0]
	priorityCounts := make(map[TaskPriority]int)
	for _, task := range tasks {
		priorityCounts[task.Priority]++
	}
	
	response := fmt.Sprintf("The most important task is \"%s\". This task is marked as %s.", 
		mostImportant.Description, mostImportant.Priority.String())
	
	// Add context if there are critical tasks
	criticalCount := priorityCounts[PriorityCritical]
	if criticalCount > 0 {
		percentage := float64(criticalCount) / float64(len(tasks)) * 100
		response += fmt.Sprintf(" Critical tasks account for %.0f%% of all tasks.", percentage)
	}
	
	return response
}

// formatDifficultyResponse formats tasks by difficulty (using time as proxy)
func formatDifficultyResponse(toolResponse string, taskList *TaskList) string {
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
		return "No tasks have time estimates to determine difficulty."
	}
	
	// Sort by time (ascending for easiest first)
	for i := 0; i < len(tasksWithTime)-1; i++ {
		for j := i + 1; j < len(tasksWithTime); j++ {
			if tasksWithTime[j].TimeEstimate < tasksWithTime[i].TimeEstimate {
				tasksWithTime[i], tasksWithTime[j] = tasksWithTime[j], tasksWithTime[i]
			}
		}
	}
	
	var response strings.Builder
	response.WriteString("Tasks ranked by difficulty (easiest first):\n\n")
	
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
		response.WriteString(fmt.Sprintf("%d. %s (%s) [%s]\n", 
			i+1, task.Description, timeStr, task.Priority.String()))
	}
	
	if len(tasksWithoutTime) > 0 {
		response.WriteString(fmt.Sprintf("\n%d tasks without time estimates.", len(tasksWithoutTime)))
	}
	
	return response.String()
}
