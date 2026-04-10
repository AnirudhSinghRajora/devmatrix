package network

import (
"fmt"
"net/http"
"sync"
"sync/atomic"

"github.com/coder/websocket"
"github.com/rs/zerolog/log"
)

// Hub manages all active WebSocket clients.
type Hub struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	nextID         atomic.Uint64
	allowedOrigins []string
}

// NewHub creates a new connection hub.
func NewHub(allowedOrigins []string) *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		allowedOrigins: allowedOrigins,
	}
}

// HandleWebSocket upgrades an HTTP request to a WebSocket connection.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
	})
	if err != nil {
		log.Error().Err(err).Msg("websocket accept failed")
		return
	}

	id := fmt.Sprintf("client_%d", h.nextID.Add(1))
	client := newClient(id, conn, h)

	h.mu.Lock()
	h.clients[id] = client
	h.mu.Unlock()

	log.Info().Str("client", id).Int("total", h.ClientCount()).Msg("client connected")

	ctx := r.Context()
	go client.writePump(ctx)
	client.readPump(ctx)
}

// unregister removes a client from the hub.
func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c.ID]; ok {
		delete(h.clients, c.ID)
		close(c.send)
		log.Info().Str("client", c.ID).Int("total", len(h.clients)).Msg("client removed")
	}
	h.mu.Unlock()
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- msg:
		default:
			log.Warn().Str("client", client.ID).Msg("slow consumer, dropping")
			go func(c *Client) {
				h.unregister(c)
				c.conn.Close(websocket.StatusPolicyViolation, "slow consumer")
			}(client)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Shutdown closes all client connections.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, client := range h.clients {
		client.Close()
	}
}
