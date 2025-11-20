package engine

import (
	"io"
	"syscall"
)

// ProcessLine represents a single line of output from a process.
// This is the raw event emitted by the Engine, consumed by renderers.
type ProcessLine struct {
	// Err contains the process exit error, if any. Only set when IsComplete is true.
	Err error
	// Line is the actual output text (already normalized).
	Line string
	// Index identifies which process emitted this line.
	Index int
	// IsComplete indicates whether this is the final event for this process.
	// When true, Err contains the exit status.
	IsComplete bool
}

// ProcessSpec describes a subprocess to run.
type ProcessSpec struct {
	// Name is a logical label for the subprocess, used in headers.
	Name string

	// Command is the executable, e.g. "sh", "bash", "cmd".
	Command string

	// Args are the command-line arguments.
	Args []string

	// MaxLines is the maximum number of output lines to keep for this process.
	// If 0, uses the global Config.MaxLinesPerProc default.
	// This allows fine-grained control over memory usage per process.
	MaxLines int

	// MaxBytes is the maximum number of bytes to keep in the output history
	// for this process. If 0, no byte limit is enforced (only line limit applies).
	// When both MaxLines and MaxBytes are set, the stricter limit applies.
	MaxBytes int
}

// Command is an abstraction over os/exec.Cmd to enable testing.
// This interface represents a runnable command with capturable output.
type Command interface {
	// StdoutPipe returns a reader for the command's stdout.
	StdoutPipe() (io.ReadCloser, error)

	// StderrPipe returns a reader for the command's stderr.
	StderrPipe() (io.ReadCloser, error)

	// Start begins execution of the command.
	Start() error

	// Wait waits for the command to exit and returns any error.
	Wait() error

	// Process returns the underlying process, if available.
	// This is used for signal handling (SIGTERM/SIGKILL).
	Process() ProcessHandle
}

// ProcessHandle is an abstraction over os.Process for signal handling.
type ProcessHandle interface {
	// Signal sends a signal to the process.
	Signal(sig syscall.Signal) error

	// Kill terminates the process.
	Kill() error
}
