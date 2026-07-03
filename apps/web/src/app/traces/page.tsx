'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Trace } from '@/lib/api';

export default function TracesPage() {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState({ agent: '', status: '', environment: '' });
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  const loadTraces = async () => {
    setLoading(true);
    try {
      const data = await api.listTraces(apiKey || undefined);
      setTraces(data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (apiKey) {
      localStorage.setItem('bb_api_key', apiKey);
      loadTraces();
    }
  }, [apiKey]);

  const toggleSelect = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  };

  const handleDeleteSelected = async () => {
    if (!confirm(`Delete ${selected.size} traces?`)) return;
    await api.deleteTraces(Array.from(selected), apiKey || undefined);
    setSelected(new Set());
    loadTraces();
  };

  const handleCompare = () => {
    if (selected.size !== 2) {
      alert('Select exactly 2 traces to compare');
      return;
    }
    const ids = Array.from(selected);
    window.location.href = `/diff?traceA=${ids[0]}&traceB=${ids[1]}`;
  };

  const filtered = traces.filter(t => {
    if (filter.agent && t.agent_name !== filter.agent) return false;
    if (filter.status && t.status !== filter.status) return false;
    if (filter.environment && t.environment !== filter.environment) return false;
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-6">
            <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
            <Link href="/traces" className="text-sm font-medium text-gray-900">Traces</Link>
            <Link href="/traces/threads" className="text-sm font-medium text-gray-500 hover:text-gray-900">Threads</Link>
            <Link href="/diffs" className="text-sm font-medium text-gray-500 hover:text-gray-900">Diffs</Link>
            <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">Issues</Link>
            <Link href="/metrics" className="text-sm font-medium text-gray-500 hover:text-gray-900">Metrics</Link>
            <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-gray-900">Traces</h1>
          <div className="flex items-center gap-2">
            <input
              type="text"
              placeholder="API Key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded text-sm w-64"
            />
            <button
              onClick={loadTraces}
              className="px-4 py-2 bg-gray-900 text-white rounded text-sm hover:bg-gray-800"
            >
              Refresh
            </button>
          </div>
        </div>

        <div className="bg-white border border-gray-200 rounded-lg mb-6 p-4">
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Agent</label>
              <input
                type="text"
                value={filter.agent}
                onChange={(e) => setFilter({ ...filter, agent: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                placeholder="Filter by agent..."
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Status</label>
              <select
                value={filter.status}
                onChange={(e) => setFilter({ ...filter, status: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
              >
                <option value="">All</option>
                <option value="success">Success</option>
                <option value="error">Error</option>
                <option value="flagged">Flagged</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Environment</label>
              <select
                value={filter.environment}
                onChange={(e) => setFilter({ ...filter, environment: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
              >
                <option value="">All</option>
                <option value="production">Production</option>
                <option value="staging">Staging</option>
                <option value="development">Development</option>
              </select>
            </div>
          </div>
        </div>

        {selected.size > 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6 flex items-center justify-between">
            <span className="text-sm text-blue-900">{selected.size} trace(s) selected</span>
            <div className="flex gap-2">
              <button
                onClick={handleCompare}
                className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
              >
                Compare Selected
              </button>
              <button
                onClick={handleDeleteSelected}
                className="px-4 py-2 bg-red-600 text-white rounded text-sm hover:bg-red-700"
              >
                Delete Selected
              </button>
            </div>
          </div>
        )}

        <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase w-10">
                  <input
                    type="checkbox"
                    checked={selected.size === filtered.length && filtered.length > 0}
                    onChange={(e) => {
                      if (e.target.checked) setSelected(new Set(filtered.map(t => t.id)));
                      else setSelected(new Set());
                    }}
                    className="rounded"
                  />
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Trace ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Agent</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Duration</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {loading ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-gray-500">Loading traces...</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-gray-500">No traces found</td></tr>
              ) : (
                filtered.map(trace => (
                  <tr key={trace.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <input
                        type="checkbox"
                        checked={selected.has(trace.id)}
                        onChange={() => toggleSelect(trace.id)}
                        className="rounded"
                      />
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
                    <td className="px-6 py-4 text-sm">
                      <Link href={`/traces/${trace.id}`} className="text-blue-600 hover:text-blue-800 mr-3">View</Link>
                      <button
                        onClick={() => {
                          if (confirm(`Delete trace ${trace.id.slice(0, 8)}?`)) {
                            api.deleteTrace(trace.id, apiKey || undefined);
                            loadTraces();
                          }
                        }}
                        className="text-red-600 hover:text-red-800"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  );
}