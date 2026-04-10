import { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import type { EntityState } from '../types';

interface ShipProps {
  entity: EntityState;
}

const _q = new THREE.Quaternion();
const _v = new THREE.Vector3();

export default function Ship({ entity }: ShipProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame(() => {
    const mesh = meshRef.current;
    if (!mesh) return;

    // Interpolate toward target position/rotation for smooth visuals
    _v.set(entity.pos[0], entity.pos[1], entity.pos[2]);
    mesh.position.lerp(_v, 0.3);

    _q.set(entity.rot[0], entity.rot[1], entity.rot[2], entity.rot[3]);
    mesh.quaternion.slerp(_q, 0.3);
  });

  return (
    <mesh ref={meshRef}>
      <boxGeometry args={[1, 0.5, 2]} />
      <meshStandardMaterial color="#00ff88" emissive="#003322" />
    </mesh>
  );
}
