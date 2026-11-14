package corpus

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// EntityType classifies extracted entities for targeted mutations.
type EntityType string

const (
	EntityVersion EntityType = "VERSION"
	EntityRole    EntityType = "ROLE"
	EntityNumber  EntityType = "NUMBER"
	EntityParam   EntityType = "PARAM"
	EntityKeyword EntityType = "KEYWORD"
)

// Entity represents an extracted semantic concept or attribute.
type Entity struct {
	Type    EntityType `json:"type"`
	Value   string     `json:"value"`
	Context string     `json:"context,omitempty"`
}

// Chunk represents a discrete unit of text ingested from the corpus.
type Chunk struct {
	ID        string            `json:"id"`
	DocID     string            `json:"doc_id"`
	DocPath   string            `json:"doc_path"`
	Heading   string            `json:"heading,omitempty"`
	Text      string            `json:"text"`
	Tokens    []string          `json:"tokens,omitempty"`
	Entities  []Entity          `json:"entities,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Index     int               `json:"index"`
	CharCount int               `json:"char_count"`
	WordCount int               `json:"word_count"`
}

// ComputeChunkID generates a deterministic, human-readable chunk identifier.
func ComputeChunkID(docPath string, chunkIndex int, text string) string {
	hasher := sha1.New()
	hasher.Write([]byte(text))
	hashPrefix := hex.EncodeToString(hasher.Sum(nil))[:6]
	return fmt.Sprintf("%s#c%d-%s", docPath, chunkIndex, hashPrefix)
}
