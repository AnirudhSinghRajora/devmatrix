package db

import "github.com/google/uuid"

// HullStats parsed from items.stats JSONB.
type HullStats struct {
	MaxHealth       float32 `json:"max_health"`
	MaxSpeed        float32 `json:"max_speed"`
	Thrust          float32 `json:"thrust"`
	CollisionRadius float32 `json:"collision_radius"`
}

// WeaponStats parsed from items.stats JSONB.
type WeaponStats struct {
	Type     string  `json:"type"`
	Damage   float32 `json:"damage"`
	Cooldown float32 `json:"cooldown"`
	Range    float32 `json:"range"`
	Speed    float32 `json:"speed"`
	Spread   float32 `json:"spread"`
}

// ShieldStats parsed from items.stats JSONB.
type ShieldStats struct {
	MaxShield float32 `json:"max_shield"`
	Regen     float32 `json:"regen"`
	Delay     float32 `json:"delay"`
}

// PlayerProfile holds profile + loadout data loaded for ship spawning.
type PlayerProfile struct {
	Coins             int
	Kills             int
	Deaths            int
	AITier            int
	HullID            string
	PrimaryWeaponID   string
	SecondaryWeaponID *string // nullable
	ShieldID          string
}

// LeaderboardEntry is one row of the kills leaderboard.
type LeaderboardEntry struct {
	Username string `json:"username"`
	Kills    int    `json:"kills"`
	Deaths   int    `json:"deaths"`
	AITier   int    `json:"ai_tier"`
}

// ShopItem is the public catalog item.
type ShopItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Price        int    `json:"price"`
	TierRequired int    `json:"tier_required"`
	Description  string `json:"description"`
	Stats        string `json:"stats"` // raw JSON
}

// CoinAward is sent to the async DB writer.
type CoinAward struct {
	PlayerID uuid.UUID
	Amount   int
}

// KillRecord is sent to the async DB writer.
type KillRecord struct {
	KillerID uuid.UUID
	VictimID uuid.UUID
}
