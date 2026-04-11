package game

import (
	"fmt"
	"strconv"
	"strings"
)

// Condition is a parsed "field op value" expression evaluated each tick.
type Condition struct {
	Field    string
	Operator string
	Value    float64
}

var validOperators = map[string]bool{
	"<": true, ">": true, "<=": true, ">=": true, "==": true,
}

var validFields = map[string]bool{
	"self.health_pct":       true,
	"self.shield_pct":       true,
	"self.speed":            true,
	"self.speed_pct":        true,
	"target.distance":       true,
	"target.health_pct":     true,
	"target.speed":          true,
	"enemy_count":           true,
	"incoming_projectiles":  true,
}

// ParseCondition parses a string like "self.health_pct < 30" into a Condition.
func ParseCondition(expr string) (*Condition, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty condition")
	}

	// Try two-char operators first to avoid partial matches (e.g. "<=" vs "<").
	for _, op := range []string{"<=", ">=", "=="} {
		if idx := strings.Index(expr, op); idx > 0 {
			return buildCondition(expr[:idx], op, expr[idx+len(op):])
		}
	}
	for _, op := range []string{"<", ">"} {
		if idx := strings.Index(expr, op); idx > 0 {
			return buildCondition(expr[:idx], op, expr[idx+len(op):])
		}
	}

	return nil, fmt.Errorf("no valid operator in condition: %q", expr)
}

func buildCondition(field, op, value string) (*Condition, error) {
	field = strings.TrimSpace(field)
	value = strings.TrimSpace(value)

	if !validFields[field] {
		return nil, fmt.Errorf("unknown field: %q", field)
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid value %q: %w", value, err)
	}

	return &Condition{Field: field, Operator: op, Value: v}, nil
}

// ShipContext provides runtime values for condition evaluation.
type ShipContext struct {
	HealthPct           float64
	ShieldPct           float64
	Speed               float64
	SpeedPct            float64
	TargetDist          float64
	TargetHPPct         float64
	TargetSpeed         float64
	EnemyCount          int
	IncomingProjectiles int
}

// Evaluate returns true if the condition is satisfied by the given context.
func (c *Condition) Evaluate(ctx *ShipContext) bool {
	actual := ctx.resolve(c.Field)
	switch c.Operator {
	case "<":
		return actual < c.Value
	case ">":
		return actual > c.Value
	case "<=":
		return actual <= c.Value
	case ">=":
		return actual >= c.Value
	case "==":
		return actual == c.Value
	default:
		return false
	}
}

func (ctx *ShipContext) resolve(field string) float64 {
	switch field {
	case "self.health_pct":
		return ctx.HealthPct
	case "self.shield_pct":
		return ctx.ShieldPct
	case "self.speed":
		return ctx.Speed
	case "self.speed_pct":
		return ctx.SpeedPct
	case "target.distance":
		return ctx.TargetDist
	case "target.health_pct":
		return ctx.TargetHPPct
	case "target.speed":
		return ctx.TargetSpeed
	case "enemy_count":
		return float64(ctx.EnemyCount)
	case "incoming_projectiles":
		return float64(ctx.IncomingProjectiles)
	default:
		return 0
	}
}
