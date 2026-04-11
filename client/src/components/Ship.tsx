import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import { useGLTF } from '@react-three/drei';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';

const HULL_MODELS: Record<string, string> = {
  hull_basic:   '/models/ships/Striker.glb',
  hull_medium:  '/models/ships/Challenger.glb',
  hull_heavy:   '/models/ships/Imperial.glb',
  hull_stealth: '/models/ships/Omen.glb',
};

// Preload all models.
Object.values(HULL_MODELS).forEach((path) => useGLTF.preload(path));

interface ShipProps {
  entityId: string;
  isOwn: boolean;
}

const _prevPos = new THREE.Vector3();
const _currPos = new THREE.Vector3();
const _prevQuat = new THREE.Quaternion();
const _currQuat = new THREE.Quaternion();

function ShipModel({ hullId, color, isOwn }: { hullId: string; color: [number, number, number]; isOwn: boolean }) {
  const modelPath = HULL_MODELS[hullId] ?? HULL_MODELS['hull_basic'];
  const { scene } = useGLTF(modelPath);

  const clonedScene = useMemo(() => {
    const clone = scene.clone(true);
    // Clone materials so instances don't share state, but preserve original colors/textures.
    clone.traverse((child) => {
      if (child instanceof THREE.Mesh && child.material) {
        child.material = (child.material as THREE.MeshStandardMaterial).clone();
      }
    });
    return clone;
  }, [scene]);

  return (
    <group>
      <primitive object={clonedScene} scale={1.5} />
      {/* Colored ring under ship for player identification */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.3, 0]}>
        <ringGeometry args={[1.8, 2.2, 32]} />
        <meshBasicMaterial
          color={new THREE.Color(color[0], color[1], color[2])}
          transparent
          opacity={isOwn ? 0.7 : 0.4}
          depthWrite={false}
          blending={THREE.AdditiveBlending}
          side={THREE.DoubleSide}
        />
      </mesh>
    </group>
  );
}

export default function Ship({ entityId, isOwn }: ShipProps) {
  const groupRef = useRef<THREE.Group>(null);
  const shieldRef = useRef<THREE.Mesh>(null);
  const prevShield = useRef<number>(-1);
  const shieldFlashTime = useRef<number>(0);

  // Read initial values for the model (these don't change per-tick).
  const entity = useGameStore((s) => s.entities.get(entityId));
  const hullId = entity?.hullId ?? 'hull_basic';
  const color = entity?.color ?? [1, 1, 1] as [number, number, number];

  useFrame(() => {
    const group = groupRef.current;
    if (!group) return;

    const state = useGameStore.getState();
    const e = state.entities.get(entityId);
    if (!e) return;

    const elapsed = performance.now() - state.lastUpdateTime;
    const t = Math.min(elapsed / state.tickDuration, 1.0);

    _prevPos.set(e.prev.position[0], e.prev.position[1], e.prev.position[2]);
    _currPos.set(e.curr.position[0], e.curr.position[1], e.curr.position[2]);

    if (_prevPos.distanceToSquared(_currPos) > 2500) {
      group.position.copy(_currPos);
    } else {
      group.position.lerpVectors(_prevPos, _currPos, t);
    }

    _prevQuat.set(e.prev.rotation[0], e.prev.rotation[1], e.prev.rotation[2], e.prev.rotation[3]);
    _currQuat.set(e.curr.rotation[0], e.curr.rotation[1], e.curr.rotation[2], e.curr.rotation[3]);
    group.quaternion.slerpQuaternions(_prevQuat, _currQuat, t);

    // Dead ships are invisible.
    group.visible = e.alive;

    // Shield flash: detect shield decrease.
    if (prevShield.current >= 0 && e.shield < prevShield.current && e.shield > 0) {
      shieldFlashTime.current = performance.now();
    }
    prevShield.current = e.shield;

    // Animate shield flash.
    const shield = shieldRef.current;
    if (shield) {
      const flashElapsed = (performance.now() - shieldFlashTime.current) / 1000;
      if (flashElapsed < 0.4) {
        shield.visible = true;
        (shield.material as THREE.MeshBasicMaterial).opacity = Math.max(0, 0.5 * (1 - flashElapsed / 0.4));
      } else {
        shield.visible = false;
      }
    }
  });

  return (
    <group ref={groupRef}>
      <ShipModel hullId={hullId} color={color} isOwn={isOwn} />
      {/* Shield hit flash bubble */}
      <mesh ref={shieldRef} visible={false}>
        <sphereGeometry args={[2.5, 16, 12]} />
        <meshBasicMaterial
          color="#44aaff"
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
          side={THREE.BackSide}
        />
      </mesh>
    </group>
  );
}
