import { useGameStore } from '../store/gameStore';
import type { ConnectionState } from '../store/gameStore';

const labels: Record<ConnectionState, string> = {
  connecting: 'CONNECTING',
  open: 'CONNECTED',
  reconnecting: 'RECONNECTING',
  disconnected: 'OFFLINE',
};

const colors: Record<ConnectionState, string> = {
  connecting: '#ffaa00',
  open: '#00ff88',
  reconnecting: '#ffaa00',
  disconnected: '#ff4444',
};

export default function ConnectionStatus() {
  const state = useGameStore((s) => s.connectionState);

  if (state === 'open') return null;

  return (
    <div style={{
      position: 'absolute',
      top: 12,
      left: '50%',
      transform: 'translateX(-50%)',
      background: 'var(--hud-bg)',
      border: `1px solid ${colors[state]}`,
      borderRadius: 'var(--hud-radius)',
      padding: '4px 14px',
      fontFamily: 'var(--hud-font)',
      fontSize: 11,
      color: colors[state],
      letterSpacing: 2,
      pointerEvents: 'none',
      zIndex: 60,
      animation: state === 'reconnecting' ? 'pulse 1.5s ease-in-out infinite' : undefined,
    }}>
      {labels[state]}
    </div>
  );
}
