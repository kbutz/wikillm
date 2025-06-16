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

// SchedulerAgent specializes in calendar management, appointment scheduling, and time planning
type SchedulerAgent struct {
	*BaseAgent
	calendar      map[string]*CalendarEvent
	schedules     map[string]*Schedule
	scheduleMutex sync.RWMutex
}

// CalendarEvent represents a scheduled event
type CalendarEvent struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	AllDay        bool                   `json:"all_day"`
	Location      string                 `json:"location"`
	Category      EventCategory          `json:"category"`
	Priority      multiagent.Priority    `json:"priority"`
	Status        EventStatus            `json:"status"`
	Attendees     []Attendee             `json:"attendees"`
	Reminders     []EventReminder        `json:"reminders"`
	Recurring     *RecurrenceRule        `json:"recurring,omitempty"`
	Tags          []string               `json:"tags"`
	Notes         string                 `json:"notes"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CreatedBy     multiagent.AgentID     `json:"created_by"`
	Timezone      string                 `json:"timezone"`
	URL           string                 `json:"url,omitempty"`
	ConferenceURL string                 `json:"conference_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// EventCategory defines different types of events
type EventCategory string

const (
	EventCategoryMeeting     EventCategory = "meeting"
	EventCategoryAppointment EventCategory = "appointment"
	EventCategoryTask        EventCategory = "task"
	EventCategoryDeadline    EventCategory = "deadline"
	EventCategoryPersonal    EventCategory = "personal"
	EventCategoryWork        EventCategory = "work"
	EventCategoryTravel      EventCategory = "travel"
	EventCategoryBreak       EventCategory = "break"
	EventCategoryFocusTime   EventCategory = "focus_time"
	EventCategoryReminder    EventCategory = "reminder"
)

// EventStatus represents the status of an event
type EventStatus string

const (
	EventStatusConfirmed  EventStatus = "confirmed"
	EventStatusTentative  EventStatus = "tentative"
	EventStatusCancelled  EventStatus = "cancelled"
	EventStatusCompleted  EventStatus = "completed"
	EventStatusPostponed  EventStatus = "postponed"
	EventStatusInProgress EventStatus = "in_progress"
)

// Attendee represents someone attending an event
type Attendee struct {
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	Role     AttendeeRole   `json:"role"`
	Status   AttendeeStatus `json:"status"`
	Response string         `json:"response,omitempty"`
	Required bool           `json:"required"`
}

// AttendeeRole defines the role of an attendee
type AttendeeRole string

const (
	AttendeeRoleOrganizer   AttendeeRole = "organizer"
	AttendeeRolePresenter   AttendeeRole = "presenter"
	AttendeeRoleParticipant AttendeeRole = "participant"
	AttendeeRoleOptional    AttendeeRole = "optional"
)

// AttendeeStatus represents attendance status
type AttendeeStatus string

const (
	AttendeeStatusAccepted  AttendeeStatus = "accepted"
	AttendeeStatusDeclined  AttendeeStatus = "declined"
	AttendeeStatusTentative AttendeeStatus = "tentative"
	AttendeeStatusPending   AttendeeStatus = "pending"
)

// EventReminder represents a reminder for an event
type EventReminder struct {
	ID       string         `json:"id"`
	Duration time.Duration  `json:"duration"` // How long before event
	Method   ReminderMethod `json:"method"`
	Message  string         `json:"message"`
	Sent     bool           `json:"sent"`
}

// ReminderMethod defines how reminders are delivered
type ReminderMethod string

const (
	ReminderMethodNotification ReminderMethod = "notification"
	ReminderMethodEmail        ReminderMethod = "email"
	ReminderMethodSMS          ReminderMethod = "sms"
	ReminderMethodPopup        ReminderMethod = "popup"
)

// RecurrenceRule defines how events repeat
type RecurrenceRule struct {
	Frequency   RecurrenceFreq `json:"frequency"`
	Interval    int            `json:"interval"`
	DaysOfWeek  []time.Weekday `json:"days_of_week,omitempty"`
	DayOfMonth  int            `json:"day_of_month,omitempty"`
	WeekOfMonth int            `json:"week_of_month,omitempty"`
	MonthOfYear int            `json:"month_of_year,omitempty"`
	EndDate     *time.Time     `json:"end_date,omitempty"`
	Count       int            `json:"count,omitempty"`
	Exceptions  []time.Time    `json:"exceptions,omitempty"`
}

// RecurrenceFreq defines frequency of recurrence
type RecurrenceFreq string

const (
	RecurrenceFreqDaily   RecurrenceFreq = "daily"
	RecurrenceFreqWeekly  RecurrenceFreq = "weekly"
	RecurrenceFreqMonthly RecurrenceFreq = "monthly"
	RecurrenceFreqYearly  RecurrenceFreq = "yearly"
)

// Schedule represents a person's schedule template
type Schedule struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Owner             string                 `json:"owner"`
	Timezone          string                 `json:"timezone"`
	WorkingHours      WorkingHours           `json:"working_hours"`
	AvailabilityRules []AvailabilityRule     `json:"availability_rules"`
	BlockedTimes      []TimeBlock            `json:"blocked_times"`
	Preferences       SchedulePreferences    `json:"preferences"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// WorkingHours defines typical working hours for each day
type WorkingHours struct {
	Monday    DaySchedule `json:"monday"`
	Tuesday   DaySchedule `json:"tuesday"`
	Wednesday DaySchedule `json:"wednesday"`
	Thursday  DaySchedule `json:"thursday"`
	Friday    DaySchedule `json:"friday"`
	Saturday  DaySchedule `json:"saturday"`
	Sunday    DaySchedule `json:"sunday"`
}

// DaySchedule defines working hours for a specific day
type DaySchedule struct {
	IsWorkingDay bool        `json:"is_working_day"`
	StartTime    time.Time   `json:"start_time"`
	EndTime      time.Time   `json:"end_time"`
	BreakTimes   []TimeBlock `json:"break_times"`
}

// AvailabilityRule defines rules for when someone is available
type AvailabilityRule struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Priority   int                     `json:"priority"`
	Conditions []AvailabilityCondition `json:"conditions"`
	Action     AvailabilityAction      `json:"action"`
	Metadata   map[string]interface{}  `json:"metadata"`
}

// AvailabilityCondition defines conditions for availability rules
type AvailabilityCondition struct {
	Type      ConditionType `json:"type"`
	Operator  string        `json:"operator"`
	Value     interface{}   `json:"value"`
	TimeRange *ScheduleTimeRange    `json:"time_range,omitempty"`
}

// ConditionType defines types of availability conditions
type ConditionType string

const (
	ConditionTypeTime     ConditionType = "time"
	ConditionTypeDate     ConditionType = "date"
	ConditionTypeWeekday  ConditionType = "weekday"
	ConditionTypeCategory ConditionType = "category"
	ConditionTypeDuration ConditionType = "duration"
)

// AvailabilityAction defines what happens when conditions are met
type AvailabilityAction string

const (
	AvailabilityActionBlock       AvailabilityAction = "block"
	AvailabilityActionAllow       AvailabilityAction = "allow"
	AvailabilityActionPrefer      AvailabilityAction = "prefer"
	AvailabilityActionDiscouraage AvailabilityAction = "discourage"
)

// TimeBlock represents a blocked time period
type TimeBlock struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Reason    string    `json:"reason"`
	Recurring bool      `json:"recurring"`
}

// SchedulePreferences defines scheduling preferences
type SchedulePreferences struct {
	PreferredMeetingDuration time.Duration        `json:"preferred_meeting_duration"`
	BufferTime               time.Duration        `json:"buffer_time"`
	MaxMeetingsPerDay        int                  `json:"max_meetings_per_day"`
	PreferredTimeSlots       []TimeSlot           `json:"preferred_time_slots"`
	AvoidBackToBack          bool                 `json:"avoid_back_to_back"`
	FocusTimeBlocks          []TimeSlot           `json:"focus_time_blocks"`
	LunchBreak               *TimeSlot            `json:"lunch_break,omitempty"`
	Notifications            NotificationSettings `json:"notifications"`
}

// TimeSlot represents a preferred time slot
type TimeSlot struct {
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	Weekdays  []time.Weekday `json:"weekdays"`
}

// NotificationSettings defines notification preferences
type NotificationSettings struct {
	EmailReminders   bool            `json:"email_reminders"`
	PopupReminders   bool            `json:"popup_reminders"`
	SMSReminders     bool            `json:"sms_reminders"`
	DefaultReminders []time.Duration `json:"default_reminders"`
	QuietHours       *TimeSlot       `json:"quiet_hours,omitempty"`
}

// ScheduleTimeRange represents a time range with start and end
type ScheduleTimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// NewSchedulerAgent creates a new scheduler agent
func NewSchedulerAgent(config BaseAgentConfig) *SchedulerAgent {
	// Ensure the agent type is correct
	config.Type = multiagent.AgentTypeScheduler

	// Add scheduling capabilities
	config.Capabilities = append(config.Capabilities,
		"calendar_management",
		"appointment_scheduling",
		"time_planning",
		"availability_checking",
		"meeting_coordination",
		"reminder_management",
		"schedule_optimization",
		"conflict_resolution",
		"time_blocking",
		"recurring_events",
	)

	return &SchedulerAgent{
		BaseAgent: NewBaseAgent(config),
		calendar:  make(map[string]*CalendarEvent),
		schedules: make(map[string]*Schedule),
	}
}

// HandleMessage processes incoming scheduling requests
func (a *SchedulerAgent) HandleMessage(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Update state to busy
	a.mu.Lock()
	a.state.Status = multiagent.AgentStatusBusy
	a.state.CurrentTask = "Managing schedule"
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state.Status = multiagent.AgentStatusIdle
		a.state.CurrentTask = ""
		a.mu.Unlock()
	}()

	// Store message in memory
	if a.memoryStore != nil {
		msgKey := fmt.Sprintf("scheduler:%s:%s", a.id, msg.ID)
		a.memoryStore.Store(ctx, msgKey, msg)
	}

	content := strings.ToLower(msg.Content)

	// Route to appropriate handler based on content
	if strings.Contains(content, "schedule") && (strings.Contains(content, "meeting") || strings.Contains(content, "appointment")) {
		return a.handleScheduleEvent(ctx, msg)
	} else if strings.Contains(content, "availability") || strings.Contains(content, "free time") || strings.Contains(content, "available") {
		return a.handleCheckAvailability(ctx, msg)
	} else if strings.Contains(content, "cancel") && (strings.Contains(content, "meeting") || strings.Contains(content, "appointment")) {
		return a.handleCancelEvent(ctx, msg)
	} else if strings.Contains(content, "reschedule") || strings.Contains(content, "move") {
		return a.handleReschedule(ctx, msg)
	} else if strings.Contains(content, "calendar") || strings.Contains(content, "schedule") {
		return a.handleViewCalendar(ctx, msg)
	} else if strings.Contains(content, "remind") || strings.Contains(content, "reminder") {
		return a.handleSetReminder(ctx, msg)
	} else if strings.Contains(content, "block time") || strings.Contains(content, "focus time") {
		return a.handleBlockTime(ctx, msg)
	} else if strings.Contains(content, "recurring") || strings.Contains(content, "repeat") {
		return a.handleRecurringEvent(ctx, msg)
	} else {
		// Use LLM for general scheduling queries
		return a.handleGeneralQuery(ctx, msg)
	}
}

// handleScheduleEvent schedules a new event
func (a *SchedulerAgent) handleScheduleEvent(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract event details
	contextPrompt := fmt.Sprintf(`
Extract event details from this scheduling request: "%s"

Provide response in JSON format:
{
  "title": "event title",
  "description": "event description",
  "start_time": "YYYY-MM-DD HH:MM",
  "end_time": "YYYY-MM-DD HH:MM if mentioned",
  "duration": "duration in minutes if end time not specified",
  "location": "location if mentioned",
  "category": "meeting|appointment|task|personal|work|etc",
  "priority": "low|medium|high|critical",
  "attendees": ["person1", "person2"] if mentioned,
  "recurring": "daily|weekly|monthly|yearly if recurring",
  "reminders": ["15", "60"] reminder times in minutes
}

Parse dates and times carefully. If no year is specified, assume current year.
If no specific time is given, suggest appropriate time slots.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, contextPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event details: %w", err)
	}

	var eventData struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		StartTime   string   `json:"start_time"`
		EndTime     string   `json:"end_time"`
		Duration    int      `json:"duration"`
		Location    string   `json:"location"`
		Category    string   `json:"category"`
		Priority    string   `json:"priority"`
		Attendees   []string `json:"attendees"`
		Recurring   string   `json:"recurring"`
		Reminders   []string `json:"reminders"`
	}

	if err := json.Unmarshal([]byte(response), &eventData); err != nil {
		return nil, fmt.Errorf("failed to parse event JSON: %w", err)
	}

	// Parse start time
	startTime, err := time.Parse("2006-01-02 15:04", eventData.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start time format: %w", err)
	}

	// Calculate end time
	var endTime time.Time
	if eventData.EndTime != "" {
		endTime, err = time.Parse("2006-01-02 15:04", eventData.EndTime)
		if err != nil {
			endTime = startTime.Add(time.Duration(eventData.Duration) * time.Minute)
		}
	} else {
		duration := eventData.Duration
		if duration == 0 {
			duration = 60 // Default 1 hour
		}
		endTime = startTime.Add(time.Duration(duration) * time.Minute)
	}

	// Check for conflicts
	conflicts := a.checkConflicts(startTime, endTime)
	if len(conflicts) > 0 {
		conflictsList := make([]string, len(conflicts))
		for i, conflict := range conflicts {
			conflictsList[i] = fmt.Sprintf("• %s (%s - %s)", conflict.Title, conflict.StartTime.Format("15:04"), conflict.EndTime.Format("15:04"))
		}

		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("⚠️ **Scheduling Conflict Detected**\n\nThe requested time slot (%s - %s) conflicts with:\n\n%s\n\nWould you like me to:\n1. Suggest alternative times\n2. Schedule anyway\n3. Cancel the conflicting event", startTime.Format("2006-01-02 15:04"), endTime.Format("15:04"), strings.Join(conflictsList, "\n")),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
			Context: map[string]interface{}{
				"action":     "conflict_detected",
				"conflicts":  conflicts,
				"event_data": eventData,
			},
		}, nil
	}

	// Create event
	event := &CalendarEvent{
		ID:          fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Title:       eventData.Title,
		Description: eventData.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Location:    eventData.Location,
		Category:    EventCategory(eventData.Category),
		Priority:    a.parsePriority(eventData.Priority),
		Status:      EventStatusConfirmed,
		Attendees:   a.parseAttendees(eventData.Attendees),
		Reminders:   a.parseReminders(eventData.Reminders),
		Tags:        []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   msg.From,
		Timezone:    "UTC",
		Metadata:    make(map[string]interface{}),
	}

	// Set recurring pattern if specified
	if eventData.Recurring != "" {
		event.Recurring = &RecurrenceRule{
			Frequency: RecurrenceFreq(eventData.Recurring),
			Interval:  1,
		}
	}

	// Store event
	a.scheduleMutex.Lock()
	a.calendar[event.ID] = event
	a.scheduleMutex.Unlock()

	// Save to memory
	if a.memoryStore != nil {
		eventKey := fmt.Sprintf("calendar_event:%s", event.ID)
		a.memoryStore.Store(ctx, eventKey, event)
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   fmt.Sprintf("✅ **Event Scheduled Successfully!**\n\n📅 **%s**\n🕐 %s - %s\n📍 %s\n🏷️ %s\n⚡ Priority: %s\n\nEvent ID: %s", event.Title, event.StartTime.Format("2006-01-02 15:04"), event.EndTime.Format("15:04"), event.Location, event.Category, event.Priority, event.ID),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"event_id": event.ID,
			"action":   "event_scheduled",
		},
	}, nil
}

// handleCheckAvailability checks availability for a given time period
func (a *SchedulerAgent) handleCheckAvailability(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Use LLM to extract time period
	availabilityPrompt := fmt.Sprintf(`
Extract availability check details from: "%s"

Provide response in JSON format:
{
  "start_date": "YYYY-MM-DD",
  "end_date": "YYYY-MM-DD if range specified",
  "duration": "duration in minutes if specific meeting duration mentioned",
  "preferred_times": ["morning", "afternoon", "evening"] if mentioned
}

If no specific dates are given, assume they want to check today or this week.`, msg.Content)

	response, err := a.llmProvider.Query(ctx, availabilityPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse availability request: %w", err)
	}

	var availData struct {
		StartDate      string   `json:"start_date"`
		EndDate        string   `json:"end_date"`
		Duration       int      `json:"duration"`
		PreferredTimes []string `json:"preferred_times"`
	}

	if err := json.Unmarshal([]byte(response), &availData); err != nil {
		return nil, fmt.Errorf("failed to parse availability JSON: %w", err)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", availData.StartDate)
	if err != nil {
		startDate = time.Now().Truncate(24 * time.Hour)
	}

	endDate := startDate.Add(24 * time.Hour)
	if availData.EndDate != "" {
		if ed, err := time.Parse("2006-01-02", availData.EndDate); err == nil {
			endDate = ed.Add(24 * time.Hour)
		}
	}

	// Find available slots
	availableSlots := a.findAvailableSlots(startDate, endDate, time.Duration(availData.Duration)*time.Minute)

	if len(availableSlots) == 0 {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("📅 **Availability Check**\n\nNo available slots found for %s to %s.\n\nYour calendar appears to be fully booked during this period.", startDate.Format("2006-01-02"), endDate.Add(-24*time.Hour).Format("2006-01-02")),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Format available slots
	var slotsBuilder strings.Builder
	slotsBuilder.WriteString(fmt.Sprintf("📅 **Available Time Slots** (%s to %s)\n\n", startDate.Format("2006-01-02"), endDate.Add(-24*time.Hour).Format("2006-01-02")))

	for i, slot := range availableSlots {
		if i >= 10 { // Limit to 10 slots
			slotsBuilder.WriteString(fmt.Sprintf("... and %d more slots available\n", len(availableSlots)-i))
			break
		}
		slotsBuilder.WriteString(fmt.Sprintf("• %s - %s (%s)\n",
			slot.Start.Format("Mon 2006-01-02 15:04"),
			slot.End.Format("15:04"),
			(*slot.End).Sub(*slot.Start).String()))
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   slotsBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// handleViewCalendar shows calendar events
func (a *SchedulerAgent) handleViewCalendar(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Load events from memory if needed
	a.loadEventsFromMemory(ctx)

	// Determine date range
	now := time.Now()
	startDate := now.Truncate(24 * time.Hour)
	endDate := startDate.Add(7 * 24 * time.Hour) // Default to 1 week

	content := strings.ToLower(msg.Content)
	if strings.Contains(content, "today") {
		endDate = startDate.Add(24 * time.Hour)
	} else if strings.Contains(content, "week") {
		endDate = startDate.Add(7 * 24 * time.Hour)
	} else if strings.Contains(content, "month") {
		endDate = startDate.Add(30 * 24 * time.Hour)
	}

	// Get events in range
	events := a.getEventsInRange(startDate, endDate)

	if len(events) == 0 {
		return &multiagent.Message{
			ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
			From:      a.id,
			To:        []multiagent.AgentID{msg.From},
			Type:      multiagent.MessageTypeResponse,
			Content:   fmt.Sprintf("📅 **Calendar View** (%s to %s)\n\nNo events scheduled for this period.", startDate.Format("2006-01-02"), endDate.Add(-24*time.Hour).Format("2006-01-02")),
			ReplyTo:   msg.ID,
			Timestamp: time.Now(),
		}, nil
	}

	// Sort events by start time
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})

	// Build calendar view
	var calendarBuilder strings.Builder
	calendarBuilder.WriteString(fmt.Sprintf("📅 **Calendar View** (%s to %s)\n\n", startDate.Format("2006-01-02"), endDate.Add(-24*time.Hour).Format("2006-01-02")))

	currentDate := ""
	for _, event := range events {
		eventDate := event.StartTime.Format("2006-01-02")
		if eventDate != currentDate {
			if currentDate != "" {
				calendarBuilder.WriteString("\n")
			}
			calendarBuilder.WriteString(fmt.Sprintf("**%s (%s)**\n", eventDate, event.StartTime.Format("Monday")))
			currentDate = eventDate
		}

		status := a.getEventStatusEmoji(event.Status)
		priority := a.getEventPriorityEmoji(event.Priority)

		calendarBuilder.WriteString(fmt.Sprintf("  %s %s %s - %s: **%s**\n", status, priority, event.StartTime.Format("15:04"), event.EndTime.Format("15:04"), event.Title))

		if event.Location != "" {
			calendarBuilder.WriteString(fmt.Sprintf("    📍 %s\n", event.Location))
		}
	}

	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   calendarBuilder.String(),
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

// Additional handler methods

func (a *SchedulerAgent) handleCancelEvent(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "❌ Event cancellation functionality is available. Please specify which event you'd like to cancel.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *SchedulerAgent) handleReschedule(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🔄 Event rescheduling is available. Please specify which event and the new time.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *SchedulerAgent) handleSetReminder(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "⏰ Reminder functionality is available. I can set reminders for events and tasks.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *SchedulerAgent) handleBlockTime(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🔒 Time blocking functionality is available. I can block time for focused work or personal activities.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *SchedulerAgent) handleRecurringEvent(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	return &multiagent.Message{
		ID:        fmt.Sprintf("msg_%s_%d", a.id, time.Now().UnixNano()),
		From:      a.id,
		To:        []multiagent.AgentID{msg.From},
		Type:      multiagent.MessageTypeResponse,
		Content:   "🔄 Recurring event functionality is available. I can set up daily, weekly, monthly, or yearly recurring events.",
		ReplyTo:   msg.ID,
		Timestamp: time.Now(),
	}, nil
}

func (a *SchedulerAgent) handleGeneralQuery(ctx context.Context, msg *multiagent.Message) (*multiagent.Message, error) {
	// Build context with calendar information
	contextPrompt := a.buildSchedulerContext(ctx, msg)

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

func (a *SchedulerAgent) parsePriority(priority string) multiagent.Priority {
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

func (a *SchedulerAgent) parseAttendees(attendees []string) []Attendee {
	var result []Attendee
	for _, name := range attendees {
		result = append(result, Attendee{
			Name:     name,
			Role:     AttendeeRoleParticipant,
			Status:   AttendeeStatusPending,
			Required: true,
		})
	}
	return result
}

func (a *SchedulerAgent) parseReminders(reminders []string) []EventReminder {
	var result []EventReminder
	for i, reminder := range reminders {
		if duration, err := time.ParseDuration(reminder + "m"); err == nil {
			result = append(result, EventReminder{
				ID:       fmt.Sprintf("reminder_%d", i),
				Duration: duration,
				Method:   ReminderMethodNotification,
				Message:  "Event reminder",
			})
		}
	}
	return result
}

func (a *SchedulerAgent) checkConflicts(startTime, endTime time.Time) []*CalendarEvent {
	var conflicts []*CalendarEvent

	a.scheduleMutex.RLock()
	defer a.scheduleMutex.RUnlock()

	for _, event := range a.calendar {
		if event.Status == EventStatusCancelled {
			continue
		}

		// Check for overlap
		if startTime.Before(event.EndTime) && endTime.After(event.StartTime) {
			conflicts = append(conflicts, event)
		}
	}

	return conflicts
}

func (a *SchedulerAgent) findAvailableSlots(startDate, endDate time.Time, duration time.Duration) []ScheduleTimeRange {
	var slots []ScheduleTimeRange

	// Simple implementation - find gaps between events
	events := a.getEventsInRange(startDate, endDate)

	// Sort events by start time
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})

	// Working hours: 9 AM to 6 PM
	workStart := 9
	workEnd := 18

	currentDate := startDate
	for currentDate.Before(endDate) {
		// Skip weekends (simple implementation)
		if currentDate.Weekday() == time.Saturday || currentDate.Weekday() == time.Sunday {
			currentDate = currentDate.Add(24 * time.Hour)
			continue
		}

		dayStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), workStart, 0, 0, 0, currentDate.Location())
		dayEnd := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), workEnd, 0, 0, 0, currentDate.Location())

		// Find gaps in this day
		dayEvents := a.getEventsForDate(currentDate)

		if len(dayEvents) == 0 {
			// Entire day is free
			if duration == 0 || dayEnd.Sub(dayStart) >= duration {
				dayStartPtr, dayEndPtr := dayStart, dayEnd
				slots = append(slots, ScheduleTimeRange{Start: &dayStartPtr, End: &dayEndPtr})
			}
		} else {
			// Find gaps between events
			currentTime := dayStart
			for _, event := range dayEvents {
				if event.StartTime.After(currentTime) {
					gapDuration := event.StartTime.Sub(currentTime)
					if duration == 0 || gapDuration >= duration {
						currentTimePtr, eventStartTimePtr := currentTime, event.StartTime
						slots = append(slots, ScheduleTimeRange{Start: &currentTimePtr, End: &eventStartTimePtr})
					}
				}
				if event.EndTime.After(currentTime) {
					currentTime = event.EndTime
				}
			}

			// Check gap after last event
			if currentTime.Before(dayEnd) {
				gapDuration := dayEnd.Sub(currentTime)
				if duration == 0 || gapDuration >= duration {
					currentTimePtr, dayEndPtr := currentTime, dayEnd
					slots = append(slots, ScheduleTimeRange{Start: &currentTimePtr, End: &dayEndPtr})
				}
			}
		}

		currentDate = currentDate.Add(24 * time.Hour)
	}

	return slots
}

func (a *SchedulerAgent) getEventsInRange(startDate, endDate time.Time) []*CalendarEvent {
	var events []*CalendarEvent

	a.scheduleMutex.RLock()
	defer a.scheduleMutex.RUnlock()

	for _, event := range a.calendar {
		if event.Status == EventStatusCancelled {
			continue
		}

		// Check if event overlaps with date range
		if event.StartTime.Before(endDate) && event.EndTime.After(startDate) {
			events = append(events, event)
		}
	}

	return events
}

func (a *SchedulerAgent) getEventsForDate(date time.Time) []*CalendarEvent {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	return a.getEventsInRange(startOfDay, endOfDay)
}

func (a *SchedulerAgent) getEventStatusEmoji(status EventStatus) string {
	switch status {
	case EventStatusConfirmed:
		return "✅"
	case EventStatusTentative:
		return "❓"
	case EventStatusCancelled:
		return "❌"
	case EventStatusCompleted:
		return "🏁"
	case EventStatusPostponed:
		return "⏸️"
	case EventStatusInProgress:
		return "⏳"
	default:
		return "📅"
	}
}

func (a *SchedulerAgent) getEventPriorityEmoji(priority multiagent.Priority) string {
	switch priority {
	case multiagent.PriorityCritical:
		return "🔥"
	case multiagent.PriorityHigh:
		return "⚠️"
	case multiagent.PriorityLow:
		return "🔽"
	default:
		return "🔸"
	}
}

func (a *SchedulerAgent) loadEventsFromMemory(ctx context.Context) {
	if a.memoryStore == nil {
		return
	}

	// List all event keys
	keys, err := a.memoryStore.List(ctx, "calendar_event:", 1000)
	if err != nil {
		return
	}

	// Load events
	events, err := a.memoryStore.GetMultiple(ctx, keys)
	if err != nil {
		return
	}

	a.scheduleMutex.Lock()
	defer a.scheduleMutex.Unlock()

	for _, eventInterface := range events {
		var event CalendarEvent
		if eventData, err := json.Marshal(eventInterface); err == nil {
			if err := json.Unmarshal(eventData, &event); err == nil {
				a.calendar[event.ID] = &event
			}
		}
	}
}

func (a *SchedulerAgent) buildSchedulerContext(ctx context.Context, msg *multiagent.Message) string {
	var contextBuilder strings.Builder

	contextBuilder.WriteString(fmt.Sprintf("You are %s, a scheduling and calendar management specialist.\n\n", a.name))
	contextBuilder.WriteString("You help users manage their calendar, schedule events, check availability, and optimize their time.\n\n")

	// Add upcoming events summary
	now := time.Now()
	upcomingEvents := a.getEventsInRange(now, now.Add(7*24*time.Hour))
	if len(upcomingEvents) > 0 {
		contextBuilder.WriteString("Upcoming Events (Next 7 Days):\n")
		for i, event := range upcomingEvents {
			if i >= 5 { // Limit to 5 events
				contextBuilder.WriteString(fmt.Sprintf("... and %d more events\n", len(upcomingEvents)-i))
				break
			}
			contextBuilder.WriteString(fmt.Sprintf("- %s: %s (%s)\n", event.StartTime.Format("Mon 15:04"), event.Title, event.Category))
		}
		contextBuilder.WriteString("\n")
	}

	contextBuilder.WriteString(fmt.Sprintf("User request: %s\n\n", msg.Content))
	contextBuilder.WriteString("Please provide helpful scheduling assistance, calendar management, or time planning advice.")

	return contextBuilder.String()
}
