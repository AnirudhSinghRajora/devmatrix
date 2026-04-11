import { useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';
import type { OrbitControls as OrbitControlsImpl } from 'three-stdlib';

const _target = new THREE.Vector3();
const _prevCam = new THREE.Vector3();
const _currCam = new THREE.Vector3();
const CAMERA_OFFSET = new THREE.Vector3(0, 12, 25);
const FOLLOW_SPEED = 0.05;

export default function CameraFollow() {
  const controlsRef = useRef<OrbitControlsImpl>(null);
  const { camera } = useThree();
  const initialized = useRef(false);

  useFrame(() => {
    const state = useGameStore.getState();
    const myId = state.myPlayerId;
    if (!myId) return;

    const myEntity = state.entities.get(myId);
    if (!myEntity) return;

    // Interpolated position of our ship (same logic as Ship.tsx).
    const elapsed = performance.now() - state.lastUpdateTime;
    const t = Math.min(elapsed / state.tickDuration, 1.0);

    _prevCam.set(myEntity.prev.position[0], myEntity.prev.position[1], myEntity.prev.position[2]);
    _currCam.set(myEntity.curr.position[0], myEntity.curr.position[1], myEntity.curr.position[2]);

    // Snap if jump is too large.
    if (_prevCam.distanceToSquared(_currCam) > 2500) {
      _target.copy(_currCam);
    } else {
      _target.lerpVectors(_prevCam, _currCam, t);
    }

    // On first frame, snap camera into position.
    if (!initialized.current) {
      camera.position.copy(_target).add(CAMERA_OFFSET);
      controlsRef.current?.target.copy(_target);
      controlsRef.current?.update();
      initialized.current = true;
      return;
    }

    // Smoothly move the orbit target to follow our ship.
    const controls = controlsRef.current;
    if (controls) {
      controls.target.lerp(_target, FOLLOW_SPEED);
      // Shift camera by the same delta to keep relative orbit angle.
      const desiredPos = _target.clone().add(
        camera.position.clone().sub(controls.target),
      );
      camera.position.lerp(desiredPos, FOLLOW_SPEED);
      controls.update();
    }
  });

  return (
    <OrbitControls
      ref={controlsRef}
      enablePan={false}
      enableZoom
      enableRotate
      minDistance={5}
      maxDistance={80}
      maxPolarAngle={Math.PI * 0.85}
    />
  );
}
