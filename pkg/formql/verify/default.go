//go:build !js

package verify

// DefaultVerifier returns the shared baseline verification pipeline used by
// CLI tooling and the PostgreSQL extension bridge.
func DefaultVerifier() Verifier {
	return Pipeline{
		Stages: []Verifier{
			PGQueryVerifier{},
		},
	}
}
