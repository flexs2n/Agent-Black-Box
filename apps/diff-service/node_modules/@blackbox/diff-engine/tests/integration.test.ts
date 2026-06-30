import { describe, it, expect } from 'vitest';
import { computeDiff, computeMetricDelta } from '../src/diffAlgorithm';
import { SpanNode } from '../src/types';

describe('computeMetricDelta', () => {
  it('computes correct delta values', () => {
    const a = {
      totalSpans: 10,
      llmCallCount: 3,
      toolCallCount: 3,
      totalInputTokens: 1000,
      totalOutputTokens: 500,
      totalDurationMs: 2000,
    };
    const b = {
      totalSpans: 12,
      llmCallCount: 4,
      toolCallCount: 3,
      totalInputTokens: 1100,
      totalOutputTokens: 600,
      totalDurationMs: 2200,
    };
    const delta = computeMetricDelta(a, b);
    expect(delta.durationMs.delta).toBe(200);
    expect(delta.inputTokens.delta).toBe(100);
    expect(delta.outputTokens.delta).toBe(100);
    expect(delta.toolCallCount.delta).toBe(0);
    expect(delta.llmCallCount.delta).toBe(1);
  });

  it('handles zero division in percent', () => {
    const a = {
      totalSpans: 0,
      llmCallCount: 0,
      toolCallCount: 0,
      totalInputTokens: 0,
      totalOutputTokens: 0,
      totalDurationMs: 0,
    };
    const b = {
      totalSpans: 5,
      llmCallCount: 2,
      toolCallCount: 2,
      totalInputTokens: 400,
      totalOutputTokens: 200,
      totalDurationMs: 1000,
    };
    const delta = computeMetricDelta(a, b);
    expect(delta.durationMs.delta).toBe(1000);
    expect(delta.durationMs.deltaPercent).toBe(0);
  });
});

describe('computeDiff', () => {
  const makeSpan = (spanId: string, name: string, spanKind: string, depth: number = 0): SpanNode => ({
    spanId,
    name,
    spanKind,
    attributes: {},
    events: [],
    startTime: 0,
    endTime: 100,
    depth,
    children: [],
  });

  it('detects unchanged spans', () => {
    const a = [makeSpan('A1', 'gen_ai.chat', 'generation')];
    const b = [makeSpan('B1', 'gen_ai.chat', 'generation')];
    const result = computeDiff(a, b, { totalSpans: 1, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 10, totalOutputTokens: 5, totalDurationMs: 100 }, { totalSpans: 1, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 10, totalOutputTokens: 5, totalDurationMs: 100 });
    expect(result.spanDiffs[0].status).toBe('unchanged');
    expect(result.similarityScore).toBe(100);
  });

  it('detects changed spans', () => {
    const aSpans = [makeSpan('A1', 'gen_ai.chat', 'generation')];
    aSpans[0]!.attributes = { 'gen_ai.prompt': 'Hello' } as any;
    const bSpans = [makeSpan('B1', 'gen_ai.chat', 'generation')];
    bSpans[0]!.attributes = { 'gen_ai.prompt': 'Hi' } as any;
    const result = computeDiff(aSpans, bSpans, { totalSpans: 1, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 5, totalOutputTokens: 2, totalDurationMs: 80 }, { totalSpans: 1, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 5, totalOutputTokens: 2, totalDurationMs: 90 });
    expect(result.spanDiffs[0].status).toBe('changed');
    expect(result.spanDiffs[0]?.attributeDiffs).toBeDefined();
    expect(result.spanDiffs[0]!.attributeDiffs!.length).toBe(1);
  });

  it('detects moved spans', () => {
    const a = [
      makeSpan('A1', 'root', 'root'),
      makeSpan('A2', 'tool.x', 'tool', 1),
      makeSpan('A3', 'gen_ai.chat', 'generation', 1),
    ];
    const b = [
      makeSpan('B1', 'root', 'root'),
      makeSpan('B3', 'gen_ai.chat', 'generation', 1),
      makeSpan('B2', 'tool.x', 'tool', 1),
    ];
    const result = computeDiff(a, b, { totalSpans: 3, llmCallCount: 1, toolCallCount: 1, totalInputTokens: 20, totalOutputTokens: 10, totalDurationMs: 300 }, { totalSpans: 3, llmCallCount: 1, toolCallCount: 1, totalInputTokens: 20, totalOutputTokens: 10, totalDurationMs: 300 });
    const moved = result.spanDiffs.filter(d => d.status === 'moved');
    expect(moved.length).toBeGreaterThanOrEqual(1);
  });

  it('returns correct metric delta', () => {
    const a = [makeSpan('A1', 'root', 'root')];
    const b = [makeSpan('B1', 'root', 'root')];
    const result = computeDiff(a, b, { totalSpans: 1, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 500, totalOutputTokens: 200, totalDurationMs: 1500 }, { totalSpans: 1, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 600, totalOutputTokens: 300, totalDurationMs: 1800 });
    expect(result.metricDelta.durationMs.delta).toBe(300);
    expect(result.metricDelta.inputTokens.delta).toBe(100);
    expect(result.metricDelta.outputTokens.delta).toBe(100);
  });

  it('returns createdAt timestamp in ISO format', () => {
    const a = [makeSpan('A1', 'root', 'root')];
    const b = [makeSpan('B1', 'root', 'root')];
    const result = computeDiff(a, b, { totalSpans: 1, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 100 }, { totalSpans: 1, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 100 });
    expect(new Date(result.createdAt).toISOString()).toBe(result.createdAt);
  });
});