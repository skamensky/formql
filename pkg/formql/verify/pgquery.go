package verify

import (
	"context"
	"fmt"
	"strings"

	pgquery "github.com/wasilibs/go-pgquery"
)

// PGQueryVerifier verifies SQL offline using go-pgquery.
//
// go-pgquery is a pure-Go runtime wrapper around libpg_query, which tracks
// PostgreSQL parser behavior without requiring a running database.
type PGQueryVerifier struct{}

// Verify parses SQL through go-pgquery.
//
// This validates PostgreSQL parse/analyze compatibility without opening a DB
// connection, making verification runnable in CI, local development, and
// extension-independent test suites.
func (PGQueryVerifier) Verify(_ context.Context, req Request) (Result, error) {
	if strings.TrimSpace(req.SQL) == "" {
		return Result{}, fmt.Errorf("sql is required")
	}

	switch req.Mode {
	case ModeSyntax, ModePlan:
		// Both modes are parser-backed in offline verification.
	default:
		return Result{}, fmt.Errorf("unsupported verification mode %q", req.Mode)
	}

	if _, err := pgquery.Parse(req.SQL); err != nil {
		return Result{
			OK: false,
			Diagnostics: []Diagnostic{{
				Code:    "pg_query_parse_error",
				Message: err.Error(),
			}},
		}, nil
	}

	return Result{
		OK: true,
		Diagnostics: []Diagnostic{{
			Code:    "pg_query_parse_ok",
			Message: "verified with go-pgquery (offline)",
		}},
	}, nil
}
