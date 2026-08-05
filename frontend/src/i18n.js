// i18n simples: dicionário PT/EN com detecção automática do navegador e
// substituição de placeholders ({chave}) via translate().

export const STRINGS = {
  pt: {
    'loading.checking': 'Carregando jogo...',
    'loading.errorTitle': 'Não foi possível conectar ao servidor',
    'loading.errorMsg': 'Verifique sua conexão e tente novamente.',
    'loading.retry': 'TENTAR NOVAMENTE',
    'start.chooseName': 'Escolha seu nome',
    'start.ballColor': 'Cor da bolinha:',
    'start.play': 'INICIAR JOGO',
    'start.instructions':
      'Use A/D ou ←/→ para mover, W ou Espaço para pular (×2 no ar). Shift para Dash (empurra oponentes). Enter também inicia. Pegue os drops: 🍄 vermelho = Tanque, 🍄 roxo = Velocista, 💎 azul = Planar (um buff por vez).',
    'game.roundOver': 'NOVA RODADA',
    'death.title': 'VOCÊ CAIU!',
    'death.placed': 'Você ficou em {pos}º de {total} {players}',
    'death.playersOne': 'jogador',
    'death.playersMany': 'jogadores',
    'death.timeSurvived': 'Tempo sobrevivido: {time}s',
    'death.livesGone':
      'Suas vidas acabaram. Continue na próxima rodada ou entre em outra partida.',
    'death.anotherMatch': 'Quer entrar em outra partida?',
    'death.retry': 'TENTAR NOVAMENTE',
    'death.otherMatch': 'OUTRA PARTIDA',
    'death.continue': 'CONTINUAR',
    'death.ranking': '🏁 Ranking desta partida',
    'leaderboard.title': '🏅 Top 10 Sobreviventes',
    'leaderboard.empty': 'Nenhum placar registrado ainda.',
    'leaderboard.anon': 'Anônimo',
    'sidebar.status': 'STATUS',
    'sidebar.name': 'Nome:',
    'sidebar.time': 'Tempo:',
    'sidebar.record': 'Recorde:',
    'sidebar.round': 'Rodada:',
    'sidebar.alive': 'Vivos:',
    'sidebar.nextDrop': 'Próximo drop:',
    'sidebar.buff': 'Buff:',
    'sidebar.lives': 'Vidas:',
    'sidebar.spectating': '💀 Espectando...',
    'hud.alive': 'Vivos:',
    'hud.buff': 'Buff:',
    'buff.tank': 'Tanque',
    'buff.speedster': 'Velocista',
    'buff.glider': 'Planar',
    'buff.none': 'Nenhum',
    'exit.label': 'Sair da partida',
    'exit.mobile': 'SAIR',
  },
  en: {
    'loading.checking': 'Loading game...',
    'loading.errorTitle': 'Could not connect to the server',
    'loading.errorMsg': 'Check your connection and try again.',
    'loading.retry': 'TRY AGAIN',
    'start.chooseName': 'Choose your name',
    'start.ballColor': 'Ball color:',
    'start.play': 'START GAME',
    'start.instructions':
      'Use A/D or ←/→ to move, W or Space to jump (×2 in the air). Shift to Dash (pushes opponents). Enter also starts. Grab the drops: 🍄 red = Tank, 🍄 purple = Speedster, 💎 blue = Glider (one buff at a time).',
    'game.roundOver': 'NEW ROUND',
    'death.title': 'YOU FELL!',
    'death.placed': 'You placed {pos}º of {total} {players}',
    'death.playersOne': 'player',
    'death.playersMany': 'players',
    'death.timeSurvived': 'Time survived: {time}s',
    'death.livesGone':
      'Your lives are gone. Keep playing in the next round or join another match.',
    'death.anotherMatch': 'Want to join another match?',
    'death.retry': 'TRY AGAIN',
    'death.otherMatch': 'ANOTHER MATCH',
    'death.continue': 'CONTINUE',
    'death.ranking': '🏁 Match ranking',
    'leaderboard.title': '🏅 Top 10 Survivors',
    'leaderboard.empty': 'No scores registered yet.',
    'leaderboard.anon': 'Anonymous',
    'sidebar.status': 'STATUS',
    'sidebar.name': 'Name:',
    'sidebar.time': 'Time:',
    'sidebar.record': 'Best:',
    'sidebar.round': 'Round:',
    'sidebar.alive': 'Alive:',
    'sidebar.nextDrop': 'Next drop:',
    'sidebar.buff': 'Buff:',
    'sidebar.lives': 'Lives:',
    'sidebar.spectating': '💀 Spectating...',
    'hud.alive': 'Alive:',
    'hud.buff': 'Buff:',
    'buff.tank': 'Tank',
    'buff.speedster': 'Speedster',
    'buff.glider': 'Glider',
    'buff.none': 'None',
    'exit.label': 'Leave match',
    'exit.mobile': 'EXIT',
  },
};

// Detecta o idioma: preferência salva > idioma do navegador (pt) > inglês.
export function detectLanguage() {
  try {
    const saved = localStorage.getItem('lang');
    if (saved === 'pt' || saved === 'en') return saved;
  } catch {
    // localStorage indisponível: segue para a detecção.
  }
  const nav = (navigator.languages?.[0] || navigator.language || '').toLowerCase();
  return nav.startsWith('pt') ? 'pt' : 'en';
}

// Traduz uma chave do dicionário atual, substituindo {placeholders}.
export function translate(dict, key, vars) {
  let str = dict[key] ?? key;
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      str = str.replaceAll(`{${k}}`, v);
    }
  }
  return str;
}
