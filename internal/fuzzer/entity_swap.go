package fuzzer

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// EntitySwapMutator mutates version flags, roles, and technical parameters to exploit vector collapse.
type EntitySwapMutator struct{}

func NewEntitySwapMutator() *EntitySwapMutator {
	return &EntitySwapMutator{}
}

func (m *EntitySwapMutator) Name() AttackVector {
	return VectorEntitySwap
}

var roleSwaps = map[string]string{
	"admin":         "viewer",
	"administrator": "read-only",
	"viewer":        "admin",
	"editor":        "auditor",
	"owner":         "member",
	"read-only":     "admin",
	"root":          "guest",
}

var versionSwaps = map[string]string{
	"v1":   "v2",
	"v2":   "v1",
	"v1.0": "v2.0",
	"v2.0": "v3.0",
	"v2.4": "v1.8",
	"v3":   "v1",
}

func (m *EntitySwapMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	if len(topology.Chunks) == 0 {
		return testCases, nil
	}

	caseIdx := 1

	for _, chunk := range topology.Chunks {
		if len(testCases) >= count {
			break
		}

		// Find swappable entities in this chunk
		for _, entity := range chunk.Entities {
			if len(testCases) >= count {
				break
			}

			var origVal, swappedVal string
			var swapType string

			lowerVal := strings.ToLower(entity.Value)
			if entity.Type == corpus.EntityVersion {
				swapType = "version_swap"
				origVal = entity.Value
				if swap, ok := versionSwaps[lowerVal]; ok {
					swappedVal = swap
				} else {
					swappedVal = entity.Value + ".deprecated"
				}
			} else if entity.Type == corpus.EntityRole {
				swapType = "role_swap"
				origVal = entity.Value
				if swap, ok := roleSwaps[lowerVal]; ok {
					swappedVal = swap
				} else {
					swappedVal = "unprivileged_user"
				}
			} else if entity.Type == corpus.EntityNumber && strings.Contains(entity.Value, "MB") {
				swapType = "quota_swap"
				origVal = entity.Value
				swappedVal = "100GB"
			}

			if origVal == "" || swappedVal == "" {
				continue
			}

			cleanQuery := fmt.Sprintf("What are the parameters for %s regarding %s?", chunk.Heading, origVal)
			mutatedQuery := fmt.Sprintf("What are the parameters for %s regarding %s?", chunk.Heading, swappedVal)

			tc := &TestCase{
				ID:                fmt.Sprintf("TC-ENT-%04d", caseIdx),
				Vector:            VectorEntitySwap,
				CleanQuery:        cleanQuery,
				MutatedQuery:      mutatedQuery,
				TargetChunkID:     chunk.ID,
				ExpectedChunkIDs:  []string{chunk.ID},
				GroundTruthAnswer: extractGroundTruthSentence(chunk.Text),
				ExpectedBehavior:  BehaviorAnswer,
				Category:          chunk.Heading,
				Metadata: map[string]string{
					"swap_type":   swapType,
					"original":    origVal,
					"substituted": swappedVal,
				},
			}

			testCases = append(testCases, tc)
			caseIdx++
		}
	}

	// Fallback generic swap if not enough specific entities
	if len(testCases) < count {
		for _, chunk := range topology.Chunks {
			if len(testCases) >= count {
				break
			}

			heading := chunk.Heading
			if heading == "" {
				heading = "the service"
			}

			cleanQuery := fmt.Sprintf("How does %s behave in version v2.0?", heading)
			mutatedQuery := fmt.Sprintf("How does %s behave in legacy version v1.0?", heading)

			tc := &TestCase{
				ID:                fmt.Sprintf("TC-ENT-%04d", caseIdx),
				Vector:            VectorEntitySwap,
				CleanQuery:        cleanQuery,
				MutatedQuery:      mutatedQuery,
				TargetChunkID:     chunk.ID,
				ExpectedChunkIDs:  []string{chunk.ID},
				GroundTruthAnswer: extractGroundTruthSentence(chunk.Text),
				ExpectedBehavior:  BehaviorAnswer,
				Category:          chunk.Heading,
				Metadata: map[string]string{
					"swap_type":   "version_fallback",
					"original":    "v2.0",
					"substituted": "v1.0",
				},
			}

			testCases = append(testCases, tc)
			caseIdx++
		}
	}

	return testCases, nil
}
