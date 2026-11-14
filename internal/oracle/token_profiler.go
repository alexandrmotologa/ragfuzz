package oracle

import (
	"strings"
)

// TokenOverheadStats records token count inflation and estimated financial overhead.
type TokenOverheadStats struct {
	CleanTokens          int     `json:"clean_tokens"`
	MutatedTokens        int     `json:"mutated_tokens"`
	OverheadTokens       int     `json:"overhead_tokens"`
	InflationRatio       float64 `json:"inflation_ratio"`
	EstimatedCostUSDPerM float64 `json:"estimated_cost_usd_per_m"`
}

// EstimateTokens calculates an approximate token count from text using character and word heuristics.
func EstimateTokens(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	words := len(strings.Fields(trimmed))
	chars := len([]rune(trimmed))

	// Approximate standard BPE tokens (4 chars/token or 1.3 tokens/word)
	tokenEstimate := (words*4 + chars) / 7
	if tokenEstimate < 1 {
		tokenEstimate = 1
	}
	return tokenEstimate
}

// ComputeTokenOverhead analyzes context chunk inflation between clean and mutated executions.
func ComputeTokenOverhead(cleanChunks, mutatedChunks []RetrievedChunk) TokenOverheadStats {
	cleanTotal := 0
	for _, c := range cleanChunks {
		cleanTotal += EstimateTokens(c.Text)
	}

	mutatedTotal := 0
	for _, c := range mutatedChunks {
		mutatedTotal += EstimateTokens(c.Text)
	}

	overhead := mutatedTotal - cleanTotal
	inflationRatio := 0.0
	if cleanTotal > 0 && overhead > 0 {
		inflationRatio = float64(overhead) / float64(cleanTotal)
	}

	// Based on $2.50 per 1M input tokens
	costPerM := (float64(overhead) * 2.50)

	return TokenOverheadStats{
		CleanTokens:          cleanTotal,
		MutatedTokens:        mutatedTotal,
		OverheadTokens:       overhead,
		InflationRatio:       inflationRatio,
		EstimatedCostUSDPerM: costPerM,
	}
}
