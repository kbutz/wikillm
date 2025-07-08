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
