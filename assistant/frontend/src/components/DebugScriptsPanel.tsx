import React, { useState, useEffect } from 'react';
import { 
  X, RefreshCw, Play, AlertCircle, CheckCircle, 
  Terminal, FileCode, Clock, ChevronDown, ChevronRight
} from 'lucide-react';
import { ApiService } from '../services/api';
import { DebugScript, ScriptResult } from '../types';

interface DebugScriptsPanelProps {
  userId: number;
  onClose: () => void;
}

const api = new ApiService();

export default function DebugScriptsPanel({ userId, onClose }: DebugScriptsPanelProps) {
  const [scripts, setScripts] = useState<DebugScript[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scriptResults, setScriptResults] = useState<Record<string, ScriptResult>>({});
  const [runningScript, setRunningScript] = useState<string | null>(null);
  const [expandedScripts, setExpandedScripts] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadScripts();
  }, []);

  const loadScripts = async () => {
    setLoading(true);
    setError(null);

    try {
      const scripts = await api.listDebugScripts();
      setScripts(scripts);
    } catch (e: any) {
      console.error('Failed to load debug scripts:', e);
      setError('Failed to load debug scripts. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const runScript = async (scriptName: string) => {
    setRunningScript(scriptName);

    try {
      const result = await api.runDebugScript(scriptName);
      setScriptResults(prev => ({
        ...prev,
        [scriptName]: result
      }));

      // Auto-expand the script result
      setExpandedScripts(prev => {
        const newSet = new Set(prev);
        newSet.add(scriptName);
        return newSet;
      });
    } catch (e: any) {
      console.error(`Failed to run script ${scriptName}:`, e);
      setScriptResults(prev => ({
        ...prev,
        [scriptName]: {
          script_name: scriptName,
          success: false,
          output: '',
          error: e.message || 'Unknown error occurred',
          execution_time: 0
        }
      }));
    } finally {
      setRunningScript(null);
    }
  };

  const toggleExpand = (scriptName: string) => {
    setExpandedScripts(prev => {
      const newSet = new Set(prev);
      if (newSet.has(scriptName)) {
        newSet.delete(scriptName);
      } else {
        newSet.add(scriptName);
      }
      return newSet;
    });
  };

  const getScriptsByType = () => {
    const scriptsByType: Record<string, DebugScript[]> = {};

    scripts.forEach(script => {
      if (!scriptsByType[script.type]) {
        scriptsByType[script.type] = [];
      }
      scriptsByType[script.type].push(script);
    });

    return scriptsByType;
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        <div className="p-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-xl font-semibold flex items-center">
            <Terminal className="w-5 h-5 mr-2" />
            Debug Scripts Panel
          </h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4 flex items-center justify-between border-b border-gray-200">
          <p className="text-sm text-gray-600">
            Run debug and verification scripts directly from the web interface.
          </p>
          <button 
            onClick={loadScripts}
            className="flex items-center text-sm text-blue-600 hover:text-blue-800"
          >
            <RefreshCw className="w-4 h-4 mr-1" />
            Refresh
          </button>
        </div>

        <div className="flex-1 overflow-auto p-4">
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            </div>
          ) : error ? (
            <div className="bg-red-50 p-4 rounded-md text-red-600 flex items-start">
              <AlertCircle className="w-5 h-5 mr-2 flex-shrink-0 mt-0.5" />
              <p>{error}</p>
            </div>
          ) : scripts.length === 0 ? (
            <div className="bg-yellow-50 p-4 rounded-md text-yellow-600">
              No debug scripts found.
            </div>
          ) : (
            <div className="space-y-6">
              {Object.entries(getScriptsByType()).map(([type, typeScripts]) => (
                <div key={type} className="border border-gray-200 rounded-md overflow-hidden">
                  <div className="bg-gray-50 p-3 font-medium text-gray-700 capitalize">
                    {type} Scripts ({typeScripts.length})
                  </div>
                  <div className="divide-y divide-gray-200">
                    {typeScripts.map(script => (
                      <div key={script.name} className="p-0">
                        <div className="p-3 flex items-center justify-between hover:bg-gray-50">
                          <div className="flex-1">
                            <div className="flex items-center">
                              <FileCode className="w-4 h-4 mr-2 text-gray-500" />
                              <h3 className="font-medium">{script.name}</h3>
                            </div>
                            <p className="text-sm text-gray-600 mt-1">{script.description}</p>
                          </div>
                          <div className="flex items-center space-x-2">
                            {runningScript === script.name ? (
                              <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
                            ) : (
                              <button
                                onClick={() => runScript(script.name)}
                                className="p-1.5 rounded-md text-green-600 hover:bg-green-50"
                                title="Run script"
                              >
                                <Play className="w-4 h-4" />
                              </button>
                            )}
                            <button
                              onClick={() => toggleExpand(script.name)}
                              className="p-1.5 rounded-md text-gray-500 hover:bg-gray-100"
                              title={expandedScripts.has(script.name) ? "Collapse" : "Expand"}
                            >
                              {expandedScripts.has(script.name) ? (
                                <ChevronDown className="w-4 h-4" />
                              ) : (
                                <ChevronRight className="w-4 h-4" />
                              )}
                            </button>
                          </div>
                        </div>

                        {expandedScripts.has(script.name) && scriptResults[script.name] && (
                          <div className="p-3 bg-gray-50 border-t border-gray-200">
                            <div className="flex items-center justify-between mb-2">
                              <div className="flex items-center">
                                {scriptResults[script.name].success ? (
                                  <CheckCircle className="w-4 h-4 text-green-600 mr-1.5" />
                                ) : (
                                  <AlertCircle className="w-4 h-4 text-red-600 mr-1.5" />
                                )}
                                <span className={scriptResults[script.name].success ? "text-green-600" : "text-red-600"}>
                                  {scriptResults[script.name].success ? "Success" : "Failed"}
                                </span>
                              </div>
                              <div className="flex items-center text-xs text-gray-500">
                                <Clock className="w-3.5 h-3.5 mr-1" />
                                {scriptResults[script.name].execution_time.toFixed(2)}s
                              </div>
                            </div>

                            <div className="mt-2 bg-gray-900 text-gray-100 p-3 rounded-md overflow-auto max-h-96 text-sm font-mono">
                              <pre className="whitespace-pre-wrap">{scriptResults[script.name].output}</pre>
                            </div>

                            {scriptResults[script.name].error && (
                              <div className="mt-2 bg-red-900 text-red-100 p-3 rounded-md overflow-auto max-h-48 text-sm font-mono">
                                <pre className="whitespace-pre-wrap">{scriptResults[script.name].error}</pre>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
