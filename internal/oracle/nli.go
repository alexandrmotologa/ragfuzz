package oracle

import (
	"regexp"
	"strings"

	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

var refusalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(i (do not|cannot|can't) (find|have|locate|determine)|not mentioned in the (provided|context|documentation)|no information (available|provided)|does not mention|is not supported in)\b`),
	regexp.MustCompile(`(?i)\b(unanswerable|cannot answer based on|outside the scope|no reference to)\b`),
}

var affirmativeNegationClashes = []struct {
	affirmative string
	negative    string
}{
	{"is supported", "is not supported"},
	{"supported", "unsupported"},
	{"permitted", "not permitted"},
	{"allowed", "not allowed"},
	{"requires", "does not require"},
	{"included", "excluded"},
}

// Claim represents a discrete factual statement extracted from generated text.
type Claim struct {
	Text      string   `json:"text"`
	Keywords  []string `json:"keywords"`
	Numbers   []string `json:"numbers"`
	IsNegated bool     `json:"is_negated"`
	Entailed  bool     `json:"entailed"`
	Reason    string   `json:"reason,omitempty"`
}

// NLIResult contains the claim decomposition and verification breakdown.
type NLIResult struct {
	TotalClaims      int     `json:"total_claims"`
	EntailedClaims   int     `json:"entailed_claims"`
	Groundedness     float64 `json:"groundedness"`
	IsRefusal        bool    `json:"is_refusal"`
	Claims           []Claim `json:"claims"`
	HallucinatedText string  `json:"hallucinated_text,omitempty"`
}

// EvaluateNLI performs deterministic Natural Language Inference between answer and context chunks.
func EvaluateNLI(answer string, contextChunksText string) *NLIResult {
	trimmedAnswer := strings.TrimSpace(answer)
	if trimmedAnswer == "" {
		return &NLIResult{
			TotalClaims:    0,
			EntailedClaims: 0,
			Groundedness:   0.0,
			IsRefusal:      true,
		}
	}

	// 1. Check for refusal phrasing
	isRefusal := false
	for _, pat := range refusalPatterns {
		if pat.MatchString(trimmedAnswer) {
			isRefusal = true
			break
		}
	}

	if isRefusal {
		return &NLIResult{
			TotalClaims:    0,
			EntailedClaims: 0,
			Groundedness:   1.0,
			IsRefusal:      true,
		}
	}

	claims := extractClaims(trimmedAnswer)
	if len(claims) == 0 {
		return &NLIResult{
			TotalClaims:    1,
			EntailedClaims: 1,
			Groundedness:   1.0,
			IsRefusal:      isRefusal,
		}
	}

	contextLower := strings.ToLower(contextChunksText)
	contextTokens := corpus.Tokenize(contextChunksText)
	contextTokenSet := make(map[string]bool, len(contextTokens))
	for _, tok := range contextTokens {
		contextTokenSet[tok] = true
	}

	entailedCount := 0
	var unentailedSnippets []string

	for i := range claims {
		claim := &claims[i]
		entailed, reason := verifyClaim(claim, contextLower, contextTokenSet)
		claim.Entailed = entailed
		claim.Reason = reason

		if entailed {
			entailedCount++
		} else {
			unentailedSnippets = append(unentailedSnippets, claim.Text)
		}
	}

	groundedness := float64(entailedCount) / float64(len(claims))

	return &NLIResult{
		TotalClaims:      len(claims),
		EntailedClaims:   entailedCount,
		Groundedness:     groundedness,
		IsRefusal:        isRefusal,
		Claims:           claims,
		HallucinatedText: strings.Join(unentailedSnippets, " | "),
	}
}

func extractClaims(text string) []Claim {
	// Split by sentence terminators or bullet points
	rawSentences := regexp.MustCompile(`[.!?\n]+`).Split(text, -1)
	var claims []Claim

	for _, s := range rawSentences {
		trimmed := strings.TrimSpace(s)
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		if len(trimmed) < 10 {
			continue
		}

		tokens := corpus.Tokenize(trimmed)
		var keywords []string
		for _, tok := range tokens {
			if len(tok) >= 4 {
				keywords = append(keywords, tok)
			}
		}

		// Extract numbers/metrics
		entities := corpus.ExtractEntities(trimmed)
		var numbers []string
		for _, e := range entities {
			if e.Type == corpus.EntityNumber || e.Type == corpus.EntityVersion {
				numbers = append(numbers, strings.ToLower(e.Value))
			}
		}

		lower := strings.ToLower(trimmed)
		isNeg := strings.Contains(lower, "not ") || strings.Contains(lower, "never ") ||
			strings.Contains(lower, "cannot ") || strings.Contains(lower, "neither ") ||
			strings.Contains(lower, "without ")

		claims = append(claims, Claim{
			Text:      trimmed,
			Keywords:  keywords,
			Numbers:   numbers,
			IsNegated: isNeg,
		})
	}

	return claims
}

func verifyClaim(claim *Claim, contextLower string, contextTokenSet map[string]bool) (bool, string) {
	claimLower := strings.ToLower(claim.Text)

	// If the context literally contains this exact claim or clause, it is entailed
	if strings.Contains(contextLower, claimLower) {
		return true, "strictly entailed by context"
	}

	// If the claim asserts specific numbers/metrics, check if they exist in context
	for _, num := range claim.Numbers {
		if !strings.Contains(contextLower, num) {
			return false, "unsupported number or metric: " + num
		}
	}

	// Keyword coverage check: at least 50% of significant keywords must appear in context
	if len(claim.Keywords) > 0 {
		matched := 0
		for _, kw := range claim.Keywords {
			if contextTokenSet[kw] {
				matched++
			}
		}
		coverage := float64(matched) / float64(len(claim.Keywords))
		if coverage < 0.50 {
			return false, "low semantic overlap with retrieved context"
		}
	}

	// Polarity check: check for direct contradiction with context
	for _, clash := range affirmativeNegationClashes {
		// Only trigger clash if the claim contains the assertion and context does NOT contain the assertion
		if strings.Contains(claimLower, clash.affirmative) && !strings.Contains(contextLower, clash.affirmative) && strings.Contains(contextLower, clash.negative) {
			return false, "polarity contradiction: affirmative claim contradicts negative context"
		}
		if strings.Contains(claimLower, clash.negative) && !strings.Contains(contextLower, clash.negative) && strings.Contains(contextLower, clash.affirmative) {
			return false, "polarity contradiction: negative claim contradicts affirmative context"
		}
	}

	return true, "strictly entailed by context"
}
