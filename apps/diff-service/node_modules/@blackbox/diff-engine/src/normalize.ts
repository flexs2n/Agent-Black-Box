import { SpanNode, NormalizedSpan } from "./types.js";

const INTERNAL_ATTR_PREFIXES = ['trace.', 'trace_id', 'span.'];

function isInternalAttr(key: string): boolean {
  return INTERNAL_ATTR_PREFIXES.some(prefix => key.startsWith(prefix));
}

export function normalizeSpan(node: SpanNode, depth: number = 0, parentId?: string): NormalizedSpan {
  const attrs = { ...node.attributes };
  for (const key of Object.keys(attrs)) {
    if (isInternalAttr(key)) {
      delete attrs[key];
    }
  }

  return {
    spanId: node.spanId,
    parentSpanId: parentId,
    name: node.name,
    spanKind: deriveSpanKind(node, depth),
    attributes: attrs,
    startTime: node.startTime,
    endTime: node.endTime,
    depth,
    children: [],
  };
}

function deriveSpanKind(node: SpanNode, depth: number): NormalizedSpan['spanKind'] {
  const { attributes } = node;

  if (depth === 0) {
    return 'root';
  }

  if (attributes['blackbox.span_kind']) {
    return attributes['blackbox.span_kind'] as NormalizedSpan['spanKind'];
  }

  for (const [key, value] of Object.entries(attributes)) {
    if (!value) continue;
    if (key.startsWith('gen_ai.')) {
      return 'generation';
    }
    if (key.startsWith('tool.')) {
      return 'tool';
    }
    if (key.startsWith('retrieval.')) {
      return 'retrieval';
    }
  }

  for (const [key, value] of Object.entries(attributes)) {
    if (!value) continue;
    if (key.startsWith('langfuse.')) {
      if (key === 'langfuse.trace.name') return 'root';
      if (key === 'langfuse.observation.type') {
        const type = (value as string).toLowerCase();
        if (type === 'generation' || type === 'tool') return type as NormalizedSpan['spanKind'];
      }
    }
  }

  for (const [key, value] of Object.entries(attributes)) {
    if (!value) continue;
    if (key.startsWith('openinference.')) {
      if (key === 'openinference.span.kind') {
        return (value as string).toLowerCase() as NormalizedSpan['spanKind'];
      }
      if (key === 'openinference.span.kind.end' || key === 'openinference.span.kind.begin') {
        return ('generation') as NormalizedSpan['spanKind'];
      }
    }
  }

  const nameLower = node.name.toLowerCase();
  if (
    nameLower.includes('chat') ||
    nameLower.includes('completion') ||
    nameLower.includes('prompt') ||
    nameLower.includes('llm') ||
    nameLower.includes('openai') ||
    nameLower.includes('anthropic') ||
    nameLower.includes('gen_ai')
  ) {
    return 'generation';
  }
  if (nameLower.includes('tool') || nameLower.includes('function') || nameLower.includes('search')) {
    return 'tool';
  }
  if (nameLower.includes('retrieval') || nameLower.includes('search')) {
    return 'retrieval';
  }

  return 'app';
}

export function normalizeTree(node: SpanNode, depth: number = 0, parentId?: string): NormalizedSpan {
  const normalized = normalizeSpan(node, depth, parentId);
  const children = node.children.map((child: SpanNode) => normalizeTree(child, depth + 1, node.spanId));
  return {
    ...normalized,
    children,
  };
}

export function flattenNormalized(node: NormalizedSpan): NormalizedSpan[] {
  const result: NormalizedSpan[] = [node];
  for (const child of node.children) {
    result.push(...flattenNormalized(child));
  }
  return result;
}