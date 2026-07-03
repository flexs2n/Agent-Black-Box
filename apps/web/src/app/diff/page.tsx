'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Trace, DiffResult, WordDiffChunk, JsonDiffNode } from '@/lib/api';

export default function DiffPage({ searchParams }: { searchParams: { traceA?: string; traceB?: string; baseline?: string } }) {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [selectedA, setSelectedA] = useState<string>(searchParams.traceA || '');
  const [selectedB, setSelectedB] = useState<string>(searchParams.traceB || '');
  const [baselineLabel, setBaselineLabel] = useState(searchParams.baseline || '');
  const [diffResult, setDiffResult] = useState<DiffResult | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  const loadTraces = async () => {
    if (!apiKey) return;
    try {
      const data = await api.listTraces(apiKey);
      setTraces(data);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    if (apiKey) loadTraces();
  }, [apiKey]);

  const handleCompare = async () => {
    if (!selectedA || !selectedB) {
      alert('Select two traces to compare');
      return;
    }
    setLoading(true);
    try {
      const result = await api.computeDiff(selectedA, selectedB, apiKey || undefined);
      setDiffResult(result);
    } catch (e) {
      alert('Failed to compute diff: ' + (e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const renderWordDiff = (chunks: WordDiffChunk[]) => (
    <div className="font-mono text-xs leading-relaxed p-2 bg-white border border-gray-200 rounded whitespace-pre-wrap">
      {chunks.map((chunk, i) => {
        if (chunk.type === 'added') return <span key={i} className="bg-green-200 text-green-900">{chunk.text}</span>;
        if (chunk.type === 'removed') return <span key={i} className="bg-red-200 text-red-900 line-through">{chunk.text}</span>;
        return <span key={i} className="text-gray-700">{chunk.text}</span>;
      })}
    </div>
  );

  const renderJsonDiff = (nodes: JsonDiffNode[], depth = 0) => (
    <div style={{ paddingLeft: `${depth * 16}px` }} className="font-mono text-xs leading-relaxed">
      {nodes.map((node, i) => {
        if (node.type === 'nested') {
          return (
            <div key={i}>
              <span className="text-gray-500">{node.key}:</span>
              {node.children ? renderJsonDiff(node.children, depth + 1) : <span className="text-gray-400"> null</span>}
            </div>
          );
        }
        const valStr = node.type === 'added' ? JSON.stringify(node.valueB) : node.type === 'removed' ? JSON.stringify(node.valueA) : `${JSON.stringify(node.valueA)} → ${JSON.stringify(node.valueB)}`;
        return (
          <div key={i}>
            <span className="text-gray-500">{node.key}:</span>{' '}
            {node.type === 'added' && <span className="text-green-600">{valStr}</span>}
            {node.type === 'removed' && <span className="text-red-600 line-through">{valStr}</span>}
            {node.type === 'changed' && (
              <>
                <span className="text-red-600 line-through">{JSON.stringify(node.valueA)}</span>
                {' → '}
                <span className="text-green-600">{JSON.stringify(node.valueB)}</span>
              </>
            )}
            {node.type === 'unchanged' && <span className="text-gray-400">{valStr}</span>}
          </div>
        );
      })}
    </div>
  );

  const similarityColor = (score: number) => {
    if (score >= 90) return 'text-green-600 bg-green-100';
    if (score >= 70) return 'text-yellow-600 bg-yellow-100';
    if (score >= 50) return 'text-orange-600 bg-orange-100';
    return 'text-red-600 bg-red-100';
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">Traces</Link>
          <Link href="/diff" className="text-sm font-medium text-gray-900">Diff</Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Trace Diff</h1>

        <div className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
          <div className="grid grid-cols-3 gap-4 mb-4">
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Trace A</label>
              <select
                value={selectedA}
                onChange={(e) => setSelectedA(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
              >
                <option value="">Select trace...</option>
                {traces.map(t => (
                  <option key={t.id} value={t.id}>{t.id.slice(0, 8)}... {t.agent_name || ''}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Trace B</label>
              <select
                value={selectedB}
                onChange={(e) => setSelectedB(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
              >
                <option value="">Select trace...</option>
                {traces.map(t => (
                  <option key={t.id} value={t.id}>{t.id.slice(0, 8)}... {t.agent_name || ''}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">Baseline (optional)</label>
              <input
                type="text"
                value={baselineLabel}
                onChange={(e) => setBaselineLabel(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                placeholder="Baseline label..."
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              placeholder="API Key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded text-sm w-64"
            />
            <button
              onClick={handleCompare}
              disabled={loading || !selectedA || !selectedB}
              className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Computing...' : 'Compute Diff'}
            </button>
          </div>
        </div>

        {diffResult && (
          <div className="space-y-6">
            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-gray-900">Similarity Score</h2>
                <span className={`px-3 py-1 text-sm font-medium rounded ${similarityColor(diffResult.similarityScore)}`}>
                  {diffResult.similarityScore.toFixed(1)}%
                </span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-4">
                <div
                  className={`h-4 rounded-full ${
                    diffResult.similarityScore >= 90 ? 'bg-green-500' :
                    diffResult.similarityScore >= 70 ? 'bg-yellow-500' :
                    diffResult.similarityScore >= 50 ? 'bg-orange-500' : 'bg-red-500'
                  }`}
                  style={{ width: `${diffResult.similarityScore}%` }}
                />
              </div>
              <div className="grid grid-cols-5 gap-4 mt-6">
                {(['durationMs' as const, 'inputTokens', 'outputTokens', 'toolCallCount', 'llmCallCount'] as const).map(metric => {
                  const item = diffResult.metricDelta[metric];
                  const labels: Record<string, string> = {
                    durationMs: 'Duration',
                    inputTokens: 'Input Tokens',
                    outputTokens: 'Output Tokens',
                    toolCallCount: 'Tool Calls',
                    llmCallCount: 'LLM Calls',
                  };
                  const delta = item.delta;
                  const deltaStr = delta > 0 ? `+${delta}` : `${delta}`;
                  const color = delta > 0 ? 'text-green-600' : delta < 0 ? 'text-red-600' : 'text-gray-500';
                  return (
                    <div key={metric} className="text-center">
                      <div className="text-xs text-gray-500 mb-1">{labels[metric]}</div>
                      <div className="text-sm font-semibold text-gray-900">{item.a} → {item.b}</div>
                      <div className={`text-xs font-medium ${color}`}>{deltaStr}</div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Span Diff</h2>
              <div className="space-y-2">
                {diffResult.spanDiffs.map((diff, idx) => (
                  <div
                    key={idx}
                    className={`p-3 rounded border ${
                      diff.status === 'unchanged' ? 'bg-gray-50 border-gray-200' :
                      diff.status === 'added' ? 'bg-green-50 border-green-200' :
                      diff.status === 'removed' ? 'bg-red-50 border-red-200 line-through opacity-60' :
                      diff.status === 'changed' ? 'bg-yellow-50 border-yellow-200' :
                      'bg-blue-50 border-blue-200'
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-0.5 text-xs font-medium rounded ${
                        diff.status === 'unchanged' ? 'bg-gray-200 text-gray-700' :
                        diff.status === 'added' ? 'bg-green-200 text-green-800' :
                        diff.status === 'removed' ? 'bg-red-200 text-red-800' :
                        diff.status === 'changed' ? 'bg-yellow-200 text-yellow-800' :
                        'bg-blue-200 text-blue-800'
                      }`}>
                        {diff.status.toUpperCase()}
                      </span>
                      <span className="font-mono text-xs text-gray-500">{diff.spanKind}</span>
                      <span className="text-sm text-gray-900">{diff.name}</span>
                    </div>
                    {diff.attributeDiffs && diff.attributeDiffs.length > 0 && (
                      <div className="mt-2 pl-4 space-y-1">
                        {diff.attributeDiffs.map((ad, i) => (
                          <div key={i} className="text-xs font-mono">
                            <span className="text-gray-500">{ad.key}:</span>{' '}
                            {ad.changeType === 'removed' && <span className="text-red-600 line-through">{ad.valueA}</span>}
                            {ad.changeType === 'added' && <span className="text-green-600">{ad.valueB}</span>}
                            {ad.changeType === 'changed' && (
                              <>
                                <span className="text-red-600 line-through">{ad.valueA}</span>
                                {' → '}
                                <span className="text-green-600">{ad.valueB}</span>
                              </>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                    {diff.contentDiff && (
                      <div className="mt-2 pl-4">
                        <div className="text-xs font-semibold text-gray-700 mb-1">
                          {diff.contentDiff.type === 'prompt' ? 'Prompt Diff' : 'Tool Args Diff'}
                        </div>
                        {diff.contentDiff.wordDiff ? (
                          renderWordDiff(diff.contentDiff.wordDiff)
                        ) : diff.contentDiff.jsonDiff ? (
                          renderJsonDiff(diff.contentDiff.jsonDiff)
                        ) : null}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}