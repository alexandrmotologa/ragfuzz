package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

func TestServerEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	chunks := []*corpus.Chunk{
		{
			ID:      "test#c1",
			DocPath: "test.md",
			Heading: "Test Section",
			Text:    "Test text.",
		},
	}
	topology := corpus.BuildTopology(chunks)

	session := runner.NewSessionState(1, cfg.Thresholds)
	session.RecordEvaluation(&oracle.CaseEvaluation{
		TestCase: &fuzzer.TestCase{ID: "TC-001", Vector: fuzzer.VectorNegation, TargetChunkID: "test#c1"},
		Response: &oracle.RAGResponse{Answer: "Test answer"},
		Passed:   true,
	})
	session.Finish()

	srv := NewServer(cfg, chunks, topology, session)
	handler := srv.Router()

	// 1. Test /api/status
	reqStatus := httptest.NewRequest("GET", "/api/status", nil)
	recStatus := httptest.NewRecorder()
	handler.ServeHTTP(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Errorf("expected 200 from /api/status, got %d", recStatus.Code)
	}

	var statusResp map[string]interface{}
	json.NewDecoder(recStatus.Body).Decode(&statusResp)
	if statusResp["status"] != "ready" {
		t.Errorf("expected status 'ready', got: %v", statusResp["status"])
	}

	// 2. Test /api/summary
	reqSummary := httptest.NewRequest("GET", "/api/summary", nil)
	recSummary := httptest.NewRecorder()
	handler.ServeHTTP(recSummary, reqSummary)

	if recSummary.Code != http.StatusOK {
		t.Errorf("expected 200 from /api/summary, got %d", recSummary.Code)
	}

	// 3. Test /api/chunks
	reqChunks := httptest.NewRequest("GET", "/api/chunks", nil)
	recChunks := httptest.NewRecorder()
	handler.ServeHTTP(recChunks, reqChunks)

	if recChunks.Code != http.StatusOK {
		t.Errorf("expected 200 from /api/chunks, got %d", recChunks.Code)
	}

	// 4. Test Embedded Static UI (GET /)
	reqUI := httptest.NewRequest("GET", "/", nil)
	recUI := httptest.NewRecorder()
	handler.ServeHTTP(recUI, reqUI)

	if recUI.Code != http.StatusOK {
		t.Errorf("expected 200 from /, got %d", recUI.Code)
	}
}
