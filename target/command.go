package target

import (
	"context"
	"io"
	"os/exec"
)

// CmdTarget is a target that executes a shell command with output capture support.
type CmdTarget struct {
	*Target
	cmdName string
	args    []string
}

// Cmd creates a target that executes a shell command with output capture support.
// The command's stdout and stderr will be captured and emitted as TargetOutputEvent.
//
// Example:
//
//	lint := target.Cmd("Lint", "Run linters", "golangci-lint", "run", "./...")
func Cmd(name, desc string, cmdName string, args ...string) *CmdTarget {
	return CmdWithDeps(name, desc, nil, cmdName, args...)
}

// CmdWithDeps creates a target that executes a shell command with dependencies.
//
// Example:
//
//	test := target.CmdWithDeps("Test", "Run tests", []target.T{lint}, "go", "test", "./...")
func CmdWithDeps(name, desc string, deps []T, cmdName string, args ...string) *CmdTarget {
	// Create the base function that uses os.Stdout/Stderr (fallback)
	fn := func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, cmdName, args...)
		cmd.Stdout = io.Discard // Discard if not captured (will use RunWithStreams)
		cmd.Stderr = io.Discard
		return cmd.Run()
	}

	return &CmdTarget{
		Target:  New(name, desc, fn, deps...),
		cmdName: cmdName,
		args:    args,
	}
}

// RunWithStreams implements the RunWithStreams interface.
// This allows the engine to capture stdout/stderr as events.
func (c *CmdTarget) RunWithStreams(ctx context.Context, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, c.cmdName, c.args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
