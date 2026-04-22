package diagnostic

import (
	"fmt"
	"strings"
)

// Severity is a compiler-visible diagnostic severity.
type Severity string

const (
	SeverityError       Severity = "error"
	SeverityWarning     Severity = "warning"
	SeverityInformation Severity = "information"
	SeverityHint        Severity = "hint"
)

// Issue is the shared compiler diagnostic payload used by errors and warnings.
type Issue struct {
	Stage    string   `json:"stage"`
	Code     string   `json:"code,omitempty"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
	Position int      `json:"position"`
}

// Error represents a fatal compiler diagnostic.
type Error struct {
	Issue
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Hint != "" {
		if e.Position >= 0 {
			return fmt.Sprintf("%s error at %d: %s\nhint: %s", e.Stage, e.Position, e.Message, e.Hint)
		}
		return fmt.Sprintf("%s error: %s\nhint: %s", e.Stage, e.Message, e.Hint)
	}
	if e.Position >= 0 {
		return fmt.Sprintf("%s error at %d: %s", e.Stage, e.Position, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Stage, e.Message)
}

// Warning is a non-fatal compiler diagnostic.
type Warning struct {
	Issue
}

// New returns a structured diagnostic error without an explicit code or hint.
func New(stage, message string, position int) error {
	return NewError(stage, "", message, "", position)
}

// NewError returns a structured fatal compiler diagnostic.
func NewError(stage, code, message, hint string, position int) error {
	return &Error{
		Issue: Issue{
			Stage:    stage,
			Code:     code,
			Severity: SeverityError,
			Message:  message,
			Hint:     hint,
			Position: position,
		},
	}
}

// NewWarning returns a structured non-fatal compiler diagnostic.
func NewWarning(stage, code, message, hint string, position int) Warning {
	return Warning{
		Issue: Issue{
			Stage:    stage,
			Code:     code,
			Severity: SeverityWarning,
			Message:  message,
			Hint:     hint,
			Position: position,
		},
	}
}

// AsError returns the structured fatal diagnostic when available.
func AsError(err error) (*Error, bool) {
	typed, ok := err.(*Error)
	return typed, ok
}

// MessageWithHint formats a user-facing message, preserving hint structure.
func MessageWithHint(issue Issue) string {
	if strings.TrimSpace(issue.Hint) == "" {
		return issue.Message
	}
	return issue.Message + "\nhint: " + issue.Hint
}
