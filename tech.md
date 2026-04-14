# How SkyWalker Works

A breakdown of the system design, architecture, and the interesting engineering behind SkyWalker.

---

## The Big Picture

SkyWalker runs as one central authoritative game server that owns all state. Clients are pure renderers — they don't simulate anything, they just display what the server tells them. This keeps the game consistent for everyone and prevents cheating without any extra effort.

The stack is split into three main layers:

```
Browser (React + Three.js)
    ↕  WebSocket (binary, 30x per second)
Go Game Server (authoritative loop at 30 TPS)
    ↕  HTTP
Self-hosted LLM (Gemma 2 9B on Google TPU)
    ↕  SQL
PostgreSQL (profiles, inventory, economy)
```

Everything that matters — position, health, combat, behaviors — lives and runs on the server. The browser just gets a snapshot of the world every 33ms and renders it.

---

## Backend: Go Game Server

The backend is written in Go. The core is a game loop that ticks at exactly 30 times per second.

Every tick, it does the same things in the same order: process any queued player inputs, run all ship behaviors (movement, combat, shields), advance projectiles, resolve collisions, then serialize the full world state and blast it to every connected client. The whole thing runs in a single goroutine with no locking needed — Go's channels handle all the concurrency at the edges (incoming WebSocket messages, LLM results, database writes).

Each tick's budget is 33ms. To protect that, database writes happen asynchronously on a background goroutine. The game loop never waits on the database — it just drops a write onto a buffered channel and keeps going.

**Key libraries:**
- `github.com/coder/websocket` for WebSocket connections
- `github.com/vmihailenco/msgpack/v5` for binary serialization
- `github.com/jackc/pgx/v5` for PostgreSQL
- `github.com/golang-jwt/jwt/v5` for auth tokens
- `github.com/rs/zerolog` for structured logging

---

## Networking: WebSocket + MessagePack

Every connected player has a WebSocket connection to the server. The server broadcasts a binary-encoded world snapshot to all players 30 times per second.

We use MessagePack instead of JSON because it's about 30% smaller on the wire for the same data. With 100 players, each tick snapshot comes out to roughly 6 KB — around 180 KB/s per player at full speed. That's manageable on any decent connection.

The message format for each entity in the snapshot looks roughly like this:

```
entity_id | position [x, y, z] | rotation [quaternion] | health | shield | alive | username | hull_type
```

Beyond state snapshots, there are a few other message types: a welcome message when you first connect (gives you your player ID), event messages for visual effects like laser hits and ship explosions, and error messages for things like cooldown violations.

Each player gets a dedicated read goroutine and write goroutine. Slow clients get their send buffer dropped rather than blocking the rest of the server — no one player can accidentally hold up the broadcast.

---

## Authentication

Auth is straightforward JWT. When a player registers or logs in through the HTTP REST API, they get back a JWT token. That token gets stored in localStorage and attached to the WebSocket URL as a query parameter when connecting.

The WebSocket hub validates the token before allowing the connection. Duplicate sessions (same user connecting twice) are rejected.

Passwords are hashed with bcrypt. All database queries are parameterized — no raw string interpolation anywhere near SQL.

Guests can play without registering, but their progress isn't saved.

---

## The LLM Behavior Pipeline

This is the most interesting part of the system.

When a player types a command, the server:

1. Checks their cooldown (30 seconds between commands, tracked in memory)
2. Queues an LLM request to a pool of 4 worker goroutines
3. Each worker builds a system prompt that includes the player's ship status, nearby enemies (depending on their AI tier), and the full vocabulary of available movement/combat/defense primitives
4. Sends an HTTP POST to the LLM server at `POST /v1/chat/completions` (OpenAI-compatible API)
5. Parses the returned JSON into a `BehaviorSet`
6. Passes the result back to the game engine via a channel
7. The engine applies the new behavior on the next tick

The LLM is Gemma 2 9B, served via JetStream on a Google TPU instance. We cap output at 200 tokens because behavior JSON is compact and we want fast responses (target: under 2 seconds). Temperature is set to 0.1 to keep outputs deterministic.

**Critically: the LLM never generates code.** It maps natural language to a fixed vocabulary of behavior primitives — things like `orbit`, `kite`, `dodge`, `fire_at`, `shield_front`. The server validates the output against a strict schema and rejects anything that doesn't fit. This keeps the LLM's role contained and means there's no risk of prompt injection affecting game logic.

If no LLM endpoint is configured (local dev), the server falls back to a keyword-based parser that handles simple cases like "chase" or "hold fire" using pattern matching. Good enough for testing without needing the TPU.

**Prompt context scales with AI tier.** A Tier 1 player's prompt includes just their own ship status. A Tier 3 player gets the 3 nearest enemies added to their context. Tier 5 gets everything. This is also how we balance server-side compute: early players get cheap, short prompts; invested players get richer but heavier prompts.

---

## Physics & Collision

Ships have mass, thrust, drag, and a maximum speed. Movement is physics-based — you accelerate toward your target velocity, coast on drag, and bump off other ships with impulse-based collision response.

Collision detection uses a spatial grid (cells of 100 units) rebuilt every tick. This gives O(n) broad-phase performance instead of O(n²) pair-checking. Close-range narrow-phase uses compound sphere shapes per hull, meaning hit detection respects the actual shape of the ship and its orientation.

Hit shapes:
- **Striker (Scout)**: Two overlapping spheres, elongated along the ship's nose
- **Challenger (Cruiser)**: Three spheres spanning the wide wings
- **Imperial (Titan)**: Three spheres along its long bulk
- **Omen (Phantom)**: Slender two-sphere shape with a thin tail

Projectiles check against these shapes every tick. Hitscan weapons (lasers) raycast against them instantly. Ballistic projectiles (plasma, railgun) advance their position each tick and check intersection.

---

## Database: PostgreSQL

Five main tables:

- **users** — auth credentials (UUID, username, email, bcrypt password hash)
- **profiles** — per-player stats (coins, kills, deaths, AI tier)
- **loadouts** — which hull, weapon, and shield a player has equipped
- **inventory** — items the player owns (user_id + item_id composite key)
- **items** — the shop catalog (static, seeded via migration)

The `items` table is loaded into memory at startup as an `ItemCache`. No DB queries happen during gameplay for item stats — everything is looked up from memory. Only profile reads and writes (on connect and on kill) touch the database at runtime.

Connection pooling via `pgxpool` with 20 max connections and 5 warm idle connections. Connection lifetime is 30 minutes to avoid stale connections on long-running server instances.

---

## Frontend: React + Three.js

The frontend is a single-page React app written in TypeScript. The 3D rendering is done with React Three Fiber (a React renderer for Three.js), which lets us describe the scene declaratively in JSX while Three.js handles the actual WebGL under the hood.

Global client state lives in Zustand — a minimal, hook-based state store. When a new WebSocket message comes in, the socket handler calls into the Zustand store to update entity positions, events, and UI state. React components that care about that data re-render automatically.

The client runs at 60 FPS. Since the server only sends 30 snapshots per second, we interpolate entity positions between the last two known states using a 100ms buffer. This smooths out any jitter from network variance.

Ship models are glTF files served from `/public/models/`. Each hull type has its own model. They're rendered with proper lighting, shadow casters, and engine trail particle effects.

**Key libraries:**
- React 19 + TypeScript
- Three.js 0.183
- React Three Fiber 9 + Drei 10 (3D helpers)
- Zustand 5 (state)
- `@msgpack/msgpack` (decode binary WebSocket frames)
- Vite 8 (build tool)

---

## Deployment

The server runs directly on a GCP instance. Caddy sits in front as a reverse proxy, handling TLS automatically via Let's Encrypt. PostgreSQL runs in a Docker container on the same machine.

The LLM inference server (JetStream) runs on the same GCP instance's attached TPU. It exposes a local HTTP endpoint on port 8000. The Go server calls it directly, no network hop needed.

```
Internet → Caddy (HTTPS/WSS) → Go Game Server (:8080)
                                     ↓
                             LLM (localhost:8000)
                                     ↓
                          PostgreSQL (Docker, :5432)
```

Config is environment-variable driven. The deployment script handles pulling the latest binary, restarting the systemd service, and warming up the DB connections.

---

## Interesting Design Decisions

**Single-goroutine game loop.** The entire game tick runs in one goroutine. No mutexes in the hot path. All external inputs come in through channels and get drained at the top of each tick. This makes the loop easy to reason about and avoids data races completely.

**LLM output is a fixed vocabulary, not free code.** The LLM can only produce values from a known set of primitives. The server validates everything. No eval, no dynamic code, no prompt injection path into game logic.

**Behavior cooldown is server-enforced.** The client shows a countdown for UX purposes, but the server checks timestamps independently. You can't race it from the client.

**All item stats are in memory.** The database is only queried on player join and on economy events (kills, purchases). During live gameplay, every stat lookup is a map access. This keeps the 30ms tick budget stable regardless of DB load.

**AI tier scales both the experience and the server cost.** Low-tier players get short, cheap prompts. High-tier players have earned the right to richer prompts by investing in the game. The progression system is also a compute budget.
