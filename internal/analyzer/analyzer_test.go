package analyzer

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
	"github.com/alexandrmotologa/ragfuzz/internal/reporter"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

func TestCompareReportsAndRecommendations(t *testing.T) {
	th := config.Thresholds{
		MinGroundedness:      0.85,
		MaxRecallDegradation: 0.15,
		MinRefusalF1:         0.90,
		MaxHallucinationRate: 0.10,
	}

	// Baseline session: 100% pass
	s1 := runner.NewSessionState(1, th)
	s1.RecordEvaluation(&oracle.CaseEvaluation{
		TestCase: &fuzzer.TestCase{ID: "TC-001", Vector: fuzzer.VectorNegation},
		Response: &oracle.RAGResponse{Answer: "Good"},
		Passed:   true,
	})
	s1.Finish()

	// Target session: failed TC-001
	s2 := runner.NewSessionState(1, th)
	s2.RecordEvaluation(&oracle.CaseEvaluation{
		TestCase:      &fuzzer.TestCase{ID: "TC-001", Vector: fuzzer.VectorNegation},
		Response:      &oracle.RAGResponse{Answer: "Bad"},
		Passed:        false,
		FailureReason: "Negation blindness",
	})
	s2.Finish()

	tmpDir := t.TempDir()
	p1 := filepath.Join(tmpDir, "base.json")
	p2 := filepath.Join(tmpDir, "target.json")

	if err := reporter.WriteJSONReport(p1, s1); err != nil {
		t.Fatalf("failed to write base report: %v", err)
	}
	if err := reporter.WriteJSONReport(p2, s2); err != nil {
		t.Fatalf("failed to write target report: %v", err)
	}

	diff, err := CompareReports(p1, p2)
	if err != nil {
		t.Fatalf("CompareReports failed: %v", err)
	}

	if !diff.HasRegressions {
		t.Error("expected regressions to be flagged")
	}
	if len(diff.RegressedCases) != 1 {
		t.Errorf("expected 1 regressed case, got: %d", len(diff.RegressedCases))
	}

	// Recommendations
	recs := GenerateRecommendations(s2.Summary)
	if len(recs) == 0 {
		t.Error("expected mitigation recommendations for low negation score")
	}
	if recs[0].Vector != fuzzer.VectorNegation {
		t.Errorf("expected negation recommendation, got: %s", recs[0].Vector)
	}
}
