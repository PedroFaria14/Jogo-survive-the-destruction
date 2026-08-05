import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Container,
  Grid,
  Paper,
  Typography,
  Button,
  TextField,
  List,
  ListItem,
  ListItemText,
  Chip,
} from '@mui/material';

import './App.css';
import { sfx } from './sfx.js';

// =======================
// Configurações
// =======================
const GAME_WS_URL = 'ws://localhost:8080/ws';
const SCORES_API_URL = 'http://localhost:8080/api/scores';
const CONFIG_API_URL = 'http://localhost:8080/api/config';

// Fallback das constantes do jogo (o servidor fornece via /api/config)
const DEFAULT_CFG = {
  arena_width: 800,
  arena_height: 600,
  tile_size: 100,
  player_radius: 25,
  move_speed: 8,
  jump_force: 20,
  gravity: 1.5,
  break_interval: 5,
  max_air_jumps: 1,
  max_lives: 3,
  dash_speed: 20,
  dash_cooldown: 1.5,
  dash_duration: 0.25,
  knockback_dash: 11,
  restitution: 0.85,
  min_islands: 3,
  max_islands: 7,
  island_width_min: 2,
  island_width_max: 5,
  max_gap_cols: 2,
};

// Configuração das animações da bola
const TRAIL_LENGTH = 12; // Comprimento do rastro
const GLOW_PULSE_SPEED = 260; // Velocidade do pulso de brilho

// =======================
// Paleta "ilhas flutuantes" (flat + sereno + terroso)
// =======================
const PALETTE = {
  skyTop: '#dcece9',
  skyMid: '#f6e3c5',
  skyBottom: '#efc58b',
  grassTop: '#8a9a4a',
  grassEdge: '#6f7d3a',
  soilTop: '#b08d5f',
  soilBottom: '#8f6f47',
  soilBorder: '#6b5a3f',
  islandShadow: 'rgba(90,70,40,0.18)',
  falling: '#c9743d',
  fallingDark: '#a8552f',
  treeTrunk: '#7c5a3a',
  treeCanopy: '#6f7d3f',
  treeCanopyLight: '#87944f',
  ballHi: '#ffe3a3',
  ballMid: '#f2b544',
  ballDark: '#d98f24',
  ballGlow: '#f4b942',
  ballMarker: '#7a4b12',
  ballOutline: 'rgba(122,75,18,0.6)',
  trailRGB: '244,185,66',
  ghostFill: 'rgba(180,150,90,0.18)',
  ghostStroke: 'rgba(160,120,60,0.4)',
  gold: '#f4b942',
  amber: '#e0a63c',
  terracotta: '#c96f4a',
  soilCrumb: '#8f6f47',
  cream: '#fdf6e9',
  creamOverlay: 'rgba(252,244,231,0.94)',
  textBrown: '#4a3b28',
  textSoft: '#7a623f',
  olive: '#6f7d3a',
  goldStrong: '#d99a2b',
  terracottaStrong: '#c96f4a',
  borderSoft: 'rgba(145,110,64,0.55)',
};

// Caminho de retângulo com cantos arredondados (sem depender de ctx.roundRect).
function roundRectPath(ctx, x, y, w, h, r) {
  const rr = Math.min(r, w / 2, h / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.lineTo(x + w - rr, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + rr);
  ctx.lineTo(x + w, y + h - rr);
  ctx.quadraticCurveTo(x + w, y + h, x + w - rr, y + h);
  ctx.lineTo(x + rr, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - rr);
  ctx.lineTo(x, y + rr);
  ctx.quadraticCurveTo(x, y, x + rr, y);
  ctx.closePath();
}

// Hash determinístico por posição (define quais blocos ganham árvore).
function tileHash(x, y) {
  let h = 17;
  h = (h * 31 + Math.round(x)) | 0;
  h = (h * 31 + Math.round(y)) | 0;
  return Math.abs(h);
}

// Árvore flat: tronco marrom + copa circular olive com meia-lua clara.
function drawTree(ctx, x, y, s) {
  ctx.fillStyle = 'rgba(60,70,30,0.15)';
  ctx.beginPath();
  ctx.ellipse(x + s * 0.5, y + s * 0.62, s * 0.18, s * 0.06, 0, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = PALETTE.treeTrunk;
  roundRectPath(ctx, x + s * 0.46, y + s * 0.46, s * 0.08, s * 0.18, 3);
  ctx.fill();

  ctx.fillStyle = PALETTE.treeCanopy;
  ctx.beginPath();
  ctx.arc(x + s * 0.5, y + s * 0.34, s * 0.22, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = PALETTE.treeCanopyLight;
  ctx.beginPath();
  ctx.arc(x + s * 0.43, y + s * 0.27, s * 0.12, 0, Math.PI * 2);
  ctx.fill();
}

// Nuvem suave (flat) no céu.
function drawCloud(ctx, x, y, s) {
  ctx.fillStyle = 'rgba(255,255,255,0.5)';
  ctx.beginPath();
  ctx.ellipse(x, y, s, s * 0.4, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(x - s * 0.6, y + s * 0.15, s * 0.55, s * 0.32, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.ellipse(x + s * 0.6, y + s * 0.12, s * 0.5, s * 0.3, 0, 0, Math.PI * 2);
  ctx.fill();
}

// =======================
// Desenho da bola (animado)
// =======================
function drawPlayerBall(ctx, player, anim, now, dt, radius) {
  // Trilha/trail
  anim.trail.push({ x: player.x, y: player.y });
  if (anim.trail.length > TRAIL_LENGTH) anim.trail.shift();

  anim.trail.forEach((t, i) => {
    const alpha = (i / TRAIL_LENGTH) * 0.22;
    const r = radius * 0.55 * (i / TRAIL_LENGTH);
    ctx.fillStyle = `rgba(${PALETTE.trailRGB},${alpha.toFixed(3)})`;
    ctx.beginPath();
    ctx.arc(t.x, t.y, r, 0, Math.PI * 2);
    ctx.fill();
  });

  // Rotação (bola rolando)
  if (Math.abs(player.vx) > 0.1) {
    anim.rotation += (player.vx * dt) / radius;
  }

  // Squash & stretch
  let scaleX = 1;
  let scaleY = 1;
  if (player.on_ground) {
    const s = anim.landSquash;
    scaleY = 1 - s * 0.35;
    scaleX = 1 + s * 0.35;
    anim.landSquash = Math.max(0, anim.landSquash - dt * 2.5);
  } else {
    const stretch = Math.min(Math.abs(player.vy) / 420, 0.32);
    scaleY = 1 + stretch;
    scaleX = 1 - stretch * 0.6;
  }
  // Detecta pouso (achata a bola)
  if (player.on_ground && anim.prevOnGround === false && anim.prevVy > 8) {
    anim.landSquash = Math.min((anim.prevVy - 8) / 28, 1);
  }
  anim.prevOnGround = player.on_ground;
  anim.prevVy = player.vy;

  ctx.save();
  ctx.translate(player.x, player.y);
  ctx.scale(scaleX, scaleY);

  // Pulso/brilho
  ctx.shadowColor = PALETTE.ballGlow;
  ctx.shadowBlur = 16 + Math.sin(now / GLOW_PULSE_SPEED) * 6;

  const grad = ctx.createRadialGradient(-6, -6, 4, 0, 0, radius);
  grad.addColorStop(0, PALETTE.ballHi);
  grad.addColorStop(0.55, PALETTE.ballMid);
  grad.addColorStop(1, PALETTE.ballDark);
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.shadowBlur = 0;

  // Marcador de rotação
  ctx.save();
  ctx.rotate(anim.rotation);
  ctx.fillStyle = PALETTE.ballMarker;
  ctx.beginPath();
  ctx.arc(radius * 0.55, 0, 4.5, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();

  // Contorno
  ctx.strokeStyle = PALETTE.ballOutline;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.restore();
}

// Jogador morto: "fantasma" translúcido (modo espectador)
function drawGhost(ctx, player, radius) {
  ctx.fillStyle = PALETTE.ghostFill;
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.strokeStyle = PALETTE.ghostStroke;
  ctx.lineWidth = 1.5;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.setLineDash([]);
}

// =======================
// Leaderboard
// =======================
const Leaderboard = ({ scores }) => (
  <Paper
    sx={{
      p: 3,
      borderRadius: 4,
      bgcolor: PALETTE.cream,
      border: `1px solid rgba(201,111,74,0.45)`,
      boxShadow: '0 10px 30px rgba(90,70,40,0.12)',
    }}
  >
    <Typography variant="h6" color={PALETTE.terracottaStrong} gutterBottom>
      🏅 Top 10 Sobreviventes
    </Typography>

    {scores.length === 0 ? (
      <Typography variant="body2" color="text.secondary">
        Nenhum placar registrado ainda.
      </Typography>
    ) : (
      <List dense>
        {scores.map((score, index) => (
          <ListItem
            key={index}
            sx={{
              bgcolor: 'rgba(240,224,196,0.6)',
              borderRadius: 2,
              mb: 1,
            }}
          >
            <Chip
              label={index + 1}
              size="small"
              sx={{ mr: 2 }}
              color={index < 3 ? 'warning' : 'default'}
            />

            <ListItemText
              primary={score.name || (score.player_id ? `Player_${score.player_id.slice(-4)}` : 'Anônimo')}
            />

            <Typography color={PALETTE.goldStrong} fontWeight="bold">
              {score.score_seconds}s
            </Typography>
          </ListItem>
        ))}
      </List>
    )}
  </Paper>
);

// =======================
// App
// =======================
export default function App() {
  const [cfg, setCfg] = useState(DEFAULT_CFG);
  const [gameState, setGameState] = useState(null);
  const [scores, setScores] = useState([]);
  const [myPlayerId, setMyPlayerId] = useState(null);
  const [gameStarted, setGameStarted] = useState(false);
  const [nickname, setNickname] = useState(
    () => `Jogador${Math.floor(1000 + Math.random() * 9000)}`
  );
  const [bestScore, setBestScore] = useState(0);
  const [isTouch, setIsTouch] = useState(false);

  const socketRef = useRef(null);
  const canvasRef = useRef(null);
  const startedRef = useRef(false);
  const nicknameRef = useRef(nickname);
  const keysRef = useRef({ left: false, right: false, jump: false, dash: false });
  const animsRef = useRef({});
  const lastDrawTimeRef = useRef(null);
  const particlesRef = useRef([]);
  const shakeRef = useRef(0);
  const prevTilesRef = useRef({});
  const prevRoundOverRef = useRef(null);
  const impactsRef = useRef({});
  const prevLivesRef = useRef(null);

  // Dimensões da arena: o servidor envia por rodada (mapa procedural muda).
  const arenaW = gameState?.arena_width ?? cfg.arena_width;
  const arenaH = gameState?.arena_height ?? cfg.arena_height;
  const tileSize = cfg.tile_size;
  const playerRadius = cfg.player_radius;

  // =======================
  // Start Game
  // =======================
  const handleStartGame = useCallback(() => {
    sfx.unlock();
    sfx.click();
    startedRef.current = true;
    setGameStarted(true);
    setGameState(null);
  }, []);

  // =======================
  // Enviar comandos (estado completo de input)
  // =======================
  const sendFrame = useCallback(() => {
    const s = socketRef.current;
    if (s && s.readyState === WebSocket.OPEN && startedRef.current) {
      s.send(
        JSON.stringify({
          type: 'input',
          left: keysRef.current.left,
          right: keysRef.current.right,
          jump: keysRef.current.jump,
          dash: keysRef.current.dash,
        })
      );
    }
  }, []);

  // =======================
  // Tentar novamente (mesmo mapa)
  // =======================
  const sendRestart = useCallback(() => {
    sfx.click();
    if (socketRef.current && socketRef.current.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify({ type: 'restart' }));
    }
  }, []);

  // =======================
  // Config compartilhada (evita drift backend/frontend)
  // =======================
  useEffect(() => {
    fetch(CONFIG_API_URL)
      .then((res) => res.json())
      .then(setCfg)
      .catch(() => {});
  }, []);

  // =======================
  // Teclado
  // =======================
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.repeat) return;
      const k = e.key.toLowerCase();

      if (k === 'a' || k === 'arrowleft') {
        keysRef.current.left = true;
        sendFrame();
      } else if (k === 'd' || k === 'arrowright') {
        keysRef.current.right = true;
        sendFrame();
      } else if (k === 'w' || k === ' ') {
        keysRef.current.jump = true;
        sfx.jump();
        sendFrame();
      } else if (k === 'shift') {
        keysRef.current.dash = true;
        sfx.dash();
        sendFrame();
      }
    };

    const handleKeyUp = (e) => {
      const k = e.key.toLowerCase();
      if (k === 'a' || k === 'arrowleft') keysRef.current.left = false;
      else if (k === 'd' || k === 'arrowright') keysRef.current.right = false;
      else if (k === 'w' || k === ' ') keysRef.current.jump = false;
      else if (k === 'shift') keysRef.current.dash = false;
      else return;
      sendFrame();
    };

    const handleStartKey = (e) => {
      if (!startedRef.current && (e.key === 'Enter' || e.key === ' ')) {
        handleStartGame();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    window.addEventListener('keydown', handleStartKey);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
      window.removeEventListener('keydown', handleStartKey);
    };
  }, [sendFrame, handleStartGame]);

  // =======================
  // Touch
  // =======================
  useEffect(() => {
    setIsTouch(window.matchMedia?.('(pointer: coarse)').matches ?? false);
  }, []);

  const touchPress = (key) => () => {
    if (key === 'jump') sfx.jump();
    else if (key === 'dash') sfx.dash();
    keysRef.current[key] = true;
    sendFrame();
  };
  const touchRelease = (key) => () => {
    keysRef.current[key] = false;
    sendFrame();
  };

  // =======================
  // WebSocket
  // =======================
  useEffect(() => {
    if (!gameStarted) return;

    const ws = new WebSocket(GAME_WS_URL);
    socketRef.current = ws;

    ws.onopen = () => {
      console.log('🟢 WebSocket conectado');
      sfx.unlock();
      ws.send(JSON.stringify({ type: 'join', name: nicknameRef.current }));
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);

      if (data.type === 'init') {
        setMyPlayerId(data.player_id);
        return;
      }

      if (data.type) return;

      setGameState(data);
    };

    ws.onerror = (err) => console.error('WebSocket erro', err);
    ws.onclose = () => console.log('🔴 WebSocket fechado');

    return () => ws.close();
  }, [gameStarted]);

  // Mantém o ref do nickname sincronizado
  useEffect(() => {
    nicknameRef.current = nickname;
  }, [nickname]);

  // =======================
  // Partículas e shake (helpers)
  // =======================
  const spawnParticles = useCallback((x, y, color, count, speed) => {
    const parts = particlesRef.current;
    for (let i = 0; i < count; i++) {
      const a = Math.random() * Math.PI * 2;
      const sp = (0.5 + Math.random() * 0.8) * (speed || 120);
      parts.push({
        x,
        y,
        vx: Math.cos(a) * sp,
        vy: Math.sin(a) * sp - 40,
        life: 1,
        decay: 2 + Math.random() * 1.5,
        color,
      });
    }
    if (parts.length > 400) parts.splice(0, parts.length - 400);
  }, []);

  const addShake = useCallback((v) => {
    shakeRef.current = Math.min(shakeRef.current + v, 20);
  }, []);

  // =======================
  // Desenho Canvas
  // =======================
  useEffect(() => {
    if (!gameState) return;

    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');

    const now = performance.now();
    const last = lastDrawTimeRef.current || now;
    const dt = Math.min((now - last) / 1000, 0.05);
    lastDrawTimeRef.current = now;

    // --- Sons e efeitos por transição de estado ---
    if (prevRoundOverRef.current === true && gameState.round_over === false) {
      sfx.roundStart();
    }
    prevRoundOverRef.current = gameState.round_over;

    const tiles = gameState.arena_tiles || {};
    Object.values(tiles).forEach((tile) => {
      const prev = prevTilesRef.current[tile.id];
      if (prev && prev.falling === false && tile.is_falling) {
        sfx.tileBreak();
        addShake(7);
        spawnParticles(
          tile.x + tileSize / 2,
          tile.y + tileSize / 2,
          PALETTE.terracotta,
          10,
          160
        );
        spawnParticles(
          tile.x + tileSize / 2,
          tile.y + tileSize / 2,
          PALETTE.soilCrumb,
          8,
          130
        );
      }
      prevTilesRef.current[tile.id] = { falling: tile.is_falling, active: tile.is_active };
    });
    Object.keys(prevTilesRef.current).forEach((id) => {
      if (!tiles[id]) delete prevTilesRef.current[id];
    });

    // --- Shake ---
    let shakeX = 0;
    let shakeY = 0;
    if (shakeRef.current > 0.1) {
      shakeX = (Math.random() * 2 - 1) * shakeRef.current;
      shakeY = (Math.random() * 2 - 1) * shakeRef.current;
      shakeRef.current *= 0.85;
    }

    ctx.save();
    ctx.translate(shakeX, shakeY);

    // Fundo: céu sereno em gradiente linear
    ctx.clearRect(0, 0, arenaW, arenaH);
    const sky = ctx.createLinearGradient(0, 0, 0, arenaH);
    sky.addColorStop(0, PALETTE.skyTop);
    sky.addColorStop(0.5, PALETTE.skyMid);
    sky.addColorStop(1, PALETTE.skyBottom);
    ctx.fillStyle = sky;
    ctx.fillRect(0, 0, arenaW, arenaH);

    // Nuvens suaves
    drawCloud(ctx, 130, 70, 58);
    drawCloud(ctx, 640, 120, 42);
    drawCloud(ctx, 380, 40, 34);

    // Ilhas: blocos arredondados estilo flat com sombra, grama e árvores
    const activeKeys = new Set();
    Object.values(tiles).forEach((t) => {
      if (t.is_active) activeKeys.add(t.id);
    });
    const hasTileAbove = (t) => {
      const col = Math.round(t.x / tileSize);
      const row = Math.round(t.y / tileSize);
      return activeKeys.has(`tile_${row - 1}_${col}`);
    };

    Object.values(tiles).forEach((tile) => {
      if (!tile.is_active) return;

      const x = tile.x;
      const y = tile.y;
      const s = tileSize;
      const falling = tile.is_falling;
      // Autotile: "top" = grama, "mid" = terra, "bottom" = ponta de pedra.
      const kind = tile.kind || (hasTileAbove(tile) ? 'mid' : 'top');

      // Sombra sólida deslocada (profundidade)
      ctx.fillStyle = PALETTE.islandShadow;
      roundRectPath(ctx, x + 5, y + 7, s, s, 12);
      ctx.fill();

      // Corpo da ilha (solo em gradiente suave)
      const soilGrad = ctx.createLinearGradient(0, y, 0, y + s);
      soilGrad.addColorStop(0, PALETTE.soilTop);
      soilGrad.addColorStop(1, PALETTE.soilBottom);
      ctx.fillStyle = falling ? PALETTE.falling : soilGrad;
      roundRectPath(ctx, x, y, s, s, 12);
      ctx.fill();

      if (falling) {
        // Rachaduras do bloco caindo
        ctx.strokeStyle = PALETTE.fallingDark;
        ctx.lineWidth = 3;
        ctx.beginPath();
        ctx.moveTo(x + s * 0.3, y + s * 0.2);
        ctx.lineTo(x + s * 0.5, y + s * 0.5);
        ctx.lineTo(x + s * 0.38, y + s * 0.8);
        ctx.stroke();
      }

      // Faixa de grama apenas na superfície ("top")
      if (kind === 'top') {
        ctx.fillStyle = PALETTE.grassTop;
        roundRectPath(ctx, x, y, s, 20, 10);
        ctx.fill();
        ctx.fillStyle = PALETTE.grassEdge;
        roundRectPath(ctx, x, y + 14, s, 8, 4);
        ctx.fill();
      }

      // Ponta de pedra na base da ilha ("bottom")
      if (kind === 'bottom') {
        ctx.fillStyle = PALETTE.soilBorder;
        ctx.beginPath();
        ctx.moveTo(x + s * 0.2, y + s * 0.6);
        ctx.lineTo(x + s * 0.5, y + s * 0.92);
        ctx.lineTo(x + s * 0.8, y + s * 0.6);
        ctx.closePath();
        ctx.fill();
      }

      // Contorno marrom suave
      ctx.strokeStyle = PALETTE.soilBorder;
      ctx.lineWidth = 2;
      roundRectPath(ctx, x, y, s, s, 12);
      ctx.stroke();

      // Árvore em blocos de superfície (autotile "top")
      if (!falling && kind === 'top' && tileHash(x, y) % 3 === 0) {
        drawTree(ctx, x, y, s);
      }
    });

    // Partículas
    const parts = particlesRef.current;
    for (let i = parts.length - 1; i >= 0; i--) {
      const p = parts[i];
      p.x += p.vx * dt;
      p.y += p.vy * dt;
      p.vy += 220 * dt;
      p.life -= p.decay * dt;
      if (p.life <= 0) parts.splice(i, 1);
    }
    parts.forEach((p) => {
      ctx.globalAlpha = Math.max(p.life, 0);
      ctx.fillStyle = p.color;
      ctx.beginPath();
      ctx.arc(p.x, p.y, 3 * p.life, 0, Math.PI * 2);
      ctx.fill();
    });
    ctx.globalAlpha = 1;

    // Jogadores
    const players = Object.values(gameState.players || {});
    const seenIds = new Set();

    players.forEach((player) => {
      seenIds.add(player.id);

      if (!animsRef.current[player.id]) {
        animsRef.current[player.id] = {
          trail: [],
          rotation: 0,
          prevVy: 0,
          prevOnGround: false,
          prevDead: false,
          prevJumpsUsed: 0,
          landSquash: 0,
        };
      }
      const anim = animsRef.current[player.id];

      // Sons para o próprio jogador
      if (player.id === myPlayerId) {
        const justLanded =
          player.on_ground && !anim.prevOnGround && anim.prevVy > 8;
        const justDied = player.is_dead && !anim.prevDead;
        const justDoubleJumped =
          !player.on_ground && player.jumps_used === 1 && anim.prevJumpsUsed === 0;
        const lostLife =
          prevLivesRef.current !== null && player.lives < prevLivesRef.current;

        if (justLanded) sfx.land();
        if (justDied) sfx.death();
        if (justDoubleJumped) {
          sfx.doubleJump();
          spawnParticles(player.x, player.y, PALETTE.gold, 10, 130);
        }
        if (lostLife) {
          sfx.respawn();
          spawnParticles(player.x, player.y, PALETTE.amber, 14, 150);
          addShake(6);
        }
        prevLivesRef.current = player.lives;
      }

      anim.prevDead = player.is_dead;
      anim.prevJumpsUsed = player.jumps_used;

      if (player.is_dead) {
        drawGhost(ctx, player, playerRadius);
      } else {
        drawPlayerBall(ctx, player, anim, now, dt, playerRadius);
      }

      // Barra de cooldown do Dash (apenas para o próprio jogador)
      if (player.id === myPlayerId && !player.is_dead) {
        const cd = player.dash_cd || 0;
        const maxCd = cfg.dash_cooldown || 1.5;
        const barW = playerRadius * 2;
        const barY = player.y + playerRadius + 8;
        ctx.fillStyle = 'rgba(90,70,40,0.25)';
        ctx.fillRect(player.x - playerRadius, barY, barW, 5);
        ctx.fillStyle = cd > 0 ? PALETTE.amber : PALETTE.goldStrong;
        ctx.fillRect(
          player.x - playerRadius,
          barY,
          barW * Math.max(0, 1 - cd / maxCd),
          5
        );
      }
    });

    // Juice de impacto: detecta pares vivos cruzando a distância mínima
    const alivePlayers = players.filter((p) => !p.is_dead);
    for (let i = 0; i < alivePlayers.length; i++) {
      for (let j = i + 1; j < alivePlayers.length; j++) {
        const a = alivePlayers[i];
        const b = alivePlayers[j];
        const key = a.id < b.id ? `${a.id}|${b.id}` : `${b.id}|${a.id}`;
        const dist = Math.hypot(a.x - b.x, a.y - b.y);
        const prev = impactsRef.current[key] ?? Infinity;
        const hitDist = playerRadius * 2 + 2;
        if (dist < hitDist && prev >= hitDist) {
          sfx.hit();
          addShake(5);
          spawnParticles((a.x + b.x) / 2, (a.y + b.y) / 2, PALETTE.terracotta, 10, 200);
          spawnParticles((a.x + b.x) / 2, (a.y + b.y) / 2, PALETTE.gold, 8, 160);
        }
        impactsRef.current[key] = dist;
      }
    }
    Object.keys(impactsRef.current).forEach((key) => {
      const [ida, idb] = key.split('|');
      const found = players.some((p) => p.id === ida) && players.some((p) => p.id === idb);
      if (!found) delete impactsRef.current[key];
    });

    // Limpa animações de jogadores que saíram
    Object.keys(animsRef.current).forEach((id) => {
      if (!seenIds.has(id)) delete animsRef.current[id];
    });

    ctx.restore();
  }, [gameState, cfg, myPlayerId, spawnParticles, addShake, arenaW, arenaH, tileSize, playerRadius]);

  // =======================
  // Scores
  // =======================
  useEffect(() => {
    const fetchScores = async () => {
      try {
        const res = await fetch(SCORES_API_URL);
        const data = await res.json();
        setScores(data);
      } catch (err) {
        console.error('Erro ao buscar scores', err);
      }
    };

    fetchScores();
    const interval = setInterval(fetchScores, 10000);
    return () => clearInterval(interval);
  }, []);

  const myPlayer =
    gameState?.players?.[myPlayerId] ?? null;

  // Recorde pessoal
  useEffect(() => {
    if (myPlayer && !myPlayer.is_dead) {
      setBestScore((b) => Math.max(b, myPlayer.score));
    }
  }, [myPlayer]);

  const othersAlive =
    gameState && gameState.players
      ? Object.values(gameState.players).some((p) => !p.is_dead)
      : false;

  const aliveCount =
    gameState && gameState.players
      ? Object.values(gameState.players).filter((p) => !p.is_dead).length
      : 0;

  const overlaySx = {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 2,
    bgcolor: PALETTE.creamOverlay,
    textAlign: 'center',
    p: 3,
  };

  // =======================
  // UI
  // =======================
  return (
    <Box
      sx={{
        minHeight: '100vh',
        background:
          'linear-gradient(180deg, #fdf0d8 0%, #f6e3c5 45%, #efc58b 100%)',
      }}
    >
      <Container maxWidth="xl" sx={{ py: 6 }}>
        <Typography
          variant="h2"
          align="center"
          sx={{
            mb: 6,
            fontWeight: 800,
            color: PALETTE.textBrown,
            background: 'linear-gradient(90deg, #d99a2b, #c96f4a)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
          }}
        >
          SURVIVE THE DESTRUCTION
        </Typography>

        <Grid container spacing={4}>
          <Grid item xs={12} lg={8} display="flex" justifyContent="center">
            <Paper
              sx={{
                position: 'relative',
                width: '100%',
                maxWidth: arenaW,
                borderRadius: 4,
                overflow: 'hidden',
                border: `2px solid ${PALETTE.borderSoft}`,
                bgcolor: PALETTE.skyTop,
                boxShadow: '0 18px 40px rgba(90,70,40,0.18)',
              }}
            >
              <canvas
                ref={canvasRef}
                width={arenaW}
                height={arenaH}
                style={{ display: 'block', width: '100%', height: 'auto' }}
              />

              {!gameStarted && (
                <Box sx={overlaySx}>
                  <Typography variant="h5" color={PALETTE.terracottaStrong} fontWeight={700}>
                    Escolha seu nome
                  </Typography>
                  <TextField
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    inputProps={{ maxLength: 20 }}
                    size="small"
                    sx={{ width: 260, bgcolor: 'rgba(255,255,255,0.6)', borderRadius: 2 }}
                  />
                  <Button
                    variant="contained"
                    size="large"
                    sx={{ bgcolor: PALETTE.olive, '&:hover': { bgcolor: '#5f6d30' } }}
                    onClick={handleStartGame}
                  >
                    INICIAR JOGO
                  </Button>
                  <Typography variant="body2" color={PALETTE.textSoft}>
                    Use A/D ou ←/→ para mover, W ou Espaço para pular (×2 no ar).
                    Shift para Dash (empurra oponentes). Enter também inicia.
                  </Typography>
                </Box>
              )}

              {gameStarted && gameState && gameState.round_over && (
                <Box sx={overlaySx}>
                  <Typography variant="h4" color={PALETTE.fallingDark} fontWeight={800}>
                    NOVA RODADA
                  </Typography>
                  <Typography
                    variant="h1"
                    color={PALETTE.goldStrong}
                    fontWeight={900}
                    sx={{ fontSize: '6rem' }}
                  >
                    {gameState.countdown}
                  </Typography>
                </Box>
              )}

              {gameStarted &&
                gameState &&
                !gameState.round_over &&
                myPlayer &&
                myPlayer.is_dead && (
                  <Box sx={overlaySx}>
                    <Typography variant="h4" color={PALETTE.fallingDark} fontWeight={800}>
                      VOCÊ CAIU!
                    </Typography>
                    <Typography variant="body2" color={PALETTE.textSoft}>
                      {othersAlive
                        ? 'Suas vidas acabaram. Observe os outros ou tente novamente.'
                        : 'A rodada terminou. Aguarde a nova rodada...'}
                    </Typography>
                    <Button
                      variant="contained"
                      size="large"
                      sx={{ bgcolor: PALETTE.olive, '&:hover': { bgcolor: '#5f6d30' } }}
                      onClick={sendRestart}
                    >
                      TENTAR NOVAMENTE
                    </Button>
                  </Box>
                )}

              {isTouch && gameStarted && (
                <Box
                  sx={{
                    position: 'absolute',
                    inset: 0,
                    pointerEvents: 'none',
                    display: 'flex',
                    alignItems: 'flex-end',
                    justifyContent: 'space-between',
                    p: 2,
                  }}
                >
                  <Button
                    variant="contained"
                    sx={{
                      pointerEvents: 'auto',
                      minWidth: 64,
                      fontSize: '1.4rem',
                      bgcolor: PALETTE.olive,
                      '&:hover': { bgcolor: '#5f6d30' },
                    }}
                    onPointerDown={touchPress('left')}
                    onPointerUp={touchRelease('left')}
                    onPointerLeave={touchRelease('left')}
                  >
                    ◀
                  </Button>
                  <Button
                    variant="contained"
                    sx={{
                      pointerEvents: 'auto',
                      minWidth: 64,
                      fontSize: '1.4rem',
                      bgcolor: PALETTE.olive,
                      '&:hover': { bgcolor: '#5f6d30' },
                    }}
                    onPointerDown={touchPress('right')}
                    onPointerUp={touchRelease('right')}
                    onPointerLeave={touchRelease('right')}
                  >
                    ▶
                  </Button>
                  <Button
                    variant="contained"
                    sx={{
                      pointerEvents: 'auto',
                      minWidth: 64,
                      fontSize: '1.4rem',
                      bgcolor: PALETTE.goldStrong,
                      '&:hover': { bgcolor: '#c2851e' },
                    }}
                    onPointerDown={touchPress('jump')}
                    onPointerUp={touchRelease('jump')}
                    onPointerLeave={touchRelease('jump')}
                  >
                    ⬆
                  </Button>
                  <Button
                    variant="contained"
                    sx={{
                      pointerEvents: 'auto',
                      minWidth: 64,
                      fontSize: '1.4rem',
                      bgcolor: PALETTE.terracottaStrong,
                      '&:hover': { bgcolor: '#ab5a3c' },
                    }}
                    onPointerDown={touchPress('dash')}
                    onPointerUp={touchRelease('dash')}
                    onPointerLeave={touchRelease('dash')}
                  >
                    ⚡
                  </Button>
                </Box>
              )}
            </Paper>
          </Grid>

          <Grid item xs={12} lg={4}>
            <Box display="flex" flexDirection="column" gap={3}>
              <Leaderboard scores={scores} />

              <Paper
                sx={{
                  p: 3,
                  borderRadius: 4,
                  bgcolor: PALETTE.cream,
                  border: `1px solid rgba(201,111,74,0.45)`,
                  boxShadow: '0 10px 30px rgba(90,70,40,0.12)',
                }}
              >
                <Typography variant="h6" color={PALETTE.textBrown}>
                  STATUS
                </Typography>
                <Typography>Nome: {nickname}</Typography>
                <Typography>
                  Tempo: {myPlayer ? `${myPlayer.score}s` : '0s'}
                </Typography>
                <Typography color={PALETTE.goldStrong} fontWeight="bold">
                  Recorde: {bestScore}s
                </Typography>
                <Typography>Rodada: {gameState?.round ?? 1}</Typography>
                <Typography>Vivos: {aliveCount}</Typography>
                <Typography>
                  Vidas:{' '}
                  {'❤'.repeat(myPlayer?.lives ?? cfg.max_lives ?? 3) ||
                    '0'}
                </Typography>
                {myPlayer && myPlayer.is_dead && (
                  <Typography color="error.main">💀 Espectando...</Typography>
                )}
              </Paper>
            </Box>
          </Grid>
        </Grid>
      </Container>
    </Box>
  );
}
