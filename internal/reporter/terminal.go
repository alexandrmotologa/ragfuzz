package reporter

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

// ANSI color escape codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// RenderTerminal writes a formatted evaluation report and scorecard to the given writer.
func RenderTerminal(w io.Writer, session *runner.SessionState) {
	if w == nil {
		w = os.Stdout
	}

	summary := session.Summary
	if summary == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%s================================================================================%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(w, "%s%s                    ragfuzz :: Adversarial Stress Evaluation                    %s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(w, "%s%s================================================================================%s\n", colorBold, colorCyan, colorReset)
	fmt.Fprintf(w, "  Total Queries:     %s%d%s\n", colorBold, summary.TotalQueries, colorReset)
	fmt.Fprintf(w, "  Passed:            %s%d%s\n", colorGreen, summary.PassedQueries, colorReset)
	fmt.Fprintf(w, "  Failed:            %s%d%s\n", colorRed, summary.FailedQueries, colorReset)
	fmt.Fprintf(w, "  Execution Time:    %dms (mean latency: %.1fms, p95: %.1fms)\n", summary.DurationMs, summary.MeanLatencyMs, summary.P95LatencyMs)
	fmt.Fprintln(w)

	// Quality Scorecard Table
	fmt.Fprintf(w, "%s%s--- Evaluation Metrics Scorecard -----------------------------------------------%s\n", colorBold, colorBlue, colorReset)
	fmt.Fprintf(w, "  Metric                  | Value     | Target Threshold | Status\n")
	fmt.Fprintf(w, "  ------------------------+-----------+------------------+---------\n")

	renderMetricRow(w, "Groundedness (NLI)", fmt.Sprintf("%.2f%%", summary.MeanGroundedness*100), fmt.Sprintf(">= %.2f%%", session.Thresholds.MinGroundedness*100), summary.MeanGroundedness >= session.Thresholds.MinGroundedness)
	renderMetricRow(w, "Recall Degradation", fmt.Sprintf("%.2f%%", summary.RecallDegradation*100), fmt.Sprintf("<= %.2f%%", session.Thresholds.MaxRecallDegradation*100), summary.RecallDegradation <= session.Thresholds.MaxRecallDegradation)
	renderMetricRow(w, "Refusal Calibration F1", fmt.Sprintf("%.2f", summary.RefusalF1), fmt.Sprintf(">= %.2f", session.Thresholds.MinRefusalF1), summary.RefusalF1 >= session.Thresholds.MinRefusalF1)
	renderMetricRow(w, "Hallucination Rate", fmt.Sprintf("%.2f%%", summary.HallucinationRate*100), fmt.Sprintf("<= %.2f%%", session.Thresholds.MaxHallucinationRate*100), summary.HallucinationRate <= session.Thresholds.MaxHallucinationRate)
	renderMetricRow(w, "Distractor Intrusions", fmt.Sprintf("%d", summary.DistractorIntrusions), "0", summary.DistractorIntrusions == 0)

	fmt.Fprintln(w)

	// 6-Axis Attack Vector Breakdown
	fmt.Fprintf(w, "%s%s--- 6-Axis Attack Vector Breakdown ---------------------------------------------%s\n", colorBold, colorBlue, colorReset)
	fmt.Fprintf(w, "  Vector                         | Total | Pass  | Fail  | Robustness | Status\n")
	fmt.Fprintf(w, "  -------------------------------+-------+-------+-------+------------+---------\n")

	for _, vec := range fuzzer.AllVectors {
		tot := session.VectorTotal[vec]
		pass := session.VectorPass[vec]
		fail := tot - pass
		score := summary.VectorScores[vec]

		status := fmt.Sprintf("%sPASS%s", colorGreen, colorReset)
		if score < 75.0 {
			status = fmt.Sprintf("%sFAIL%s", colorRed, colorReset)
		} else if score < 90.0 {
			status = fmt.Sprintf("%sWARN%s", colorYellow, colorReset)
		}

		fmt.Fprintf(w, "  %-30s | %-5d | %-5d | %-5d | %-9.1f%% | %s\n",
			formatVectorName(vec), tot, pass, fail, score, status)
	}

	fmt.Fprintln(w)

	// CI/CD Quality Gate Outcome
	if summary.GatePassed {
		fmt.Fprintf(w, "%s%s[+] CI/CD QUALITY GATE: PASSED%s (All thresholds satisfied)\n", colorBold, colorGreen, colorReset)
	} else {
		fmt.Fprintf(w, "%s%s[-] CI/CD QUALITY GATE: FAILED%s (Regression thresholds breached)\n", colorBold, colorRed, colorReset)
		for _, b := range summary.GateBreaches {
			fmt.Fprintf(w, "    %s* %s%s\n", colorRed, b, colorReset)
		}
	}
	fmt.Fprintf(w, "%s%s================================================================================%s\n\n", colorBold, colorCyan, colorReset)
}

func renderMetricRow(w io.Writer, name, val, target string, passed bool) {
	status := fmt.Sprintf("%sPASS%s", colorGreen, colorReset)
	if !passed {
		status = fmt.Sprintf("%sFAIL%s", colorRed, colorReset)
	}
	fmt.Fprintf(w, "  %-23s | %-9s | %-16s | %s\n", name, val, target, status)
}

func formatVectorName(v fuzzer.AttackVector) string {
	parts := strings.Split(string(v), "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
