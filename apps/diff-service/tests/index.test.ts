import { describe, it, expect } from 'vitest';
import app from './src/index';

describe('diff-service', () => {
  it('responds to healthz', async () => {
    const res = await app.request('/healthz');
    expect(res.status).toBe(200);
    const json = (await res.json()) as any;
    expect(json.status).toBe('ok');
  });

  it('returns a diff result for identical traces', async () => {
    const payload = {
      traceA: {
        spanId: 'a1',
        name: 'root',
        spanKind: 'root',
        attributes: {},
        children: [
          { spanId: 'a2', name: 'gen_ai.chat', spanKind: 'generation', attributes: { 'gen_ai.prompt': 'Hello' }, children: [] },
        ],
      },
      traceB: {
        spanId: 'b1',
        name: 'root',
        spanKind: 'root',
        attributes: {},
        children: [
          { spanId: 'b2', name: 'gen_ai.chat', spanKind: 'generation', attributes: { 'gen_ai.prompt': 'Hello' }, children: [] },
        ],
      },
      statsA: { totalSpans: 2, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 10, totalOutputTokens: 5, totalDurationMs: 100 },
      statsB: { totalSpans: 2, llmCallCount: 1, toolCallCount: 0, totalInputTokens: 10, totalOutputTokens: 5, totalDurationMs: 100 },
    };

    const res = await app.request('/internal/diff', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    expect(res.status).toBe(200);
    const json = (await res.json()) as any;
    expect(json.similarityScore).toBe(100);
    expect(json.spanDiffs).toHaveLength(2);
    expect(json.metricDelta).toBeDefined();
  });

  it('detects added and changed spans', async () => {
    const payload = {
      traceA: {
        spanId: 'a1',
        name: 'root',
        spanKind: 'root',
        attributes: {},
        children: [],
      },
      traceB: {
        spanId: 'b1',
        name: 'root',
        spanKind: 'root',
        attributes: {},
        children: [
          { spanId: 'b2', name: 'tool.new', spanKind: 'tool', attributes: {}, children: [] },
        ],
      },
      statsA: { totalSpans: 1, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 50 },
      statsB: { totalSpans: 2, llmCallCount: 0, toolCallCount: 1, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 80 },
    };

    const res = await app.request('/internal/diff', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    expect(res.status).toBe(200);
    const json = (await res.json()) as any;
    expect(json.spanDiffs.some((d: any) => d.status === 'added')).toBe(true);
    expect(json.similarityScore).toBeLessThan(100);
  });

  it('returns 400 on invalid request', async () => {
    const res = await app.request('/internal/diff', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ invalid: true }),
    });
    expect(res.status).toBe(400);
  });
});