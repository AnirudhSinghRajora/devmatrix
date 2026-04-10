import { useEffect } from 'react';
import { connect, disconnect } from './network/socket';
import { useGameStore } from './store/gameStore';
import Scene from './components/Scene';

function App() {
  const connected = useGameStore((s) => s.connected);
  const tick = useGameStore((s) => s.tick);

  useEffect(() => {
    connect();
    return () => disconnect();
  }, []);

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
        }}
      >
        <div>{connected ? '● CONNECTED' : '○ DISCONNECTED'}</div>
        <div>Tick: {tick}</div>
      </div>
    </>
  );
}

export default App;
