package llm

import "fmt"

// BuildSystemPrompt builds the LLM system prompt with ship context.
// The AI tier controls which sections are included.
func BuildSystemPrompt(tier int, shipCtx ShipInfo, enemies []EnemyInfo) string {
	p := `You are a spaceship AI translator. Convert the captain's natural language order into a JSON behavior command.
Respond with ONLY valid JSON. No explanation, no markdown.

## Available Behaviors

### Movement
- idle: Stop and coast
- chase: Follow target directly
- flee: Run away from target
- orbit: Circle around target at a radius
- wander: Drift randomly
- patrol: Follow waypoints in sequence
- strafe: Move perpendicular to target (direction: "left"/"right")
- move_to: Go to position [x,y,z]
- dodge: Erratic lateral jinking to avoid fire (needs target)
- barrel_roll: Corkscrew spiral along heading (radius: spiral width, default 8)
- juke: Hard 90° cuts at random intervals (needs target)
- evade: Dodge actual incoming projectiles intelligently
- intercept: Predict target's future position and cut them off
- kite: Maintain ideal distance — flee if too close, chase if too far (radius: ideal distance, default 120)
- flank: Approach from side/behind (direction: "left"/"right"/"behind")
- ram: Full-speed collision course at target
- escort: Follow and guard a target ship (radius: follow distance, default 30)
- zigzag: Sawtooth approach/retreat (direction: "toward"/"away")
- anchor: Hold current position against drift

### Combat
- fire_at: Fire at specified target
- hold_fire: Cease fire
- burst_fire: 3 rapid shots then pause (more burst DPS)
- fire_at_will: Fire at any enemy in range (opportunistic)

### Defense
- shield_front: 1.5x absorption from front, 0.5x from rear
- shield_balanced: 1.0x absorption from all directions
- shield_rear: 1.5x absorption from rear, 0.5x from front
- shield_omni: 1.2x absorption all directions but shield regens 20% slower

## Available Targets
nearest_enemy, weakest_enemy, strongest_enemy, nearest_threat, lowest_shield, random_enemy, player:<name>

`

	// Ship status (all tiers).
	p += fmt.Sprintf(`## Ship Status
Health: %.0f%% | Shield: %.0f%%
Position: (%.0f, %.0f, %.0f)
`, shipCtx.HealthPct, shipCtx.ShieldPct, shipCtx.Pos[0], shipCtx.Pos[1], shipCtx.Pos[2])

	// Nearby enemies (tier 3+).
	if tier >= 3 && len(enemies) > 0 {
		maxEnemies := 3
		if tier >= 4 {
			maxEnemies = 5
		}
		if tier >= 5 {
			maxEnemies = len(enemies)
		}
		if maxEnemies > len(enemies) {
			maxEnemies = len(enemies)
		}
		p += "\n## Nearby Ships\n"
		for i := 0; i < maxEnemies; i++ {
			e := enemies[i]
			p += fmt.Sprintf("- %s: Distance %.0f, Health %.0f%%, Shield %.0f%%\n",
				e.ID, e.Distance, e.HealthPct, e.ShieldPct)
		}
	}

	// Output schema.
	p += `
## Output Format
{
  "primary": {
    "movement": "<behavior>",
    "movement_params": { "target": "<selector>", "speed": <number>, ... },
    "combat": "<behavior>",
    "combat_params": { "target": "<selector>", "weapon": "primary" },
    "defense": "<behavior>"
  }`

	if tier >= 2 {
		p += `,
  "conditionals": [
    {
      "condition": "<field> <op> <value>",
      "movement": "...", "movement_params": {...},
      "combat": "...", "defense": "..."
    }
  ]`
	}

	p += `
}

## Condition Fields (for conditionals)
self.health_pct, self.shield_pct, self.speed, self.speed_pct, target.distance, target.health_pct, target.speed, enemy_count, incoming_projectiles
Operators: <, >, <=, >=, ==

## Examples

Captain: "orbit the nearest enemy and fire"
{"primary":{"movement":"orbit","movement_params":{"target":"nearest_enemy","radius":150,"speed":30},"combat":"fire_at","combat_params":{"target":"nearest_enemy","weapon":"primary"},"defense":"shield_balanced"}}

Captain: "run away from everyone"
{"primary":{"movement":"flee","movement_params":{"target":"nearest_enemy","speed":50},"combat":"hold_fire","defense":"shield_rear"}}

Captain: "dodge and fire at the weakest, run if low health"
{"primary":{"movement":"dodge","movement_params":{"target":"weakest_enemy","speed":35},"combat":"fire_at","combat_params":{"target":"weakest_enemy","weapon":"primary"},"defense":"shield_balanced"},"conditionals":[{"condition":"self.health_pct < 25","movement":"flee","movement_params":{"target":"nearest_enemy","speed":50},"combat":"hold_fire","defense":"shield_rear"}]}

Captain: "kite at range and burst fire, dodge if taking fire"
{"primary":{"movement":"kite","movement_params":{"target":"nearest_enemy","speed":40,"radius":120},"combat":"burst_fire","combat_params":{"target":"nearest_enemy","weapon":"primary"},"defense":"shield_front"},"conditionals":[{"condition":"incoming_projectiles > 0","movement":"evade","movement_params":{"speed":50},"combat":"burst_fire","combat_params":{"target":"nearest_enemy","weapon":"primary"},"defense":"shield_omni"}]}

Now translate the captain's order:
`
	return p
}

// ShipInfo is a lightweight snapshot of ship state for prompt building.
type ShipInfo struct {
	HealthPct float32
	ShieldPct float32
	Pos       [3]float32
}

// EnemyInfo describes a nearby enemy for the system prompt.
type EnemyInfo struct {
	ID        string
	Distance  float32
	HealthPct float32
	ShieldPct float32
}
