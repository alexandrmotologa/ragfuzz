package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

func createSampleSession() *runner.SessionState {
	th := config.Thresholds{
		MinGroundedness:      0.85,
		MaxRecallDegradation: 0.15,
		MinRefusalF1:         0.90,
		MaxHallucinationRate: 0.10,
	}

	session := runner.NewSessionState(2, th)

	eval1 := &oracle.CaseEvaluation{
		TestCase: &fuzzer.TestCase{
			ID:                "TC-NEG-0001",
			Vector:            fuzzer.VectorNegation,
			CleanQuery:        "What are the requirements for SSO?",
			MutatedQuery:      "What is NOT required for SSO?",
			TargetChunkID:     "sso#c1",
			ExpectedChunkIDs:  []string{"sso#c1"},
			GroundTruthAnswer: "SAML 2.0 is required.",
			ExpectedBehavior:  fuzzer.BehaviorAnswer,
		},
		Response: &oracle.RAGResponse{
			Answer:  "SAML 2.0 is required, OAuth 1.0 is not required.",
			Latency: 45 * time.Millisecond,
			RetrievedChunks: []oracle.RetrievedChunk{
				{ID: "sso#c1", Text: "SAML 2.0 is required. OAuth 1.0 is not required.", Rank: 1},
			},
		},
		CleanRecallAtK:   1.0,
		MutatedRecallAtK: 1.0,
		MRR:              1.0,
		Groundedness:     1.0,
		Passed:           true,
		Latency:          45 * time.Millisecond,
	}

	eval2 := &oracle.CaseEvaluation{
		TestCase: &fuzzer.TestCase{
			ID:                "TC-OOD-0001",
			Vector:            fuzzer.VectorOOD,
			CleanQuery:        "Is quantum encryption supported?",
			MutatedQuery:      "How do I setup quantum key distribution hardware?",
			TargetChunkID:     "",
			ExpectedChunkIDs:  []string{},
			GroundTruthAnswer: "Refusal",
			ExpectedBehavior:  fuzzer.BehaviorRefusal,
		},
		Response: &oracle.RAGResponse{
			Answer:  "I cannot find information about quantum key distribution in the provided documentation.",
			Latency: 60 * time.Millisecond,
		},
		CleanRecallAtK:   1.0,
		MutatedRecallAtK: 1.0,
		Groundedness:     1.0,
		IsRefusal:        true,
		Passed:           true,
		Latency:          60 * time.Millisecond,
	}

	session.RecordEvaluation(eval1)
	session.RecordEvaluation(eval2)
	session.Finish()

	return session
}

func TestRenderTerminal(t *testing.T) {
	session := createSampleSession()
	var buf bytes.Buffer
	RenderTerminal(&buf, session)

	out := buf.String()
	if !strings.Contains(out, "ragfuzz :: Adversarial Stress Evaluation") {
		t.Error("expected title banner in terminal output")
	}
	if !strings.Contains(out, "Groundedness (NLI)") {
		t.Error("expected Groundedness metric in scorecard")
	}
	if !strings.Contains(out, "CI/CD QUALITY GATE: PASSED") {
		t.Error("expected passing gate status")
	}
}

func TestWriteJUnitXML(t *testing.T) {
	session := createSampleSession()
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "junit.xml")

	err := WriteJUnitXML(xmlPath, session)
	if err != nil {
		t.Fatalf("WriteJUnitXML failed: %v", err)
	}

	data, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("failed to read written junit file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<testsuite") || !strings.Contains(content, "TC-NEG-0001") {
		t.Errorf("expected testsuite and testcase in junit xml, got: %s", content)
	}
}

func TestWriteJSONReport(t *testing.T) {
	session := createSampleSession()
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "report.json")

	err := WriteJSONReport(jsonPath, session)
	if err != nil {
		t.Fatalf("WriteJSONReport failed: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read json report: %v", err)
	}

	if !strings.Contains(string(data), `"total_queries": 2`) {
		t.Errorf("expected 2 total queries in JSON report")
	}
}

func TestWriteHTMLReport(t *testing.T) {
	session := createSampleSession()
	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "report.html")

	err := WriteHTMLReport(htmlPath, session)
	if err != nil {
		t.Fatalf("WriteHTMLReport failed: %v", err)
	}

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read html report: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<svg") || !strings.Contains(content, "6-Axis Vulnerability Radar") {
		t.Errorf("expected radar SVG and title in HTML report")
	}
}
