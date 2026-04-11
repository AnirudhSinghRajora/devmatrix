import { useGameStore } from '../store/gameStore';

export default function KillFeed() {
  const killFeed = useGameStore((s) => s.killFeed);

  if (killFeed.length === 0) return null;

  return (
    <div
      style={{
        position: 'absolute',
        bottom: 80,
        right: 12,
        color: 'var(--hud-text)',
        fontFamily: 'var(--hud-font)',
        fontSize: 12,
        textAlign: 'right',
        pointerEvents: 'none',
      }}
    >
      {killFeed.slice(-5).map((entry) => (
        <div
          key={entry.id}
          style={{
            padding: '3px 10px',
            marginBottom: 3,
            background: 'var(--hud-bg)',
            borderRadius: 'var(--hud-radius)',
            border: '1px solid var(--hud-border)',
          }}
        >
          <span style={{ color: 'var(--hud-red)' }}>{entry.killerName || entry.killer.slice(0, 8)}</span>
          {' \u26A1 '}
          <span style={{ color: 'var(--hud-text-dim)' }}>{entry.victimName || entry.victim.slice(0, 8)}</span>
        </div>
      ))}
    </div>
  );
}
