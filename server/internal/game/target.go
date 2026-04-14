package game

import (
	"math/rand"
	"strings"
)

// resolveTarget picks a ship from the game state matching the given selector string.
// Returns nil if no valid target is found (e.g. the player is alone).
func (e *Engine) resolveTarget(ship *Ship, selector string) *Ship {
	if selector == "" {
		return nil
	}

	// Specific player by name or ID: "player:<name>"
	if strings.HasPrefix(selector, "player:") {
		name := strings.TrimPrefix(selector, "player:")
		nameLower := strings.ToLower(name)
		for _, s := range e.state.Ships {
			if s.ID == ship.ID || !s.IsAlive {
				continue
			}
			if s.ID == name || strings.EqualFold(s.Username, nameLower) {
				return s
			}
		}
		return nil
	}

	switch selector {
	case "random_enemy":
		return e.randomEnemy(ship)
	default:
		return e.bestEnemy(ship, selector)
	}
}

// bestEnemy uses a scoring function to pick the best target.
func (e *Engine) bestEnemy(ship *Ship, selector string) *Ship {
	var best *Ship
	var bestScore float32
	first := true

	for _, s := range e.state.Ships {
		if s.ID == ship.ID || !s.IsAlive {
			continue
		}

		var score float32
		switch selector {
		case "nearest_enemy":
			// Lower distance is better → negate so "less is better" becomes "more is better".
			score = -ship.Position.DistTo(s.Position)
		case "weakest_enemy":
			score = -s.HealthPct() // lower HP = better target
		case "strongest_enemy":
			score = s.HealthPct() // higher HP = better target
		case "lowest_shield":
			score = -s.ShieldPct() // lower shield = better target
		case "nearest_threat":
			// Phase 4 will track who's targeting whom. For now, use nearest.
			score = -ship.Position.DistTo(s.Position)
		default:
			score = -ship.Position.DistTo(s.Position) // fallback: nearest
		}

		if first || score > bestScore {
			best = s
			bestScore = score
			first = false
		}
	}
	return best
}

// randomEnemy picks a random ship that is not this ship.
func (e *Engine) randomEnemy(ship *Ship) *Ship {
	candidates := make([]*Ship, 0, len(e.state.Ships)-1)
	for _, s := range e.state.Ships {
		if s.ID != ship.ID && s.IsAlive {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rand.Intn(len(candidates))]
}
