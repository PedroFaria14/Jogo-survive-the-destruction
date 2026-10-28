# Executar e demonstrar o jogo

## Demonstração manual

1. Configure PostgreSQL e `DATABASE_URL` conforme o README; nunca use banco de produção para testes automatizados.
2. Em `backend/`, execute `go run .`. O endereço padrão é `:8080`.
3. Em `frontend/`, configure `VITE_API_URL=http://localhost:8080` e `VITE_WS_URL=ws://localhost:8080/ws`, e execute `corepack pnpm dev`.
4. Abra `http://localhost:5173` em duas janelas. A home deve aparecer imediatamente, mesmo com backend desligado.
5. A home começa em inglês: clique em **Play now**, personalize nomes/cores diferentes e clique em **Enter the arena** nas duas janelas. O botão **PT** permite usar a interface em português e salva essa escolha explícita.
6. Confira o mesmo ID de sala e `2/5` jogadores no HUD. Use A/D ou setas para mover, Espaço/W para pular, Shift para dash. A outra janela deve refletir o movimento.
7. Observe os blocos avisando antes de cair e os drops de power-up. Após eliminação, confira contagem e nova rodada. Placar inferior a 3 s não é salvo; o ranking atualiza a cada 10 s.
8. Nas ferramentas de desenvolvimento, coloque a rede offline/online: deve aparecer **Reconectando**, seguido de uma nova participação ao recuperar a conexão. Não existe retomada do jogador anterior.
9. Clique em **Sair da partida**: retorna à home, o outro cliente passa a mostrar uma conexão a menos e não deve haver reconexão da janela que saiu.
10. No celular, a home pode ser usada em retrato. A partida oferece toque em paisagem; fullscreen/orientation lock dependem do navegador.

## E2E automatizado com PostgreSQL descartável

Pré-requisitos: Go 1.25+, Node **20.19+ ou 22.12+** (compatível com Vite 7), npm e Docker. As portas **55432**, **18080** e **4173** devem estar livres. A suíte usa portas diferentes das habituais para não conectar por engano a um dev server já aberto.

Na raiz, crie um PostgreSQL temporário, sem volume persistente:

```powershell
docker run --rm -d --name survive-demo-postgres -p 127.0.0.1:55432:5432 -e POSTGRES_USER=survive -e POSTGRES_PASSWORD=local-demo-only -e POSTGRES_DB=survive postgres:17-alpine
```

`local-demo-only` é uma credencial pública de banco descartável local, não um secret de deploy. Aguarde `docker exec survive-demo-postgres pg_isready -U survive -d survive` retornar prontidão.

Em outro terminal PowerShell, dentro de `backend/`:

```powershell
$env:DATABASE_URL = 'postgres://survive:local-demo-only@127.0.0.1:55432/survive?sslmode=disable'
$env:HTTP_ADDR = '127.0.0.1:18080'
$env:ALLOWED_ORIGINS = 'http://localhost:4173'
# PORT, se estiver definida no ambiente, tem prioridade sobre HTTP_ADDR.
go run .
```

Em `frontend/`:

```powershell
corepack pnpm install --frozen-lockfile
corepack pnpm exec playwright install chromium
corepack pnpm test
corepack pnpm test:e2e
```

A configuração Playwright inicia/encerra o Vite na porta 4173 com URLs locais explícitas, sem fallback para Render. Não reutiliza um Vite já aberto. O backend é iniciado separadamente para que falhas de banco/boot sejam visíveis. pnpm 10.34.9 é o gerenciador de referência, também usado na Vercel. Se Corepack estiver indisponível, substitua `corepack pnpm` por `npx --yes pnpm@10.34.9`.

### Validar o build de produção

Com o mesmo backend de teste em execução, em um terminal exclusivo para teste dentro de `frontend/`:

```powershell
$env:VITE_API_URL = 'http://127.0.0.1:18080'
$env:VITE_WS_URL = 'ws://127.0.0.1:18080/ws'
corepack pnpm build
$env:E2E_PREVIEW = '1'
corepack pnpm test:e2e
```

Esse modo serve os arquivos de `dist/` via `vite preview`, verificando a home na raiz e imagens com URLs finais geradas pelo Vite. As URLs locais precisam ser definidas **antes do build**; o preview não altera variáveis já incorporadas no JavaScript. A Vercel executa seu próprio build com as variáveis do painel, não usa este `dist/` local.

Use banco descartável **vazio** em cada demonstração de persistência: o endpoint expõe apenas o Top 10. A suíte cria nomes com um sufixo único para não aceitar scores de execuções anteriores; em uma base já cheia de tempos superiores, um score novo pode não entrar no ranking. Recrie o container local de teste nesse caso, sem limpar bancos externos.

### O que a suíte verifica

Além dos quatro cenários de jogo abaixo, `tests/e2e/social-preview.spec.js` verifica os metadados de compartilhamento e o JPEG com JavaScript desabilitado. Esse teste isolado não precisa de backend/banco; veja [o guia da miniatura](deploy-vercel.md#miniatura-no-linkedin-e-outras-redes).

- Home em inglês mesmo com navegador PT-BR e a chave antiga `lang=pt`; troca manual e persistência de português; personalização sem nenhuma chamada `/api/*`.
- Imagem hero, favicon e logo do jogo carregados de fato, sem depender de caminhos de assets inexistentes.
- Home mobile em retrato, sem overflow horizontal, com CTA acessível.
- Dois contextos independentes de navegador, mesma sala e perfis visíveis.
- Input real de teclado observado nos snapshots recebidos pelo outro cliente.
- Tile em destruição, encerramento de rodada e incremento do número de rodada.
- Score consultável pelo endpoint HTTP real conectado ao PostgreSQL.
- Fechamento real do WebSocket e reconexão com novo `player_id`.
- Saída para home e remoção do jogador na outra janela.
- Ausência de exceções JavaScript nas duas janelas durante o fluxo.

Não são mockados snapshots ou respostas de scores na demonstração multiplayer. O teste intercepta `/api/*` apenas no cenário de home independente do backend. O cenário de reconexão fecha o socket real por instrumentação de teste, sem alterar dados do servidor.

### Evidências e resultado

Capturas geradas em [evidence/](evidence/README.md). Logs de teste, screenshots de falha, vídeo de falha e traces ficam em `frontend/test-results/` (ignorados pelo Git).

Validação de referência em 28/10/2026: Windows, Go 1.26.5, Node 24.18, PostgreSQL 17 em Docker, Chromium Playwright 1.64.1. A suíte cobre quatro cenários E2E, incluindo o fluxo multiplayer e gameplay mobile em paisagem com eventos de toque reais pelo Chromium. O frontend possui **14 testes unitários**, além de lint/build e instalação congelada via pnpm 10.34.9. O detector de races Go foi validado em container Linux porque este Windows não possui compilador C/CGO habilitado. Consulte os resultados da execução atual antes de afirmar aprovação de um novo build.

Na correção do deploy/idioma/imagens, a instalação congelada, os 14 testes unitários, lint, build e os quatro E2E sobre **`dist/` servido por preview** passaram; `pnpm audit` não apontou vulnerabilidades conhecidas. As capturas em `evidence/` foram atualizadas com a home e personalização em inglês. O backend e o banco descartáveis dessa validação foram encerrados após o teste.

Para repetir o race detector na raiz sem instalar compilador C no Windows:

```powershell
docker run --rm --mount "type=bind,source=${PWD}\backend,target=/app,readonly" -w /app golang:1.26 go test -race ./...
```

### Limpeza

Encerre **somente** o backend de teste com Ctrl+C no terminal em que foi iniciado. Depois:

```powershell
docker stop survive-demo-postgres
```

Como o container foi criado com `--rm` e sem volume, os dados de demonstração são descartados. Não pare processos/containers de desenvolvimento que já estavam abertos antes da demonstração.
