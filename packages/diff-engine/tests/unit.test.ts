import { describe, it, expect } from 'vitest';
import { computeDiff } from './src/diffAlgorithm';

describe('diff-engine', () => {
  const makeStats = (spawn: number) => ({
    totalSpans: spawn,
    llmCallCount: Math.floor(spawn / 2),
    toolCallCount: Math.floor(spawn / 2),
    totalInputTokens: spawn * 100,
    totalOutputTokens: spawn * 50,
    totalDurationMs: spawn * 200,
  });

  it('returns 100% similarity for identical traces', () => {
    const spans = [
      { spanId: 'a1', name: 'root', spanKind: 'root', attributes: {}, startTime: 0, endTime: 100, depth: 0, children: [] },
    ] as any;
    const result = computeDiff(spans, spans, makeStats(1), makeStats(1));
    expect(result.similarityScore).toBeGreaterThanOrEqual(99);
    expect(result.spanDiffs.every(d => d.status === 'unchanged')).toBe(true);
  });

  it('detects added spans', () => {
    const shared = { spanId: 'a1', name: 'root', spanKind: 'root', attributes: {}, startTime: 0, endTime: 100, depth: 0, children: [] } as any;
    const a = [shared];
    const b = [...a, { spanId: 'b1', name: 'extra', spanKind: 'tool', attributes: {}, startTime: 50, endTime: 150, depth: 1, children: [] } as any];
    const result = computeDiff(a, b, makeStats(1), makeStats(2));
    expect(result.spanDiffs.some(d => d.status === 'added')).toBe(true);
    expect(result.similarityScore).toBeLessThan(100);
  });

  it('detects removed spans', () => {
    const shared = { spanId: 'a1', name: 'root', spanKind: 'root', attributes: {}, startTime: 0, endTime: 100, depth: 0, children: [] } as any;
    const a = [shared, { spanId: 'a2', name: 'extra', spanKind: 'tool', attributes: {}, startTime: 50, endTime: 150, depth: 1, children: [] } as any];
    const b = [shared];
    const result = computeDiff(a, b, makeStats(2), makeStats(1));
    expect(result.spanDiffs.some(d => d.status === 'removed')).toBe(true);
  });
});