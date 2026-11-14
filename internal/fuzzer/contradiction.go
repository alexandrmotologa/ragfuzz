package fuzzer

import (
	"fmt"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// ContradictionMutator creates adversarial queries that test how the RAG pipeline resolves conflicting statements.
type ContradictionMutator struct{}

func NewContradictionMutator() *ContradictionMutator {
	return &ContradictionMutator{}
}

func (m *ContradictionMutator) Name() AttackVector {
	return VectorContradiction
}

func (m *ContradictionMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	if len(topology.Chunks) == 0 {
		return testCases, nil
	}

	caseIdx := 1

	// Look for chunks mentioning numeric values, quotas, or limits
	for _, chunk := range topology.Chunks {
		if len(testCases) >= count {
			break
		}

		var numEntity *corpus.Entity
		for _, e := range chunk.Entities {
			if e.Type == corpus.EntityNumber {
				numEntity = &e
				break
			}
		}

		heading := chunk.Heading
		if heading == "" {
			heading = "system resource allocation"
		}

		var cleanQuery, mutatedQuery string
		var conflictVal string

		if numEntity != nil {
			cleanQuery = fmt.Sprintf("What is the stated limit for %s?", heading)
			conflictVal = numEntity.Value + " (vs 2x legacy limit)"
			mutatedQuery = fmt.Sprintf("What is the verified limit for %s given that one section states %s and another says it is doubled?", heading, numEntity.Value)
		} else {
			cleanQuery = fmt.Sprintf("What is the official procedure for %s?", heading)
			conflictVal = "deprecated alternative"
			mutatedQuery = fmt.Sprintf("What is the official procedure for %s if older notes mention an alternative setup?", heading)
		}

		tc := &TestCase{
			ID:                fmt.Sprintf("TC-CON-%04d", caseIdx),
			Vector:            VectorContradiction,
			CleanQuery:        cleanQuery,
			MutatedQuery:      mutatedQuery,
			TargetChunkID:     chunk.ID,
			ExpectedChunkIDs:  []string{chunk.ID},
			GroundTruthAnswer: extractGroundTruthSentence(chunk.Text),
			ExpectedBehavior:  BehaviorReconciliation,
			Category:          chunk.Heading,
			Metadata: map[string]string{
				"conflict_type": "discrepancy_trap",
				"conflict_val":  conflictVal,
			},
		}

		testCases = append(testCases, tc)
		caseIdx++
	}

	return testCases, nil
}
