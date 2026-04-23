package models

import "fmt"

// DiagnosticLevel represents level of diagnostic.
type DiagnosticLevel int

const (
	LevelError   DiagnosticLevel = 1
	LevelWarning DiagnosticLevel = 2
	LevelNote    DiagnosticLevel = 3
	LevelHelp    DiagnosticLevel = 4
)

func (l DiagnosticLevel) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	case LevelNote:
		return "note"
	case LevelHelp:
		return "help"
	default:
		return fmt.Sprintf("DiagnosticLevel(%d)", l)
	}
}

func (l DiagnosticLevel) MarshalText() ([]byte, error) {
	switch l {
	case LevelError:
		return []byte("error"), nil
	case LevelWarning:
		return []byte("warning"), nil
	case LevelNote:
		return []byte("note"), nil
	case LevelHelp:
		return []byte("help"), nil
	default:
		return nil, fmt.Errorf("unsupported level: %d", l)
	}
}

func (l *DiagnosticLevel) UnmarshalText(data []byte) error {
	switch s := string(data); s {
	case "error":
		*l = LevelError
	case "warning":
		*l = LevelWarning
	case "note":
		*l = LevelNote
	case "help":
		*l = LevelHelp
	default:
		return fmt.Errorf("unsupported level: %q", s)
	}
	return nil
}

// Position represents a position in source code.
// Line is 0-based line number.
// Column is 0-based unicode code point offset from line start.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Span represents a range in source code.
type Span struct {
	Start Position  `json:"start"`
	End   *Position `json:"end,omitempty"`
}

// Diagnostic represents a single compiler diagnostic.
type Diagnostic struct {
	Level    DiagnosticLevel `json:"level"`
	Span     *Span           `json:"span,omitempty"`
	Message  string          `json:"message"`
	Code     string          `json:"code,omitempty"`
	Details  []Diagnostic    `json:"details,omitempty"`
}
