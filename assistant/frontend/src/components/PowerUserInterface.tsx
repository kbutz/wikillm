import React, { useState, useEffect } from 'react';
import { 
  Users, Database, Brain, MessageSquare, Settings, Activity, 
  TrendingUp, Clock, Eye, Edit3, Trash2, Plus, Search, Filter,
  User, ArrowRight, BarChart3, PieChart, GitBranch, ArrowLeft,
  Calendar, Tag, Star, Zap, Shield, BookOpen, Network, Download
} from 'lucide-react';
import { powerUserApi, PowerUserData, UserAnalytics } from '../services/powerUserApi';
import { User as UserType, UserMemory, Conversation, UserPreference } from '../types';

interface PowerUserInterfaceProps {
  onBack: () => void;
}

const PowerUserInterface: React.FC<PowerUserInterfaceProps> = ({ onBack }) => {
  const [users, setUsers] = useState<UserType[]>([]);
  const [selectedUser, setSelectedUser] = useState<UserType | null>(null);
  const [userData, setUserData] = useState<PowerUserData | null>(null);
  const [activeTab, setActiveTab] = useState('overview');
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [newUserData, setNewUserData] = useState({ username: '', email: '', full_name: '' });
  const [error, setError] = useState<string | null>(null);

  // Load users on component mount
  useEffect(() => {
    loadUsers();
  }, []);

  // Load user data when selected user changes
  useEffect(() => {
    if (selectedUser) {
      loadUserData(selectedUser.id);
    }
  }, [selectedUser]);

  const loadUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      const usersData = await powerUserApi.getAllUsers();
      setUsers(usersData);
      if (usersData.length > 0 && !selectedUser) {
        setSelectedUser(usersData[0]);
      }
    } catch (error) {
      console.error('Failed to load users:', error);
      setError('Failed to load users. Please check your connection and try again.');
    } finally {
      setLoading(false);
    }
  };

  const loadUserData = async (userId: number) => {
    try {
      const data = await powerUserApi.getUserData(userId);
      setUserData(data);
    } catch (error) {
      console.error('Failed to load user data:', error);
      setError('Failed to load user data. Some features may not be available.');
    }
  };

  const handleCreateUser = async () => {
    if (!newUserData.username) return;
    
    try {
      const user = await powerUserApi.createUser(newUserData);
      setUsers(prev => [...prev, user]);
      setNewUserData({ username: '', email: '', full_name: '' });
      setShowCreateUser(false);
    } catch (error) {
      console.error('Failed to create user:', error);
      setError('Failed to create user. Please try again.');
    }
  };

  const handleSwitchUser = async (user: UserType) => {
    try {
      await powerUserApi.switchActiveUser(user.id);
      setSelectedUser(user);
      localStorage.setItem('currentUserId', user.id.toString());
    } catch (error) {
      console.error('Failed to switch user:', error);
      setError('Failed to switch user. Please try again.');
    }
  };

  const handleDeleteUser = async (userId: number) => {
    if (!window.confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
      return;
    }
    
    try {
      await powerUserApi.deleteUser(userId);
      setUsers(prev => prev.filter(u => u.id !== userId));
      if (selectedUser?.id === userId) {
        const remainingUsers = users.filter(u => u.id !== userId);
        setSelectedUser(remainingUsers.length > 0 ? remainingUsers[0] : null);
      }
    } catch (error) {
      console.error('Failed to delete user:', error);
      setError('Failed to delete user. Please try again.');
    }
  };

  const handleExportUserData = async (userId: number, format: 'json' | 'csv' = 'json') => {
    try {
      const blob = await powerUserApi.exportUserData(userId, format);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `user_${userId}_data.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Failed to export user data:', error);
      setError('Failed to export user data. Please try again.');
    }
  };

  const filteredUsers = users.filter(user => 
    user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.email?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.full_name?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Data visualization components
  const MemoryVisualization = ({ memories }: { memories: UserMemory[] }) => (
    <div className="bg-white rounded-lg p-6 shadow-sm border">
      <h3 className="text-lg font-semibold mb-4 flex items-center">
        <Brain className="w-5 h-5 mr-2 text-purple-600" />
        Memory Analysis ({memories.length} entries)
      </h3>
      <div className="space-y-4 max-h-96 overflow-y-auto">
        {memories.slice(0, 10).map(memory => (
          <div key={memory.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className={`px-2 py-1 rounded text-xs font-medium ${
                  memory.memory_type === 'explicit' ? 'bg-blue-100 text-blue-800' :
                  memory.memory_type === 'implicit' ? 'bg-green-100 text-green-800' :
                  'bg-purple-100 text-purple-800'
                }`}>
                  {memory.memory_type}
                </span>
                <span className="font-medium">{memory.key}</span>
              </div>
              <p className="text-sm text-gray-600 mt-1 truncate">{memory.value}</p>
            </div>
            <div className="text-right">
              <div className="text-sm font-medium">{Math.round(memory.confidence * 100)}%</div>
              <div className="text-xs text-gray-500">{memory.access_count} accesses</div>
            </div>
          </div>
        ))}
        {memories.length > 10 && (
          <div className="text-center text-gray-500 text-sm">
            ... and {memories.length - 10} more memories
          </div>
        )}
      </div>
    </div>
  );

  const ConversationTimeline = ({ conversations }: { conversations: Conversation[] }) => (
    <div className="bg-white rounded-lg p-6 shadow-sm border">
      <h3 className="text-lg font-semibold mb-4 flex items-center">
        <GitBranch className="w-5 h-5 mr-2 text-blue-600" />
        Conversation Timeline ({conversations.length} conversations)
      </h3>
      <div className="space-y-4 max-h-96 overflow-y-auto">
        {conversations.slice(0, 10).map(conv => (
          <div key={conv.id} className="flex items-center p-3 bg-gray-50 rounded-lg">
            <div className={`w-3 h-3 rounded-full mr-3 ${conv.is_active ? 'bg-green-500' : 'bg-gray-400'}`}></div>
            <div className="flex-1">
              <h4 className="font-medium">{conv.title}</h4>
              <p className="text-sm text-gray-600">{conv.messages?.length || 0} messages</p>
            </div>
            <div className="text-right">
              <div className="text-sm text-gray-500">
                {new Date(conv.updated_at).toLocaleDateString()}
              </div>
            </div>
          </div>
        ))}
        {conversations.length > 10 && (
          <div className="text-center text-gray-500 text-sm">
            ... and {conversations.length - 10} more conversations
          </div>
        )}
      </div>
    </div>
  );

  const AnalyticsDashboard = ({ analytics }: { analytics: UserAnalytics }) => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <div className="bg-white rounded-lg p-4 shadow-sm border">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-gray-600">Total Messages</p>
            <p className="text-2xl font-bold text-blue-600">{analytics.totalMessages}</p>
          </div>
          <MessageSquare className="w-8 h-8 text-blue-600" />
        </div>
      </div>
      
      <div className="bg-white rounded-lg p-4 shadow-sm border">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-gray-600">Avg Response Time</p>
            <p className="text-2xl font-bold text-green-600">{analytics.averageResponseTime}s</p>
          </div>
          <Clock className="w-8 h-8 text-green-600" />
        </div>
      </div>
      
      <div className="bg-white rounded-lg p-4 shadow-sm border">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-gray-600">Memory Usage</p>
            <p className="text-2xl font-bold text-purple-600">{Math.round(analytics.memoryUtilization * 100)}%</p>
          </div>
          <Brain className="w-8 h-8 text-purple-600" />
        </div>
      </div>
      
      <div className="bg-white rounded-lg p-4 shadow-sm border">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-gray-600">Engagement</p>
            <p className="text-2xl font-bold text-orange-600">{Math.round(analytics.conversationEngagement * 100)}%</p>
          </div>
          <TrendingUp className="w-8 h-8 text-orange-600" />
        </div>
      </div>
    </div>
  );

  const DataRelationshipGraph = ({ userData }: { userData: PowerUserData }) => (
    <div className="bg-white rounded-lg p-6 shadow-sm border">
      <h3 className="text-lg font-semibold mb-4 flex items-center">
        <Network className="w-5 h-5 mr-2 text-indigo-600" />
        Data Relationships
      </h3>
      <div className="relative h-64 bg-gray-50 rounded-lg p-4">
        <div className="absolute top-4 left-4 bg-blue-500 text-white px-3 py-2 rounded-full text-sm">
          User Profile
        </div>
        <div className="absolute top-4 right-4 bg-green-500 text-white px-3 py-2 rounded-full text-sm">
          {userData.conversations.length} Conversations
        </div>
        <div className="absolute bottom-4 left-4 bg-purple-500 text-white px-3 py-2 rounded-full text-sm">
          {userData.memories.length} Memories
        </div>
        <div className="absolute bottom-4 right-4 bg-orange-500 text-white px-3 py-2 rounded-full text-sm">
          {userData.preferences.length} Preferences
        </div>
        
        <svg className="absolute inset-0 w-full h-full">
          <line x1="120" y1="30" x2="200" y2="30" stroke="#94a3b8" strokeWidth="2" strokeDasharray="5,5" />
          <line x1="100" y1="50" x2="100" y2="200" stroke="#94a3b8" strokeWidth="2" strokeDasharray="5,5" />
          <line x1="220" y1="50" x2="220" y2="200" stroke="#94a3b8" strokeWidth="2" strokeDasharray="5,5" />
          <line x1="120" y1="220" x2="200" y2="220" stroke="#94a3b8" strokeWidth="2" strokeDasharray="5,5" />
        </svg>
      </div>
    </div>
  );

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading power user tools...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <button
                onClick={onBack}
                className="mr-4 p-2 hover:bg-gray-100 rounded-lg transition-colors"
              >
                <ArrowLeft className="w-5 h-5 text-gray-600" />
              </button>
              <Shield className="w-8 h-8 text-blue-600 mr-3" />
              <h1 className="text-xl font-bold text-gray-900">Power User Console</h1>
            </div>
            <div className="flex items-center space-x-4">
              <button
                onClick={() => setShowCreateUser(true)}
                className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center"
              >
                <Plus className="w-4 h-4 mr-2" />
                Create User
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
            {error}
            <button
              onClick={() => setError(null)}
              className="float-right text-red-500 hover:text-red-700"
            >
              ×
            </button>
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* User Management Sidebar */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-lg shadow-sm border">
              <div className="p-4 border-b">
                <h2 className="text-lg font-semibold flex items-center">
                  <Users className="w-5 h-5 mr-2 text-blue-600" />
                  User Management
                </h2>
                <div className="mt-3">
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
                    <input
                      type="text"
                      placeholder="Search users..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                </div>
              </div>
              
              <div className="p-2 max-h-96 overflow-y-auto">
                {filteredUsers.map(user => (
                  <div
                    key={user.id}
                    className={`p-3 rounded-lg cursor-pointer transition-colors ${
                      selectedUser?.id === user.id 
                        ? 'bg-blue-50 border-blue-200 border' 
                        : 'hover:bg-gray-50'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center" onClick={() => handleSwitchUser(user)}>
                        <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                          <User className="w-4 h-4 text-blue-600" />
                        </div>
                        <div className="ml-3">
                          <p className="text-sm font-medium text-gray-900">{user.username}</p>
                          <p className="text-xs text-gray-500">{user.email}</p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-1">
                        {selectedUser?.id === user.id && (
                          <div className="w-2 h-2 bg-blue-600 rounded-full"></div>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteUser(user.id);
                          }}
                          className="p-1 hover:bg-red-100 rounded text-red-500 hover:text-red-700"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Main Content Area */}
          <div className="lg:col-span-3">
            {selectedUser && userData && (
              <>
                {/* User Info Header */}
                <div className="bg-white rounded-lg shadow-sm border p-6 mb-6">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
                        <User className="w-6 h-6 text-blue-600" />
                      </div>
                      <div className="ml-4">
                        <h2 className="text-xl font-bold text-gray-900">{selectedUser.full_name || selectedUser.username}</h2>
                        <p className="text-gray-600">{selectedUser.email}</p>
                        <p className="text-sm text-gray-500">
                          Active since {new Date(selectedUser.created_at).toLocaleDateString()}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => handleExportUserData(selectedUser.id, 'json')}
                        className="p-2 hover:bg-gray-100 rounded-lg"
                        title="Export user data as JSON"
                      >
                        <Download className="w-5 h-5 text-gray-600" />
                      </button>
                      <button className="p-2 hover:bg-gray-100 rounded-lg">
                        <Edit3 className="w-5 h-5 text-gray-600" />
                      </button>
                      <button className="p-2 hover:bg-gray-100 rounded-lg">
                        <Settings className="w-5 h-5 text-gray-600" />
                      </button>
                    </div>
                  </div>
                </div>

                {/* Analytics Dashboard */}
                <AnalyticsDashboard analytics={userData.analytics} />

                {/* Navigation Tabs */}
                <div className="bg-white rounded-lg shadow-sm border mb-6">
                  <div className="border-b">
                    <nav className="flex space-x-8 px-6">
                      {[
                        { id: 'overview', label: 'Overview', icon: Activity },
                        { id: 'memories', label: 'Memories', icon: Brain },
                        { id: 'conversations', label: 'Conversations', icon: MessageSquare },
                        { id: 'preferences', label: 'Preferences', icon: Settings },
                        { id: 'relationships', label: 'Data Graph', icon: Network }
                      ].map(tab => (
                        <button
                          key={tab.id}
                          onClick={() => setActiveTab(tab.id)}
                          className={`flex items-center px-1 py-4 text-sm font-medium border-b-2 ${
                            activeTab === tab.id
                              ? 'border-blue-500 text-blue-600'
                              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                          }`}
                        >
                          <tab.icon className="w-4 h-4 mr-2" />
                          {tab.label}
                        </button>
                      ))}
                    </nav>
                  </div>

                  {/* Tab Content */}
                  <div className="p-6">
                    {activeTab === 'overview' && (
                      <div className="space-y-6">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          <div className="bg-gray-50 rounded-lg p-4">
                            <h3 className="font-semibold mb-3">Recent Activity</h3>
                            <div className="space-y-2">
                              <div className="flex items-center text-sm">
                                <Clock className="w-4 h-4 mr-2 text-gray-500" />
                                Most active at: {userData.analytics.mostActiveHour}:00
                              </div>
                              <div className="flex items-center text-sm">
                                <MessageSquare className="w-4 h-4 mr-2 text-gray-500" />
                                Total messages: {userData.analytics.totalMessages}
                              </div>
                              <div className="flex items-center text-sm">
                                <Star className="w-4 h-4 mr-2 text-gray-500" />
                                Top topics: {userData.analytics.topicsDiscussed.slice(0, 3).join(', ')}
                              </div>
                            </div>
                          </div>
                          <div className="bg-gray-50 rounded-lg p-4">
                            <h3 className="font-semibold mb-3">Data Summary</h3>
                            <div className="space-y-2">
                              <div className="flex justify-between text-sm">
                                <span>Conversations:</span>
                                <span className="font-medium">{userData.conversations.length}</span>
                              </div>
                              <div className="flex justify-between text-sm">
                                <span>Memories:</span>
                                <span className="font-medium">{userData.memories.length}</span>
                              </div>
                              <div className="flex justify-between text-sm">
                                <span>Preferences:</span>
                                <span className="font-medium">{userData.preferences.length}</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}

                    {activeTab === 'memories' && (
                      <MemoryVisualization memories={userData.memories} />
                    )}

                    {activeTab === 'conversations' && (
                      <ConversationTimeline conversations={userData.conversations} />
                    )}

                    {activeTab === 'preferences' && (
                      <div className="bg-gray-50 rounded-lg p-4">
                        <h3 className="font-semibold mb-4">User Preferences ({userData.preferences.length})</h3>
                        <div className="space-y-3">
                          {userData.preferences.map(pref => (
                            <div key={pref.id} className="flex items-center justify-between p-3 bg-white rounded-lg">
                              <div>
                                <span className="font-medium">{pref.category}</span>
                                <span className="text-gray-500 mx-2">•</span>
                                <span className="text-gray-700">{pref.key}</span>
                              </div>
                              <div className="text-sm bg-gray-100 px-3 py-1 rounded">
                                {typeof pref.value === 'object' ? JSON.stringify(pref.value) : pref.value}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {activeTab === 'relationships' && (
                      <DataRelationshipGraph userData={userData} />
                    )}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Create User Modal */}
      {showCreateUser && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">Create New User</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Username</label>
                <input
                  type="text"
                  value={newUserData.username}
                  onChange={(e) => setNewUserData(prev => ({ ...prev, username: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter username"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Email</label>
                <input
                  type="email"
                  value={newUserData.email}
                  onChange={(e) => setNewUserData(prev => ({ ...prev, email: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter email"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Full Name</label>
                <input
                  type="text"
                  value={newUserData.full_name}
                  onChange={(e) => setNewUserData(prev => ({ ...prev, full_name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter full name"
                />
              </div>
            </div>
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowCreateUser(false)}
                className="px-4 py-2 text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateUser}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                Create User
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default PowerUserInterface;
