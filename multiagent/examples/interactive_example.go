// This example demonstrates how to create an interactive command-line interface
// for the multiagent service. It accepts user input from the command line,
// sends it to the multiagent service, and displays the response.
//
// To run this example:
//
//	cd multiagent/examples
//	go run interactive_example.go
//
// This example uses a simple mock LLM provider for demonstration purposes.
// For LMStudio integration, see the lmstudio_example.go file.
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
	"time"

	"github.com/kbutz/wikillm/multiagent"
	"github.com/kbutz/wikillm/multiagent/service"
)

// LMStudioProvider implements the LLMProvider interface for LMStudio
type LMStudioProvider struct {
	ServerURL   string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
}

// NewLMStudioProvider creates a new LMStudio provider
func NewLMStudioProvider(serverURL string, options ...func(*LMStudioProvider)) *LMStudioProvider {
	provider := &LMStudioProvider{
		ServerURL:   serverURL,
		Model:       "default", // LMStudio typically uses the loaded model
		MaxTokens:   1024,
		Temperature: 0.7,
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
		"temperature": p.Temperature,
		"max_tokens":  p.MaxTokens,
		"stream":      false,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.ServerURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned error: %s", body)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content format")
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

func main() {
	// Create a base directory for the service
	baseDir := filepath.Join(os.TempDir(), "wikillm_interactive_example")
	os.RemoveAll(baseDir) // Clean up any previous run
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	// Create LMStudio provider
	// Default LMStudio server URL is http://localhost:1234/v1
	llmProvider := NewLMStudioProvider("http://localhost:1234/v1",
		WithTemperature(0.7),
		WithMaxTokens(2048),
	)

	// Create the multi-agent service
	svc, err := service.NewMultiAgentService(service.ServiceConfig{
		BaseDir:     baseDir,
		LLMProvider: llmProvider,
	})
	if err != nil {
		log.Fatalf("Failed to create multi-agent service: %v", err)
	}

	// Start the service
	ctx := context.Background()
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	fmt.Println("\n=== WikiLLM MultiAgent Interactive Example ===")
	fmt.Println("Type your messages and press Enter to interact with the agents")
	fmt.Println("Type 'exit' to quit the application")
	fmt.Println("==============================================\n")

	// Generate a unique user ID
	userID := fmt.Sprintf("user_%d", time.Now().UnixNano())

	// Create a scanner for user input
	scanner := bufio.NewScanner(os.Stdin)

	// Main interaction loop
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()

		// Check if user wants to exit
		if strings.ToLower(input) == "exit" {
			fmt.Println("Exiting...")
			break
		}

		// Process the user message
		response, err := svc.ProcessUserMessage(ctx, userID, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Print the response
		fmt.Printf("\n%s\n\n", response)
	}

	// Stop the service
	if err := svc.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}

	log.Println("Example completed successfully")
}
