import { useEffect, useState } from 'react';
import { connect, disconnect } from './network/socket';
import { getToken, clearToken } from './network/api';
import Scene from './components/Scene';
import PromptInput from './components/PromptInput';
import BehaviorIndicator from './components/BehaviorIndicator';
import KillFeed from './components/KillFeed';
import HealthPanel from './components/HealthPanel';
import LiveLeaderboard from './components/LiveLeaderboard';
import DeathScreen from './components/DeathScreen';
import AuthScreen from './components/AuthScreen';
import LobbyScreen from './components/LobbyScreen';
import Shop from './components/Shop';
import Leaderboard from './components/Leaderboard';
import ConnectionStatus from './components/ConnectionStatus';

function App() {
  const [authed, setAuthed] = useState<boolean | null>(null); // null = checking
  const [inGame, setInGame] = useState(false);
  const [launchHull, setLaunchHull] = useState<string | null>(null);
  const [showShop, setShowShop] = useState(false);
  const [showLeaderboard, setShowLeaderboard] = useState(false);

  // Check for existing token on mount.
  useEffect(() => {
    setAuthed(getToken() !== null);
  }, []);

  // Connect WS only after player launches from lobby.
  useEffect(() => {
    if (!inGame) return;
    connect(launchHull ?? undefined);
    return () => disconnect();
  }, [inGame]);

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

  // Show lobby if authed but not yet in-game.
  if (!inGame) {
    return <LobbyScreen onLaunch={(hullId) => { setLaunchHull(hullId); setInGame(true); }} />;
  }

  const isLoggedIn = getToken() !== null;

  return (
    <>
      <Scene />
      <ConnectionStatus />
      <LiveLeaderboard />

      {/* HUD buttons */}
      {isLoggedIn && (
        <div style={{
          position: 'absolute', bottom: 'max(12px, env(safe-area-inset-bottom, 0px))', left: 12,
          display: 'flex', gap: 6, pointerEvents: 'auto', flexWrap: 'wrap',
        }}>
          <button onClick={() => setShowShop(true)} style={hudBtnStyle}>SHOP</button>
          <button onClick={() => setShowLeaderboard(true)} style={hudBtnStyle}>LB</button>
          <button onClick={() => { clearToken(); setAuthed(false); setInGame(false); setLaunchHull(null); disconnect(); }} style={{
            ...hudBtnStyle, color: '#f44', borderColor: '#f44',
          }}>OUT</button>
        </div>
      )}

      <BehaviorIndicator />
      <HealthPanel />
      <KillFeed />
      <PromptInput />
      <DeathScreen />

      {showShop && <Shop onClose={() => setShowShop(false)} />}
      {showLeaderboard && <Leaderboard onClose={() => setShowLeaderboard(false)} />}
    </>
  );
}

const hudBtnStyle: React.CSSProperties = {
  background: 'rgba(0,0,0,0.7)', color: '#0f0', border: '1px solid #0f03',
  padding: '8px 14px', borderRadius: 4, cursor: 'pointer',
  fontFamily: 'monospace', fontSize: 12, fontWeight: 'bold',
  minHeight: 36, minWidth: 36,
};

export default App;
