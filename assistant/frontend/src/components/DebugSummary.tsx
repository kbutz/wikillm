import React, { useState, useEffect } from 'react';
import { DebugData } from '../types';
import { fetchDebugData } from '../services/api';
import { X } from 'lucide-react';

interface DebugSummaryProps {
  conversationId: number;
  userId: number;
  onClose: () => void;
}

const DebugSummary: React.FC<DebugSummaryProps> = ({ conversationId, userId, onClose }) => {
  const [debugData, setDebugData] = useState<DebugData | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadDebugData = async () => {
      try {
        setLoading(true);
        const data = await fetchDebugData(conversationId, userId);
        setDebugData(data);
        setError(null);
      } catch (err) {
        setError('Failed to load debug data');
        console.error('Error loading debug data:', err);
      } finally {
        setLoading(false);
      }
    };

    loadDebugData();
  }, [conversationId, userId]);

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-3/4 max-w-4xl max-h-[80vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-semibold">Debug Summary</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex-1 overflow-auto p-6">
          {loading && (
            <div className="flex items-center justify-center h-40">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
          )}

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
              {error}
            </div>
          )}

          {!loading && !error && !debugData && (
            <div className="text-center text-gray-500 py-10">
              No debug data available for this conversation
            </div>
          )}

          {!loading && !error && debugData && (
            <div className="space-y-4">
              <div className="bg-gray-50 p-4 rounded-lg">
                <h3 className="font-medium mb-2">Debug Information</h3>
                <div className="text-sm">
                  <p><span className="font-medium">Timestamp:</span> {debugData.timestamp}</p>
                  {debugData.request_id && (
                    <p><span className="font-medium">Request ID:</span> {debugData.request_id}</p>
                  )}
                  {debugData.processing_time_ms && (
                    <p><span className="font-medium">Processing Time:</span> {debugData.processing_time_ms}ms</p>
                  )}
                </div>
              </div>

              {debugData.summary && (
                <div className="bg-blue-50 p-4 rounded-lg">
                  <h3 className="font-medium mb-2">Summary</h3>
                  <p className="text-sm">{debugData.summary}</p>
                </div>
              )}

              {debugData.error && (
                <div className="bg-red-50 p-4 rounded-lg">
                  <h3 className="font-medium mb-2">Error</h3>
                  <p className="text-sm text-red-700">{debugData.error}</p>
                </div>
              )}

              {debugData.steps && debugData.steps.length > 0 && (
                <div>
                  <h3 className="font-medium mb-2">Processing Steps</h3>
                  <div className="space-y-2">
                    {debugData.steps.map((step, index) => (
                      <div key={step.step_id || index} className="border rounded-lg p-3">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-medium">{step.title}</span>
                          <span className={`text-xs px-2 py-0.5 rounded ${
                            step.success ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                          }`}>
                            {step.success ? 'Success' : 'Failed'}
                          </span>
                        </div>
                        <p className="text-xs text-gray-500 mb-2">
                          {step.step_type} • {step.timestamp} 
                          {step.duration_ms && ` • ${step.duration_ms}ms`}
                        </p>
                        {step.description && (
                          <p className="text-sm mb-2">{step.description}</p>
                        )}
                        {step.error_message && (
                          <p className="text-xs text-red-600 mb-2">{step.error_message}</p>
                        )}
                        {Object.keys(step.data).length > 0 && (
                          <div className="mt-2">
                            <details>
                              <summary className="text-xs text-blue-600 cursor-pointer">View Data</summary>
                              <pre className="text-xs bg-gray-50 p-2 mt-1 rounded overflow-x-auto">
                                {JSON.stringify(step.data, null, 2)}
                              </pre>
                            </details>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default DebugSummary;
