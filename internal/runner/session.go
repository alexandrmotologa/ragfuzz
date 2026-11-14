package runner

import (
	"sort"
	"sync"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

// SessionState tracks the runtime progress and accumulated results of a fuzzing execution.
type SessionState struct {
	mu           sync.RWMutex
	Total        int                      `json:"total"`
	Completed    int                      `json:"completed"`
	Passed       int                      `json:"passed"`
	Failed       int                      `json:"failed"`
	StartTime    time.Time                `json:"start_time"`
	DurationMs   int64                    `json:"duration_ms"`
	Evaluations  []*oracle.CaseEvaluation `json:"evaluations"`
	VectorPass   map[fuzzer.AttackVector]int `json:"vector_pass"`
	VectorTotal  map[fuzzer.AttackVector]int `json:"vector_total"`
	DistractorHits int                   `json:"distractor_hits"`
	Latencies    []time.Duration          `json:"-"`
	IsRunning    bool                     `json:"is_running"`
	Thresholds   config.Thresholds        `json:"thresholds"`
	Summary      *oracle.SuiteSummary     `json:"summary,omitempty"`
}

func NewSessionState(total int, th config.Thresholds) *SessionState {
	return &SessionState{
		Total:       total,
		StartTime:   time.Now(),
		VectorPass:  make(map[fuzzer.AttackVector]int),
		VectorTotal: make(map[fuzzer.AttackVector]int),
		IsRunning:   true,
		Thresholds:  th,
	}
}

func (s *SessionState) RecordEvaluation(eval *oracle.CaseEvaluation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Completed++
	s.Evaluations = append(s.Evaluations, eval)
	s.VectorTotal[eval.TestCase.Vector]++

	if eval.Passed {
		s.Passed++
		s.VectorPass[eval.TestCase.Vector]++
	} else {
		s.Failed++
	}

	if eval.DistractorIntruded {
		s.DistractorHits++
	}

	s.Latencies = append(s.Latencies, eval.Latency)
	s.DurationMs = time.Since(s.StartTime).Milliseconds()
}

func (s *SessionState) Finish() *oracle.SuiteSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.IsRunning = false
	s.DurationMs = time.Since(s.StartTime).Milliseconds()

	summary := &oracle.SuiteSummary{
		TotalQueries:         s.Total,
		PassedQueries:        s.Passed,
		FailedQueries:        s.Failed,
		DistractorIntrusions: s.DistractorHits,
		ExecutedAt:           s.StartTime,
		DurationMs:           s.DurationMs,
		VectorScores:         make(map[fuzzer.AttackVector]float64),
	}

	if s.Completed > 0 {
		summary.PassRate = float64(s.Passed) / float64(s.Completed)
	}

	// Compute averages
	var sumGroundedness, sumRecallClean, sumRecallMutated, sumMRR float64
	var sumLatencyMs float64
	oodTrueRefusals := 0
	oodTotal := 0
	inDomainFalseRefusals := 0
	inDomainTotal := 0

	for _, ev := range s.Evaluations {
		sumGroundedness += ev.Groundedness
		sumRecallClean += ev.CleanRecallAtK
		sumRecallMutated += ev.MutatedRecallAtK
		sumMRR += ev.MRR
		sumLatencyMs += float64(ev.Latency.Milliseconds())

		if ev.TestCase.Vector == fuzzer.VectorOOD {
			oodTotal++
			if ev.IsRefusal {
				oodTrueRefusals++
			}
		} else {
			inDomainTotal++
			if ev.IsRefusal {
				inDomainFalseRefusals++
			}
		}
	}

	n := float64(len(s.Evaluations))
	if n > 0 {
		summary.MeanGroundedness = sumGroundedness / n
		summary.MeanRecallClean = sumRecallClean / n
		summary.MeanRecallMutated = sumRecallMutated / n
		summary.RecallDegradation = summary.MeanRecallClean - summary.MeanRecallMutated
		summary.MeanMRR = sumMRR / n
		summary.HallucinationRate = 1.0 - summary.MeanGroundedness
		summary.MeanLatencyMs = sumLatencyMs / n
	}

	// Refusal Calibration metrics
	if oodTotal > 0 {
		summary.RefusalRecall = float64(oodTrueRefusals) / float64(oodTotal)
		totalRefusalsReported := oodTrueRefusals + inDomainFalseRefusals
		if totalRefusalsReported > 0 {
			summary.RefusalPrecision = float64(oodTrueRefusals) / float64(totalRefusalsReported)
		} else {
			summary.RefusalPrecision = 1.0
		}
		if summary.RefusalPrecision+summary.RefusalRecall > 0 {
			summary.RefusalF1 = 2.0 * (summary.RefusalPrecision * summary.RefusalRecall) / (summary.RefusalPrecision + summary.RefusalRecall)
		}
	} else {
		summary.RefusalF1 = 1.0
	}

	// Percentile Latencies
	if len(s.Latencies) > 0 {
		sortedLats := make([]float64, len(s.Latencies))
		for i, l := range s.Latencies {
			sortedLats[i] = float64(l.Milliseconds())
		}
		sort.Float64s(sortedLats)
		p95Idx := int(float64(len(sortedLats)) * 0.95)
		if p95Idx >= len(sortedLats) {
			p95Idx = len(sortedLats) - 1
		}
		summary.P95LatencyMs = sortedLats[p95Idx]
	}

	// Per-vector scores (0.0 to 100.0)
	for _, vec := range fuzzer.AllVectors {
		tot := s.VectorTotal[vec]
		pass := s.VectorPass[vec]
		if tot > 0 {
			summary.VectorScores[vec] = (float64(pass) / float64(tot)) * 100.0
		} else {
			summary.VectorScores[vec] = 100.0
		}
	}

	// Evaluate CI/CD Quality Gates
	gatePassed := true
	var breaches []string

	if summary.MeanGroundedness < s.Thresholds.MinGroundedness {
		gatePassed = false
		breaches = append(breaches, "Mean Groundedness below threshold")
	}
	if summary.RecallDegradation > s.Thresholds.MaxRecallDegradation {
		gatePassed = false
		breaches = append(breaches, "Recall Degradation exceeded threshold")
	}
	if summary.RefusalF1 < s.Thresholds.MinRefusalF1 {
		gatePassed = false
		breaches = append(breaches, "Refusal F1 Calibration below threshold")
	}
	if summary.HallucinationRate > s.Thresholds.MaxHallucinationRate {
		gatePassed = false
		breaches = append(breaches, "Hallucination Rate exceeded threshold")
	}

	summary.GatePassed = gatePassed
	summary.GateBreaches = breaches
	s.Summary = summary

	return summary
}
