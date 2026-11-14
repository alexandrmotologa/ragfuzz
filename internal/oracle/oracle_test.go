package oracle

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/fuzzer"
)

func TestEvaluateNLI(t *testing.T) {
	contextText := `SAML 2.0 authentication is supported for Okta and Google Workspace.
The default session timeout is 3600 seconds.
Viewer accounts cannot access administrative API keys.`

	// Case 1: Grounded answer
	goodAnswer := "SAML 2.0 is supported for Okta. The session timeout is 3600 seconds."
	resGood := EvaluateNLI(goodAnswer, contextText)
	if resGood.Groundedness < 0.99 {
		t.Errorf("expected high groundedness for verified facts, got: %f", resGood.Groundedness)
	}
	if resGood.IsRefusal {
		t.Error("expected isRefusal=false for good answer")
	}

	// Case 2: Hallucinated metric
	badAnswer := "The session timeout is 7200 seconds and supports PingFederate."
	resBad := EvaluateNLI(badAnswer, contextText)
	if resBad.Groundedness >= 0.99 {
		t.Errorf("expected groundedness drop for hallucinated metric 7200, got: %f", resBad.Groundedness)
	}

	// Case 3: Refusal statement
	refusalAnswer := "I cannot find information about quantum encryption in the provided documentation."
	resRefusal := EvaluateNLI(refusalAnswer, contextText)
	if !resRefusal.IsRefusal {
		t.Error("expected isRefusal=true for explicit refusal statement")
	}
}

func TestEvaluateCase(t *testing.T) {
	tc := &fuzzer.TestCase{
		ID:                 "TC-001",
		Vector:             fuzzer.VectorNegation,
		CleanQuery:         "How does SAML auth work?",
		MutatedQuery:       "What is NOT supported in SAML auth?",
		TargetChunkID:      "chunk-1",
		ExpectedChunkIDs:   []string{"chunk-1"},
		DistractorChunkIDs: []string{"chunk-99"},
		ExpectedBehavior:   fuzzer.BehaviorAnswer,
	}

	// Subtest 1: Perfect retrieval and grounded generation
	respGood := &RAGResponse{
		Answer: "Okta is supported while legacy LDAP is not supported.",
		RetrievedChunks: []RetrievedChunk{
			{ID: "chunk-1", Text: "Okta is supported while legacy LDAP is not supported.", Rank: 1},
			{ID: "chunk-2", Text: "Some extra notes.", Rank: 2},
		},
		Latency: 50 * time.Millisecond,
	}

	evalGood := EvaluateCase(tc, respGood, 5, 0.85)
	if !evalGood.Passed {
		t.Errorf("expected test case to pass, failure reason: %s", evalGood.FailureReason)
	}
	if evalGood.MutatedRecallAtK != 1.0 {
		t.Errorf("expected recall 1.0, got: %f", evalGood.MutatedRecallAtK)
	}

	// Subtest 2: Distractor intrusion
	respDistractor := &RAGResponse{
		Answer: "Some answer.",
		RetrievedChunks: []RetrievedChunk{
			{ID: "chunk-99", Text: "Distractor text.", Rank: 1},
		},
		Latency: 50 * time.Millisecond,
	}
	evalDistractor := EvaluateCase(tc, respDistractor, 5, 0.85)
	if evalDistractor.Passed {
		t.Error("expected failure due to distractor intrusion")
	}
	if !evalDistractor.DistractorIntruded {
		t.Error("expected distractor_intruded to be true")
	}
}
