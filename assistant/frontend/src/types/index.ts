export interface User {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
}

export interface IntermediaryStep {
  step_id: string;
  step_type: 'tool_call' | 'tool_result' | 'memory_retrieval' | 'context_building' | 'llm_request' | 'llm_response' | 'error';
  timestamp: string;
  title: string;
  description?: string;
  data: Record<string, any>;
  duration_ms?: number;
  success: boolean;
  error_message?: string;
}

export interface ToolCall {
  tool_name: string;
  arguments: Record<string, any>;
  server_id?: string;
}

export interface ToolResult {
  tool_name: string;
  success: boolean;
  result: any;
  error_message?: string;
  execution_time_ms?: number;
}

export interface LLMRequest {
  model: string;
  messages: Array<Record<string, any>>;
  temperature?: number;
  max_tokens?: number;
  tools?: Array<Record<string, any>>;
  tool_choice?: string;
  stream: boolean;
  timestamp: string;
}

export interface LLMResponse {
  response: Record<string, any>;
  timestamp: string;
  processing_time_ms: number;
  token_usage?: Record<string, any>;
}

export interface Message {
  id: number;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  processing_time?: number;
  token_count?: number;
  llm_model?: string;
  temperature?: number;

  // Enhanced debug fields
  debug_enabled?: boolean;
  debug_data?: Record<string, any>;
  intermediary_steps?: IntermediaryStep[];
  llm_request?: LLMRequest;
  llm_response?: LLMResponse;
  tool_calls?: ToolCall[];
  tool_results?: ToolResult[];
  total_processing_time_ms?: number;
  step_count?: number;
  error_count?: number;
}

export interface Conversation {
  id: number;
  title: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
  user_id: number;
  messages: Message[];
}

export interface UserMemory {
  id: number;
  user_id: number;
  memory_type: 'explicit' | 'implicit' | 'preference';
  key: string;
  value: string;
  confidence: number;
  source?: string;
  created_at: string;
  updated_at: string;
  last_accessed: string;
  access_count: number;
}

export interface UserPreference {
  id: number;
  user_id: number;
  category: string;
  key: string;
  value: any;
  created_at: string;
  updated_at: string;
}

export interface ChatRequest {
  message: string;
  conversation_id?: number;
  user_id: number;
  temperature?: number;
  max_tokens?: number;
  stream?: boolean;
}

export interface ChatRequestWithDebug extends ChatRequest {
  enable_tool_trace?: boolean;
  show_debug_steps?: boolean;
  trace_level?: 'basic' | 'detailed' | 'verbose';
  include_intermediary_steps?: boolean;
  include_llm_request?: boolean;
  include_tool_details?: boolean;
  include_context_building?: boolean;
}

export interface ChatResponse {
  message: Message;
  conversation_id: number;
  processing_time: number;
  token_count?: number;
}

export interface ChatResponseWithDebug extends ChatResponse {
  total_steps?: number;
  successful_steps?: number;
  failed_steps?: number;
  tools_used?: string[];
  tool_trace?: any;
  debug_enabled?: boolean;
}

export interface SystemStatus {
  status: string;
  version: string;
  lmstudio_connected: boolean;
  database_connected: boolean;
  active_conversations: number;
  total_users: number;
}

export interface DebugScript {
  name: string;
  description: string;
  type: string;
  path: string;
}

export interface ScriptResult {
  script_name: string;
  success: boolean;
  output: string;
  error?: string;
  execution_time: number;
}

export interface DebugData {
  timestamp: string;
  request_id?: string;
  processing_time_ms?: number;
  steps?: IntermediaryStep[];
  error?: string;
  summary?: string;
}

// New debug persistence types
export interface DebugSession {
  session_id: string;
  started_at: string;
  ended_at?: string;
  is_active: boolean;
  total_messages: number;
  total_steps: number;
  total_tools_used: number;
  total_processing_time: number;
}

export interface DebugStep {
  step_id: string;
  step_type: string;
  step_order: number;
  title: string;
  description?: string;
  timestamp: string;
  duration_ms?: number;
  success: boolean;
  error_message?: string;
  input_data?: Record<string, any>;
  output_data?: Record<string, any>;
  step_metadata?: Record<string, any>;
}

export interface LLMRequestPersistent {
  request_id: string;
  model: string;
  temperature?: number;
  max_tokens?: number;
  stream: boolean;
  timestamp: string;
  processing_time_ms?: number;
  token_usage?: Record<string, any>;
  tools_available?: Array<Record<string, any>>;
  tools_used?: string[];
  tool_calls?: Array<Record<string, any>>;
  tool_results?: Array<Record<string, any>>;
  request_messages: Array<Record<string, any>>;
  response_data: Record<string, any>;
}

export interface DebugPreference {
  enabled: boolean;
  auto_enable?: boolean;
  save_llm_requests?: boolean;
  save_tool_details?: boolean;
  retention_days?: number;
}

export interface ConversationDebugData {
  conversation_id: number;
  conversation_title: string;
  debug_sessions: DebugSession[];
  messages: Array<{
    message_id: number;
    role: string;
    content: string;
    timestamp: string;
    processing_time?: number;
    token_count?: number;
    debug_steps: DebugStep[];
    llm_requests: LLMRequestPersistent[];
  }>;
}

export interface DebugSummary {
  conversation_id: number;
  has_debug_data: boolean;
  total_sessions: number;
  active_sessions: number;
  total_messages: number;
  total_steps: number;
  total_tools_used: number;
  total_processing_time: number;
  sessions: DebugSession[];
}

// Enhanced message interface with debug information
export interface EnhancedMessage extends Message {
  debug_steps?: DebugStep[];
  llm_requests?: LLMRequestPersistent[];
  debug_session_id?: string;
  has_debug_data?: boolean;
}

// Debug visualization types
export interface DebugStepVisualization {
  step: DebugStep;
  expanded: boolean;
  indent_level: number;
  has_children: boolean;
  parent_step_id?: string;
}

export interface DebugTimeline {
  events: Array<{
    timestamp: string;
    event_type: 'step' | 'llm_request' | 'tool_call';
    title: string;
    description?: string;
    duration_ms?: number;
    success: boolean;
    data?: Record<string, any>;
  }>;
  total_duration_ms: number;
  start_time: string;
  end_time: string;
}

export interface DebugMetrics {
  total_steps: number;
  successful_steps: number;
  failed_steps: number;
  total_processing_time: number;
  average_step_time: number;
  tool_usage_count: number;
  llm_requests_count: number;
  error_rate: number;
  performance_score: number;
}

// Filter and search types for debug data
export interface DebugFilter {
  step_types?: string[];
  success_only?: boolean;
  date_range?: {
    start: string;
    end: string;
  };
  message_role?: 'user' | 'assistant' | 'system';
  has_errors?: boolean;
  min_duration_ms?: number;
  max_duration_ms?: number;
}

export interface DebugSearchQuery {
  query: string;
  filters?: DebugFilter;
  sort_by?: 'timestamp' | 'duration' | 'success' | 'step_type';
  sort_order?: 'asc' | 'desc';
  limit?: number;
  offset?: number;
}

export interface DebugSearchResult {
  total_results: number;
  results: Array<{
    step: DebugStep;
    message_id: number;
    conversation_id: number;
    relevance_score: number;
    highlight?: string;
  }>;
  aggregations?: {
    step_types: Record<string, number>;
    success_rate: number;
    average_duration: number;
    date_distribution: Array<{
      date: string;
      count: number;
    }>;
  };
}

// Export settings interface
export interface DebugExportSettings {
  format: 'json' | 'csv' | 'xlsx';
  include_llm_requests: boolean;
  include_tool_details: boolean;
  include_metadata: boolean;
  date_range?: {
    start: string;
    end: string;
  };
  conversation_ids?: number[];
  compress: boolean;
}

// Real-time debug monitoring
export interface DebugMonitoringEvent {
  event_type: 'step_started' | 'step_completed' | 'step_failed' | 'session_started' | 'session_ended';
  timestamp: string;
  conversation_id: number;
  user_id: number;
  session_id?: string;
  step_id?: string;
  data?: Record<string, any>;
}

export interface DebugMonitoringSubscription {
  user_id?: number;
  conversation_id?: number;
  event_types?: string[];
  include_data?: boolean;
}
