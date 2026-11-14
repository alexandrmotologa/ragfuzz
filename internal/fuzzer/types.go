package fuzzer

import (
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// AttackVector identifies one of the six targeted adversarial stress vectors.
type AttackVector string

const (
	VectorNegation      AttackVector = "negation_inversion"
	VectorDistractor    AttackVector = "semantic_distractor"
	VectorEntitySwap    AttackVector = "entity_version_collision"
	VectorTypoNoise     AttackVector = "orthographic_noise"
	VectorContradiction AttackVector = "contradiction_trap"
	VectorOOD           AttackVector = "out_of_domain"
)

// AllVectors contains the complete list of supported attack vectors.
var AllVectors = []AttackVector{
	VectorNegation,
	VectorDistractor,
	VectorEntitySwap,
	VectorTypoNoise,
	VectorContradiction,
	VectorOOD,
}

// ExpectedBehavior defines how a well-calibrated RAG pipeline should respond.
type ExpectedBehavior string

const (
	BehaviorAnswer         ExpectedBehavior = "answer"
	BehaviorRefusal        ExpectedBehavior = "refusal"
	BehaviorReconciliation ExpectedBehavior = "reconciliation"
)

// TestCase represents an individual adversarial test item ready for execution.
type TestCase struct {
	ID                 string            `json:"id"`
	Vector             AttackVector      `json:"vector"`
	CleanQuery         string            `json:"clean_query"`
	MutatedQuery       string            `json:"mutated_query"`
	TargetChunkID      string            `json:"target_chunk_id,omitempty"`
	ExpectedChunkIDs   []string          `json:"expected_chunk_ids,omitempty"`
	DistractorChunkIDs []string          `json:"distractor_chunk_ids,omitempty"`
	GroundTruthAnswer  string            `json:"ground_truth_answer,omitempty"`
	ExpectedBehavior   ExpectedBehavior  `json:"expected_behavior"`
	Category           string            `json:"category,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// Mutator defines the common contract for test case generators.
type Mutator interface {
	Name() AttackVector
	Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error)
}
