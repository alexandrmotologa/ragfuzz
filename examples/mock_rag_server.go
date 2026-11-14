package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/corpus"
)

type queryRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

type chunkResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Rank int    `json:"rank"`
}

type queryResponse struct {
	Answer          string          `json:"answer"`
	RetrievedChunks []chunkResponse `json:"chunks"`
}

func main() {
	port := flag.Int("port", 8080, "HTTP port to bind mock RAG server")
	corpusPath := flag.String("corpus", "./examples/corpus", "Path to corpus directory")
	flag.Parse()

	cfg := &config.CorpusConfig{
		Paths:          []string{*corpusPath},
		ChunkSize:      800,
		ChunkOverlap:   150,
		MinChunkLength: 30,
	}

	chunks, err := corpus.IngestCorpus(cfg)
	if err != nil {
		log.Fatalf("Failed to ingest corpus from %q: %v", *corpusPath, err)
	}

	topology := corpus.BuildTopology(chunks)
	fmt.Printf("[+] Mock RAG server loaded %d corpus chunks from %q\n", len(chunks), *corpusPath)

	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req queryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		query := strings.TrimSpace(req.Query)
		topK := req.TopK
		if topK <= 0 {
			topK = 3
		}

		// Check for Out-of-Domain indicators
		qLower := strings.ToLower(query)
		isOOD := strings.Contains(qLower, "quantum") ||
			strings.Contains(qLower, "supersonic") ||
			strings.Contains(qLower, "ipfs") ||
			strings.Contains(qLower, "underwater") ||
			strings.Contains(qLower, "zero-gravity") ||
			strings.Contains(qLower, "smart contracts") ||
			strings.Contains(qLower, "cryogenic") ||
			strings.Contains(qLower, "mars orbital") ||
			strings.Contains(qLower, "palm vein") ||
			strings.Contains(qLower, "drone delivery") ||
			strings.Contains(qLower, "photonic") ||
			strings.Contains(qLower, "solar superflare")

		if isOOD {
			resp := queryResponse{
				Answer:          "I do not have information regarding this capability in the provided documentation.",
				RetrievedChunks: []chunkResponse{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Perform BM25 Retrieval
		retrieved := topology.SearchBM25(query, topK)

		var chunkResponses []chunkResponse
		for rank, ch := range retrieved {
			chunkResponses = append(chunkResponses, chunkResponse{
				ID:   ch.ID,
				Text: ch.Text,
				Rank: rank + 1,
			})
		}

		// Synthesize answer based on top retrieved chunk
		var answer string
		if len(retrieved) > 0 {
			topText := retrieved[0].Text
			sentences := strings.Split(topText, ".")
			if len(sentences) > 0 && len(strings.TrimSpace(sentences[0])) > 15 {
				answer = strings.TrimSpace(sentences[0]) + "."
				if len(sentences) > 1 && len(strings.TrimSpace(sentences[1])) > 15 {
					answer += " " + strings.TrimSpace(sentences[1]) + "."
				}
			} else {
				answer = topText
			}
		} else {
			answer = "No relevant documentation was retrieved to answer this inquiry."
		}

		// Handle negation queries
		if strings.Contains(qLower, " not ") || strings.Contains(qLower, "unsupported") || strings.Contains(qLower, "exclude") {
			// Grounded answer stating the limitation
			if len(retrieved) > 0 {
				answer = "Based on our policy: " + answer
			}
		}

		resp := queryResponse{
			Answer:          answer,
			RetrievedChunks: chunkResponses,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("[+] Mock RAG server listening on http://localhost%s/query\n", addr)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
