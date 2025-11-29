package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/a2y-d5l/ci/wip/engine"
)

// ClientEvent represents a JSON-serializable event sent to the web client.
type ClientEvent struct {
	Time       time.Time     `json:"time"`
	Result     *ClientResult `json:"result,omitempty"`
	Type       string        `json:"type"`
	TargetName string        `json:"targetName,omitempty"`
	TargetDesc string        `json:"targetDesc,omitempty"`
	TargetDeps []string      `json:"targetDeps,omitempty"`
}

// ClientResult represents a JSON-serializable result.
type ClientResult struct {
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Name        string    `json:"name"`
	Error       string    `json:"error,omitempty"`
	DurationMs  int64     `json:"durationMs"`
	Skipped     bool      `json:"skipped"`
}

// ClientSummary represents the final run summary for the client.
type ClientSummary struct {
	Results map[string]*ClientResult `json:"results"`
	Type    string                   `json:"type"`
	Error   string                   `json:"error,omitempty"`
	Failed  bool                     `json:"failed"`
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
	Targets []string `json:"targets"`
	Index   int      `json:"index"`
}

// TargetDefinition is a pre-defined target that users can select.
type TargetDefinition struct {
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	Deps        []string `json:"deps"`
	DurationMin int      `json:"durationMin"` // min duration in ms
	DurationMax int      `json:"durationMax"` // max duration in ms
	CanFail     bool     `json:"canFail"`     // if true, can be set to fail
}

// AvailableTargetsResponse is sent to clients listing available targets.
type AvailableTargetsResponse struct {
	Targets []TargetDefinition `json:"targets"`
}

// RunRequest is the request body for starting a run.
type RunRequest struct {
	Targets     []string `json:"targets"`     // target names to include in the plan
	RootTargets []string `json:"rootTargets"` // targets to execute (and their deps)
	FailTargets []string `json:"failTargets"` // targets that should fail
}

// PreviewRequest is the request body for previewing a plan.
type PreviewRequest struct {
	Targets     []string `json:"targets"`     // target names to include in the plan
	RootTargets []string `json:"rootTargets"` // targets to execute (and their deps)
}

// PreviewResponse shows what would be executed.
type PreviewResponse struct {
	Error   string   `json:"error,omitempty"`
	Targets []Target `json:"targets,omitempty"`
	Stages  []Stage  `json:"stages,omitempty"`
	Valid   bool     `json:"valid"`
}

// SavedPlan represents a saved/pre-compiled plan configuration.
type SavedPlan struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	Targets     []string `json:"targets"`
	IsBuiltin   bool     `json:"isBuiltin"`
}

// SavePlanRequest is the request body for saving a plan.
type SavePlanRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Targets     []string   `json:"targets"`
	Stages      [][]string `json:"stages"`
}

// SavedPlansResponse returns all saved plans.
type SavedPlansResponse struct {
	Plans []SavedPlan `json:"plans"`
}

// RunFromPlanRequest is the request to run from a saved plan.
type RunFromPlanRequest struct {
	PlanID      string   `json:"planId"`
	Mode        string   `json:"mode"`        // 'all', 'target', or 'stage'
	TargetName  string   `json:"targetName"`  // for 'target' mode
	RootTargets []string `json:"rootTargets"` // specific targets to run (optional)
	StageIndex  *int     `json:"stageIndex"`  // specific stage to run (optional)
	FailTargets []string `json:"failTargets"` // targets that should fail
}

// savedPlans stores user-saved plans (in-memory for this demo)
var (
	savedPlansMu sync.RWMutex
	savedPlans   = make(map[string]*SavedPlan)
	planCounter  int
)

// initBuiltinPlans creates some pre-defined demo plans.
func initBuiltinPlans() {
	builtinPlans := []SavedPlan{
		{
			ID:          "builtin-full",
			Name:        "Full CI/CD Pipeline",
			Description: "Complete pipeline from checkout to production deployment",
			Targets:     []string{"checkout", "install-deps", "lint", "format-check", "unit-tests", "integration-tests", "build", "security-scan", "package", "docker-build", "push-registry", "deploy-staging", "smoke-tests", "deploy-prod", "notify"},
			CreatedAt:   time.Now().Format(time.RFC3339),
			IsBuiltin:   true,
		},
		{
			ID:          "builtin-test",
			Name:        "Test Suite Only",
			Description: "Run linting and all tests without deployment",
			Targets:     []string{"checkout", "install-deps", "lint", "format-check", "unit-tests", "integration-tests"},
			CreatedAt:   time.Now().Format(time.RFC3339),
			IsBuiltin:   true,
		},
		{
			ID:          "builtin-build",
			Name:        "Build & Package",
			Description: "Build artifacts and create Docker image",
			Targets:     []string{"checkout", "install-deps", "lint", "format-check", "build", "package", "docker-build"},
			CreatedAt:   time.Now().Format(time.RFC3339),
			IsBuiltin:   true,
		},
		{
			ID:          "builtin-deploy-staging",
			Name:        "Deploy to Staging",
			Description: "Full pipeline up to staging deployment with smoke tests",
			Targets:     []string{"checkout", "install-deps", "lint", "format-check", "unit-tests", "integration-tests", "build", "security-scan", "package", "docker-build", "push-registry", "deploy-staging", "smoke-tests"},
			CreatedAt:   time.Now().Format(time.RFC3339),
			IsBuiltin:   true,
		},
		{
			ID:          "builtin-quick",
			Name:        "Quick Check",
			Description: "Fast lint and format check only",
			Targets:     []string{"checkout", "install-deps", "lint", "format-check"},
			CreatedAt:   time.Now().Format(time.RFC3339),
			IsBuiltin:   true,
		},
	}

	for i := range builtinPlans {
		savedPlans[builtinPlans[i].ID] = &builtinPlans[i]
	}
}

// sseEventSink implements engine.EventHandler and broadcasts events to SSE clients.
type sseEventSink struct {
	clients map[chan string]bool
	mu      sync.RWMutex
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

// availableTargets is the registry of all pre-defined targets users can select.
var availableTargets = []TargetDefinition{
	{Name: "checkout", Desc: "Checkout source code from repository", Deps: nil, DurationMin: 200, DurationMax: 500, CanFail: false},
	{Name: "install-deps", Desc: "Install project dependencies", Deps: []string{"checkout"}, DurationMin: 400, DurationMax: 800, CanFail: false},
	{Name: "lint", Desc: "Run static analysis and linters", Deps: []string{"install-deps"}, DurationMin: 300, DurationMax: 500, CanFail: true},
	{Name: "format-check", Desc: "Check code formatting", Deps: []string{"install-deps"}, DurationMin: 200, DurationMax: 350, CanFail: true},
	{Name: "unit-tests", Desc: "Run unit test suite", Deps: []string{"install-deps"}, DurationMin: 600, DurationMax: 1000, CanFail: true},
	{Name: "integration-tests", Desc: "Run integration test suite", Deps: []string{"unit-tests"}, DurationMin: 800, DurationMax: 1300, CanFail: true},
	{Name: "build", Desc: "Compile and build artifacts", Deps: []string{"lint", "format-check"}, DurationMin: 500, DurationMax: 800, CanFail: true},
	{Name: "security-scan", Desc: "Run security vulnerability scan", Deps: []string{"build"}, DurationMin: 400, DurationMax: 700, CanFail: true},
	{Name: "package", Desc: "Package build artifacts", Deps: []string{"build", "integration-tests"}, DurationMin: 300, DurationMax: 500, CanFail: false},
	{Name: "docker-build", Desc: "Build Docker container image", Deps: []string{"package"}, DurationMin: 700, DurationMax: 1100, CanFail: true},
	{Name: "push-registry", Desc: "Push image to container registry", Deps: []string{"docker-build", "security-scan"}, DurationMin: 400, DurationMax: 700, CanFail: true},
	{Name: "deploy-staging", Desc: "Deploy to staging environment", Deps: []string{"push-registry"}, DurationMin: 500, DurationMax: 800, CanFail: true},
	{Name: "smoke-tests", Desc: "Run smoke tests on staging", Deps: []string{"deploy-staging"}, DurationMin: 300, DurationMax: 500, CanFail: true},
	{Name: "deploy-prod", Desc: "Deploy to production environment", Deps: []string{"smoke-tests"}, DurationMin: 600, DurationMax: 900, CanFail: true},
	{Name: "notify", Desc: "Send deployment notifications", Deps: []string{"deploy-prod"}, DurationMin: 100, DurationMax: 200, CanFail: false},
}

// targetDefByName returns a target definition by name.
func targetDefByName(name string) (TargetDefinition, bool) {
	for _, t := range availableTargets {
		if t.Name == name {
			return t, true
		}
	}
	return TargetDefinition{}, false
}

// buildPlanFromSelection creates a Plan from selected target names.
func buildPlanFromSelection(targetNames []string, failTargets map[string]bool) (*engine.Plan, error) {
	targets := make([]engine.Target, 0, len(targetNames))

	for _, name := range targetNames {
		def, ok := targetDefByName(name)
		if !ok {
			return nil, fmt.Errorf("unknown target: %s", name)
		}

		// Filter deps to only include targets in the selection
		selectedSet := make(map[string]bool)
		for _, n := range targetNames {
			selectedSet[n] = true
		}

		filteredDeps := make([]string, 0)
		for _, dep := range def.Deps {
			if selectedSet[dep] {
				filteredDeps = append(filteredDeps, dep)
			}
		}

		shouldFail := failTargets[name]
		durationMin := def.DurationMin
		durationMax := def.DurationMax

		targets = append(targets, engine.Target{
			Name: def.Name,
			Desc: def.Desc,
			Deps: filteredDeps,
			Run: func(ctx context.Context) error {
				duration := time.Duration(durationMin+rand.Intn(durationMax-durationMin)) * time.Millisecond
				time.Sleep(duration)
				if shouldFail {
					return fmt.Errorf("target failed (simulated)")
				}
				return nil
			},
		})
	}

	return engine.BuildPlan(targets...)
}

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

// handleTargets returns the list of available targets for plan composition.
func handleTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AvailableTargetsResponse{
		Targets: availableTargets,
	})
}

// handlePreview previews what a plan would look like without executing it.
func handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PreviewResponse{
			Valid: false,
			Error: "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Targets) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PreviewResponse{
			Valid: false,
			Error: "No targets selected",
		})
		return
	}

	plan, err := buildPlanFromSelection(req.Targets, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PreviewResponse{
			Valid: false,
			Error: err.Error(),
		})
		return
	}

	// Build response
	stages := plan.Stages()
	resp := PreviewResponse{
		Valid:   true,
		Targets: make([]Target, 0),
		Stages:  make([]Stage, len(stages)),
	}

	for _, name := range plan.TargetNames() {
		t, _ := plan.Target(name)
		resp.Targets = append(resp.Targets, Target{
			Name: t.Name,
			Desc: t.Desc,
			Deps: t.Deps,
		})
	}

	for i, stage := range stages {
		resp.Stages[i] = Stage{
			Index:   stage.Index,
			Targets: stage.Targets,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleRunCustom runs a custom plan with selected targets.
func handleRunCustom(w http.ResponseWriter, r *http.Request) {
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

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Targets) == 0 {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "No targets selected",
		})
		return
	}

	// Build fail targets map
	failTargets := make(map[string]bool)
	for _, name := range req.FailTargets {
		failTargets[name] = true
	}

	plan, err := buildPlanFromSelection(req.Targets, failTargets)
	if err != nil {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	isRunning = true
	currentCtx, currentCancel = context.WithCancel(context.Background())
	runMutex.Unlock()

	// Determine root targets
	rootTargets := req.RootTargets
	if len(rootTargets) == 0 {
		// If no roots specified, find leaf nodes (targets with no dependents)
		rootTargets = findLeafTargets(req.Targets)
	}

	go func() {
		defer func() {
			runMutex.Lock()
			isRunning = false
			currentCancel = nil
			runMutex.Unlock()
		}()

		exec := engine.NewExecutor(plan, engine.WithEventSink(globalSink))

		// Drain the channels in goroutines
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

		summary, runErr := exec.Run(currentCtx, rootTargets...)
		globalSink.sendSummary(summary, runErr)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "started",
		"message": "Custom run started",
		"roots":   rootTargets,
	})
}

// findLeafTargets finds targets that are not dependencies of any other target.
func findLeafTargets(targetNames []string) []string {
	// Build set of all dependencies
	depSet := make(map[string]bool)
	for _, name := range targetNames {
		def, ok := targetDefByName(name)
		if !ok {
			continue
		}
		for _, dep := range def.Deps {
			depSet[dep] = true
		}
	}

	// Targets not in depSet are leaves
	leaves := make([]string, 0)
	for _, name := range targetNames {
		if !depSet[name] {
			leaves = append(leaves, name)
		}
	}

	// If somehow all are deps, return all
	if len(leaves) == 0 {
		return targetNames
	}

	return leaves
}

// handleSavedPlans returns all saved plans.
func handleSavedPlans(w http.ResponseWriter, r *http.Request) {
	savedPlansMu.RLock()
	plans := make([]SavedPlan, 0, len(savedPlans))
	for _, p := range savedPlans {
		plans = append(plans, *p)
	}
	savedPlansMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SavedPlansResponse{Plans: plans})
}

// handleSavePlan saves a new plan.
func handleSavePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SavePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan name is required",
		})
		return
	}

	if len(req.Targets) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "At least one target is required",
		})
		return
	}

	savedPlansMu.Lock()
	planCounter++
	id := fmt.Sprintf("user-%d", planCounter)
	plan := &SavedPlan{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Targets:     req.Targets,
		CreatedAt:   time.Now().Format(time.RFC3339),
		IsBuiltin:   false,
	}
	savedPlans[id] = plan
	savedPlansMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "saved",
		"plan":   plan,
	})
}

// handleDeletePlan deletes a saved plan.
func handleDeletePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	planID := r.URL.Query().Get("id")
	if planID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan ID is required",
		})
		return
	}

	savedPlansMu.Lock()
	plan, exists := savedPlans[planID]
	if !exists {
		savedPlansMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan not found",
		})
		return
	}

	if plan.IsBuiltin {
		savedPlansMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Cannot delete built-in plans",
		})
		return
	}

	delete(savedPlans, planID)
	savedPlansMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "deleted",
		"message": "Plan deleted successfully",
	})
}

// handleGetPlanDetails returns details of a saved plan including its stages.
func handleGetPlanDetails(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("id")
	if planID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan ID is required",
		})
		return
	}

	savedPlansMu.RLock()
	plan, exists := savedPlans[planID]
	savedPlansMu.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan not found",
		})
		return
	}

	// Build the plan to get stages
	builtPlan, err := buildPlanFromSelection(plan.Targets, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Get stages and targets info
	stages := builtPlan.Stages()
	stagesInfo := make([]Stage, len(stages))
	for i, stage := range stages {
		stagesInfo[i] = Stage{
			Index:   stage.Index,
			Targets: stage.Targets,
		}
	}

	targetsInfo := make([]Target, 0)
	for _, name := range builtPlan.TargetNames() {
		t, _ := builtPlan.Target(name)
		targetsInfo = append(targetsInfo, Target{
			Name: t.Name,
			Desc: t.Desc,
			Deps: t.Deps,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"plan":    plan,
		"targets": targetsInfo,
		"stages":  stagesInfo,
	})
}

// handleRunFromPlan runs a saved plan with optional target/stage selection.
func handleRunFromPlan(w http.ResponseWriter, r *http.Request) {
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

	var req RunFromPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Invalid request body: " + err.Error(),
		})
		return
	}

	savedPlansMu.RLock()
	plan, exists := savedPlans[req.PlanID]
	savedPlansMu.RUnlock()

	if !exists {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": "Plan not found",
		})
		return
	}

	// Build fail targets map
	failTargets := make(map[string]bool)
	for _, name := range req.FailTargets {
		failTargets[name] = true
	}

	builtPlan, err := buildPlanFromSelection(plan.Targets, failTargets)
	if err != nil {
		runMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// Determine root targets based on mode
	var rootTargets []string

	switch req.Mode {
	case "target":
		// Run a specific target (and its dependencies)
		if req.TargetName != "" {
			rootTargets = []string{req.TargetName}
		}
	case "stage":
		// Run targets up to and including a specific stage
		if req.StageIndex != nil {
			stages := builtPlan.Stages()
			if *req.StageIndex >= 0 && *req.StageIndex < len(stages) {
				rootTargets = stages[*req.StageIndex].Targets
			}
		}
	default:
		// "all" mode or unspecified - use provided rootTargets or find leaves
		rootTargets = req.RootTargets
	}

	// If no roots determined, find leaf nodes
	if len(rootTargets) == 0 {
		rootTargets = findLeafTargets(plan.Targets)
	}

	isRunning = true
	currentCtx, currentCancel = context.WithCancel(context.Background())
	runMutex.Unlock()

	go func() {
		defer func() {
			runMutex.Lock()
			isRunning = false
			currentCancel = nil
			runMutex.Unlock()
		}()

		exec := engine.NewExecutor(builtPlan, engine.WithEventSink(globalSink))

		// Drain the channels in goroutines
		go func() {
			for range exec.Events() {
			}
		}()
		go func() {
			for range exec.Results() {
			}
		}()

		// Send plan info first
		globalSink.sendPlanInfo(builtPlan)

		// Small delay so client receives plan info before events
		time.Sleep(100 * time.Millisecond)

		summary, runErr := exec.Run(currentCtx, rootTargets...)
		globalSink.sendSummary(summary, runErr)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "started",
		"message":  "Running from saved plan",
		"planName": plan.Name,
		"roots":    rootTargets,
	})
}

func Run(indexHTML []byte, static fs.FS) {
	globalSink = newSSEEventSink()
	initBuiltinPlans()

	// Serve static files from the frontend sub-filesystem
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	// Serve index.html at root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(indexHTML); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	})

	// API endpoints
	http.HandleFunc("/api/events", handleSSE)
	http.HandleFunc("/api/run", handleRun)
	http.HandleFunc("/api/run/custom", handleRunCustom)
	http.HandleFunc("/api/run/plan", handleRunFromPlan)
	http.HandleFunc("/api/cancel", handleCancel)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/targets", handleTargets)
	http.HandleFunc("/api/preview", handlePreview)
	http.HandleFunc("/api/plans", handleSavedPlans)
	http.HandleFunc("/api/plans/save", handleSavePlan)
	http.HandleFunc("/api/plans/delete", handleDeletePlan)
	http.HandleFunc("/api/plans/details", handleGetPlanDetails)

	addr := ":8080"
	fmt.Printf("🚀 CI Dashboard server starting on http://localhost%s\n", addr)
	fmt.Println("   Open your browser to view the CI pipeline visualization")
	fmt.Println("   Press Ctrl+C to stop the server")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
