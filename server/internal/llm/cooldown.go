package llm

import (
	"sync"
	"time"
)

// CooldownTracker enforces per-player prompt cooldowns.
// Thread-safe — called from readPump goroutines and the engine goroutine.
type CooldownTracker struct {
	mu       sync.RWMutex
	last     map[string]time.Time
	cooldown time.Duration
}

// NewCooldownTracker creates a tracker with the given cooldown interval.
func NewCooldownTracker(cooldown time.Duration) *CooldownTracker {
	return &CooldownTracker{
		last:     make(map[string]time.Time),
		cooldown: cooldown,
	}
}

// CanSubmit checks if a player can submit a prompt.
// Returns (true, 0) if allowed, or (false, remaining duration) if on cooldown.
func (ct *CooldownTracker) CanSubmit(playerID string) (bool, time.Duration) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	last, exists := ct.last[playerID]
	if !exists {
		return true, 0
	}

	elapsed := time.Since(last)
	if elapsed >= ct.cooldown {
		return true, 0
	}
	return false, ct.cooldown - elapsed
}

// Record marks the current time as the player's last prompt submission.
func (ct *CooldownTracker) Record(playerID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.last[playerID] = time.Now()
}

// Remove deletes a player's cooldown record (on disconnect).
func (ct *CooldownTracker) Remove(playerID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	delete(ct.last, playerID)
}
