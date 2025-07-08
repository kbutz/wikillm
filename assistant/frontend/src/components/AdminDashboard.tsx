import React, { useState, useEffect } from 'react';
import { Users, Database, MessageSquare, Settings, Download, Search, Plus, Eye, Edit, Trash2, ArrowLeft } from 'lucide-react';
import { AdminService, AdminUser, AdminConversation, AdminMemory, AdminSystemStats } from '../services/admin';
import MemoryInspector from './MemoryInspector';

const adminService = new AdminService();

interface AdminDashboardProps {
  onBack: () => void;
}

export default function AdminDashboard({ onBack }: AdminDashboardProps) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
  const [activeTab, setActiveTab] = useState<'users' | 'memory' | 'conversations' | 'system'>('users');
  const [searchTerm, setSearchTerm] = useState('');
  const [userMemory, setUserMemory] = useState<AdminMemory | null>(null);
  const [userConversations, setUserConversations] = useState<AdminConversation[]>([]);
  const [systemStats, setSystemStats] = useState<AdminSystemStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [newUserData, setNewUserData] = useState({ username: '', email: '', full_name: '' });
  const [showMemoryInspector, setShowMemoryInspector] = useState(false);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const data = await adminService.getAdminUsers();
      setUsers(data);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    }
    setLoading(false);
  };

  const fetchUserMemory = async (userId: number) => {
    setLoading(true);
    try {
      const data = await adminService.getUserMemory(userId);
      setUserMemory(data);
    } catch (error) {
      console.error('Failed to fetch user memory:', error);
    }
    setLoading(false);
  };

  const fetchUserConversations = async (userId: number) => {
    setLoading(true);
    try {
      const data = await adminService.getAdminUserConversations(userId);
      setUserConversations(data);
    } catch (error) {
      console.error('Failed to fetch user conversations:', error);
    }
    setLoading(false);
  };

  const fetchSystemStats = async () => {
    setLoading(true);
    try {
      const data = await adminService.getSystemStats();
      setSystemStats(data);
    } catch (error) {
      console.error('Failed to fetch system stats:', error);
    }
    setLoading(false);
  };

  const createUser = async () => {
    if (!newUserData.username.trim()) return;
    
    try {
      await adminService.createAdminUser(newUserData);
      setNewUserData({ username: '', email: '', full_name: '' });
      setShowCreateUser(false);
      fetchUsers();
    } catch (error) {
      console.error('Failed to create user:', error);
    }
  };

  const deleteUser = async (userId: number) => {
    if (!confirm('Are you sure you want to delete this user and all their data?')) return;
    
    try {
      await adminService.deleteAdminUser(userId);
      setUsers(users.filter(u => u.id !== userId));
      if (selectedUser?.id === userId) {
        setSelectedUser(null);
      }
    } catch (error) {
      console.error('Failed to delete user:', error);
    }
  };

  const exportUserData = async (userId: number) => {
    try {
      await adminService.exportUserData(userId);
    } catch (error) {
      console.error('Failed to export user data:', error);
    }
  };

  const impersonateUser = async (userId: number) => {
    try {
      const result = await adminService.impersonateUser(userId);
      console.log('Impersonation result:', result);
      // Here you could switch the main app to impersonate this user
    } catch (error) {
      console.error('Failed to impersonate user:', error);
    }
  };

  useEffect(() => {
    fetchUsers();
    fetchSystemStats();
  }, []);

  useEffect(() => {
    if (selectedUser) {
      if (activeTab === 'memory') {
        fetchUserMemory(selectedUser.id);
      } else if (activeTab === 'conversations') {
        fetchUserConversations(selectedUser.id);
      }
    }
  }, [selectedUser, activeTab]);

  const filteredUsers = users.filter(user =>
    user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
    (user.email && user.email.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  const TabButton = ({ id, label, icon: Icon }: { id: string; label: string; icon: any }) => (
    <button
      onClick={() => setActiveTab(id as any)}
      className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
        activeTab === id 
          ? 'bg-blue-600 text-white' 
          : 'bg-gray-100 hover:bg-gray-200 text-gray-700'
      }`}
    >
      <Icon size={18} />
      {label}
    </button>
  );

  const UserCard = ({ user }: { user: AdminUser }) => (
    <div 
      className={`p-4 border rounded-lg cursor-pointer transition-all ${
        selectedUser?.id === user.id 
          ? 'border-blue-500 bg-blue-50' 
          : 'border-gray-200 hover:border-gray-300'
      }`}
      onClick={() => setSelectedUser(user)}
    >
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="font-semibold text-lg">{user.username}</h3>
          <p className="text-gray-600 text-sm">{user.email || 'No email'}</p>
          {user.full_name && <p className="text-gray-500 text-sm">{user.full_name}</p>}
        </div>
        <div className="flex gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              exportUserData(user.id);
            }}
            className="p-2 hover:bg-gray-100 rounded"
            title="Export user data"
          >
            <Download size={16} />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              impersonateUser(user.id);
            }}
            className="p-2 hover:bg-blue-100 rounded"
            title="Impersonate user"
          >
            <Eye size={16} />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              deleteUser(user.id);
            }}
            className="p-2 hover:bg-red-100 rounded text-red-600"
            title="Delete user"
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-gray-500">Created:</span>
          <span className="ml-2">{new Date(user.created_at).toLocaleDateString()}</span>
        </div>
        <div>
          <span className="text-gray-500">Last Active:</span>
          <span className="ml-2">
            {user.last_active ? new Date(user.last_active).toLocaleDateString() : 'Never'}
          </span>
        </div>
        <div>
          <span className="text-gray-500">Conversations:</span>
          <span className="ml-2">{user.conversation_count}</span>
        </div>
        <div>
          <span className="text-gray-500">Memory:</span>
          <span className="ml-2">{Math.round(user.memory_size / 1024)}KB</span>
        </div>
      </div>
    </div>
  );

  const MemorySection = ({ title, data }: { title: string; data: any }) => (
    <div className="border rounded-lg p-4 mb-4">
      <h4 className="font-semibold mb-3">{title}</h4>
      <pre className="bg-gray-50 p-3 rounded text-sm overflow-x-auto">
        {JSON.stringify(data, null, 2)}
      </pre>
    </div>
  );

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="mb-6">
        <div className="flex items-center gap-4 mb-2">
          <button
            onClick={onBack}
            className="p-2 hover:bg-gray-100 rounded-lg"
            title="Back to chat"
          >
            <ArrowLeft size={20} />
          </button>
          <h1 className="text-3xl font-bold">WikiLLM Admin Dashboard</h1>
        </div>
        <p className="text-gray-600">Manage users, inspect memory, and monitor system activity</p>
      </div>

      {/* System Stats Overview */}
      {systemStats && (
        <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6">
          <div className="bg-blue-50 p-4 rounded-lg">
            <div className="text-2xl font-bold text-blue-600">{systemStats.total_users}</div>
            <div className="text-sm text-gray-600">Total Users</div>
          </div>
          <div className="bg-green-50 p-4 rounded-lg">
            <div className="text-2xl font-bold text-green-600">{systemStats.active_users}</div>
            <div className="text-sm text-gray-600">Active Users</div>
          </div>
          <div className="bg-purple-50 p-4 rounded-lg">
            <div className="text-2xl font-bold text-purple-600">{systemStats.total_conversations}</div>
            <div className="text-sm text-gray-600">Conversations</div>
          </div>
          <div className="bg-orange-50 p-4 rounded-lg">
            <div className="text-2xl font-bold text-orange-600">{systemStats.total_messages}</div>
            <div className="text-sm text-gray-600">Messages</div>
          </div>
          <div className="bg-red-50 p-4 rounded-lg">
            <div className="text-2xl font-bold text-red-600">{systemStats.total_memory_entries}</div>
            <div className="text-sm text-gray-600">Memory Entries</div>
          </div>
        </div>
      )}

      {/* Navigation Tabs */}
      <div className="flex gap-4 mb-6">
        <TabButton id="users" label="Users" icon={Users} />
        <TabButton id="memory" label="Memory" icon={Database} />
        <TabButton id="conversations" label="Conversations" icon={MessageSquare} />
        <TabButton id="system" label="System" icon={Settings} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Panel - User List */}
        <div className="lg:col-span-1">
          <div className="mb-4">
            <div className="flex gap-2 mb-4">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search users..."
                  className="w-full pl-10 pr-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
              <button
                onClick={() => setShowCreateUser(true)}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
              >
                <Plus size={16} />
                Add User
              </button>
            </div>
          </div>

          <div className="space-y-4 max-h-96 overflow-y-auto">
            {loading ? (
              <div className="text-center py-8">Loading users...</div>
            ) : filteredUsers.length > 0 ? (
              filteredUsers.map(user => (
                <UserCard key={user.id} user={user} />
              ))
            ) : (
              <div className="text-center py-8 text-gray-500">No users found</div>
            )}
          </div>
        </div>

        {/* Right Panel - User Details */}
        <div className="lg:col-span-2">
          {selectedUser ? (
            <div>
              <div className="mb-6">
                <h2 className="text-2xl font-bold mb-2">{selectedUser.username}</h2>
                <p className="text-gray-600">User ID: {selectedUser.id}</p>
              </div>

              {activeTab === 'users' && (
                <div>
                  <h3 className="text-lg font-semibold mb-4">User Overview</h3>
                  <div className="grid grid-cols-2 gap-4 mb-6">
                    <div className="p-4 bg-blue-50 rounded-lg">
                      <div className="text-2xl font-bold text-blue-600">{selectedUser.conversation_count}</div>
                      <div className="text-sm text-gray-600">Total Conversations</div>
                    </div>
                    <div className="p-4 bg-green-50 rounded-lg">
                      <div className="text-2xl font-bold text-green-600">{Math.round(selectedUser.memory_size / 1024)}KB</div>
                      <div className="text-sm text-gray-600">Memory Usage</div>
                    </div>
                  </div>
                  <div className="space-y-4">
                    <div className="border rounded-lg p-4">
                      <h4 className="font-semibold mb-2">User Details</h4>
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div><strong>Username:</strong> {selectedUser.username}</div>
                        <div><strong>Email:</strong> {selectedUser.email || 'Not set'}</div>
                        <div><strong>Full Name:</strong> {selectedUser.full_name || 'Not set'}</div>
                        <div><strong>Created:</strong> {new Date(selectedUser.created_at).toLocaleString()}</div>
                        <div><strong>Last Updated:</strong> {new Date(selectedUser.updated_at).toLocaleString()}</div>
                        <div><strong>Memory Entries:</strong> {selectedUser.memory_entries}</div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {activeTab === 'memory' && (
                <div>
                  <h3 className="text-lg font-semibold mb-4">Memory Inspection</h3>
                  {loading ? (
                    <div className="text-center py-8">Loading memory data...</div>
                  ) : userMemory ? (
                    <div>
                      <div className="mb-4 p-4 bg-gray-50 rounded-lg">
                        <div className="text-sm text-gray-600">
                          <strong>Total Size:</strong> {Math.round(userMemory.size / 1024)}KB |{' '}
                          <strong>Last Updated:</strong> {new Date(userMemory.last_updated).toLocaleString()}
                        </div>
                      </div>
                      <div className="mb-4">
                        <button
                          onClick={() => setShowMemoryInspector(true)}
                          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
                        >
                          <Database size={16} />
                          Open Memory Inspector
                        </button>
                      </div>
                      <MemorySection title="Personal Information" data={userMemory.personal_info} />
                      <MemorySection title="Conversation History" data={userMemory.conversation_history} />
                      <MemorySection title="Context Memory" data={userMemory.context_memory} />
                      <MemorySection title="Preferences" data={userMemory.preferences} />
                    </div>
                  ) : (
                    <div className="text-center py-8 text-gray-500">No memory data available</div>
                  )}
                </div>
              )}

              {activeTab === 'conversations' && (
                <div>
                  <h3 className="text-lg font-semibold mb-4">Conversation History</h3>
                  {loading ? (
                    <div className="text-center py-8">Loading conversations...</div>
                  ) : userConversations.length > 0 ? (
                    <div className="space-y-4">
                      {userConversations.map(conv => (
                        <div key={conv.id} className="border rounded-lg p-4">
                          <div className="flex justify-between items-start mb-2">
                            <h4 className="font-semibold">{conv.title}</h4>
                            <div className="flex gap-2">
                              <button
                                onClick={() => console.log('View conversation', conv.id)}
                                className="p-1 hover:bg-gray-100 rounded"
                                title="View conversation"
                              >
                                <Eye size={16} />
                              </button>
                              <button
                                onClick={() => {
                                  if (confirm('Are you sure you want to delete this conversation?')) {
                                    adminService.deleteAdminConversation(conv.id).then(() => {
                                      fetchUserConversations(selectedUser.id);
                                    });
                                  }
                                }}
                                className="p-1 hover:bg-gray-100 rounded text-red-600"
                                title="Delete conversation"
                              >
                                <Trash2 size={16} />
                              </button>
                            </div>
                          </div>
                          <div className="text-sm text-gray-600 mb-2">
                            Created: {new Date(conv.created_at).toLocaleString()} | {conv.message_count} messages
                          </div>
                          {conv.last_message && (
                            <div className="text-sm text-gray-700 italic">
                              "{conv.last_message}"
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-center py-8 text-gray-500">No conversations found</div>
                  )}
                </div>
              )}

              {activeTab === 'system' && (
                <div>
                  <h3 className="text-lg font-semibold mb-4">System Operations</h3>
                  <div className="space-y-4">
                    <button
                      onClick={() => {
                        if (confirm('Are you sure you want to clear all memory for this user?')) {
                          adminService.clearUserMemory(selectedUser.id).then(() => {
                            fetchUserMemory(selectedUser.id);
                          });
                        }
                      }}
                      className="w-full p-4 border border-red-200 rounded-lg hover:bg-red-50 text-left"
                    >
                      <div className="font-semibold text-red-600">Clear All User Memory</div>
                      <div className="text-sm text-gray-600">Permanently delete all memory entries</div>
                    </button>
                    <button
                      onClick={() => exportUserData(selectedUser.id)}
                      className="w-full p-4 border border-blue-200 rounded-lg hover:bg-blue-50 text-left"
                    >
                      <div className="font-semibold text-blue-600">Export All Data</div>
                      <div className="text-sm text-gray-600">Download complete user data archive</div>
                    </button>
                    <button
                      onClick={() => impersonateUser(selectedUser.id)}
                      className="w-full p-4 border border-green-200 rounded-lg hover:bg-green-50 text-left"
                    >
                      <div className="font-semibold text-green-600">Impersonate User</div>
                      <div className="text-sm text-gray-600">Switch to this user's perspective</div>
                    </button>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center py-12">
              <Users className="mx-auto h-12 w-12 text-gray-400 mb-4" />
              <h3 className="text-lg font-semibold text-gray-600 mb-2">No User Selected</h3>
              <p className="text-gray-500">Select a user from the list to view their details</p>
            </div>
          )}
        </div>
      </div>

      {/* Create User Modal */}
      {showCreateUser && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">Create New User</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Username *
                </label>
                <input
                  type="text"
                  value={newUserData.username}
                  onChange={(e) => setNewUserData({ ...newUserData, username: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Email
                </label>
                <input
                  type="email"
                  value={newUserData.email}
                  onChange={(e) => setNewUserData({ ...newUserData, email: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Full Name
                </label>
                <input
                  type="text"
                  value={newUserData.full_name}
                  onChange={(e) => setNewUserData({ ...newUserData, full_name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <button
                onClick={createUser}
                className="flex-1 bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700"
              >
                Create User
              </button>
              <button
                onClick={() => setShowCreateUser(false)}
                className="flex-1 bg-gray-300 text-gray-700 py-2 px-4 rounded-md hover:bg-gray-400"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
      
      {/* Memory Inspector */}
      {showMemoryInspector && selectedUser && (
        <MemoryInspector
          userId={selectedUser.id}
          onClose={() => setShowMemoryInspector(false)}
        />
      )}
    </div>
  );
}
