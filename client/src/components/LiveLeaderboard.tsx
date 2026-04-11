import { useRef, useEffect, useState, useCallback } from 'react';
import { useGameStore } from '../store/gameStore';

interface LeaderboardRow {
  id: string;
  name: string;
  kills: number;
  deaths: number;
  isOwn: boolean;
  alive: boolean;
}

const ROW_HEIGHT = 32;

export default function LiveLeaderboard() {
  const myPlayerId = useGameStore((s) => s.myPlayerId);
  const entities = useGameStore((s) => s.entities);
  const [prevOrder, setPrevOrder] = useState<string[]>([]);
  const [animOffsets, setAnimOffsets] = useState<Map<string, number>>(new Map());
  const animFrameRef = useRef<number>(0);

  // Build sorted rows.
  const rows: LeaderboardRow[] = [];
  entities.forEach((e) => {
    rows.push({
      id: e.id,
      name: e.username || e.id.slice(0, 8),
      kills: e.kills,
      deaths: e.deaths,
      isOwn: e.id === myPlayerId,
      alive: e.alive,
    });
  });
  rows.sort((a, b) => {
    if (b.kills !== a.kills) return b.kills - a.kills;
    if (a.deaths !== b.deaths) return a.deaths - b.deaths;
    return a.name.localeCompare(b.name);
  });

  const currentOrder = rows.map((r) => r.id);

  // Detect position changes and trigger animation.
  const startAnimation = useCallback(() => {
    if (prevOrder.length === 0) {
      setPrevOrder(currentOrder);
      return;
    }

    const offsets = new Map<string, number>();
    let hasChange = false;

    for (let i = 0; i < currentOrder.length; i++) {
      const id = currentOrder[i];
      const oldIndex = prevOrder.indexOf(id);
      if (oldIndex !== -1 && oldIndex !== i) {
        offsets.set(id, (oldIndex - i) * ROW_HEIGHT);
        hasChange = true;
      }
    }

    if (hasChange) {
      setAnimOffsets(offsets);
      // Animate to 0 over ~350ms.
      const startTime = performance.now();
      const duration = 350;

      const animate = () => {
        const elapsed = performance.now() - startTime;
        const t = Math.min(elapsed / duration, 1);
        // Ease-out cubic.
        const ease = 1 - Math.pow(1 - t, 3);

        const updated = new Map<string, number>();
        offsets.forEach((startOffset, id) => {
          const current = startOffset * (1 - ease);
          if (Math.abs(current) > 0.5) {
            updated.set(id, current);
          }
        });

        setAnimOffsets(updated);
        if (t < 1) {
          animFrameRef.current = requestAnimationFrame(animate);
        }
      };

      cancelAnimationFrame(animFrameRef.current);
      animFrameRef.current = requestAnimationFrame(animate);
    }

    setPrevOrder(currentOrder);
  }, [currentOrder, prevOrder]);

  useEffect(() => {
    startAnimation();
  }, [entities]);

  useEffect(() => {
    return () => cancelAnimationFrame(animFrameRef.current);
  }, []);

  if (rows.length === 0) return null;

  return (
    <div style={containerStyle}>
      <div style={headerStyle}>LEADERBOARD</div>
      <div style={tableHeaderStyle}>
        <span style={{ flex: 1 }}>PLAYER</span>
        <span style={colStyle}>K</span>
        <span style={colStyle}>D</span>
      </div>
      <div style={{ position: 'relative' }}>
        {rows.map((row, index) => {
          const offset = animOffsets.get(row.id) ?? 0;
          const isSwapping = Math.abs(offset) > 1;
          return (
            <div
              key={row.id}
              style={{
                ...rowStyle,
                transform: `translateY(${offset}px)`,
                transition: isSwapping ? 'none' : undefined,
                borderLeft: row.isOwn ? '2px solid var(--hud-accent)' : '2px solid transparent',
                background: row.isOwn
                  ? 'rgba(0, 200, 255, 0.08)'
                  : index % 2 === 0
                    ? 'rgba(0, 8, 16, 0.3)'
                    : 'transparent',
                opacity: row.alive ? 1 : 0.4,
              }}
            >
              <span style={{
                flex: 1,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                color: row.isOwn ? 'var(--hud-accent)' : 'var(--hud-text)',
              }}>
                {index === 0 && row.kills > 0 ? '👑 ' : ''}{row.name}
              </span>
              <span style={{ ...colStyle, color: 'var(--hud-green)' }}>{row.kills}</span>
              <span style={{ ...colStyle, color: 'var(--hud-red)' }}>{row.deaths}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const containerStyle: React.CSSProperties = {
  position: 'absolute',
  top: 12,
  left: 12,
  fontFamily: 'var(--hud-font)',
  fontSize: 12,
  color: 'var(--hud-text)',
  background: 'var(--hud-bg)',
  border: '1px solid var(--hud-border)',
  borderRadius: 'var(--hud-radius)',
  padding: '8px 0',
  pointerEvents: 'none',
  minWidth: 180,
  maxWidth: 220,
  boxShadow: 'var(--hud-glow)',
  backdropFilter: 'blur(4px)',
};

const headerStyle: React.CSSProperties = {
  fontSize: 10,
  letterSpacing: 3,
  color: 'var(--hud-text-dim)',
  textAlign: 'center',
  paddingBottom: 6,
  borderBottom: '1px solid var(--hud-border)',
  marginBottom: 2,
};

const tableHeaderStyle: React.CSSProperties = {
  display: 'flex',
  padding: '3px 10px',
  fontSize: 10,
  color: 'var(--hud-text-dim)',
  letterSpacing: 1,
};

const colStyle: React.CSSProperties = {
  width: 28,
  textAlign: 'right',
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  padding: '4px 10px',
  height: ROW_HEIGHT,
  alignItems: 'center',
  willChange: 'transform',
};
