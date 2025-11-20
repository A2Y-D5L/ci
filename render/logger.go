package render

import (
	"fmt"
	"io"
	"sync"

	"github.com/a2y-d5l/ci"
)

// LogRenderer provides timestamped, namespaced log output.
// Suitable for CI/CD environments where output is captured to files.
type LogRenderer struct {
	w  io.Writer
	mu sync.Mutex
}

// NewLogRenderer creates a new log renderer that writes to w.
func NewLogRenderer(w io.Writer) *LogRenderer {
	return &LogRenderer{w: w}
}

// HandleEvent processes events and outputs them as structured logs.
func (r *LogRenderer) HandleEvent(event ci.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	timestamp := event.Timestamp().Format("2006-01-02 15:04:05.000")

	switch e := event.(type) {
	case ci.PipelineStartedEvent:
		fmt.Fprintf(r.w, "[%s] [PIPELINE] Starting with %d targets\n",
			timestamp, e.TotalTargets)

	case ci.TargetStartedEvent:
		fmt.Fprintf(r.w, "[%s] [%s] Started\n",
			timestamp, e.TargetName())

	case ci.TargetOutputEvent:
		stream := "stdout"
		if e.Stream == ci.StreamStderr {
			stream = "stderr"
		}
		fmt.Fprintf(r.w, "[%s] [%s] [%s] %s\n",
			timestamp, e.TargetName(), stream, e.Line)

	case ci.TargetCompletedEvent:
		status := "completed"
		if e.Error != nil {
			status = fmt.Sprintf("failed: %v", e.Error)
		} else if e.Skipped {
			status = "skipped"
		}
		fmt.Fprintf(r.w, "[%s] [%s] %s (duration: %v)\n",
			timestamp, e.TargetName(), status, e.Duration)

	case ci.PipelineCompletedEvent:
		status := "completed"
		if e.Error != nil {
			status = fmt.Sprintf("failed: %v", e.Error)
		}
		fmt.Fprintf(r.w, "[%s] [PIPELINE] %s (duration: %v)\n",
			timestamp, status, e.Duration)
	}
}
