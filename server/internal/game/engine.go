package game

import (
"context"
"math"
"time"

"github.com/DevMatrix/server/internal/network"
"github.com/rs/zerolog/log"
)

// Engine runs the main game loop at a fixed tick rate.
type Engine struct {
	tickRate int
	tick     uint64
	hub      *network.Hub
}

// NewEngine creates a game engine with the given tick rate.
func NewEngine(tickRate int, hub *network.Hub) *Engine {
	return &Engine{
		tickRate: tickRate,
		hub:      hub,
	}
}

// Run starts the game loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	interval := time.Second / time.Duration(e.tickRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Int("tickRate", e.tickRate).Msg("game engine started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Uint64("finalTick", e.tick).Msg("game engine stopped")
			return
		case <-ticker.C:
			start := time.Now()
			e.tick++
			e.update()
			e.broadcast()

			elapsed := time.Since(start)
			if elapsed > interval {
				log.Warn().
					Dur("elapsed", elapsed).
					Dur("budget", interval).
					Uint64("tick", e.tick).
					Msg("tick exceeded budget")
			}
		}
	}
}

// update computes the world state for this tick.
func (e *Engine) update() {
	// Phase 1: position computed directly from tick counter in broadcast
}

// broadcast serializes the current state and sends it to all clients.
func (e *Engine) broadcast() {
	// Phase 1: one entity orbiting the origin to visually confirm the pipeline
	angle := float64(e.tick) * 0.02
	x := float32(math.Cos(angle) * 5)
	z := float32(math.Sin(angle) * 5)

	payload := network.StateUpdatePayload{
		Tick: e.tick,
		Entities: []network.EntityState{
			{
				ID:       1,
				Position: [3]float32{x, 0, z},
				Rotation: [4]float32{0, float32(math.Sin(angle / 2)), 0, float32(math.Cos(angle / 2))},
			},
		},
	}

	data, err := network.EncodeEnvelope(network.MsgTypeStateUpdate, payload)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode state")
		return
	}

	e.hub.Broadcast(data)
}
