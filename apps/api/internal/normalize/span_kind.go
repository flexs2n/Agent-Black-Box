package normalize

import (
	"strings"
	"unicode"
)

const (
	SpanKindGeneration = "generation"
	SpanKindTool       = "tool"
	SpanKindRetrieval  = "retrieval"
	SpanKindApp        = "app"
	SpanKindRoot       = "root"
)

const (
	AttrBlackboxSpanKind        = "blackbox.span_kind"
	AttrGenAIRequestModel       = "gen_ai.request.model"
	AttrGenAIOperationName      = "gen_ai.operation.name"
	AttrGenAISystem             = "gen_ai.system"
	AttrLangfuseObservationType = "langfuse.observation.type"
	AttrLangfuseSpanType        = "langfuse.span.type"
	AttrOpenInferenceSpanKind   = "openinference.span.kind"
)

func DeriveSpanKind(attrs map[string]any, depth int, spanName string) string {
	if depth == 0 {
		return SpanKindRoot
	}

	if v := getStringValue(attrs, AttrBlackboxSpanKind); v != "" {
		return normalizeKind(v)
	}

	if v := getStringValue(attrs, AttrGenAIOperationName); v != "" {
		if v == "chat" || v == "completion" || v == "generate" {
			return SpanKindGeneration
		}
		if v == "tool" || v == "function" {
			return SpanKindTool
		}
		if v == "retrieval" || v == "search" {
			return SpanKindRetrieval
		}
	}

	if v := getStringValue(attrs, AttrGenAISystem); v != "" {
		if v != "unknown" {
			return SpanKindGeneration
		}
	}

	if v := getStringValue(attrs, AttrGenAIRequestModel); v != "" {
		return SpanKindGeneration
	}

	if v := getStringValue(attrs, AttrLangfuseObservationType); v != "" {
		if v == "generation" || v == "llm" {
			return SpanKindGeneration
		}
		if v == "tool" || v == "function" {
			return SpanKindTool
		}
		if v == "retrieval" || v == "search" {
			return SpanKindRetrieval
		}
	}

	if v := getStringValue(attrs, AttrLangfuseSpanType); v != "" {
		if v == "generation" || v == "llm" {
			return SpanKindGeneration
		}
		if v == "tool" || v == "function" {
			return SpanKindTool
		}
		if v == "retrieval" || v == "search" {
			return SpanKindRetrieval
		}
		if v == "chain" || v == "agent" || v == "workflow" {
			return SpanKindApp
		}
	}

	if v := getStringValue(attrs, AttrOpenInferenceSpanKind); v != "" {
		switch v {
		case "llm", "generation", "chat", "completion":
			return SpanKindGeneration
		case "tool", "function":
			return SpanKindTool
		case "retrieval", "search":
			return SpanKindRetrieval
		case "chain", "agent", "workflow", "pipeline":
			return SpanKindApp
		}
	}

	return heuristicSpanKind(spanName)
}

func normalizeKind(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "generation", "llm", "chat", "completion", "generate":
		return SpanKindGeneration
	case "tool", "function":
		return SpanKindTool
	case "retrieval", "search":
		return SpanKindRetrieval
	case "app", "chain", "agent", "workflow", "pipeline":
		return SpanKindApp
	case "root":
		return SpanKindRoot
	default:
		return SpanKindApp
	}
}

func heuristicSpanKind(spanName string) string {
	lower := strings.ToLower(spanName)
	if containsAny(lower, "chat", "completion", "llm", "generate", "embedding", "inference", "model") {
		return SpanKindGeneration
	}
	if containsAny(lower, "tool", "function", "call_", "invoke") {
		return SpanKindTool
	}
	if containsAny(lower, "retriev", "search", "lookup", "query", "vector", "embed") {
		return SpanKindRetrieval
	}
	if containsAny(lower, "chain", "agent", "workflow", "pipeline", "orchestrat", "plan") {
		return SpanKindApp
	}
	return SpanKindApp
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
