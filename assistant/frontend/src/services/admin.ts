import { ApiService } from './api';

export interface AdminUser {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
  last_active?: string;
  conversation_count: number;
  memory_size: number;
  memory_entries: number;
}

export interface AdminConversation {
  id: number;
  title: string;
  created_at: string;
  updated_at: string;
  message_count: number;
  last_message?: string;
  user_id: number;
  username: string;
}

export interface AdminMemory {
  personal_info: Record<string, any>;
  conversation_history: any[];
  context_memory: Record<string, any>;
  preferences: Record<string, any>;
  size: number;
  last_updated: string;
}

export interface AdminSystemStats {
  total_users: number;
  active_users: number;
  total_conversations: number;
  total_messages: number;
  total_memory_entries: number;
  system_health: {
    status: string;
    recent_errors: number;
    database_size: string;
    uptime: string;
  };
}

export class AdminService extends ApiService {
  // User Management
  async getAdminUsers(skip: number = 0, limit: number = 100): Promise<AdminUser[]> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users?skip=${skip}&limit=${limit}`);
    if (!response.ok) {
      throw new Error('Failed to fetch admin users');
    }
    return response.json();
  }

  async getUserDetails(userId: number): Promise<any> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}`);
    if (!response.ok) {
      throw new Error('Failed to fetch user details');
    }
    return response.json();
  }

  async createAdminUser(userData: { username: string; email?: string; full_name?: string }): Promise<any> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(userData)
    });
    if (!response.ok) {
      throw new Error('Failed to create user');
    }
    return response.json();
  }

  async deleteAdminUser(userId: number): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete user');
    }
  }

  // Memory Management
  async getAdminUserMemory(userId: number): Promise<AdminMemory> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}/memory`);
    if (!response.ok) {
      throw new Error('Failed to fetch user memory');
    }
    return response.json();
  }

  async updateAdminUserMemory(userId: number, memoryData: any): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}/memory`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(memoryData)
    });
    if (!response.ok) {
      throw new Error('Failed to update user memory');
    }
  }

  async clearUserMemory(userId: number, memoryType?: string): Promise<void> {
    const url = memoryType 
      ? `${this.getBaseUrl()}/admin/users/${userId}/memory?memory_type=${memoryType}`
      : `${this.getBaseUrl()}/admin/users/${userId}/memory`;

    const response = await fetch(url, { method: 'DELETE' });
    if (!response.ok) {
      throw new Error('Failed to clear user memory');
    }
  }

  // Conversation Management
  async getAdminUserConversations(userId: number, limit: number = 100): Promise<AdminConversation[]> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}/conversations?limit=${limit}`);
    if (!response.ok) {
      throw new Error('Failed to fetch user conversations');
    }
    return response.json();
  }

  async getConversationMessages(conversationId: number): Promise<any> {
    const response = await fetch(`${this.getBaseUrl()}/admin/conversations/${conversationId}/messages`);
    if (!response.ok) {
      throw new Error('Failed to fetch conversation messages');
    }
    return response.json();
  }

  async deleteAdminConversation(conversationId: number): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/admin/conversations/${conversationId}`, {
      method: 'DELETE'
    });
    if (!response.ok) {
      throw new Error('Failed to delete conversation');
    }
  }

  // Data Export
  async exportUserData(userId: number): Promise<void> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}/export`);
    if (!response.ok) {
      throw new Error('Failed to export user data');
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `user_${userId}_export.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
  }

  // System Statistics
  async getSystemStats(): Promise<AdminSystemStats> {
    const response = await fetch(`${this.getBaseUrl()}/admin/system/stats`);
    if (!response.ok) {
      throw new Error('Failed to fetch system stats');
    }
    return response.json();
  }

  // User Impersonation
  async impersonateUser(userId: number): Promise<any> {
    const response = await fetch(`${this.getBaseUrl()}/admin/users/${userId}/impersonate`, {
      method: 'POST'
    });
    if (!response.ok) {
      throw new Error('Failed to impersonate user');
    }
    return response.json();
  }

  // Admin Health Check
  async adminHealthCheck(): Promise<any> {
    const response = await fetch(`${this.getBaseUrl()}/admin/health`);
    if (!response.ok) {
      throw new Error('Admin health check failed');
    }
    return response.json();
  }
}
