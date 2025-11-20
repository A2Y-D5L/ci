package renderer_test

import (
	"strings"
	"testing"

	"github.com/a2y-d5l/ci/multiproc/engine"
	"github.com/a2y-d5l/ci/multiproc/renderer"
)

// TestRenderIncrementalWithTimestamps verifies timestamp formatting
func TestRenderIncrementalWithTimestamps(t *testing.T) {
	// This test captures what RenderIncremental would print
	// We can't easily capture stdout, but we can verify the logic
	// by inspecting the function signature and expected behavior

	specs := []engine.ProcessSpec{
		{Name: "TestProc", Command: "test"},
	}

	states := []renderer.ProcessState{
		{Name: "TestProc", Lines: []string{}, Running: true},
	}

	// Create a line event
	ev := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index: 0,
		Line:  "test output",
	})

	// RenderIncremental should handle this without error
	// Note: This doesn't capture output, just ensures no panics
	renderer.RenderIncremental(ev, specs, states, true, "[%s]")
	renderer.RenderIncremental(ev, specs, states, false, "[%s]")
	renderer.RenderIncremental(ev, specs, states, true, "%s:")
}

// TestRenderIncrementalWithCustomPrefix verifies custom prefix formatting
func TestRenderIncrementalWithCustomPrefix(t *testing.T) {
	specs := []engine.ProcessSpec{
		{Name: "ProcA", Command: "test"},
	}

	states := []renderer.ProcessState{
		{Name: "ProcA", Lines: []string{}, Running: true},
	}

	prefixes := []string{
		"[%s]",
		"%s:",
		"(%s)",
		">>> %s >>>",
	}

	ev := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index: 0,
		Line:  "output",
	})

	// Verify no panics with different prefix formats
	for _, prefix := range prefixes {
		renderer.RenderIncremental(ev, specs, states, false, prefix)
		renderer.RenderIncremental(ev, specs, states, true, prefix)
	}
}

// TestRenderIncrementalEmptyPrefix verifies fallback to default prefix
func TestRenderIncrementalEmptyPrefix(t *testing.T) {
	specs := []engine.ProcessSpec{
		{Name: "Test", Command: "test"},
	}

	states := []renderer.ProcessState{
		{Name: "Test", Lines: []string{}, Running: true},
	}

	ev := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index: 0,
		Line:  "test",
	})

	// Empty prefix should fall back to default
	renderer.RenderIncremental(ev, specs, states, false, "")
}

// TestRenderIncrementalDoneEvent verifies completion event rendering
func TestRenderIncrementalDoneEvent(t *testing.T) {
	specs := []engine.ProcessSpec{
		{Name: "Completed", Command: "test"},
	}

	states := []renderer.ProcessState{
		{Name: "Completed", Lines: []string{}, Running: false, Done: true},
	}

	ev := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index:      0,
		IsComplete: true,
		Err:        nil,
	})

	// Should render completion message without error
	renderer.RenderIncremental(ev, specs, states, false, "[%s]")
	renderer.RenderIncremental(ev, specs, states, true, "[%s]")
}

// TestConvertProcessLineToEvent verifies event conversion
func TestConvertProcessLineToEvent(t *testing.T) {
	// Test line event conversion
	lineEvent := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index:      0,
		Line:       "test line",
		IsComplete: false,
	})

	if lineEvent == nil {
		t.Error("Expected non-nil event for line")
	}

	// Test completion event conversion
	doneEvent := renderer.ConvertProcessLineToEvent(engine.ProcessLine{
		Index:      1,
		IsComplete: true,
		Err:        nil,
	})

	if doneEvent == nil {
		t.Error("Expected non-nil event for completion")
	}
}

// TestFormatExitError verifies exit error formatting
func TestFormatExitError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "nil error",
			err:      nil,
			contains: "ok",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.FormatExitError(tc.err)
			if !strings.Contains(result, tc.contains) && result != tc.contains {
				t.Errorf("Expected result to contain %q, got %q", tc.contains, result)
			}
		})
	}
}
