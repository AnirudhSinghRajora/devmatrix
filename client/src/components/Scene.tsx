import { Canvas } from '@react-three/fiber';
import { Stars, OrbitControls } from '@react-three/drei';
import { useGameStore } from '../store/gameStore';
import Ship from './Ship';

export default function Scene() {
  const entities = useGameStore((s) => s.entities);

  return (
    <Canvas
      camera={{ position: [0, 10, 20], fov: 60, near: 0.1, far: 1000 }}
      style={{ width: '100vw', height: '100vh', background: '#000' }}
    >
      <ambientLight intensity={0.3} />
      <directionalLight position={[10, 10, 5]} intensity={1} />
      <Stars radius={200} depth={60} count={2000} factor={4} fade />
      <OrbitControls enablePan enableZoom enableRotate />
      <gridHelper args={[50, 50, '#222', '#111']} />

      {Array.from(entities.values()).map((entity) => (
        <Ship key={entity.id} entity={entity} />
      ))}
    </Canvas>
  );
}
