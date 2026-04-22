package verify

import (
	"context"
	"errors"
	"testing"
)

type stubVerifier struct {
	result Result
	err    error
}

func (s stubVerifier) Verify(_ context.Context, _ Request) (Result, error) {
	if s.err != nil {
		return Result{}, s.err
	}
	return s.result, nil
}

func TestPipelineRequiresAtLeastOneStage(t *testing.T) {
	_, err := (Pipeline{}).Verify(context.Background(), Request{SQL: "SELECT 1", Mode: ModeSyntax})
	if err == nil {
		t.Fatalf("expected error for empty pipeline")
	}
}

func TestPipelineStopsAtFirstFailure(t *testing.T) {
	pipeline := Pipeline{Stages: []Verifier{
		stubVerifier{result: Result{OK: true, Diagnostics: []Diagnostic{{Code: "syntax_ok", Message: "ok"}}}},
		stubVerifier{result: Result{OK: false, Diagnostics: []Diagnostic{{Code: "plan_error", Message: "bad plan"}}}},
		stubVerifier{result: Result{OK: true, Diagnostics: []Diagnostic{{Code: "extension_ok", Message: "ok"}}}},
	}}

	res, err := pipeline.Verify(context.Background(), Request{SQL: "SELECT 1", Mode: ModePlan})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected non-OK result")
	}
	if got, want := len(res.Diagnostics), 2; got != want {
		t.Fatalf("diagnostics length=%d, want %d", got, want)
	}
}

func TestPipelinePropagatesStageErrors(t *testing.T) {
	expected := errors.New("connection lost")
	pipeline := Pipeline{Stages: []Verifier{stubVerifier{err: expected}}}

	_, err := pipeline.Verify(context.Background(), Request{SQL: "SELECT 1", Mode: ModeSyntax})
	if !errors.Is(err, expected) {
		t.Fatalf("error=%v, want %v", err, expected)
	}
}
