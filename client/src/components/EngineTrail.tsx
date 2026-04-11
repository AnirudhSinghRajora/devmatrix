import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';

const TRAIL_LENGTH = 48;
const FADE_RATE = 0.92; // per-frame multiplier — dies in ~25 frames
const DRIFT_SPEED = 0.15; // how fast particles drift backward
const SPREAD = 0.08; // random lateral spread on emit

interface EngineTrailProps {
  entityId: string;
}

export default function EngineTrail({ entityId }: EngineTrailProps) {
  const pointsRef = useRef<THREE.Points>(null);
  const head = useRef(0);
  const lastEmit = useRef(0);

  // Per-particle: position (3), velocity (3), life (1) = 7 floats each.
  const particles = useMemo(() => new Float32Array(TRAIL_LENGTH * 7), []);

  const { positions, sizes } = useMemo(() => {
    const pos = new Float32Array(TRAIL_LENGTH * 3);
    const sz = new Float32Array(TRAIL_LENGTH);
    return { positions: pos, sizes: sz };
  }, []);

  const geometry = useMemo(() => {
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    geo.setAttribute('size', new THREE.BufferAttribute(sizes, 1));
    return geo;
  }, [positions, sizes]);

  useFrame(() => {
    const state = useGameStore.getState();
    const e = state.entities.get(entityId);
    if (!e || !e.alive) {
      // Hide all particles.
      for (let j = 0; j < TRAIL_LENGTH; j++) {
        positions[j * 3 + 1] = -9999;
        sizes[j] = 0;
      }
      geometry.attributes.position.needsUpdate = true;
      geometry.attributes.size.needsUpdate = true;
      return;
    }

    const elapsed = performance.now() - state.lastUpdateTime;
    const t = Math.min(elapsed / state.tickDuration, 1.0);

    // Interpolated position.
    const x = e.prev.position[0] + (e.curr.position[0] - e.prev.position[0]) * t;
    const y = e.prev.position[1] + (e.curr.position[1] - e.prev.position[1]) * t;
    const z = e.prev.position[2] + (e.curr.position[2] - e.prev.position[2]) * t;

    // Velocity from prev→curr.
    const vx = e.curr.position[0] - e.prev.position[0];
    const vy = e.curr.position[1] - e.prev.position[1];
    const vz = e.curr.position[2] - e.prev.position[2];
    const speed = Math.sqrt(vx * vx + vy * vy + vz * vz);

    // Emit new particle when moving (throttled to every ~2 frames).
    const now = performance.now();
    if (speed > 0.05 && now - lastEmit.current > 50) {
      const i = head.current % TRAIL_LENGTH;
      const base = i * 7;
      // Position with slight random spread.
      particles[base] = x + (Math.random() - 0.5) * SPREAD;
      particles[base + 1] = y + (Math.random() - 0.5) * SPREAD;
      particles[base + 2] = z + (Math.random() - 0.5) * SPREAD;
      // Velocity: opposite to travel direction (drift backward).
      const invSpeed = speed > 0 ? DRIFT_SPEED / speed : 0;
      particles[base + 3] = -vx * invSpeed + (Math.random() - 0.5) * 0.02;
      particles[base + 4] = -vy * invSpeed + (Math.random() - 0.5) * 0.02;
      particles[base + 5] = -vz * invSpeed + (Math.random() - 0.5) * 0.02;
      // Life.
      particles[base + 6] = 1.0;
      head.current++;
      lastEmit.current = now;
    }

    // Update all particles: drift + fade.
    for (let j = 0; j < TRAIL_LENGTH; j++) {
      const base = j * 7;
      const life = particles[base + 6];
      if (life <= 0.01) {
        // Dead — park offscreen.
        positions[j * 3] = 0;
        positions[j * 3 + 1] = -9999;
        positions[j * 3 + 2] = 0;
        sizes[j] = 0;
        continue;
      }
      // Drift.
      particles[base] += particles[base + 3];
      particles[base + 1] += particles[base + 4];
      particles[base + 2] += particles[base + 5];
      // Fade.
      particles[base + 6] *= FADE_RATE;
      const l = particles[base + 6];
      // Write to render buffers.
      positions[j * 3] = particles[base];
      positions[j * 3 + 1] = particles[base + 1];
      positions[j * 3 + 2] = particles[base + 2];
      sizes[j] = l * 0.5;
    }

    geometry.attributes.position.needsUpdate = true;
    geometry.attributes.size.needsUpdate = true;
  });

  return (
    <points ref={pointsRef} geometry={geometry} frustumCulled={false}>
      <pointsMaterial
        size={0.4}
        color="#00ccff"
        transparent
        opacity={0.6}
        depthWrite={false}
        blending={THREE.AdditiveBlending}
        sizeAttenuation
      />
    </points>
  );
}
