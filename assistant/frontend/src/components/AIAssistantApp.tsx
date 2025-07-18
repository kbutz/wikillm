import React, { useState, useEffect, useRef } from 'react';
import { Send, MessageSquare, Plus, Trash2, User, Settings, Brain, Clock, Database, Shield, Bug, BarChart3, History, Archive, Zap } from 'lucide-react';
import { ApiService } from '../services/api';
import { 
  User as UserType, 
  Message, 
  Conversation, 
  UserMemory, 
  ChatRequestWithDebug, 
  ChatResponse, 
  ChatResponseWithDebug,
  DebugPreference,
  DebugSummary
} from '../types';
import MessageBubble from './MessageBubble';
import LoadingMessage from './LoadingMessage';
import UserSetupModal from './UserSetupModal';
import MemoryPanel from './MemoryPanel';
import DebugPanel from './DebugPanel';
import MCPDebugPanel from './MCPDebugPanel';
import DebugSummaryComponent from './DebugSummary';
import EnhancedDebugPanel from './EnhancedDebugPanel';

const api = new ApiService();

interface AIAssistantAppProps {
  onAdminAccess: () => void;
  onPowerUserAccess: () => void;
}

export default function AIAssistantApp({ onAdminAccess, onPowerUserAccess }: AIAssistantAppProps) {
  const [currentUser, setCurrentUser] = useState<UserType | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConversation, setActiveConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [showUserSetup, setShowUserSetup] = useState(true);
  const [userMemories, setUserMemories] = useState<UserMemory[]>([]);
  const [showMemories, setShowMemories] = useState(false);
  const [showDebugPanel, setShowDebugPanel] = useState(false);
  const [showDebugSummary, setShowDebugSummary] = useState(false);
  const [showEnhancedDebugPanel, setShowEnhancedDebugPanel] = useState(false);
  const [enhancedDebugPanelTab, setEnhancedDebugPanelTab] = useState<'overview' | 'steps' | 'llm' | 'timeline' | 'export'>('overview');
  const [debugMode, setDebugMode] = useState(false);
  const [debugPreference, setDebugPreference] = useState<DebugPreference>({ enabled: false });
  const [debugSummary, setDebugSummary] = useState<DebugSummary | null>(null);
  const [isLoadingDebugData, setIsLoadingDebugData] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Load debug preference from backend and localStorage
  useEffect(() => {
    const loadDebugPreference = async () => {
      if (currentUser) {
        try {
          // Load from backend
          const response = await api.getUserDebugPreference(currentUser.id);
          const serverPreference = response.data.enabled;

          // Check localStorage for session-specific preference
          const localPreference = localStorage.getItem(`debugMode_${currentUser.id}`);
          const localEnabled = localPreference === 'true';

          // Use localStorage value if it exists, otherwise use server preference
          const finalEnabled = localPreference !== null ? localEnabled : serverPreference;

          setDebugMode(finalEnabled);
          setDebugPreference({ enabled: finalEnabled });

          // Sync localStorage with final decision
          localStorage.setItem(`debugMode_${currentUser.id}`, finalEnabled.toString());
        } catch (error) {
          console.error('Failed to load debug preference:', error);
          // Fallback to localStorage only
          const localPreference = localStorage.getItem(`debugMode_${currentUser.id}`);
          if (localPreference !== null) {
            const enabled = localPreference === 'true';
            setDebugMode(enabled);
            setDebugPreference({ enabled });
          }
        }
      }
    };

    loadDebugPreference();
  }, [currentUser]);

  // Save debug preference to backend and localStorage
  const updateDebugPreference = async (enabled: boolean) => {
    if (!currentUser) return;

    try {
      // Update localStorage immediately for instant UI response
      localStorage.setItem(`debugMode_${currentUser.id}`, enabled.toString());
      setDebugMode(enabled);
      setDebugPreference({ enabled });

      // Update backend preference
      await api.setUserDebugPreference(currentUser.id, enabled);
    } catch (error) {
      console.error('Failed to save debug preference:', error);
      // Keep localStorage change even if backend fails
    }
  };

  // Load debug summary for active conversation
  useEffect(() => {
    const loadDebugSummary = async () => {
      if (activeConversation && currentUser && debugMode) {
        try {
          setIsLoadingDebugData(true);
          const response = await api.getConversationDebugSummary(activeConversation.id, currentUser.id);
          setDebugSummary(response.data);
        } catch (error) {
          console.error('Failed to load debug summary:', error);
          setDebugSummary(null);
        } finally {
          setIsLoadingDebugData(false);
        }
      } else {
        setDebugSummary(null);
      }
    };

    loadDebugSummary();
  }, [activeConversation, currentUser, debugMode]);

  // Initialize user
  const handleUserSetup = async (userData: { username: string; email?: string; full_name?: string }) => {
    try {
      const user = await api.createUser(userData);
      setCurrentUser(user);
      setShowUserSetup(false);
      localStorage.setItem('currentUserId', user.id.toString());
      loadConversations(user.id);
    } catch (error) {
      console.error('Failed to create user:', error);
    }
  };

  // Select existing user
  const handleUserSelect = (user: UserType) => {
    setCurrentUser(user);
    setShowUserSetup(false);
    localStorage.setItem('currentUserId', user.id.toString());
    loadConversations(user.id);
  };

  // Load conversations
  const loadConversations = async (userId: number) => {
    try {
      const convs = await api.getUserConversations(userId);
      setConversations(convs);
    } catch (error) {
      console.error('Failed to load conversations:', error);
    }
  };

  // Load user memories
  const loadUserMemories = async (userId: number) => {
    try {
      const memories = await api.getUserMemory(userId);
      setUserMemories(memories);
    } catch (error) {
      console.error('Failed to load memories:', error);
    }
  };

  // Create new conversation
  const createNewConversation = async () => {
    if (!currentUser) return;

    try {
      const conv = await api.createConversation(currentUser.id, 'New Conversation');
      setConversations(prev => [conv, ...prev]);
      setActiveConversation(conv);
      setMessages([]);
      setDebugSummary(null); // Reset debug summary for new conversation
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }
  };

  // Send message
  const sendMessage = async () => {
    if (!newMessage.trim() || !currentUser || isLoading) return;

    const userMessage: Message = {
      id: Date.now(),
      role: 'user',
      content: newMessage,
      timestamp: new Date().toISOString()
    };

    setMessages(prev => [...prev, userMessage]);
    setNewMessage('');
    setIsLoading(true);

    try {
      let response: ChatResponse | ChatResponseWithDebug;

      if (debugMode) {
        // Use debug endpoint
        const debugRequest: ChatRequestWithDebug = {
          message: newMessage,
          user_id: currentUser.id,
          conversation_id: activeConversation?.id,
          enable_tool_trace: true,
          show_debug_steps: true,
          trace_level: "detailed",
          include_intermediary_steps: true,
          include_llm_request: true,
          include_tool_details: true,
          include_context_building: true
        };

        response = await api.sendMessageWithDebug(debugRequest);
      } else {
        // Use regular endpoint
        response = await api.sendMessage({
          message: newMessage,
          user_id: currentUser.id,
          conversation_id: activeConversation?.id
        });
      }

      setMessages(prev => [...prev, response.message]);

      // Debug: Log the response message to see what debug data is included
      if (debugMode) {
        console.log('=== DEBUG MODE RESPONSE ===');
        console.log('Full response:', response);
        console.log('Response message:', response.message);
        
        // Log specific debug fields
        if (response.message) {
          console.log('Message ID:', response.message.id);
          console.log('Debug enabled:', response.message.debug_enabled);
          console.log('Intermediary steps:', response.message.intermediary_steps);
          console.log('LLM request:', response.message.llm_request);
          console.log('LLM response:', response.message.llm_response);
          console.log('Tool calls:', response.message.tool_calls);
          console.log('Tool results:', response.message.tool_results);
        }
        
        // Log the full ChatResponseWithDebug fields
        if ('tool_trace' in response) {
          console.log('Tool trace:', response.tool_trace);
          console.log('Debug enabled flag:', response.debug_enabled);
          console.log('Total steps:', response.total_steps);
          console.log('Tools used:', response.tools_used);
        }
        console.log('=== END DEBUG INFO ===');
      }

      // Update active conversation or create new one
      if (!activeConversation) {
        const updatedConvs = await api.getUserConversations(currentUser.id);
        setConversations(updatedConvs);
        const newConv = updatedConvs.find(c => c.id === response.conversation_id);
        if (newConv) setActiveConversation(newConv);
      }

      // Reload debug summary if in debug mode
      if (debugMode && activeConversation) {
        try {
          const debugResponse = await api.getConversationDebugSummary(activeConversation.id, currentUser.id);
          setDebugSummary(debugResponse.data);
          
          // If the message doesn't have debug data, try to fetch it separately
          if (!response.message.intermediary_steps && !response.message.llm_request) {
            console.log('Message missing debug data, attempting to fetch from debug data endpoint');
            
            const debugData = await api.getConversationDebugData(activeConversation.id, currentUser.id);
            
            if (debugData.has_debug_data && debugData.debug_data.messages.length > 0) {
              // Find the most recent message (should be our response)
              const latestDebugMessage = debugData.debug_data.messages[debugData.debug_data.messages.length - 1];
              
              if (latestDebugMessage.message_id === response.message.id) {
                console.log('Found matching debug data for message:', latestDebugMessage);
                
                // Update the message with debug data
                setMessages(prev => prev.map(msg => 
                  msg.id === response.message.id 
                    ? {
                        ...msg,
                        intermediary_steps: latestDebugMessage.debug_steps.map(step => ({
                          step_id: step.step_id,
                          step_type: step.step_type as 'tool_call' | 'tool_result' | 'memory_retrieval' | 'context_building' | 'llm_request' | 'llm_response' | 'error',
                          timestamp: step.timestamp,
                          title: step.title,
                          description: step.description,
                          data: step.input_data || {},
                          duration_ms: step.duration_ms,
                          success: step.success,
                          error_message: step.error_message
                        })),
                        llm_request: latestDebugMessage.llm_requests[0] ? {
                          model: latestDebugMessage.llm_requests[0].model,
                          messages: latestDebugMessage.llm_requests[0].request_messages,
                          temperature: latestDebugMessage.llm_requests[0].temperature,
                          max_tokens: latestDebugMessage.llm_requests[0].max_tokens,
                          tools: latestDebugMessage.llm_requests[0].tools_available,
                          tool_choice: latestDebugMessage.llm_requests[0].tools_available && latestDebugMessage.llm_requests[0].tools_available.length > 0 ? 'auto' : undefined,
                          stream: latestDebugMessage.llm_requests[0].stream,
                          timestamp: latestDebugMessage.llm_requests[0].timestamp
                        } : undefined,
                        llm_response: latestDebugMessage.llm_requests[0] ? {
                          response: latestDebugMessage.llm_requests[0].response_data,
                          timestamp: latestDebugMessage.llm_requests[0].timestamp,
                          processing_time_ms: latestDebugMessage.llm_requests[0].processing_time_ms ?? 0,
                          token_usage: latestDebugMessage.llm_requests[0].token_usage
                        } : undefined,
                        tool_calls: latestDebugMessage.llm_requests[0]?.tool_calls,
                        tool_results: latestDebugMessage.llm_requests[0]?.tool_results,
                        debug_enabled: true
                      } as Message
                    : msg
                ));
                
                console.log('Successfully merged debug data into message');
              }
            }
          }
        } catch (error) {
          console.error('Failed to reload debug summary:', error);
        }
      }
    } catch (error) {
      console.error('Failed to send message:', error);
      // Add error message to chat
      const errorMessage: Message = {
        id: Date.now() + 1,
        role: 'assistant',
        content: 'Sorry, I encountered an error processing your message. Please make sure LMStudio is running and try again.',
        timestamp: new Date().toISOString()
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  // Select conversation
  const selectConversation = async (conversation: Conversation) => {
    if (currentUser) {
      try {
        // Refresh conversations list from backend to ensure we have the latest data
        const updatedConvs = await api.getUserConversations(currentUser.id);
        setConversations(updatedConvs);

        // Find the selected conversation in the updated list
        const updatedConversation = updatedConvs.find(c => c.id === conversation.id);
        if (updatedConversation) {
          setActiveConversation(updatedConversation);
          setMessages(updatedConversation.messages || []);
        } else {
          // Fallback to the provided conversation if not found in updated list
          setActiveConversation(conversation);
          setMessages(conversation.messages || []);
        }
      } catch (error) {
        console.error('Failed to refresh conversations:', error);
        // Fallback to the provided conversation if refresh fails
        setActiveConversation(conversation);
        setMessages(conversation.messages || []);
      }
    } else {
      // If no user, just use the provided conversation
      setActiveConversation(conversation);
      setMessages(conversation.messages || []);
    }
  };

  // Delete conversation
  const deleteConversation = async (conversationId: number) => {
    if (!currentUser) return;

    try {
      await api.deleteConversation(conversationId, currentUser.id);
      setConversations(prev => prev.filter(c => c.id !== conversationId));
      if (activeConversation?.id === conversationId) {
        setActiveConversation(null);
        setMessages([]);
        setDebugSummary(null);
      }
    } catch (error) {
      console.error('Failed to delete conversation:', error);
    }
  };

  // Helper functions for debug panel navigation
  const openDebugSteps = () => {
    setEnhancedDebugPanelTab('steps');
    setShowEnhancedDebugPanel(true);
  };

  const openDebugLLM = () => {
    setEnhancedDebugPanelTab('llm');
    setShowEnhancedDebugPanel(true);
  };

  const openDebugOverview = () => {
    setEnhancedDebugPanelTab('overview');
    setShowEnhancedDebugPanel(true);
  };

  // Clear debug data for conversation
  const clearDebugData = async () => {
    if (!activeConversation || !currentUser) return;

    try {
      // End all active debug sessions for this conversation
      if (debugSummary?.sessions) {
        for (const session of debugSummary.sessions) {
          if (session.is_active) {
            await api.endDebugSession(session.session_id);
          }
        }
      }

      // Reload debug summary
      const response = await api.getConversationDebugSummary(activeConversation.id, currentUser.id);
      setDebugSummary(response.data);
    } catch (error) {
      console.error('Failed to clear debug data:', error);
    }
  };

  // Load user from localStorage on mount
  useEffect(() => {
    const savedUserId = localStorage.getItem('currentUserId');
    if (savedUserId) {
      api.getUser(parseInt(savedUserId))
        .then(user => {
          setCurrentUser(user);
          setShowUserSetup(false);
          loadConversations(user.id);
        })
        .catch(() => {
          localStorage.removeItem('currentUserId');
        });
    }
  }, []);

  // Auto-scroll to bottom of messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Keyboard shortcuts for debug features
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Only handle shortcuts if debug mode is active and we have an active conversation
      if (!debugMode || !activeConversation) return;

      // Ctrl/Cmd + D for debug steps
      if ((event.ctrlKey || event.metaKey) && event.key === 'd') {
        event.preventDefault();
        openDebugSteps();
      }
      // Ctrl/Cmd + L for LLM requests
      else if ((event.ctrlKey || event.metaKey) && event.key === 'l') {
        event.preventDefault();
        openDebugLLM();
      }
      // Ctrl/Cmd + Shift + D for debug overview
      else if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'D') {
        event.preventDefault();
        openDebugOverview();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [debugMode, activeConversation]);

  if (showUserSetup) {
    return <UserSetupModal onSetup={handleUserSetup} onSelectUser={handleUserSelect} />;
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-bold text-gray-900">AI Assistant</h1>
            <div className="flex items-center gap-2">
              <button
                onClick={() => updateDebugPreference(!debugMode)}
                className={`p-2 rounded-lg transition-colors ${
                  debugMode 
                    ? 'bg-green-100 text-green-600 hover:bg-green-200' 
                    : 'hover:bg-gray-100 text-gray-600'
                }`}
                title={debugMode ? 'Disable Debug Mode' : 'Enable Debug Mode'}
              >
                <Bug className="w-5 h-5" />
              </button>
              <button
                onClick={() => {
                  if (currentUser) loadUserMemories(currentUser.id);
                  setShowMemories(!showMemories);
                }}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                title="View Memories"
              >
                <Brain className="w-5 h-5 text-gray-600" />
              </button>
              <button
                onClick={() => setShowDebugPanel(true)}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                title="Debug Panel"
              >
                <Settings className="w-5 h-5 text-gray-600" />
              </button>
              <button
                onClick={onPowerUserAccess}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                title="Power User Console"
              >
                <User className="w-5 h-5 text-gray-600" />
              </button>
              <button
                onClick={onAdminAccess}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                title="Admin Dashboard"
              >
                <Shield className="w-5 h-5 text-gray-600" />
              </button>
            </div>
          </div>

          <button
            onClick={createNewConversation}
            className="w-full flex items-center justify-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus className="w-4 h-4 mr-2" />
            New Conversation
          </button>
        </div>

        {/* Conversations List */}
        <div className="flex-1 overflow-y-auto">
          {conversations.map(conversation => (
            <div
              key={conversation.id}
              className={`p-3 border-b border-gray-100 cursor-pointer hover:bg-gray-50 transition-colors group ${
                activeConversation?.id === conversation.id ? 'bg-blue-50 border-blue-200' : ''
              }`}
              onClick={() => selectConversation(conversation)}
            >
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-medium text-gray-900 truncate">
                      {conversation.title}
                    </h3>
                    {/* Debug indicator */}
                    {debugMode && activeConversation?.id === conversation.id && debugSummary?.has_debug_data && (
                      <div className="flex items-center gap-1">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            openDebugSteps();
                          }}
                          className="flex items-center gap-1 px-1 py-0.5 bg-green-100 text-green-700 rounded text-xs hover:bg-green-200 transition-colors"
                          title="Click to view debug steps"
                        >
                          <Bug className="w-3 h-3" />
                          <span className="font-medium">{debugSummary.total_steps}</span>
                        </button>
                        {debugSummary.total_tools_used > 0 && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              openDebugLLM();
                            }}
                            className="flex items-center gap-1 px-1 py-0.5 bg-blue-100 text-blue-700 rounded text-xs hover:bg-blue-200 transition-colors"
                            title="Click to view LLM requests"
                          >
                            <Database className="w-3 h-3" />
                            <span className="font-medium">{debugSummary.total_tools_used}</span>
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 mt-1">
                    {new Date(conversation.updated_at).toLocaleDateString()}
                  </p>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteConversation(conversation.id);
                  }}
                  className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-100 rounded transition-all"
                >
                  <Trash2 className="w-4 h-4 text-red-500" />
                </button>
              </div>
            </div>
          ))}

          {conversations.length === 0 && (
            <div className="p-8 text-center text-gray-500">
              <MessageSquare className="w-12 h-12 mx-auto mb-3 text-gray-300" />
              <p className="text-sm">No conversations yet</p>
              <p className="text-xs mt-1">Create your first conversation to get started</p>
            </div>
          )}
        </div>

        {/* User Info */}
        {currentUser && (
          <div className="p-4 border-t border-gray-200">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                  <User className="w-4 h-4 text-blue-600" />
                </div>
                <div className="ml-3">
                  <p className="text-sm font-medium text-gray-900">{currentUser.username}</p>
                  <p className="text-xs text-gray-500">{currentUser.email}</p>
                </div>
              </div>
              {debugMode && (
                <div className="flex items-center gap-1">
                  <button
                    onClick={openDebugOverview}
                    className="flex items-center gap-1 px-2 py-1 bg-green-100 text-green-600 rounded hover:bg-green-200 transition-colors"
                    title="Debug mode active - Click for overview"
                  >
                    <Bug className="w-4 h-4" />
                    <span className="text-xs font-medium">Debug</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {activeConversation ? (
          <>
            {/* Chat Header */}
            <div className="px-6 py-4 border-b border-gray-200 bg-white">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <h2 className="text-lg font-semibold text-gray-900">
                    {activeConversation.title}
                  </h2>
                  {debugMode && debugSummary?.has_debug_data && (
                    <div className="flex items-center gap-2 text-sm text-gray-600">
                      <Clock className="w-4 h-4" />
                      <span>{debugSummary.total_steps} steps</span>
                      <span>•</span>
                      <span>{debugSummary.total_processing_time.toFixed(2)}s</span>
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {debugMode && (
                    <>
                      <button
                        onClick={openDebugSteps}
                        className="flex items-center gap-1 px-2 py-1 bg-purple-100 text-purple-800 rounded hover:bg-purple-200 transition-colors"
                        title="View Debug Steps (Ctrl/Cmd + D)"
                      >
                        <Zap className="w-3 h-3" />
                        <span className="text-xs font-medium">Steps</span>
                      </button>
                      <button
                        onClick={openDebugLLM}
                        className="flex items-center gap-1 px-2 py-1 bg-green-100 text-green-800 rounded hover:bg-green-200 transition-colors"
                        title="View LLM Requests (Ctrl/Cmd + L)"
                      >
                        <Database className="w-3 h-3" />
                        <span className="text-xs font-medium">LLM</span>
                      </button>
                      <button
                        onClick={openDebugOverview}
                        className="flex items-center gap-1 px-2 py-1 bg-blue-100 text-blue-800 rounded hover:bg-blue-200 transition-colors"
                        title="View Debug Overview (Ctrl/Cmd + Shift + D)"
                      >
                        <BarChart3 className="w-3 h-3" />
                        <span className="text-xs font-medium">Overview</span>
                      </button>
                      <button
                        onClick={() => setShowDebugSummary(true)}
                        className="flex items-center gap-1 px-2 py-1 bg-orange-100 text-orange-800 rounded hover:bg-orange-200 transition-colors"
                        title="View Debug Summary"
                      >
                        <History className="w-3 h-3" />
                        <span className="text-xs font-medium">Summary</span>
                      </button>
                      {debugSummary?.has_debug_data && (
                        <button
                          onClick={clearDebugData}
                          className="flex items-center gap-1 px-2 py-1 bg-red-100 text-red-800 rounded hover:bg-red-200 transition-colors"
                          title="Clear Debug Data"
                        >
                          <Archive className="w-3 h-3" />
                          <span className="text-xs font-medium">Clear</span>
                        </button>
                      )}
                    </>
                  )}
                </div>
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
              {debugMode && (
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3 mb-4">
                <div className="flex items-center gap-2">
                <Bug className="w-4 h-4 text-yellow-600" />
                <span className="text-sm font-medium text-yellow-800">Debug Mode Active</span>
                {isLoadingDebugData && (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-yellow-600"></div>
                )}
                </div>
                <p className="text-xs text-yellow-700 mt-1">
                Messages will include detailed processing steps, tool calls, and LLM requests. 
                Debug data is automatically saved and persists between sessions.
                </p>
                <div className="mt-2 flex items-center gap-2 text-xs text-yellow-700">
                <span>Quick access:</span>
                <kbd className="px-1 py-0.5 bg-yellow-200 rounded text-yellow-800 font-mono">Ctrl+D</kbd>
                <span>Steps</span>
                <kbd className="px-1 py-0.5 bg-yellow-200 rounded text-yellow-800 font-mono">Ctrl+L</kbd>
                <span>LLM</span>
                <kbd className="px-1 py-0.5 bg-yellow-200 rounded text-yellow-800 font-mono">Ctrl+Shift+D</kbd>
                  <span>Overview</span>
                  </div>
                {debugSummary?.has_debug_data && (
                  <div className="mt-2 flex items-center gap-4 text-xs text-yellow-700">
                    <span>Sessions: {debugSummary.total_sessions}</span>
                    <span>Steps: {debugSummary.total_steps}</span>
                    <span>Tools: {debugSummary.total_tools_used}</span>
                    <span>Time: {debugSummary.total_processing_time.toFixed(2)}s</span>
                  </div>
                )}
              </div>
              )}
              {messages.map(message => (
                <MessageBubble 
                  key={message.id} 
                  message={message} 
                  showDebugInfo={debugMode} 
                />
              ))}
              {isLoading && <LoadingMessage />}
              <div ref={messagesEndRef} />
            </div>

            {/* Message Input */}
            <div className="p-4 border-t border-gray-200 bg-white">
              <div className="flex items-center space-x-3">
                <input
                  type="text"
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                  placeholder="Type your message..."
                  className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                  disabled={isLoading}
                />
                <button
                  onClick={sendMessage}
                  disabled={isLoading || !newMessage.trim()}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  <Send className="w-4 h-4" />
                </button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <MessageSquare className="w-16 h-16 text-gray-300 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                No conversation selected
              </h3>
              <p className="text-gray-500">
                Choose a conversation from the sidebar or create a new one
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Memory Panel */}
      {showMemories && (
        <MemoryPanel
          memories={userMemories}
          onClose={() => setShowMemories(false)}
        />
      )}

      {/* Debug Panel */}
      {showDebugPanel && currentUser && (
        <MCPDebugPanel
          userId={currentUser.id}
          onClose={() => setShowDebugPanel(false)}
        />
      )}

      {/* Enhanced Debug Panel */}
      {showEnhancedDebugPanel && activeConversation && currentUser && (
        <EnhancedDebugPanel
          conversationId={activeConversation.id}
          userId={currentUser.id}
          initialTab={enhancedDebugPanelTab}
          onClose={() => setShowEnhancedDebugPanel(false)}
        />
      )}

      {/* Debug Summary */}
      {showDebugSummary && activeConversation && currentUser && (
        <DebugSummaryComponent
          conversationId={activeConversation.id}
          userId={currentUser.id}
          onClose={() => setShowDebugSummary(false)}
        />
      )}
    </div>
  );
}
