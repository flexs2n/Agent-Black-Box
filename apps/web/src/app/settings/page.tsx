'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { api, Project, APIKey, Baseline } from '@/lib/api';

export default function SettingsPage() {
  const [apiKey, setApiKey] = useState('');
  const [projects, setProjects] = useState<Project[]>([]);
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [baselines, setBaselines] = useState<Baseline[]>([]);
  const [newProjectName, setNewProjectName] = useState('');
  const [newProjectSlug, setNewProjectSlug] = useState('');
  const [newKeyLabel, setNewKeyLabel] = useState('');
  const [newBaselineLabel, setNewBaselineLabel] = useState('');
  const [loading, setLoading] = useState(false);
  const [activeProjectId, setActiveProjectId] = useState('');

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('bb_api_key') : null;
    if (saved) setApiKey(saved);
  }, []);

  const loadProjects = async () => {
    if (!apiKey) return;
    try {
      const data = await api.listProjects(apiKey);
      setProjects(data);
      if (data.length > 0 && !activeProjectId) setActiveProjectId(data[0].id);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    if (apiKey) loadProjects();
  }, [apiKey]);

  const loadKeys = async (projectId: string) => {
    if (!apiKey || !projectId) return;
    try {
      const data = await api.listApiKeys(projectId, apiKey);
      setKeys(data);
    } catch (e) {
      console.error(e);
    }
  };

  const loadBaselines = async () => {
    if (!apiKey) return;
    try {
      const data = await api.listBaselines(apiKey);
      setBaselines(data);
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    if (activeProjectId) loadKeys(activeProjectId);
  }, [activeProjectId, apiKey]);

  useEffect(() => {
    loadBaselines();
  }, [apiKey]);

  const handleCreateProject = async () => {
    if (!newProjectName || !newProjectSlug) return;
    setLoading(true);
    try {
      const project = await api.createProject(newProjectName, newProjectSlug, apiKey);
      setProjects([...projects, project]);
      setNewProjectName('');
      setNewProjectSlug('');
      setActiveProjectId(project.id);
    } catch (e) {
      alert('Failed to create project: ' + (e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateKey = async () => {
    if (!newKeyLabel || !activeProjectId) return;
    setLoading(true);
    try {
      const result = await api.createApiKey(activeProjectId, newKeyLabel, apiKey);
      alert('New API key (save this now): ' + result.plain_key);
      setNewKeyLabel('');
      loadKeys(activeProjectId);
    } catch (e) {
      alert('Failed to create key: ' + (e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteKey = async (projectId: string, keyId: string) => {
    if (!confirm('Revoke this API key?')) return;
    await api.deleteApiKey(projectId, keyId, apiKey);
    loadKeys(projectId);
  };

  const handleDeleteBaseline = async (id: string) => {
    if (!confirm('Delete this baseline?')) return;
    await api.deleteBaseline(id, apiKey);
    loadBaselines();
  };

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
          <Link href="/settings" className="text-sm font-medium text-gray-900">Settings</Link>
        </div>
      </nav>

      <main className="max-w-4xl mx-auto px-6 py-8 space-y-8">
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">API Key</h2>
          <div className="flex gap-2">
            <input
              type="text"
              placeholder="Enter API key..."
              value={apiKey}
              onChange={(e) => {
                setApiKey(e.target.value);
                if (e.target.value) localStorage.setItem('bb_api_key', e.target.value);
              }}
              className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm"
            />
          </div>
        </div>

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Projects</h2>
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              placeholder="Project name"
              value={newProjectName}
              onChange={(e) => setNewProjectName(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded text-sm"
            />
            <input
              type="text"
              placeholder="Slug"
              value={newProjectSlug}
              onChange={(e) => setNewProjectSlug(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded text-sm w-32"
            />
            <button
              onClick={handleCreateProject}
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              Create
            </button>
          </div>

          {projects.length === 0 ? (
            <p className="text-sm text-gray-500">No projects yet</p>
          ) : (
            <div className="space-y-2">
              {projects.map(p => (
                <div
                  key={p.id}
                  onClick={() => setActiveProjectId(p.id)}
                  className={`p-3 rounded cursor-pointer border ${
                    activeProjectId === p.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  <div className="font-medium text-gray-900">{p.name}</div>
                  <div className="text-xs text-gray-500">/{p.slug}</div>
                </div>
              ))}
            </div>
          )}
        </div>

        {activeProjectId && (
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">API Keys</h2>
            <div className="flex gap-2 mb-4">
              <input
                type="text"
                placeholder="Key label"
                value={newKeyLabel}
                onChange={(e) => setNewKeyLabel(e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded text-sm"
              />
              <button
                onClick={handleCreateKey}
                disabled={loading}
                className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700 disabled:opacity-50"
              >
                Create Key
              </button>
            </div>

            {keys.length === 0 ? (
              <p className="text-sm text-gray-500">No API keys</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-gray-500">
                    <th className="pb-2">Label</th>
                    <th className="pb-2">Prefix</th>
                    <th className="pb-2">Created</th>
                    <th className="pb-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {keys.map(k => (
                    <tr key={k.id} className="border-t border-gray-100">
                      <td className="py-2">{k.label}</td>
                      <td className="py-2 font-mono text-xs">{k.key_prefix}...</td>
                      <td className="py-2">{new Date(k.created_at).toLocaleDateString()}</td>
                      <td className="py-2">
                        <button
                          onClick={() => handleDeleteKey(activeProjectId, k.id)}
                          className="text-red-600 hover:text-red-800 text-xs"
                        >
                          Revoke
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Baselines</h2>
          {baselines.length === 0 ? (
            <p className="text-sm text-gray-500">No baselines saved</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-gray-500">
                  <th className="pb-2">Label</th>
                  <th className="pb-2">Trace ID</th>
                  <th className="pb-2">Created</th>
                  <th className="pb-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {baselines.map(b => (
                  <tr key={b.id} className="border-t border-gray-100">
                    <td className="py-2">{b.label}</td>
                    <td className="py-2 font-mono text-xs">{b.trace_id.slice(0, 8)}...</td>
                    <td className="py-2">{new Date(b.created_at).toLocaleDateString()}</td>
                    <td className="py-2">
                      <Link href={`/diff?baseline=${b.label}`} className="text-blue-600 hover:text-blue-800 mr-3 text-xs">
                        Compare
                      </Link>
                      <button
                        onClick={() => handleDeleteBaseline(b.id)}
                        className="text-red-600 hover:text-red-800 text-xs"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </main>
    </div>
  );
}