export interface DiffResult {
  traceAId: string;
  traceBId: string;
  similarityScore: number;
  spanDiffs: SpanDiff[];
  metricDelta: MetricDelta;
  createdAt: string;
}

export interface SpanDiff {
  status: 'unchanged' | 'changed' | 'added' | 'removed' | 'moved';
  spanAId?: string;
  spanBId?: string;
  name: string;
  spanKind: string;
  depth: number;
  attributeDiffs?: AttributeDiff[];
  contentDiff?: ContentDiff;
}

export interface AttributeDiff {
  key: string;
  valueA?: string;
  valueB?: string;
  changeType: 'added' | 'removed' | 'changed';
}

export interface ContentDiff {
  type: 'prompt' | 'tool_args' | 'tool_output';
  wordDiff?: WordDiffChunk[];
  jsonDiff?: JsonDiffNode[];
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

export interface MetricDelta {
  durationMs: { a: number; b: number; delta: number; deltaPercent: number };
  inputTokens: { a: number; b: number; delta: number };
  outputTokens: { a: number; b: number; delta: number };
  toolCallCount: { a: number; b: number; delta: number };
  llmCallCount: { a: number; b: number; delta: number };
}

export interface SpanNode {
  spanId: string;
  parentSpanId?: string;
  name: string;
  spanKind: string;
  attributes: Record<string, string>;
  events: Array<{ name: string; timestamp: number; attributes: Record<string, string> }>;
  startTime: number;
  endTime: number;
  depth?: number;
  children: SpanNode[];
}

export interface NormalizedSpan {
  spanId: string;
  parentSpanId?: string;
  name: string;
  spanKind: string;
  attributes: Record<string, string>;
  startTime: number;
  endTime: number;
  depth?: number;
  children: NormalizedSpan[];
}

export interface TraceStats {
  totalSpans: number;
  llmCallCount: number;
  toolCallCount: number;
  totalInputTokens: number;
  totalOutputTokens: number;
  totalDurationMs: number;
}