package llm

import (
	"strings"

	"github.com/DevMatrix/server/internal/game"
)

// MockGenerate parses simple natural language commands into a BehaviorSet
// without calling an LLM. Used when LLM_URL is empty (local dev).
func MockGenerate(text string) (*game.BehaviorSet, error) {
	text = strings.ToLower(strings.TrimSpace(text))

	movement := "wander"
	target := "nearest_enemy"
	var speed float32 = 30
	var radius float32 = 150
	direction := "right"
	combat := ""
	defense := ""

	// Speed modifiers.
	switch {
	case has(text, "fast", "full speed", "quickly", "max speed"):
		speed = 50
	case has(text, "slow", "slowly", "carefully"):
		speed = 15
	}

	// Movement primitives.
	switch {
	case has(text, "chase", "pursue", "follow", "hunt"):
		movement = "chase"
	case has(text, "orbit", "circle", "revolve"):
		movement = "orbit"
	case has(text, "flee", "run away", "escape", "retreat"):
		movement = "flee"
	case has(text, "strafe"):
		movement = "strafe"
	case has(text, "patrol"):
		movement = "patrol"
	case has(text, "wander", "roam", "drift", "explore"):
		movement = "wander"
	case has(text, "stop", "idle", "halt", "wait", "hold position"):
		movement = "idle"
	case has(text, "move to", "go to", "fly to"):
		movement = "move_to"
	}

	// Target selectors.
	switch {
	case has(text, "weakest"):
		target = "weakest_enemy"
	case has(text, "strongest"):
		target = "strongest_enemy"
	case has(text, "random"):
		target = "random_enemy"
	case has(text, "lowest shield"):
		target = "lowest_shield"
	}

	// Combat.
	switch {
	case has(text, "fire", "shoot", "attack", "hit"):
		combat = "fire_at"
	case has(text, "hold fire", "don't shoot", "ceasefire"):
		combat = "hold_fire"
	}

	// Defense.
	switch {
	case has(text, "shield front", "front shield"):
		defense = "shield_front"
	case has(text, "shield rear", "rear shield"):
		defense = "shield_rear"
	case has(text, "shield balanced", "balanced shield"):
		defense = "shield_balanced"
	}

	// Direction for strafe.
	if has(text, "left") {
		direction = "left"
	}

	bs := &game.BehaviorSet{
		Primary: game.BehaviorBlock{
			Movement: movement,
			MovementParams: game.MovementParams{
				Target:    target,
				Speed:     speed,
				Radius:    radius,
				Direction: direction,
			},
			Combat:  combat,
			Defense: defense,
		},
	}

	// Add a simple conditional for common patterns.
	if has(text, "but run", "but flee", "but escape") && has(text, "low", "health", "dying") {
		bs.Conditionals = []game.ConditionalBlock{
			{
				ConditionStr: "self.health_pct < 25",
				Condition:    &game.Condition{Field: "self.health_pct", Operator: "<", Value: 25},
				BehaviorBlock: game.BehaviorBlock{
					Movement: "flee",
					MovementParams: game.MovementParams{
						Target: "nearest_enemy",
						Speed:  50,
					},
					Combat:  "hold_fire",
					Defense: "shield_rear",
				},
			},
		}
	}

	// Set combat params target if combat is active.
	if bs.Primary.Combat == "fire_at" {
		bs.Primary.CombatParams = game.CombatParams{
			Target: target,
			Weapon: "primary",
		}
	}

	return bs, nil
}

// has returns true if text contains any of the given substrings.
func has(text string, substrs ...string) bool {
	for _, s := range substrs {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}
