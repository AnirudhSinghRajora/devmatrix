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
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 3 }}>
      <span style={{ width: 18, fontSize: 10, textAlign: 'right', color: 'var(--hud-text-dim)', letterSpacing: 1 }}>{label}</span>
      <div style={{ flex: 1, background: 'rgba(0,200,255,0.08)', height: 5, borderRadius: 3, overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: 3, transition: 'width 0.1s', boxShadow: `0 0 6px ${color}` }} />
      </div>
      <span style={{ width: 44, fontSize: 10, textAlign: 'right', color: 'var(--hud-text-dim)' }}>
        {Math.ceil(value)}/{max}
      </span>
    </div>
  );
}

export default function HealthPanel() {
  const myPlayerId = useGameStore((s) => s.myPlayerId);
  const entities = useGameStore((s) => s.entities);

  if (!myPlayerId) return null;

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
        color: 'var(--hud-text)',
        fontFamily: 'var(--hud-font)',
        fontSize: 12,
        pointerEvents: 'none',
        width: 190,
      }}
    >
      {sorted.map((e) => {
        const isOwn = e.id === myPlayerId;
        return (
          <div
            key={e.id}
            style={{
              background: 'var(--hud-bg)',
              borderRadius: 'var(--hud-radius)',
              padding: '5px 10px',
              marginBottom: 4,
              border: isOwn ? '1px solid var(--hud-accent)' : '1px solid var(--hud-border)',
              boxShadow: isOwn ? 'var(--hud-glow)' : 'none',
              opacity: e.alive ? 1 : 0.3,
            }}
          >
            <div style={{ fontSize: 10, marginBottom: 3, color: isOwn ? 'var(--hud-accent)' : 'var(--hud-text-dim)', letterSpacing: 1 }}>
              {isOwn ? '▸ YOU' : (e.username || e.id).slice(0, 12)}
              {!e.alive && <span style={{ color: 'var(--hud-red)', marginLeft: 6 }}>DEAD</span>}
            </div>
            <BarRow label="HP" value={e.health} max={e.maxHealth} color="var(--hud-green)" />
            <BarRow label="SH" value={e.shield} max={e.maxShield} color="var(--hud-accent)" />
          </div>
        );
      })}
    </div>
  );
}
