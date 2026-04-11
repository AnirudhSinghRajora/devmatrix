import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface ExplosionProps {
  position: [number, number, number];
  time: number;
}

const EXPLOSION_DURATION = 0.8; // seconds
const PARTICLE_COUNT = 24;

export default function Explosion({ position, time }: ExplosionProps) {
  const coreRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);
  const lightRef = useRef<THREE.PointLight>(null);
  const pointsRef = useRef<THREE.Points>(null);

  // Generate random particle velocities once.
  const particleVelocities = useMemo(() => {
    const vels = new Float32Array(PARTICLE_COUNT * 3);
    for (let i = 0; i < PARTICLE_COUNT; i++) {
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      const speed = 3 + Math.random() * 5;
      vels[i * 3] = Math.sin(phi) * Math.cos(theta) * speed;
      vels[i * 3 + 1] = Math.sin(phi) * Math.sin(theta) * speed;
      vels[i * 3 + 2] = Math.cos(phi) * speed;
    }
    return vels;
  }, []);

  const particleGeo = useMemo(() => {
    const geo = new THREE.BufferGeometry();
    geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array(PARTICLE_COUNT * 3), 3));
    return geo;
  }, []);

  useFrame(() => {
    const elapsed = (performance.now() - time) / 1000;
    const t = elapsed / EXPLOSION_DURATION;
    if (t >= 1) {
      if (coreRef.current) coreRef.current.visible = false;
      if (ringRef.current) ringRef.current.visible = false;
      if (lightRef.current) lightRef.current.intensity = 0;
      if (pointsRef.current) pointsRef.current.visible = false;
      return;
    }

    // Core fireball: expand then fade.
    if (coreRef.current) {
      coreRef.current.scale.setScalar(1 + t * 6);
      (coreRef.current.material as THREE.MeshBasicMaterial).opacity = Math.max(0, 1 - t * 1.3);
      coreRef.current.visible = true;
    }

    // Shockwave ring: expand fast, fade quick.
    if (ringRef.current) {
      const ringScale = 1 + t * 15;
      ringRef.current.scale.set(ringScale, ringScale, 1);
      (ringRef.current.material as THREE.MeshBasicMaterial).opacity = Math.max(0, 1 - t * 2);
      ringRef.current.visible = t < 0.5;
    }

    // Flash light.
    if (lightRef.current) {
      lightRef.current.intensity = Math.max(0, 8 * (1 - t * 2));
    }

    // Particle debris.
    if (pointsRef.current) {
      const posAttr = particleGeo.attributes.position as THREE.BufferAttribute;
      const arr = posAttr.array as Float32Array;
      for (let i = 0; i < PARTICLE_COUNT; i++) {
        arr[i * 3] = particleVelocities[i * 3] * elapsed;
        arr[i * 3 + 1] = particleVelocities[i * 3 + 1] * elapsed;
        arr[i * 3 + 2] = particleVelocities[i * 3 + 2] * elapsed;
      }
      posAttr.needsUpdate = true;
      (pointsRef.current.material as THREE.PointsMaterial).opacity = Math.max(0, 1 - t);
      pointsRef.current.visible = true;
    }
  });

  return (
    <group position={position}>
      {/* Core fireball */}
      <mesh ref={coreRef}>
        <icosahedronGeometry args={[0.5, 1]} />
        <meshBasicMaterial
          color="#ff6600"
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
        />
      </mesh>

      {/* Shockwave ring */}
      <mesh ref={ringRef} rotation={[-Math.PI / 2, 0, 0]}>
        <ringGeometry args={[0.8, 1.0, 32]} />
        <meshBasicMaterial
          color="#ffaa00"
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
          side={THREE.DoubleSide}
        />
      </mesh>

      {/* Flash light */}
      <pointLight ref={lightRef} color="#ff8800" intensity={8} distance={30} decay={2} />

      {/* Particle debris */}
      <points ref={pointsRef} geometry={particleGeo}>
        <pointsMaterial
          size={0.3}
          color="#ffcc44"
          transparent
          depthWrite={false}
          blending={THREE.AdditiveBlending}
          sizeAttenuation
        />
      </points>
    </group>
  );
}
