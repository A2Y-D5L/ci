package render

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/a2y-d5l/ci"
)

// TUIRenderer provides a rich terminal interface.
// Output is organized by target, not interleaved.
type TUIRenderer struct {
	w              io.Writer
	mu             sync.Mutex
	targetOutputs  map[string][]string
	targetStatus   map[string]string
	runningTargets []string
	totalTargets   int
	completed      int
}

// NewTUIRenderer creates a new TUI renderer that writes to w.
func NewTUIRenderer(w io.Writer) *TUIRenderer {
	return &TUIRenderer{
		w:             w,
		targetOutputs: make(map[string][]string),
		targetStatus:  make(map[string]string),
	}
}

// HandleEvent processes events and updates the TUI display.
func (r *TUIRenderer) HandleEvent(event ci.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch e := event.(type) {
	case ci.PipelineStartedEvent:
		r.totalTargets = e.TotalTargets
		r.renderPipelineStart(e)

	case ci.TargetStartedEvent:
		r.targetStatus[e.TargetName()] = "running"
		r.runningTargets = append(r.runningTargets, e.TargetName())
		r.render()

	case ci.TargetOutputEvent:
		r.targetOutputs[e.TargetName()] = append(
			r.targetOutputs[e.TargetName()],
			e.Line,
		)
		r.render()

	case ci.TargetCompletedEvent:
		if e.Error != nil {
			r.targetStatus[e.TargetName()] = "failed"
		} else if e.Skipped {
			r.targetStatus[e.TargetName()] = "skipped"
		} else {
			r.targetStatus[e.TargetName()] = "success"
		}
		r.removeRunning(e.TargetName())
		r.completed++
		r.render()

	case ci.PipelineCompletedEvent:
		r.renderPipelineComplete(e)
	}
}

func (r *TUIRenderer) renderPipelineStart(e ci.PipelineStartedEvent) {
	fmt.Fprintf(r.w, "╔═══════════════════════════════════════╗\n")
	fmt.Fprintf(r.w, "║  CI Pipeline Starting                 ║\n")
	fmt.Fprintf(r.w, "║  Total Targets: %-22d║\n", e.TotalTargets)
	fmt.Fprintf(r.w, "╚═══════════════════════════════════════╝\n\n")
}

func (r *TUIRenderer) render() {
	// For each running target, show its status and latest output
	for _, targetName := range r.runningTargets {
		status := r.targetStatus[targetName]
		fmt.Fprintf(r.w, "\n┌─ %s ", targetName)
		r.w.Write([]byte(strings.Repeat("─", max(0, 40-len(targetName)-3))))
		fmt.Fprintf(r.w, "┐\n")

		statusSymbol := "▶"
		if status == "running" {
			statusSymbol = "⚙"
		}
		fmt.Fprintf(r.w, "│ %s Running...%s│\n",
			statusSymbol,
			strings.Repeat(" ", max(0, 40-len("Running...")-3)))

		// Show last few lines of output
		outputs := r.targetOutputs[targetName]
		startIdx := 0
		if len(outputs) > 5 {
			startIdx = len(outputs) - 5
		}
		for i := startIdx; i < len(outputs); i++ {
			line := outputs[i]
			if len(line) > 37 {
				line = line[:34] + "..."
			}
			fmt.Fprintf(r.w, "│ %s%s│\n",
				line,
				strings.Repeat(" ", max(0, 40-len(line)-2)))
		}

		fmt.Fprintf(r.w, "└")
		r.w.Write([]byte(strings.Repeat("─", 40)))
		fmt.Fprintf(r.w, "┘\n")
	}
}

func (r *TUIRenderer) removeRunning(targetName string) {
	for i, name := range r.runningTargets {
		if name == targetName {
			r.runningTargets = append(r.runningTargets[:i], r.runningTargets[i+1:]...)
			break
		}
	}
}

func (r *TUIRenderer) renderPipelineComplete(e ci.PipelineCompletedEvent) {
	fmt.Fprintf(r.w, "\n╔═══════════════════════════════════════╗\n")
	if e.Error != nil {
		fmt.Fprintf(r.w, "║  CI Pipeline FAILED                   ║\n")
	} else {
		fmt.Fprintf(r.w, "║  CI Pipeline COMPLETED                ║\n")
	}
	fmt.Fprintf(r.w, "║  Duration: %-27s║\n", e.Duration.String())
	fmt.Fprintf(r.w, "╚═══════════════════════════════════════╝\n\n")

	// Summary of all targets
	fmt.Fprintf(r.w, "Summary:\n")
	for targetName, status := range r.targetStatus {
		symbol := "✓"
		switch status {
		case "failed":
			symbol = "✗"
		case "skipped":
			symbol = "○"
		case "success":
			symbol = "✓"
		}
		fmt.Fprintf(r.w, "  %s %s\n", symbol, targetName)
	}
	fmt.Fprintf(r.w, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
