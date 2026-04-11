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
		return fmt.Errorf("missing movement")
	}
	if !validMovements[b.Movement] {
		return fmt.Errorf("unknown movement: %q", b.Movement)
	}
	if !validCombats[b.Combat] {
		return fmt.Errorf("unknown combat: %q", b.Combat)
	}
	if !validDefenses[b.Defense] {
		return fmt.Errorf("unknown defense: %q", b.Defense)
	}

	// Validate target selectors.
	if t := b.MovementParams.Target; t != "" {
		if !isValidTarget(t) {
			return fmt.Errorf("unknown movement target: %q", t)
		}
	}
	if t := b.CombatParams.Target; t != "" {
		if !isValidTarget(t) {
			return fmt.Errorf("unknown combat target: %q", t)
		}
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

	// Validate patrol waypoints.
	if b.Movement == "patrol" && len(b.MovementParams.Waypoints) == 0 {
		return fmt.Errorf("patrol requires at least one waypoint")
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
