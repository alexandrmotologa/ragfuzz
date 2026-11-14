package fuzzer

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// NegationMutator generates queries requiring exclusionary and negative constraint logic.
type NegationMutator struct{}

func NewNegationMutator() *NegationMutator {
	return &NegationMutator{}
}

func (m *NegationMutator) Name() AttackVector {
	return VectorNegation
}

var negationTemplates = []struct {
	cleanTemplate   string
	mutatedTemplate string
}{
	{
		cleanTemplate:   "What are the requirements for %s?",
		mutatedTemplate: "What are the scenarios that do NOT require %s?",
	},
	{
		cleanTemplate:   "Which users are permitted to %s?",
		mutatedTemplate: "Which users are explicitly NOT permitted or restricted from %s?",
	},
	{
		cleanTemplate:   "How do I configure %s?",
		mutatedTemplate: "What should NOT be configured when setting up %s?",
	},
	{
		cleanTemplate:   "What options are supported for %s?",
		mutatedTemplate: "What options are unsupported or excluded from %s?",
	},
	{
		cleanTemplate:   "Under what conditions does %s apply?",
		mutatedTemplate: "Under what conditions does %s NOT apply or fail?",
	},
}

func (m *NegationMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	if len(topology.Chunks) == 0 {
		return testCases, nil
	}

	caseIdx := 1
	for _, chunk := range topology.Chunks {
		if len(testCases) >= count {
			break
		}

		subject := chunk.Heading
		if subject == "" || subject == "Overview" || subject == "Document" {
			// Pick top entity or keyword
			for _, ent := range chunk.Entities {
				if ent.Type == corpus.EntityParam || ent.Type == corpus.EntityRole || ent.Type == corpus.EntityVersion {
					subject = ent.Value
					break
				}
			}
		}

		if subject == "" {
			tokens := corpus.Tokenize(chunk.Text)
			if len(tokens) > 3 {
				subject = tokens[2] + " " + tokens[3]
			} else {
				subject = "this feature"
			}
		}

		for _, tpl := range negationTemplates {
			if len(testCases) >= count {
				break
			}

			cleanQuery := fmt.Sprintf(tpl.cleanTemplate, subject)
			mutatedQuery := fmt.Sprintf(tpl.mutatedTemplate, subject)

			tc := &TestCase{
				ID:                fmt.Sprintf("TC-NEG-%04d", caseIdx),
				Vector:            VectorNegation,
				CleanQuery:        cleanQuery,
				MutatedQuery:      mutatedQuery,
				TargetChunkID:     chunk.ID,
				ExpectedChunkIDs:  []string{chunk.ID},
				GroundTruthAnswer: extractGroundTruthSentence(chunk.Text),
				ExpectedBehavior:  BehaviorAnswer,
				Category:          chunk.Heading,
				Metadata: map[string]string{
					"subject":       subject,
					"inversion_key": "negation_exclusion",
				},
			}

			testCases = append(testCases, tc)
			caseIdx++
		}
	}

	return testCases, nil
}

func extractGroundTruthSentence(text string) string {
	sentences := strings.Split(text, ".")
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if len(trimmed) > 30 {
			return trimmed + "."
		}
	}
	if len(text) > 120 {
		return text[:120] + "..."
	}
	return text
}
