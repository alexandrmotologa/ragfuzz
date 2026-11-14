package fuzzer

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// DistractorMutator synthesizes queries with distractor keywords to trigger retrieval cross-talk.
type DistractorMutator struct{}

func NewDistractorMutator() *DistractorMutator {
	return &DistractorMutator{}
}

func (m *DistractorMutator) Name() AttackVector {
	return VectorDistractor
}

func (m *DistractorMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	if len(topology.Chunks) == 0 {
		return testCases, nil
	}

	caseIdx := 1

	// 1. First use precomputed high-overlap pairs
	for _, pair := range topology.HighOverlapPairs {
		if len(testCases) >= count {
			break
		}

		target := pair.ChunkA
		distractor := pair.ChunkB

		cleanQuery := fmt.Sprintf("What are the details regarding %s in the system?", target.Heading)
		if target.Heading == "" || target.Heading == "Overview" {
			cleanQuery = fmt.Sprintf("Explain the guidelines for %s", target.DocPath)
		}

		// Pick distractor keywords from chunk B
		var distractorWords []string
		for _, e := range distractor.Entities {
			distractorWords = append(distractorWords, e.Value)
			if len(distractorWords) >= 2 {
				break
			}
		}
		if len(distractorWords) == 0 && len(pair.SharedKeys) > 0 {
			distractorWords = pair.SharedKeys[:1]
		}
		if len(distractorWords) == 0 {
			distractorWords = []string{distractor.Heading}
		}

		distractorPhrase := strings.Join(distractorWords, " and ")
		mutatedQuery := fmt.Sprintf("%s specifically in connection with %s", cleanQuery, distractorPhrase)

		tc := &TestCase{
			ID:                 fmt.Sprintf("TC-DST-%04d", caseIdx),
			Vector:             VectorDistractor,
			CleanQuery:         cleanQuery,
			MutatedQuery:       mutatedQuery,
			TargetChunkID:      target.ID,
			ExpectedChunkIDs:   []string{target.ID},
			DistractorChunkIDs: []string{distractor.ID},
			GroundTruthAnswer:  extractGroundTruthSentence(target.Text),
			ExpectedBehavior:   BehaviorAnswer,
			Category:           target.Heading,
			Metadata: map[string]string{
				"target_chunk":     target.ID,
				"distractor_chunk": distractor.ID,
				"distractor_keys":  distractorPhrase,
				"topic_similarity": fmt.Sprintf("%.2f", pair.Similarity),
			},
		}

		testCases = append(testCases, tc)
		caseIdx++
	}

	// 2. If count not reached, loop through all chunks and pair with adjacent
	if len(testCases) < count {
		for _, chunk := range topology.Chunks {
			if len(testCases) >= count {
				break
			}

			distractor, shared := topology.FindDistractorChunk(chunk.ID)
			if distractor == nil || distractor.ID == chunk.ID {
				continue
			}

			cleanQuery := fmt.Sprintf("What is the policy defined for %s?", chunk.Heading)
			lure := distractor.Heading
			if len(shared) > 0 {
				lure = shared[0]
			}
			mutatedQuery := fmt.Sprintf("%s also considering %s mechanisms", cleanQuery, lure)

			tc := &TestCase{
				ID:                 fmt.Sprintf("TC-DST-%04d", caseIdx),
				Vector:             VectorDistractor,
				CleanQuery:         cleanQuery,
				MutatedQuery:       mutatedQuery,
				TargetChunkID:      chunk.ID,
				ExpectedChunkIDs:   []string{chunk.ID},
				DistractorChunkIDs: []string{distractor.ID},
				GroundTruthAnswer:  extractGroundTruthSentence(chunk.Text),
				ExpectedBehavior:   BehaviorAnswer,
				Category:           chunk.Heading,
				Metadata: map[string]string{
					"target_chunk":     chunk.ID,
					"distractor_chunk": distractor.ID,
					"distractor_keys":  lure,
				},
			}

			testCases = append(testCases, tc)
			caseIdx++
		}
	}

	return testCases, nil
}
