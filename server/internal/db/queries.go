package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Queries provides parameterized database operations.
type Queries struct {
	Pool *pgxpool.Pool
}

func NewQueries(pool *pgxpool.Pool) *Queries {
	return &Queries{Pool: pool}
}

// GetProfile loads a player's profile + loadout for ship spawning.
func (q *Queries) GetProfile(ctx context.Context, userID uuid.UUID) (*PlayerProfile, error) {
	row := q.Pool.QueryRow(ctx, `
		SELECT p.coins, p.kills, p.deaths, p.ai_tier,
		       l.hull_id, l.primary_weapon, l.secondary_weapon, l.shield_id
		FROM profiles p
		JOIN loadouts l ON l.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID)

	var profile PlayerProfile
	err := row.Scan(
		&profile.Coins, &profile.Kills, &profile.Deaths, &profile.AITier,
		&profile.HullID, &profile.PrimaryWeaponID, &profile.SecondaryWeaponID, &profile.ShieldID,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// AwardCoins atomically adds coins to a player.
func (q *Queries) AwardCoins(ctx context.Context, userID uuid.UUID, amount int) error {
	_, err := q.Pool.Exec(ctx, `
		UPDATE profiles SET coins = coins + $2, updated_at = now() WHERE user_id = $1
	`, userID, amount)
	return err
}

// RecordKill logs a kill and increments counters in a transaction.
func (q *Queries) RecordKill(ctx context.Context, killerID, victimID uuid.UUID) error {
	tx, err := q.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `UPDATE profiles SET kills = kills + 1, updated_at = now() WHERE user_id = $1`, killerID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE profiles SET deaths = deaths + 1, updated_at = now() WHERE user_id = $1`, victimID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO kill_log (killer_id, victim_id) VALUES ($1, $2)`, killerID, victimID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PurchaseItem validates and executes an item purchase atomically.
func (q *Queries) PurchaseItem(ctx context.Context, userID uuid.UUID, itemID string) error {
	tx, err := q.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var price, tierReq int
	err = tx.QueryRow(ctx, `SELECT price, tier_required FROM items WHERE id = $1`, itemID).Scan(&price, &tierReq)
	if err != nil {
		return fmt.Errorf("item not found: %s", itemID)
	}

	var coins, aiTier int
	err = tx.QueryRow(ctx, `SELECT coins, ai_tier FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&coins, &aiTier)
	if err != nil {
		return err
	}

	if aiTier < tierReq {
		return fmt.Errorf("requires AI tier %d, you have %d", tierReq, aiTier)
	}
	if coins < price {
		return fmt.Errorf("insufficient coins: need %d, have %d", price, coins)
	}

	var exists bool
	_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory WHERE user_id=$1 AND item_id=$2)`, userID, itemID).Scan(&exists)
	if exists {
		return fmt.Errorf("item already owned")
	}

	if _, err = tx.Exec(ctx, `UPDATE profiles SET coins = coins - $2, updated_at = now() WHERE user_id = $1`, userID, price); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO inventory (user_id, item_id) VALUES ($1, $2)`, userID, itemID); err != nil {
		return err
	}

	// AI core upgrades the tier.
	if strings.HasPrefix(itemID, "ai_core_") {
		var newTier int
		err = tx.QueryRow(ctx, `SELECT (stats->>'ai_tier')::int FROM items WHERE id = $1`, itemID).Scan(&newTier)
		if err == nil && newTier > aiTier {
			if _, err = tx.Exec(ctx, `UPDATE profiles SET ai_tier = $2, updated_at = now() WHERE user_id = $1`, userID, newTier); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// EquipItem validates ownership and sets a loadout slot.
func (q *Queries) EquipItem(ctx context.Context, userID uuid.UUID, itemID, slot string) error {
	var exists bool
	_ = q.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory WHERE user_id=$1 AND item_id=$2)`, userID, itemID).Scan(&exists)
	if !exists {
		return fmt.Errorf("item not owned")
	}

	var column string
	switch slot {
	case "hull":
		column = "hull_id"
	case "primary_weapon":
		column = "primary_weapon"
	case "secondary_weapon":
		column = "secondary_weapon"
	case "shield":
		column = "shield_id"
	default:
		return fmt.Errorf("invalid slot: %s", slot)
	}

	expectedCategory := map[string]string{
		"hull": "hull", "primary_weapon": "weapon", "secondary_weapon": "weapon", "shield": "shield",
	}[slot]
	var category string
	err := q.Pool.QueryRow(ctx, `SELECT category FROM items WHERE id = $1`, itemID).Scan(&category)
	if err != nil {
		return fmt.Errorf("item not found")
	}
	if category != expectedCategory {
		return fmt.Errorf("item category %s doesn't match slot %s", category, slot)
	}

	// Safe: column is from a hardcoded switch, never user input
	_, err = q.Pool.Exec(ctx,
		fmt.Sprintf(`UPDATE loadouts SET %s = $2, updated_at = now() WHERE user_id = $1`, column),
		userID, itemID)
	return err
}

// GetInventory returns item IDs owned by a player.
func (q *Queries) GetInventory(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := q.Pool.Query(ctx, `SELECT item_id FROM inventory WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, nil
}

// ListShopItems returns all items in the catalog.
func (q *Queries) ListShopItems(ctx context.Context) ([]ShopItem, error) {
	rows, err := q.Pool.Query(ctx, `SELECT id, name, category, price, tier_required, description, stats::text FROM items ORDER BY category, price`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ShopItem
	for rows.Next() {
		var item ShopItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Price, &item.TierRequired, &item.Description, &item.Stats); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// GetLeaderboard returns top players by kills.
func (q *Queries) GetLeaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	rows, err := q.Pool.Query(ctx, `
		SELECT u.username, p.kills, p.deaths, p.ai_tier
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.kills DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Username, &e.Kills, &e.Deaths, &e.AITier); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
