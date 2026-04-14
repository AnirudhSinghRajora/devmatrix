package game

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/DevMatrix/server/internal/db"
	"github.com/DevMatrix/server/internal/network"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// PromptCooldown is implemented by llm.CooldownTracker.
// Defined here to avoid an import cycle (game ↔ llm).
type PromptCooldown interface {
	CanSubmit(playerID string) (bool, time.Duration)
	Record(playerID string)
	Remove(playerID string)
}

// Engine runs the authoritative game loop on a single goroutine.
// External inputs arrive via channels and are drained at the start of each tick.
type Engine struct {
	tickRate int
	dt       float32
	tick     uint64
	state    *GameState
	hub      *network.Hub
	grid     *SpatialGrid

	joinCh   <-chan network.JoinRequest
	leaveCh  <-chan string
	promptCh <-chan network.PromptRequest

	// LLM pipeline.
	llmReqCh    chan<- LLMRequest
	llmResultCh <-chan LLMResult
	cooldown    PromptCooldown

	// Persistence (nil = anonymous mode).
	itemCache *db.ItemCache
	dbWriter  *db.DBWriter
	queries   *db.Queries

	// Monitoring (atomic for thread-safe reads from HTTP handler).
	lastTickNanos  atomic.Int64
	playerCount    atomic.Int32
}

// EngineStats holds a snapshot of engine metrics for the monitoring endpoint.
type EngineStats struct {
	Tick        uint64  `json:"tick"`
	TickRate    int     `json:"tick_rate"`
	LastTickMs  float64 `json:"last_tick_ms"`
	PlayerCount int     `json:"player_count"`
	Clients     int     `json:"clients"`
}

// Stats returns a thread-safe snapshot of engine metrics.
func (e *Engine) Stats() EngineStats {
	return EngineStats{
		Tick:        atomic.LoadUint64(&e.tick),
		TickRate:    e.tickRate,
		LastTickMs:  float64(e.lastTickNanos.Load()) / 1e6,
		PlayerCount: int(e.playerCount.Load()),
		Clients:     e.hub.ClientCount(),
	}
}

// NewEngine creates a game engine wired to the hub, input channels, and LLM pipeline.
func NewEngine(
	tickRate int,
	hub *network.Hub,
	joinCh <-chan network.JoinRequest,
	leaveCh <-chan string,
	promptCh <-chan network.PromptRequest,
	llmReqCh chan<- LLMRequest,
	llmResultCh <-chan LLMResult,
	cooldown PromptCooldown,
) *Engine {
	return &Engine{
		tickRate:    tickRate,
		dt:          1.0 / float32(tickRate),
		state:       NewGameState(),
		hub:         hub,
		grid:        NewSpatialGrid(),
		joinCh:      joinCh,
		leaveCh:     leaveCh,
		promptCh:    promptCh,
		llmReqCh:    llmReqCh,
		llmResultCh: llmResultCh,
		cooldown:    cooldown,
	}
}

// SetDB enables persistence mode with the item cache, async writer, and queries.
func (e *Engine) SetDB(cache *db.ItemCache, writer *db.DBWriter, queries *db.Queries) {
	e.itemCache = cache
	e.dbWriter = writer
	e.queries = queries
}

// Run starts the game loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	interval := time.Second / time.Duration(e.tickRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Int("tickRate", e.tickRate).Msg("game engine started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Uint64("finalTick", e.tick).Msg("game engine stopped")
			return
		case <-ticker.C:
			start := time.Now()
			e.tick++
			e.processInputs()
			e.updateEntities()
			e.buildAndBroadcast()

			elapsed := time.Since(start)
			e.lastTickNanos.Store(elapsed.Nanoseconds())
			e.playerCount.Store(int32(len(e.state.Ships)))

			if elapsed > interval {
				log.Warn().
					Dur("elapsed", elapsed).
					Dur("budget", interval).
					Uint64("tick", e.tick).
					Msg("tick exceeded budget")
			}
		}
	}
}

// processInputs drains join/leave/prompt/result channels without blocking.
func (e *Engine) processInputs() {
	e.drainJoins()
	e.drainLeaves()
	e.drainPrompts()
	e.drainLLMResults()
}

func (e *Engine) drainJoins() {
	for {
		select {
		case req := <-e.joinCh:
			ship := e.buildShipForPlayer(req)
			e.state.Ships[req.PlayerID] = ship
			e.sendWelcome(req.Client, req.PlayerID)
			log.Info().
				Str("player", req.PlayerID).
				Str("username", req.Username).
				Int("totalShips", len(e.state.Ships)).
				Msg("ship spawned")
		default:
			return
		}
	}
}

// buildShipForPlayer creates a Ship from the DB loadout if available,
// otherwise uses default stats for guest / anonymous mode.
func (e *Engine) buildShipForPlayer(req network.JoinRequest) *Ship {
	ship := &Ship{
		ID:       req.PlayerID,
		Username: req.Username,
		Position: randomPosition(),
		Velocity: Vec3{},
		Rotation: Quaternion{W: 1},
		Color:    nextColor(),
		IsAlive:  true,
		Behavior: &BehaviorSet{
			Primary: BehaviorBlock{
				Movement:       "wander",
				MovementParams: MovementParams{Speed: 15},
			},
		},
	}

	if e.itemCache != nil && e.queries != nil {
		// Try authenticated mode: load full profile from DB.
		userID, err := uuid.Parse(req.PlayerID)
		if err == nil {
			profile, err := e.queries.GetProfile(context.Background(), userID)
			if err == nil {
				hull := e.itemCache.GetHull(profile.HullID)
				wpn := e.itemCache.GetWeapon(profile.PrimaryWeaponID)
				shld := e.itemCache.GetShield(profile.ShieldID)

				ship.MaxHealth = hull.MaxHealth
				ship.Health = hull.MaxHealth
				ship.MaxSpeed = hull.MaxSpeed
				ship.Thrust = hull.Thrust
				ship.CollisionRadius = hull.CollisionRadius
				ship.Mass = hull.MaxHealth * 0.1
				ship.Drag = 0.3
				ship.TurnRate = 3.0
				ship.MaxShield = shld.MaxShield
				ship.Shield = shld.MaxShield
				ship.ShieldRegen = shld.Regen
				ship.ShieldDelay = shld.Delay
				ship.PrimaryWeapon = Weapon{
					Type: wpn.Type, Damage: wpn.Damage, Cooldown: wpn.Cooldown,
					Range: wpn.Range, Speed: wpn.Speed, Spread: wpn.Spread,
				}
				ship.AITier = profile.AITier
				ship.HullID = profile.HullID
				ship.HitShape = hullShapeFor(profile.HullID)
				ship.CollisionRadius = boundingRadius(ship.HitShape)
				return ship
			}
			log.Warn().Err(err).Str("player", req.PlayerID).Msg("failed to load profile, using defaults")
		}
	}

	// Guest / anonymous: use the requested hull from the lobby (if valid),
	// with default weapon and shield. Falls back to hull_basic.
	hullID := req.HullID
	if hullID == "" {
		hullID = "hull_basic"
	}

	if e.itemCache != nil {
		hull := e.itemCache.GetHull(hullID)
		shld := e.itemCache.GetShield("shield_basic")

		ship.MaxHealth = hull.MaxHealth
		ship.Health = hull.MaxHealth
		ship.MaxSpeed = hull.MaxSpeed
		ship.Thrust = hull.Thrust
		ship.Mass = hull.MaxHealth * 0.1
		ship.Drag = 0.3
		ship.TurnRate = 3.0
		ship.MaxShield = shld.MaxShield
		ship.Shield = shld.MaxShield
		ship.ShieldRegen = shld.Regen
		ship.ShieldDelay = shld.Delay
		ship.PrimaryWeapon = StarterLaser
		ship.AITier = 1
		ship.HullID = hullID
		ship.HitShape = hullShapeFor(hullID)
		ship.CollisionRadius = boundingRadius(ship.HitShape)
		return ship
	}

	// Pure anonymous (no DB at all): hardcoded defaults.
	ship.MaxSpeed = 50
	ship.Thrust = 40
	ship.Mass = 10
	ship.Drag = 0.3
	ship.TurnRate = 3.0
	ship.Health = 100
	ship.MaxHealth = 100
	ship.Shield = 50
	ship.MaxShield = 50
	ship.ShieldRegen = 5
	ship.ShieldDelay = 3
	ship.PrimaryWeapon = StarterLaser
	ship.CollisionRadius = 2.0
	ship.AITier = 1
	ship.HullID = hullID
	ship.HitShape = hullShapeFor(hullID)
	ship.CollisionRadius = boundingRadius(ship.HitShape)
	return ship
}

func (e *Engine) drainLeaves() {
	for {
		select {
		case id := <-e.leaveCh:
			delete(e.state.Ships, id)
			e.cooldown.Remove(id)
			log.Info().
				Str("player", id).
				Int("totalShips", len(e.state.Ships)).
				Msg("ship removed")
		default:
			return
		}
	}
}

func (e *Engine) drainPrompts() {
	for {
		select {
		case req := <-e.promptCh:
			e.handlePrompt(req)
		default:
			return
		}
	}
}

func (e *Engine) drainLLMResults() {
	for {
		select {
		case result := <-e.llmResultCh:
			ship := e.state.Ships[result.PlayerID]
			if ship == nil {
				continue // player left while prompt was processing
			}
			if result.Error != nil {
				e.sendPlayerError(result.PlayerID, result.Error.Error(), 0)
				log.Warn().
					Str("player", result.PlayerID).
					Err(result.Error).
					Msg("LLM processing failed")
				continue
			}
			ship.Behavior = result.Behavior
			// Reset per-behavior transient state.
			ship.WanderTimer = 0
			ship.PatrolIndex = 0
			ship.DodgeTimer = 0
			ship.DodgeDir = Vec3{}
			ship.BarrelAngle = 0
			ship.JukeTimer = 0
			ship.JukePhase = 0
			ship.JukeDir = Vec3{}
			ship.ZigTimer = 0
			ship.ZigLeft = false
			ship.FlankPhase = 0
			ship.AnchorSet = false
			ship.BurstCount = 0
			ship.BurstTimer = 0
			e.sendBehaviorConfirmation(result.PlayerID, result.Behavior)
			log.Info().
				Str("player", result.PlayerID).
				Str("movement", result.Behavior.Primary.Movement).
				Msg("behavior applied")
		default:
			return
		}
	}
}

// handlePrompt validates a prompt request and queues it for LLM processing.
func (e *Engine) handlePrompt(req network.PromptRequest) {
	ship := e.state.Ships[req.PlayerID]
	if ship == nil {
		return
	}

	// Enforce cooldown.
	if ok, remaining := e.cooldown.CanSubmit(req.PlayerID); !ok {
		e.sendPlayerError(req.PlayerID, "Cooldown active", float32(remaining.Seconds()))
		return
	}

	// Enforce max prompt length (Tier 1 default: 200 chars).
	const maxLen = 200
	text := req.Text
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	// Record cooldown.
	e.cooldown.Record(req.PlayerID)

	// Gather nearby enemy info for the LLM prompt.
	var enemies []EnemySnapshot
	for _, s := range e.state.Ships {
		if s.ID == ship.ID || !s.IsAlive {
			continue
		}
		enemies = append(enemies, EnemySnapshot{
			Username:  s.Username,
			Distance:  ship.Position.DistTo(s.Position),
			HealthPct: s.HealthPct(),
			ShieldPct: s.ShieldPct(),
		})
	}

	// Queue LLM request.
	llmReq := LLMRequest{
		PlayerID:   req.PlayerID,
		PromptText: text,
		ShipPos:    [3]float32{ship.Position.X, ship.Position.Y, ship.Position.Z},
		HealthPct:  ship.HealthPct(),
		ShieldPct:  ship.ShieldPct(),
		AITier:     ship.AITier,
		Enemies:    enemies,
	}
	select {
	case e.llmReqCh <- llmReq:
		log.Info().Str("player", req.PlayerID).Str("prompt", text).Msg("prompt queued")
	default:
		log.Warn().Str("player", req.PlayerID).Msg("LLM queue full")
		e.sendPlayerError(req.PlayerID, "Server busy, try again shortly", 0)
	}
}

// sendPlayerError sends an error message to a specific player.
func (e *Engine) sendPlayerError(playerID, message string, cooldown float32) {
	data, err := network.EncodeEnvelope(network.MsgTypeError, network.ErrorPayload{
		Message:  message,
		Cooldown: cooldown,
	})
	if err != nil {
		return
	}
	e.hub.SendTo(playerID, data)
}

// sendBehaviorConfirmation tells the client their new behavior was applied.
func (e *Engine) sendBehaviorConfirmation(playerID string, bs *BehaviorSet) {
	data, err := network.EncodeEnvelope(network.MsgTypeEvent, network.BehaviorEventPayload{
		Movement: bs.Primary.Movement,
		Combat:   bs.Primary.Combat,
		Defense:  bs.Primary.Defense,
	})
	if err != nil {
		return
	}
	e.hub.SendTo(playerID, data)
}

// updateEntities runs the full physics + combat pipeline each tick.
func (e *Engine) updateEntities() {
	// Rebuild spatial grid for collision queries.
	e.grid.Rebuild(e.state.Ships)

	// Clear one-shot events from the previous tick.
	e.state.Events = e.state.Events[:0]

	for _, ship := range e.state.Ships {
		if !ship.IsAlive {
			continue
		}

		// 1. Behavior → desired velocity + combat.
		e.executeBehavior(ship)

		// 2. Physics pipeline.
		applyThrust(ship, e.dt)
		applyDrag(ship, e.dt)
		applyBoundaryForces(ship, e.dt)
		clampSpeed(ship)
		integratePosition(ship, e.dt)
		applyRotation(ship, e.dt)

		// 3. Shield regeneration.
		e.updateShields(ship)
	}

	// 3b. Ship-to-ship collision detection & response.
	e.resolveShipCollisions()

	// 4. Projectile physics + collision.
	e.updateProjectiles()

	// 5. Respawn dead ships.
	e.updateRespawns()
}

// buildAndBroadcast serializes the world state and sends it to every client.
func (e *Engine) buildAndBroadcast() {
	if len(e.state.Ships) == 0 {
		return
	}

	data, err := network.EncodeEnvelope(network.MsgTypeStateUpdate, e.buildSnapshot())
	if err != nil {
		log.Error().Err(err).Msg("failed to encode state update")
		return
	}
	e.hub.Broadcast(data)
}

// sendWelcome sends the new player their ID and the current full world state.
func (e *Engine) sendWelcome(client *network.Client, playerID string) {
	welcome := network.WelcomePayload{
		PlayerID: playerID,
		Tick:     e.tick,
		Entities: e.buildEntityList(),
	}
	data, err := network.EncodeEnvelope(network.MsgTypeWelcome, welcome)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode welcome")
		return
	}
	client.Send(data)
}

// buildSnapshot returns a StateUpdatePayload for the current tick.
func (e *Engine) buildSnapshot() network.StateUpdatePayload {
	payload := network.StateUpdatePayload{
		Tick:     e.tick,
		Entities: e.buildEntityList(),
	}

	// Projectiles.
	if len(e.state.Projectiles) > 0 {
		payload.Projectiles = make([]network.ProjectileSnapshot, len(e.state.Projectiles))
		for i, p := range e.state.Projectiles {
			payload.Projectiles[i] = network.ProjectileSnapshot{
				ID:       p.ID,
				Position: [3]float32{p.Position.X, p.Position.Y, p.Position.Z},
				Owner:    p.OwnerID,
			}
		}
	}

	// One-shot events.
	if len(e.state.Events) > 0 {
		payload.Events = make([]network.GameEventWire, len(e.state.Events))
		for i, ev := range e.state.Events {
			payload.Events[i] = network.GameEventWire{
				Type:   ev.Type,
				From:   ev.From,
				To:     ev.To,
				Hit:    ev.Hit,
				Killer: ev.Killer,
				Victim: ev.Victim,
			}
		}
	}

	return payload
}

// buildEntityList converts the internal ship map to a wire-format slice.
func (e *Engine) buildEntityList() []network.EntitySnapshot {
	out := make([]network.EntitySnapshot, 0, len(e.state.Ships))
	for _, ship := range e.state.Ships {
		out = append(out, network.EntitySnapshot{
			ID:       ship.ID,
			Position: [3]float32{ship.Position.X, ship.Position.Y, ship.Position.Z},
			Rotation: [4]float32{ship.Rotation.X, ship.Rotation.Y, ship.Rotation.Z, ship.Rotation.W},
			Color:    ship.Color,
			Health:   ship.Health,
			MaxHP:    ship.MaxHealth,
			Shield:   ship.Shield,
			MaxShld:  ship.MaxShield,
			Alive:    ship.IsAlive,
			Username: ship.Username,
			HullID:   ship.HullID,
		})
	}
	return out
}
