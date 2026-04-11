package game

import (
	"math"
	"math/rand"
)

// ═══════════════════════════════════════════════════════════════════════════
// EVASIVE BEHAVIORS
// ═══════════════════════════════════════════════════════════════════════════

// desiredDodge produces erratic lateral jinking perpendicular to the threat
// direction.  The ship alternates its dodge direction every 0.3–0.8 s while
// drifting toward/away from the target, making it very hard to hit.
func (e *Engine) desiredDodge(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	// Tick dodge timer; pick a new random perpendicular when it expires.
	ship.DodgeTimer -= e.dt
	if ship.DodgeTimer <= 0 {
		ship.DodgeTimer = 0.3 + rand.Float32()*0.5 // 0.3 – 0.8 s

		if target != nil {
			toTarget := target.Position.Sub(ship.Position).Normalize()
			// Build two perpendicular axes.
			up := Vec3{Y: 1}
			if abs32(toTarget.Y) > 0.99 {
				up = Vec3{X: 1}
			}
			perpA := up.Cross(toTarget).Normalize()
			perpB := toTarget.Cross(perpA).Normalize()
			// Random blend of both perpendicular axes.
			a := rand.Float32()*2 - 1
			b := rand.Float32()*2 - 1
			ship.DodgeDir = perpA.Scale(a).Add(perpB.Scale(b)).Normalize()
		} else {
			ship.DodgeDir = randomDirection3D()
		}
	}

	// 70% lateral, 30% forward-drift toward target (or random if no target).
	lateral := ship.DodgeDir.Scale(speed * 0.7)
	var forward Vec3
	if target != nil {
		forward = target.Position.Sub(ship.Position).Normalize().Scale(speed * 0.3)
	}
	ship.DesiredVelocity = lateral.Add(forward)
}

// desiredBarrelRoll traces a helical corkscrew along the ship's current
// heading.  The ship spirals around its forward axis, making it extremely
// hard to hit while still making forward progress.
func (e *Engine) desiredBarrelRoll(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	radius := b.MovementParams.Radius
	if radius == 0 {
		radius = 8
	}

	// Forward direction: current velocity or facing direction.
	forward := ship.Velocity.Normalize()
	if ship.Velocity.Length() < 1 {
		forward = quatForward(ship.Rotation)
	}

	// Advance the spiral angle.  Angular velocity = speed / radius.
	ship.BarrelAngle += (speed / radius) * e.dt

	// Build two perpendicular axes to the forward direction.
	up := Vec3{Y: 1}
	if abs32(forward.Y) > 0.99 {
		up = Vec3{X: 1}
	}
	perpA := up.Cross(forward).Normalize()
	perpB := forward.Cross(perpA).Normalize()

	// Offset on the spiral circle.
	sin := float32(math.Sin(float64(ship.BarrelAngle)))
	cos := float32(math.Cos(float64(ship.BarrelAngle)))
	offset := perpA.Scale(cos * radius).Add(perpB.Scale(sin * radius))

	// DesiredVelocity = forward progress + lateral spiral component.
	// The lateral component magnitude is speed so the ship keeps up.
	ship.DesiredVelocity = forward.Scale(speed * 0.7).Add(offset.Normalize().Scale(speed * 0.7))
}

// desiredJuke performs a hard 90° cut in a random direction, then resumes
// the original heading.  Jukes repeat every 1.5–3.0 s, with the cut lasting
// ~0.3 s.  This makes the ship periodically snap sideways unpredictably.
func (e *Engine) desiredJuke(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	ship.JukeTimer -= e.dt

	if ship.JukeTimer <= 0 {
		if ship.JukePhase == 0 {
			// Start a new juke cut.
			ship.JukePhase = 1
			ship.JukeTimer = 0.3 // cut duration

			// Pick a random perpendicular direction.
			var baseDir Vec3
			if target != nil {
				baseDir = target.Position.Sub(ship.Position).Normalize()
			} else {
				baseDir = ship.Velocity.Normalize()
				if baseDir.Length() < 0.1 {
					baseDir = quatForward(ship.Rotation)
				}
			}
			up := Vec3{Y: 1}
			if abs32(baseDir.Y) > 0.99 {
				up = Vec3{X: 1}
			}
			perpA := up.Cross(baseDir).Normalize()
			perpB := baseDir.Cross(perpA).Normalize()
			if rand.Float32() < 0.5 {
				ship.JukeDir = perpA
			} else {
				ship.JukeDir = perpB
			}
			if rand.Float32() < 0.5 {
				ship.JukeDir = ship.JukeDir.Scale(-1)
			}
		} else {
			// End the cut, start cooldown.
			ship.JukePhase = 0
			ship.JukeTimer = 1.5 + rand.Float32()*1.5 // 1.5 – 3.0 s
		}
	}

	if ship.JukePhase == 1 {
		// During cut: full speed sideways.
		ship.DesiredVelocity = ship.JukeDir.Scale(speed)
	} else {
		// Between cuts: drift toward target or wander.
		if target != nil {
			dir := target.Position.Sub(ship.Position).Normalize()
			ship.DesiredVelocity = dir.Scale(speed * 0.6)
		} else {
			e.desiredWander(ship, b, speed*0.5)
		}
	}
}

// desiredEvade is an intelligent dodge that reacts to actual incoming
// projectiles.  It computes an escape vector perpendicular to each threat's
// velocity and averages them.  Falls back to gentle wandering when safe.
func (e *Engine) desiredEvade(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	const detectRadius float32 = 80

	var escapeSum Vec3
	threats := 0

	for i := range e.state.Projectiles {
		p := &e.state.Projectiles[i]
		if p.OwnerID == ship.ID {
			continue
		}

		// Is this projectile close enough and heading toward us?
		toShip := ship.Position.Sub(p.Position)
		dist := toShip.Length()
		if dist > detectRadius {
			continue
		}

		// Dot > 0 means the projectile is heading roughly toward the ship.
		pDir := p.Velocity.Normalize()
		if pDir.Dot(toShip.Normalize()) < 0.3 {
			continue
		}

		// Escape direction: perpendicular to the projectile velocity.
		up := Vec3{Y: 1}
		if abs32(pDir.Y) > 0.99 {
			up = Vec3{X: 1}
		}
		perp := up.Cross(pDir).Normalize()

		// Weight closer projectiles more heavily.
		weight := 1 - (dist / detectRadius)
		escapeSum = escapeSum.Add(perp.Scale(weight))
		threats++
	}

	if threats > 0 {
		ship.DesiredVelocity = escapeSum.Normalize().Scale(speed)
	} else {
		// No threats — gentle wander.
		e.desiredWander(ship, b, speed*0.4)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TACTICAL BEHAVIORS
// ═══════════════════════════════════════════════════════════════════════════

// desiredIntercept predicts where the target will be based on its velocity
// and moves to cut it off, rather than chasing its current position.
func (e *Engine) desiredIntercept(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	dist := ship.Position.DistTo(target.Position)
	// Time estimate: how long until we reach target at our speed.
	timeToTarget := dist / speed
	if timeToTarget > 3 {
		timeToTarget = 3 // cap prediction horizon
	}

	// Predict future position.
	predicted := target.Position.Add(target.Velocity.Scale(timeToTarget * 0.8))
	dir := predicted.Sub(ship.Position).Normalize()
	ship.DesiredVelocity = dir.Scale(speed)
}

// desiredKite maintains an ideal distance: flee when too close, chase when
// too far, strafe tangentially when at the right range.  Perfect for ranged
// combat — keeps firing while staying safe.
func (e *Engine) desiredKite(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	radius := b.MovementParams.Radius
	if radius == 0 {
		radius = 120
	}

	toTarget := target.Position.Sub(ship.Position)
	dist := toTarget.Length()
	if dist < 1 {
		dist = 1
	}
	norm := toTarget.Scale(1 / dist)

	const deadband float32 = 20

	if dist < radius-deadband {
		// Too close — flee directly away.
		ship.DesiredVelocity = norm.Scale(-speed)
	} else if dist > radius+deadband {
		// Too far — chase toward target.
		ship.DesiredVelocity = norm.Scale(speed)
	} else {
		// In the sweet spot — strafe tangentially.
		up := Vec3{Y: 1}
		if abs32(norm.Y) > 0.99 {
			up = Vec3{X: 1}
		}
		tangent := up.Cross(norm).Normalize()
		ship.DesiredVelocity = tangent.Scale(speed)
	}
}

// desiredFlank approaches the target from the side or behind by first
// moving to a flanking position, then closing in for the attack.
func (e *Engine) desiredFlank(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	// Determine the target's forward direction from its velocity.
	targetFwd := target.Velocity.Normalize()
	if target.Velocity.Length() < 1 {
		targetFwd = quatForward(target.Rotation)
	}

	const flankDist float32 = 80

	// Compute the flank position offset.
	var flankOffset Vec3
	switch b.MovementParams.Direction {
	case "left":
		up := Vec3{Y: 1}
		if abs32(targetFwd.Y) > 0.99 {
			up = Vec3{X: 1}
		}
		flankOffset = up.Cross(targetFwd).Normalize().Scale(flankDist)
	case "right":
		up := Vec3{Y: 1}
		if abs32(targetFwd.Y) > 0.99 {
			up = Vec3{X: 1}
		}
		flankOffset = targetFwd.Cross(up).Normalize().Scale(flankDist)
	default: // "behind"
		flankOffset = targetFwd.Scale(-flankDist)
	}

	flankPos := target.Position.Add(flankOffset)
	distToFlank := ship.Position.DistTo(flankPos)

	if ship.FlankPhase == 0 && distToFlank < 20 {
		ship.FlankPhase = 1 // in position, attack
	} else if ship.FlankPhase == 1 && distToFlank > flankDist*1.5 {
		ship.FlankPhase = 0 // target moved, reposition
	}

	if ship.FlankPhase == 0 {
		// Move to the flanking position.
		dir := flankPos.Sub(ship.Position).Normalize()
		ship.DesiredVelocity = dir.Scale(speed)
	} else {
		// In position — chase directly.
		dir := target.Position.Sub(ship.Position).Normalize()
		ship.DesiredVelocity = dir.Scale(speed)
	}
}

// desiredRam is a full-throttle collision course.  The ship moves directly
// at the target at maximum speed.  Collision damage is handled separately
// in the collision resolution system.
func (e *Engine) desiredRam(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}

	dir := target.Position.Sub(ship.Position).Normalize()
	// Ram uses full MaxSpeed — no clamping from MovementParams.Speed.
	ship.DesiredVelocity = dir.Scale(maxSpeed)
}

// desiredEscort follows a target ship and matches its velocity, maintaining
// formation behind or beside it.
func (e *Engine) desiredEscort(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	followDist := b.MovementParams.Radius
	if followDist == 0 {
		followDist = 30
	}

	// Desired formation position: behind the target.
	targetFwd := target.Velocity.Normalize()
	if target.Velocity.Length() < 1 {
		targetFwd = quatForward(target.Rotation)
	}
	formationPos := target.Position.Add(targetFwd.Scale(-followDist))

	toFormation := formationPos.Sub(ship.Position)
	dist := toFormation.Length()

	if dist < followDist*0.3 {
		// Close enough — match target velocity for clean formation.
		ship.DesiredVelocity = target.Velocity
	} else {
		// Move toward formation position.
		arrivalSpeed := speed
		if dist < speed*1.5 {
			arrivalSpeed = speed * (dist / (speed * 1.5))
		}
		ship.DesiredVelocity = toFormation.Normalize().Scale(arrivalSpeed)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ADVANCED MOVEMENT
// ═══════════════════════════════════════════════════════════════════════════

// desiredZigzag approaches or retreats from a target in a sawtooth pattern,
// alternating ±40° from the direct line every 1.0–1.5 s.
func (e *Engine) desiredZigzag(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	// Base direction: toward or away from target.
	baseDir := target.Position.Sub(ship.Position).Normalize()
	if b.MovementParams.Direction == "away" {
		baseDir = baseDir.Scale(-1)
	}

	// Tick zigzag timer.
	ship.ZigTimer -= e.dt
	if ship.ZigTimer <= 0 {
		ship.ZigLeft = !ship.ZigLeft
		ship.ZigTimer = 1.0 + rand.Float32()*0.5 // 1.0 – 1.5 s
	}

	// Build perpendicular axis.
	up := Vec3{Y: 1}
	if abs32(baseDir.Y) > 0.99 {
		up = Vec3{X: 1}
	}
	perp := up.Cross(baseDir).Normalize()

	// Rotate ±40° from baseDir.
	const angle float32 = 40.0 * math.Pi / 180.0
	sin := float32(math.Sin(float64(angle)))
	cos := float32(math.Cos(float64(angle)))

	var zigDir Vec3
	if ship.ZigLeft {
		zigDir = baseDir.Scale(cos).Add(perp.Scale(sin))
	} else {
		zigDir = baseDir.Scale(cos).Add(perp.Scale(-sin))
	}

	ship.DesiredVelocity = zigDir.Normalize().Scale(speed)
}

// desiredAnchor actively holds the ship at a fixed position.  Unlike "idle"
// which lets the ship coast, anchor thrusts against drift to maintain station.
func (e *Engine) desiredAnchor(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	// Capture anchor position on first tick of this behavior.
	if !ship.AnchorSet {
		if b.MovementParams.Position != [3]float32{} {
			ship.AnchorPos = Vec3{
				X: b.MovementParams.Position[0],
				Y: b.MovementParams.Position[1],
				Z: b.MovementParams.Position[2],
			}
		} else {
			ship.AnchorPos = ship.Position
		}
		ship.AnchorSet = true
	}

	// P-controller: thrust proportional to displacement.
	disp := ship.AnchorPos.Sub(ship.Position)
	desired := disp.Scale(5.0) // gain = 5

	// Clamp to half max speed to avoid overshooting.
	limit := maxSpeed * 0.5
	if desired.Length() > limit {
		desired = desired.Normalize().Scale(limit)
	}

	ship.DesiredVelocity = desired
}
