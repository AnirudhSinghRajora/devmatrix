import { useCallback, useEffect, useRef, useState } from 'react';
import { sendPrompt } from '../network/socket';
import { useGameStore } from '../store/gameStore';

const MAX_LENGTH = 200;
const COOLDOWN_SECONDS = 30;

export default function PromptInput() {
  const connected = useGameStore((s) => s.connected);
  const errorMessage = useGameStore((s) => s.errorMessage);
  const errorCooldown = useGameStore((s) => s.errorCooldown);

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

  const submit = useCallback(() => {
    const trimmed = text.trim();
    if (!trimmed || cooldown > 0 || !connected) return;

    sendPrompt(trimmed);
    setText('');
    setCooldown(COOLDOWN_SECONDS);
  }, [text, cooldown, connected]);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        submit();
      }
    },
    [submit],
  );

  const disabled = !connected || cooldown > 0;

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
  fontFamily: 'monospace',
  zIndex: 10,
};

const rowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
};

const inputStyle: React.CSSProperties = {
  width: 400,
  padding: '8px 12px',
  background: 'rgba(0, 0, 0, 0.7)',
  border: '1px solid rgba(0, 255, 0, 0.4)',
  borderRadius: 4,
  color: '#0f0',
  fontFamily: 'monospace',
  fontSize: 14,
  outline: 'none',
};

const charCountStyle: React.CSSProperties = {
  color: 'rgba(0, 255, 0, 0.5)',
  fontSize: 12,
  minWidth: 50,
  textAlign: 'right',
};

const buttonStyle: React.CSSProperties = {
  padding: '8px 16px',
  background: 'rgba(0, 255, 0, 0.15)',
  border: '1px solid rgba(0, 255, 0, 0.4)',
  borderRadius: 4,
  color: '#0f0',
  fontFamily: 'monospace',
  fontSize: 14,
  cursor: 'pointer',
};

const errorStyle: React.CSSProperties = {
  color: '#f44',
  fontSize: 13,
  padding: '4px 10px',
  background: 'rgba(255, 0, 0, 0.1)',
  borderRadius: 4,
  border: '1px solid rgba(255, 0, 0, 0.3)',
};
