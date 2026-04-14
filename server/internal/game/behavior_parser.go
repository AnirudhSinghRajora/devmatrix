package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseBehaviorJSON extracts, parses, and validates a BehaviorSet from LLM output text.
// Handles markdown code fences, leading prose, etc.
func ParseBehaviorJSON(raw string) (*BehaviorSet, error) {
	jsonStr := extractJSON(raw)

	var bs BehaviorSet
	if err := json.Unmarshal([]byte(jsonStr), &bs); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validateBehaviorBlock(&bs.Primary); err != nil {
		return nil, fmt.Errorf("primary: %w", err)
	}

	for i := range bs.Conditionals {
		cond, err := ParseCondition(bs.Conditionals[i].ConditionStr)
		if err != nil {
			return nil, fmt.Errorf("conditional[%d]: %w", i, err)
		}
		bs.Conditionals[i].Condition = cond

		if err := validateBehaviorBlock(&bs.Conditionals[i].BehaviorBlock); err != nil {
			return nil, fmt.Errorf("conditional[%d]: %w", i, err)
		}
	}

	return &bs, nil
}

func validateBehaviorBlock(b *BehaviorBlock) error {
	if b.Movement == "" {
		b.Movement = "idle"
	}
	if !validMovements[b.Movement] {
		b.Movement = fuzzyMovement(b.Movement)
	}
	if !validCombats[b.Combat] {
		b.Combat = fuzzyCombat(b.Combat)
	}
	if !validDefenses[b.Defense] {
		b.Defense = fuzzyDefense(b.Defense)
	}

	// Normalize target selectors — remap invalid targets to nearest_enemy.
	if t := b.MovementParams.Target; t != "" && !isValidTarget(t) {
		b.MovementParams.Target = fuzzyTarget(t)
	}
	if t := b.CombatParams.Target; t != "" && !isValidTarget(t) {
		b.CombatParams.Target = fuzzyTarget(t)
	}

	// Clamp numeric parameters to safe ranges.
	b.MovementParams.Speed = clampF32(b.MovementParams.Speed, 0, 100)
	if b.MovementParams.Radius != 0 {
		b.MovementParams.Radius = clampF32(b.MovementParams.Radius, 30, 500)
	}

	// Validate direction for strafe, flank, zigzag.
	switch b.Movement {
	case "strafe":
		if b.MovementParams.Direction == "" {
			b.MovementParams.Direction = "right"
		}
	case "flank":
		d := b.MovementParams.Direction
		if d != "left" && d != "right" && d != "behind" {
			b.MovementParams.Direction = "behind"
		}
	case "zigzag":
		d := b.MovementParams.Direction
		if d != "toward" && d != "away" {
			b.MovementParams.Direction = "toward"
		}
	}

	// Patrol without waypoints falls back to wander.
	if b.Movement == "patrol" && len(b.MovementParams.Waypoints) == 0 {
		b.Movement = "wander"
	}

	return nil
}

func isValidTarget(t string) bool {
	switch t {
	case "nearest_enemy", "weakest_enemy", "strongest_enemy",
		"nearest_threat", "lowest_shield", "random_enemy":
		return true
	}
	return strings.HasPrefix(t, "player:")
}

// fuzzyTarget maps an invalid target string to the best valid selector.
func fuzzyTarget(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))

	// If it looks like a player name reference, wrap it.
	for _, prefix := range []string{"player:", "player_", "player "} {
		if strings.HasPrefix(t, prefix) {
			name := strings.TrimPrefix(t, prefix)
			return "player:" + strings.TrimSpace(name)
		}
	}

	// Keyword matching.
	switch {
	case strings.Contains(t, "weak") || strings.Contains(t, "low health") || strings.Contains(t, "damaged"):
		return "weakest_enemy"
	case strings.Contains(t, "strong") || strings.Contains(t, "tough") || strings.Contains(t, "full health"):
		return "strongest_enemy"
	case strings.Contains(t, "shield") || strings.Contains(t, "unshielded"):
		return "lowest_shield"
	case strings.Contains(t, "threat") || strings.Contains(t, "danger"):
		return "nearest_threat"
	case strings.Contains(t, "random") || strings.Contains(t, "any"):
		return "random_enemy"
	default:
		// "other_ship", "enemy", "the ship", etc. → nearest
		return "nearest_enemy"
	}
}

// extractJSON finds the JSON object in a response that may contain markdown fences or prose.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Strip markdown code fences.
	if idx := strings.Index(s, "```json"); idx != -1 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx != -1 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end != -1 {
			s = s[:end]
		}
	}

	// Find the outermost { ... }.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

func clampF32(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// fuzzyMovement maps an invalid movement string to the closest valid one.
func fuzzyMovement(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	switch {
	case strings.Contains(m, "chase") || strings.Contains(m, "follow") || strings.Contains(m, "pursue") || strings.Contains(m, "attack"):
		return "chase"
	case strings.Contains(m, "flee") || strings.Contains(m, "run") || strings.Contains(m, "escape") || strings.Contains(m, "retreat"):
		return "flee"
	case strings.Contains(m, "orbit") || strings.Contains(m, "circle"):
		return "orbit"
	case strings.Contains(m, "dodge") || strings.Contains(m, "jink"):
		return "dodge"
	case strings.Contains(m, "barrel") || strings.Contains(m, "roll") || strings.Contains(m, "spin"):
		return "barrel_roll"
	case strings.Contains(m, "juke") || strings.Contains(m, "fake"):
		return "juke"
	case strings.Contains(m, "evade") || strings.Contains(m, "avoid"):
		return "evade"
	case strings.Contains(m, "intercept") || strings.Contains(m, "cut off"):
		return "intercept"
	case strings.Contains(m, "kite") || strings.Contains(m, "keep distance"):
		return "kite"
	case strings.Contains(m, "flank") || strings.Contains(m, "behind"):
		return "flank"
	case strings.Contains(m, "ram") || strings.Contains(m, "crash") || strings.Contains(m, "collide"):
		return "ram"
	case strings.Contains(m, "escort") || strings.Contains(m, "guard") || strings.Contains(m, "protect"):
		return "escort"
	case strings.Contains(m, "zigzag") || strings.Contains(m, "zig") || strings.Contains(m, "weave"):
		return "zigzag"
	case strings.Contains(m, "anchor") || strings.Contains(m, "stay") || strings.Contains(m, "hold") || strings.Contains(m, "stop"):
		return "anchor"
	case strings.Contains(m, "strafe") || strings.Contains(m, "side"):
		return "strafe"
	case strings.Contains(m, "patrol") || strings.Contains(m, "scout"):
		return "patrol"
	case strings.Contains(m, "wander") || strings.Contains(m, "roam") || strings.Contains(m, "explore"):
		return "wander"
	case strings.Contains(m, "destroy") || strings.Contains(m, "kill") || strings.Contains(m, "eliminate"):
		return "chase"
	default:
		return "chase"
	}
}

// fuzzyCombat maps an invalid combat string to the closest valid one.
func fuzzyCombat(c string) string {
	c = strings.ToLower(strings.TrimSpace(c))
	switch {
	case strings.Contains(c, "fire") || strings.Contains(c, "shoot") || strings.Contains(c, "attack") || strings.Contains(c, "destroy") || strings.Contains(c, "kill"):
		return "fire_at"
	case strings.Contains(c, "burst"):
		return "burst_fire"
	case strings.Contains(c, "hold") || strings.Contains(c, "stop") || strings.Contains(c, "cease") || strings.Contains(c, "peace"):
		return "hold_fire"
	default:
		return "fire_at"
	}
}

// fuzzyDefense maps an invalid defense string to the closest valid one.
func fuzzyDefense(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	switch {
	case strings.Contains(d, "front") || strings.Contains(d, "forward"):
		return "shield_front"
	case strings.Contains(d, "rear") || strings.Contains(d, "back"):
		return "shield_rear"
	case strings.Contains(d, "omni") || strings.Contains(d, "all"):
		return "shield_omni"
	default:
		return "shield_balanced"
	}
}
