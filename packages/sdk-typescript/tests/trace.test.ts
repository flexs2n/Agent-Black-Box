import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("uuid", () => ({
  v4: () => "test-uuid-1234",
}));

const originalEnv = process.env;

beforeEach(() => {
  process.env = { ...originalEnv };
  process.env.BLACKBOX_API_KEY = "test-key";
  process.env.BLACKBOX_PROJECT_ID = "test-project";
  process.env.BLACKBOX_BASE_URL = "http://localhost:4000";
});

describe("SDK typescript", () => {
  it("resolves config", async () => {
    const { resolveConfig } = await import("./src/trace");
    const config = resolveConfig();
    expect(config.apiKey).toBe("test-key");
    expect(config.projectId).toBe("test-project");
    expect(config.baseUrl).toBe("http://localhost:4000");
  });

  it("creates a trace with generation span", async () => {
    const exported: unknown[] = [];
    const originalFetch = globalThis.fetch;
    globalThis.fetch = async () =>
      new Response(JSON.stringify({ ok: true }), { status: 200 });

    try {
      const { createTraceContext } = await import("./src/trace");
      const config = {
        apiKey: "test-key",
        projectId: "test-project",
        baseUrl: "http://localhost:4000",
      };
      const ctx = createTraceContext("my-trace", config, {
        input: { message: "hi" },
      });

      expect(ctx.traceId).toBe("test-uuid-1234");
      expect(ctx.name).toBe("my-trace");
      expect(ctx.input).toEqual({ message: "hi" });

      const gen = ctx.generation("greet", { model: "gpt-4o" });
      gen.record({
        input: [{ role: "user", content: "Say hi" }],
        output: "Hello!",
        inputTokens: 5,
        outputTokens: 8,
      });

      expect(gen.output).toEqual("Hello!");
      expect(gen.outputTokens).toBe(8);
      expect(gen.durationMs).toBeGreaterThanOrEqual(0);
      expect(ctx.spans).toHaveLength(1);

      await ctx.end();
    } finally {
      globalThis.fetch = originalFetch;
    }
  });

  it("adds a trace span of kind tool", async () => {
    const { createTraceContext } = await import("./src/trace");
    const config = {
      apiKey: "test-key",
      projectId: "test-project",
      baseUrl: "http://localhost:4000",
    };
    const ctx = createTraceContext("tool-trace", config);
    const tool = ctx.tool("search_docs", { input: { query: "password reset" } });
    tool.record({ output: { result: "ok" } });
    expect(tool.spanKind).toBe("tool");
    expect(tool.input).toEqual({ query: "password reset" });
    expect(tool.output).toEqual({ result: "ok" });
    expect(ctx.spans).toHaveLength(1);
  });

  it("sets trace output", async () => {
    const { createTraceContext } = await import("./src/trace");
    const config = {
      apiKey: "test-key",
      projectId: "test-project",
      baseUrl: "http://localhost:4000",
    };
    const ctx = createTraceContext("out-trace", config);
    ctx.setOutput({ reply: "done" });
    expect(ctx.output).toEqual({ reply: "done" });
  });
});