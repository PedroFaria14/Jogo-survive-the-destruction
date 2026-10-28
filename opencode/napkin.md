# Guardanapo (Napkin)

## Regras de Curadoria

- Repriorizar a cada leitura; manter apenas orientações recorrentes, não histórico de execução.
- Máximo de 10 itens por categoria. Cada item contém data e ação explícita.

## Execução e Validação (Prioridade Máxima)

1. **[2026-10-28] Isolar demonstrações dos serviços já abertos**
   Faça isto em vez disso: use `127.0.0.1:18080` no backend E2E e `localhost:4173` no Vite; configure `ALLOWED_ORIGINS` e URLs explícitas. Não reutilize portas 8080/5173 sem verificar: IPv4 e IPv6 podem atender processos diferentes e validar o servidor errado.
2. **[2026-10-28] Não depender do banco/Render de produção para testar**
   Faça isto em vez disso: siga `docs/demonstracao.md`, usando PostgreSQL Docker descartável e credenciais locais; preserve `.env` e encerre apenas os recursos criados pela própria execução.
3. **[2026-10-28] O deploy usa pnpm, não o lockfile npm**
   Faça isto em vez disso: use pnpm 10.34.9 fixado no manifesto e valide `install --frozen-lockfile` ao alterar dependências. Inclua o lockfile pnpm no mesmo commit; atualizar apenas `package-lock.json` não corrige o build da Vercel.

## Confiabilidade de Shell e Comandos

1. **[2026-10-28] Race detector no Windows sem CGO**
   Faça isto em vez disso: execute `go test -race ./...` em container Linux com o backend montado read-only; não confunda ausência de compilador C com falha dos testes.

## Regras de Comportamento do Domínio

1. **[2026-10-28] Input após respawn**
   Faça isto em vez disso: mantenha o heartbeat de input completo da sessão e teste teclas seguradas ao perder vidas. O servidor limpa as teclas no respawn e não receberia novo keydown de uma tecla que continua pressionada.
2. **[2026-10-28] Identidade é local à conexão/sala**
   Faça isto em vez disso: trate `(room_id, player_id)` como identidade temporária; reconexão não recupera score, vidas ou sala. Não use `player_id` como identificador global de placar/participação.
