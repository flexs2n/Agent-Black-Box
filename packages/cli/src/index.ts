#!/usr/bin/env node

const API_URL = process.env.BLACKBOX_BASE_URL || "http://localhost:4000";

export interface Args {
  _: string[];
  trace?: string;
  baseline?: string;
  "min-similarity"?: number;
  output?: string;
  limit?: number;
}

export function parseArgs(argv: string[]): Args {
  const args: Args = { _: [] };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith("--")) {
      const key = a.slice(2);
      const val = argv[++i];
      if (key === "min-similarity") {
        args["min-similarity"] = Number(val);
      } else if (key === "limit") {
        args.limit = Number(val);
      } else {
        (args as any)[key] = val;
      }
    } else {
      args._.push(a);
    }
  }
  return args;
}

function apiKey(): string {
  const k = process.env.BLACKBOX_API_KEY;
  if (!k) {
    console.error("BLACKBOX_API_KEY environment variable required");
    process.exit(1);
  }
  return k;
}

async function request(path: string, init?: RequestInit): Promise<any> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${apiKey()}`,
      ...init?.headers,
    },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

async function listTraces(limit: number) {
  const traces = await request(`/api/v1/traces?page=1`);
  const items = Array.isArray(traces) ? traces.slice(0, limit) : [];
  for (const t of items) {
    console.log(`${t.id}\t${t.agent_name || "?"}\t${t.status}\t${t.duration_ms || 0}ms`);
  }
}

async function computeBatchDiff(referenceTraceId: string, compareIds: string[], format: string) {
  const result = await request("/api/v1/diffs/batch", {
    method: "POST",
    body: JSON.stringify({ reference_trace_id: referenceTraceId, compare_trace_ids: compareIds }),
  });
  if (format === "json") {
    console.log(JSON.stringify(result, null, 2));
  } else {
    console.log(`Reference: ${referenceTraceId}`);
    console.log(`Comparisons: ${result.comparisons?.length || 0}`);
    console.log(`\nVariance Report:`);
    const v = result.variance || {};
    for (const [key, stats] of Object.entries(v)) {
      console.log(`  ${key}: mean=${(stats as any).mean}, stdDev=${(stats as any).stdDev}, range=[${(stats as any).min}, ${(stats as any).max}]`);
    }
    if (result.outliers?.length > 0) {
      console.log(`\nOutliers:`);
      for (const o of result.outliers) {
        console.log(`  ${o.traceId}: similarity=${o.similarityScore}, z-score=${o.zScore}`);
      }
    }
  }
}

async function computeDiff(traceA: string, traceB: string, format: string) {
  const result = await request("/api/v1/diffs", {
    method: "POST",
    body: JSON.stringify({ trace_a_id: traceA, trace_b_id: traceB }),
  });
  if (format === "json") {
    console.log(JSON.stringify(result, null, 2));
  } else {
    console.log(`Similarity: ${result.similarityScore}%`);
    for (const d of result.spanDiffs || []) {
      const status = d.status.toUpperCase().padEnd(10);
      console.log(`  ${status} ${d.name} (${d.spanKind})`);
    }
  }
}

async function assertSimilarity(baseline: string, minSimilarity: number) {
  const projectID = process.env.BLACKBOX_PROJECT_ID;
  if (!projectID) {
    console.error("BLACKBOX_PROJECT_ID environment variable required");
    process.exit(1);
  }
  const baselines = await request("/api/v1/baselines");
  const bl = baselines.find((b: any) => b.label === baseline);
  if (!bl) {
    console.error(`Baseline "${baseline}" not found`);
    process.exit(1);
  }
  const traces = await request("/api/v1/traces?page=1");
  const latest = Array.isArray(traces) && traces.length > 0 ? traces[0] : null;
  if (!latest) {
    console.error("No traces found");
    process.exit(1);
  }
  const result = await request("/api/v1/diffs", {
    method: "POST",
    body: JSON.stringify({ trace_a_id: latest.id, trace_b_id: bl.trace_id }),
  });
  const score = result.similarityScore;
  console.log(`Similarity vs "${baseline}": ${score}% (threshold: ${minSimilarity}%)`);
  if (score < minSimilarity) {
    console.error(`FAIL: similarity ${score}% is below threshold ${minSimilarity}%`);
    process.exit(1);
  }
  console.log(`PASS: similarity ${score}% meets threshold ${minSimilarity}%`);
}

export async function main() {
  const args = parseArgs(process.argv);

  if (args._.length === 0) {
    if (args.trace && args.baseline) {
      return computeDiff(args.trace, args.baseline, args.output || "text");
    }
    console.error("Usage: blackbox diff <traceA> <traceB> [--output json|text]");
    console.error("       blackbox diff --trace <id> --baseline <label>");
    console.error("       blackbox diff-batch <reference-trace-id> <compare-trace-id>... [--output json|text]");
    console.error("       blackbox assert --baseline <label> --min-similarity <N>");
    console.error("       blackbox list-traces [--limit N]");
    process.exit(1);
  }

  const cmd = args._[0];
  switch (cmd) {
    case "diff": {
      if (args._.length >= 3) {
        await computeDiff(args._[1], args._[2], args.output || "text");
      } else if (args.trace && args.baseline) {
        await computeDiff(args.trace, args.baseline, args.output || "text");
      } else {
        console.error("Usage: blackbox diff <traceA> <traceB> [--output json|text]");
        process.exit(1);
      }
      break;
    }
    case "diff-batch": {
      const refId = args._[1] || args.trace;
      const compareIds = args._.slice(2);
      if (!refId || compareIds.length === 0) {
        console.error("Usage: blackbox diff-batch <reference-trace-id> <compare-trace-id>... [--output json|text]");
        process.exit(1);
      }
      await computeBatchDiff(refId, compareIds, args.output || "text");
      break;
    }
    case "assert": {
      const bl = args.baseline;
      const ms = args["min-similarity"];
      if (!bl || !ms) {
        console.error("Usage: blackbox assert --baseline <label> --min-similarity <N>");
        process.exit(1);
      }
      await assertSimilarity(bl, ms);
      break;
    }
    case "list-traces": {
      await listTraces(args.limit || 50);
      break;
    }
    default:
      console.error(`Unknown command: ${cmd}`);
      process.exit(1);
  }
}

if (process.env.VITEST === undefined) {
  main().catch((e) => {
    console.error(e.message);
    process.exit(1);
  });
}
