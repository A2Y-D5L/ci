package renderer

import (
	"github.com/a2y-d5l/ci/multiproc/engine"
)

// ProcessState holds renderable state for a subprocess.
type ProcessState struct {
	Err      error    // Exit error, if any.
	Name     string   // Display name.
	Lines    []string // Captured log lines (stdout + stderr).
	ByteSize int      // Total bytes currently stored in Lines.
	MaxLines int      // Maximum lines for this process (from spec or config).
	MaxBytes int      // Maximum bytes for this process (from spec or config).
	Done     bool     // True once the process has exited.
	Running  bool     // True until Done is set.
	Dirty    bool     // True when this process needs re-rendering.
}

// Event is a marker interface for renderer events.
type Event interface{ isEvent() }

// lineEvent represents a single line of output for one process.
type lineEvent struct {
	Line  string
	Index int
}

func (lineEvent) isEvent() {}

// doneEvent signals that a process has exited.
type doneEvent struct {
	Err   error
	Index int
}

func (doneEvent) isEvent() {}

// ConvertProcessLineToEvent converts a ProcessLine from the engine to an Event for the renderer.
func ConvertProcessLineToEvent(pl engine.ProcessLine) Event {
	if pl.IsComplete {
		return doneEvent{Index: pl.Index, Err: pl.Err}
	}
	return lineEvent{Index: pl.Index, Line: pl.Line}
}

// ApplyEvent mutates the in-memory state with a single event.
func ApplyEvent(states []ProcessState, ev Event) {
	switch e := ev.(type) {
	case lineEvent:
		if e.Index < 0 || e.Index >= len(states) {
			return
		}
		ps := &states[e.Index]

		// Append line and track byte size.
		lineBytes := len(e.Line)
		ps.Lines = append(ps.Lines, e.Line)
		ps.ByteSize += lineBytes

		// Enforce limits: evict oldest lines if either limit is exceeded.
		// We need to keep removing lines until both constraints are satisfied.
		for {
			exceedsLineLimit := ps.MaxLines > 0 && len(ps.Lines) > ps.MaxLines
			exceedsByteLimit := ps.MaxBytes > 0 && ps.ByteSize > ps.MaxBytes

			if !exceedsLineLimit && !exceedsByteLimit {
				break
			}

			if len(ps.Lines) == 0 {
				break
			}

			// Remove the oldest line.
			oldestLine := ps.Lines[0]
			ps.Lines = ps.Lines[1:]
			ps.ByteSize -= len(oldestLine)
		}

		ps.Dirty = true

	case doneEvent:
		if e.Index < 0 || e.Index >= len(states) {
			return
		}
		ps := &states[e.Index]
		ps.Done = true
		ps.Running = false
		ps.Err = e.Err
		ps.Dirty = true
	}
}

// ExitCodeFromStates returns an appropriate process exit code.
func ExitCodeFromStates(states []ProcessState) int {
	for _, ps := range states {
		if ps.Err != nil {
			return 1
		}
	}
	return 0
}
