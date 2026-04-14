import { useRef, useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { Stars } from '@react-three/drei';
import { EffectComposer, Bloom } from '@react-three/postprocessing';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';
import Ship from './Ship';
import EngineTrail from './EngineTrail';
import CameraFollow from './CameraFollow';
import LaserBeam from './LaserBeam';
import Explosion from './Explosion';

/**
 * Subscribe only to the sorted list of entity IDs so Scene re-renders
 * only when players join/leave — NOT on every 33ms position tick.
 */
function useEntityIds(): string[] {
  const prevRef = useRef<string[]>([]);
  return useGameStore((s) => {
    const ids = Array.from(s.entities.keys()).sort();
    // Return the same array reference if IDs haven't changed,
    // so Zustand's shallow equality check skips the re-render.
    const prev = prevRef.current;
    if (
      ids.length === prev.length &&
      ids.every((id, i) => id === prev[i])
    ) {
      return prev;
    }
    prevRef.current = ids;
    return ids;
  });
}

function CombatVfx() {
  const laserBeams = useGameStore((s) => s.laserBeams);
  const explosions = useGameStore((s) => s.explosions);
  const projectiles = useGameStore((s) => s.projectiles);

  return (
    <>
      {laserBeams.map((beam) => (
        <LaserBeam
          key={beam.id}
          from={beam.from}
          to={beam.to}
          hit={beam.hit}
          time={beam.time}
        />
      ))}
      {explosions.map((exp) => (
        <Explosion key={exp.id} position={exp.position} time={exp.time} />
      ))}
      {projectiles.map((p) => (
        <mesh key={p.id} position={p.position}>
          <sphereGeometry args={[0.3, 6, 6]} />
          <meshBasicMaterial color="#00ccff" />
        </mesh>
      ))}
    </>
  );
}

function ArenaBoundary() {
  const geo = useMemo(() => new THREE.EdgesGeometry(new THREE.BoxGeometry(500, 100, 500)), []);
  return (
    <lineSegments geometry={geo}>
      <lineBasicMaterial color="#1a3a5c" transparent opacity={0.3} />
    </lineSegments>
  );
}

export default function Scene() {
  const entityIds = useEntityIds();
  const myPlayerId = useGameStore((s) => s.myPlayerId);

  return (
    <Canvas
      camera={{ position: [0, 15, 30], fov: 60, near: 0.1, far: 2000 }}
      style={{ width: '100vw', height: '100vh' }}
    >
      <ambientLight intensity={0.25} />
      <directionalLight position={[10, 10, 5]} intensity={0.8} />
      <directionalLight position={[-5, -3, -10]} intensity={0.15} color="#4488ff" />
      <Stars radius={400} depth={100} count={5000} factor={8} saturation={0.2} fade />
      <CameraFollow />
      <gridHelper args={[1000, 200, '#1a1a2e', '#0a0a15']} />
      <ArenaBoundary />

      {entityIds.map((id) => (
        <Ship
          key={id}
          entityId={id}
          isOwn={id === myPlayerId}
        />
      ))}
      {entityIds.map((id) => (
        <EngineTrail key={`trail-${id}`} entityId={id} />
      ))}
      <CombatVfx />
      <EffectComposer>
        <Bloom
          luminanceThreshold={0.6}
          luminanceSmoothing={0.4}
          intensity={0.8}
          mipmapBlur
        />
      </EffectComposer>
    </Canvas>
  );
}
