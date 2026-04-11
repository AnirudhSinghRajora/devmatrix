import { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';

interface ShipProps {
  entityId: string;
  isOwn: boolean;
}

const _prevPos = new THREE.Vector3();
const _currPos = new THREE.Vector3();
const _prevQuat = new THREE.Quaternion();
const _currQuat = new THREE.Quaternion();
const _color = new THREE.Color();
const _emissive = new THREE.Color();

export default function Ship({ entityId, isOwn }: ShipProps) {
  const groupRef = useRef<THREE.Group>(null);
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame(() => {
    const group = groupRef.current;
    const mesh = meshRef.current;
    if (!group || !mesh) return;

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
    mesh.quaternion.slerpQuaternions(_prevQuat, _currQuat, t);

    const mat = mesh.material as THREE.MeshStandardMaterial;
    _color.setRGB(e.color[0], e.color[1], e.color[2]);
    _emissive.setRGB(e.color[0] * 0.3, e.color[1] * 0.3, e.color[2] * 0.3);
    mat.color.copy(_color);
    mat.emissive.copy(_emissive);
    mat.emissiveIntensity = isOwn ? 0.6 : 0.2;

    // Dead ships are invisible.
    mesh.visible = e.alive;
  });

  return (
    <group ref={groupRef}>
      <mesh ref={meshRef}>
        <boxGeometry args={[1, 0.5, 2]} />
        <meshStandardMaterial />
      </mesh>
    </group>
  );
}
