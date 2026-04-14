import { useEffect, useState } from 'react';
import type { LeaderboardEntry } from '../types';
import { getLeaderboard } from '../network/api';

export default function Leaderboard({ onClose }: { onClose: () => void }) {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    getLeaderboard().then(setEntries).catch(() => setError('Failed to load leaderboard'));
  }, []);

  return (
    <div style={overlayStyle}>
      <div style={panelStyle}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 style={{ margin: 0, color: '#0f0', fontSize: 'clamp(16px, 4vw, 20px)' }}>LEADERBOARD</h2>
          <button onClick={onClose} style={closeBtnStyle}>X</button>
        </div>

        {error && <div style={{ color: '#f44', fontSize: 12, marginBottom: 8 }}>{error}</div>}

        <div style={tableHeaderStyle}>
          <span style={{ width: 28, flexShrink: 0 }}>#</span>
          <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>PLAYER</span>
          <span style={colNumStyle}>KILLS</span>
          <span style={colNumStyle}>DEATHS</span>
          <span style={{ width: 36, textAlign: 'right', flexShrink: 0 }}>TIER</span>
        </div>

        {entries.map((e, i) => (
          <div key={e.username} style={{
            display: 'flex', padding: '6px 8px', fontSize: 'clamp(11px, 2.5vw, 12px)',
            borderBottom: '1px solid #222',
            color: i < 3 ? '#ff0' : '#ddd',
          }}>
            <span style={{ width: 28, flexShrink: 0 }}>{i + 1}</span>
            <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.username}</span>
            <span style={colNumStyle}>{e.kills}</span>
            <span style={colNumStyle}>{e.deaths}</span>
            <span style={{ width: 36, textAlign: 'right', flexShrink: 0 }}>{e.ai_tier}</span>
          </div>
        ))}

        {entries.length === 0 && !error && (
          <div style={{ textAlign: 'center', color: '#666', padding: 24 }}>No entries yet</div>
        )}
      </div>
    </div>
  );
}

const overlayStyle: React.CSSProperties = {
  position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.85)',
  display: 'flex', justifyContent: 'center', alignItems: 'center',
  fontFamily: 'monospace', color: '#ddd', zIndex: 100,
  padding: 16, boxSizing: 'border-box',
};

const panelStyle: React.CSSProperties = {
  background: '#111', border: '1px solid #333', borderRadius: 8,
  padding: 'clamp(16px, 3vw, 24px)', width: '100%', maxWidth: 400,
  maxHeight: '85vh', overflowY: 'auto',
  boxSizing: 'border-box',
};

const closeBtnStyle: React.CSSProperties = {
  background: 'none', border: '1px solid #555', color: '#aaa', cursor: 'pointer',
  fontFamily: 'monospace', width: 36, height: 36, borderRadius: 4, fontSize: 14,
  flexShrink: 0,
};

const tableHeaderStyle: React.CSSProperties = {
  display: 'flex', padding: '6px 8px', borderBottom: '1px solid #333',
  fontSize: 'clamp(10px, 2.2vw, 11px)', color: '#888',
};

const colNumStyle: React.CSSProperties = {
  width: 'clamp(36px, 10vw, 60px)', textAlign: 'right', flexShrink: 0,
};
