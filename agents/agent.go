package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kbutz/wikillm/agents/models"
)

//go:generate mockgen -source=agent.go -package=main -destination=./mocks/agent_mock.go

// Agent represents an LLM agent with access to tools
type Agent struct {
	model models.LLMModel
	tools []models.Tool
}

// NewAgent creates a new Agent with the given model and tools
func NewAgent(model models.LLMModel, tools []models.Tool) *Agent {
	return &Agent{
		model: model,
		tools: tools,
	}
}

// Run Start an interactive session with the user
func (a *Agent) Run() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("LLM Agent with Tools")
	fmt.Printf("Using model: %s\n", a.model.Name())
	fmt.Println("Available tools:")
	for _, tool := range a.tools {
		fmt.Printf("- %s: %s\n", tool.Name(), strings.Split(tool.Description(), "\n")[0])
	}
	fmt.Println("Type 'exit' to quit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		query := scanner.Text()
		if strings.ToLower(query) == "exit" {
			break
		}

		// Process the query
		fmt.Println("Processing your request...")
		startTime := time.Now()

		response, err := a.ProcessQuery(context.Background(), query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		fmt.Printf("\nResponse:\n%s\n", response)
		fmt.Printf("\nResponse generated in %.2f seconds.\n", elapsed.Seconds())
	}
}

// ProcessQuery processes a user query and returns a response
func (a *Agent) ProcessQuery(ctx context.Context, query string) (string, error) {
	// Use the model's QueryWithTools method to handle tool calls
	response, err := a.model.QueryWithTools(ctx, query, a.tools)
	if err != nil {
		return "", fmt.Errorf("error querying LLM with tools: %w", err)
	}

	return response, nil
}
