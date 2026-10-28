# Arquitetura e avaliação técnica

## Estilo escolhido

Monólito de jogo em Go, com fronteiras entre transporte, orquestração de salas, regras do simulador e persistência; frontend React separado. É uma escolha proporcional ao projeto: **não é necessário introduzir microserviços, Kafka ou uma arquitetura Clean completa para demonstrar engenharia profissional aqui**.

```text
App.jsx → Home (explicação + personalização, sem I/O de backend)
       └→ import dinâmico de Game.jsx
           ├→ network/backend_ready.js → HTTP /api/health + /api/config
           ├→ hooks/use_game_controls.js → intenção do jogador
           ├→ network/game_session.js ↔ WebSocket /ws
           └→ renderer/sprites.js + requestAnimationFrame → Canvas

routes.NewHandler → upgrade/limites/origem → Hub.ServeClient
  ReadPump → Hub.Command → Room.handleCommand → models.GameState
  Hub.Run(ctx) → Room.tick → regras de física/rodada → snapshot da sala
  Room.broadcastState → Client.Send → WritePump → navegador
  Room.saveFinalScores → ScoreService → PostgresScoreService → PostgreSQL
```

## Responsabilidades e concorrência

| Área | Responsabilidade | Não deve fazer |
|---|---|---|
| `routes` | HTTP, CORS, upgrade, origem e limites de handshake | Física/decisões de rodada. |
| `controller/client.go` | Um reader e um writer por conexão; deadlines e fechamento | Alterar mapas da sala diretamente. |
| `controller/hub.go` | Admitir/registrar/remover clientes, distribuir salas, serializar eventos, shutdown | Escrever queries ou implementar física. |
| `controller/room.go` | Ciclo da partida, comandos, broadcast somente para seus clientes, encaminhar scores | Processar eventos de outra sala. |
| `models` | Estado, geração, física, colisões, drops, respawn | Depender de HTTP, WebSocket, PostgreSQL ou React. |
| `service/poster.go` | Porta `ScoreService` e implementação PostgreSQL existente | Controlar o jogo ou bloquear o tick com I/O. |

`Hub.Run(ctx)` é o único escritor das salas, jogadores e mapas do simulador. Os pumps comunicam eventos por canais. `ConnCount` é atômico; mutex de ciclo de vida impede adicionar pumps enquanto o shutdown já aguarda o WaitGroup. A reserva é feita **antes** do upgrade e liberada se ele falhar.

O transporte fecha exatamente uma vez (`closeOnce`) e gera no máximo um evento de remoção (`unregisterOnce`). A fila `Send` só é fechada pela remoção na sala, dentro do loop do Hub. Cliente lento é removido em vez de atrasar o tick de todos. Não adicionar writers concorrentes de dados no socket.

Gravações capturam ID, nome, score e duração **por valor** antes de iniciar a goroutine: nunca leem o `Player` mutável enquanto a próxima rodada o reinicia. O Hub acompanha gravações pendentes com WaitGroup. Consultas/boot do banco têm prazo de 5 s; gravação mantém o comportamento existente de até duas tentativas de 5 s.

## Ciclo de vida

1. `main` configura sinais e banco, cria Hub e inicia `Run(ctx)`.
2. `routes.NewHandler` retorna um mux por instância, com upgrader e limiter próprios (sem alterar `http.DefaultServeMux`).
3. O navegador só monta `Game` depois da personalização. Health/config têm timeout e cancelamento ao sair; o jogo só conecta após a configuração válida.
4. `SIGINT/SIGTERM` cancela o Hub, impede novas admissões, encerra conexões e aguarda pumps/gravações. HTTP e Hub compartilham prazo global de 15 s; o banco fecha depois da espera.
5. A home não exige servidor ligado, não força paisagem e não abre WebSocket. Gameplay mobile usa controles de toque/paisagem best-effort, conforme suporte do navegador.

## O que torna a base mais profissional

- Regras autoritativas no servidor, sem confiar no score/posição enviados pelo cliente.
- Salas isoladas por estado e broadcast, com capacidade explícita.
- Contrato [WebSocket](websocket.md) documentando também limitações e semântica de reconexão.
- Dependências HTTP isoladas por instância, serviço de placar substituível em testes e domínio sem dependência de transporte.
- Sessão de rede independente de React e dos desenhos, com testes de timers, reconexão e descarte de eventos antigos.
- Home carregada separadamente do bundle do jogo, textos PT/EN e controles de teclado isolados.
- Testes de domínio, integração WebSocket, execução com detector de races e E2E reais, não apenas screenshots estáticas.

## Limites conhecidos — não confundir com prontidão de produção em escala

1. **Escala:** salas vivem na memória de um único processo, com um loop para todas. Réplicas não compartilham partidas. Exige roteamento de sessões/particionamento de salas antes de escalar horizontalmente. `MaxClients=300` é proteção, não benchmark comprovando 300 jogadores a 60 Hz.
2. **Banda:** snapshots completos a cada tick; sem deltas, compressão ou interpolação. Medir bytes/s e tempo de tick antes de otimizar.
3. **Persistência:** scores não têm ID global de participação nem constraint de idempotência. Uma resposta de INSERT perdida seguida de retry pode duplicar registros; `ScoreSaved` evita repetição na memória, não entrega exactly-once. Não há outbox, fila durável ou recuperação de gravações após crash.
4. **Backpressure de gravação:** cada placar usa uma goroutine e o pool limita o banco a 10 conexões; não existe worker pool/limite explícito de jobs pendentes. Adicionar fila limitada se a carga real justificar.
5. **Segurança:** sem autenticação; origem não substitui identidade. O limiter atual usa `X-Forwarded-For` quando presente; em exposição direta, esse header pode ser forjado. Restringir entrada a proxy confiável e revisar a política de IP antes de produção pública de maior alcance. Não há rate limit por mensagem autenticada.
6. **Observabilidade:** logs existentes ainda são majoritariamente texto, sem métricas de tick/fila, tracing ou alertas. Não foi introduzida uma stack de observabilidade externa nesta entrega.
7. **Frontend:** `Game.jsx` ainda coordena cena, efeitos e HUD e pode ser subdividido quando houver evolução nessas áreas. Sprites, rede, input e apresentação inicial já têm fronteiras próprias. As imagens JPEG originais ainda podem ser otimizadas.
8. **Testes:** E2E cobre Chromium desktop/mobile emulado. Não comprova Safari/iOS físico, load test, latência em rede móvel, indisponibilidade de banco ou todos os caminhos de power-up. O jogo Canvas não oferece experiência equivalente para leitores de tela; a home/formulário são semânticos, mas isso não certifica acessibilidade integral do gameplay.
9. **Tempo e acoplamento:** simulador usa relógio real e aleatoriedade; evoluir para fontes de tempo/seed injetáveis se cenários determinísticos mais complexos forem necessários. `ScoreService` ainda fica no pacote de infraestrutura e inclui `Close`, podendo ser reduzido no consumidor em uma evolução dedicada.

## Critério de evolução

Priorizar mudanças motivadas por problemas medidos: teste de carga → frequência/custo do snapshot; confiabilidade de placares → identidade de participação + idempotência; operação pública → proxy confiável + métricas + limites por mensagem. Manter o monólito enquanto estas fronteiras forem suficientes.

Não há pasta `.agent/`, catálogo de erros ou biblioteca de envelope INTEGRA neste repositório. Preservamos os endpoints e formatos públicos existentes; não foi criado endpoint REST nem introduzida uma dependência de governança de outra plataforma.
