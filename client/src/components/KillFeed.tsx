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
        color: '#fff',
        fontFamily: 'monospace',
        fontSize: 13,
        textAlign: 'right',
        pointerEvents: 'none',
      }}
    >
      {killFeed.slice(-5).map((entry) => (
        <div
          key={entry.id}
          style={{
            padding: '2px 8px',
            marginBottom: 2,
            background: 'rgba(0,0,0,0.6)',
            borderRadius: 3,
          }}
        >
          <span style={{ color: '#f55' }}>{entry.killerName || entry.killer.slice(0, 8)}</span>
          {' \u26A1 '}
          <span style={{ color: '#aaa' }}>{entry.victimName || entry.victim.slice(0, 8)}</span>
        </div>
      ))}
    </div>
  );
}
