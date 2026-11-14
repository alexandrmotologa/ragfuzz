package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEHub manages active browser client SSE connections.
type SSEHub struct {
	mu      sync.Mutex
	clients map[chan []byte]bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]bool),
	}
}

func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan []byte, 64)

	h.mu.Lock()
	h.clients[msgChan] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, msgChan)
		close(msgChan)
		h.mu.Unlock()
	}()

	// Send initial connection event
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected"}`)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
			flusher.Flush()
		}
	}
}

// Broadcast sends an event payload to all connected SSE clients.
func (h *SSEHub) Broadcast(eventType string, payload map[string]interface{}) {
	payload["type"] = eventType
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.clients {
		select {
		case ch <- data:
		default:
		}
	}
}
