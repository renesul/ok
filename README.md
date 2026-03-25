# OK

Assistente pessoal de IA autonomo. Um binario Go, zero dependencias externas.

Conecta LLMs a qualquer plataforma de mensagem. Executa tarefas no seu computador com 19 ferramentas — do terminal ao browser — com 3 camadas de seguranca.

---

## Como funciona

Voce manda uma mensagem. O agente decide sozinho o caminho:

1. **Resposta direta** — conversa normal, sem ferramentas
2. **Execucao unica** — uma ferramenta, um resultado
3. **Plano autonomo** — multiplas ferramentas em sequencia (OBSERVE → PLAN → ACT → REFLECT)

Tudo em tempo real via WebSocket. O workspace `/agent` e o monitor central — mostra o processamento de qualquer canal (chat web, WhatsApp, Telegram, Discord).

---

## Stack

| Componente | Tecnologia |
|-----------|------------|
| Linguagem | Go (CGO_ENABLED=0, binario unico) |
| Banco | SQLite WAL mode (modernc, pure Go) |
| HTTP | Fiber |
| DB Layer | database/sql puro (sem ORM) |
| Config | Viper |
| Logging | Zap (estruturado) |
| Vector | chromem-go (busca semantica) |
| Frontend | HTML + CSS + JS vanilla (zero frameworks) |

---

## Ferramentas (19)

| Nivel | Ferramentas |
|-------|------------|
| **safe** | echo, math, timestamp, json_parse, base64, text_extract, file_read, search, schedule, learn_rule |
| **restricted** | http, file_write, folder_index, browser, delegate, web_search |
| **dangerous** | shell, file_edit, repl |

Cada ferramenta perigosa passa por: **Safety Gate** → **Rate Limiter** → **Confirmation HIL** → **Audit Log**

---

## Seguranca

- **SecretScrubber** — remove chaves AWS, OpenAI, JWT, RSA, Bearer, GitHub, Slack antes de enviar ao LLM
- **Shell 3 Tiers** — Tier 1 bloqueio total (rm -rf /, dd, mkfs, fork bombs), Tier 2 requer confirmacao (sudo, chmod, kill), Tier 3 execucao normal
- **Path Traversal** — sandbox enforced em file_read/file_write, system paths bloqueados em file_edit
- **SSRF Protection** — http tool bloqueia localhost, IPs privados (10.x, 192.168.x, 172.16.x)
- **Rate Limiting** — shell 5/min, http 20/min, file_write 10/min
- **Panic Recovery** — defer recover() em todos os adapters
- **Timeouts** — 3min adapters, 5min WebSocket
- **Zip Bomb** — import limitado a 100MB

---

## Canais

| Canal | Status |
|-------|--------|
| Web (chat + workspace) | Integrado |
| WhatsApp | Integrado (whatsmeow) |
| Telegram | Integrado (telegram-bot-api) |
| Discord | Integrado (discordgo) |
| CLI | Integrado |

Todos os canais alimentam o workspace `/agent` em tempo real via global event broadcast.

---

## Rodar

```bash
# Servidor web
go run cmd/ok/main.go

# CLI interativo
go run cmd/cli/main.go
```

Configuracao via `data/.env`:

```env
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4.1-mini
AUTH_PASSWORD=sua-senha
```

Opcional:
```env
LLM_FAST_BASE_URL=https://api.openai.com/v1
LLM_FAST_API_KEY=sk-...
LLM_FAST_MODEL=gpt-4.1-nano
EMBED_PROVIDER=openai
EMBED_BASE_URL=https://api.openai.com/v1
EMBED_API_KEY=sk-...
EMBED_MODEL=text-embedding-3-small
WHATSAPP_OWNER_NUMBER=5511999999999
TELEGRAM_BOT_TOKEN=...
TELEGRAM_OWNER_ID=123456789
DISCORD_BOT_TOKEN=...
DISCORD_OWNER_ID=...
```

---

## Testes

346 testes em 15 pacotes. 100 cenarios E2E stress com LLM real.

```bash
# Testes unitarios (nao precisam de LLM)
go test ./domain/... ./application/... ./adapters/... ./infrastructure/... ./interfaces/... -count=1

# Testes de integracao (precisam de SQLite)
go test ./tests/integration/... -count=1

# Stress E2E completo (precisa de LLM configurado)
go test ./tests/integration/... -run TestStressE2E -timeout=25m -count=1
```

### Baterias E2E

| Bateria | Cenarios | Requer LLM |
|---------|----------|------------|
| FullBattery | 15 | Sim |
| ToolChaining | 8 | Sim |
| Security | 10 | Sim |
| Resilience | 10 | Sim |
| DirectMode | 8 | Sim |
| Memory | 8 | Sim |
| FTS5 | 5 | Nao |
| ChatFlow | 6 | Sim |
| Concurrent | 2 | Sim |
| Offline | 10 | Nao |
| Delegate | 1 | Sim |
| SemanticSearch | 1 | Embedding |
| ConcurrentImports | 1 | Nao |
| SchedulerCRUD | 8 | Nao |
| Confirmation | 4 | Sim |

---

## API

### Autenticacao
```
POST /api/auth/login        → { password }
POST /api/auth/logout
```

### Conversas
```
GET    /api/conversations
GET    /api/conversations/search?q=texto
POST   /api/conversations          → { title }
GET    /api/conversations/:id/messages
POST   /api/conversations/:id/messages → { content } (SSE streaming)
DELETE /api/conversations/:id
```

### Agente
```
POST /api/agent/run              → { input } (sincrono)
POST /api/agent/stream           → { input } (SSE streaming)
GET  /api/agent/status
GET  /api/agent/executions
GET  /api/agent/executions/:id
GET  /api/agent/metrics
GET  /api/agent/limits
PUT  /api/agent/limits           → { max_steps, max_attempts, timeout_ms }
GET  /api/agent/config/:key
PUT  /api/agent/config/:key      → { value }
POST /api/agent/confirm/:id      → { approved }
POST /api/agent/cancel
```

### Scheduler
```
GET    /api/scheduler/jobs
POST   /api/scheduler/jobs       → { name, task_type, input, interval_seconds }
PUT    /api/scheduler/jobs/:id   → { enabled, interval_seconds }
DELETE /api/scheduler/jobs/:id
```

### WebSocket
```
GET /ws/agent                    → bidirectional, JSON messages
    → { type: "input", content: "..." }
    → { type: "confirm", id: "...", approved: true }
    → { type: "cancel" }
    ← { type: "hydration", running, phase, terminal_history }
    ← { type: "phase", content: "observe" }
    ← { type: "step", name, tool, status, elapsed_ms }
    ← { type: "stream", tool, content }
    ← { type: "message", content }
    ← { type: "done" }
```

### Import
```
POST /api/import/chatgpt         → multipart/form-data (zip file)
```

---

## Arquitetura

```
cmd/
  ok/          → servidor web
  cli/         → terminal interativo
domain/        → entidades e interfaces
application/   → services e engine do agente
  engine/      → loop OBSERVE/PLAN/ACT/REFLECT
infrastructure/
  agent/       → planner, executor, safety, memory, tools (19)
  database/    → SQLite puro (database/sql)
  llm/         → client OpenAI-compatible
  embedding/   → client OpenAI/Ollama
  scheduler/   → background job runner
  security/    → SecretScrubber
  repository/  → conversation, message, session
  bootstrap/   → wiring de dependencias
interfaces/
  http/        → server Fiber, handlers, middlewares
adapters/      → WhatsApp, Telegram, Discord, CLI
web/           → templates HTML, CSS, JS vanilla
```

---

## Licenca

MIT
