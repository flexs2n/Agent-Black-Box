export { normalizeTree, normalizeSpan, flattenNormalized } from "./normalize.js";
export { computeDiff, computeMetricDelta } from "./diffAlgorithm.js";
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