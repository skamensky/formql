package verify

import (
	"context"
	"testing"
)

func TestPGQueryVerifierSyntaxAcceptsValidSQL(t *testing.T) {
	res, err := (PGQueryVerifier{}).Verify(context.Background(), Request{SQL: "SELECT 1", Mode: ModeSyntax})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected OK result")
	}
}

func TestPGQueryVerifierRejectsInvalidSQL(t *testing.T) {
	res, err := (PGQueryVerifier{}).Verify(context.Background(), Request{SQL: "SELECT FROM", Mode: ModeSyntax})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected non-OK result")
	}
	if got, want := res.Diagnostics[0].Code, "pg_query_parse_error"; got != want {
		t.Fatalf("diagnostic code=%q, want=%q", got, want)
	}
}

func TestPGQueryVerifierPlanModeUsesOfflineParser(t *testing.T) {
	res, err := (PGQueryVerifier{}).Verify(context.Background(), Request{SQL: "SELECT 1", Mode: ModePlan})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected OK result")
	}
}
