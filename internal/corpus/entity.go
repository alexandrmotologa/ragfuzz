package corpus

import (
	"regexp"
	"strings"
)

var (
	versionPattern = regexp.MustCompile(`(?i)\b(v[0-9]+(?:\.[0-9]+)*(?:-[a-z0-9]+)?|version\s+[0-9]+(?:\.[0-9]+)*)\b`)
	rolePattern    = regexp.MustCompile(`(?i)\b(admin|administrator|viewer|editor|owner|auditor|operator|billing_admin|member|guest|read-only|root)\b`)
	paramPattern   = regexp.MustCompile(`\b([a-z0-9]+_[a-z0-9_]+|[a-z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*|[A-Z0-9_]{3,})\b`)
	numberPattern  = regexp.MustCompile(`\b([0-9]+(?:\.[0-9]+)?(?:\s*(?:ms|s|sec|min|h|MB|GB|TB|KB|%|rps|rpm|req/s|USD|\$|EUR|tokens|chunks))?)\b`)
	headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

var commonStopwords = map[string]bool{
	"the": true, "and": true, "of": true, "to": true, "a": true, "in": true, "for": true,
	"is": true, "on": true, "that": true, "by": true, "this": true, "with": true, "i": true,
	"you": true, "it": true, "not": true, "or": true, "be": true, "are": true, "from": true,
	"at": true, "as": true, "your": true, "all": true, "have": true, "new": true, "more": true,
	"an": true, "was": true, "we": true, "will": true, "home": true, "can": true, "us": true,
	"about": true, "if": true, "page": true, "my": true, "has": true, "search": true, "free": true,
	"but": true, "our": true, "one": true, "other": true, "do": true, "no": true, "information": true,
	"time": true, "they": true, "site": true, "he": true, "up": true, "may": true, "what": true,
	"which": true, "their": true, "news": true, "out": true, "use": true, "any": true, "there": true,
	"see": true, "only": true, "so": true, "his": true, "when": true, "contact": true, "here": true,
	"business": true, "who": true, "web": true, "also": true, "now": true, "help": true, "get": true,
	"pm": true, "view": true, "online": true, "first": true, "am": true, "been": true, "would": true,
	"how": true, "were": true, "me": true, "s": true, "services": true, "some": true, "these": true,
	"click": true, "its": true, "like": true, "service": true, "than": true, "find": true, "price": true,
	"date": true, "back": true, "top": true, "people": true, "had": true, "list": true, "name": true,
	"just": true, "over": true, "state": true, "year": true, "day": true, "into": true, "email": true,
	"two": true, "health": true, "n": true, "world": true, "re": true, "next": true, "used": true,
	"go": true, "b": true, "work": true, "last": true, "most": true, "products": true, "music": true,
	"buy": true, "data": true, "make": true, "them": true, "should": true, "product": true, "system": true,
	"post": true, "her": true, "city": true, "t": true, "add": true, "policy": true, "number": true,
	"such": true, "please": true, "available": true, "must": true, "note": true, "following": true,
}

// ExtractEntities scans chunk text and isolates structured entity mentions.
func ExtractEntities(text string) []Entity {
	var entities []Entity
	seen := make(map[string]bool)

	addEntity := func(t EntityType, val string) {
		val = strings.TrimSpace(val)
		if len(val) < 2 {
			return
		}
		key := string(t) + ":" + strings.ToLower(val)
		if !seen[key] {
			seen[key] = true
			entities = append(entities, Entity{Type: t, Value: val})
		}
	}

	// 1. Version strings
	for _, match := range versionPattern.FindAllString(text, -1) {
		addEntity(EntityVersion, match)
	}

	// 2. Roles
	for _, match := range rolePattern.FindAllString(text, -1) {
		addEntity(EntityRole, strings.ToLower(match))
	}

	// 3. Technical parameters / config keys
	for _, match := range paramPattern.FindAllString(text, -1) {
		lower := strings.ToLower(match)
		if !commonStopwords[lower] && len(match) > 3 {
			addEntity(EntityParam, match)
		}
	}

	// 4. Quantities & numeric thresholds
	for _, match := range numberPattern.FindAllString(text, -1) {
		trimmed := strings.TrimSpace(match)
		if len(trimmed) > 0 {
			addEntity(EntityNumber, trimmed)
		}
	}

	// 5. Salient keywords (non-stopwords, alphabetic tokens)
	tokens := Tokenize(text)
	tf := make(map[string]int)
	for _, tok := range tokens {
		if !commonStopwords[tok] && len(tok) >= 4 {
			tf[tok]++
		}
	}
	for tok, count := range tf {
		if count >= 2 {
			addEntity(EntityKeyword, tok)
		}
	}

	return entities
}

// Tokenize converts text to lowercase alphanumeric tokens.
func Tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
