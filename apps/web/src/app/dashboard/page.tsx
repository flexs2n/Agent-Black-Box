'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, DashboardResponse } from '@/lib/api';

export default function DashboardPage() {
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  useEffect(() => {
    if (apiKey) {
      api
        .getDashboard(apiKey)
        .then(setData)
        .catch(console.error)
        .finally(() => setLoading(false));
    }
  }, [apiKey]);

  const successRateColor = (rate: number) => {
    if (rate >= 95) return 'text-green-600 bg-green-100';
    if (rate >= 85) return 'text-yellow-600 bg-yellow-100';
    return 'text-red-600 bg-red-100';
  };

  const severityColor = (s: string) => {
    switch (s) {
      case 'critical':
        return 'bg-red-500';
      case 'high':
        return 'bg-orange-500';
      case 'medium':
        return 'bg-yellow-500';
      default:
        return 'bg-gray-500';
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-500">Loading dashboard...</div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-500">No data available. Make sure you have a valid API key.</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-xl font-bold text-gray-900">
            Blackbox
          </Link>
          <Link href="/dashboard" className="text-sm font-medium text-gray-900">
            Dashboard
          </Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Traces
          </Link>
          <Link href="/diff" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Diff
          </Link>
          <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Issues
          </Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Settings
          </Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>

        {/* Stat Cards */}
        <div className="grid grid-cols-5 gap-4 mb-8">
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="text-xs text-gray-500 mb-1">Total Traces</div>
            <div className="text-2xl font-bold text-gray-900">{data.stats.total_traces}</div>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="text-xs text-gray-500 mb-1">Success Rate</div>
            <div
              className={`text-2xl font-bold px-2 py-1 rounded ${successRateColor(data.stats.success_rate)}`}
            >
              {data.stats.success_rate.toFixed(1)}%
            </div>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="text-xs text-gray-500 mb-1">P95 Latency</div>
            <div className="text-2xl font-bold text-gray-900">{data.stats.p95_latency_ms}ms</div>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="text-xs text-gray-500 mb-1">Open Issues</div>
            <div className="text-2xl font-bold text-gray-900">{data.stats.open_issues}</div>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="text-xs text-gray-500 mb-1">Active Incidents</div>
            <div className="text-2xl font-bold text-gray-900">{data.stats.active_incidents}</div>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-6">
          {/* Charts */}
          <div className="space-y-6">
            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Trace Volume (24h)</h2>
              <div className="h-48 flex items-end gap-1">
                {data.traces_by_hour.map((t, i) => (
                  <div key={i} className="flex-1 flex flex-col items-center">
                    <div
                      className="w-full bg-gray-200 rounded-t"
                      style={{ height: `${Math.max(4, t.count * 4)}px` }}
                    >
                      <div
                        className="w-full h-full bg-green-500 rounded-t"
                        style={{ height: `${Math.max(4, t.success_count * 4)}px` }}
                      ></div>
                    </div>
                    <div className="text-xs text-gray-500 mt-1">{t.hour.slice(-5)}</div>
                  </div>
                ))}
              </div>
            </div>

            <div className="bg-white border border-gray-200 rounded-lg p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Token Usage</h2>
              <div className="text-sm text-gray-600">
                <div>Input: {data.stats.total_input_tokens.toLocaleString()}</div>
                <div>Output: {data.stats.total_output_tokens.toLocaleString()}</div>
                <div>
                  Total:{' '}
                  {(
                    data.stats.total_input_tokens + data.stats.total_output_tokens
                  ).toLocaleString()}
                </div>
              </div>
            </div>
          </div>

          {/* Issues Summary Panel */}
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Open Issues</h2>
            {data.open_issues.length === 0 ? (
              <p className="text-gray-500 text-sm">No open issues</p>
            ) : (
              <div className="space-y-2">
                {data.open_issues.map((issue) => (
                  <div
                    key={issue.id}
                    className="flex items-center justify-between p-2 border border-gray-200 rounded"
                  >
                    <div className="flex items-center gap-2">
                      <span
                        className={`w-2 h-2 rounded-full ${severityColor(issue.severity)}`}
                      ></span>
                      <span className="text-sm text-gray-900">{issue.title}</span>
                    </div>
                    <span className="text-xs text-gray-500">
                      {issue.occurrence_count} occurrences
                    </span>
                  </div>
                ))}
                <Link
                  href="/issues"
                  className="block mt-4 text-sm text-blue-600 hover:text-blue-800"
                >
                  View all issues →
                </Link>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
