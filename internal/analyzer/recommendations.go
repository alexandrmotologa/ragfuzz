package analyzer

import (
	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

// Recommendation provides actionable advice for fixing specific RAG failure modes.
type Recommendation struct {
	Vector       fuzzer.AttackVector `json:"vector"`
	Score        float64             `json:"score"`
	Severity     string              `json:"severity"`
	Title        string              `json:"title"`
	ActionPlan   string              `json:"action_plan"`
	SuggestedFix string              `json:"suggested_fix"`
}

// GenerateRecommendations inspects the summary metrics and emits targeted remediation guidance.
func GenerateRecommendations(summary *oracle.SuiteSummary) []Recommendation {
	var recs []Recommendation

	// 1. Negation Inversion Vulnerability
	negScore := summary.VectorScores[fuzzer.VectorNegation]
	if negScore < 80.0 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorNegation,
			Score:        negScore,
			Severity:     getSeverity(negScore),
			Title:        "High Negation & Exclusion Blindness",
			ActionPlan:   "Embedding models frequently map affirmative and negative statements into adjacent vector spaces.",
			SuggestedFix: "Implement dense-sparse hybrid search with Reciprocal Rank Fusion (RRF), or add metadata boolean tags for excluded features.",
		})
	}

	// 2. Semantic Distractor Intrusion
	distScore := summary.VectorScores[fuzzer.VectorDistractor]
	if distScore < 80.0 || summary.DistractorIntrusions > 0 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorDistractor,
			Score:        distScore,
			Severity:     getSeverity(distScore),
			Title:        "Retriever Susceptible to Semantic Distractors",
			ActionPlan:   "Adjacent chunks with topical overlap are polluting top-K context, pushing out true ground-truth chunks.",
			SuggestedFix: "Integrate a Cross-Encoder Reranker (such as bge-reranker-large or Cohere Rerank) and tighten similarity cutoff from 0.70 to 0.78.",
		})
	}

	// 3. Entity & Version Collisions
	entScore := summary.VectorScores[fuzzer.VectorEntitySwap]
	if entScore < 80.0 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorEntitySwap,
			Score:        entScore,
			Severity:     getSeverity(entScore),
			Title:        "Temporal & Version Collapse in Retrieval",
			ActionPlan:   "Queries mentioning deprecated or newer versions are returning outdated specification chunks.",
			SuggestedFix: "Enforce strict metadata filtering by doc version/temporal tag before vector similarity search.",
		})
	}

	// 4. Typographic Noise Fragility
	typoScore := summary.VectorScores[fuzzer.VectorTypoNoise]
	if typoScore < 80.0 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorTypoNoise,
			Score:        typoScore,
			Severity:     getSeverity(typoScore),
			Title:        "Retriever Degrades Under Typographic Noise",
			ActionPlan:   "Human typing errors cause significant drop in Recall@K without lexical fuzzy matching.",
			SuggestedFix: "Add character n-gram tokenization (e.g. subword BPE) or lightweight spell-correction pre-processing before vector embedding.",
		})
	}

	// 5. Contradiction Traps
	conScore := summary.VectorScores[fuzzer.VectorContradiction]
	if conScore < 80.0 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorContradiction,
			Score:        conScore,
			Severity:     getSeverity(conScore),
			Title:        "Corpus Inconsistency Hallucination",
			ActionPlan:   "The pipeline generates ungrounded reconciliations when conflicting chunks are retrieved.",
			SuggestedFix: "Update system prompt instructions with strict contradiction reporting: 'If documents conflict, state the discrepancy explicitly instead of reconciling.'",
		})
	}

	// 6. Out-of-Domain Refusal Calibration
	oodScore := summary.VectorScores[fuzzer.VectorOOD]
	if oodScore < 85.0 || summary.RefusalF1 < 0.90 {
		recs = append(recs, Recommendation{
			Vector:       fuzzer.VectorOOD,
			Score:        oodScore,
			Severity:     getSeverity(oodScore),
			Title:        "Poor Out-of-Domain Refusal Calibration",
			ActionPlan:   "The model fabricates answers for queries with zero supporting context rather than refusing.",
			SuggestedFix: "Enforce a retrieval distance threshold guardrail: if top-1 chunk distance > 0.40, bypass generation and return calibrated refusal immediately.",
		})
	}

	return recs
}

func getSeverity(score float64) string {
	if score < 60.0 {
		return "CRITICAL"
	}
	if score < 75.0 {
		return "HIGH"
	}
	return "MEDIUM"
}
