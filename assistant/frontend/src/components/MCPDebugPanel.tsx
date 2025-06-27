import React, { useState, useEffect } from 'react';
import { 
  X, RefreshCw, Settings, Database, Cpu, Wifi, WifiOff, 
  CheckCircle, XCircle, AlertCircle, Plus, Trash2, Edit3,
  Play, Square, Code, FileText, Zap, Activity, BarChart3,
  ChevronDown, ChevronRight, Eye, Terminal, Clock
} from 'lucide-react';
import { ApiService } from '../services/api';

interface MCPDebugPanelProps {
  userId: number;
  onClose: () => void;
}

interface MCPServer {
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
}

interface MCPTool {
  name: string;
  description: string;
  input_schema: any;
  server_id: string;
}

interface MCPResource {
  uri: string;
  name: string;
  description?: string;
  mime_type?: string;
  server_id: string;
}

interface SystemStatus {
  status: string;
  version: string;
  lmstudio_connected: boolean;
  database_connected: boolean;
  active_conversations: number;
  total_users: number;
  mcp_servers_connected?: number;
  mcp_servers_total?: number;
  mcp_tools_available?: number;
}

const api = new ApiService();

export default function MCPDebugPanel({ userId, onClose }: MCPDebugPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'servers' | 'tools' | 'resources' | 'add-server'>('overview');
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [mcpServers, setMcpServers] = useState<MCPServer[]>([]);
  const [mcpTools, setMcpTools] = useState<MCPTool[]>([]);
  const [mcpResources, setMcpResources] = useState<MCPResource[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['system']));
  const [testResults, setTestResults] = useState<Record<string, any>>({});
  const [newServerConfig, setNewServerConfig] = useState({
    server_id: '',
    name: '',
    description: '',
    type: 'stdio' as 'stdio' | 'http' | 'websocket',
    command: '',
    args: [''],
    url: '',
    timeout: 30,
    enabled: true
  });

  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Load system status
      const status = await api.getSystemStatus();
      setSystemStatus(status);

      // Load MCP servers
      try {
        const serversResponse = await api.listMCPServers();
        setMcpServers(serversResponse.data.servers);
      } catch (e) {
        console.warn('MCP servers not available:', e);
        setMcpServers([]);
      }

      // Load MCP tools
      try {
        const toolsResponse = await api.listMCPTools();
        setMcpTools(toolsResponse.data.tools);
      } catch (e) {
        console.warn('MCP tools not available:', e);
        setMcpTools([]);
      }

      // Load MCP resources
      try {
        const resourcesResponse = await api.listMCPResources();
        setMcpResources(resourcesResponse.data.resources);
      } catch (e) {
        console.warn('MCP resources not available:', e);
        setMcpResources([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load debug data');
    } finally {
      setLoading(false);
    }
  };

  const testServerConnection = async (serverId: string) => {
    try {
      setTestResults(prev => ({ ...prev, [serverId]: { testing: true } }));
      
      const response = await api.connectMCPServer(serverId);
      setTestResults(prev => ({
        ...prev,
        [serverId]: {
          success: true,
          message: response.data.message,
          timestamp: new Date().toISOString()
        }
      }));
      
      // Reload server status
      loadData();
    } catch (err) {
      setTestResults(prev => ({
        ...prev,
        [serverId]: {
          success: false,
          error: err instanceof Error ? err.message : 'Connection failed',
          timestamp: new Date().toISOString()
        }
      }));
    }
  };

  const testTool = async (toolName: string, serverId: string) => {
    try {
      setTestResults(prev => ({ ...prev, [`tool_${toolName}`]: { testing: true } }));
      
      // Use a simple test with minimal arguments
      const response = await api.callMCPTool(toolName, {}, serverId);
      setTestResults(prev => ({
        ...prev,
        [`tool_${toolName}`]: {
          success: true,
          result: response.data,
          timestamp: new Date().toISOString()
        }
      }));
    } catch (err) {
      setTestResults(prev => ({
        ...prev,
        [`tool_${toolName}`]: {
          success: false,
          error: err instanceof Error ? err.message : 'Tool test failed',
          timestamp: new Date().toISOString()
        }
      }));
    }
  };

  const addServer = async () => {
    try {
      await api.addMCPServer({
        ...newServerConfig,
        args: newServerConfig.args.filter(arg => arg.trim() !== '')
      });
      
      setNewServerConfig({
        server_id: '',
        name: '',
        description: '',
        type: 'stdio',
        command: '',
        args: [''],
        url: '',
        timeout: 30,
        enabled: true
      });
      
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add server');
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const StatusBadge = ({ status, error }: { status: string; error?: string }) => {
    const getStatusConfig = () => {
      switch (status) {
        case 'connected':
          return { icon: CheckCircle, color: 'text-green-600', bg: 'bg-green-100', text: 'Connected' };
        case 'connecting':
          return { icon: Activity, color: 'text-yellow-600', bg: 'bg-yellow-100', text: 'Connecting' };
        case 'disconnected':
          return { icon: WifiOff, color: 'text-gray-600', bg: 'bg-gray-100', text: 'Disconnected' };
        case 'error':
          return { icon: XCircle, color: 'text-red-600', bg: 'bg-red-100', text: 'Error' };
        default:
          return { icon: AlertCircle, color: 'text-gray-600', bg: 'bg-gray-100', text: status };
      }
    };

    const config = getStatusConfig();
    const Icon = config.icon;

    return (
      <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${config.bg} ${config.color}`} title={error}>
        <Icon className="w-3 h-3 mr-1" />
        {config.text}
      </div>
    );
  };

  if (loading && !systemStatus) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-8">
          <div className="flex items-center">
            <RefreshCw className="w-5 h-5 animate-spin mr-3" />
            <span>Loading debug information...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-6xl h-full max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center">
            <Database className="w-6 h-6 text-blue-600 mr-3" />
            <div>
              <h2 className="text-xl font-semibold text-gray-900">MCP Debug Panel</h2>
              <p className="text-sm text-gray-500">Model Context Protocol Integration Status</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadData}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
              title="Refresh"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={async () => {
                try {
                  setLoading(true);
                  const response = await fetch(`${api['baseUrl']}/mcp/reload`, { method: 'POST' });
                  if (response.ok) {
                    const result = await response.json();
                    console.log('Configuration reloaded:', result);
                    await loadData();
                  } else {
                    console.error('Failed to reload configuration');
                  }
                } catch (err) {
                  console.error('Reload error:', err);
                } finally {
                  setLoading(false);
                }
              }}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
              title="Reload Configuration"
            >
              <Settings className="w-4 h-4" />
            </button>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-200">
          {[
            { id: 'overview', label: 'Overview', icon: BarChart3 },
            { id: 'servers', label: 'Servers', icon: Database },
            { id: 'tools', label: 'Tools', icon: Zap },
            { id: 'resources', label: 'Resources', icon: FileText },
            { id: 'add-server', label: 'Add Server', icon: Plus }
          ].map(tab => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`flex items-center px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                <Icon className="w-4 h-4 mr-2" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {error && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
              <div className="flex items-center">
                <XCircle className="w-5 h-5 text-red-600 mr-2" />
                <span className="text-red-800">{error}</span>
              </div>
            </div>
          )}

          {/* Overview Tab */}
          {activeTab === 'overview' && (
            <div className="space-y-6">
              {/* System Status */}
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-gray-900 flex items-center">
                    <Cpu className="w-5 h-5 mr-2" />
                    System Status
                  </h3>
                  {systemStatus && (
                    <StatusBadge status={systemStatus.status} />
                  )}
                </div>
                
                {systemStatus && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="bg-white p-4 rounded-lg">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">LMStudio</span>
                        {systemStatus.lmstudio_connected ? (
                          <CheckCircle className="w-4 h-4 text-green-600" />
                        ) : (
                          <XCircle className="w-4 h-4 text-red-600" />
                        )}
                      </div>
                      <p className="text-lg font-semibold text-gray-900 mt-1">
                        {systemStatus.lmstudio_connected ? 'Connected' : 'Disconnected'}
                      </p>
                    </div>
                    
                    <div className="bg-white p-4 rounded-lg">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">MCP Servers</span>
                        <Database className="w-4 h-4 text-blue-600" />
                      </div>
                      <p className="text-lg font-semibold text-gray-900 mt-1">
                        {systemStatus.mcp_servers_connected || 0} / {systemStatus.mcp_servers_total || 0}
                      </p>
                    </div>
                    
                    <div className="bg-white p-4 rounded-lg">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">Tools Available</span>
                        <Zap className="w-4 h-4 text-green-600" />
                      </div>
                      <p className="text-lg font-semibold text-gray-900 mt-1">
                        {systemStatus.mcp_tools_available || 0}
                      </p>
                    </div>
                    
                    <div className="bg-white p-4 rounded-lg">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">Active Users</span>
                        <Activity className="w-4 h-4 text-purple-600" />
                      </div>
                      <p className="text-lg font-semibold text-gray-900 mt-1">
                        {systemStatus.total_users}
                      </p>
                    </div>
                  </div>
                )}
              </div>

              {/* MCP Overview */}
              <div className="bg-gray-50 rounded-lg p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-4 flex items-center">
                  <Settings className="w-5 h-5 mr-2" />
                  MCP Integration Overview
                </h3>
                
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="bg-white p-4 rounded-lg border">
                    <h4 className="font-medium text-gray-900 mb-2">Configured Servers</h4>
                    <p className="text-2xl font-bold text-blue-600">{mcpServers.length}</p>
                    <p className="text-sm text-gray-500 mt-1">
                      {mcpServers.filter(s => s.status === 'connected').length} connected
                    </p>
                  </div>
                  
                  <div className="bg-white p-4 rounded-lg border">
                    <h4 className="font-medium text-gray-900 mb-2">Available Tools</h4>
                    <p className="text-2xl font-bold text-green-600">{mcpTools.length}</p>
                    <p className="text-sm text-gray-500 mt-1">
                      Ready for use in conversations
                    </p>
                  </div>
                  
                  <div className="bg-white p-4 rounded-lg border">
                    <h4 className="font-medium text-gray-900 mb-2">Resources</h4>
                    <p className="text-2xl font-bold text-purple-600">{mcpResources.length}</p>
                    <p className="text-sm text-gray-500 mt-1">
                      Accessible data sources
                    </p>
                  </div>
                </div>
              </div>

              {/* Quick Actions */}
              <div className="bg-gray-50 rounded-lg p-6">
                <h3 className="text-lg font-medium text-gray-900 mb-4">Quick Actions</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <button
                    onClick={() => setActiveTab('add-server')}
                    className="flex items-center justify-center p-4 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    <Plus className="w-5 h-5 mr-2" />
                    Add MCP Server
                  </button>
                  
                  <button
                    onClick={() => setActiveTab('tools')}
                    className="flex items-center justify-center p-4 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                  >
                    <Zap className="w-5 h-5 mr-2" />
                    Test Tools
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Servers Tab */}
          {activeTab === 'servers' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-medium text-gray-900">MCP Servers</h3>
                <button
                  onClick={() => setActiveTab('add-server')}
                  className="flex items-center px-3 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                  <Plus className="w-4 h-4 mr-1" />
                  Add Server
                </button>
              </div>

              {mcpServers.length === 0 ? (
                <div className="text-center py-12">
                  <Database className="w-12 h-12 text-gray-300 mx-auto mb-4" />
                  <h4 className="text-lg font-medium text-gray-900 mb-2">No MCP Servers</h4>
                  <p className="text-gray-500 mb-4">No MCP servers are currently configured.</p>
                  <button
                    onClick={() => setActiveTab('add-server')}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    Add Your First Server
                  </button>
                </div>
              ) : (
                <div className="space-y-4">
                  {mcpServers.map(server => (
                    <div key={server.server_id} className="bg-white border border-gray-200 rounded-lg p-6">
                      <div className="flex items-center justify-between mb-4">
                        <div>
                          <h4 className="text-lg font-medium text-gray-900">{server.name}</h4>
                          <p className="text-sm text-gray-500">{server.description}</p>
                          <div className="flex items-center mt-2 space-x-4">
                            <span className="text-xs text-gray-500">ID: {server.server_id}</span>
                            <span className="text-xs text-gray-500">Type: {server.type}</span>
                          </div>
                        </div>
                        <div className="flex items-center space-x-3">
                          <StatusBadge status={server.status} error={server.error} />
                          <button
                            onClick={() => testServerConnection(server.server_id)}
                            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                            title="Test Connection"
                          >
                            <Play className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                      
                      <div className="grid grid-cols-3 gap-4 mb-4">
                        <div className="text-center p-3 bg-gray-50 rounded-lg">
                          <Zap className="w-5 h-5 text-green-600 mx-auto mb-1" />
                          <p className="text-sm font-medium text-gray-900">{server.capabilities.tools}</p>
                          <p className="text-xs text-gray-500">Tools</p>
                        </div>
                        <div className="text-center p-3 bg-gray-50 rounded-lg">
                          <FileText className="w-5 h-5 text-blue-600 mx-auto mb-1" />
                          <p className="text-sm font-medium text-gray-900">{server.capabilities.resources}</p>
                          <p className="text-xs text-gray-500">Resources</p>
                        </div>
                        <div className="text-center p-3 bg-gray-50 rounded-lg">
                          <Code className="w-5 h-5 text-purple-600 mx-auto mb-1" />
                          <p className="text-sm font-medium text-gray-900">{server.capabilities.prompts}</p>
                          <p className="text-xs text-gray-500">Prompts</p>
                        </div>
                      </div>
                      
                      {server.error && (
                        <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                          <p className="text-sm text-red-800">{server.error}</p>
                        </div>
                      )}
                      
                      {testResults[server.server_id] && (
                        <div className={`p-3 rounded-lg mt-3 ${
                          testResults[server.server_id].success 
                            ? 'bg-green-50 border border-green-200' 
                            : 'bg-red-50 border border-red-200'
                        }`}>
                          <div className="flex items-center justify-between">
                            <span className={`text-sm ${
                              testResults[server.server_id].success ? 'text-green-800' : 'text-red-800'
                            }`}>
                              {testResults[server.server_id].testing 
                                ? 'Testing connection...' 
                                : testResults[server.server_id].success 
                                  ? testResults[server.server_id].message
                                  : testResults[server.server_id].error
                              }
                            </span>
                            <span className="text-xs text-gray-500">
                              {testResults[server.server_id].timestamp && 
                                new Date(testResults[server.server_id].timestamp).toLocaleTimeString()
                              }
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Tools Tab */}
          {activeTab === 'tools' && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Available MCP Tools</h3>
              
              {mcpTools.length === 0 ? (
                <div className="text-center py-12">
                  <Zap className="w-12 h-12 text-gray-300 mx-auto mb-4" />
                  <h4 className="text-lg font-medium text-gray-900 mb-2">No Tools Available</h4>
                  <p className="text-gray-500">Connect MCP servers to access their tools.</p>
                </div>
              ) : (
                <div className="grid gap-4">
                  {mcpTools.map((tool, index) => (
                    <div key={`${tool.server_id}-${tool.name}`} className="bg-white border border-gray-200 rounded-lg p-6">
                      <div className="flex items-center justify-between mb-3">
                        <div>
                          <h4 className="text-lg font-medium text-gray-900">{tool.name}</h4>
                          <p className="text-sm text-gray-500">{tool.description}</p>
                          <span className="inline-block mt-2 px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded-full">
                            {tool.server_id}
                          </span>
                        </div>
                        <button
                          onClick={() => testTool(tool.name, tool.server_id)}
                          className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                          title="Test Tool"
                        >
                          <Play className="w-4 h-4" />
                        </button>
                      </div>
                      
                      {tool.input_schema && Object.keys(tool.input_schema).length > 0 && (
                        <div className="mt-4">
                          <details className="text-sm">
                            <summary className="cursor-pointer text-gray-600 hover:text-gray-900">
                              View Schema
                            </summary>
                            <pre className="mt-2 p-3 bg-gray-100 rounded text-xs overflow-x-auto">
                              {JSON.stringify(tool.input_schema, null, 2)}
                            </pre>
                          </details>
                        </div>
                      )}
                      
                      {testResults[`tool_${tool.name}`] && (
                        <div className={`p-3 rounded-lg mt-3 ${
                          testResults[`tool_${tool.name}`].success 
                            ? 'bg-green-50 border border-green-200' 
                            : 'bg-red-50 border border-red-200'
                        }`}>
                          <div className="flex items-center justify-between mb-2">
                            <span className={`text-sm font-medium ${
                              testResults[`tool_${tool.name}`].success ? 'text-green-800' : 'text-red-800'
                            }`}>
                              {testResults[`tool_${tool.name}`].testing 
                                ? 'Testing tool...' 
                                : testResults[`tool_${tool.name}`].success 
                                  ? 'Tool test successful'
                                  : 'Tool test failed'
                              }
                            </span>
                            <span className="text-xs text-gray-500">
                              {testResults[`tool_${tool.name}`].timestamp && 
                                new Date(testResults[`tool_${tool.name}`].timestamp).toLocaleTimeString()
                              }
                            </span>
                          </div>
                          {testResults[`tool_${tool.name}`].result && (
                            <pre className="text-xs bg-white p-2 rounded border overflow-x-auto">
                              {JSON.stringify(testResults[`tool_${tool.name}`].result, null, 2)}
                            </pre>
                          )}
                          {testResults[`tool_${tool.name}`].error && (
                            <p className="text-sm text-red-800">
                              {testResults[`tool_${tool.name}`].error}
                            </p>
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Resources Tab */}
          {activeTab === 'resources' && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Available MCP Resources</h3>
              
              {mcpResources.length === 0 ? (
                <div className="text-center py-12">
                  <FileText className="w-12 h-12 text-gray-300 mx-auto mb-4" />
                  <h4 className="text-lg font-medium text-gray-900 mb-2">No Resources Available</h4>
                  <p className="text-gray-500">Connect MCP servers to access their resources.</p>
                </div>
              ) : (
                <div className="grid gap-4">
                  {mcpResources.map((resource) => (
                    <div key={`${resource.server_id}-${resource.uri}`} className="bg-white border border-gray-200 rounded-lg p-6">
                      <div className="flex items-center justify-between mb-3">
                        <div>
                          <h4 className="text-lg font-medium text-gray-900">{resource.name}</h4>
                          <p className="text-sm text-gray-500">{resource.description}</p>
                          <div className="flex items-center mt-2 space-x-4">
                            <span className="text-xs text-gray-500">URI: {resource.uri}</span>
                            {resource.mime_type && (
                              <span className="text-xs text-gray-500">Type: {resource.mime_type}</span>
                            )}
                          </div>
                          <span className="inline-block mt-2 px-2 py-1 bg-purple-100 text-purple-800 text-xs rounded-full">
                            {resource.server_id}
                          </span>
                        </div>
                        <button
                          onClick={() => console.log('Read resource:', resource.uri)}
                          className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                          title="Read Resource"
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Add Server Tab */}
          {activeTab === 'add-server' && (
            <div className="max-w-2xl">
              <h3 className="text-lg font-medium text-gray-900 mb-6">Add New MCP Server</h3>
              
              <form onSubmit={(e) => { e.preventDefault(); addServer(); }} className="space-y-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Server ID</label>
                    <input
                      type="text"
                      value={newServerConfig.server_id}
                      onChange={(e) => setNewServerConfig(prev => ({ ...prev, server_id: e.target.value }))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="filesystem-local"
                      required
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Name</label>
                    <input
                      type="text"
                      value={newServerConfig.name}
                      onChange={(e) => setNewServerConfig(prev => ({ ...prev, name: e.target.value }))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="Local Filesystem"
                      required
                    />
                  </div>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Description</label>
                  <input
                    type="text"
                    value={newServerConfig.description}
                    onChange={(e) => setNewServerConfig(prev => ({ ...prev, description: e.target.value }))}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="Access to local filesystem"
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">Type</label>
                  <select
                    value={newServerConfig.type}
                    onChange={(e) => setNewServerConfig(prev => ({ ...prev, type: e.target.value as any }))}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="stdio">STDIO</option>
                    <option value="http">HTTP</option>
                    <option value="websocket">WebSocket</option>
                  </select>
                </div>
                
                {newServerConfig.type === 'stdio' && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Command</label>
                      <input
                        type="text"
                        value={newServerConfig.command}
                        onChange={(e) => setNewServerConfig(prev => ({ ...prev, command: e.target.value }))}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="npx"
                        required
                      />
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">Arguments</label>
                      {newServerConfig.args.map((arg, index) => (
                        <div key={index} className="flex items-center mb-2">
                          <input
                            type="text"
                            value={arg}
                            onChange={(e) => {
                              const newArgs = [...newServerConfig.args];
                              newArgs[index] = e.target.value;
                              setNewServerConfig(prev => ({ ...prev, args: newArgs }));
                            }}
                            className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent mr-2"
                            placeholder={index === 0 ? "-y" : index === 1 ? "@modelcontextprotocol/server-filesystem" : "/path/to/directory"}
                          />
                          {index > 0 && (
                            <button
                              type="button"
                              onClick={() => {
                                const newArgs = newServerConfig.args.filter((_, i) => i !== index);
                                setNewServerConfig(prev => ({ ...prev, args: newArgs }));
                              }}
                              className="p-2 text-red-600 hover:bg-red-100 rounded-lg transition-colors"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          )}
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={() => setNewServerConfig(prev => ({ ...prev, args: [...prev.args, ''] }))}
                        className="flex items-center text-sm text-blue-600 hover:text-blue-700"
                      >
                        <Plus className="w-4 h-4 mr-1" />
                        Add Argument
                      </button>
                    </div>
                  </>
                )}
                
                {newServerConfig.type === 'http' && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">URL</label>
                    <input
                      type="url"
                      value={newServerConfig.url}
                      onChange={(e) => setNewServerConfig(prev => ({ ...prev, url: e.target.value }))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="http://localhost:3001"
                      required
                    />
                  </div>
                )}
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">Timeout (seconds)</label>
                    <input
                      type="number"
                      value={newServerConfig.timeout}
                      onChange={(e) => setNewServerConfig(prev => ({ ...prev, timeout: parseInt(e.target.value) }))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      min="5"
                      max="300"
                    />
                  </div>
                  
                  <div className="flex items-center">
                    <input
                      type="checkbox"
                      id="enabled"
                      checked={newServerConfig.enabled}
                      onChange={(e) => setNewServerConfig(prev => ({ ...prev, enabled: e.target.checked }))}
                      className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                    />
                    <label htmlFor="enabled" className="ml-2 text-sm text-gray-700">
                      Enable server immediately
                    </label>
                  </div>
                </div>
                
                <div className="flex justify-end space-x-3">
                  <button
                    type="button"
                    onClick={() => setNewServerConfig({
                      server_id: '',
                      name: '',
                      description: '',
                      type: 'stdio',
                      command: '',
                      args: [''],
                      url: '',
                      timeout: 30,
                      enabled: true
                    })}
                    className="px-4 py-2 text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    Reset
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                  >
                    Add Server
                  </button>
                </div>
              </form>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}