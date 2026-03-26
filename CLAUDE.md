# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Projeto

**OK** — Framework de assistente pessoal de IA ultra-leve em Go.
Conecta LLMs a plataformas de mensagem via agentes autonomos com ferramentas.

- **Linguagem**: Go (CGO_ENABLED=0)
- **Licenca**: MIT

## Design — Produto SaaS Premium

Aparencia de produto pronto para vender. Pensar como designer de produto, nao programador.

- Menos elementos, mais impacto visual
- Espaco em branco e prioridade
- Cada elemento deve ter proposito claro
- Cortar tudo que nao agrega valor visual
- Clareza, hierarquia e estetica acima de tudo
- Evitar qualquer estilo "generico de dev"

## Principios de Design

1. **Problema humano primeiro** — Proposito em uma frase simples
2. **Simplicidade absoluta** — Leigo entende em 10 segundos ou simplifique
3. **Regra dos 3 cliques** — Nenhuma acao importante exige mais que 3 interacoes
4. **Nunca expor complexidade interna** — Sem logs, pipelines ou termos tecnicos ao usuario
5. **Arquitetura limpa** — Dominio, aplicacao, infra e interface separados
6. **Menos funcionalidades, mais clareza** — Remover o que nao resolve o problema principal

## Frontend — Performance e UX

UI moderna, limpa, consistente. Design minimalista. Mobile-first. Zero frameworks ou dependencias externas.

### Obrigatorio

- Evitar reflows/repaints desnecessarios
- Minimizar manipulacao de DOM
- CSS eficiente: classes reutilizaveis, evitar inline styles
- Preferir CSS sobre JS para animacoes
- Evitar listeners excessivos
- Bundle o menor possivel
- Responsividade mobile-first
- Acessibilidade basica (sem exagero)

### Codigo

- Simples, legivel e reutilizavel
- HTML, CSS e JavaScript vanilla apenas
- Nao adicionar complexidade desnecessaria
- Baixo uso de CPU/memoria

## Backend — Go

### Stack

- **HTTP**: Fiber
- **DB**: database/sql puro (glebarez/go-sqlite, modernc — sem CGO)
- **Config**: Viper
- **Logging**: Zap (estruturado)
- **Banco**: SQLite WAL mode, MaxOpenConns=15, busy_timeout=5000ms
- **Vector**: chromem-go (busca semantica em memoria)
- **Vision**: configurable via VISION_BASE_URL/API_KEY/MODEL
- **Function Calling**: native OpenAI tool calling via USE_NATIVE_TOOLS flag

### Arquitetura

Clean Architecture / Hexagonal:
- `/cmd` — entrypoints
- `/internal` — codigo privado
- `/domain` — entidades e regras de negocio
- `/application` — use cases
- `/infrastructure` — banco, APIs externas, adapters
- `/interfaces` — handlers HTTP, middlewares

Baixo acoplamento + inversao de dependencia.

### API — OpenAPI 3

Endpoints devem seguir convencoes OpenAPI 3, sem arquivo de definicao (sem openapi.json/yaml).

- Rotas RESTful: `GET /api/resources`, `POST /api/resources`, `GET /api/resources/:id`, `PUT /api/resources/:id`, `DELETE /api/resources/:id`
- Content-Type: `application/json` em requests e responses
- Status codes corretos: 200 (ok), 201 (created), 204 (no content), 400 (bad request), 404 (not found), 422 (unprocessable entity), 500 (internal error)
- Respostas de erro padronizadas: `{"error": "mensagem descritiva"}`
- Respostas de lista como array direto: `[{...}, {...}]`
- Versionamento via path quando necessario: `/api/v1/...`
- Nomes de recursos no plural e lowercase: `/api/users`, nao `/api/user` ou `/api/Users`

### Testes — Integracao + Unitarios

**Integracao** (`tests/integration/`):
- Fluxo completo: HTTP → handler → service → repository → DB
- Banco de teste separado (arquivo SQLite separado)
- Migrations automaticas no ambiente de teste
- **Nunca usar mocks de DB** nos testes de integracao

**Unitarios** (500+ tests across 20+ packages):
- `adapters/` — mock via interface AgentRunner, spy de chamadas
- `application/engine/` — httptest.NewServer para mock de LLM, mock Planner/Executor
- `application/` — mock de repositorios (interfaces do dominio), services completos
- `infrastructure/llm/` — httptest para Decide, Reflect, Stream
- `infrastructure/agent/tools/` — sandbox com t.TempDir(), tiers de seguranca
- `infrastructure/agent/` — safety gate, rate limiter, executor, confirmation manager
- `infrastructure/security/` — pattern matching do SecretScrubber
- `interfaces/http/middleware/` — Fiber test app com mock de sessao, recovery

**E2E Stress** (122 scenarios em 18 batteries):
- Tudo via API HTTP (`/api/agent/run`, `/api/conversations/...`, etc.)
- Baterias: FullBattery, ToolChaining, Security, Resilience, DirectMode, Memory, FTS5, ChatFlow, Concurrent, Offline, Delegate, SemanticSearch, ConcurrentImports, SchedulerCRUD, Confirmation, SkillBattery, BrowserActions, ConfigTool

### Praticas

- Codigo explicito — sem "magic"
- Uso correto de context.Context
- Interfaces bem definidas
- Minimizar custo de tokens e chamadas de API
- Nomes completos e descritivos — sem abreviacoes (`userService`, nao `usrSvc`)
- Nomes genericos proibidos (`Data`, `Info`, `Manager`)
- Funcoes curtas (10-20 linhas), responsabilidade unica
- Early return — evitar ifs aninhados
- Erros sempre com contexto (`fmt.Errorf("create user: %w", err)`)
- Se nao e obvio na primeira leitura, simplificar

### Logging

- **Console silencioso por padrao** — log level default e `error`. So loga quando tem problema real.
- Requests HTTP: logar em Debug (invisivel por padrao), exceto status 500+ que loga em Error
- Startup/shutdown: logar em Debug (invisivel por padrao)
- Logger injetado em todas as camadas via construtor (repository, service, handler)

### Debug Mode

Quando `DEBUG=true`, todas as camadas logam via `log.Debug(...)`:
- **Config**: valores carregados
- **Database**: queries logadas em Debug
- **Repository**: operacao, parametros, resultado
- **Service**: entrada de metodo, validacao, erros
- **Handler**: request body, response status, erros de parse
- Em producao (debug off): zero output — console limpo

### Evitar

- Frameworks monoliticos (Buffalo)
- ORMs (GORM removido — usar database/sql puro)
- Mocks de DB em testes de integracao
- Globals desnecessarios
- Service locator
- Overengineering

## Principios do Agent

1. **Previsibilidade > Inteligencia** — Comportamento consistente, nunca "esperto"
2. **Simplicidade > Flexibilidade** — Fluxos diretos e claros
3. **Determinismo > Autonomia** — Evitar decisoes abertas do LLM quando possivel
4. **Seguranca > Capacidade** — Nunca executar acoes perigosas sem controle
5. **80% dos casos devem ser simples** — Maioria dos inputs resolve com Direct Mode (1 chamada LLM, sem tools)

### Fluxo de Execucao (AgentEngine)

Todo input passa pelo AgentEngine que decide o caminho:

1. **LLM.Decide()** — LLM recebe system prompt + input e retorna JSON
2. Se `done=true` e `tool=""` → **resposta direta** (maioria dos inputs)
3. Se `tool` especificado e `done=true` → **execucao de 1 tool** (planner → executor)
4. Se `tool` especificado e `done=false` → **loop completo OBSERVE/PLAN/ACT/REFLECT**

O loop autonomo:
- **OBSERVE**: busca memorias relevantes, constroi contexto
- **PLAN**: Token Pruner comprime historico se >80% da janela. LLM gera plano com 2-6 steps
- **ACT**: executa cada step via planner → executor. `delegate` spawna sub-engine isolado
- **REFLECT**: Token Pruner comprime se necessario. LLM avalia: continue | replan | done | error

Streaming reativo:
- **PTY**: shell e repl usam pseudo-terminal (creack/pty) — output com cores ANSI character-by-character
- **LLM Token Streaming**: CreatePlanStreaming goteja tokens de raciocinio para o frontend em tempo real
- **WebSocket** (`/ws/agent`): canal bidirecional com state hydration (reconstroi UI no F5)
- **StreamingTool interface**: tools com PTY emitem chunks via `EmitStream` para o xterm.js

Defesas automaticas:
- **Sliding Window**: descarta metade antiga do historico se contexto >80% da janela — sem chamada LLM, O(1)
- **Output Truncation**: trunca outputs longos de tools com TruncateWithEllipsis — sem chamada LLM
- **Memory Condenser**: goroutine a cada 6h comprime memorias >7 dias via LLM (shutdown gracioso via stopCh)
- **SecretScrubber**: regex patterns (AWS, OpenAI, JWT, RSA, .env, GitHub, Slack) limpam segredos ANTES de enviar a LLM
- **Input grande**: se >8K chars, pula llmFast e usa llmHeavy direto no Decide()
- **Panic Recovery**: defer recover() em todos os handleMessage dos adapters
- **Timeouts**: context.WithTimeout (3min adapters, 5min WebSocket) em todas as chamadas ao agente
- **Zip Bomb Protection**: import limitado a 100MB via io.LimitReader
- **Delegate Reset**: contador de sub-agentes resetado por execucao
- **Thread Safety**: cachedPrompt com sync.RWMutex, cancelFunc do WS com sync.Mutex
- **Global Event Broadcast**: AgentService emite eventos para WebSocket hub de qualquer canal (chat, adapters, CLI)
- **Budget Accuracy**: Attempts conta tool executions (nao chamadas LLM) — usa MaxSteps do AgentLimits
- **Cache Invalidation**: cachedPrompt invalidado no inicio de cada Run/RunStream para incluir regras recentes

### Tool Safety (28 tools)

- `safe` — Execucao direta: echo, math, timestamp, json_parse, base64, text_extract, file_read (paginado, MIME filter), search, schedule, learn_rule, skill_loader (reads skill content)
- `restricted` — Validar input: http (bloqueia localhost/IPs internos), file_write (sandbox), folder_index (sandbox), browser (headless, recover panic, 7 action types — `wait` selector, `click` selector, `fill` selector+value, `js` script blocklist enforced, `screenshot`, `text` selector, `analyze` vision AI via VISION_MODEL), delegate (spawna sub-engine, max 3), web_search (DuckDuckGo, zero API key), skill_creator (creates .md files), config_manager (reads/writes agent config), gcal_manager (Google Calendar), gmail_read (Gmail read), gmail_send (Gmail send), sql_inspector (read-only SQL), python_rpa (Python scripts)
- `dangerous` — Exige confirmacao com preview head+tail: shell, file_edit, repl, docker_replicator (Docker containers)

### Skills System
- Runtime-expandable via `.md` files in `data/sandbox/skills/`
- Frontmatter YAML format (name, description) + markdown content
- `skill_creator` tool creates skills, `skill_loader` reads them
- Skills listed in system prompt automatically
- `FileSkillRepository` in `infrastructure/agent/skill_repository.go`

### Auto-Learn / Auto-Forget
- Patterns like "always", "never", "from now on", "remember this" auto-save as rules
- Patterns like "forget", "remove rule" auto-delete matching rules
- Rules appear in system prompt under "Learned Rules"

### Seguranca

- **SecretScrubber** intercepta prompts e memorias — remove chaves AWS, OpenAI, JWT, RSA, Bearer, .env secrets
- Audit log de todas execucoes de tools
- Rate limiting: shell 5/min, http 20/min, file_write 10/min
- Shell: 3 tiers de seguranca
  - **Tier 1** (bloqueio total): rm -rf /, dd if=/dev, mkfs, fork bombs, write em /dev /etc /proc /sys
  - **Tier 2** (requer confirmacao): sudo, rm -rf, chmod, chown, kill, shutdown, reboot
  - **Tier 3** (execucao normal): pipes, redirects e subshells permitidos

## Restricoes

- **Nunca usar iframes** — UI nativa/embutida
- **Nunca adicionar co-autor nos commits**
- **Nunca criar arquivos fora do diretorio do projeto**
- **CGO desabilitado** — alternativas pure-Go (modernc para SQLite)
- **Binario unico** — sem dependencias externas em runtime
- **Zero frameworks frontend** — HTML, CSS, JS vanilla apenas
