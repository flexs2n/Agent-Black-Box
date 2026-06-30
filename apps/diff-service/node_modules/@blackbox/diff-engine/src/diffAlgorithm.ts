import type {
  NormalizedSpan,
  SpanNode,
  DiffResult,
  SpanDiff,
  AttributeDiff,
  ContentDiff,
  WordDiffChunk,
  JsonDiffNode,
  MetricDelta,
} from "./types.js";

interface SpanEntry {
  aSpanIndex?: number;
  bSpanIndex?: number;
  spanA?: SpanNode;
  spanB?: SpanNode;
  status: "unchanged" | "added" | "removed" | "changed" | "moved";
  similarQualities: number[];
}

function areSpansEqual(a: NormalizedSpan, b: NormalizedSpan): boolean {
  if (a.name !== b.name || a.spanKind !== b.spanKind) {
    return false;
  }

  const aKeys = Object.keys(a.attributes).sort();
  const bKeys = Object.keys(b.attributes).sort();
  if (aKeys.length !== bKeys.length) {
    return false;
  }
  for (let i = 0; i < aKeys.length; i++) {
    if (aKeys[i] !== bKeys[i]) {
      return false;
    }
    if (a.attributes[aKeys[i]] !== b.attributes[bKeys[i]]) {
      return false;
    }
  }

  return true;
}

function computeAttributeDiff(
  aAttributes: Record<string, string>,
  bAttributes: Record<string, string>
): AttributeDiff[] {
  const diffs: AttributeDiff[] = [];
  const allKeys = new Set([...Object.keys(aAttributes), ...Object.keys(bAttributes)]);

  for (const key of Array.from(allKeys).sort()) {
    const aVal = aAttributes[key];
    const bVal = bAttributes[key];

    if (aVal === undefined || aVal === null || aVal === "") {
      if (bVal !== undefined && bVal !== null && bVal !== "") {
        diffs.push({ key, valueB: bVal, changeType: "added" });
      }
    } else if (bVal === undefined || bVal === null || bVal === "") {
      diffs.push({ key, valueA: aVal, changeType: "removed" });
    } else if (aVal !== bVal) {
      diffs.push({ key, valueA: aVal, valueB: bVal, changeType: "changed" });
    }
  }

  return diffs;
}

function computeContentDiff(
  aSpan: NormalizedSpan,
  bSpan: NormalizedSpan
): ContentDiff | undefined {
  const aAttrs = aSpan.attributes;
  const bAttrs = bSpan.attributes;

  if (aSpan.spanKind === "generation" || aAttrs["gen_ai.prompt"] || bAttrs["gen_ai.prompt"]) {
    const wordDiff = computeWordDiff(
      aAttrs["gen_ai.prompt"] || "",
      bAttrs["gen_ai.prompt"] || ""
    );
    if (wordDiff.length > 0) {
      return { type: "prompt", wordDiff };
    }
  }

  if (aSpan.spanKind === "tool" || aAttrs["tool.input"] || bAttrs["tool.input"]) {
    const aInput = aAttrs["tool.input"] ? JSON.parse(aAttrs["tool.input"]) : undefined;
    const bInput = bAttrs["tool.input"] ? JSON.parse(bAttrs["tool.input"]) : undefined;
    const jsonDiff = computeJsonDiff(aInput, bInput);
    if (jsonDiff.length > 0) {
      return { type: "tool_args", jsonDiff };
    }
  }

  return undefined;
}

function computeWordDiff(a: string, b: string): WordDiffChunk[] {
  const chunks: WordDiffChunk[] = [];

  if (a === b) {
    return chunks;
  }

  const aWords = a.split(/(\s+)/);
  const bWords = b.split(/(\s+)/);

  const aIdx = new Map<string, number[]>();
  for (let i = 0; i < aWords.length; i++) {
    if (!aWords[i].match(/^\s+$/)) {
      if (!aIdx.has(aWords[i])) {
        aIdx.set(aWords[i], []);
      }
      aIdx.get(aWords[i])!.push(i);
    }
  }

  const matches: { a: number; b: number }[] = [];
  for (let i = 0; i < bWords.length; i++) {
    const key = bWords[i];
    if (key.match(/^\s+$/)) continue;
    const positions = aIdx.get(key);
    if (positions && positions.length > 0) {
      matches.push({ a: positions.shift()!, b: i });
    }
  }

  matches.sort((x, y) => x.b - y.b);

  const usedA = new Set<number>(matches.map((m) => m.a));
  let aPtr = 0;
  let bPtr = 0;

  for (const m of matches) {
    while (aPtr < m.a) {
      if (!usedA.has(aPtr)) {
        chunks.push({ type: "removed", text: aWords[aPtr] });
      }
      aPtr++;
    }
    while (bPtr < m.b) {
      chunks.push({ type: "added", text: bWords[bPtr] });
      bPtr++;
    }

    let matchLen = 0;
    while (
      aPtr + matchLen < aWords.length &&
      bPtr + matchLen < bWords.length &&
      aWords[aPtr + matchLen] === bWords[bPtr + matchLen]
    ) {
      matchLen++;
    }
    for (let i = 0; i < matchLen; i++) {
      chunks.push({ type: "unchanged", text: bWords[bPtr + i] });
    }
    aPtr += matchLen;
    bPtr += matchLen;
  }

  while (aPtr < aWords.length) {
    if (!usedA.has(aPtr)) {
      chunks.push({ type: "removed", text: aWords[aPtr] });
    }
    aPtr++;
  }
  while (bPtr < bWords.length) {
    chunks.push({ type: "added", text: bWords[bPtr] });
    bPtr++;
  }

  return chunks.length > 0 ? chunks : [];
}

function computeJsonDiff(a?: unknown, b?: unknown): JsonDiffNode[] {
  if (a === undefined && b === undefined) return [];
  if (a === undefined) {
    return [{ key: "", type: "added", valueB: b }];
  }
  if (b === undefined) {
    return [{ key: "", type: "removed", valueA: a }];
  }

  const keysA = new Set(a && typeof a === "object" ? Object.keys(a as Record<string, unknown>) : []);
  const keysB = new Set(b && typeof b === "object" ? Object.keys(b as Record<string, unknown>) : []);
  const nodes: JsonDiffNode[] = [];

  const allKeys = new Set([...Array.from(keysA), ...Array.from(keysB)]);
  for (const key of Array.from(allKeys).sort()) {
    const aVal = (a as Record<string, unknown>)[key];
    const bVal = (b as Record<string, unknown>)[key];

    if (aVal === undefined) {
      nodes.push({ key, type: "added", valueB: bVal });
    } else if (bVal === undefined) {
      nodes.push({ key, type: "removed", valueA: aVal });
    } else if (JSON.stringify(aVal) !== JSON.stringify(bVal)) {
      if (typeof aVal === "object" && typeof bVal === "object" && aVal !== null && bVal !== null) {
        nodes.push({
          key,
          type: "nested",
          children: computeJsonDiff(aVal, bVal),
        });
      } else {
        nodes.push({ key, type: "changed", valueA: String(aVal), valueB: String(bVal) });
      }
    } else {
      nodes.push({ key, type: "unchanged", valueA: String(aVal) });
    }
  }

  return nodes.length > 0 ? nodes : [];
}

export function computeMetricDelta(
  statsA: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number },
  statsB: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number }
): MetricDelta {
  const calc = <K extends keyof MetricDelta>(a: MetricDelta[K]["a"], b: MetricDelta[K]["b"], _k: K): MetricDelta[K] => {
    const delta = b - a;
    const deltaPercent = a !== 0 ? Math.round(((b - a) / a) * 10000) / 100 : 0;
    return { a, b, delta, deltaPercent };
  };

  return {
    durationMs: calc(statsA.totalDurationMs, statsB.totalDurationMs, "durationMs"),
    inputTokens: calc(statsA.totalInputTokens, statsB.totalInputTokens, "inputTokens"),
    outputTokens: calc(statsA.totalOutputTokens, statsB.totalOutputTokens, "outputTokens"),
    toolCallCount: calc(statsA.toolCallCount, statsB.toolCallCount, "toolCallCount"),
    llmCallCount: calc(statsA.llmCallCount, statsB.llmCallCount, "llmCallCount"),
  };
}

export function computeDiff(
  aSpans: SpanNode[],
  bSpans: SpanNode[],
  statsA: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number },
  statsB: { totalSpans: number; llmCallCount: number; toolCallCount: number; totalInputTokens: number; totalOutputTokens: number; totalDurationMs: number }
): DiffResult {
  const aNormMap = new Map<string, SpanNode>();
  const bNormMap = new Map<string, SpanNode>();

  for (const s of aSpans) aNormMap.set(s.spanId, s);
  for (const s of bSpans) bNormMap.set(s.spanId, s);

  const aEntries = new Map<string, number>();
  const bEntries = new Map<string, number>();

  for (let i = 0; i < aSpans.length; i++) {
    aEntries.set(`${aSpans[i].spanKind}:${aSpans[i].name}`, i);
  }
  for (let i = 0; i < bSpans.length; i++) {
    bEntries.set(`${bSpans[i].spanKind}:${bSpans[i].name}`, i);
  }

  const matched: { aIdx: number; bIdx: number; quality: number }[] = [];
  const usedB = new Set<number>();

  for (const [key, aIdx] of aEntries) {
    const bIdx = bEntries.get(key);
    if (bIdx !== undefined && !usedB.has(bIdx)) {
      const a = aSpans[aIdx];
      const b = bSpans[bIdx];
      const eq = areSpansEqual(a, b);
      const quality = eq ? 1 : a.name === b.name ? 0.5 : 0.1;
      matched.push({ aIdx, bIdx, quality });
      usedB.add(bIdx);
    }
  }

  matched.sort((x, y) => y.quality - x.quality);

  const pairedA = new Set(matched.map((m) => m.aIdx));
  const pairedB = new Set(matched.map((m) => m.bIdx));

  const merged: SpanEntry[] = [];

  let aPtr = 0;
  let bPtr = 0;

  while (aPtr < aSpans.length || bPtr < bSpans.length) {
    if (aPtr < aSpans.length && pairedA.has(aPtr) && !usedB.has(bPtr)) {
      const m = matched.find((ma) => ma.aIdx === aPtr && !usedB.has(ma.bIdx) && ma.quality > 0);
      if (m) {
        usedB.add(m.bIdx);
        const a = aSpans[aPtr];
        const b = bSpans[m.bIdx];
        merged.push({
          aSpanIndex: aPtr,
          bSpanIndex: m.bIdx,
          spanA: a,
          spanB: b,
          status: m.quality === 1 ? "unchanged" : "changed",
          similarQualities: [m.quality],
        });
        aPtr++;
        bPtr = m.bIdx + 1;
        continue;
      }
    }

    if (aPtr < aSpans.length && !pairedA.has(aPtr)) {
      const a = aSpans[aPtr];
      merged.push({
        aSpanIndex: aPtr,
        spanA: a,
        status: "removed",
        similarQualities: [],
      });
      aPtr++;
      continue;
    }

    if (bPtr < bSpans.length && !pairedB.has(bPtr)) {
      const b = bSpans[bPtr];
      merged.push({
        bSpanIndex: bPtr,
        spanB: b,
        status: "added",
        similarQualities: [],
      });
      bPtr++;
      continue;
    }

    if (aPtr < aSpans.length) {
      const a = aSpans[aPtr];
      const m = matched.find((ma) => ma.aIdx === aPtr);
      if (m) {
        const b = bSpans[m.bIdx];
        const eq = areSpansEqual(a, b);
        merged.push({
          aSpanIndex: aPtr,
          bSpanIndex: m.bIdx,
          spanA: a,
          spanB: b,
          status: eq ? "unchanged" : "changed",
          similarQualities: [eq ? 1 : a.name === b.name ? 0.5 : 0.1],
        });
        bPtr = m.bIdx + 1;
      } else {
        merged.push({
          aSpanIndex: aPtr,
          spanA: a,
          status: "removed",
          similarQualities: [],
        });
      }
      aPtr++;
      continue;
    }

    if (bPtr < bSpans.length) {
      merged.push({
        bSpanIndex: bPtr,
        spanB: bSpans[bPtr],
        status: "added",
        similarQualities: [],
      });
      bPtr++;
      continue;
    }

    break;
  }

  const spanDiffs: SpanDiff[] = merged.map((entry) => {
    const name = entry.spanA?.name ?? entry.spanB?.name ?? "unknown";
    const spanKind = entry.spanA?.spanKind ?? entry.spanB?.spanKind ?? "app";
    const depth = entry.spanA?.depth ?? entry.spanB?.depth ?? 0;

    const base: SpanDiff = {
      status: entry.status,
      name,
      spanKind,
      depth,
    };

    if (entry.spanA?.spanId) base.spanAId = entry.spanA.spanId;
    if (entry.spanB?.spanId) base.spanBId = entry.spanB.spanId;

    if (entry.status === "changed" && entry.spanA && entry.spanB) {
      base.attributeDiffs = computeAttributeDiff(entry.spanA.attributes, entry.spanB.attributes);
      const contentDiff = computeContentDiff(entry.spanA, entry.spanB);
      if (contentDiff) {
        base.contentDiff = contentDiff;
      }
    }

    return base;
  });

  const totalSpansUnion = Math.max(aSpans.length, bSpans.length);
  const matchedCount = merged.filter((e) => e.status === "unchanged").length;
  const similarityScore = totalSpansUnion > 0 ? (matchedCount / totalSpansUnion) * 100 : 100;

  return {
    traceAId: aSpans[0]?.spanId ?? "",
    traceBId: bSpans[0]?.spanId ?? "",
    similarityScore: Math.round(similarityScore * 100) / 100,
    spanDiffs,
    metricDelta: computeMetricDelta(statsA, statsB),
    createdAt: new Date().toISOString(),
  };
}