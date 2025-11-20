package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/a2y-d5l/ci/multiproc/engine"
	"github.com/a2y-d5l/ci/multiproc/renderer"
)

// Config holds high-level configuration for running multiple processes.
type Config struct {
	// IsTTY indicates whether stdout is attached to a TTY. When nil,
	// the value is auto-detected using renderer.IsTTY(). Callers can override
	// this to force a particular render mode (e.g., in tests).
	IsTTY *bool

	// Specs defines the processes to run.
	Specs []engine.ProcessSpec

	// MaxLinesPerProc is the default maximum number of output lines to keep per process.
	MaxLinesPerProc int

	// ShutdownTimeout is the maximum time to wait for graceful shutdown
	// before force-killing processes. If zero, uses a default of 5 seconds.
	ShutdownTimeout time.Duration

	// FullScreen enables full-screen terminal rendering with screen clearing.
	FullScreen bool

	// ShowSummary enables printing a summary to stderr after execution completes.
	ShowSummary bool

	// ShowTimestamps prefixes each output line with an RFC3339 timestamp.
	// Useful for debugging timing issues and analyzing slow commands.
	ShowTimestamps bool

	// LogPrefix defines the format for prefixing process names in non-TTY mode.
	// Common values: "[%s]", "%s:", "(%s)", etc.
	// The %s placeholder is replaced with the process name.
	// If empty, defaults to "[%s]".
	LogPrefix string
}

// DefaultConfig returns sensible defaults. Callers can override fields.
func DefaultConfig() Config {
	return Config{
		Specs:           nil,
		MaxLinesPerProc: 1000,
		FullScreen:      true,
		ShowSummary:     true,
		IsTTY:           nil,
		ShutdownTimeout: 5 * time.Second,
		ShowTimestamps:  false,
		LogPrefix:       "[%s]",
	}
}

// Run executes the configured processes and manages rendering.
// It returns a process-style exit code. Callers are responsible for
// mapping this to os.Exit in a CLI.
func Run(ctx context.Context, cfg Config) int {
	// Derive effective configuration, falling back to defaults.
	base := DefaultConfig()
	if cfg.MaxLinesPerProc <= 0 {
		cfg.MaxLinesPerProc = base.MaxLinesPerProc
	}
	if cfg.Specs == nil {
		cfg.Specs = base.Specs
	}
	if cfg.IsTTY == nil {
		val := renderer.IsTTY()
		cfg.IsTTY = &val
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = base.ShutdownTimeout
	}
	if cfg.LogPrefix == "" {
		cfg.LogPrefix = base.LogPrefix
	}

	// In non-TTY environments, full-screen rendering is not useful, so
	// force it off. Incremental renderer will still run.
	if cfg.IsTTY != nil && !*cfg.IsTTY {
		cfg.FullScreen = false
	}

	specs := cfg.Specs

	// Build initial render state.
	states := make([]renderer.ProcessState, len(specs))
	for i, spec := range specs {
		// Determine effective limits for this process.
		maxLines := spec.MaxLines
		if maxLines <= 0 {
			maxLines = cfg.MaxLinesPerProc
		}
		maxBytes := spec.MaxBytes
		// Note: if maxBytes is 0, no byte limit is enforced.

		states[i] = renderer.ProcessState{
			Name:     spec.Name,
			Lines:    nil,
			Done:     false,
			Err:      nil,
			Running:  true,
			Dirty:    true, // initial state should be rendered
			ByteSize: 0,
			MaxLines: maxLines,
			MaxBytes: maxBytes,
		}
	}

	events := make(chan renderer.Event, 128)

	// Use the Engine to run processes.
	eng := engine.New(specs, cfg.ShutdownTimeout)

	// Convert ProcessLine events from engine to Event for rendering.
	processLines := make(chan engine.ProcessLine, 128)
	var engineWG sync.WaitGroup
	engineWG.Go(func() {
		eng.Run(ctx, processLines)
	})

	// Convert engine events to renderer events.
	go func() {
		for pl := range processLines {
			events <- renderer.ConvertProcessLineToEvent(pl)
		}
		close(events)
	}()

	var renderCh chan renderer.RenderRequest
	if cfg.FullScreen && cfg.IsTTY != nil && *cfg.IsTTY {
		renderCh = make(chan renderer.RenderRequest, 1)
		// Dedicated render loop with debouncing.
		go func() {
			for range renderCh {
				renderer.RenderScreen(states)
			}
		}()

		// Queue initial render to show "starting" status for all processes.
		renderCh <- renderer.RenderRequest{}
	} else if cfg.IsTTY != nil && !*cfg.IsTTY {
		// In non-TTY mode, print initial status for all processes
		for i, spec := range specs {
			name := spec.Name
			if name == "" {
				name = fmt.Sprintf("proc-%d", i)
			}
			prefix := fmt.Sprintf(cfg.LogPrefix, name)
			if cfg.ShowTimestamps {
				timestamp := time.Now().UTC().Format(time.RFC3339)
				fmt.Printf("[%s] %s starting...\n", timestamp, prefix)
			} else {
				fmt.Printf("%s starting...\n", prefix)
			}
		}
	}

	// Main event loop: update state and re-render in real time.
	for ev := range events {
		renderer.ApplyEvent(states, ev)
		if cfg.IsTTY != nil && *cfg.IsTTY && cfg.FullScreen {
			// Non-blocking send to debounce renders.
			select {
			case renderCh <- renderer.RenderRequest{}:
			default:
			}
		} else {
			// Non-TTY incremental renderer.
			renderer.RenderIncremental(ev, specs, states, cfg.ShowTimestamps, cfg.LogPrefix)
		}
	}

	// Final render (in case we exited without drawing the last frame).
	if renderCh != nil {
		// Ensure the last state is rendered, then close the loop.
		renderCh <- renderer.RenderRequest{}
		close(renderCh)
	}

	// Print a short summary to stderr.
	if cfg.ShowSummary {
		renderer.WriteFinalSummary(states)
	}

	// Return exit code for caller to handle.
	return renderer.ExitCodeFromStates(states)
}
