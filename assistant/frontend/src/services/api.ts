import { User, Conversation, UserMemory, Message } from '../types';

export class ApiService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = process.env.REACT_APP_API_URL || 'http://localhost:8000';
  }

  async createUser(userData: { username: string; email?: string; full_name?: string }): Promise<User> {
    const response = await fetch(`${this.baseUrl}/users/`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(userData)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to create user');
    }

    return response.json();
  }

  async getUser(userId: number): Promise<User> {
    const response = await fetch(`${this.baseUrl}/users/${userId}`);

    if (!response.ok) {
      throw new Error('Failed to get user');
    }

    return response.json();
  }

  async getUserConversations(userId: number): Promise<Conversation[]> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/conversations`);

    if (!response.ok) {
      throw new Error('Failed to get conversations');
    }

    return response.json();
  }

  async createConversation(userId: number, title: string): Promise<Conversation> {
    const response = await fetch(`${this.baseUrl}/conversations/`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user_id: userId, title })
    });

    if (!response.ok) {
      throw new Error('Failed to create conversation');
    }

    return response.json();
  }

  async sendMessage(data: { 
    message: string; 
    user_id: number; 
    conversation_id?: number;
    temperature?: number;
  }): Promise<{
    message: Message;
    conversation_id: number;
    processing_time: number;
    token_count?: number;
  }> {
    const response = await fetch(`${this.baseUrl}/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to send message');
    }

    return response.json();
  }

  async getUserMemory(userId: number): Promise<UserMemory[]> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/memory`);

    if (!response.ok) {
      throw new Error('Failed to get user memory');
    }

    return response.json();
  }

  async deleteConversation(conversationId: number, userId: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/conversations/${conversationId}?user_id=${userId}`, {
      method: 'DELETE'
    });

    if (!response.ok) {
      throw new Error('Failed to delete conversation');
    }
  }

  async addUserMemory(userId: number, memory: {
    memory_type: 'explicit' | 'implicit' | 'preference';
    key: string;
    value: string;
    confidence?: number;
    source?: string;
  }): Promise<UserMemory> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/memory`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(memory)
    });

    if (!response.ok) {
      throw new Error('Failed to add user memory');
    }

    return response.json();
  }

  async updateUserMemory(userId: number, memoryId: number, memory: {
    memory_type: 'explicit' | 'implicit' | 'preference';
    key: string;
    value: string;
    confidence?: number;
    source?: string;
  }): Promise<UserMemory> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/memory/${memoryId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(memory)
    });

    if (!response.ok) {
      throw new Error('Failed to update user memory');
    }

    return response.json();
  }

  async deleteUserMemory(userId: number, memoryId: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/memory/${memoryId}`, {
      method: 'DELETE'
    });

    if (!response.ok) {
      throw new Error('Failed to delete user memory');
    }
  }

  async getSystemStatus(): Promise<{
    status: string;
    version: string;
    lmstudio_connected: boolean;
    database_connected: boolean;
    active_conversations: number;
    total_users: number;
  }> {
    const response = await fetch(`${this.baseUrl}/status`);

    if (!response.ok) {
      throw new Error('Failed to get system status');
    }

    return response.json();
  }
}
