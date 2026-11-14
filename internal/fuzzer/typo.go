package fuzzer

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// TypoMutator introduces controlled Levenshtein edits and QWERTY adjacency typos.
type TypoMutator struct {
	maxDistance int
	rng         *rand.Rand
}

func NewTypoMutator(maxDistance int, seed int64) *TypoMutator {
	if maxDistance <= 0 {
		maxDistance = 2
	}
	if seed == 0 {
		seed = 42
	}
	return &TypoMutator{
		maxDistance: maxDistance,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

func (m *TypoMutator) Name() AttackVector {
	return VectorTypoNoise
}

var qwertyNeighbors = map[rune][]rune{
	'a': {'q', 'w', 's', 'z'},
	'b': {'v', 'g', 'h', 'n'},
	'c': {'x', 'd', 'f', 'v'},
	'd': {'s', 'e', 'r', 'f', 'c', 'x'},
	'e': {'w', 's', 'd', 'r'},
	'f': {'d', 'r', 't', 'g', 'v', 'c'},
	'g': {'f', 't', 'y', 'h', 'b', 'v'},
	'h': {'g', 'y', 'u', 'j', 'n', 'b'},
	'i': {'u', 'j', 'k', 'o'},
	'j': {'h', 'u', 'i', 'k', 'm', 'n'},
	'k': {'j', 'i', 'o', 'l'},
	'l': {'k', 'o', 'p'},
	'm': {'n', 'j', 'k'},
	'n': {'b', 'h', 'j', 'm'},
	'o': {'i', 'k', 'l', 'p'},
	'p': {'o', 'l'},
	'q': {'w', 'a'},
	'r': {'e', 'd', 'f', 't'},
	's': {'a', 'w', 'e', 'd', 'x', 'z'},
	't': {'r', 'f', 'g', 'y'},
	'u': {'y', 'h', 'j', 'i'},
	'v': {'c', 'f', 'g', 'b'},
	'w': {'q', 'a', 's', 'e'},
	'x': {'z', 's', 'd', 'c'},
	'y': {'t', 'g', 'h', 'u'},
	'z': {'a', 's', 'x'},
}

// InjectTypo applies a controlled 1-2 edit distance typo to a word.
func (m *TypoMutator) InjectTypo(word string) string {
	runes := []rune(word)
	if len(runes) < 4 {
		return word
	}

	op := m.rng.Intn(4)
	idx := m.rng.Intn(len(runes)-2) + 1 // Pick middle position

	switch op {
	case 0:
		// Transposition of adjacent letters
		runes[idx], runes[idx+1] = runes[idx+1], runes[idx]
		return string(runes)
	case 1:
		// QWERTY keyboard neighbor replacement
		char := runes[idx]
		if neighbors, exists := qwertyNeighbors[char]; exists && len(neighbors) > 0 {
			runes[idx] = neighbors[m.rng.Intn(len(neighbors))]
			return string(runes)
		}
		return string(runes)
	case 2:
		// Deletion of a character
		return string(runes[:idx]) + string(runes[idx+1:])
	case 3:
		// Duplication (insertion)
		return string(runes[:idx]) + string(runes[idx:idx+1]) + string(runes[idx:])
	default:
		return word
	}
}

func (m *TypoMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	if len(topology.Chunks) == 0 {
		return testCases, nil
	}

	caseIdx := 1

	for _, chunk := range topology.Chunks {
		if len(testCases) >= count {
			break
		}

		heading := chunk.Heading
		if heading == "" {
			heading = "system configuration"
		}

		cleanQuery := fmt.Sprintf("How do I configure %s?", heading)

		// Select key words in heading or query to corrupt
		words := strings.Fields(cleanQuery)
		var mutatedWords []string
		typoApplied := false

		for _, w := range words {
			cleanWord := strings.ToLower(strings.Trim(w, "?.,!"))
			if !typoApplied && len(cleanWord) >= 5 && cleanWord != "configure" && cleanWord != "system" {
				mutatedWords = append(mutatedWords, m.InjectTypo(w))
				typoApplied = true
			} else {
				mutatedWords = append(mutatedWords, w)
			}
		}

		// Fallback if no specific word was mutated
		if !typoApplied && len(words) > 3 {
			words[len(words)-1] = m.InjectTypo(words[len(words)-1])
			mutatedWords = words
		}

		mutatedQuery := strings.Join(mutatedWords, " ")

		tc := &TestCase{
			ID:                fmt.Sprintf("TC-TYP-%04d", caseIdx),
			Vector:            VectorTypoNoise,
			CleanQuery:        cleanQuery,
			MutatedQuery:      mutatedQuery,
			TargetChunkID:     chunk.ID,
			ExpectedChunkIDs:  []string{chunk.ID},
			GroundTruthAnswer: extractGroundTruthSentence(chunk.Text),
			ExpectedBehavior:  BehaviorAnswer,
			Category:          chunk.Heading,
			Metadata: map[string]string{
				"noise_type": "levenshtein_qwerty",
			},
		}

		testCases = append(testCases, tc)
		caseIdx++
	}

	return testCases, nil
}
