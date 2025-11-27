// file: wip/examples/clients/web/main.go
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/a2y-d5l/ci/wip/engine"
)

//go:embed static/*
var staticFiles embed.FS

// ClientEvent represents a JSON-serializable event sent to the web client.
type ClientEvent struct {
	Type       string        `json:"type"`
	Time       time.Time     `json:"time"`
	TargetName string        `json:"targetName,omitempty"`
	TargetDesc string        `json:"targetDesc,omitempty"`
	TargetDeps []string      `json:"targetDeps,omitempty"`
	Result     *ClientResult `json:"result,omitempty"`
}

// ClientResult represents a JSON-serializable result.
type ClientResult struct {
	Name        string    `json:"name"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Skipped     bool      `json:"skipped"`
	DurationMs  int64     `json:"durationMs"`
}

// ClientSummary represents the final run summary for the client.
type ClientSummary struct {
	Type    string                   `json:"type"`
	Failed  bool                     `json:"failed"`
	Results map[string]*ClientResult `json:"results"`
	Error   string                   `json:"error,omitempty"`
}

// PlanInfo represents plan metadata sent to the client.
type PlanInfo struct {
	Type    string   `json:"type"`
	Targets []Target `json:"targets"`
	Stages  []Stage  `json:"stages"`
}

// Target represents target info for the client.
type Target struct {
	Name string   `json:"name"`
	Desc string   `json:"desc"`
	Deps []string `json:"deps"`
}

// Stage represents a stage for the client.
type Stage struct {
	Index   int      `json:"index"`
	Targets []string `json:"targets"`
}

// sseEventSink implements engine.EventHandler and broadcasts events to SSE clients.
type sseEventSink struct {
	mu      sync.RWMutex
	clients map[chan string]bool
}

func newSSEEventSink() *sseEventSink {
	return &sseEventSink{
		clients: make(map[chan string]bool),
	}
}

func (s *sseEventSink) addClient(ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[ch] = true
}

func (s *sseEventSink) removeClient(ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

func (s *sseEventSink) broadcast(data string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- data:
		default:
			// Skip if client buffer is full
		}
	}
}

func (s *sseEventSink) HandleEvent(ev engine.Event) {
	clientEv := ClientEvent{
		Time: ev.Time,
	}

	if ev.Target != nil {
		clientEv.TargetName = ev.Target.Name
		clientEv.TargetDesc = ev.Target.Desc
		clientEv.TargetDeps = ev.Target.Deps
	}

	switch ev.Type {
	case engine.EventTargetStarted:
		clientEv.Type = "started"
	case engine.EventTargetCompleted:
		clientEv.Type = "completed"
	case engine.EventTargetSkipped:
		clientEv.Type = "skipped"
	default:
		clientEv.Type = "unknown"
	}

	if ev.Result != nil {
		clientEv.Result = &ClientResult{
			Name:        ev.Result.Name,
			StartedAt:   ev.Result.StartedAt,
			CompletedAt: ev.Result.CompletedAt,
			Skipped:     ev.Result.Skipped,
			DurationMs:  ev.Result.CompletedAt.Sub(ev.Result.StartedAt).Milliseconds(),
		}
		if ev.Result.Err != nil {
			clientEv.Result.Error = ev.Result.Err.Error()
		}
	}

	data, err := json.Marshal(clientEv)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}
	s.broadcast(string(data))
}

func (s *sseEventSink) sendPlanInfo(plan *engine.Plan) {
	stages := plan.Stages()
	planInfo := PlanInfo{
		Type:    "plan",
		Targets: make([]Target, 0),
		Stages:  make([]Stage, len(stages)),
	}

	for _, name := range plan.TargetNames() {
		t, _ := plan.Target(name)
		planInfo.Targets = append(planInfo.Targets, Target{
			Name: t.Name,
			Desc: t.Desc,
			Deps: t.Deps,
		})
	}

	for i, stage := range stages {
		planInfo.Stages[i] = Stage{
			Index:   stage.Index,
			Targets: stage.Targets,
		}
	}

	data, err := json.Marshal(planInfo)
	if err != nil {
		log.Printf("Error marshaling plan info: %v", err)
		return
	}
	s.broadcast(string(data))
}

func (s *sseEventSink) sendSummary(summary engine.RunSummary, runErr error) {
	clientSummary := ClientSummary{
		Type:    "summary",
		Failed:  summary.Failed,
		Results: make(map[string]*ClientResult),
	}

	if runErr != nil {
		clientSummary.Error = runErr.Error()
	}

	for name, res := range summary.Results {
		cr := &ClientResult{
			Name:        res.Name,
			StartedAt:   res.StartedAt,
			CompletedAt: res.CompletedAt,
			Skipped:     res.Skipped,
			DurationMs:  res.CompletedAt.Sub(res.StartedAt).Milliseconds(),
		}
		if res.Err != nil {
			cr.Error = res.Err.Error()
		}
		clientSummary.Results[name] = cr
	}

	data, err := json.Marshal(clientSummary)
	if err != nil {
		log.Printf("Error marshaling summary: %v", err)
		return
	}
	s.broadcast(string(data))
}

var (
	globalSink    *sseEventSink
	runMutex      sync.Mutex
	isRunning     bool
	currentCtx    context.Context
	currentCancel context.CancelFunc
)

func createDemoPlan(withFailure bool) *engine.Plan {
	targets := []engine.Target{
		{
			Name: "checkout",
			Desc: "Checkout source code from repository",
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "install-deps",
			Desc: "Install project dependencies",
			Deps: []string{"checkout"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(400+rand.Intn(400)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "lint",
			Desc: "Run static analysis and linters",
			Deps: []string{"install-deps"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "format-check",
			Desc: "Check code formatting",
			Deps: []string{"install-deps"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(200+rand.Intn(150)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "unit-tests",
			Desc: "Run unit test suite",
			Deps: []string{"install-deps"},
			Run: func(ctx context.Context) error {
				time.Sleep(time.Duration(600+rand.Intn(400)) * time.Millisecond)
				if withFailure {
					return fmt.Errorf("test assertion failed: expected 42, got 0")
				}
				return nil
			},
		},
		{
			Name: "integration-tests",
			Desc: "Run integration test suite",
			Deps: []string{"unit-tests"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(800+rand.Intn(500)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "build",
			Desc: "Compile and build artifacts",
			Deps: []string{"lint", "format-check"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(500+rand.Intn(300)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "security-scan",
			Desc: "Run security vulnerability scan",
			Deps: []string{"build"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(400+rand.Intn(300)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "package",
			Desc: "Package build artifacts",
			Deps: []string{"build", "integration-tests"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "docker-build",
			Desc: "Build Docker container image",
			Deps: []string{"package"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(700+rand.Intn(400)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "push-registry",
			Desc: "Push image to container registry",
			Deps: []string{"docker-build", "security-scan"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(400+rand.Intn(300)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "deploy-staging",
			Desc: "Deploy to staging environment",
			Deps: []string{"push-registry"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(500+rand.Intn(300)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "smoke-tests",
			Desc: "Run smoke tests on staging",
			Deps: []string{"deploy-staging"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(300+rand.Intn(200)) * time.Millisecond)
				return nil
			},
		},
		{
			Name: "all",
			Desc: "Complete CI/CD pipeline",
			Deps: []string{"smoke-tests"},
			Run: func(_ context.Context) error {
				time.Sleep(time.Duration(100+rand.Intn(100)) * time.Millisecond)
				return nil
			},
		},
	}

	plan, err := engine.BuildPlan(targets...)
	if err != nil {
		panic(err)
	}
	return plan
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientCh := make(chan string, 100)
	globalSink.addClient(clientCh)
	defer globalSink.removeClient(clientCh)

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-clientCh:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runMutex.Lock()
	if isRunning {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "A run is already in progress",
		})
		return
	}
	isRunning = true
	currentCtx, currentCancel = context.WithCancel(context.Background())
	runMutex.Unlock()

	// Check for failure simulation
	withFailure := r.URL.Query().Get("fail") == "true"

	go func() {
		defer func() {
			runMutex.Lock()
			isRunning = false
			currentCancel = nil
			runMutex.Unlock()
		}()

		plan := createDemoPlan(withFailure)
		exec := engine.NewExecutor(plan, engine.WithEventSink(globalSink))

		// Drain the channels in a goroutine
		go func() {
			for range exec.Events() {
			}
		}()
		go func() {
			for range exec.Results() {
			}
		}()

		// Send plan info first
		globalSink.sendPlanInfo(plan)

		// Small delay so client receives plan info before events
		time.Sleep(100 * time.Millisecond)

		summary, err := exec.Run(currentCtx, "all")
		globalSink.sendSummary(summary, err)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "started",
		"message": "Run started",
	})
}

func handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runMutex.Lock()
	if !isRunning || currentCancel == nil {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "No run in progress",
		})
		return
	}
	currentCancel()
	runMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "cancelled",
		"message": "Run cancellation requested",
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	runMutex.Lock()
	running := isRunning
	runMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"running": running,
	})
}

func main() {
	globalSink = newSSEEventSink()

	// Serve static files from embedded FS
	staticFS := http.FS(staticFiles)
	http.Handle("/static/", http.FileServer(staticFS))

	// Serve index.html at root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "Index not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})

	// API endpoints
	http.HandleFunc("/api/events", handleSSE)
	http.HandleFunc("/api/run", handleRun)
	http.HandleFunc("/api/cancel", handleCancel)
	http.HandleFunc("/api/status", handleStatus)

	addr := ":8080"
	fmt.Printf("🚀 CI Dashboard server starting on http://localhost%s\n", addr)
	fmt.Println("   Open your browser to view the CI pipeline visualization")
	fmt.Println("   Press Ctrl+C to stop the server")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
