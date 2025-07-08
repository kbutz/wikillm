import React, { useState } from 'react';
import { Clock, ChevronDown, ChevronRight, Zap, Code, AlertCircle, CheckCircle, XCircle, Settings } from 'lucide-react';
import { Message, IntermediaryStep, ToolCall, ToolResult, LLMRequest, LLMResponse } from '../types';

interface MessageBubbleProps {
  message: Message;
  showDebugInfo?: boolean;
}

interface CollapsibleSectionProps {
  title: string;
  children: React.ReactNode;
  defaultOpen?: boolean;
  icon?: React.ReactNode;
  badge?: string;
}

function CollapsibleSection({ title, children, defaultOpen = false, icon, badge }: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-gray-200 rounded-lg mt-2 overflow-hidden">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between p-3 bg-gray-50 hover:bg-gray-100 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          {icon}
          <span className="text-sm font-medium text-gray-700">{title}</span>
          {badge && (
            <span className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded-full">
              {badge}
            </span>
          )}
        </div>
        {isOpen ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
      </button>
      {isOpen && (
        <div className="p-3 bg-white border-t border-gray-200">
          {children}
        </div>
      )}
    </div>
  );
}

function IntermediaryStepComponent({ step }: { step: IntermediaryStep }) {
  const getStepIcon = (stepType: string) => {
    switch (stepType) {
      case 'tool_call':
        return <Settings className="w-4 h-4 text-blue-600" />;
      case 'tool_result':
        return <Zap className="w-4 h-4 text-green-600" />;
      case 'llm_request':
      case 'llm_response':
        return <Code className="w-4 h-4 text-purple-600" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-600" />;
      default:
        return <CheckCircle className="w-4 h-4 text-gray-600" />;
    }
  };

  const getStepColor = (success: boolean) => {
    return success ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50';
  };

  return (
    <div className={`border rounded-lg p-3 mb-2 ${getStepColor(step.success)}`}>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {getStepIcon(step.step_type)}
          <span className="font-medium text-sm">{step.title}</span>
          {step.success ? (
            <CheckCircle className="w-4 h-4 text-green-600" />
          ) : (
            <XCircle className="w-4 h-4 text-red-600" />
          )}
        </div>
        <div className="text-xs text-gray-500">
          {step.duration_ms ? `${step.duration_ms}ms` : ''}
        </div>
      </div>

      {step.description && (
        <p className="text-sm text-gray-600 mb-2">{step.description}</p>
      )}

      {step.error_message && (
        <div className="text-sm text-red-600 bg-red-100 p-2 rounded border border-red-200">
          <strong>Error:</strong> {step.error_message}
        </div>
      )}

      {Object.keys(step.data).length > 0 && (
        <details className="mt-2">
          <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-700">
            View step data
          </summary>
          <pre className="text-xs bg-gray-100 p-2 mt-1 rounded overflow-x-auto">
            {JSON.stringify(step.data, null, 2)}
          </pre>
        </details>
      )}
    </div>
  );
}

function ToolCallComponent({ toolCall }: { toolCall: ToolCall }) {
  return (
    <div className="border border-blue-200 rounded-lg p-3 mb-2 bg-blue-50">
      <div className="flex items-center gap-2 mb-2">
        <Settings className="w-4 h-4 text-blue-600" />
        <span className="font-medium text-sm">{toolCall.tool_name}</span>
        {toolCall.server_id && (
          <span className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">
            {toolCall.server_id}
          </span>
        )}
      </div>

      <details>
        <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-800">
          View arguments
        </summary>
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto">
          {JSON.stringify(toolCall.arguments, null, 2)}
        </pre>
      </details>
    </div>
  );
}

function ToolResultComponent({ toolResult }: { toolResult: ToolResult }) {
  return (
    <div className={`border rounded-lg p-3 mb-2 ${
      toolResult.success ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'
    }`}>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Zap className="w-4 h-4 text-green-600" />
          <span className="font-medium text-sm">{toolResult.tool_name}</span>
          {toolResult.success ? (
            <CheckCircle className="w-4 h-4 text-green-600" />
          ) : (
            <XCircle className="w-4 h-4 text-red-600" />
          )}
        </div>
        <div className="text-xs text-gray-500">
          {toolResult.execution_time_ms ? `${toolResult.execution_time_ms}ms` : ''}
        </div>
      </div>

      {toolResult.error_message && (
        <div className="text-sm text-red-600 bg-red-100 p-2 rounded border border-red-200 mb-2">
          <strong>Error:</strong> {toolResult.error_message}
        </div>
      )}

      <details>
        <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-800">
          View result
        </summary>
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto">
          {JSON.stringify(toolResult.result, null, 2)}
        </pre>
      </details>
    </div>
  );
}

function LLMRequestComponent({ llmRequest }: { llmRequest: LLMRequest }) {
  return (
    <div className="border border-purple-200 rounded-lg p-3 bg-purple-50">
      <div className="grid grid-cols-2 gap-4 mb-3">
        <div>
          <span className="text-xs text-gray-600 block">Model:</span>
          <span className="text-sm font-medium">{llmRequest.model}</span>
        </div>
        <div>
          <span className="text-xs text-gray-600 block">Temperature:</span>
          <span className="text-sm font-medium">{llmRequest.temperature ?? 'default'}</span>
        </div>
        <div>
          <span className="text-xs text-gray-600 block">Max Tokens:</span>
          <span className="text-sm font-medium">{llmRequest.max_tokens ?? 'default'}</span>
        </div>
        <div>
          <span className="text-xs text-gray-600 block">Tools:</span>
          <span className="text-sm font-medium">{llmRequest.tools?.length ?? 0}</span>
        </div>
      </div>

      <details>
        <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-800">
          View full request ({llmRequest.messages.length} messages)
        </summary>
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto max-h-64">
          {JSON.stringify(llmRequest, null, 2)}
        </pre>
      </details>
    </div>
  );
}

function LLMResponseComponent({ llmResponse }: { llmResponse: LLMResponse }) {
  return (
    <div className="border border-purple-200 rounded-lg p-3 bg-purple-50">
      <div className="grid grid-cols-2 gap-4 mb-3">
        <div>
          <span className="text-xs text-gray-600 block">Processing Time:</span>
          <span className="text-sm font-medium">{llmResponse.processing_time_ms}ms</span>
        </div>
        <div>
          <span className="text-xs text-gray-600 block">Token Usage:</span>
          <span className="text-sm font-medium">
            {llmResponse.token_usage?.total_tokens ?? 'N/A'}
          </span>
        </div>
      </div>

      <details>
        <summary className="text-xs text-gray-600 cursor-pointer hover:text-gray-800">
          View full response
        </summary>
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto max-h-64">
          {JSON.stringify(llmResponse.response, null, 2)}
        </pre>
      </details>
    </div>
  );
}

export default function MessageBubble({ message, showDebugInfo = false }: MessageBubbleProps) {
  const isUser = message.role === 'user';
  const hasDebugInfo = message.intermediary_steps?.length || message.llm_request || message.tool_calls?.length;

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} animate-fade-in`}>
      <div className={`max-w-xs lg:max-w-md xl:max-w-lg px-4 py-2 rounded-lg ${
        isUser
          ? 'bg-blue-600 text-white'
          : 'bg-white border border-gray-200 text-gray-900 shadow-sm'
      }`}>
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
          {hasDebugInfo && showDebugInfo && (
            <span className="ml-2 px-2 py-1 bg-blue-100 text-blue-800 rounded text-xs">
              Debug
            </span>
          )}
        </div>

        {/* Debug Information */}
        {!isUser && hasDebugInfo && showDebugInfo && (
          <div className="mt-3 space-y-2">
            {/* Intermediary Steps */}
            {message.intermediary_steps && message.intermediary_steps.length > 0 && (
              <CollapsibleSection
                title="Processing Steps"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={`${message.intermediary_steps.length} steps`}
              >
                <div className="space-y-2">
                  {message.intermediary_steps.map((step, index) => (
                    <IntermediaryStepComponent key={step.step_id || index} step={step} />
                  ))}
                </div>
              </CollapsibleSection>
            )}

            {/* Tool Calls */}
            {message.tool_calls && message.tool_calls.length > 0 && (
              <CollapsibleSection
                title="Tool Calls"
                icon={<Settings className="w-4 h-4 text-blue-600" />}
                badge={`${message.tool_calls.length} calls`}
              >
                <div className="space-y-2">
                  {message.tool_calls.map((toolCall, index) => (
                    <ToolCallComponent key={index} toolCall={toolCall} />
                  ))}
                </div>
              </CollapsibleSection>
            )}

            {/* Tool Results */}
            {message.tool_results && message.tool_results.length > 0 && (
              <CollapsibleSection
                title="Tool Results"
                icon={<Zap className="w-4 h-4 text-green-600" />}
                badge={`${message.tool_results.length} results`}
              >
                <div className="space-y-2">
                  {message.tool_results.map((toolResult, index) => (
                    <ToolResultComponent key={index} toolResult={toolResult} />
                  ))}
                </div>
              </CollapsibleSection>
            )}

            {/* LLM Request */}
            {message.llm_request && (
              <CollapsibleSection
                title="LLM Request"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={message.llm_request.model}
              >
                <LLMRequestComponent llmRequest={message.llm_request} />
              </CollapsibleSection>
            )}

            {/* LLM Response */}
            {message.llm_response && (
              <CollapsibleSection
                title="LLM Response"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={`${message.llm_response.processing_time_ms}ms`}
              >
                <LLMResponseComponent llmResponse={message.llm_response} />
              </CollapsibleSection>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
