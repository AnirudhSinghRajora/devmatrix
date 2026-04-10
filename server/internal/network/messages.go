package network

import (
"github.com/vmihailenco/msgpack/v5"
)

// Message type constants
const (
MsgTypeStateUpdate uint8 = 1
MsgTypePrompt      uint8 = 2
MsgTypeError       uint8 = 3
)

// Envelope wraps all messages with a type discriminator.
type Envelope struct {
	Type    uint8              `msgpack:"t"`
	Payload msgpack.RawMessage `msgpack:"p"`
}

// EntityState is the per-entity snapshot sent to clients each tick.
type EntityState struct {
	ID       uint32     `msgpack:"id"`
	Position [3]float32 `msgpack:"pos"`
	Rotation [4]float32 `msgpack:"rot"`
}

// StateUpdatePayload is the full world snapshot sent each tick.
type StateUpdatePayload struct {
	Tick     uint64        `msgpack:"tick"`
	Entities []EntityState `msgpack:"entities"`
}

// EncodeEnvelope serializes an envelope with an already-encoded payload.
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
