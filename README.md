# OK

Framework de assistente pessoal de IA ultra-leve em Go. Conecta LLMs a plataformas de mensagem via agentes autonomos com ferramentas.

## Stack

- **Go** (CGO_ENABLED=0, binario unico)
- **SQLite** (modernc, sem CGO)
- **Fiber** (HTTP) + **GORM** (ORM)
- **19 tools** com 3 niveis de seguranca (safe/restricted/dangerous)
- **SecretScrubber** — redact automatico de chaves/tokens antes de enviar a LLMs
- **PTY streaming** — output de terminal em tempo real via WebSocket + xterm.js
- **Adapters**: WhatsApp, Telegram, Discord, CLI

## Rodar

```bash
# Servidor web
go run cmd/ok/main.go

# CLI interativo
go run cmd/cli/main.go
```

Configuracao via `data/.env` ou variaveis de ambiente. Minimo necessario:

```
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4.1-mini
AUTH_PASSWORD=sua-senha
```

## Testes

```bash
go test ./tests/integration/... -count=1
```

## Licenca

MIT
