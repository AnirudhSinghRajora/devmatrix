// Wire protocol types - field names match server msgpack short tags.

export const MsgType = {
  StateUpdate: 1,
  Welcome: 2,
  Prompt: 3,
  Event: 4,
  Error: 5,
} as const;

export const GameEventType = {
  LaserFired: 1,
  Kill: 2,
  Respawn: 3,
} as const;

// Raw envelope decoded from msgpack binary frame.
// Note: `p` is already decoded by the outer decode() because Go's
// msgpack.RawMessage embeds bytes inline in the stream.
export interface WireEnvelope {
  t: number; // message type
  p: unknown; // already-decoded nested payload
}

// Raw entity snapshot from the wire (short keys).
export interface WireEntitySnapshot {
  i: string; // id
  p: [number, number, number]; // position
  r: [number, number, number, number]; // rotation quaternion (x,y,z,w)
  c: [number, number, number]; // color RGB 0-1
  h: number;  // health
  mh: number; // max health
  s: number;  // shield
  ms: number; // max shield
  a: boolean; // alive
  u?: string; // username (omitempty)
  hl?: string; // hull ID (omitempty)
}

export interface WireProjectileSnapshot {
  i: number;
  p: [number, number, number];
  o: string;
}

export interface WireGameEvent {
  t: number;
  f?: [number, number, number];
  to?: [number, number, number];
  h?: boolean;
  k?: string;
  v?: string;
}

// Server → Client: world state, 30 TPS.
export interface WireStateUpdate {
  t: number; // tick
  e: WireEntitySnapshot[];
  pr?: WireProjectileSnapshot[];
  ev?: WireGameEvent[];
}

// Server → Client: sent once on connect.
export interface WireWelcome {
  pid: string; // assigned player ID
  t: number; // current tick
  e: WireEntitySnapshot[];
}

// --- App-level types (clean names for components) ---

export interface Snapshot {
  position: [number, number, number];
  rotation: [number, number, number, number];
}

export interface InterpolatedEntity {
  id: string;
  username: string;
  hullId: string;
  color: [number, number, number];
  prev: Snapshot;
  curr: Snapshot;
  health: number;
  maxHealth: number;
  shield: number;
  maxShield: number;
  alive: boolean;
  kills: number;
  deaths: number;
}

// Server → Client: error message.
export interface WireError {
  msg: string;
  cd?: number; // remaining cooldown in seconds
}

// Server → Client: behavior confirmation event.
export interface WireBehaviorEvent {
  m: string;  // movement
  c?: string; // combat
  d?: string; // defense
}

// Current behavior displayed in the HUD.
export interface BehaviorInfo {
  movement: string;
  combat: string;
  defense: string;
}

// --- VFX types ---

export interface LaserBeam {
  id: number;
  from: [number, number, number];
  to: [number, number, number];
  hit: boolean;
  time: number;
}

export interface ExplosionVfx {
  id: number;
  position: [number, number, number];
  time: number;
}

export interface KillFeedEntry {
  id: number;
  killer: string;
  victim: string;
  killerName: string;
  victimName: string;
  time: number;
}

export interface ProjectileEntity {
  id: number;
  position: [number, number, number];
  owner: string;
}

// --- Auth & Economy types ---

export interface AuthResponse {
  token: string;
  user_id: string;
  username: string;
}

export interface PlayerProfile {
  username: string;
  coins: number;
  kills: number;
  deaths: number;
  ai_tier: number;
  inventory: string[];
  loadout: {
    hull: string;
    primary_weapon: string;
    secondary_weapon: string | null;
    shield: string;
  };
}

export interface ShopItem {
  id: string;
  name: string;
  category: string;
  price: number;
  tier_required: number;
  description: string;
  stats: string;
}

export interface LeaderboardEntry {
  username: string;
  kills: number;
  deaths: number;
  ai_tier: number;
}
