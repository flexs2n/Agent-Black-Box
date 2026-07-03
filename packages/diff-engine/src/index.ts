export { normalizeTree, normalizeSpan, flattenNormalized } from "./normalize.js";
export { computeDiff, computeMetricDelta, computeBatchDiff } from "./diffAlgorithm.js";
export type {
  DiffResult,
  SpanDiff,
  AttributeDiff,
  ContentDiff,
  WordDiffChunk,
  JsonDiffNode,
  MetricDelta,
  SpanNode,
  NormalizedSpan,
  TraceStats,
} from "./types.js";
export type { BatchDiffResult } from "./diffAlgorithm.js";