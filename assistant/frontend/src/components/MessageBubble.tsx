import React, { useState } from 'react';
import { Clock, ChevronDown, ChevronRight, Zap, Code, AlertCircle, CheckCircle, XCircle, Settings, Database, MessageSquare } from 'lucide-react';
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
  prominent?: boolean;
}

function CollapsibleSection({ title, children, defaultOpen = false, icon, badge, prominent = false }: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className={`border rounded-lg mt-2 overflow-hidden ${
      prominent ? 'border-blue-300 bg-blue-50' : 'border-gray-200'
    }`}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`w-full flex items-center justify-between p-3 hover:bg-gray-100 transition-colors text-left ${
          prominent ? 'bg-blue-100' : 'bg-gray-50'
        }`}
      >
        <div className="flex items-center gap-2">
          {icon}
          <span className={`text-sm font-medium ${
            prominent ? 'text-blue-900' : 'text-gray-700'
          }`}>
            {title}
          </span>
          {badge && (
            <span className={`px-2 py-1 text-xs rounded-full ${
              prominent 
                ? 'bg-blue-200 text-blue-900' 
                : 'bg-blue-100 text-blue-800'
            }`}>
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
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

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
          <pre className="text-xs bg-gray-100 p-2 mt-1 rounded overflow-x-auto whitespace-pre-wrap">
            {formatJsonWithLineBreaks(step.data)}
          </pre>
        </details>
      )}
    </div>
  );
}

function ToolCallComponent({ toolCall }: { toolCall: ToolCall }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

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
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto whitespace-pre-wrap">
          {formatJsonWithLineBreaks(toolCall.arguments)}
        </pre>
      </details>
    </div>
  );
}

function ToolResultComponent({ toolResult }: { toolResult: ToolResult }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

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
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto whitespace-pre-wrap">
          {formatJsonWithLineBreaks(toolResult.result)}
        </pre>
      </details>
    </div>
  );
}

function LLMRequestComponent({ llmRequest }: { llmRequest: LLMRequest }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

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
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto max-h-64 whitespace-pre-wrap">
          {formatJsonWithLineBreaks(llmRequest)}
        </pre>
      </details>
    </div>
  );
}

function LLMResponseComponent({ llmResponse }: { llmResponse: LLMResponse }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

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
        <pre className="text-xs bg-white p-2 mt-1 rounded border overflow-x-auto max-h-64 whitespace-pre-wrap">
          {formatJsonWithLineBreaks(llmResponse.response)}
        </pre>
      </details>
    </div>
  );
}

// NEW: Enhanced Raw LLM Request Display Component
function RawLLMRequestComponent({ llmRequest }: { llmRequest: LLMRequest }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

  // Create the full request object as it would be sent to LMStudio
  const fullRequest = {
    model: llmRequest.model,
    messages: llmRequest.messages,
    temperature: llmRequest.temperature,
    max_tokens: llmRequest.max_tokens,
    stream: llmRequest.stream,
    ...(llmRequest.tools && llmRequest.tools.length > 0 && {
      tools: llmRequest.tools,
      tool_choice: llmRequest.tool_choice || "auto"
    })
  };

  return (
    <div className="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-xs overflow-x-auto mb-4">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Database className="w-4 h-4 text-green-400" />
          <span className="text-green-300 font-semibold">LLM Request (Full JSON)</span>
        </div>
        <div className="text-green-500 text-xs">
          {llmRequest.model} • {llmRequest.messages.length} messages • {llmRequest.tools?.length || 0} tools
        </div>
      </div>
      <div className="text-green-600 text-xs mb-2">
        This is the exact JSON request sent to LMStudio:
      </div>
      <pre className="whitespace-pre-wrap text-green-400 leading-relaxed">
        {formatJsonWithLineBreaks(fullRequest)}
      </pre>
    </div>
  );
}

// NEW: Enhanced Raw LLM Response Display Component
function RawLLMResponseComponent({ llmResponse }: { llmResponse: LLMResponse }) {
  // Format JSON with proper line breaks for better readability
  const formatJsonWithLineBreaks = (obj: any): string => {
    // First, stringify the object with indentation
    const jsonString = JSON.stringify(obj, null, 2);

    // Replace escaped newlines with actual newlines in content fields
    return jsonString.replace(/(\\n)/g, '\n');
  };

  return (
    <div className="bg-gray-900 text-blue-400 p-4 rounded-lg font-mono text-xs overflow-x-auto mb-4">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <MessageSquare className="w-4 h-4 text-blue-400" />
          <span className="text-blue-300 font-semibold">LLM Response (Full JSON)</span>
        </div>
        <div className="text-blue-500 text-xs">
          {llmResponse.processing_time_ms}ms • {llmResponse.token_usage?.total_tokens ?? 'N/A'} tokens
        </div>
      </div>
      <div className="text-blue-600 text-xs mb-2">
        This is the exact JSON response received from LMStudio:
      </div>
      <pre className="whitespace-pre-wrap text-blue-400 leading-relaxed">
        {formatJsonWithLineBreaks(llmResponse.response)}
      </pre>
    </div>
  );
}

export default function MessageBubble({ message, showDebugInfo = false }: MessageBubbleProps) {
  const isUser = message.role === 'user';
  const hasDebugInfo = message.intermediary_steps?.length || message.llm_request || message.tool_calls?.length;

  // Debug: Log what debug data is available
  if (showDebugInfo && !isUser) {
    console.log('=== MESSAGE BUBBLE DEBUG ===');
    console.log('Message ID:', message.id);
    console.log('Message role:', message.role);
    console.log('Message content length:', message.content.length);
    console.log('Debug enabled:', message.debug_enabled);
    console.log('Debug data:', message.debug_data);
    
    // Check each debug field
    console.log('Intermediary steps:', {
      exists: !!message.intermediary_steps,
      length: message.intermediary_steps?.length || 0,
      data: message.intermediary_steps
    });
    
    console.log('LLM request:', {
      exists: !!message.llm_request,
      model: message.llm_request?.model,
      messages_count: message.llm_request?.messages?.length || 0,
      tools_count: message.llm_request?.tools?.length || 0,
      data: message.llm_request
    });
    
    console.log('LLM response:', {
      exists: !!message.llm_response,
      processing_time: message.llm_response?.processing_time_ms,
      tokens: message.llm_response?.token_usage?.total_tokens,
      data: message.llm_response
    });
    
    console.log('Tool calls:', {
      exists: !!message.tool_calls,
      length: message.tool_calls?.length || 0,
      data: message.tool_calls
    });
    
    console.log('Tool results:', {
      exists: !!message.tool_results,
      length: message.tool_results?.length || 0,
      data: message.tool_results
    });
    
    console.log('hasDebugInfo calculated:', hasDebugInfo);
    console.log('showDebugInfo prop:', showDebugInfo);
    console.log('=== END MESSAGE BUBBLE DEBUG ===');
  }

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} animate-fade-in`}>
      <div className={`${
        hasDebugInfo && showDebugInfo
          ? 'max-w-full lg:max-w-4xl xl:max-w-5xl'
          : 'max-w-xs lg:max-w-md xl:max-w-lg'
      } px-4 py-2 rounded-lg ${
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
        {!isUser && showDebugInfo && (
          <div className="mt-3 space-y-2">
            {/* Show diagnostic if debug mode is active but no debug data found */}
            {!hasDebugInfo && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                <div className="flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 text-yellow-600" />
                  <span className="text-sm font-medium text-yellow-800">Debug Mode Active - No Debug Data Found</span>
                </div>
                <p className="text-xs text-yellow-700 mt-1">
                  Debug mode is enabled but no debug data was found for this message. 
                  This may indicate an issue with debug data processing or the message may not have generated debug data.
                </p>
                <details className="mt-2">
                  <summary className="text-xs text-yellow-600 cursor-pointer">View message details</summary>
                  <pre className="text-xs bg-yellow-100 p-2 mt-1 rounded overflow-x-auto whitespace-pre-wrap">
                    {JSON.stringify({
                      id: message.id,
                      role: message.role,
                      timestamp: message.timestamp,
                      debug_enabled: message.debug_enabled,
                      has_debug_data: !!message.debug_data,
                      debug_fields: {
                        intermediary_steps: !!message.intermediary_steps,
                        llm_request: !!message.llm_request,
                        llm_response: !!message.llm_response,
                        tool_calls: !!message.tool_calls,
                        tool_results: !!message.tool_results
                      }
                    }, null, 2)}
                  </pre>
                </details>
              </div>
            )}

            {/* NEW: Raw LLM Request - Most Prominent Display */}
            {message.llm_request && (
              <RawLLMRequestComponent llmRequest={message.llm_request} />
            )}

            {/* NEW: Raw LLM Response - Most Prominent Display */}
            {message.llm_response && (
              <RawLLMResponseComponent llmResponse={message.llm_response} />
            )}

            {/* Intermediary Steps */}
            {message.intermediary_steps && message.intermediary_steps.length > 0 && (
              <CollapsibleSection
                title="Processing Steps"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={`${message.intermediary_steps.length} steps`}
                defaultOpen={true}
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
                defaultOpen={true}
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
                defaultOpen={true}
              >
                <div className="space-y-2">
                  {message.tool_results.map((toolResult, index) => (
                    <ToolResultComponent key={index} toolResult={toolResult} />
                  ))}
                </div>
              </CollapsibleSection>
            )}

            {/* LLM Request - Secondary Display (Detailed) */}
            {message.llm_request && (
              <CollapsibleSection
                title="LLM Request Details"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={message.llm_request.model}
                defaultOpen={false}
              >
                <LLMRequestComponent llmRequest={message.llm_request} />
              </CollapsibleSection>
            )}

            {/* LLM Response - Secondary Display (Detailed) */}
            {message.llm_response && (
              <CollapsibleSection
                title="LLM Response Details"
                icon={<Code className="w-4 h-4 text-purple-600" />}
                badge={`${message.llm_response.processing_time_ms}ms`}
                defaultOpen={false}
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
