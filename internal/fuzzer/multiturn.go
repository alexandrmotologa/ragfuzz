package fuzzer

import (
	"fmt"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// Turn represents an individual exchange in a multi-turn adversarial dialogue.
type Turn struct {
	Index            int              `json:"index"`
	Role             string           `json:"role"` // "user"
	Query            string           `json:"query"`
	ExpectedChunkIDs []string         `json:"expected_chunk_ids,omitempty"`
	ExpectedBehavior ExpectedBehavior `json:"expected_behavior"`
	IsAdversarial    bool             `json:"is_adversarial"`
	DriftType        string           `json:"drift_type,omitempty"`
}

// MultiTurnTestCase represents a multi-turn conversation designed to provoke context drift.
type MultiTurnTestCase struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	TargetChunkID      string            `json:"target_chunk_id"`
	Turns              []Turn            `json:"turns"`
	ExpectedFinalState ExpectedBehavior  `json:"expected_final_state"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// MultiTurnMutator synthesizes multi-turn dialogue trees that gradually drift context.
type MultiTurnMutator struct{}

func NewMultiTurnMutator() *MultiTurnMutator {
	return &MultiTurnMutator{}
}

// GenerateDialogues creates multi-turn test scenarios against the corpus topology.
func (m *MultiTurnMutator) GenerateDialogues(topology *corpus.TopologyGraph, count int) []*MultiTurnTestCase {
	var dialogs []*MultiTurnTestCase
	if len(topology.Chunks) == 0 {
		return dialogs
	}

	caseIdx := 1
	for _, chunk := range topology.Chunks {
		if len(dialogs) >= count {
			break
		}

		heading := chunk.Heading
		if heading == "" || heading == "Overview" {
			heading = "the service configuration"
		}

		distractor, _ := topology.FindDistractorChunk(chunk.ID)
		distractorHeading := "secondary features"
		if distractor != nil && distractor.Heading != "" {
			distractorHeading = distractor.Heading
		}

		// Turn 1: Grounded in-domain query
		turn1 := Turn{
			Index:            1,
			Role:             "user",
			Query:            fmt.Sprintf("Can you explain how %s is configured?", heading),
			ExpectedChunkIDs: []string{chunk.ID},
			ExpectedBehavior: BehaviorAnswer,
			IsAdversarial:    false,
		}

		// Turn 2: Subtle contextual distraction introducing distractor entity
		turn2 := Turn{
			Index:            2,
			Role:             "user",
			Query:            fmt.Sprintf("Does that setup also apply when integrating with %s?", distractorHeading),
			ExpectedChunkIDs: []string{chunk.ID},
			ExpectedBehavior: BehaviorAnswer,
			IsAdversarial:    true,
			DriftType:        "semantic_distractor_injection",
		}

		// Turn 3: Target probing question testing for context drift or hallucination
		turn3 := Turn{
			Index:            3,
			Role:             "user",
			Query:            "Given our previous discussion, what are the strict limitations that do NOT apply here?",
			ExpectedChunkIDs: []string{chunk.ID},
			ExpectedBehavior: BehaviorAnswer,
			IsAdversarial:    true,
			DriftType:        "conversational_drift_probe",
		}

		dt := &MultiTurnTestCase{
			ID:                 fmt.Sprintf("TC-MT-%04d", caseIdx),
			Title:              fmt.Sprintf("Context Drift Dialogue: %s", heading),
			TargetChunkID:      chunk.ID,
			Turns:              []Turn{turn1, turn2, turn3},
			ExpectedFinalState: BehaviorAnswer,
			Metadata: map[string]string{
				"target_heading":     heading,
				"distractor_heading": distractorHeading,
			},
		}

		dialogs = append(dialogs, dt)
		caseIdx++
	}

	return dialogs
}
