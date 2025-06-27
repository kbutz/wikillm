import React from 'react';

export default function LoadingMessage() {
  return (
    <div className="flex justify-start animate-fade-in">
      <div className="bg-white border border-gray-200 rounded-lg px-4 py-2 shadow-sm">
        <div className="flex items-center space-x-2">
          <div className="flex space-x-1">
            <div className="w-2 h-2 bg-gray-400 rounded-full loading-dot"></div>
            <div className="w-2 h-2 bg-gray-400 rounded-full loading-dot"></div>
            <div className="w-2 h-2 bg-gray-400 rounded-full loading-dot"></div>
          </div>
          <span className="text-xs text-gray-500">AI is thinking...</span>
        </div>
      </div>
    </div>
  );
}
