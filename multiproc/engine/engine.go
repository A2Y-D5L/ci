package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Run executes all configured processes and emits ProcessLine events to the output channel.
// The channel is closed when all processes complete.
// This method blocks until all processes finish or the context is cancelled.
func (eng *Engine) Run(ctx context.Context, output chan<- ProcessLine) {
	defer close(output)

	factory := eng.CommandFactory
	if factory == nil {
		factory = DefaultCommandFactory
	}

	var wg sync.WaitGroup
	for i, spec := range eng.Specs {
		wg.Add(1)
		go eng.runProcess(ctx, i, spec, factory, output, &wg)
	}

	wg.Wait()
}

// runProcess executes a single process and emits its output as ProcessLine events.
func (eng *Engine) runProcess(
	ctx context.Context,
	idx int,
	spec ProcessSpec,
	factory CommandFactory,
	output chan<- ProcessLine,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	cmd, err := factory(ctx, spec)
	if err != nil {
		output <- ProcessLine{
			Index:      idx,
			IsComplete: true,
			Err:        fmt.Errorf("create command: %w", err),
		}
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		output <- ProcessLine{
			Index:      idx,
			IsComplete: true,
			Err:        fmt.Errorf("stdout pipe: %w", err),
		}
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		output <- ProcessLine{
			Index:      idx,
			IsComplete: true,
			Err:        fmt.Errorf("stderr pipe: %w", err),
		}
		return
	}

	if err := cmd.Start(); err != nil {
		output <- ProcessLine{
			Index:      idx,
			IsComplete: true,
			Err:        fmt.Errorf("start: %w", err),
		}
		return
	}

	var streamsWG sync.WaitGroup
	streamsWG.Add(2)

	// Helper: read from a pipe line-by-line and emit ProcessLine events.
	stream := func(scanner *bufio.Scanner) {
		defer streamsWG.Done()

		// Increase buffer size for long lines.
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			// Normalize line endings for cross-platform compatibility.
			line = strings.TrimRight(line, "\r\n")
			output <- ProcessLine{
				Index:      idx,
				Line:       line,
				IsComplete: false,
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			output <- ProcessLine{
				Index:      idx,
				Line:       fmt.Sprintf("[stream error: %v]", err),
				IsComplete: false,
			}
		}
	}

	go stream(bufio.NewScanner(stdout))
	go stream(bufio.NewScanner(stderr))

	// Monitor for process completion and context cancellation concurrently.
	done := make(chan error, 1)
	go func() {
		streamsWG.Wait()
		done <- cmd.Wait()
	}()

	shutdownTimeout := eng.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5 * time.Second
	}

	select {
	case waitErr := <-done:
		// Process completed normally before cancellation.
		output <- ProcessLine{
			Index:      idx,
			IsComplete: true,
			Err:        waitErr,
		}

	case <-ctx.Done():
		// Context cancelled - initiate graceful shutdown.
		cause := context.Cause(ctx)
		if cause != nil && cause != context.Canceled {
			output <- ProcessLine{
				Index: idx,
				Line:  fmt.Sprintf("[cancellation: %v]", cause),
			}
		}

		// Try graceful termination with SIGTERM first.
		proc := cmd.Process()
		if proc != nil {
			output <- ProcessLine{
				Index: idx,
				Line:  "[sending SIGTERM for graceful shutdown...]",
			}
			_ = proc.Signal(syscall.SIGTERM)

			// Wait for graceful shutdown with timeout.
			select {
			case waitErr := <-done:
				output <- ProcessLine{
					Index: idx,
					Line:  "[gracefully terminated]",
				}
				output <- ProcessLine{
					Index:      idx,
					IsComplete: true,
					Err:        waitErr,
				}

			case <-time.After(shutdownTimeout):
				// Timeout exceeded, force kill.
				output <- ProcessLine{
					Index: idx,
					Line:  fmt.Sprintf("[graceful shutdown timeout (%v), force killing...]", shutdownTimeout),
				}
				_ = proc.Kill()

				// Wait for kill to complete.
				waitErr := <-done
				output <- ProcessLine{
					Index: idx,
					Line:  "[force killed]",
				}
				output <- ProcessLine{
					Index:      idx,
					IsComplete: true,
					Err:        waitErr,
				}
			}
		} else {
			// Process already exited, just emit the done event.
			waitErr := <-done
			output <- ProcessLine{
				Index:      idx,
				IsComplete: true,
				Err:        waitErr,
			}
		}
	}
}
