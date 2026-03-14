<p align="center">
  <img src="assets/logo.svg" width="120" alt="OK logo" />
</p>

<h1 align="center">OK</h1>

<p align="center">
  <b>Lightweight personal AI assistant — single binary, zero config files to edit</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25.7-00ADD8?logo=go&logoColor=white" alt="Go 1.25.7" />
  <img src="https://img.shields.io/badge/License-MIT-blue" alt="MIT License" />
  <img src="https://img.shields.io/badge/Platform-linux%2Famd64%20%7C%20linux%2Farm64-lightgrey" alt="Platforms" />
  <img src="https://img.shields.io/badge/CGO-disabled-green" alt="Zero CGO" />
</p>

---

OK connects LLMs to messaging apps. One binary, no CLI to learn — everything is configured through the embedded web UI.

- **14 LLM vendors** — OpenAI, Anthropic, Gemini, DeepSeek, Groq, Ollama, OpenRouter, and more
- **4 chat channels** — Telegram, Discord, WhatsApp, Slack
- **Web UI** — responsive multi-column config editor with i18n (EN/PT-BR/ES), real-time logs, test chat
- **RAG** — semantic long-term memory via vector embeddings, flat-file storage
- **MCP** — Model Context Protocol support (stdio + HTTP/SSE)
- **Skills** — extensible skill system with built-in defaults
- **Agent loop** — ReAct planner + parallel tool execution + memory manager

## Quick Start

```bash
git clone https://github.com/renesul/ok.git && cd ok
make build && make install

ok              # starts gateway + web UI on http://localhost:18800
ok -version     # show version info
ok -debug       # verbose logging
```

On first run, OK creates `~/.ok/` with a default config and workspace. Open the web UI to add your API keys and enable channels.

## Configuration

Config file: `~/.ok/config.json` — edit via web UI or directly.

All fields overridable with `OK_` prefixed env vars:

```bash
OK_AGENTS_DEFAULTS_MODEL=claude-sonnet-4.6 ok
OK_HOME=/srv/ok ok
```

### Minimal Config

```json
{
  "model_list": [
    { "model_name": "gpt-5.2", "model": "openai/gpt-5.2", "api_key": "sk-..." }
  ],
  "agents": { "defaults": { "model": "gpt-5.2" } },
  "channels": {
    "telegram": { "enabled": true, "token": "BOT_TOKEN", "allow_from": ["USER_ID"] }
  }
}
```

### Supported Vendors

| Vendor | Prefix | Protocol |
|--------|--------|----------|
| OpenAI | `openai/` | OpenAI |
| Anthropic | `anthropic/` | Anthropic |
| Google Gemini | `gemini/` | OpenAI |
| DeepSeek | `deepseek/` | OpenAI |
| Groq | `groq/` | OpenAI |
| Ollama | `ollama/` | OpenAI |
| OpenRouter | `openrouter/` | OpenAI |
| NVIDIA | `nvidia/` | OpenAI |
| Cerebras | `cerebras/` | OpenAI |
| Qwen | `qwen/` | OpenAI |
| Zhipu AI | `zhipu/` | OpenAI |
| LiteLLM | `litellm/` | OpenAI |
| vLLM | `vllm/` | OpenAI |

Multiple entries with the same `model_name` are automatically load-balanced (round-robin).

### Workspace

```
~/.ok/workspace/
├── sessions/        # Conversation history
├── memory/          # Long-term memory
├── skills/          # Skill packages
├── IDENTITY.md      # Agent identity
├── SOUL.md          # Agent personality
└── USER.md          # User preferences
```

## Web UI

The web UI starts automatically with the gateway on port `18800`. From there you manage models, channels, agents, tools, MCP servers, RAG, and more.

Forms use a responsive two-column grid layout on desktop, collapsing to single column on smaller screens.

```json
{ "web_ui": { "enabled": true, "host": "127.0.0.1", "port": 18800 } }
```

## Docker

```bash
docker build -t ok .
docker run -d --name ok -v ~/.ok:/home/ok/.ok -p 18800:18800 ok
```

## Architecture

Layered architecture under `app/`, strict dependency direction (top imports bottom):

```
main.go              — Entry point: flag parsing + gateway startup
internal/startup/    — Gateway lifecycle, onboarding, graceful shutdown
app/orchestrator/    — AgentLoop, AgentInstance, Registry
app/planning/        — ReAct loop: LLM → tool calls → observe → repeat
app/execution/       — Tool registry (~20 tools)
app/memory/          — JSONL sessions, RAG (vector embeddings)
app/context/         — System prompt assembly from persona files + RAG
app/routing/         — Route resolver, model router
app/input/           — Channel adapters + message bus
providers/           — LLM backends (Anthropic native + OpenAI-compatible)
internal/            — Config, logger, auth, skills, webui, MCP
```

## Credits

Fork of [PicoClaw](https://github.com/pico-claw/picoclaw), based on [OpenClaw](https://github.com/claw-project/openclaw).

## License

[MIT](LICENSE)
