# Deploy do frontend na Vercel

## Configuração do projeto

No painel da Vercel, confira:

| Opção | Valor |
|---|---|
| Production Branch | `main` |
| Root Directory | `frontend` |
| Framework | Vite |
| Output Directory | `dist` |
| Install Command | Definido em `frontend/vercel.json`: `npx --yes pnpm@10.34.9 install --frozen-lockfile` |
| Build Command | Definido em `frontend/vercel.json`: `npx --yes pnpm@10.34.9 run build` |

Remova overrides antigos de instalação/build no painel que contrariem essa configuração. O `vercel.json` da raiz do repositório não é a configuração do frontend quando o Root Directory é `frontend`.

Configure `VITE_API_URL` e `VITE_WS_URL` para o backend correto. Essas variáveis são incorporadas no build; alterá-las exige novo deploy. O CSP em `frontend/vercel.json` permite o backend Render atual; se trocar de domínio, revise também essa lista.

## Gerenciador e lockfile

O frontend usa **pnpm 10.34.9**, fixado em `package.json#packageManager`. A instalação da Vercel também usa essa versão explicitamente, sem depender da versão global ou de habilitar Corepack no ambiente remoto.

- `pnpm-lock.yaml` é o lockfile de referência para o deploy.
- `pnpm-workspace.yaml` permite o script de instalação do `esbuild`, com uma lista explícita de pacotes aprovados.
- `package-lock.json` foi preservado como arquivo legado; não é usado pela instalação da Vercel. Evite alternar gerenciadores ao adicionar dependências.
- Ao alterar dependências, rode `pnpm install` com a versão fixada e inclua `package.json` **e** `pnpm-lock.yaml` no mesmo commit.
- Não use `--no-frozen-lockfile` para esconder divergência em CI. Valide com instalação congelada antes de publicar.

Com Corepack disponível, execute em `frontend/`:

```powershell
corepack pnpm install --frozen-lockfile
corepack pnpm test
corepack pnpm lint
corepack pnpm build
```

Alternativa independente de instalação global: substitua `corepack pnpm` por `npx --yes pnpm@10.34.9`.

## Falha `ERR_PNPM_OUTDATED_LOCKFILE`

O deploy de `f02b5d9` adicionava `@playwright/test` no manifesto, mas não no lockfile pnpm. A Vercel interrompia a instalação antes do build, mantendo o deploy anterior ativo. A correção é sincronizar o lockfile, não alterar o roteamento para mascarar o problema.

Após publicar um commit com os arquivos corrigidos, confirme nos logs:

1. O commit é o novo commit, não o `f02b5d9`.
2. A versão usada é pnpm `10.34.9` e a instalação congelada conclui.
3. O Vite gera `dist/` e o deploy termina como **Ready**.
4. A URL de produção `/` mostra **The floor disappears. You stay.**, antes de qualquer conexão ao jogo.

Um redeploy do commit antigo repete o erro. Commit, push e publicação não são executados automaticamente pelo agente.

## Idioma, imagens e fluxo esperados

- `/`: home explicativa em inglês por padrão, inclusive em navegador em português.
- Botão **PT**: troca para português. A escolha manual fica em `localStorage.preferredLanguage` e é respeitada nas próximas visitas.
- A antiga chave `lang`, que também guardava detecção automática do navegador, não define mais o idioma inicial.
- **Play now → personalização → Enter the arena**: só então carrega a configuração e conecta ao backend.
- Logo do jogo é importado de `frontend/assets/logo.jpeg`: o Vite gera uma URL com hash, em vez de requisitar o caminho inexistente `/logo.jpeg`.
- Hero e telas de carregamento também usam assets processados pelo Vite. O favicon permanece em `/logo.png`, que existe em `public/`.

Para repetir os testes no **build de produção** com backend local, use o roteiro em [demonstracao.md](demonstracao.md). O modo `E2E_PREVIEW=1` serve `dist/` em vez do dev server; as URLs de backend locais devem estar presentes durante o build de teste.

## Miniatura no LinkedIn e outras redes

O HTML estático de `frontend/index.html` contém Open Graph e Twitter Cards, com a URL oficial `https://survive.phfaria.com/`. Essas tags estão presentes antes de executar React, pois os robôs de compartilhamento não devem depender de JavaScript.

A imagem é `frontend/public/social-preview.jpg`, JPEG de **1200 × 630 px**, copiado para `dist/social-preview.jpg` pelo Vite. O `og:image` utiliza a URL absoluta `https://survive.phfaria.com/social-preview.jpg`, com MIME, dimensões e texto alternativo. O favicon não substitui essa imagem de compartilhamento.

Para gerar novamente a miniatura a partir da arte existente, em `frontend/`:

```powershell
corepack pnpm exec playwright install chromium
corepack pnpm generate:social-preview
```

Versione o JPEG gerado. A geração é manual, somente em desenvolvimento; o deploy não baixa Chromium nem executa o gerador.

Para validar somente a prévia sobre o build, sem backend ou PostgreSQL:

```powershell
corepack pnpm build
$env:E2E_PREVIEW = '1'
corepack pnpm test:e2e tests/e2e/social-preview.spec.js
```

O teste desabilita JavaScript, envia o User-Agent `LinkedInBot/1.0` e verifica as tags no HTML, a resposta HTTP e o tipo/conteúdo/dimensões reais do JPEG servido pelo preview. Não acessa a imagem remota antes de ela ser publicada.

### Após publicar o novo commit

1. Aguarde o deploy ficar **Ready** e promover a versão nova para o domínio oficial.
2. Abra `https://survive.phfaria.com/social-preview.jpg` sem login. Deve responder `200`, com a miniatura, não HTML ou uma página de autenticação.
3. Acesse [LinkedIn Post Inspector](https://www.linkedin.com/post-inspector/) e inspecione `https://survive.phfaria.com/` para solicitar uma nova leitura.
4. Confira a prévia no formulário de link em destaque. Se ele mantiver o resultado antigo, reabra o formulário depois da atualização de cache.

O LinkedIn mantém cache próprio: publicar os metadados não garante substituição instantânea de uma prévia já armazenada. O teste local comprova o artefato gerado; o resultado final no LinkedIn só pode ser conferido após deploy e reprocessamento.
