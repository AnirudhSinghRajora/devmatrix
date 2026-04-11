package db

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ItemCache holds all item stats in memory to avoid DB reads during gameplay.
type ItemCache struct {
	Hulls   map[string]HullStats
	Weapons map[string]WeaponStats
	Shields map[string]ShieldStats
}

func NewItemCache(ctx context.Context, pool *pgxpool.Pool) (*ItemCache, error) {
	cache := &ItemCache{
		Hulls:   make(map[string]HullStats),
		Weapons: make(map[string]WeaponStats),
		Shields: make(map[string]ShieldStats),
	}

	rows, err := pool.Query(ctx, `SELECT id, category, stats FROM items`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, category string
		var stats json.RawMessage
		if err := rows.Scan(&id, &category, &stats); err != nil {
			return nil, err
		}

		switch category {
		case "hull":
			var s HullStats
			if err := json.Unmarshal(stats, &s); err != nil {
				log.Warn().Err(err).Str("item", id).Msg("bad hull stats")
				continue
			}
			cache.Hulls[id] = s
		case "weapon":
			var s WeaponStats
			if err := json.Unmarshal(stats, &s); err != nil {
				log.Warn().Err(err).Str("item", id).Msg("bad weapon stats")
				continue
			}
			cache.Weapons[id] = s
		case "shield":
			var s ShieldStats
			if err := json.Unmarshal(stats, &s); err != nil {
				log.Warn().Err(err).Str("item", id).Msg("bad shield stats")
				continue
			}
			cache.Shields[id] = s
		}
	}

	log.Info().
		Int("hulls", len(cache.Hulls)).
		Int("weapons", len(cache.Weapons)).
		Int("shields", len(cache.Shields)).
		Msg("item cache loaded")

	return cache, nil
}

// GetHull returns hull stats for the given ID, falling back to hull_basic defaults.
func (c *ItemCache) GetHull(id string) HullStats {
	if s, ok := c.Hulls[id]; ok {
		return s
	}
	return HullStats{MaxHealth: 100, MaxSpeed: 50, Thrust: 40, CollisionRadius: 2}
}

// GetWeapon returns weapon stats for the given ID, falling back to starter laser defaults.
func (c *ItemCache) GetWeapon(id string) WeaponStats {
	if s, ok := c.Weapons[id]; ok {
		return s
	}
	return WeaponStats{Type: "laser", Damage: 8, Cooldown: 0.5, Range: 200, Speed: 0, Spread: 2}
}

// GetShield returns shield stats for the given ID, falling back to basic shield defaults.
func (c *ItemCache) GetShield(id string) ShieldStats {
	if s, ok := c.Shields[id]; ok {
		return s
	}
	return ShieldStats{MaxShield: 50, Regen: 5, Delay: 3}
}
