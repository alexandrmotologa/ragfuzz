package oracle

import "time"

// RAGResponse holds the extracted output from a RAG pipeline request.
type RAGResponse struct {
	Answer          string                 `json:"answer"`
	RetrievedChunks []RetrievedChunk       `json:"retrieved_chunks"`
	RawResponse     map[string]interface{} `json:"raw_response,omitempty"`
	Latency         time.Duration          `json:"latency"`
	StatusCode      int                    `json:"status_code"`
	Error           string                 `json:"error,omitempty"`
}

// RetrievedChunk represents an individual chunk returned by the RAG retriever.
type RetrievedChunk struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Rank int    `json:"rank"`
}
