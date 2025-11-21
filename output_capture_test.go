package ci_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/a2y-d5l/ci"
	"github.com/a2y-d5l/ci/render"
	"github.com/a2y-d5l/ci/target"
)

// TestOutputCapture verifies that target output is captured and emitted as events.
func TestOutputCapture(t *testing.T) {
	collector := render.NewCollector()

	// Use the Cmd helper which implements RunWithStreams
	tgt := target.Cmd("Echo", "Echo test", "echo", "test output line")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, tgt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify output was captured
	outputEvents := 0
	for _, e := range collector.GetEvents() {
		if output, ok := e.(ci.TargetOutputEvent); ok {
			if output.TargetName() == "Echo" && output.Line == "test output line" {
				outputEvents++
			}
		}
	}

	if outputEvents != 1 {
		t.Errorf("Expected 1 output event, got %d", outputEvents)
	}
}

// TestOutputCaptureMultipleLines verifies that multiple output lines are captured.
func TestOutputCaptureMultipleLines(t *testing.T) {
	collector := render.NewCollector()

	// Use Cmd helper to create a target that outputs multiple lines
	tgt := target.Cmd("MultiLine", "Multiple lines", "sh", "-c", "echo line1; echo line2; echo line3")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, tgt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Count output events
	outputCount := collector.TargetOutputCount("MultiLine")
	if outputCount != 3 {
		t.Errorf("Expected 3 output events, got %d", outputCount)
	}

	// Verify all three lines were captured
	events := collector.GetEvents()
	lines := []string{}
	for _, e := range events {
		if output, ok := e.(ci.TargetOutputEvent); ok && output.TargetName() == "MultiLine" {
			lines = append(lines, output.Line)
		}
	}

	expectedLines := []string{"line1", "line2", "line3"}
	for i, expected := range expectedLines {
		if i >= len(lines) || lines[i] != expected {
			t.Errorf("Expected line %d to be %q, got %q", i, expected, lines[i])
		}
	}
}

// TestOutputCaptureStderr verifies that stderr is captured separately.
func TestOutputCaptureStderr(t *testing.T) {
	collector := render.NewCollector()

	// Use sh -c to write to both stdout and stderr
	tgt := target.Cmd("StdErr", "Test stderr", "sh", "-c", "echo stdout; echo stderr >&2")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, tgt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify we got both stdout and stderr events
	var stdoutEvents, stderrEvents int
	for _, e := range collector.GetEvents() {
		if output, ok := e.(ci.TargetOutputEvent); ok && output.TargetName() == "StdErr" {
			if output.Stream == ci.StreamStdout && output.Line == "stdout" {
				stdoutEvents++
			}
			if output.Stream == ci.StreamStderr && output.Line == "stderr" {
				stderrEvents++
			}
		}
	}

	if stdoutEvents != 1 {
		t.Errorf("Expected 1 stdout event, got %d", stdoutEvents)
	}
	if stderrEvents != 1 {
		t.Errorf("Expected 1 stderr event, got %d", stderrEvents)
	}
}

// TestNoOutputCaptureWithoutHandler verifies that targets without
// RunWithStreams still work (fallback behavior).
func TestNoOutputCaptureWithoutHandler(t *testing.T) {
	// Run without handler - should not capture output
	tgt := target.New("NoCapture", "No capture test", func(_ context.Context) error {
		fmt.Println("This should not be captured")
		return nil
	})

	_, err := ci.RunTargets(context.Background(), tgt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Success if no panic or error
}

// TestOutputCaptureWithDependencies verifies output capture works in DAGs.
func TestOutputCaptureWithDependencies(t *testing.T) {
	collector := render.NewCollector()

	a := target.Cmd("A", "Base", "echo", "output from A")
	b := target.CmdWithDeps("B", "Depends on A", []target.T{a}, "echo", "output from B")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify both targets had output captured
	if collector.TargetOutputCount("A") != 1 {
		t.Errorf("Expected 1 output event for A, got %d", collector.TargetOutputCount("A"))
	}
	if collector.TargetOutputCount("B") != 1 {
		t.Errorf("Expected 1 output event for B, got %d", collector.TargetOutputCount("B"))
	}

	// Verify output content
	events := collector.GetEvents()
	aOutput := false
	bOutput := false
	for _, e := range events {
		if output, ok := e.(ci.TargetOutputEvent); ok {
			if output.TargetName() == "A" && output.Line == "output from A" {
				aOutput = true
			}
			if output.TargetName() == "B" && output.Line == "output from B" {
				bOutput = true
			}
		}
	}

	if !aOutput {
		t.Error("Did not capture output from A")
	}
	if !bOutput {
		t.Error("Did not capture output from B")
	}
}

// TestOutputCaptureIntegrationWithLogRenderer verifies output
// appears correctly in the LogRenderer.
func TestOutputCaptureIntegrationWithLogRenderer(t *testing.T) {
	var buf strings.Builder
	renderer := render.NewLogRenderer(&buf)

	tgt := target.Cmd("Integration", "Integration test", "echo", "integration output")

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, tgt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify output contains the expected elements
	if !strings.Contains(output, "[Integration]") {
		t.Error("Expected target name in output")
	}
	if !strings.Contains(output, "[stdout]") {
		t.Error("Expected stdout indicator in output")
	}
	if !strings.Contains(output, "integration output") {
		t.Error("Expected captured output in log")
	}
}

// TestMixedTargets verifies that targets with and without
// RunWithStreams can coexist in the same pipeline.
func TestMixedTargets(t *testing.T) {
	collector := render.NewCollector()

	// Target WITH capture
	withCapture := target.Cmd("WithCapture", "Has capture", "echo", "captured")

	// Target WITHOUT capture
	withoutCapture := target.New("WithoutCapture", "No capture", func(_ context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, withCapture, withoutCapture)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// WithCapture should have output events
	if collector.TargetOutputCount("WithCapture") != 1 {
		t.Errorf("Expected 1 output event for WithCapture, got %d",
			collector.TargetOutputCount("WithCapture"))
	}

	// WithoutCapture should have no output events
	if collector.TargetOutputCount("WithoutCapture") != 0 {
		t.Errorf("Expected 0 output events for WithoutCapture, got %d",
			collector.TargetOutputCount("WithoutCapture"))
	}

	// Both should have completed
	if collector.TargetCompletedCount("WithCapture") != 1 {
		t.Error("WithCapture should have completed")
	}
	if collector.TargetCompletedCount("WithoutCapture") != 1 {
		t.Error("WithoutCapture should have completed")
	}
}
