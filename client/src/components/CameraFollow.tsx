import { useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';
import * as THREE from 'three';
import { useGameStore } from '../store/gameStore';
import type { OrbitControls as OrbitControlsImpl } from 'three-stdlib';

const _target = new THREE.Vector3();
const _prevCam = new THREE.Vector3();
const _currCam = new THREE.Vector3();
const _desiredPos = new THREE.Vector3();
const _camDelta = new THREE.Vector3();
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
    const t = elapsed / state.tickDuration;

    _prevCam.set(myEntity.prev.position[0], myEntity.prev.position[1], myEntity.prev.position[2]);
    _currCam.set(myEntity.curr.position[0], myEntity.curr.position[1], myEntity.curr.position[2]);

    // Snap if jump is too large.
    if (_prevCam.distanceToSquared(_currCam) > 2500) {
      _target.copy(_currCam);
    } else if (t <= 1.0) {
      _target.lerpVectors(_prevCam, _currCam, t);
    } else {
      // Dead-reckoning extrapolation (capped at 5 missed ticks).
      const extra = Math.min(t - 1.0, 5.0);
      _target.set(
        _currCam.x + (_currCam.x - _prevCam.x) * extra,
        _currCam.y + (_currCam.y - _prevCam.y) * extra,
        _currCam.z + (_currCam.z - _prevCam.z) * extra,
      );
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
      _camDelta.copy(camera.position).sub(controls.target);
      _desiredPos.copy(_target).add(_camDelta);
      camera.position.lerp(_desiredPos, FOLLOW_SPEED);
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
