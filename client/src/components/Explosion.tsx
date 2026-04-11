import { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface ExplosionProps {
  position: [number, number, number];
  time: number;
}

const EXPLOSION_DURATION = 0.6; // seconds

export default function Explosion({ position, time }: ExplosionProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  useFrame(() => {
    const mesh = meshRef.current;
    if (!mesh) return;
    const elapsed = (performance.now() - time) / 1000;
    const t = elapsed / EXPLOSION_DURATION;
    mesh.scale.setScalar(1 + t * 8);
    const mat = mesh.material as THREE.MeshBasicMaterial;
    mat.opacity = Math.max(0, 1 - t);
    mesh.visible = t < 1;
  });

  return (
    <mesh ref={meshRef} position={position}>
      <icosahedronGeometry args={[0.5, 1]} />
      <meshBasicMaterial color="#ff6600" transparent depthWrite={false} />
    </mesh>
  );
}
