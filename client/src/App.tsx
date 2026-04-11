import { useEffect, useState } from 'react';
import { connect, disconnect } from './network/socket';
import { getToken, clearToken } from './network/api';
import { useGameStore } from './store/gameStore';
import Scene from './components/Scene';
import PromptInput from './components/PromptInput';
import BehaviorIndicator from './components/BehaviorIndicator';
import KillFeed from './components/KillFeed';
import HealthPanel from './components/HealthPanel';
import AuthScreen from './components/AuthScreen';
import Shop from './components/Shop';
import Leaderboard from './components/Leaderboard';

function App() {
  const connected = useGameStore((s) => s.connected);
  const tick = useGameStore((s) => s.tick);
  const myPlayerId = useGameStore((s) => s.myPlayerId);
  const entityCount = useGameStore((s) => s.entities.size);

  const [authed, setAuthed] = useState<boolean | null>(null); // null = checking
  const [showShop, setShowShop] = useState(false);
  const [showLeaderboard, setShowLeaderboard] = useState(false);

  // Check for existing token on mount.
  useEffect(() => {
    setAuthed(getToken() !== null);
  }, []);

  // Connect WS once authed or skipped.
  useEffect(() => {
    if (authed === null) return; // still checking
    connect();
    return () => disconnect();
  }, [authed]);

  // Show auth screen if not yet decided.
  if (authed === null) return null;
  if (authed === false) {
    return (
      <AuthScreen
        onAuth={() => setAuthed(true)}
        onSkip={() => setAuthed(true)}
      />
    );
  }

  const isLoggedIn = getToken() !== null;

  return (
    <>
      <Scene />
      <div
        style={{
          position: 'absolute',
          top: 12,
          left: 12,
          color: '#0f0',
          fontFamily: 'monospace',
          fontSize: 14,
          pointerEvents: 'none',
          lineHeight: 1.6,
        }}
      >
        <div>{connected ? '● CONNECTED' : '○ DISCONNECTED'}</div>
        {myPlayerId && <div>ID: {myPlayerId}</div>}
        <div>Tick: {tick}</div>
        <div>Ships: {entityCount}</div>
      </div>

      {/* HUD buttons */}
      {isLoggedIn && (
        <div style={{
          position: 'absolute', bottom: 12, left: 12,
          display: 'flex', gap: 8, pointerEvents: 'auto',
        }}>
          <button onClick={() => setShowShop(true)} style={hudBtnStyle}>SHOP</button>
          <button onClick={() => setShowLeaderboard(true)} style={hudBtnStyle}>LEADERBOARD</button>
          <button onClick={() => { clearToken(); setAuthed(false); disconnect(); }} style={{
            ...hudBtnStyle, color: '#f44', borderColor: '#f44',
          }}>LOGOUT</button>
        </div>
      )}

      <BehaviorIndicator />
      <HealthPanel />
      <KillFeed />
      <PromptInput />

      {showShop && <Shop onClose={() => setShowShop(false)} />}
      {showLeaderboard && <Leaderboard onClose={() => setShowLeaderboard(false)} />}
    </>
  );
}

const hudBtnStyle: React.CSSProperties = {
  background: 'rgba(0,0,0,0.7)', color: '#0f0', border: '1px solid #0f03',
  padding: '6px 14px', borderRadius: 4, cursor: 'pointer',
  fontFamily: 'monospace', fontSize: 12, fontWeight: 'bold',
};

export default App;
