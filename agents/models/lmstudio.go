package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
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
	return m.QueryWithTools(ctx, prompt, nil)
}

// QueryWithTools sends a prompt to the LM Studio model with available tools and returns the response
func (m *LMStudioModel) QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error) {
	// Formats the request for LM Studio with tools, config defaults and system prompt
	request := m.getRequest(prompt, tools)

	// Posts the request to the LM Studio API until there are no tool calls or a maximum number of attempts is reached
	// TODO: The max of 10 is just a placeholder to avoid an infinite loop, but we should implement a better way to handle this
	for i := range 10 {
		if m.debug {
			fmt.Printf("DEBUG: LMStudio sending request, iteration #: %v\n", i)
		}

		response, err := m.sendRequest(ctx, request)
		if err != nil {
			return "", fmt.Errorf("error sending request: %w", err)
		}

		if len(response.Choices[0].Message.ToolCalls) == 0 {
			// No tool calls, return the content directly
			if m.debug {
				fmt.Printf("DEBUG: No tool calls found in response, returning content now: %s\n", response.Choices[0].Message.Content)
			}
			return response.Choices[0].Message.Content, nil
		}

		// Process tool calls and add them to the conversation in the messages slice
		for _, toolCall := range response.Choices[0].Message.ToolCalls {
			if m.debug {
				fmt.Printf("DEBUG: Processing tool call: %s with arguments: %s\n", toolCall.Function.Name, toolCall.Function.Arguments)
			}
			// Find the tool
			var selectedTool Tool
			for _, tool := range tools {
				if tool.Name() == toolCall.Function.Name {
					selectedTool = tool
					break
				}
			}

			if selectedTool == nil {
				// TODO: Need to tell the conversation that this tool is not available so it doesn't try again
				fmt.Printf("Tool '%s' not found.\n", toolCall.Function.Name)
				continue
			}

			// Execute the tool
			toolResult, err := selectedTool.Execute(ctx, toolCall.Function.Arguments)
			if err != nil {
				// TODO: Need to tell the conversation that this tool failed so we don't try again
				fmt.Printf("Error executing tool '%s': %v\n", toolCall.Function.Name, err)
				continue
			}

			// Add the tool result to the messages
			toolCallObj := LMStudioToolCall{
				ID:   toolCall.ID,
				Type: "function",
			}
			toolCallObj.Function.Name = toolCall.Function.Name
			toolCallObj.Function.Arguments = toolCall.Function.Arguments

			request.Messages = append(request.Messages, LMStudioMessage{
				Role:      "assistant",
				Content:   "",
				ToolCalls: []LMStudioToolCall{toolCallObj},
			})

			request.Messages = append(request.Messages, LMStudioMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})

			if m.debug {
				fmt.Printf("Tool '%s' executed with result: %s\n", toolCall.Function.Name, toolResult)
			}
		}

	}

	return "", fmt.Errorf("no valid response after multiple attempts")
}

func (m *LMStudioModel) getRequest(prompt string, tools []Tool) LMStudioChatRequest {
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

	return LMStudioChatRequest{
		Model:       m.name,
		Messages:    messages,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		Stop:        config.StopSequences,
		Tools:       lmStudioTools,
	}
}

func (m *LMStudioModel) sendRequest(ctx context.Context, request LMStudioChatRequest) (LMStudioChatResponse, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return LMStudioChatResponse{}, err
	}
	if m.debug {
		fmt.Printf("DEBUG: LMStudio HTTP Request: %s\n", string(reqBytes))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.apiURL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return LMStudioChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return LMStudioChatResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return LMStudioChatResponse{}, fmt.Errorf("failed to generate response: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	if m.debug {
		body, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("DEBUG: LMStudio HTTP Response: %s\n", string(body))
	}

	var chatResp LMStudioChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return LMStudioChatResponse{}, err
	}

	if len(chatResp.Choices) == 0 {
		return chatResp, fmt.Errorf("no response generated")
	}

	return chatResp, nil
}
