import { ApiService } from './api';
import { User, Conversation, UserMemory, UserPreference } from '../types';

export interface PowerUserData {
  user: User;
  memories: UserMemory[];
  conversations: Conversation[];
  preferences: UserPreference[];
  analytics: UserAnalytics;
}

export interface UserAnalytics {
  totalMessages: number;
  averageResponseTime: number;
  mostActiveHour: number;
  topicsDiscussed: string[];
  memoryUtilization: number;
  conversationEngagement: number;
  toolUsage: {
    totalToolCalls: number;
    mostUsedTools: Record<string, number>;
    successRate: number;
  };
  temporalPatterns: {
    hourlyActivity: number[];
    dailyActivity: number[];
    weeklyActivity: number[];
  };
}

export class PowerUserApiService extends ApiService {
  constructor() {
    super();
  }

  // Enhanced user management methods
  async getAllUsers(): Promise<User[]> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users`);
    
    if (!response.ok) {
      throw new Error('Failed to get all users');
    }
    
    return response.json();
  }

  async deleteUser(userId: number): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete user');
    }
  }

  async updateUser(userId: number, userData: Partial<User>): Promise<User> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(userData)
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to update user');
    }
    
    return response.json();
  }

  // Comprehensive user data aggregation
  async getUserData(userId: number): Promise<PowerUserData> {
    try {
      const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/data`);
      
      if (!response.ok) {
        throw new Error('Failed to get user data');
      }
      
      return response.json();
    } catch (error) {
      console.error('Failed to get comprehensive user data:', error);
      throw error;
    }
  }

  // User preferences management
  async getUserPreferences(userId: number): Promise<UserPreference[]> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/preferences`);
    
    if (!response.ok) {
      throw new Error('Failed to get user preferences');
    }
    
    return response.json();
  }

  async updateUserPreference(userId: number, preferenceId: number, preference: Partial<UserPreference>): Promise<UserPreference> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/preferences/${preferenceId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(preference)
    });
    
    if (!response.ok) {
      throw new Error('Failed to update user preference');
    }
    
    return response.json();
  }

  async addUserPreference(userId: number, preference: Omit<UserPreference, 'id' | 'user_id' | 'created_at' | 'updated_at'>): Promise<UserPreference> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/preferences`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(preference)
    });
    
    if (!response.ok) {
      throw new Error('Failed to add user preference');
    }
    
    return response.json();
  }

  async deleteUserPreference(userId: number, preferenceId: number): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/preferences/${preferenceId}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete user preference');
    }
  }

  // Enhanced analytics and insights
  async getUserAnalytics(userId: number): Promise<UserAnalytics> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/analytics`);
    
    if (!response.ok) {
      throw new Error('Failed to get user analytics');
    }
    
    return response.json();
  }

  async getConversationSummaries(userId: number): Promise<Array<{
    conversation_id: number;
    title: string;
    summary: string;
    keywords: string[];
    priority_score: number;
    created_at: string;
    updated_at: string;
  }>> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/conversations/summaries`);
    
    if (!response.ok) {
      throw new Error('Failed to get conversation summaries');
    }
    
    return response.json();
  }

  async getMemoryTimeline(userId: number): Promise<Array<{
    date: string;
    memories_created: number;
    memories_accessed: number;
    memory_types: Record<string, number>;
  }>> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/memory/timeline`);
    
    if (!response.ok) {
      throw new Error('Failed to get memory timeline');
    }
    
    return response.json();
  }

  async getConversationMetrics(userId: number): Promise<{
    total_conversations: number;
    active_conversations: number;
    average_messages_per_conversation: number;
    conversation_duration_avg: number;
    topics_distribution: Record<string, number>;
    temporal_patterns: {
      hourly: number[];
      daily: number[];
      weekly: number[];
    };
  }> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/conversations/metrics`);
    
    if (!response.ok) {
      throw new Error('Failed to get conversation metrics');
    }
    
    return response.json();
  }

  // Advanced search and filtering
  async searchUserData(userId: number, query: string, filters?: {
    type?: 'memories' | 'conversations' | 'preferences' | 'all';
    dateRange?: { start: string; end: string };
    confidence?: number;
  }): Promise<{
    memories: UserMemory[];
    conversations: Conversation[];
    preferences: UserPreference[];
    total_results: number;
  }> {
    const queryParams = new URLSearchParams({ q: query });
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, typeof value === 'object' ? JSON.stringify(value) : value.toString());
        }
      });
    }

    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/search?${queryParams}`);
    
    if (!response.ok) {
      throw new Error('Failed to search user data');
    }
    
    return response.json();
  }

  // Data export and backup
  async exportUserData(userId: number, format: 'json' | 'csv' = 'json'): Promise<Blob> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/export?format=${format}`);
    
    if (!response.ok) {
      throw new Error('Failed to export user data');
    }
    
    return response.blob();
  }

  async getUserDataStats(userId: number): Promise<{
    total_data_points: number;
    storage_usage: number;
    data_quality_score: number;
    completeness_metrics: {
      profile_completeness: number;
      memory_density: number;
      preference_coverage: number;
    };
    growth_metrics: {
      weekly_growth: number;
      monthly_growth: number;
      engagement_trend: number;
    };
  }> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/stats`);
    
    if (!response.ok) {
      throw new Error('Failed to get user data statistics');
    }
    
    return response.json();
  }

  // System-wide power user features
  async getSystemOverview(): Promise<{
    total_users: number;
    total_conversations: number;
    total_memories: number;
    system_health: {
      database_performance: number;
      memory_usage: number;
      response_time_avg: number;
    };
    user_engagement_metrics: {
      daily_active_users: number;
      weekly_active_users: number;
      monthly_active_users: number;
    };
    trending_topics: Array<{
      topic: string;
      frequency: number;
      trend: 'up' | 'down' | 'stable';
    }>;
  }> {
    const response = await fetch(`${this.getBaseUrl()}/admin/system/overview`);
    
    if (!response.ok) {
      throw new Error('Failed to get system overview');
    }
    
    return response.json();
  }

  // User switching and session management
  async switchActiveUser(userId: number): Promise<{ success: boolean; message: string }> {
    const response = await fetch(`${this.getBaseUrl()}/api/power-user/users/${userId}/switch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ active: true })
    });
    
    if (!response.ok) {
      throw new Error('Failed to switch active user');
    }
    
    return response.json();
  }

  async getUserSessions(userId: number): Promise<Array<{
    session_id: string;
    start_time: string;
    end_time?: string;
    duration: number;
    activity_count: number;
    device_info?: string;
  }>> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/sessions`);
    
    if (!response.ok) {
      throw new Error('Failed to get user sessions');
    }
    
    return response.json();
  }

  // Advanced data relationships
  async getUserDataRelationships(userId: number): Promise<{
    memory_conversation_links: Array<{
      memory_id: number;
      conversation_id: number;
      strength: number;
      created_at: string;
    }>;
    topic_clusters: Array<{
      topic: string;
      related_memories: number[];
      related_conversations: number[];
      centrality_score: number;
    }>;
    preference_impacts: Array<{
      preference_id: number;
      affected_conversations: number[];
      influence_score: number;
    }>;
  }> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/relationships`);
    
    if (!response.ok) {
      throw new Error('Failed to get user data relationships');
    }
    
    return response.json();
  }

  // Bulk operations
  async bulkUpdateMemories(userId: number, updates: Array<{
    memory_id: number;
    updates: Partial<UserMemory>;
  }>): Promise<{ success: boolean; updated_count: number; errors: string[] }> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/memory/bulk`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ updates })
    });
    
    if (!response.ok) {
      throw new Error('Failed to bulk update memories');
    }
    
    return response.json();
  }

  async bulkDeleteConversations(userId: number, conversationIds: number[]): Promise<{
    success: boolean;
    deleted_count: number;
    errors: string[];
  }> {
    const response = await fetch(`${this.getBaseUrl()}/users/${userId}/conversations/bulk`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ conversation_ids: conversationIds })
    });
    
    if (!response.ok) {
      throw new Error('Failed to bulk delete conversations');
    }
    
    return response.json();
  }
}

export const powerUserApi = new PowerUserApiService();
