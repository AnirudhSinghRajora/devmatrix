package game

// BehaviorSet is the LLM output: a primary behavior with optional conditional overrides.
type BehaviorSet struct {
	Primary      BehaviorBlock      `json:"primary"`
	Conditionals []ConditionalBlock `json:"conditionals,omitempty"`
}

// BehaviorBlock describes what a ship should do each tick.
type BehaviorBlock struct {
	Movement       string         `json:"movement"`
	MovementParams MovementParams `json:"movement_params,omitempty"`
	Combat         string         `json:"combat,omitempty"`
	CombatParams   CombatParams   `json:"combat_params,omitempty"`
	Defense        string         `json:"defense,omitempty"`
}

// MovementParams holds parameters for the active movement primitive.
type MovementParams struct {
	Target    string       `json:"target,omitempty"`    // selector for chase/orbit/flee/strafe
	Speed     float32      `json:"speed,omitempty"`     // units per second
	Radius    float32      `json:"radius,omitempty"`    // for orbit
	Direction string       `json:"direction,omitempty"` // "left" or "right" for strafe
	Waypoints [][3]float32 `json:"waypoints,omitempty"` // for patrol
	Position  [3]float32   `json:"position,omitempty"`  // for move_to
}

// CombatParams holds parameters for the active combat primitive.
type CombatParams struct {
	Target string `json:"target,omitempty"`
	Weapon string `json:"weapon,omitempty"` // "primary" or "secondary"
}

// ConditionalBlock is a behavior override that activates when its condition is true.
type ConditionalBlock struct {
	ConditionStr string     `json:"condition"`
	Condition    *Condition `json:"-"` // parsed at validation time
	BehaviorBlock
}

// --- Validation sets ---

var validMovements = map[string]bool{
	"idle": true, "move_to": true, "orbit": true, "chase": true,
	"flee": true, "patrol": true, "strafe": true, "wander": true,
}

var validCombats = map[string]bool{
	"fire_at": true, "hold_fire": true, "": true,
}

var validDefenses = map[string]bool{
	"shield_front": true, "shield_balanced": true, "shield_rear": true, "": true,
}
