'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, PresetMetric, MetricWithSparkline, Monitor, MonitorCreate, Incident, IncidentStatus } from '@/lib/api';

export default function MetricsPage() {
  const [apiKey, setApiKey] = useState('');
  const [presetMetrics, setPresetMetrics] = useState<PresetMetric[]>([]);
  const [customMetrics, setCustomMetrics] = useState<MetricWithSparkline[]>([]);
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newMonitor, setNewMonitor] = useState<MonitorCreate>({
    metric_id: '',
    condition: 'above',
    threshold: 0,
    severity: 'medium',
  });

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  useEffect(() => {
    if (!apiKey) return;
    Promise.all([
      api.listMetrics(apiKey).then(r => { setPresetMetrics(r.preset); setCustomMetrics(r.custom); }),
      api.listMonitors(apiKey).then(setMonitors),
      api.listIncidents(apiKey).then(setIncidents),
    ]).catch(console.error).finally(() => setLoading(false));
  }, [apiKey]);

  const handleCreateMonitor = async () => {
    if (!newMonitor.metric_id) return;
    try {
      const created = await api.createMonitor(newMonitor, apiKey);
      setMonitors([...monitors, created]);
      setShowCreate(false);
      setNewMonitor({ metric_id: '', condition: 'above', threshold: 0, severity: 'medium' });
    } catch (e) {
      alert('Failed to create monitor: ' + (e as Error).message);
    }
  };

  const handleDeleteMonitor = async (id: string) => {
    if (!confirm('Delete this monitor?')) return;
    await api.deleteMonitor(id, apiKey);
    setMonitors(monitors.filter(m => m.id !== id));
  };

  const handleResolveIncident = async (id: string) => {
    await api.updateIncidentStatus(id, 'resolved', apiKey);
    setIncidents(incidents.map(i => i.id === id ? { ...i, status: 'resolved' as IncidentStatus } : i));
  };

  const handleDismissIncident = async (id: string) => {
    await api.updateIncidentStatus(id, 'dismissed', apiKey);
    setIncidents(incidents.map(i => i.id === id ? { ...i, status: 'dismissed' as IncidentStatus } : i));
  };

  const statusDotColor = (status: string) => {
    switch (status) {
      case 'ok': return 'bg-green-500';
      case 'alerting': return 'bg-red-500';
      case 'resolved': return 'bg-yellow-500';
      default: return 'bg-gray-500';
    }
  };

  const formatValue = (metric: PresetMetric) => {
    if (metric.format === 'percentage') return (metric.value * 100).toFixed(1) + '%';
    if (metric.format === 'duration') return metric.value.toFixed(0) + 'ms';
    if (metric.format === 'tokens') return metric.value.toLocaleString();
    return metric.value.toFixed(2);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-gray-500">Loading metrics...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-xl font-bold text-gray-900">Blackbox</Link>
          <Link href="/dashboard" className="text-sm font-medium text-gray-500 hover:text-gray-900">Dashboard</Link>
          <Link href="/traces" className="text-sm font-medium text-gray-500 hover:text-gray-900">Traces</Link>
          <Link href="/traces/threads" className="text-sm font-medium text-gray-500 hover:text-gray-900">Threads</Link>
          <Link href="/diff" className="text-sm font-medium text-gray-500 hover:text-gray-900">Diff</Link>
          <Link href="/issues" className="text-sm font-medium text-gray-500 hover:text-gray-900">Issues</Link>
          <Link href="/metrics" className="text-sm font-medium text-gray-900">Metrics</Link>
          <Link href="/settings" className="text-sm font-medium text-gray-500 hover:text-gray-900">Settings</Link>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">Metrics & Monitors</h1>

        {/* Preset Metrics Grid */}
        <section className="mb-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Preset Metrics</h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {presetMetrics.map((m) => {
              const relatedMonitor = monitors.find(mo => mo.metric_id === m.slug);
              return (
                <div key={m.slug} className="bg-white border border-gray-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs text-gray-500 uppercase tracking-wide">{m.name}</span>
                    {relatedMonitor && (
                      <span className={`w-2 h-2 rounded-full ${statusDotColor(relatedMonitor.status)}`} title={relatedMonitor.status}></span>
                    )}
                  </div>
                  <div className="text-2xl font-bold text-gray-900">{formatValue(m)}</div>
                  {m.sparkline && m.sparkline.length > 0 && (
                    <div className="mt-2 h-8 flex items-end gap-px">
                      {m.sparkline.map((v, i) => (
                        <div
                          key={i}
                          className="flex-1 bg-blue-200 rounded-t"
                          style={{ height: `${Math.max(3, (v / Math.max(...m.sparkline)) * 100)}%` }}
                        ></div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </section>

        {/* Custom Metrics */}
        {customMetrics.length > 0 && (
          <section className="mb-8">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Custom Metrics</h2>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {customMetrics.map((m) => (
                <div key={m.id} className="bg-white border border-gray-200 rounded-lg p-4">
                  <div className="text-xs text-gray-500 uppercase tracking-wide mb-1">{m.name}</div>
                  <div className="text-2xl font-bold text-gray-900">{m.current_value.toFixed(2)}</div>
                  {m.sparkline.length > 0 && (
                    <div className="mt-2 h-8 flex items-end gap-px">
                      {m.sparkline.map((v, i) => (
                        <div
                          key={i}
                          className="flex-1 bg-green-200 rounded-t"
                          style={{ height: `${Math.max(3, (v / Math.max(...m.sparkline)) * 100)}%` }}
                        ></div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Monitors */}
        <section className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">Monitors</h2>
            <button
              onClick={() => setShowCreate(true)}
              className="px-3 py-1.5 bg-black text-white text-sm rounded hover:bg-gray-800"
            >
              Create Monitor
            </button>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Metric</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Condition</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Threshold</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Status</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Last Fired</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Actions</th>
                </tr>
              </thead>
              <tbody>
                {monitors.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-gray-500">No monitors configured</td>
                  </tr>
                ) : (
                  monitors.map((m) => (
                    <tr key={m.id} className="border-b border-gray-200">
                      <td className="px-4 py-3 font-medium text-gray-900">{m.metric_id}</td>
                      <td className="px-4 py-3 text-gray-600">{m.condition}</td>
                      <td className="px-4 py-3 text-gray-600">{m.threshold}</td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1.5 px-2 py-1 text-xs rounded ${
                          m.status === 'alerting' ? 'bg-red-100 text-red-800' : m.status === 'ok' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                        }`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${statusDotColor(m.status)}`}></span>
                          {m.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-500">
                        {m.last_fired_at ? new Date(m.last_fired_at).toLocaleString() : '-'}
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => handleDeleteMonitor(m.id)}
                          className="text-red-600 hover:text-red-800 text-xs"
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
        </section>

        {/* Incidents */}
        <section>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Incidents</h2>
          <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Monitor</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Status</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Created</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Resolved</th>
                  <th className="text-left px-4 py-3 font-medium text-gray-700">Actions</th>
                </tr>
              </thead>
              <tbody>
                {incidents.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-gray-500">No incidents</td>
                  </tr>
                ) : (
                  incidents.map((inc) => (
                    <tr key={inc.id} className="border-b border-gray-200">
                      <td className="px-4 py-3 font-medium text-gray-900">{inc.monitor_id}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 text-xs rounded ${
                          inc.status === 'unresolved' ? 'bg-red-100 text-red-800' : inc.status === 'resolved' ? 'bg-green-100 text-green-800' : inc.status === 'dismissed' ? 'bg-gray-100 text-gray-800' : 'bg-blue-100 text-blue-800'
                        }`}>
                          {inc.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-600">{new Date(inc.created_at).toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-600">
                        {inc.resolved_at ? new Date(inc.resolved_at).toLocaleString() : '-'}
                      </td>
                      <td className="px-4 py-3 flex gap-2">
                        {inc.status === 'unresolved' && (
                          <>
                            <button
                              onClick={() => handleResolveIncident(inc.id)}
                              className="text-green-600 hover:text-green-800 text-xs"
                            >
                              Resolve
                            </button>
                            <button
                              onClick={() => handleDismissIncident(inc.id)}
                              className="text-gray-600 hover:text-gray-800 text-xs"
                            >
                              Dismiss
                            </button>
                          </>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>

      {/* Create Monitor Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Create Monitor</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Metric ID</label>
                <select
                  value={newMonitor.metric_id}
                  onChange={(e) => setNewMonitor({ ...newMonitor, metric_id: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                >
                  <option value="">Select a metric...</option>
                  {presetMetrics.map((m) => (
                    <option key={m.slug} value={m.slug}>{m.name} ({m.slug})</option>
                  ))}
                  {customMetrics.map((m) => (
                    <option key={m.id} value={m.id}>{m.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Condition</label>
                <select
                  value={newMonitor.condition}
                  onChange={(e) => setNewMonitor({ ...newMonitor, condition: e.target.value as 'above' | 'below' })}
                  className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                >
                  <option value="above">Above</option>
                  <option value="below">Below</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Threshold</label>
                <input
                  type="number"
                  step="any"
                  value={newMonitor.threshold}
                  onChange={(e) => setNewMonitor({ ...newMonitor, threshold: parseFloat(e.target.value) || 0 })}
                  className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Severity</label>
                <select
                  value={newMonitor.severity}
                  onChange={(e) => setNewMonitor({ ...newMonitor, severity: e.target.value as 'critical' | 'high' | 'medium' | 'low' })}
                  className="w-full px-3 py-2 border border-gray-300 rounded text-sm"
                >
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                </select>
              </div>
              <div className="flex justify-end gap-3 pt-4">
                <button
                  onClick={() => setShowCreate(false)}
                  className="px-4 py-2 text-sm text-gray-700 hover:text-gray-900"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreateMonitor}
                  className="px-4 py-2 bg-black text-white text-sm rounded hover:bg-gray-800"
                >
                  Create
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
