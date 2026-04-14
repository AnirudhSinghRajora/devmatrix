// Dev-only audio test page. Access at http://localhost:5173/?audio
// Lets you trigger every procedural sound without a server or database.

import { audioEngine } from './audioEngine';

const btn: React.CSSProperties = {
  background: 'rgba(0,255,0,0.08)',
  border: '1px solid #0f04',
  color: '#0f0',
  fontFamily: 'monospace',
  fontSize: 13,
  padding: '10px 18px',
  borderRadius: 4,
  cursor: 'pointer',
  textAlign: 'left',
  transition: 'background 0.1s',
};

const section: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
};

const label: React.CSSProperties = {
  color: '#0f08',
  fontFamily: 'monospace',
  fontSize: 11,
  letterSpacing: 2,
  textTransform: 'uppercase',
  marginBottom: 2,
};

interface SoundButton {
  name: string;
  desc: string;
  fn: () => void;
}

export default function AudioTestPage() {
  const init = () => audioEngine.init();

  const sounds: { group: string; buttons: SoundButton[] }[] = [
    {
      group: 'Combat',
      buttons: [
        { name: 'Laser — miss', desc: 'tone sweep, no hit', fn: () => { init(); audioEngine.playLaser(false, 0, 0.4); } },
        { name: 'Laser — hit (left)', desc: 'tone + crunch, panned left', fn: () => { init(); audioEngine.playLaser(true, -0.7, 0.4); } },
        { name: 'Laser — hit (right)', desc: 'tone + crunch, panned right', fn: () => { init(); audioEngine.playLaser(true, 0.7, 0.4); } },
        { name: 'Explosion (near)', desc: 'thump + rumble, full volume', fn: () => { init(); audioEngine.playExplosion(0, 0.8); } },
        { name: 'Explosion (far left)', desc: 'distant, panned', fn: () => { init(); audioEngine.playExplosion(-0.6, 0.3); } },
      ],
    },
    {
      group: 'Own Ship',
      buttons: [
        { name: 'Shield hit', desc: 'metallic harmonic ring', fn: () => { init(); audioEngine.playShieldHit(0); } },
        { name: 'Hull impact', desc: 'heavier — shields down', fn: () => { init(); audioEngine.playHullImpact(); } },
        { name: 'Death', desc: 'power-down drone', fn: () => { init(); audioEngine.playDeath(); } },
        { name: 'Respawn', desc: 'ascending 4-note arpeggio', fn: () => { init(); audioEngine.playRespawn(); } },
      ],
    },
    {
      group: 'UI / HUD',
      buttons: [
        { name: 'Behavior confirmed', desc: 'two-note confirm blip', fn: () => { init(); audioEngine.playBehaviorConfirm(); } },
        { name: 'Error / cooldown', desc: 'descending square beeps', fn: () => { init(); audioEngine.playError(); } },
        { name: 'Kill — own', desc: 'triumphant 3-note fanfare', fn: () => { init(); audioEngine.playKill(true); } },
        { name: 'Kill — nearby', desc: 'quiet distant pop', fn: () => { init(); audioEngine.playKill(false); } },
      ],
    },
    {
      group: 'Continuous',
      buttons: [
        {
          name: 'Start ambient',
          desc: 'deep space hum (fades in over 3s)',
          fn: () => { init(); audioEngine.startAmbient(); },
        },
        {
          name: 'Stop ambient',
          desc: '',
          fn: () => audioEngine.stopAmbient(),
        },
        {
          name: 'Engine: idle → full → idle',
          desc: 'bandpass noise, speed ramps up then down',
          fn: () => {
            init();
            const eng = audioEngine.createEngineSound();
            let t = 0;
            const interval = setInterval(() => {
              t += 0.05;
              const speed = t < 1 ? t : t < 2 ? 1 : Math.max(0, 3 - t);
              eng.update(speed, 0);
              if (t >= 3.2) { clearInterval(interval); eng.stop(); }
            }, 50);
          },
        },
      ],
    },
  ];

  return (
    <div style={{
      minHeight: '100vh',
      background: '#050a05',
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'center',
      padding: '48px 24px',
    }}>
      <div style={{ maxWidth: 640, width: '100%' }}>
        <div style={{ fontFamily: 'monospace', color: '#0f0', marginBottom: 32 }}>
          <div style={{ fontSize: 22, fontWeight: 'bold', letterSpacing: 3 }}>AUDIO TEST</div>
          <div style={{ color: '#0f06', fontSize: 12, marginTop: 4 }}>
            DevMatrix procedural audio — no backend required
          </div>
        </div>

        <div style={{ background: 'rgba(255,200,0,0.07)', border: '1px solid #fa04', borderRadius: 4, padding: '10px 14px', marginBottom: 32, fontFamily: 'monospace', fontSize: 12, color: '#fa0' }}>
          Click any button to hear the sound. The first click also unlocks the AudioContext.
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 32 }}>
          {sounds.map(({ group, buttons }) => (
            <div key={group} style={section}>
              <div style={label}>{group}</div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                {buttons.map(({ name, desc, fn }) => (
                  <button key={name} style={btn} onClick={fn}
                    onMouseEnter={e => (e.currentTarget.style.background = 'rgba(0,255,0,0.15)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'rgba(0,255,0,0.08)')}
                  >
                    <span style={{ color: '#0f0' }}>{name}</span>
                    {desc && <span style={{ color: '#0f05', marginLeft: 12 }}>{desc}</span>}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
