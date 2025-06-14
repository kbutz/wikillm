package models

import (
	"context"
	"fmt"
)

// MemoryEnabledModelWrapper wraps an LLMModel with memory-aware system prompts
type MemoryEnabledModelWrapper struct {
	baseModel    LLMModel
	systemPrompt string
}

// NewMemoryEnabledModel creates a new memory-enabled model wrapper
func NewMemoryEnabledModel(modelName, provider string, debug bool) (LLMModel, error) {
	// Create base model
	baseModel, err := New(modelName, provider, debug)
	if err != nil {
		return nil, err
	}

	// Wrap with memory-enabled system prompt
	return &MemoryEnabledModelWrapper{
		baseModel:    baseModel,
		systemPrompt: EnhancedMemorySystemPrompt(),
	}, nil
}

// Name returns the name of the model
func (m *MemoryEnabledModelWrapper) Name() string {
	return m.baseModel.Name() + " (Memory-Enhanced)"
}

// Query sends a prompt to the model with memory-enhanced system prompt
func (m *MemoryEnabledModelWrapper) Query(ctx context.Context, prompt string) (string, error) {
	// Prepend system prompt context
	enhancedPrompt := fmt.Sprintf("System Context: %s\n\nUser Query: %s", m.systemPrompt, prompt)
	return m.baseModel.Query(ctx, enhancedPrompt)
}

// QueryWithTools sends a prompt to the model with tools and memory-enhanced system prompt
func (m *MemoryEnabledModelWrapper) QueryWithTools(ctx context.Context, prompt string, tools []Tool) (string, error) {
	// Check if enhanced_memory tool is available
	hasMemoryTool := false
	for _, tool := range tools {
		if tool.Name() == "enhanced_memory" {
			hasMemoryTool = true
			break
		}
	}

	if hasMemoryTool {
		// Add memory instructions to the prompt
		enhancedPrompt := fmt.Sprintf(`Before responding, check if there's relevant context in memory using the enhanced_memory tool.
If the user mentions information worth storing (projects, preferences, decisions, tasks), use the enhanced_memory tool to store it.

User Query: %s`, prompt)

		// For LMStudio, we need to modify the system prompt in the request
		// This is a bit hacky but works with the current implementation
		if lmStudio, ok := m.baseModel.(*LMStudioModel); ok {
			// Create a new LMStudioModel with enhanced system prompt
			enhancedModel := &LMStudioModel{
				apiURL: lmStudio.apiURL,
				name:   lmStudio.name,
				debug:  lmStudio.debug,
			}

			// Override getRequest to use enhanced system prompt
			request := enhancedModel.getRequestWithSystemPrompt(enhancedPrompt, tools, m.systemPrompt)

			// Send the request
			var lastResponse string
			for i := 0; i < 10; i++ {
				response, err := enhancedModel.sendRequest(ctx, request)
				if err != nil {
					return "", err
				}

				if len(response.Choices[0].Message.ToolCalls) == 0 {
					return response.Choices[0].Message.Content, nil
				}

				// Process tool calls
				for _, toolCall := range response.Choices[0].Message.ToolCalls {
					var selectedTool Tool
					for _, tool := range tools {
						if tool.Name() == toolCall.Function.Name {
							selectedTool = tool
							break
						}
					}

					if selectedTool == nil {
						continue
					}

					toolResult, err := selectedTool.Execute(ctx, toolCall.Function.Arguments)
					if err != nil {
						continue
					}

					// Add tool result to messages
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

					lastResponse = response.Choices[0].Message.Content
				}
			}

			return lastResponse, nil
		}

		// For other models, use the enhanced prompt
		return m.baseModel.QueryWithTools(ctx, enhancedPrompt, tools)
	}

	// No memory tool, use base model
	return m.baseModel.QueryWithTools(ctx, prompt, tools)
}

// Add this method to LMStudioModel
func (m *LMStudioModel) getRequestWithSystemPrompt(prompt string, tools []Tool, systemPrompt string) LMStudioChatRequest {
	// Build messages array with custom system prompt
	messages := []LMStudioMessage{
		{
			Role:    "system",
			Content: systemPrompt,
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

	config := defaultLLMConfig()

	return LMStudioChatRequest{
		Model:       m.name,
		Messages:    messages,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
		Stop:        config.StopSequences,
		Tools:       lmStudioTools,
	}
}
