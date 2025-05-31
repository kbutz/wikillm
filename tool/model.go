package main

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

// LLMModel defines the interface for interacting with language models
type LLMModel interface {
	// Query sends a prompt to the model and returns the response
	Query(ctx context.Context, prompt string) (string, error)

	// Name returns the name of the model
	Name() string
}

// OllamaModel implements the LLMModel interface using Ollama
type OllamaModel struct {
	apiURL string
	name   string
}

// OllamaGenerateRequest represents the request body for the Ollama generate API
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
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
	reqBody := OllamaGenerateRequest{
		Model:  m.name,
		Prompt: prompt,
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

// LMStudioRequest represents the request body for the LM Studio API
type LMStudioRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// LMStudioResponse represents the response from the LM Studio API
type LMStudioResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// NewLMStudioModel creates a new instance of LMStudioModel
func NewLMStudioModel(modelName string) (*LMStudioModel, error) {
	apiURL := "http://localhost:1234/v1/completions"

	return &LMStudioModel{
		apiURL: apiURL,
		name:   modelName,
	}, nil
}

// Query sends a prompt to the LM Studio model and returns the response
func (m *LMStudioModel) Query(ctx context.Context, prompt string) (string, error) {
	reqBody := LMStudioRequest{
		Model:       m.name,
		Prompt:      prompt,
		MaxTokens:   2048,
		Temperature: 0.7,
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

	var generateResp LMStudioResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", err
	}

	if len(generateResp.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	// Trim any leading/trailing whitespace
	return strings.TrimSpace(generateResp.Choices[0].Text), nil
}

// Name returns the name of the model
func (m *LMStudioModel) Name() string {
	return m.name
}