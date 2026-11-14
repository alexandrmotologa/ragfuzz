package corpus

import (
	"math"
	"sort"
)

// ChunkPair records the topical similarity between two distinct chunks in the corpus.
type ChunkPair struct {
	ChunkA     *Chunk  `json:"chunk_a"`
	ChunkB     *Chunk  `json:"chunk_b"`
	Similarity float64 `json:"similarity"`
	SharedKeys []string `json:"shared_keys"`
}

// TopologyGraph maintains the BM25 index and chunk similarity topology.
type TopologyGraph struct {
	Chunks         []*Chunk
	ChunkMap       map[string]*Chunk
	DocFreq        map[string]int
	AvgDocLength   float64
	HighOverlapPairs []ChunkPair
	K1             float64
	B              float64
}

// BuildTopology constructs BM25 statistics and calculates pairwise topic similarities.
func BuildTopology(chunks []*Chunk) *TopologyGraph {
	g := &TopologyGraph{
		Chunks:   chunks,
		ChunkMap: make(map[string]*Chunk, len(chunks)),
		DocFreq:  make(map[string]int),
		K1:       1.5,
		B:        0.75,
	}

	totalTokens := 0
	for _, ch := range chunks {
		g.ChunkMap[ch.ID] = ch
		totalTokens += len(ch.Tokens)

		uniqueInDoc := make(map[string]bool)
		for _, tok := range ch.Tokens {
			if !uniqueInDoc[tok] {
				uniqueInDoc[tok] = true
				g.DocFreq[tok]++
			}
		}
	}

	if len(chunks) > 0 {
		g.AvgDocLength = float64(totalTokens) / float64(len(chunks))
	}

	// Compute pairwise similarities for finding distractors and contradiction candidates
	g.HighOverlapPairs = g.computePairwiseSimilarities()

	return g
}

// ScoreBM25 calculates the BM25 score of a query against a specific chunk.
func (g *TopologyGraph) ScoreBM25(queryTokens []string, chunk *Chunk) float64 {
	score := 0.0
	numDocs := float64(len(g.Chunks))
	docLen := float64(len(chunk.Tokens))

	// Term frequency in this chunk
	tf := make(map[string]int)
	for _, tok := range chunk.Tokens {
		tf[tok]++
	}

	for _, qTok := range queryTokens {
		df := float64(g.DocFreq[qTok])
		if df == 0 {
			continue
		}

		// Robertson-Sparck Jones IDF
		idf := math.Log(1.0 + (numDocs-df+0.5)/(df+0.5))
		if idf < 0 {
			idf = 0.01
		}

		termFreq := float64(tf[qTok])
		numerator := termFreq * (g.K1 + 1.0)
		denominator := termFreq + g.K1*(1.0-g.B+g.B*(docLen/g.AvgDocLength))

		score += idf * (numerator / denominator)
	}

	return score
}

// SearchBM25 returns the top-K chunks ranked by BM25 score for the given query.
func (g *TopologyGraph) SearchBM25(query string, topK int) []*Chunk {
	qTokens := Tokenize(query)
	type candidate struct {
		chunk *Chunk
		score float64
	}

	var candidates []candidate
	for _, ch := range g.Chunks {
		score := g.ScoreBM25(qTokens, ch)
		if score > 0 {
			candidates = append(candidates, candidate{chunk: ch, score: score})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}

	results := make([]*Chunk, topK)
	for i := 0; i < topK; i++ {
		results[i] = candidates[i].chunk
	}
	return results
}

func (g *TopologyGraph) computePairwiseSimilarities() []ChunkPair {
	var pairs []ChunkPair
	n := len(g.Chunks)
	if n < 2 {
		return pairs
	}

	// Calculate TF-IDF vectors for each chunk
	numDocs := float64(n)
	vectors := make([]map[string]float64, n)
	norms := make([]float64, n)

	for i, ch := range g.Chunks {
		vec := make(map[string]float64)
		tf := make(map[string]int)
		for _, tok := range ch.Tokens {
			if !commonStopwords[tok] && len(tok) > 2 {
				tf[tok]++
			}
		}

		sumSq := 0.0
		for tok, count := range tf {
			df := float64(g.DocFreq[tok])
			idf := math.Log(1.0 + numDocs/(1.0+df))
			val := float64(count) * idf
			vec[tok] = val
			sumSq += val * val
		}
		vectors[i] = vec
		norms[i] = math.Sqrt(sumSq)
	}

	// Pairwise cosine similarity
	for i := 0; i < n; i++ {
		if norms[i] == 0 {
			continue
		}
		for j := i + 1; j < n; j++ {
			if norms[j] == 0 {
				continue
			}

			// Do not compare chunks of same doc if they are contiguous
			if g.Chunks[i].DocID == g.Chunks[j].DocID && math.Abs(float64(g.Chunks[i].Index-g.Chunks[j].Index)) <= 1 {
				continue
			}

			dotProduct := 0.0
			var shared []string
			for tok, valI := range vectors[i] {
				if valJ, exists := vectors[j][tok]; exists {
					dotProduct += valI * valJ
					shared = append(shared, tok)
				}
			}

			sim := dotProduct / (norms[i] * norms[j])
			// Select pairs that are related but distinct (0.25 <= sim <= 0.90)
			if sim >= 0.25 && sim <= 0.90 {
				pairs = append(pairs, ChunkPair{
					ChunkA:     g.Chunks[i],
					ChunkB:     g.Chunks[j],
					Similarity: sim,
					SharedKeys: shared,
				})
			}
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})

	return pairs
}

// FindDistractorChunk returns the highest-similarity non-identical chunk for a given target chunk.
func (g *TopologyGraph) FindDistractorChunk(targetChunkID string) (*Chunk, []string) {
	for _, p := range g.HighOverlapPairs {
		if p.ChunkA.ID == targetChunkID && p.ChunkB.ID != targetChunkID {
			return p.ChunkB, p.SharedKeys
		}
		if p.ChunkB.ID == targetChunkID && p.ChunkA.ID != targetChunkID {
			return p.ChunkA, p.SharedKeys
		}
	}
	// Fallback to any different chunk
	for _, ch := range g.Chunks {
		if ch.ID != targetChunkID {
			return ch, []string{"configuration"}
		}
	}
	return nil, nil
}
