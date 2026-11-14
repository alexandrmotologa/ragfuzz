package fuzzer

import (
	"testing"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

func createTestTopology() *corpus.TopologyGraph {
	chunks := []*corpus.Chunk{
		{
			ID:       "sso#c0",
			DocID:    "sso.md",
			Heading:  "SAML Authentication",
			Text:     "Enterprise SSO requires SAML 2.0. Users with admin role can configure identity providers.",
			Tokens:   []string{"enterprise", "sso", "requires", "saml", "users", "admin", "role", "configure"},
			Entities: corpus.ExtractEntities("Enterprise SSO requires SAML 2.0. Users with admin role can configure identity providers."),
		},
		{
			ID:       "sso#c1",
			DocID:    "sso.md",
			Heading:  "OAuth2 Integration",
			Text:     "OAuth 2.0 tokens expire after 3600 seconds. Viewer role cannot access tokens.",
			Tokens:   []string{"oauth", "tokens", "expire", "after", "seconds", "viewer", "role", "cannot"},
			Entities: corpus.ExtractEntities("OAuth 2.0 tokens expire after 3600 seconds. Viewer role cannot access tokens."),
		},
		{
			ID:       "billing#c0",
			DocID:    "billing.md",
			Heading:  "Billing Quotas v2.0",
			Text:     "In API v2.0, storage quota is 100GB for Enterprise tier and rate limit is 500req/s.",
			Tokens:   []string{"api", "storage", "quota", "enterprise", "tier", "rate", "limit"},
			Entities: corpus.ExtractEntities("In API v2.0, storage quota is 100GB for Enterprise tier and rate limit is 500req/s."),
		},
	}
	return corpus.BuildTopology(chunks)
}

func TestAllSixMutators(t *testing.T) {
	topology := createTestTopology()

	// 1. Negation
	neg := NewNegationMutator()
	negCases, err := neg.Generate(topology, 5)
	if err != nil || len(negCases) == 0 {
		t.Fatalf("negation mutator failed: %v, count=%d", err, len(negCases))
	}
	if negCases[0].Vector != VectorNegation {
		t.Errorf("expected vector %s, got %s", VectorNegation, negCases[0].Vector)
	}

	// 2. Distractor
	dst := NewDistractorMutator()
	dstCases, err := dst.Generate(topology, 5)
	if err != nil || len(dstCases) == 0 {
		t.Fatalf("distractor mutator failed: %v, count=%d", err, len(dstCases))
	}
	if dstCases[0].Vector != VectorDistractor {
		t.Errorf("expected vector %s, got %s", VectorDistractor, dstCases[0].Vector)
	}

	// 3. Entity Swap
	ent := NewEntitySwapMutator()
	entCases, err := ent.Generate(topology, 5)
	if err != nil || len(entCases) == 0 {
		t.Fatalf("entity swap mutator failed: %v, count=%d", err, len(entCases))
	}
	if entCases[0].Vector != VectorEntitySwap {
		t.Errorf("expected vector %s, got %s", VectorEntitySwap, entCases[0].Vector)
	}

	// 4. Typo
	typ := NewTypoMutator(2, 42)
	typCases, err := typ.Generate(topology, 5)
	if err != nil || len(typCases) == 0 {
		t.Fatalf("typo mutator failed: %v, count=%d", err, len(typCases))
	}
	if typCases[0].Vector != VectorTypoNoise {
		t.Errorf("expected vector %s, got %s", VectorTypoNoise, typCases[0].Vector)
	}

	// 5. Contradiction
	con := NewContradictionMutator()
	conCases, err := con.Generate(topology, 5)
	if err != nil || len(conCases) == 0 {
		t.Fatalf("contradiction mutator failed: %v, count=%d", err, len(conCases))
	}
	if conCases[0].Vector != VectorContradiction {
		t.Errorf("expected vector %s, got %s", VectorContradiction, conCases[0].Vector)
	}

	// 6. OOD
	ood := NewOODMutator(42)
	oodCases, err := ood.Generate(topology, 5)
	if err != nil || len(oodCases) == 0 {
		t.Fatalf("ood mutator failed: %v, count=%d", err, len(oodCases))
	}
	if oodCases[0].Vector != VectorOOD {
		t.Errorf("expected vector %s, got %s", VectorOOD, oodCases[0].Vector)
	}
	if oodCases[0].ExpectedBehavior != BehaviorRefusal {
		t.Errorf("expected refusal behavior for OOD, got: %s", oodCases[0].ExpectedBehavior)
	}

	// 7. Multi-turn Conversational Drift
	mt := NewMultiTurnMutator()
	mtDialogs := mt.GenerateDialogues(topology, 2)
	if len(mtDialogs) == 0 {
		t.Fatal("expected multi-turn dialogs to be generated")
	}
	if len(mtDialogs[0].Turns) != 3 {
		t.Errorf("expected 3 turns in dialogue, got %d", len(mtDialogs[0].Turns))
	}
}

func TestGeneratorSuiteDistribution(t *testing.T) {
	topology := createTestTopology()
	cfg := &config.FuzzerConfig{
		TotalQueries:           60,
		EnabledVectors:         []string{},
		LevenshteinMaxDistance: 2,
		Seed:                   123,
	}

	generator := NewGenerator(cfg, topology)
	suite, err := generator.GenerateSuite()
	if err != nil {
		t.Fatalf("GenerateSuite failed: %v", err)
	}

	if len(suite) == 0 {
		t.Fatal("expected non-empty suite")
	}

	// Verify all 6 vectors are represented
	vectorCounts := make(map[AttackVector]int)
	for _, tc := range suite {
		vectorCounts[tc.Vector]++
	}

	for _, vec := range AllVectors {
		if vectorCounts[vec] == 0 {
			t.Errorf("vector %s has 0 test cases in generated suite", vec)
		}
	}
}
