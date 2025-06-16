package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// ResearchAssistantAgent specializes in information gathering, research, and knowledge synthesis
type ResearchAssistantAgent struct {
	*BaseAgent
	activeResearch map[string]*ResearchSession
	researchMutex  sync.RWMutex
}

// ResearchSession represents an ongoing research session
type ResearchSession struct {
	ID           string                     `json:"id"`
	Topic        string                     `json:"topic"`
	Query        string                     `json:"query"`
	Status       ResearchStatus             `json:"status"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	Sources      []ResearchSource           `json:"sources"`
	Findings     []ResearchFinding          `json:"findings"`
	Summary      string                     `json:"summary"`
	Tags         []string                   `json:"tags"`
	Priority     multiagent.Priority        `json:"priority"`
	Deadline     *time.Time                 `json:"deadline,omitempty"`
	RequestedBy  multiagent.AgentID         `json:"requested_by"`
	Methodology  ResearchMethodology        `json:"methodology"`
	Scope        ResearchScope              `json:"scope"`
	Metadata     map[string]interface{}     `json:"metadata"`
}

// ResearchStatus represents the status of a research session
type ResearchStatus string

const (
	ResearchStatusInitiated  ResearchStatus = "initiated"
	ResearchStatusInProgress ResearchStatus = "in_progress"
	ResearchStatusAnalyzing  ResearchStatus = "analyzing"
	ResearchStatusCompleted  ResearchStatus = "completed"
	ResearchStatusOnHold     ResearchStatus = "on_hold"
	ResearchStatusCancelled  ResearchStatus = "cancelled"
)

// ResearchSource represents a source of information
type ResearchSource struct {
	ID          string                 `json:"id"`
	Type        SourceType             `json:"type"`
	Title       string                 `json:"title"`
	URL         string                 `json:"url,omitempty"`
	Author      string                 `json:"author,omitempty"`
	PublishedAt *time.Time             `json:"published_at,omitempty"`
	AccessedAt  time.Time              `json:"accessed_at"`
	Reliability float64                `json:"reliability"` // 0-1 scale
	Relevance   float64                `json:"relevance"`   // 0-1 scale
	Summary     string                 `json:"summary"`
	KeyPoints   []string               `json:"key_points"`
	Citations   []string               `json:"citations"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SourceType defines different types of research sources
type SourceType string

const (
	SourceTypeWeb         SourceType = "web"
	SourceTypeAcademic    SourceType = "academic"
	SourceTypeBook        SourceType = "book"
	SourceTypeArticle     SourceType = "article"
	SourceTypeReport      SourceType = "report"
	SourceTypeInterview   SourceType = "interview"
	SourceTypeExpert      SourceType = "expert"
	SourceTypeDatabase    SourceType = "database"
	SourceTypeInternal    SourceType = "internal"
)

// ResearchFinding represents a key finding from research
type ResearchFinding struct {
	ID          string                 `json:"id"`
	Topic       string                 `json:"topic"`
	Finding     string                 `json:"finding"`
	Evidence    []string               `json:"evidence"`
	Confidence  float64                `json:"confidence"` // 0-1 scale
	Importance  float64                `json:"importance"` // 0-1 scale
	Sources     []string               `json:"sources"`    // Source IDs
	Tags        []string               `json:"tags"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ResearchMethodology defines the approach to research
type ResearchMethodology struct {
	Type        MethodologyType `json:"type"`
	Depth       ResearchDepth   `json:"depth"`
	TimeLimit   time.Duration   `json:"time_limit"`
	SourceLimit int             `json:"source_limit"`
	Focus       []string        `json:"focus"`
	Exclude     []string        `json:"exclude"`
}

// MethodologyType defines different research methodologies
type MethodologyType string

const (
	MethodologyComprehensive MethodologyType = "comprehensive"
	MethodologyQuick         MethodologyType = "quick"
	MethodologyDeep          MethodologyType = "deep"
	MethodologyComparative   MethodologyType = "comparative"
	MethodologyFactual       MethodologyType = "factual"
)

// ResearchDepth defines how deep the research should go
type ResearchDepth string

const (
	ResearchDepthSurface ResearchDepth = "surface"
	ResearchDepthMedium  ResearchDepth = "medium"
	ResearchDepthDeep    ResearchDepth = "deep"
	ResearchDepthExpert  ResearchDepth = "expert"
)

// ResearchScope defines what areas to cover
type ResearchScope struct {
	Areas       []string  `json:"areas"`
	TimeRange   TimeRange `json:"time_range,omitempty"`
	Geographic  []string  `json:"geographic,omitempty"`
	Languages   []string  `json:"languages,omitempty"`
	SourceTypes []SourceType `json:"source_types,omitempty"`
}

// TimeRange defines a time period for research
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// NewResearchAssistantAgent creates a new research assistant agent
func NewResearchAssistantAgent(config BaseAgentConfig) *ResearchAssistantAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeResearch

	// Add research-specific capabilities
	config.Capabilities = append(config.Capabilities,
		"information_gathering",
		"source_evaluation",
		"fact_checking",
		"knowledge_synthesis",
		"research_methodology",
		"citation_management",
		"trend_analysis",
		"competitive_intelligence",
		"market_research",
		"academic_research",
	)

	return &ResearchAssistantAgent{
		BaseAgent:      NewBaseAgent(config),
		activeResearch: make(map[string]*ResearchSession),
	}
}

// HandleMessage processes incoming research requests
func (a *ResearchAssistantAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Conducting research"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("research_assistant:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	content := strings.ToLower(msg.Content)

	// Route to appropriate handler based on content
	if strings.Contains(content, "research") || strings.Contains(content, "find information") || strings.Contains(content, "look up") {
		return a.handleResearchRequest(ctx, msg)
	} else if strings.Contains(content, "fact check") || strings.Contains(content, "verify") {
		return a.handleFactCheck(ctx, msg)
	} else if strings.Contains(content, "summarize") || strings.Contains(content, "summary") {
		return a.handleSummarize(ctx, msg)
	} else if strings.Contains(content, "compare") || strings.Contains(content, "comparison") {
		return a.handleComparison(ctx, msg)
	} else if strings.Contains(content, "trends") || strings.Contains(content, "analysis") {
		return a.handleTrendAnalysis(ctx, msg)
	} else if strings.Contains(content, "sources") || strings.Contains(content, "references") {
		return a.handleSourceManagement(ctx, msg)
	} else {
		// Use LLM for general research queries
		return a.handleGeneralQuery(ctx, msg)
	}
}

// handleResearchRequest processes a research request
func (a *ResearchAssistantAgent) handleResearchRequest(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract research parameters
	contextPrompt := fmt.Sprintf(`
Extract research parameters from this request: "%s"

Provide response in JSON format:
{
  "topic": "main research topic",
  "query": "specific research question",
  "methodology": "comprehensive|quick|deep|comparative|factual",
  "depth": "surface|medium|deep|expert",
  "time_limit": "duration in hours if mentioned, otherwise 2",
  "priority": "low|medium|high|critical",
  "focus_areas": ["area1", "area2"] if specific areas mentioned,
  "source_types": ["web", "academic", "article"] preferred source types,
  "deadline": "YYYY-MM-DD if mentioned, otherwise null"
}

Make reasonable assumptions for missing information.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse research parameters: %w", err)
	}

	var researchData struct {
		Topic       string   `json:"topic"`
		Query       string   `json:"query"`
		Methodology string   `json:"methodology"`
		Depth       string   `json:"depth"`
		TimeLimit   int      `json:"time_limit"`
		Priority    string   `json:"priority"`
		FocusAreas  []string `json:"focus_areas"`
		SourceTypes []string `json:"source_types"`
		Deadline    string   `json:"deadline"`
	}

	if err := json.Unmarshal([]byte(response), &researchData); err != nil {
		// Fallback to basic research
		researchData.Topic = msg.Content
		researchData.Query = msg.Content
		researchData.Methodology = "comprehensive"
		researchData.Depth = "medium"
		researchData.TimeLimit = 2
		researchData.Priority = "medium"
	}

	// Create research session
	session := &ResearchSession{
		ID:          fmt.Sprintf("research_%d", time.Now().UnixNano()),
		Topic:       researchData.Topic,
		Query:       researchData.Query,
		Status:      ResearchStatusInitiated,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Sources:     []ResearchSource{},
		Findings:    []ResearchFinding{},
		Tags:        []string{},
		Priority:    a.parsePriority(researchData.Priority),
		RequestedBy: msg.From,
		Methodology: ResearchMethodology{
			Type:        MethodologyType(researchData.Methodology),
			Depth:       ResearchDepth(researchData.Depth),
			TimeLimit:   time.Duration(researchData.TimeLimit) * time.Hour,
			SourceLimit: a.getSourceLimit(researchData.Methodology),
			Focus:       researchData.FocusAreas,
		},
		Scope: ResearchScope{
			Areas:       researchData.FocusAreas,
			SourceTypes: a.parseSourceTypes(researchData.SourceTypes),
		},
		Metadata: make(map[string]interface{}),
	}

	// Set deadline if provided
	if researchData.Deadline != "" {
		if deadline, err := time.Parse("2006-01-02", researchData.Deadline); err == nil {
			session.Deadline = &deadline
		}
	}

	// Store session
	a.researchMutex.Lock()
	a.activeResearch[session.ID] = session
	a.researchMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		sessionKey := fmt.Sprintf("research_session:%s", session.ID)
		a.memoryStore.Store(ctx, sessionKey, session)
	}

	// Start research process
	go a.conductResearch(ctx, session)

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("🔍 Research session '%s' started!\n\n📋 **Research Details:**\n• Topic: %s\n• Methodology: %s\n• Depth: %s\n• Time Limit: %v\n• Priority: %s\n\nI'll begin gathering information and will provide updates as I find relevant sources and insights.", session.Topic, session.ID, session.Methodology.Type, session.Methodology.Depth, session.Methodology.TimeLimit, session.Priority),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"research_session_id": session.ID,
			"action":              "research_started",
		},
	}, nil
}

// handleFactCheck processes fact-checking requests
func (a *ResearchAssistantAgent) handleFactCheck(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Extract claims to verify
	contextPrompt := fmt.Sprintf(`
Identify factual claims to verify from: "%s"

Provide response in JSON format:
{
  "claims": [
    {
      "claim": "specific factual claim",
      "category": "statistic|date|name|event|definition|etc",
      "importance": "high|medium|low"
    }
  ],
  "context": "additional context for verification"
}`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fact-check request: %w", err)
	}

	var factCheckData struct {
		Claims []struct {
			Claim      string `json:"claim"`
			Category   string `json:"category"`
			Importance string `json:"importance"`
		} `json:"claims"`
		Context string `json:"context"`
	}

	if err := json.Unmarshal([]byte(response), &factCheckData); err != nil {
		return nil, fmt.Errorf("failed to parse fact-check JSON: %w", err)
	}

	// Create fact-check session
	session := &ResearchSession{
		ID:          fmt.Sprintf("factcheck_%d", time.Now().UnixNano()),
		Topic:       "Fact Verification",
		Query:       msg.Content,
		Status:      ResearchStatusInProgress,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Sources:     []ResearchSource{},
		Findings:    []ResearchFinding{},
		Tags:        []string{"fact-check"},
		Priority:    multiagent.PriorityHigh,
		RequestedBy: msg.From,
		Methodology: ResearchMethodology{
			Type:        MethodologyFactual,
			Depth:       ResearchDepthMedium,
			TimeLimit:   30 * time.Minute,
			SourceLimit: 10,
		},
		Metadata: map[string]interface{}{
			"claims":         factCheckData.Claims,
			"original_text":  msg.Content,
			"context":        factCheckData.Context,
		},
	}

	// Store session
	a.researchMutex.Lock()
	a.activeResearch[session.ID] = session
	a.researchMutex.Unlock()

	// Perform fact-checking using LLM
	factCheckPrompt := fmt.Sprintf(`
You are a fact-checking specialist. Verify the following claims based on your knowledge:

Original text: "%s"

Claims to verify:
%s

For each claim, provide:
1. Verification status (TRUE/FALSE/PARTIALLY TRUE/UNVERIFIED)
2. Explanation with reasoning
3. Confidence level (0-100%)
4. Suggested sources for verification

Format your response clearly for each claim.`, msg.Content, a.formatClaimsForPrompt(factCheckData.Claims))

	factCheckResult, err := a.llmProvider.Query(ctx, factCheckPrompt)
	if err != nil {
		return nil, fmt.Errorf("fact-check analysis failed: %w", err)
	}

	// Update session with results
	session.Status = ResearchStatusCompleted
	session.Summary = factCheckResult
	session.UpdatedAt = time.Now()

	// Save updated session
	if a.memoryStore != nil {
		sessionKey := fmt.Sprintf("research_session:%s", session.ID)
		a.memoryStore.Store(ctx, sessionKey, session)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ **Fact-Check Results**\n\n%s\n\n---\n\n*Note: This analysis is based on my training data. For critical decisions, please verify with authoritative sources.*", factCheckResult),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"research_session_id": session.ID,
			"action":              "fact_check_completed",
		},
	}, nil
}

// handleSummarize creates summaries of research or content
func (a *ResearchAssistantAgent) handleSummarize(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to create summary
	summaryPrompt := fmt.Sprintf(`
Create a comprehensive summary of the following content: "%s"

Provide:
1. Executive Summary (2-3 sentences)
2. Key Points (bullet points)
3. Important Details
4. Conclusions/Takeaways

Structure your response clearly with headers.`, msg.Content)

	summary, err := a.llmProvider.Query(ctx, summaryPrompt)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	// Store summary session
	session := &ResearchSession{
		ID:          fmt.Sprintf("summary_%d", time.Now().UnixNano()),
		Topic:       "Content Summary",
		Query:       msg.Content,
		Status:      ResearchStatusCompleted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Summary:     summary,
		Tags:        []string{"summary"},
		Priority:    multiagent.PriorityMedium,
		RequestedBy: msg.From,
		Methodology: ResearchMethodology{
			Type:      MethodologyQuick,
			Depth:     ResearchDepthMedium,
			TimeLimit: 15 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"original_content_length": len(msg.Content),
			"summary_length":          len(summary),
		},
	}

	if a.memoryStore != nil {
		sessionKey := fmt.Sprintf("research_session:%s", session.ID)
		a.memoryStore.Store(ctx, sessionKey, session)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("📝 **Content Summary**\n\n%s", summary),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"research_session_id": session.ID,
			"action":              "summary_completed",
		},
	}, nil
}

// handleComparison performs comparative analysis
func (a *ResearchAssistantAgent) handleComparison(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract comparison elements
	comparisonPrompt := fmt.Sprintf(`
Perform a comparative analysis based on: "%s"

Extract what is being compared and provide a structured comparison including:
1. Items being compared
2. Comparison criteria
3. Detailed comparison
4. Pros and cons for each
5. Recommendations or conclusions

Present the analysis in a clear, structured format.`, msg.Content)

	comparison, err := a.llmProvider.Query(ctx, comparisonPrompt)
	if err != nil {
		return nil, fmt.Errorf("comparison analysis failed: %w", err)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("⚖️ **Comparative Analysis**\n\n%s", comparison),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleTrendAnalysis analyzes trends and patterns
func (a *ResearchAssistantAgent) handleTrendAnalysis(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	trendPrompt := fmt.Sprintf(`
Analyze trends and patterns from: "%s"

Provide:
1. Current trends identified
2. Historical context
3. Future projections
4. Key drivers
5. Potential impacts
6. Recommendations

Structure your analysis with clear sections.`, msg.Content)

	analysis, err := a.llmProvider.Query(ctx, trendPrompt)
	if err != nil {
		return nil, fmt.Errorf("trend analysis failed: %w", err)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("📈 **Trend Analysis**\n\n%s", analysis),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleSourceManagement manages research sources
func (a *ResearchAssistantAgent) handleSourceManagement(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "📚 Source management functionality is available. I can help you organize, evaluate, and cite research sources.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleGeneralQuery handles general research questions
func (a *ResearchAssistantAgent) handleGeneralQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context with research capabilities
	contextPrompt := a.buildResearchContext(ctx, msg)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   response,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// conductResearch performs the actual research process
func (a *ResearchAssistantAgent) conductResearch(ctx context.Context, session *ResearchSession) {
	// Update status
	a.researchMutex.Lock()
	session.Status = ResearchStatusInProgress
	session.UpdatedAt = time.Now()
	a.researchMutex.Unlock()

	// Use LLM to conduct research based on methodology
	researchPrompt := fmt.Sprintf(`
Conduct research on: "%s"

Research parameters:
- Methodology: %s
- Depth: %s
- Focus areas: %v
- Time limit: %v

Provide a comprehensive research report including:
1. Executive Summary
2. Key Findings (with confidence levels)
3. Supporting Evidence
4. Potential Sources to Verify
5. Areas for Further Research
6. Conclusions and Insights

Structure your response professionally.`, session.Query, session.Methodology.Type, session.Methodology.Depth, session.Scope.Areas, session.Methodology.TimeLimit)

	researchResult, err := a.llmProvider.Query(ctx, researchPrompt)
	if err != nil {
		// Mark as failed
		a.researchMutex.Lock()
		session.Status = ResearchStatusCancelled
		session.UpdatedAt = time.Now()
		session.Metadata["error"] = err.Error()
		a.researchMutex.Unlock()
		return
	}

	// Update session with results
	a.researchMutex.Lock()
	session.Status = ResearchStatusCompleted
	session.Summary = researchResult
	session.UpdatedAt = time.Now()
	a.researchMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		sessionKey := fmt.Sprintf("research_session:%s", session.ID)
		a.memoryStore.Store(ctx, sessionKey, session)

		// Send completion notification
		completionMsg := &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{session.RequestedBy},
			Type:      multiagent.MessageTypeNotification,
			Content:   fmt.Sprintf("🔍 **Research Completed: %s**\n\n%s", session.Topic, researchResult),
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"research_session_id": session.ID,
				"action":              "research_completed",
			},
		}

		// Send through orchestrator if available
		if a.orchestrator != nil {
			a.orchestrator.RouteMessage(ctx, completionMsg)
		}
	}
}

// Helper methods

func (a *ResearchAssistantAgent) parsePriority(priority string) multiagent.Priority {
	switch strings.ToLower(priority) {
	case "critical":
		return multiagent.PriorityCritical
	case "high":
		return multiagent.PriorityHigh
	case "low":
		return multiagent.PriorityLow
	default:
		return multiagent.PriorityMedium
	}
}

func (a *ResearchAssistantAgent) getSourceLimit(methodology string) int {
	switch methodology {
	case "quick":
		return 5
	case "deep":
		return 25
	case "comprehensive":
		return 50
	case "expert":
		return 100
	default:
		return 15
	}
}

func (a *ResearchAssistantAgent) parseSourceTypes(sourceTypes []string) []SourceType {
	var types []SourceType
	for _, st := range sourceTypes {
		types = append(types, SourceType(st))
	}
	if len(types) == 0 {
		types = []SourceType{SourceTypeWeb, SourceTypeArticle, SourceTypeAcademic}
	}
	return types
}

func (a *ResearchAssistantAgent) formatClaimsForPrompt(claims []struct {
	Claim      string `json:"claim"`
	Category   string `json:"category"`
	Importance string `json:"importance"`
}) string {
	var formatted strings.Builder
	for i, claim := range claims {
		formatted.WriteString(fmt.Sprintf("%d. %s (Category: %s, Importance: %s)\n", i+1, claim.Claim, claim.Category, claim.Importance))
	}
	return formatted.String()
}

func (a *ResearchAssistantAgent) buildResearchContext(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(fmt.Sprintf("You are %s, a research assistant specialist.\n\n", a.name))
	contextBuilder.WriteString("You help users gather information, verify facts, analyze trends, and synthesize knowledge from various sources.\n\n")

	// Add active research sessions summary
	a.researchMutex.RLock()
	if len(a.activeResearch) > 0 {
		contextBuilder.WriteString("Active Research Sessions:\n")
		for _, session := range a.activeResearch {
			contextBuilder.WriteString(fmt.Sprintf("- %s (%s) - %s\n", session.Topic, session.Status, session.Methodology.Type))
		}
		contextBuilder.WriteString("\n")
	}
	a.researchMutex.RUnlock()

	contextBuilder.WriteString(fmt.Sprintf("User request: %s\n\n", msg.Content))
	contextBuilder.WriteString("Please provide helpful research assistance, information, or analysis as requested.")

	return contextBuilder.String()
}
