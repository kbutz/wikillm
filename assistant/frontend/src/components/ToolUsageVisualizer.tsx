import React, { useState, useEffect } from 'react';
import { Clock, CheckCircle, XCircle, AlertCircle, Database, Brain, Settings, Zap, TrendingUp } from 'lucide-react';

interface ToolUsageStep {
  step_id: string;
  tool_name: string;
  tool_type: string;
  server_id?: string;
  step_type: string;
  description: string;
  timestamp: string;
  duration_ms?: number;
  input_data?: any;
  output_data?: any;
  status: 'pending' | 'success' | 'error' | 'timeout';
  error_message?: string;
  metadata?: any;
}

interface ToolUsageTrace {
  trace_id: string;
  conversation_id: number;
  message_id?: number;
  user_id: number;
  start_time: string;
  end_time?: string;
  total_duration_ms?: number;
  steps: ToolUsageStep[];
  total_steps: number;
  successful_steps: number;
  failed_steps: number;
  tools_used: string[];
  rag_queries: any[];
  memories_retrieved: any[];
  context_size?: number;
}

interface ToolUsageVisualizerProps {
  trace: ToolUsageTrace;
  showDetails?: boolean;
  compact?: boolean;
}

const ToolUsageVisualizer: React.FC<ToolUsageVisualizerProps> = ({ 
  trace, 
  showDetails = true, 
  compact = false 
}) => {
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const [selectedStep, setSelectedStep] = useState<string | null>(null);

  const toggleStepExpansion = (stepId: string) => {
    const newExpanded = new Set(expandedSteps);
    if (newExpanded.has(stepId)) {
      newExpanded.delete(stepId);
    } else {
      newExpanded.add(stepId);
    }
    setExpandedSteps(newExpanded);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <XCircle className="w-4 h-4 text-red-500" />;
      case 'pending':
        return <AlertCircle className="w-4 h-4 text-yellow-500 animate-pulse" />;
      default:
        return <Clock className="w-4 h-4 text-gray-500" />;
    }
  };

  const getToolIcon = (toolType: string) => {
    switch (toolType) {
      case 'rag':
        return <Database className="w-4 h-4 text-blue-500" />;
      case 'memory':
        return <Brain className="w-4 h-4 text-purple-500" />;
      case 'mcp':
        return <Zap className="w-4 h-4 text-yellow-500" />;
      case 'internal':
        return <Settings className="w-4 h-4 text-gray-500" />;
      default:
        return <Settings className="w-4 h-4 text-gray-400" />;
    }
  };

  const formatDuration = (durationMs?: number) => {
    if (!durationMs) return 'N/A';
    if (durationMs < 1000) return `${durationMs}ms`;
    return `${(durationMs / 1000).toFixed(2)}s`;
  };

  const getStepTypeColor = (stepType: string) => {
    switch (stepType) {
      case 'query':
        return 'bg-blue-100 text-blue-800';
      case 'retrieval':
        return 'bg-green-100 text-green-800';
      case 'processing':
        return 'bg-yellow-100 text-yellow-800';
      case 'inference':
        return 'bg-purple-100 text-purple-800';
      case 'storage':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  };

  if (compact) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 text-sm">
        <div className="flex items-center justify-between mb-2">
          <span className="font-medium text-gray-700">Tool Usage Trace</span>
          <span className="text-xs text-gray-500">
            {formatDuration(trace.total_duration_ms)}
          </span>
        </div>
        <div className="flex items-center space-x-4 text-xs">
          <div className="flex items-center space-x-1">
            <CheckCircle className="w-3 h-3 text-green-500" />
            <span>{trace.successful_steps}</span>
          </div>
          <div className="flex items-center space-x-1">
            <XCircle className="w-3 h-3 text-red-500" />
            <span>{trace.failed_steps}</span>
          </div>
          <div className="flex items-center space-x-1">
            <TrendingUp className="w-3 h-3 text-blue-500" />
            <span>{trace.tools_used.length} tools</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-sm">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-50 to-indigo-50 px-4 py-3 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="flex items-center space-x-2">
              <Settings className="w-5 h-5 text-blue-600" />
              <h3 className="font-semibold text-gray-900">Tool Usage Trace</h3>
            </div>
            <span className="text-xs text-gray-500">
              ID: {trace.trace_id.slice(0, 8)}...
            </span>
          </div>
          <div className="flex items-center space-x-4 text-sm">
            <div className="flex items-center space-x-1">
              <Clock className="w-4 h-4 text-gray-500" />
              <span className="text-gray-600">{formatDuration(trace.total_duration_ms)}</span>
            </div>
            <div className="flex items-center space-x-1">
              <CheckCircle className="w-4 h-4 text-green-500" />
              <span className="text-green-600">{trace.successful_steps}</span>
            </div>
            <div className="flex items-center space-x-1">
              <XCircle className="w-4 h-4 text-red-500" />
              <span className="text-red-600">{trace.failed_steps}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="px-4 py-3 bg-gray-50 border-b border-gray-200">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div className="flex items-center space-x-2">
            <TrendingUp className="w-4 h-4 text-blue-500" />
            <span className="text-gray-600">
              {trace.tools_used.length} tools used
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <Database className="w-4 h-4 text-purple-500" />
            <span className="text-gray-600">
              {trace.rag_queries.length} RAG queries
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <Brain className="w-4 h-4 text-green-500" />
            <span className="text-gray-600">
              {trace.memories_retrieved.length} memories
            </span>
          </div>
          <div className="flex items-center space-x-2">
            <Settings className="w-4 h-4 text-gray-500" />
            <span className="text-gray-600">
              {trace.total_steps} total steps
            </span>
          </div>
        </div>
      </div>

      {/* Steps Timeline */}
      <div className="px-4 py-3">
        <div className="space-y-2">
          {trace.steps.map((step, index) => (
            <div
              key={step.step_id}
              className={`border rounded-lg transition-all duration-200 ${
                selectedStep === step.step_id 
                  ? 'border-blue-300 bg-blue-50' 
                  : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              {/* Step Header */}
              <div
                className="flex items-center justify-between p-3 cursor-pointer"
                onClick={() => {
                  toggleStepExpansion(step.step_id);
                  setSelectedStep(step.step_id);
                }}
              >
                <div className="flex items-center space-x-3">
                  <div className="flex items-center space-x-2">
                    <span className="text-xs font-mono text-gray-500 w-6">
                      {index + 1}
                    </span>
                    {getStatusIcon(step.status)}
                    {getToolIcon(step.tool_type)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center space-x-2">
                      <span className="font-medium text-gray-900 truncate">
                        {step.tool_name}
                      </span>
                      <span className={`px-2 py-1 text-xs rounded-full ${getStepTypeColor(step.step_type)}`}>
                        {step.step_type}
                      </span>
                      {step.server_id && (
                        <span className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                          {step.server_id}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-gray-600 truncate mt-1">
                      {step.description}
                    </p>
                  </div>
                </div>
                <div className="flex items-center space-x-2 text-sm text-gray-500">
                  <span>{formatDuration(step.duration_ms)}</span>
                  <span className="text-xs">
                    {new Date(step.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              </div>

              {/* Expanded Step Details */}
              {expandedSteps.has(step.step_id) && (
                <div className="px-3 pb-3 border-t border-gray-100">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-3">
                    {/* Input Data */}
                    {step.input_data && (
                      <div>
                        <h4 className="text-sm font-medium text-gray-700 mb-2">Input</h4>
                        <div className="bg-gray-50 rounded p-2 text-xs font-mono max-h-32 overflow-y-auto">
                          <pre className="whitespace-pre-wrap text-gray-700">
                            {JSON.stringify(step.input_data, null, 2)}
                          </pre>
                        </div>
                      </div>
                    )}

                    {/* Output Data */}
                    {step.output_data && (
                      <div>
                        <h4 className="text-sm font-medium text-gray-700 mb-2">Output</h4>
                        <div className="bg-gray-50 rounded p-2 text-xs font-mono max-h-32 overflow-y-auto">
                          <pre className="whitespace-pre-wrap text-gray-700">
                            {JSON.stringify(step.output_data, null, 2)}
                          </pre>
                        </div>
                      </div>
                    )}

                    {/* Error Message */}
                    {step.error_message && (
                      <div className="md:col-span-2">
                        <h4 className="text-sm font-medium text-red-700 mb-2">Error</h4>
                        <div className="bg-red-50 border border-red-200 rounded p-2 text-sm text-red-700">
                          {step.error_message}
                        </div>
                      </div>
                    )}

                    {/* Metadata */}
                    {step.metadata && Object.keys(step.metadata).length > 0 && (
                      <div className="md:col-span-2">
                        <h4 className="text-sm font-medium text-gray-700 mb-2">Metadata</h4>
                        <div className="bg-gray-50 rounded p-2 text-xs">
                          {Object.entries(step.metadata).map(([key, value]) => (
                            <div key={key} className="flex justify-between py-1 border-b border-gray-200 last:border-0">
                              <span className="text-gray-600">{key}:</span>
                              <span className="text-gray-900 font-mono">
                                {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* RAG and Memory Summary */}
      {showDetails && (trace.rag_queries.length > 0 || trace.memories_retrieved.length > 0) && (
        <div className="px-4 py-3 bg-gray-50 border-t border-gray-200">
          <h4 className="text-sm font-medium text-gray-700 mb-3">RAG & Memory Analysis</h4>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* RAG Queries */}
            {trace.rag_queries.length > 0 && (
              <div>
                <h5 className="text-xs font-medium text-gray-600 mb-2">RAG Queries</h5>
                <div className="space-y-2">
                  {trace.rag_queries.map((query, index) => (
                    <div key={index} className="bg-white rounded border border-gray-200 p-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-gray-500">Query {index + 1}</span>
                        <span className="text-blue-600">{query.results_count} results</span>
                      </div>
                      <p className="text-sm text-gray-700 mt-1 truncate">
                        {query.query || 'No query text'}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Memory Retrieval */}
            {trace.memories_retrieved.length > 0 && (
              <div>
                <h5 className="text-xs font-medium text-gray-600 mb-2">Memory Retrieval</h5>
                <div className="space-y-2">
                  {trace.memories_retrieved.map((memory, index) => (
                    <div key={index} className="bg-white rounded border border-gray-200 p-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="text-gray-500">{memory.memory_type || 'Unknown'}</span>
                        <span className="text-purple-600">{memory.count} memories</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default ToolUsageVisualizer;
