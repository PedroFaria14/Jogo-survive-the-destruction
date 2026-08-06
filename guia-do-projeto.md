# 📘 Guia do Projeto — SURVIVE THE DESTRUCTION

## Descrição

Jogo multiplayer em tempo real em que a arena de ilhas flutuantes é destruída aos poucos. O jogador controla uma bolinha que precisa pular entre as ilhas, coletar power-ups e empurrar oponentes, enquanto os blocos desaparecem debaixo dos pés. Quem sobreviver mais tempo vence.

## Objetivo

- Sobreviver ao maior tempo possível em cada rodada.
- Escalar as ilhas antes que o mapa seja destruído.
- Usar power-ups para virar Tanque, Velocista ou Planar.
- Empurrar oponentes com o dash para eliminar a concorrência.
- Acumular placar por tempo sobrevivido e aparecer no Top 10.

## Tecnologias

| Camada | Tecnologia |
| ------ | ------ |
| Frontend | React 19, Vite 7, MUI 7, Canvas API |
| Backend | Go 1.25, gorilla/websocket |
| Banco | PostgreSQL (Neon) |
| Deploy | Vercel (frontend) + Render (backend) |

## Instalação

### Pré-requisitos

- Node.js 20+ e npm
- Go 1.25+
- PostgreSQL acessível (local, Docker ou Neon)

### Backend

```bash
cp .env.example .env   # na raiz do repositório
# preencha DATABASE_URL com a conexão do seu PostgreSQL
cd backend
go mod download
go run .
```

O servidor sobe em `http://localhost:8080`.

> O backend não inicia sem `DATABASE_URL` válida — isso é intencional.

### Frontend

```bash
cd frontend
npm install
cp .env.example .env   # opcional: aponte para o backend local
# VITE_WS_URL=ws://localhost:8080/ws
# VITE_API_URL=http://localhost:8080
npm run dev
```

Abra `http://localhost:5173`.

> Sem `.env`, o frontend usa os endpoints de produção (Render) como fallback.

## Execução

1. Suba o backend (`go run .` em `backend/`).
2. Suba o frontend (`npm run dev` em `frontend/`).
3. Acesse a tela inicial, escolha o nome e a cor da bolinha.
4. Clique em **INICIAR JOGO** e sobreviva à destruição.

No celular, o jogo trava em **paisagem** e oferece controles de toque na tela. A orientação em paisagem também é aplicada à tela de carregamento, com aviso de rotação se o aparelho estiver em retrato.

## Variáveis de Ambiente

### Raiz (backend)

| Variável | Obrigatória | Descrição |
| ---- | ---- | ---- |
| `DATABASE_URL` | Sim | Conexão com o PostgreSQL |
| `HTTP_ADDR` | Não | Endereço do servidor (padrão `:8080`) |
| `ALLOWED_ORIGINS` | Não | Origens permitidas (padrão `http://localhost:5173`) |

### Frontend

| Variável | Obrigatória | Descrição |
| ---- | ---- | ---- |
| `VITE_WS_URL` | Não | URL do WebSocket (fallback: produção) |
| `VITE_API_URL` | Não | URL base da API (fallback: produção) |

## Comandos

```bash
# Backend
cd backend
go test ./...          # testes
go build ./...         # build
go vet ./...           # análise estática

# Frontend
cd frontend
npm run lint           # eslint
npm run build          # build de produção
npm run preview        # serve o build localmente
```

## Funcionalidades

- **Multiplayer em tempo real** (WebSocket, sala única).
- **Arena procedural** — ilhas sorteadas a cada rodada.
- **Destruição contínua** — um bloco é destruído a cada 3s (fica vermelho por 1s antes).
- **Quadrado perdido** — plataforma extra dourada que aparece de vez em quando em posição aleatória.
- **Power-ups**: 🍄 Tanque, 🍄 Velocista, 💎 Planar.
- **Cor da bolinha** personalizada e lembrada no navegador.
- **Física polida**: pulo duplo, coyote time, jump buffer, dash com knockback, anti-trava.
- **Vidas e rodadas**: 3 vidas por rodada, placar por tempo, Top 10.
- **Mobile**: paisagem, tela cheia, toque, safe-area e telas de carregamento sorteadas.
- **Tela de carregamento em tela cheia**: fundo integral + spinner + "Procurando partida...".
- **Health check**: a tela de carregamento só sai quando `/api/health` confirma que o backend ligou (auto-retry ~4s, erro após 5 min).

## Estrutura

```text
backend/
  main.go              # Bootstrap, env, HTTP e shutdown
  controller/          # hub.go (loop do jogo) + client.go (WebSocket)
  models/              # movement.go (física/arena), powerup.go (buffs)
  routes/              # routes.go (rotas + CORS)
  service/             # poster.go (placar no PostgreSQL)
frontend/
  src/App.jsx          # UI, canvas, WebSocket, controles, telas
  src/sfx.js           # efeitos sonoros
  src/index.css        # estilos globais
  assets/              # imagens (telas de carregamento, etc.)
```

## Observações

- A física roda **no servidor**; o frontend só renderiza a posição autoritativa. Todos os jogadores ficam sincronizados.
- O front aguenta o **cold start do Render**: fica consultando `/api/health` até o servidor responder, então carrega `/api/config` e inicia.
- Cores duplicadas de bolinha são possíveis (escolha livre de cada jogador).
- Com a destruição a cada 3s, o jogo é mais frenético que a versão anterior (era 5s).
- Adicionar novas telas de carregamento é só colocar arquivos com prefixo `carregamento*` em `frontend/assets/` — elas entram no sorteio automaticamente.
- Nunca commite `.env` reais; use sempre os `.env.example`.
