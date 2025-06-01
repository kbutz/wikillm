package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// OllamaModel implements the LLMModel interface using Ollama
type OllamaModel struct {
	apiURL string
	name   string
}

// OllamaGenerateRequest represents the request body for the Ollama generate API
type OllamaGenerateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Stream      bool     `json:"stream"`
	Temperature float64  `json:"temperature,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// OllamaGenerateResponse represents the response from the Ollama generate API
type OllamaGenerateResponse struct {
	Response string `json:"response"`
}

// OllamaListResponse represents the response from the Ollama list API
type OllamaListResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// NewOllamaModel creates a new instance of OllamaModel
func NewOllamaModel(modelName string) (*OllamaModel, error) {
	apiURL := "http://localhost:11434/api"

	// Check if the model exists, if not, pull it
	modelExists, err := checkModelExists(apiURL, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if model exists: %w", err)
	}

	if !modelExists {
		log.Printf("Model %s not found locally. Pulling from Ollama...", modelName)
		if err := pullModel(apiURL, modelName); err != nil {
			return nil, fmt.Errorf("failed to pull model %s: %w", modelName, err)
		}
		log.Printf("Model %s pulled successfully", modelName)
	}

	return &OllamaModel{
		apiURL: apiURL,
		name:   modelName,
	}, nil
}

// checkModelExists checks if a model exists in Ollama
func checkModelExists(apiURL, modelName string) (bool, error) {
	resp, err := http.Get(apiURL + "/tags")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to list models: status code %d", resp.StatusCode)
	}

	var listResp OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return false, err
	}

	for _, model := range listResp.Models {
		if model.Name == modelName {
			return true, nil
		}
	}

	return false, nil
}

// pullModel pulls a model from Ollama
func pullModel(apiURL, modelName string) error {
	reqBody := map[string]string{"name": modelName}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(apiURL+"/pull", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to pull model: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Name returns the name of the model
func (m *OllamaModel) Name() string {
	return m.name
}

// Query sends a prompt to the Ollama model and returns the response
func (m *OllamaModel) Query(ctx context.Context, prompt string) (string, error) {
	// Get default config for task queries
	config := defaultLLMConfig()

	reqBody := OllamaGenerateRequest{
		Model:       m.name,
		Prompt:      config.SystemPrompt + "\n\n" + prompt,
		Stream:      false,
		Temperature: config.Temperature,
		NumPredict:  config.MaxTokens,
		Stop:        config.StopSequences,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.apiURL+"/generate", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to generate response: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var generateResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", err
	}

	return generateResp.Response, nil
}

// QueryWithTools sends a prompt to the Ollama model with available tools and returns the response
// Note: Ollama doesn't natively support the OpenAI API's tool calling format, so we use a simplified approach
func (m *OllamaModel) QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error) {
	// Build a prompt that includes information about available tools
	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are an AI assistant with access to the following tools:\n\n")

	for _, tool := range tools {
		promptBuilder.WriteString(fmt.Sprintf("### Tool: %s\n%s\n\n", tool.Name(), tool.Description()))
	}

	promptBuilder.WriteString("\n## How to Use Tools\n\n")
	promptBuilder.WriteString("To use a tool, respond with a JSON object in a code block:\n\n")
	promptBuilder.WriteString("```json\n{\n  \"tool\": \"tool_name\",\n  \"args\": \"command and arguments\"\n}\n```\n\n")
	promptBuilder.WriteString("### Important Notes:\n")
	promptBuilder.WriteString("- Always use the exact tool name as shown above\n")
	promptBuilder.WriteString("- The 'args' field should contain the complete command with all parameters\n")
	promptBuilder.WriteString("- If you don't need to use a tool, just respond normally without JSON\n\n")
	promptBuilder.WriteString("User query: " + prompt + "\n\n")
	promptBuilder.WriteString("Your response:")

	// Send the prompt to the model
	response, err := m.Query(ctx, promptBuilder.String())
	if err != nil {
		return "", fmt.Errorf("error querying LLM: %w", err)
	}

	// Create a temporary struct to match the tools.ToolCall structure
	type tempToolCall struct {
		Tool string
		Args string
	}

	// Extract tool call using a simplified version of tools.ExtractToolCall
	var toolCall tempToolCall
	var found bool

	// Look for JSON blocks in the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStr := response[jsonStart : jsonEnd+1]

		// Try to unmarshal into our temporary struct
		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &rawMap); err == nil {
			// Extract the tool name
			if toolName, ok := rawMap["tool"].(string); ok && toolName != "" {
				toolCall.Tool = toolName

				// Extract args
				if args, ok := rawMap["args"].(string); ok {
					toolCall.Args = args
					found = true
				} else if args, ok := rawMap["arguments"].(string); ok {
					toolCall.Args = args
					found = true
				}
			}
		}
	}

	if !found {
		return response, nil
	}

	// Find the requested tool
	var selectedTool Tool
	for _, tool := range tools {
		if strings.EqualFold(tool.Name(), toolCall.Tool) {
			selectedTool = tool
			break
		}
	}

	if selectedTool == nil {
		return fmt.Sprintf("I tried to use the %s tool, but it's not available. Here's what I know:\n\n%s",
			toolCall.Tool, response), nil
	}

	// Execute the tool
	toolResult, err := selectedTool.Execute(ctx, toolCall.Args)
	if err != nil {
		return "", fmt.Errorf("error executing tool %s: %w", toolCall.Tool, err)
	}

	// Create a follow-up prompt with the tool result
	followUpPrompt := fmt.Sprintf(
		"Based on the following tool result, provide a clear and concise response to the user's query.\n\n"+
			"If the tool result itself is a satisfactory answer, simply summarize the tool result.\n\n"+
			"User's original query: %s\n\n"+
			"Tool result:\n%s\n\n",
		prompt, toolResult)

	// Send the follow-up prompt to the LLM
	finalResponse, err := m.Query(ctx, followUpPrompt)
	if err != nil {
		return "", fmt.Errorf("error querying LLM for final response: %w", err)
	}

	return finalResponse, nil
}
