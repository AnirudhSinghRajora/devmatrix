import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface LaserBeamProps {
  from: [number, number, number];
  to: [number, number, number];
  hit: boolean;
  time: number;
}

const LASER_DURATION = 0.3; // seconds

export default function LaserBeam({ from, to, hit, time }: LaserBeamProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  const { midpoint, length, quaternion } = useMemo(() => {
    const start = new THREE.Vector3(...from);
    const end = new THREE.Vector3(...to);
    const mid = start.clone().add(end).multiplyScalar(0.5);
    const dir = end.clone().sub(start);
    const len = dir.length();
    const q = new THREE.Quaternion();
    q.setFromUnitVectors(new THREE.Vector3(0, 1, 0), dir.clone().normalize());
    return { midpoint: mid, length: len, quaternion: q };
  }, [from, to]);

  useFrame(() => {
    const mesh = meshRef.current;
    if (!mesh) return;
    const elapsed = (performance.now() - time) / 1000;
    const opacity = Math.max(0, 1 - elapsed / LASER_DURATION);
    const mat = mesh.material as THREE.MeshBasicMaterial;
    mat.opacity = opacity;
    mesh.visible = opacity > 0;
  });

  return (
    <mesh ref={meshRef} position={midpoint} quaternion={quaternion}>
      <cylinderGeometry args={[0.06, 0.06, length, 4]} />
      <meshBasicMaterial
        color={hit ? '#ff2222' : '#ff6644'}
        transparent
        depthWrite={false}
      />
    </mesh>
  );
}
