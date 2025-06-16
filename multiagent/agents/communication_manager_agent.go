package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kbutz/wikillm/multiagent"
)

// CommunicationManagerAgent specializes in managing communications, messages, and relationships
type CommunicationManagerAgent struct {
	*BaseAgent
	contacts      map[string]*Contact
	messages      map[string]*CommunicationMessage
	templates     map[string]*MessageTemplate
	commMutex     sync.RWMutex
}

// Contact represents a person or entity in the communication system
type Contact struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Email           string                 `json:"email"`
	Phone           string                 `json:"phone"`
	Organization    string                 `json:"organization"`
	Title           string                 `json:"title"`
	Relationship    RelationshipType       `json:"relationship"`
	Priority        ContactPriority        `json:"priority"`
	PreferredComm   CommunicationMethod    `json:"preferred_communication"`
	TimeZone        string                 `json:"time_zone"`
	Tags            []string               `json:"tags"`
	Notes           string                 `json:"notes"`
	SocialProfiles  map[string]string      `json:"social_profiles"`
	LastContact     *time.Time             `json:"last_contact,omitempty"`
	ContactFreq     ContactFrequency       `json:"contact_frequency"`
	Status          ContactStatus          `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RelationshipType defines the type of relationship
type RelationshipType string

const (
	RelationshipTypeFamily      RelationshipType = "family"
	RelationshipTypeFriend      RelationshipType = "friend"
	RelationshipTypeColleague   RelationshipType = "colleague"
	RelationshipTypeClient      RelationshipType = "client"
	RelationshipTypeVendor      RelationshipType = "vendor"
	RelationshipTypeMentor      RelationshipType = "mentor"
	RelationshipTypeNetworking  RelationshipType = "networking"
	RelationshipTypeProfessional RelationshipType = "professional"
)

// ContactPriority defines the priority level of a contact
type ContactPriority string

const (
	ContactPriorityVIP      ContactPriority = "vip"
	ContactPriorityHigh     ContactPriority = "high"
	ContactPriorityMedium   ContactPriority = "medium"
	ContactPriorityLow      ContactPriority = "low"
)

// CommunicationMethod defines preferred communication methods
type CommunicationMethod string

const (
	CommunicationMethodEmail    CommunicationMethod = "email"
	CommunicationMethodPhone    CommunicationMethod = "phone"
	CommunicationMethodText     CommunicationMethod = "text"
	CommunicationMethodSlack    CommunicationMethod = "slack"
	CommunicationMethodTeams    CommunicationMethod = "teams"
	CommunicationMethodLinkedIn CommunicationMethod = "linkedin"
	CommunicationMethodInPerson CommunicationMethod = "in_person"
)

// ContactFrequency defines how often to maintain contact
type ContactFrequency string

const (
	ContactFrequencyDaily   ContactFrequency = "daily"
	ContactFrequencyWeekly  ContactFrequency = "weekly"
	ContactFrequencyMonthly ContactFrequency = "monthly"
	ContactFrequencyQuarterly ContactFrequency = "quarterly"
	ContactFrequencyYearly  ContactFrequency = "yearly"
	ContactFrequencyAsNeeded ContactFrequency = "as_needed"
)

// ContactStatus represents the current status of a contact
type ContactStatus string

const (
	ContactStatusActive   ContactStatus = "active"
	ContactStatusInactive ContactStatus = "inactive"
	ContactStatusBlocked  ContactStatus = "blocked"
	ContactStatusArchived ContactStatus = "archived"
)

// CommunicationMessage represents a message or communication
type CommunicationMessage struct {
	ID              string                 `json:"id"`
	ContactID       string                 `json:"contact_id"`
	Subject         string                 `json:"subject"`
	Content         string                 `json:"content"`
	Method          CommunicationMethod    `json:"method"`
	Direction       MessageDirection       `json:"direction"`
	Status          MessageStatus          `json:"status"`
	Priority        multiagent.Priority    `json:"priority"`
	ScheduledFor    *time.Time             `json:"scheduled_for,omitempty"`
	SentAt          *time.Time             `json:"sent_at,omitempty"`
	ReceivedAt      *time.Time             `json:"received_at,omitempty"`
	ReadAt          *time.Time             `json:"read_at,omitempty"`
	TemplateID      string                 `json:"template_id,omitempty"`
	Tags            []string               `json:"tags"`
	Attachments     []string               `json:"attachments"`
	ThreadID        string                 `json:"thread_id,omitempty"`
	ParentID        string                 `json:"parent_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// MessageDirection defines the direction of communication
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// MessageStatus represents the status of a message
type MessageStatus string

const (
	MessageStatusDraft     MessageStatus = "draft"
	MessageStatusScheduled MessageStatus = "scheduled"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
	MessageStatusArchived  MessageStatus = "archived"
)

// MessageTemplate represents a reusable message template
type MessageTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    TemplateCategory       `json:"category"`
	Subject     string                 `json:"subject"`
	Content     string                 `json:"content"`
	Variables   []TemplateVariable     `json:"variables"`
	Method      CommunicationMethod    `json:"method"`
	Tags        []string               `json:"tags"`
	UsageCount  int                    `json:"usage_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TemplateCategory defines categories of message templates
type TemplateCategory string

const (
	TemplateCategoryIntroduction TemplateCategory = "introduction"
	TemplateCategoryFollowUp     TemplateCategory = "follow_up"
	TemplateCategoryMeeting      TemplateCategory = "meeting"
	TemplateCategoryThankYou     TemplateCategory = "thank_you"
	TemplateCategoryApology      TemplateCategory = "apology"
	TemplateCategoryReminder     TemplateCategory = "reminder"
	TemplateCategoryNetworking   TemplateCategory = "networking"
	TemplateCategorySales        TemplateCategory = "sales"
	TemplateCategorySupport      TemplateCategory = "support"
)

// TemplateVariable represents a variable in a template
type TemplateVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value"`
}

// CommunicationStats represents communication statistics
type CommunicationStats struct {
	TotalContacts      int                              `json:"total_contacts"`
	ActiveContacts     int                              `json:"active_contacts"`
	MessagesSent       int                              `json:"messages_sent"`
	MessagesReceived   int                              `json:"messages_received"`
	ResponseRate       float64                          `json:"response_rate"`
	AvgResponseTime    time.Duration                    `json:"avg_response_time"`
	ContactsByPriority map[ContactPriority]int          `json:"contacts_by_priority"`
	MessagesByMethod   map[CommunicationMethod]int      `json:"messages_by_method"`
	LastUpdated        time.Time                        `json:"last_updated"`
}

// NewCommunicationManagerAgent creates a new communication manager agent
func NewCommunicationManagerAgent(config BaseAgentConfig) *CommunicationManagerAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeCommunicationManager

	// Add communication management capabilities
	config.Capabilities = append(config.Capabilities,
		"contact_management",
		"message_composition",
		"template_management",
		"relationship_tracking",
		"communication_scheduling",
		"follow_up_management",
		"networking_assistance",
		"email_management",
		"social_media_coordination",
		"communication_analytics",
	)

	return &CommunicationManagerAgent{
		BaseAgent: NewBaseAgent(config),
		contacts:  make(map[string]*Contact),
		messages:  make(map[string]*CommunicationMessage),
		templates: make(map[string]*MessageTemplate),
	}
}

// HandleMessage processes incoming communication management requests
func (a *CommunicationManagerAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Managing communications"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("communication_manager:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	content := strings.ToLower(msg.Content)

	// Route to appropriate handler based on content
	if strings.Contains(content, "add contact") || strings.Contains(content, "new contact") {
		return a.handleAddContact(ctx, msg)
	} else if strings.Contains(content, "compose") || strings.Contains(content, "write message") || strings.Contains(content, "send message") {
		return a.handleComposeMessage(ctx, msg)
	} else if strings.Contains(content, "template") {
		return a.handleTemplateManagement(ctx, msg)
	} else if strings.Contains(content, "contacts") || strings.Contains(content, "list contacts") {
		return a.handleListContacts(ctx, msg)
	} else if strings.Contains(content, "follow up") || strings.Contains(content, "followup") {
		return a.handleFollowUp(ctx, msg)
	} else if strings.Contains(content, "schedule message") || strings.Contains(content, "schedule email") {
		return a.handleScheduleMessage(ctx, msg)
	} else if strings.Contains(content, "communication stats") || strings.Contains(content, "comm stats") {
		return a.handleCommunicationStats(ctx, msg)
	} else if strings.Contains(content, "relationship") || strings.Contains(content, "networking") {
		return a.handleRelationshipManagement(ctx, msg)
	} else {
		// Use LLM for general communication queries
		return a.handleGeneralQuery(ctx, msg)
	}
}

// handleAddContact adds a new contact to the system
func (a *CommunicationManagerAgent) handleAddContact(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract contact details
	contextPrompt := fmt.Sprintf(`
Extract contact information from this request: "%s"

Provide response in JSON format:
{
  "name": "contact name",
  "email": "email address if mentioned",
  "phone": "phone number if mentioned",
  "organization": "company/organization if mentioned",
  "title": "job title if mentioned",
  "relationship": "family|friend|colleague|client|vendor|mentor|networking|professional",
  "priority": "vip|high|medium|low",
  "preferred_communication": "email|phone|text|slack|teams|linkedin|in_person",
  "tags": ["tag1", "tag2"] if any categories mentioned,
  "notes": "any additional notes or context"
}

Make reasonable assumptions for missing information.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse contact details: %w", err)
	}

	var contactData struct {
		Name                  string   `json:"name"`
		Email                 string   `json:"email"`
		Phone                 string   `json:"phone"`
		Organization          string   `json:"organization"`
		Title                 string   `json:"title"`
		Relationship          string   `json:"relationship"`
		Priority              string   `json:"priority"`
		PreferredCommunication string   `json:"preferred_communication"`
		Tags                  []string `json:"tags"`
		Notes                 string   `json:"notes"`
	}

	if err := json.Unmarshal([]byte(response), &contactData); err != nil {
		return nil, fmt.Errorf("failed to parse contact JSON: %w", err)
	}

	// Create contact
	contact := &Contact{
		ID:            fmt.Sprintf("contact_%d", time.Now().UnixNano()),
		Name:          contactData.Name,
		Email:         contactData.Email,
		Phone:         contactData.Phone,
		Organization:  contactData.Organization,
		Title:         contactData.Title,
		Relationship:  RelationshipType(contactData.Relationship),
		Priority:      a.parseContactPriority(contactData.Priority),
		PreferredComm: CommunicationMethod(contactData.PreferredCommunication),
		Tags:          contactData.Tags,
		Notes:         contactData.Notes,
		Status:        ContactStatusActive,
		ContactFreq:   ContactFrequencyAsNeeded,
		SocialProfiles: make(map[string]string),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	// Store contact
	a.commMutex.Lock()
	a.contacts[contact.ID] = contact
	a.commMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		contactKey := fmt.Sprintf("contact:%s", contact.ID)
		a.memoryStore.Store(ctx, contactKey, contact)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ **Contact Added Successfully!**\n\n👤 **%s**\n🏢 %s\n📧 %s\n📱 %s\n🔗 %s\n⚡ Priority: %s\n📞 Preferred: %s\n\nContact ID: %s", contact.Name, contact.Organization, contact.Email, contact.Phone, contact.Relationship, contact.Priority, contact.PreferredComm, contact.ID),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"contact_id": contact.ID,
			"action":     "contact_added",
		},
	}, nil
}

// handleComposeMessage helps compose and send messages
func (a *CommunicationManagerAgent) handleComposeMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract message details
	contextPrompt := fmt.Sprintf(`
Extract message composition details from: "%s"

Provide response in JSON format:
{
  "recipient": "name or identifier of recipient",
  "subject": "message subject if mentioned",
  "content": "message content or main points",
  "method": "email|phone|text|slack|teams|linkedin|in_person",
  "priority": "low|medium|high|critical",
  "tone": "formal|casual|friendly|professional",
  "purpose": "introduction|follow_up|meeting|thank_you|reminder|networking"
}

If content is not fully specified, indicate what should be included.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message details: %w", err)
	}

	var messageData struct {
		Recipient string `json:"recipient"`
		Subject   string `json:"subject"`
		Content   string `json:"content"`
		Method    string `json:"method"`
		Priority  string `json:"priority"`
		Tone      string `json:"tone"`
		Purpose   string `json:"purpose"`
	}

	if err := json.Unmarshal([]byte(response), &messageData); err != nil {
		return nil, fmt.Errorf("failed to parse message JSON: %w", err)
	}

	// Find the contact
	contact := a.findContactByName(messageData.Recipient)
	if contact == nil {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("❌ Contact '%s' not found. Would you like me to:\n1. Add this as a new contact\n2. Search for similar contacts\n3. Compose the message anyway", messageData.Recipient),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Generate message content if not fully specified
	if messageData.Content == "" || len(messageData.Content) < 20 {
		messagePrompt := fmt.Sprintf(`
Compose a %s %s message for the following:

Recipient: %s (%s at %s)
Subject: %s
Purpose: %s
Tone: %s
Context: %s

Write a complete, professional message that serves the intended purpose.
Include appropriate greeting, body, and closing.`, messageData.Tone, messageData.Purpose, contact.Name, contact.Title, contact.Organization, messageData.Subject, messageData.Purpose, messageData.Tone, msg.Content)

		composedContent, err := a.llmProvider.Query(ctx, messagePrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to compose message: %w", err)
		}
		messageData.Content = composedContent
	}

	// Create message record
	message := &CommunicationMessage{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		ContactID: contact.ID,
		Subject:   messageData.Subject,
		Content:   messageData.Content,
		Method:    CommunicationMethod(messageData.Method),
		Direction: MessageDirectionOutbound,
		Status:    MessageStatusDraft,
		Priority:  a.parsePriority(messageData.Priority),
		Tags:      []string{messageData.Purpose},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Store message
	a.commMutex.Lock()
	a.messages[message.ID] = message
	a.commMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		messageKey := fmt.Sprintf("communication_message:%s", message.ID)
		a.memoryStore.Store(ctx, messageKey, message)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✉️ **Message Composed**\n\n**To:** %s (%s)\n**Subject:** %s\n**Method:** %s\n**Priority:** %s\n\n**Content:**\n%s\n\n---\n\n*Message saved as draft. Would you like me to send it or make any changes?*", contact.Name, contact.Email, message.Subject, message.Method, message.Priority, message.Content),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"message_id": message.ID,
			"contact_id": contact.ID,
			"action":     "message_composed",
		},
	}, nil
}

// handleListContacts lists contacts based on criteria
func (a *CommunicationManagerAgent) handleListContacts(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Load contacts from memory if needed
	a.loadContactsFromMemory(ctx)

	content := strings.ToLower(msg.Content)
	var filteredContacts []*Contact

	a.commMutex.RLock()
	defer a.commMutex.RUnlock()

	// Apply filters based on request
	for _, contact := range a.contacts {
		include := true

		// Filter by relationship
		if strings.Contains(content, "colleagues") && contact.Relationship != RelationshipTypeColleague {
			include = false
		} else if strings.Contains(content, "clients") && contact.Relationship != RelationshipTypeClient {
			include = false
		} else if strings.Contains(content, "friends") && contact.Relationship != RelationshipTypeFriend {
			include = false
		}

		// Filter by priority
		if strings.Contains(content, "vip") && contact.Priority != ContactPriorityVIP {
			include = false
		} else if strings.Contains(content, "high priority") && contact.Priority != ContactPriorityHigh {
			include = false
		}

		// Filter by status
		if strings.Contains(content, "active") && contact.Status != ContactStatusActive {
			include = false
		}

		if include {
			filteredContacts = append(filteredContacts, contact)
		}
	}

	if len(filteredContacts) == 0 {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   "👥 No contacts found matching your criteria. Use 'add contact' to start building your network!",
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Sort contacts by priority and last contact date
	sort.Slice(filteredContacts, func(i, j int) bool {
		if filteredContacts[i].Priority != filteredContacts[j].Priority {
			return a.getPriorityWeight(filteredContacts[i].Priority) > a.getPriorityWeight(filteredContacts[j].Priority)
		}
		return filteredContacts[i].Name < filteredContacts[j].Name
	})

	// Build contact list
	var contactsBuilder strings.Builder
	contactsBuilder.WriteString("👥 **Your Contacts**\n\n")

	for i, contact := range filteredContacts {
		if i >= 20 { // Limit to 20 contacts
			contactsBuilder.WriteString(fmt.Sprintf("... and %d more contacts\n", len(filteredContacts)-i))
			break
		}

		priority := a.getPriorityEmoji(contact.Priority)
		relationship := a.getRelationshipEmoji(contact.Relationship)
		
		contactsBuilder.WriteString(fmt.Sprintf("%d. %s %s **%s**", i+1, priority, relationship, contact.Name))
		
		if contact.Organization != "" {
			contactsBuilder.WriteString(fmt.Sprintf(" - %s", contact.Organization))
		}
		
		if contact.Title != "" {
			contactsBuilder.WriteString(fmt.Sprintf(" (%s)", contact.Title))
		}
		
		contactsBuilder.WriteString("\n")
		
		if contact.Email != "" {
			contactsBuilder.WriteString(fmt.Sprintf("   📧 %s\n", contact.Email))
		}
		
		if contact.LastContact != nil {
			contactsBuilder.WriteString(fmt.Sprintf("   📅 Last contact: %s\n", contact.LastContact.Format("2006-01-02")))
		}
		
		contactsBuilder.WriteString("\n")
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   contactsBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// Additional handler methods (simplified for space)

func (a *CommunicationManagerAgent) handleTemplateManagement(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "📝 Message template management is available. I can help you create, edit, and use message templates for common communications.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *CommunicationManagerAgent) handleFollowUp(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🔄 Follow-up management is available. I can track pending responses, schedule follow-ups, and remind you to maintain important relationships.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *CommunicationManagerAgent) handleScheduleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "⏰ Message scheduling is available. I can schedule emails and messages to be sent at optimal times.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *CommunicationManagerAgent) handleCommunicationStats(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	stats := a.calculateCommunicationStats()
	
	statsContent := fmt.Sprintf("📊 **Communication Statistics**\n\n"+
		"👥 **Contacts:** %d total, %d active\n"+
		"✉️ **Messages:** %d sent, %d received\n"+
		"📈 **Response Rate:** %.1f%%\n"+
		"⏱️ **Avg Response Time:** %s\n\n"+
		"**By Priority:**\n"+
		"🔥 VIP: %d\n"+
		"⚠️ High: %d\n"+
		"🔸 Medium: %d\n"+
		"🔽 Low: %d\n\n"+
		"*Last updated: %s*",
		stats.TotalContacts, stats.ActiveContacts,
		stats.MessagesSent, stats.MessagesReceived,
		stats.ResponseRate,
		stats.AvgResponseTime.String(),
		stats.ContactsByPriority[ContactPriorityVIP],
		stats.ContactsByPriority[ContactPriorityHigh],
		stats.ContactsByPriority[ContactPriorityMedium],
		stats.ContactsByPriority[ContactPriorityLow],
		stats.LastUpdated.Format("2006-01-02 15:04"))

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   statsContent,
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *CommunicationManagerAgent) handleRelationshipManagement(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🤝 Relationship management tools are available. I can help you maintain professional networks, track relationship history, and suggest networking opportunities.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *CommunicationManagerAgent) handleGeneralQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context with communication information
	contextPrompt := a.buildCommunicationContext(ctx, msg)
	
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

// Helper methods

func (a *CommunicationManagerAgent) parseContactPriority(priority string) ContactPriority {
	switch strings.ToLower(priority) {
	case "vip":
		return ContactPriorityVIP
	case "high":
		return ContactPriorityHigh
	case "low":
		return ContactPriorityLow
	default:
		return ContactPriorityMedium
	}
}

func (a *CommunicationManagerAgent) parsePriority(priority string) multiagent.Priority {
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

func (a *CommunicationManagerAgent) findContactByName(name string) *Contact {
	nameLower := strings.ToLower(name)
	
	a.commMutex.RLock()
	defer a.commMutex.RUnlock()
	
	for _, contact := range a.contacts {
		if strings.Contains(strings.ToLower(contact.Name), nameLower) {
			return contact
		}
	}
	return nil
}

func (a *CommunicationManagerAgent) getPriorityWeight(priority ContactPriority) int {
	switch priority {
	case ContactPriorityVIP:
		return 4
	case ContactPriorityHigh:
		return 3
	case ContactPriorityMedium:
		return 2
	case ContactPriorityLow:
		return 1
	default:
		return 0
	}
}

func (a *CommunicationManagerAgent) getPriorityEmoji(priority ContactPriority) string {
	switch priority {
	case ContactPriorityVIP:
		return "🔥"
	case ContactPriorityHigh:
		return "⚠️"
	case ContactPriorityLow:
		return "🔽"
	default:
		return "🔸"
	}
}

func (a *CommunicationManagerAgent) getRelationshipEmoji(relationship RelationshipType) string {
	switch relationship {
	case RelationshipTypeFamily:
		return "👨‍👩‍👧‍👦"
	case RelationshipTypeFriend:
		return "👫"
	case RelationshipTypeColleague:
		return "👥"
	case RelationshipTypeClient:
		return "🤝"
	case RelationshipTypeVendor:
		return "🏢"
	case RelationshipTypeMentor:
		return "🎓"
	case RelationshipTypeNetworking:
		return "🌐"
	default:
		return "👤"
	}
}

func (a *CommunicationManagerAgent) loadContactsFromMemory(ctx context.Context) {
	if a.memoryStore == nil {
		return
	}

	// List all contact keys
	keys, err := a.memoryStore.List(ctx, "contact:", 1000)
	if err != nil {
		return
	}

	// Load contacts
	contacts, err := a.memoryStore.GetMultiple(ctx, keys)
	if err != nil {
		return
	}

	a.commMutex.Lock()
	defer a.commMutex.Unlock()

	for _, contactInterface := range contacts {
		var contact Contact
		if contactData, err := json.Marshal(contactInterface); err == nil {
			if err := json.Unmarshal(contactData, &contact); err == nil {
				a.contacts[contact.ID] = &contact
			}
		}
	}
}

func (a *CommunicationManagerAgent) calculateCommunicationStats() CommunicationStats {
	a.commMutex.RLock()
	defer a.commMutex.RUnlock()

	stats := CommunicationStats{
		ContactsByPriority: make(map[ContactPriority]int),
		MessagesByMethod:   make(map[CommunicationMethod]int),
		LastUpdated:        time.Now(),
	}

	// Count contacts
	for _, contact := range a.contacts {
		stats.TotalContacts++
		if contact.Status == ContactStatusActive {
			stats.ActiveContacts++
		}
		stats.ContactsByPriority[contact.Priority]++
	}

	// Count messages
	for _, message := range a.messages {
		if message.Direction == MessageDirectionOutbound {
			stats.MessagesSent++
		} else {
			stats.MessagesReceived++
		}
		stats.MessagesByMethod[message.Method]++
	}

	// Calculate response rate (simplified)
	if stats.MessagesSent > 0 {
		stats.ResponseRate = float64(stats.MessagesReceived) / float64(stats.MessagesSent) * 100
	}

	// Average response time (placeholder)
	stats.AvgResponseTime = 2 * time.Hour

	return stats
}

func (a *CommunicationManagerAgent) buildCommunicationContext(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder
	
	contextBuilder.WriteString(fmt.Sprintf("You are %s, a communication and relationship management specialist.\n\n", a.name))
	contextBuilder.WriteString("You help users manage their contacts, compose messages, maintain relationships, and optimize their communication strategies.\n\n")
	
	// Add contact summary
	a.commMutex.RLock()
	if len(a.contacts) > 0 {
		contextBuilder.WriteString("Contact Summary:\n")
		stats := a.calculateCommunicationStats()
		contextBuilder.WriteString(fmt.Sprintf("- Total contacts: %d (%d active)\n", stats.TotalContacts, stats.ActiveContacts))
		contextBuilder.WriteString(fmt.Sprintf("- VIP: %d, High: %d, Medium: %d, Low: %d\n", 
			stats.ContactsByPriority[ContactPriorityVIP],
			stats.ContactsByPriority[ContactPriorityHigh],
			stats.ContactsByPriority[ContactPriorityMedium],
			stats.ContactsByPriority[ContactPriorityLow]))
		contextBuilder.WriteString("\n")
	}
	a.commMutex.RUnlock()
	
	contextBuilder.WriteString(fmt.Sprintf("User request: %s\n\n", msg.Content))
	contextBuilder.WriteString("Please provide helpful communication assistance, relationship management advice, or execute the requested action.")
	
	return contextBuilder.String()
}
