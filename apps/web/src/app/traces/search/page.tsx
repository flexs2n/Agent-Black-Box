'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Trace } from '@/lib/api';

interface TraceWithScore extends Trace {
  similarity?: number;
}

export default function SearchPage() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<TraceWithScore[]>([]);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  const handleSearch = async () => {
    if (!query.trim()) return;
    if (!apiKey) {
      setError('Enter an API key first');
      return;
    }
    localStorage.setItem('bb_api_key', apiKey);
    setLoading(true);
    setError('');
    setSearched(true);
    try {
      const data = await api.semanticSearch(query, apiKey || undefined);
      setResults(data);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Search failed';
      setError(msg);
      if (msg.includes('OPENAI_API_KEY')) {
        setError('Semantic search requires OPENAI_API_KEY to be set on the server');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-6">
            <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
            <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">Traces</Link>
            <Link href="/traces/threads" className="text-sm font-medium text-gray-500 hover:text-gray-900">Threads</Link>
            <Link href="/traces/search" className="text-sm font-medium text-gray-900">Search</Link>
            <Link href="/diffs" className="text-sm font-medium text-gray-500 hover:text-gray-900">Diffs</Link>
            <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">Issues</Link>
            <Link href="/metrics" className="text-sm font-medium text-gray-500 hover:text-gray-900">Metrics</Link>
            <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
          </div>
        </div>
      </nav>

      <main className="max-w-4xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Semantic Trace Search</h1>

        <div className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
          <div className="flex gap-3">
            <input
              type="text"
              placeholder="Describe what you're looking for... e.g. 'traces where the agent failed to call a tool correctly'"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="flex-1 px-4 py-3 border border-gray-300 rounded-lg text-sm"
            />
            <button
              onClick={handleSearch}
              disabled={loading}
              className="px-6 py-3 bg-gray-900 text-white rounded-lg text-sm hover:bg-gray-800 disabled:opacity-50"
            >
              {loading ? 'Searching...' : 'Search'}
            </button>
          </div>

          <div className="mt-4">
            <label className="block text-xs font-medium text-gray-700 mb-1">API Key</label>
            <input
              type="text"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
              placeholder="Enter your API key..."
            />
          </div>

          {error && (
            <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
              {error}
            </div>
          )}
        </div>

        {loading && (
          <div className="bg-white border border-gray-200 rounded-lg p-8 text-center text-gray-500">
            Searching traces by semantic similarity...
          </div>
        )}

        {searched && !loading && results.length === 0 && !error && (
          <div className="bg-white border border-gray-200 rounded-lg p-8 text-center text-gray-500">
            No matching traces found. Traces may not be indexed yet — the indexer runs periodically.
          </div>
        )}

        {results.length > 0 && (
          <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
            <div className="px-6 py-3 bg-gray-50 border-b border-gray-200">
              <span className="text-sm text-gray-600">
                Found {results.length} result(s)
              </span>
            </div>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Similarity</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Trace ID</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Agent</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Duration</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {results.map((trace) => (
                  <tr key={trace.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm">
                      <span className="inline-flex px-2 py-1 text-xs font-medium rounded bg-blue-100 text-blue-800">
                        {trace.similarity ? (trace.similarity * 100).toFixed(1) + '%' : '-'}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-mono text-gray-900">
                      <Link href={`/traces/${trace.id}`} className="hover:text-blue-600">
                        {trace.id.slice(0, 8)}...
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">{trace.agent_name || '-'}</td>
                    <td className="px-6 py-4">
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded ${
                        trace.status === 'success' ? 'bg-green-100 text-green-800' :
                        trace.status === 'error' ? 'bg-red-100 text-red-800' :
                        'bg-yellow-100 text-yellow-800'
                      }`}>
                        {trace.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">{trace.duration_ms ? `${trace.duration_ms}ms` : '-'}</td>
                    <td className="px-6 py-4 text-sm text-gray-500">{new Date(trace.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
}
