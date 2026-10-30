# 🎮 Survive the Destruction — Frontend

Interface web do jogo **Survive the Destruction** (React 19 + Vite 7 + MUI 7 + Canvas API).

> Documentação completa do projeto em [`../README.md`](../README.md).

## Como rodar

```bash
corepack pnpm install --frozen-lockfile
corepack pnpm dev
```

Abra `http://localhost:5173`. Por padrão as URLs do backend apontam para produção (Render);
para usar um backend local, copie `.env.example` para `.env` e defina:

```
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_URL=http://localhost:8080
```

## Scripts

Gerenciador de referência: **pnpm 10.34.9**, fixado no manifesto. Com Corepack indisponível, use `npx --yes pnpm@10.34.9 <comando>`. O deploy usa o lockfile pnpm; consulte o [guia da Vercel](../docs/deploy-vercel.md).

| Script | Descrição |
| ------ | ------ |
| `pnpm dev` | Servidor de desenvolvimento (HMR) |
| `pnpm build` | Build de produção em `dist/` |
| `pnpm lint` | ESLint |
| `pnpm preview` | Serve o build de produção localmente |
| `pnpm test` | Testes unitários de rede/readiness/sprites/idioma, via node:test |
| `pnpm test:e2e` | Chromium real desktop/mobile com backend e banco locais |
| `pnpm generate:social-preview` | Gera o JPEG de compartilhamento em `public/`, usando a arte do jogo |

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

O idioma inicial é **inglês**, independentemente do idioma do navegador. A seleção manual de português/inglês é persistida em `preferredLanguage`; o estado antigo salvo automaticamente em `lang` não força mais português.

Em produção, a API usa `https://api.survive.phfaria.com` e o WebSocket usa `wss://api.survive.phfaria.com/ws`. URLs são resolvidas em `src/network/backend_config.js`; espaços/barras finais são removidos, e um WebSocket não configurado deriva da API. Confira também as variáveis na Vercel e a origem permitida no Render no [guia de deploy](../docs/deploy-vercel.md#backend-atual-vercel-e-render).

A miniatura de compartilhamento é `public/social-preview.jpg` (1200 × 630), referenciada por metadados estáticos Open Graph/Twitter no HTML. Veja [como validar e atualizar a prévia no LinkedIn](../docs/deploy-vercel.md#miniatura-no-linkedin-e-outras-redes).
