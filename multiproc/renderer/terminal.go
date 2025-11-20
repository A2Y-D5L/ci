package renderer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// clearScreen writes ANSI escape codes to clear the screen and move
// the cursor to the top-left. Most terminals support this. When
// output is redirected (e.g. to a file), the escape codes will
// simply be part of the output, which is still readable.
func clearScreen() {
	fmt.Print("\x1b[H\x1b[2J")
}

// RenderScreen re-renders the entire screen from current state.
//
// This keeps each process' stream visually grouped and avoids
// interleaving output, at the cost of re-rendering on each update.
func RenderScreen(states []ProcessState) {
	// Fast path: if nothing is dirty, skip the render entirely.
	hasDirty := false
	for _, ps := range states {
		if ps.Dirty {
			hasDirty = true
			break
		}
	}
	if !hasDirty {
		return
	}

	clearScreen()

	for i := range states {
		ps := &states[i]
		status := "running"
		if ps.Done {
			status = FormatExitError(ps.Err)
		}

		// Header: "Running Subprocess A… [running]"
		fmt.Printf("Running %s… [%s]\n", ps.Name, status)

		for _, line := range ps.Lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				fmt.Println()
				continue
			}
			fmt.Printf("    %s\n", line)
		}

		fmt.Println()
		ps.Dirty = false
	}

	fmt.Println("Press Ctrl+C to cancel. Output updates in real time.")
}

// FormatExitError formats an exit error with detailed information.
// It extracts exit codes, signals, and other termination details from
// exec.ExitError to provide human-readable error messages.
func FormatExitError(err error) string {
	if err == nil {
		return "ok"
	}

	// Check if it's an exec.ExitError
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		// Not an ExitError, return the error string as-is
		return fmt.Sprintf("error: %v", err)
	}

	// Extract exit code
	exitCode := exitErr.ExitCode()

	// Check for signal-based termination
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
		if status.Signaled() {
			signal := status.Signal()
			return fmt.Sprintf("killed by signal %v (exit code %d)", signal, exitCode)
		}
	}

	// Standard exit code
	return fmt.Sprintf("exit code %d", exitCode)
}

// WriteFinalSummary prints a concise summary after the real-time view
// completes. This is useful when scrollback is long or output is
// redirected.
func WriteFinalSummary(states []ProcessState) {
	fmt.Fprintln(os.Stderr, "\nSummary:")
	for _, ps := range states {
		status := FormatExitError(ps.Err)
		fmt.Fprintf(os.Stderr, "  - %s: %s\n", ps.Name, status)
	}
}

// IsTTY reports whether the current stdout is a TTY (character device).
// This is used to decide between full-screen and incremental renderers.
func IsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode&os.ModeCharDevice != 0
}
