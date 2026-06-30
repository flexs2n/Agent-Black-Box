const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:4000';

export interface Trace {
  id: string;
  project_id: string;
  run_id?: string;
  agent_name?: string;
  status: 'success' | 'error' | 'flagged';
  thread_id?: string;
  user_id?: string;
  environment: string;
  input?: string;
  output?: string;
  error?: string;
  started_at?: string;
  ended_at?: string;
  duration_ms?: number;
  created_at: string;
}

export interface Span {
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  project_id: string;
  name: string;
  span_kind: string;
  status: string;
  started_at: string;
  ended_at: string;
  duration_ms: number;
  attributes: string;
}

export interface DiffResult {
  traceAId: string;
  traceBId: string;
  similarityScore: number;
  spanDiffs: Array<{
    status: 'unchanged' | 'changed' | 'added' | 'removed' | 'moved';
    spanAId?: string;
    spanBId?: string;
    name: string;
    spanKind: string;
    depth: number;
    attributeDiffs?: Array<{
      key: string;
      valueA?: string;
      valueB?: string;
      changeType: 'added' | 'removed' | 'changed';
    }>;
  }>;
  metricDelta: {
    durationMs: { a: number; b: number; delta: number; deltaPercent: number };
    inputTokens: { a: number; b: number; delta: number };
    outputTokens: { a: number; b: number; delta: number };
    toolCallCount: { a: number; b: number; delta: number };
    llmCallCount: { a: number; b: number; delta: number };
  };
  createdAt: string;
}

export interface Project {
  id: string;
  name: string;
  slug: string;
  created_at: string;
  settings?: string;
}

export interface APIKey {
  id: string;
  project_id: string;
  label: string;
  key_prefix: string;
  last_used_at?: string;
  created_at: string;
}

export interface Baseline {
  id: string;
  project_id: string;
  trace_id: string;
  label: string;
  notes?: string;
  created_at: string;
}

async function request<T>(path: string, options?: RequestInit, apiKey?: string): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
    ...(options?.headers as Record<string, string> | undefined),
  };
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

export const api = {
  listTraces(apiKey?: string): Promise<Trace[]> {
    return request<Trace[]>('/api/v1/traces', undefined, apiKey);
  },
  getTrace(id: string, apiKey?: string): Promise<Trace & { spans?: Span[] }> {
    return request<Trace & { spans?: Span[] }>(`/api/v1/traces/${id}`, undefined, apiKey);
  },
  getSpans(traceId: string, apiKey?: string): Promise<Span[]> {
    return request<Span[]>(`/api/v1/traces/${traceId}/spans`, undefined, apiKey);
  },
  deleteTrace(id: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/traces/${id}`, { method: 'DELETE' }, apiKey);
  },
  deleteTraces(ids: string[], apiKey?: string): Promise<void> {
    return request<void>('/api/v1/traces', {
      method: 'DELETE',
      body: JSON.stringify({ ids }),
    }, apiKey);
  },
  computeDiff(traceAId: string, traceBId: string, apiKey?: string): Promise<DiffResult> {
    return request<DiffResult>('/api/v1/diffs', {
      method: 'POST',
      body: JSON.stringify({ trace_a_id: traceAId, trace_b_id: traceBId }),
    }, apiKey);
  },
  getDiff(id: string, apiKey?: string): Promise<DiffResult & { id: string; created_at: string }> {
    return request<DiffResult & { id: string; created_at: string }>(`/api/v1/diffs/${id}`, undefined, apiKey);
  },
  listProjects(apiKey?: string): Promise<Project[]> {
    return request<Project[]>('/api/v1/projects', undefined, apiKey);
  },
  createProject(name: string, slug: string, apiKey?: string): Promise<Project> {
    return request<Project>('/api/v1/projects', {
      method: 'POST',
      body: JSON.stringify({ name, slug }),
    }, apiKey);
  },
  createApiKey(projectId: string, label: string, apiKey?: string): Promise<{ id: string; plain_key: string }> {
    return request<{ id: string; plain_key: string }>(`/api/v1/projects/${projectId}/api-keys`, {
      method: 'POST',
      body: JSON.stringify({ label }),
    }, apiKey);
  },
  listApiKeys(projectId: string, apiKey?: string): Promise<APIKey[]> {
    return request<APIKey[]>(`/api/v1/projects/${projectId}/api-keys`, undefined, apiKey);
  },
  deleteApiKey(projectId: string, keyId: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/projects/${projectId}/api-keys/${keyId}`, { method: 'DELETE' }, apiKey);
  },
  createBaseline(projectId: string, traceId: string, label: string, notes?: string, apiKey?: string): Promise<Baseline> {
    return request<Baseline>('/api/v1/baselines', {
      method: 'POST',
      body: JSON.stringify({ project_id: projectId, trace_id: traceId, label, notes }),
    }, apiKey);
  },
  listBaselines(apiKey?: string): Promise<Baseline[]> {
    return request<Baseline[]>('/api/v1/baselines', undefined, apiKey);
  },
  deleteBaseline(id: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/baselines/${id}`, { method: 'DELETE' }, apiKey);
  },
};