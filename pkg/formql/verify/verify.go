package verify

import (
	"context"
	"fmt"
)

// Mode controls how strict verification should be.
type Mode string

const (
	// ModeSyntax validates SQL shape and parseability.
	ModeSyntax Mode = "syntax"
	// ModePlan validates SQL by asking PostgreSQL to produce an execution plan.
	ModePlan Mode = "plan"
)

// Request is a backend-agnostic SQL verification request.
type Request struct {
	SQL  string
	Args []any
	Mode Mode
}

// Diagnostic is a machine-readable verification message.
type Diagnostic struct {
	Code    string
	Message string
}

// Result is the normalized output of any verifier implementation.
type Result struct {
	OK          bool
	Diagnostics []Diagnostic
}

// Verifier validates generated SQL before execution.
//
// Implementations are intentionally decoupled from extension internals so the
// same contract can be tested in pure Go while still allowing a PostgreSQL
// extension-backed verifier later.
type Verifier interface {
	Verify(ctx context.Context, req Request) (Result, error)
}

// Pipeline composes multiple verifiers and merges diagnostics.
//
// It fails fast on the first stage that returns OK=false.
type Pipeline struct {
	Stages []Verifier
}

// Verify executes each stage in order.
func (p Pipeline) Verify(ctx context.Context, req Request) (Result, error) {
	if len(p.Stages) == 0 {
		return Result{}, fmt.Errorf("at least one verifier stage is required")
	}

	combined := Result{OK: true}
	for _, stage := range p.Stages {
		res, err := stage.Verify(ctx, req)
		if err != nil {
			return Result{}, err
		}
		combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		if !res.OK {
			combined.OK = false
			return combined, nil
		}
	}

	return combined, nil
}
