import React, { useState, useEffect, useRef } from 'react';
import { Send, MessageSquare, Plus, Trash2, User, Settings, Brain, Clock, Database } from 'lucide-react';
import { ApiService } from '../services/api';
import { User as UserType, Message, Conversation, UserMemory } from '../types';
import MessageBubble from './MessageBubble';
import LoadingMessage from './LoadingMessage';
import UserSetupModal from './UserSetupModal';
import MemoryPanel from './MemoryPanel';
import DebugPanel from './DebugPanel';

const api = new ApiService();

export default function AIAssistantApp() {
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
  const messagesEndRef = useRef<HTMLDivElement>(null);

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
      const response = await api.sendMessage({
        message: newMessage,
        user_id: currentUser.id,
        conversation_id: activeConversation?.id
      });

      setMessages(prev => [...prev, response.message]);

      // Update active conversation or create new one
      if (!activeConversation) {
        const updatedConvs = await api.getUserConversations(currentUser.id);
        setConversations(updatedConvs);
        const newConv = updatedConvs.find(c => c.id === response.conversation_id);
        if (newConv) setActiveConversation(newConv);
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
      }
    } catch (error) {
      console.error('Failed to delete conversation:', error);
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

  if (showUserSetup) {
    return <UserSetupModal onSetup={handleUserSetup} />;
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-bold text-gray-900">AI Assistant</h1>
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
              <Database className="w-5 h-5 text-gray-600" />
            </button>
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
                  <h3 className="text-sm font-medium text-gray-900 truncate">
                    {conversation.title}
                  </h3>
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
            <div className="flex items-center">
              <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                <User className="w-4 h-4 text-blue-600" />
              </div>
              <div className="ml-3">
                <p className="text-sm font-medium text-gray-900">{currentUser.username}</p>
                <p className="text-xs text-gray-500">{currentUser.email}</p>
              </div>
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
              <h2 className="text-lg font-semibold text-gray-900">
                {activeConversation.title}
              </h2>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
              {messages.map(message => (
                <MessageBubble key={message.id} message={message} />
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
        <DebugPanel
          userId={currentUser.id}
          onClose={() => setShowDebugPanel(false)}
        />
      )}
    </div>
  );
}
