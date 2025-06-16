// Package llmprovider provides implementations of the multiagent.LLMProvider interface
// for various language model providers.
package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// LMStudioProvider implements the LLMProvider interface for LMStudio
type LMStudioProvider struct {
	ServerURL   string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Debug       bool
}

// NewLMStudioProvider creates a new LMStudio provider
func NewLMStudioProvider(serverURL string, options ...func(*LMStudioProvider)) *LMStudioProvider {
	provider := &LMStudioProvider{
		ServerURL:   serverURL,
		Model:       "default", // LMStudio typically uses the loaded model
		MaxTokens:   2048,      // Increased for more comprehensive responses
		Temperature: 0.7,
		Debug:       false,
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

// WithDebug enables or disables debug mode
func WithDebug(debug bool) func(*LMStudioProvider) {
	return func(p *LMStudioProvider) {
		p.Debug = debug
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
		"model":       p.Model,
		"temperature": p.Temperature,
		"max_tokens":  p.MaxTokens,
		"stream":      false,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Print request payload in debug mode
	if p.Debug {
		log.Printf("Request payload: %s", string(jsonData))
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.ServerURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	// Send request
	if p.Debug {
		log.Printf("Sending request to LMStudio at %s", p.ServerURL+"/chat/completions")
	}
	client := &http.Client{
		Timeout: 600 * time.Second, // Increased timeout to 10 minutes for longer generations
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Print response in debug mode
	if p.Debug {
		log.Printf("Response: %s", string(body))
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LMStudio API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content from response
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format: missing choices")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: invalid choice")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: missing message")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing content")
	}

	return content, nil
}

// QueryWithTools sends a prompt to the LMStudio server with tool definitions
func (p *LMStudioProvider) QueryWithTools(ctx context.Context, prompt string, tools []multiagent.Tool) (string, error) {
	// Convert tools to OpenAI format
	toolDefs := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolDefs = append(toolDefs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}

	// Create a more detailed prompt that explains the tools
	fullPrompt := prompt
	if len(tools) > 0 {
		fullPrompt += "\n\nAvailable tools:\n"
		for _, tool := range tools {
			fullPrompt += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
		}
	}

	// Send the query
	response, err := p.Query(ctx, fullPrompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
