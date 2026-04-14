package game

// resolveShipCollisions detects and resolves collisions between all living
// ships using compound hit-shapes for narrow phase and the spatial grid for
// broad phase.  Impulse-based response with positional correction and
// collision damage proportional to impact speed.
func (e *Engine) resolveShipCollisions() {
	const (
		restitution    float32 = 0.5  // bounciness (0 = perfectly inelastic, 1 = elastic)
		corrPct        float32 = 0.6  // positional correction strength
		slop           float32 = 0.05 // penetration allowed before correction kicks in
		collisionDmg   float32 = 0.4  // HP per unit of relative impact speed
		minImpactSpeed float32 = 10.0 // ignore gentle bumps below this threshold
	)

	// Collect ships into a slice for pair-wise iteration.
	ships := make([]*Ship, 0, len(e.state.Ships))
	for _, s := range e.state.Ships {
		if s.IsAlive {
			ships = append(ships, s)
		}
	}

	n := len(ships)
	for i := 0; i < n; i++ {
		a := ships[i]
		// Use spatial grid to narrow candidates instead of O(n²).
		nearby := e.grid.GetNearby(a.Position, a.CollisionRadius+10)
		for _, b := range nearby {
			if b.ID <= a.ID {
				continue // avoid duplicate pairs and self
			}
			if !b.IsAlive || !a.IsAlive {
				continue
			}

			// Broad phase: bounding-sphere quick reject.
			diff := a.Position.Sub(b.Position)
			distSq := diff.Dot(diff)
			radSum := a.CollisionRadius + b.CollisionRadius
			if distSq >= radSum*radSum {
				continue
			}

			// Narrow phase: compound hit-shape overlap.
			shapeA := a.HitShape
			if len(shapeA) == 0 {
				shapeA = defaultShape
			}
			shapeB := b.HitShape
			if len(shapeB) == 0 {
				shapeB = defaultShape
			}

			normal, penetration, hit := compoundOverlap(
				a.Position, a.Rotation, shapeA,
				b.Position, b.Rotation, shapeB,
			)
			if !hit {
				continue
			}

			// Relative velocity (A relative to B).
			relVel := a.Velocity.Sub(b.Velocity)
			velAlongNormal := relVel.Dot(normal)

			invMassA := 1.0 / a.Mass
			invMassB := 1.0 / b.Mass

			// Only resolve if ships are moving toward each other.
			if velAlongNormal <= 0 {
				impactSpeed := -velAlongNormal

				// Collision damage: both ships take damage proportional to
				// the approach speed.  Kill credit goes to the other ship.
				if impactSpeed > minImpactSpeed {
					damage := (impactSpeed - minImpactSpeed) * collisionDmg
					e.applyDamage(a, damage, b.ID)
					if b.IsAlive {
						e.applyDamage(b, damage, a.ID)
					}
				}

				// Impulse magnitude: j = -(1+e)(V_rel · n) / (1/m_A + 1/m_B)
				j := -(1 + restitution) * velAlongNormal / (invMassA + invMassB)

				// Apply impulse.
				impulse := normal.Scale(j)
				a.Velocity = a.Velocity.Add(impulse.Scale(invMassA))
				b.Velocity = b.Velocity.Sub(impulse.Scale(invMassB))
			}

			// Positional correction to resolve overlap (linear projection).
			if penetration > slop {
				corr := normal.Scale((penetration - slop) / (invMassA + invMassB) * corrPct)
				a.Position = a.Position.Add(corr.Scale(invMassA))
				b.Position = b.Position.Sub(corr.Scale(invMassB))
			}
		}
	}
}
