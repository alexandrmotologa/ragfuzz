package analyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

type fullReport struct {
	Summary     *oracle.SuiteSummary     `json:"summary"`
	Evaluations []*oracle.CaseEvaluation `json:"evaluations"`
}

// DiffResult captures regressions and improvements between two evaluation runs.
type DiffResult struct {
	BaselineSummary   *oracle.SuiteSummary            `json:"baseline_summary"`
	TargetSummary     *oracle.SuiteSummary            `json:"target_summary"`
	PassRateDelta     float64                         `json:"pass_rate_delta"`
	GroundednessDelta float64                         `json:"groundedness_delta"`
	RecallDelta       float64                         `json:"recall_delta"`
	RefusalF1Delta    float64                         `json:"refusal_f1_delta"`
	VectorDeltas      map[fuzzer.AttackVector]float64 `json:"vector_deltas"`
	RegressedCases    []string                        `json:"regressed_cases"`
	ResolvedCases     []string                        `json:"resolved_cases"`
	HasRegressions    bool                            `json:"has_regressions"`
}

// CompareReports parses two JSON reports and computes a structured regression diff.
func CompareReports(baselinePath, targetPath string) (*DiffResult, error) {
	bData, err := os.ReadFile(baselinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline report: %w", err)
	}
	var baseline fullReport
	if err := json.Unmarshal(bData, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline report: %w", err)
	}

	tData, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read target report: %w", err)
	}
	var target fullReport
	if err := json.Unmarshal(tData, &target); err != nil {
		return nil, fmt.Errorf("failed to parse target report: %w", err)
	}

	bSum := baseline.Summary
	tSum := target.Summary

	diff := &DiffResult{
		BaselineSummary:   bSum,
		TargetSummary:     tSum,
		PassRateDelta:     tSum.PassRate - bSum.PassRate,
		GroundednessDelta: tSum.MeanGroundedness - bSum.MeanGroundedness,
		RecallDelta:       tSum.MeanRecallMutated - bSum.MeanRecallMutated,
		RefusalF1Delta:    tSum.RefusalF1 - bSum.RefusalF1,
		VectorDeltas:      make(map[fuzzer.AttackVector]float64),
	}

	for _, vec := range fuzzer.AllVectors {
		bScore := bSum.VectorScores[vec]
		tScore := tSum.VectorScores[vec]
		diff.VectorDeltas[vec] = tScore - bScore
		if tScore < bScore-5.0 {
			diff.HasRegressions = true
		}
	}

	// Compare individual test case regressions
	bCases := make(map[string]bool)
	for _, ev := range baseline.Evaluations {
		bCases[ev.TestCase.ID] = ev.Passed
	}

	for _, ev := range target.Evaluations {
		id := ev.TestCase.ID
		bPassed, existed := bCases[id]
		if existed {
			if bPassed && !ev.Passed {
				diff.RegressedCases = append(diff.RegressedCases, fmt.Sprintf("%s (%s): %s", id, ev.TestCase.Vector, ev.FailureReason))
				diff.HasRegressions = true
			} else if !bPassed && ev.Passed {
				diff.ResolvedCases = append(diff.ResolvedCases, id)
			}
		}
	}

	return diff, nil
}

// RenderDiffTerminal prints a formatted regression diff between two benchmark runs.
func RenderDiffTerminal(w io.Writer, diff *DiffResult) {
	fmt.Fprintf(w, "\n=== ragfuzz :: Baseline Comparison Diff ======================================\n")
	fmt.Fprintf(w, "  Metric                  | Baseline | Current  | Delta\n")
	fmt.Fprintf(w, "  ------------------------+----------+----------+----------------------------\n")
	fmt.Fprintf(w, "  Pass Rate               | %7.1f%% | %7.1f%% | %s\n",
		diff.BaselineSummary.PassRate*100, diff.TargetSummary.PassRate*100, formatDeltaPct(diff.PassRateDelta*100))
	fmt.Fprintf(w, "  Groundedness (NLI)      | %7.1f%% | %7.1f%% | %s\n",
		diff.BaselineSummary.MeanGroundedness*100, diff.TargetSummary.MeanGroundedness*100, formatDeltaPct(diff.GroundednessDelta*100))
	fmt.Fprintf(w, "  Retrieval Recall@K      | %7.1f%% | %7.1f%% | %s\n",
		diff.BaselineSummary.MeanRecallMutated*100, diff.TargetSummary.MeanRecallMutated*100, formatDeltaPct(diff.RecallDelta*100))
	fmt.Fprintf(w, "  Refusal Calibration F1  | %8.2f | %8.2f | %s\n",
		diff.BaselineSummary.RefusalF1, diff.TargetSummary.RefusalF1, formatDeltaFloat(diff.RefusalF1Delta))

	fmt.Fprintf(w, "\n--- Vector Robustness Deltas -------------------------------------------------\n")
	for _, vec := range fuzzer.AllVectors {
		delta := diff.VectorDeltas[vec]
		status := "UNCHANGED"
		if delta < -5.0 {
			status = "REGRESSION"
		} else if delta > 5.0 {
			status = "IMPROVED"
		}
		fmt.Fprintf(w, "  %-30s : %s [%s]\n", vec, formatDeltaPct(delta), status)
	}

	if len(diff.RegressedCases) > 0 {
		fmt.Fprintf(w, "\n[!] Regressed Test Cases (%d):\n", len(diff.RegressedCases))
		for _, rc := range diff.RegressedCases {
			fmt.Fprintf(w, "    * %s\n", rc)
		}
	}

	if len(diff.ResolvedCases) > 0 {
		fmt.Fprintf(w, "\n[+] Resolved Test Cases (%d):\n", len(diff.ResolvedCases))
		for _, rc := range diff.ResolvedCases {
			fmt.Fprintf(w, "    * %s\n", rc)
		}
	}
	fmt.Fprintf(w, "==============================================================================\n\n")
}

func formatDeltaPct(delta float64) string {
	if delta > 0.1 {
		return fmt.Sprintf("+%.1f%%", delta)
	}
	if delta < -0.1 {
		return fmt.Sprintf("%.1f%%", delta)
	}
	return " 0.0%"
}

func formatDeltaFloat(delta float64) string {
	if delta > 0.005 {
		return fmt.Sprintf("+%.2f", delta)
	}
	if delta < -0.005 {
		return fmt.Sprintf("%.2f", delta)
	}
	return " 0.00"
}
