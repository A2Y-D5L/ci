package render

import (
	"fmt"
	"io"
	"sync"

	"github.com/a2y-d5l/ci"
)

// SimpleRenderer provides minimal output.
// Displays only target start/completion with simple symbols.
type SimpleRenderer struct {
	w  io.Writer
	mu sync.Mutex
}

// NewSimpleRenderer creates a new simple renderer that writes to w.
func NewSimpleRenderer(w io.Writer) *SimpleRenderer {
	return &SimpleRenderer{w: w}
}

// HandleEvent processes events and outputs them in a minimal format.
func (r *SimpleRenderer) HandleEvent(event ci.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch e := event.(type) {
	case ci.TargetStartedEvent:
		fmt.Fprintf(r.w, "▶ %s\n", e.TargetName())

	case ci.TargetCompletedEvent:
		symbol := "✓"
		if e.Error != nil {
			symbol = "✗"
		} else if e.Skipped {
			symbol = "○"
		}
		fmt.Fprintf(r.w, "%s %s\n", symbol, e.TargetName())
	}
}
