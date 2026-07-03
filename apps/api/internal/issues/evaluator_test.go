package issues

import (
	"context"
	"testing"

	"github.com/blackbox-agentdiff/api/internal/model"
)

func TestOutputFormatMismatch_DetectsInvalidJSON(t *testing.T) {
	spans := []model.Span{
		{
			SpanKind:   "generation",
			Name:       "test_gen",
			Attributes: `{"gen_ai.prompt": "Return a JSON object with name and age", "gen_ai.completion": "not json at all"}`,
		},
	}
	findings, err := OutputFormatMismatch(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings for non-JSON output when prompt expects JSON")
	}
	if findings[0].Evaluator != EvaluatorOutputFormatMismatch {
		t.Fatalf("expected output_format_mismatch evaluator, got %s", findings[0].Evaluator)
	}
}

func TestOutputFormatMismatch_DetectsMalformedJSON(t *testing.T) {
	spans := []model.Span{
		{
			SpanKind:   "generation",
			Name:       "test_gen",
			Attributes: `{"gen_ai.prompt": "Return JSON please", "gen_ai.completion": "{name: John}"}`,
		},
	}
	findings, err := OutputFormatMismatch(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings for malformed JSON")
	}
}

func TestOutputFormatMismatch_ValidJSONPasses(t *testing.T) {
	spans := []model.Span{
		{
			SpanKind:   "generation",
			Name:       "test_gen",
			Attributes: `{"gen_ai.prompt": "Return as JSON", "gen_ai.completion": "{\"name\": \"John\", \"age\": 30}"}`,
		},
	}
	findings, err := OutputFormatMismatch(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for valid JSON, got %d", len(findings))
	}
}

func TestOutputFormatMismatch_SkipsNonGenerationSpans(t *testing.T) {
	spans := []model.Span{
		{
			SpanKind:   "tool",
			Name:       "test_tool",
			Attributes: `{"tool.input": "{}", "tool.output": "not json"}`,
		},
	}
	findings, err := OutputFormatMismatch(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-generation spans, got %d", len(findings))
	}
}

func TestOutputFormatMismatch_SkipsWhenNoJSONHint(t *testing.T) {
	spans := []model.Span{
		{
			SpanKind:   "generation",
			Name:       "test_gen",
			Attributes: `{"gen_ai.prompt": "Say hello", "gen_ai.completion": "Hello!"}`,
		},
	}
	findings, err := OutputFormatMismatch(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings when no JSON hint in prompt, got %d", len(findings))
	}
}

func TestHallucination_SkipsWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	spans := []model.Span{
		{
			SpanKind:   "generation",
			Name:       "test_gen",
			Attributes: `{"gen_ai.prompt": "Say hello", "gen_ai.completion": "Hello!"}`,
		},
	}
	findings, err := Hallucination(context.Background(), model.Trace{}, spans, model.TraceStats{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings without API key, got %d", len(findings))
	}
}
