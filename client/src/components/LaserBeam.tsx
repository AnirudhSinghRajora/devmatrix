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
  const coreRef = useRef<THREE.Mesh>(null);
  const glowRef = useRef<THREE.Mesh>(null);

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
    const core = coreRef.current;
    const glow = glowRef.current;
    if (!core || !glow) return;
    const elapsed = (performance.now() - time) / 1000;
    const opacity = Math.max(0, 1 - elapsed / LASER_DURATION);
    (core.material as THREE.MeshBasicMaterial).opacity = opacity;
    (glow.material as THREE.MeshBasicMaterial).opacity = opacity * 0.4;
    core.visible = opacity > 0;
    glow.visible = opacity > 0;
  });

  const coreColor = hit ? '#ff4444' : '#ff8844';
  const glowColor = hit ? '#ff0000' : '#ff6622';

  return (
    <group position={midpoint} quaternion={quaternion}>
      {/* Inner bright core */}
      <mesh ref={coreRef}>
        <cylinderGeometry args={[0.05, 0.05, length, 6]} />
        <meshBasicMaterial
          color={coreColor}
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </mesh>
      {/* Outer glow */}
      <mesh ref={glowRef}>
        <cylinderGeometry args={[0.2, 0.2, length, 6]} />
        <meshBasicMaterial
          color={glowColor}
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </mesh>
    </group>
  );
}
