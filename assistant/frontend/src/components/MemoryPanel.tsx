import React from 'react';
import { Brain } from 'lucide-react';
import { UserMemory } from '../types';

interface MemoryPanelProps {
  memories: UserMemory[];
  onClose: () => void;
}

export default function MemoryPanel({ memories, onClose }: MemoryPanelProps) {
  const groupedMemories = memories.reduce((acc, memory) => {
    if (!acc[memory.memory_type]) acc[memory.memory_type] = [];
    acc[memory.memory_type].push(memory);
    return acc;
  }, {} as Record<string, UserMemory[]>);

  return (
    <div className="w-80 bg-white border-l border-gray-200 flex flex-col animate-slide-up">
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900">User Memory</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors text-xl font-bold w-6 h-6 flex items-center justify-center"
          >
            ×
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {Object.entries(groupedMemories).map(([type, typeMemories]) => (
          <div key={type} className="space-y-2">
            <h4 className="text-sm font-medium text-gray-700 capitalize flex items-center">
              <span className={`w-2 h-2 rounded-full mr-2 ${
                type === 'explicit' ? 'bg-green-500' : 
                type === 'implicit' ? 'bg-blue-500' : 'bg-purple-500'
              }`}></span>
              {type} Memory ({typeMemories.length})
            </h4>
            <div className="space-y-2">
              {typeMemories.map(memory => (
                <div key={memory.id} className="p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-xs font-medium text-gray-600">
                      {memory.key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                    </span>
                    <div className="flex items-center space-x-2">
                      <span className={`text-xs px-1.5 py-0.5 rounded ${
                        memory.confidence >= 0.8 ? 'bg-green-100 text-green-700' :
                        memory.confidence >= 0.6 ? 'bg-yellow-100 text-yellow-700' :
                        'bg-red-100 text-red-700'
                      }`}>
                        {Math.round(memory.confidence * 100)}%
                      </span>
                    </div>
                  </div>
                  <p className="text-sm text-gray-800">{memory.value}</p>
                  <div className="flex items-center justify-between mt-2 text-xs text-gray-500">
                    <span>{new Date(memory.created_at).toLocaleDateString()}</span>
                    {memory.source && (
                      <span className="text-xs bg-gray-200 px-1.5 py-0.5 rounded">
                        {memory.source}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}

        {memories.length === 0 && (
          <div className="text-center py-8">
            <Brain className="w-12 h-12 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-500 text-sm">
              No memories stored yet. Start chatting to build your personalized experience!
            </p>
            <p className="text-gray-400 text-xs mt-2">
              The AI will automatically learn your preferences and remember important information.
            </p>
          </div>
        )}
      </div>

      {memories.length > 0 && (
        <div className="p-4 border-t border-gray-200 bg-gray-50">
          <div className="text-xs text-gray-600 space-y-1">
            <div className="flex items-center">
              <span className="w-2 h-2 rounded-full bg-green-500 mr-2"></span>
              <span>Explicit: Information you've directly shared</span>
            </div>
            <div className="flex items-center">
              <span className="w-2 h-2 rounded-full bg-blue-500 mr-2"></span>
              <span>Implicit: Patterns learned from conversations</span>
            </div>
            <div className="flex items-center">
              <span className="w-2 h-2 rounded-full bg-purple-500 mr-2"></span>
              <span>Preference: Your communication preferences</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
