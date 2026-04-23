//go:build js && wasm

package verify

import "context"

type unsupportedVerifier struct{}

// DefaultVerifier returns a graceful verifier implementation for js/wasm builds.
// The compiler and schema tooling remain available in the browser, but SQL
// syntax verification still requires a non-browser runtime.
func DefaultVerifier() Verifier {
	return unsupportedVerifier{}
}

func (unsupportedVerifier) Verify(_ context.Context, request Request) (Result, error) {
	return Result{
		OK: false,
		Diagnostics: []Diagnostic{{
			Code:    "verification_unavailable",
			Message: "SQL verification is not supported in js/wasm builds yet",
		}},
	}, nil
}
