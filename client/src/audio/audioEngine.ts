// Procedural audio engine — pure Web Audio API, zero dependencies.
// All sounds are synthesized at runtime; no audio files are loaded.

type Vec3 = [number, number, number];

interface EngineSound {
  update: (speed: number, pan: number) => void;
  stop: () => void;
}

class AudioEngine {
  private ctx: AudioContext | null = null;
  private masterGain: GainNode | null = null;
  private ambientSource: AudioBufferSourceNode | null = null;
  private ambientGain: GainNode | null = null;

  // Lazily create AudioContext and master chain on first call.
  // Also resumes a suspended context (browsers suspend until user gesture).
  private getCtx(): AudioContext {
    if (!this.ctx) {
      this.ctx = new AudioContext();

      this.masterGain = this.ctx.createGain();
      this.masterGain.gain.value = 0.75;

      // Limiter to prevent clipping when many sounds overlap.
      const limiter = this.ctx.createDynamicsCompressor();
      limiter.threshold.value = -3;
      limiter.knee.value = 0;
      limiter.ratio.value = 20;
      limiter.attack.value = 0.001;
      limiter.release.value = 0.1;

      this.masterGain.connect(limiter);
      limiter.connect(this.ctx.destination);
    }
    if (this.ctx.state === 'suspended') {
      this.ctx.resume().catch(() => {});
    }
    return this.ctx;
  }

  private getOutput(): AudioNode {
    this.getCtx();
    return this.masterGain!;
  }

  // Call this on first user interaction to unlock the AudioContext.
  init(): void {
    this.getCtx();
  }

  // Create a white-noise buffer (reusable for many synthesis tasks).
  private makeNoiseBuffer(seconds: number): AudioBuffer {
    const ctx = this.getCtx();
    const length = Math.floor(ctx.sampleRate * seconds);
    const buffer = ctx.createBuffer(1, length, ctx.sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < length; i++) {
      data[i] = Math.random() * 2 - 1;
    }
    return buffer;
  }

  // Compute stereo pan and distance-based volume for a world-space event.
  // ownPos is the listener (player's ship); eventPos is the sound source.
  spatialise(ownPos: Vec3, eventPos: Vec3): { pan: number; volume: number } {
    const dx = eventPos[0] - ownPos[0];
    const dz = eventPos[2] - ownPos[2];
    const dist = Math.sqrt(dx * dx + dz * dz);
    return {
      pan: Math.max(-1, Math.min(1, dx / 350)),
      volume: Math.max(0, 1 - dist / 1400),
    };
  }

  // ── LASER FIRE ──────────────────────────────────────────────────────────────
  // hit=true: descending sweep + noise crunch (projectile connected)
  // hit=false: cleaner tone-only sweep (miss)
  playLaser(hit: boolean, pan = 0, volume = 0.35): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    const panNode = ctx.createStereoPanner();
    panNode.pan.value = Math.max(-1, Math.min(1, pan));
    panNode.connect(this.getOutput());

    const gainNode = ctx.createGain();
    gainNode.gain.setValueAtTime(volume, now);
    gainNode.gain.exponentialRampToValueAtTime(0.001, now + 0.14);
    gainNode.connect(panNode);

    // Main tone: pitch sweep downward
    const osc = ctx.createOscillator();
    osc.type = 'sine';
    osc.frequency.setValueAtTime(hit ? 1500 : 1050, now);
    osc.frequency.exponentialRampToValueAtTime(hit ? 280 : 580, now + 0.11);
    osc.connect(gainNode);
    osc.start(now);
    osc.stop(now + 0.16);

    if (hit) {
      // Impact crunch: bandpass-filtered noise burst
      const noiseSource = ctx.createBufferSource();
      noiseSource.buffer = this.makeNoiseBuffer(0.08);

      const bandpass = ctx.createBiquadFilter();
      bandpass.type = 'bandpass';
      bandpass.frequency.value = 1800;
      bandpass.Q.value = 1.2;

      const crunchGain = ctx.createGain();
      crunchGain.gain.setValueAtTime(0, now + 0.05);
      crunchGain.gain.linearRampToValueAtTime(volume * 0.5, now + 0.07);
      crunchGain.gain.exponentialRampToValueAtTime(0.001, now + 0.14);

      noiseSource.connect(bandpass);
      bandpass.connect(crunchGain);
      crunchGain.connect(panNode);
      noiseSource.start(now + 0.05);
    }
  }

  // ── EXPLOSION ───────────────────────────────────────────────────────────────
  // Ship destroyed: deep thump + decaying noise rumble.
  playExplosion(pan = 0, volume = 0.7): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    const panNode = ctx.createStereoPanner();
    panNode.pan.value = Math.max(-1, Math.min(1, pan));
    panNode.connect(this.getOutput());

    // Noise rumble through a sweeping low-pass filter
    const noiseSource = ctx.createBufferSource();
    noiseSource.buffer = this.makeNoiseBuffer(1.3);

    const lowpass = ctx.createBiquadFilter();
    lowpass.type = 'lowpass';
    lowpass.frequency.setValueAtTime(3000, now);
    lowpass.frequency.exponentialRampToValueAtTime(55, now + 1.0);

    const rumbleGain = ctx.createGain();
    rumbleGain.gain.setValueAtTime(0.001, now);
    rumbleGain.gain.linearRampToValueAtTime(volume * 0.85, now + 0.008);
    rumbleGain.gain.exponentialRampToValueAtTime(0.001, now + 1.1);

    noiseSource.connect(lowpass);
    lowpass.connect(rumbleGain);
    rumbleGain.connect(panNode);
    noiseSource.start(now);

    // Sub-bass thump: sine wave dropping fast
    const thump = ctx.createOscillator();
    thump.type = 'sine';
    thump.frequency.setValueAtTime(90, now);
    thump.frequency.exponentialRampToValueAtTime(22, now + 0.35);

    const thumpGain = ctx.createGain();
    thumpGain.gain.setValueAtTime(volume * 0.9, now);
    thumpGain.gain.exponentialRampToValueAtTime(0.001, now + 0.38);

    thump.connect(thumpGain);
    thumpGain.connect(panNode);
    thump.start(now);
    thump.stop(now + 0.4);

    // High-frequency crack (percussive transient)
    const crackSource = ctx.createBufferSource();
    crackSource.buffer = this.makeNoiseBuffer(0.05);

    const crackFilter = ctx.createBiquadFilter();
    crackFilter.type = 'highpass';
    crackFilter.frequency.value = 3000;

    const crackGain = ctx.createGain();
    crackGain.gain.setValueAtTime(volume * 0.6, now);
    crackGain.gain.exponentialRampToValueAtTime(0.001, now + 0.05);

    crackSource.connect(crackFilter);
    crackFilter.connect(crackGain);
    crackGain.connect(panNode);
    crackSource.start(now);
  }

  // ── SHIELD HIT ──────────────────────────────────────────────────────────────
  // Deflection: metallic harmonic ring.
  playShieldHit(pan = 0): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    const panNode = ctx.createStereoPanner();
    panNode.pan.value = Math.max(-1, Math.min(1, pan));
    panNode.connect(this.getOutput());

    // Three harmonic partials ringing off together
    const partials: [number, number][] = [
      [1100, 0.28],
      [2200, 0.14],
      [550, 0.10],
    ];

    for (const [freq, amp] of partials) {
      const osc = ctx.createOscillator();
      osc.type = 'sine';
      osc.frequency.value = freq;

      const gain = ctx.createGain();
      gain.gain.setValueAtTime(amp, now);
      gain.gain.exponentialRampToValueAtTime(0.001, now + 0.28);

      osc.connect(gain);
      gain.connect(panNode);
      osc.start(now);
      osc.stop(now + 0.32);
    }
  }

  // ── OWN SHIP HIT (hull damage) ──────────────────────────────────────────────
  // Heavier impact when shields are down and hull takes a hit.
  playHullImpact(): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    const noiseSource = ctx.createBufferSource();
    noiseSource.buffer = this.makeNoiseBuffer(0.15);

    const filter = ctx.createBiquadFilter();
    filter.type = 'bandpass';
    filter.frequency.value = 400;
    filter.Q.value = 0.8;

    const gain = ctx.createGain();
    gain.gain.setValueAtTime(0.45, now);
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.18);

    noiseSource.connect(filter);
    filter.connect(gain);
    gain.connect(this.getOutput());
    noiseSource.start(now);

    // Metal clunk tone
    const osc = ctx.createOscillator();
    osc.type = 'triangle';
    osc.frequency.setValueAtTime(180, now);
    osc.frequency.exponentialRampToValueAtTime(80, now + 0.12);

    const oscGain = ctx.createGain();
    oscGain.gain.setValueAtTime(0.3, now);
    oscGain.gain.exponentialRampToValueAtTime(0.001, now + 0.15);

    osc.connect(oscGain);
    oscGain.connect(this.getOutput());
    osc.start(now);
    osc.stop(now + 0.18);
  }

  // ── OWN DEATH ───────────────────────────────────────────────────────────────
  // Descending power-down drone when the player's ship is destroyed.
  playDeath(): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    // Sawtooth power-down sweep
    const osc = ctx.createOscillator();
    osc.type = 'sawtooth';
    osc.frequency.setValueAtTime(280, now);
    osc.frequency.exponentialRampToValueAtTime(30, now + 1.4);

    const filter = ctx.createBiquadFilter();
    filter.type = 'lowpass';
    filter.frequency.setValueAtTime(1200, now);
    filter.frequency.exponentialRampToValueAtTime(80, now + 1.4);

    const gain = ctx.createGain();
    gain.gain.setValueAtTime(0.5, now);
    gain.gain.setValueAtTime(0.5, now + 0.8);
    gain.gain.exponentialRampToValueAtTime(0.001, now + 1.5);

    osc.connect(filter);
    filter.connect(gain);
    gain.connect(this.getOutput());
    osc.start(now);
    osc.stop(now + 1.6);

    // Noise tail
    const noiseSource = ctx.createBufferSource();
    noiseSource.buffer = this.makeNoiseBuffer(0.8);

    const noiseFilter = ctx.createBiquadFilter();
    noiseFilter.type = 'lowpass';
    noiseFilter.frequency.value = 300;

    const noiseGain = ctx.createGain();
    noiseGain.gain.setValueAtTime(0.2, now + 0.1);
    noiseGain.gain.exponentialRampToValueAtTime(0.001, now + 0.9);

    noiseSource.connect(noiseFilter);
    noiseFilter.connect(noiseGain);
    noiseGain.connect(this.getOutput());
    noiseSource.start(now + 0.1);
  }

  // ── RESPAWN ─────────────────────────────────────────────────────────────────
  // Ascending four-note arpeggio — player is back.
  playRespawn(): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;
    const notes = [330, 415, 494, 660];

    notes.forEach((freq, i) => {
      const t = now + i * 0.095;

      const osc = ctx.createOscillator();
      osc.type = 'triangle';
      osc.frequency.value = freq;

      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0, t);
      gain.gain.linearRampToValueAtTime(0.28, t + 0.015);
      gain.gain.exponentialRampToValueAtTime(0.001, t + 0.13);

      osc.connect(gain);
      gain.connect(this.getOutput());
      osc.start(t);
      osc.stop(t + 0.15);
    });
  }

  // ── BEHAVIOR CONFIRMED ──────────────────────────────────────────────────────
  // Two-note ascending UI confirm chime — command accepted.
  playBehaviorConfirm(): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    [[560, 0], [840, 0.07]].forEach(([freq, delay]) => {
      const t = now + delay;

      const osc = ctx.createOscillator();
      osc.type = 'sine';
      osc.frequency.value = freq;

      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0.18, t);
      gain.gain.exponentialRampToValueAtTime(0.001, t + 0.1);

      osc.connect(gain);
      gain.connect(this.getOutput());
      osc.start(t);
      osc.stop(t + 0.12);
    });
  }

  // ── ERROR / COOLDOWN ────────────────────────────────────────────────────────
  // Two descending square-wave beeps — command rejected.
  playError(): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    [[440, 0], [330, 0.1]].forEach(([freq, delay]) => {
      const t = now + delay;

      const osc = ctx.createOscillator();
      osc.type = 'square';
      osc.frequency.value = freq;

      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0.12, t);
      gain.gain.exponentialRampToValueAtTime(0.001, t + 0.08);

      osc.connect(gain);
      gain.connect(this.getOutput());
      osc.start(t);
      osc.stop(t + 0.1);
    });
  }

  // ── KILL NOTIFICATION ───────────────────────────────────────────────────────
  // isOwnKill=true: triumphant 3-note fanfare
  // isOwnKill=false: quiet distant pop (others killing each other)
  playKill(isOwnKill: boolean): void {
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    if (isOwnKill) {
      [880, 1108, 1320].forEach((freq, i) => {
        const t = now + i * 0.08;

        const osc = ctx.createOscillator();
        osc.type = 'triangle';
        osc.frequency.value = freq;

        const gain = ctx.createGain();
        gain.gain.setValueAtTime(0, t);
        gain.gain.linearRampToValueAtTime(0.3, t + 0.012);
        gain.gain.exponentialRampToValueAtTime(0.001, t + 0.12);

        osc.connect(gain);
        gain.connect(this.getOutput());
        osc.start(t);
        osc.stop(t + 0.14);
      });
    } else {
      // Quiet low pop for nearby kill
      const osc = ctx.createOscillator();
      osc.type = 'sine';
      osc.frequency.setValueAtTime(220, now);
      osc.frequency.exponentialRampToValueAtTime(70, now + 0.07);

      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0.1, now);
      gain.gain.exponentialRampToValueAtTime(0.001, now + 0.09);

      osc.connect(gain);
      gain.connect(this.getOutput());
      osc.start(now);
      osc.stop(now + 0.1);
    }
  }

  // ── ENGINE THRUSTER (own ship) ──────────────────────────────────────────────
  // Bandpass-filtered noise loop. Call update() each frame with [0..1] speed.
  createEngineSound(): EngineSound {
    const ctx = this.getCtx();

    // 3-second looping noise buffer for smooth engine sound
    const source = ctx.createBufferSource();
    source.buffer = this.makeNoiseBuffer(3);
    source.loop = true;

    // Two bandpass filters layered for richer texture
    const band1 = ctx.createBiquadFilter();
    band1.type = 'bandpass';
    band1.frequency.value = 180;
    band1.Q.value = 1.8;

    const band2 = ctx.createBiquadFilter();
    band2.type = 'bandpass';
    band2.frequency.value = 320;
    band2.Q.value = 1.2;

    const gainNode = ctx.createGain();
    gainNode.gain.value = 0;

    const panNode = ctx.createStereoPanner();
    panNode.pan.value = 0;

    source.connect(band1);
    source.connect(band2);
    band1.connect(gainNode);
    band2.connect(gainNode);
    gainNode.connect(panNode);
    panNode.connect(this.getOutput());
    source.start();

    return {
      update: (speed: number, pan: number): void => {
        const s = Math.max(0, Math.min(1, speed));
        const t = ctx.currentTime;
        gainNode.gain.setTargetAtTime(s * 0.14, t, 0.08);
        band1.frequency.setTargetAtTime(140 + s * 280, t, 0.1);
        band2.frequency.setTargetAtTime(260 + s * 300, t, 0.1);
        panNode.pan.setTargetAtTime(Math.max(-1, Math.min(1, pan)), t, 0.05);
      },
      stop: (): void => {
        const t = ctx.currentTime;
        gainNode.gain.setTargetAtTime(0, t, 0.08);
        setTimeout(() => {
          try { source.stop(); } catch { /* already stopped */ }
        }, 600);
      },
    };
  }

  // ── AMBIENT SPACE ──────────────────────────────────────────────────────────
  // Deep looping space hum: two overlapping bandpass noise filters.
  // Fades in over 3 seconds; fades out over 1 second.
  startAmbient(): void {
    if (this.ambientSource) return;
    const ctx = this.getCtx();
    const now = ctx.currentTime;

    this.ambientSource = ctx.createBufferSource();
    this.ambientSource.buffer = this.makeNoiseBuffer(5);
    this.ambientSource.loop = true;

    const sub = ctx.createBiquadFilter();
    sub.type = 'bandpass';
    sub.frequency.value = 60;
    sub.Q.value = 0.4;

    const mid = ctx.createBiquadFilter();
    mid.type = 'bandpass';
    mid.frequency.value = 110;
    mid.Q.value = 0.3;

    // Slow LFO on the mid filter for a gentle evolving texture
    const lfo = ctx.createOscillator();
    lfo.type = 'sine';
    lfo.frequency.value = 0.12;
    const lfoGain = ctx.createGain();
    lfoGain.gain.value = 25;
    lfo.connect(lfoGain);
    lfoGain.connect(mid.frequency);
    lfo.start(now);

    this.ambientGain = ctx.createGain();
    this.ambientGain.gain.setValueAtTime(0, now);
    this.ambientGain.gain.linearRampToValueAtTime(0.07, now + 3);

    this.ambientSource.connect(sub);
    this.ambientSource.connect(mid);
    sub.connect(this.ambientGain);
    mid.connect(this.ambientGain);
    this.ambientGain.connect(this.getOutput());
    this.ambientSource.start(now);
  }

  stopAmbient(): void {
    if (!this.ambientGain || !this.ambientSource || !this.ctx) return;
    const now = this.ctx.currentTime;
    this.ambientGain.gain.setValueAtTime(this.ambientGain.gain.value, now);
    this.ambientGain.gain.linearRampToValueAtTime(0, now + 1);
    const src = this.ambientSource;
    this.ambientSource = null;
    this.ambientGain = null;
    setTimeout(() => {
      try { src.stop(); } catch { /* already stopped */ }
    }, 1200);
  }

  // ── CLEANUP ─────────────────────────────────────────────────────────────────
  destroy(): void {
    this.stopAmbient();
    if (this.ctx) {
      this.ctx.close().catch(() => {});
      this.ctx = null;
      this.masterGain = null;
    }
  }
}

export const audioEngine = new AudioEngine();
