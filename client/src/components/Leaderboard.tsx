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
    <div style={{
      position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.85)',
      display: 'flex', justifyContent: 'center', alignItems: 'center',
      fontFamily: 'monospace', color: '#ddd', zIndex: 100,
    }}>
      <div style={{
        background: '#111', border: '1px solid #333', borderRadius: 8,
        padding: 24, width: 400, maxHeight: '80vh', overflowY: 'auto',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <h2 style={{ margin: 0, color: '#0f0' }}>LEADERBOARD</h2>
          <button onClick={onClose} style={{
            background: 'none', border: '1px solid #555', color: '#aaa', cursor: 'pointer',
            fontFamily: 'monospace', width: 28, height: 28, borderRadius: 4,
          }}>X</button>
        </div>

        {error && <div style={{ color: '#f44', fontSize: 12, marginBottom: 8 }}>{error}</div>}

        <div style={{ display: 'flex', padding: '6px 8px', borderBottom: '1px solid #333', fontSize: 11, color: '#888' }}>
          <span style={{ width: 30 }}>#</span>
          <span style={{ flex: 1 }}>PLAYER</span>
          <span style={{ width: 60, textAlign: 'right' }}>KILLS</span>
          <span style={{ width: 60, textAlign: 'right' }}>DEATHS</span>
          <span style={{ width: 40, textAlign: 'right' }}>TIER</span>
        </div>

        {entries.map((e, i) => (
          <div key={e.username} style={{
            display: 'flex', padding: '6px 8px', fontSize: 12,
            borderBottom: '1px solid #222',
            color: i < 3 ? '#ff0' : '#ddd',
          }}>
            <span style={{ width: 30 }}>{i + 1}</span>
            <span style={{ flex: 1 }}>{e.username}</span>
            <span style={{ width: 60, textAlign: 'right' }}>{e.kills}</span>
            <span style={{ width: 60, textAlign: 'right' }}>{e.deaths}</span>
            <span style={{ width: 40, textAlign: 'right' }}>{e.ai_tier}</span>
          </div>
        ))}

        {entries.length === 0 && !error && (
          <div style={{ textAlign: 'center', color: '#666', padding: 24 }}>No entries yet</div>
        )}
      </div>
    </div>
  );
}
