package game

import (
	"github.com/DevMatrix/server/internal/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// executeCombat processes weapon firing for a ship based on its active behavior.
func (e *Engine) executeCombat(ship *Ship, b *BehaviorBlock) {
	// Tick weapon cooldown regardless.
	if ship.PrimaryWeapon.CoolTimer > 0 {
		ship.PrimaryWeapon.CoolTimer -= e.dt
	}

	switch b.Combat {
	case "fire_at":
		e.combatFireAt(ship, b)
	case "burst_fire":
		e.combatBurstFire(ship, b)
	case "fire_at_will":
		e.combatFireAtWill(ship)
	}
}

// combatFireAt fires at the specified target when in range and off cooldown.
func (e *Engine) combatFireAt(ship *Ship, b *BehaviorBlock) {
	target := e.resolveTarget(ship, b.CombatParams.Target)
	if target == nil || !target.IsAlive {
		return
	}
	e.tryFire(ship, target)
}

// combatBurstFire fires 3 rapid shots (0.15 s apart) then pauses for 1.5 s.
func (e *Engine) combatBurstFire(ship *Ship, b *BehaviorBlock) {
	target := e.resolveTarget(ship, b.CombatParams.Target)
	if target == nil || !target.IsAlive {
		return
	}

	// Tick burst pause timer.
	if ship.BurstTimer > 0 {
		ship.BurstTimer -= e.dt
		return
	}

	w := &ship.PrimaryWeapon
	if w.CoolTimer > 0 {
		return
	}

	dist := ship.Position.DistTo(target.Position)
	if dist > w.Range {
		return
	}

	// Fire one shot in the burst.
	w.CoolTimer = 0.15 // rapid fire interval
	if w.Speed == 0 {
		e.processHitscan(ship, target, w)
	} else {
		e.spawnProjectile(ship, target, w)
	}

	ship.BurstCount++
	if ship.BurstCount >= 3 {
		ship.BurstCount = 0
		ship.BurstTimer = 1.5 // pause between bursts
		w.CoolTimer = 0       // burst timer controls the pause, not weapon cooldown
	}
}

// combatFireAtWill fires at whatever enemy is closest and in range,
// switching targets opportunistically each shot.
func (e *Engine) combatFireAtWill(ship *Ship) {
	w := &ship.PrimaryWeapon
	if w.CoolTimer > 0 {
		return
	}

	// Find the closest in-range enemy.
	var best *Ship
	var bestDist float32
	for _, s := range e.state.Ships {
		if s.ID == ship.ID || !s.IsAlive {
			continue
		}
		d := ship.Position.DistTo(s.Position)
		if d > w.Range {
			continue
		}
		if best == nil || d < bestDist {
			best = s
			bestDist = d
		}
	}

	if best == nil {
		return
	}
	e.tryFire(ship, best)
}

// tryFire performs the actual fire sequence against a specific target.
func (e *Engine) tryFire(ship *Ship, target *Ship) {
	w := &ship.PrimaryWeapon
	if w.CoolTimer > 0 {
		return
	}
	dist := ship.Position.DistTo(target.Position)
	if dist > w.Range {
		return
	}
	w.CoolTimer = w.Cooldown
	if w.Speed == 0 {
		e.processHitscan(ship, target, w)
	} else {
		e.spawnProjectile(ship, target, w)
	}
}

// processHitscan performs an instant ray test against compound hit-shape (for lasers).
func (e *Engine) processHitscan(shooter, target *Ship, w *Weapon) {
	dir := target.Position.Sub(shooter.Position).Normalize()

	if w.Spread > 0 {
		dir = applySpread(dir, w.Spread)
	}

	shape := target.HitShape
	if len(shape) == 0 {
		shape = defaultShape
	}
	hit := compoundRayHit(shooter.Position, dir, target.Position, target.Rotation, shape)

	if hit {
		e.applyDamage(target, w.Damage, shooter.ID)
	}

	// Always emit a visual event so clients see the beam.
	endPos := target.Position
	if !hit {
		endPos = shooter.Position.Add(dir.Scale(w.Range))
	}
	e.state.Events = append(e.state.Events, GameEvent{
		Type: EvtLaserFired,
		From: [3]float32{shooter.Position.X, shooter.Position.Y, shooter.Position.Z},
		To:   [3]float32{endPos.X, endPos.Y, endPos.Z},
		Hit:  hit,
	})
}

// spawnProjectile creates a moving projectile entity.
func (e *Engine) spawnProjectile(shooter, target *Ship, w *Weapon) {
	dir := target.Position.Sub(shooter.Position).Normalize()
	spawnPos := shooter.Position.Add(dir.Scale(shooter.CollisionRadius + 1))

	e.state.Projectiles = append(e.state.Projectiles, Projectile{
		ID:       e.state.nextProjectileID(),
		OwnerID:  shooter.ID,
		Position: spawnPos,
		Velocity: dir.Scale(w.Speed),
		Damage:   w.Damage,
		Lifetime: w.Range / w.Speed,
		Radius:   0.5,
	})
}

// updateProjectiles moves all projectiles and checks collisions.
func (e *Engine) updateProjectiles() {
	alive := e.state.Projectiles[:0] // reuse backing array

	for i := range e.state.Projectiles {
		p := &e.state.Projectiles[i]
		p.Lifetime -= e.dt
		if p.Lifetime <= 0 {
			continue
		}

		p.Position = p.Position.Add(p.Velocity.Scale(e.dt))

		// Check collision with nearby ships using compound hit-shapes.
		nearby := e.grid.GetNearby(p.Position, p.Radius+10)
		hit := false
		for _, ship := range nearby {
			if ship.ID == p.OwnerID || !ship.IsAlive {
				continue
			}
			shape := ship.HitShape
			if len(shape) == 0 {
				shape = defaultShape
			}
			if compoundSphereHit(p.Position, p.Radius, ship.Position, ship.Rotation, shape) {
				e.applyDamage(ship, p.Damage, p.OwnerID)
				hit = true
				break
			}
		}

		if !hit {
			alive = append(alive, *p)
		}
	}

	e.state.Projectiles = alive
}

// applyDamage reduces shield then health, and processes kill on death.
func (e *Engine) applyDamage(target *Ship, damage float32, attackerID string) {
	if target.SpawnProtection > 0 {
		return
	}
	target.LastDamagedBy = attackerID
	target.ShieldTimer = target.ShieldDelay // reset regen timer

	// Shield direction multiplier based on defense stance.
	mult := e.shieldMultiplier(target, attackerID)
	shieldAbsorb := damage * mult

	if target.Shield > 0 {
		absorbed := shieldAbsorb
		if absorbed > target.Shield {
			absorbed = target.Shield
		}
		target.Shield -= absorbed
		// Remaining damage (accounting for multiplier) hits hull.
		damage -= absorbed / mult
		if damage <= 0 {
			return
		}
	}

	target.Health -= damage
	if target.Health <= 0 {
		target.Health = 0
		e.processKill(target, attackerID)
	}
}

// shieldMultiplier returns a damage multiplier based on shield stance and attack angle.
func (e *Engine) shieldMultiplier(target *Ship, attackerID string) float32 {
	attacker := e.state.Ships[attackerID]
	if attacker == nil {
		return 1
	}

	// Dot product of target's facing vs direction from attacker to target.
	attackDir := target.Position.Sub(attacker.Position).Normalize()
	facing := quatForward(target.Rotation)
	dot := facing.Dot(attackDir)

	switch target.CurrentDefense {
	case "shield_front":
		if dot > 0 {
			return 1.5 // attacker is in front — more shield absorption
		}
		return 0.5
	case "shield_rear":
		if dot < 0 {
			return 1.5 // attacker is behind — more shield absorption
		}
		return 0.5
	case "shield_omni":
		return 1.2 // good absorption from all directions
	default: // "shield_balanced" or empty
		return 1
	}
}

// quatForward returns the forward (+Z) direction of a quaternion.
func quatForward(q Quaternion) Vec3 {
	// Rotate (0,0,1) by quaternion: q * v * q^-1 simplified.
	return Vec3{
		X: 2 * (q.X*q.Z + q.W*q.Y),
		Y: 2 * (q.Y*q.Z - q.W*q.X),
		Z: 1 - 2*(q.X*q.X+q.Y*q.Y),
	}.Normalize()
}

// processKill marks a ship as dead, emits a kill event,
// and sends async coin/kill records to the DB writer.
func (e *Engine) processKill(victim *Ship, killerID string) {
	victim.IsAlive = false
	victim.Velocity = Vec3{}
	victim.DesiredVelocity = Vec3{}
	victim.RespawnTimer = 5.0

	// Increment killer's streak.
	var streak int
	if killer := e.state.Ships[killerID]; killer != nil {
		killer.KillStreak++
		streak = killer.KillStreak
	}

	e.state.Events = append(e.state.Events, GameEvent{
		Type:   EvtKill,
		Killer: killerID,
		Victim: victim.ID,
		From:   [3]float32{victim.Position.X, victim.Position.Y, victim.Position.Z},
		Streak: streak,
	})

	// Async DB writes (non-blocking).
	if e.dbWriter != nil {
		killerUUID, err1 := uuid.Parse(killerID)
		victimUUID, err2 := uuid.Parse(victim.ID)
		if err1 == nil && err2 == nil {
			select {
			case e.dbWriter.CoinCh <- db.CoinAward{PlayerID: killerUUID, Amount: 50}:
			default:
				log.Warn().Msg("coin award channel full, dropping")
			}
			select {
			case e.dbWriter.KillCh <- db.KillRecord{KillerID: killerUUID, VictimID: victimUUID}:
			default:
				log.Warn().Msg("kill record channel full, dropping")
			}
		}
	}
}

// updateShields regenerates shield after the delay timer expires.
func (e *Engine) updateShields(ship *Ship) {
	if !ship.IsAlive {
		return
	}
	if ship.ShieldTimer > 0 {
		ship.ShieldTimer -= e.dt
		return
	}
	if ship.Shield < ship.MaxShield {
		regen := ship.ShieldRegen
		if ship.CurrentDefense == "shield_omni" {
			regen *= 0.8 // omni shield drains 20% faster
		}
		ship.Shield += regen * e.dt
		if ship.Shield > ship.MaxShield {
			ship.Shield = ship.MaxShield
		}
	}
}

// updateRespawns ticks down respawn timers and revives dead ships.
func (e *Engine) updateRespawns() {
	for _, ship := range e.state.Ships {
		if ship.IsAlive {
			continue
		}
		ship.RespawnTimer -= e.dt
		if ship.RespawnTimer <= 0 {
			ship.Position = randomPosition()
			ship.Velocity = Vec3{}
			ship.DesiredVelocity = Vec3{}
			ship.Health = ship.MaxHealth
			ship.Shield = ship.MaxShield
			ship.IsAlive = true
			ship.ShieldTimer = 0
			ship.PrimaryWeapon.CoolTimer = 0
			ship.SpawnProtection = 3.0 // 3 seconds of invulnerability
			ship.KillStreak = 0
			ship.Behavior = nil // clear behavior — player must re-prompt

			// Reset prompt cooldown so player can immediately issue orders.
			e.cooldown.Remove(ship.ID)

			e.state.Events = append(e.state.Events, GameEvent{
				Type:   EvtRespawn,
				Victim: ship.ID,
				From:   [3]float32{ship.Position.X, ship.Position.Y, ship.Position.Z},
			})
		}
	}
}
