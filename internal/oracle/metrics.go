package oracle

import (
	"strings"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
)

// CaseEvaluation encapsulates evaluation results for an individual test case.
type CaseEvaluation struct {
	TestCase           *fuzzer.TestCase      `json:"test_case"`
	Response           *RAGResponse          `json:"response"`
	CleanRecallAtK     float64               `json:"clean_recall_at_k"`
	MutatedRecallAtK   float64               `json:"mutated_recall_at_k"`
	RecallDelta        float64               `json:"recall_delta"`
	MRR                float64               `json:"mrr"`
	Groundedness       float64               `json:"groundedness"`
	HallucinationRate  float64               `json:"hallucination_rate"`
	DistractorIntruded bool                  `json:"distractor_intruded"`
	IsRefusal          bool                  `json:"is_refusal"`
	Passed             bool                  `json:"passed"`
	FailureReason      string                `json:"failure_reason,omitempty"`
	NLI                *NLIResult            `json:"nli,omitempty"`
	TokenOverhead      TokenOverheadStats    `json:"token_overhead"`
	Latency            time.Duration         `json:"latency"`
}

// SuiteSummary holds aggregated metrics for a full fuzz run across all vectors.
type SuiteSummary struct {
	TotalQueries          int                        `json:"total_queries"`
	PassedQueries         int                        `json:"passed_queries"`
	FailedQueries         int                        `json:"failed_queries"`
	PassRate              float64                    `json:"pass_rate"`
	MeanGroundedness      float64                    `json:"mean_groundedness"`
	MeanRecallClean       float64                    `json:"mean_recall_clean"`
	MeanRecallMutated     float64                    `json:"mean_recall_mutated"`
	RecallDegradation     float64                    `json:"recall_degradation"`
	MeanMRR               float64                    `json:"mean_mrr"`
	HallucinationRate     float64                    `json:"hallucination_rate"`
	RefusalPrecision      float64                    `json:"refusal_precision"`
	RefusalRecall         float64                    `json:"refusal_recall"`
	RefusalF1             float64                    `json:"refusal_f1"`
	DistractorIntrusions  int                        `json:"distractor_intrusions"`
	TokenOverhead         TokenOverheadStats         `json:"token_overhead"`
	MeanLatencyMs         float64                    `json:"mean_latency_ms"`
	P95LatencyMs          float64                    `json:"p95_latency_ms"`
	VectorScores          map[fuzzer.AttackVector]float64 `json:"vector_scores"`
	GatePassed            bool                       `json:"gate_passed"`
	GateBreaches          []string                   `json:"gate_breaches,omitempty"`
	ExecutedAt            time.Time                  `json:"executed_at"`
	DurationMs            int64                      `json:"duration_ms"`
}

// EvaluateCase assesses a single query execution against retrieval and generation criteria.
func EvaluateCase(tc *fuzzer.TestCase, resp *RAGResponse, topK int, minGroundedness float64) *CaseEvaluation {
	if topK <= 0 {
		topK = 5
	}
	if minGroundedness <= 0 {
		minGroundedness = 0.85
	}

	eval := &CaseEvaluation{
		TestCase: tc,
		Response: resp,
		Latency:  resp.Latency,
	}

	if resp.Error != "" {
		eval.Passed = false
		eval.FailureReason = "RAG endpoint error: " + resp.Error
		return eval
	}

	// 1. Evaluate Retrieval
	retrievedIDs := make([]string, 0, len(resp.RetrievedChunks))
	var contextBuilder strings.Builder
	for idx, chunk := range resp.RetrievedChunks {
		if idx < topK {
			retrievedIDs = append(retrievedIDs, chunk.ID)
		}
		contextBuilder.WriteString(chunk.Text)
		contextBuilder.WriteString("\n")
	}

	retrievedSet := make(map[string]bool, len(retrievedIDs))
	for _, id := range retrievedIDs {
		retrievedSet[id] = true
	}

	// Calculate Recall@K
	eval.MutatedRecallAtK = computeRecall(retrievedSet, tc.ExpectedChunkIDs)
	eval.CleanRecallAtK = 1.0 // Ideal clean retrieval baseline

	// For OOD queries, expected chunks are empty
	if tc.Vector == fuzzer.VectorOOD {
		eval.CleanRecallAtK = 1.0
		eval.MutatedRecallAtK = 1.0
	}
	eval.RecallDelta = eval.CleanRecallAtK - eval.MutatedRecallAtK

	// Calculate MRR
	eval.MRR = computeMRR(retrievedIDs, tc.ExpectedChunkIDs)

	// Compute token overhead
	eval.TokenOverhead = ComputeTokenOverhead(nil, resp.RetrievedChunks)

	// Check distractor intrusion
	for _, dID := range tc.DistractorChunkIDs {
		if retrievedSet[dID] {
			eval.DistractorIntruded = true
			break
		}
	}

	// 2. Evaluate Generation & Groundedness via NLI
	nli := EvaluateNLI(resp.Answer, contextBuilder.String())
	eval.NLI = nli
	eval.Groundedness = nli.Groundedness
	eval.HallucinationRate = 1.0 - nli.Groundedness
	eval.IsRefusal = nli.IsRefusal

	// 3. Determine pass / fail based on expected behavior
	switch tc.ExpectedBehavior {
	case fuzzer.BehaviorRefusal:
		if eval.IsRefusal {
			eval.Passed = true
		} else {
			eval.Passed = false
			eval.FailureReason = "Failed Refusal Calibration: generated ungrounded answer for out-of-domain query"
		}

	case fuzzer.BehaviorReconciliation:
		if eval.Groundedness >= minGroundedness {
			eval.Passed = true
		} else {
			eval.Passed = false
			eval.FailureReason = "Hallucinated reconciliation for contradictory chunks"
		}

	default: // BehaviorAnswer
		if eval.DistractorIntruded {
			eval.Passed = false
			eval.FailureReason = "Distractor chunk intruded into top-K retrieval"
		} else if eval.MutatedRecallAtK == 0 && len(tc.ExpectedChunkIDs) > 0 {
			eval.Passed = false
			eval.FailureReason = "Zero recall: retriever failed to return expected target chunk"
		} else if eval.Groundedness < minGroundedness {
			eval.Passed = false
			eval.FailureReason = "Groundedness below threshold (claims not entailed by retrieved context)"
		} else {
			eval.Passed = true
		}
	}

	return eval
}

func computeRecall(retrievedSet map[string]bool, expectedIDs []string) float64 {
	if len(expectedIDs) == 0 {
		return 1.0
	}
	matches := 0
	for _, exp := range expectedIDs {
		if retrievedSet[exp] {
			matches++
		}
	}
	return float64(matches) / float64(len(expectedIDs))
}

func computeMRR(retrievedIDs []string, expectedIDs []string) float64 {
	if len(expectedIDs) == 0 {
		return 1.0
	}
	expSet := make(map[string]bool, len(expectedIDs))
	for _, e := range expectedIDs {
		expSet[e] = true
	}

	for rank, id := range retrievedIDs {
		if expSet[id] {
			return 1.0 / float64(rank+1)
		}
	}
	return 0.0
}
