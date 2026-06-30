import { Hono } from 'hono';
import { z } from 'zod';
import { computeDiff, computeMetricDelta, normalizeTree } from '@blackbox/diff-engine';

type BEnv = {
  Variables: {
    STATS_A?: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number };
    STATS_B?: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number };
  };
};

const DiffRequestSchema = z.object({
  traceA: z.object({
    spanId: z.string().optional(),
    name: z.string().optional(),
    spanKind: z.string().optional(),
    attributes: z.record(z.string()).optional(),
    startTime: z.number().optional(),
    endTime: z.number().optional(),
    children: z.array(z.any()).optional(),
  }),
  traceB: z.object({
    spanId: z.string().optional(),
    name: z.string().optional(),
    spanKind: z.string().optional(),
    attributes: z.record(z.string()).optional(),
    startTime: z.number().optional(),
    endTime: z.number().optional(),
    children: z.array(z.any()).optional(),
  }),
  statsA: z.object({
    totalSpans: z.number(),
    llmCallCount: z.number(),
    toolCallCount: z.number(),
    totalInputTokens: z.number(),
    totalOutputTokens: z.number(),
    totalDurationMs: z.number(),
  }).optional(),
  statsB: z.object({
    totalSpans: z.number(),
    llmCallCount: z.number(),
    toolCallCount: z.number(),
    totalInputTokens: z.number(),
    totalOutputTokens: z.number(),
    totalDurationMs: z.number(),
  }).optional(),
});

const app = new Hono<{ Variables: BEnv['Variables'] }>();

app.get('/healthz', (c) => c.json({ status: 'ok' }));

app.post('/internal/diff', async (c) => {
  try {
    const body = await c.req.json();
    const parsed = DiffRequestSchema.parse(body);

    const rawA = parsed.traceA as any;
    const rawB = parsed.traceB as any;

    const makeNorm = (node: any, depth = 0, parentId?: string) => ({
      spanId: node.spanId ?? 'unknown',
      parentSpanId: parentId,
      name: node.name ?? 'unknown',
      spanKind: node.spanKind ?? node.name ?? 'app',
      attributes: node.attributes ?? {},
      startTime: node.startTime ?? 0,
      endTime: node.endTime ?? 0,
      depth,
      children: (node.children ?? []).map((child: any) => makeNorm(child, depth + 1, node.spanId)),
    });

    const aTree = makeNorm(rawA);
    const bTree = makeNorm(rawB);

    const aSpans = [aTree, ...aTree.children.map((c: any) => c)];
    const bSpans = [bTree, ...bTree.children.map((c: any) => c)];

    const statsA = parsed.statsA ?? { totalSpans: aSpans.length, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 0 };
    const statsB = parsed.statsB ?? { totalSpans: bSpans.length, llmCallCount: 0, toolCallCount: 0, totalInputTokens: 0, totalOutputTokens: 0, totalDurationMs: 0 };

    const result = computeDiff(aSpans, bSpans, statsA, statsB);
    return c.json(result);
  } catch (err: any) {
    console.error('/internal/diff failed:', err);
    return c.json({ error: 'Invalid request', message: err.message }, 400);
  }
});

const port = Number(process.env.PORT ?? 5001);
console.log(`diff-service listening on :${port}`);
export default app;