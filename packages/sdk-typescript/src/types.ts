export interface BlackboxConfig {
  apiKey: string;
  projectId: string;
  baseUrl?: string;
}

export interface TraceOptions {
  input?: Record<string, unknown>;
  threadId?: string;
  userId?: string;
  environment?: string;
  agentName?: string;
  runId?: string;
}

export interface GenerationOptions {
  model?: string;
}

export interface ToolOptions {
  input?: Record<string, unknown>;
}

export interface RecordOptions {
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  inputTokens?: number;
  outputTokens?: number;
  error?: string;
}

export interface SpanContext {
  traceId: string;
  projectId: string;
  name: string;
  spanKind: string;
  startTime: number;
  endTime?: number;
  spanId: string;
  attributes: Record<string, unknown>;
  model?: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  inputTokens?: number;
  outputTokens?: number;
  durationMs?: number;
  error?: string;
  record(options?: RecordOptions): void;
}

export interface TraceContext {
  traceId: string;
  projectId: string;
  name: string;
  startTime: number;
  endTime?: number;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  spans: SpanContext[];
  generation(name: string, options?: GenerationOptions): SpanContext;
  tool(name: string, options?: ToolOptions): SpanContext;
  retrieval(name: string, options?: ToolOptions): SpanContext;
  record(options?: RecordOptions): void;
  setOutput(output: Record<string, unknown>): void;
  end(): Promise<void>;
}

export interface OpenTelemetryExporterConfig {
  url: string;
  headers?: Record<string, string>;
  apiKey?: string;
  projectId?: string;
}