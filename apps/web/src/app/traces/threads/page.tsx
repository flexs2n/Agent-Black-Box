'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, ThreadSummary, Trace } from '@/lib/api';

export default function ThreadsPage() {
  const [threads, setThreads] = useState<ThreadSummary[]>([]);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);
  const [selectedThread, setSelectedThread] = useState<string | null>(null);
  const [threadTraces, setThreadTraces] = useState<Trace[]>([]);
  const [tracesLoading, setTracesLoading] = useState(false);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  useEffect(() => {
    if (!apiKey) return;
    api.listThreads(apiKey)
      .then(setThreads)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [apiKey]);

  useEffect(() => {
    if (!selectedThread || !apiKey) return;
    setTracesLoading(true);
    api.listTraces(apiKey).then(allTraces => {
      const filtered = allTraces.filter(t => t.thread_id === selectedThread);
      filtered.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
      setThreadTraces(filtered);
    }).catch(console.error).finally(() => setTracesLoading(false));
  }, [selectedThread, apiKey]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'success': return 'text-green-600 bg-green-100';
      case 'error': return 'text-red-600 bg-red-100';
      default: return 'text-yellow-600 bg-yellow-100';
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-500">Loading threads...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">Traces</Link>
          <Link href="/traces/threads" className="text-sm font-medium text-gray-900">Threads</Link>
          <Link href="/diff" className="text-sm font-medium text-gray-500 hover:text-gray-900">Diff</Link>
          <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">Issues</Link>
          <Link href="/metrics" className="text-sm font-medium text-gray-500 hover:text-gray-900">Metrics</Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Threads</h1>

        <div className="flex gap-6">
          <div className="flex-1">
            <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Thread ID</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Traces</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Agent</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Status</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">First Seen</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Last Seen</th>
                  </tr>
                </thead>
                <tbody>
                  {threads.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-gray-500">No threads found</td>
                    </tr>
                  ) : (
                    threads.map((thread) => (
                      <tr
                        key={thread.thread_id}
                        className={`border-b border-gray-200 cursor-pointer hover:bg-gray-50 ${
                          selectedThread === thread.thread_id ? 'bg-blue-50' : ''
                        }`}
                        onClick={() => setSelectedThread(thread.thread_id)}
                      >
                        <td className="px-4 py-3 font-mono text-xs text-gray-900">{thread.thread_id.slice(0, 16)}...</td>
                        <td className="px-4 py-3 text-gray-900">{thread.trace_count}</td>
                        <td className="px-4 py-3 text-gray-600">{thread.last_agent_name || '-'}</td>
                        <td className="px-4 py-3">
                          <span className={`px-2 py-1 text-xs rounded ${statusColor(thread.last_status)}`}>
                            {thread.last_status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-500">{new Date(thread.first_seen_at).toLocaleString()}</td>
                        <td className="px-4 py-3 text-gray-500">{new Date(thread.last_seen_at).toLocaleString()}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {selectedThread && (
            <div className="w-96 bg-white border border-gray-200 rounded-lg p-4">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Thread Traces</h2>
              {tracesLoading ? (
                <div className="text-gray-500 text-sm">Loading traces...</div>
              ) : threadTraces.length === 0 ? (
                <div className="text-gray-500 text-sm">No traces in this thread</div>
              ) : (
                <div className="space-y-3 max-h-[70vh] overflow-y-auto">
                  {threadTraces.map((trace) => (
                    <Link
                      key={trace.id}
                      href={`/traces/${trace.id}`}
                      className="block p-3 border border-gray-200 rounded hover:bg-gray-50"
                    >
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs font-mono text-gray-500">{trace.id.slice(0, 12)}...</span>
                        <span className={`px-1.5 py-0.5 text-xs rounded ${statusColor(trace.status)}`}>
                          {trace.status}
                        </span>
                      </div>
                      <div className="text-xs text-gray-600">
                        {trace.agent_name && <span className="mr-2">{trace.agent_name}</span>}
                        {trace.duration_ms && <span>{trace.duration_ms}ms</span>}
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        {new Date(trace.created_at).toLocaleString()}
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
