import React, { useState, useEffect } from 'react';
import { Database, Edit, Save, Trash2, Plus, X, RefreshCw } from 'lucide-react';
import { UserMemory } from '../types';
import { ApiService } from '../services/api';

interface DebugPanelProps {
  userId: number;
  onClose: () => void;
}

const api = new ApiService();

export default function DebugPanel({ userId, onClose }: DebugPanelProps) {
  const [memories, setMemories] = useState<UserMemory[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingMemory, setEditingMemory] = useState<UserMemory | null>(null);
  const [newMemory, setNewMemory] = useState<{
    memory_type: 'explicit' | 'implicit' | 'preference';
    key: string;
    value: string;
    confidence: number;
    source: string;
  }>({
    memory_type: 'explicit',
    key: '',
    value: '',
    confidence: 1.0,
    source: 'debug_panel'
  });
  const [showAddForm, setShowAddForm] = useState(false);

  useEffect(() => {
    loadMemories();
  }, [userId]);

  const loadMemories = async () => {
    setLoading(true);
    try {
      const data = await api.getUserMemory(userId);
      setMemories(data);
    } catch (error) {
      console.error('Failed to load memories:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleEditMemory = (memory: UserMemory) => {
    setEditingMemory(memory);
  };

  const handleSaveEdit = async () => {
    if (!editingMemory) return;

    try {
      const updated = await api.updateUserMemory(userId, editingMemory.id, {
        memory_type: editingMemory.memory_type as 'explicit' | 'implicit' | 'preference',
        key: editingMemory.key,
        value: editingMemory.value,
        confidence: editingMemory.confidence,
        source: editingMemory.source
      });

      setMemories(memories.map(m => m.id === updated.id ? updated : m));
      setEditingMemory(null);
    } catch (error) {
      console.error('Failed to update memory:', error);
    }
  };

  const handleCancelEdit = () => {
    setEditingMemory(null);
  };

  const handleDeleteMemory = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this memory?')) return;

    try {
      await api.deleteUserMemory(userId, id);
      setMemories(memories.filter(m => m.id !== id));
    } catch (error) {
      console.error('Failed to delete memory:', error);
    }
  };

  const handleAddMemory = async () => {
    try {
      const created = await api.addUserMemory(userId, newMemory);
      setMemories([created, ...memories]);
      setShowAddForm(false);
      setNewMemory({
        memory_type: 'explicit',
        key: '',
        value: '',
        confidence: 1.0,
        source: 'debug_panel'
      });
    } catch (error) {
      console.error('Failed to add memory:', error);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        <div className="p-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-xl font-semibold flex items-center">
            <Database className="w-5 h-5 mr-2" />
            Debug Panel
          </h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4 border-b border-gray-200 flex items-center justify-between">
          <div className="flex space-x-2">
            <button 
              onClick={() => setShowAddForm(true)}
              className="px-3 py-1 bg-blue-500 text-white rounded flex items-center text-sm"
              disabled={showAddForm}
            >
              <Plus className="w-4 h-4 mr-1" />
              Add Memory
            </button>
            <button 
              onClick={loadMemories}
              className="px-3 py-1 bg-gray-200 text-gray-700 rounded flex items-center text-sm"
            >
              <RefreshCw className="w-4 h-4 mr-1" />
              Refresh
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {loading ? (
            <div className="flex justify-center items-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
            </div>
          ) : (
            <>
              {showAddForm && (
                <div className="mb-6 p-4 border border-blue-200 rounded-lg bg-blue-50">
                  <h3 className="text-lg font-medium mb-3">Add New Memory</h3>
                  <div className="grid grid-cols-2 gap-4 mb-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                      <select
                        className="w-full p-2 border border-gray-300 rounded"
                        value={newMemory.memory_type}
                        onChange={(e) => setNewMemory({...newMemory, memory_type: e.target.value as any})}
                      >
                        <option value="explicit">Explicit</option>
                        <option value="implicit">Implicit</option>
                        <option value="preference">Preference</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Key</label>
                      <input
                        type="text"
                        className="w-full p-2 border border-gray-300 rounded"
                        value={newMemory.key}
                        onChange={(e) => setNewMemory({...newMemory, key: e.target.value})}
                      />
                    </div>
                  </div>
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-gray-700 mb-1">Value</label>
                    <textarea
                      className="w-full p-2 border border-gray-300 rounded"
                      rows={3}
                      value={newMemory.value}
                      onChange={(e) => setNewMemory({...newMemory, value: e.target.value})}
                    ></textarea>
                  </div>
                  <div className="grid grid-cols-2 gap-4 mb-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Confidence</label>
                      <input
                        type="number"
                        min="0"
                        max="1"
                        step="0.1"
                        className="w-full p-2 border border-gray-300 rounded"
                        value={newMemory.confidence}
                        onChange={(e) => setNewMemory({...newMemory, confidence: parseFloat(e.target.value)})}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Source</label>
                      <input
                        type="text"
                        className="w-full p-2 border border-gray-300 rounded"
                        value={newMemory.source}
                        onChange={(e) => setNewMemory({...newMemory, source: e.target.value})}
                      />
                    </div>
                  </div>
                  <div className="flex justify-end space-x-2">
                    <button
                      onClick={() => setShowAddForm(false)}
                      className="px-3 py-1 bg-gray-200 text-gray-700 rounded"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleAddMemory}
                      className="px-3 py-1 bg-green-500 text-white rounded"
                      disabled={!newMemory.key || !newMemory.value}
                    >
                      Add Memory
                    </button>
                  </div>
                </div>
              )}

              <div className="space-y-4">
                {memories.length === 0 ? (
                  <div className="text-center py-8 text-gray-500">
                    No memories found for this user.
                  </div>
                ) : (
                  memories.map(memory => (
                    <div key={memory.id} className="border border-gray-200 rounded-lg p-4">
                      {editingMemory && editingMemory.id === memory.id ? (
                        <div className="space-y-3">
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                              <select
                                className="w-full p-2 border border-gray-300 rounded"
                                value={editingMemory.memory_type}
                                onChange={(e) => setEditingMemory({...editingMemory, memory_type: e.target.value as any})}
                              >
                                <option value="explicit">Explicit</option>
                                <option value="implicit">Implicit</option>
                                <option value="preference">Preference</option>
                              </select>
                            </div>
                            <div>
                              <label className="block text-sm font-medium text-gray-700 mb-1">Key</label>
                              <input
                                type="text"
                                className="w-full p-2 border border-gray-300 rounded"
                                value={editingMemory.key}
                                onChange={(e) => setEditingMemory({...editingMemory, key: e.target.value})}
                              />
                            </div>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">Value</label>
                            <textarea
                              className="w-full p-2 border border-gray-300 rounded"
                              rows={3}
                              value={editingMemory.value}
                              onChange={(e) => setEditingMemory({...editingMemory, value: e.target.value})}
                            ></textarea>
                          </div>
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium text-gray-700 mb-1">Confidence</label>
                              <input
                                type="number"
                                min="0"
                                max="1"
                                step="0.1"
                                className="w-full p-2 border border-gray-300 rounded"
                                value={editingMemory.confidence}
                                onChange={(e) => setEditingMemory({...editingMemory, confidence: parseFloat(e.target.value)})}
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium text-gray-700 mb-1">Source</label>
                              <input
                                type="text"
                                className="w-full p-2 border border-gray-300 rounded"
                                value={editingMemory.source || ''}
                                onChange={(e) => setEditingMemory({...editingMemory, source: e.target.value})}
                              />
                            </div>
                          </div>
                          <div className="flex justify-end space-x-2 mt-3">
                            <button
                              onClick={handleCancelEdit}
                              className="px-3 py-1 bg-gray-200 text-gray-700 rounded"
                            >
                              Cancel
                            </button>
                            <button
                              onClick={handleSaveEdit}
                              className="px-3 py-1 bg-green-500 text-white rounded"
                            >
                              Save
                            </button>
                          </div>
                        </div>
                      ) : (
                        <>
                          <div className="flex justify-between items-start mb-2">
                            <div>
                              <span className={`inline-block px-2 py-1 text-xs rounded mr-2 ${
                                memory.memory_type === 'explicit' ? 'bg-green-100 text-green-800' : 
                                memory.memory_type === 'implicit' ? 'bg-blue-100 text-blue-800' : 
                                'bg-purple-100 text-purple-800'
                              }`}>
                                {memory.memory_type}
                              </span>
                              <span className="font-medium">{memory.key}</span>
                            </div>
                            <div className="flex space-x-1">
                              <button
                                onClick={() => handleEditMemory(memory)}
                                className="p-1 text-gray-500 hover:text-blue-500"
                              >
                                <Edit className="w-4 h-4" />
                              </button>
                              <button
                                onClick={() => handleDeleteMemory(memory.id)}
                                className="p-1 text-gray-500 hover:text-red-500"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          </div>
                          <p className="text-gray-700 mb-2 whitespace-pre-wrap">{memory.value}</p>
                          <div className="flex justify-between text-xs text-gray-500">
                            <div>
                              <span className="mr-2">Confidence: {Math.round(memory.confidence * 100)}%</span>
                              {memory.source && <span>Source: {memory.source}</span>}
                            </div>
                            <div>
                              <span className="mr-2">Created: {new Date(memory.created_at).toLocaleString()}</span>
                              <span>Updated: {new Date(memory.updated_at).toLocaleString()}</span>
                            </div>
                          </div>
                        </>
                      )}
                    </div>
                  ))
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
