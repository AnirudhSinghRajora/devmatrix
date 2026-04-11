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
  fontFamily: 'monospace',
  fontSize: 13,
  color: '#0f0',
  background: 'rgba(0, 0, 0, 0.5)',
  border: '1px solid rgba(0, 255, 0, 0.2)',
  borderRadius: 4,
  padding: '6px 10px',
  pointerEvents: 'none',
  lineHeight: 1.6,
};

const labelStyle: React.CSSProperties = {
  fontSize: 11,
  color: 'rgba(0, 255, 0, 0.5)',
  letterSpacing: 2,
  marginBottom: 2,
};

const emptyStyle: React.CSSProperties = {
  color: 'rgba(0, 255, 0, 0.35)',
  fontStyle: 'italic',
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  gap: 8,
};

const rowLabelStyle: React.CSSProperties = {
  color: 'rgba(0, 255, 0, 0.5)',
  minWidth: 30,
};

const rowValueStyle: React.CSSProperties = {
  color: '#0f0',
};
