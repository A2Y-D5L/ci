package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/a2y-d5l/ci/multiproc/engine"
)

// RenderIncremental renders events directly to standard output without
// clearing the screen or buffering. It is intended for non-TTY
// environments such as CI logs. Lines are prefixed with the process
// name to keep streams distinguishable.
//
// Parameters:
//   - showTimestamps: if true, prefix each line with an RFC3339 timestamp
//   - logPrefix: format string for process name prefix (e.g., "[%s]", "%s:")
func RenderIncremental(ev Event, specs []engine.ProcessSpec, states []ProcessState, showTimestamps bool, logPrefix string) {
	// Default prefix format if not specified
	if logPrefix == "" {
		logPrefix = "[%s]"
	}

	switch e := ev.(type) {
	case lineEvent:
		if e.Index < 0 || e.Index >= len(specs) {
			return
		}
		name := specs[e.Index].Name
		if name == "" {
			name = fmt.Sprintf("proc-%d", e.Index)
		}
		line := strings.TrimRight(e.Line, "\r\n")

		// Build the output line with optional timestamp and configurable prefix
		var output string
		if showTimestamps {
			timestamp := time.Now().UTC().Format(time.RFC3339)
			prefix := fmt.Sprintf(logPrefix, name)
			output = fmt.Sprintf("[%s] %s %s", timestamp, prefix, line)
		} else {
			prefix := fmt.Sprintf(logPrefix, name)
			output = fmt.Sprintf("%s %s", prefix, line)
		}
		fmt.Println(output)

	case doneEvent:
		if e.Index < 0 || e.Index >= len(specs) {
			return
		}
		name := specs[e.Index].Name
		if name == "" {
			name = fmt.Sprintf("proc-%d", e.Index)
		}
		status := FormatExitError(e.Err)

		// Build the completion message with optional timestamp
		var output string
		if showTimestamps {
			timestamp := time.Now().UTC().Format(time.RFC3339)
			prefix := fmt.Sprintf(logPrefix, name)
			output = fmt.Sprintf("[%s] %s %s", timestamp, prefix, status)
		} else {
			prefix := fmt.Sprintf(logPrefix, name)
			output = fmt.Sprintf("%s %s", prefix, status)
		}
		fmt.Println(output)
	}
}

// RenderRequest is a signal used to debounce full-screen renders.
type RenderRequest struct{}
