package reporter

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/runner"
)

// WriteHTMLReport creates an interactive, single-file HTML report with embedded SVG radar charts.
func WriteHTMLReport(filePath string, session *runner.SessionState) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	summary := session.Summary
	if summary == nil {
		return fmt.Errorf("session summary is nil")
	}

	radarSVG := generateRadarSVG(summary.VectorScores)

	gateBadge := `<span style="background: rgba(16, 185, 129, 0.2); color: #10b981; border: 1px solid #10b981; padding: 4px 12px; border-radius: 9999px; font-weight: 600; font-size: 0.875rem;">PASSED</span>`
	if !summary.GatePassed {
		gateBadge = `<span style="background: rgba(239, 68, 68, 0.2); color: #ef4444; border: 1px solid #ef4444; padding: 4px 12px; border-radius: 9999px; font-weight: 600; font-size: 0.875rem;">FAILED</span>`
	}

	var breachesHTML strings.Builder
	if len(summary.GateBreaches) > 0 {
		breachesHTML.WriteString(`<div style="margin-top: 1rem; padding: 0.75rem 1rem; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 8px;">`)
		breachesHTML.WriteString(`<div style="color: #ef4444; font-weight: 600; margin-bottom: 0.25rem;">Quality Gate Breaches:</div><ul style="margin: 0; padding-left: 1.25rem; color: #fca5a5; font-size: 0.875rem;">`)
		for _, b := range summary.GateBreaches {
			breachesHTML.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(b)))
		}
		breachesHTML.WriteString(`</ul></div>`)
	}

	var rowsHTML strings.Builder
	for idx, ev := range session.Evaluations {
		statusPill := `<span style="color: #10b981; font-weight: 600;">PASS</span>`
		rowBg := "rgba(15, 23, 42, 0.4)"
		if !ev.Passed {
			statusPill = `<span style="color: #ef4444; font-weight: 600;">FAIL</span>`
			rowBg = "rgba(239, 68, 68, 0.05)"
		}

		vectorClass := string(ev.TestCase.Vector)
		cleanQuery := html.EscapeString(ev.TestCase.CleanQuery)
		mutatedQuery := html.EscapeString(ev.TestCase.MutatedQuery)
		ans := html.EscapeString(ev.Response.Answer)
		reason := html.EscapeString(ev.FailureReason)

		var chunksHTML strings.Builder
		for rIdx, c := range ev.Response.RetrievedChunks {
			chunksHTML.WriteString(fmt.Sprintf(`<div style="font-size: 0.75rem; background: #0f172a; padding: 6px 10px; border-radius: 4px; margin-top: 4px; border: 1px solid #334155;"><strong>Rank %d (%s):</strong> %s</div>`,
				rIdx+1, html.EscapeString(c.ID), html.EscapeString(c.Text)))
		}

		rowsHTML.WriteString(fmt.Sprintf(`
		<tr class="case-row" data-vector="%s" data-status="%v" style="background: %s; border-bottom: 1px solid #1e293b;">
			<td style="padding: 10px 14px; font-mono; font-size: 0.8rem; color: #94a3b8;">#%04d</td>
			<td style="padding: 10px 14px;"><span style="background: #1e293b; color: #38bdf8; padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; font-family: monospace;">%s</span></td>
			<td style="padding: 10px 14px; max-width: 380px;">
				<div style="font-weight: 500; color: #f1f5f9;">%s</div>
				<div style="font-size: 0.75rem; color: #64748b; margin-top: 2px;">Baseline: %s</div>
				%s
			</td>
			<td style="padding: 10px 14px; text-align: center;">%.2f</td>
			<td style="padding: 10px 14px; text-align: center;">%.2f</td>
			<td style="padding: 10px 14px; text-align: center;">%dms</td>
			<td style="padding: 10px 14px; text-align: center;">%s</td>
		</tr>
		<tr class="details-row" style="background: #090d16; border-bottom: 1px solid #1e293b;">
			<td colspan="7" style="padding: 10px 18px;">
				<details>
					<summary style="font-size: 0.8rem; color: #38bdf8; cursor: pointer; user-select: none;">View Response & Retrieved Context Details</summary>
					<div style="margin-top: 8px; display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
						<div>
							<div style="font-size: 0.75rem; font-weight: 600; color: #94a3b8; text-transform: uppercase;">Generated Answer:</div>
							<div style="font-size: 0.8rem; color: #cbd5e1; background: #0f172a; padding: 10px; border-radius: 6px; margin-top: 4px; border: 1px solid #334155;">%s</div>
							%s
						</div>
						<div>
							<div style="font-size: 0.75rem; font-weight: 600; color: #94a3b8; text-transform: uppercase;">Retrieved Context Chunks (%d):</div>
							%s
						</div>
					</div>
				</details>
			</td>
		</tr>
		`, vectorClass, ev.Passed, rowBg, idx+1, html.EscapeString(string(ev.TestCase.Vector)), mutatedQuery, cleanQuery,
			renderFailureInline(reason), ev.MutatedRecallAtK, ev.Groundedness, ev.Latency.Milliseconds(), statusPill, ans, renderReasonBox(reason), len(ev.Response.RetrievedChunks), chunksHTML.String()))
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>ragfuzz Evaluation Report</title>
	<style>
		* { box-sizing: border-box; }
		body {
			margin: 0;
			background: #0b0f19;
			color: #e2e8f0;
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			font-size: 14px;
			line-height: 1.5;
		}
		.container {
			max-width: 1300px;
			margin: 0 auto;
			padding: 2rem 1.5rem;
		}
		.header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding-bottom: 1.5rem;
			border-bottom: 1px solid #1e293b;
		}
		.title {
			font-size: 1.75rem;
			font-weight: 700;
			color: #f8fafc;
			letter-spacing: -0.025em;
		}
		.subtitle {
			font-size: 0.875rem;
			color: #64748b;
			margin-top: 0.25rem;
		}
		.grid-cards {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 1rem;
			margin-top: 1.5rem;
		}
		.card {
			background: #111827;
			border: 1px solid #1f2937;
			border-radius: 8px;
			padding: 1.25rem;
		}
		.card-label {
			font-size: 0.75rem;
			text-transform: uppercase;
			color: #94a3b8;
			letter-spacing: 0.05em;
		}
		.card-value {
			font-size: 1.75rem;
			font-weight: 700;
			color: #f1f5f9;
			margin-top: 0.25rem;
		}
		.card-target {
			font-size: 0.75rem;
			color: #64748b;
			margin-top: 0.25rem;
		}
		.radar-section {
			display: grid;
			grid-template-columns: 420px 1fr;
			gap: 1.5rem;
			margin-top: 1.5rem;
		}
		.filter-bar {
			display: flex;
			gap: 0.75rem;
			margin-top: 2rem;
			margin-bottom: 1rem;
		}
		.filter-btn {
			background: #1e293b;
			color: #94a3b8;
			border: 1px solid #334155;
			border-radius: 6px;
			padding: 6px 14px;
			font-size: 0.8rem;
			cursor: pointer;
		}
		.filter-btn.active {
			background: #38bdf8;
			color: #0b0f19;
			font-weight: 600;
			border-color: #38bdf8;
		}
		table {
			width: 100%%;
			border-collapse: collapse;
			text-align: left;
			background: #111827;
			border: 1px solid #1f2937;
			border-radius: 8px;
			overflow: hidden;
		}
		th {
			background: #1e293b;
			color: #94a3b8;
			font-size: 0.75rem;
			text-transform: uppercase;
			padding: 10px 14px;
			font-weight: 600;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<div>
				<div class="title">ragfuzz <span style="font-size: 1rem; color: #38bdf8; font-weight: normal; margin-left: 8px;">Adversarial Stress Report</span></div>
				<div class="subtitle">Evaluated on %s | Duration: %dms | %d total queries</div>
			</div>
			<div>
				%s
			</div>
		</div>

		%s

		<div class="grid-cards">
			<div class="card">
				<div class="card-label">Pass Rate</div>
				<div class="card-value" style="color: %s;">%.1f%%</div>
				<div class="card-target">%d passed / %d failed</div>
			</div>
			<div class="card">
				<div class="card-label">Groundedness (NLI)</div>
				<div class="card-value">%.1f%%</div>
				<div class="card-target">Threshold: >= %.1f%%</div>
			</div>
			<div class="card">
				<div class="card-label">Recall Degradation</div>
				<div class="card-value">%.1f%%</div>
				<div class="card-target">Max Allowed: <= %.1f%%</div>
			</div>
			<div class="card">
				<div class="card-label">Refusal Calibration</div>
				<div class="card-value">%.2f</div>
				<div class="card-target">F1 Score (Target: >= %.2f)</div>
			</div>
			<div class="card">
				<div class="card-label">Hallucination Rate</div>
				<div class="card-value">%.1f%%</div>
				<div class="card-target">Threshold: <= %.1f%%</div>
			</div>
		</div>

		<div class="radar-section">
			<div class="card" style="text-align: center;">
				<div class="card-label" style="margin-bottom: 1rem;">6-Axis Vulnerability Radar</div>
				%s
			</div>
			<div class="card">
				<div class="card-label" style="margin-bottom: 1rem;">Vector Robustness Breakdown</div>
				%s
			</div>
		</div>

		<div class="filter-bar">
			<button class="filter-btn active" onclick="filterCases('all', this)">All Vectors (%d)</button>
			<button class="filter-btn" onclick="filterCases('failed', this)">Failures Only (%d)</button>
			<button class="filter-btn" onclick="filterCases('negation_inversion', this)">Negation</button>
			<button class="filter-btn" onclick="filterCases('semantic_distractor', this)">Distractor</button>
			<button class="filter-btn" onclick="filterCases('entity_version_collision', this)">Entity / Version</button>
			<button class="filter-btn" onclick="filterCases('orthographic_noise', this)">Typo Noise</button>
			<button class="filter-btn" onclick="filterCases('contradiction_trap', this)">Contradiction</button>
			<button class="filter-btn" onclick="filterCases('out_of_domain', this)">Out-of-Domain</button>
		</div>

		<table>
			<thead>
				<tr>
					<th style="width: 70px;">ID</th>
					<th style="width: 170px;">Vector</th>
					<th>Query (Mutated & Baseline)</th>
					<th style="width: 90px; text-align: center;">Recall@K</th>
					<th style="width: 100px; text-align: center;">Grounded</th>
					<th style="width: 80px; text-align: center;">Latency</th>
					<th style="width: 80px; text-align: center;">Status</th>
				</tr>
			</thead>
			<tbody id="cases-tbody">
				%s
			</tbody>
		</table>
	</div>

	<script>
		function filterCases(filter, btn) {
			document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
			btn.classList.add('active');

			const rows = document.querySelectorAll('#cases-tbody tr.case-row');
			rows.forEach(r => {
				const details = r.nextElementSibling;
				const vec = r.getAttribute('data-vector');
				const passed = r.getAttribute('data-status') === 'true';

				let show = false;
				if (filter === 'all') {
					show = true;
				} else if (filter === 'failed') {
					show = !passed;
				} else if (filter === vec) {
					show = true;
				}

				r.style.display = show ? '' : 'none';
				if (details && details.classList.contains('details-row')) {
					details.style.display = show ? '' : 'none';
				}
			});
		}
	</script>
</body>
</html>`,
		summary.ExecutedAt.Format("2006-01-02 15:04:05 MST"), summary.DurationMs, summary.TotalQueries,
		gateBadge, breachesHTML.String(),
		getPassColor(summary.PassRate), summary.PassRate*100, summary.PassedQueries, summary.FailedQueries,
		summary.MeanGroundedness*100, session.Thresholds.MinGroundedness*100,
		summary.RecallDegradation*100, session.Thresholds.MaxRecallDegradation*100,
		summary.RefusalF1, session.Thresholds.MinRefusalF1,
		summary.HallucinationRate*100, session.Thresholds.MaxHallucinationRate*100,
		radarSVG,
		generateVectorBars(session),
		session.Total, session.Failed,
		rowsHTML.String(),
	)

	return os.WriteFile(filePath, []byte(htmlContent), 0644)
}

func getPassColor(rate float64) string {
	if rate >= 0.90 {
		return "#10b981"
	}
	if rate >= 0.75 {
		return "#f59e0b"
	}
	return "#ef4444"
}

func renderFailureInline(reason string) string {
	if reason == "" {
		return ""
	}
	return fmt.Sprintf(`<div style="font-size: 0.75rem; color: #ef4444; margin-top: 4px;"><strong>Failure:</strong> %s</div>`, reason)
}

func renderReasonBox(reason string) string {
	if reason == "" {
		return ""
	}
	return fmt.Sprintf(`<div style="margin-top: 8px; font-size: 0.75rem; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); color: #fca5a5; padding: 6px 10px; border-radius: 4px;"><strong>Vulnerability Reason:</strong> %s</div>`, reason)
}

func generateVectorBars(session *runner.SessionState) string {
	var sb strings.Builder
	for _, vec := range fuzzer.AllVectors {
		tot := session.VectorTotal[vec]
		pass := session.VectorPass[vec]
		score := session.Summary.VectorScores[vec]

		color := "#10b981"
		if score < 75.0 {
			color = "#ef4444"
		} else if score < 90.0 {
			color = "#f59e0b"
		}

		sb.WriteString(fmt.Sprintf(`
		<div style="margin-bottom: 0.75rem;">
			<div style="display: flex; justify-content: space-between; font-size: 0.8rem; margin-bottom: 0.25rem;">
				<span style="color: #cbd5e1;">%s</span>
				<span style="font-weight: 600; color: %s;">%.1f%% (%d/%d)</span>
			</div>
			<div style="height: 8px; background: #1e293b; border-radius: 4px; overflow: hidden;">
				<div style="height: 100%%; width: %.1f%%; background: %s; border-radius: 4px;"></div>
			</div>
		</div>
		`, formatVectorName(vec), color, score, pass, tot, score, color))
	}
	return sb.String()
}

func generateRadarSVG(scores map[fuzzer.AttackVector]float64) string {
	cx, cy, r := 180.0, 160.0, 100.0
	numAxes := len(fuzzer.AllVectors)

	var gridLines strings.Builder
	// Rings at 25%, 50%, 75%, 100%
	for _, pct := range []float64{0.25, 0.50, 0.75, 1.0} {
		var points []string
		for i := 0; i < numAxes; i++ {
			angle := float64(i)*(2*math.Pi/float64(numAxes)) - math.Pi/2
			px := cx + (r*pct)*math.Cos(angle)
			py := cy + (r*pct)*math.Sin(angle)
			points = append(points, fmt.Sprintf("%.1f,%.1f", px, py))
		}
		gridLines.WriteString(fmt.Sprintf(`<polygon points="%s" fill="none" stroke="#1f2937" stroke-width="1"/>`, strings.Join(points, " ")))
	}

	var axisLines strings.Builder
	var dataPoints []string
	labels := []string{"Negation", "Distractor", "Entity Swap", "Typo Noise", "Contradiction", "Out of Domain"}

	for i, vec := range fuzzer.AllVectors {
		angle := float64(i)*(2*math.Pi/float64(numAxes)) - math.Pi/2
		ax := cx + r*math.Cos(angle)
		ay := cy + r*math.Sin(angle)
		axisLines.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#334155" stroke-dasharray="2,2"/>`, cx, cy, ax, ay))

		score := scores[vec]
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		ratio := score / 100.0
		dx := cx + (r*ratio)*math.Cos(angle)
		dy := cy + (r*ratio)*math.Sin(angle)
		dataPoints = append(dataPoints, fmt.Sprintf("%.1f,%.1f", dx, dy))

		// Label position
		lx := cx + (r+24)*math.Cos(angle)
		ly := cy + (r+24)*math.Sin(angle)
		textAnchor := "middle"
		if math.Abs(math.Cos(angle)) > 0.2 {
			if math.Cos(angle) > 0 {
				textAnchor = "start"
			} else {
				textAnchor = "end"
			}
		}
		axisLines.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="#94a3b8" font-size="10" text-anchor="%s" alignment-baseline="middle">%s</text>`,
			lx, ly, textAnchor, labels[i]))
	}

	polygonStr := strings.Join(dataPoints, " ")

	return fmt.Sprintf(`
	<svg width="360" height="320" viewBox="0 0 360 320" xmlns="http://www.w3.org/2000/svg">
		%s
		%s
		<polygon points="%s" fill="rgba(56, 189, 248, 0.25)" stroke="#38bdf8" stroke-width="2"/>
	</svg>
	`, gridLines.String(), axisLines.String(), polygonStr)
}
