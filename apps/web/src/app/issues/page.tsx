'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Issue, IssueStatus } from '@/lib/api';

export default function IssuesPage() {
  const [issues, setIssues] = useState<Issue[]>([]);
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('open');

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  useEffect(() => {
    if (apiKey) {
      api
        .listIssues(apiKey)
        .then(setIssues)
        .catch(console.error)
        .finally(() => setLoading(false));
    }
  }, [apiKey, statusFilter]);

  const severityColors: Record<string, string> = {
    critical: 'bg-red-500',
    high: 'bg-orange-500',
    medium: 'bg-yellow-500',
    low: 'bg-gray-500',
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-500">Loading issues...</div>
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
          <Link href="/dashboard" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Dashboard
          </Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Traces
          </Link>
          <Link href="/diff" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Diff
          </Link>
          <Link href="/issues" className="text-sm font-medium text-gray-900">
            Issues
          </Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">
            Settings
          </Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-2xl font-bold text-gray-900">Issues</h1>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded text-sm"
          >
            <option value="">All Statuses</option>
            <option value="open">Open</option>
            <option value="acknowledged">Acknowledged</option>
            <option value="resolved">Resolved</option>
            <option value="dismissed">Dismissed</option>
          </select>
        </div>

        <div className="flex gap-6">
          {/* Issue Table */}
          <div className="flex-1">
            <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Severity</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Title</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Occurrences</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">First Seen</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Last Seen</th>
                    <th className="text-left px-4 py-3 font-medium text-gray-700">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {issues.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                        No issues found
                      </td>
                    </tr>
                  ) : (
                    issues.map((issue) => (
                      <tr
                        key={issue.id}
                        className={`border-b border-gray-200 cursor-pointer hover:bg-gray-50 ${
                          selectedIssue?.id === issue.id ? 'bg-blue-50' : ''
                        }`}
                        onClick={() => setSelectedIssue(issue)}
                      >
                        <td className="px-4 py-3">
                          <span
                            className={`w-2 h-2 rounded-full inline-block ${severityColors[issue.severity]}`}
                          ></span>
                        </td>
                        <td className="px-4 py-3 font-medium text-gray-900">{issue.title}</td>
                        <td className="px-4 py-3 text-gray-600">{issue.occurrence_count}</td>
                        <td className="px-4 py-3 text-gray-600">
                          {new Date(issue.first_seen_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {new Date(issue.last_seen_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-1 text-xs rounded ${
                              issue.status === 'open'
                                ? 'bg-yellow-100 text-yellow-800'
                                : issue.status === 'acknowledged'
                                  ? 'bg-blue-100 text-blue-800'
                                  : 'bg-green-100 text-green-800'
                            }`}
                          >
                            {issue.status}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Issue Detail Panel */}
          {selectedIssue && (
            <div className="w-96 bg-white border border-gray-200 rounded-lg p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">{selectedIssue.title}</h2>
              <div className="space-y-4">
                <div>
                  <span className="text-xs text-gray-500">Evaluator</span>
                  <div className="text-sm text-gray-900">{selectedIssue.evaluator}</div>
                </div>
                <div>
                  <span className="text-xs text-gray-500">Severity</span>
                  <div className="text-sm">
                    <span
                      className={`px-2 py-1 text-xs rounded ${severityColors[selectedIssue.severity]} text-white`}
                    >
                      {selectedIssue.severity}
                    </span>
                  </div>
                </div>
                {selectedIssue.root_cause && (
                  <div>
                    <span className="text-xs text-gray-500">Root Cause</span>
                    <div className="text-sm text-gray-900">{selectedIssue.root_cause}</div>
                  </div>
                )}
                {selectedIssue.suggested_fix && (
                  <div>
                    <span className="text-xs text-gray-500">Suggested Fix</span>
                    <div className="text-sm text-gray-900">{selectedIssue.suggested_fix}</div>
                  </div>
                )}
                <div className="pt-4 border-t border-gray-200">
                  <span className="text-xs text-gray-500 mb-2 block">Status</span>
                  <select
                    value={selectedIssue.status}
                    onChange={(e) => {
                      if (apiKey) {
                        const newStatus = e.target.value as IssueStatus;
                        api
                          .updateIssueStatus(selectedIssue.id, newStatus, apiKey)
                          .then(() => {
                            setIssues(
                              issues.map((i) =>
                                i.id === selectedIssue.id
                                  ? { ...i, status: newStatus }
                                  : i
                              )
                            );
                            setSelectedIssue({ ...selectedIssue, status: newStatus });
                          });
                      }
                    }}
                    className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                  >
                    <option value="open">Open</option>
                    <option value="acknowledged">Acknowledged</option>
                    <option value="resolved">Resolved</option>
                    <option value="dismissed">Dismissed</option>
                  </select>
                </div>
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
