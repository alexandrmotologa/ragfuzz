package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/ragfuzz/internal/config"
	"github.com/alexandrmotologa/ragfuzz/internal/oracle"
)

// RAGClient handles HTTP communication with the target RAG system.
type RAGClient struct {
	cfg        *config.TargetConfig
	httpClient *http.Client
	rateTicker *time.Ticker
	rateChan   <-chan time.Time
}

// NewRAGClient initializes a high-performance HTTP client with rate-limiting.
func NewRAGClient(cfg *config.TargetConfig) *RAGClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	var rateChan <-chan time.Time
	if cfg.RateLimitRPS > 0 {
		interval := time.Second / time.Duration(cfg.RateLimitRPS)
		ticker := time.NewTicker(interval)
		rateChan = ticker.C
	}

	return &RAGClient{
		cfg:        cfg,
		httpClient: client,
		rateChan:   rateChan,
	}
}

// Query sends an adversarial query to the RAG endpoint with exponential backoff on retries.
func (c *RAGClient) Query(ctx context.Context, query string) (*oracle.RAGResponse, error) {
	if c.rateChan != nil {
		select {
		case <-c.rateChan:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Prepare payload by substituting {query} into request_template
	bodyStr := strings.ReplaceAll(c.cfg.RequestTemplate, "{query}", escapeJSONString(query))

	maxRetries := 3
	backoff := 200 * time.Millisecond

	var lastErr error
	var resp *oracle.RAGResponse

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
			select {
			case <-time.After(backoff + jitter):
				backoff *= 2
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, c.cfg.Method, c.cfg.Endpoint, bytes.NewBufferString(bodyStr))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Apply headers
		for k, v := range c.cfg.Headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		httpResp, err := c.httpClient.Do(req)
		latency := time.Since(start)

		if err != nil {
			lastErr = err
			continue
		}

		bodyBytes, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()

		if readErr != nil {
			lastErr = readErr
			continue
		}

		// Check for rate limit or server busy status codes
		if httpResp.StatusCode == 429 || httpResp.StatusCode == 502 || httpResp.StatusCode == 503 || httpResp.StatusCode == 504 {
			lastErr = fmt.Errorf("server responded with status %d: %s", httpResp.StatusCode, string(bodyBytes))
			continue
		}

		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			return &oracle.RAGResponse{
				StatusCode: httpResp.StatusCode,
				Latency:    latency,
				Error:      fmt.Sprintf("HTTP %d: %s", httpResp.StatusCode, string(bodyBytes)),
			}, nil
		}

		parsed, err := c.parseResponse(bodyBytes)
		if err != nil {
			return &oracle.RAGResponse{
				StatusCode: httpResp.StatusCode,
				Latency:    latency,
				Error:      fmt.Sprintf("failed to parse response JSON: %v", err),
			}, nil
		}

		parsed.StatusCode = httpResp.StatusCode
		parsed.Latency = latency
		return parsed, nil
	}

	return resp, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

func (c *RAGClient) parseResponse(data []byte) (*oracle.RAGResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	ansPath := c.cfg.ResponseMapping.AnswerPath
	if ansPath == "" {
		ansPath = "answer"
	}

	chunksPath := c.cfg.ResponseMapping.ChunksPath
	if chunksPath == "" {
		chunksPath = "chunks"
	}

	answer := extractStringByPath(raw, ansPath)
	chunksList := extractChunksByPath(raw, chunksPath, c.cfg.ResponseMapping.ChunkIDPath, c.cfg.ResponseMapping.ChunkTextPath)

	return &oracle.RAGResponse{
		Answer:          answer,
		RetrievedChunks: chunksList,
		RawResponse:     raw,
	}, nil
}

func extractStringByPath(data map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = m[part]
	}

	if s, ok := current.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", current)
}

func extractChunksByPath(data map[string]interface{}, path string, idKey, textKey string) []oracle.RetrievedChunk {
	if idKey == "" {
		idKey = "id"
	}
	if textKey == "" {
		textKey = "text"
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}

	slice, ok := current.([]interface{})
	if !ok {
		return nil
	}

	var chunks []oracle.RetrievedChunk
	for rank, item := range slice {
		switch v := item.(type) {
		case string:
			chunks = append(chunks, oracle.RetrievedChunk{
				ID:   fmt.Sprintf("chunk-%d", rank+1),
				Text: v,
				Rank: rank + 1,
			})
		case map[string]interface{}:
			idStr := ""
			if idVal, exists := v[idKey]; exists {
				idStr = fmt.Sprintf("%v", idVal)
			}
			textStr := ""
			if textVal, exists := v[textKey]; exists {
				textStr = fmt.Sprintf("%v", textVal)
			} else if textVal, exists := v["content"]; exists {
				textStr = fmt.Sprintf("%v", textVal)
			}
			chunks = append(chunks, oracle.RetrievedChunk{
				ID:   idStr,
				Text: textStr,
				Rank: rank + 1,
			})
		}
	}

	return chunks
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	// Remove outer quotes
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
