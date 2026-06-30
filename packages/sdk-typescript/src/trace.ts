import { v4 as uuidv4 } from "uuid";
import type {
  BlackboxConfig,
  TraceContext,
  TraceOptions,
  GenerationOptions,
  RecordOptions,
  SpanContext,
  ToolOptions,
} from "./types.js";

function getEnv(key: string, fallback?: string): string {
  const value = process.env[key];
  if (value === undefined || value === "") {
    if (fallback !== undefined) return fallback;
    throw new Error(`Missing required env var: ${key}`);
  }
  return value;
}

export function resolveConfig(
  explicit?: Partial<BlackboxConfig>
): BlackboxConfig {
  return {
    apiKey: explicit?.apiKey ?? getEnv("BLACKBOX_API_KEY"),
    projectId: explicit?.projectId ?? getEnv("BLACKBOX_PROJECT_ID"),
    baseUrl: explicit?.baseUrl ?? getEnv("BLACKBOX_BASE_URL", "http://localhost:4000"),
  };
}

function encodeRecordOptions(span: SpanContext, options?: RecordOptions): void {
  if (!options) return;
  if (options.input !== undefined) {
    span.input = options.input;
    span.attributes["gen_ai.prompt"] = JSON.stringify(options.input);
  }
  if (options.output !== undefined) {
    span.output = options.output;
    span.attributes["gen_ai.completion"] = JSON.stringify(options.output);
  }
  if (options.inputTokens !== undefined) {
    span.inputTokens = options.inputTokens;
    span.attributes["gen_ai.usage.input_tokens"] = options.inputTokens;
  }
  if (options.outputTokens !== undefined) {
    span.outputTokens = options.outputTokens;
    span.attributes["gen_ai.usage.output_tokens"] = options.outputTokens;
  }
  if (options.error !== undefined) {
    span.error = options.error;
    span.attributes["exception.message"] = options.error;
  }
}

export function createTraceContext(
  name: string,
  config: BlackboxConfig,
  options?: TraceOptions
): TraceContext {
  const traceId = uuidv4();
  const startTime = Date.now();
  const ctx: TraceContext = {
    traceId,
    projectId: config.projectId,
    name,
    startTime,
    input: options?.input,
    spans: [],
    generation(name: string, options?: GenerationOptions): SpanContext {
      const span = createSpan({
        traceId,
        projectId: config.projectId,
        name,
        spanKind: "generation",
        startTime: Date.now(),
        model: options?.model,
      });
      ctx.spans.push(span);
      return span;
    },
    tool(name: string, options?: ToolOptions): SpanContext {
      const span = createSpan({
        traceId,
        projectId: config.projectId,
        name,
        spanKind: "tool",
        startTime: Date.now(),
        input: options?.input,
      });
      if (options?.input) {
        span.attributes["tool.input"] = JSON.stringify(options.input);
      }
      ctx.spans.push(span);
      return span;
    },
    retrieval(name: string, options?: ToolOptions): SpanContext {
      const span = createSpan({
        traceId,
        projectId: config.projectId,
        name,
        spanKind: "retrieval",
        startTime: Date.now(),
        input: options?.input,
      });
      ctx.spans.push(span);
      return span;
    },
    record(options?: RecordOptions): void {
      const lastSpan = ctx.spans[ctx.spans.length - 1];
      if (!lastSpan) return;
      lastSpan.endTime = Date.now();
      lastSpan.durationMs = lastSpan.endTime - lastSpan.startTime;
      encodeRecordOptions(lastSpan, options);
    },
    setOutput(output: Record<string, unknown>): void {
      ctx.output = output;
    },
    async end(): Promise<void> {
      ctx.endTime = Date.now();
      await exportTrace(ctx, config);
    },
  };

  if (options?.threadId) {
    ctx.input = { ...ctx.input, blackbox_thread_id: options.threadId } as Record<
      string,
      unknown
    >;
  }

  return ctx;
}

function createSpan(init: {
  traceId: string;
  projectId: string;
  name: string;
  spanKind: string;
  startTime: number;
  model?: string;
  input?: Record<string, unknown>;
}): SpanContext {
  return {
    traceId: init.traceId,
    projectId: init.projectId,
    name: init.name,
    spanKind: init.spanKind,
    startTime: init.startTime,
    spanId: uuidv4(),
    attributes: {},
    model: init.model,
    input: init.input,
  };
}

async function exportTrace(trace: TraceContext, config: BlackboxConfig): Promise<void> {
  const spanEvents = trace.spans.map((span) => {
    const attrs: Record<string, unknown> = {
      blackbox: {
        span_kind: span.spanKind,
        trace_id: trace.traceId,
        span_id: span.spanId,
      },
    };

    for (const [key, value] of Object.entries(span.attributes)) {
      attrs[key] = value;
    }

    if (span.model) {
      attrs["gen_ai.request.model"] = span.model;
    }
    if (span.input) {
      attrs["gen_ai.prompt"] = JSON.stringify(span.input);
    }
    if (span.output) {
      attrs["gen_ai.completion"] = JSON.stringify(span.output);
    }
    if (span.inputTokens !== undefined) {
      attrs["gen_ai.usage.input_tokens"] = span.inputTokens;
    }
    if (span.outputTokens !== undefined) {
      attrs["gen_ai.usage.output_tokens"] = span.outputTokens;
    }
    if (span.error) {
      attrs["exception.message"] = span.error;
    }

    const durationMs = span.endTime
      ? span.endTime - span.startTime
      : 50;

    return {
      traceId: trace.traceId,
      spanId: span.spanId,
      parentSpanId: span.spanKind === "generation" ? trace.traceId : undefined,
      name: span.name,
      kind: span.spanKind === "generation" ? 1 : span.spanKind === "tool" ? 4 : 13,
      startTimeUnixNano: BigInt(span.startTime * 1_000_000),
      endTimeUnixNano: BigInt((span.endTime ?? span.startTime + durationMs) * 1_000_000),
      attributes: Object.entries(attrs).map(([key, value]) => ({
        key,
        value: { stringValue: JSON.stringify(value) },
      })),
      status: { code: span.error ? 2 : 1 },
    };
  });

  const payload = {
    resourceSpans: [
      {
        resource: {
          attributes: [
            { key: "service.name", value: { stringValue: trace.name } },
            { key: "blackbox.project_id", value: { stringValue: trace.projectId } },
          ],
        },
        scopeSpans: [
          {
            scope: { name: "blackbox" },
            spans: spanEvents as any[],
          },
        ],
      },
    ],
  };

  const response = await fetch(`${config.baseUrl}/otel/v1/traces`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${config.apiKey}`,
      "X-Blackbox-Project-ID": trace.projectId,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(
      `OTLP export failed: ${response.status} ${response.statusText}: ${body}`
    );
  }
}

export { createTraceContext as trace };