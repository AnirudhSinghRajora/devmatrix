# CLAUDE.md — SkyWalker Codebase Context

This file gives an AI assistant everything it needs to work efficiently in this repo without re-exploring the codebase from scratch each time.

---

## What This Project Is

**SkyWalker** is a real-time multiplayer 3D space combat game where players command AI-controlled ships using natural language. Players type plain-English instructions (e.g. "orbit the nearest enemy and fire"), a self-hosted LLM translates them into structured JSON behaviors, and the ships execute those behaviors autonomously in a live arena.

Core loop: `type command → LLM parses it → ship executes behaviors → fight other players → earn coins → upgrade ship/AI processor → unlock more complex tactics`

---

## Repo Structure

```
skywalker/
├── client/          # React + Three.js SPA (TypeScript)
├── server/          # Go game server (authoritative)
│   ├── cmd/skywalker/main.go     # Entry point
│   └── internal/
│       ├── api/       # HTTP REST handlers (register, login, shop, profile)
│       ├── auth/      # JWT auth service + bcrypt password hashing
│       ├── config/    # Env-var-driven config (Config struct)
│       ├── db/        # pgx pool, queries, item cache, async writer
│       ├── game/      # Game loop, behaviors, physics, combat, LLM types
│       ├── llm/       # LLM client, service (worker pool), prompt builder, cooldown
│       └── network/   # WebSocket hub + per-client read/write pumps
├── deploy/          # Caddyfile, deploy.sh, systemd service file
├── docs/planning/   # Architecture docs + phase planning (read for deep context)
└── docker-compose.yml  # Spins up PostgreSQL only
```

---

## Tech Stack (Quick Reference)

| Layer | Tech |
|---|---|
| Backend language | Go 1.26 |
| WebSocket | `github.com/coder/websocket` v1.8.14 |
| Serialization | MessagePack (`vmihailenco/msgpack/v5`) |
| Database | PostgreSQL 16 via `jackc/pgx/v5` |
| Auth | JWT (`golang-jwt/jwt/v5`) + bcrypt |
| Logging | `rs/zerolog` |
| LLM | Gemma 2 9B-IT served via JetStream (OpenAI-compatible API) |
| Frontend | React 19 + TypeScript |
| 3D rendering | Three.js 0.183 + React Three Fiber 9 + Drei 10 |
| State management | Zustand 5 |
| Build tool | Vite 8 |
| Reverse proxy | Caddy (auto-TLS) |
| DB container | Docker (postgres:16-alpine) |

---

## How to Run Locally

### 1. Start the database
```bash
docker compose up -d
```
Postgres starts on port 5432. Credentials: `skywalker_app` / `dev_password` / db `skywalker`.

### 2. Run the server (mock LLM mode — no TPU needed)
```bash
cd server
go run ./cmd/skywalker
```
Server starts on `:8080`. With no `LLM_URL` env var set, it uses the keyword-based mock parser automatically. DB migrations are embedded and run on startup.

### 3. Run the frontend
```bash
cd client
npm install
npm run dev
```
Vite dev server starts on `http://localhost:5173`. The client connects to the server at `ws://localhost:8080/ws`.

### 4. Run with a real LLM endpoint
```bash
LLM_URL=http://localhost:8000 LLM_MODEL=gemma-2-9b-it go run ./cmd/skywalker
```

---

## Key Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP/WS server port |
| `DATABASE_URL` | `postgres://skywalker_app:dev_password@localhost:5432/skywalker?sslmode=disable` | Postgres DSN |
| `JWT_SECRET` | `skywalker-dev-secret-change-in-prod` | JWT signing key |
| `LLM_URL` | `""` (mock mode) | LLM HTTP endpoint (OpenAI-compatible) |
| `LLM_MODEL` | `""` | Model name for `/v1/chat/completions` |
| `LLM_WORKERS` | `4` | Concurrent LLM goroutines |
| `PROMPT_COOLDOWN` | `30s` | Min time between player prompts |
| `TICK_RATE` | `30` | Game loop ticks per second |
| `MAX_PLAYERS` | `200` | Max concurrent connections |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | CORS allowed origins (comma-separated) |

---

## Architecture in One Paragraph

The Go server runs a single-goroutine game loop at 30 TPS. All inputs (WebSocket messages, LLM results) come in through channels and are drained at the top of each tick — no mutexes in the hot path. Every tick: process inputs → run behaviors → advance physics → resolve collisions → serialize snapshot → broadcast to all clients via MessagePack over WebSocket. LLM calls happen in a 4-worker goroutine pool, completely off the game loop. Database writes are async via a buffered channel — the game loop never waits on Postgres. The frontend runs at 60 FPS, interpolating between 30 TPS server snapshots.

---

## The LLM Behavior Pipeline

1. Player submits prompt via WebSocket (`MsgTypePrompt`)
2. Server checks per-player cooldown (tracked in `CooldownTracker`)
3. `LLMRequest` is queued to `llm.Service` channel
4. Worker goroutine builds a tier-aware system prompt (higher AI tier = more enemy context injected)
5. `POST /v1/chat/completions` to LLM server (15s timeout, 200 max tokens, temp=0.1)
6. Response parsed + validated against strict schema into a `BehaviorSet`
7. `BehaviorSet` pushed to engine via result channel
8. Engine applies behavior on next tick, broadcasts `MsgTypeEvent` back to player

**Critical constraint**: The LLM only ever outputs values from a fixed vocabulary of primitives (`orbit`, `chase`, `kite`, `fire_at`, `shield_front`, etc.). It never generates code. Server validates and rejects anything outside the schema.

---

## Game Entities & Types

Key types live in `server/internal/game/`:

- `Ship` — position, velocity, health, shield, behavior, hull/weapon/shield stats
- `BehaviorSet` — primary `Behavior` + slice of `ConditionalBehavior`
- `Behavior` — `Movement` string + `MovementParams` + `Combat` string + `Defense` string
- `Projectile` — position, velocity, owner, weapon type
- `HullStats`, `WeaponStats`, `ShieldStats` — loaded from DB at startup into `ItemCache`

Client types are in `client/src/types.ts`.

---

## Database Schema (Summary)

- `users` — UUID, username, email, bcrypt hash
- `profiles` — coins, kills, deaths, ai_tier (1–5)
- `loadouts` — hull_id, primary_weapon, secondary_weapon, shield_id
- `inventory` — (user_id, item_id) pairs
- `items` — shop catalog (static, seeded via `002_seed_items.sql`)

Migrations live at `server/internal/db/migrations/` and run automatically on server start. Item stats are cached in memory at startup (`db.ItemCache`) — never queried during live gameplay.

---

## Frontend Architecture

- **Entry**: `client/src/main.tsx` → `App.tsx`
- **Routing**: No router — `App.tsx` switches between screens based on Zustand game state (`phase` field: `auth | lobby | game`)
- **State**: `client/src/store/gameStore.ts` (Zustand) — single store for all client state
- **WebSocket**: `client/src/network/socket.ts` — connects, decodes MessagePack frames, calls store actions
- **REST API**: `client/src/network/api.ts` — login, register, shop, profile calls
- **3D scene**: `client/src/components/Scene.tsx` — React Three Fiber canvas, renders all ships/projectiles/effects
- **HUD**: Separate React components layered over the canvas (`HealthPanel`, `LiveLeaderboard`, `KillFeed`, `BehaviorIndicator`, `PromptInput`)

---

## Build & Deploy

```bash
# Build client
cd client && npm run build   # outputs to client/dist/

# Build server binary (Linux amd64 for deployment)
cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o skywalker ./cmd/skywalker

# Full deploy (SSH to production)
./deploy/deploy.sh skywalker-server
```

Production runs on GCP. Caddy handles TLS (auto via Let's Encrypt) and proxies `/ws`, `/api/*`, `/health` to the Go server on `:8080`. Static client files served from `/home/Anirudh/skywalker/client/dist`. Postgres runs in Docker on the same machine. LLM inference server (JetStream) runs on `localhost:8000` via attached TPU.

---

## Common Tasks & Where to Look

| Task | Where |
|---|---|
| Add a new movement primitive | `server/internal/game/movement.go` + update prompt vocabulary in `server/internal/llm/prompt.go` |
| Add a new item to the shop | `server/internal/db/migrations/` (add a new SQL migration) |
| Change LLM prompt structure | `server/internal/llm/prompt.go` |
| Add a new WebSocket message type | `server/internal/network/messages.go` + `client/src/network/socket.ts` |
| Change game loop tick logic | `server/internal/game/engine.go` |
| Modify combat damage/shield math | `server/internal/game/combat.go` |
| Add a new HUD component | `client/src/components/` + mount in `App.tsx` |
| Change ship physics | `server/internal/game/movement.go` (thrust/drag/boundary) |
| Add a new API endpoint | `server/internal/api/handler.go` + register route in `server/cmd/skywalker/main.go` |
| Adjust collision shapes | `server/internal/game/hitshape.go` |

---

## Conventions & Important Notes

- **Game loop is single-goroutine.** Never add mutexes to game-state fields. Route all inputs through channels and drain them at the top of a tick.
- **Never block the game loop on I/O.** All DB writes go through `db.DBWriter` async channels. LLM calls are off-loop in worker goroutines.
- **LLM output must be validated.** Always check against the schema before applying a `BehaviorSet`. Reject silently on invalid output (don't crash, don't apply partial state).
- **MessagePack field names are short single letters** in server structs (`p` = position, `h` = health, etc.) to minimize wire size. Don't rename them without updating the client decoder.
- **Item stats are in-memory only during gameplay.** If you add a new item stat field, update both the DB migration, `db/types.go`, and `db.ItemCache`.
- **CORS is strict.** `ALLOWED_ORIGINS` must include the client origin. In dev it defaults to `http://localhost:5173`.
- **JWT secret must be changed in production.** Default is a plaintext dev string.
- **Guest mode**: If no `DATABASE_URL` is set or user plays as guest, profiles aren't persisted. This is intentional for dev/demo use.

---

## Docs

The `docs/planning/` directory has detailed phase-by-phase planning documents that explain design decisions thoroughly. If something in the code seems non-obvious, check there first:

- `architecture_overview.md` — high-level system design rationale
- `phase_3_llm_behavior_pipeline.md` — LLM integration design decisions
- `phase_4_combat_physics.md` — physics + collision design
- `phase_5_persistence_economy.md` — economy + progression design
