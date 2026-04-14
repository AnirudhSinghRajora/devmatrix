import { useState, useEffect, useRef, useMemo, Suspense } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { useGLTF, OrbitControls, Environment } from '@react-three/drei';
import * as THREE from 'three';
import { getToken, getProfile, equipItem } from '../network/api';
import type { PlayerProfile } from '../types';
import Shop from './Shop';

const HULLS = [
  {
    id: 'hull_basic', name: 'Striker', model: '/models/ships/Striker.glb',
    health: 100, speed: 50, thrust: 40, tier: 1, price: 0,
    desc: 'Light and nimble scout craft.',
  },
  {
    id: 'hull_medium', name: 'Challenger', model: '/models/ships/Challenger.glb',
    health: 150, speed: 35, thrust: 30, tier: 2, price: 600,
    desc: 'Balanced cruiser with solid defenses.',
  },
  {
    id: 'hull_heavy', name: 'Imperial', model: '/models/ships/Imperial.glb',
    health: 250, speed: 25, thrust: 20, tier: 3, price: 1500,
    desc: 'A fortress in space.',
  },
  {
    id: 'hull_stealth', name: 'Omen', model: '/models/ships/Omen.glb',
    health: 80, speed: 60, thrust: 50, tier: 4, price: 1800,
    desc: 'Fragile but incredibly fast.',
  },
];

// Preload all.
HULLS.forEach((h) => useGLTF.preload(h.model));

function RotatingShip({ model }: { model: string }) {
  const { scene } = useGLTF(model);
  const ref = useRef<THREE.Group>(null!);

  // Re-clone whenever the source scene (model) changes.
  const cloned = useMemo(() => scene.clone(true), [scene]);

  useFrame((_, delta) => {
    if (ref.current) ref.current.rotation.y += delta * 0.4;
  });

  return (
    <group ref={ref}>
      <primitive object={cloned} scale={0.8} />
    </group>
  );
}

interface Props {
  onLaunch: (hullId: string) => void;
}

export default function LobbyScreen({ onLaunch }: Props) {
  const isLoggedIn = getToken() !== null;
  const [profile, setProfile] = useState<PlayerProfile | null>(null);
  const [selected, setSelected] = useState(0); // index into HULLS
  const [equipping, setEquipping] = useState(false);
  const [showShop, setShowShop] = useState(false);

  const refreshProfile = () => {
    if (!isLoggedIn) return;
    getProfile().then((p) => {
      setProfile(p);
      const idx = HULLS.findIndex((h) => h.id === p.loadout.hull);
      if (idx >= 0) setSelected(idx);
    }).catch(() => {});
  };

  // Load profile for logged-in users.
  useEffect(() => {
    if (!isLoggedIn) return;
    refreshProfile();
  }, [isLoggedIn]);

  const hull = HULLS[selected];
  const owned = !isLoggedIn || profile?.inventory.includes(hull.id);
  const equipped = profile?.loadout.hull === hull.id;

  const handleSelect = async (idx: number) => {
    setSelected(idx);
    const h = HULLS[idx];
    // Auto-equip for logged-in users if they own it.
    if (isLoggedIn && profile?.inventory.includes(h.id) && profile.loadout.hull !== h.id) {
      setEquipping(true);
      try {
        await equipItem(h.id, 'hull');
        setProfile((p) => p ? { ...p, loadout: { ...p.loadout, hull: h.id } } : p);
      } catch { /* ignore */ }
      setEquipping(false);
    }
  };

  const canLaunch = !isLoggedIn || owned;

  return (
    <div style={wrapperStyle}>
      {/* Background stars */}
      <div style={starsStyle} />

      {/* Title */}
      <div style={titleAreaStyle}>
        <h1 style={titleStyle}>DEVMATRIX</h1>
        <p style={subtitleStyle}>SELECT YOUR SHIP</p>
      </div>

      {/* Ship selector cards */}
      <div style={selectorRowStyle}>
        {HULLS.map((h, i) => {
          const isOwned = !isLoggedIn || profile?.inventory.includes(h.id);
          const isSelected = i === selected;
          return (
            <button
              key={h.id}
              onClick={() => handleSelect(i)}
              style={{
                ...cardStyle,
                borderColor: isSelected ? '#00ccff' : '#333',
                background: isSelected ? 'rgba(0, 204, 255, 0.08)' : 'rgba(0, 0, 0, 0.6)',
              }}
            >
              <div style={{ fontSize: 14, fontWeight: 'bold', color: isSelected ? '#00ccff' : '#ccc' }}>
                {h.name}
              </div>
              <div style={{ fontSize: 10, color: isOwned ? '#888' : '#ff4444', marginTop: 2 }}>
                {!isOwned ? `🔒 ${h.price} coins` : isSelected ? 'SELECTED' : ''}
              </div>
            </button>
          );
        })}
      </div>

      {/* Main layout: 3D preview + stats */}
      <div style={mainStyle}>
        {/* 3D Preview */}
        <div style={previewStyle}>
          <Canvas camera={{ position: [0, 5, 20], fov: 30 }}>
            <ambientLight intensity={0.5} />
            <directionalLight position={[5, 5, 5]} intensity={1} />
            <pointLight position={[-5, 2, -3]} intensity={0.5} color="#00ccff" />
            <Suspense fallback={null}>
              <RotatingShip key={hull.id} model={hull.model} />
              <Environment preset="night" />
            </Suspense>
            <OrbitControls
              enablePan={false} enableZoom={true}
              minDistance={8} maxDistance={40}
              minPolarAngle={Math.PI / 4} maxPolarAngle={Math.PI / 1.8}
              autoRotate={false}
            />
          </Canvas>
        </div>

        {/* Stats panel */}
        <div style={statsPanelStyle}>
          <h2 style={shipNameStyle}>{hull.name}</h2>
          <p style={descStyle}>{hull.desc}</p>

          <div style={statsGridStyle}>
            <StatBar label="HULL" value={hull.health} max={250} color="#00ff88" />
            <StatBar label="SPEED" value={hull.speed} max={60} color="#00ccff" />
            <StatBar label="THRUST" value={hull.thrust} max={50} color="#ffaa00" />
          </div>

          {isLoggedIn && !owned && (
            <div style={{ color: '#ff4444', fontSize: 12, marginTop: 12 }}>
              Purchase this hull from the in-game SHOP
            </div>
          )}

          {isLoggedIn && owned && equipped && (
            <div style={{ color: '#00ff88', fontSize: 11, marginTop: 12 }}>EQUIPPED</div>
          )}
          {equipping && (
            <div style={{ color: '#888', fontSize: 11, marginTop: 12 }}>Equipping...</div>
          )}
        </div>
      </div>

      {/* Launch button + Shop */}
      <div style={{ display: 'flex', gap: 12, alignItems: 'center', zIndex: 1 }}>
        {isLoggedIn && (
          <button onClick={() => setShowShop(true)} style={shopBtnStyle}>SHOP</button>
        )}
        <button
          onClick={() => onLaunch(hull.id)}
          disabled={!canLaunch}
          style={{
            ...launchBtnStyle,
            opacity: canLaunch ? 1 : 0.4,
            cursor: canLaunch ? 'pointer' : 'not-allowed',
          }}
        >
          LAUNCH
        </button>
      </div>

      {/* Profile info */}
      {isLoggedIn && profile && (
        <div style={profileInfoStyle}>
          {profile.username} &mdash; {profile.coins} coins &bull; Tier {profile.ai_tier}
        </div>
      )}
      {!isLoggedIn && (
        <div style={profileInfoStyle}>Playing as Guest</div>
      )}

      {showShop && <Shop onClose={() => { setShowShop(false); refreshProfile(); }} />}
    </div>
  );
}

function StatBar({ label, value, max, color }: { label: string; value: number; max: number; color: string }) {
  const pct = Math.min(100, (value / max) * 100);
  return (
    <div style={{ marginBottom: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, marginBottom: 3 }}>
        <span style={{ color: '#888' }}>{label}</span>
        <span style={{ color }}>{value}</span>
      </div>
      <div style={barBgStyle}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: 2, transition: 'width 0.3s' }} />
      </div>
    </div>
  );
}

/* ── Styles ── */

const wrapperStyle: React.CSSProperties = {
  position: 'absolute', inset: 0,
  background: '#0a0a0f',
  display: 'flex', flexDirection: 'column',
  alignItems: 'center', justifyContent: 'center',
  fontFamily: 'var(--hud-font, monospace)', color: '#ddd',
  overflow: 'hidden',
};

const starsStyle: React.CSSProperties = {
  position: 'absolute', inset: 0,
  background: 'radial-gradient(1px 1px at 20% 30%, #fff3 0%, transparent 100%), radial-gradient(1px 1px at 80% 70%, #fff2 0%, transparent 100%), radial-gradient(1px 1px at 50% 50%, #fff1 0%, transparent 100%)',
  opacity: 0.4,
};

const titleAreaStyle: React.CSSProperties = {
  textAlign: 'center', marginBottom: 24, zIndex: 1,
};

const titleStyle: React.CSSProperties = {
  fontSize: 36, fontWeight: 'bold', letterSpacing: 12,
  color: '#00ccff',
  textShadow: '0 0 20px rgba(0, 204, 255, 0.4)',
  margin: 0,
};

const subtitleStyle: React.CSSProperties = {
  fontSize: 12, color: '#666', letterSpacing: 6, marginTop: 8,
};

const selectorRowStyle: React.CSSProperties = {
  display: 'flex', gap: 8, marginBottom: 20, zIndex: 1,
};

const cardStyle: React.CSSProperties = {
  padding: '10px 20px', borderRadius: 6,
  border: '1px solid #333',
  cursor: 'pointer', fontFamily: 'inherit',
  transition: 'border-color 0.2s, background 0.2s',
  textAlign: 'center', minWidth: 90,
};

const mainStyle: React.CSSProperties = {
  display: 'flex', gap: 32, alignItems: 'center',
  zIndex: 1, marginBottom: 24,
};

const previewStyle: React.CSSProperties = {
  width: 400, height: 300,
  borderRadius: 8,
  overflow: 'hidden',
  border: '1px solid #222',
  background: 'rgba(0, 0, 0, 0.4)',
};

const statsPanelStyle: React.CSSProperties = {
  width: 220,
};

const shipNameStyle: React.CSSProperties = {
  margin: '0 0 6px 0', fontSize: 22, color: '#fff',
  letterSpacing: 3,
};

const descStyle: React.CSSProperties = {
  fontSize: 12, color: '#888', margin: '0 0 20px 0', lineHeight: 1.5,
};

const statsGridStyle: React.CSSProperties = {};

const barBgStyle: React.CSSProperties = {
  width: '100%', height: 4, background: '#222', borderRadius: 2, overflow: 'hidden',
};

const launchBtnStyle: React.CSSProperties = {
  padding: '14px 60px', fontSize: 16, fontWeight: 'bold',
  fontFamily: 'inherit', letterSpacing: 6,
  color: '#000', background: '#00ccff',
  border: 'none', borderRadius: 6,
  boxShadow: '0 0 20px rgba(0, 204, 255, 0.3)',
  transition: 'transform 0.15s, box-shadow 0.15s',
  zIndex: 1,
};

const shopBtnStyle: React.CSSProperties = {
  padding: '14px 30px', fontSize: 14, fontWeight: 'bold',
  fontFamily: 'inherit', letterSpacing: 4,
  color: '#ffaa00', background: 'transparent',
  border: '1px solid #ffaa0055', borderRadius: 6,
  cursor: 'pointer',
  transition: 'border-color 0.2s',
};

const profileInfoStyle: React.CSSProperties = {
  position: 'absolute', bottom: 16,
  fontSize: 11, color: '#555', letterSpacing: 2,
};
