package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
)

// LMStudioModel implements the LLMModel interface using LM Studio
type LMStudioModel struct {
	apiURL string
	name   string
	debug  bool
}

// LMStudioMessage represents a message in the chat completion request
type LMStudioMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []LMStudioToolCall `json:"tool_calls,omitempty"`
}

// LMStudioToolCall represents a tool call in a message
type LMStudioToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// LMStudioChatRequest represents the request body for the LM Studio chat completions API
type LMStudioChatRequest struct {
	Model       string            `json:"model"`
	Messages    []LMStudioMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Tools       []LMStudioTool    `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"`
}

// LMStudioTool represents a tool in the chat completion request
type LMStudioTool struct {
	Type     string               `json:"type"`
	Function LMStudioToolFunction `json:"function"`
}

// LMStudioToolFunction represents a function in a tool
type LMStudioToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Strict      bool                   `json:"strict,omitempty"`
}

// LMStudioChatResponse represents the response from the LM Studio chat completions API
type LMStudioChatResponse struct {
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
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
func NewLMStudioModel(modelName string, debug bool) (*LMStudioModel, error) {
	apiURL := "http://localhost:1234/v1/chat/completions"

	return &LMStudioModel{
		apiURL: apiURL,
		name:   modelName,
		debug:  debug,
	}, nil
}

// Name returns the name of the model
func (m *LMStudioModel) Name() string {
	return m.name
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

	if m.debug {
		body, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("DEBUG: LMStudio HTTP Response: %s\n", string(body))
	}

	var chatResp LMStudioChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	if m.debug {
		fmt.Printf("DEBUG: LMStudio decoded choices: %v\n", chatResp)
	}
	// Trim any leading/trailing whitespace
	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// QueryWithTools sends a prompt to the LM Studio model with available tools and returns the response
func (m *LMStudioModel) QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error) {
	// Get default config for task queries
	config := defaultLLMConfig()

	// Build messages array with the proper role structure
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

	// Convert tools to LMStudioTool format
	lmStudioTools := make([]LMStudioTool, len(tools))
	for i, tool := range tools {
		lmStudioTools[i] = LMStudioTool{
			Type: "function",
			Function: LMStudioToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
				Strict:      true,
			},
		}
	}

	reqBody := LMStudioChatRequest{
		Model:       m.name,
		Messages:    messages,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		Stop:        config.StopSequences,
		Tools:       lmStudioTools,
		ToolChoice:  "auto", // Let the model decide whether to use tools
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	if m.debug {
		fmt.Printf("DEBUG: LMStudio HTTP Request: %s\n", string(reqBytes))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to generate response: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	if m.debug {
		body, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("DEBUG: LMStudio HTTP Response: %s\n", string(body))
	}

	var chatResp LMStudioChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	// Check if the response contains tool calls
	if len(chatResp.Choices[0].Message.ToolCalls) > 0 {
		// Process tool calls
		var result strings.Builder
		for _, toolCall := range chatResp.Choices[0].Message.ToolCalls {
			// Find the tool
			var selectedTool Tool
			for _, tool := range tools {
				if tool.Name() == toolCall.Function.Name {
					selectedTool = tool
					break
				}
			}

			if selectedTool == nil {
				result.WriteString(fmt.Sprintf("Tool '%s' not found.\n", toolCall.Function.Name))
				continue
			}

			// Execute the tool
			toolResult, err := selectedTool.Execute(ctx, toolCall.Function.Arguments)
			if err != nil {
				result.WriteString(fmt.Sprintf("Error executing tool '%s': %v\n", toolCall.Function.Name, err))
				continue
			}

			// Add the tool result to the messages
			toolCallObj := LMStudioToolCall{
				ID:   toolCall.ID,
				Type: "function",
			}
			toolCallObj.Function.Name = toolCall.Function.Name
			toolCallObj.Function.Arguments = toolCall.Function.Arguments

			messages = append(messages, LMStudioMessage{
				Role:      "assistant",
				Content:   "",
				ToolCalls: []LMStudioToolCall{toolCallObj},
			})

			messages = append(messages, LMStudioMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})

			result.WriteString(fmt.Sprintf("Tool '%s' executed with result: %s\n", toolCall.Function.Name, toolResult))
		}

		// Send a follow-up request with the tool results
		reqBody.Messages = messages
		reqBody.Tools = []LMStudioTool{} // Empty array instead of nil
		reqBody.ToolChoice = ""          // Empty string instead of nil

		reqBytes, err = json.Marshal(reqBody)
		if err != nil {
			return "", err
		}
		if m.debug {
			fmt.Printf("DEBUG: LMStudio HTTP Request: %s\n", string(reqBytes))
		}

		req, err = http.NewRequestWithContext(ctx, "POST", m.apiURL, bytes.NewBuffer(reqBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if m.debug {
			body, _ := httputil.DumpResponse(resp, true)
			fmt.Printf("DEBUG: LMStudio HTTP Response: %s\n", string(body))
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("failed to generate response: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
		}

		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			return "", err
		}

		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("no response generated")
		}
	}

	if m.debug {
		fmt.Printf("DEBUG: LMStudio decoded choices: %v\n", chatResp)
	}

	// Trim any leading/trailing whitespace
	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
