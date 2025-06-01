package main

import (
	"regexp"
	"strings"
)

// ResponseFilter filters and cleans LLM responses
type ResponseFilter struct {
	// Patterns that indicate internal reasoning
	reasoningPatterns []string
	// Patterns that indicate the start of the actual response
	responseMarkers []string
}

// NewResponseFilter creates a new response filter
func NewResponseFilter() *ResponseFilter {
	return &ResponseFilter{
		reasoningPatterns: []string{
			"(?i)i should",
			"(?i)i need to",
			"(?i)let me",
			"(?i)i think",
			"(?i)i'll",
			"(?i)now that i've",
			"(?i)based on the information",
			"(?i)that's all i have to say",
			"(?i)i've analyzed",
			"(?i)based on the analysis",
			"(?i)wait,",
			"(?i)okay,",
			"(?i)yes, that's",
			"(?i)the user",
			"(?i)according to",
			"(?i)but the user",
			"(?i)the response should",
			"(?i)i'll mention",
			"(?i)make sure",
			"(?i)so the",
		},
		responseMarkers: []string{
			"**Answer:**",
			"Answer:",
			"Response:",
			"---",
		},
	}
}

// FilterResponse removes internal reasoning and duplicates from LLM responses
func (rf *ResponseFilter) FilterResponse(response string) string {
	// First, extract content after response markers if present
	for _, marker := range rf.responseMarkers {
		if idx := strings.Index(response, marker); idx != -1 {
			response = response[idx+len(marker):]
			break
		}
	}

	// Remove code blocks if present
	if strings.Contains(response, "```") {
		parts := strings.Split(response, "```")
		if len(parts) >= 3 {
			// Take the content after the last code block
			response = parts[len(parts)-1]
		}
	}

	// Split into lines for processing
	lines := strings.Split(response, "\n")
	var filteredLines []string
	seenContent := make(map[string]bool)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		// Skip lines matching reasoning patterns
		isReasoning := false
		for _, pattern := range rf.reasoningPatterns {
			if matched, _ := regexp.MatchString(pattern, trimmedLine); matched {
				isReasoning = true
				break
			}
		}
		if isReasoning {
			continue
		}

		// Skip duplicate content
		normalizedLine := strings.ToLower(strings.ReplaceAll(trimmedLine, " ", ""))
		if seenContent[normalizedLine] {
			continue
		}
		seenContent[normalizedLine] = true

		filteredLines = append(filteredLines, trimmedLine)
	}

	// Join the filtered lines
	result := strings.Join(filteredLines, "\n")

	// Find and extract the most coherent response if multiple attempts exist
	result = rf.extractBestResponse(result)

	// Final trim
	result = strings.TrimSpace(result)

	if result == "" {
		return "I've completed the analysis of your tasks."
	}

	return result
}

// extractBestResponse finds the most complete and coherent response
func (rf *ResponseFilter) extractBestResponse(text string) string {
	// Look for complete sentences that answer the query
	sentences := strings.Split(text, ".")

	// Find the most informative sentence/paragraph
	var bestResponse strings.Builder
	foundMainContent := false

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Look for sentences that contain key information
		if strings.Contains(strings.ToLower(sentence), "most important") ||
			strings.Contains(strings.ToLower(sentence), "task is") ||
			strings.Contains(strings.ToLower(sentence), "critical") ||
			strings.Contains(strings.ToLower(sentence), "priority") ||
			strings.Contains(strings.ToLower(sentence), "you have") {

			if !foundMainContent {
				bestResponse.Reset()
				foundMainContent = true
			}
			bestResponse.WriteString(sentence)
			bestResponse.WriteString(". ")
		} else if foundMainContent {
			// Include follow-up sentences after main content
			bestResponse.WriteString(sentence)
			bestResponse.WriteString(". ")
		}
	}

	if bestResponse.Len() > 0 {
		return strings.TrimSpace(bestResponse.String())
	}

	return text
}

// LLMConfig provides configuration for LLM behavior
type LLMConfig struct {
	MaxTokens     int
	Temperature   float64
	StopSequences []string
	SystemPrompt  string
}

// DefaultLLMConfig returns default configuration for task queries
func DefaultLLMConfig() LLMConfig {
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
