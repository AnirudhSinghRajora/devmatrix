// React hook that wires the Zustand game store to the procedural audio engine.
// Mount once at the top of the component tree (inside App, only when inGame).

import { useEffect, useRef } from 'react';
import { useGameStore } from '../store/gameStore';
import type { BehaviorInfo } from '../types';
import { audioEngine } from './audioEngine';

type Vec3 = [number, number, number];

export function useGameAudio(): void {
  // Sets of VFX IDs we have already played audio for.
  // We use IDs rather than array indices because the arrays can shrink
  // when expired entries are cleaned up each tick.
  const seenLaserIds = useRef(new Set<number>());
  const seenExplosionIds = useRef(new Set<number>());
  const seenKillIds = useRef(new Set<number>());

  // Previous values for detecting transitions.
  const prevMyDeathTime = useRef<number | null>(null);
  const prevJustRespawned = useRef(false);
  const prevBehavior = useRef<BehaviorInfo | null>(null);
  const prevErrorMsg = useRef<string | null>(null);
  const prevOwnShield = useRef<number>(-1);
  const prevOwnHealth = useRef<number>(-1);
  const prevOwnAlive = useRef<boolean>(true);

  // Handle for the continuous engine thruster sound.
  const engineRef = useRef<ReturnType<typeof audioEngine.createEngineSound> | null>(null);

  // ── Unlock AudioContext on first user gesture ──────────────────────────────
  useEffect(() => {
    const unlock = (): void => audioEngine.init();
    document.addEventListener('click', unlock, { once: true });
    document.addEventListener('keydown', unlock, { once: true });
    return () => {
      document.removeEventListener('click', unlock);
      document.removeEventListener('keydown', unlock);
    };
  }, []);

  // ── Ambient + engine sound: start/stop with connection ────────────────────
  useEffect(() => {
    const unsub = useGameStore.subscribe((state) => {
      if (state.connected && !engineRef.current) {
        audioEngine.startAmbient();
        engineRef.current = audioEngine.createEngineSound();
      } else if (!state.connected && engineRef.current) {
        audioEngine.stopAmbient();
        engineRef.current.stop();
        engineRef.current = null;
        // Clear seen sets so we don't skip sounds after reconnect.
        seenLaserIds.current.clear();
        seenExplosionIds.current.clear();
        seenKillIds.current.clear();
        prevMyDeathTime.current = null;
        prevJustRespawned.current = false;
        prevBehavior.current = null;
        prevErrorMsg.current = null;
        prevOwnShield.current = -1;
        prevOwnHealth.current = -1;
        prevOwnAlive.current = true;
      }
    });
    return () => {
      unsub();
      audioEngine.stopAmbient();
      engineRef.current?.stop();
      engineRef.current = null;
    };
  }, []);

  // ── Main event subscriber ──────────────────────────────────────────────────
  useEffect(() => {
    const unsub = useGameStore.subscribe((state) => {
      const myId = state.myPlayerId;
      const ownEntity = myId ? state.entities.get(myId) : undefined;
      const ownPos: Vec3 = ownEntity?.curr.position ?? [0, 0, 0];

      // Helper: compute spatial pan + volume for a world-position event.
      const spatial = (pos: Vec3): { pan: number; volume: number } =>
        audioEngine.spatialise(ownPos, pos);

      // ── Laser beams ─────────────────────────────────────────────────────────
      for (const beam of state.laserBeams) {
        if (!seenLaserIds.current.has(beam.id)) {
          seenLaserIds.current.add(beam.id);
          const { pan, volume } = spatial(beam.from);
          audioEngine.playLaser(beam.hit, pan, volume * 0.38);
        }
      }

      // ── Explosions ──────────────────────────────────────────────────────────
      for (const exp of state.explosions) {
        if (!seenExplosionIds.current.has(exp.id)) {
          seenExplosionIds.current.add(exp.id);
          const { pan, volume } = spatial(exp.position);
          audioEngine.playExplosion(pan, volume * 0.75);
        }
      }

      // ── Kill feed ───────────────────────────────────────────────────────────
      for (const kill of state.killFeed) {
        if (!seenKillIds.current.has(kill.id)) {
          seenKillIds.current.add(kill.id);
          const isOwnKill = kill.killer === myId;
          // Play own-kill fanfare always; play nearby-kill pop only for
          // explosions that are spatially close to avoid audio spam.
          if (isOwnKill) {
            audioEngine.playKill(true);
          } else {
            // We already played an explosion for the victim's position above;
            // only add the quiet kill-pop for kills without an explosion (edge case).
            const victimEntity = state.entities.get(kill.victim);
            if (victimEntity) {
              const { volume } = spatial(victimEntity.curr.position);
              if (volume > 0.4) {
                audioEngine.playKill(false);
              }
            }
          }
        }
      }

      // ── Own ship death ──────────────────────────────────────────────────────
      if (state.myDeathTime !== null && prevMyDeathTime.current === null) {
        audioEngine.playDeath();
      }
      prevMyDeathTime.current = state.myDeathTime;

      // ── Respawn ─────────────────────────────────────────────────────────────
      if (state.justRespawned && !prevJustRespawned.current) {
        audioEngine.playRespawn();
        // Reset shield/health baseline after respawn so we don't trigger
        // hit sounds from the health value snap.
        prevOwnShield.current = -1;
        prevOwnHealth.current = -1;
      }
      prevJustRespawned.current = state.justRespawned;

      // ── Behavior confirmed ──────────────────────────────────────────────────
      if (
        state.currentBehavior !== null &&
        state.currentBehavior !== prevBehavior.current
      ) {
        audioEngine.playBehaviorConfirm();
      }
      prevBehavior.current = state.currentBehavior;

      // ── Error / cooldown ────────────────────────────────────────────────────
      if (
        state.errorMessage !== null &&
        state.errorMessage !== prevErrorMsg.current
      ) {
        audioEngine.playError();
      }
      prevErrorMsg.current = state.errorMessage;

      // ── Own ship taking damage ───────────────────────────────────────────────
      if (ownEntity && ownEntity.alive && !state.justRespawned) {
        if (prevOwnShield.current >= 0) {
          const shieldDelta = prevOwnShield.current - ownEntity.shield;
          const healthDelta = prevOwnHealth.current - ownEntity.health;

          if (shieldDelta > 1) {
            // Shield absorbed a hit — metallic ring
            audioEngine.playShieldHit(0);
          } else if (healthDelta > 1 && ownEntity.shield <= 0) {
            // Shields down, hull taking damage — heavier impact
            audioEngine.playHullImpact();
          }
        }

        // Only update baseline when values have settled (not on respawn snap)
        if (prevOwnShield.current === -1) {
          prevOwnShield.current = ownEntity.shield;
          prevOwnHealth.current = ownEntity.health;
        } else {
          prevOwnShield.current = ownEntity.shield;
          prevOwnHealth.current = ownEntity.health;
        }
      }
      prevOwnAlive.current = ownEntity?.alive ?? true;

      // ── Engine thruster: update speed each tick ─────────────────────────────
      if (engineRef.current && ownEntity) {
        const prev = ownEntity.prev.position;
        const curr = ownEntity.curr.position;
        const dx = curr[0] - prev[0];
        const dz = curr[2] - prev[2];
        // Speed normalised: typical tick displacement of ~3 units at full speed
        const speed = Math.min(1, Math.sqrt(dx * dx + dz * dz) / 4);
        engineRef.current.update(speed, 0); // pan=0: own ship is always centered
      }
    });

    return () => unsub();
  }, []);
}
