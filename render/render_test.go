package render_test

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

// testTarget creates a simple target for testing.
func testTarget(name string) target.T {
	return target.New(name, "Test target", func(ctx context.Context) error {
		return nil
	})
}

func TestLogRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	now := time.Now()

	// Simulate events
	renderer.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 1,
	})

	renderer.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    nil,
		Skipped:  false,
	})

	renderer.HandleEvent(ci.PipelineCompletedEvent{
		Time:     now,
		Duration: 150 * time.Millisecond,
		Results:  make(map[string]ci.Result),
		Error:    nil,
	})

	output := buf.String()

	// Verify output contains expected elements
	if !strings.Contains(output, "[PIPELINE]") {
		t.Error("Expected [PIPELINE] in output")
	}

	if !strings.Contains(output, "[A]") {
		t.Error("Expected [A] in output")
	}

	if !strings.Contains(output, "Starting with 1 targets") {
		t.Error("Expected 'Starting with 1 targets' in output")
	}

	if !strings.Contains(output, "Started") {
		t.Error("Expected 'Started' in output")
	}

	if !strings.Contains(output, "completed") {
		t.Error("Expected 'completed' in output")
	}
}

func TestLogRendererWithFailure(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	now := time.Now()
	testErr := errors.New("test error")

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    testErr,
		Skipped:  false,
	})

	output := buf.String()

	if !strings.Contains(output, "failed: test error") {
		t.Error("Expected 'failed: test error' in output")
	}
}

func TestLogRendererWithSkipped(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	now := time.Now()

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 0,
		Error:    nil,
		Skipped:  true,
	})

	output := buf.String()

	if !strings.Contains(output, "skipped") {
		t.Error("Expected 'skipped' in output")
	}
}

func TestLogRendererWithOutput(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	now := time.Now()

	renderer.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStdout,
		Line:   "test output line",
	})

	renderer.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStderr,
		Line:   "test error line",
	})

	output := buf.String()

	if !strings.Contains(output, "[stdout] test output line") {
		t.Error("Expected stdout output in log")
	}

	if !strings.Contains(output, "[stderr] test error line") {
		t.Error("Expected stderr output in log")
	}
}

func TestSimpleRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	now := time.Now()

	// Test target started
	renderer.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	output := buf.String()
	if !strings.Contains(output, "▶ A") {
		t.Error("Expected '▶ A' in output")
	}

	// Reset buffer
	buf.Reset()

	// Test target completed successfully
	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("B"),
		Duration: 100 * time.Millisecond,
		Error:    nil,
		Skipped:  false,
	})

	output = buf.String()
	if !strings.Contains(output, "✓ B") {
		t.Error("Expected '✓ B' in output")
	}
}

func TestSimpleRendererWithFailure(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	now := time.Now()
	testErr := errors.New("test error")

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    testErr,
		Skipped:  false,
	})

	output := buf.String()
	if !strings.Contains(output, "✗ A") {
		t.Error("Expected '✗ A' in output for failed target")
	}
}

func TestSimpleRendererWithSkipped(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	now := time.Now()

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 0,
		Error:    nil,
		Skipped:  true,
	})

	output := buf.String()
	if !strings.Contains(output, "○ A") {
		t.Error("Expected '○ A' in output for skipped target")
	}
}

func TestCollector(t *testing.T) {
	collector := render.NewCollector()

	now := time.Now()

	// Add various events
	collector.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 2,
	})

	collector.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	collector.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    nil,
		Skipped:  false,
	})

	collector.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("B"),
	})

	collector.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("B"),
		Duration: 200 * time.Millisecond,
		Error:    nil,
		Skipped:  false,
	})

	collector.HandleEvent(ci.PipelineCompletedEvent{
		Time:     now,
		Duration: 300 * time.Millisecond,
		Results:  make(map[string]ci.Result),
		Error:    nil,
	})

	// Verify event count
	events := collector.GetEvents()
	if len(events) != 6 {
		t.Errorf("Expected 6 events, got %d", len(events))
	}

	// Verify target started counts
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for A, got %d", collector.TargetStartedCount("A"))
	}

	if collector.TargetStartedCount("B") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for B, got %d", collector.TargetStartedCount("B"))
	}

	// Verify target completed counts
	if collector.TargetCompletedCount("A") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for A, got %d", collector.TargetCompletedCount("A"))
	}

	if collector.TargetCompletedCount("B") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for B, got %d", collector.TargetCompletedCount("B"))
	}
}

func TestCollectorOutputCount(t *testing.T) {
	collector := render.NewCollector()

	now := time.Now()

	// Add output events
	collector.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStdout,
		Line:   "line 1",
	})

	collector.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStdout,
		Line:   "line 2",
	})

	collector.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("B"),
		Stream: ci.StreamStdout,
		Line:   "line 1",
	})

	// Verify output counts
	if collector.TargetOutputCount("A") != 2 {
		t.Errorf("Expected 2 output events for A, got %d", collector.TargetOutputCount("A"))
	}

	if collector.TargetOutputCount("B") != 1 {
		t.Errorf("Expected 1 output event for B, got %d", collector.TargetOutputCount("B"))
	}
}

func TestTUIRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	now := time.Now()

	// Test pipeline start
	renderer.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 1,
	})

	output := buf.String()
	if !strings.Contains(output, "CI Pipeline Starting") {
		t.Error("Expected 'CI Pipeline Starting' in output")
	}

	if !strings.Contains(output, "Total Targets: 1") {
		t.Error("Expected 'Total Targets: 1' in output")
	}

	// Reset buffer
	buf.Reset()

	// Test target started and running
	renderer.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	output = buf.String()
	if !strings.Contains(output, "A") {
		t.Error("Expected target name 'A' in output")
	}

	// Reset buffer
	buf.Reset()

	// Test target completed
	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    nil,
		Skipped:  false,
	})

	// Test pipeline complete
	buf.Reset()
	renderer.HandleEvent(ci.PipelineCompletedEvent{
		Time:     now,
		Duration: 150 * time.Millisecond,
		Results:  make(map[string]ci.Result),
		Error:    nil,
	})

	output = buf.String()
	if !strings.Contains(output, "CI Pipeline COMPLETED") {
		t.Error("Expected 'CI Pipeline COMPLETED' in output")
	}

	if !strings.Contains(output, "Summary:") {
		t.Error("Expected 'Summary:' in output")
	}

	if !strings.Contains(output, "✓ A") {
		t.Error("Expected '✓ A' in summary")
	}
}

func TestTUIRendererWithFailure(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	now := time.Now()
	testErr := errors.New("test error")

	renderer.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 1,
	})

	renderer.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 100 * time.Millisecond,
		Error:    testErr,
		Skipped:  false,
	})

	buf.Reset()
	renderer.HandleEvent(ci.PipelineCompletedEvent{
		Time:     now,
		Duration: 150 * time.Millisecond,
		Results:  make(map[string]ci.Result),
		Error:    testErr,
	})

	output := buf.String()
	if !strings.Contains(output, "CI Pipeline FAILED") {
		t.Error("Expected 'CI Pipeline FAILED' in output")
	}

	if !strings.Contains(output, "✗ A") {
		t.Error("Expected '✗ A' in summary for failed target")
	}
}

func TestTUIRendererWithSkipped(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	now := time.Now()

	renderer.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 1,
	})

	renderer.HandleEvent(ci.TargetCompletedEvent{
		Time:     now,
		Target:   testTarget("A"),
		Duration: 0,
		Error:    nil,
		Skipped:  true,
	})

	buf.Reset()
	renderer.HandleEvent(ci.PipelineCompletedEvent{
		Time:     now,
		Duration: 150 * time.Millisecond,
		Results:  make(map[string]ci.Result),
		Error:    nil,
	})

	output := buf.String()
	if !strings.Contains(output, "○ A") {
		t.Error("Expected '○ A' in summary for skipped target")
	}
}

func TestTUIRendererWithOutput(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	now := time.Now()

	renderer.HandleEvent(ci.PipelineStartedEvent{
		Time:         now,
		TotalTargets: 1,
	})

	renderer.HandleEvent(ci.TargetStartedEvent{
		Time:   now,
		Target: testTarget("A"),
	})

	renderer.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStdout,
		Line:   "line 1 from A",
	})

	renderer.HandleEvent(ci.TargetOutputEvent{
		Time:   now,
		Target: testTarget("A"),
		Stream: ci.StreamStdout,
		Line:   "line 2 from A",
	})

	output := buf.String()

	// Verify output appears in the display
	if !strings.Contains(output, "line 1 from A") {
		t.Error("Expected 'line 1 from A' in output")
	}

	if !strings.Contains(output, "line 2 from A") {
		t.Error("Expected 'line 2 from A' in output")
	}
}

// Integration tests with actual CI execution

func TestEndToEndWithLogRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewLogRenderer(&buf)

	a := target.New("A", "Base", func(ctx context.Context) error {
		return nil
	})

	b := target.New("B", "Depends on A", func(ctx context.Context) error {
		return nil
	}, a)

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Verify structured log output
	lines := strings.Split(output, "\n")

	// Should have timestamps
	hasTimestamps := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[20") {
			hasTimestamps = true
			break
		}
	}

	if !hasTimestamps {
		t.Error("Expected timestamps in log output")
	}

	// Should mention both targets
	if !strings.Contains(output, "[A]") {
		t.Error("Expected [A] in output")
	}

	if !strings.Contains(output, "[B]") {
		t.Error("Expected [B] in output")
	}
}

func TestEndToEndWithSimpleRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewSimpleRenderer(&buf)

	a := target.New("A", "Test", func(ctx context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Should show target started and completed
	if !strings.Contains(output, "▶ A") {
		t.Error("Expected '▶ A' in output")
	}

	if !strings.Contains(output, "✓ A") {
		t.Error("Expected '✓ A' in output")
	}
}

func TestEndToEndWithTUIRenderer(t *testing.T) {
	var buf bytes.Buffer
	renderer := render.NewTUIRenderer(&buf)

	a := target.New("A", "Test", func(ctx context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), renderer, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buf.String()

	// Should show pipeline start and complete
	if !strings.Contains(output, "CI Pipeline Starting") {
		t.Error("Expected 'CI Pipeline Starting' in output")
	}

	if !strings.Contains(output, "CI Pipeline COMPLETED") {
		t.Error("Expected 'CI Pipeline COMPLETED' in output")
	}

	if !strings.Contains(output, "Summary:") {
		t.Error("Expected 'Summary:' in output")
	}
}

func TestEndToEndWithCollector(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Base", func(ctx context.Context) error {
		return nil
	})

	b := target.New("B", "Depends on A", func(ctx context.Context) error {
		return nil
	}, a)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, b)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Should have events for both targets
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for A, got %d", collector.TargetStartedCount("A"))
	}

	if collector.TargetStartedCount("B") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for B, got %d", collector.TargetStartedCount("B"))
	}

	// Should have pipeline events
	hasPipelineStarted := false
	hasPipelineCompleted := false
	for _, e := range events {
		if _, ok := e.(ci.PipelineStartedEvent); ok {
			hasPipelineStarted = true
		}
		if _, ok := e.(ci.PipelineCompletedEvent); ok {
			hasPipelineCompleted = true
		}
	}

	if !hasPipelineStarted {
		t.Error("Expected PipelineStartedEvent")
	}

	if !hasPipelineCompleted {
		t.Error("Expected PipelineCompletedEvent")
	}
}
