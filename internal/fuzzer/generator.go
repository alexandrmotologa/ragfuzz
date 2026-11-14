package fuzzer

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// Generator coordinates all adversarial mutators to produce a comprehensive fuzz suite.
type Generator struct {
	cfg      *config.FuzzerConfig
	topology *corpus.TopologyGraph
	mutators map[AttackVector]Mutator
}

// NewGenerator instantiates a Generator with all 6 adversarial mutators registered.
func NewGenerator(cfg *config.FuzzerConfig, topology *corpus.TopologyGraph) *Generator {
	mutators := map[AttackVector]Mutator{
		VectorNegation:      NewNegationMutator(),
		VectorDistractor:    NewDistractorMutator(),
		VectorEntitySwap:    NewEntitySwapMutator(),
		VectorTypoNoise:     NewTypoMutator(cfg.LevenshteinMaxDistance, cfg.Seed),
		VectorContradiction: NewContradictionMutator(),
		VectorOOD:           NewOODMutator(cfg.Seed),
	}

	return &Generator{
		cfg:      cfg,
		topology: topology,
		mutators: mutators,
	}
}

// GenerateSuite synthesizes the target number of adversarial test cases distributed across active vectors.
func (g *Generator) GenerateSuite() ([]*TestCase, error) {
	activeVectors := g.getActiveVectors()
	if len(activeVectors) == 0 {
		return nil, fmt.Errorf("no valid attack vectors enabled")
	}

	total := g.cfg.TotalQueries
	if total <= 0 {
		total = 100
	}

	perVector := total / len(activeVectors)
	remainder := total % len(activeVectors)

	var allCases []*TestCase

	for idx, vec := range activeVectors {
		count := perVector
		if idx < remainder {
			count++
		}

		mutator, exists := g.mutators[vec]
		if !exists {
			continue
		}

		cases, err := mutator.Generate(g.topology, count)
		if err != nil {
			return nil, fmt.Errorf("error generating cases for vector %s: %w", vec, err)
		}

		allCases = append(allCases, cases...)
	}

	// Interleave test cases across vectors for balanced execution ordering
	interleaved := interleaveCases(allCases, len(activeVectors))
	return interleaved, nil
}

func (g *Generator) getActiveVectors() []AttackVector {
	if len(g.cfg.EnabledVectors) == 0 {
		return AllVectors
	}

	var active []AttackVector
	enabledMap := make(map[string]bool)
	for _, v := range g.cfg.EnabledVectors {
		enabledMap[strings.ToLower(strings.TrimSpace(v))] = true
	}

	for _, v := range AllVectors {
		if enabledMap[strings.ToLower(string(v))] {
			active = append(active, v)
		}
	}

	if len(active) == 0 {
		return AllVectors
	}
	return active
}

func interleaveCases(cases []*TestCase, numBuckets int) []*TestCase {
	if numBuckets <= 1 || len(cases) == 0 {
		return cases
	}

	// Group by vector
	buckets := make(map[AttackVector][]*TestCase)
	for _, tc := range cases {
		buckets[tc.Vector] = append(buckets[tc.Vector], tc)
	}

	var interleaved []*TestCase
	maxLen := 0
	for _, b := range buckets {
		if len(b) > maxLen {
			maxLen = len(b)
		}
	}

	for i := 0; i < maxLen; i++ {
		for _, vec := range AllVectors {
			b := buckets[vec]
			if i < len(b) {
				interleaved = append(interleaved, b[i])
			}
		}
	}

	return interleaved
}
