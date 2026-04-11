package game

import (
	"math"
	"math/rand"
)

// Vec3 represents a 3D vector.
type Vec3 struct {
	X, Y, Z float32
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(f float32) Vec3 { return Vec3{v.X * f, v.Y * f, v.Z * f} }

func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (v Vec3) LengthXZ() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Z*v.Z)))
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l < 1e-6 {
		return Vec3{}
	}
	return v.Scale(1 / l)
}

func (v Vec3) DistTo(o Vec3) float32 { return v.Sub(o).Length() }

func (v Vec3) Dot(o Vec3) float32 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// Cross returns the cross product of v and o.
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

// LerpTo returns the vector linearly interpolated toward o by t ∈ [0,1].
func (v Vec3) LerpTo(o Vec3, t float32) Vec3 {
	return Vec3{
		v.X + (o.X-v.X)*t,
		v.Y + (o.Y-v.Y)*t,
		v.Z + (o.Z-v.Z)*t,
	}
}

// Quaternion represents a rotation.
type Quaternion struct {
	X, Y, Z, W float32
}

// Weapon represents a ship-mounted weapon.
type Weapon struct {
	Type      string  // "laser", "plasma"
	Damage    float32
	Cooldown  float32 // seconds between shots
	CoolTimer float32 // current countdown
	Range     float32 // max effective range
	Speed     float32 // projectile speed; 0 = hitscan
	Spread    float32 // accuracy cone in degrees
}

// Projectile is a moving damage entity (e.g. plasma bolt).
type Projectile struct {
	ID       uint64
	OwnerID  string
	Position Vec3
	Velocity Vec3
	Damage   float32
	Lifetime float32 // seconds remaining
	Radius   float32 // collision sphere
}

// Ship is the server-authoritative representation of a player's ship.
type Ship struct {
	ID       string
	Username string
	Position Vec3
	Velocity Vec3
	Rotation Quaternion
	Color    [3]float32

	// Behavior system (Phase 3).
	Behavior       *BehaviorSet
	CurrentDefense string // active defense mode resolved each tick
	MaxSpeed       float32

	// Physics tuning.
	Thrust   float32 // acceleration, units/s²
	Drag     float32 // linear damping coefficient, 1/s
	TurnRate float32 // rotation speed, rad/s

	// Per-behavior transient state.
	DesiredVelocity Vec3
	WanderDir       Vec3
	WanderTimer     float32
	PatrolIndex     int

	// Combat.
	Health          float32
	MaxHealth       float32
	Shield          float32
	MaxShield       float32
	ShieldRegen     float32 // per second
	ShieldDelay     float32 // seconds after hit before regen
	ShieldTimer     float32 // countdown to regen
	PrimaryWeapon   Weapon
	IsAlive         bool
	RespawnTimer    float32 // countdown when dead
	CollisionRadius float32
	LastDamagedBy   string  // for kill credit
	AITier          int     // LLM behavior tier (1-5)
	HullID          string  // item ID for model selection
}

// HealthPct returns current health as a percentage 0–100.
func (s *Ship) HealthPct() float32 {
	if s.MaxHealth == 0 {
		return 100
	}
	return s.Health / s.MaxHealth * 100
}

// ShieldPct returns current shield as a percentage 0–100.
func (s *Ship) ShieldPct() float32 {
	if s.MaxShield == 0 {
		return 100
	}
	return s.Shield / s.MaxShield * 100
}

// GameState holds the entire mutable world state for one game loop.
// Only accessed from the engine goroutine — no locks needed.
type GameState struct {
	Ships       map[string]*Ship
	Projectiles []Projectile
	Events      []GameEvent
	nextProjID  uint64
}

// GameEvent is a one-shot event generated during a tick (laser hit, kill, etc.).
// Broadcast to all clients alongside the state update, then discarded.
type GameEvent struct {
	Type   uint8
	From   [3]float32 // source position
	To     [3]float32 // impact/end position
	Hit    bool       // whether it connected
	Killer string     // for kill events
	Victim string
}

// Event type constants.
const (
	EvtLaserFired uint8 = 1
	EvtKill       uint8 = 2
	EvtRespawn    uint8 = 3
)

// NewGameState returns an empty game state.
func NewGameState() *GameState {
	return &GameState{
		Ships: make(map[string]*Ship),
	}
}

// nextProjectileID returns a monotonically increasing projectile ID.
func (gs *GameState) nextProjectileID() uint64 {
	gs.nextProjID++
	return gs.nextProjID
}

// --- Ship color palette (visually distinct, bright colors) ---

var shipColors = [][3]float32{
	{0.0, 1.0, 0.53},  // green
	{0.0, 0.75, 1.0},  // cyan
	{1.0, 0.3, 0.3},   // red
	{1.0, 0.85, 0.0},  // yellow
	{0.7, 0.3, 1.0},   // purple
	{1.0, 0.5, 0.0},   // orange
	{0.3, 0.5, 1.0},   // blue
	{1.0, 0.4, 0.7},   // pink
}

var colorIndex int

func nextColor() [3]float32 {
	c := shipColors[colorIndex%len(shipColors)]
	colorIndex++
	return c
}

// --- Random helpers ---

func randomPosition() Vec3 {
	return Vec3{
		X: (rand.Float32() - 0.5) * 200, // ±100
		Y: (rand.Float32() - 0.5) * 200, // ±100
		Z: (rand.Float32() - 0.5) * 200,
	}
}

// boundaryRepulsion returns an acceleration that pushes inward when pos is
// within <margin> units of the [min, max] boundary.  Uses quadratic ramp for
// a smooth, physically plausible deceleration curve.
func boundaryRepulsion(pos, min, max, margin, strength float32) float32 {
	if pos > max-margin {
		t := (pos - (max - margin)) / margin
		if t > 1 {
			t = 1
		}
		return -strength * t * t
	}
	if pos < min+margin {
		t := ((min + margin) - pos) / margin
		if t > 1 {
			t = 1
		}
		return strength * t * t
	}
	return 0
}

// quatLookDir returns a quaternion facing the given 3D direction (yaw + pitch).
func quatLookDir(dir Vec3) Quaternion {
	l := dir.Length()
	if l < 0.001 {
		return Quaternion{W: 1}
	}
	d := dir.Scale(1 / l)

	yaw := float32(math.Atan2(float64(d.X), float64(d.Z)))
	horiz := float32(math.Sqrt(float64(d.X*d.X + d.Z*d.Z)))
	pitch := float32(-math.Atan2(float64(d.Y), float64(horiz)))

	sy := float32(math.Sin(float64(yaw) * 0.5))
	cy := float32(math.Cos(float64(yaw) * 0.5))
	sp := float32(math.Sin(float64(pitch) * 0.5))
	cp := float32(math.Cos(float64(pitch) * 0.5))

	// Qyaw * Qpitch (Y rotation then X rotation).
	return Quaternion{
		X: cy*sp,
		Y: sy*cp,
		Z: -sy * sp,
		W: cy * cp,
	}
}

// quatDot returns the dot product of two quaternions.
func quatDot(a, b Quaternion) float32 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}

// quatSlerp interpolates between two quaternions by t ∈ [0,1].
// Uses the short-path variant (negates b if dot < 0).
func quatSlerp(a, b Quaternion, t float32) Quaternion {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}

	dot := quatDot(a, b)
	// Take the short path.
	if dot < 0 {
		b = Quaternion{-b.X, -b.Y, -b.Z, -b.W}
		dot = -dot
	}

	// If very close, linear interpolate to avoid division by zero.
	if dot > 0.9995 {
		return Quaternion{
			X: a.X + (b.X-a.X)*t,
			Y: a.Y + (b.Y-a.Y)*t,
			Z: a.Z + (b.Z-a.Z)*t,
			W: a.W + (b.W-a.W)*t,
		}.normalize()
	}

	theta := float32(math.Acos(float64(dot)))
	sin := float32(math.Sin(float64(theta)))
	wa := float32(math.Sin(float64((1 - t) * theta))) / sin
	wb := float32(math.Sin(float64(t*theta))) / sin

	return Quaternion{
		X: wa*a.X + wb*b.X,
		Y: wa*a.Y + wb*b.Y,
		Z: wa*a.Z + wb*b.Z,
		W: wa*a.W + wb*b.W,
	}
}

func (q Quaternion) normalize() Quaternion {
	l := float32(math.Sqrt(float64(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)))
	if l < 1e-6 {
		return Quaternion{W: 1}
	}
	inv := 1 / l
	return Quaternion{q.X * inv, q.Y * inv, q.Z * inv, q.W * inv}
}

// --- Collision math ---

// spheresOverlap returns true if two bounding spheres intersect.
func spheresOverlap(p1 Vec3, r1 float32, p2 Vec3, r2 float32) bool {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	dz := p1.Z - p2.Z
	distSq := dx*dx + dy*dy + dz*dz
	radSum := r1 + r2
	return distSq <= radSum*radSum
}

// rayHitsSphere tests if a ray from origin in direction dir intersects a sphere.
func rayHitsSphere(origin, dir, center Vec3, radius float32) bool {
	oc := origin.Sub(center)
	a := dir.Dot(dir)
	b := 2.0 * oc.Dot(dir)
	c := oc.Dot(oc) - radius*radius
	return b*b-4*a*c >= 0
}

// applySpread randomly perturbs a direction vector within a cone of spreadDeg degrees.
func applySpread(dir Vec3, spreadDeg float32) Vec3 {
	spreadRad := spreadDeg * math.Pi / 180
	theta := rand.Float32() * 2 * math.Pi
	phi := rand.Float32() * float32(spreadRad)

	up := Vec3{Y: 1}
	if abs32(dir.Y) > 0.99 {
		up = Vec3{X: 1}
	}
	right := dir.Cross(up).Normalize()
	up = right.Cross(dir).Normalize()

	sinPhi := float32(math.Sin(float64(phi)))
	offset := right.Scale(sinPhi * float32(math.Cos(float64(theta)))).
		Add(up.Scale(sinPhi * float32(math.Sin(float64(theta)))))

	return dir.Add(offset).Normalize()
}

// StarterLaser is the default weapon for new ships.
var StarterLaser = Weapon{
	Type:     "laser",
	Damage:   8,
	Cooldown: 0.5,
	Range:    200,
	Speed:    0, // hitscan
	Spread:   2,
}
