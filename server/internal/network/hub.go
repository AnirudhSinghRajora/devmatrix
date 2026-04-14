package network

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
)

// Hub manages all active WebSocket clients and bridges connections to the game engine.
type Hub struct {
	mu             sync.RWMutex
	clients        map[string]*Client
	nextID         atomic.Uint64
	allowedOrigins []string
	maxPlayers     int
	joinCh         chan<- JoinRequest
	leaveCh        chan<- string
	promptCh       chan<- PromptRequest
	authValidator  AuthValidator // nil = anonymous mode
}

// AuthValidator validates a JWT token and returns a player identity.
type AuthValidator interface {
	ValidateWSToken(tokenStr string) (playerID string, username string, err error)
}

// NewHub creates a new connection hub.
// joinCh and leaveCh are write-only handles to the engine's input channels.
func NewHub(allowedOrigins []string, maxPlayers int, joinCh chan<- JoinRequest, leaveCh chan<- string, promptCh chan<- PromptRequest) *Hub {
	return &Hub{
		clients:        make(map[string]*Client),
		allowedOrigins: allowedOrigins,
		maxPlayers:     maxPlayers,
		joinCh:         joinCh,
		leaveCh:        leaveCh,
		promptCh:       promptCh,
	}
}

// SetAuthValidator enables JWT authentication on WebSocket connections.
func (h *Hub) SetAuthValidator(v AuthValidator) {
	h.authValidator = v
}

// HandleWebSocket upgrades an HTTP request and manages the full connection lifecycle.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h.ClientCount() >= h.maxPlayers {
		http.Error(w, `{"error":"server full"}`, http.StatusServiceUnavailable)
		return
	}

	var id, username string

	tokenStr := r.URL.Query().Get("token")
	hullID := r.URL.Query().Get("hull")

	if h.authValidator != nil && tokenStr != "" {
		// Authenticated mode: validate JWT.
		var err error
		id, username, err = h.authValidator.ValidateWSToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		// Prevent duplicate sessions.
		h.mu.RLock()
		_, alreadyConnected := h.clients[id]
		h.mu.RUnlock()
		if alreadyConnected {
			http.Error(w, `{"error":"already connected"}`, http.StatusConflict)
			return
		}
	} else {
		// Guest / anonymous mode.
		id = fmt.Sprintf("guest_%d", h.nextID.Add(1))
		username = fmt.Sprintf("Guest_%d", h.nextID.Load())
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.allowedOrigins,
	})
	if err != nil {
		log.Error().Err(err).Msg("websocket accept failed")
		return
	}

	client := newClient(id, conn, h)

	h.mu.Lock()
	h.clients[id] = client
	h.mu.Unlock()

	log.Info().Str("client", id).Str("username", username).Int("total", h.ClientCount()).Msg("client connected")

	// Notify engine of new player.
	h.joinCh <- JoinRequest{PlayerID: id, Username: username, Client: client, HullID: hullID}

	// Connection-scoped context: when either pump dies, the other is cancelled.
	connCtx, connCancel := context.WithCancel(r.Context())
	defer connCancel()

	go func() {
		client.writePump(connCtx)
		connCancel() // write failure kills the read side
	}()
	client.readPump(connCtx)

	// Client disconnected — clean up.
	h.unregister(client)
	conn.CloseNow()
	h.leaveCh <- id
}

// unregister removes a client from the hub and closes its send channel.
func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[c.ID]; ok {
		delete(h.clients, c.ID)
		c.closeSend()
		log.Info().Str("client", c.ID).Int("total", len(h.clients)).Msg("client disconnected")
	}
}

// Broadcast sends a message to all connected clients.
// Drops slow consumers whose send buffers are full.
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
				c.conn.CloseNow()
				h.leaveCh <- c.ID
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

// SendTo sends a message to a specific client by player ID (non-blocking).
func (h *Hub) SendTo(playerID string, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if c, ok := h.clients[playerID]; ok {
		c.Send(msg)
	}
}

// Shutdown closes all client connections.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		client.conn.CloseNow()
	}
}
