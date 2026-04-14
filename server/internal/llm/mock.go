package llm

import (
	"strings"

	"github.com/DevMatrix/server/internal/game"
)

// MockGenerate parses simple natural language commands into a BehaviorSet
// without calling an LLM. Used when LLM_URL is empty (local dev) or as
// fallback when the real LLM returns unparseable output.
func MockGenerate(text string) (*game.BehaviorSet, error) {
	text = strings.ToLower(strings.TrimSpace(text))

	movement := "wander"
	target := "nearest_enemy"
	var speed float32 = 30
	var radius float32 = 150
	direction := ""
	combat := "fire_at"
	defense := "shield_balanced"

	// Speed modifiers.
	switch {
	case has(text, "fast", "full speed", "quickly", "max speed", "maximum"):
		speed = 50
	case has(text, "slow", "slowly", "carefully", "cautious"):
		speed = 15
	}

	// Movement primitives — ordered from most specific to least to avoid
	// false positives (e.g. "hold position" before "hold fire").
	switch {
	// Evasive.
	case has(text, "barrel roll", "barrel_roll", "corkscrew", "spiral"):
		movement = "barrel_roll"
		radius = 8
	case has(text, "juke", "fake out", "feint"):
		movement = "juke"
	case has(text, "evade", "avoid projectile", "dodge fire"):
		movement = "evade"
	case has(text, "dodge", "jink"):
		movement = "dodge"
	// Tactical.
	case has(text, "intercept", "cut off", "head off"):
		movement = "intercept"
	case has(text, "kite", "keep distance", "stay at range", "hit and run"):
		movement = "kite"
		radius = 120
	case has(text, "flank", "get behind", "from behind", "from the side"):
		movement = "flank"
		if has(text, "left") {
			direction = "left"
		} else if has(text, "right") {
			direction = "right"
		} else {
			direction = "behind"
		}
	case has(text, "ram", "crash into", "collide", "smash"):
		movement = "ram"
		speed = 50
	case has(text, "escort", "guard", "protect", "bodyguard"):
		movement = "escort"
		radius = 30
	// Advanced.
	case has(text, "zigzag", "zig zag", "zig-zag", "weave"):
		movement = "zigzag"
		if has(text, "away", "retreat") {
			direction = "away"
		} else {
			direction = "toward"
		}
	case has(text, "anchor", "hold position", "stay here", "park"):
		movement = "anchor"
		combat = ""
	// Core.
	case has(text, "chase", "pursue", "follow", "hunt", "go after"):
		movement = "chase"
	case has(text, "orbit", "circle", "revolve", "loop around"):
		movement = "orbit"
	case has(text, "flee", "run away", "escape", "retreat", "get away"):
		movement = "flee"
		defense = "shield_rear"
	case has(text, "strafe"):
		movement = "strafe"
		if has(text, "left") {
			direction = "left"
		} else {
			direction = "right"
		}
	case has(text, "patrol", "scout"):
		movement = "wander" // patrol needs waypoints; wander is the safe fallback
	case has(text, "stop", "idle", "halt", "wait"):
		movement = "idle"
		combat = ""
	case has(text, "move to", "go to", "fly to"):
		movement = "move_to"
	case has(text, "wander", "roam", "drift", "explore"):
		movement = "wander"
	case has(text, "attack", "destroy", "kill", "eliminate", "engage"):
		movement = "chase"
	}

	// Target selectors.
	switch {
	case has(text, "weakest", "low health", "damaged"):
		target = "weakest_enemy"
	case has(text, "strongest", "toughest", "full health"):
		target = "strongest_enemy"
	case has(text, "random", "any enemy"):
		target = "random_enemy"
	case has(text, "lowest shield", "no shield", "unshielded"):
		target = "lowest_shield"
	case has(text, "threat", "danger"):
		target = "nearest_threat"
	}

	// Combat — check specific multi-word phrases before single keywords.
	switch {
	case has(text, "hold fire", "don't shoot", "ceasefire", "cease fire", "no fire", "stop shooting"):
		combat = "hold_fire"
	case has(text, "burst fire", "burst_fire", "burst shot", "rapid fire"):
		combat = "burst_fire"
	case has(text, "fire at will", "fire_at_will", "shoot everything", "shoot anyone"):
		combat = "fire_at_will"
	case has(text, "fire", "shoot", "attack", "hit", "blast", "laser"):
		combat = "fire_at"
	}

	// Defense.
	switch {
	case has(text, "shield front", "front shield", "forward shield"):
		defense = "shield_front"
	case has(text, "shield rear", "rear shield", "back shield"):
		defense = "shield_rear"
	case has(text, "shield omni", "omni shield", "all around shield", "360 shield"):
		defense = "shield_omni"
	case has(text, "shield balanced", "balanced shield"):
		defense = "shield_balanced"
	}

	// Infer sensible defaults based on movement.
	switch movement {
	case "flee", "evade":
		if !has(text, "fire", "shoot", "attack") {
			combat = "hold_fire"
		}
		if defense == "shield_balanced" {
			defense = "shield_rear"
		}
	case "ram":
		if defense == "shield_balanced" {
			defense = "shield_front"
		}
	case "idle", "anchor", "wander":
		if !has(text, "fire", "shoot", "attack") {
			combat = ""
		}
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

	// Set combat params when combat is active.
	if bs.Primary.Combat != "" && bs.Primary.Combat != "hold_fire" {
		bs.Primary.CombatParams = game.CombatParams{
			Target: target,
			Weapon: "primary",
		}
	}

	// Add a simple conditional for common survival patterns.
	if has(text, "but run", "but flee", "but escape", "if low", "when low", "if dying", "when hurt") {
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
