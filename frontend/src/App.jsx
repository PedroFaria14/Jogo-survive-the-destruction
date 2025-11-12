import React, { useState, useEffect, useRef, useCallback } from 'react';
import './App.css';
const GAME_WS_URL = 'ws://localhost:8080/ws';
const SCORES_API_URL = 'http://localhost:8080/api/scores';

const ARENA_WIDTH = 800;
const ARENA_HEIGHT = 600;
const TILE_SIZE = 100;
const PLAYER_RADIUS = 25;
const LINE_THICKNESS = 4;

const Leaderboard = ({ scores }) => (
  <div className="bg-gradient-to-br from-gray-800/80 to-gray-900/80 backdrop-blur-xl border border-teal-500/30 rounded-2xl p-6 shadow-lg">
    <h2 className="text-xl font-bold text-teal-300 flex items-center gap-2 mb-4 tracking-wide">
      🏅 Top 10 Sobreviventes
    </h2>
    {scores.length === 0 ? (
      <p className="text-gray-400 italic">Nenhum placar registrado ainda.</p>
    ) : (
      <ul className="space-y-2">
        {scores.map((score, index) => (
          <li
            key={index}
            className="flex justify-between items-center text-sm p-2 rounded-lg bg-gray-800/70 hover:bg-gray-700 transition duration-150 border border-gray-700/30"
          >
            <span className="font-mono text-gray-400 w-6">{index + 1}.</span>
            <span
              className={`flex-1 truncate font-semibold ${
                index < 3 ? 'text-yellow-300' : 'text-white'
              }`}
            >
              {score.player_id
                ? `Player_${score.player_id.substring(score.player_id.length - 4)}`
                : 'Anônimo'}
            </span>
            <span className="font-bold text-green-400">{score.score_seconds}s</span>
          </li>
        ))}
      </ul>
    )}
  </div>
);

export default function App() {
  const [gameState, setGameState] = useState(null);
  const [scores, setScores] = useState([]);
  const [myPlayerId, setMyPlayerId] = useState(null);
  const [gameOver, setGameOver] = useState(false);
  const [finalScore, setFinalScore] = useState(0);
  const [gameStarted, setGameStarted] = useState(false);

  const socketRef = useRef(null);
  const canvasRef = useRef(null);

  const handleStartGame = () => {
    setGameStarted(true);
    setGameOver(false);
  };

  const sendCommand = useCallback(
    (command) => {
      if (
        socketRef.current &&
        socketRef.current.readyState === WebSocket.OPEN &&
        gameStarted &&
        !gameOver
      ) {
        socketRef.current.send(JSON.stringify(command));
      }
    },
    [gameStarted, gameOver]
  );

  // WebSocket
  useEffect(() => {
    if (!gameStarted) {
      if (socketRef.current && socketRef.current.readyState !== WebSocket.CLOSED) {
        socketRef.current.close();
      }
      return;
    }

    socketRef.current = new WebSocket(GAME_WS_URL);
    const ws = socketRef.current;
    let isDeadLocal = false;

    ws.onopen = () => console.log('WebSocket conectado!');
    ws.onclose = () => {
      console.log('WebSocket desconectado.');
      if (gameStarted && !isDeadLocal) setGameOver(true);
    };
    ws.onerror = (error) => console.error('WebSocket Error:', error);

    ws.onmessage = (event) => {
      try {
        const newState = JSON.parse(event.data);
        setGameState(newState);
        if (newState.players) {
          if (!myPlayerId && Object.keys(newState.players).length > 0)
            setMyPlayerId(Object.keys(newState.players)[0]);
          const player = myPlayerId ? newState.players[myPlayerId] : null;
          if (player && player.is_dead && !gameOver) {
            setGameOver(true);
            setFinalScore(player.score);
            isDeadLocal = true;
            ws.close();
          }
        }
      } catch (e) {
        console.error('Erro ao parsear GameState JSON:', e);
      }
    };
    return () => ws.close();
  }, [gameStarted, myPlayerId, gameOver]); // Adicionado myPlayerId e gameOver como dependências

  // Fetch Scores
  useEffect(() => {
    const fetchScores = async () => {
      try {
        const response = await fetch(SCORES_API_URL);
        const data = await response.json();
        const formatted = data.map((s) => ({
          ...s,
          score_seconds: Math.floor(s.duration_ms / 1000),
        }));
        setScores(formatted);
      } catch (err) {
        console.error('Erro ao buscar placares:', err);
      }
    };
    fetchScores();
    const intervalId = setInterval(fetchScores, 10000);
    return () => clearInterval(intervalId);
  }, []);

  // Canvas Drawing
  useEffect(() => {
    if (!gameState) return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');

    const draw = () => {
      const bgGradient = ctx.createLinearGradient(0, 0, 0, ARENA_HEIGHT);
      bgGradient.addColorStop(0, '#0a0f1c');
      bgGradient.addColorStop(1, '#0d1626');
      ctx.fillStyle = bgGradient;
      ctx.fillRect(0, 0, ARENA_WIDTH, ARENA_HEIGHT);

      ctx.strokeStyle = '#1d2230';
      ctx.lineWidth = 1.2;
      for (let i = 0; i < ARENA_WIDTH; i += TILE_SIZE / 2) {
        ctx.beginPath();
        ctx.moveTo(i, 0);
        ctx.lineTo(i, ARENA_HEIGHT);
        ctx.stroke();
      }
      for (let i = 0; i < ARENA_HEIGHT; i += TILE_SIZE / 2) {
        ctx.beginPath();
        ctx.moveTo(0, i);
        ctx.lineTo(ARENA_WIDTH, i);
        ctx.stroke();
      }

      if (gameState.arena_tiles) {
        Object.values(gameState.arena_tiles).forEach((tile) => {
          if (tile.is_active) {
            const color = tile.is_falling ? '#ef4444' : '#3b82f6';
            ctx.strokeStyle = color;
            ctx.lineWidth = LINE_THICKNESS;
            ctx.beginPath();
            ctx.moveTo(tile.x, tile.y);
            ctx.lineTo(tile.x + TILE_SIZE, tile.y);
            ctx.stroke();
          }
        });
      }

      if (gameState.players) {
        Object.values(gameState.players).forEach((player) => {
          if (player.is_dead) return;
          ctx.shadowColor = 'rgba(0,0,0,0.6)';
          ctx.shadowBlur = 10;
          ctx.fillStyle = player.id === myPlayerId ? '#10b981' : '#facc15';
          ctx.beginPath();
          ctx.arc(player.x, player.y, PLAYER_RADIUS, 0, Math.PI * 2);
          ctx.fill();

          ctx.shadowBlur = 0;
          ctx.fillStyle = 'white';
          ctx.font = '10px Inter';
          ctx.textAlign = 'center';
          ctx.fillText(`${player.score}s`, player.x, player.y - PLAYER_RADIUS - 5);
        });
      }

      if (gameOver) {
        ctx.fillStyle = 'rgba(0,0,0,0.85)';
        ctx.fillRect(0, 0, ARENA_WIDTH, ARENA_HEIGHT);
        ctx.fillStyle = '#10b981';
        ctx.font = '48px Inter, sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('GAME OVER', ARENA_WIDTH / 2, ARENA_HEIGHT / 2 - 20);
        ctx.font = '24px Inter';
        ctx.fillText(`Score: ${finalScore}s`, ARENA_WIDTH / 2, ARENA_HEIGHT / 2 + 20);
      }
    };
    draw();
  }, [gameState, gameOver, finalScore, myPlayerId]);

  const myPlayer = gameState && myPlayerId ? gameState.players[myPlayerId] : null;

  const GameArea = gameStarted ? (
    <canvas
      ref={canvasRef}
      width={ARENA_WIDTH}
      height={ARENA_HEIGHT}
      className="bg-gray-950 rounded-xl border border-gray-700 shadow-2xl"
    />
  ) : (
    <div
      className="flex flex-col items-center justify-center text-center bg-gray-900/60 backdrop-blur-lg rounded-2xl border border-teal-500/30 shadow-xl"
      style={{ width: ARENA_WIDTH, height: ARENA_HEIGHT }}
    >
      <h2 className="text-4xl font-extrabold text-teal-400 mb-6 animate-pulse drop-shadow-lg">
        SOBREVIVA À DESTRUIÇÃO
      </h2>
      <p className="text-gray-300 mb-8 max-w-md">
        Mova-se com <span className="text-yellow-300 font-semibold">A/D</span> e pule com{' '}
        <span className="text-yellow-300 font-semibold">W/Espaço</span>.
      </p>
      <button
        onClick={handleStartGame}
        className="px-10 py-3 bg-gradient-to-r from-green-500 to-teal-500 hover:from-teal-400 hover:to-green-400 
                 text-white font-bold text-lg rounded-xl shadow-xl transition-all transform hover:scale-105 active:scale-95"
      >
        INICIAR JOGO
      </button>
    </div>
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-[#0a0f1c] via-[#111827] to-[#0f172a] text-gray-100 font-sans flex flex-col items-center p-6">
      <header className="text-center mb-10">
        <h1 className="text-5xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-teal-400 via-green-400 to-blue-500 drop-shadow-lg">
          Survive the Destruction
        </h1>
        <p className="mt-2 text-gray-400 text-sm">
          Plataforma 2D de Sobrevivência em Tempo Real (Go + React)
        </p>
      </header>

      <div className="flex flex-col lg:flex-row w-full max-w-7xl gap-8">
        {/* Direita (Jogo) - Vem primeiro no HTML para mobile, mas 'order-2' no desktop */}
        <div className="lg:w-2/3 flex flex-col items-center justify-center lg:order-2">
          <div
            className="relative rounded-2xl overflow-hidden border-2 border-teal-400/40 shadow-[0_0_20px_#14b8a6aa] hover:shadow-[0_0_35px_#14b8a6ff] transition-all"
            style={{ width: ARENA_WIDTH, height: ARENA_HEIGHT }}
  _         >
            {GameArea}
          </div>
        </div>

        {/* Esquerda (Informações) - Vem depois no HTML, mas 'order-1' (esquerda) no desktop */}
        <div className="lg:w-1/3 flex flex-col gap-6 lg:order-1">
          <Leaderboard scores={scores} />

          <div className="bg-gray-800/70 backdrop-blur-lg border border-teal-500/30 rounded-2xl p-6 shadow-lg">
            <h2 className="text-xl font-bold text-teal-300 mb-3">Meu Status</h2>
            <p className="text-sm mb-1">
              👤 <span className="font-medium text-white">ID:</span>{' '}
              <span className="font-mono">{myPlayerId ? myPlayerId.substring(0, 8) : 'N/A'}</span>
            </p>
            <p className="text-sm">
              🏆 <span className="font-medium text-white">Placar:</span>{' '}
              <span className="font-extrabold text-green-400 text-lg">
                {myPlayer ? myPlayer.score : '0'}s
              </span>
            </p>
            <p className="mt-3 text-sm text-gray-400">
              🎮 <span className="text-yellow-300">A/D</span> mover •{' '}
              <span className="text-yellow-300">W/Espaço</span> pular
            </p>
          </div>

          <div className="text-center text-sm font-semibold mt-2">
        _   {gameOver ? (
              <p className="text-red-500">❌ Game Over! Pontuação salva.</p>
            ) : (
              <p className="text-teal-400">
                {gameStarted
                  ? gameState
                   ? `🟢 Online (${Object.keys(gameState.players).length} jogadores)`
                    : '🟡 Conectando...'
                  : 'Clique em INICIAR JOGO.'}
              </p>
            )}
          </div>
        </div>
      </div>

      <footer className="text-xs text-gray-500 mt-10">
        © 2025 — Feito com ⚡ Go, React e Tailwind
      </footer>
    </div>
  );
}