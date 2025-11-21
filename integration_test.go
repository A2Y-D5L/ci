package ci_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/a2y-d5l/ci"
	"github.com/a2y-d5l/ci/render"
	"github.com/a2y-d5l/ci/target"
)

// TestEndToEndWithLogRenderer verifies the complete pipeline with LogRenderer.
// This is what users experience in CI/CD environments.
func TestEndToEndWithLogRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	a := target.Cmd("A", "Base", "echo", "output from A")
	b := target.CmdWithDeps("B", "Depends on A", []target.T{a}, "echo", "output from B")

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify structured log output
	lines := strings.Split(output, "\n")

	// Should have timestamps on all non-empty lines
	timestampCount := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[20") {
			timestampCount++
		} else {
			t.Errorf("Line missing timestamp: %s", line)
		}
	}

	if timestampCount == 0 {
		t.Error("No timestamped lines found")
	}

	// Verify pipeline events
	if !strings.Contains(output, "[PIPELINE] Starting") {
		t.Error("Missing pipeline start event")
	}

	if !strings.Contains(output, "[PIPELINE] completed") {
		t.Error("Missing pipeline completion event")
	}

	// Verify target events
	if !strings.Contains(output, "[A] Started") {
		t.Error("Missing target A started event")
	}

	if !strings.Contains(output, "[B] Started") {
		t.Error("Missing target B started event")
	}

	if !strings.Contains(output, "[A] completed") {
		t.Error("Missing target A completion event")
	}

	if !strings.Contains(output, "[B] completed") {
		t.Error("Missing target B completion event")
	}

	// Verify output was captured
	if !strings.Contains(output, "[A] [stdout] output from A") {
		t.Error("Missing captured output from A")
	}

	if !strings.Contains(output, "[B] [stdout] output from B") {
		t.Error("Missing captured output from B")
	}

	// Verify duration is logged
	if !strings.Contains(output, "duration:") {
		t.Error("Missing duration information")
	}
}

// TestEndToEndWithLogRendererFailure verifies error handling in LogRenderer.
func TestEndToEndWithLogRendererFailure(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	testErr := errors.New("intentional failure")
	failing := target.New("Failing", "Will fail", func(_ context.Context) error {
		return testErr
	})

	dependent := target.New("Dependent", "Should be skipped", func(_ context.Context) error {
		t.Error("Dependent should not run")
		return nil
	}, failing)

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, dependent)
	if err == nil {
		t.Fatal("Expected error from pipeline")
	}

	output := buf.String()

	// Verify failure is logged
	if !strings.Contains(output, "[Failing] failed:") {
		t.Error("Missing failure log for Failing target")
	}

	if !strings.Contains(output, "intentional failure") {
		t.Error("Missing error message in log")
	}

	// Verify dependent was skipped
	if !strings.Contains(output, "[Dependent] skipped") {
		t.Error("Dependent should be marked as skipped")
	}

	// Verify pipeline failed
	if !strings.Contains(output, "[PIPELINE] failed:") {
		t.Error("Pipeline should be marked as failed")
	}
}

// TestEndToEndWithSimpleRenderer verifies the complete pipeline with SimpleRenderer.
func TestEndToEndWithSimpleRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	a := target.New("A", "First", func(_ context.Context) error {
		return nil
	})

	b := target.New("B", "Second", func(_ context.Context) error {
		return nil
	}, a)

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Should show start symbols
	if !strings.Contains(output, "▶ A") {
		t.Error("Expected '▶ A' for started target")
	}

	if !strings.Contains(output, "▶ B") {
		t.Error("Expected '▶ B' for started target")
	}

	// Should show completion symbols
	if !strings.Contains(output, "✓ A") {
		t.Error("Expected '✓ A' for completed target")
	}

	if !strings.Contains(output, "✓ B") {
		t.Error("Expected '✓ B' for completed target")
	}
}

// TestEndToEndWithSimpleRendererFailure verifies failure display in SimpleRenderer.
func TestEndToEndWithSimpleRendererFailure(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	failing := target.New("Failing", "Will fail", func(_ context.Context) error {
		return errors.New("test error")
	})

	skipped := target.New("Skipped", "Will skip", func(_ context.Context) error {
		t.Error("Should not run")
		return nil
	}, failing)

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, skipped)
	if err == nil {
		t.Fatal("Expected error")
	}

	output := buf.String()

	// Failed target should show ✗
	if !strings.Contains(output, "✗ Failing") {
		t.Error("Expected '✗ Failing' for failed target")
	}

	// Skipped target should show ○
	if !strings.Contains(output, "○ Skipped") {
		t.Error("Expected '○ Skipped' for skipped target")
	}
}

// TestEndToEndWithTUIRenderer verifies the complete pipeline with TUIRenderer.
func TestEndToEndWithTUIRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	a := target.New("A", "Test target", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Should show pipeline start
	if !strings.Contains(output, "CI Pipeline Starting") {
		t.Error("Expected pipeline start banner")
	}

	// Should show total targets
	if !strings.Contains(output, "Total Targets:") {
		t.Error("Expected total targets count")
	}

	// Should show pipeline completion
	if !strings.Contains(output, "CI Pipeline COMPLETED") {
		t.Error("Expected pipeline completion banner")
	}

	// Should show summary
	if !strings.Contains(output, "Summary:") {
		t.Error("Expected summary section")
	}

	// Should show target success
	if !strings.Contains(output, "✓ A") {
		t.Error("Expected '✓ A' in summary")
	}

	// Should show duration
	if !strings.Contains(output, "Duration:") {
		t.Error("Expected duration in completion banner")
	}
}

// TestEndToEndWithTUIRendererComplex verifies TUI with a complex DAG.
func TestEndToEndWithTUIRendererComplex(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	// Create a diamond DAG
	a := target.New("A", "Base", func(_ context.Context) error { return nil })
	b := target.New("B", "Left", func(_ context.Context) error { return nil }, a)
	c := target.New("C", "Right", func(_ context.Context) error { return nil }, a)
	d := target.New("D", "Final", func(_ context.Context) error { return nil }, b, c)

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, d)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Should show all targets in summary
	for _, name := range []string{"A", "B", "C", "D"} {
		if !strings.Contains(output, "✓ "+name) {
			t.Errorf("Expected '✓ %s' in summary", name)
		}
	}

	// Should show correct total
	if !strings.Contains(output, "Total Targets: 4") {
		t.Error("Expected 4 total targets")
	}
}

// TestEndToEndWithCollector verifies programmatic event access.
func TestEndToEndWithCollector(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Base", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	b := target.New("B", "Depends on A", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, a)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Verify event count (minimum expected)
	// PipelineStarted, TargetStarted(A), TargetCompleted(A), TargetStarted(B), TargetCompleted(B), PipelineCompleted
	if len(events) < 6 {
		t.Errorf("Expected at least 6 events, got %d", len(events))
	}

	// Verify event order and types
	eventTypes := []string{}
	for _, event := range events {
		switch event.(type) {
		case ci.PipelineStartedEvent:
			eventTypes = append(eventTypes, "PipelineStarted")
		case ci.TargetStartedEvent:
			eventTypes = append(eventTypes, "TargetStarted")
		case ci.TargetCompletedEvent:
			eventTypes = append(eventTypes, "TargetCompleted")
		case ci.PipelineCompletedEvent:
			eventTypes = append(eventTypes, "PipelineCompleted")
		}
	}

	// First event should be PipelineStarted
	if len(eventTypes) == 0 || eventTypes[0] != "PipelineStarted" {
		t.Errorf("First event should be PipelineStarted, got: %v", eventTypes)
	}

	// Last event should be PipelineCompleted
	if len(eventTypes) == 0 || eventTypes[len(eventTypes)-1] != "PipelineCompleted" {
		t.Errorf("Last event should be PipelineCompleted, got: %v", eventTypes)
	}

	// Verify target counts
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected A to start once, got %d", collector.TargetStartedCount("A"))
	}

	if collector.TargetStartedCount("B") != 1 {
		t.Errorf("Expected B to start once, got %d", collector.TargetStartedCount("B"))
	}

	if collector.TargetCompletedCount("A") != 1 {
		t.Errorf("Expected A to complete once, got %d", collector.TargetCompletedCount("A"))
	}

	if collector.TargetCompletedCount("B") != 1 {
		t.Errorf("Expected B to complete once, got %d", collector.TargetCompletedCount("B"))
	}
}

// TestEndToEndWithOutputCapture verifies output capture with Cmd targets.
func TestEndToEndWithOutputCapture(t *testing.T) {
	collector := render.NewCollector()

	echo := target.Cmd("Echo", "Echo test", "echo", "Hello, World!")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, echo)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify output was captured
	outputEvents := 0
	var capturedLine string
	for _, event := range collector.GetEvents() {
		if output, ok := event.(ci.TargetOutputEvent); ok {
			if output.TargetName() == "Echo" {
				outputEvents++
				capturedLine = output.Line
			}
		}
	}

	if outputEvents != 1 {
		t.Errorf("Expected 1 output event, got %d", outputEvents)
	}

	if capturedLine != "Hello, World!" {
		t.Errorf("Expected captured line 'Hello, World!', got %q", capturedLine)
	}
}

// TestEndToEndMixedRenderers verifies that different renderers produce different output.
func TestEndToEndMixedRenderers(t *testing.T) {
	tgt := target.New("Test", "Test target", func(_ context.Context) error {
		return nil
	})

	// Run with each renderer
	var logBuf, simpleBuf, tuiBuf bytes.Buffer

	logRenderer := render.NewLogRenderer(&logBuf)
	simpleRenderer := render.NewSimpleRenderer(&simpleBuf)
	tuiRenderer := render.NewTUIRenderer(&tuiBuf)

	for _, renderer := range []ci.EventHandler{logRenderer, simpleRenderer, tuiRenderer} {
		_, err := ci.RunTargetsWithHandler(context.Background(), renderer, tgt)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	// Verify each renderer produced output
	if logBuf.Len() == 0 {
		t.Error("LogRenderer produced no output")
	}

	if simpleBuf.Len() == 0 {
		t.Error("SimpleRenderer produced no output")
	}

	if tuiBuf.Len() == 0 {
		t.Error("TUIRenderer produced no output")
	}

	// Verify output is different (each renderer has unique style)
	logOut := logBuf.String()
	simpleOut := simpleBuf.String()
	tuiOut := tuiBuf.String()

	// LogRenderer should have timestamps
	if !strings.Contains(logOut, "[20") {
		t.Error("LogRenderer should include timestamps")
	}

	// SimpleRenderer should have symbols
	if !strings.Contains(simpleOut, "▶") && !strings.Contains(simpleOut, "✓") {
		t.Error("SimpleRenderer should include symbols")
	}

	// TUIRenderer should have boxes
	if !strings.Contains(tuiOut, "╔") && !strings.Contains(tuiOut, "═") {
		t.Error("TUIRenderer should include box drawing characters")
	}
}

// TestEndToEndPerformance ensures the pipeline completes in reasonable time.
func TestEndToEndPerformance(t *testing.T) {
	// Create 10 independent targets
	targets := make([]target.T, 10)
	for i := range 10 {
		name := string(rune('A' + i))
		targets[i] = target.New(name, "Test", func(_ context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	collector := render.NewCollector()

	start := time.Now()
	_, err := ci.RunTargetsWithHandler(context.Background(), collector, targets...)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// With concurrent execution, should complete in ~10ms, not 100ms
	// Allow some overhead for startup/teardown
	if elapsed > 100*time.Millisecond {
		t.Errorf("Pipeline took too long: %v (expected < 100ms)", elapsed)
	}

	// Verify all targets ran
	for i := range 10 {
		name := string(rune('A' + i))
		if collector.TargetCompletedCount(name) != 1 {
			t.Errorf("Target %s should have completed once", name)
		}
	}
}

// TestEndToEndStdoutStderrSeparation verifies stdout and stderr are distinguished.
func TestEndToEndStdoutStderrSeparation(t *testing.T) {
	collector := render.NewCollector()

	// Use shell to write to both stdout and stderr
	both := target.Cmd("Both", "Stdout and stderr",
		"sh", "-c", "echo 'to stdout'; echo 'to stderr' >&2")

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, both)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var stdoutCount, stderrCount int
	var stdoutLine, stderrLine string

	for _, event := range collector.GetEvents() {
		if output, ok := event.(ci.TargetOutputEvent); ok && output.TargetName() == "Both" {
			switch output.Stream {
			case ci.StreamStdout:
				stdoutCount++
				stdoutLine = output.Line
			case ci.StreamStderr:
				stderrCount++
				stderrLine = output.Line
			}
		}
	}

	if stdoutCount != 1 {
		t.Errorf("Expected 1 stdout event, got %d", stdoutCount)
	}

	if stderrCount != 1 {
		t.Errorf("Expected 1 stderr event, got %d", stderrCount)
	}

	if stdoutLine != "to stdout" {
		t.Errorf("Expected stdout line 'to stdout', got %q", stdoutLine)
	}

	if stderrLine != "to stderr" {
		t.Errorf("Expected stderr line 'to stderr', got %q", stderrLine)
	}
}

// TestEndToEndComplexDAGWithEvents verifies event ordering in complex DAG.
//
//nolint:gocognit // Test function complexity is acceptable for comprehensive testing
func TestEndToEndComplexDAGWithEvents(t *testing.T) {
	collector := render.NewCollector()

	// Create complex DAG: F -> (D, E); D -> (B, C); E -> C; B -> A; C -> A
	a := target.New("A", "Base", func(_ context.Context) error { return nil })
	b := target.New("B", "Level 2", func(_ context.Context) error { return nil }, a)
	c := target.New("C", "Level 2", func(_ context.Context) error { return nil }, a)
	d := target.New("D", "Level 3", func(_ context.Context) error { return nil }, b, c)
	e := target.New("E", "Level 3", func(_ context.Context) error { return nil }, c)
	f := target.New("F", "Final", func(_ context.Context) error { return nil }, d, e)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, f)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Verify all targets completed
	for _, name := range []string{"A", "B", "C", "D", "E", "F"} {
		if collector.TargetCompletedCount(name) != 1 {
			t.Errorf("Target %s should complete once, got %d",
				name, collector.TargetCompletedCount(name))
		}
	}

	// Verify dependency ordering: A must complete before B and C start
	var aCompleted, bStarted, cStarted int
	for i, event := range events {
		switch e := event.(type) {
		case ci.TargetCompletedEvent:
			if e.TargetName() == "A" {
				aCompleted = i
			}
		case ci.TargetStartedEvent:
			if e.TargetName() == "B" {
				bStarted = i
			}
			if e.TargetName() == "C" {
				cStarted = i
			}
		}
	}

	if bStarted > 0 && aCompleted >= bStarted {
		t.Error("A should complete before B starts")
	}

	if cStarted > 0 && aCompleted >= cStarted {
		t.Error("A should complete before C starts")
	}

	// Verify pipeline events bookend execution
	if _, ok := events[0].(ci.PipelineStartedEvent); !ok {
		t.Error("First event should be PipelineStartedEvent")
	}

	if _, ok := events[len(events)-1].(ci.PipelineCompletedEvent); !ok {
		t.Error("Last event should be PipelineCompletedEvent")
	}
}
