# DevMatrix — Enhancements

Identified bugs, gameplay improvements, network/UX polish, and performance fixes.

---

## Bug Fixes

### 1. Dead ships targeted by AI movement
**Files:** `server/internal/game/target.go`
- `bestEnemy()` iterates all ships without checking `IsAlive`, so chase/orbit/flee can lock onto corpses.
- `randomEnemy()` includes dead ships in the candidate pool.
- `player:<name>` selector returns a dead player, causing movement to drive at a wreck.
**Fix:** Add `!s.IsAlive` guard in all three functions.

### 2. Prompts accepted while dead / LLM results applied to dead ships
**Files:** `server/internal/game/engine.go`
- `handlePrompt()` doesn't check `ship.IsAlive` — dead players can queue LLM work.
- `drainLLMResults()` applies behavior to ships that may have died during LLM processing.
**Fix:** Early-return in both functions when `!ship.IsAlive`.

### 3. Cooldown consumed when LLM queue is full
**Files:** `server/internal/game/engine.go`
- `cooldown.Record()` is called before the non-blocking channel send. If the queue is full, the player gets "Server busy" but still pays the cooldown.
**Fix:** Move `cooldown.Record()` inside the successful send branch.

### 4. EnemyCount includes dead ships
**Files:** `server/internal/game/movement.go`
- `buildShipContext()` sets `EnemyCount: len(e.state.Ships) - 1`, counting dead ships.
**Fix:** Count only living ships with `IsAlive`.

### 5. Zero-length laser beam causes invalid quaternion
**Files:** `client/src/components/LaserBeam.tsx`
- When `from` and `to` coincide, `dir.length()` is 0 and `setFromUnitVectors()` produces NaN.
**Fix:** Guard against zero-length direction; skip rendering if length < epsilon.

### 6. Logout doesn't reset inGame state
**Files:** `client/src/App.tsx`
- After LOGOUT, `inGame` remains `true`. Re-authenticating skips the lobby and shows a broken game state.
**Fix:** Set `inGame = false` and clear `launchHull` on logout.

---

## Gameplay Enhancements

### 7. Spawn protection (invulnerability on respawn)
**Files:** `server/internal/game/state.go`, `server/internal/game/combat.go`
- Ships can be killed immediately after respawning with no counterplay.
**Fix:** Add a `SpawnProtection` timer (3 seconds); skip incoming damage while active.

### 8. Kill streak tracking
**Files:** `server/internal/game/state.go`, `server/internal/game/combat.go`, `server/internal/network/messages.go`, `client/src/types.ts`, `client/src/store/gameStore.ts`, `client/src/components/KillFeed.tsx`
- No indication of consecutive kills — a big engagement motivator in combat games.
**Fix:** Track `KillStreak` on each ship, include streak count in kill events, display in kill feed.

### 9. Death screen shows killer name
**Files:** `client/src/store/gameStore.ts`, `client/src/components/DeathScreen.tsx`
- "YOU DIED" screen doesn't tell you who killed you.
**Fix:** Track killer name in store when own player dies; display on death screen.

---

## Network & UX

### 10. Connection status indicator
**Files:** `client/src/store/gameStore.ts`, new component `client/src/components/ConnectionStatus.tsx`, `client/src/App.tsx`
- No persistent visual indicator for connection state (connecting, reconnecting, disconnected).
**Fix:** Add `connectionState` to store; render a small pill indicator in the HUD.

### 11. Exponential backoff on WebSocket reconnect
**Files:** `client/src/network/socket.ts`
- Fixed 3-second retry forever with no backoff or attempt limit.
**Fix:** Exponential backoff (1s → 2s → 4s → 8s, capped at 15s), reset on successful connect.

---

## Performance

### 12. Reuse scratch vectors in CameraFollow
**Files:** `client/src/components/CameraFollow.tsx`
- `.clone()` and `.sub()` allocate new `Vector3` objects every frame in `useFrame`.
**Fix:** Use module-level scratch vectors, mutate in place.

### 13. Reuse SpatialGrid GetNearby result slice
**Files:** `server/internal/game/grid.go`
- `GetNearby` allocates a new `[]*Ship` slice on every call (many calls per tick from projectiles + collisions).
**Fix:** Add a reusable `scratch` slice on the grid, reset and return it.

### 14. Clamp Mass to minimum positive value
**Files:** `server/internal/game/movement.go`
- `ship.Mass` appears in denominators (`Thrust / Mass`). A misconfigured zero mass causes NaN/Inf.
**Fix:** Clamp mass to minimum of 1.0 in `applyThrust` and `applyRotation`.
