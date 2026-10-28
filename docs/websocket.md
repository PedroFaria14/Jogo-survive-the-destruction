# Protocolo WebSocket do jogo

Contrato atual de `GET /ws` (upgrade HTTP/1.1 para WebSocket). Implementado em `backend/routes/routes.go`, `controller/client.go`, `controller/room.go` e `models/`. O cliente de referência é `frontend/src/network/game_session.js`.

## Transporte e identificação

- Produção: `wss://<backend>/ws`; desenvolvimento: `ws://localhost:8080/ws`.
- O handshake exige um `Origin` não vazio e presente em `ALLOWED_ORIGINS`. Isso não é autenticação; não há conta, token ou sessão persistente.
- Payloads são **frames de texto UTF-8 com um objeto JSON por mensagem**. Não há subprotocolo negociado ou versão no payload atual.
- Cada conexão recebe um jogador e é alocada automaticamente em uma sala com vaga. Capacidade: **5 jogadores por sala**, até **300 conexões/vagas de handshake** no processo.
- `player_id` só é único dentro da sala. A identidade completa é `(room_id, player_id)` e só vale durante aquela conexão. Não usar o nome como identificador.
- A partida começa no registro da conexão, não ao receber `join`. O comando `join` configura apenas nome/cor e pode ser reenviado.

## Sequência de conexão

```text
Cliente                                      Servidor
GET /ws + Origin --------------------------> valida origem, rate limit e capacidade
                 <------------------------- 101 Switching Protocols
                 <------------------------- init {player_id, room_id}
join {name, color} ------------------------> configura o jogador já criado
input {left, right, jump, dash} -----------> atualiza input da própria conexão
                 <------------------------- snapshot completo da sala, sem "type"
                 <------------------------- snapshots seguintes (~62,5 Hz nominais)
close ------------------------------------> remove jogador, descarta sala se vazia
```

O cliente de referência envia `join` no `open`, sem esperar por `init`; o servidor registra o jogador antes de iniciar a leitura dos comandos. `init` é a primeira mensagem enviada pelo servidor. As quatro mensagens a seguir são objetos independentes, não um único array ou texto concatenado.

## Cliente → servidor

### `join`

```json
{ "type": "join", "name": "Pedro", "color": "#f2b544" }
```

| Campo | Tipo | Regra |
|---|---|---|
| `type` | string | Literal `join`. |
| `name` | string | Opcional. Remove espaços externos e controles ASCII; limita a 20 caracteres Unicode. Vazio é permitido. |
| `color` | string | Opcional. Apenas `#RRGGBB`, normalizado para minúsculas. Inválido/vazio vira `""`; o renderer usa âmbar. |

### `input`

```json
{ "type": "input", "left": false, "right": true, "jump": false, "dash": false }
```

Os quatro campos são booleanos. **Enviar o estado completo das teclas**, não apenas a tecla alterada: campos ausentes são decodificados como `false`. O servidor mantém a intenção até outro comando, respawn ou fim da conexão. Pulo/dash possuem regras de borda e cooldown no servidor; não enviar posições, velocidades ou placar.

O cliente envia nas mudanças de input e repete o estado a cada **100 ms** enquanto o socket está aberto. Isso mantém o input coerente após o servidor limpá-lo no respawn. Ao perder foco/visibilidade ou desativar os controles, envia todas as teclas como `false`. O `input` de jogadores eliminados é ignorado.

### `restart`

```json
{ "type": "restart" }
```

Só reinicia o próprio jogador **eliminado**, durante rodada ainda ativa, restaurando vidas e zerando o tempo da participação. É ignorado para jogadores vivos e durante a contagem de fim de rodada. Não recria a sala nem reinicia todos os jogadores.

### Validação e mensagens desconhecidas

- Tamanho máximo recebido: **4.096 bytes** por mensagem; exceder encerra a conexão pelo mecanismo do gorilla/websocket.
- JSON malformado, campos com tipos incompatíveis e tipos de comando desconhecidos são ignorados. Frames binários não são processados como comandos.
- Campos extras são ignorados. Não há ACK, mensagem `error`, sequência ou confirmação por comando. O próximo snapshot é a observação do resultado.
- Não há catálogo de códigos de erro do jogo ou envelope REST no WebSocket. Nenhum código fictício foi introduzido na documentação.

## Servidor → cliente

### `init`

```json
{ "type": "init", "player_id": "player_1", "room_id": "sala_1" }
```

Guardar a identidade antes de processar os snapshots. Os IDs são exemplos; não são credenciais, não devem ser fixados no cliente e podem mudar após reconexão/reinício do processo.

### Snapshot completo da sala

Não possui `type`. Os dicionários `players`, `arena_tiles` e `power_ups` **não são arrays**. Substituir o estado anterior pelo novo snapshot, sem fazer merge de entidades removidas.

Exemplo estrutural reduzido, não uma arena completa:

```json
{
  "players": {
    "player_1": {
      "id": "player_1", "name": "Pedro", "color": "#f2b544",
      "x": 450, "y": 275, "vx": 0, "vy": 0,
      "on_ground": true, "is_dead": false, "score": 12,
      "lives": 3, "dash_cd": 0, "buff": "", "buff_remaining": 0,
      "jumps_used": 0, "respawn_left": 0
    }
  },
  "arena_tiles": {
    "tile_3_4": { "id": "tile_3_4", "x": 400, "y": 300, "is_falling": false, "is_active": true, "kind": "top" }
  },
  "power_ups": {
    "powerup_1": { "id": "powerup_1", "type": "blue_crystal", "x": 450, "y": 220, "landed": false }
  },
  "status": "playing", "round": 1, "round_over": false,
  "countdown": 0, "arena_width": 800, "arena_height": 1000, "drop_countdown": 2.5,
  "room_id": "sala_1", "room_players": 1, "room_capacity": 5
}
```

| Campo | Tipo/unidade | Significado |
|---|---|---|
| `players` | objeto `id → Player` | Somente jogadores da sala, incluindo eliminados/espectadores. |
| `arena_tiles` | objeto `id → ArenaTile` | Plataformas atuais. Ausência ou `is_active=false` significa sem piso. |
| `power_ups` | objeto `id → PowerUp` | Drops presentes. A ausência no próximo snapshot significa remoção. |
| `status` | string | `waiting`, `playing` ou `round_over`; usar `round_over` para o overlay de fim. |
| `round` | inteiro | Número da rodada na sala. |
| `round_over` | boolean | Simulação de rodada encerrada/contagem em andamento. |
| `countdown` | inteiro, segundos | Contagem para a próxima rodada (delay nominal de 3 s). |
| `arena_width`, `arena_height` | número, pixels lógicos | Dimensões da rodada. A largura procedural pode diferir de `/api/config`. |
| `drop_countdown` | número, segundos | Tempo até o próximo drop. |
| `room_id` | string | Identificador da sala atual. |
| `room_players`, `room_capacity` | inteiros | Conexões na sala (inclui eliminados) e capacidade máxima. |

**Player:** `id`, `name`, `color` são strings. `x/y` são coordenadas em pixels lógicos; `vx/vy` são velocidades do simulador por tick, não pixels/segundo. `on_ground`/`is_dead` são booleanos. `score`, `lives`, `jumps_used` e `respawn_left` são inteiros; `score` é o tempo desde o último respawn que zera o placar, não a soma das vidas. `dash_cd` e `buff_remaining` são segundos numéricos. `buff` vale `""`, `red_mushroom`, `purple_mushroom` ou `blue_crystal`. `respawn_left > 0` indica espera antes de voltar após perder uma vida.

**ArenaTile:** `id` string, `x/y` números, `is_active/is_falling` booleanos; `kind` vale `top`, `mid`, `bottom` ou `lost`. `is_falling=true` avisa a destruição após aproximadamente 1 s.

**PowerUp:** `id/type` strings, `x/y` números, `landed` booleano. `type` usa os mesmos três valores dos buffs não vazios.

Campos internos (endereço de transporte, relógios, teclas mantidas, marcação de gravação, dados do banco) não são parte do snapshot público. O endereço `conn_id`, anteriormente serializado, foi removido por privacidade; clientes não devem depender dele.

## Rodadas e persistência

- O servidor simula a cada 16 ms: frequência nominal de **62,5 ticks/s**, aproximada como ~60 Hz; não é garantia de FPS, latência ou entrega sob carga.
- Arena sem blocos ativos, todos eliminados ou apenas um sobrevivente em multiplayer encerram a rodada. Solo não encerra só por haver um jogador vivo.
- Durante a contagem, o servidor continua enviando snapshots. A rodada seguinte gera outra arena, restaura vidas e mantém conexões, IDs e perfis.
- Eliminados e sobreviventes ao fim da rodada têm o placar final encaminhado ao PostgreSQL. Valores abaixo de 3 s são descartados. `/api/scores` retorna o Top 10 global; não é histórico completo e não expõe `player_id`.

## Keepalive, backpressure e reconexão

| Mecanismo | Comportamento |
|---|---|
| Ping do servidor | A cada 54 s; é frame de controle, não JSON. O navegador responde pong automaticamente. |
| Deadline de leitura | 60 s, renovado por pong. |
| Deadline de escrita | 10 s por mensagem/ping. |
| Fila de saída | 8 mensagens por cliente; cheia, o cliente lento é removido e a conexão fechada sem bloquear as outras salas. |
| Watchdog do navegador | 15 s para conectar/receber estado, renovado em cada snapshot válido. |
| Retry do navegador | 1 s, 1,5 s, 2,25 s… até 8 s, com jitter de 0–300 ms. Só zera após snapshot válido. |

Reconexão cria **outro jogador** e pode levar a outra sala. Não existe token de retomada, garantia de manter score/vidas/sala ou replay de comandos. O cliente reenvia nome, cor e input e troca o ID ao receber o novo `init`. Ao sair para a home, timers, listeners, HTTPs pendentes e socket são encerrados; não há reconexão depois disso.

No shutdown, o Hub para de admitir clientes, fecha sockets e aguarda os pumps e as gravações já iniciadas. O binário permite até 15 s para encerramento global. O close de controle é best-effort; quedas de rede e shutdown podem aparecer como fechamento anormal. Não interpretar todo close como eliminação no jogo.

## Falhas antes do upgrade

- `403`: origem não permitida.
- `429`: mais de 20 tentativas de handshake por IP na janela de um minuto.
- `503`: capacidade atingida ou Hub encerrando.
- `405`: método diferente de GET.

O navegador não expõe o corpo/status HTTP detalhado do handshake para o callback WebSocket; o cliente mostra reconexão e permite voltar à home. Estas respostas HTTP existentes são texto, não mensagens JSON do protocolo.

## Evolução e testes

Adicionar campos é compatível com o cliente atual, que ignora os desconhecidos. Renomear/remover campos públicos, alterar unidades ou adicionar `type` aos snapshots exige evolução coordenada do cliente e deste contrato. Não existe versionamento formal negociado nesta versão.

Validação: `backend/routes/websocket_test.go` usa sockets reais e `httptest`; `frontend/tests/unit/game_session.test.js` testa o transporte com socket/relógio controlados; `frontend/tests/e2e/game.spec.js` usa dois navegadores e PostgreSQL real. Consulte [a demonstração](demonstracao.md).
