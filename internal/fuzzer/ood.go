package fuzzer

import (
	"fmt"
	"math/rand"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

// OODMutator generates ungrounded out-of-domain queries to evaluate refusal calibration.
type OODMutator struct {
	rng *rand.Rand
}

func NewOODMutator(seed int64) *OODMutator {
	if seed == 0 {
		seed = 42
	}
	return &OODMutator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (m *OODMutator) Name() AttackVector {
	return VectorOOD
}

var enterpriseOODLures = []string{
	"How do I configure quantum key distribution (QKD) hardware modules for our production clusters?",
	"What is the corporate travel reimbursement rate for transatlantic supersonic flights?",
	"How can I migrate our on-premise mainframe COBOL databases to the interplanetary IPFS node?",
	"What are the specific temperature operating thresholds for underwater datacenter pods?",
	"How do I enable zero-gravity synchronization protocols in the mobile client SDK?",
	"What is the recommended cryptographic cipher suite for post-quantum homomorphic smart contracts?",
	"Where can I submit an internal ticket to requisition a cryogenic cooling server rack?",
	"What are the compliance requirements for operating a blockchain validator node on Mars orbital relay?",
	"How do I setup biometric palm vein scanning for external contractor portal logins?",
	"What is our corporate policy regarding autonomous drone delivery of hardware security tokens?",
	"How do I configure photonic optical routing on our virtual private cloud border routers?",
	"What is the disaster recovery SLA in the event of a geomagnetic solar superflare disruption?",
}

func (m *OODMutator) Generate(topology *corpus.TopologyGraph, count int) ([]*TestCase, error) {
	var testCases []*TestCase
	caseIdx := 1

	for i := 0; i < count; i++ {
		lure := enterpriseOODLures[i%len(enterpriseOODLures)]
		cleanQuery := "Is this capability documented in the platform guides?"
		mutatedQuery := lure

		tc := &TestCase{
			ID:                fmt.Sprintf("TC-OOD-%04d", caseIdx),
			Vector:            VectorOOD,
			CleanQuery:        cleanQuery,
			MutatedQuery:      mutatedQuery,
			TargetChunkID:     "",
			ExpectedChunkIDs:  []string{},
			DistractorChunkIDs: []string{},
			GroundTruthAnswer: "I do not have information regarding this capability in the provided documentation.",
			ExpectedBehavior:  BehaviorRefusal,
			Category:          "Out-of-Domain",
			Metadata: map[string]string{
				"ood_class": "enterprise_fabrication_lure",
			},
		}

		testCases = append(testCases, tc)
		caseIdx++
	}

	return testCases, nil
}
