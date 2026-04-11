import { useGameStore } from '../store/gameStore';

export default function BehaviorIndicator() {
  const behavior = useGameStore((s) => s.currentBehavior);

  return (
    <div style={containerStyle}>
      <div style={labelStyle}>BEHAVIOR</div>
      {behavior ? (
        <>
          <Row label="MOV" value={behavior.movement} />
          <Row label="CMB" value={behavior.combat} />
          <Row label="DEF" value={behavior.defense} />
        </>
      ) : (
        <div style={emptyStyle}>No orders</div>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div style={rowStyle}>
      <span style={rowLabelStyle}>{label}</span>
      <span style={rowValueStyle}>{value}</span>
    </div>
  );
}

// ---------- inline styles ----------

const containerStyle: React.CSSProperties = {
  position: 'absolute',
  bottom: 80,
  left: 12,
  fontFamily: 'var(--hud-font)',
  fontSize: 13,
  color: 'var(--hud-accent)',
  background: 'var(--hud-bg)',
  border: '1px solid var(--hud-border)',
  borderRadius: 'var(--hud-radius)',
  padding: '6px 12px',
  pointerEvents: 'none',
  lineHeight: 1.6,
  boxShadow: 'var(--hud-glow)',
};

const labelStyle: React.CSSProperties = {
  fontSize: 10,
  color: 'var(--hud-text-dim)',
  letterSpacing: 3,
  marginBottom: 2,
  textTransform: 'uppercase',
};

const emptyStyle: React.CSSProperties = {
  color: 'var(--hud-text-dim)',
  fontStyle: 'italic',
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  gap: 8,
};

const rowLabelStyle: React.CSSProperties = {
  color: 'var(--hud-text-dim)',
  minWidth: 30,
};

const rowValueStyle: React.CSSProperties = {
  color: 'var(--hud-accent)',
};
