import { create } from 'zustand';
import type {
  BehaviorInfo,
  ExplosionVfx,
  InterpolatedEntity,
  KillFeedEntry,
  LaserBeam,
  ProjectileEntity,
  Snapshot,
  WireBehaviorEvent,
  WireEntitySnapshot,
  WireStateUpdate,
  WireWelcome,
} from '../types';
import { GameEventType } from '../types';

const TICK_DURATION = 1000 / 30; // 33.33ms

interface GameState {
  connected: boolean;
  myPlayerId: string | null;
  tick: number;
  lastUpdateTime: number;
  entities: Map<string, InterpolatedEntity>;
  tickDuration: number;

  // Death/respawn.
  myDeathTime: number | null; // performance.now() when own player died, null if alive
  justRespawned: boolean; // true for one tick after respawn, triggers cooldown reset

  // Phase 3: behavior + prompt feedback.
  currentBehavior: BehaviorInfo | null;
  errorMessage: string | null;
  errorCooldown: number; // remaining cooldown in seconds (from server)

  // Phase 4: combat VFX.
  laserBeams: LaserBeam[];
  explosions: ExplosionVfx[];
  killFeed: KillFeedEntry[];
  projectiles: ProjectileEntity[];
  nextVfxId: number;

  setConnected: (connected: boolean) => void;
  applyWelcome: (welcome: WireWelcome) => void;
  applyStateUpdate: (update: WireStateUpdate) => void;
  applyBehaviorEvent: (event: WireBehaviorEvent) => void;
  setError: (msg: string, cooldown?: number) => void;
  clearError: () => void;
}

function wireToSnapshot(e: WireEntitySnapshot): Snapshot {
  return { position: e.p, rotation: e.r };
}

export const useGameStore = create<GameState>((set) => ({
  connected: false,
  myPlayerId: null,
  tick: 0,
  lastUpdateTime: 0,
  entities: new Map(),
  tickDuration: TICK_DURATION,
  myDeathTime: null,
  justRespawned: false,
  currentBehavior: null,
  errorMessage: null,
  errorCooldown: 0,
  laserBeams: [],
  explosions: [],
  killFeed: [],
  projectiles: [],
  nextVfxId: 1,

  setConnected: (connected) => {
    // Keep entities and world state on disconnect so ships extrapolate
    // smoothly during brief network hiccups.  applyWelcome will
    // replace everything when the connection is re-established.
    set({ connected });
  },

  applyWelcome: (welcome) =>
    set(() => {
      const entities = new Map<string, InterpolatedEntity>();
      for (const e of welcome.e) {
        const snap = wireToSnapshot(e);
        entities.set(e.i, {
          id: e.i,
          username: e.u || e.i,
          hullId: e.hl || 'hull_basic',
          color: e.c,
          prev: snap,
          curr: snap,
          health: e.h,
          maxHealth: e.mh,
          shield: e.s,
          maxShield: e.ms,
          alive: e.a,
          kills: 0,
          deaths: 0,
        });
      }
      return {
        myPlayerId: welcome.pid,
        tick: welcome.t,
        lastUpdateTime: performance.now(),
        entities,
      };
    }),

  applyStateUpdate: (update) =>
    set((state) => {
      const now = performance.now();
      const entities = new Map<string, InterpolatedEntity>();

      for (const e of update.e) {
        const existing = state.entities.get(e.i);
        const snap = wireToSnapshot(e);

        if (existing) {
          entities.set(e.i, {
            ...existing,
            username: e.u || existing.username,
            hullId: e.hl || existing.hullId,
            prev: existing.curr,
            curr: snap,
            health: e.h,
            maxHealth: e.mh,
            shield: e.s,
            maxShield: e.ms,
            alive: e.a,
            kills: existing.kills,
            deaths: existing.deaths,
          });
        } else {
          entities.set(e.i, {
            id: e.i,
            username: e.u || e.i,
            hullId: e.hl || 'hull_basic',
            color: e.c,
            prev: snap,
            curr: snap,
            health: e.h,
            maxHealth: e.mh,
            shield: e.s,
            maxShield: e.ms,
            alive: e.a,
            kills: 0,
            deaths: 0,
          });
        }
      }

      // Clean up expired VFX.
      let laserBeams = state.laserBeams.length > 0
        ? state.laserBeams.filter((b) => now - b.time < 500)
        : state.laserBeams;
      let explosions = state.explosions.length > 0
        ? state.explosions.filter((x) => now - x.time < 1000)
        : state.explosions;
      let killFeed = state.killFeed.length > 0
        ? state.killFeed.filter((k) => now - k.time < 8000)
        : state.killFeed;
      let nextVfxId = state.nextVfxId;

      // Process combat events.
      let myDeathTime = state.myDeathTime;
      let justRespawned = false;
      if (update.ev && update.ev.length > 0) {
        for (const ev of update.ev) {
          switch (ev.t) {
            case GameEventType.LaserFired:
              if (ev.f && ev.to) {
                laserBeams = [...laserBeams, {
                  id: nextVfxId++,
                  from: ev.f,
                  to: ev.to,
                  hit: ev.h ?? false,
                  time: now,
                }];
              }
              break;
            case GameEventType.Kill:
              if (ev.k && ev.v) {
                const killerEntity = entities.get(ev.k);
                const victimEntity = entities.get(ev.v);
                // Increment kill/death counters.
                if (killerEntity) {
                  entities.set(ev.k, { ...killerEntity, kills: killerEntity.kills + 1 });
                }
                if (victimEntity) {
                  entities.set(ev.v, { ...victimEntity, deaths: victimEntity.deaths + 1 });
                }
                killFeed = [...killFeed, {
                  id: nextVfxId++,
                  killer: ev.k,
                  victim: ev.v,
                  killerName: killerEntity?.username || ev.k.slice(0, 8),
                  victimName: victimEntity?.username || ev.v.slice(0, 8),
                  time: now,
                }];
                const victim = entities.get(ev.v);
                if (victim) {
                  explosions = [...explosions, {
                    id: nextVfxId++,
                    position: victim.curr.position,
                    time: now,
                  }];
                }
                // Track own death.
                if (ev.v === state.myPlayerId) {
                  myDeathTime = now;
                }
              }
              break;
            case GameEventType.Respawn:
              // Own player respawned — clear death, signal cooldown reset.
              if (ev.v === state.myPlayerId) {
                myDeathTime = null;
                justRespawned = true;
              }
              break;
          }
        }
      }

      // Update projectiles from server state.
      const projectiles: ProjectileEntity[] = update.pr
        ? update.pr.map((p) => ({ id: p.i, position: p.p, owner: p.o }))
        : [];

      return {
        tick: update.t,
        lastUpdateTime: now,
        entities,
        laserBeams,
        explosions,
        killFeed,
        projectiles,
        nextVfxId,
        myDeathTime,
        justRespawned,
      };
    }),

  applyBehaviorEvent: (event) =>
    set(() => ({
      currentBehavior: {
        movement: event.m,
        combat: event.c || 'hold_fire',
        defense: event.d || 'shield_balanced',
      },
      errorMessage: null,
    })),

  setError: (msg, cooldown) =>
    set(() => ({
      errorMessage: msg,
      errorCooldown: cooldown ?? 0,
    })),

  clearError: () => set({ errorMessage: null, errorCooldown: 0 }),
}));
