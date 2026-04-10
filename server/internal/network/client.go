package network

import (
"context"
"time"

"github.com/coder/websocket"
"github.com/rs/zerolog/log"
)

const (
sendBufferSize = 256
writeTimeout   = 5 * time.Second
)

// Client represents a single WebSocket connection.
type Client struct {
	ID   string
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
}

func newClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
		hub:  hub,
	}
}

// writePump reads from the send channel and writes to the WebSocket.
func (c *Client) writePump(ctx context.Context) {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageBinary, msg)
			cancel()
			if err != nil {
				log.Warn().Err(err).Str("client", c.ID).Msg("write failed")
				return
			}
		}
	}
}

// readPump reads from the WebSocket. Phase 1: drains messages to keep connection alive.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, _, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				log.Info().Str("client", c.ID).Msg("client disconnected")
			} else {
				log.Warn().Err(err).Str("client", c.ID).Msg("read error")
			}
			return
		}
	}
}

// Close shuts down the client connection.
func (c *Client) Close() {
	c.conn.Close(websocket.StatusGoingAway, "server shutting down")
}
