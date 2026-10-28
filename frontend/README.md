# 🎮 Survive the Destruction — Frontend

Interface web do jogo **Survive the Destruction** (React 19 + Vite 7 + MUI 7 + Canvas API).

> Documentação completa do projeto em [`../README.md`](../README.md).

## Como rodar

```bash
npm install
npm run dev
```

Abra `http://localhost:5173`. Por padrão as URLs do backend apontam para produção (Render);
para usar um backend local, copie `.env.example` para `.env` e defina:

```
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_URL=http://localhost:8080
```

## Scripts

| Script | Descrição |
| ------ | ------ |
| `npm run dev` | Servidor de desenvolvimento (HMR) |
| `npm run build` | Build de produção em `dist/` |
| `npm run lint` | ESLint |
| `npm run preview` | Serve o build de produção localmente |
| `npm test` | 11 testes unitários de rede/readiness/sprites, via node:test |
| `npm run test:e2e` | Chromium real desktop/mobile com backend e banco locais |

Para E2E, siga [`../docs/demonstracao.md`](../docs/demonstracao.md). A suíte usa Vite em `localhost:4173` e backend em `127.0.0.1:18080`, sem acessar produção.

## Estrutura

```text
src/
  App.jsx     # Home/jogo e idioma compartilhado; jogo importado sob demanda
  Game.jsx    # Coordenação da cena, Canvas, HUD, efeitos e carregamento
  components/ # Home explicativa e personalização, sem I/O de backend
  network/    # Sessão WebSocket isolada + readiness HTTP cancelável
  hooks/      # Controles de teclado e cleanup de listeners
  renderer/   # Sprites e paleta Canvas sem React/rede
  sfx.js      # Efeitos sonoros (Web Audio API)
  i18n.js     # Dicionário PT/EN
  physics.js  # (não usado em produção) réplica da física p/ referência futura
  index.css   # Reset e estilos globais
assets/       # Imagens de carregamento, logo, etc.
```

A home abre imediatamente. Após **Jogar agora → personalização → Entrar na arena**, a tela de carregamento consulta `GET /api/health` e `/api/config` até obter configuração válida. Sair cancela as requisições. A sessão possui reconexão com backoff/jitter e aviso de que reconectar cria um novo jogador. Veja o [protocolo](../docs/websocket.md).
