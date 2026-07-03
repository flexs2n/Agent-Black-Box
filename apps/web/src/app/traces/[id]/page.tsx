'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Trace, Span } from '@/lib/api';

export default function TraceDetailPage({ params }: { params: { id: string } }) {
  const [trace, setTrace] = useState<(Trace & { spans?: Span[] }) | null>(null);
  const [selectedSpan, setSelectedSpan] = useState<Span | null>(null);
  const [tab, setTab] = useState<'overview' | 'inputoutput' | 'attributes' | 'timeline'>('overview');
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  useEffect(() => {
    if (!apiKey) return;
    setLoading(true);
    api.getTrace(params.id, apiKey).then(data => {
      setTrace(data);
      if (data.spans && data.spans.length > 0) {
        setSelectedSpan(data.spans[0]);
      }
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [params.id, apiKey]);

  if (loading) return <div className="min-h-screen flex items-center justify-center">Loading...</div>;
  if (!trace) return <div className="min-h-screen flex items-center justify-center">Trace not found</div>;

  const statusColor = trace.status === 'success' ? 'text-green-600 bg-green-100' : trace.status === 'error' ? 'text-red-600 bg-red-100' : 'text-yellow-600 bg-yellow-100';

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">Traces</Link>
          <Link href="/traces/threads" className="text-sm font-medium text-gray-500 hover:text-gray-900">Threads</Link>
          <Link href="/diff" className="text-sm font-medium text-gray-500 hover:text-gray-900">Diff</Link>
          <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">Issues</Link>
          <Link href="/metrics" className="text-sm font-medium text-gray-500 hover:text-gray-900">Metrics</Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="mb-6">
          <div className="flex items-center gap-4 mb-4">
            <h1 className="text-2xl font-bold text-gray-900">Trace Detail</h1>
            <span className={`px-2 py-1 text-xs font-medium rounded ${statusColor}`}>{trace.status}</span>
          </div>
          <div className="text-sm text-gray-500 space-x-4">
            <span>ID: <code className="font-mono">{trace.id}</code></span>
            {trace.agent_name && <span>Agent: {trace.agent_name}</span>}
            {trace.duration_ms && <span>Duration: {trace.duration_ms}ms</span>}
            {trace.started_at && <span>Started: {new Date(trace.started_at).toLocaleString()}</span>}
          </div>
          <div className="mt-4 flex gap-2">
            <Link
              href={`/diff?traceA=${trace.id}`}
              className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
            >
              Compare Trace
            </Link>
          </div>
        </div>

        <div className="grid grid-cols-12 gap-6">
          <div className="col-span-4 bg-white border border-gray-200 rounded-lg p-4 h-[600px] overflow-auto">
            <h2 className="text-sm font-semibold text-gray-900 mb-3">Spans</h2>
            <div className="space-y-1">
              {trace.spans?.map(span => (
                <div
                  key={span.span_id}
                  onClick={() => setSelectedSpan(span)}
                  className={`p-2 rounded cursor-pointer text-sm ${
                    selectedSpan?.span_id === span.span_id
                      ? 'bg-blue-50 border border-blue-200'
                      : 'hover:bg-gray-50 border border-transparent'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <span className={`w-2 h-2 rounded-full ${
                      span.status === 'ok' ? 'bg-green-500' :
                      span.status === 'error' ? 'bg-red-500' : 'bg-gray-400'
                    }`} />
                    <span className="font-mono text-xs text-gray-500">{span.span_kind}</span>
                    <span className="truncate">{span.name}</span>
                  </div>
                  <div className="text-xs text-gray-400 mt-1">{span.duration_ms}ms</div>
                </div>
              ))}
            </div>
          </div>

          <div className="col-span-8 bg-white border border-gray-200 rounded-lg">
            <div className="border-b border-gray-200">
              <nav className="flex gap-4 px-4">
                {(['overview', 'inputoutput', 'attributes', 'timeline'] as const).map(t => (
                  <button
                    key={t}
                    onClick={() => setTab(t)}
                    className={`py-3 text-sm font-medium border-b-2 ${
                      tab === t
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                  >
                    {t === 'inputoutput' ? 'Input/Output' : t.charAt(0).toUpperCase() + t.slice(1)}
                  </button>
                ))}
              </nav>
            </div>

            <div className="p-6">
              {!selectedSpan ? (
                <p className="text-gray-500 text-sm">Select a span to view details</p>
              ) : (
                <div>
                  <div className="mb-4">
                    <h3 className="text-lg font-semibold text-gray-900">{selectedSpan.name}</h3>
                    <p className="text-sm text-gray-500">Span ID: {selectedSpan.span_id}</p>
                  </div>

                  {tab === 'overview' && (
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label className="text-xs text-gray-500">Status</label>
                          <p className="text-sm font-medium text-gray-900">{selectedSpan.status}</p>
                        </div>
                        <div>
                          <label className="text-xs text-gray-500">Duration</label>
                          <p className="text-sm font-medium text-gray-900">{selectedSpan.duration_ms}ms</p>
                        </div>
                        <div>
                          <label className="text-xs text-gray-500">Span Kind</label>
                          <p className="text-sm font-medium text-gray-900">{selectedSpan.span_kind}</p>
                        </div>
                        <div>
                          <label className="text-xs text-gray-500">Started At</label>
                          <p className="text-sm font-medium text-gray-900">{new Date(selectedSpan.started_at).toLocaleString()}</p>
                        </div>
                      </div>
                    </div>
                  )}

                  {tab === 'inputoutput' && (
                    <div className="space-y-4">
                      {(() => {
                        try {
                          const attrs = JSON.parse(selectedSpan.attributes || '{}');
                          if (attrs['tool.input']) {
                            return (
                              <div>
                                <label className="text-xs text-gray-500 mb-1 block">Tool Input</label>
                                <pre className="bg-gray-900 text-gray-100 p-4 rounded text-xs overflow-x-auto">
                                  {JSON.stringify(JSON.parse(attrs['tool.input']), null, 2)}
                                </pre>
                              </div>
                            );
                          }
                          if (attrs['gen_ai.prompt']) {
                            return (
                              <div>
                                <label className="text-xs text-gray-500 mb-1 block">Prompt</label>
                                <pre className="bg-gray-900 text-gray-100 p-4 rounded text-xs whitespace-pre-wrap">
                                  {attrs['gen_ai.prompt']}
                                </pre>
                              </div>
                            );
                          }
                          return <p className="text-sm text-gray-500">No input/output data for this span</p>;
                        } catch {
                          return <p className="text-sm text-gray-500">No input/output data</p>;
                        }
                      })()}
                    </div>
                  )}

                  {tab === 'attributes' && (
                    <div>
                      {(() => {
                        try {
                          const attrs = JSON.parse(selectedSpan.attributes || '{}');
                          const entries = Object.entries(attrs);
                          if (entries.length === 0) return <p className="text-sm text-gray-500">No attributes</p>;
                          return (
                            <table className="w-full text-sm">
                              <tbody>
                                {entries.map(([key, value]) => (
                                  <tr key={key} className="border-b border-gray-100">
                                    <td className="py-2 pr-4 font-mono text-xs text-gray-500">{key}</td>
                                    <td className="py-2 text-gray-900 font-mono text-xs break-all">{String(value)}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          );
                        } catch {
                          return <p className="text-sm text-gray-500">No attributes</p>;
                        }
                      })()}
                    </div>
                  )}

                  {tab === 'timeline' && (
                    <div>
                      <p className="text-sm text-gray-500">Timeline visualization</p>
                      <div className="mt-4 space-y-2">
                        {trace.spans?.map(span => {
                          const maxDur = Math.max(...(trace.spans?.map(s => s.duration_ms) || [1]));
                          const width = (span.duration_ms / maxDur) * 100;
                          return (
                            <div key={span.span_id} className="flex items-center gap-2">
                              <span className="text-xs font-mono text-gray-500 w-24 truncate">{span.name}</span>
                              <div className="flex-1 bg-gray-100 rounded-full h-4 relative">
                                <div
                                  className={`absolute h-full rounded-full ${
                                    span.status === 'ok' ? 'bg-green-500' : span.status === 'error' ? 'bg-red-500' : 'bg-gray-400'
                                  }`}
                                  style={{ width: `${Math.max(width, 5)}%` }}
                                />
                              </div>
                              <span className="text-xs text-gray-500 w-16 text-right">{span.duration_ms}ms</span>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}