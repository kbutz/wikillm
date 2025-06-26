import React from 'react';
import { Clock } from 'lucide-react';
import { Message } from '../types';

interface MessageBubbleProps {
  message: Message;
}

export default function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user';
  
  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} animate-fade-in`}>
      <div
        className={`message-bubble max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
          isUser
            ? 'bg-blue-600 text-white'
            : 'bg-white border border-gray-200 text-gray-900 shadow-sm'
        }`}
      >
        <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        <div className={`flex items-center mt-1 text-xs ${
          isUser ? 'text-blue-200' : 'text-gray-500'
        }`}>
          <Clock className="w-3 h-3 mr-1" />
          {new Date(message.timestamp).toLocaleTimeString()}
          {message.processing_time && (
            <span className="ml-2">
              ({message.processing_time.toFixed(1)}s)
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
