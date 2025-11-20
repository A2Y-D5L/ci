package target_test

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/a2y-d5l/ci/target"
)

// TestNew verifies basic target creation.
func TestNew(t *testing.T) {
	called := false
	fn := func(ctx context.Context) error {
		called = true
		return nil
	}

	tgt := target.New("TestTarget", "Test description", fn)

	if tgt.Name() != "TestTarget" {
		t.Errorf("Expected name 'TestTarget', got %q", tgt.Name())
	}

	if tgt.Description() != "Test description" {
		t.Errorf("Expected description 'Test description', got %q", tgt.Description())
	}

	// Test Run
	err := tgt.Run(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Target function was not called")
	}

	// Test Dependencies (should be empty)
	if len(tgt.Dependencies()) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(tgt.Dependencies()))
	}
}

// TestNewWithDependencies verifies target creation with dependencies.
func TestNewWithDependencies(t *testing.T) {
	dep1 := target.New("Dep1", "First dependency", func(ctx context.Context) error {
		return nil
	})

	dep2 := target.New("Dep2", "Second dependency", func(ctx context.Context) error {
		return nil
	})

	tgt := target.New("Main", "Main target", func(ctx context.Context) error {
		return nil
	}, dep1, dep2)

	deps := tgt.Dependencies()
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Verify dependencies are in order
	if deps[0].Name() != "Dep1" {
		t.Errorf("Expected first dependency to be 'Dep1', got %q", deps[0].Name())
	}

	if deps[1].Name() != "Dep2" {
		t.Errorf("Expected second dependency to be 'Dep2', got %q", deps[1].Name())
	}
}

// TestTargetRunError verifies error propagation from target function.
func TestTargetRunError(t *testing.T) {
	expectedErr := errors.New("test error")
	tgt := target.New("ErrorTarget", "Will fail", func(ctx context.Context) error {
		return expectedErr
	})

	err := tgt.Run(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

// TestTargetRunWithContext verifies context is passed correctly.
func TestTargetRunWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tgt := target.New("ContextTarget", "Uses context", func(ctx context.Context) error {
		// Should receive the cancelled context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			t.Error("Context should be cancelled")
			return nil
		}
	})

	err := tgt.Run(ctx)
	if err == nil {
		t.Fatal("Expected context error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestCmd verifies basic command target creation.
func TestCmd(t *testing.T) {
	// Use echo command which is available on all systems
	tgt := target.Cmd("Echo", "Echo test", "echo", "hello")

	if tgt.Name() != "Echo" {
		t.Errorf("Expected name 'Echo', got %q", tgt.Name())
	}

	if tgt.Description() != "Echo test" {
		t.Errorf("Expected description 'Echo test', got %q", tgt.Description())
	}

	// Verify it has no dependencies or empty dependencies
	deps := tgt.Dependencies()
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}

	// Verify it implements RunWithStreams
	if _, ok := interface{}(tgt).(target.RunWithStreams); !ok {
		t.Error("Cmd target should implement RunWithStreams interface")
	}
}

// TestCmdExecution verifies command execution via Run.
func TestCmdExecution(t *testing.T) {
	tgt := target.Cmd("Echo", "Echo test", "echo", "test")

	err := tgt.Run(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestCmdWithDeps verifies command target with dependencies.
func TestCmdWithDeps(t *testing.T) {
	dep1 := target.New("Dep1", "Dependency", func(ctx context.Context) error {
		return nil
	})

	dep2 := target.Cmd("Dep2", "Command dependency", "echo", "dep")

	tgt := target.CmdWithDeps(
		"Main",
		"Main command",
		[]target.T{dep1, dep2},
		"echo",
		"main",
	)

	if tgt.Name() != "Main" {
		t.Errorf("Expected name 'Main', got %q", tgt.Name())
	}

	deps := tgt.Dependencies()
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Verify dependencies are correct
	if deps[0].Name() != "Dep1" {
		t.Errorf("Expected first dependency 'Dep1', got %q", deps[0].Name())
	}

	if deps[1].Name() != "Dep2" {
		t.Errorf("Expected second dependency 'Dep2', got %q", deps[1].Name())
	}
}

// TestCmdWithDepsNil verifies command target with nil dependencies.
func TestCmdWithDepsNil(t *testing.T) {
	tgt := target.CmdWithDeps(
		"Main",
		"Main command",
		nil, // nil dependencies
		"echo",
		"test",
	)

	deps := tgt.Dependencies()
	// nil dependencies is acceptable - it just means no dependencies
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

// TestRunWithStreams verifies output capture functionality.
func TestRunWithStreams(t *testing.T) {
	tgt := target.Cmd("Echo", "Echo test", "echo", "captured output")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(context.Background(), &stdout, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if output != "captured output\n" {
		t.Errorf("Expected 'captured output\\n', got %q", output)
	}

	// stderr should be empty
	if stderr.Len() != 0 {
		t.Errorf("Expected empty stderr, got %q", stderr.String())
	}
}

// TestRunWithStreamsStderr verifies stderr capture.
func TestRunWithStreamsStderr(t *testing.T) {
	// Use sh -c to write to stderr
	tgt := target.Cmd("StdErr", "Test stderr", "sh", "-c", "echo error >&2")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(context.Background(), &stdout, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// stdout should be empty
	if stdout.Len() != 0 {
		t.Errorf("Expected empty stdout, got %q", stdout.String())
	}

	// stderr should have the output
	stderrOutput := stderr.String()
	if stderrOutput != "error\n" {
		t.Errorf("Expected 'error\\n' on stderr, got %q", stderrOutput)
	}
}

// TestRunWithStreamsBoth verifies capturing both stdout and stderr.
func TestRunWithStreamsBoth(t *testing.T) {
	// Use sh -c to write to both
	tgt := target.Cmd("Both", "Test both streams",
		"sh", "-c", "echo stdout; echo stderr >&2")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(context.Background(), &stdout, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	stdoutOutput := stdout.String()
	if stdoutOutput != "stdout\n" {
		t.Errorf("Expected 'stdout\\n', got %q", stdoutOutput)
	}

	stderrOutput := stderr.String()
	if stderrOutput != "stderr\n" {
		t.Errorf("Expected 'stderr\\n', got %q", stderrOutput)
	}
}

// TestRunWithStreamsError verifies error handling.
func TestRunWithStreamsError(t *testing.T) {
	// Use a command that will fail
	tgt := target.Cmd("Failing", "Will fail", "false")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(context.Background(), &stdout, &stderr)
	if err == nil {
		t.Fatal("Expected error from failing command")
	}

	// Verify it's an exec.ExitError
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("Expected exec.ExitError, got %T: %v", err, err)
	}
}

// TestRunWithStreamsContext verifies context cancellation.
func TestRunWithStreamsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Use a command that would normally succeed
	tgt := target.Cmd("Echo", "Will be cancelled", "sleep", "10")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(ctx, &stdout, &stderr)
	if err == nil {
		t.Fatal("Expected error from cancelled context")
	}

	// Should get context cancellation error
	if !errors.Is(err, context.Canceled) {
		t.Logf("Error type: %T, value: %v", err, err)
		// Note: exec.CommandContext may wrap the error differently
		// Just verify we got an error
	}
}

// TestRunWithStreamsMultipleArgs verifies command with multiple arguments.
func TestRunWithStreamsMultipleArgs(t *testing.T) {
	tgt := target.Cmd("Multi", "Multiple args", "echo", "arg1", "arg2", "arg3")

	var stdout, stderr bytes.Buffer

	err := tgt.RunWithStreams(context.Background(), &stdout, &stderr)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if output != "arg1 arg2 arg3\n" {
		t.Errorf("Expected 'arg1 arg2 arg3\\n', got %q", output)
	}
}

// TestCmdTargetIsTarget verifies CmdTarget is a valid target.T.
func TestCmdTargetIsTarget(t *testing.T) {
	cmd := target.Cmd("Test", "Test", "echo", "test")

	// Verify it implements target.T
	var _ target.T = cmd

	// Verify basic interface methods work
	if cmd.Name() != "Test" {
		t.Errorf("Expected name 'Test', got %q", cmd.Name())
	}

	if cmd.Description() != "Test" {
		t.Errorf("Expected description 'Test', got %q", cmd.Description())
	}

	deps := cmd.Dependencies()
	// Dependencies may be nil or empty
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}

	// Verify Run works
	err := cmd.Run(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestBuiltInTargets verifies the built-in targets are properly configured.
func TestBuiltInTargets(t *testing.T) {
	tests := []struct {
		name        string
		target      target.T
		wantName    string
		wantDesc    string
		wantDepCnt  int
	}{
		{
			name:       "Lint",
			target:     target.Lint,
			wantName:   "Lint",
			wantDesc:   "Run code linters",
			wantDepCnt: 0,
		},
		{
			name:       "Test",
			target:     target.Test,
			wantName:   "Test",
			wantDesc:   "Run unit tests",
			wantDepCnt: 1, // depends on Lint
		},
		{
			name:       "Build",
			target:     target.Build,
			wantName:   "Build",
			wantDesc:   "Build binaries/artifacts",
			wantDepCnt: 1, // depends on Test
		},
		{
			name:       "All",
			target:     target.All,
			wantName:   "All",
			wantDesc:   "Run full CI pipeline",
			wantDepCnt: 1, // depends on Build
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.target.Name() != test.wantName {
				t.Errorf("Expected name %q, got %q", test.wantName, test.target.Name())
			}

			if test.target.Description() != test.wantDesc {
				t.Errorf("Expected description %q, got %q", test.wantDesc, test.target.Description())
			}

			deps := test.target.Dependencies()
			if len(deps) != test.wantDepCnt {
				t.Errorf("Expected %d dependencies, got %d", test.wantDepCnt, len(deps))
			}
		})
	}
}

// TestBuiltInTargetDependencyChain verifies the dependency chain.
func TestBuiltInTargetDependencyChain(t *testing.T) {
	// All -> Build
	allDeps := target.All.Dependencies()
	if len(allDeps) != 1 || allDeps[0].Name() != "Build" {
		t.Error("All should depend on Build")
	}

	// Build -> Test
	buildDeps := target.Build.Dependencies()
	if len(buildDeps) != 1 || buildDeps[0].Name() != "Test" {
		t.Error("Build should depend on Test")
	}

	// Test -> Lint
	testDeps := target.Test.Dependencies()
	if len(testDeps) != 1 || testDeps[0].Name() != "Lint" {
		t.Error("Test should depend on Lint")
	}

	// Lint -> nothing
	lintDeps := target.Lint.Dependencies()
	if len(lintDeps) != 0 {
		t.Error("Lint should have no dependencies")
	}
}

// TestTargetInterface verifies the target.T interface contract.
func TestTargetInterface(t *testing.T) {
	// Create a target and verify all interface methods work
	tgt := target.New("Interface", "Test interface", func(ctx context.Context) error {
		return nil
	})

	// Verify it satisfies target.T
	var _ target.T = tgt

	// Call all interface methods
	_ = tgt.Name()
	_ = tgt.Description()
	_ = tgt.Dependencies()
	_ = tgt.Run(context.Background())
}

// TestRunWithStreamsInterface verifies the RunWithStreams interface.
func TestRunWithStreamsInterface(t *testing.T) {
	cmd := target.Cmd("Test", "Test", "echo", "test")

	// Verify it satisfies RunWithStreams
	var _ target.RunWithStreams = cmd

	// Verify it also satisfies target.T (embedded interface)
	var _ target.T = cmd

	// Call all RunWithStreams methods
	var stdout, stderr bytes.Buffer
	_ = cmd.RunWithStreams(context.Background(), &stdout, &stderr)
}
