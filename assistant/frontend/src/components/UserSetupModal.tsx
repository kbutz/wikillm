import React, { useState, useEffect } from 'react';
import { User, UserPlus, ArrowLeft, RefreshCw } from 'lucide-react';
import { ApiService } from '../services/api';

interface User {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
}

interface UserSetupModalProps {
  onSetup: (data: { username: string; email?: string; full_name?: string }) => void;
  onSelectUser?: (user: User) => void;
}

const api = new ApiService();

export default function UserSetupModal({ onSetup, onSelectUser }: UserSetupModalProps) {
  const [mode, setMode] = useState<'choose' | 'select' | 'create'>('choose');
  const [existingUsers, setExistingUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    full_name: ''
  });

  // Load existing users when component mounts or when switching to select mode
  useEffect(() => {
    if (mode === 'select') {
      loadExistingUsers();
    }
  }, [mode]);

  const loadExistingUsers = async () => {
    setLoading(true);
    setError(null);
    try {
      const users = await api.listUsers();
      setExistingUsers(users);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = (e: React.FormEvent | React.MouseEvent) => {
    e.preventDefault();
    if (formData.username.trim()) {
      onSetup(formData);
    }
  };

  const handleUserSelect = (user: User) => {
    if (onSelectUser) {
      onSelectUser(user);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center px-4">
      <div className="max-w-md w-full bg-white rounded-xl shadow-xl p-6 animate-slide-up">
        {/* Header */}
        <div className="text-center mb-6">
          <div className="w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <User className="w-8 h-8 text-white" />
          </div>
          <h2 className="text-2xl font-bold text-gray-900">Welcome to AI Assistant</h2>
          <p className="text-gray-600 mt-2">
            {mode === 'choose' && "Choose how you'd like to get started"}
            {mode === 'select' && "Select an existing user"}
            {mode === 'create' && "Create your profile to get started"}
          </p>
        </div>

        {/* Choose Mode */}
        {mode === 'choose' && (
          <div className="space-y-3">
            <button
              onClick={() => setMode('select')}
              className="w-full flex items-center justify-center px-4 py-3 bg-gradient-to-r from-blue-500 to-indigo-600 text-white rounded-lg hover:from-blue-600 hover:to-indigo-700 transition-all duration-200 font-medium shadow-lg hover:shadow-xl transform hover:scale-105"
            >
              <User className="w-5 h-5 mr-2" />
              Select Existing User
            </button>
            <button
              onClick={() => setMode('create')}
              className="w-full flex items-center justify-center px-4 py-3 bg-gradient-to-r from-green-500 to-emerald-600 text-white rounded-lg hover:from-green-600 hover:to-emerald-700 transition-all duration-200 font-medium shadow-lg hover:shadow-xl transform hover:scale-105"
            >
              <UserPlus className="w-5 h-5 mr-2" />
              Create New User
            </button>
          </div>
        )}

        {/* Select User Mode */}
        {mode === 'select' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <button
                onClick={() => setMode('choose')}
                className="flex items-center text-gray-600 hover:text-gray-800 transition-colors"
              >
                <ArrowLeft className="w-4 h-4 mr-1" />
                Back
              </button>
              <button
                onClick={loadExistingUsers}
                disabled={loading}
                className="flex items-center text-blue-600 hover:text-blue-700 transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </button>
            </div>

            {error && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-3 text-red-700 text-sm">
                {error}
              </div>
            )}

            {loading ? (
              <div className="text-center py-8">
                <div className="animate-spin w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full mx-auto mb-2"></div>
                <p className="text-gray-500">Loading users...</p>
              </div>
            ) : existingUsers.length > 0 ? (
              <div className="space-y-2 max-h-60 overflow-y-auto">
                {existingUsers.map(user => (
                  <div
                    key={user.id}
                    onClick={() => handleUserSelect(user)}
                    className="p-3 border border-gray-200 rounded-lg cursor-pointer hover:bg-blue-50 hover:border-blue-300 transition-all duration-200 group"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <h3 className="font-medium text-gray-900 group-hover:text-blue-700">
                          {user.username}
                        </h3>
                        {user.full_name && (
                          <p className="text-sm text-gray-600">{user.full_name}</p>
                        )}
                        {user.email && (
                          <p className="text-xs text-gray-500">{user.email}</p>
                        )}
                      </div>
                      <div className="text-right">
                        <p className="text-xs text-gray-500">
                          Created {formatDate(user.created_at)}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-gray-500">
                <User className="w-12 h-12 mx-auto mb-2 text-gray-300" />
                <p>No users found</p>
                <button
                  onClick={() => setMode('create')}
                  className="mt-2 text-blue-600 hover:text-blue-700 font-medium"
                >
                  Create the first user
                </button>
              </div>
            )}
          </div>
        )}

        {/* Create User Mode */}
        {mode === 'create' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <button
                onClick={() => setMode('choose')}
                className="flex items-center text-gray-600 hover:text-gray-800 transition-colors"
              >
                <ArrowLeft className="w-4 h-4 mr-1" />
                Back
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Username *
                </label>
                <input
                  type="text"
                  required
                  value={formData.username}
                  onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
                  onKeyDown={(e) => e.key === 'Enter' && handleSubmit(e)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
                  placeholder="Enter your username"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Email
                </label>
                <input
                  type="email"
                  value={formData.email}
                  onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
                  onKeyDown={(e) => e.key === 'Enter' && handleSubmit(e)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
                  placeholder="Enter your email"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Full Name
                </label>
                <input
                  type="text"
                  value={formData.full_name}
                  onChange={(e) => setFormData(prev => ({ ...prev, full_name: e.target.value }))}
                  onKeyDown={(e) => e.key === 'Enter' && handleSubmit(e)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none transition-all"
                  placeholder="Enter your full name"
                />
              </div>

              <button
                onClick={handleSubmit}
                disabled={!formData.username.trim()}
                className="w-full bg-gradient-to-r from-blue-500 to-indigo-600 text-white py-2 px-4 rounded-lg hover:from-blue-600 hover:to-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium shadow-lg hover:shadow-xl transform hover:scale-105 disabled:transform-none"
              >
                Create User
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
