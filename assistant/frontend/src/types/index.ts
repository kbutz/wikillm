export interface User {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: number;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  processing_time?: number;
  token_count?: number;
  model_used?: string;
  temperature?: number;
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

export interface ChatResponse {
  message: Message;
  conversation_id: number;
  processing_time: number;
  token_count?: number;
}

export interface SystemStatus {
  status: string;
  version: string;
  lmstudio_connected: boolean;
  database_connected: boolean;
  active_conversations: number;
  total_users: number;
}
