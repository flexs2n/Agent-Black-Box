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

export interface WordDiffChunk {
  type: 'added' | 'removed' | 'unchanged';
  text: string;
}

export interface JsonDiffNode {
  key: string;
  type: 'added' | 'removed' | 'changed' | 'unchanged' | 'nested';
  valueA?: unknown;
  valueB?: unknown;
  children?: JsonDiffNode[];
}

export interface ContentDiff {
  type: 'prompt' | 'tool_args' | 'tool_output';
  wordDiff?: WordDiffChunk[];
  jsonDiff?: JsonDiffNode[];
}

export interface SpanDiff {
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
  contentDiff?: ContentDiff;
}

export interface DiffResult {
  traceAId: string;
  traceBId: string;
  similarityScore: number;
  spanDiffs: SpanDiff[];
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

export interface Issue {
  id: string;
  project_id: string;
  fingerprint: string;
  title: string;
  evaluator: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: 'open' | 'acknowledged' | 'resolved' | 'dismissed';
  first_seen_at: string;
  last_seen_at: string;
  occurrence_count: number;
  root_cause?: string;
  suggested_fix?: string;
  created_at: string;
  updated_at: string;
}

export interface IssueOccurrence {
  id: string;
  issue_id: string;
  trace_id: string;
  evidence: string;
  created_at: string;
}

export interface Metric {
  id: string;
  project_id: string;
  name: string;
  aggregation: string;
  filter_json: string;
  window_secs: number;
  created_at: string;
  updated_at: string;
}

export interface MetricWithSparkline extends Metric {
  current_value: number;
  sparkline: number[];
}

export interface ThreadSummary {
  thread_id: string;
  trace_count: number;
  first_seen_at: string;
  last_seen_at: string;
  last_agent_name?: string;
  last_status: string;
}

export interface PresetMetric {
  slug: string;
  name: string;
  value: number;
  format: string;
  sparkline: number[];
}

export interface MetricsResponse {
  preset: PresetMetric[];
  custom: MetricWithSparkline[];
}

export interface DashboardStats {
  total_traces: number;
  success_rate: number;
  p95_latency_ms: number;
  open_issues: number;
  active_incidents: number;
  total_llm_calls: number;
  total_tool_calls: number;
  total_input_tokens: number;
  total_output_tokens: number;
}

export interface TraceByHour {
  hour: string;
  count: number;
  success_count: number;
  error_count: number;
  avg_duration: number;
}

export interface DashboardResponse {
  stats: DashboardStats;
  traces_by_hour: TraceByHour[];
  open_issues: Issue[];
}

export type IssueStatus = 'open' | 'acknowledged' | 'resolved' | 'dismissed';

export interface Monitor {
  id: string;
  metric_id: string;
  project_id: string;
  condition: 'above' | 'below';
  threshold: number;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: 'ok' | 'alerting' | 'resolved';
  last_fired_at?: string;
  notify_json: string;
  created_at: string;
  updated_at: string;
}

export interface MonitorCreate {
  metric_id: string;
  condition: 'above' | 'below';
  threshold: number;
  severity?: 'critical' | 'high' | 'medium' | 'low';
  notify_json?: string;
}

export interface MonitorUpdate {
  condition?: 'above' | 'below';
  threshold?: number;
  severity?: 'critical' | 'high' | 'medium' | 'low';
  status?: 'ok' | 'alerting' | 'resolved';
  notify_json?: string;
}

export interface Incident {
  id: string;
  monitor_id: string;
  project_id: string;
  status: 'unresolved' | 'analyzed' | 'resolved' | 'dismissed';
  root_cause?: string;
  affected_trace_count: number;
  created_at: string;
  resolved_at?: string;
}

export type IncidentStatus = 'unresolved' | 'analyzed' | 'resolved' | 'dismissed';

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
  listThreads(apiKey?: string): Promise<ThreadSummary[]> {
    return request<ThreadSummary[]>('/api/v1/threads', undefined, apiKey);
  },
  getTrace(id: string, apiKey?: string): Promise<Trace & { spans?: Span[] }> {
    return request<Trace & { spans?: Span[] }>(`/api/v1/traces/${id}`, undefined, apiKey);
  },
  getSpans(traceId: string, apiKey?: string): Promise<Span[]> {
    return request<Span[]>(`/api/v1/traces/${traceId}/spans`, undefined, apiKey);
  },
  searchTraces(query: string, filters?: Record<string,string>, apiKey?: string): Promise<Trace[]> {
    return request<Trace[]>('/api/v1/traces/search', {
      method: 'POST',
      body: JSON.stringify({ mode: 'structural', query: '', filters: filters || {} }),
    }, apiKey);
  },
  semanticSearch(query: string, apiKey?: string): Promise<(Trace & { similarity: number })[]> {
    return request<(Trace & { similarity: number })[]>('/api/v1/traces/search', {
      method: 'POST',
      body: JSON.stringify({ mode: 'semantic', query, filters: {} }),
    }, apiKey);
  },
  deleteTrace(id: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/traces/${id}`, { method: 'DELETE' }, apiKey);
  },
  deleteTraces(ids: string[], apiKey?: string): Promise<void> {
    return request<void>(
      '/api/v1/traces',
      {
        method: 'DELETE',
        body: JSON.stringify({ ids }),
      },
      apiKey
    );
  },
  computeBatchDiff(referenceTraceId: string, compareTraceIds: string[], apiKey?: string): Promise<unknown> {
    return request<unknown>(
      '/api/v1/diffs/batch',
      {
        method: 'POST',
        body: JSON.stringify({ reference_trace_id: referenceTraceId, compare_trace_ids: compareTraceIds }),
      },
      apiKey
    );
  },
  computeDiff(traceAId: string, traceBId: string, apiKey?: string): Promise<DiffResult> {
    return request<DiffResult>(
      '/api/v1/diffs',
      {
        method: 'POST',
        body: JSON.stringify({ trace_a_id: traceAId, trace_b_id: traceBId }),
      },
      apiKey
    );
  },
  getDiff(id: string, apiKey?: string): Promise<DiffResult & { id: string; created_at: string }> {
    return request<DiffResult & { id: string; created_at: string }>(
      `/api/v1/diffs/${id}`,
      undefined,
      apiKey
    );
  },
  listProjects(apiKey?: string): Promise<Project[]> {
    return request<Project[]>('/api/v1/projects', undefined, apiKey);
  },
  createProject(name: string, slug: string, apiKey?: string): Promise<Project> {
    return request<Project>(
      '/api/v1/projects',
      {
        method: 'POST',
        body: JSON.stringify({ name, slug }),
      },
      apiKey
    );
  },
  createApiKey(
    projectId: string,
    label: string,
    apiKey?: string
  ): Promise<{ id: string; plain_key: string }> {
    return request<{ id: string; plain_key: string }>(
      `/api/v1/projects/${projectId}/api-keys`,
      {
        method: 'POST',
        body: JSON.stringify({ label }),
      },
      apiKey
    );
  },
  listApiKeys(projectId: string, apiKey?: string): Promise<APIKey[]> {
    return request<APIKey[]>(`/api/v1/projects/${projectId}/api-keys`, undefined, apiKey);
  },
  deleteApiKey(projectId: string, keyId: string, apiKey?: string): Promise<void> {
    return request<void>(
      `/api/v1/projects/${projectId}/api-keys/${keyId}`,
      { method: 'DELETE' },
      apiKey
    );
  },
  createBaseline(
    projectId: string,
    traceId: string,
    label: string,
    notes?: string,
    apiKey?: string
  ): Promise<Baseline> {
    return request<Baseline>(
      '/api/v1/baselines',
      {
        method: 'POST',
        body: JSON.stringify({ project_id: projectId, trace_id: traceId, label, notes }),
      },
      apiKey
    );
  },
  listBaselines(apiKey?: string): Promise<Baseline[]> {
    return request<Baseline[]>('/api/v1/baselines', undefined, apiKey);
  },
  deleteBaseline(id: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/baselines/${id}`, { method: 'DELETE' }, apiKey);
  },
  getDashboard(apiKey?: string): Promise<DashboardResponse> {
    return request<DashboardResponse>('/api/v1/dashboard', undefined, apiKey);
  },
  listIssues(apiKey?: string): Promise<Issue[]> {
    return request<Issue[]>('/api/v1/issues', undefined, apiKey);
  },
  getIssue(id: string, apiKey?: string): Promise<Issue & { occurrences?: IssueOccurrence[] }> {
    return request<Issue & { occurrences?: IssueOccurrence[] }>(
      `/api/v1/issues/${id}`,
      undefined,
      apiKey
    );
  },
  updateIssueStatus(id: string, status: IssueStatus, apiKey?: string): Promise<Issue> {
    return request<Issue>(
      `/api/v1/issues/${id}/status`,
      {
        method: 'PATCH',
        body: JSON.stringify({ status }),
      },
      apiKey
    );
  },
  listMetrics(apiKey?: string): Promise<MetricsResponse> {
    return request<MetricsResponse>('/api/v1/metrics', undefined, apiKey);
  },

  // Monitors
  listMonitors(apiKey?: string): Promise<Monitor[]> {
    return request<Monitor[]>('/api/v1/monitors', undefined, apiKey);
  },
  createMonitor(data: MonitorCreate, apiKey?: string): Promise<Monitor> {
    return request<Monitor>(
      '/api/v1/monitors',
      { method: 'POST', body: JSON.stringify(data) },
      apiKey
    );
  },
  updateMonitor(id: string, data: MonitorUpdate, apiKey?: string): Promise<Monitor> {
    return request<Monitor>(
      `/api/v1/monitors/${id}`,
      { method: 'PUT', body: JSON.stringify(data) },
      apiKey
    );
  },
  deleteMonitor(id: string, apiKey?: string): Promise<void> {
    return request<void>(`/api/v1/monitors/${id}`, { method: 'DELETE' }, apiKey);
  },
  monitorHistory(id: string, apiKey?: string): Promise<Incident[]> {
    return request<Incident[]>(`/api/v1/monitors/${id}/history`, undefined, apiKey);
  },

  // Incidents
  listIncidents(apiKey?: string): Promise<Incident[]> {
    return request<Incident[]>('/api/v1/incidents', undefined, apiKey);
  },
  getIncident(id: string, apiKey?: string): Promise<Incident> {
    return request<Incident>(`/api/v1/incidents/${id}`, undefined, apiKey);
  },
  updateIncidentStatus(id: string, status: IncidentStatus, apiKey?: string): Promise<Incident> {
    return request<Incident>(
      `/api/v1/incidents/${id}/status`,
      { method: 'PATCH', body: JSON.stringify({ status }) },
      apiKey
    );
  },
};
