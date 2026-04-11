import { useGameStore } from '../store/gameStore';

function BarRow({
  label,
  value,
  max,
  color,
}: {
  label: string;
  value: number;
  max: number;
  color: string;
}) {
  const pct = max > 0 ? (value / max) * 100 : 0;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
      <span style={{ width: 16, fontSize: 11, textAlign: 'right' }}>{label}</span>
      <div style={{ flex: 1, background: '#222', height: 6, borderRadius: 3, overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: 3, transition: 'width 0.1s' }} />
      </div>
      <span style={{ width: 42, fontSize: 11, textAlign: 'right' }}>
        {Math.ceil(value)}/{max}
      </span>
    </div>
  );
}

export default function HealthPanel() {
  const myPlayerId = useGameStore((s) => s.myPlayerId);
  const entities = useGameStore((s) => s.entities);

  if (!myPlayerId) return null;

  // Sort: own ship first, then others by id.
  const sorted = Array.from(entities.values()).sort((a, b) => {
    if (a.id === myPlayerId) return -1;
    if (b.id === myPlayerId) return 1;
    return a.id.localeCompare(b.id);
  });

  return (
    <div
      style={{
        position: 'absolute',
        top: 12,
        right: 12,
        color: '#ddd',
        fontFamily: 'monospace',
        fontSize: 12,
        pointerEvents: 'none',
        width: 180,
      }}
    >
      {sorted.map((e) => {
        const isOwn = e.id === myPlayerId;
        return (
          <div
            key={e.id}
            style={{
              background: 'rgba(0,0,0,0.6)',
              borderRadius: 4,
              padding: '4px 8px',
              marginBottom: 4,
              border: isOwn ? '1px solid #0f0' : '1px solid transparent',
              opacity: e.alive ? 1 : 0.35,
            }}
          >
            <div style={{ fontSize: 11, marginBottom: 2, color: isOwn ? '#0f0' : '#aaa' }}>
              {isOwn ? 'YOU' : (e.username || e.id).slice(0, 12)}
              {!e.alive && <span style={{ color: '#f44', marginLeft: 4 }}>DEAD</span>}
            </div>
            <BarRow label="HP" value={e.health} max={e.maxHealth} color="#0f0" />
            <BarRow label="SH" value={e.shield} max={e.maxShield} color="#0af" />
          </div>
        );
      })}
    </div>
  );
}
