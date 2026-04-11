import { useState, useEffect } from 'react';
import { useGameStore } from '../store/gameStore';

const RESPAWN_TIME = 5; // must match server's RespawnTimer (5.0s)

export default function DeathScreen() {
  const myDeathTime = useGameStore((s) => s.myDeathTime);
  const [countdown, setCountdown] = useState(RESPAWN_TIME);

  useEffect(() => {
    if (myDeathTime === null) {
      setCountdown(RESPAWN_TIME);
      return;
    }

    // Start counting down from 5.
    const tick = () => {
      const elapsed = (performance.now() - myDeathTime) / 1000;
      const remaining = Math.max(0, RESPAWN_TIME - elapsed);
      setCountdown(remaining);
    };

    tick();
    const id = setInterval(tick, 50);
    return () => clearInterval(id);
  }, [myDeathTime]);

  if (myDeathTime === null) return null;

  const isRespawning = countdown <= 0;

  return (
    <div style={overlayStyle}>
      <div style={contentStyle}>
        <div style={titleStyle}>YOU DIED</div>
        <div style={subtitleStyle}>
          {isRespawning ? 'Respawning...' : `Respawning in ${Math.ceil(countdown)}s`}
        </div>
        <div style={barContainerStyle}>
          <div
            style={{
              ...barFillStyle,
              width: `${Math.min(100, ((RESPAWN_TIME - countdown) / RESPAWN_TIME) * 100)}%`,
            }}
          />
        </div>
        <div style={tipStyle}>Your behavior will be cleared — issue new orders after respawn</div>
      </div>
    </div>
  );
}

const overlayStyle: React.CSSProperties = {
  position: 'absolute',
  inset: 0,
  display: 'flex',
  justifyContent: 'center',
  alignItems: 'center',
  background: 'rgba(0, 0, 0, 0.6)',
  zIndex: 50,
  pointerEvents: 'none',
};

const contentStyle: React.CSSProperties = {
  textAlign: 'center',
  fontFamily: 'var(--hud-font)',
};

const titleStyle: React.CSSProperties = {
  fontSize: 48,
  fontWeight: 'bold',
  color: 'var(--hud-red)',
  letterSpacing: 8,
  textShadow: '0 0 30px rgba(255, 68, 68, 0.6), 0 0 60px rgba(255, 68, 68, 0.3)',
  marginBottom: 16,
};

const subtitleStyle: React.CSSProperties = {
  fontSize: 18,
  color: 'var(--hud-text)',
  letterSpacing: 2,
  marginBottom: 20,
};

const barContainerStyle: React.CSSProperties = {
  width: 240,
  height: 4,
  background: 'rgba(255, 255, 255, 0.1)',
  borderRadius: 2,
  overflow: 'hidden',
  margin: '0 auto 24px auto',
};

const barFillStyle: React.CSSProperties = {
  height: '100%',
  background: 'var(--hud-red)',
  borderRadius: 2,
  transition: 'width 0.05s linear',
  boxShadow: '0 0 8px rgba(255, 68, 68, 0.5)',
};

const tipStyle: React.CSSProperties = {
  fontSize: 12,
  color: 'var(--hud-text-dim)',
  letterSpacing: 1,
};
