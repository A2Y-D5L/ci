package ci_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/a2y-d5l/ci"
	"github.com/a2y-d5l/ci/render"
	"github.com/a2y-d5l/ci/target"
)

func TestEventEmission(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Test", func(_ context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Verify we got at least 3 events: PipelineStarted, TargetStarted, TargetCompleted, PipelineCompleted
	if len(events) < 4 {
		t.Errorf("Expected at least 4 events, got %d", len(events))
	}

	// First event should be PipelineStarted
	if _, ok := events[0].(ci.PipelineStartedEvent); !ok {
		t.Errorf("First event should be PipelineStartedEvent, got %T", events[0])
	}

	// Should have TargetStartedEvent for A
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for A, got %d", collector.TargetStartedCount("A"))
	}

	// Should have TargetCompletedEvent for A
	if collector.TargetCompletedCount("A") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for A, got %d", collector.TargetCompletedCount("A"))
	}

	// Last event should be PipelineCompleted
	if _, ok := events[len(events)-1].(ci.PipelineCompletedEvent); !ok {
		t.Errorf("Last event should be PipelineCompletedEvent, got %T", events[len(events)-1])
	}
}

func TestEventOrder(t *testing.T) {
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

	// Verify event order:
	// 1. PipelineStarted
	// 2. TargetStarted (A)
	// 3. TargetCompleted (A)
	// 4. TargetStarted (B)
	// 5. TargetCompleted (B)
	// 6. PipelineCompleted

	expectedOrder := []string{
		"PipelineStarted",
		"TargetStarted:A",
		"TargetCompleted:A",
		"TargetStarted:B",
		"TargetCompleted:B",
		"PipelineCompleted",
	}

	if len(events) != len(expectedOrder) {
		t.Fatalf("Expected %d events, got %d", len(expectedOrder), len(events))
	}

	for i, event := range events {
		var eventName string
		switch e := event.(type) {
		case ci.PipelineStartedEvent:
			eventName = "PipelineStarted"
		case ci.TargetStartedEvent:
			eventName = fmt.Sprintf("TargetStarted:%s", e.TargetName())
		case ci.TargetCompletedEvent:
			eventName = fmt.Sprintf("TargetCompleted:%s", e.TargetName())
		case ci.PipelineCompletedEvent:
			eventName = "PipelineCompleted"
		default:
			eventName = fmt.Sprintf("Unknown:%T", e)
		}

		if eventName != expectedOrder[i] {
			t.Errorf("Event %d: expected %s, got %s", i, expectedOrder[i], eventName)
		}
	}
}

func TestEventTimestamps(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Test", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	start := time.Now()
	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	end := time.Now()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// All events should have timestamps within the execution window
	for i, event := range events {
		ts := event.Timestamp()
		if ts.Before(start) {
			t.Errorf("Event %d timestamp (%v) is before start (%v)", i, ts, start)
		}
		if ts.After(end) {
			t.Errorf("Event %d timestamp (%v) is after end (%v)", i, ts, end)
		}
	}

	// Events should be in chronological order
	for i := 1; i < len(events); i++ {
		prev := events[i-1].Timestamp()
		curr := events[i].Timestamp()
		if curr.Before(prev) {
			t.Errorf("Event %d timestamp (%v) is before event %d timestamp (%v)",
				i, curr, i-1, prev)
		}
	}
}

func TestPipelineStartedEvent(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "A", func(_ context.Context) error { return nil })
	b := target.New("B", "B", func(_ context.Context) error { return nil }, a)
	c := target.New("C", "C", func(_ context.Context) error { return nil }, a)
	d := target.New("D", "D", func(_ context.Context) error { return nil }, b, c)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, d)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()
	if len(events) == 0 {
		t.Fatal("No events captured")
	}

	pipelineStarted, ok := events[0].(ci.PipelineStartedEvent)
	if !ok {
		t.Fatalf("First event is not PipelineStartedEvent: %T", events[0])
	}

	// Should report 4 total targets
	if pipelineStarted.TotalTargets != 4 {
		t.Errorf("Expected TotalTargets=4, got %d", pipelineStarted.TotalTargets)
	}

	// TargetName should be empty for pipeline events
	if pipelineStarted.TargetName() != "" {
		t.Errorf("PipelineStartedEvent should have empty TargetName, got %q", pipelineStarted.TargetName())
	}
}

func TestTargetCompletedEventSuccess(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Test", func(_ context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Find the TargetCompletedEvent
	var completedEvent *ci.TargetCompletedEvent
	for _, e := range events {
		if completed, ok := e.(ci.TargetCompletedEvent); ok && completed.TargetName() == "A" {
			completedEvent = &completed
			break
		}
	}

	if completedEvent == nil {
		t.Fatal("No TargetCompletedEvent found for A")
	}

	// Should not be skipped
	if completedEvent.Skipped {
		t.Error("Target should not be skipped")
	}

	// Should have no error
	if completedEvent.Error != nil {
		t.Errorf("Expected no error, got %v", completedEvent.Error)
	}

	// Duration should be >= 50ms
	if completedEvent.Duration < 50*time.Millisecond {
		t.Errorf("Expected duration >= 50ms, got %v", completedEvent.Duration)
	}

	// Target reference should be set
	if completedEvent.Target == nil {
		t.Error("Target reference should not be nil")
	}
}

func TestTargetCompletedEventFailure(t *testing.T) {
	collector := render.NewCollector()

	testErr := errors.New("test error")
	a := target.New("A", "Test", func(_ context.Context) error {
		return testErr
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	if err == nil {
		t.Fatal("Expected error")
	}

	events := collector.GetEvents()

	// Find the TargetCompletedEvent
	var completedEvent *ci.TargetCompletedEvent
	for _, e := range events {
		if completed, ok := e.(ci.TargetCompletedEvent); ok && completed.TargetName() == "A" {
			completedEvent = &completed
			break
		}
	}

	if completedEvent == nil {
		t.Fatal("No TargetCompletedEvent found for A")
	}

	// Should not be skipped
	if completedEvent.Skipped {
		t.Error("Failed target should not be marked as skipped")
	}

	// Should have error
	if completedEvent.Error == nil {
		t.Error("Expected error, got nil")
	}

	if !errors.Is(completedEvent.Error, testErr) {
		t.Errorf("Expected error %v, got %v", testErr, completedEvent.Error)
	}
}

func TestTargetCompletedEventSkipped(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Failing", func(_ context.Context) error {
		return errors.New("A failed")
	})

	b := target.New("B", "Should be skipped", func(_ context.Context) error {
		t.Error("B should not run")
		return nil
	}, a)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, b)
	if err == nil {
		t.Fatal("Expected error")
	}

	events := collector.GetEvents()

	// Find the TargetCompletedEvent for B
	var completedEvent *ci.TargetCompletedEvent
	for _, e := range events {
		if completed, ok := e.(ci.TargetCompletedEvent); ok && completed.TargetName() == "B" {
			completedEvent = &completed
			break
		}
	}

	if completedEvent == nil {
		t.Fatal("No TargetCompletedEvent found for B")
	}

	// Should be skipped
	if !completedEvent.Skipped {
		t.Error("Target B should be skipped")
	}

	// Should have no error (skipped, not failed)
	if completedEvent.Error != nil {
		t.Errorf("Skipped target should have no error, got %v", completedEvent.Error)
	}

	// Duration should be 0 for skipped targets
	if completedEvent.Duration != 0 {
		t.Errorf("Skipped target should have duration=0, got %v", completedEvent.Duration)
	}
}

func TestPipelineCompletedEvent(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Test", func(_ context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	start := time.Now()
	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Last event should be PipelineCompletedEvent
	pipelineCompleted, ok := events[len(events)-1].(ci.PipelineCompletedEvent)
	if !ok {
		t.Fatalf("Last event is not PipelineCompletedEvent: %T", events[len(events)-1])
	}

	// Should have no error
	if pipelineCompleted.Error != nil {
		t.Errorf("Expected no error, got %v", pipelineCompleted.Error)
	}

	// Duration should be reasonable (>= 50ms, <= actual elapsed time)
	if pipelineCompleted.Duration < 50*time.Millisecond {
		t.Errorf("Expected duration >= 50ms, got %v", pipelineCompleted.Duration)
	}

	if pipelineCompleted.Duration > elapsed+10*time.Millisecond {
		t.Errorf("Duration (%v) exceeds actual elapsed time (%v)", pipelineCompleted.Duration, elapsed)
	}

	// Results should be populated
	if len(pipelineCompleted.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(pipelineCompleted.Results))
	}

	// TargetName should be empty for pipeline events
	if pipelineCompleted.TargetName() != "" {
		t.Errorf("PipelineCompletedEvent should have empty TargetName, got %q", pipelineCompleted.TargetName())
	}
}

func TestPipelineCompletedEventWithError(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Failing", func(_ context.Context) error {
		return errors.New("test error")
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	if err == nil {
		t.Fatal("Expected error")
	}

	events := collector.GetEvents()

	// Last event should be PipelineCompletedEvent
	pipelineCompleted, ok := events[len(events)-1].(ci.PipelineCompletedEvent)
	if !ok {
		t.Fatalf("Last event is not PipelineCompletedEvent: %T", events[len(events)-1])
	}

	// Should have error
	if pipelineCompleted.Error == nil {
		t.Error("Expected error, got nil")
	}

	// Error should match the returned error
	if !errors.Is(pipelineCompleted.Error, err) {
		t.Errorf("PipelineCompletedEvent error (%v) should match returned error (%v)",
			pipelineCompleted.Error, err)
	}
}

func TestEventHandlerFunc(t *testing.T) {
	var events []ci.Event
	var mu sync.Mutex

	handler := ci.EventHandlerFunc(func(event ci.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})

	a := target.New("A", "Test", func(_ context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), handler, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) < 4 {
		t.Errorf("Expected at least 4 events, got %d", len(events))
	}
}

func TestNoEventHandlerBackwardCompatibility(t *testing.T) {
	// Calling RunTargets (without handler) should still work
	a := target.New("A", "Test", func(_ context.Context) error {
		return nil
	})

	results, err := ci.RunTargets(context.Background(), a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if _, ok := results["A"]; !ok {
		t.Error("Expected result for target A")
	}
}

func TestConcurrentEventHandling(t *testing.T) {
	collector := render.NewCollector()

	// Create multiple independent targets that run concurrently
	a := target.New("A", "Independent 1", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	b := target.New("B", "Independent 1", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	c := target.New("C", "Independent 1", func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a, b, c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Should have events for all targets
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for A, got %d", collector.TargetStartedCount("A"))
	}
	if collector.TargetStartedCount("B") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for B, got %d", collector.TargetStartedCount("B"))
	}
	if collector.TargetStartedCount("C") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for C, got %d", collector.TargetStartedCount("C"))
	}

	// All events should have valid timestamps
	for i, event := range events {
		if event.Timestamp().IsZero() {
			t.Errorf("Event %d has zero timestamp", i)
		}
	}
}

func TestEventTargetReferences(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Test target", func(_ context.Context) error {
		return nil
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	events := collector.GetEvents()

	// Find target-specific events and verify they have proper target references
	for _, event := range events {
		switch e := event.(type) {
		case ci.TargetStartedEvent:
			if e.Target == nil {
				t.Error("TargetStartedEvent should have Target reference")
			}
			if e.Target != nil && e.Target.Name() != "A" {
				t.Errorf("Expected target name A, got %s", e.Target.Name())
			}

		case ci.TargetCompletedEvent:
			if e.Target == nil {
				t.Error("TargetCompletedEvent should have Target reference")
			}
			if e.Target != nil && e.Target.Name() != "A" {
				t.Errorf("Expected target name A, got %s", e.Target.Name())
			}
		}
	}
}

//nolint:gocognit // Test function complexity is acceptable for comprehensive testing
func TestDiamondDAGEvents(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Base", func(_ context.Context) error {
		return nil
	})

	b := target.New("B", "Left", func(_ context.Context) error {
		return nil
	}, a)

	c := target.New("C", "Right", func(_ context.Context) error {
		return nil
	}, a)

	d := target.New("D", "Final", func(_ context.Context) error {
		return nil
	}, b, c)

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, d)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have events for all 4 targets
	if collector.TargetStartedCount("A") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for A, got %d", collector.TargetStartedCount("A"))
	}
	if collector.TargetStartedCount("B") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for B, got %d", collector.TargetStartedCount("B"))
	}
	if collector.TargetStartedCount("C") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for C, got %d", collector.TargetStartedCount("C"))
	}
	if collector.TargetStartedCount("D") != 1 {
		t.Errorf("Expected 1 TargetStartedEvent for D, got %d", collector.TargetStartedCount("D"))
	}

	// All targets should complete
	if collector.TargetCompletedCount("A") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for A, got %d", collector.TargetCompletedCount("A"))
	}
	if collector.TargetCompletedCount("B") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for B, got %d", collector.TargetCompletedCount("B"))
	}
	if collector.TargetCompletedCount("C") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for C, got %d", collector.TargetCompletedCount("C"))
	}
	if collector.TargetCompletedCount("D") != 1 {
		t.Errorf("Expected 1 TargetCompletedEvent for D, got %d", collector.TargetCompletedCount("D"))
	}

	events := collector.GetEvents()

	// Verify A completes before B and C start
	var aCompletedIdx, bStartedIdx, cStartedIdx int
	for i, event := range events {
		switch e := event.(type) {
		case ci.TargetCompletedEvent:
			if e.TargetName() == "A" {
				aCompletedIdx = i
			}
		case ci.TargetStartedEvent:
			if e.TargetName() == "B" {
				bStartedIdx = i
			}
			if e.TargetName() == "C" {
				cStartedIdx = i
			}
		}
	}

	if aCompletedIdx >= bStartedIdx && bStartedIdx > 0 {
		t.Error("A should complete before B starts")
	}
	if aCompletedIdx >= cStartedIdx && cStartedIdx > 0 {
		t.Error("A should complete before C starts")
	}
}

func TestMultipleFailuresInEvents(t *testing.T) {
	collector := render.NewCollector()

	a := target.New("A", "Fails", func(_ context.Context) error {
		return errors.New("error A")
	})

	b := target.New("B", "Also fails", func(_ context.Context) error {
		return errors.New("error B")
	})

	_, err := ci.RunTargetsWithHandler(context.Background(), collector, a, b)
	if err == nil {
		t.Fatal("Expected error")
	}

	events := collector.GetEvents()

	// Find both completed events
	var aCompleted, bCompleted *ci.TargetCompletedEvent
	for _, e := range events {
		if completed, ok := e.(ci.TargetCompletedEvent); ok {
			switch completed.TargetName() {
			case "A":
				aCompleted = &completed
			case "B":
				bCompleted = &completed
			}
		}
	}

	if aCompleted == nil {
		t.Error("No TargetCompletedEvent for A")
	} else if aCompleted.Error == nil {
		t.Error("A should have error")
	}

	if bCompleted == nil {
		t.Error("No TargetCompletedEvent for B")
	} else if bCompleted.Error == nil {
		t.Error("B should have error")
	}

	// Pipeline completed event should have error
	pipelineCompleted, ok := events[len(events)-1].(ci.PipelineCompletedEvent)
	if !ok {
		t.Fatal("Last event should be PipelineCompletedEvent")
	}

	if pipelineCompleted.Error == nil {
		t.Error("PipelineCompletedEvent should have error")
	}

	// Error message should contain both errors (alphabetically ordered)
	errMsg := pipelineCompleted.Error.Error()
	posA := strings.Index(errMsg, "A")
	posB := strings.Index(errMsg, "B")

	if posA == -1 || posB == -1 {
		t.Errorf("Error message should contain both A and B: %s", errMsg)
	}

	if posA >= posB {
		t.Errorf("Errors should be in alphabetical order: %s", errMsg)
	}
}
