# Behavior & Hitbox Overhaul Plan

## Status: DRAFT (planning only, not implemented)

## Files that will need changes:
- server/internal/game/state.go — Ship struct (compound hitbox fields)
- server/internal/game/collision.go — compound shape collision math
- server/internal/game/movement.go — new behavior implementations
- server/internal/game/behavior.go — validMovements map
- server/internal/game/behavior_parser.go — validation for new params
- server/internal/game/condition.go — new condition fields
- server/internal/game/engine.go — updateEntities pipeline, new context fields
- server/internal/llm/prompt.go — system prompt with new behaviors
- server/internal/db/migrations/002_seed_items.sql — hull hitbox shapes (if stored in DB)

---

## Part 1: Precise Hitboxes

**Current:** Single sphere per ship (`CollisionRadius`: 1.5–3.5). Every hull uses one bounding sphere regardless of actual geometry. A flat, wide ship like Imperial gets the same shape as a narrow Omen.

**Proposed: Compound Collision Shapes** — each hull defines an array of spheres that approximate its mesh silhouette. The server checks all sub-spheres for both ship-to-ship and projectile-to-ship collisions.

### Per-Hull Shape Definitions

| Hull | Current | Proposed Compound Shape |
|------|---------|------------------------|
| **Striker** (scout) | sphere r=2.0 | 2 spheres: fuselage (r=1.2, offset 0,0,0) + nose (r=0.8, offset 0,0,1.5) |
| **Challenger** (cruiser) | sphere r=2.5 | 3 spheres: body (r=1.5, offset 0,0,0) + left wing (r=0.8, offset -1.8,0,0) + right wing (r=0.8, offset 1.8,0,0) |
| **Imperial** (titan) | sphere r=3.5 | 3 spheres: core (r=2.0, offset 0,0,0) + front (r=1.5, offset 0,0,2.0) + rear (r=1.2, offset 0,0,-2.0) |
| **Omen** (stealth) | sphere r=1.5 | 2 spheres: body (r=1.0, offset 0,0,0) + tail (r=0.6, offset 0,0,-1.5) |

### Data Model Changes

**`state.go` — Ship struct:**
```go
type CollisionSphere struct {
    Offset Vec3    // local-space offset from ship center
    Radius float32
}

// Ship gains:
HitShape []CollisionSphere // compound hitbox (replaces single CollisionRadius for collision checks)
```

**Collision math:** For each pair of (shipA sub-sphere, shipB sub-sphere), transform offsets by ship rotation, then test sphere-sphere. Use the deepest-penetrating pair for impulse normal.

**Projectile hits:** `rayHitsSphere` and `spheresOverlap` loop over sub-spheres instead of the single radius. The broad-phase `CollisionRadius` stays as the bounding radius of the entire compound shape.

### Impact
- Ship-to-ship: Narrow ships can slip past wide ships. Wings clip before body.
- Lasers: Rays can miss if aimed at empty space between sub-spheres.
- Projectiles: Plasma bolts only damage on actual geometry contact.

---

## Part 2: New Behaviors

### Currently Available (8 movement + 2 combat + 3 defense)

| Movement | Combat | Defense |
|----------|--------|---------|
| idle, chase, flee, orbit, wander, patrol, strafe, move_to | fire_at, hold_fire | shield_front, shield_balanced, shield_rear |

### Proposed New Behaviors (11 new movements + 2 new combat + 1 new defense)

---

#### **EVASIVE BEHAVIORS** (4 new)

**1. `dodge`** — Erratic lateral jinking to avoid incoming fire. Rapidly alternates perpendicular direction every 0.3–0.8s while maintaining general heading toward/away from target.
```
Params: target (who to dodge from), speed
Logic:
  - toTarget = normalize(target.pos - ship.pos)
  - Every 0.3-0.8s: pick random perpendicular axis (up×toTarget or toTarget×up), flip sign randomly
  - desiredVel = lateralDir × speed × 0.7 + toTarget × speed × 0.3
  - Net effect: ship jinks left-right-up-down unpredictably while slowly drifting toward/away from threat
Use case: "dodge that guy's attacks", "evade incoming fire"
```

**2. `barrel_roll`** — Corkscrew spiral along current velocity direction. Ship traces a helix, making it extremely hard to hit with projectiles.
```
Params: speed, radius (spiral radius, default 8)
Logic:
  - forwardDir = normalize(ship.Velocity) or last non-zero velocity
  - angle += speed / radius × dt (angular velocity around forward axis)
  - perpA, perpB = two orthogonal axes perpendicular to forwardDir
  - offset = perpA × cos(angle) × radius + perpB × sin(angle) × radius
  - desiredVel = forwardDir × speed + (offset - lastOffset) / dt
  - Needs new transient fields: BarrelAngle float32, BarrelAxis Vec3
Use case: "do a barrel roll", "spiral toward them", "corkscrew in"
```

**3. `juke`** — Hard 90° cut in a random direction, then resume original heading after a brief delay. One-shot burst of lateral acceleration.
```
Params: target, speed
Logic:
  - JukeTimer counts down (starts at 0.8–1.2s)
  - Phase 1 (first 0.3s): desiredVel = randomPerpendicular × speed (hard cut)
  - Phase 2 (remaining): resume previous DesiredVelocity (chase/flee/etc.)
  - Reset JukeTimer to new random interval (1.5–3.0s) between jukes
  - Needs: JukeTimer, JukePhase, JukeDir transient fields
Use case: "juke left and right", "make unpredictable moves"
```

**4. `evade`** — Intelligent dodge: tracks incoming projectiles and moves perpendicular to their velocity vectors. More sophisticated than `dodge` — reacts to actual threats.
```
Params: speed
Logic:
  - Scan projectiles within 80 units heading toward this ship
  - For each incoming projectile: compute perpendicular escape direction
  - Average all escape vectors, normalize, scale by speed
  - If no threats detected: fallback to gentle wander
  - Needs: access to Projectiles list in engine (pass as param or via engine method)
Use case: "evade all incoming fire", "dodge those plasma bolts"
```

---

#### **TACTICAL BEHAVIORS** (5 new)

**5. `intercept`** — Predict target's future position based on their velocity and move to cut them off. Leads the target instead of chasing their current position.
```
Params: target, speed
Logic:
  - timeToTarget = dist / speed
  - predictedPos = target.pos + target.velocity × timeToTarget × 0.8
  - desiredVel = normalize(predictedPos - ship.pos) × speed
  - Clamp prediction horizon to 3 seconds max
Use case: "intercept that ship", "cut them off", "head them off"
```

**6. `kite`** — Maintain a specific distance from target: flee when too close, chase when too far. Ideal for ranged combat — keep firing while staying out of danger.
```
Params: target, speed, radius (ideal distance, default 120)
Logic:
  - dist = distance(ship, target)
  - if dist < radius - 20: flee (outward)
  - if dist > radius + 20: chase (inward)
  - else: strafe tangentially (orbit-like at ideal distance)
  - Always face target for weapon lock
Use case: "kite them at range", "keep your distance and fire", "hit and run"
```

**7. `flank`** — Approach target from the side or behind. Move to a position offset from the target's facing direction, then close in.
```
Params: target, speed, direction ("left" | "right" | "behind", default "behind")
Logic:
  - Get target's forward direction from their velocity
  - Compute flank position: target.pos + perpendicular × 80 (or -forward × 80 for behind)
  - If far from flank position: move_to flank position
  - If within 20 units of flank position: chase target directly
  - Needs: FlankPhase transient field
Use case: "flank them from behind", "get behind that ship", "attack from the side"
```

**8. `ram`** — Full-speed direct collision course toward target. Ignores MaxSpeed safety and boosts thrust temporarily for devastating kinetic impact.
```
Params: target, speed (uses max possible)
Logic:
  - dir = normalize(target.pos - ship.pos)
  - desiredVel = dir × ship.MaxSpeed × 1.2 (exceed normal cap temporarily)
  - On collision: deal damage proportional to relative velocity (bonus ram damage)
  - Needs: collision.go to check for ram state and apply bonus damage
Use case: "ram into them", "kamikaze", "full speed collision"
```

**9. `escort`** — Stay near a specific allied ship and match their velocity. Keep formation behind/beside them.
```
Params: target (player:<name>), speed, radius (follow distance, default 30)
Logic:
  - offset = -target.velocity.normalize() × radius (stay behind target)
  - formationPos = target.pos + offset
  - desiredVel = toward formationPos, scaled down when close
  - If within radius: match target.velocity exactly
Use case: "escort player:Bob", "follow that ship", "guard my wingman"
```

---

#### **ADVANCED MOVEMENT** (2 new)

**10. `zigzag`** — Move toward/away from target in a sawtooth pattern. Alternates between angled approaches.
```
Params: target, speed, direction ("toward" | "away", default "toward")
Logic:
  - toTarget = normalize(target.pos - ship.pos) (or negated for "away")
  - ZigTimer alternates every 1.0–1.5s
  - perpAxis = cross(toTarget, up).normalize()
  - angle = alternating +40° / -40° from toTarget
  - desiredVel = rotate(toTarget, ±40°, perpAxis) × speed
  - Needs: ZigTimer, ZigDirection transient fields
Use case: "zigzag toward them", "approach in zigzag"
```

**11. `anchor`** — Hold position at current location. Unlike `idle` (which coasts), this actively thrusts to maintain exact position against drift, collisions, and boundary forces.
```
Params: (none, or optional position override)
Logic:
  - anchorPos = ship.pos at time of behavior set (or MovementParams.Position if given)
  - desiredVel = (anchorPos - ship.pos) × 5.0 (P-controller to hold station)
  - Clamp desiredVel magnitude to maxSpeed × 0.5
Use case: "hold position", "stay here", "anchor at current location"
```

---

#### **NEW COMBAT ACTIONS** (2 new)

**12. `burst_fire`** — Fire rapidly in short bursts then pause. 3 shots in quick succession (0.15s interval), then 1.5s cooldown.
```
Params: target, weapon
Logic:
  - BurstCount tracks shots in current burst (0–3)
  - If BurstCount < 3: fire with 0.15s cooldown between shots
  - If BurstCount >= 3: wait 1.5s, reset BurstCount to 0
  - Needs: BurstCount, BurstTimer transient fields on Weapon or Ship
Use case: "burst fire at nearest enemy"
```

**13. `fire_at_will`** — Fire at ANY enemy in range, not just the specified target. Switches targets each shot for maximum chaos.
```
Params: weapon
Logic:
  - Each tick: find all enemies within weapon.Range
  - Pick the one closest to current aim direction (cheapest rotation)
  - Fire at them
  - No target loyalty — pure opportunistic
Use case: "fire at everything", "shoot anything in range", "fire at will"
```

---

#### **NEW DEFENSE MODE** (1 new)

**14. `shield_omni`** — Rapidly oscillating shield that provides 1.2x absorption from all directions but drains shield 20% faster.
```
Logic:
  - shieldMultiplier always returns 1.2
  - Shield regen is reduced by 20% while active
Use case: "full shields", "shields everywhere", "maximum shield coverage"
```

---

### New Condition Fields

| Field | Type | Description |
|-------|------|-------------|
| `self.speed` | float | Ship's current speed (0–MaxSpeed) |
| `self.speed_pct` | float | Speed as % of MaxSpeed |
| `incoming_projectiles` | int | Count of projectiles heading toward this ship within 100 units |
| `target.speed` | float | Target's current speed |

---

### New MovementParams Fields

```go
type MovementParams struct {
    Target    string       `json:"target,omitempty"`
    Speed     float32      `json:"speed,omitempty"`
    Radius    float32      `json:"radius,omitempty"`
    Direction string       `json:"direction,omitempty"`     // strafe, flank, zigzag
    Waypoints [][3]float32 `json:"waypoints,omitempty"`
    Position  [3]float32   `json:"position,omitempty"`
}
// No new fields needed — Direction already covers flank/zigzag side selection
```

### New Transient Fields on Ship

```go
// Dodge/evasive state.
DodgeTimer    float32  // countdown for next direction change
DodgeDir      Vec3     // current dodge direction

// Barrel roll state.
BarrelAngle   float32  // current angle in the spiral

// Juke state.
JukeTimer     float32  // countdown between jukes
JukePhase     int      // 0=normal, 1=cutting
JukeDir       Vec3     // direction of current juke

// Zigzag state.
ZigTimer      float32  // alternation timer
ZigLeft       bool     // current zigzag side

// Flank state.
FlankPhase    int      // 0=approaching flank pos, 1=attacking

// Anchor state.
AnchorPos     Vec3     // held position
AnchorSet     bool     // whether anchor position has been captured
```

### New Burst Fire Fields on Ship

```go
BurstCount int     // shots fired in current burst
BurstTimer float32 // cooldown between bursts
```

---

### Updated validMovements Map

```go
var validMovements = map[string]bool{
    "idle": true, "move_to": true, "orbit": true, "chase": true,
    "flee": true, "patrol": true, "strafe": true, "wander": true,
    // New evasive
    "dodge": true, "barrel_roll": true, "juke": true, "evade": true,
    // New tactical
    "intercept": true, "kite": true, "flank": true, "ram": true, "escort": true,
    // New advanced
    "zigzag": true, "anchor": true,
}

var validCombats = map[string]bool{
    "fire_at": true, "hold_fire": true, "": true,
    "burst_fire": true, "fire_at_will": true,
}

var validDefenses = map[string]bool{
    "shield_front": true, "shield_balanced": true, "shield_rear": true, "": true,
    "shield_omni": true,
}
```

---

### Summary Table

| Category | Behavior | Params | Difficulty | Description |
|----------|----------|--------|------------|-------------|
| **Evasive** | `dodge` | target, speed | Medium | Random lateral jinking |
| **Evasive** | `barrel_roll` | speed, radius | Medium | Helical spiral along heading |
| **Evasive** | `juke` | target, speed | Medium | Hard 90° cuts with recovery |
| **Evasive** | `evade` | speed | Hard | Track & dodge real projectiles |
| **Tactical** | `intercept` | target, speed | Easy | Lead target by velocity prediction |
| **Tactical** | `kite` | target, speed, radius | Easy | Maintain range + fire |
| **Tactical** | `flank` | target, speed, direction | Medium | Attack from side/behind |
| **Tactical** | `ram` | target | Easy | Full-speed collision + bonus damage |
| **Tactical** | `escort` | target, speed, radius | Medium | Follow and guard an ally |
| **Advanced** | `zigzag` | target, speed, direction | Easy | Sawtooth approach/retreat |
| **Advanced** | `anchor` | position | Easy | Station-keeping |
| **Combat** | `burst_fire` | target | Medium | 3-shot burst pattern |
| **Combat** | `fire_at_will` | - | Medium | Opportunistic multi-target |
| **Defense** | `shield_omni` | - | Easy | 1.2x all-around, faster drain |

**Total after implementation:** 19 movements + 4 combat + 4 defense + 9 condition fields
