package diagnostic

import "fmt"

// Error represents a compiler diagnostic with stage and position metadata.
type Error struct {
	Stage    string `json:"stage"`
	Message  string `json:"message"`
	Position int    `json:"position"`
}

func (e *Error) Error() string {
	if e.Position >= 0 {
		return fmt.Sprintf("%s error at %d: %s", e.Stage, e.Position, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Stage, e.Message)
}

// New returns a structured diagnostic error.
func New(stage, message string, position int) error {
	return &Error{
		Stage:    stage,
		Message:  message,
		Position: position,
	}
}

// Warning is a non-fatal compiler diagnostic.
type Warning struct {
	Stage    string `json:"stage"`
	Message  string `json:"message"`
	Position int    `json:"position"`
}
