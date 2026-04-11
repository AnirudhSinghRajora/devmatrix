package network

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	sendBufferSize = 256
	writeTimeout   = 5 * time.Second
	pingInterval   = 15 * time.Second
	pongTimeout    = 10 * time.Second
)

// Client represents a single WebSocket connection.
type Client struct {
	ID       string
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	closeOne sync.Once
}

func newClient(id string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   id,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
		hub:  hub,
	}
}

// Send enqueues a message for this specific client (non-blocking).
// Safe to call after the send channel has been closed.
func (c *Client) Send(msg []byte) (sent bool) {
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// closeSend closes the send channel exactly once.
func (c *Client) closeSend() {
	c.closeOne.Do(func() {
		close(c.send)
	})
}

// writePump reads from the send channel and writes to the WebSocket.
// Also sends periodic pings to detect dead connections.
func (c *Client) writePump(ctx context.Context) {
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return // channel closed by hub
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageBinary, msg)
			cancel()
			if err != nil {
				log.Warn().Err(err).Str("client", c.ID).Msg("write failed")
				return
			}
		case <-pingTicker.C:
			pingCtx, cancel := context.WithTimeout(ctx, pongTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				log.Warn().Err(err).Str("client", c.ID).Msg("ping timeout")
				return
			}
		}
	}
}

// readPump reads from the WebSocket. Blocks until the connection dies.
// Routes incoming messages by type.
func (c *Client) readPump(ctx context.Context) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
				log.Info().Str("client", c.ID).Msg("client closed connection")
			} else {
				log.Warn().Err(err).Str("client", c.ID).Msg("read error")
			}
			return
		}

		var env Envelope
		if err := msgpack.Unmarshal(data, &env); err != nil {
			log.Warn().Err(err).Str("client", c.ID).Msg("malformed message")
			continue
		}

		switch env.Type {
		case MsgTypePrompt:
			var prompt PromptPayload
			if err := msgpack.Unmarshal(env.Payload, &prompt); err != nil {
				log.Warn().Err(err).Str("client", c.ID).Msg("malformed prompt payload")
				continue
			}
			if prompt.Text == "" {
				continue
			}
			if c.hub.promptCh != nil {
				select {
				case c.hub.promptCh <- PromptRequest{PlayerID: c.ID, Text: prompt.Text}:
				default:
					log.Warn().Str("client", c.ID).Msg("prompt queue full, dropping")
				}
			}
		default:
			// Unknown message types are silently ignored.
		}
	}
}

// Close shuts down the client connection.
func (c *Client) Close() {
	c.conn.Close(websocket.StatusGoingAway, "server shutting down")
}
