package llm

import "fmt"

// BuildSystemPrompt builds the LLM system prompt with ship context.
// The AI tier controls which sections are included.
func BuildSystemPrompt(tier int, shipCtx ShipInfo, enemies []EnemyInfo) string {
	p := `You are a spaceship AI translator. Convert the captain's natural language order into a JSON behavior command.
Respond with ONLY valid JSON. No explanation, no markdown.

## Available Behaviors
Movement: idle, move_to, orbit, chase, flee, patrol, strafe, wander
Combat: fire_at, hold_fire
Defense: shield_front, shield_balanced, shield_rear

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
self.health_pct, self.shield_pct, target.distance, target.health_pct, enemy_count
Operators: <, >, <=, >=, ==

## Examples

Captain: "orbit the nearest enemy and fire"
{"primary":{"movement":"orbit","movement_params":{"target":"nearest_enemy","radius":150,"speed":30},"combat":"fire_at","combat_params":{"target":"nearest_enemy","weapon":"primary"},"defense":"shield_balanced"}}

Captain: "run away from everyone"
{"primary":{"movement":"flee","movement_params":{"target":"nearest_enemy","speed":50},"combat":"hold_fire","defense":"shield_rear"}}

Captain: "chase the weakest ship but run if low health"
{"primary":{"movement":"chase","movement_params":{"target":"weakest_enemy","speed":40},"combat":"fire_at","combat_params":{"target":"weakest_enemy","weapon":"primary"},"defense":"shield_front"},"conditionals":[{"condition":"self.health_pct < 25","movement":"flee","movement_params":{"target":"nearest_enemy","speed":50},"combat":"hold_fire","defense":"shield_rear"}]}

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
