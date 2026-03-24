# OK — Arquitetura

## Visao Geral

```
+--------------------------------------------------------------------------+
|                           ENTRYPOINTS                                    |
|                                                                          |
|   cmd/ok/main.go              cmd/cli/main.go                            |
|   (HTTP Server + Adapters)    (Interactive REPL)                         |
|         |                            |                                   |
|         +--------+-------------------+                                   |
|                  |                                                       |
|                  v                                                       |
|   +----------------------------------------------+                       |
|   |     infrastructure/bootstrap/bootstrap.go    |                       |
|   |     (inicializacao compartilhada de agent)   |                       |
|   +--------------------+-------------------------+                       |
|                        |                                                 |
|                        v                                                 |
|   +----------------------------------------------+                       |
|   |           APPLICATION LAYER                  |                       |
|   |                                              |                       |
|   |   AgentService (centro de tudo)              |                       |
|   |   AgentEngine (loop OBSERVE/PLAN/ACT/REFLECT)|                       |
|   |   ChatService                                |                       |
|   |   ConversationService                        |                       |
|   |   EmbeddingService                           |                       |
|   |   SessionService                             |                       |
|   |   ImportService                              |                       |
|   |   SchedulerService                           |                       |
|   +--------------------+-------------------------+                       |
|                        |                                                 |
|                        v                                                 |
|   +----------------------------------------------+                       |
|   |         INFRASTRUCTURE LAYER                 |                       |
|   |                                              |                       |
|   |   LLM Client    Embedding Client             |                       |
|   |   Planner        Executor (+ SafetyGate)     |                       |
|   |   Memory         VectorStore                 |                       |
|   |   ConfigRepo     ExecutionRepo               |                       |
|   |   FeedbackRepo   AuditLog                    |                       |
|   |   RateLimiter    ConfirmationManager         |                       |
|   |   FileWatcher    Scheduler                   |                       |
|   |   SecretScrubber (security)                  |                       |
|   |   Tools (19, PTY via creack/pty)             |                       |
|   +--------------------+-------------------------+                       |
|                        |                                                 |
|                        v                                                 |
|   +----------------------------------------------+                       |
|   |           SQLite (modernc)                   |                       |
|   |   11 tabelas + 1 FTS5 virtual               |                       |
|   +----------------------------------------------+                       |
+--------------------------------------------------------------------------+
```

---

## Fluxo Principal: AgentEngine

```
  USER INPUT
       |
       v
  AgentEngine.RunLoop(input, emitter)
       |
       v
  OBSERVE: memory.SearchSemantic(input) + buildSystemPrompt()
       |
       v
  LLM.Decide(systemPrompt, context) --> Decision{tool, input, done}
       |
       +-- done=true, tool="" --> resposta direta (emit message + done)
       |
       +-- tool!="", done=true --> executeSingleStep (1 tool)
       |
       +-- tool!="", done=false --> loop completo:
              |
              v
           pruneContextIfNeeded() -- comprime historico se >80% da janela
              |
              v
           PLAN: LLM.CreatePlan() --> ExecutionPlan{steps, reasoning}
              |
              v
           ACT + REFLECT loop:
             for step in plan (budget check):
               Planner.Plan(decision) --> Plan{tool, input}
               Executor.Execute(plan) --> result
                 (se tool=delegate: spawna sub-engine isolado, max 3)
                 |
                 v
               pruneContextIfNeeded() -- comprime se necessario
               REFLECT: LLM.Reflect(goal, steps, result)
                 --> continue: proximo step
                 --> replan: novos steps (anti-loop: FilterRepeatedSteps)
                 --> done: resposta final
                 --> error: falha
              |
              v
           saveResults() + reflectAndLearn() (em transacao unica)
```

---

## Fluxo de Requests HTTP

```
  Browser / API Client
         |
         v
  +--------------------------------------------------------------+
  |                    FIBER HTTP SERVER                           |
  |                                                               |
  |  Middleware:  Recovery -> Logger -> Auth (session cookie)      |
  |                                                               |
  |  PUBLIC                                                       |
  |    GET  /health .......................... {"status":"ok"}    |
  |    GET  /login ........................... login.html         |
  |    POST /api/auth/login .................. session cookie     |
  |                                                               |
  |  PROTECTED (requer session)                                   |
  |                                                               |
  |  Pages                                                        |
  |    GET  /chat ............................ chat.html          |
  |    GET  /agent ........................... agent.html         |
  |    GET  /profile ......................... profile.html       |
  |                                                               |
  |  Conversations                                                |
  |    GET  /api/conversations/ .............. listar             |
  |    GET  /api/conversations/search?q= ..... buscar (FTS5)     |
  |    POST /api/conversations/ .............. criar              |
  |    GET  /api/conversations/:id/messages .. mensagens          |
  |    POST /api/conversations/:id/messages .. enviar (SSE)      |
  |    DELETE /api/conversations/:id ......... deletar            |
  |                                                               |
  |  Agent                                                        |
  |    POST /api/agent/run ................... execucao sincrona  |
  |    POST /api/agent/stream ................ execucao SSE       |
  |    GET  /api/agent/status ................ canais ativos      |
  |    GET  /api/agent/metrics ............... metricas           |
  |    GET  /api/agent/executions ............ historico          |
  |    GET  /api/agent/executions/:id ........ replay             |
  |    GET  /api/agent/limits ................ limites atuais     |
  |    PUT  /api/agent/limits ................ definir limites    |
  |    GET  /api/agent/config/:key ........... ler config         |
  |    PUT  /api/agent/config/:key ........... salvar config      |
  |                                                               |
  |  Config                                                       |
  |    GET  /api/config ...................... config publica      |
  |                                                               |
  |  Scheduler                                                    |
  |    GET  /api/scheduler/jobs .............. listar jobs        |
  |    POST /api/scheduler/jobs .............. criar job          |
  |    PUT  /api/scheduler/jobs/:id .......... atualizar job      |
  |    DELETE /api/scheduler/jobs/:id ........ deletar job        |
  |                                                               |
  |  Import                                                       |
  |    POST /api/import/chatgpt .............. importar ZIP       |
  |                                                               |
  |  Health                                                       |
  |    GET  /api/health/services ............. LLM + Embed        |
  |    POST /api/auth/logout ................. encerrar sessao    |
  |                                                               |
  |  WebSocket (bidirecional, hydration on reconnect)             |
  |    WS   /ws/agent ....................... stream + input      |
  +---------------------------------------------------------------+
```

---

## SSE Streaming (Chat)

```
  POST /api/conversations/:id/messages
         |
         v
  ChatService.SendMessage(onEvent callback)
         |
         v
  AgentService.RunStream(input, emit)
         |
         +-- Direct: emit(message) --> {"type":"message","content":"Ola"}
         |           emit(done)    --> {"type":"done"}
         |
         +-- 1 Tool: emit(step)    --> {"type":"step","tool":"math","status":"running"}
         |           emit(step)    --> {"type":"step","tool":"math","status":"done"}
         |           emit(message) --> {"type":"message","content":"14"}
         |           emit(done)    --> {"type":"done"}
         |
         +-- Loop:   emit(phase)   --> {"type":"phase","content":"observe"}
                     emit(step)    --> {"type":"step","tool":"http","status":"running"}
                     ...
                     emit(message) --> {"type":"message","content":"resultado"}
                     emit(done)    --> {"type":"done"}
```

---

## Tools do Agent (19)

```
  SAFE (execucao direta)
  +----------+ +----------+ +----------+ +----------+
  | echo     | | math     | | json_    | | base64   |
  +----------+ +----------+ | parse    | +----------+
  +----------+ +----------+ +----------+ +----------+
  | timestamp| | text_    | | file_    | | search   |
  +----------+ | extract  | | read     | +----------+
               +----------+ +----------+ +----------+
               +----------+              | schedule |
               | learn_   |              +----------+
               | rule     |
               +----------+

  RESTRICTED (validar input)
  +------------------+ +------------------+ +------------------+
  | file_write       | | http             | | folder_index     |
  | sandbox path     | | bloqueia         | | sandbox path     |
  +------------------+ | localhost/IPs    | +------------------+
                       | 10s timeout      | +------------------+
                       +------------------+ | browser          |
                                            | go-rod headless  |
                                            +------------------+
                                            +------------------+
                                            | delegate         |
                                            | spawna sub-engine|
                                            | max 3 por exec   |
                                            +------------------+

  DANGEROUS (exige confirmacao com preview head+tail)
  +------------------+ +------------------+ +------------------+
  | shell            | | file_edit        | | repl             |
  | 3 tiers segur.   | | confirmacao req  | | confirmacao req  |
  | 10s timeout      | |                  | |                  |
  +------------------+ +------------------+ +------------------+
```

---

## Seguranca

```
  Tool execution request
         |
         v
  +-------------------------------+
  | RateLimiter.Allow(tool)       |
  |   shell:      5/min           |
  |   http:       20/min          |
  |   file_write: 10/min          |
  |   outros:     sem limite      |
  +---------------+---------------+
                  |
                  v
  +-------------------------------+
  | SafetyGate.Check(tool, input) |
  |   safe       --> executa      |
  |   restricted --> valida input |
  |   dangerous  --> ErrRequires  |
  |                  Confirmation |
  +---------------+---------------+
                  |
                  v
  +-------------------------------+
  | tool.Run(input)               |
  +---------------+---------------+
                  |
                  v
  +-------------------------------+
  | AuditLog.Record(entry)       |
  |   tool, input, output,        |
  |   safety, approved, timestamp |
  +-------------------------------+
```

---

## Aprendizado (via AgentEngine)

```
  Execucao N                        Execucao N+1
  ----------                        ------------

  ACT: tool X executa               OBSERVE:
    |                                 |
    v                                 v
  REFLECT:                          memory.SearchSemantic(input, 5)
  reflectAndLearn()                   +-- "X: in->out [done]"
    +-- memoria: "X: in->out"        +-- "X falhou em 'Y' - reason"
    +-- memoria falha: "X falhou"      |
         |                             v
         v                           LLM recebe memorias no contexto
  agent_memory (persistido)            +-- usa historico para melhor plano
  agent_executions (persistido)
  (tudo em transacao unica)

  Background (a cada 6h):
  CondenseOldMemories()
    +-- busca facts >7 dias (batch 30)
    +-- llmFast resume em 1 paragrafo
    +-- delete velhas + insert sintese (atomico)
```

---

## Adapters (Messaging)

```
  +------------------------------------------------------+
  |                    ADAPTERS                           |
  |            (todos chamam AgentService.Run)            |
  |                                                      |
  |  +----------+  +----------+  +----------+            |
  |  | WhatsApp |  | Telegram |  | Discord  |            |
  |  | whatsmeow|  | bot-api  |  | discordgo|            |
  |  | QR login |  | polling  |  | gateway  |            |
  |  | owner    |  | ownerID  |  | ownerID  |            |
  |  | filter   |  | filter   |  | filter   |            |
  |  +----+-----+  +----+-----+  +----+-----+            |
  |       |             |             |                   |
  |       +----------+--+-------------+                   |
  |                  |                                    |
  |                  v                                    |
  |       agentService.Run(input)                         |
  |                  |                                    |
  |                  v                                    |
  |           NormalizeResponse()                         |
  |                                                      |
  |  +----------+                                         |
  |  |   CLI    |  Interactive REPL                       |
  |  |  stdin   |  > input -> Run() -> print              |
  |  +----------+                                         |
  |                                                      |
  |  +----------+                                         |
  |  | Scheduler|  Background goroutine                   |
  |  | 10s tick |  jobRepo.FindEnabled()                  |
  |  | 30s max  |  if due -> Run(job.Input)               |
  |  | 3 fails  |  auto-disable after 3 failures         |
  |  | = disable|                                         |
  |  +----------+                                         |
  |                                                      |
  |  +----------+                                         |
  |  | Memory   |  Background goroutine (6h)              |
  |  | Condenser|  CondenseOldMemories()                  |
  |  | facts>7d |  llmFast resume batch 30                |
  |  | atomico  |  delete+insert sintese                  |
  |  +----------+                                         |
  +------------------------------------------------------+
```

---

## Banco de Dados (SQLite)

```
  +-------------------------------------------------------------+
  |                    SQLite (modernc, sem CGO)                  |
  |                                                              |
  |  AUTH & SESSAO                                               |
  |  +------------------------------------------------+         |
  |  | sessions                                       |         |
  |  |   id TEXT PK, expires_at, created_at           |         |
  |  +------------------------------------------------+         |
  |                                                              |
  |  CONVERSAS                                                   |
  |  +------------------------------------------------+         |
  |  | conversations                                  |         |
  |  |   id INTEGER PK, title, source, channel,       |         |
  |  |   created_at, updated_at                       |         |
  |  +------------------+-----------------------------+         |
  |                     | 1:N                                    |
  |  +------------------v-----------------------------+         |
  |  | messages                                       |         |
  |  |   id INTEGER PK, conversation_id FK,           |         |
  |  |   role, content, sort_order, created_at        |         |
  |  +------------------+-----------------------------+         |
  |                     |                                        |
  |  +------------------v-----------------------------+         |
  |  | messages_fts (FTS5 virtual)                    |         |
  |  |   conversation_id UNINDEXED, content           |         |
  |  +------------------------------------------------+         |
  |  +------------------------------------------------+         |
  |  | message_embeddings                             |         |
  |  |   message_id PK, conversation_id,              |         |
  |  |   embedding BLOB (float32[])                   |         |
  |  +------------------------------------------------+         |
  |                                                              |
  |  AGENT                                                       |
  |  +------------------------------------------------+         |
  |  | agent_memory                                   |         |
  |  |   id TEXT PK, content TEXT, category TEXT,      |         |
  |  |   embedding BLOB, created_at                   |         |
  |  +------------------------------------------------+         |
  |  +------------------------------------------------+         |
  |  | agent_feedback                                 |         |
  |  |   id TEXT PK, tool_name, task_type,            |         |
  |  |   success, duration_ms, cost, error,           |         |
  |  |   created_at                                   |         |
  |  +------------------------------------------------+         |
  |  +------------------------------------------------+         |
  |  | agent_executions                               |         |
  |  |   id TEXT PK, goal, status,                    |         |
  |  |   steps JSON, timeline JSON,                   |         |
  |  |   total_ms, step_count,                        |         |
  |  |   tools_used TEXT, failure_reason TEXT,         |         |
  |  |   created_at                                   |         |
  |  +------------------------------------------------+         |
  |  +------------------------------------------------+         |
  |  | agent_config (key-value)                       |         |
  |  |   key TEXT PK, value TEXT                      |         |
  |  |   Keys: soul, identity, user_profile,          |         |
  |  |         environment_notes, agent_limits         |         |
  |  +------------------------------------------------+         |
  |  +------------------------------------------------+         |
  |  | agent_audit                                    |         |
  |  |   id TEXT PK, tool, input, output,             |         |
  |  |   safety, approved, created_at                 |         |
  |  +------------------------------------------------+         |
  |                                                              |
  |  SCHEDULER                                                   |
  |  +------------------------------------------------+         |
  |  | scheduled_jobs                                 |         |
  |  |   id TEXT PK, name, task_type, input,          |         |
  |  |   interval_seconds, enabled, last_run,         |         |
  |  |   last_status, fail_count, created_at          |         |
  |  +------------------------------------------------+         |
  +--------------------------------------------------------------+
```

---

## Estrutura de Arquivos

```
ok/
+-- cmd/
|   +-- ok/main.go .................. Entrypoint HTTP
|   +-- cli/main.go ................. Entrypoint CLI
|
+-- domain/
|   +-- agent.go .................... Tool, Planner, Executor, Decision,
|   |                                 ExecutionState, AgentLimits, etc.
|   +-- intent.go ................... ToolSafety, SafeTool
|   +-- feedback.go ................. Feedback
|   +-- conversation.go ............. Conversation, Message, Embedding
|   +-- session.go .................. Session
|   +-- scheduler.go ................ Job
|
+-- application/
|   +-- agent_service.go ............ System prompt, configs, build engine
|   +-- chat_service.go ............. Chat com streaming de eventos
|   +-- conversation_service.go ..... CRUD de conversas
|   +-- embedding_service.go ........ Busca semantica
|   +-- session_service.go .......... Autenticacao
|   +-- import_service.go ........... Importacao ChatGPT
|   +-- scheduler_service.go ........ CRUD de jobs
|   +-- engine/
|       +-- agent_engine.go ......... Loop OBSERVE/PLAN/ACT/REFLECT
|       +-- emitter.go .............. Interface Emitter
|       +-- buffer_emitter.go ....... Emitter sincrono
|       +-- callback_emitter.go ..... Emitter SSE/streaming
|
+-- infrastructure/
|   +-- bootstrap/
|   |   +-- bootstrap.go ............ Inicializacao compartilhada
|   +-- agent/
|   |   +-- safety.go ............... SafetyGate (safe/restricted/dangerous)
|   |   +-- confirmation.go ......... Confirmacao de tools perigosas
|   |   +-- audit.go ................ Log de auditoria
|   |   +-- rate_limiter.go ......... Rate limiting por tool
|   |   +-- planner.go .............. Valida Decision -> Plan
|   |   +-- executor.go ............. Executa Plan (+ safety + audit)
|   |   +-- execution_state.go ...... State machine + budget
|   |   +-- execution_repository.go . ExecutionRepository + Metrics
|   |   +-- memory.go ............... SQLiteMemory (+ category + chunks)
|   |   +-- vectorstore.go .......... VectorStore (chromem-go)
|   |   +-- watcher.go .............. FileWatcher (re-indexa no save)
|   |   +-- feedback.go ............. FeedbackRepository
|   |   +-- config_repository.go .... ConfigRepository (key-value)
|   |   +-- prompts.go .............. BuildPlanningPrompt, Reflection
|   |   +-- truncate.go ............. TruncateUTF8, TruncateWithEllipsis
|   |   +-- tools/
|   |       +-- echo.go, math.go, json_parse.go, base64.go
|   |       +-- timestamp.go, text_extract.go
|   |       +-- file_read.go, file_write.go, file_edit.go
|   |       +-- http.go, shell.go, repl.go, browser.go
|   |       +-- search.go, folder_index.go
|   |       +-- schedule.go, learn_rule.go, delegate.go
|   |       +-- web_search.go
|   |       +-- (19 tools, cada uma com Safety())
|   +-- security/
|   |   +-- scrubber.go ............ SecretScrubber (regex redact)
|   +-- llm/
|   |   +-- client.go ............... Decide, CreatePlan, Reflect, Stream
|   +-- embedding/
|   |   +-- client.go ............... Embed (OpenAI/Ollama)
|   +-- database/
|   |   +-- database.go ............. SQLite connection
|   |   +-- migrations.go ........... Schema + seeds (11 tabelas)
|   +-- repository/
|   |   +-- conversation_repository.go
|   |   +-- message_repository.go
|   |   +-- session_repository.go
|   +-- scheduler/
|       +-- repository.go ........... JobRepository
|       +-- scheduler.go ............ Background loop (10s tick)
|
+-- interfaces/
|   +-- http/
|       +-- server.go ............... Fiber routes + middleware
|       +-- middleware/
|       |   +-- auth.go ............. Session validation
|       |   +-- logger.go ........... Request logging
|       |   +-- recovery.go ......... Panic recovery
|       +-- handler/
|           +-- agent_handler.go .... Run, Stream, Status, Metrics, Config
|           +-- auth_handler.go ..... Login, Logout, Pages
|           +-- chat_handler.go ..... Conversations, Messages, SSE
|           +-- health_handler.go ... LLM + Embed health check
|           +-- import_handler.go ... ChatGPT ZIP import
|           +-- scheduler_handler.go  Job CRUD
|           +-- ws_handler.go ....... WebSocket hub + hydration
|           +-- templates.go ........ Template embedding
|
+-- adapters/
|   +-- whatsapp.go ................. WhatsApp (whatsmeow)
|   +-- telegram.go ................. Telegram (bot-api)
|   +-- discord.go .................. Discord (discordgo)
|   +-- cli.go ...................... Interactive REPL
|   +-- normalize.go ................ Response formatting
|
+-- web/
|   +-- embed.go .................... Go embed directives
|   +-- templates/
|   |   +-- login.html, chat.html, agent.html, profile.html
|   +-- static/
|       +-- css/ (theme.css, chat.css, agent.css, profile.css)
|       +-- js/ (login.js, chat.js, agent.js, profile.js)
|
+-- internal/
|   +-- config/config.go ............ Viper config loading
|   +-- logger/logger.go ............ Zap structured logging
|
+-- tests/integration/ .............. 169 testes de integracao
|   +-- setup_test.go ............... Infra de teste
|   +-- safety_test.go .............. SafetyGate + shell + rate limiter
|   +-- agent_test.go ............... Tools, planner, executor, memory, state
|   +-- agent_endpoints_test.go ..... Agent HTTP endpoints
|   +-- auth_test.go, conversation_test.go, health_test.go, etc.
|
+-- CLAUDE.md ....................... Regras do projeto
+-- ARCHITECTURE.md ................. Este arquivo
```
