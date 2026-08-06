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

## Estrutura

```text
src/
  App.jsx     # UI, canvas, WebSocket, controles e telas (início/carregamento/jogo)
  sfx.js      # Efeitos sonoros (Web Audio API)
  i18n.js     # Dicionário PT/EN
  physics.js  # (não usado em produção) réplica da física p/ referência futura
  index.css   # Reset e estilos globais
assets/       # Imagens de carregamento, logo, etc.
```

A tela de carregamento consulta `GET /api/health` em loop (~4s) até o backend responder;
só então carrega o `/api/config` e inicia a partida.
