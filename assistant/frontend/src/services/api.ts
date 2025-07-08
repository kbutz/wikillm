import { User, Conversation, UserMemory, Message } from '../types';

export class ApiService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = process.env.REACT_APP_API_URL || 'http://localhost:8000';
  }

  protected getBaseUrl(): string {
    return this.baseUrl;
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

  async listUsers(): Promise<User[]> {
    const response = await fetch(`${this.baseUrl}/users/`);

    if (!response.ok) {
      throw new Error('Failed to list users');
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
    mcp_servers_connected?: number;
    mcp_servers_total?: number;
    mcp_tools_available?: number;
  }> {
    const response = await fetch(`${this.baseUrl}/status`);

    if (!response.ok) {
      throw new Error('Failed to get system status');
    }

    return response.json();
  }

  // MCP-related methods
  async getMCPStatus(): Promise<{
    success: boolean;
    data: {
      servers: Record<string, {
        name: string;
        type: string;
        enabled: boolean;
        status: string;
        error?: string;
        tools_count: number;
        resources_count: number;
        prompts_count: number;
      }>;
      total_servers: number;
      connected_servers: number;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/status`);

    if (!response.ok) {
      throw new Error('Failed to get MCP status');
    }

    return response.json();
  }

  async listMCPServers(): Promise<{
    success: boolean;
    data: {
      servers: Array<{
        server_id: string;
        name: string;
        description?: string;
        type: string;
        enabled: boolean;
        status: string;
        error?: string;
        capabilities: {
          tools: number;
          resources: number;
          prompts: number;
        };
      }>;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers`);

    if (!response.ok) {
      throw new Error('Failed to list MCP servers');
    }

    return response.json();
  }

  async addMCPServer(config: {
    server_id: string;
    name: string;
    description?: string;
    type: 'stdio' | 'http' | 'websocket';
    command?: string;
    args?: string[];
    url?: string;
    env?: Record<string, string>;
    timeout?: number;
    enabled?: boolean;
    auto_reconnect?: boolean;
  }): Promise<{
    success: boolean;
    data: { message: string };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to add MCP server');
    }

    return response.json();
  }

  async updateMCPServer(serverId: string, config: {
    server_id: string;
    name: string;
    description?: string;
    type: 'stdio' | 'http' | 'websocket';
    command?: string;
    args?: string[];
    url?: string;
    env?: Record<string, string>;
    timeout?: number;
    enabled?: boolean;
    auto_reconnect?: boolean;
  }): Promise<{
    success: boolean;
    data: { message: string };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers/${serverId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to update MCP server');
    }

    return response.json();
  }

  async deleteMCPServer(serverId: string): Promise<{
    success: boolean;
    data: { message: string };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers/${serverId}`, {
      method: 'DELETE'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to delete MCP server');
    }

    return response.json();
  }

  async connectMCPServer(serverId: string): Promise<{
    success: boolean;
    data: { message: string };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers/${serverId}/connect`, {
      method: 'POST'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to connect to MCP server');
    }

    return response.json();
  }

  async disconnectMCPServer(serverId: string): Promise<{
    success: boolean;
    data: { message: string };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/servers/${serverId}/disconnect`, {
      method: 'POST'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to disconnect from MCP server');
    }

    return response.json();
  }

  async listMCPTools(): Promise<{
    success: boolean;
    data: {
      tools: Array<{
        name: string;
        description: string;
        input_schema: any;
        server_id: string;
      }>;
      total_count: number;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/tools`);

    if (!response.ok) {
      throw new Error('Failed to list MCP tools');
    }

    return response.json();
  }

  async callMCPTool(toolName: string, toolArgs: any, serverId?: string): Promise<{
    success: boolean;
    data: any;
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/tools/call`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        tool_name: toolName,
        arguments: toolArgs,
        server_id: serverId
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to call MCP tool');
    }

    return response.json();
  }

  async listMCPResources(): Promise<{
    success: boolean;
    data: {
      resources: Array<{
        uri: string;
        name: string;
        description?: string;
        mime_type?: string;
        server_id: string;
      }>;
      total_count: number;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/resources`);

    if (!response.ok) {
      throw new Error('Failed to list MCP resources');
    }

    return response.json();
  }

  async readMCPResource(uri: string, serverId?: string): Promise<{
    success: boolean;
    data: any;
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/resources/read`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        uri,
        server_id: serverId
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to read MCP resource');
    }

    return response.json();
  }

  async listMCPPrompts(): Promise<{
    success: boolean;
    data: {
      prompts: Array<{
        name: string;
        description: string;
        arguments: any[];
        server_id: string;
      }>;
      total_count: number;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/prompts`);

    if (!response.ok) {
      throw new Error('Failed to list MCP prompts');
    }

    return response.json();
  }

  async getMCPPrompt(name: string, promptArgs?: any, serverId?: string): Promise<{
    success: boolean;
    data: any;
  }> {
    const response = await fetch(`${this.baseUrl}/mcp/prompts/get`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        arguments: promptArgs,
        server_id: serverId
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to get MCP prompt');
    }

    return response.json();
  }

  async getToolAnalytics(conversationId: number, userId: number): Promise<{
    success: boolean;
    data: {
      total_tool_calls: number;
      tools_used: Record<string, number>;
      success_rate: number;
      most_used_tool?: string;
      tool_timeline: Array<{
        tool_name: string;
        timestamp: string;
        success: boolean;
      }>;
    };
  }> {
    const response = await fetch(`${this.baseUrl}/conversations/${conversationId}/tools/analytics?user_id=${userId}`);

    if (!response.ok) {
      throw new Error('Failed to get tool analytics');
    }

    return response.json();
  }

  // Debug script methods
  async listDebugScripts(): Promise<Array<{
    name: string;
    description: string;
    type: string;
    path: string;
  }>> {
    const response = await fetch(`${this.baseUrl}/debug/scripts`);

    if (!response.ok) {
      throw new Error('Failed to list debug scripts');
    }

    return response.json();
  }

  async runDebugScript(scriptName: string): Promise<{
    script_name: string;
    success: boolean;
    output: string;
    error?: string;
    execution_time: number;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/scripts/${scriptName}/run`, {
      method: 'POST'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || `Failed to run debug script: ${scriptName}`);
    }

    return response.json();
  }
}
