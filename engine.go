package ci

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/a2y-d5l/ci/target"
)

// Result captures the outcome of a single target execution.
type Result struct {
	// Target is the executed target.
	Target target.T

	// Err is the error returned from the target function, if any.
	Err error

	// StartedAt is when the target began running.
	StartedAt time.Time

	// CompletedAt is when the target finished running.
	CompletedAt time.Time

	// Skipped indicates the target did not run because of upstream failure
	// or cancellation.
	Skipped bool
}

// node represents a target at runtime with dependency and result tracking.
type node struct {
	tgt  target.T
	deps []*node
	done chan struct{}

	once   sync.Once
	mu     sync.Mutex
	result Result
}

// RunTargets runs CI targets without event handling.
func RunTargets(ctx context.Context, targets ...target.T) (map[string]Result, error) {
	return RunTargetsWithHandler(ctx, nil, targets...)
}

// RunTargetsWithHandler runs CI targets with an event handler.
func RunTargetsWithHandler(ctx context.Context, handler EventHandler, targets ...target.T) (map[string]Result, error) {
	// Ensure the dependency graph is acyclic.
	if err := detectCycles(targets...); err != nil {
		return nil, err
	}

	return run(ctx, handler, targets...)
}

// detectCycles performs DFS-based cycle detection over the definitions.
func detectCycles(targets ...target.T) error {
	const (
		unvisited = iota
		visiting
		visited
	)

	visitationStatus := make(map[string]int, len(targets))
	var stack []string

	var visit func(target.T) error
	visit = func(tgt target.T) error {
		switch visitationStatus[tgt.Name()] {
		case visiting:
			// Cycle detected; construct a rough path.
			idx := -1
			for i, name := range stack {
				if name == tgt.Name() {
					idx = i
					break
				}
			}

			var cycle []string
			if idx >= 0 {
				cycle = stack[idx:]
				return fmt.Errorf("invalid dependency cycle: %s", strings.Join(cycle, " -> "))
			}

			return fmt.Errorf("invalid dependency cycle involving %q", tgt.Name())

		case visited:
			return nil
		}

		visitationStatus[tgt.Name()] = visiting
		stack = append(stack, tgt.Name())
		for _, dep := range tgt.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		visitationStatus[tgt.Name()] = visited
		return nil
	}

	for i := range targets {
		if visitationStatus[targets[i].Name()] == unvisited {
			if err := visit(targets[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// requiredTargets computes the set of targets needed to satisfy the given list.
func requiredTargets(targets ...target.T) (map[string]struct{}, error) {
	required := make(map[string]struct{})
	var visit func(target.T) error

	visit = func(tgt target.T) error {
		if _, ok := required[tgt.Name()]; ok {
			return nil
		}
		required[tgt.Name()] = struct{}{}
		for _, dep := range tgt.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		return nil
	}

	for i := range targets {
		if err := visit(targets[i]); err != nil {
			return nil, err
		}
	}

	return required, nil
}

// buildRuntimeGraph constructs node instances for the required target set.
func buildRuntimeGraph(targets ...target.T) (map[string]*node, error) {
	nodes := make(map[string]*node, len(targets))
	for _, tgt := range targets {
		nodes[tgt.Name()] = &node{
			tgt:  tgt,
			deps: nil,
			done: make(chan struct{}),
		}
	}

	// Populate dependency links.
	for id, n := range nodes {
		for _, dep := range n.tgt.Dependencies() {
			depNode, ok := nodes[dep.Name()]
			if !ok {
				// Should not happen because requiredTargets guarantees closure.
				return nil, fmt.Errorf("run: internal error: missing dependency %q for %q", dep.Name(), id)
			}
			n.deps = append(n.deps, depNode)
		}
	}

	return nodes, nil
}

// sortStrings sorts a slice of strings in place using a simple insertion sort.
// This avoids importing the sort package for a simple use case.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

// run executes the DAG for the specified targets and their dependencies.
func run(ctx context.Context, handler EventHandler, targets ...target.T) (map[string]Result, error) {
	if ctx == nil {
		return nil, errors.New("context must not be nil")
	}

	required, err := requiredTargets(targets...)
	if err != nil {
		return nil, err
	}
	if len(required) == 0 {
		return map[string]Result{}, nil
	}

	pipelineStart := time.Now()

	// Collect all required targets (including dependencies) for graph building
	allTargets := make([]target.T, 0, len(required))
	visited := make(map[string]bool)

	var collect func(target.T)
	collect = func(tgt target.T) {
		if visited[tgt.Name()] {
			return
		}
		visited[tgt.Name()] = true
		allTargets = append(allTargets, tgt)
		for _, dep := range tgt.Dependencies() {
			collect(dep)
		}
	}

	for i := range targets {
		collect(targets[i])
	}

	nodes, err := buildRuntimeGraph(allTargets...)
	if err != nil {
		return nil, err
	}

	// Emit pipeline started event
	if handler != nil {
		handler.HandleEvent(PipelineStartedEvent{
			Time:         pipelineStart,
			TotalTargets: len(nodes),
		})
	}

	// Concurrency limiter (semaphore).
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// startNode ensures each node's execution goroutine starts at most once.
	var startNode func(*node)
	startNode = func(n *node) {
		n.once.Do(func() {
			// Recursively start all dependencies first
			for _, dep := range n.deps {
				startNode(dep)
			}

			wg.Go(func() {
				// Wait for all dependencies to complete
				for _, dep := range n.deps {
					<-dep.done
				}

				// Check if any dependency failed or was skipped
				for _, dep := range n.deps {
					dep.mu.Lock()
					depFailed := dep.result.Err != nil || dep.result.Skipped
					dep.mu.Unlock()

					if depFailed {
						now := time.Now()
						n.mu.Lock()
						n.result = Result{
							Target:      n.tgt,
							StartedAt:   now,
							CompletedAt: now,
							Skipped:     true,
						}
						n.mu.Unlock()

						// Emit target completed event for skipped target
						if handler != nil {
							handler.HandleEvent(TargetCompletedEvent{
								Time:     now,
								Target:   n.tgt,
								Duration: 0,
								Error:    nil,
								Skipped:  true,
							})
						}

						close(n.done)
						return
					}
				}

				// If context is already canceled, skip this target.
				select {
				case <-ctx.Done():
					now := time.Now()
					n.mu.Lock()
					n.result = Result{
						Target:      n.tgt,
						StartedAt:   now,
						CompletedAt: now,
						Skipped:     true,
					}
					n.mu.Unlock()

					// Emit target completed event for skipped target
					if handler != nil {
						handler.HandleEvent(TargetCompletedEvent{
							Time:     now,
							Target:   n.tgt,
							Duration: 0,
							Error:    nil,
							Skipped:  true,
						})
					}

					close(n.done)
					return
				default:
				}

				// Acquire concurrency slot.
				sem <- struct{}{}
				start := time.Now()

				// Emit target started event
				if handler != nil {
					handler.HandleEvent(TargetStartedEvent{
						Time:   start,
						Target: n.tgt,
					})
				}

				// Execute target with optional output capture
				var err error
				if targetWithStreams, ok := n.tgt.(target.RunWithStreams); ok && handler != nil {
					// Target supports output capture - set up pipes
					err = runWithCapture(ctx, targetWithStreams, n.tgt, handler)
				} else {
					// Fall back to regular Run() - output goes to os.Stdout/Stderr
					err = n.tgt.Run(ctx)
				}

				end := time.Now()

				n.mu.Lock()
				n.result = Result{
					Target:      n.tgt,
					Err:         err,
					StartedAt:   start,
					CompletedAt: end,
					Skipped:     false,
				}
				n.mu.Unlock()

				// Emit target completed event
				if handler != nil {
					handler.HandleEvent(TargetCompletedEvent{
						Time:     end,
						Target:   n.tgt,
						Duration: end.Sub(start),
						Error:    err,
						Skipped:  false,
					})
				}

				<-sem // Release concurrency slot.
				close(n.done)
			})
		})
	}

	// Kick off execution starting from the requested targets (or all, if none provided).
	if len(targets) == 0 {
		for _, n := range nodes {
			startNode(n)
		}
	} else {
		for _, tgt := range targets {
			n, ok := nodes[tgt.Name()]
			if !ok {
				return nil, fmt.Errorf("run: unknown target %q (after graph build)", tgt.Name())
			}
			startNode(n)
		}
	}

	// Wait for all started nodes to finish.
	wg.Wait()

	results := make(map[string]Result, len(nodes))

	// Collect node names in sorted order for deterministic error reporting
	nodeNames := make([]string, 0, len(nodes))
	for id := range nodes {
		nodeNames = append(nodeNames, id)
	}
	// Sort to ensure deterministic iteration order
	sortStrings(nodeNames)

	var errs []error
	for _, id := range nodeNames {
		n := nodes[id]
		<-n.done

		n.mu.Lock()
		res := n.result
		// Ensure Target and Description are always set.
		if res.Target == nil {
			res.Target = n.tgt
		}
		results[id] = res
		if res.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, res.Err))
		}
		n.mu.Unlock()
	}

	var aggErr error
	if len(errs) > 0 {
		aggErr = errors.Join(errs...)
	}

	// Emit pipeline completed event
	if handler != nil {
		handler.HandleEvent(PipelineCompletedEvent{
			Time:     time.Now(),
			Duration: time.Since(pipelineStart),
			Results:  results,
			Error:    aggErr,
		})
	}

	return results, aggErr
}

// runWithCapture executes a target with output capture enabled.
// It creates pipes for stdout/stderr, starts goroutines to read from them
// and emit TargetOutputEvent, then runs the target with the pipe writers.
func runWithCapture(ctx context.Context, targetWithStreams target.RunWithStreams, tgt target.T, handler EventHandler) error {
	// Create pipes for stdout and stderr
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	// Start goroutines to capture output
	var captureWg sync.WaitGroup
	captureWg.Add(2)

	// Capture stdout
	go func() {
		defer captureWg.Done()
		scanner := bufio.NewScanner(stdoutR)
		for scanner.Scan() {
			handler.HandleEvent(TargetOutputEvent{
				Time:   time.Now(),
				Target: tgt,
				Stream: StreamStdout,
				Line:   scanner.Text(),
			})
		}
	}()

	// Capture stderr
	go func() {
		defer captureWg.Done()
		scanner := bufio.NewScanner(stderrR)
		for scanner.Scan() {
			handler.HandleEvent(TargetOutputEvent{
				Time:   time.Now(),
				Target: tgt,
				Stream: StreamStderr,
				Line:   scanner.Text(),
			})
		}
	}()

	// Run the target with custom streams
	err := targetWithStreams.RunWithStreams(ctx, stdoutW, stderrW)

	// Close the write ends of the pipes to signal EOF to the readers
	stdoutW.Close()
	stderrW.Close()

	// Wait for all output to be captured
	captureWg.Wait()

	return err
}
