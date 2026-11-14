package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
)

func TestWorkerPoolExecution(t *testing.T) {
	// Setup a mock RAG server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)

		query := req["query"]
		resp := map[string]interface{}{
			"answer": "SAML SSO is supported for enterprise organizations.",
			"chunks": []map[string]string{
				{
					"id":   "chunk-1",
					"text": "SAML SSO is supported for enterprise organizations.",
				},
			},
		}
		if query == "OOD" {
			resp["answer"] = "I cannot find information on that in the documentation."
			resp["chunks"] = []map[string]string{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		Target: config.TargetConfig{
			Endpoint:        server.URL,
			Method:          "POST",
			Timeout:         2 * time.Second,
			Concurrency:     2,
			RateLimitRPS:    100,
			RequestTemplate: `{"query":"{query}"}`,
			ResponseMapping: config.ResponseMapping{
				AnswerPath:    "answer",
				ChunksPath:    "chunks",
				ChunkIDPath:   "id",
				ChunkTextPath: "text",
			},
		},
		Oracle: config.OracleConfig{
			TopK: 3,
		},
		Thresholds: config.Thresholds{
			MinGroundedness:      0.80,
			MaxRecallDegradation: 0.20,
			MinRefusalF1:         0.80,
			MaxHallucinationRate: 0.20,
		},
	}

	testSuite := []*fuzzer.TestCase{
		{
			ID:                "TC-001",
			Vector:            fuzzer.VectorNegation,
			MutatedQuery:      "How does SAML auth work?",
			TargetChunkID:     "chunk-1",
			ExpectedChunkIDs:  []string{"chunk-1"},
			ExpectedBehavior:  fuzzer.BehaviorAnswer,
			GroundTruthAnswer: "SAML SSO is supported.",
		},
		{
			ID:                "TC-002",
			Vector:            fuzzer.VectorOOD,
			MutatedQuery:      "OOD",
			TargetChunkID:     "",
			ExpectedChunkIDs:  []string{},
			ExpectedBehavior:  fuzzer.BehaviorRefusal,
			GroundTruthAnswer: "Refusal",
		},
	}

	pool := NewWorkerPool(cfg, nil)
	session, err := pool.Run(context.Background(), testSuite)
	if err != nil {
		t.Fatalf("WorkerPool.Run failed: %v", err)
	}

	if session.Completed != 2 {
		t.Errorf("expected 2 completed queries, got %d", session.Completed)
	}
	if session.Passed != 2 {
		t.Errorf("expected 2 passed queries, got %d (failed: %d)", session.Passed, session.Failed)
	}

	summary := session.Summary
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if !summary.GatePassed {
		t.Errorf("expected gate to pass, breaches: %v", summary.GateBreaches)
	}
}
