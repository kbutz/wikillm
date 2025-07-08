import React, { useState, useEffect } from 'react';
import { Database, Search, Edit3, Save, X, Download, AlertTriangle, CheckCircle } from 'lucide-react';
import { AdminService, AdminMemory } from '../services/admin';

const adminService = new AdminService();

interface MemoryInspectorProps {
  userId: number;
  onClose: () => void;
}

export default function MemoryInspector({ userId, onClose }: MemoryInspectorProps) {
  const [memoryData, setMemoryData] = useState<AdminMemory | null>(null);
  const [editingSection, setEditingSection] = useState<string | null>(null);
  const [editedData, setEditedData] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [loading, setLoading] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'saving' | 'success' | 'error' | null>(null);

  const fetchMemoryData = async () => {
    setLoading(true);
    try {
      const data = await adminService.getAdminUserMemory(userId);
      setMemoryData(data);
    } catch (error) {
      console.error('Failed to fetch memory data:', error);
    }
    setLoading(false);
  };

  const saveMemoryData = async (section: string, data: any) => {
    setSaveStatus('saving');
    try {
      await adminService.updateAdminUserMemory(userId, { [section]: data });
      setSaveStatus('success');
      await fetchMemoryData();
      setTimeout(() => setSaveStatus(null), 2000);
    } catch (error) {
      console.error('Failed to save memory data:', error);
      setSaveStatus('error');
    }
  };

  const exportMemoryData = () => {
    if (!memoryData) return;

    const dataStr = JSON.stringify(memoryData, null, 2);
    const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr);
    const exportFileDefaultName = `user_${userId}_memory.json`;

    const linkElement = document.createElement('a');
    linkElement.setAttribute('href', dataUri);
    linkElement.setAttribute('download', exportFileDefaultName);
    linkElement.click();
  };

  const clearMemorySection = async (section: string) => {
    if (window.confirm(`Are you sure you want to clear the ${section} section?`)) {
      const emptyData = section === 'conversation_history' ? [] : {};
      await saveMemoryData(section, emptyData);
    }
  };

  useEffect(() => {
    fetchMemoryData();
  }, [userId]);

  const startEditing = (section: string, data: any) => {
    setEditingSection(section);
    setEditedData(JSON.stringify(data, null, 2));
  };

  const saveEdit = async () => {
    if (!editingSection) return;

    try {
      const parsedData = JSON.parse(editedData);
      await saveMemoryData(editingSection, parsedData);
      setEditingSection(null);
      setEditedData('');
    } catch (error) {
      alert('Invalid JSON format. Please check your syntax.');
    }
  };

  const cancelEdit = () => {
    setEditingSection(null);
    setEditedData('');
  };

  const filterData = (data: any, searchTerm: string): any => {
    if (!searchTerm) return data;

    const search = searchTerm.toLowerCase();

    if (Array.isArray(data)) {
      return data.filter(item => 
        JSON.stringify(item).toLowerCase().includes(search)
      );
    }

    if (typeof data === 'object' && data !== null) {
      const filtered: any = {};
      Object.keys(data).forEach(key => {
        const value = data[key];
        if (typeof value === 'string' && value.toLowerCase().includes(search)) {
          filtered[key] = value;
        } else if (typeof value === 'object' && value !== null) {
          const nestedFiltered = filterData(value, searchTerm);
          if (Object.keys(nestedFiltered).length > 0) {
            filtered[key] = nestedFiltered;
          }
        } else if (key.toLowerCase().includes(search)) {
          filtered[key] = value;
        }
      });
      return filtered;
    }

    return data;
  };

  const MemorySection = ({ title, data, sectionKey }: { title: string; data: any; sectionKey: string }) => {
    const isEditing = editingSection === sectionKey;
    const displayData = searchTerm ? filterData(data, searchTerm) : data;

    return (
      <div className="border rounded-lg p-4 mb-4 bg-white">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Database size={18} />
            {title}
          </h3>
          <div className="flex gap-2">
            <button
              onClick={() => startEditing(sectionKey, data)}
              className="p-2 hover:bg-gray-100 rounded-lg"
              title="Edit section"
            >
              <Edit3 size={16} />
            </button>
            <button
              onClick={() => clearMemorySection(sectionKey)}
              className="p-2 hover:bg-red-100 rounded-lg text-red-600"
              title="Clear section"
            >
              <X size={16} />
            </button>
          </div>
        </div>

        {isEditing ? (
          <div className="space-y-4">
            <textarea
              value={editedData}
              onChange={(e) => setEditedData(e.target.value)}
              className="w-full h-64 p-3 border rounded-lg font-mono text-sm"
              placeholder="Enter JSON data..."
            />
            <div className="flex gap-2">
              <button
                onClick={saveEdit}
                className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 flex items-center gap-2"
              >
                <Save size={16} />
                Save
              </button>
              <button
                onClick={cancelEdit}
                className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 flex items-center gap-2"
              >
                <X size={16} />
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <div className="bg-gray-50 rounded-lg p-4">
            <pre className="text-sm overflow-x-auto whitespace-pre-wrap">
              {JSON.stringify(displayData, null, 2)}
            </pre>
          </div>
        )}
      </div>
    );
  };

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-8">
          <div className="text-center">Loading memory data...</div>
        </div>
      </div>
    );
  }

  if (!memoryData) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-8">
          <div className="text-center">No memory data available</div>
          <button
            onClick={onClose}
            className="mt-4 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg w-full max-w-4xl max-h-[90vh] overflow-hidden">
        <div className="p-6 border-b">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-2xl font-bold">Memory Inspector - User {userId}</h2>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-100 rounded-lg"
            >
              <X size={20} />
            </button>
          </div>

          <div className="flex gap-4 items-center">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
              <input
                type="text"
                placeholder="Search memory data..."
                className="w-full pl-10 pr-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <button
              onClick={exportMemoryData}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
            >
              <Download size={16} />
              Export
            </button>
          </div>

          {saveStatus && (
            <div className={`mt-4 p-3 rounded-lg flex items-center gap-2 ${
              saveStatus === 'success' ? 'bg-green-100 text-green-800' :
              saveStatus === 'error' ? 'bg-red-100 text-red-800' :
              'bg-blue-100 text-blue-800'
            }`}>
              {saveStatus === 'success' ? <CheckCircle size={16} /> :
               saveStatus === 'error' ? <AlertTriangle size={16} /> :
               <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />}
              {saveStatus === 'success' ? 'Changes saved successfully' :
               saveStatus === 'error' ? 'Failed to save changes' :
               'Saving changes...'}
            </div>
          )}
        </div>

        <div className="p-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          <div className="mb-6">
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="p-4 bg-blue-50 rounded-lg">
                <div className="text-2xl font-bold text-blue-600">{Math.round(memoryData.size / 1024)}KB</div>
                <div className="text-sm text-gray-600">Total Memory Size</div>
              </div>
              <div className="p-4 bg-green-50 rounded-lg">
                <div className="text-2xl font-bold text-green-600">
                  {memoryData.conversation_history?.length || 0}
                </div>
                <div className="text-sm text-gray-600">Conversations</div>
              </div>
              <div className="p-4 bg-orange-50 rounded-lg">
                <div className="text-2xl font-bold text-orange-600">
                  {Object.keys(memoryData.personal_info || {}).length}
                </div>
                <div className="text-sm text-gray-600">Personal Info Items</div>
              </div>
            </div>
          </div>

          <MemorySection
            title="Personal Information"
            data={memoryData.personal_info}
            sectionKey="personal_info"
          />

          <MemorySection
            title="Conversation History"
            data={memoryData.conversation_history}
            sectionKey="conversation_history"
          />

          <MemorySection
            title="Context Memory"
            data={memoryData.context_memory}
            sectionKey="context_memory"
          />

          <MemorySection
            title="Preferences"
            data={memoryData.preferences}
            sectionKey="preferences"
          />

          <div className="mt-6 p-4 bg-gray-50 rounded-lg">
            <div className="text-sm text-gray-600">
              <strong>Last Updated:</strong> {new Date(memoryData.last_updated).toLocaleString()}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
