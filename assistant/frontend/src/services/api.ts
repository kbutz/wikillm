import { 
  User, 
  Conversation, 
  UserMemory, 
  Message, 
  ChatRequestWithDebug, 
  ChatResponseWithDebug,
  DebugData, 
  DebugSession, 
  DebugStep, 
  LLMRequestPersistent,
  ConversationDebugData,
  DebugSummary
} from '../types';

export class ApiService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = process.env.REACT_APP_API_URL || 'http://localhost:8000';
  }

  protected getBaseUrl(): string {
    return this.baseUrl;
  }

  // User management methods (unchanged)
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

  // Conversation management methods (unchanged)
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

  async deleteConversation(conversationId: number, userId: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/conversations/${conversationId}?user_id=${userId}`, {
      method: 'DELETE'
    });

    if (!response.ok) {
      throw new Error('Failed to delete conversation');
    }
  }

  // Chat methods (unchanged)
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

  async sendMessageWithDebug(data: ChatRequestWithDebug): Promise<ChatResponseWithDebug> {
    const response = await fetch(`${this.baseUrl}/debug/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...data,
        enable_tool_trace: true,
        show_debug_steps: true,
        trace_level: "detailed"
      })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to send debug message');
    }

    return response.json();
  }

  // Debug persistence methods (NEW)
  async getConversationDebugData(conversationId: number, userId: number): Promise<{
    conversation_id: number;
    has_debug_data: boolean;
    debug_data: ConversationDebugData;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/conversations/${conversationId}/data?user_id=${userId}`);

    if (!response.ok) {
      throw new Error('Failed to get conversation debug data');
    }

    return response.json();
  }

  async getConversationDebugSummary(conversationId: number, userId: number): Promise<{
    success: boolean;
    data: DebugSummary;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/conversations/${conversationId}/summary?user_id=${userId}`);

    if (!response.ok) {
      throw new Error('Failed to get conversation debug summary');
    }

    return response.json();
  }

  async getUserDebugPreference(userId: number): Promise<{
    success: boolean;
    data: { enabled: boolean };
  }> {
    const response = await fetch(`${this.baseUrl}/debug/users/${userId}/preference`);

    if (!response.ok) {
      throw new Error('Failed to get user debug preference');
    }

    return response.json();
  }

  async setUserDebugPreference(userId: number, enabled: boolean): Promise<{
    success: boolean;
    message: string;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/users/${userId}/preference`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to set debug preference');
    }

    return response.json();
  }

  async endDebugSession(sessionId: string): Promise<{
    success: boolean;
    message: string;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/sessions/${sessionId}/end`, {
      method: 'POST'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to end debug session');
    }

    return response.json();
  }

  async cleanupOldDebugData(daysOld: number = 30): Promise<{
    success: boolean;
    message: string;
  }> {
    const response = await fetch(`${this.baseUrl}/debug/cleanup?days_old=${daysOld}`, {
      method: 'POST'
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.detail || 'Failed to cleanup debug data');
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

  // Debug script methods (unchanged)
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

  // Memory management methods (unchanged)
  async getUserMemory(userId: number): Promise<UserMemory[]> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/memory`);

    if (!response.ok) {
      throw new Error('Failed to get user memory');
    }

    return response.json();
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

  // System status methods (unchanged)
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

  // MCP-related methods (unchanged)
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

  // Legacy support for existing code
  async getDebugData(conversationId: number, userId: number): Promise<DebugData> {
    const response = await this.getConversationDebugData(conversationId, userId);
    
    // Transform the new format to match the old DebugData interface
    const debugData: DebugData = {
      timestamp: new Date().toISOString(),
      steps: response.debug_data.messages.flatMap(msg => 
        msg.debug_steps.map(step => ({
          step_id: step.step_id,
          step_type: step.step_type as any,
          timestamp: step.timestamp,
          title: step.title,
          description: step.description,
          data: step.input_data || {},
          duration_ms: step.duration_ms,
          success: step.success,
          error_message: step.error_message
        }))
      )
    };

    return debugData;
  }
}

// Create a singleton instance of ApiService
const apiService = new ApiService();

// Export standalone functions that use the ApiService instance
export const fetchDebugData = (conversationId: number, userId: number): Promise<DebugData> => {
  return apiService.getDebugData(conversationId, userId);
};
