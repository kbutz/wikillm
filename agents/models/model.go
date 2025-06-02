package models

import (
	"context"
	"fmt"
	"strings"
)

//go:generate mockgen -source=model.go -package=main -destination=./mocks/model_mock.go

// LLMModel defines the interface for interacting with language models
type LLMModel interface {
	// Name returns the name of the model
	Name() string
	// Query sends a prompt to the model and returns the response
	Query(ctx context.Context, prompt string) (string, error)
	// QueryWithTools sends a prompt to the model with available tools and returns the response
	QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error)
}

// Tool defines the interface for tools that the model can use
type Tool interface {
	// Name returns the name of the tool
	Name() string
	// Description returns a description of what the tool does
	Description() string
	// Parameters returns the parameter schema for the tool
	Parameters() map[string]interface{}
	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, args string) (string, error)
}

func New(modelName, provider string, debug bool) (LLMModel, error) {
	switch strings.ToLower(provider) {
	case "ollama":
		return NewOllamaModel(modelName)
	case "lmstudio":
		return NewLMStudioModel(modelName, debug)
	default:
		return nil, fmt.Errorf("unknown model provider: %s", provider)
	}
}

// LLMConfig provides configuration for LLM behavior
type LLMConfig struct {
	MaxTokens     int
	Temperature   float64
	StopSequences []string
	SystemPrompt  string
}

// DefaultLLMConfig returns default configuration for task queries
func defaultLLMConfig() LLMConfig {
	return LLMConfig{
		//MaxTokens:   1000,
		Temperature: 0.3,
		SystemPrompt: "You an expert Project Manager with expertise in breaking down and prioritizing tasks. " +
			"Please, provide assistance with task management, project planning, and software development processes.",
	}
}
