package engine

import (
	"context"
	"errors"
	"fmt"

	// "log"
	"runtime"
	"sync"
	"time"
)

// Logger is a minimal logging function used by the Executor.
//
// It mirrors log.Printf.
type Logger func(format string, args ...any)

// ExecutorOption configures an Executor at construction time.
type ExecutorOption func(*Executor)

// WithMaxWorkers sets the maximum number of concurrent workers.
//
// A value <= 0 will be normalized to runtime.NumCPU().
func WithMaxWorkers(n int) ExecutorOption {
	return func(e *Executor) {
		e.maxWorkers = n
	}
}

// // WithLogger customizes logging for the Executor.
// //
// // If not set, log.Printf is used.
// func WithLogger(l Logger) ExecutorOption {
// 	return func(e *Executor) {
// 		e.logf = l
// 	}
// }

// WithFailFast stops scheduling new work after the first failure.
//
// Already-running targets are allowed to complete; all remaining pending
// targets that depend (directly or transitively) on a failed target are
// marked as skipped.
func WithFailFast() ExecutorOption {
	return func(e *Executor) {
		e.failFast = true
	}
}

// WithEventSink attaches a custom sink to receive all events.
//
// This allows callers to plug in TUIs, JSON loggers, or test recorders.
func WithEventSink(sink EventHandler) ExecutorOption {
	return func(e *Executor) {
		e.sink = sink
	}
}

// RunSummary is the top-level result of an Executor run.
type RunSummary struct {
	// Results contains a Result for every *reachable* target.
	Results map[string]Result

	// Failed is true if at least one reachable target failed.
	Failed bool
}

// Executor is a per-run execution engine for a compiled Plan.
//
// It is NOT safe to copy. Use a new Executor per Run.
type Executor struct {
	plan *Plan

	maxWorkers int
	// logf       Logger
	failFast bool

	// event and result streams for observers
	events  chan Event
	results chan Result
	sink    EventHandler

	// internal per-run state
	mu       sync.Mutex // protects firstErr only
	firstErr error
}

// NewExecutor creates a new Executor for the given Plan.
//
// It is cheap compared to building the Plan.
func NewExecutor(plan *Plan, opts ...ExecutorOption) *Executor {
	e := &Executor{
		plan:       plan,
		maxWorkers: runtime.NumCPU(),
		// logf:       log.Printf,
		// small default buffers; callers should consume promptly, but
		// emit() is non-blocking to avoid deadlocks.
		events:  make(chan Event, 128),
		results: make(chan Result, 128),
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.maxWorkers <= 0 {
		e.maxWorkers = runtime.NumCPU()
	}
	return e
}

// Events returns a read-only channel of target lifecycle events.
//
// The channel is closed when Run completes.
func (e *Executor) Events() <-chan Event {
	return e.events
}

// Results returns a read-only channel of Result snapshots.
//
// It is a convenience view over Completed/Skipped events. The channel is
// closed when Run completes.
func (e *Executor) Results() <-chan Result {
	return e.results
}

// Run executes the plan for the given root targets.
//
// roots is the set of target names whose transitive dependencies will be
// considered reachable. If roots is empty, the entire plan is executed.
//
// The returned RunSummary only includes results for reachable targets.
//
// NOTE: An Executor instance is intended for a single Run call.
func (e *Executor) Run(ctx context.Context, roots ...string) (RunSummary, error) {
	if ctx == nil {
		return RunSummary{}, errors.New("executor run: nil context")
	}

	reachable, reachableCount, err := e.computeReachable(roots)
	if err != nil {
		return RunSummary{}, err
	}
	if reachableCount == 0 {
		return RunSummary{}, errors.New("executor run: no reachable targets")
	}

	n := len(e.plan.targets)

	remainingDeps := make([]int, n)
	status := make([]targetStatus, n)
	results := make([]Result, n)

	for i := range n {
		if !reachable[i] {
			continue
		}

		// Copy indegree for reachable nodes only.
		remainingDeps[i] = len(e.plan.deps[i])

		status[i] = statusPending
		results[i] = Result{Name: e.plan.targets[i].Name}
	}

	readyCh := make(chan int)
	// Buffered so workers can complete sending even if Run stops reading.
	doneCh := make(chan taskDone, reachableCount)

	var wg sync.WaitGroup

	workerCount := min(e.maxWorkers, reachableCount)
	if workerCount <= 0 {
		workerCount = 1
	}

	for range workerCount {
		wg.Go(func() {
			e.worker(ctx, readyCh, doneCh)
		})
	}

	// Seed initial ready queue (nodes with remainingDeps == 0).
	for i := range n {
		if !reachable[i] {
			continue
		}
		if remainingDeps[i] == 0 {
			results[i].StartedAt = time.Now()
			status[i] = statusRunning

			e.emit(Event{
				Type:   EventTargetStarted,
				Time:   results[i].StartedAt,
				Target: e.plan.targets[i],
				Result: nil,
			})

			// We don't check ctx here; workers will see ctx.Done() and
			// short-circuit if cancellation has already occurred.
			readyCh <- i
		}
	}

	completed := 0
	e.firstErr = nil
	done := false

	for !done && completed < reachableCount {
		select {
		case <-ctx.Done():
			// Mark pending nodes as skipped.
			for i := range n {
				if !reachable[i] {
					continue
				}
				if status[i] == statusPending {
					status[i] = statusSkipped
					results[i].Skipped = true
					results[i].CompletedAt = time.Now()
					completed++

					e.emit(Event{
						Type:   EventTargetSkipped,
						Time:   results[i].CompletedAt,
						Target: e.plan.targets[i],
						Result: &results[i],
					})
				}
			}
			done = true

		case td := <-doneCh:
			i := td.index
			t := e.plan.targets[i]

			if td.err != nil && !errors.Is(td.err, context.Canceled) {
				// Hard failure from target.
				// e.logf("target %s failed: %v", t.Name, td.err)
				status[i] = statusFailed
				results[i].Err = td.err
				results[i].CompletedAt = time.Now()
				completed++

				e.setFirstErr(td.err)

				// Emit completed event with error.
				e.emit(Event{
					Type:   EventTargetCompleted,
					Time:   results[i].CompletedAt,
					Target: t,
					Result: &results[i],
				})

				// Mark dependents as skipped if they haven't started.
				for _, depIdx := range e.plan.dependents[i] {
					if !reachable[depIdx] {
						continue
					}
					if status[depIdx] == statusPending {
						status[depIdx] = statusSkipped
						results[depIdx].Skipped = true
						results[depIdx].CompletedAt = time.Now()
						completed++

						e.emit(Event{
							Type:   EventTargetSkipped,
							Time:   results[depIdx].CompletedAt,
							Target: e.plan.targets[depIdx],
							Result: &results[depIdx],
						})
					}
				}

				if e.failFast {
					// Don't schedule any new work after failure.
					// Already running tasks continue.
					continue
				}

				continue
			}

			if status[i] == statusRunning {
				// Success path.
				status[i] = statusCompleted
				results[i].CompletedAt = time.Now()
				completed++

				e.emit(Event{
					Type:   EventTargetCompleted,
					Time:   results[i].CompletedAt,
					Target: t,
					Result: &results[i],
				})
			}

			// Decrement remainingDeps for dependents and enqueue newly ready ones.
			for _, depIdx := range e.plan.dependents[i] {
				if !reachable[depIdx] {
					continue
				}
				if status[depIdx] != statusPending {
					continue
				}
				remainingDeps[depIdx]--
				if remainingDeps[depIdx] == 0 {
					results[depIdx].StartedAt = time.Now()
					status[depIdx] = statusRunning

					e.emit(Event{
						Type:   EventTargetStarted,
						Time:   results[depIdx].StartedAt,
						Target: e.plan.targets[depIdx],
						Result: nil,
					})

					// Again, we don't gate scheduling on ctx here; workers
					// will check ctx before running the target body.
					readyCh <- depIdx
				}
			}
		}
	}

	// Stop workers.
	close(readyCh)
	wg.Wait()
	close(doneCh)

	// Close observer channels to signal completion.
	close(e.events)
	close(e.results)

	// Build summary.
	summary := RunSummary{
		Results: make(map[string]Result, reachableCount),
		Failed:  e.firstErr != nil,
	}
	for i := range n {
		if !reachable[i] {
			continue
		}
		summary.Results[e.plan.targets[i].Name] = results[i]
	}

	if e.firstErr != nil {
		return summary, e.firstErr
	}

	// If context was canceled but no target returned an error,
	// propagate ctx.Err() so callers can distinguish.
	if err := ctx.Err(); err != nil {
		return summary, err
	}

	return summary, nil
}

// computeReachable marks all nodes reachable from roots.
//
// If roots is empty, all nodes are reachable.
func (e *Executor) computeReachable(roots []string) ([]bool, int, error) {
	n := len(e.plan.targets)
	reachable := make([]bool, n)

	// If no roots specified, everything is reachable.
	if len(roots) == 0 {
		for i := range reachable {
			reachable[i] = true
		}
		return reachable, n, nil
	}

	// Map roots to indices.
	rootIdx := make([]int, 0, len(roots))
	for _, name := range roots {
		idx, ok := e.plan.indexByName[name]
		if !ok {
			return nil, 0, fmt.Errorf("executor run: unknown root target %q", name)
		}
		rootIdx = append(rootIdx, idx)
	}

	// DFS/BFS from each root following dependencies backwards.
	var stack []int
	seen := make([]bool, n)

	for _, r := range rootIdx {
		if seen[r] {
			continue
		}
		stack = append(stack[:0], r)
		for len(stack) > 0 {
			curr := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if seen[curr] {
				continue
			}
			seen[curr] = true
			reachable[curr] = true

			for _, dep := range e.plan.deps[curr] {
				if !seen[dep] {
					stack = append(stack, dep)
				}
			}
		}
	}

	count := 0
	for i := range reachable {
		if reachable[i] {
			count++
		}
	}

	return reachable, count, nil
}

type taskDone struct {
	index int
	err   error
}

// worker executes targets sent on readyCh and reports completion on doneCh.
func (e *Executor) worker(
	ctx context.Context,
	readyCh <-chan int,
	doneCh chan<- taskDone,
) {
	for idx := range readyCh {
		t := e.plan.targets[idx]

		select {
		case <-ctx.Done():
			// Don't run new tasks after context cancellation.
			doneCh <- taskDone{index: idx, err: ctx.Err()}
			continue
		default:
		}

		err := t.Run(ctx)
		doneCh <- taskDone{index: idx, err: err}
	}
}

func (e *Executor) setFirstErr(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.firstErr == nil {
		e.firstErr = err
	}
}

// emit sends an event to the internal channels and optional external sink.
//
// Event is passed by value; Result pointer (if present) is treated as a
// snapshot at the time of emission.
func (e *Executor) emit(ev Event) {
	// Non-blocking send to event stream; drop if buffer is full.
	if e.events != nil {
		select {
		case e.events <- ev:
		default:
		}
	}

	// Convenience result stream: only for events with a Result snapshot.
	if ev.Result != nil && e.results != nil {
		select {
		case e.results <- *ev.Result:
		default:
		}
	}

	// Optional pluggable sink (synchronous).
	if e.sink != nil {
		e.sink.HandleEvent(ev)
	}
}
