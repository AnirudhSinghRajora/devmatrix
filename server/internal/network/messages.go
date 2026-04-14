package network

import (
	"github.com/vmihailenco/msgpack/v5"
)

// Message type constants.
const (
	MsgTypeStateUpdate uint8 = 1
	MsgTypeWelcome     uint8 = 2
	MsgTypePrompt      uint8 = 3
	MsgTypeEvent       uint8 = 4
	MsgTypeError       uint8 = 5
)

// Envelope wraps all messages with a type discriminator.
type Envelope struct {
	Type    uint8              `msgpack:"t"`
	Payload msgpack.RawMessage `msgpack:"p"`
}

// EntitySnapshot is the per-entity state sent to clients each tick.
// All msgpack tags are single-character for minimal wire size.
type EntitySnapshot struct {
	ID       string     `msgpack:"i"`
	Position [3]float32 `msgpack:"p"`
	Rotation [4]float32 `msgpack:"r"`
	Color    [3]float32 `msgpack:"c"`
	Health   float32    `msgpack:"h"`
	MaxHP    float32    `msgpack:"mh"`
	Shield   float32    `msgpack:"s"`
	MaxShld  float32    `msgpack:"ms"`
	Alive    bool       `msgpack:"a"`
	Username string     `msgpack:"u,omitempty"`
	HullID   string     `msgpack:"hl,omitempty"`
}

// ProjectileSnapshot is a projectile's state sent to clients.
type ProjectileSnapshot struct {
	ID       uint64     `msgpack:"i"`
	Position [3]float32 `msgpack:"p"`
	Owner    string     `msgpack:"o"`
}

// GameEventWire is a one-shot visual event sent alongside the tick update.
type GameEventWire struct {
	Type   uint8      `msgpack:"t"`
	From   [3]float32 `msgpack:"f,omitempty"`
	To     [3]float32 `msgpack:"to,omitempty"`
	Hit    bool       `msgpack:"h,omitempty"`
	Killer string     `msgpack:"k,omitempty"`
	Victim string     `msgpack:"v,omitempty"`
}

// StateUpdatePayload is the full world snapshot sent each tick.
type StateUpdatePayload struct {
	Tick        uint64               `msgpack:"t"`
	Entities    []EntitySnapshot     `msgpack:"e"`
	Projectiles []ProjectileSnapshot `msgpack:"pr,omitempty"`
	Events      []GameEventWire      `msgpack:"ev,omitempty"`
}

// WelcomePayload is sent once to a newly connected client.
type WelcomePayload struct {
	PlayerID string           `msgpack:"pid"`
	Tick     uint64           `msgpack:"t"`
	Entities []EntitySnapshot `msgpack:"e"`
}

// JoinRequest is sent from the Hub to the Engine when a client connects.
type JoinRequest struct {
	PlayerID string
	Username string
	Client   *Client
	HullID   string // optional hull selection from query param (guests)
}

// PromptRequest is sent from the readPump to the Engine when a client submits a prompt.
type PromptRequest struct {
	PlayerID string
	Text     string
}

// PromptPayload is the client → server prompt message payload.
type PromptPayload struct {
	Text string `msgpack:"text"`
}

// ErrorPayload is a server → client error message.
type ErrorPayload struct {
	Message  string  `msgpack:"msg"`
	Cooldown float32 `msgpack:"cd,omitempty"` // remaining cooldown in seconds
}

// BehaviorEventPayload is sent to confirm a behavior was applied.
type BehaviorEventPayload struct {
	Movement string `msgpack:"m"`
	Combat   string `msgpack:"c,omitempty"`
	Defense  string `msgpack:"d,omitempty"`
}

// EncodeEnvelope serializes a typed envelope wrapping an already-marshalled payload.
func EncodeEnvelope(msgType uint8, payload interface{}) ([]byte, error) {
	raw, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return msgpack.Marshal(&Envelope{
		Type:    msgType,
		Payload: raw,
	})
}
