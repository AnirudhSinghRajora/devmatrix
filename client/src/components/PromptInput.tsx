import { useCallback, useEffect, useRef, useState } from 'react';
import { sendPrompt } from '../network/socket';
import { useGameStore } from '../store/gameStore';

const MAX_LENGTH = 200;
const COOLDOWN_SECONDS = 30;

export default function PromptInput() {
  const connected = useGameStore((s) => s.connected);
  const errorMessage = useGameStore((s) => s.errorMessage);
  const errorCooldown = useGameStore((s) => s.errorCooldown);
  const myDead = useGameStore((s) => s.myDeathTime !== null);
  const justRespawned = useGameStore((s) => s.justRespawned);

  const [text, setText] = useState('');
  const [cooldown, setCooldown] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // When server sends a cooldown via error, sync it to local timer.
  useEffect(() => {
    if (errorCooldown > 0) {
      setCooldown(errorCooldown);
    }
  }, [errorCooldown]);

  // Tick down the local cooldown timer every second.
  useEffect(() => {
    if (cooldown <= 0) return;
    const id = setInterval(() => {
      setCooldown((prev) => Math.max(0, prev - 1));
    }, 1000);
    return () => clearInterval(id);
  }, [cooldown]);

  // Clear error after 5 seconds.
  useEffect(() => {
    if (!errorMessage) return;
    const id = setTimeout(() => {
      useGameStore.getState().clearError();
    }, 5000);
    return () => clearTimeout(id);
  }, [errorMessage]);

  // Reset cooldown on respawn (server also clears its cooldown tracker).
  useEffect(() => {
    if (justRespawned) {
      setCooldown(0);
    }
  }, [justRespawned]);

  const submit = useCallback(() => {
    const trimmed = text.trim();
    if (!trimmed || cooldown > 0 || !connected || myDead) return;

    sendPrompt(trimmed);
    setText('');
    setCooldown(COOLDOWN_SECONDS);
  }, [text, cooldown, connected, myDead]);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        submit();
      }
    },
    [submit],
  );

  const disabled = !connected || cooldown > 0 || myDead;

  return (
    <div style={containerStyle}>
      {errorMessage && <div style={errorStyle}>{errorMessage}</div>}

      <div style={rowStyle}>
        <input
          ref={inputRef}
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value.slice(0, MAX_LENGTH))}
          onKeyDown={onKeyDown}
          disabled={disabled}
          placeholder={
            !connected
              ? 'Connecting…'
              : myDead
                ? 'Waiting for respawn…'
                : cooldown > 0
                  ? `Cooldown: ${cooldown}s`
                  : 'Command your ship…'
          }
          style={inputStyle}
          spellCheck={false}
          autoComplete="off"
        />
        <span style={charCountStyle}>
          {text.length}/{MAX_LENGTH}
        </span>
        <button onClick={submit} disabled={disabled} style={buttonStyle}>
          {cooldown > 0 ? `${cooldown}s` : 'Execute'}
        </button>
      </div>
    </div>
  );
}

// ---------- inline styles ----------

const containerStyle: React.CSSProperties = {
  position: 'absolute',
  bottom: 24,
  left: '50%',
  transform: 'translateX(-50%)',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  gap: 6,
  fontFamily: 'var(--hud-font)',
  zIndex: 10,
  width: '95%',
  maxWidth: 520,
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  width: '100%',
};

const inputStyle: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  padding: '10px 14px',
  background: 'var(--hud-bg)',
  border: '1px solid var(--hud-border)',
  borderRadius: 'var(--hud-radius)',
  color: 'var(--hud-accent)',
  fontFamily: 'var(--hud-font)',
  fontSize: 14,
  outline: 'none',
  boxShadow: 'var(--hud-glow)',
};

const charCountStyle: React.CSSProperties = {
  color: 'var(--hud-text-dim)',
  fontSize: 11,
  minWidth: 40,
  textAlign: 'right',
  flexShrink: 0,
};

const buttonStyle: React.CSSProperties = {
  padding: '10px 18px',
  background: 'rgba(0, 200, 255, 0.12)',
  border: '1px solid var(--hud-border)',
  borderRadius: 'var(--hud-radius)',
  color: 'var(--hud-accent)',
  fontFamily: 'var(--hud-font)',
  fontSize: 14,
  cursor: 'pointer',
  letterSpacing: 1,
  flexShrink: 0,
};

const errorStyle: React.CSSProperties = {
  color: 'var(--hud-red)',
  fontSize: 12,
  padding: '4px 12px',
  background: 'rgba(255, 0, 0, 0.08)',
  borderRadius: 'var(--hud-radius)',
  border: '1px solid rgba(255, 68, 68, 0.3)',
  maxWidth: '100%',
  textAlign: 'center',
};
