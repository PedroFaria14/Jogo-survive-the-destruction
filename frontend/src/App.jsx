import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import {
  Box,
  Container,
  Paper,
  Typography,
  Button,
  TextField,
  List,
  ListItem,
  ListItemText,
  Chip,
  CircularProgress,
} from '@mui/material';

import { sfx } from './sfx.js';
import { STRINGS, detectLanguage, translate } from './i18n.js';

// Telas de carregamento sorteadas: carrega automaticamente qualquer imagem
// com prefixo "carregamento" em frontend/assets/ (sem mexer no código).
// "-mobile" = variante em retrato para telas verticais (celular em pé).
const loadingImages = [];
const loadingImagesMobile = [];
for (const [key, url] of Object.entries(
  import.meta.glob('../assets/carregamento*.jpeg', {
    eager: true,
    import: 'default',
  })
)) {
  if (key.includes('-mobile')) loadingImagesMobile.push(url);
  else loadingImages.push(url);
}

// =======================
// Configurações
// =======================
// URLs configuráveis via variáveis de ambiente Vite (VITE_*) com fallback local.
const GAME_WS_URL = import.meta.env.VITE_WS_URL || 'wss://jogo-survive-the-destruction.onrender.com/ws';
const API_BASE_URL = import.meta.env.VITE_API_URL || 'https://jogo-survive-the-destruction.onrender.com';
const SCORES_API_URL = `${API_BASE_URL}/api/scores`;
const CONFIG_API_URL = `${API_BASE_URL}/api/config`;
const HEALTH_API_URL = `${API_BASE_URL}/api/health`;

// Fallback das constantes do jogo (o servidor fornece via /api/config)
const DEFAULT_CFG = {
  arena_width: 800,
  arena_height: 1000,
  tile_size: 100,
  player_radius: 25,
  move_speed: 8,
  jump_force: 20,
  gravity: 1.5,
  break_interval: 3,
  max_air_jumps: 1,
  max_lives: 3,
  dash_speed: 20,
  dash_cooldown: 1.5,
  dash_duration: 0.25,
  knockback_dash: 11,
  restitution: 0.85,
  min_islands: 3,
  max_islands: 7,
  island_width_min: 1,
  island_width_max: 6,
  max_gap_cols: 3,
  powerup_interval: 4,
  powerup_lifetime: 6,
  red_duration: 8,
  purple_duration: 6,
  blue_duration: 8,
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
  // Power-ups
  redMushTop: '#ff6b5e',
  redMushDark: '#c93a3a',
  redMushSpot: '#ffe9dc',
  purpleMushTop: '#b06fd6',
  purpleMushDark: '#7d44a3',
  purpleMushSpot: '#f3e2ff',
  crystalLight: '#cfefff',
  crystalMid: '#5fb9e8',
  crystalDark: '#2f7fb5',
  buffTintRed: 'rgba(255,107,94,0.22)',
  buffTintPurple: 'rgba(176,111,214,0.22)',
  buffTintBlue: 'rgba(95,185,232,0.22)',
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

// Converte "#rrggbb" para {r, g, b} (ou null se inválido).
function hexToRgb(hex) {
  const m = /^#([0-9a-fA-F]{6})$/.exec(hex || '');
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
}

// Escurece/clareia um hex por fator (1 = mantém, <1 escurece, >1 clareia).
function shadeHex(hex, f) {
  const c = hexToRgb(hex);
  if (!c) return hex;
  const to = (v) => Math.round(Math.min(255, Math.max(0, v * f)));
  return `rgb(${to(c.r)}, ${to(c.g)}, ${to(c.b)})`;
}

// Monta a paleta visual da bolinha a partir da cor base escolhida pelo jogador.
// Cor inválida/vazia (ex.: servidor antigo) usa o âmbar padrão.
function ballPalette(hex) {
  const c = hexToRgb(hex);
  if (!c) {
    return {
      base: PALETTE.ballMid,
      light: PALETTE.ballHi,
      dark: PALETTE.ballDark,
      marker: PALETTE.ballMarker,
      outline: PALETTE.ballOutline,
      trail: PALETTE.trailRGB,
      glow: PALETTE.ballGlow,
    };
  }
  return {
    base: hex,
    light: shadeHex(hex, 1.5),
    dark: shadeHex(hex, 0.6),
    marker: shadeHex(hex, 0.35),
    outline: `rgba(${c.r}, ${c.g}, ${c.b}, 0.6)`,
    trail: `${c.r},${c.g},${c.b}`,
    glow: hex,
  };
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

// Cogumelo (red = Tanque, purple = Velocista): capa arredondada + caule.
function drawMushroom(ctx, x, y, type) {
  const red = type === 'red_mushroom';
  const cap = red ? PALETTE.redMushTop : PALETTE.purpleMushTop;
  const capDark = red ? PALETTE.redMushDark : PALETTE.purpleMushDark;
  const spot = red ? PALETTE.redMushSpot : PALETTE.purpleMushSpot;

  // Caule
  ctx.fillStyle = '#f6ead2';
  roundRectPath(ctx, x - 4, y + 2, 8, 9, 3);
  ctx.fill();

  // Capa
  ctx.fillStyle = cap;
  ctx.beginPath();
  ctx.moveTo(x - 13, y + 2);
  ctx.quadraticCurveTo(x - 13, y - 13, x, y - 13);
  ctx.quadraticCurveTo(x + 13, y - 13, x + 13, y + 2);
  ctx.closePath();
  ctx.fill();

  // Brilho na capa
  ctx.fillStyle = 'rgba(255,255,255,0.35)';
  ctx.beginPath();
  ctx.ellipse(x - 5, y - 7, 4, 2.4, -0.5, 0, Math.PI * 2);
  ctx.fill();

  // Pintas
  ctx.fillStyle = spot;
  ctx.beginPath();
  ctx.arc(x - 6, y - 4, 1.8, 0, Math.PI * 2);
  ctx.fill();
  ctx.beginPath();
  ctx.arc(x + 6, y - 2, 1.4, 0, Math.PI * 2);
  ctx.fill();

  // Borda da capa
  ctx.strokeStyle = capDark;
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.moveTo(x - 13, y + 2);
  ctx.quadraticCurveTo(x - 13, y - 13, x, y - 13);
  ctx.quadraticCurveTo(x + 13, y - 13, x + 13, y + 2);
  ctx.stroke();
}

// Cristal azul (Planar): gema facetada flutuante.
function drawCrystal(ctx, x, y) {
  ctx.fillStyle = PALETTE.crystalDark;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x + 9, y - 4);
  ctx.lineTo(x + 6, y + 9);
  ctx.lineTo(x - 6, y + 9);
  ctx.lineTo(x - 9, y - 4);
  ctx.closePath();
  ctx.fill();

  ctx.fillStyle = PALETTE.crystalMid;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x + 9, y - 4);
  ctx.lineTo(x, y);
  ctx.closePath();
  ctx.fill();

  ctx.fillStyle = PALETTE.crystalLight;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x - 9, y - 4);
  ctx.lineTo(x - 3, y - 4);
  ctx.closePath();
  ctx.fill();

  ctx.strokeStyle = PALETTE.crystalDark;
  ctx.lineWidth = 1.2;
  ctx.beginPath();
  ctx.moveTo(x, y - 14);
  ctx.lineTo(x, y);
  ctx.moveTo(x - 9, y - 4);
  ctx.lineTo(x + 9, y - 4);
  ctx.stroke();
}

// Drop de power-up: glow + shape + bobbing suave (sempre visível a todos).
function drawPowerUpDrop(ctx, pu, now) {
  const bob = Math.sin(now / 320 + (pu.id ? pu.id.length : 0)) * 2;
  const y = pu.y + bob;
  const glowColor =
    pu.type === 'red_mushroom'
      ? PALETTE.redMushTop
      : pu.type === 'purple_mushroom'
        ? PALETTE.purpleMushTop
        : PALETTE.crystalMid;

  ctx.save();
  ctx.translate(pu.x, y);

  // Anel de brilho pulsante
  ctx.shadowColor = glowColor;
  ctx.shadowBlur = 14;
  ctx.fillStyle = glowColor;
  ctx.globalAlpha = 0.16 + Math.sin(now / 260) * 0.05;
  ctx.beginPath();
  ctx.arc(0, 0, 15, 0, Math.PI * 2);
  ctx.fill();
  ctx.globalAlpha = 1;

  if (pu.type === 'blue_crystal') {
    drawCrystal(ctx, 0, 0);
  } else {
    drawMushroom(ctx, 0, 0, pu.type);
  }
  ctx.restore();
}

// =======================
// Desenho da bola (animado)
// =======================
function drawPlayerBall(ctx, player, anim, now, dt, radius, buff) {
  // Paleta da bolinha a partir da cor escolhida pelo jogador.
  const pal = ballPalette(player.color);
  // Cores e glow do rastro conforme o buff ativo.
  const trailColor =
    buff === 'red_mushroom'
      ? PALETTE.redMushTop
      : buff === 'purple_mushroom'
        ? PALETTE.purpleMushTop
        : buff === 'blue_crystal'
          ? PALETTE.crystalMid
          : pal.trail;
  const glowColor =
    buff === 'red_mushroom'
      ? PALETTE.redMushDark
      : buff === 'purple_mushroom'
        ? PALETTE.purpleMushDark
        : buff === 'blue_crystal'
          ? PALETTE.crystalLight
          : pal.glow;

  // Trilha/trail
  anim.trail.push({ x: player.x, y: player.y });
  if (anim.trail.length > TRAIL_LENGTH) anim.trail.shift();

  anim.trail.forEach((t, i) => {
    const alpha = (i / TRAIL_LENGTH) * 0.22;
    const r = radius * 0.55 * (i / TRAIL_LENGTH);
    ctx.fillStyle = `rgba(${trailColor},${alpha.toFixed(3)})`;
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

  // Pulso/brilho (cor depende do buff)
  ctx.shadowColor = glowColor;
  ctx.shadowBlur = 16 + Math.sin(now / GLOW_PULSE_SPEED) * 6;

  // Tinta do buff sobre a bola
  if (buff) {
    const tint =
      buff === 'red_mushroom'
        ? PALETTE.buffTintRed
        : buff === 'purple_mushroom'
          ? PALETTE.buffTintPurple
          : PALETTE.buffTintBlue;
    ctx.fillStyle = tint;
    ctx.beginPath();
    ctx.arc(0, 0, radius, 0, Math.PI * 2);
    ctx.fill();
  }

  const grad = ctx.createRadialGradient(-6, -6, 4, 0, 0, radius);
  grad.addColorStop(0, pal.light);
  grad.addColorStop(0.55, pal.base);
  grad.addColorStop(1, pal.dark);
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.shadowBlur = 0;

  // Marcador de rotação
  ctx.save();
  ctx.rotate(anim.rotation);
  ctx.fillStyle = pal.marker;
  ctx.beginPath();
  ctx.arc(radius * 0.55, 0, 4.5, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();

  // Contorno
  ctx.strokeStyle = pal.outline;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(0, 0, radius, 0, Math.PI * 2);
  ctx.stroke();

  // Planar: hélice de vento ao planar (partículas espirais de gelo)
  if (buff === 'blue_crystal') {
    ctx.strokeStyle = 'rgba(207,239,255,0.7)';
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    for (let a = 0; a < Math.PI * 2; a += 0.3) {
      const rx = radius * 1.35 * Math.cos(a);
      const ry = radius * 1.35 * Math.sin(a) * 0.5;
      ctx.moveTo(rx * 0.85, ry * 0.85);
      ctx.lineTo(rx, ry);
    }
    ctx.stroke();
  }
  ctx.restore();
}

// Jogador morto: "fantasma" translúcido (modo espectador) na cor do jogador.
function drawGhost(ctx, player, radius) {
  const c = hexToRgb(player.color);
  const fill = c ? `rgba(${c.r}, ${c.g}, ${c.b}, 0.18)` : PALETTE.ghostFill;
  const stroke = c ? `rgba(${c.r}, ${c.g}, ${c.b}, 0.4)` : PALETTE.ghostStroke;
  ctx.fillStyle = fill;
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.fill();
  ctx.strokeStyle = stroke;
  ctx.lineWidth = 1.5;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.arc(player.x, player.y, radius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.setLineDash([]);
}

// Ícone do buff flutuando acima da cabeça (visível para todos os jogadores).
function drawBuffIcon(ctx, x, y, buff, now) {
  const bob = Math.sin(now / 280) * 2;
  ctx.save();
  ctx.translate(x, y - 34 + bob);
  if (buff === 'blue_crystal') {
    drawCrystal(ctx, 0, 0);
  } else {
    drawMushroom(ctx, 0, 0, buff);
  }
  ctx.restore();
}

// =======================
// Leaderboard
// =======================
const Leaderboard = React.memo(({ scores, highlightName, lang }) => {
  const dict = STRINGS[lang];
  return (
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
        {dict['leaderboard.title']}
      </Typography>

      {scores.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {dict['leaderboard.empty']}
        </Typography>
      ) : (
        <List dense>
          {scores.map((score, index) => {
            const name =
              score.name ||
              (score.player_id
                ? `Player_${score.player_id.slice(-4)}`
                : dict['leaderboard.anon']);
            const hl = highlightName && name === highlightName;
            return (
              <ListItem
                key={index}
                sx={{
                  bgcolor: hl
                    ? 'rgba(217,154,43,0.28)'
                    : 'rgba(240,224,196,0.6)',
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
                  primary={name}
                  primaryTypographyProps={{ fontWeight: hl ? 800 : 400 }}
                />

                <Typography color={PALETTE.goldStrong} fontWeight="bold">
                  {score.score_seconds}s
                </Typography>
              </ListItem>
            );
          })}
        </List>
      )}
    </Paper>
  );
});

// Verdadeiro se o evento veio de um campo de texto (não acionar atalhos).
function isTypingTarget(e) {
  const t = e.target;
  return t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable);
}

// =======================
// App
// =======================
export default function App() {
  const [cfg, setCfg] = useState(DEFAULT_CFG);
  const [scores, setScores] = useState([]);
  const [myPlayerId, setMyPlayerId] = useState(null);
  const [gameStarted, setGameStarted] = useState(false);
  const [nickname, setNickname] = useState(
    () => `Jogador${Math.floor(1000 + Math.random() * 9000)}`
  );
  const [ballColor, setBallColor] = useState(
    () => localStorage.getItem('ballColor') || '#f2b544'
  );
  const [bestScore, setBestScore] = useState(0);
  const [isTouch, setIsTouch] = useState(false);
  const [isLandscape, setIsLandscape] = useState(true);
  const [backendStatus, setBackendStatus] = useState('checking');
  const [lang, setLang] = useState(detectLanguage);
  const t = useCallback((key, vars) => translate(STRINGS[lang], key, vars), [lang]);
  const toggleLang = useCallback(() => {
    setLang((l) => (l === 'pt' ? 'en' : 'pt'));
  }, []);
  // Sorteio de uma tela de carregamento por visita (por orientação).
  const carregamentoImgRef = useRef(null);
  const carregamentoImg = useMemo(() => {
    const pool = isLandscape ? loadingImages : loadingImagesMobile;
    const idx =
      carregamentoImgRef.current ??
      (carregamentoImgRef.current = Math.floor(Math.random() * pool.length));
    return pool[idx % pool.length];
  }, [isLandscape]);

  useEffect(() => {
    try {
      localStorage.setItem('lang', lang);
    } catch {
      // localStorage indisponível: ignora.
    }
    document.documentElement.lang = lang === 'pt' ? 'pt-BR' : 'en';
  }, [lang]);

  // O estado do jogo (60 FPS) é guardado em ref e desenhado no canvas via
  // requestAnimationFrame. A UI (overlays/sidebar) usa um snapshot `ui`
  // atualizado em baixa frequência, evitando re-render do React a cada frame.
  const gameStateRef = useRef(null);
  const [ui, setUi] = useState({
    round: 1,
    round_over: false,
    countdown: 0,
    drop_countdown: 0,
    arena_width: DEFAULT_CFG.arena_width,
    arena_height: DEFAULT_CFG.arena_height,
    room_id: '',
    room_players: 0,
    room_capacity: 0,
    players: {},
    myPlayer: null,
    aliveCount: 0,
    aliveAny: false,
  });

  // Ranking da partida atual: todos os jogadores ordenados por score (tempo
  // sobrevivido), usado no overlay de derrota.
  const matchRanking = useMemo(() => {
    return Object.values(ui.players || {})
      .filter((p) => p && p.id)
      .sort((a, b) => (b.score || 0) - (a.score || 0));
  }, [ui.players]);
  const myPos =
    ui.myPlayer && matchRanking.length > 0
      ? matchRanking.findIndex((p) => p.id === ui.myPlayer.id) + 1
      : 0;

  // Prompt persistente de "outra partida" (disparado ao perder as 3 vidas).
  // Fica aberto até o jogador decidir, mesmo que a rodada reinicie por baixo.
  const [deathPrompt, setDeathPrompt] = useState(false);
  const [deathInfo, setDeathInfo] = useState(null);

  const socketRef = useRef(null);
  const canvasRef = useRef(null);
  const startedRef = useRef(false);
  const nicknameRef = useRef(nickname);
  const ballColorRef = useRef(ballColor);
  const myPlayerIdRef = useRef(null);
  const keysRef = useRef({ left: false, right: false, jump: false, dash: false });
  const animsRef = useRef({});
  const particlesRef = useRef([]);
  const shakeRef = useRef(0);
  const prevTilesRef = useRef({});
  const prevRoundOverRef = useRef(null);
  const impactsRef = useRef({});
  const prevLivesRef = useRef(null);
  const prevDeadRef = useRef(false);
  const lastUiRef = useRef(0);
  const cfgRef = useRef(cfg);
  const powerUpPrevRef = useRef({});
  const prevBuffRef = useRef(null);

  // Dimensões da arena: o servidor envia por rodada (mapa procedural muda).
  const arenaW = ui.arena_width ?? cfg.arena_width;
  const arenaH = ui.arena_height ?? cfg.arena_height;

  // =======================
  // Start Game
  // =======================
  const handleStartGame = useCallback(() => {
    sfx.unlock();
    sfx.click();
    startedRef.current = true;
    setGameStarted(true);
    gameStateRef.current = null;
    prevDeadRef.current = false;
    setDeathPrompt(false);
    setDeathInfo(null);
    setUi((u) => ({
      ...u,
      round_over: false,
      countdown: 0,
      players: {},
      myPlayer: null,
      aliveCount: 0,
      aliveAny: false,
    }));
    // Tela cheia no celular (Android): esconde a barra do navegador. iOS ignora.
    if (isTouch && document.documentElement.requestFullscreen) {
      document.documentElement
        .requestFullscreen()
        .catch(() => {});
    }
    // Trava a orientação em paisagem no celular (Android; iOS ignora).
    if (isTouch && window.screen?.orientation?.lock) {
      window.screen.orientation
        .lock('landscape')
        .catch(() => {});
    }
  }, [isTouch]);

  // Sai da tela cheia (mobile) sem recarregar a página.
  const exitFullscreen = useCallback(() => {
    if (document.fullscreenElement && document.exitFullscreen) {
      document.exitFullscreen().catch(() => {});
    }
  }, []);

  // =======================
  // Enviar comandos (estado completo de input)
  // =======================
  const sendFrame = useCallback(() => {
    const s = socketRef.current;
    const input = {
      left: keysRef.current.left,
      right: keysRef.current.right,
      jump: keysRef.current.jump,
      dash: keysRef.current.dash,
    };
    if (s && s.readyState === WebSocket.OPEN && startedRef.current) {
      s.send(JSON.stringify({ type: 'input', ...input }));
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
  // Entrar em outra partida (reinicia tudo: volta à tela inicial e reconecta)
  // =======================
  const startNewMatch = useCallback(() => {
    sfx.click();
    window.location.reload();
  }, []);

  // =======================
  // Sair da partida (volta ao menu sem recarregar a página)
  // =======================
  const handleExit = useCallback(() => {
    sfx.click();
    startedRef.current = false;
    gameStateRef.current = null;
    keysRef.current = { left: false, right: false, jump: false, dash: false };
    animsRef.current = {};
    particlesRef.current = [];
    shakeRef.current = 0;
    prevTilesRef.current = {};
    prevRoundOverRef.current = null;
    impactsRef.current = {};
    prevLivesRef.current = null;
    prevDeadRef.current = false;
    lastUiRef.current = 0;
    powerUpPrevRef.current = {};
    prevBuffRef.current = null;
    setDeathPrompt(false);
    setDeathInfo(null);
    setMyPlayerId(null);
    setUi((u) => ({
      ...u,
      round_over: false,
      countdown: 0,
      players: {},
      myPlayer: null,
      aliveCount: 0,
      aliveAny: false,
    }));
    setGameStarted(false);
    if (document.fullscreenElement && document.exitFullscreen) {
      document.exitFullscreen().catch(() => {});
    }
  }, []);

  // =======================
  // Config compartilhada + verificação do backend (tela de carregamento)
  // =======================
  // Loop de auto-retry no /api/health: a tela de carregamento só sai quando o
  // servidor (Render) ligar. Aguenta o cold start (até 5 minutos) antes de
  // renderizar o erro.
  const checkBackend = useCallback(async () => {
    setBackendStatus('checking');
    const deadline = Date.now() + 5 * 60 * 1000; // 5 minutos de tentativas

    while (Date.now() < deadline) {
      let healthOk = false;
      const ctrl = new AbortController();
      const timer = setTimeout(() => ctrl.abort(), 6000);
      try {
        const res = await fetch(HEALTH_API_URL, { signal: ctrl.signal });
        healthOk = res.ok;
      } catch {
        healthOk = false;
      } finally {
        clearTimeout(timer);
      }

      if (healthOk) {
        try {
          const res = await fetch(CONFIG_API_URL);
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          const data = await res.json();
          setCfg(data);
          setBackendStatus('ready');
          return;
        } catch {
          // Health ok mas config falhou: tenta de novo no próximo ciclo.
        }
      }

      await new Promise((resolve) => setTimeout(resolve, 4000));
    }

    setBackendStatus('error');
  }, []);

  useEffect(() => {
    checkBackend();
  }, [checkBackend]);

  // Detecta a 3ª queda (is_dead false → true) e abre o prompt persistente de
  // "outra partida". Em partida solo o round_over acontece no mesmo instante,
  // por isso o prompt não depende mais de round_over.
  useEffect(() => {
    const dead = !!ui.myPlayer?.is_dead;
    if (dead && !prevDeadRef.current) {
      prevDeadRef.current = true;
      setDeathPrompt(true);
      setDeathInfo({
        score: ui.myPlayer.score ?? 0,
        pos: myPos,
        ranking: matchRanking,
      });
    } else if (!dead) {
      prevDeadRef.current = false;
    }
  }, [ui.myPlayer, myPos, matchRanking]);

  // Mantém refs sincronizadas
  useEffect(() => {
    nicknameRef.current = nickname;
  }, [nickname]);

  useEffect(() => {
    ballColorRef.current = ballColor;
    localStorage.setItem('ballColor', ballColor);
  }, [ballColor]);

  useEffect(() => {
    myPlayerIdRef.current = myPlayerId;
  }, [myPlayerId]);

  useEffect(() => {
    cfgRef.current = cfg;
  }, [cfg]);

  // =======================
  // Teclado
  // =======================
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (isTypingTarget(e)) return;
      // Não processa teclas de gameplay antes de o jogo iniciar (evita que
      // Espaço na tela inicial dispare pulo/som e seta double-start).
      if (!startedRef.current) return;
      const k = e.key.toLowerCase();

      // Bloqueia o comportamento padrão ANTES do auto-repeat (segurar a tecla
      // não pode rolar a página nem re-acionar botões em foco).
      if (
        k === ' ' ||
        k === 'w' ||
        k === 'a' ||
        k === 'd' ||
        k === 'arrowleft' ||
        k === 'arrowright' ||
        k === 'shift'
      ) {
        e.preventDefault();
      }

      if (e.repeat) return;

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
      // Não inicia o jogo ao digitar espaço/Enter em um campo de texto.
      if (!startedRef.current && !isTypingTarget(e) && (e.key === 'Enter' || e.key === ' ')) {
        e.preventDefault();
        handleStartGame();
      }
    };

    // Se a página perder o foco, nenhuma tecla deve permanecer pressionada.
    const resetKeys = () => {
      const k = keysRef.current;
      if (k.left || k.right || k.jump || k.dash) {
        k.left = k.right = k.jump = k.dash = false;
        sendFrame();
      }
    };
    const handleVisibility = () => {
      if (document.hidden) resetKeys();
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    window.addEventListener('keydown', handleStartKey);
    window.addEventListener('blur', resetKeys);
    document.addEventListener('visibilitychange', handleVisibility);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
      window.removeEventListener('keydown', handleStartKey);
      window.removeEventListener('blur', resetKeys);
      document.removeEventListener('visibilitychange', handleVisibility);
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
  // Modo paisagem no celular (tela de carregamento + jogo)
  // =======================
  useEffect(() => {
    if (!isTouch) return;

    const update = () => {
      setIsLandscape(window.innerWidth >= window.innerHeight);
    };
    update();

    // Força tela cheia + paisagem já no carregamento (best-effort, silencioso).
    if (document.documentElement.requestFullscreen) {
      document.documentElement
        .requestFullscreen()
        .catch(() => {});
    }
    if (window.screen?.orientation?.lock) {
      window.screen.orientation
        .lock('landscape')
        .catch(() => {});
    }

    window.addEventListener('resize', update);
    window.addEventListener('orientationchange', update);
    return () => {
      window.removeEventListener('resize', update);
      window.removeEventListener('orientationchange', update);
    };
  }, [isTouch]);

  // =======================
  // WebSocket (com reconexão automática)
  // =======================
  useEffect(() => {
    if (!gameStarted) return;

    let ws = null;
    let timer = null;
    let closed = false;
    let attempts = 0;

    const connect = () => {
      ws = new WebSocket(GAME_WS_URL);
      socketRef.current = ws;

      ws.onopen = () => {
        attempts = 0;
        console.log('🟢 WebSocket conectado');
        sfx.unlock();
        ws.send(
          JSON.stringify({
            type: 'join',
            name: nicknameRef.current,
            color: ballColorRef.current,
          })
        );
        // Reconexão: reenvia o estado atual de input para que teclas seguradas
        // continuem funcionando sem o jogador precisar soltar/repressionar.
        const k = keysRef.current;
        if (k.left || k.right || k.jump || k.dash) {
          ws.send(
            JSON.stringify({
              type: 'input',
              left: k.left,
              right: k.right,
              jump: k.jump,
              dash: k.dash,
            })
          );
        }
      };

      ws.onmessage = (event) => {
        let data;
        try {
          data = JSON.parse(event.data);
        } catch {
          return;
        }

        if (data.type === 'init') {
          setMyPlayerId(data.player_id);
          return;
        }

        if (data.type) return;

        // Estado do jogo: guarda no ref para o canvas e atualiza a UI
        // (throttled) para os elementos de texto/HUD.
        gameStateRef.current = data;

        const nowMs = performance.now();
        if (nowMs - lastUiRef.current >= 250) {
          lastUiRef.current = nowMs;
          const players = data.players || {};
          const aliveCount = Object.values(players).filter((p) => !p.is_dead).length;
          setUi({
            round: data.round ?? 1,
            round_over: data.round_over ?? false,
            countdown: data.countdown ?? 0,
            drop_countdown: data.drop_countdown ?? 0,
            arena_width: data.arena_width ?? cfgRef.current.arena_width,
            arena_height: data.arena_height ?? cfgRef.current.arena_height,
            room_id: data.room_id ?? '',
            room_players: data.room_players ?? 0,
            room_capacity: data.room_capacity ?? 0,
            players,
            myPlayer: players[myPlayerIdRef.current] || null,
            aliveCount,
            aliveAny: aliveCount > 0,
          });
        }
      };

      ws.onerror = (err) => console.error('WebSocket erro', err);

      ws.onclose = () => {
        console.log('🔴 WebSocket fechado');
        if (closed) return;
        // Reconexão com backoff (máx. 8s).
        attempts++;
        const delay = Math.min(1000 * Math.pow(1.5, attempts), 8000);
        timer = setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      closed = true;
      if (timer) clearTimeout(timer);
      if (ws) ws.close();
    };
  }, [gameStarted]);

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
  // Desenho Canvas (rAF, desacoplado do re-render do React)
  // =======================
  useEffect(() => {
    if (!gameStarted) return;

    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    let rafId;
    let last = performance.now();

    const draw = () => {
      rafId = requestAnimationFrame(draw);
      const gameState = gameStateRef.current;
      if (!gameState) return;

      const arenaW = gameState.arena_width ?? cfg.arena_width;
      const arenaH = gameState.arena_height ?? cfg.arena_height;
      const tileSize = cfg.tile_size;
      const playerRadius = cfg.player_radius;

      const now = performance.now();
      const dt = Math.min((now - last) / 1000, 0.05);
      last = now;

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

        // Quadrado perdido: plataforma extra com halo dourado pulsante para
        // destacar que é um tile especial aparecendo fora do lugar.
        if (kind === 'lost') {
          const pulse = 0.14 + Math.sin(now / 260) * 0.05;
          ctx.fillStyle = `rgba(244,185,66,${pulse.toFixed(3)})`;
          roundRectPath(ctx, x - 7, y - 7, s + 14, s + 14, 18);
          ctx.fill();
          ctx.strokeStyle = 'rgba(217,154,43,0.7)';
          ctx.lineWidth = 2.5;
          roundRectPath(ctx, x - 7, y - 7, s + 14, s + 14, 18);
          ctx.stroke();
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

      // Power-ups: desenha os drops ativos e detecta pickup/despawn
      const powerUps = gameState.power_ups || {};
      const currentPUIds = new Set(Object.keys(powerUps));
      Object.values(powerUps).forEach((pu) => {
        drawPowerUpDrop(ctx, pu, now);
      });

      // Detecção de pickup do próprio jogador: buff acabou de ativar e um drop
      // foi removido nesta frame próximo a ele (distingue de despawn).
      const myPU = myPlayerId ? gameState.players?.[myPlayerId] : null;
      const hadBuff = prevBuffRef.current;
      const hasBuff = !!(myPU && myPU.buff && myPU.buff !== '');

      if (myPU && !myPU.is_dead && !hadBuff && hasBuff) {
        const myBuffType = myPU.buff;
        let bestDrop = null;
        let bestDist = Infinity;
        Object.entries(powerUpPrevRef.current).forEach(([id, prev]) => {
          if (!currentPUIds.has(id) && prev.type === myBuffType) {
            const d = Math.hypot(prev.x - myPU.x, prev.y - myPU.y);
            if (d < bestDist) {
              bestDist = d;
              bestDrop = prev;
            }
          }
        });
        if (bestDrop) {
          sfx.pickup();
          spawnParticles(bestDrop.x, bestDrop.y, bestDrop.glow, 14, 170);
        }
      }

      // Fim do buff do próprio jogador (som de aviso).
      if (myPU && !myPU.is_dead && hadBuff && !hasBuff) {
        sfx.buffEnd();
      }
      prevBuffRef.current = hasBuff ? myPU?.buff : null;
      // Atualiza o snapshot das posições dos drops.
      const puSnap = {};
      Object.values(powerUps).forEach((pu) => {
        puSnap[pu.id] = {
          x: pu.x,
          y: pu.y,
          type: pu.type,
          glow:
            pu.type === 'red_mushroom'
              ? PALETTE.redMushTop
              : pu.type === 'purple_mushroom'
                ? PALETTE.purpleMushTop
                : PALETTE.crystalMid,
        };
      });
      powerUpPrevRef.current = puSnap;

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

        // Raio exibido e buff ativo (Tanque dobra o raio visual).
        const buff = player.buff || null;
        const dispRadius = buff === 'red_mushroom' ? playerRadius * 2 : playerRadius;

        if (player.is_dead) {
          drawGhost(ctx, player, dispRadius);
        } else {
          drawPlayerBall(ctx, player, anim, now, dt, dispRadius, buff);
          if (buff) drawBuffIcon(ctx, player.x, player.y, buff, now);
        }

        // Seta branca: aponta para o próprio jogador quando ele sai da área
        // visível. Com a arena sempre visível por inteiro (desktop e mobile em
        // paisagem), os limites são a arena completa.
        if (player.id === myPlayerId && !player.is_dead) {
          const vMinX = 0;
          const vMaxX = arenaW;
          const vMinY = 0;
          const vMaxY = arenaH;

          if (
            player.x < vMinX ||
            player.x > vMaxX ||
            player.y < vMinY ||
            player.y > vMaxY
          ) {
            const pad = 26;
            const clampX = Math.max(vMinX + pad, Math.min(vMaxX - pad, player.x));
            const clampY = Math.max(vMinY + pad, Math.min(vMaxY - pad, player.y));
            const angle = Math.atan2(player.y - clampY, player.x - clampX);
            const offset = pad - 12;
            const tipX = clampX + Math.cos(angle) * offset;
            const tipY = clampY + Math.sin(angle) * offset;
            const size = 12 + Math.sin(now / 160) * 2;

            ctx.save();
            ctx.translate(tipX, tipY);
            ctx.rotate(angle);
            ctx.shadowColor = '#ffffff';
            ctx.shadowBlur = 10;
            ctx.fillStyle = '#ffffff';
            ctx.beginPath();
            ctx.moveTo(size, 0);
            ctx.lineTo(-size * 0.6, -size * 0.7);
            ctx.lineTo(-size * 0.6, size * 0.7);
            ctx.closePath();
            ctx.fill();
            ctx.restore();
          }
        }

        // Barra de cooldown do Dash (apenas para o próprio jogador)
        if (player.id === myPlayerId && !player.is_dead) {
          const cd = player.dash_cd || 0;
          const maxCd = cfg.dash_cooldown || 1.5;
          const barW = dispRadius * 2;
          const barY = player.y + dispRadius + 8;
          ctx.fillStyle = 'rgba(90,70,40,0.25)';
          ctx.fillRect(player.x - dispRadius, barY, barW, 5);
          ctx.fillStyle = cd > 0 ? PALETTE.amber : PALETTE.goldStrong;
          ctx.fillRect(
            player.x - dispRadius,
            barY,
            barW * Math.max(0, 1 - cd / maxCd),
            5
          );
        }

        // Barra do tempo restante do buff (apenas para o próprio jogador)
        if (player.id === myPlayerId && buff && !player.is_dead) {
          const remaining = Math.max(0, player.buff_remaining || 0);
          const duration =
            buff === 'red_mushroom'
              ? cfg.red_duration || 8
              : buff === 'purple_mushroom'
                ? cfg.purple_duration || 6
                : cfg.blue_duration || 8;
          const barW = dispRadius * 2;
          const barY = player.y + dispRadius + 15;
          const frac = Math.max(0, Math.min(1, remaining / duration));
          ctx.fillStyle = 'rgba(90,70,40,0.25)';
          ctx.fillRect(player.x - dispRadius, barY, barW, 5);
          ctx.fillStyle =
            buff === 'red_mushroom'
              ? PALETTE.redMushDark
              : buff === 'purple_mushroom'
                ? PALETTE.purpleMushDark
                : PALETTE.crystalMid;
          ctx.fillRect(player.x - dispRadius, barY, barW * frac, 5);
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
          const rA = (a.buff === 'red_mushroom' ? playerRadius * 2 : playerRadius);
          const rB = (b.buff === 'red_mushroom' ? playerRadius * 2 : playerRadius);
          const hitDist = rA + rB + 2;
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
    };

    rafId = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(rafId);
  }, [gameStarted, cfg, myPlayerId, isTouch, spawnParticles, addShake]);

  // =======================
  // Scores
  // =======================
  useEffect(() => {
    const fetchScores = async () => {
      try {
        const res = await fetch(SCORES_API_URL);
        const data = await res.json();
        // Garante array (o backend pode responder [] ou um objeto de erro).
        setScores(Array.isArray(data) ? data : []);
      } catch (err) {
        console.error('Erro ao buscar scores', err);
      }
    };

    fetchScores();
    const interval = setInterval(fetchScores, 10000);
    return () => clearInterval(interval);
  }, []);

  // Recorde pessoal
  useEffect(() => {
    const my = ui.myPlayer;
    if (my && !my.is_dead) {
      setBestScore((b) => Math.max(b, my.score));
    }
  }, [ui.myPlayer]);

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

  // Botão de controle de toque (mobile): alvo grande, sem zoom/scroll.
  const touchBtnSx = {
    pointerEvents: 'auto',
    minWidth: 60,
    minHeight: 60,
    fontSize: '1.4rem',
    borderRadius: 3,
    touchAction: 'none',
    userSelect: 'none',
    boxShadow: '0 4px 14px rgba(0,0,0,0.35)',
  };

  // =======================
  // UI
  // =======================
  // Tela de carregamento/erro enquanto o backend não confirma estar no ar
  // =======================
  if (backendStatus !== 'ready') {
    const isError = backendStatus === 'error';
    const showRotate = isTouch && !isLandscape;
    return (
      <Box
        sx={{
          height: '100dvh',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 3,
          p: 3,
          textAlign: 'center',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {/* Fundo integral: imagem de carregamento cobre 100% da tela */}
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            backgroundImage: `url(${carregamentoImg})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        />
        {/* Overlay translúcido para legibilidade do texto */}
        <Box
          sx={{
            position: 'absolute',
            inset: 0,
            background: 'rgba(0,0,0,0.32)',
          }}
        />
        <Button
          variant="outlined"
          size="small"
          sx={{
            position: 'absolute',
            zIndex: 2,
            top: 'calc(12px + env(safe-area-inset-top))',
            right: 'calc(12px + env(safe-area-inset-right))',
            minWidth: 52,
            color: '#ffffff',
            borderColor: 'rgba(255,255,255,0.7)',
            bgcolor: 'rgba(0,0,0,0.25)',
          }}
          onClick={toggleLang}
        >
          {lang === 'pt' ? 'EN' : 'PT'}
        </Button>
        {showRotate ? (
          <>
            <Typography
              variant="h5"
              sx={{ zIndex: 1, color: '#ffffff', fontWeight: 800 }}
            >
              ↔
            </Typography>
            <Typography
              variant="h6"
              sx={{ zIndex: 1, color: '#ffffff', fontWeight: 700 }}
            >
              {t('loading.rotate')}
            </Typography>
          </>
        ) : (
          <>
            <Box sx={{ zIndex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
              {isError ? (
                <>
                  <Typography variant="h4" color="#ffffff" fontWeight={800}>
                    {t('loading.errorTitle')}
                  </Typography>
                  <Typography color="rgba(255,255,255,0.85)">
                    {t('loading.errorMsg')}
                  </Typography>
                  <Button
                    variant="contained"
                    size="large"
                    sx={{ bgcolor: PALETTE.olive, '&:hover': { bgcolor: '#5f6d30' } }}
                    onClick={checkBackend}
                  >
                    {t('loading.retry')}
                  </Button>
                </>
              ) : (
                <>
                  <CircularProgress sx={{ color: PALETTE.goldStrong }} size={52} />
                  <Typography variant="h6" color="#ffffff" fontWeight={700}>
                    {t('loading.searching')}
                  </Typography>
                </>
              )}
            </Box>
          </>
        )}
      </Box>
    );
  }

  return (
    <Box
      sx={{
        height: '100dvh',
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        background:
          'linear-gradient(180deg, #fdf0d8 0%, #f6e3c5 45%, #efc58b 100%)',
      }}
    >
        <Container
          maxWidth={false}
          disableGutters
          sx={{
            maxWidth: '1680px',
            mx: 'auto',
            px: { xs: isTouch && gameStarted ? 0 : 1.5, md: 2 },
            py: { xs: isTouch && gameStarted ? 0 : 2, md: 2 },
            display: 'flex',
            flexDirection: 'column',
            flexGrow: 1,
            minHeight: 0,
          }}
        >
        <img
          src="/logo.jpeg"
          alt="SURVIVE THE DESTRUCTION"
          style={{
            display: gameStarted ? 'none' : 'block',
            margin: '0 auto',
            maxHeight: '14dvh',
            width: 'auto',
            objectFit: 'contain',
            borderRadius: 12,
            boxShadow: '0 10px 24px rgba(90,70,40,0.25)',
            mb: 1,
          }}
        />
        <Typography
          variant="h3"
          align="center"
          sx={{
            mb: 2,
            fontWeight: 800,
            color: PALETTE.textBrown,
            background: 'linear-gradient(90deg, #d99a2b, #c96f4a)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            display: { xs: gameStarted ? 'none' : 'block', md: 'block' },
          }}
        >
          SURVIVE THE DESTRUCTION
        </Typography>

        <Box
          sx={{
            display: 'flex',
            flexDirection: 'row',
            gap: 3,
            alignItems: 'stretch',
            flexGrow: 1,
            minHeight: 0,
          }}
        >
          <Box sx={{ flex: '1 1 auto', minWidth: 0, display: 'flex', justifyContent: 'center' }}>
            <Paper
              sx={{
                position: 'relative',
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                height: '100%',
                borderRadius: 4,
                overflow: 'hidden',
                border: isTouch && gameStarted ? 'none' : `2px solid ${PALETTE.borderSoft}`,
                bgcolor: PALETTE.skyTop,
                boxShadow: '0 18px 40px rgba(90,70,40,0.18)',
              }}
            >
              <canvas
                ref={canvasRef}
                className="game-canvas"
                width={arenaW}
                height={arenaH}
                style={{
                  display: 'block',
                  width: 'auto',
                  height: 'auto',
                  maxWidth: '100%',
                  maxHeight: '100%',
                }}
              />

              {!gameStarted && (
                <Box sx={overlaySx}>
                  <Button
                    variant="outlined"
                    size="small"
                    sx={{
                      position: 'absolute',
                      top: 'calc(8px + env(safe-area-inset-top))',
                      right: 'calc(8px + env(safe-area-inset-right))',
                      minWidth: 52,
                      color: PALETTE.textBrown,
                      borderColor: PALETTE.borderSoft,
                    }}
                    onClick={toggleLang}
                  >
                    {lang === 'pt' ? 'EN' : 'PT'}
                  </Button>
                  <Typography variant="h5" color={PALETTE.terracottaStrong} fontWeight={700}>
                    {t('start.chooseName')}
                  </Typography>
                  <TextField
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    inputProps={{ maxLength: 20 }}
                    size="small"
                    sx={{ width: 260, bgcolor: 'rgba(255,255,255,0.6)', borderRadius: 2 }}
                  />
                  <Box
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1.5,
                      bgcolor: 'rgba(255,255,255,0.6)',
                      borderRadius: 2,
                      px: 1.5,
                      py: 0.75,
                    }}
                  >
                    <Typography variant="body2" color={PALETTE.textSoft}>
                      {t('start.ballColor')}
                    </Typography>
                    <input
                      type="color"
                      value={ballColor}
                      onChange={(e) => setBallColor(e.target.value)}
                      aria-label={t('start.ballColor')}
                      style={{
                        width: 44,
                        height: 44,
                        padding: 0,
                        border: 'none',
                        background: 'none',
                        cursor: 'pointer',
                      }}
                    />
                  </Box>
                  <Button
                    variant="contained"
                    size="large"
                    sx={{ bgcolor: PALETTE.olive, '&:hover': { bgcolor: '#5f6d30' } }}
                    onClick={handleStartGame}
                  >
                    {t('start.play')}
                  </Button>
                  <Typography variant="body2" color={PALETTE.textSoft}>
                    {t('start.instructions')}
                  </Typography>
                </Box>
              )}

              {gameStarted && ui.round_over && !deathPrompt && (
                <Box sx={overlaySx}>
                  <Typography variant="h4" color={PALETTE.fallingDark} fontWeight={800}>
                    {t('game.roundOver')}
                  </Typography>
                  <Typography
                    variant="h1"
                    color={PALETTE.goldStrong}
                    fontWeight={900}
                    sx={{ fontSize: '6rem' }}
                  >
                    {ui.countdown}
                  </Typography>
                </Box>
              )}

              {gameStarted && !ui.round_over && !deathPrompt && ui.myPlayer?.respawn_left > 0 && (
                <Box sx={overlaySx}>
                  <Typography variant="h4" color={PALETTE.fallingDark} fontWeight={800}>
                    {t('game.reviving')}
                  </Typography>
                  <Typography
                    variant="h1"
                    color={PALETTE.goldStrong}
                    fontWeight={900}
                    sx={{ fontSize: '6rem' }}
                  >
                    {ui.myPlayer.respawn_left}
                  </Typography>
                </Box>
              )}

              {gameStarted && deathPrompt && (
                <Box
                  sx={{
                    ...overlaySx,
                    overflowY: 'auto',
                    justifyContent: 'flex-start',
                    gap: 1,
                  }}
                >
                  <Typography
                    variant="h4"
                    color={PALETTE.fallingDark}
                    fontWeight={800}
                  >
                    {t('death.title')}
                  </Typography>

                  {deathInfo && deathInfo.pos > 0 && (
                    <Typography
                      variant="h6"
                      color={PALETTE.goldStrong}
                      fontWeight={800}
                    >
                      {t('death.placed', {
                        pos: deathInfo.pos,
                        total: deathInfo.ranking.length,
                        players: t(
                          deathInfo.ranking.length === 1
                            ? 'death.playersOne'
                            : 'death.playersMany'
                        ),
                      })}
                    </Typography>
                  )}

                  <Typography variant="body2" color={PALETTE.textSoft}>
                    {t('death.timeSurvived', { time: deathInfo?.score ?? 0 })}
                  </Typography>
                  <Typography variant="body2" color={PALETTE.textSoft}>
                    {t('death.livesGone')}
                  </Typography>

                  <Typography
                    variant="body1"
                    color={PALETTE.textBrown}
                    fontWeight={700}
                    sx={{ mt: 1 }}
                  >
                    {t('death.anotherMatch')}
                  </Typography>
                  <Box
                    sx={{
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 1.5,
                      width: '100%',
                      mt: 0.5,
                    }}
                  >
                    <Button
                      variant="contained"
                      size="large"
                      sx={{
                        flex: 1,
                        minWidth: 180,
                        bgcolor: PALETTE.olive,
                        '&:hover': { bgcolor: '#5f6d30' },
                      }}
                      onClick={() => {
                        sendRestart();
                        setDeathPrompt(false);
                      }}
                      disabled={!ui.aliveAny}
                    >
                      {t('death.retry')}
                    </Button>
                    <Button
                      variant="outlined"
                      size="large"
                      sx={{
                        flex: 1,
                        minWidth: 180,
                        color: PALETTE.terracottaStrong,
                        borderColor: PALETTE.terracottaStrong,
                        '&:hover': {
                          borderColor: PALETTE.terracottaStrong,
                          bgcolor: 'rgba(201,111,74,0.08)',
                        },
                      }}
                      onClick={startNewMatch}
                    >
                      {t('death.otherMatch')}
                    </Button>
                    <Button
                      variant="outlined"
                      size="large"
                      sx={{
                        flex: 1,
                        minWidth: 180,
                        color: PALETTE.textSoft,
                        borderColor: PALETTE.textSoft,
                        '&:hover': {
                          borderColor: PALETTE.textSoft,
                          bgcolor: 'rgba(120,105,80,0.08)',
                        },
                      }}
                      onClick={() => setDeathPrompt(false)}
                    >
                      {t('death.continue')}
                    </Button>
                  </Box>

                    <Paper
                      elevation={0}
                      sx={{
                        width: '100%',
                        maxWidth: 480,
                        bgcolor: 'rgba(240,224,196,0.55)',
                        borderRadius: 3,
                        p: 1.5,
                        mt: 1,
                      }}
                    >
                      <Typography
                        variant="subtitle2"
                        fontWeight={700}
                        color={PALETTE.textBrown}
                      >
                        {t('death.ranking')}
                      </Typography>
                      <List dense>
                        {(deathInfo?.ranking ?? matchRanking).slice(0, 10).map((p, i) => (
                          <ListItem
                            key={p.id}
                            sx={{
                              bgcolor:
                                p.id === ui.myPlayer.id
                                  ? 'rgba(217,154,43,0.28)'
                                  : 'transparent',
                              borderRadius: 2,
                            }}
                          >
                            <Chip
                              label={i + 1}
                              size="small"
                              sx={{ mr: 1 }}
                              color={i < 3 ? 'warning' : 'default'}
                            />
                            <ListItemText
                              primary={
                                p.name || `Player_${p.id.slice(-4)}`
                              }
                              primaryTypographyProps={{
                                fontWeight:
                                  p.id === ui.myPlayer.id ? 800 : 400,
                              }}
                            />
                            <Typography
                              color={PALETTE.goldStrong}
                              fontWeight="bold"
                            >
                              {p.score}s
                            </Typography>
                          </ListItem>
                        ))}
                      </List>
                    </Paper>

                    {scores.length > 0 && (
                      <Box
                        sx={{
                          width: '100%',
                          maxWidth: 480,
                          mt: 1,
                        }}
                      >
                        <Leaderboard scores={scores} highlightName={nickname} lang={lang} />
                      </Box>
                    )}
                  </Box>
                )}

              {isTouch && gameStarted && (
                <Box sx={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
                  {/* HUD compacto mobile */}
                  <Box
                    sx={{
                      position: 'absolute',
                      top: 0,
                      left: 0,
                      display: 'flex',
                      gap: 1,
                      pt: 'calc(10px + env(safe-area-inset-top))',
                      pl: 'calc(10px + env(safe-area-inset-left))',
                      pr: 1,
                      maxWidth: '70%',
                    }}
                  >
                    <Box
                      sx={{
                        bgcolor: 'rgba(90,70,40,0.55)',
                        color: '#fff',
                        borderRadius: 2,
                        px: 1.25,
                        py: 1,
                        fontSize: '0.85rem',
                        lineHeight: 1.5,
                        fontWeight: 700,
                      }}
                    >
                      <Box component="div">
                        ⏱ {ui.myPlayer ? `${ui.myPlayer.score}s` : '0s'}
                      </Box>
                      <Box component="div">
                        {'❤'.repeat(ui.myPlayer?.lives ?? cfg.max_lives ?? 3) ||
                          '0'}
                      </Box>
                      <Box component="div">
                        {t('hud.buff')}{' '}
                        {ui.myPlayer?.buff
                          ? ui.myPlayer.buff === 'red_mushroom'
                            ? '🟥'
                            : ui.myPlayer.buff === 'purple_mushroom'
                              ? '🟪'
                              : '🟦'
                          : '—'}
                      </Box>
                      <Box component="div">{t('hud.alive')} {ui.aliveCount}</Box>
                      {ui.room_id && (
                        <Box component="div" sx={{ fontSize: '0.72rem', opacity: 0.85 }}>
                          {t('hud.room')} {ui.room_id.slice(-12)} · {ui.room_players}/{ui.room_capacity}
                        </Box>
                      )}
                    </Box>
                    {matchRanking.length > 0 && (
                      <Box
                        sx={{
                          bgcolor: 'rgba(90,70,40,0.55)',
                          color: '#fff',
                          borderRadius: 2,
                          px: 1.25,
                          py: 1,
                          fontSize: '0.75rem',
                          lineHeight: 1.5,
                        }}
                      >
                        {matchRanking.slice(0, 3).map((p, i) => (
                          <Box
                            component="div"
                            key={p.id}
                            sx={{
                              fontWeight:
                                p.id === ui.myPlayer?.id ? 800 : 400,
                            }}
                          >
                            {i + 1}º {p.name || `Player_${p.id.slice(-4)}`}{' '}
                            {p.score}s
                          </Box>
                        ))}
                      </Box>
                    )}
                  </Box>

                  {/* Botões superiores: idioma, sair da tela cheia, sair da partida */}
                  <Box
                    sx={{
                      position: 'absolute',
                      top: 'calc(10px + env(safe-area-inset-top))',
                      right: 'calc(10px + env(safe-area-inset-right))',
                      display: 'flex',
                      gap: 1,
                      pointerEvents: 'none',
                    }}
                  >
                    <Button
                      variant="contained"
                      size="small"
                      sx={{
                        pointerEvents: 'auto',
                        minWidth: 44,
                        minHeight: 44,
                        fontSize: '0.8rem',
                        bgcolor: 'rgba(90,70,40,0.55)',
                        '&:hover': { bgcolor: 'rgba(90,70,40,0.8)' },
                      }}
                      onClick={toggleLang}
                    >
                      {lang === 'pt' ? 'EN' : 'PT'}
                    </Button>
                    <Button
                      variant="contained"
                      size="small"
                      sx={{
                        pointerEvents: 'auto',
                        minWidth: 44,
                        minHeight: 44,
                        fontSize: '1.1rem',
                        bgcolor: 'rgba(90,70,40,0.55)',
                        '&:hover': { bgcolor: 'rgba(90,70,40,0.8)' },
                      }}
                      onClick={exitFullscreen}
                    >
                      ⛶
                    </Button>
                    <Button
                      variant="contained"
                      size="small"
                      sx={{
                        pointerEvents: 'auto',
                        minWidth: 44,
                        minHeight: 44,
                        fontSize: '0.7rem',
                        px: 1,
                        bgcolor: 'rgba(140,60,50,0.75)',
                        '&:hover': { bgcolor: 'rgba(140,60,50,0.95)' },
                      }}
                      onClick={handleExit}
                    >
                      {t('exit.mobile')}
                    </Button>
                  </Box>

                  {/* Controles: ◀ ▶ à esquerda, ⬆ ⚡ à direita */}
                  <Box
                    sx={{
                      position: 'absolute',
                      bottom: 'calc(12px + env(safe-area-inset-bottom))',
                      left: 'calc(12px + env(safe-area-inset-left))',
                      right: 'calc(12px + env(safe-area-inset-right))',
                      display: 'flex',
                      alignItems: 'flex-end',
                      justifyContent: 'space-between',
                    }}
                  >
                    <Box sx={{ display: 'flex', gap: 1.5 }}>
                      <Button
                        variant="contained"
                        sx={{
                          ...touchBtnSx,
                          bgcolor: PALETTE.olive,
                          '&:hover': { bgcolor: '#5f6d30' },
                        }}
                        onPointerDown={touchPress('left')}
                        onPointerUp={touchRelease('left')}
                        onPointerLeave={touchRelease('left')}
                        onPointerCancel={touchRelease('left')}
                      >
                        ◀
                      </Button>
                      <Button
                        variant="contained"
                        sx={{
                          ...touchBtnSx,
                          bgcolor: PALETTE.olive,
                          '&:hover': { bgcolor: '#5f6d30' },
                        }}
                        onPointerDown={touchPress('right')}
                        onPointerUp={touchRelease('right')}
                        onPointerLeave={touchRelease('right')}
                        onPointerCancel={touchRelease('right')}
                      >
                        ▶
                      </Button>
                    </Box>
                    <Box sx={{ display: 'flex', gap: 1.5 }}>
                      <Button
                        variant="contained"
                        sx={{
                          ...touchBtnSx,
                          bgcolor: PALETTE.goldStrong,
                          '&:hover': { bgcolor: '#c2851e' },
                        }}
                        onPointerDown={touchPress('jump')}
                        onPointerUp={touchRelease('jump')}
                        onPointerLeave={touchRelease('jump')}
                        onPointerCancel={touchRelease('jump')}
                      >
                        ⬆
                      </Button>
                      <Button
                        variant="contained"
                        sx={{
                          ...touchBtnSx,
                          bgcolor: PALETTE.terracottaStrong,
                          '&:hover': { bgcolor: '#ab5a3c' },
                        }}
                        onPointerDown={touchPress('dash')}
                        onPointerUp={touchRelease('dash')}
                        onPointerLeave={touchRelease('dash')}
                        onPointerCancel={touchRelease('dash')}
                      >
                        ⚡
                      </Button>
                    </Box>
                  </Box>
                </Box>
              )}
            </Paper>
          </Box>

          <Box
            sx={{
              display: { xs: 'none', md: 'block' },
              flex: '0 0 300px',
              minWidth: 0,
              height: '100%',
            }}
          >
            <Box
              display="flex"
              flexDirection="column"
              gap={3}
              sx={{ height: '100%', overflowY: 'auto' }}
            >
              <Leaderboard scores={scores} lang={lang} />

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
                  {t('sidebar.status')}
                </Typography>
                <Typography>{t('sidebar.name')} {nickname}</Typography>
                <Typography>
                  {t('sidebar.time')} {ui.myPlayer ? `${ui.myPlayer.score}s` : '0s'}
                </Typography>
                <Typography color={PALETTE.goldStrong} fontWeight="bold">
                  {t('sidebar.record')} {bestScore}s
                </Typography>
                <Typography>{t('sidebar.round')} {ui.round}</Typography>
                <Typography>
                  {t('sidebar.room')}{' '}
                  {ui.room_id ? `${ui.room_id} · ${ui.room_players}/${ui.room_capacity}` : '—'}
                </Typography>
                <Typography>{t('sidebar.alive')} {ui.aliveCount}</Typography>
                <Typography>
                  {t('sidebar.nextDrop')}{' '}
                  {ui.myPlayer && !ui.myPlayer.is_dead
                    ? `${ui.drop_countdown.toFixed(1)}s`
                    : '—'}
                </Typography>
                <Typography>
                  {t('sidebar.buff')}{' '}
                  {ui.myPlayer?.buff
                    ? ui.myPlayer.buff === 'red_mushroom'
                      ? `🟥 ${t('buff.tank')}`
                      : ui.myPlayer.buff === 'purple_mushroom'
                        ? `🟪 ${t('buff.speedster')}`
                        : `🟦 ${t('buff.glider')}`
                    : t('buff.none')}
                </Typography>
                <Typography>
                  {t('sidebar.lives')}{' '}
                  {'❤'.repeat(ui.myPlayer?.lives ?? cfg.max_lives ?? 3) ||
                    '0'}
                </Typography>
                {ui.myPlayer && ui.myPlayer.is_dead && (
                  <Typography color="error.main">{t('sidebar.spectating')}</Typography>
                )}
                <Box
                  sx={{
                    mt: 1.5,
                    pt: 1.5,
                    borderTop: `1px dashed rgba(201,111,74,0.35)`,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 1,
                  }}
                >
                  <Button
                    variant="outlined"
                    size="small"
                    fullWidth
                    sx={{
                      color: PALETTE.textSoft,
                      borderColor: PALETTE.textSoft,
                      '&:hover': {
                        borderColor: PALETTE.textSoft,
                        bgcolor: 'rgba(120,105,80,0.08)',
                      },
                    }}
                    onClick={handleExit}
                  >
                    {t('exit.label')}
                  </Button>
                  <Button
                    variant="text"
                    size="small"
                    fullWidth
                    sx={{ color: PALETTE.textBrown }}
                    onClick={toggleLang}
                  >
                    {lang === 'pt' ? 'EN' : 'PT'}
                  </Button>
                </Box>
              </Paper>
            </Box>
          </Box>
        </Box>
      </Container>
    </Box>
  );
}
