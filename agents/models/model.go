package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

// LLMModel defines the interface for interacting with language models
type LLMModel interface {
	// Query sends a prompt to the model and returns the response
	Query(ctx context.Context, prompt string) (string, error)

	// Name returns the name of the model
	Name() string
}

func New(modelName, provider string) (LLMModel, error) {
	switch strings.ToLower(provider) {
	case "ollama":
		return NewOllamaModel(modelName)
	case "lmstudio":
		return NewLMStudioModel(modelName)
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
		MaxTokens:   300,
		Temperature: 0.3,
		StopSequences: []string{
			"**Answer:**",
			"Wait,",
			"I should",
			"Let me",
			"\n\n\n",
		},
		SystemPrompt: "You are a helpful task management assistant. " +
			"Provide direct, concise answers without explaining your reasoning process. " +
			"Start immediately with the answer to the user's question.",
	}
}

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

// Name returns the name of the model
func (m *OllamaModel) Name() string {
	return m.name
}

// LMStudioModel implements the LLMModel interface using LM Studio
type LMStudioModel struct {
	apiURL string
	name   string
}

// LMStudioMessage represents a message in the chat completion request
type LMStudioMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LMStudioChatRequest represents the request body for the LM Studio chat completions API
type LMStudioChatRequest struct {
	Model       string             `json:"model"`
	Messages    []LMStudioMessage  `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Stop        []string           `json:"stop,omitempty"`
}

// LMStudioChatResponse represents the response from the LM Studio chat completions API
type LMStudioChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

// NewLMStudioModel creates a new instance of LMStudioModel
func NewLMStudioModel(modelName string) (*LMStudioModel, error) {
	apiURL := "http://localhost:1234/v1/chat/completions"

	return &LMStudioModel{
		apiURL: apiURL,
		name:   modelName,
	}, nil
}

// Query sends a prompt to the LM Studio model and returns the response
func (m *LMStudioModel) Query(ctx context.Context, prompt string) (string, error) {
	// Get default config for task queries
	config := defaultLLMConfig()

	// Build messages array with proper role structure
	messages := []LMStudioMessage{
		{
			Role:    "system",
			Content: config.SystemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody := LMStudioChatRequest{
		Model:       m.name,
		Messages:    messages,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		Stop:        config.StopSequences,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.apiURL, bytes.NewBuffer(reqBytes))
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

	body, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("LMStudio HTTP Response: %s\n", string(body))

	var chatResp LMStudioChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	fmt.Printf("LMStudio decoded choices: %v\n", chatResp)
	// Trim any leading/trailing whitespace
	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// Name returns the name of the model
func (m *LMStudioModel) Name() string {
	return m.name
}
