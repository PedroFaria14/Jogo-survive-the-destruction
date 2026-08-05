# Documentação Técnica — SURVIVE THE DESTRUCTION

Documentação técnica para manutenção, onboarding de desenvolvedores e trabalho com agentes de IA (OpenCode).

## Visão Geral Técnica

Jogo multiplayer em tempo real composto por:

- **Frontend**: React 19 + Vite 7 + MUI 7. Renderização do jogo em `<canvas>` com `requestAnimationFrame`, desacoplada do ciclo de render do React. WebSocket para estado em tempo real.
- **Backend**: Go 1.25 + gorilla/websocket. Toda a lógica do jogo (física, colisão, destruição da arena, power-ups, placar) roda no servidor em um loop de ~60 FPS.
- **Banco**: PostgreSQL (Neon). Apenas o placar (`scores`) é persistido.

Princípio central: **o servidor é a fonte da verdade**. O frontend apenas renderiza o `GameState` recebido pelo WebSocket e envia inputs; nunca calcula física localmente.

## Arquitetura

```text
Navegador (React + Canvas)
        │  WS /ws (JSON, 60 FPS)
        ▼
backend/controller/hub.go  ──►  Loop Run() com ticker de 16ms
        │                         (único goroutine que escreve no GameState)
        ▼
backend/models/*.go
        │  GameState (Players, ArenaTiles, PowerUps)
        ▼
broadcastState()  ──►  JSON serializado  ──►  todos os Clients
```

Fluxo de dados:

- `hub.Run()` consome os canais `Register`, `Unregister`, `Command` e o ticker de física.
- A cada tick: física → power-ups → destruição → quadrado perdido → placar → broadcast.
- Entradas do jogador chegam pelo canal `Command` (`PlayerCommand`) e são aplicadas por `GameState.ProcessInput`.

## Estrutura de Pastas

```text
backend/
  main.go              # Bootstrap: .env, DATABASE_URL, ScoreService, Hub, rotas, HTTP
  controller/
    hub.go             # Loop principal (Run/tick), clientes, rodadas, broadcast
    client.go          # ReadPump/WritePump do WebSocket (por conexão)
    hub_test.go        # Testes de fim de rodada
  models/
    movement.go        # GameState, Player, ArenaTile, física, arena procedural,
                       #   destruição (CheckArenaDestruction) e quadrado perdido
                       #   (SpawnLostTile)
    powerup.go         # Power-ups/buffs e spawn de drops (UpdatePowerUps)
    movement_test.go   # Testes de geração, física, cor e quadrado perdido
  routes/
    routes.go          # /ws, /api/scores, /api/config + CORS/origin whitelist
  service/
    poster.go          # Interface ScoreService + PostgresScoreService
frontend/
  index.html           # viewport trava zoom; título
  src/
    App.jsx            # Tudo de UI: canvas, WebSocket, controles, telas, HUD
    sfx.js             # Efeitos sonoros via Web Audio
    index.css          # Reset, overflow oculto, canvas
  assets/              # carregamento*.jpeg (sorteadas), logo.jpeg
```

## Fluxos Principais

### 1. Entrada e início de partida

```text
Abre a página
↓
checkBackend(): fetch GET /api/config (timeout 8s via AbortController)
↓ 'checking' → imagem de carregamento sorteada (import.meta.glob)
↓ 'error'    → "Servidor indisponível" + TENTAR NOVAMENTE
↓ 'ready'
Tela inicial: nickname + cor da bolinha (input type=color, localStorage 'ballColor')
↓ INICIAR JOGO
Fullscreen + window.screen.orientation.lock('landscape') (Android)
↓
WebSocket /ws
↓ open → envia { type:'join', name, color }
↓ init  → recebe { type:'init', player_id }
↓
Servidor passa a enviar GameState completo a ~60 FPS
```

### 2. Gameplay (por frame)

```text
hub.tick(now) [16ms]
↓
saveDeadScores() → placar de mortos salvos uma única vez (goroutine)
↓
UpdatePowerUps(now) → spawn/expiração de drops
↓
ApplyPhysics() → gravidade, pulo, dash, colisões, knockback
↓
CheckArenaDestruction() → a cada 3s marca um tile ativo como falling (FallDelay 1s)
↓
ExpireFallingTiles(now) → remove tiles cujo FallAt passou (vira buraco)
↓
SpawnLostTile(now) → a cada 8–14s cria tile kind:"lost" em célula vazia
↓
atualiza Score dos vivos (tempo desde StartTime)
↓
broadcastState() → JSON para todos os clientes
```

Frontend:

```text
onmessage → gameStateRef.current = data (sem re-render)
↓
requestAnimationFrame → draw() → renderiza sky, ilhas, power-ups, jogadores
↓
setUi() throttled a cada 250ms → apenas HUD/overlays re-renderizam
```

### 3. Destruição da arena

- `CheckArenaDestruction` escolhe um tile ativo aleatório a cada `BreakInterval` (3s).
- O tile fica `IsFalling=true` com `FallAt = now + FallDelay` (1s) — visual vermelho no frontend.
- `ExpireFallingTiles` remove o tile quando `now > FallAt` → `IsActive=false` (buraco).
- Quando `NoActiveTiles()` → `roundShouldEnd()` → `round_over` → countdown → `ResetRound()`.

### 4. Power-ups

- Servidor controla spawn/expiração (`UpdatePowerUps`) e aplicação de buffs no `Player`.
- Frontend detecta pickup do próprio jogador comparando snapshots de drops (`powerUpPrevRef` → `currentPUIds`).
- Buffs: `red_mushroom` (Tanque, raio ×2), `purple_mushroom` (Velocista), `blue_crystal` (Planar).

### 5. Morte e fim de rodada

- `is_dead=true` → jogador vira espectador (fantasma).
- `saveDeadScores` persiste o placar uma vez por jogador morto.
- 3 vidas (`MaxLives`) → ao zerar, prompt "VOCÊ CAIU!" com ranking.
- `roundShouldEnd`: sem tiles ativos, todos mortos, ou (multiplayer) restar 1 vivo.

## Mapeamento de Chamadas

### Join com nome e cor

```text
App.jsx (join) ──► WS /ws ──► hub.Command ──► hub.go:236 (join)
      └─► models.SanitizeColor(color) ──► p.Name / p.Color
      ──► próximo broadcastState inclui players[].color
```

### Input do jogador

```text
App.jsx sendFrame() (por frame) ──► WS ──► ReadPump ──► PlayerCommand{Cmd, PlayerID}
      ──► hub.Command ──► hub.go:261 ProcessInput ──► ApplyPhysics no próximo tick
```

### Placar

```text
Frontend fetch GET /api/scores (a cada 10s) ──► routes.getScoresHandler
      ──► ScoreService.GetTopScores() ──► SELECT ... LIMIT 10

Morte do jogador ──► hub.saveDeadScores() ──► ScoreService.SaveScore (goroutine)
      ──► INSERT INTO scores
```

### Config

```text
checkBackend() e fallback ──► GET /api/config ──► routes.getConfigHandler
      ──► models.GetGameConfig() (constantes compartilhadas)
```

## Endpoints

### `GET /api/config`

- **Objetivo**: constantes compartilhadas do jogo (evita drift backend/frontend).
- **Entrada**: nenhuma.
- **Saída**: objeto `GameConfig` (arena, física, power-ups).
- **Regras**: GET apenas; sem autenticação.

### `GET /api/scores`

- **Objetivo**: Top 10 de sobreviventes.
- **Entrada**: nenhuma.
- **Saída**: `[{ player_id, name, score_seconds }]`.
- **Regras**: GET apenas; retorna `[]` quando vazio (nunca `null`).

### `WS /ws`

- **Objetivo**: conexão do jogo.
- **Handshake**: `Origin` deve estar em `ALLOWED_ORIGINS`, senão a conexão é recusada.
- **Entrada (cliente)**:
  - `{ "type": "join", "name": "...", "color": "#rrggbb" }`
  - `{ "type": "input", "left": bool, "right": bool, "jump": bool, "dash": bool }`
  - `{ "type": "restart" }`
- **Saída (servidor)**:
  - `{ "type": "init", "player_id": "..." }` logo após conectar.
  - `GameState` serializado a ~60 FPS (sem campo `type`).

### `WS /ws` — estrutura do `GameState` (pacote de tick)

Enviado a ~60 FPS para cada cliente conectado. Não possui campo `type`
(pacotes com `type` são mensagens de controle).

```text
{
  "round": 1,
  "round_over": false,
  "countdown": 3,
  "drop_countdown": 1.2,
  "arena_width": 800,
  "arena_height": 1000,
  "arena_tiles": { "tile_0_0": { "id", "x", "y", "kind": "top"|"mid"|"bottom"|"lost",
                                  "is_active": true, "is_falling": true } },
  "players": {
    "<player_id>": {
      "id", "name", "color": "#rrggbb",
      "x", "y", "vx", "vy", "is_dead": false, "score": 12,
      "lives": 3, "buff": "red_mushroom"|"purple_mushroom"|"blue_crystal"|"",
      "buff_remaining": 4.0, "dash_cd": 0.5
    }
  },
  "power_ups": {
    "powerup_1": { "id", "type": "red_mushroom"|"purple_mushroom"|"blue_crystal",
                   "x", "y", "landed": false }
  }
}
```

- `arena_tiles`: mapa `id → tile`. `kind` em `movement.go`: `top` (grama),
  `mid` (terra), `bottom` (ponta de pedra) e `lost` (plataforma extra gerada
  aleatoriamente, com halo dourado no frontend).
- `players[].color`: cor da bolinha do jogador (`#rrggbb`), sanitizada no join.
- `power_ups`: **mapa** `id → drop` (não é array). Tipos: `red_mushroom`
  (Tanque), `purple_mushroom` (Velocista), `blue_crystal` (Planar).
- `countdown`: contagem regressiva de início de rodada; `drop_countdown`:
  tempo até o próximo power-up cair.
- Quando `round_over === true`, o frontend mostra o overlay "NOVA RODADA"
  com o `countdown`.

## Banco de Dados

- **SGBD**: PostgreSQL (Neon, conexão via `DATABASE_URL`).
- **Driver**: `github.com/lib/pq` (driver puro Go; `pgx`/`pgxpool` **não** é usado).
- **Pool**: `database/sql` com limites explícitos (`SetMaxOpenConns(10)`,
  `SetMaxIdleConns(5)`, `SetConnMaxLifetime(30m)`).
- **Tabela única**: `scores` (placar de sobrevivência).

### Schema (migração inline em `service/poster.go` → `initSchema`)

```sql
CREATE TABLE IF NOT EXISTS scores (
  id SERIAL PRIMARY KEY,
  player_id VARCHAR(50) NOT NULL,
  player_name VARCHAR(50) NOT NULL,
  score_seconds INTEGER NOT NULL,
  duration_ms BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

- A migração roda no boot do backend (idempotente), dentro de
  `NewPostgresScoreService` (`initSchema`).
- `SaveScore` tem **timeout de contexto (5s) e 1 retry**. Placares triviais
  (menos de `MinScoreToSave = 3s`) são descartados no `hub.go`.

### Regras de leitura

- `ScoreService.GetTopScores()` retorna os **10 melhores** tempos
  (ordem `DESC`).
- A resposta JSON **não expõe** `player_id` interno (campo oculto).
- Para tempos de jogo multiplayer (cada um é um "match"), sem regra de
  agrupamento — os scores são globais por jogador.

## Variáveis de Ambiente

### Backend (Render)

| Variável | Obrigatória | Descrição |
|---|---|---|
| `DATABASE_URL` | **Sim** | Connection string do PostgreSQL (Neon). Sem ela o backend **não sobe**. Definida como **secret no painel do Render** (`render.yaml` usa `sync: false` — nunca versionar a senha). |
| `PORT` | Não | Porta do servidor HTTP (Render injeta automaticamente). |
| `HTTP_ADDR` | Não | Endereço HTTP local (default `:8080`); usado quando `PORT` não existe. |
| `ALLOWED_ORIGINS` | Sim (produção) | Lista de origens permitidas para CORS/WebSocket (ex.: `https://jogo-survive-the-destruction.vercel.app`). |

### Frontend (Vercel)

| Variável | Obrigatória | Descrição |
|---|---|---|
| `VITE_API_URL` | **Sim** | URL base da API (ex.: `https://jogo-survive-the-destruction.onrender.com`). |
| `VITE_WS_URL` | **Sim** | URL do WebSocket (ex.: `wss://jogo-survive-the-destruction.onrender.com/ws`). |

- Fallbacks em `frontend/src/App.jsx` apontam para as URLs de produção do
  Render. Para dev local, definir as variáveis Vite apontando para
  `http://localhost:8080`.

## Integrações

- **PostgreSQL (Neon)**: persistência dos scores. Conexão criada no boot e
  compartilhada via `*sql.DB` injetado nos handlers.
- **WebSocket (gorilla/websocket)**: comunicação em tempo real do jogo.
- **CORS**: restrito a `ALLOWED_ORIGINS` (configurado no `main.go`).

## Comandos

### Backend

```bash
cd backend
go test ./...          # testes unitários (física, tiles, sanitize, config)
go run .               # sobe API + WS em localhost:8080
go build -o jogo .     # build do binário
```

### Frontend

```bash
cd frontend
npm install
npm run dev            # servidor de desenvolvimento Vite
npm run lint           # ESLint (usar antes de finalizar mudanças)
npm run build          # build de produção (Vite)
```

- O frontend em dev chama `http://localhost:8080` por padrão; configurar as
  variáveis Vite se a API estiver em outra porta.

## Decisões Técnicas Importantes

1. **Estado do jogo fica no servidor**; o frontend renderiza o que recebe
   (thin client). Física e morte de tiles são processadas em `movement.go`.
2. **60 FPS no canvas**: `GameState` chega a cada tick (~60 Hz) e o `rAF`
   desenha a partir do `gameStateRef.current`, desacoplado do re-render do
   React. O HUD/texto é atualizado com throttle de 250 ms.
3. **Tiles por id + autotile**: `kind` derivado da vizinhança (`hasTileAbove`)
   para grama/terra/ponta de pedra. O tile `lost` é gerado como plataforma
   extra (intervalo 8–14 s) e entra no pool normal de destruição.
4. **Destruição progressiva**: tiles caem após `FallDelay` (1 s) do
   `BreakInterval` (3 s) e são removidos do `is_active`. Piso seguro nas
   bordas até o fim da rodada.
5. **Power-ups e buffs**: drops em intervalos configuráveis (4 s), um buff
   por vez. Tipos: `red_mushroom` (Tanque), `purple_mushroom` (Velocista),
   `blue_crystal` (Planar/gravidade lunar).
6. **Cor da bolinha**: o cliente escolhe `#rrggbb` (armazenado em
   `localStorage`) e o servidor sanitiza no join (`SanitizeColor`) com
   fallback para o dourado `#f2b544`.
7. **Loading screen com gate**: o frontend não desenha o jogo até o
   `GET /api/config` responder (timeout de 8 s), garantindo que os
   parâmetros de física vêm do servidor. As imagens `carregamento*.jpeg`
   são sorteadas por visita via `import.meta.glob`.
8. **i18n simples**: dicionário PT/EN em `frontend/src/i18n.js`, detecção
   automática por `navigator.language`, persistência em `localStorage.lang`
   e botão de troca em tela inicial, loading, HUD mobile e sidebar.
9. **Botão SAIR**: `handleExit` fecha o WebSocket (via cleanup do efeito de
   `gameStarted`), zera todos os refs de estado e volta ao menu sem
   recarregar a página (ao contrário de "OUTRA PARTIDA" que recarrega).
10. **Reconexão**: o WebSocket reconecta com backoff (máx. 8 s) se cair
    durante a partida e **reenvia o estado atual de input** (teclas seguradas
    continuam funcionando). Tela cheia/orientação são travadas apenas no
    mobile.
11. **Segurança do backend**: limite global de conexões (`MaxClients = 300`)
    + rate limit de handshake por IP; `recover()` nas goroutines (Hub e
    pumps) evita que um panic derrube o servidor; headers de segurança nas
    respostas; pool do banco com limites; `player_id` oculto no leaderboard.
12. **Placar do vencedor**: ao encerrar a rodada (`roundOver`), o placar dos
    sobreviventes também é persistido (`saveFinalScores`). Placares triviais
    (< 3 s) são descartados (anti-spam de restart).

## Gotchas

- **Driver do banco**: o projeto usa `database/sql` com `lib/pq`. Não
  adicionar `pgx`/`pgxpool` sem necessidade.
- **CORS do WS**: a validação de `Origin` está no handshake do WebSocket.
  Origem não listada → conexão recusada (ajuda a evitar conexões de sites
  não autorizados).
- **`document.documentElement.requestFullscreen()`**: só existe em navegador;
  checar com `if (isTouch && document.documentElement.requestFullscreen)`.
  iOS Safari ignora fullscreen e o lock de orientação.
- **Textos hardcoded**: textos de UI ficam em `frontend/src/i18n.js` —
  **nunca** escrever textos de UI soltos em `App.jsx` (falha na tradução).
- **`import.meta.glob`**: as telas de carregamento são carregadas com
  `eager: true`; não referenciar imagens por caminho dinâmico.
- **Pacotes de controle vs estado**: mensagens com campo `type` são
  controles; o `GameState` de tick não tem `type`. Não confundir no
  `onmessage`.
- **Save de score em goroutine**: o `ScoreService.SaveScore` roda em
  goroutine; não bloquear o tick de física com escrita no banco.
- **Mudanças de física**: alterar constantes em `movement.go` sem atualizar
  `/api/config` quebra o fallback do frontend.
- **Secrets nunca versionados**: `render.yaml` usa `sync: false` para
  `DATABASE_URL` (secret configurado no painel do Render) e `.env.example`
  contém apenas placeholders. Rodar `gitleaks`/`git secrets` antes de
  commitar.
- **`touch` mobile**: além de `onPointerUp`/`onPointerLeave`, os botões de
  toque precisam de `onPointerCancel` — sem ele, uma interrupção do sistema
  (notificação, gesto) deixa a tecla "presa" e o jogador anda sozinho.

## Convenções

- **Português no código**: comentários e mensagens de UI em pt-BR (UI vai
  para o dicionário i18n).
- **Nomes em inglês**: identificadores, tipos e funções em inglês
  (PascalCase exportado, camelCase local).
- **CSS em JS**: usar MUI `sx` e o objeto `PALETTE` (paleta central em
  `frontend/src/App.jsx`); não criar arquivos `.css` soltos.
- **Paleta de cores**: `PALETTE` no topo de `App.jsx` centraliza todas as
  cores do jogo (céu, ilhas, queda, texto). Adicionar novas cores lá.
- **Sound effects**: via `frontend/src/sfx.js` (`sfx.unlock()`,
  `sfx.click()`, `sfx.tileBreak()`, `sfx.roundStart()`); chamar `sfx.unlock()`
  no primeiro evento de usuário.
- **Comunicação com o servidor**: JSON no WebSocket; estado do jogo sempre
  espelhado em `gameStateRef` para o canvas e em `ui` (throttled) para o HUD.

## Guia Para Agentes (OpenCode)

Este arquivo é o guia técnico do projeto. Ao trabalhar aqui:

1. **Sempre** executar `go test ./...` (backend) e `npm run lint` +
   `npm run build` (frontend) antes de declarar uma tarefa concluída.
2. Para mudanças de física/gameplay, alterar `movement.go` **e** o contrato
   em `/api/config` (verificar que `cfg` no frontend bate com o servidor).
3. Para mudanças de UI, **sempre** mover os textos para o dicionário
   `frontend/src/i18n.js` (chaves pt/en) — nunca deixar string solta.
4. Para novas telas de carregamento, adicionar arquivo
   `carregamento<N>.jpeg` em `frontend/assets/` — o glob os inclui
   automaticamente.
5. Nunca alterar o schema do banco sem confirmar a migração inline em
   `service/poster.go` (`initSchema`) e o `ScoreService`.
6. Verificar as variáveis de ambiente do deploy (Render/Vercel) antes de
   mudar URLs de API/WS.
7. Respeitar o fluxo do `AGENTS.md` (skills/agents/references) e responder
   em pt-BR, com etapas curtas e validação final.
