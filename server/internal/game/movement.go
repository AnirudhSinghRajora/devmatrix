package game

import "math/rand"

// executeBehavior evaluates the ship's active behavior and computes a desired
// velocity.  The actual physics (steering, drag, boundaries, rotation) are
// applied in the engine's updateEntities pipeline afterwards.
func (e *Engine) executeBehavior(ship *Ship) {
	if ship.Behavior == nil {
		// No orders — desired velocity is zero (ship will coast to stop via drag).
		ship.DesiredVelocity = Vec3{}
		return
	}

	// Evaluate conditionals in order; first match overrides primary.
	active := &ship.Behavior.Primary
	ctx := e.buildShipContext(ship)
	for i := range ship.Behavior.Conditionals {
		c := &ship.Behavior.Conditionals[i]
		if c.Condition != nil && c.Condition.Evaluate(ctx) {
			active = &c.BehaviorBlock
			break
		}
	}

	// Set defense mode from the active behavior block.
	ship.CurrentDefense = active.Defense

	e.computeDesiredVelocity(ship, active)

	// Combat: weapon cooldowns, firing, target resolution.
	e.executeCombat(ship, active)
}

// buildShipContext gathers runtime values for condition evaluation.
func (e *Engine) buildShipContext(ship *Ship) *ShipContext {
	ctx := &ShipContext{
		HealthPct:  float64(ship.HealthPct()),
		ShieldPct:  float64(ship.ShieldPct()),
		EnemyCount: len(e.state.Ships) - 1,
	}

	// Resolve the primary target to fill target-related fields.
	if ship.Behavior != nil {
		target := e.resolveTarget(ship, ship.Behavior.Primary.MovementParams.Target)
		if target != nil {
			ctx.TargetDist = float64(ship.Position.DistTo(target.Position))
			ctx.TargetHPPct = float64(target.HealthPct())
		}
	}
	return ctx
}

// computeDesiredVelocity sets ship.DesiredVelocity based on the movement primitive.
// It does NOT touch ship.Velocity or ship.Rotation — that's the physics layer's job.
func (e *Engine) computeDesiredVelocity(ship *Ship, b *BehaviorBlock) {
	maxSpeed := ship.MaxSpeed
	if maxSpeed == 0 {
		maxSpeed = 50
	}

	switch b.Movement {
	case "idle":
		ship.DesiredVelocity = Vec3{}
	case "chase":
		e.desiredChase(ship, b, maxSpeed)
	case "flee":
		e.desiredFlee(ship, b, maxSpeed)
	case "orbit":
		e.desiredOrbit(ship, b, maxSpeed)
	case "wander":
		e.desiredWander(ship, b, maxSpeed)
	case "patrol":
		e.desiredPatrol(ship, b, maxSpeed)
	case "strafe":
		e.desiredStrafe(ship, b, maxSpeed)
	case "move_to":
		e.desiredMoveTo(ship, b, maxSpeed)
	default:
		ship.DesiredVelocity = Vec3{}
	}
}

// --- Desired-velocity primitives ---

func (e *Engine) desiredChase(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	dir := target.Position.Sub(ship.Position).Normalize()
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	ship.DesiredVelocity = dir.Scale(speed)
}

func (e *Engine) desiredFlee(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}
	dir := ship.Position.Sub(target.Position).Normalize()
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	ship.DesiredVelocity = dir.Scale(speed)
}

func (e *Engine) desiredOrbit(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}

	radius := b.MovementParams.Radius
	if radius == 0 {
		radius = 150
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	toTarget := target.Position.Sub(ship.Position)
	dist := toTarget.Length()

	if dist < 1 {
		ship.DesiredVelocity = Vec3{X: speed}
		return
	}

	norm := toTarget.Normalize()

	// Compute 3D tangent for orbiting.
	up := Vec3{Y: 1}
	if abs32(norm.Y) > 0.99 {
		up = Vec3{X: 1}
	}
	tangent := up.Cross(norm).Normalize()

	// Blend radial correction with tangential orbit.
	const deadband = 10.0
	var radial Vec3
	if dist < radius-deadband {
		// Too close — push outward.
		radial = norm.Scale(-speed * 0.5)
	} else if dist > radius+deadband {
		// Too far — pull inward.
		radial = norm.Scale(speed * 0.5)
	}

	ship.DesiredVelocity = tangent.Scale(speed).Add(radial)
}

func (e *Engine) desiredWander(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	ship.WanderTimer -= e.dt
	if ship.WanderTimer <= 0 {
		ship.WanderDir = randomDirection3D()
		ship.WanderTimer = 2.0 + rand.Float32()*3.0
	}
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed*0.5)
	ship.DesiredVelocity = ship.WanderDir.Scale(speed)
}

func (e *Engine) desiredPatrol(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	wps := b.MovementParams.Waypoints
	if len(wps) == 0 {
		ship.DesiredVelocity = Vec3{}
		return
	}
	wp := wps[ship.PatrolIndex%len(wps)]
	target := Vec3{X: wp[0], Y: wp[1], Z: wp[2]}
	dir := target.Sub(ship.Position)
	dist := dir.Length()

	if dist < 10 {
		ship.PatrolIndex = (ship.PatrolIndex + 1) % len(wps)
	}

	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	if dist > 0.1 {
		ship.DesiredVelocity = dir.Normalize().Scale(speed)
	} else {
		ship.DesiredVelocity = Vec3{}
	}
}

func (e *Engine) desiredStrafe(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	target := e.resolveTarget(ship, b.MovementParams.Target)
	if target == nil {
		e.desiredWander(ship, b, maxSpeed*0.3)
		return
	}

	toTarget := target.Position.Sub(ship.Position)
	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)

	norm := toTarget.Normalize()
	up := Vec3{Y: 1}
	if abs32(norm.Y) > 0.99 {
		up = Vec3{X: 1}
	}
	var tangent Vec3
	if b.MovementParams.Direction == "left" {
		tangent = up.Cross(norm).Normalize()
	} else {
		tangent = norm.Cross(up).Normalize()
	}
	ship.DesiredVelocity = tangent.Scale(speed)
}

func (e *Engine) desiredMoveTo(ship *Ship, b *BehaviorBlock, maxSpeed float32) {
	dest := Vec3{X: b.MovementParams.Position[0], Y: b.MovementParams.Position[1], Z: b.MovementParams.Position[2]}
	dir := dest.Sub(ship.Position)
	dist := dir.Length()

	if dist < 5 {
		ship.DesiredVelocity = Vec3{}
		return
	}

	speed := clampF32(b.MovementParams.Speed, 1, maxSpeed)
	// Slow down when approaching the destination for a smooth arrival.
	arrivalDist := speed * 1.5 // begin decelerating within ~1.5s of travel
	if dist < arrivalDist {
		speed *= dist / arrivalDist
	}
	ship.DesiredVelocity = dir.Normalize().Scale(speed)
}

// --- Physics layer ---
// These are called by engine.updateEntities() after executeBehavior().

// applyThrust steers ship.Velocity toward ship.DesiredVelocity using the
// ship's thrust as the maximum acceleration per second.
func applyThrust(ship *Ship, dt float32) {
	steer := ship.DesiredVelocity.Sub(ship.Velocity)
	steerLen := steer.Length()
	if steerLen < 0.01 {
		return
	}

	maxAccel := ship.Thrust * dt
	if steerLen > maxAccel {
		steer = steer.Scale(maxAccel / steerLen)
	}
	ship.Velocity = ship.Velocity.Add(steer)
}

// applyDrag applies linear velocity damping.
func applyDrag(ship *Ship, dt float32) {
	factor := 1 - ship.Drag*dt
	if factor < 0 {
		factor = 0
	}
	ship.Velocity = ship.Velocity.Scale(factor)
}

// applyBoundaryForces pushes ships away from world edges with a smooth
// quadratic ramp.  Called per-axis in updateEntities.
const (
	worldMin       float32 = -500
	worldMax       float32 = 500
	boundaryMargin float32 = 100
	boundaryForce  float32 = 80
)

func applyBoundaryForces(ship *Ship, dt float32) {
	ship.Velocity.X += boundaryRepulsion(ship.Position.X, worldMin, worldMax, boundaryMargin, boundaryForce) * dt
	ship.Velocity.Y += boundaryRepulsion(ship.Position.Y, worldMin, worldMax, boundaryMargin, boundaryForce) * dt
	ship.Velocity.Z += boundaryRepulsion(ship.Position.Z, worldMin, worldMax, boundaryMargin, boundaryForce) * dt
}

// clampSpeed enforces the hard speed cap.
func clampSpeed(ship *Ship) {
	speed := ship.Velocity.Length()
	if speed > ship.MaxSpeed {
		ship.Velocity = ship.Velocity.Scale(ship.MaxSpeed / speed)
	}
}

// applyRotation smoothly slerps the ship's quaternion toward the velocity
// direction at TurnRate rad/s.
func applyRotation(ship *Ship, dt float32) {
	speed := ship.Velocity.Length()
	if speed < 0.5 {
		return // don't rotate when nearly stationary
	}
	target := quatLookDir(ship.Velocity)
	t := ship.TurnRate * dt
	if t > 1 {
		t = 1
	}
	ship.Rotation = quatSlerp(ship.Rotation, target, t)
}

// integratePosition applies velocity to position (Euler integration).
func integratePosition(ship *Ship, dt float32) {
	ship.Position.X += ship.Velocity.X * dt
	ship.Position.Y += ship.Velocity.Y * dt
	ship.Position.Z += ship.Velocity.Z * dt
}

// --- Helpers ---

// randomDirection3D returns a unit vector in a random 3D direction.
func randomDirection3D() Vec3 {
	return Vec3{
		X: rand.Float32()*2 - 1,
		Y: rand.Float32()*2 - 1,
		Z: rand.Float32()*2 - 1,
	}.Normalize()
}

// abs32 returns the absolute value of a float32.
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
