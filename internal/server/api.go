package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/analyzer"
	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

// Server coordinates the REST API, SSE streaming, and embedded web studio.
type Server struct {
	cfg        *config.Config
	topology   *corpus.TopologyGraph
	chunks     []*corpus.Chunk
	hub        *SSEHub
	session    *runner.SessionState
	mu         sync.RWMutex
	httpServer *http.Server
	isRunning  bool
}

// NewServer initializes the embedded studio server.
func NewServer(cfg *config.Config, chunks []*corpus.Chunk, topology *corpus.TopologyGraph, initialSession *runner.SessionState) *Server {
	return &Server{
		cfg:      cfg,
		chunks:   chunks,
		topology: topology,
		hub:      NewSSEHub(),
		session:  initialSession,
	}
}

// Router sets up HTTP handlers for API and static files.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/summary", s.handleSummary)
	mux.HandleFunc("/api/cases", s.handleCases)
	mux.HandleFunc("/api/chunks", s.handleChunks)
	mux.HandleFunc("/api/graph", s.handleGraph)
	mux.HandleFunc("/api/probe", s.handleProbe)
	mux.HandleFunc("/api/recommendations", s.handleRecommendations)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.Handle("/api/stream", s.hub)

	// Embedded Static UI
	mux.Handle("/", DistHandler())

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"status":          "ready",
		"version":         "1.0.0",
		"target_endpoint": s.cfg.Target.Endpoint,
		"corpus_chunks":   len(s.chunks),
		"is_running":      s.isRunning,
	}
	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil || s.session.Summary == nil {
		respondJSON(w, http.StatusOK, nil)
		return
	}
	respondJSON(w, http.StatusOK, s.session.Summary)
}

func (s *Server) handleCases(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil {
		respondJSON(w, http.StatusOK, []*oracle.CaseEvaluation{})
		return
	}

	evals := s.session.Evaluations
	vecFilter := r.URL.Query().Get("vector")
	statusFilter := r.URL.Query().Get("status")

	var filtered []*oracle.CaseEvaluation
	for _, ev := range evals {
		if vecFilter != "" && string(ev.TestCase.Vector) != vecFilter {
			continue
		}
		if statusFilter == "passed" && !ev.Passed {
			continue
		}
		if statusFilter == "failed" && ev.Passed {
			continue
		}
		filtered = append(filtered, ev)
	}

	respondJSON(w, http.StatusOK, filtered)
}

type chunkVulnerabilityResponse struct {
	ChunkID           string  `json:"chunk_id"`
	DocPath           string  `json:"doc_path"`
	Heading           string  `json:"heading"`
	TotalTested       int     `json:"total_tested"`
	FailedCount       int     `json:"failed_count"`
	DistractorHits    int     `json:"distractor_hits"`
	VulnerabilityRate float64 `json:"vulnerability_rate"`
	Text              string  `json:"text"`
}

func (s *Server) handleChunks(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalTestedMap := make(map[string]int)
	failedMap := make(map[string]int)
	distractorMap := make(map[string]int)

	if s.session != nil {
		for _, ev := range s.session.Evaluations {
			targetID := ev.TestCase.TargetChunkID
			if targetID != "" {
				totalTestedMap[targetID]++
				if !ev.Passed {
					failedMap[targetID]++
				}
			}
			if ev.DistractorIntruded {
				for _, dID := range ev.TestCase.DistractorChunkIDs {
					distractorMap[dID]++
				}
			}
		}
	}

	var results []chunkVulnerabilityResponse
	for _, ch := range s.chunks {
		tested := totalTestedMap[ch.ID]
		failed := failedMap[ch.ID]
		dHits := distractorMap[ch.ID]

		vulnRate := 0.0
		if tested > 0 {
			vulnRate = float64(failed) / float64(tested)
		}

		results = append(results, chunkVulnerabilityResponse{
			ChunkID:           ch.ID,
			DocPath:           ch.DocPath,
			Heading:           ch.Heading,
			TotalTested:       tested,
			FailedCount:       failed,
			DistractorHits:    dHits,
			VulnerabilityRate: vulnRate,
			Text:              ch.Text,
		})
	}

	respondJSON(w, http.StatusOK, results)
}

func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.session == nil || s.session.Summary == nil {
		respondJSON(w, http.StatusOK, []analyzer.Recommendation{})
		return
	}

	recs := analyzer.GenerateRecommendations(s.session.Summary)
	respondJSON(w, http.StatusOK, recs)
}

type graphNode struct {
	ID            string  `json:"id"`
	Heading       string  `json:"heading"`
	DocPath       string  `json:"doc_path"`
	WordCount     int     `json:"word_count"`
	Vulnerability float64 `json:"vulnerability"`
}

type graphEdge struct {
	Source       string   `json:"source"`
	Target       string   `json:"target"`
	Similarity   float64  `json:"similarity"`
	SharedKeys   []string `json:"shared_keys"`
	IsDistractor bool     `json:"is_distractor"`
}

type graphResponse struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nodes []graphNode
	for _, ch := range s.chunks {
		nodes = append(nodes, graphNode{
			ID:        ch.ID,
			Heading:   ch.Heading,
			DocPath:   ch.DocPath,
			WordCount: ch.WordCount,
		})
	}

	var edges []graphEdge
	if s.topology != nil {
		for _, p := range s.topology.HighOverlapPairs {
			edges = append(edges, graphEdge{
				Source:       p.ChunkA.ID,
				Target:       p.ChunkB.ID,
				Similarity:   p.Similarity,
				SharedKeys:   p.SharedKeys,
				IsDistractor: p.Similarity > 0.40,
			})
		}
	}

	respondJSON(w, http.StatusOK, graphResponse{Nodes: nodes, Edges: edges})
}

type probeRequest struct {
	Query         string `json:"query"`
	Vector        string `json:"vector"`
	TargetChunkID string `json:"target_chunk_id"`
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req probeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	cleanQuery := strings.TrimSpace(req.Query)
	if cleanQuery == "" {
		cleanQuery = "Explain the system configuration."
	}

	mutatedQuery := cleanQuery
	vec := fuzzer.AttackVector(req.Vector)
	if vec == "" {
		vec = fuzzer.VectorNegation
	}

	// Apply quick live mutation
	switch vec {
	case fuzzer.VectorNegation:
		mutatedQuery = fmt.Sprintf("What components do NOT allow or exclude %s?", cleanQuery)
	case fuzzer.VectorTypoNoise:
		mutator := fuzzer.NewTypoMutator(2, time.Now().UnixNano())
		words := strings.Fields(cleanQuery)
		if len(words) > 0 {
			words[len(words)-1] = mutator.InjectTypo(words[len(words)-1])
			mutatedQuery = strings.Join(words, " ")
		}
	case fuzzer.VectorDistractor:
		if s.topology != nil {
			dist, shared := s.topology.FindDistractorChunk(req.TargetChunkID)
			lure := "legacy protocols"
			if len(shared) > 0 {
				lure = shared[0]
			} else if dist != nil && dist.Heading != "" {
				lure = dist.Heading
			}
			mutatedQuery = fmt.Sprintf("%s specifically concerning %s", cleanQuery, lure)
		}
	case fuzzer.VectorOOD:
		mutatedQuery = "How do I configure quantum key distribution modules for orbital relay?"
	}

	tc := &fuzzer.TestCase{
		ID:                "TC-PROBE-LIVE",
		Vector:            vec,
		CleanQuery:        cleanQuery,
		MutatedQuery:      mutatedQuery,
		TargetChunkID:     req.TargetChunkID,
		ExpectedChunkIDs:  []string{req.TargetChunkID},
		ExpectedBehavior:  fuzzer.BehaviorAnswer,
		GroundTruthAnswer: "Live probe ground truth.",
	}
	if vec == fuzzer.VectorOOD {
		tc.ExpectedBehavior = fuzzer.BehaviorRefusal
		tc.ExpectedChunkIDs = []string{}
	}

	client := runner.NewRAGClient(&s.cfg.Target)
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Target.Timeout)
	defer cancel()

	resp, err := client.Query(ctx, mutatedQuery)
	if err != nil {
		resp = &oracle.RAGResponse{Error: err.Error()}
	}

	eval := oracle.EvaluateCase(tc, resp, s.cfg.Oracle.TopK, s.cfg.Thresholds.MinGroundedness)
	respondJSON(w, http.StatusOK, eval)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		http.Error(w, "fuzzing run already in progress", http.StatusConflict)
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	go s.executeAsyncRun()

	respondJSON(w, http.StatusAccepted, map[string]string{
		"message": "adversarial fuzz run initiated",
	})
}

func (s *Server) executeAsyncRun() {
	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	generator := fuzzer.NewGenerator(&s.cfg.Fuzzer, s.topology)
	suite, err := generator.GenerateSuite()
	if err != nil {
		s.hub.Broadcast("error", map[string]interface{}{"error": err.Error()})
		return
	}

	s.hub.Broadcast("start", map[string]interface{}{
		"total": len(suite),
	})

	wp := runner.NewWorkerPool(s.cfg, func(eval *oracle.CaseEvaluation, current, total int) {
		s.hub.Broadcast("progress", map[string]interface{}{
			"completed": current,
			"total":     total,
			"vector":    string(eval.TestCase.Vector),
		})
	})

	session, err := wp.Run(context.Background(), suite)
	if err != nil {
		s.hub.Broadcast("error", map[string]interface{}{"error": err.Error()})
		return
	}

	s.mu.Lock()
	s.session = session
	s.mu.Unlock()

	s.hub.Broadcast("complete", map[string]interface{}{
		"passed": session.Passed,
		"failed": session.Failed,
		"total":  session.Total,
	})
}

// Start begins serving HTTP traffic on the configured port.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Studio.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully terminates the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
