import React, { useState, useEffect } from 'react';
import { 
  X, RefreshCw, Search, Filter, Download, Clock, CheckCircle, 
  XCircle, AlertCircle, ChevronDown, ChevronRight, Database,
  Zap, MessageSquare, Settings, BarChart3, History, Archive
} from 'lucide-react';
import { ApiService } from '../services/api';
import { 
  ConversationDebugData, 
  DebugStep, 
  LLMRequestPersistent, 
  DebugSummary,
  DebugFilter 
} from '../types';

interface EnhancedDebugPanelProps {
  conversationId: number;
  userId: number;
  onClose: () => void;
  initialTab?: 'overview' | 'steps' | 'llm' | 'timeline' | 'export';
}

const api = new ApiService();

export default function EnhancedDebugPanel({ conversationId, userId, onClose, initialTab = 'overview' }: EnhancedDebugPanelProps) {
  const [debugData, setDebugData] = useState<ConversationDebugData | null>(null);
  const [debugSummary, setDebugSummary] = useState<DebugSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'steps' | 'llm' | 'timeline' | 'export'>(initialTab);
  const [expandedMessages, setExpandedMessages] = useState<Set<number>>(new Set());
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  const [filters, setFilters] = useState<DebugFilter>({});
  const [showFilters, setShowFilters] = useState(false);

  useEffect(() => {
    loadDebugData();
  }, [conversationId, userId]);

  const loadDebugData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const [dataResponse, summaryResponse] = await Promise.all([
        api.getConversationDebugData(conversationId, userId),
        api.getConversationDebugSummary(conversationId, userId)
      ]);
      
      setDebugData(dataResponse.debug_data);
      setDebugSummary(summaryResponse.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load debug data');
    } finally {
      setLoading(false);
    }
  };

  const toggleMessageExpansion = (messageId: number) => {
    setExpandedMessages(prev => {
      const newSet = new Set(prev);
      if (newSet.has(messageId)) {
        newSet.delete(messageId);
      } else {
        newSet.add(messageId);
      }
      return newSet;
    });
  };

  const toggleStepExpansion = (stepId: string) => {
    setExpandedSteps(prev => {
      const newSet = new Set(prev);
      if (newSet.has(stepId)) {
        newSet.delete(stepId);
      } else {
        newSet.add(stepId);
      }
      return newSet;
    });
  };

  const formatDuration = (ms?: number) => {
    if (!ms) return 'N/A';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const getStepIcon = (step: DebugStep) => {
    if (step.success) {
      return <CheckCircle className="w-4 h-4 text-green-600" />;
    } else {
      return <XCircle className="w-4 h-4 text-red-600" />;
    }
  };

  const getStepTypeColor = (stepType: string) => {
    const colors: Record<string, string> = {
      'context_building': 'bg-blue-100 text-blue-800',
      'tool_discovery': 'bg-green-100 text-green-800',
      'llm_request': 'bg-purple-100 text-purple-800',
      'tool_processing': 'bg-yellow-100 text-yellow-800',
      'followup_processing': 'bg-orange-100 text-orange-800',
      'memory_retrieval': 'bg-pink-100 text-pink-800',
      'error': 'bg-red-100 text-red-800'
    };
    return colors[stepType] || 'bg-gray-100 text-gray-800';
  };

  const filteredMessages = debugData?.messages.filter(message => {
    if (!searchQuery && !filters.step_types?.length) return true;
    
    const matchesSearch = !searchQuery || 
      message.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
      message.debug_steps.some(step => 
        step.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        step.description?.toLowerCase().includes(searchQuery.toLowerCase())
      );
    
    const matchesFilters = !filters.step_types?.length ||
      message.debug_steps.some(step => filters.step_types?.includes(step.step_type));
    
    return matchesSearch && matchesFilters;
  }) || [];

  const exportDebugData = () => {
    if (!debugData) return;
    
    const exportData = {
      conversation_id: debugData.conversation_id,
      conversation_title: debugData.conversation_title,
      export_timestamp: new Date().toISOString(),
      debug_sessions: debugData.debug_sessions,
      messages: debugData.messages,
      summary: debugSummary
    };
    
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `debug_data_${conversationId}_${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const clearDebugData = async () => {
    if (!window.confirm('Are you sure you want to clear all debug data for this conversation?')) {
      return;
    }
    
    try {
      if (debugSummary?.sessions) {
        for (const session of debugSummary.sessions) {
          if (session.is_active) {
            await api.endDebugSession(session.session_id);
          }
        }
      }
      
      await loadDebugData();
    } catch (err) {
      setError('Failed to clear debug data');
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-8">
          <div className="flex items-center">
            <RefreshCw className="w-5 h-5 animate-spin mr-3" />
            <span>Loading debug data...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-7xl h-full max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex items-center">
            <Database className="w-6 h-6 text-blue-600 mr-3" />
            <div>
              <h2 className="text-xl font-semibold text-gray-900">Enhanced Debug Panel</h2>
              <p className="text-sm text-gray-500">
                {debugData ? debugData.conversation_title : 'Loading conversation...'}
              </p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadDebugData}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
              title="Refresh Data"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
            <button
              onClick={exportDebugData}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
              title="Export Debug Data"
            >
              <Download className="w-4 h-4" />
            </button>
            <button
              onClick={clearDebugData}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
              title="Clear Debug Data"
            >
              <Archive className="w-4 h-4" />
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
            { id: 'steps', label: 'Debug Steps', icon: Settings },
            { id: 'llm', label: 'LLM Requests', icon: MessageSquare },
            { id: 'timeline', label: 'Timeline', icon: History },
            { id: 'export', label: 'Export', icon: Download }
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
              {/* Summary Cards */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="bg-blue-50 p-4 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-blue-600">Debug Sessions</span>
                    <Database className="w-4 h-4 text-blue-600" />
                  </div>
                  <p className="text-2xl font-bold text-blue-900 mt-1">
                    {debugSummary?.total_sessions || 0}
                  </p>
                  <p className="text-xs text-blue-600 mt-1">
                    {debugSummary?.active_sessions || 0} active
                  </p>
                </div>
                
                <div className="bg-green-50 p-4 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-green-600">Debug Steps</span>
                    <Settings className="w-4 h-4 text-green-600" />
                  </div>
                  <p className="text-2xl font-bold text-green-900 mt-1">
                    {debugSummary?.total_steps || 0}
                  </p>
                  <p className="text-xs text-green-600 mt-1">
                    Across {debugSummary?.total_messages || 0} messages
                  </p>
                </div>
                
                <div className="bg-purple-50 p-4 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-purple-600">Tools Used</span>
                    <Zap className="w-4 h-4 text-purple-600" />
                  </div>
                  <p className="text-2xl font-bold text-purple-900 mt-1">
                    {debugSummary?.total_tools_used || 0}
                  </p>
                  <p className="text-xs text-purple-600 mt-1">
                    Tool executions
                  </p>
                </div>
                
                <div className="bg-orange-50 p-4 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-orange-600">Processing Time</span>
                    <Clock className="w-4 h-4 text-orange-600" />
                  </div>
                  <p className="text-2xl font-bold text-orange-900 mt-1">
                    {debugSummary?.total_processing_time.toFixed(2) || '0.00'}s
                  </p>
                  <p className="text-xs text-orange-600 mt-1">
                    Total time
                  </p>
                </div>
              </div>

              {/* Recent Sessions */}
              {debugSummary?.sessions && debugSummary.sessions.length > 0 && (
                <div className="bg-gray-50 rounded-lg p-6">
                  <h3 className="text-lg font-medium text-gray-900 mb-4">Debug Sessions</h3>
                  <div className="space-y-3">
                    {debugSummary.sessions.map(session => (
                      <div key={session.session_id} className="bg-white p-4 rounded-lg border">
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium text-gray-900">
                            Session {session.session_id.slice(-8)}
                          </span>
                          <span className={`px-2 py-1 text-xs rounded-full ${
                            session.is_active 
                              ? 'bg-green-100 text-green-800' 
                              : 'bg-gray-100 text-gray-800'
                          }`}>
                            {session.is_active ? 'Active' : 'Completed'}
                          </span>
                        </div>
                        <div className="grid grid-cols-4 gap-4 text-sm text-gray-600">
                          <div>
                            <span className="font-medium">Messages:</span> {session.total_messages}
                          </div>
                          <div>
                            <span className="font-medium">Steps:</span> {session.total_steps}
                          </div>
                          <div>
                            <span className="font-medium">Tools:</span> {session.total_tools_used}
                          </div>
                          <div>
                            <span className="font-medium">Time:</span> {session.total_processing_time.toFixed(2)}s
                          </div>
                        </div>
                        <div className="mt-2 text-xs text-gray-500">
                          Started: {new Date(session.started_at).toLocaleString()}
                          {session.ended_at && (
                            <span> • Ended: {new Date(session.ended_at).toLocaleString()}</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Debug Steps Tab */}
          {activeTab === 'steps' && (
            <div className="space-y-4">
              {/* Search and Filter */}
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div className="relative">
                    <Search className="w-4 h-4 absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
                    <input
                      type="text"
                      placeholder="Search debug steps..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                  <button
                    onClick={() => setShowFilters(!showFilters)}
                    className="p-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    <Filter className="w-4 h-4" />
                  </button>
                </div>
                <div className="text-sm text-gray-500">
                  {filteredMessages.length} messages with debug data
                </div>
              </div>

              {/* Messages with Debug Steps */}
              <div className="space-y-4">
                {filteredMessages.map(message => (
                  <div key={message.message_id} className="bg-white border border-gray-200 rounded-lg">
                    <div 
                      className="p-4 cursor-pointer hover:bg-gray-50 transition-colors"
                      onClick={() => toggleMessageExpansion(message.message_id)}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          {expandedMessages.has(message.message_id) ? (
                            <ChevronDown className="w-4 h-4 text-gray-500" />
                          ) : (
                            <ChevronRight className="w-4 h-4 text-gray-500" />
                          )}
                          <span className={`px-2 py-1 text-xs rounded-full ${
                            message.role === 'user' 
                              ? 'bg-blue-100 text-blue-800' 
                              : 'bg-green-100 text-green-800'
                          }`}>
                            {message.role}
                          </span>
                          <span className="font-medium text-gray-900">
                            {message.content.length > 50 
                              ? message.content.substring(0, 50) + '...' 
                              : message.content
                            }
                          </span>
                        </div>
                        <div className="flex items-center space-x-2 text-sm text-gray-500">
                          <span>{message.debug_steps.length} steps</span>
                          <span>•</span>
                          <span>{new Date(message.timestamp).toLocaleString()}</span>
                        </div>
                      </div>
                    </div>
                    
                    {expandedMessages.has(message.message_id) && (
                      <div className="border-t border-gray-200 p-4 bg-gray-50">
                        <div className="space-y-3">
                          {message.debug_steps.map(step => (
                            <div key={step.step_id} className="bg-white p-3 rounded border">
                              <div 
                                className="flex items-center justify-between cursor-pointer"
                                onClick={() => toggleStepExpansion(step.step_id)}
                              >
                                <div className="flex items-center space-x-3">
                                  {getStepIcon(step)}
                                  <span className={`px-2 py-1 text-xs rounded-full ${getStepTypeColor(step.step_type)}`}>
                                    {step.step_type}
                                  </span>
                                  <span className="font-medium">{step.title}</span>
                                </div>
                                <div className="flex items-center space-x-2 text-sm text-gray-500">
                                  <span>{formatDuration(step.duration_ms)}</span>
                                  {expandedSteps.has(step.step_id) ? (
                                    <ChevronDown className="w-4 h-4" />
                                  ) : (
                                    <ChevronRight className="w-4 h-4" />
                                  )}
                                </div>
                              </div>
                              
                              {expandedSteps.has(step.step_id) && (
                                <div className="mt-3 space-y-2">
                                  {step.description && (
                                    <p className="text-sm text-gray-600">{step.description}</p>
                                  )}
                                  {step.error_message && (
                                    <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
                                      {step.error_message}
                                    </div>
                                  )}
                                  {step.input_data && Object.keys(step.input_data).length > 0 && (
                                    <div>
                                      <span className="text-sm font-medium text-gray-700">Input:</span>
                                      <pre className="text-xs bg-gray-100 p-2 rounded mt-1 overflow-x-auto">
                                        {JSON.stringify(step.input_data, null, 2)}
                                      </pre>
                                    </div>
                                  )}
                                  {step.output_data && Object.keys(step.output_data).length > 0 && (
                                    <div>
                                      <span className="text-sm font-medium text-gray-700">Output:</span>
                                      <pre className="text-xs bg-gray-100 p-2 rounded mt-1 overflow-x-auto">
                                        {JSON.stringify(step.output_data, null, 2)}
                                      </pre>
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* LLM Requests Tab */}
          {activeTab === 'llm' && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">LLM Requests</h3>
              <div className="space-y-4">
                {debugData?.messages.map(message => (
                  message.llm_requests.map(request => (
                    <div key={request.request_id} className="bg-white border border-gray-200 rounded-lg p-4">
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center space-x-3">
                          <span className="font-medium text-gray-900">{request.model}</span>
                          <span className="text-sm text-gray-500">
                            {new Date(request.timestamp).toLocaleString()}
                          </span>
                        </div>
                        <div className="flex items-center space-x-2 text-sm text-gray-500">
                          <span>{formatDuration(request.processing_time_ms)}</span>
                          {request.token_usage && (
                            <span>• {request.token_usage.total_tokens} tokens</span>
                          )}
                        </div>
                      </div>
                      
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="font-medium text-gray-700">Temperature:</span> {request.temperature || 'default'}
                        </div>
                        <div>
                          <span className="font-medium text-gray-700">Max Tokens:</span> {request.max_tokens || 'default'}
                        </div>
                        <div>
                          <span className="font-medium text-gray-700">Stream:</span> {request.stream ? 'Yes' : 'No'}
                        </div>
                        <div>
                          <span className="font-medium text-gray-700">Tools Available:</span> {request.tools_available?.length || 0}
                        </div>
                      </div>
                      
                      {request.tools_used && request.tools_used.length > 0 && (
                        <div className="mt-3">
                          <span className="text-sm font-medium text-gray-700">Tools Used:</span>
                          <div className="flex flex-wrap gap-2 mt-1">
                            {request.tools_used.map(tool => (
                              <span key={tool} className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">
                                {tool}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))
                ))}
              </div>
            </div>
          )}

          {/* Timeline Tab */}
          {activeTab === 'timeline' && (
            <div className="space-y-4">
              <h3 className="text-lg font-medium text-gray-900">Debug Timeline</h3>
              <div className="space-y-2">
                {debugData?.messages.map(message => (
                  <div key={message.message_id} className="space-y-2">
                    <div className="flex items-center space-x-3 py-2">
                      <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                      <span className="font-medium text-gray-900">
                        {message.role === 'user' ? 'User Message' : 'Assistant Response'}
                      </span>
                      <span className="text-sm text-gray-500">
                        {new Date(message.timestamp).toLocaleString()}
                      </span>
                    </div>
                    <div className="ml-5 space-y-1">
                      {message.debug_steps.map(step => (
                        <div key={step.step_id} className="flex items-center space-x-3 py-1">
                          <div className={`w-1 h-1 rounded-full ${step.success ? 'bg-green-500' : 'bg-red-500'}`}></div>
                          <span className="text-sm text-gray-700">{step.title}</span>
                          <span className="text-xs text-gray-500">{formatDuration(step.duration_ms)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Export Tab */}
          {activeTab === 'export' && (
            <div className="space-y-6">
              <h3 className="text-lg font-medium text-gray-900">Export Debug Data</h3>
              
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                <div className="flex items-center">
                  <AlertCircle className="w-5 h-5 text-yellow-600 mr-2" />
                  <span className="text-sm text-yellow-800">
                    Export functionality allows you to save debug data for external analysis or reporting.
                  </span>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="bg-white p-6 border border-gray-200 rounded-lg">
                  <h4 className="font-medium text-gray-900 mb-3">Export Options</h4>
                  <div className="space-y-3">
                    <button
                      onClick={exportDebugData}
                      className="w-full flex items-center justify-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                    >
                      <Download className="w-4 h-4 mr-2" />
                      Export as JSON
                    </button>
                    <button
                      onClick={clearDebugData}
                      className="w-full flex items-center justify-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
                    >
                      <Archive className="w-4 h-4 mr-2" />
                      Clear Debug Data
                    </button>
                  </div>
                </div>

                <div className="bg-white p-6 border border-gray-200 rounded-lg">
                  <h4 className="font-medium text-gray-900 mb-3">Export Information</h4>
                  <div className="space-y-2 text-sm text-gray-600">
                    <div>
                      <span className="font-medium">Total Messages:</span> {debugData?.messages.length || 0}
                    </div>
                    <div>
                      <span className="font-medium">Total Steps:</span> {debugSummary?.total_steps || 0}
                    </div>
                    <div>
                      <span className="font-medium">Total Sessions:</span> {debugSummary?.total_sessions || 0}
                    </div>
                    <div>
                      <span className="font-medium">Processing Time:</span> {debugSummary?.total_processing_time.toFixed(2) || '0.00'}s
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
