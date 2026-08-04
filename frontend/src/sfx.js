// Módulo de efeitos sonoros sintetizados via Web Audio API (sem arquivos).
let audioCtx = null;

function ensureCtx() {
  if (!audioCtx) {
    const AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return null;
    audioCtx = new AC();
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume();
  }
  return audioCtx;
}

function tone(freqStart, freqEnd, duration, type, volume, delay = 0) {
  const ctx = ensureCtx();
  if (!ctx) return;

  const t0 = ctx.currentTime + delay;
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();

  osc.type = type || 'square';
  osc.frequency.setValueAtTime(freqStart, t0);
  osc.frequency.exponentialRampToValueAtTime(Math.max(freqEnd, 1), t0 + duration);

  gain.gain.setValueAtTime(volume || 0.15, t0);
  gain.gain.exponentialRampToValueAtTime(0.0001, t0 + duration);

  osc.connect(gain);
  gain.connect(ctx.destination);

  osc.start(t0);
  osc.stop(t0 + duration + 0.02);
}

// Ações do jogo
export const sfx = {
  unlock() {
    ensureCtx();
  },
  jump() {
    tone(240, 480, 0.14, 'square', 0.09);
  },
  doubleJump() {
    tone(360, 720, 0.16, 'triangle', 0.11);
  },
  land() {
    tone(140, 60, 0.1, 'sine', 0.12);
  },
  tileBreak() {
    tone(120, 45, 0.28, 'sawtooth', 0.14);
  },
  death() {
    tone(320, 50, 0.55, 'triangle', 0.16);
  },
  dash() {
    tone(180, 900, 0.18, 'sawtooth', 0.12);
    tone(400, 1200, 0.12, 'square', 0.06, 0.02);
  },
  hit() {
    tone(200, 70, 0.14, 'square', 0.15);
    tone(90, 40, 0.12, 'sine', 0.14, 0.01);
  },
  respawn() {
    tone(300, 300, 0.08, 'square', 0.08, 0);
    tone(450, 450, 0.1, 'square', 0.08, 0.09);
  },
  roundStart() {
    tone(260, 260, 0.1, 'square', 0.08, 0);
    tone(390, 390, 0.1, 'square', 0.08, 0.12);
    tone(520, 520, 0.16, 'square', 0.09, 0.24);
  },
  click() {
    tone(600, 400, 0.06, 'square', 0.06);
  },
};

export default sfx;
