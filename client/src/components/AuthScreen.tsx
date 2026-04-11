import { useState } from 'react';
import { register, login } from '../network/api';

interface Props {
  onAuth: () => void;
  onSkip: () => void;
}

export default function AuthScreen({ onAuth, onSkip }: Props) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (mode === 'register') {
        await register(username, email, password);
      } else {
        await login(email, password);
      }
      onAuth();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      display: 'flex', justifyContent: 'center', alignItems: 'center',
      height: '100vh', background: '#0a0a0f',
      fontFamily: 'monospace', color: '#0f0',
    }}>
      <div style={{
        background: 'rgba(0,20,0,0.8)', border: '1px solid #0f03',
        borderRadius: 8, padding: 32, width: 340,
      }}>
        <h1 style={{ textAlign: 'center', fontSize: 24, marginBottom: 4 }}>DEVMATRIX</h1>
        <p style={{ textAlign: 'center', color: '#0a0', fontSize: 12, marginBottom: 24 }}>
          {mode === 'login' ? 'Sign in to your account' : 'Create a new account'}
        </p>

        <form onSubmit={handleSubmit}>
          {mode === 'register' && (
            <input
              type="text" placeholder="Username" value={username}
              onChange={(e) => setUsername(e.target.value)}
              style={inputStyle} autoComplete="username" required
            />
          )}
          <input
            type="email" placeholder="Email" value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={inputStyle} autoComplete="email" required
          />
          <input
            type="password" placeholder="Password" value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={inputStyle} autoComplete={mode === 'register' ? 'new-password' : 'current-password'}
            required minLength={8}
          />

          {error && (
            <div style={{ color: '#f44', fontSize: 12, marginBottom: 12 }}>{error}</div>
          )}

          <button type="submit" disabled={loading} style={{
            width: '100%', padding: '10px 0', background: '#0a0', color: '#000',
            border: 'none', borderRadius: 4, cursor: 'pointer', fontFamily: 'monospace',
            fontWeight: 'bold', fontSize: 14, marginBottom: 12,
            opacity: loading ? 0.6 : 1,
          }}>
            {loading ? '...' : mode === 'login' ? 'SIGN IN' : 'CREATE ACCOUNT'}
          </button>
        </form>

        <div style={{ textAlign: 'center', fontSize: 12 }}>
          <button onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(''); }}
            style={linkStyle}>
            {mode === 'login' ? 'Need an account? Register' : 'Have an account? Sign in'}
          </button>
        </div>

        <div style={{ textAlign: 'center', marginTop: 16, borderTop: '1px solid #0f02', paddingTop: 16 }}>
          <button onClick={onSkip} style={linkStyle}>
            Play as Guest
          </button>
        </div>
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  width: '100%', padding: '10px 12px', marginBottom: 12,
  background: '#111', color: '#0f0', border: '1px solid #0f03',
  borderRadius: 4, fontFamily: 'monospace', fontSize: 14,
  boxSizing: 'border-box',
};

const linkStyle: React.CSSProperties = {
  background: 'none', border: 'none', color: '#0a0', cursor: 'pointer',
  fontFamily: 'monospace', fontSize: 12, textDecoration: 'underline',
};
