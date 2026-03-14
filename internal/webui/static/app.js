// ── Early declarations (needed before switchTab init) ────
let gatewayRunning = true;
let waQrSource = null;
let logPollTimer = null;

const panelHelp = {
    en: {
        panelModels: '<strong>Models</strong> — Configure which LLMs your agents use.<ul><li>Format: <code>vendor/model-id</code> (e.g. <code>openai/gpt-4o</code>, <code>anthropic/claude-sonnet-4-20250514</code>, <code>ollama/llama3</code>)</li><li>The <strong>Primary</strong> model is the default for all agents — pick your best model here</li><li>Add a <strong>Light</strong> model for fast, simple replies (saves cost)</li><li>Multiple entries with the same name are automatically load-balanced</li><li>Models without an API key appear grayed out — set the key in the Auth panel</li></ul>',
        panelAuth: '<strong>Auth</strong> — API keys and credentials for LLM providers.<ul><li><strong>OpenAI</strong>: Click "Sign In" to authenticate via device code (no browser redirect needed)</li><li><strong>Anthropic</strong>: Paste your API key (starts with <code>sk-ant-</code>)</li><li><strong>Google</strong>: Browser-based OAuth flow</li><li>Keys are stored locally in <code>~/.ok/auth.json</code> — never sent anywhere except the provider</li></ul>',
        panelAgents: '<strong>Agents</strong> — Your AI assistants. Each agent can have different personality, model, and tools.<ul><li><strong>Defaults</strong> apply to all agents — override per agent only when needed</li><li><strong>Model</strong>: which LLM this agent uses (leave empty to use the default)</li><li><strong>Workspace</strong>: the directory where the agent reads/writes files and stores memory</li><li><strong>Summarize threshold</strong>: after this many messages, the conversation is auto-summarized to save context</li><li><strong>Max media size</strong>: largest file (in bytes) the agent can process — 0 means unlimited</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> — Connect a Telegram bot.<ol><li>Open Telegram, search for <strong>@BotFather</strong>, send <code>/newbot</code></li><li>Copy the bot token and paste it here</li><li>Enable the channel and restart the gateway</li></ol><ul><li>Supports groups, DMs, inline commands, photos, documents, and voice messages</li><li>Use <strong>Allow From</strong> to restrict which users can talk to the bot</li></ul>',
        panelCh_discord: '<strong>Discord</strong> — Connect a Discord bot.<ol><li>Go to Discord Developer Portal, create an Application, then a Bot</li><li>Copy the bot token and paste it here</li><li>Invite the bot to your server with the OAuth2 URL generator (scopes: bot, messages)</li></ol><ul><li><strong>Mention Only</strong>: the bot only responds when @mentioned — useful in busy servers</li><li>Supports threads, reactions, file attachments, and slash commands</li></ul>',
        panelCh_slack: '<strong>Slack</strong> — Connect via Socket Mode (no public URL needed).<ol><li>Create a Slack App at api.slack.com with <strong>Socket Mode</strong> enabled</li><li>Generate a <strong>Bot Token</strong> (<code>xoxb-</code>) with chat:write, app_mentions:read scopes</li><li>Generate an <strong>App Token</strong> (<code>xapp-</code>) with connections:write scope</li></ol><ul><li>Supports threads, DMs, channel messages, and file uploads</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> — Connect your WhatsApp account directly (no external bridge needed).<ol><li>Enable the channel and start the gateway</li><li>A QR code will appear below — scan it with WhatsApp &gt; <strong>Linked Devices</strong> &gt; Link a Device</li><li>Wait for the pairing to complete (takes a few seconds)</li></ol><ul><li>The QR refreshes automatically every ~20 seconds if not scanned</li><li>Once paired, the session persists across gateway restarts</li><li>Use <strong>Allowed Contacts</strong> to restrict who can interact with the bot</li></ul>',
        panelSkills: '<strong>Skills</strong> — Reusable prompt modules that give agents specialized abilities (e.g. coding, research, summarization).<ul><li><strong>Workspace</strong> skills (<code>~/.ok/workspace/skills/</code>): project-specific, highest priority</li><li><strong>Global</strong> skills (<code>~/.ok/skills/</code>): available to all agents</li><li><strong>Builtin</strong> skills: ship with OK, always available</li><li>Use the search to find and install community skills from the registry</li></ul>',
        panelMCP: '<strong>MCP Servers</strong> — Connect external tools via the Model Context Protocol. Agents can call tools provided by MCP servers (databases, APIs, file systems, etc.).<ul><li><strong>stdio</strong>: launches a local process (e.g. <code>npx @modelcontextprotocol/server-filesystem</code>)</li><li><strong>http</strong>: connects to a remote server via SSE URL</li><li>After adding a server, use <strong>Test Connection</strong> to verify it works and see discovered tools</li><li>All discovered tools are automatically available to every agent</li></ul>',
        panelToolSettings: '<strong>Tool Settings</strong> — Control which built-in tools agents can use.<ul><li>Toggle individual tools on/off to restrict agent capabilities</li><li><strong>Shell Exec</strong>: set timeout (seconds) and deny patterns to block dangerous commands</li><li><strong>Path Restrictions</strong>: limit which directories agents can read/write</li><li><strong>Cron</strong>: enable scheduled tasks — agents can create recurring actions</li><li>When in doubt, start restrictive and enable tools as needed</li></ul>',
        panelRAG: '<strong>RAG</strong> — Long-term semantic memory for agents. Past conversations are automatically indexed and retrieved when relevant.<ul><li>Requires an <strong>embeddings endpoint</strong> — works with OpenAI, Ollama (<code>nomic-embed-text</code>), or any OpenAI-compatible API</li><li><strong>Base URL</strong>: the <code>/v1/embeddings</code> endpoint (e.g. <code>http://localhost:11434/v1</code> for Ollama)</li><li><strong>Top K</strong>: how many past memories to retrieve per message (start with 3-5)</li><li><strong>Min Similarity</strong>: ignore memories below this relevance score (0.5 is a good default)</li></ul>',
        panelGateway: '<strong>Gateway</strong> — The main OK server process.<ul><li><strong>Host</strong>: <code>127.0.0.1</code> = local only, <code>0.0.0.0</code> = accessible from other devices on the network</li><li><strong>Port</strong>: HTTP port for the web UI and API (default: 3000)</li><li>The gateway must be running for agents to receive and respond to messages</li></ul>',
        panelSession: '<strong>Session</strong> — How conversation history is organized.<ul><li><strong>Per Channel Peer</strong> (recommended): each user on each platform has a separate history — your Telegram and WhatsApp conversations stay independent</li><li><strong>Per Peer</strong>: the same user shares history across platforms — useful if you want continuity between channels</li><li><strong>Global</strong>: everyone shares one conversation — only useful for single-user setups</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> — Periodic health check pings so you know the gateway is alive.<ul><li>Configure a <strong>channel</strong> and <strong>chat ID</strong> to receive the heartbeat message</li><li><strong>Interval</strong>: how often to send (in minutes, minimum 5)</li><li>If you stop receiving heartbeats, the gateway may be down</li></ul>',
        panelDevices: '<strong>Devices</strong> — Hardware monitoring on Linux.<ul><li><strong>Monitor USB</strong>: detects when USB devices are plugged in or removed</li><li>Agents receive notifications about hardware changes and can react accordingly</li><li>Useful for automation workflows triggered by physical devices</li></ul>',
        panelDebug: '<strong>Debug</strong> — Verbose logging for troubleshooting.<ul><li>Shows full LLM request/response payloads</li><li>Logs every tool call with inputs and outputs</li><li>Increases log volume significantly — disable when not needed</li><li>Changes take effect on the next gateway restart</li></ul>',
        panelChat: '<strong>Chat</strong> — Test your agents directly from this UI.<ul><li>Connects via WebSocket to the running gateway</li><li>Messages are sent to the <strong>default agent</strong></li><li>Green dot = connected, red dot = disconnected — check if the gateway is running</li><li>Conversations here use the <code>webui</code> channel for session/binding purposes</li></ul>',
        panelLogs: '<strong>Logs</strong> — Real-time gateway output.<ul><li>Filter by <strong>level</strong> (error, warn, info), <strong>component</strong>, or free-text search</li><li><strong>Debug toggle</strong>: turn verbose logging on/off without restarting</li><li><strong>Data</strong> column shows structured fields — click to expand</li><li>Logs are streamed live and auto-scroll to the latest entry</li></ul>',
        panelRawJson: '<strong>Raw JSON</strong> — Direct access to the configuration file (<code>~/.ok/config.json</code>).<ul><li>Every change made in other panels is saved to this file</li><li>You can edit here directly for advanced configuration</li><li><strong>Format</strong> button pretty-prints the JSON</li><li>Red border = invalid JSON — fix syntax errors before saving</li></ul>',
        panelRouting: '<strong>Model Routing</strong> — Route simple messages to a lighter model.<ul><li>Enable routing and set a <strong>Light Model</strong> for simple queries</li><li>The <strong>Threshold</strong> controls the complexity cutoff (0-1)</li><li>Messages below the threshold use the light model; others use the primary model</li></ul>',
        panelWebSearch: '<strong>Web Search</strong> — Configure which search engines agents can use.<ul><li>Toggle individual search providers on/off</li><li>Each provider has its own API key and settings</li><li><strong>DuckDuckGo</strong> works without an API key</li></ul>',
        panelSummarization: '<strong>Summarization</strong> — Automatic conversation history compression.<ul><li><strong>Message Threshold</strong>: number of messages before summarization triggers</li><li><strong>Token Percent</strong>: percentage of max tokens that triggers summarization</li></ul>',
        panelWebUI: '<strong>Web UI</strong> — Settings for this configuration interface.<ul><li><strong>Host</strong>: network interface to bind to</li><li><strong>Port</strong>: HTTP port for the web UI</li></ul>',
    },
    'pt-BR': {
        panelModels: '<strong>Modelos</strong> — Configure quais LLMs seus agentes usam.<ul><li>Formato: <code>vendor/model-id</code> (ex: <code>openai/gpt-4o</code>, <code>anthropic/claude-sonnet-4-20250514</code>, <code>ollama/llama3</code>)</li><li>O modelo <strong>Primário</strong> é o padrão para todos os agentes — escolha seu melhor modelo aqui</li><li>Adicione um modelo <strong>Light</strong> para respostas rápidas e simples (economia de custo)</li><li>Múltiplas entradas com o mesmo nome fazem balanceamento de carga automático</li><li>Modelos sem chave API aparecem esmaecidos — configure a chave no painel Auth</li></ul>',
        panelAuth: '<strong>Auth</strong> — Chaves de API e credenciais dos provedores de LLM.<ul><li><strong>OpenAI</strong>: Clique em "Sign In" para autenticar via código de dispositivo (sem redirecionamento)</li><li><strong>Anthropic</strong>: Cole sua chave API (começa com <code>sk-ant-</code>)</li><li><strong>Google</strong>: Fluxo OAuth via navegador</li><li>As chaves ficam armazenadas localmente em <code>~/.ok/auth.json</code> — nunca são enviadas a outro lugar além do provedor</li></ul>',
        panelAgents: '<strong>Agentes</strong> — Seus assistentes de IA. Cada agente pode ter personalidade, modelo e ferramentas diferentes.<ul><li><strong>Padrões</strong> se aplicam a todos os agentes — sobrescreva por agente apenas quando necessário</li><li><strong>Modelo</strong>: qual LLM este agente usa (deixe vazio para usar o padrão)</li><li><strong>Workspace</strong>: diretório onde o agente lê/escreve arquivos e armazena memória</li><li><strong>Limite de resumo</strong>: após essa quantidade de mensagens, a conversa é resumida automaticamente para economizar contexto</li><li><strong>Máx. mídia</strong>: maior arquivo (em bytes) que o agente pode processar — 0 = ilimitado</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> — Conecte um bot do Telegram.<ol><li>Abra o Telegram, procure <strong>@BotFather</strong>, envie <code>/newbot</code></li><li>Copie o token do bot e cole aqui</li><li>Ative o canal e reinicie o gateway</li></ol><ul><li>Suporta grupos, DMs, comandos inline, fotos, documentos e mensagens de voz</li><li>Use <strong>Permitir De</strong> para restringir quais usuários podem conversar com o bot</li></ul>',
        panelCh_discord: '<strong>Discord</strong> — Conecte um bot do Discord.<ol><li>Vá ao Discord Developer Portal, crie um Application e depois um Bot</li><li>Copie o token do bot e cole aqui</li><li>Convide o bot para seu servidor com o gerador de URL OAuth2 (scopes: bot, messages)</li></ol><ul><li><strong>Apenas Menções</strong>: o bot só responde quando @mencionado — útil em servidores movimentados</li><li>Suporta threads, reações, anexos e slash commands</li></ul>',
        panelCh_slack: '<strong>Slack</strong> — Conecte via Socket Mode (sem URL pública necessária).<ol><li>Crie um Slack App em api.slack.com com <strong>Socket Mode</strong> ativado</li><li>Gere um <strong>Bot Token</strong> (<code>xoxb-</code>) com scopes chat:write, app_mentions:read</li><li>Gere um <strong>App Token</strong> (<code>xapp-</code>) com scope connections:write</li></ol><ul><li>Suporta threads, DMs, mensagens de canal e upload de arquivos</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> — Conecte sua conta WhatsApp diretamente (sem bridge externo).<ol><li>Ative o canal e inicie o gateway</li><li>Um QR code aparecerá abaixo — escaneie com WhatsApp &gt; <strong>Aparelhos Conectados</strong> &gt; Conectar Aparelho</li><li>Aguarde o pareamento completar (leva alguns segundos)</li></ol><ul><li>O QR atualiza automaticamente a cada ~20 segundos se não for escaneado</li><li>Uma vez pareado, a sessão persiste entre reinicializações do gateway</li><li>Use <strong>Contatos Permitidos</strong> para restringir quem pode interagir com o bot</li></ul>',
        panelSkills: '<strong>Skills</strong> — Módulos de prompt reutilizáveis que dão habilidades especializadas aos agentes (ex: programação, pesquisa, resumos).<ul><li>Skills de <strong>Workspace</strong> (<code>~/.ok/workspace/skills/</code>): específicas do projeto, prioridade máxima</li><li>Skills <strong>Globais</strong> (<code>~/.ok/skills/</code>): disponíveis para todos os agentes</li><li>Skills <strong>Builtin</strong>: vêm com o OK, sempre disponíveis</li><li>Use a busca para encontrar e instalar skills da comunidade no registro</li></ul>',
        panelMCP: '<strong>Servidores MCP</strong> — Conecte ferramentas externas via Model Context Protocol. Agentes podem chamar ferramentas providas por servidores MCP (bancos de dados, APIs, sistemas de arquivos, etc.).<ul><li><strong>stdio</strong>: executa um processo local (ex: <code>npx @modelcontextprotocol/server-filesystem</code>)</li><li><strong>http</strong>: conecta a um servidor remoto via URL SSE</li><li>Após adicionar um servidor, use <strong>Testar Conexão</strong> para verificar e ver as ferramentas descobertas</li><li>Todas as ferramentas descobertas ficam automaticamente disponíveis para todos os agentes</li></ul>',
        panelToolSettings: '<strong>Config. Ferramentas</strong> — Controle quais ferramentas os agentes podem usar.<ul><li>Ative/desative ferramentas individuais para restringir capacidades</li><li><strong>Shell Exec</strong>: defina timeout (segundos) e padrões de bloqueio para comandos perigosos</li><li><strong>Restrições de Caminho</strong>: limite quais diretórios os agentes podem ler/escrever</li><li><strong>Cron</strong>: habilite tarefas agendadas — agentes podem criar ações recorrentes</li><li>Na dúvida, comece restritivo e habilite ferramentas conforme necessário</li></ul>',
        panelRAG: '<strong>RAG</strong> — Memória semântica de longo prazo para agentes. Conversas passadas são indexadas automaticamente e recuperadas quando relevantes.<ul><li>Requer um <strong>endpoint de embeddings</strong> — funciona com OpenAI, Ollama (<code>nomic-embed-text</code>), ou qualquer API compatível com OpenAI</li><li><strong>Base URL</strong>: o endpoint <code>/v1/embeddings</code> (ex: <code>http://localhost:11434/v1</code> para Ollama)</li><li><strong>Top K</strong>: quantas memórias passadas recuperar por mensagem (comece com 3-5)</li><li><strong>Similaridade Mín.</strong>: ignora memórias abaixo deste score de relevância (0.5 é um bom padrão)</li></ul>',
        panelGateway: '<strong>Gateway</strong> — O processo principal do OK.<ul><li><strong>Host</strong>: <code>127.0.0.1</code> = apenas local, <code>0.0.0.0</code> = acessível de outros dispositivos na rede</li><li><strong>Porta</strong>: porta HTTP para a web UI e API (padrão: 3000)</li><li>O gateway precisa estar rodando para os agentes receberem e responderem mensagens</li></ul>',
        panelSession: '<strong>Sessão</strong> — Como o histórico de conversas é organizado.<ul><li><strong>Por Canal e Peer</strong> (recomendado): cada usuário em cada plataforma tem histórico separado — suas conversas do Telegram e WhatsApp ficam independentes</li><li><strong>Por Peer</strong>: o mesmo usuário compartilha histórico entre plataformas — útil para continuidade entre canais</li><li><strong>Global</strong>: todos compartilham uma conversa — útil apenas para uso individual</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> — Pings periódicos de saúde para saber se o gateway está ativo.<ul><li>Configure um <strong>canal</strong> e <strong>chat ID</strong> para receber a mensagem de heartbeat</li><li><strong>Intervalo</strong>: frequência de envio (em minutos, mínimo 5)</li><li>Se parar de receber heartbeats, o gateway pode estar fora do ar</li></ul>',
        panelDevices: '<strong>Dispositivos</strong> — Monitoramento de hardware no Linux.<ul><li><strong>Monitorar USB</strong>: detecta quando dispositivos USB são conectados ou removidos</li><li>Agentes recebem notificações sobre mudanças de hardware e podem reagir</li><li>Útil para automações disparadas por dispositivos físicos</li></ul>',
        panelDebug: '<strong>Debug</strong> — Logs detalhados para diagnóstico.<ul><li>Mostra payloads completos de requisição/resposta ao LLM</li><li>Registra cada chamada de ferramenta com entradas e saídas</li><li>Aumenta significativamente o volume de logs — desative quando não precisar</li><li>Alterações entram em vigor na próxima reinicialização do gateway</li></ul>',
        panelChat: '<strong>Chat</strong> — Teste seus agentes diretamente desta interface.<ul><li>Conecta via WebSocket ao gateway em execução</li><li>Mensagens são enviadas ao <strong>agente padrão</strong></li><li>Ponto verde = conectado, vermelho = desconectado — verifique se o gateway está rodando</li><li>Conversas aqui usam o canal <code>webui</code> para fins de sessão/vínculo</li></ul>',
        panelLogs: '<strong>Logs</strong> — Saída do gateway em tempo real.<ul><li>Filtre por <strong>nível</strong> (error, warn, info), <strong>componente</strong>, ou busca de texto</li><li><strong>Toggle Debug</strong>: ative/desative logs detalhados sem reiniciar</li><li>Coluna <strong>Data</strong> mostra campos estruturados — clique para expandir</li><li>Logs são transmitidos ao vivo com auto-scroll para a última entrada</li></ul>',
        panelRawJson: '<strong>JSON Bruto</strong> — Acesso direto ao arquivo de configuração (<code>~/.ok/config.json</code>).<ul><li>Toda alteração feita em outros painéis é salva neste arquivo</li><li>Você pode editar aqui diretamente para configurações avançadas</li><li>Botão <strong>Formatar</strong> indenta o JSON</li><li>Borda vermelha = JSON inválido — corrija erros de sintaxe antes de salvar</li></ul>',
        panelRouting: '<strong>Roteamento de Modelo</strong> — Direcione mensagens simples para um modelo mais leve.<ul><li>Ative o roteamento e defina um <strong>Modelo Leve</strong> para consultas simples</li><li>O <strong>Limiar</strong> controla o corte de complexidade (0-1)</li><li>Mensagens abaixo do limiar usam o modelo leve; as demais usam o modelo primário</li></ul>',
        panelWebSearch: '<strong>Busca Web</strong> — Configure quais motores de busca os agentes podem usar.<ul><li>Ative/desative provedores de busca individualmente</li><li>Cada provedor tem sua própria chave API e configurações</li><li><strong>DuckDuckGo</strong> funciona sem chave API</li></ul>',
        panelSummarization: '<strong>Sumarização</strong> — Compressão automática do histórico de conversas.<ul><li><strong>Limiar de Mensagens</strong>: número de mensagens antes da sumarização</li><li><strong>Percentual de Tokens</strong>: percentual de tokens máx. que dispara sumarização</li></ul>',
        panelWebUI: '<strong>Web UI</strong> — Configurações desta interface de configuração.<ul><li><strong>Host</strong>: interface de rede para vincular</li><li><strong>Porta</strong>: porta HTTP para a web UI</li></ul>',
    },
    es: {
        panelModels: '<strong>Modelos</strong> — Configure qué LLMs usan sus agentes.<ul><li>Formato: <code>vendor/model-id</code> (ej: <code>openai/gpt-4o</code>, <code>anthropic/claude-sonnet-4-20250514</code>, <code>ollama/llama3</code>)</li><li>El modelo <strong>Primario</strong> es el predeterminado para todos los agentes — elija su mejor modelo aquí</li><li>Agregue un modelo <strong>Light</strong> para respuestas rápidas y simples (ahorra costos)</li><li>Múltiples entradas con el mismo nombre se balancean automáticamente</li><li>Modelos sin clave API aparecen atenuados — configure la clave en el panel Auth</li></ul>',
        panelAuth: '<strong>Auth</strong> — Claves de API y credenciales de proveedores de LLM.<ul><li><strong>OpenAI</strong>: Haga clic en "Sign In" para autenticar via código de dispositivo (sin redirección)</li><li><strong>Anthropic</strong>: Pegue su clave API (comienza con <code>sk-ant-</code>)</li><li><strong>Google</strong>: Flujo OAuth en navegador</li><li>Las claves se almacenan localmente en <code>~/.ok/auth.json</code> — nunca se envían a otro lugar que no sea el proveedor</li></ul>',
        panelAgents: '<strong>Agentes</strong> — Sus asistentes de IA. Cada agente puede tener personalidad, modelo y herramientas diferentes.<ul><li><strong>Predeterminados</strong> se aplican a todos los agentes — sobreescriba por agente solo cuando sea necesario</li><li><strong>Modelo</strong>: qué LLM usa este agente (deje vacío para usar el predeterminado)</li><li><strong>Workspace</strong>: directorio donde el agente lee/escribe archivos y almacena memoria</li><li><strong>Umbral de resumen</strong>: tras esta cantidad de mensajes, la conversación se resume automáticamente para ahorrar contexto</li><li><strong>Máx. media</strong>: archivo más grande (en bytes) que el agente puede procesar — 0 = ilimitado</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> — Conecte un bot de Telegram.<ol><li>Abra Telegram, busque <strong>@BotFather</strong>, envíe <code>/newbot</code></li><li>Copie el token del bot y péguelo aquí</li><li>Active el canal y reinicie el gateway</li></ol><ul><li>Soporta grupos, DMs, comandos inline, fotos, documentos y mensajes de voz</li><li>Use <strong>Permitir De</strong> para restringir qué usuarios pueden hablar con el bot</li></ul>',
        panelCh_discord: '<strong>Discord</strong> — Conecte un bot de Discord.<ol><li>Vaya al Discord Developer Portal, cree una Application y luego un Bot</li><li>Copie el token del bot y péguelo aquí</li><li>Invite al bot a su servidor con el generador de URL OAuth2 (scopes: bot, messages)</li></ol><ul><li><strong>Solo Menciones</strong>: el bot solo responde cuando es @mencionado — útil en servidores activos</li><li>Soporta threads, reacciones, archivos adjuntos y slash commands</li></ul>',
        panelCh_slack: '<strong>Slack</strong> — Conecte via Socket Mode (sin URL pública necesaria).<ol><li>Cree una Slack App en api.slack.com con <strong>Socket Mode</strong> activado</li><li>Genere un <strong>Bot Token</strong> (<code>xoxb-</code>) con scopes chat:write, app_mentions:read</li><li>Genere un <strong>App Token</strong> (<code>xapp-</code>) con scope connections:write</li></ol><ul><li>Soporta threads, DMs, mensajes de canal y subida de archivos</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> — Conecte su cuenta WhatsApp directamente (sin bridge externo).<ol><li>Active el canal e inicie el gateway</li><li>Un código QR aparecerá abajo — escanéelo con WhatsApp &gt; <strong>Dispositivos Vinculados</strong> &gt; Vincular Dispositivo</li><li>Espere a que se complete el emparejamiento (toma unos segundos)</li></ol><ul><li>El QR se actualiza automáticamente cada ~20 segundos si no se escanea</li><li>Una vez emparejado, la sesión persiste entre reinicios del gateway</li><li>Use <strong>Contactos Permitidos</strong> para restringir quién puede interactuar con el bot</li></ul>',
        panelSkills: '<strong>Skills</strong> — Módulos de prompt reutilizables que dan habilidades especializadas a los agentes (ej: programación, investigación, resúmenes).<ul><li>Skills de <strong>Workspace</strong> (<code>~/.ok/workspace/skills/</code>): específicas del proyecto, prioridad máxima</li><li>Skills <strong>Globales</strong> (<code>~/.ok/skills/</code>): disponibles para todos los agentes</li><li>Skills <strong>Builtin</strong>: vienen con OK, siempre disponibles</li><li>Use la búsqueda para encontrar e instalar skills de la comunidad en el registro</li></ul>',
        panelMCP: '<strong>Servidores MCP</strong> — Conecte herramientas externas via Model Context Protocol. Los agentes pueden llamar herramientas provistas por servidores MCP (bases de datos, APIs, sistemas de archivos, etc.).<ul><li><strong>stdio</strong>: ejecuta un proceso local (ej: <code>npx @modelcontextprotocol/server-filesystem</code>)</li><li><strong>http</strong>: conecta a un servidor remoto via URL SSE</li><li>Tras agregar un servidor, use <strong>Probar Conexión</strong> para verificar y ver las herramientas descubiertas</li><li>Todas las herramientas descubiertas quedan automáticamente disponibles para todos los agentes</li></ul>',
        panelToolSettings: '<strong>Config. Herramientas</strong> — Controle qué herramientas pueden usar los agentes.<ul><li>Active/desactive herramientas individuales para restringir capacidades</li><li><strong>Shell Exec</strong>: defina timeout (segundos) y patrones de bloqueo para comandos peligrosos</li><li><strong>Restricciones de Ruta</strong>: limite qué directorios los agentes pueden leer/escribir</li><li><strong>Cron</strong>: habilite tareas programadas — los agentes pueden crear acciones recurrentes</li><li>En caso de duda, comience restrictivo y habilite herramientas según necesite</li></ul>',
        panelRAG: '<strong>RAG</strong> — Memoria semántica a largo plazo para agentes. Conversaciones pasadas se indexan automáticamente y se recuperan cuando son relevantes.<ul><li>Requiere un <strong>endpoint de embeddings</strong> — funciona con OpenAI, Ollama (<code>nomic-embed-text</code>), o cualquier API compatible con OpenAI</li><li><strong>Base URL</strong>: el endpoint <code>/v1/embeddings</code> (ej: <code>http://localhost:11434/v1</code> para Ollama)</li><li><strong>Top K</strong>: cuántas memorias pasadas recuperar por mensaje (comience con 3-5)</li><li><strong>Similitud Mín.</strong>: ignora memorias debajo de este score de relevancia (0.5 es un buen valor predeterminado)</li></ul>',
        panelGateway: '<strong>Gateway</strong> — El proceso principal de OK.<ul><li><strong>Host</strong>: <code>127.0.0.1</code> = solo local, <code>0.0.0.0</code> = accesible desde otros dispositivos en la red</li><li><strong>Puerto</strong>: puerto HTTP para la web UI y API (predeterminado: 3000)</li><li>El gateway debe estar ejecutándose para que los agentes reciban y respondan mensajes</li></ul>',
        panelSession: '<strong>Sesión</strong> — Cómo se organiza el historial de conversaciones.<ul><li><strong>Por Canal y Peer</strong> (recomendado): cada usuario en cada plataforma tiene historial separado — sus conversaciones de Telegram y WhatsApp son independientes</li><li><strong>Por Peer</strong>: el mismo usuario comparte historial entre plataformas — útil para continuidad entre canales</li><li><strong>Global</strong>: todos comparten una conversación — solo útil para uso individual</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> — Pings periódicos de salud para saber si el gateway está activo.<ul><li>Configure un <strong>canal</strong> y <strong>chat ID</strong> para recibir el mensaje de heartbeat</li><li><strong>Intervalo</strong>: frecuencia de envío (en minutos, mínimo 5)</li><li>Si deja de recibir heartbeats, el gateway puede estar caído</li></ul>',
        panelDevices: '<strong>Dispositivos</strong> — Monitoreo de hardware en Linux.<ul><li><strong>Monitorear USB</strong>: detecta cuando dispositivos USB se conectan o desconectan</li><li>Los agentes reciben notificaciones sobre cambios de hardware y pueden reaccionar</li><li>Útil para automatizaciones disparadas por dispositivos físicos</li></ul>',
        panelDebug: '<strong>Debug</strong> — Logs detallados para diagnóstico.<ul><li>Muestra payloads completos de solicitud/respuesta al LLM</li><li>Registra cada llamada de herramienta con entradas y salidas</li><li>Aumenta significativamente el volumen de logs — desactive cuando no lo necesite</li><li>Los cambios aplican en el próximo reinicio del gateway</li></ul>',
        panelChat: '<strong>Chat</strong> — Pruebe sus agentes directamente desde esta interfaz.<ul><li>Conecta via WebSocket al gateway en ejecución</li><li>Los mensajes se envían al <strong>agente predeterminado</strong></li><li>Punto verde = conectado, rojo = desconectado — verifique si el gateway está ejecutándose</li><li>Las conversaciones aquí usan el canal <code>webui</code> para fines de sesión/vínculo</li></ul>',
        panelLogs: '<strong>Logs</strong> — Salida del gateway en tiempo real.<ul><li>Filtre por <strong>nivel</strong> (error, warn, info), <strong>componente</strong>, o búsqueda de texto</li><li><strong>Toggle Debug</strong>: active/desactive logs detallados sin reiniciar</li><li>Columna <strong>Data</strong> muestra campos estructurados — haga clic para expandir</li><li>Los logs se transmiten en vivo con auto-scroll a la última entrada</li></ul>',
        panelRawJson: '<strong>JSON Crudo</strong> — Acceso directo al archivo de configuración (<code>~/.ok/config.json</code>).<ul><li>Todo cambio hecho en otros paneles se guarda en este archivo</li><li>Puede editar aquí directamente para configuraciones avanzadas</li><li>Botón <strong>Formatear</strong> indenta el JSON</li><li>Borde rojo = JSON inválido — corrija errores de sintaxis antes de guardar</li></ul>',
        panelRouting: '<strong>Enrutamiento de Modelo</strong> — Dirija mensajes simples a un modelo más ligero.<ul><li>Active el enrutamiento y defina un <strong>Modelo Ligero</strong> para consultas simples</li><li>El <strong>Umbral</strong> controla el corte de complejidad (0-1)</li><li>Mensajes debajo del umbral usan el modelo ligero; los demás usan el modelo primario</li></ul>',
        panelWebSearch: '<strong>Búsqueda Web</strong> — Configure qué motores de búsqueda pueden usar los agentes.<ul><li>Active/desactive proveedores de búsqueda individualmente</li><li>Cada proveedor tiene su propia clave API y configuraciones</li><li><strong>DuckDuckGo</strong> funciona sin clave API</li></ul>',
        panelSummarization: '<strong>Sumarización</strong> — Compresión automática del historial de conversaciones.<ul><li><strong>Umbral de Mensajes</strong>: número de mensajes antes de la sumarización</li><li><strong>Porcentaje de Tokens</strong>: porcentaje de tokens máx. que dispara sumarización</li></ul>',
        panelWebUI: '<strong>Web UI</strong> — Configuraciones de esta interfaz de configuración.<ul><li><strong>Host</strong>: interfaz de red para vincular</li><li><strong>Puerto</strong>: puerto HTTP para la web UI</li></ul>',
    },
};

function getHelpText(panelId) {
    const langHelp = panelHelp[currentLang] || panelHelp.en;
    return langHelp[panelId] || (panelHelp.en && panelHelp.en[panelId]) || '';
}

function helpTip(panelId) {
    const text = getHelpText(panelId);
    if (!text) return '';
    return `<button class="help-tip" onclick="toggleHelpTip(this, event)" aria-label="Help" data-help="${panelId}">?</button>`;
}

function toggleHelpTip(el, e) {
    e.stopPropagation();
    const existing = el.querySelector('.help-popover');
    if (existing) {
        existing.remove();
        el.classList.remove('active');
        return;
    }
    document.querySelectorAll('.help-popover').forEach(p => { p.remove(); });
    document.querySelectorAll('.help-tip.active').forEach(t => { t.classList.remove('active'); });

    const panelId = el.dataset.help;
    const pop = document.createElement('div');
    pop.className = 'help-popover';
    pop.innerHTML = getHelpText(panelId);
    pop.addEventListener('click', e2 => e2.stopPropagation());
    el.appendChild(pop);
    el.classList.add('active');

    requestAnimationFrame(() => {
        const rect = pop.getBoundingClientRect();
        if (rect.right > window.innerWidth - 16) {
            pop.style.left = 'auto';
            pop.style.right = '0';
        }
    });
}

document.addEventListener('click', () => {
    document.querySelectorAll('.help-popover').forEach(p => p.remove());
    document.querySelectorAll('.help-tip.active').forEach(t => t.classList.remove('active'));
});

function panelHeader(title, panelId) {
    return `<div class="panel-header"><div class="panel-title">${title}</div>${helpTip(panelId)}</div>`;
}

// ── Theme ───────────────────────────────────────────
const themeIcons = { light: '\u2600\uFE0F', dark: '\uD83C\uDF19', system: '\uD83D\uDCBB' };
const themeOrder = ['system', 'light', 'dark'];
let currentThemeSetting = localStorage.getItem('ok-theme') || 'system';

function getSystemTheme() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}
function applyTheme() {
    const resolved = currentThemeSetting === 'system' ? getSystemTheme() : currentThemeSetting;
    document.documentElement.setAttribute('data-theme', resolved);
    const btn = document.getElementById('btnTheme');
    if (btn) btn.textContent = themeIcons[currentThemeSetting];
}
function cycleTheme() {
    const idx = themeOrder.indexOf(currentThemeSetting);
    currentThemeSetting = themeOrder[(idx + 1) % themeOrder.length];
    localStorage.setItem('ok-theme', currentThemeSetting);
    applyTheme();
}
// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (currentThemeSetting === 'system') applyTheme();
});
// Apply immediately to avoid flash
applyTheme();

// ── i18n ────────────────────────────────────────────
let currentLang = localStorage.getItem('ok-lang') || (navigator.language.startsWith('pt') ? 'pt-BR' : navigator.language.startsWith('es') ? 'es' : 'en');

const i18nData = {
    en: {
        // Header & global
        'start': 'Start',
        'stop': 'Stop',
        'save': 'Save',
        'cancel': 'Cancel',
        'edit': 'Edit',
        'delete': 'Delete',
        'enabled': 'Enabled',
        'disabled': 'Disabled',
        'comingSoon': 'Coming soon',
        // Sidebar
        'sidebar.providers': 'Providers',
        'sidebar.models': 'Models',
        'sidebar.auth': 'Auth',
        'sidebar.channels': 'Channels',
        'sidebar.agents': 'Agents',
        'sidebar.agentsList': 'Agents',
        'sidebar.bindings': 'Bindings',
        'sidebar.tools': 'Tools',
        'sidebar.mcp': 'MCP Servers',
        'sidebar.toolSettings': 'Tool Settings',
        'sidebar.rag': 'RAG',
        'sidebar.system': 'System',
        'sidebar.gateway': 'Gateway',
        'sidebar.session': 'Session',
        'sidebar.heartbeat': 'Heartbeat',
        'sidebar.devices': 'Devices',
        'sidebar.debug': 'Debug',
        'sidebar.chat': 'Chat',
        'sidebar.logs': 'Logs',
        'sidebar.rawjson': 'Raw JSON',
        'sidebar.input': 'Input',
        'sidebar.routing': 'Routing',
        'sidebar.planning': 'Planning',
        'sidebar.execution': 'Execution',
        'sidebar.memory': 'Memory',
        'sidebar.orchestrator': 'Orchestrator',
        'sidebar.webSearch': 'Web Search',
        'sidebar.summarization': 'Summarization',
        'sidebar.agentDefaults': 'Agent Defaults',
        'sidebar.webui': 'Web UI',
        'sidebar.providers': 'Providers',
        'sidebar.modelRouting': 'Model Routing',
        // Models
        'models.title': 'Models',
        'models.desc': 'Manage LLM model configurations. Models without an API key are grayed out. Only available models can be set as primary.',
        'models.add': '+ Add Model',
        'models.noModels': 'No models configured.',
        'models.primary': 'Primary',
        'models.noKey': 'No Key',
        'models.setPrimary': 'Set Primary',
        'models.editModel': 'Edit Model',
        'models.addModel': 'Add Model',
        'models.advancedOptions': 'Advanced Options',
        'models.deleteConfirm': 'Delete model "{name}"?',
        'models.requiredFields': 'Model Name and Model ID are required',
        'models.test': 'Test',
        'models.testing': 'Testing...',
        'models.testSuccess': '{model}: {response}',
        'models.testFail': '{model}: {error}',
        // Model fields
        'field.modelName': 'Model Name',
        'field.modelId': 'Model ID',
        'field.modelIdHint': 'Format: protocol/model-id',
        'field.provider': 'Provider',
        'field.providerManual': '(Manual - API Key)',
        'field.apiKey': 'API Key',
        'field.apiBase': 'API Base',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Auth Method',
        'field.connectMode': 'Connect Mode',
        'field.thinkingLevel': 'Thinking Level',
        'field.maxTokensField': 'Max Tokens Field',
        'field.workspace': 'Workspace',
        'field.rpm': 'RPM Limit',
        'field.requestTimeout': 'Request Timeout (s)',
        // Auth
        'auth.title': 'Provider Authentication',
        'auth.desc': 'Login to providers using OAuth or API tokens. Credentials are stored locally in <code style="font-family:\'JetBrains Mono\',monospace;font-size:12px;background:var(--bg-elevated);padding:2px 6px;border-radius:4px;">~/.ok/auth.json</code>.',
        'auth.notLoggedIn': 'Not logged in',
        'auth.active': 'Active',
        'auth.expired': 'Expired',
        'auth.needsRefresh': 'Needs Refresh',
        'auth.authenticating': 'Authenticating...',
        'auth.loginDevice': 'Login (Device Code)',
        'auth.loginToken': 'Login (API Token)',
        'auth.loginOAuth': 'Login (Browser OAuth)',
        'auth.logout': 'Logout',
        'auth.retry': 'Retry',
        'auth.waiting': 'Waiting...',
        'auth.pasteKey': 'Paste your API key here...',
        'auth.step1': 'Step 1: Click the link below',
        'auth.step2': 'Step 2: Enter this code',
        'auth.step3': 'Step 3: Complete auth in browser, this page updates automatically',
        'auth.method': 'Method',
        'auth.email': 'Email',
        'auth.account': 'Account',
        'auth.project': 'Project',
        'auth.expires': 'Expires',
        'auth.custom1': 'Custom Provider 1',
        'auth.custom2': 'Custom Provider 2',
        'auth.configure': 'Configure',
        'auth.customLabel': 'Provider Name',
        'auth.customApiBase': 'API Base URL',
        'auth.customToken': 'API Token',
        'auth.customLabelPlaceholder': 'e.g. Together AI',
        'auth.customApiBasePlaceholder': 'e.g. https://api.together.xyz/v1',
        // Channel
        'ch.configure': 'Configure {name} channel settings.',
        'ch.docLink': 'Configuration Guide',
        'ch.accessControl': 'Access Control',
        'ch.allowFrom': 'Allow From (User IDs)',
        'ch.allowedGroups': 'Allowed Groups',
        'ch.allowedContacts': 'Allowed Contacts',
        'ch.addItem': '+ Add',
        'ch.allowDirect': 'Allow Direct Messages',
        'ch.allowDirectHint': 'Respond to direct (private) messages',
        'ch.allowGroups': 'Allow Group Messages',
        'ch.allowGroupsHint': 'Respond in groups when mentioned',
        'ch.allowSelfHint': 'Respond to messages from the connected number',
        'ch.mentionOnly': 'Mention Only',
        'ch.mentionOnlyHint': 'Only respond when mentioned',
        'ch.groupTrigger': 'Group Trigger Prefixes',
        // Logs
        'logs.title': 'Gateway Logs',
        'logs.desc': 'Real-time output from the gateway process.',
        'logs.clear': 'Clear Display',
        'logs.autoScroll': 'Auto-scroll',
        'logs.noLogs': 'No logs available. Start the gateway to see output here.',
        'logs.noCapture': 'Logs are not available. The gateway was not started from this launcher.',
        'logs.debugOn': 'Debug enabled (applies on next start)',
        'logs.debugOff': 'Debug disabled',
        'logs.allComponents': 'All Components',
        'logs.filterLevel': 'All Levels',
        'logs.filterSearch': 'Filter...',
        // Raw JSON
        'raw.title': 'Raw JSON',
        'raw.desc': 'Directly edit the configuration file.',
        'raw.reload': 'Reload',
        'raw.format': 'Format',
        // Status messages
        'status.configLoaded': 'Config loaded',
        'status.configSaved': 'Config saved',
        'status.loadFailed': 'Load failed',
        'status.saveFailed': 'Save failed',
        'status.formatted': 'Formatted',
        'status.invalidJson': 'Invalid JSON, please fix before saving',
        'status.jsonValid': 'JSON valid',
        'status.jsonInvalid': 'JSON invalid',
        'status.saved': '{name} saved',
        'status.tokenEmpty': 'Token cannot be empty',
        'status.tokenSaved': 'Token saved for {name}',
        'status.loggedOut': 'Logged out from {name}',
        'status.loginFailed': 'Login failed',
        'status.logoutFailed': 'Logout failed',
        'status.openingBrowser': 'Opening browser for authentication...',
        'status.loginStarted': 'Login started...',
        'status.loginSuccess': 'Login successful!',
        // Process
        'process.running': 'Service running',
        'process.notRunning': 'Service not running',
        'process.starting': 'Starting gateway...',
        'process.started': 'Gateway started',
        'process.startFailed': 'Failed to start gateway',
        'process.stopping': 'Stopping gateway...',
        'process.stopped': 'Gateway stopped',
        'process.stopFailed': 'Failed to stop gateway',
        'process.needModel': 'At least one model with API key is required',
        'process.needChannel': 'At least one channel must be enabled',
        'process.checkLogs': 'Check Logs for details',
        'process.needBoth': 'Need model with API key and enabled channel to start',
        // Onboarding
        'onboarding.title': 'Getting Started',
        'onboarding.install': 'Install OK',
        'onboarding.model': 'Add an LLM model',
        'onboarding.channel': 'Connect a channel',
        'onboarding.start': 'Start the gateway',
        // Status indicator
        'status.setup': 'Setup needed',
        'status.ready': 'Ready',
        'status.running': 'Running',
        // Agents
        'agents.title': 'Agents',
        'agents.desc': 'Configure agent defaults and create specialized agents.',
        'agents.defaults': 'Default Settings',
        'agents.defaultsDesc': 'Base configuration inherited by all agents. Individual agents can override these values.',
        'agents.agentList': 'Agents',
        'agents.addAgent': '+ Add Agent',
        'agents.editAgent': 'Edit Agent',
        'agents.addAgentTitle': 'Add Agent',
        'agents.noAgents': 'No custom agents configured. All messages are handled by the default agent with the settings above.',
        'agents.saveDefaults': 'Save Defaults',
        'agents.defaultsSaved': 'Agent defaults saved',
        'agents.deleteConfirm': 'Delete agent "{name}"?',
        'agents.idRequired': 'Agent ID is required',
        'agents.duplicateId': 'Agent with ID "{id}" already exists',
        'agents.restrictWorkspace': 'Restrict to Workspace',
        'agents.allowReadOutside': 'Allow Read Outside',
        'agents.modelName': 'Default Model',
        'agents.modelNameHint': 'Model used by all agents unless overridden',
        'agents.maxTokens': 'Max Tokens',
        'agents.maxToolIter': 'Max Tool Iterations',
        'agents.summarizeThreshold': 'Summarize Threshold',
        'agents.summarizeThresholdHint': 'Messages before summarization triggers',
        'agents.summarizeTokenPct': 'Summarize Token %',
        'agents.maxMediaSize': 'Max Media Size (bytes)',
        'agents.maxMediaSizeHint': '0 = default (20MB)',
        'agents.defaultAgent': 'Default Agent',
        'agents.friendlyName': 'e.g. Research Assistant',
        'agents.inheritsDefaults': 'Inherits from defaults',
        'agents.modelHint': 'Model name (must match a model_name from Models)',
        'agents.modelPrimary': 'Model',
        'agents.modelFallbacks': 'Fallback Models',
        'agents.sectionIdentity': 'Identity',
        'agents.sectionModel': 'Model',
        'agents.sectionAdvanced': 'Advanced',
        'agents.subagents': 'Allowed Subagents',
        'agents.subagentsHint': 'Other agents this agent can invoke',
        'agents.inheritedFrom': 'inherited',
        'agents.overridden': 'custom',
        'agents.agentSaved': 'Agent saved',
        'agents.skillsHint': 'Select which installed skills this agent can use',
        // Bindings
        // MCP
        'mcp.title': 'MCP Servers',
        'mcp.desc': 'Connect to Model Context Protocol servers to extend agent capabilities with external tools.',
        'mcp.addServer': '+ Add Server',
        'mcp.addServerTitle': 'Add MCP Server',
        'mcp.editServer': 'Edit: {name}',
        'mcp.noServers': 'No MCP Servers',
        'mcp.noServersDesc': 'Add an MCP server to enable external tool integrations.',
        'mcp.deleteConfirm': 'Delete MCP server "{name}"?',
        'mcp.nameRequired': 'Server name is required',
        'mcp.testing': 'Testing connection to {name}...',
        'mcp.testResult': '{name}: found {count} tool(s)',
        'mcp.testFailed': 'Test failed: {msg}',
        'mcp.serverName': 'Server Name',
        'mcp.transport': 'Transport',
        'mcp.transportStdio': 'stdio (local process)',
        'mcp.transportHttp': 'HTTP/SSE (remote)',
        'mcp.command': 'Command',
        'mcp.arguments': 'Arguments',
        'mcp.envVars': 'Environment Variables',
        'mcp.serverUrl': 'Server URL',
        'mcp.httpHeaders': 'HTTP Headers',
        'mcp.timeout': 'Timeout (seconds)',
        'mcp.toolPrefix': 'Tool Prefix',
        'mcp.toolPrefixHint': 'Prefixed to tool names to avoid collisions, e.g. "fs" &rarr; "fs_read_file"',
        'mcp.test': 'Test',
        // Tool Settings
        'tools.title': 'Tool Settings',
        'tools.desc': 'Enable or disable individual tools and configure tool-specific settings.',
        'tools.toggles': 'Tool Toggles',
        'tools.webSearch': 'Web Search',
        'tools.webEnabled': 'Web Search Enabled',
        'tools.webProxy': 'Proxy',
        'tools.webFetchLimit': 'Fetch Limit (bytes)',
        'tools.shellExec': 'Shell Exec',
        'tools.execEnabled': 'Exec Enabled',
        'tools.execDenyPatterns': 'Enable Deny Patterns',
        'tools.execTimeout': 'Timeout (seconds)',
        'tools.cron': 'Cron',
        'tools.cronEnabled': 'Cron Enabled',
        'tools.cronTimeout': 'Exec Timeout (minutes)',
        'tools.paths': 'Path Restrictions',
        'tools.allowReadPaths': 'Allowed Read Paths',
        'tools.allowWritePaths': 'Allowed Write Paths',
        'tools.customDenyPatterns': 'Custom Deny Patterns',
        'tools.customAllowPatterns': 'Custom Allow Patterns',
        'tools.mediaCleanup': 'Media Cleanup',
        'tools.mediaMaxAge': 'Max Age (minutes)',
        'tools.mediaInterval': 'Cleanup Interval (minutes)',
        'tools.saved': 'Tool settings saved',
        // RAG
        'rag.title': 'RAG (Semantic Memory)',
        'rag.desc': 'Configure retrieval-augmented generation for long-term semantic memory. Requires an OpenAI-compatible embeddings endpoint.',
        'rag.embeddingsUrl': 'Embeddings API URL',
        'rag.apiKey': 'API Key',
        'rag.embeddingModel': 'Embedding Model',
        'rag.topK': 'Top K Results',
        'rag.minSimilarity': 'Min Similarity',
        'rag.minSimilarityHint': 'Cosine similarity threshold (0-1)',
        'rag.saved': 'RAG settings saved',
        // Gateway
        'gateway.title': 'Gateway',
        'gateway.desc': 'Network settings for the gateway server.',
        'gateway.host': 'Host',
        'gateway.port': 'Port',
        'gateway.proxy': 'Proxy',
        'gateway.proxyHint': 'Global HTTP proxy (http/https/socks5). Used by all channels, tools, and providers.',
        'gateway.saved': 'Gateway settings saved',
        // Session
        'session.title': 'Session',
        'session.desc': 'Configure conversation session behavior and identity linking.',
        'session.dmScope': 'DM Scope',
        'session.perChannelPeer': 'Per Channel Peer (default)',
        'session.perPeer': 'Per Peer',
        'session.perChannel': 'Per Channel',
        'session.global': 'Global',
        'session.saved': 'Session settings saved',
        // Heartbeat
        'heartbeat.title': 'Heartbeat',
        'heartbeat.desc': 'Periodic health check notifications.',
        'heartbeat.interval': 'Interval (minutes)',
        'heartbeat.intervalHint': 'Minimum 5 minutes',
        'heartbeat.saved': 'Heartbeat settings saved',
        // Devices
        'devices.title': 'Devices',
        'devices.desc': 'Hardware device monitoring (Linux only).',
        'devices.monitorUsb': 'Monitor USB',
        'devices.saved': 'Device settings saved',
        // Debug
        'debug.title': 'Debug',
        'debug.desc': 'Toggle debug mode for verbose logging.',
        'debug.debugMode': 'Debug Mode',
        'debug.hint': 'Enables verbose logging. Changes take effect on next gateway restart.',
        'debug.saved': 'Debug setting saved',
        // Skills
        'skills.title': 'Skills',
        'skills.desc': 'Manage installed skills and discover new ones from the registry.',
        'skills.installed': 'Installed Skills',
        'skills.searchInstall': 'Search & Install',
        'skills.search': 'Search',
        'skills.searchPlaceholder': 'Search skills...',
        'skills.loading': 'Loading...',
        'skills.noSkills': 'No Skills Installed',
        'skills.noSkillsDesc': 'Install skills from the registry below or place them in your workspace.',
        'skills.show': 'Show',
        'skills.remove': 'Remove',
        'skills.loadFailed': 'Failed to load skills: {msg}',
        // Routing
        'routing.title': 'Model Routing',
        'routing.desc': 'Configure automatic model routing based on message complexity.',
        'routing.lightModel': 'Light Model',
        'routing.threshold': 'Threshold',
        'routing.thresholdHint': 'Complexity threshold (0-1) for routing to light model',
        'routing.saved': 'Routing settings saved',
        // Providers
        'providers.provider': 'Provider',
        // Web Search
        'webSearch.title': 'Web Search',
        'webSearch.desc': 'Configure web search providers for agent web access.',
        'webSearch.global': 'Global Settings',
        'webSearch.saved': 'Web search settings saved',
        // Summarization
        'summarization.title': 'Summarization',
        'summarization.desc': 'Configure when and how conversations are summarized to save context.',
        'summarization.saved': 'Summarization settings saved',
        // Agent Defaults
        'agentDefaults.temperature': 'Temperature',
        // WebUI
        'webui.title': 'Web UI',
        'webui.desc': 'Configure the embedded web interface settings.',
        'webui.saved': 'Web UI settings saved',
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Offline',
        'chat.placeholder': 'Type a message...',
        'chat.send': 'Send',
        // Additional keys
        'close': 'Close',
        'addItem': '+ Add',
        'status.reloadTriggered': 'Config reload triggered',
        'logs.deleteAllConfirm': 'Delete all log files? This cannot be undone.',
        'logs.allDeleted': 'All logs deleted',
        'logs.deleteAll': 'Delete All Logs',
        'skills.removeConfirm': 'Remove skill "{name}"? This cannot be undone.',
        'skills.removed': 'Skill "{name}" removed',
        'skills.searching': 'Searching...',
        'skills.noResults': 'No results found.',
        'skills.alreadyInstalled': 'Installed',
        'skills.install': 'Install',
        'skills.installing': 'Installing...',
        'skills.blockedMalware': 'Skill blocked: flagged as malware',
        'skills.installedSuspicious': 'Skill installed but flagged as suspicious',
        'skills.installSuccess': 'Skill "{slug}" installed (v{version})',
        'skills.installFailed': 'Install failed: {msg}',
        'skills.showFailed': 'Failed to load skill: {msg}',
        'skills.removeFailed': 'Failed to remove skill: {msg}',
    },
    'pt-BR': {
        // Header & global
        'start': 'Iniciar',
        'stop': 'Parar',
        'save': 'Salvar',
        'cancel': 'Cancelar',
        'edit': 'Editar',
        'delete': 'Excluir',
        'enabled': 'Ativado',
        'disabled': 'Desativado',
        'comingSoon': 'Em breve',
        // Sidebar
        'sidebar.providers': 'Provedores',
        'sidebar.models': 'Modelos',
        'sidebar.auth': 'Autenticação',
        'sidebar.channels': 'Canais',
        'sidebar.agents': 'Agentes',
        'sidebar.agentsList': 'Agentes',
        'sidebar.bindings': 'Vínculos',
        'sidebar.tools': 'Ferramentas',
        'sidebar.mcp': 'Servidores MCP',
        'sidebar.toolSettings': 'Config. Ferramentas',
        'sidebar.rag': 'RAG',
        'sidebar.system': 'Sistema',
        'sidebar.gateway': 'Gateway',
        'sidebar.session': 'Sessão',
        'sidebar.heartbeat': 'Heartbeat',
        'sidebar.devices': 'Dispositivos',
        'sidebar.debug': 'Debug',
        'sidebar.chat': 'Chat',
        'sidebar.logs': 'Logs',
        'sidebar.rawjson': 'JSON Bruto',
        'sidebar.input': 'Entrada',
        'sidebar.routing': 'Roteamento',
        'sidebar.planning': 'Planejamento',
        'sidebar.execution': 'Execução',
        'sidebar.memory': 'Memória',
        'sidebar.orchestrator': 'Orquestrador',
        'sidebar.webSearch': 'Busca Web',
        'sidebar.summarization': 'Sumarização',
        'sidebar.agentDefaults': 'Padrões do Agente',
        'sidebar.webui': 'Web UI',
        'sidebar.modelRouting': 'Roteamento de Modelo',
        // Models
        'models.title': 'Modelos',
        'models.desc': 'Gerencie configurações de modelos LLM. Modelos sem chave API ficam esmaecidos. Apenas modelos disponíveis podem ser definidos como primário.',
        'models.add': '+ Adicionar Modelo',
        'models.noModels': 'Nenhum modelo configurado.',
        'models.primary': 'Primário',
        'models.noKey': 'Sem Chave',
        'models.setPrimary': 'Definir Primário',
        'models.editModel': 'Editar Modelo',
        'models.addModel': 'Adicionar Modelo',
        'models.advancedOptions': 'Opções Avançadas',
        'models.deleteConfirm': 'Excluir modelo "{name}"?',
        'models.requiredFields': 'Nome do Modelo e ID do Modelo são obrigatórios',
        'models.test': 'Testar',
        'models.testing': 'Testando...',
        'models.testSuccess': '{model}: {response}',
        'models.testFail': '{model}: {error}',
        // Model fields
        'field.modelName': 'Nome do Modelo',
        'field.modelId': 'ID do Modelo',
        'field.modelIdHint': 'Formato: protocolo/id-modelo',
        'field.provider': 'Provedor',
        'field.providerManual': '(Manual - Chave API)',
        'field.apiKey': 'Chave API',
        'field.apiBase': 'URL Base da API',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Método de Autenticação',
        'field.connectMode': 'Modo de Conexão',
        'field.thinkingLevel': 'Nível de Pensamento',
        'field.maxTokensField': 'Campo Max Tokens',
        'field.workspace': 'Workspace',
        'field.rpm': 'Limite RPM',
        'field.requestTimeout': 'Timeout da Requisição (s)',
        // Auth
        'auth.title': 'Autenticação de Provedores',
        'auth.desc': 'Faça login nos provedores usando OAuth ou tokens de API. As credenciais são armazenadas localmente em <code style="font-family:\'JetBrains Mono\',monospace;font-size:12px;background:var(--bg-elevated);padding:2px 6px;border-radius:4px;">~/.ok/auth.json</code>.',
        'auth.notLoggedIn': 'Não autenticado',
        'auth.active': 'Ativo',
        'auth.expired': 'Expirado',
        'auth.needsRefresh': 'Precisa Atualizar',
        'auth.authenticating': 'Autenticando...',
        'auth.loginDevice': 'Login (Código de Dispositivo)',
        'auth.loginToken': 'Login (Token de API)',
        'auth.loginOAuth': 'Login (OAuth no Navegador)',
        'auth.logout': 'Sair',
        'auth.retry': 'Tentar Novamente',
        'auth.waiting': 'Aguardando...',
        'auth.pasteKey': 'Cole sua chave API aqui...',
        'auth.step1': 'Passo 1: Clique no link abaixo',
        'auth.step2': 'Passo 2: Insira este código',
        'auth.step3': 'Passo 3: Complete a autenticação no navegador, esta página atualiza automaticamente',
        'auth.method': 'Método',
        'auth.email': 'E-mail',
        'auth.account': 'Conta',
        'auth.project': 'Projeto',
        'auth.expires': 'Expira',
        'auth.custom1': 'Provedor Customizado 1',
        'auth.custom2': 'Provedor Customizado 2',
        'auth.configure': 'Configurar',
        'auth.customLabel': 'Nome do Provedor',
        'auth.customApiBase': 'URL Base da API',
        'auth.customToken': 'Token da API',
        'auth.customLabelPlaceholder': 'ex: Together AI',
        'auth.customApiBasePlaceholder': 'ex: https://api.together.xyz/v1',
        // Channel
        'ch.configure': 'Configurar canal {name}.',
        'ch.docLink': 'Guia de Configuração',
        'ch.accessControl': 'Controle de Acesso',
        'ch.allowFrom': 'Permitir de (IDs de Usuário)',
        'ch.allowedGroups': 'Grupos Permitidos',
        'ch.allowedContacts': 'Contatos Permitidos',
        'ch.addItem': '+ Adicionar',
        'ch.allowDirect': 'Permitir Mensagens Diretas',
        'ch.allowDirectHint': 'Responder a mensagens diretas (privadas)',
        'ch.allowGroups': 'Permitir Mensagens em Grupo',
        'ch.allowGroupsHint': 'Responder em grupos quando mencionado',
        'ch.allowSelfHint': 'Responder a mensagens do número conectado',
        'ch.mentionOnly': 'Apenas Menções',
        'ch.mentionOnlyHint': 'Responder apenas quando mencionado',
        'ch.groupTrigger': 'Prefixos de Gatilho em Grupo',
        // Logs
        'logs.title': 'Logs do Gateway',
        'logs.desc': 'Saída em tempo real do processo do gateway.',
        'logs.clear': 'Limpar Tela',
        'logs.autoScroll': 'Auto-rolagem',
        'logs.noLogs': 'Nenhum log disponível. Inicie o gateway para ver a saída aqui.',
        'logs.noCapture': 'Logs indisponíveis. O gateway não foi iniciado por este launcher.',
        'logs.debugOn': 'Debug ativado (aplica na próxima inicialização)',
        'logs.debugOff': 'Debug desativado',
        'logs.allComponents': 'Todos os Componentes',
        'logs.filterLevel': 'Todos os Níveis',
        'logs.filterSearch': 'Filtrar...',
        // Raw JSON
        'raw.title': 'JSON Bruto',
        'raw.desc': 'Edite diretamente o arquivo de configuração.',
        'raw.reload': 'Recarregar',
        'raw.format': 'Formatar',
        // Status messages
        'status.configLoaded': 'Configuração carregada',
        'status.configSaved': 'Configuração salva',
        'status.loadFailed': 'Falha ao carregar',
        'status.saveFailed': 'Falha ao salvar',
        'status.formatted': 'Formatado',
        'status.invalidJson': 'JSON inválido, corrija antes de salvar',
        'status.jsonValid': 'JSON válido',
        'status.jsonInvalid': 'JSON inválido',
        'status.saved': '{name} salvo',
        'status.tokenEmpty': 'O token não pode estar vazio',
        'status.tokenSaved': 'Token salvo para {name}',
        'status.loggedOut': 'Desconectado de {name}',
        'status.loginFailed': 'Falha no login',
        'status.logoutFailed': 'Falha ao sair',
        'status.openingBrowser': 'Abrindo navegador para autenticação...',
        'status.loginStarted': 'Login iniciado...',
        'status.loginSuccess': 'Login realizado com sucesso!',
        // Process
        'process.running': 'Serviço em execução',
        'process.notRunning': 'Serviço parado',
        'process.starting': 'Iniciando gateway...',
        'process.started': 'Gateway iniciado',
        'process.startFailed': 'Falha ao iniciar gateway',
        'process.stopping': 'Parando gateway...',
        'process.stopped': 'Gateway parado',
        'process.stopFailed': 'Falha ao parar gateway',
        'process.needModel': 'Pelo menos um modelo com chave API é necessário',
        'process.needChannel': 'Pelo menos um canal deve estar ativado',
        'process.checkLogs': 'Verifique os Logs para detalhes',
        'process.needBoth': 'Necessário modelo com chave API e canal ativado para iniciar',
        // Onboarding
        'onboarding.title': 'Primeiros Passos',
        'onboarding.install': 'Instalar OK',
        'onboarding.model': 'Adicionar um modelo LLM',
        'onboarding.channel': 'Conectar um canal',
        'onboarding.start': 'Iniciar o gateway',
        // Status indicator
        'status.setup': 'Configuração necessária',
        'status.ready': 'Pronto',
        'status.running': 'Em execução',
        // Agents
        'agents.title': 'Agentes',
        'agents.desc': 'Configure padrões de agentes e crie agentes especializados.',
        'agents.defaults': 'Configurações Padrão',
        'agents.defaultsDesc': 'Configuração base herdada por todos os agentes. Agentes individuais podem sobrescrever estes valores.',
        'agents.agentList': 'Agentes',
        'agents.addAgent': '+ Adicionar Agente',
        'agents.editAgent': 'Editar Agente',
        'agents.addAgentTitle': 'Adicionar Agente',
        'agents.noAgents': 'Nenhum agente personalizado. Todas as mensagens são tratadas pelo agente padrão com as configurações acima.',
        'agents.saveDefaults': 'Salvar Padrões',
        'agents.defaultsSaved': 'Padrões de agente salvos',
        'agents.deleteConfirm': 'Excluir agente "{name}"?',
        'agents.idRequired': 'ID do agente é obrigatório',
        'agents.duplicateId': 'Agente com ID "{id}" já existe',
        'agents.restrictWorkspace': 'Restringir ao Workspace',
        'agents.allowReadOutside': 'Permitir Leitura Externa',
        'agents.modelName': 'Modelo Padrão',
        'agents.modelNameHint': 'Modelo usado por todos os agentes, exceto quando sobrescrito',
        'agents.maxTokens': 'Máx. Tokens',
        'agents.maxToolIter': 'Máx. Iterações de Ferramentas',
        'agents.summarizeThreshold': 'Limite de Resumo',
        'agents.summarizeThresholdHint': 'Mensagens antes do resumo automático',
        'agents.summarizeTokenPct': '% Tokens para Resumo',
        'agents.maxMediaSize': 'Tamanho Máx. de Mídia (bytes)',
        'agents.maxMediaSizeHint': '0 = padrão (20MB)',
        'agents.defaultAgent': 'Agente Padrão',
        'agents.friendlyName': 'ex: Assistente de Pesquisa',
        'agents.inheritsDefaults': 'Herda dos padrões',
        'agents.modelHint': 'Nome do modelo (deve corresponder a um model_name de Modelos)',
        'agents.modelPrimary': 'Modelo',
        'agents.modelFallbacks': 'Modelos de Reserva',
        'agents.sectionIdentity': 'Identidade',
        'agents.sectionModel': 'Modelo',
        'agents.sectionAdvanced': 'Avançado',
        'agents.subagents': 'Subagentes Permitidos',
        'agents.subagentsHint': 'Outros agentes que este agente pode invocar',
        'agents.inheritedFrom': 'herdado',
        'agents.overridden': 'personalizado',
        'agents.agentSaved': 'Agente salvo',
        'agents.skillsHint': 'Selecione quais skills instaladas este agente pode usar',
        // Bindings
        // MCP
        'mcp.title': 'Servidores MCP',
        'mcp.desc': 'Conecte a servidores Model Context Protocol para estender as capacidades dos agentes com ferramentas externas.',
        'mcp.addServer': '+ Adicionar Servidor',
        'mcp.addServerTitle': 'Adicionar Servidor MCP',
        'mcp.editServer': 'Editar: {name}',
        'mcp.noServers': 'Nenhum Servidor MCP',
        'mcp.noServersDesc': 'Adicione um servidor MCP para habilitar integrações com ferramentas externas.',
        'mcp.deleteConfirm': 'Excluir servidor MCP "{name}"?',
        'mcp.nameRequired': 'Nome do servidor é obrigatório',
        'mcp.testing': 'Testando conexão com {name}...',
        'mcp.testResult': '{name}: encontrada(s) {count} ferramenta(s)',
        'mcp.testFailed': 'Teste falhou: {msg}',
        'mcp.serverName': 'Nome do Servidor',
        'mcp.transport': 'Transporte',
        'mcp.transportStdio': 'stdio (processo local)',
        'mcp.transportHttp': 'HTTP/SSE (remoto)',
        'mcp.command': 'Comando',
        'mcp.arguments': 'Argumentos',
        'mcp.envVars': 'Variáveis de Ambiente',
        'mcp.serverUrl': 'URL do Servidor',
        'mcp.httpHeaders': 'Headers HTTP',
        'mcp.timeout': 'Timeout (segundos)',
        'mcp.toolPrefix': 'Prefixo de Ferramentas',
        'mcp.toolPrefixHint': 'Prefixado aos nomes das ferramentas para evitar colisões, ex. "fs" &rarr; "fs_read_file"',
        'mcp.test': 'Testar',
        // Tool Settings
        'tools.title': 'Config. Ferramentas',
        'tools.desc': 'Ative ou desative ferramentas individuais e configure opções específicas.',
        'tools.toggles': 'Ativar/Desativar Ferramentas',
        'tools.webSearch': 'Busca Web',
        'tools.webEnabled': 'Busca Web Ativada',
        'tools.webProxy': 'Proxy',
        'tools.webFetchLimit': 'Limite de Fetch (bytes)',
        'tools.shellExec': 'Execução Shell',
        'tools.execEnabled': 'Execução Ativada',
        'tools.execDenyPatterns': 'Ativar Padrões de Negação',
        'tools.execTimeout': 'Timeout (segundos)',
        'tools.cron': 'Cron',
        'tools.cronEnabled': 'Cron Ativado',
        'tools.cronTimeout': 'Timeout de Execução (minutos)',
        'tools.paths': 'Restrições de Caminho',
        'tools.allowReadPaths': 'Caminhos de Leitura Permitidos',
        'tools.allowWritePaths': 'Caminhos de Escrita Permitidos',
        'tools.customDenyPatterns': 'Padrões de Bloqueio Personalizados',
        'tools.customAllowPatterns': 'Padrões de Permissão Personalizados',
        'tools.mediaCleanup': 'Limpeza de Mídia',
        'tools.mediaMaxAge': 'Idade Máxima (minutos)',
        'tools.mediaInterval': 'Intervalo de Limpeza (minutos)',
        'tools.saved': 'Configurações de ferramentas salvas',
        // RAG
        'rag.title': 'RAG (Memória Semântica)',
        'rag.desc': 'Configure geração aumentada por recuperação para memória semântica de longo prazo. Requer um endpoint de embeddings compatível com OpenAI.',
        'rag.embeddingsUrl': 'URL da API de Embeddings',
        'rag.apiKey': 'Chave API',
        'rag.embeddingModel': 'Modelo de Embedding',
        'rag.topK': 'Top K Resultados',
        'rag.minSimilarity': 'Similaridade Mínima',
        'rag.minSimilarityHint': 'Limiar de similaridade cosseno (0-1)',
        'rag.saved': 'Configurações RAG salvas',
        // Gateway
        'gateway.title': 'Gateway',
        'gateway.desc': 'Configurações de rede do servidor gateway.',
        'gateway.host': 'Host',
        'gateway.port': 'Porta',
        'gateway.proxy': 'Proxy',
        'gateway.proxyHint': 'Proxy HTTP global (http/https/socks5). Usado por todos os canais, ferramentas e provedores.',
        'gateway.saved': 'Configurações do gateway salvas',
        // Session
        'session.title': 'Sessão',
        'session.desc': 'Configure o comportamento de sessão e vinculação de identidade.',
        'session.dmScope': 'Escopo de DM',
        'session.perChannelPeer': 'Por Canal e Peer (padrão)',
        'session.perPeer': 'Por Peer',
        'session.perChannel': 'Por Canal',
        'session.global': 'Global',
        'session.saved': 'Configurações de sessão salvas',
        // Heartbeat
        'heartbeat.title': 'Heartbeat',
        'heartbeat.desc': 'Notificações periódicas de verificação de saúde.',
        'heartbeat.interval': 'Intervalo (minutos)',
        'heartbeat.intervalHint': 'Mínimo 5 minutos',
        'heartbeat.saved': 'Configurações de heartbeat salvas',
        // Devices
        'devices.title': 'Dispositivos',
        'devices.desc': 'Monitoramento de dispositivos de hardware (somente Linux).',
        'devices.monitorUsb': 'Monitorar USB',
        'devices.saved': 'Configurações de dispositivos salvas',
        // Debug
        'debug.title': 'Debug',
        'debug.desc': 'Ative o modo debug para logs detalhados.',
        'debug.debugMode': 'Modo Debug',
        'debug.hint': 'Ativa logs detalhados. Alterações entram em vigor na próxima reinicialização do gateway.',
        'debug.saved': 'Configuração de debug salva',
        // Skills
        'skills.title': 'Skills',
        'skills.desc': 'Gerencie skills instaladas e descubra novas no registro.',
        'skills.installed': 'Skills Instaladas',
        'skills.searchInstall': 'Buscar e Instalar',
        'skills.search': 'Buscar',
        'skills.searchPlaceholder': 'Buscar skills...',
        'skills.loading': 'Carregando...',
        'skills.noSkills': 'Nenhuma Skill Instalada',
        'skills.noSkillsDesc': 'Instale skills do registro abaixo ou coloque-as no seu workspace.',
        'skills.show': 'Ver',
        'skills.remove': 'Remover',
        'skills.loadFailed': 'Falha ao carregar skills: {msg}',
        // Routing
        'routing.title': 'Roteamento de Modelos',
        'routing.desc': 'Configure o roteamento automático de modelos baseado na complexidade da mensagem.',
        'routing.lightModel': 'Modelo Leve',
        'routing.threshold': 'Limite',
        'routing.thresholdHint': 'Limite de complexidade (0-1) para roteamento ao modelo leve',
        'routing.saved': 'Configurações de roteamento salvas',
        // Providers
        'providers.provider': 'Provedor',
        // Web Search
        'webSearch.title': 'Busca Web',
        'webSearch.desc': 'Configure provedores de busca web para acesso dos agentes à internet.',
        'webSearch.global': 'Configurações Globais',
        'webSearch.saved': 'Configurações de busca web salvas',
        // Summarization
        'summarization.title': 'Resumo',
        'summarization.desc': 'Configure quando e como as conversas são resumidas para economizar contexto.',
        'summarization.saved': 'Configurações de resumo salvas',
        // Agent Defaults
        'agentDefaults.temperature': 'Temperatura',
        // WebUI
        'webui.title': 'Interface Web',
        'webui.desc': 'Configure as opções da interface web embutida.',
        'webui.saved': 'Configurações da interface web salvas',
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Desconectado',
        'chat.placeholder': 'Digite uma mensagem...',
        'chat.send': 'Enviar',
        // Additional keys
        'close': 'Fechar',
        'addItem': '+ Adicionar',
        'status.reloadTriggered': 'Recarga de configuração acionada',
        'logs.deleteAllConfirm': 'Excluir todos os arquivos de log? Isso não pode ser desfeito.',
        'logs.allDeleted': 'Todos os logs excluídos',
        'logs.deleteAll': 'Excluir Todos os Logs',
        'skills.removeConfirm': 'Remover skill "{name}"? Isso não pode ser desfeito.',
        'skills.removed': 'Skill "{name}" removida',
        'skills.searching': 'Buscando...',
        'skills.noResults': 'Nenhum resultado encontrado.',
        'skills.alreadyInstalled': 'Instalada',
        'skills.install': 'Instalar',
        'skills.installing': 'Instalando...',
        'skills.blockedMalware': 'Skill bloqueada: identificada como malware',
        'skills.installedSuspicious': 'Skill instalada mas marcada como suspeita',
        'skills.installSuccess': 'Skill "{slug}" instalada (v{version})',
        'skills.installFailed': 'Falha na instalação: {msg}',
        'skills.showFailed': 'Falha ao carregar skill: {msg}',
        'skills.removeFailed': 'Falha ao remover skill: {msg}',
    },
    es: {
        // Header & global
        'start': 'Iniciar',
        'stop': 'Detener',
        'save': 'Guardar',
        'cancel': 'Cancelar',
        'edit': 'Editar',
        'delete': 'Eliminar',
        'enabled': 'Activado',
        'disabled': 'Desactivado',
        'comingSoon': 'Próximamente',
        // Sidebar
        'sidebar.providers': 'Proveedores',
        'sidebar.models': 'Modelos',
        'sidebar.auth': 'Autenticación',
        'sidebar.channels': 'Canales',
        'sidebar.agents': 'Agentes',
        'sidebar.agentsList': 'Agentes',
        'sidebar.bindings': 'Vínculos',
        'sidebar.tools': 'Herramientas',
        'sidebar.mcp': 'Servidores MCP',
        'sidebar.toolSettings': 'Config. Herramientas',
        'sidebar.rag': 'RAG',
        'sidebar.system': 'Sistema',
        'sidebar.gateway': 'Gateway',
        'sidebar.session': 'Sesión',
        'sidebar.heartbeat': 'Heartbeat',
        'sidebar.devices': 'Dispositivos',
        'sidebar.debug': 'Debug',
        'sidebar.chat': 'Chat',
        'sidebar.logs': 'Logs',
        'sidebar.rawjson': 'JSON Crudo',
        'sidebar.input': 'Entrada',
        'sidebar.routing': 'Enrutamiento',
        'sidebar.planning': 'Planificación',
        'sidebar.execution': 'Ejecución',
        'sidebar.memory': 'Memoria',
        'sidebar.orchestrator': 'Orquestador',
        'sidebar.webSearch': 'Búsqueda Web',
        'sidebar.summarization': 'Sumarización',
        'sidebar.agentDefaults': 'Predeterminados del Agente',
        'sidebar.webui': 'Web UI',
        'sidebar.modelRouting': 'Enrutamiento de Modelo',
        // Models
        'models.title': 'Modelos',
        'models.desc': 'Administre configuraciones de modelos LLM. Los modelos sin clave API aparecen atenuados. Solo los modelos disponibles pueden definirse como primario.',
        'models.add': '+ Agregar Modelo',
        'models.noModels': 'Ningún modelo configurado.',
        'models.primary': 'Primario',
        'models.noKey': 'Sin Clave',
        'models.setPrimary': 'Definir Primario',
        'models.editModel': 'Editar Modelo',
        'models.addModel': 'Agregar Modelo',
        'models.advancedOptions': 'Opciones Avanzadas',
        'models.deleteConfirm': 'Eliminar modelo "{name}"?',
        'models.requiredFields': 'Nombre del Modelo e ID del Modelo son requeridos',
        'models.test': 'Probar',
        'models.testing': 'Probando...',
        'models.testSuccess': '{model}: {response}',
        'models.testFail': '{model}: {error}',
        // Model fields
        'field.modelName': 'Nombre del Modelo',
        'field.modelId': 'ID del Modelo',
        'field.modelIdHint': 'Formato: protocolo/id-modelo',
        'field.provider': 'Proveedor',
        'field.providerManual': '(Manual - Clave API)',
        'field.apiKey': 'Clave API',
        'field.apiBase': 'URL Base de API',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Método de Autenticación',
        'field.connectMode': 'Modo de Conexión',
        'field.thinkingLevel': 'Nivel de Pensamiento',
        'field.maxTokensField': 'Campo Max Tokens',
        'field.workspace': 'Workspace',
        'field.rpm': 'Límite RPM',
        'field.requestTimeout': 'Timeout de Solicitud (s)',
        // Auth
        'auth.title': 'Autenticación de Proveedores',
        'auth.desc': 'Inicie sesión en proveedores usando OAuth o tokens de API. Las credenciales se almacenan localmente en <code style="font-family:\'JetBrains Mono\',monospace;font-size:12px;background:var(--bg-elevated);padding:2px 6px;border-radius:4px;">~/.ok/auth.json</code>.',
        'auth.notLoggedIn': 'No autenticado',
        'auth.active': 'Activo',
        'auth.expired': 'Expirado',
        'auth.needsRefresh': 'Necesita Actualizar',
        'auth.authenticating': 'Autenticando...',
        'auth.loginDevice': 'Login (Código de Dispositivo)',
        'auth.loginToken': 'Login (Token de API)',
        'auth.loginOAuth': 'Login (OAuth en Navegador)',
        'auth.logout': 'Cerrar Sesión',
        'auth.retry': 'Reintentar',
        'auth.waiting': 'Esperando...',
        'auth.pasteKey': 'Pegue su clave API aquí...',
        'auth.step1': 'Paso 1: Haga clic en el enlace de abajo',
        'auth.step2': 'Paso 2: Ingrese este código',
        'auth.step3': 'Paso 3: Complete la autenticación en el navegador, esta página se actualiza automáticamente',
        'auth.method': 'Método',
        'auth.email': 'Correo',
        'auth.account': 'Cuenta',
        'auth.project': 'Proyecto',
        'auth.expires': 'Expira',
        'auth.custom1': 'Proveedor Personalizado 1',
        'auth.custom2': 'Proveedor Personalizado 2',
        'auth.configure': 'Configurar',
        'auth.customLabel': 'Nombre del Proveedor',
        'auth.customApiBase': 'URL Base de la API',
        'auth.customToken': 'Token de API',
        'auth.customLabelPlaceholder': 'ej: Together AI',
        'auth.customApiBasePlaceholder': 'ej: https://api.together.xyz/v1',
        // Channel
        'ch.configure': 'Configurar canal {name}.',
        'ch.docLink': 'Guía de Configuración',
        'ch.accessControl': 'Control de Acceso',
        'ch.allowFrom': 'Permitir Desde (IDs de Usuario)',
        'ch.allowedGroups': 'Grupos Permitidos',
        'ch.allowedContacts': 'Contactos Permitidos',
        'ch.addItem': '+ Agregar',
        'ch.allowDirect': 'Permitir Mensajes Directos',
        'ch.allowDirectHint': 'Responder a mensajes directos (privados)',
        'ch.allowGroups': 'Permitir Mensajes en Grupo',
        'ch.allowGroupsHint': 'Responder en grupos cuando sea mencionado',
        'ch.allowSelfHint': 'Responder a mensajes del número conectado',
        'ch.mentionOnly': 'Solo Menciones',
        'ch.mentionOnlyHint': 'Responder solo cuando sea mencionado',
        'ch.groupTrigger': 'Prefijos de Activación en Grupo',
        // Logs
        'logs.title': 'Logs del Gateway',
        'logs.desc': 'Salida en tiempo real del proceso del gateway.',
        'logs.clear': 'Limpiar Pantalla',
        'logs.autoScroll': 'Auto-desplazamiento',
        'logs.noLogs': 'No hay logs disponibles. Inicie el gateway para ver la salida aquí.',
        'logs.noCapture': 'Logs no disponibles. El gateway no fue iniciado desde este launcher.',
        'logs.debugOn': 'Debug activado (aplica en el próximo inicio)',
        'logs.debugOff': 'Debug desactivado',
        'logs.allComponents': 'Todos los Componentes',
        'logs.filterLevel': 'Todos los Niveles',
        'logs.filterSearch': 'Filtrar...',
        // Raw JSON
        'raw.title': 'JSON Crudo',
        'raw.desc': 'Edite directamente el archivo de configuración.',
        'raw.reload': 'Recargar',
        'raw.format': 'Formatear',
        // Status messages
        'status.configLoaded': 'Configuración cargada',
        'status.configSaved': 'Configuración guardada',
        'status.loadFailed': 'Error al cargar',
        'status.saveFailed': 'Error al guardar',
        'status.formatted': 'Formateado',
        'status.invalidJson': 'JSON inválido, corríjalo antes de guardar',
        'status.jsonValid': 'JSON válido',
        'status.jsonInvalid': 'JSON inválido',
        'status.saved': '{name} guardado',
        'status.tokenEmpty': 'El token no puede estar vacío',
        'status.tokenSaved': 'Token guardado para {name}',
        'status.loggedOut': 'Sesión cerrada de {name}',
        'status.loginFailed': 'Error de inicio de sesión',
        'status.logoutFailed': 'Error al cerrar sesión',
        'status.openingBrowser': 'Abriendo navegador para autenticación...',
        'status.loginStarted': 'Inicio de sesión iniciado...',
        'status.loginSuccess': '¡Inicio de sesión exitoso!',
        // Process
        'process.running': 'Servicio en ejecución',
        'process.notRunning': 'Servicio detenido',
        'process.starting': 'Iniciando gateway...',
        'process.started': 'Gateway iniciado',
        'process.startFailed': 'Error al iniciar gateway',
        'process.stopping': 'Deteniendo gateway...',
        'process.stopped': 'Gateway detenido',
        'process.stopFailed': 'Error al detener gateway',
        'process.needModel': 'Se requiere al menos un modelo con clave API',
        'process.needChannel': 'Al menos un canal debe estar activado',
        'process.checkLogs': 'Revise los Logs para detalles',
        'process.needBoth': 'Se necesita modelo con clave API y canal activado para iniciar',
        // Onboarding
        'onboarding.title': 'Primeros Pasos',
        'onboarding.install': 'Instalar OK',
        'onboarding.model': 'Agregar un modelo LLM',
        'onboarding.channel': 'Conectar un canal',
        'onboarding.start': 'Iniciar el gateway',
        // Status indicator
        'status.setup': 'Configuración necesaria',
        'status.ready': 'Listo',
        'status.running': 'En ejecución',
        // Agents
        'agents.title': 'Agentes',
        'agents.desc': 'Configure valores predeterminados de agentes y cree agentes especializados.',
        'agents.defaults': 'Configuración Predeterminada',
        'agents.defaultsDesc': 'Configuración base heredada por todos los agentes. Los agentes individuales pueden sobrescribir estos valores.',
        'agents.agentList': 'Agentes',
        'agents.addAgent': '+ Agregar Agente',
        'agents.editAgent': 'Editar Agente',
        'agents.addAgentTitle': 'Agregar Agente',
        'agents.noAgents': 'Sin agentes personalizados. Todos los mensajes son manejados por el agente predeterminado con la configuración anterior.',
        'agents.saveDefaults': 'Guardar Predeterminados',
        'agents.defaultsSaved': 'Valores predeterminados de agente guardados',
        'agents.deleteConfirm': 'Eliminar agente "{name}"?',
        'agents.idRequired': 'ID del agente es requerido',
        'agents.duplicateId': 'Ya existe un agente con ID "{id}"',
        'agents.restrictWorkspace': 'Restringir al Workspace',
        'agents.allowReadOutside': 'Permitir Lectura Externa',
        'agents.modelName': 'Modelo Predeterminado',
        'agents.modelNameHint': 'Modelo usado por todos los agentes, excepto cuando se sobrescribe',
        'agents.maxTokens': 'Máx. Tokens',
        'agents.maxToolIter': 'Máx. Iteraciones de Herramientas',
        'agents.summarizeThreshold': 'Umbral de Resumen',
        'agents.summarizeThresholdHint': 'Mensajes antes del resumen automático',
        'agents.summarizeTokenPct': '% Tokens para Resumen',
        'agents.maxMediaSize': 'Tamaño Máx. de Media (bytes)',
        'agents.maxMediaSizeHint': '0 = predeterminado (20MB)',
        'agents.defaultAgent': 'Agente Predeterminado',
        'agents.friendlyName': 'ej: Asistente de Investigación',
        'agents.inheritsDefaults': 'Hereda de los predeterminados',
        'agents.modelHint': 'Nombre del modelo (debe coincidir con un model_name de Modelos)',
        'agents.modelPrimary': 'Modelo',
        'agents.modelFallbacks': 'Modelos de Respaldo',
        'agents.sectionIdentity': 'Identidad',
        'agents.sectionModel': 'Modelo',
        'agents.sectionAdvanced': 'Avanzado',
        'agents.subagents': 'Subagentes Permitidos',
        'agents.subagentsHint': 'Otros agentes que este agente puede invocar',
        'agents.inheritedFrom': 'heredado',
        'agents.overridden': 'personalizado',
        'agents.agentSaved': 'Agente guardado',
        'agents.skillsHint': 'Seleccione qué skills instaladas puede usar este agente',
        // Bindings
        // MCP
        'mcp.title': 'Servidores MCP',
        'mcp.desc': 'Conéctese a servidores Model Context Protocol para extender las capacidades de los agentes con herramientas externas.',
        'mcp.addServer': '+ Agregar Servidor',
        'mcp.addServerTitle': 'Agregar Servidor MCP',
        'mcp.editServer': 'Editar: {name}',
        'mcp.noServers': 'Sin Servidores MCP',
        'mcp.noServersDesc': 'Agregue un servidor MCP para habilitar integraciones con herramientas externas.',
        'mcp.deleteConfirm': '¿Eliminar servidor MCP "{name}"?',
        'mcp.nameRequired': 'El nombre del servidor es requerido',
        'mcp.testing': 'Probando conexión con {name}...',
        'mcp.testResult': '{name}: encontrada(s) {count} herramienta(s)',
        'mcp.testFailed': 'Prueba fallida: {msg}',
        'mcp.serverName': 'Nombre del Servidor',
        'mcp.transport': 'Transporte',
        'mcp.transportStdio': 'stdio (proceso local)',
        'mcp.transportHttp': 'HTTP/SSE (remoto)',
        'mcp.command': 'Comando',
        'mcp.arguments': 'Argumentos',
        'mcp.envVars': 'Variables de Entorno',
        'mcp.serverUrl': 'URL del Servidor',
        'mcp.httpHeaders': 'Headers HTTP',
        'mcp.timeout': 'Timeout (segundos)',
        'mcp.toolPrefix': 'Prefijo de Herramientas',
        'mcp.toolPrefixHint': 'Se antepone a los nombres de herramientas para evitar colisiones, ej. "fs" &rarr; "fs_read_file"',
        'mcp.test': 'Probar',
        // Tool Settings
        'tools.title': 'Config. Herramientas',
        'tools.desc': 'Active o desactive herramientas individuales y configure opciones específicas.',
        'tools.toggles': 'Activar/Desactivar Herramientas',
        'tools.webSearch': 'Búsqueda Web',
        'tools.webEnabled': 'Búsqueda Web Activada',
        'tools.webProxy': 'Proxy',
        'tools.webFetchLimit': 'Límite de Fetch (bytes)',
        'tools.shellExec': 'Ejecución Shell',
        'tools.execEnabled': 'Ejecución Activada',
        'tools.execDenyPatterns': 'Activar Patrones de Denegación',
        'tools.execTimeout': 'Timeout (segundos)',
        'tools.cron': 'Cron',
        'tools.cronEnabled': 'Cron Activado',
        'tools.cronTimeout': 'Timeout de Ejecución (minutos)',
        'tools.paths': 'Restricciones de Ruta',
        'tools.allowReadPaths': 'Rutas de Lectura Permitidas',
        'tools.allowWritePaths': 'Rutas de Escritura Permitidas',
        'tools.customDenyPatterns': 'Patrones de Denegación Personalizados',
        'tools.customAllowPatterns': 'Patrones de Permiso Personalizados',
        'tools.mediaCleanup': 'Limpieza de Medios',
        'tools.mediaMaxAge': 'Edad Máxima (minutos)',
        'tools.mediaInterval': 'Intervalo de Limpieza (minutos)',
        'tools.saved': 'Configuraciones de herramientas guardadas',
        // RAG
        'rag.title': 'RAG (Memoria Semántica)',
        'rag.desc': 'Configure generación aumentada por recuperación para memoria semántica a largo plazo. Requiere un endpoint de embeddings compatible con OpenAI.',
        'rag.embeddingsUrl': 'URL de API de Embeddings',
        'rag.apiKey': 'Clave API',
        'rag.embeddingModel': 'Modelo de Embedding',
        'rag.topK': 'Top K Resultados',
        'rag.minSimilarity': 'Similitud Mínima',
        'rag.minSimilarityHint': 'Umbral de similitud coseno (0-1)',
        'rag.saved': 'Configuraciones RAG guardadas',
        // Gateway
        'gateway.title': 'Gateway',
        'gateway.desc': 'Configuraciones de red del servidor gateway.',
        'gateway.host': 'Host',
        'gateway.port': 'Puerto',
        'gateway.proxy': 'Proxy',
        'gateway.proxyHint': 'Proxy HTTP global (http/https/socks5). Usado por todos los canales, herramientas y proveedores.',
        'gateway.saved': 'Configuraciones del gateway guardadas',
        // Session
        'session.title': 'Sesión',
        'session.desc': 'Configure el comportamiento de sesión y vinculación de identidad.',
        'session.dmScope': 'Ámbito de DM',
        'session.perChannelPeer': 'Por Canal y Peer (predeterminado)',
        'session.perPeer': 'Por Peer',
        'session.perChannel': 'Por Canal',
        'session.global': 'Global',
        'session.saved': 'Configuraciones de sesión guardadas',
        // Heartbeat
        'heartbeat.title': 'Heartbeat',
        'heartbeat.desc': 'Notificaciones periódicas de verificación de salud.',
        'heartbeat.interval': 'Intervalo (minutos)',
        'heartbeat.intervalHint': 'Mínimo 5 minutos',
        'heartbeat.saved': 'Configuraciones de heartbeat guardadas',
        // Devices
        'devices.title': 'Dispositivos',
        'devices.desc': 'Monitoreo de dispositivos de hardware (solo Linux).',
        'devices.monitorUsb': 'Monitorear USB',
        'devices.saved': 'Configuraciones de dispositivos guardadas',
        // Debug
        'debug.title': 'Debug',
        'debug.desc': 'Active el modo debug para logs detallados.',
        'debug.debugMode': 'Modo Debug',
        'debug.hint': 'Activa logs detallados. Los cambios aplican en el próximo reinicio del gateway.',
        'debug.saved': 'Configuración de debug guardada',
        // Skills
        'skills.title': 'Skills',
        'skills.desc': 'Administre skills instaladas y descubra nuevas en el registro.',
        'skills.installed': 'Skills Instaladas',
        'skills.searchInstall': 'Buscar e Instalar',
        'skills.search': 'Buscar',
        'skills.searchPlaceholder': 'Buscar skills...',
        'skills.loading': 'Cargando...',
        'skills.noSkills': 'Sin Skills Instaladas',
        'skills.noSkillsDesc': 'Instale skills del registro o colóquelas en su workspace.',
        'skills.show': 'Ver',
        'skills.remove': 'Eliminar',
        'skills.loadFailed': 'Error al cargar skills: {msg}',
        // Routing
        'routing.title': 'Enrutamiento de Modelos',
        'routing.desc': 'Configure el enrutamiento automático de modelos basado en la complejidad del mensaje.',
        'routing.lightModel': 'Modelo Ligero',
        'routing.threshold': 'Umbral',
        'routing.thresholdHint': 'Umbral de complejidad (0-1) para enrutamiento al modelo ligero',
        'routing.saved': 'Configuración de enrutamiento guardada',
        // Providers
        'providers.provider': 'Proveedor',
        // Web Search
        'webSearch.title': 'Búsqueda Web',
        'webSearch.desc': 'Configure proveedores de búsqueda web para acceso de los agentes a internet.',
        'webSearch.global': 'Configuración Global',
        'webSearch.saved': 'Configuración de búsqueda web guardada',
        // Summarization
        'summarization.title': 'Resumen',
        'summarization.desc': 'Configure cuándo y cómo se resumen las conversaciones para ahorrar contexto.',
        'summarization.saved': 'Configuración de resumen guardada',
        // Agent Defaults
        'agentDefaults.temperature': 'Temperatura',
        // WebUI
        'webui.title': 'Interfaz Web',
        'webui.desc': 'Configure las opciones de la interfaz web integrada.',
        'webui.saved': 'Configuración de interfaz web guardada',
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Desconectado',
        'chat.placeholder': 'Escriba un mensaje...',
        'chat.send': 'Enviar',
        // Additional keys
        'close': 'Cerrar',
        'addItem': '+ Agregar',
        'status.reloadTriggered': 'Recarga de configuración activada',
        'logs.deleteAllConfirm': '¿Eliminar todos los archivos de log? Esto no se puede deshacer.',
        'logs.allDeleted': 'Todos los logs eliminados',
        'logs.deleteAll': 'Eliminar Todos los Logs',
        'skills.removeConfirm': '¿Eliminar skill "{name}"? Esto no se puede deshacer.',
        'skills.removed': 'Skill "{name}" eliminada',
        'skills.searching': 'Buscando...',
        'skills.noResults': 'No se encontraron resultados.',
        'skills.alreadyInstalled': 'Instalada',
        'skills.install': 'Instalar',
        'skills.installing': 'Instalando...',
        'skills.blockedMalware': 'Skill bloqueada: identificada como malware',
        'skills.installedSuspicious': 'Skill instalada pero marcada como sospechosa',
        'skills.installSuccess': 'Skill "{slug}" instalada (v{version})',
        'skills.installFailed': 'Error de instalación: {msg}',
        'skills.showFailed': 'Error al cargar skill: {msg}',
        'skills.removeFailed': 'Error al eliminar skill: {msg}',
    }
};

function t(key, params) {
    let s = (i18nData[currentLang] && i18nData[currentLang][key]) || i18nData.en[key] || key;
    if (params) {
        Object.keys(params).forEach(k => { s = s.replace('{' + k + '}', params[k]); });
    }
    return s;
}

function switchLang(lang) {
    currentLang = lang;
    localStorage.setItem('ok-lang', lang);
    const sel = document.getElementById('langSelect');
    if (sel) sel.value = lang;
    applyI18n();
    // Re-render all panels
    if (configData) {
        renderModels(); renderAgents();
        renderMCP(); renderToolSettings(); renderRAG();
        renderGateway(); renderSession(); renderHeartbeat();
        renderDevices(); renderDebug(); renderSkills();
        loadAuthStatus();
        // Re-render active channel
        const activeCh = document.querySelector('.content-panel.active[id^="panelCh_"]');
        if (activeCh) renderChannelForm(activeCh.id.replace('panelCh_', ''));
    }
}

function applyI18n() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        const val = t(key);
        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
            el.placeholder = val;
        } else {
            el.textContent = val;
        }
    });
    document.getElementById('authDesc').innerHTML = t('auth.desc') + ' <button class="btn btn-sm btn-primary" onclick="showAddProviderModal()" style="margin-left:12px">+ Add Provider</button>';
    // Sync selector
    const sel = document.getElementById('langSelect');
    if (sel) sel.value = currentLang;
}

// ── State ───────────────────────────────────────────
let configData = null;
let configPath = '';
let authPollTimer = null;
let editingModelIndex = -1;

// ── Nested value helpers ────────────────────────────
function getNestedValue(obj, path) {
    return path.split('.').reduce((o, k) => o && o[k], obj);
}
function setNestedValue(obj, path, val) {
    const keys = path.split('.');
    const last = keys.pop();
    const target = keys.reduce((o, k) => { if (!o[k]) o[k] = {}; return o[k]; }, obj);
    target[last] = val;
}

// ── Channel schemas ─────────────────────────────────
const channelSchemas = {
    chat: {
        title: 'Web Chat', configKey: 'chat', docSlug: null,
        fields: []
    },
    telegram: {
        title: 'Telegram', configKey: 'telegram', docSlug: 'telegram',
        fields: [
            { key: 'token', label: 'Bot Token', type: 'password', placeholder: 'Telegram bot token from @BotFather' },
            { key: 'allow_direct', label: 'ch.allowDirect', type: 'toggle', hint: 'ch.allowDirectHint', i18nLabel: true, i18nHint: true },
            { key: 'allow_groups', label: 'ch.allowGroups', type: 'toggle', hint: 'ch.allowGroupsHint', i18nLabel: true, i18nHint: true },
            { key: 'group_trigger.mention_only', label: 'Mention Only (Groups)', type: 'toggle' },
            { key: 'group_trigger.prefixes', label: 'Group Prefixes', type: 'array', placeholder: '/ok' },
            { key: 'typing.enabled', label: 'Show Typing Indicator', type: 'toggle' },
            { key: 'placeholder.enabled', label: 'Placeholder Message', type: 'toggle' },
            { key: 'placeholder.text', label: 'Placeholder Text', type: 'text', placeholder: 'Thinking...' },
            { key: 'reasoning_channel_id', label: 'Reasoning Channel ID', type: 'text', hint: 'Channel ID for reasoning output' },
        ]
    },
    discord: {
        title: 'Discord', configKey: 'discord', docSlug: 'discord',
        fields: [
            { key: 'token', label: 'Bot Token', type: 'password', placeholder: 'Discord bot token' },
            { key: 'allow_direct', label: 'ch.allowDirect', type: 'toggle', hint: 'ch.allowDirectHint', i18nLabel: true, i18nHint: true },
            { key: 'allow_groups', label: 'ch.allowGroups', type: 'toggle', hint: 'ch.allowGroupsHint', i18nLabel: true, i18nHint: true },
            { key: 'mention_only', label: 'ch.mentionOnly', type: 'toggle', hint: 'ch.mentionOnlyHint', i18nLabel: true },
            { key: 'group_trigger.mention_only', label: 'Mention Only (Groups)', type: 'toggle' },
            { key: 'group_trigger.prefixes', label: 'Group Prefixes', type: 'array', placeholder: '/ok' },
            { key: 'typing.enabled', label: 'Show Typing Indicator', type: 'toggle' },
            { key: 'placeholder.enabled', label: 'Placeholder Message', type: 'toggle' },
            { key: 'placeholder.text', label: 'Placeholder Text', type: 'text', placeholder: 'Thinking...' },
            { key: 'reasoning_channel_id', label: 'Reasoning Channel ID', type: 'text', hint: 'Channel ID for reasoning output' },
        ]
    },
    slack: {
        title: 'Slack', configKey: 'slack', docSlug: 'slack',
        fields: [
            { key: 'bot_token', label: 'Bot Token', type: 'password', placeholder: 'xoxb-...' },
            { key: 'app_token', label: 'App Token', type: 'password', placeholder: 'xapp-...' },
            { key: 'allow_direct', label: 'ch.allowDirect', type: 'toggle', hint: 'ch.allowDirectHint', i18nLabel: true, i18nHint: true },
            { key: 'allow_groups', label: 'ch.allowGroups', type: 'toggle', hint: 'ch.allowGroupsHint', i18nLabel: true, i18nHint: true },
            { key: 'group_trigger.mention_only', label: 'Mention Only (Groups)', type: 'toggle' },
            { key: 'group_trigger.prefixes', label: 'Group Prefixes', type: 'array', placeholder: '/ok' },
            { key: 'typing.enabled', label: 'Show Typing Indicator', type: 'toggle' },
            { key: 'placeholder.enabled', label: 'Placeholder Message', type: 'toggle' },
            { key: 'placeholder.text', label: 'Placeholder Text', type: 'text', placeholder: 'Thinking...' },
            { key: 'reasoning_channel_id', label: 'Reasoning Channel ID', type: 'text', hint: 'Channel ID for reasoning output' },
        ]
    },
    whatsapp: {
        title: 'WhatsApp', configKey: 'whatsapp', docSlug: null,
        fields: [
            { key: 'session_store_path', label: 'Session Store Path', type: 'text', placeholder: '~/.ok/workspace/whatsapp' },
            { key: 'allow_self', label: 'Allow Self Chat', type: 'toggle', hint: 'ch.allowSelfHint', i18nHint: true },
            { key: 'allow_direct', label: 'ch.allowDirect', type: 'toggle', hint: 'ch.allowDirectHint', i18nLabel: true, i18nHint: true },
            { key: 'allow_groups', label: 'ch.allowGroups', type: 'toggle', hint: 'ch.allowGroupsHint', i18nLabel: true, i18nHint: true },
            { key: 'reasoning_channel_id', label: 'Reasoning Channel ID', type: 'text', hint: 'Channel ID for reasoning output' },
        ]
    },
};

// ── Tab Navigation ──────────────────────────────────
const tabDefs = {
    input: [
        { panel: 'panelCh_chat', label: 'Web Chat', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M2 3h12v8H6l-3 2v-2H2z"/></svg>' },
        { panel: 'panelCh_telegram', label: 'Telegram', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M14 2L7 9m7-7l-4 12-3-6L1 6z"/></svg>' },
        { panel: 'panelCh_discord', label: 'Discord', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5.5 3S7 2 8 2s2.5 1 2.5 1M3.5 5.5C3 7 2.5 9 3 11c.3 1 1.5 2 2 2.5l1-1.5m5-6.5c.5 1.5 1 3.5.5 5.5-.3 1-1.5 2-2 2.5l-1-1.5"/><circle cx="6" cy="8.5" r=".8" fill="currentColor" stroke="none"/><circle cx="10" cy="8.5" r=".8" fill="currentColor" stroke="none"/></svg>' },
        { panel: 'panelCh_slack', label: 'Slack', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"><path d="M3 6h10M3 10h10M6 3v10M10 3v10"/></svg>' },
        { panel: 'panelCh_whatsapp', label: 'WhatsApp', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M6.5 6c0-.5.5-.8.8-.5l.5.5c.3.5 0 1.2 0 1.2s.8.8 1.2 1.2c0 0 .7-.3 1.2 0l.5.5c.3.3 0 .8-.5.8C8.5 10.5 5.5 7.5 6.5 6z"/></svg>' },
    ],
    routing: [
        { panel: 'panelSession', label: 'Session', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M8 2a6 6 0 110 12A6 6 0 018 2zM8 5v3l2 2"/></svg>' },
        { panel: 'panelRouting', label: 'Model Routing', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 8h4l2-4 2 8 2-4h4"/></svg>' },
    ],
    planning: [
        { panel: 'panelAuth', label: 'Providers', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M4 4h8v8H4z"/><path d="M6 2v2M10 2v2M6 12v2M10 12v2M2 6h2M2 10h2M12 6h2M12 10h2"/></svg>' },
        { panel: 'panelModels', label: 'Models', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M8 2L2 5l6 3 6-3Zm0 6L2 11l6 3 6-3Z"/></svg>' },
    ],
    execution: [
        { panel: 'panelToolSettings', label: 'Tools', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M6.5 2v3M6.5 9v5M11.5 2v7M11.5 13v1"/><circle cx="6.5" cy="7" r="2"/><circle cx="11.5" cy="11" r="2"/></svg>' },
        { panel: 'panelMCP', label: 'MCP', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="3" width="12" height="10" rx="2"/><path d="M5 8h6M8 6v4"/></svg>' },
        { panel: 'panelSkills', label: 'Skills', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M8 2l2 4h4l-3 3 1 4-4-2-4 2 1-4-3-3h4z"/></svg>' },
        { panel: 'panelWebSearch', label: 'Web Search', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="7" cy="7" r="4"/><path d="M10 10l3 3"/></svg>' },
    ],
    memory: [
        { panel: 'panelRAG', label: 'RAG', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 4h12M2 8h8M2 12h10"/><circle cx="13" cy="10" r="2"/></svg>' },
        { panel: 'panelSummarization', label: 'Summarization', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M3 3h10M3 6h8M3 9h6M3 12h4"/></svg>' },
    ],
    orchestrator: [
        { panel: 'panelAgents', label: 'Agents', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="5" r="3"/><path d="M3 14c0-2.8 2.2-5 5-5s5 2.2 5 5"/></svg>' },
    ],
    system: [
        { panel: 'panelGateway', label: 'Gateway', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="2" width="12" height="5" rx="1.5"/><rect x="2" y="9" width="12" height="5" rx="1.5"/><circle cx="5" cy="4.5" r="1" fill="currentColor" stroke="none"/><circle cx="5" cy="11.5" r="1" fill="currentColor" stroke="none"/></svg>' },
        { panel: 'panelHeartbeat', label: 'Heartbeat', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 8h2l2-4 3 8 2-4h3"/></svg>' },
        { panel: 'panelDevices', label: 'Devices', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="3" y="2" width="10" height="8" rx="1.5"/><path d="M6 13h4M8 10v3"/></svg>' },
        { panel: 'panelDebug', label: 'Debug', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="3"/><path d="M8 2v2M8 12v2M2 8h2M12 8h2M3.8 3.8l1.4 1.4M10.8 10.8l1.4 1.4M3.8 12.2l1.4-1.4M10.8 5.2l1.4-1.4"/></svg>' },
        { panel: 'panelWebUI', label: 'Web UI', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="2" width="12" height="10" rx="1.5"/><path d="M2 5h12M5 2v3"/></svg>' },
    ],
    chat: [{ panel: 'panelChat' }],
    logs: [{ panel: 'panelLogs' }],
    json: [{ panel: 'panelRawJson' }],
};

// Track which sub-tab was last active per group
const lastSubTab = { input: 'panelCh_telegram', routing: 'panelSession', planning: 'panelAuth', execution: 'panelToolSettings', memory: 'panelRAG', orchestrator: 'panelAgents', system: 'panelGateway' };

// ── Hash routing ────────────────────────────────────
// Maps panel IDs to short URL hashes and vice versa.
const panelToHash = {};
const hashToPanel = {};
const panelToGroup = {};
(function buildHashMaps() {
    for (const [group, items] of Object.entries(tabDefs)) {
        for (const it of items) {
            // panelModels → models, panelCh_whatsapp → whatsapp, panelRawJson → json, panelChat → chat
            let slug = it.panel
                .replace(/^panelCh_/, '')
                .replace(/^panel/, '')
                .replace(/([a-z])([A-Z])/g, '$1-$2')
                .toLowerCase();
            // Use label when available for cleaner slugs
            if (it.label) slug = it.label.toLowerCase().replace(/\s+/g, '-');
            panelToHash[it.panel] = slug;
            hashToPanel[slug] = it.panel;
            panelToGroup[it.panel] = group;
        }
    }
})();

let hashUpdateSilent = false; // prevent hashchange loop

function activatePanel(panelId) {
    document.querySelectorAll('.content-panel').forEach(p => p.classList.remove('active'));
    const panel = document.getElementById(panelId);
    if (panel) panel.classList.add('active');

    // Update URL hash
    const slug = panelToHash[panelId];
    if (slug && window.location.hash !== '#' + slug) {
        hashUpdateSilent = true;
        window.location.hash = slug;
    }

    if (panelId === 'panelModels') renderModels();
    if (panelId === 'panelAuth') loadAuthStatus();
    if (panelId === 'panelRawJson') syncEditorFromConfig();
    if (panelId === 'panelChat') chatConnect();
    if (panelId === 'panelLogs') { loadLogComponents(); startLogPolling(); } else { stopLogPolling(); }
    if (panelId === 'panelMCP') renderMCP();
    if (panelId === 'panelAgents') renderAgents();
    if (panelId === 'panelToolSettings') renderToolSettings();
    if (panelId === 'panelRAG') renderRAG();
    if (panelId === 'panelGateway') renderGateway();
    if (panelId === 'panelSession') renderSession();
    if (panelId === 'panelHeartbeat') renderHeartbeat();
    if (panelId === 'panelDevices') renderDevices();
    if (panelId === 'panelSkills') renderSkills();
    if (panelId === 'panelDebug') renderDebug();
    if (panelId === 'panelRouting') renderRouting();
    if (panelId === 'panelWebSearch') renderWebSearch();
    if (panelId === 'panelSummarization') renderSummarization();
    // panelAgentDefaults merged into panelAgents
    if (panelId === 'panelWebUI') renderWebUI();
    if (panelId.startsWith('panelCh_')) {
        renderChannelForm(panelId.replace('panelCh_', ''));
    }
}

function renderSubTabs(tabId) {
    const subBar = document.getElementById('subTabBar');
    const items = tabDefs[tabId];
    if (!items || items.length <= 1) {
        subBar.innerHTML = '';
        return;
    }
    const active = lastSubTab[tabId] || items[0].panel;
    subBar.innerHTML = items.map(it =>
        `<button class="sub-tab${it.panel === active ? ' active' : ''}" data-panel="${it.panel}">${it.icon || ''}${it.label}</button>`
    ).join('');

    subBar.querySelectorAll('.sub-tab').forEach(btn => {
        btn.addEventListener('click', () => {
            subBar.querySelectorAll('.sub-tab').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const pid = btn.dataset.panel;
            lastSubTab[tabId] = pid;
            activatePanel(pid);
        });
    });
}

function switchTab(tabId) {
    // Update rail buttons
    document.querySelectorAll('.rail-btn').forEach(t => t.classList.remove('active'));
    const tabBtn = document.querySelector(`.rail-btn[data-tab="${tabId}"]`);
    if (tabBtn) tabBtn.classList.add('active');

    // Render sub-tabs
    renderSubTabs(tabId);

    // Activate the panel
    const items = tabDefs[tabId];
    if (items.length === 1) {
        activatePanel(items[0].panel);
    } else {
        const active = lastSubTab[tabId] || items[0].panel;
        activatePanel(active);
    }
}

// Bind rail button clicks
document.querySelectorAll('.rail-btn').forEach(btn => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab));
});

// Navigate to panel from URL hash
function navigateToHash() {
    const slug = window.location.hash.replace(/^#/, '');
    const panelId = hashToPanel[slug];
    if (!panelId) return false;
    const group = panelToGroup[panelId];
    if (!group) return false;
    lastSubTab[group] = panelId;
    switchTab(group);
    return true;
}

window.addEventListener('hashchange', function() {
    if (hashUpdateSilent) { hashUpdateSilent = false; return; }
    navigateToHash();
});

// Initialize: navigate from hash or show Core > Models
if (!navigateToHash()) switchTab('input');

// ── Status messages ─────────────────────────────────
function showStatus(text, type) {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    toast.textContent = (type === 'success' ? '\u2713 ' : '\u2717 ') + text;
    container.appendChild(toast);
    setTimeout(() => {
        toast.classList.add('fade-out');
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// ── Config API ──────────────────────────────────────
async function loadConfig() {
    try {
        const res = await fetch('/api/config');
        if (!res.ok) throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
        const data = await res.json();
        configData = data.config;
        // Sync debug toggle
        debugEnabled = !!(configData && configData.debug);
        const dt = document.getElementById('debugToggle');
        if (dt) dt.classList.toggle('on', debugEnabled);
        configPath = data.path || '';
        document.getElementById('filePath').textContent = configPath || '-';
        renderModels();
        updateRunStopButton(gatewayRunning);
        updateChatVisibility(!!(configData.channels && configData.channels.chat && configData.channels.chat.enabled));
    } catch (e) {
        showStatus(t('status.loadFailed') + ': ' + e.message, 'error');
    }
}

async function saveConfig() {
    if (!configData) return;
    try {
        const res = await fetch('/api/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(configData),
        });
        if (!res.ok) throw new Error('HTTP ' + res.status + ': ' + (await res.text()));
        showStatus(t('status.configSaved'), 'success');
    } catch (e) {
        showStatus(t('status.saveFailed') + ': ' + e.message, 'error');
    }
    // Refresh start button state after config changes
    updateRunStopButton(gatewayRunning);
}

// ── Raw JSON editor ─────────────────────────────────
function syncEditorFromConfig() {
    const editor = document.getElementById('editor');
    if (configData) editor.value = JSON.stringify(configData, null, 2);
    document.getElementById('filePath').textContent = configPath || '-';
    validateJson();
}

async function saveRawConfig() {
    const obj = validateJson();
    if (obj === null) { showStatus(t('status.invalidJson'), 'error'); return; }
    configData = obj;
    await saveConfig();
    document.getElementById('editor').value = JSON.stringify(configData, null, 2);
}

function formatJson() {
    const obj = validateJson();
    if (obj !== null) {
        document.getElementById('editor').value = JSON.stringify(obj, null, 2);
        showStatus(t('status.formatted'), 'success');
    }
}

function validateJson() {
    const editor = document.getElementById('editor');
    const jsonStatus = document.getElementById('jsonStatus');
    const editorWrap = document.getElementById('editorWrapper');
    const val = editor.value.trim();
    if (!val) { jsonStatus.textContent = '-'; editorWrap.classList.remove('error'); return null; }
    try {
        const obj = JSON.parse(val);
        jsonStatus.textContent = '\u2713 ' + t('status.jsonValid');
        jsonStatus.style.color = 'var(--success)';
        editorWrap.classList.remove('error');
        return obj;
    } catch (e) {
        jsonStatus.textContent = '\u2717 ' + t('status.jsonInvalid') + ': ' + e.message;
        jsonStatus.style.color = 'var(--error)';
        editorWrap.classList.add('error');
        return null;
    }
}

document.getElementById('editor').addEventListener('input', validateJson);
document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        if (document.getElementById('panelRawJson').classList.contains('active')) saveRawConfig();
        else saveConfig();
    }
});

// ── Models Panel ────────────────────────────────────
function renderModels() {
    const grid = document.getElementById('modelGrid');
    if (!configData || !configData.model_list || configData.model_list.length === 0) {
        grid.innerHTML = `<div class="empty-state">
            <div class="empty-state-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"><path d="M8 2L2 5l6 3 6-3Zm0 6L2 11l6 3 6-3Z"/></svg></div>
            <div class="empty-state-title">${t('models.noModels')}</div>
            <div class="empty-state-desc">Add an LLM model to get started.</div>
        </div>`;
        updateOnboarding();
        updateStatusIndicator();
        return;
    }

    const primaryModelName = (configData.agents && configData.agents.defaults)
        ? (configData.agents.defaults.model_name || configData.agents.defaults.model || '')
        : '';

    // Build index array sorted: primary first, then the rest in original order
    const indices = configData.model_list.map((_, i) => i);
    indices.sort((a, b) => {
        const ap = configData.model_list[a].model_name === primaryModelName ? 0 : 1;
        const bp = configData.model_list[b].model_name === primaryModelName ? 0 : 1;
        return ap !== bp ? ap - bp : a - b;
    });

    const isModelAvailable = isModelAvailableGlobal;

    let html = '';
    indices.forEach(idx => {
        const m = configData.model_list[idx];
        const available = isModelAvailable(m);
        const isPrimary = m.model_name === primaryModelName;
        const protocol = m.model ? m.model.split('/')[0] : '';

        html += `<div class="model-card ${available ? '' : 'unavailable'}">`;
        html += `<div class="model-card-head">`;
        html += `<div class="model-name">${esc(m.model_name)}`;
        if (protocol) html += ` <span class="model-protocol">${esc(protocol)}</span>`;
        html += `</div>`;
        if (isPrimary) html += `<span class="badge-primary">${t('models.primary')}</span>`;
        else if (!available) html += `<span class="badge-nokey">${t('models.noKey')}</span>`;
        html += `</div>`;

        html += `<div class="model-detail"><strong>Model:</strong> ${esc(m.model || '-')}</div>`;
        if (m.provider) html += `<div class="model-detail"><strong>Provider:</strong> <span class="model-protocol">${esc(m.provider)}</span></div>`;

        html += `<div class="model-actions">`;
        html += `<button class="btn btn-sm" onclick="showEditModelModal(${idx})">${t('edit')}</button>`;
        if (available) {
            html += `<button class="btn btn-sm" id="testBtn_${idx}" onclick="testModel(${idx})">${t('models.test')}</button>`;
        }
        const isEmbedding = m.model_name === 'embedding';
        const isTranscription = m.model_name === 'transcription';
        if (available && !isPrimary && !isEmbedding && !isTranscription) {
            html += `<button class="btn btn-sm btn-success" onclick="setPrimaryModel(${idx})">${t('models.setPrimary')}</button>`;
        }
        const isBuiltin = m.model_name === 'default' || m.model_name === 'embedding' || m.model_name === 'transcription';
        if (!isBuiltin) {
            html += `<button class="btn btn-sm btn-danger" onclick="deleteModel(${idx})">${t('delete')}</button>`;
        }
        html += `</div></div>`;
    });
    grid.innerHTML = html;
    updateOnboarding();
    updateStatusIndicator();
}

function setPrimaryModel(idx) {
    if (!configData || !configData.model_list[idx]) return;
    if (!configData.agents) configData.agents = {};
    if (!configData.agents.defaults) configData.agents.defaults = {};
    configData.agents.defaults.model_name = configData.model_list[idx].model_name;
    saveConfig().then(renderModels);
}

function deleteModel(idx) {
    if (!configData || !configData.model_list) return;
    const name = configData.model_list[idx].model_name;
    if (!confirm(t('models.deleteConfirm', { name }))) return;
    configData.model_list.splice(idx, 1);
    saveConfig().then(renderModels);
}

async function testModel(idx) {
    const btn = document.getElementById('testBtn_' + idx);
    if (!btn) return;
    const origText = btn.textContent;
    btn.textContent = t('models.testing');
    btn.disabled = true;
    try {
        const res = await fetch('/api/models/test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ model_index: idx }),
        });
        const data = await res.json();
        if (data.status === 'ok') {
            btn.textContent = '✓ OK';
            btn.classList.add('btn-success');
            showStatus(t('models.testSuccess', { model: data.model, response: data.response }), 'success');
        } else {
            btn.textContent = '✗ Fail';
            btn.classList.add('btn-danger');
            showStatus(t('models.testFail', { model: data.model || '', error: data.error }), 'error');
        }
    } catch (e) {
        btn.textContent = '✗ Fail';
        btn.classList.add('btn-danger');
        showStatus(t('models.testFail', { model: '', error: e.message }), 'error');
    }
    setTimeout(() => {
        btn.textContent = origText;
        btn.disabled = false;
        btn.classList.remove('btn-success', 'btn-danger');
    }, 3000);
}

// ── Model Modal ─────────────────────────────────────
const PROVIDER_INFO = {
    openai:               { label: 'OpenAI',             apiBase: 'https://api.openai.com/v1',      prefix: 'openai/' },
    anthropic:            { label: 'Anthropic',          apiBase: 'https://api.anthropic.com/v1',   prefix: 'anthropic/' },
    groq:                 { label: 'Groq',               apiBase: 'https://api.groq.com/openai/v1', prefix: 'groq/' },
    deepseek:             { label: 'DeepSeek',           apiBase: 'https://api.deepseek.com/v1',    prefix: 'deepseek/' },
    mistral:              { label: 'Mistral',            apiBase: 'https://api.mistral.ai/v1',      prefix: 'mistral/' },
    xai:                  { label: 'xAI',                apiBase: 'https://api.x.ai/v1',            prefix: 'xai/' },
    'google-antigravity': { label: 'Google Antigravity', apiBase: '',                                prefix: 'antigravity/' },
};

function detectProviderFromModel(model) {
    if (!model) return '';
    for (const [name, info] of Object.entries(PROVIDER_INFO)) {
        if (model.startsWith(info.prefix)) return name;
    }
    return '';
}

function getAllProviders() {
    const result = [];
    for (const [name, info] of Object.entries(PROVIDER_INFO)) {
        const auth = authProviderMap[name];
        const authenticated = auth && auth.status === 'active';
        result.push({ name, ...info, authenticated });
    }
    return result;
}

function isProviderAuthenticated(name) {
    const auth = authProviderMap[name];
    return auth && auth.status === 'active';
}

// Renders a <select> with all configured model_list entries.
// dataField: the data-field attribute name
// label: form label text
// currentValue: the currently selected model_name
// opts: { allowEmpty: bool, emptyLabel: string, hint: string }
function renderModelSelect(dataField, label, currentValue, opts) {
    opts = opts || {};
    const models = (configData && configData.model_list) || [];
    let html = '<div class="form-group">';
    html += `<label class="form-label">${esc(label)}</label>`;
    html += `<select class="form-input" data-field="${esc(dataField)}">`;
    if (opts.allowEmpty) {
        html += `<option value="">${esc(opts.emptyLabel || '')}</option>`;
    }
    models.forEach(m => {
        const name = m.model_name || '';
        const sel = name === currentValue ? ' selected' : '';
        const displayLabel = name + (m.model ? ' (' + m.model + ')' : '');
        html += `<option value="${esc(name)}"${sel}>${esc(displayLabel)}</option>`;
    });
    html += '</select>';
    if (opts.hint) html += `<div class="form-hint">${opts.hint}</div>`;
    html += '</div>';
    return html;
}

function onProviderSelectChange(selectEl) {
    const selected = selectEl.value;
    const info = PROVIDER_INFO[selected];
    const modelInput = document.querySelector('#modalBody input[data-field="model"]');

    if (selected && info && modelInput) {
        const current = modelInput.value;
        let bare = current;
        for (const pi of Object.values(PROVIDER_INFO)) {
            if (current.startsWith(pi.prefix)) { bare = current.slice(pi.prefix.length); break; }
        }
        modelInput.value = info.prefix + bare;
    }
}

const modelFieldsRequired = [
    { key: 'model_name', labelKey: 'field.modelName', type: 'text', placeholder: 'e.g. gpt-4o', required: true },
    { key: 'model', labelKey: 'field.modelId', type: 'text', placeholder: 'e.g. openai/gpt-4o', required: true, hintKey: 'field.modelIdHint' },
];
const modelFieldsOptional = [
    { key: 'thinking_level', labelKey: 'field.thinkingLevel', type: 'select', options: ['', 'off', 'low', 'medium', 'high', 'xhigh', 'adaptive'] },
    { key: 'max_tokens_field', labelKey: 'field.maxTokensField', type: 'text', placeholder: 'e.g. max_completion_tokens' },
    { key: 'rpm', labelKey: 'field.rpm', type: 'number', placeholder: 'RPM' },
    { key: 'request_timeout', labelKey: 'field.requestTimeout', type: 'number', placeholder: 'Seconds' },
];
const modelFields = [...modelFieldsRequired, ...modelFieldsOptional];

function showEditModelModal(idx) {
    editingModelIndex = idx;
    const m = configData.model_list[idx];
    document.getElementById('modalTitle').textContent = t('models.editModel') + ': ' + m.model_name;
    document.getElementById('modalSaveBtn').setAttribute('onclick', 'saveModelFromModal()');
    renderModalBody(m, m.provider || '');
    document.getElementById('modelModal').classList.add('active');
}

function showAddModelModal() {
    editingModelIndex = -1;
    document.getElementById('modalTitle').textContent = t('models.addModel');
    document.getElementById('modalSaveBtn').setAttribute('onclick', 'saveModelFromModal()');
    // Auto-select first provider if available
    const provList = (configData && configData.provider_list) || [];
    const defaultProv = provList.length > 0 ? provList[0].name : '';
    renderModalBody({}, defaultProv);
    document.getElementById('modelModal').classList.add('active');
}

function closeModelModal() {
    document.getElementById('modelModal').classList.remove('active');
}

function renderModalBody(data, preselectedProvider) {
    const providerList = (configData && configData.provider_list) || [];

    function renderField(f) {
        const val = data[f.key] !== undefined && data[f.key] !== null ? data[f.key] : '';
        const label = t(f.labelKey);
        let h = `<div class="form-group">`;
        h += `<label class="form-label">${label}${f.required ? ' *' : ''}</label>`;
        if (f.type === 'select' && f.options) {
            h += `<select class="form-input" data-field="${f.key}">`;
            f.options.forEach(opt => {
                const optLabel = opt === '' ? '(default)' : opt;
                h += `<option value="${esc(opt)}"${val === opt ? ' selected' : ''}>${esc(optLabel)}</option>`;
            });
            h += `</select>`;
        } else {
            h += `<input class="form-input ${f.type === 'number' ? 'form-input-number' : ''}" `;
            h += `type="${f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'}" `;
            h += `data-field="${f.key}" value="${esc(String(val))}" placeholder="${f.placeholder || ''}">`;
        }
        if (f.hintKey) h += `<div class="form-hint">${t(f.hintKey)}</div>`;
        h += `</div>`;
        return h;
    }

    let html = '';

    // Provider select (from provider_list)
    const currentProvider = data.provider || preselectedProvider || '';
    html += `<div class="form-group">`;
    html += `<label class="form-label">${t('field.provider')} *</label>`;
    html += `<select class="form-input" data-field="provider" onchange="onProviderSelectChange(this)">`;
    const availableProviders = providerList.filter(p => {
        const authInfo = authProviderMap[p.name];
        return p.api_key || (authInfo && authInfo.status === 'active') || (data.provider && p.name === data.provider);
    });
    if (availableProviders.length === 0) {
        html += `<option value="">No connected providers</option>`;
    }
    availableProviders.forEach(p => {
        html += `<option value="${esc(p.name)}"${currentProvider === p.name ? ' selected' : ''}>${esc(p.name)}</option>`;
    });
    html += `</select>`;
    html += `</div>`;

    modelFieldsRequired.forEach(f => { html += renderField(f); });

    html += `<div class="collapsible-header" onclick="this.classList.toggle('open');this.nextElementSibling.classList.toggle('open')">`;
    html += `<span class="arrow">&#9656;</span> ${t('models.advancedOptions')}`;
    html += `</div>`;
    html += `<div class="collapsible-body">`;
    modelFieldsOptional.forEach(f => { html += renderField(f); });
    html += `</div>`;

    document.getElementById('modalBody').innerHTML = html;
}

function saveModelFromModal() {
    const inputs = document.querySelectorAll('#modalBody input[data-field], #modalBody select[data-field]');
    const obj = {};
    inputs.forEach(input => {
        const key = input.dataset.field;
        let val = input.value.trim();
        if (input.type === 'number' && val) val = parseInt(val, 10) || 0;
        if (val !== '' && val !== 0) obj[key] = val;
        else if (key === 'model_name' || key === 'model' || key === 'provider') obj[key] = val;
    });

    // Auto-derive model_name from model ID (strip provider prefix)
    if (!obj.model_name && obj.model) {
        let name = obj.model;
        for (const pi of Object.values(PROVIDER_INFO)) {
            if (name.startsWith(pi.prefix)) { name = name.slice(pi.prefix.length); break; }
        }
        obj.model_name = name;
    }

    if (!obj.model_name || !obj.model) {
        showStatus(t('models.requiredFields'), 'error');
        return;
    }

    if (!configData.model_list) configData.model_list = [];

    if (editingModelIndex >= 0) {
        configData.model_list[editingModelIndex] = { ...configData.model_list[editingModelIndex], ...obj };
        modelFields.forEach(f => {
            if (!f.required && (obj[f.key] === '' || obj[f.key] === 0)) {
                delete configData.model_list[editingModelIndex][f.key];
            }
        });
    } else {
        configData.model_list.push(obj);
        // Auto-set as primary model
        if (!configData.agents) configData.agents = {};
        if (!configData.agents.defaults) configData.agents.defaults = {};
        configData.agents.defaults.model_name = obj.model_name;
    }

    closeModelModal();
    saveConfig().then(renderModels);
}



// ── Provider CRUD ───────────────────────────────────
function findProviderIndex(name) {
    const provList = (configData && configData.provider_list) || [];
    const idx = provList.findIndex(p => p.name === name);
    if (idx >= 0) return idx;
    // Auto-create if not found
    if (!configData.provider_list) configData.provider_list = [];
    configData.provider_list.push({ name });
    return configData.provider_list.length - 1;
}

function renderProviders() {
    const panel = document.getElementById('panelProviders');
    if (!panel || !configData) return;
    if (!configData.provider_list) configData.provider_list = [];
    const providers = configData.provider_list;

    let html = '<div class="panel-header"><div class="panel-title">Providers</div></div>';
    html += '<div class="panel-desc">Configure connectivity for LLM providers. Models reference a provider by name.</div>';
    html += '<div style="margin-bottom:14px;"><button class="btn btn-sm btn-primary" onclick="showAddProviderModal()">+ Add Provider</button></div>';

    if (providers.length === 0) {
        html += '<div class="empty-state"><div class="empty-state-title">No providers configured</div></div>';
    } else {
        html += '<div class="model-grid">';
        providers.forEach((p, idx) => {
            const authInfo = authProviderMap[p.name];
            const hasKey = !!(p.api_key || (authInfo && authInfo.status === 'active'));
            html += `<div class="model-card ${hasKey ? '' : 'unavailable'}">`;
            html += `<div class="model-card-head"><div class="model-name">${esc(p.name)}</div>`;
            if (hasKey) html += `<span class="badge-primary">Active</span>`;
            else html += `<span class="badge-nokey">No Key</span>`;
            html += `</div>`;
            if (p.api_base) html += `<div class="model-detail"><strong>API Base:</strong> ${esc(p.api_base)}</div>`;
            if (p.api_key) html += `<div class="model-detail"><strong>API Key:</strong> ${maskKey(p.api_key)}</div>`;
            if (p.auth_method) html += `<div class="model-detail"><strong>Auth:</strong> ${esc(p.auth_method)}</div>`;
            if (p.connect_mode) html += `<div class="model-detail"><strong>Connect Mode:</strong> ${esc(p.connect_mode)}</div>`;
            if (p.workspace) html += `<div class="model-detail"><strong>Workspace:</strong> ${esc(p.workspace)}</div>`;
            html += `<div class="model-actions">`;
            html += `<button class="btn btn-sm" onclick="showEditProviderModal(${idx})">Edit</button>`;
            const isBuiltin = p.name === 'openai' || p.name === 'groq';
            if (!isBuiltin) html += `<button class="btn btn-sm btn-danger" onclick="deleteProvider(${idx})">Delete</button>`;
            html += `</div></div>`;
        });
        html += '</div>';
    }
    panel.innerHTML = html;
}

const providerFields = [
    { key: 'name', label: 'Name', type: 'text', placeholder: 'e.g. openai', required: true },
    { key: 'api_base', label: 'API Base', type: 'text', placeholder: 'https://api.openai.com/v1' },
    { key: 'api_key', label: 'API Key', type: 'password', placeholder: 'API key' },
    { key: 'auth_method', label: 'Auth Method', type: 'text', placeholder: 'oauth / token' },
    { key: 'connect_mode', label: 'Connect Mode', type: 'text', placeholder: 'stdio / grpc' },
    { key: 'workspace', label: 'Workspace', type: 'text', placeholder: 'Workspace path' },
];

let editingProviderIndex = -1;

function showAddProviderModal() {
    editingProviderIndex = -1;
    document.getElementById('modalTitle').textContent = 'Add Provider';
    renderProviderModalBody({});
    document.getElementById('modelModal').classList.add('active');
}

function showEditProviderModal(idx) {
    editingProviderIndex = idx;
    const p = configData.provider_list[idx];
    document.getElementById('modalTitle').textContent = 'Edit Provider: ' + p.name;
    renderProviderModalBody(p);
    document.getElementById('modelModal').classList.add('active');
}

function renderProviderModalBody(data) {
    let html = '';
    providerFields.forEach(f => {
        const val = data[f.key] !== undefined && data[f.key] !== null ? data[f.key] : '';
        html += `<div class="form-group">`;
        html += `<label class="form-label">${f.label}${f.required ? ' *' : ''}</label>`;
        html += `<input class="form-input" type="${f.type === 'password' ? 'password' : 'text'}" `;
        html += `data-field="${f.key}" value="${esc(String(val))}" placeholder="${f.placeholder || ''}">`;
        html += `</div>`;
    });
    document.getElementById('modalBody').innerHTML = html;
    document.getElementById('modalSaveBtn').setAttribute('onclick', 'saveProviderFromModal()');
}

function saveProviderFromModal() {
    const inputs = document.querySelectorAll('#modalBody input[data-field]');
    const obj = {};
    inputs.forEach(input => {
        const key = input.dataset.field;
        const val = input.value.trim();
        if (val) obj[key] = val;
        else if (key === 'name') obj[key] = val;
    });
    if (!obj.name) {
        showStatus('Provider name is required', 'error');
        return;
    }
    if (!configData.provider_list) configData.provider_list = [];
    if (editingProviderIndex >= 0) {
        configData.provider_list[editingProviderIndex] = obj;
    } else {
        // Check duplicate
        if (configData.provider_list.some(p => p.name === obj.name)) {
            showStatus('Provider "' + obj.name + '" already exists', 'error');
            return;
        }
        configData.provider_list.push(obj);
    }
    editingProviderIndex = -1;
    closeModelModal();
    saveConfig().then(() => { renderProviders(); loadAuthStatus(); });
}

function deleteProvider(idx) {
    if (!configData || !configData.provider_list) return;
    const name = configData.provider_list[idx].name;
    if (name === 'openai') { showStatus('Cannot delete builtin provider', 'error'); return; }
    if (!confirm('Delete provider "' + name + '"?')) return;
    configData.provider_list.splice(idx, 1);
    saveConfig().then(() => { renderProviders(); loadAuthStatus(); });
}

// ── Channel Forms ───────────────────────────────────
function renderChannelForm(chKey) {
    const schema = channelSchemas[chKey];
    if (!schema) return;
    const panel = document.getElementById('panelCh_' + chKey);
    const chData = (configData && configData.channels && configData.channels[schema.configKey]) || {};

    let html = panelHeader(schema.title, 'panelCh_' + chKey);
    html += `<div class="panel-desc">${t('ch.configure', { name: schema.title })}`;
    html += `</div>`;

    // WhatsApp: side-by-side layout (form left, QR right)
    if (chKey === 'whatsapp') {
        html += `<div style="display:flex; flex-wrap:wrap; gap:24px 32px; align-items:flex-start;">`;
        html += `<div style="flex:1 1 300px; min-width:0;">`;
    }

    html += `<div class="channel-form form-grid" id="chForm_${chKey}">`;

    // Enabled toggle
    html += `<div class="toggle-row">`;
    html += `<div class="toggle ${chData.enabled ? 'on' : ''}" id="chToggle_${chKey}" onclick="toggleChannelEnabled('${chKey}', this)"></div>`;
    html += `<span class="toggle-label">${t('enabled')}</span>`;
    html += `</div>`;

    schema.fields.forEach(f => {
        const label = f.i18nLabel ? t(f.label) : f.label;
        if (f.type === 'toggle') {
            const hint = (f.i18nHint || f.i18nLabel) && f.hint ? t(f.hint) : (f.hint || '');
            html += `<div class="toggle-row">`;
            html += `<div class="toggle ${getNestedValue(chData, f.key) ? 'on' : ''}" data-chfield="${f.key}" onclick="this.classList.toggle('on')"></div>`;
            html += `<span class="toggle-label">${label}</span>`;
            if (hint) html += `<span class="form-hint" style="margin-left:8px;">${hint}</span>`;
            html += `</div>`;
        } else if (f.type === 'array') {
            const arr = getNestedValue(chData, f.key) || [];
            html += `<div class="form-group">`;
            html += `<label class="form-label">${label}</label>`;
            html += `<div class="array-editor" data-chfield="${f.key}" data-placeholder="${f.placeholder || ''}">`;
            arr.forEach(v => {
                html += `<div class="array-row">`;
                html += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="${f.placeholder || ''}">`;
                html += `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
                html += `</div>`;
            });
            html += `<div class="array-add" onclick="addArrayRow(this.parentElement)">${t('ch.addItem')}</div>`;
            html += `</div></div>`;
        } else {
            const rawVal = getNestedValue(chData, f.key);
            const val = rawVal !== undefined && rawVal !== null ? rawVal : '';
            html += `<div class="form-group">`;
            html += `<label class="form-label">${label}</label>`;
            html += `<input class="form-input ${f.type === 'number' ? 'form-input-number' : ''}" `;
            html += `type="${f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'}" `;
            html += `data-chfield="${f.key}" value="${esc(String(val))}" placeholder="${f.placeholder || ''}">`;
            if (f.hint) html += `<span class="form-hint">${f.hint}</span>`;
            html += `</div>`;
        }
    });

    // Access control — WhatsApp uses allowed_groups + allowed_contacts; others use allow_from
    html += `<div class="form-section-title">${t('ch.accessControl')}</div>`;
    if (chKey === 'whatsapp') {
        // Allowed Groups
        const allowedGroups = chData.allowed_groups || [];
        html += `<div class="form-group">`;
        html += `<label class="form-label">${t('ch.allowedGroups')}</label>`;
        html += `<div class="array-editor" data-chfield="allowed_groups" data-placeholder="Group JID (e.g. 120363012345678901)">`;
        allowedGroups.forEach(v => {
            html += `<div class="array-row">`;
            html += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="Group JID (e.g. 120363012345678901)">`;
            html += `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
            html += `</div>`;
        });
        html += `<div class="array-add" onclick="addArrayRow(this.parentElement)">${t('ch.addItem')}</div>`;
        html += `</div></div>`;

        // Allowed Contacts
        const allowedContacts = chData.allowed_contacts || [];
        html += `<div class="form-group">`;
        html += `<label class="form-label">${t('ch.allowedContacts')}</label>`;
        html += `<div class="array-editor" data-chfield="allowed_contacts" data-placeholder="Phone number (e.g. 5511999999999)">`;
        allowedContacts.forEach(v => {
            html += `<div class="array-row">`;
            html += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="Phone number (e.g. 5511999999999)">`;
            html += `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
            html += `</div>`;
        });
        html += `<div class="array-add" onclick="addArrayRow(this.parentElement)">${t('ch.addItem')}</div>`;
        html += `</div></div>`;
    } else {
        const allowFrom = chData.allow_from || [];
        html += `<div class="form-group">`;
        html += `<label class="form-label">${t('ch.allowFrom')}</label>`;
        html += `<div class="array-editor" data-chfield="allow_from" data-placeholder="User / Chat ID">`;
        allowFrom.forEach(v => {
            html += `<div class="array-row">`;
            html += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="User / Chat ID">`;
            html += `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
            html += `</div>`;
        });
        html += `<div class="array-add" onclick="addArrayRow(this.parentElement)">${t('ch.addItem')}</div>`;
        html += `</div></div>`;
    }

    html += `<div style="margin-top:20px;">`;
    html += `<button class="btn btn-primary" onclick="saveChannelForm('${chKey}')">${t('save')}</button>`;
    html += `</div></div>`;

    // WhatsApp: close left column, add QR right column, close flex container
    if (chKey === 'whatsapp') {
        html += `</div>`; // close left column
        html += `<div id="waQrSection" style="flex:0 1 280px; min-width:220px; text-align:center; padding-top:8px;">`;
        html += `<div style="font-weight:600; margin-bottom:12px;">WhatsApp Pairing</div>`;
        html += `<div id="waQrCode" style="margin-bottom:8px; max-width:100%; overflow:hidden;"></div>`;
        html += `<div id="waQrStatus" style="color:var(--muted); font-size:13px;">Start the gateway to pair</div>`;
        html += `</div>`;
        html += `</div>`; // close flex container
    }

    panel.innerHTML = html;

    // Wire paste handlers for multi-value paste on existing array inputs
    panel.querySelectorAll('.array-editor').forEach(container => {
        container.querySelectorAll('.array-row input').forEach(input => {
            input.addEventListener('paste', function(e) { handleArrayPaste(e, container); });
        });
    });

    if (chKey === 'whatsapp') connectWhatsAppQR();
}

function toggleChannelEnabled(chKey, el) {
    el.classList.toggle('on');
    if (chKey === 'chat') updateChatVisibility(el.classList.contains('on'));
}

function updateChatVisibility(enabled) {
    const btn = document.querySelector('.rail-btn[data-tab="chat"]');
    if (btn) btn.style.display = enabled ? '' : 'none';
    // If chat panel is active and just disabled, switch away
    if (!enabled) {
        const chatPanel = document.getElementById('panelChat');
        if (chatPanel && chatPanel.classList.contains('active')) {
            switchTab('input');
        }
    }
}

function addArrayRow(container) {
    const placeholder = container.dataset.placeholder || '';
    const addBtn = container.querySelector('.array-add');
    const row = document.createElement('div');
    row.className = 'array-row';
    row.innerHTML = `<input class="form-input" type="text" value="" placeholder="${placeholder}">` +
        `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
    container.insertBefore(row, addBtn);
    const input = row.querySelector('input');
    input.addEventListener('paste', function(e) { handleArrayPaste(e, container); });
    input.focus();
}

function handleArrayPaste(e, container) {
    const text = (e.clipboardData || window.clipboardData).getData('text');
    const values = text.split(/[,;\n\r]+/).map(v => v.trim()).filter(v => v);
    if (values.length <= 1) return; // single value, let default paste handle it
    e.preventDefault();
    const placeholder = container.dataset.placeholder || '';
    const addBtn = container.querySelector('.array-add');
    values.forEach(val => {
        const row = document.createElement('div');
        row.className = 'array-row';
        row.innerHTML = `<input class="form-input" type="text" value="${esc(val)}" placeholder="${placeholder}">` +
            `<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>`;
        container.insertBefore(row, addBtn);
        row.querySelector('input').addEventListener('paste', function(ev) { handleArrayPaste(ev, container); });
    });
    // Remove the empty row that triggered the paste if it's still empty
    const triggerRow = e.target.closest('.array-row');
    if (triggerRow && !e.target.value.trim()) triggerRow.remove();
}

function removeArrayRow(btn) { btn.parentElement.remove(); }

function saveChannelForm(chKey) {
    const schema = channelSchemas[chKey];
    if (!schema || !configData) return;

    if (!configData.channels) configData.channels = {};
    const chObj = configData.channels[schema.configKey] || {};

    const toggle = document.getElementById('chToggle_' + chKey);
    chObj.enabled = toggle ? toggle.classList.contains('on') : false;

    const form = document.getElementById('chForm_' + chKey);
    schema.fields.forEach(f => {
        if (f.type === 'toggle') {
            const el = form.querySelector(`[data-chfield="${f.key}"].toggle`);
            if (el) setNestedValue(chObj, f.key, el.classList.contains('on'));
        } else if (f.type === 'array') {
            const container = form.querySelector(`.array-editor[data-chfield="${f.key}"]`);
            if (container) {
                const vals = [];
                container.querySelectorAll('.array-row input').forEach(input => {
                    const v = input.value.trim();
                    if (v) vals.push(v);
                });
                setNestedValue(chObj, f.key, vals);
            }
        } else {
            const input = form.querySelector(`input[data-chfield="${f.key}"]`);
            if (input) {
                let val = input.value.trim();
                if (f.type === 'number' && val) {
                    val = parseInt(val, 10);
                    if (isNaN(val)) val = 0;
                }
                setNestedValue(chObj, f.key, val === '' ? (f.type === 'number' ? 0 : '') : val);
            }
        }
    });

    // Collect all array-editor fields not in schema (allow_from, allowed_groups, allowed_contacts)
    form.querySelectorAll('.array-editor[data-chfield]').forEach(container => {
        const field = container.dataset.chfield;
        // Skip fields already handled by schema
        if (schema.fields.some(f => f.key === field && f.type === 'array')) return;
        const vals = [];
        container.querySelectorAll('.array-row input').forEach(input => {
            const v = input.value.trim();
            if (v) vals.push(v);
        });
        chObj[field] = vals;
    });

    configData.channels[schema.configKey] = chObj;
    saveConfig().then(() => showStatus(t('status.saved', { name: schema.title }), 'success'));
}

// ── Auth API ────────────────────────────────────────
let authProviderMap = {}; // { 'openai': { status: 'active', ... }, ... }

async function loadAuthStatus() {
    try {
        const res = await fetch('/api/auth/status');
        if (!res.ok) return;
        const data = await res.json();
        const providers = data.providers || [];
        authProviderMap = {};
        providers.forEach(p => { authProviderMap[p.provider] = p; });
        renderAuthStatus(providers, data.pending_device);
    } catch (e) {
        console.error('Failed to load auth status:', e);
    }
}

function renderAuthStatus(providersList, pendingDevice) {
    const providerMap = {};
    providersList.forEach(p => { providerMap[p.provider] = p; });


    ['openai', 'anthropic', 'google-antigravity', 'groq', 'deepseek', 'mistral', 'xai'].forEach(name => {
        const badge = document.getElementById('badge-' + name);
        const details = document.getElementById('details-' + name);
        const actions = document.getElementById('actions-' + name);
        if (!badge || !details || !actions) return;
        const p = providerMap[name];

        // Resolve provider config from provider_list
        const provList = (configData && configData.provider_list) || [];
        const provCfg = provList.find(pc => pc.name === name);

        if (p) {
            const badgeClass = p.status === 'active' ? 'badge-active' :
                p.status === 'expired' ? 'badge-expired' : 'badge-pending';
            const badgeText = p.status === 'active' ? t('auth.active') :
                p.status === 'expired' ? t('auth.expired') : t('auth.needsRefresh');
            badge.className = 'provider-badge ' + badgeClass;
            badge.textContent = badgeText;

            let dh = '';
            if (p.auth_method) dh += `<div class="provider-detail"><strong>${t('auth.method')}:</strong> ${p.auth_method}</div>`;
            if (p.email) dh += `<div class="provider-detail"><strong>${t('auth.email')}:</strong> ${p.email}</div>`;
            if (p.account_id) dh += `<div class="provider-detail"><strong>${t('auth.account')}:</strong> ${p.account_id}</div>`;
            if (p.project_id) dh += `<div class="provider-detail"><strong>${t('auth.project')}:</strong> ${p.project_id}</div>`;
            if (p.expires_at) {
                const d = new Date(p.expires_at);
                dh += `<div class="provider-detail"><strong>${t('auth.expires')}:</strong> ${d.toLocaleString()}</div>`;
            }
            // Show provider config fields from provider_list
            if (provCfg) {
                if (provCfg.api_base) dh += `<div class="provider-detail"><strong>API Base:</strong> ${provCfg.api_base}</div>`;
                if (provCfg.api_key) dh += `<div class="provider-detail"><strong>API Key:</strong> ${maskKey(provCfg.api_key)}</div>`;
                if (provCfg.connect_mode) dh += `<div class="provider-detail"><strong>Connect Mode:</strong> ${provCfg.connect_mode}</div>`;
                if (provCfg.workspace) dh += `<div class="provider-detail"><strong>Workspace:</strong> ${provCfg.workspace}</div>`;
            }
            details.innerHTML = dh;
            actions.innerHTML = `<button class="btn btn-sm" onclick="showEditProviderModal(findProviderIndex('${name}'))">Edit</button> <button class="btn btn-sm btn-danger" onclick="logoutProvider('${name}')">${t('auth.logout')}</button>`;
        } else {
            badge.className = 'provider-badge badge-none';
            badge.textContent = t('auth.notLoggedIn');
            // Still show provider config if it exists
            let dh = '';
            if (provCfg) {
                if (provCfg.api_base) dh += `<div class="provider-detail"><strong>API Base:</strong> ${provCfg.api_base}</div>`;
                if (provCfg.api_key) dh += `<div class="provider-detail"><strong>API Key:</strong> ${maskKey(provCfg.api_key)}</div>`;
            }
            details.innerHTML = dh;
            if (name === 'openai') {
                actions.innerHTML = `<button class="btn btn-sm" onclick="showEditProviderModal(findProviderIndex('${name}'))">Edit</button> <button class="btn btn-sm btn-primary" onclick="loginProvider('openai')">${t('auth.loginDevice')}</button> <button class="btn btn-sm" onclick="showTokenInput('openai')">${t('auth.loginToken')}</button>`;
            } else if (name === 'anthropic') {
                actions.innerHTML = `<button class="btn btn-sm" onclick="showEditProviderModal(findProviderIndex('${name}'))">Edit</button> <button class="btn btn-sm btn-primary" onclick="showTokenInput('anthropic')">${t('auth.loginToken')}</button>`;
            } else if (name === 'google-antigravity') {
                actions.innerHTML = `<button class="btn btn-sm" onclick="showEditProviderModal(findProviderIndex('${name}'))">Edit</button> <button class="btn btn-sm btn-primary" onclick="loginProvider('google-antigravity')">${t('auth.loginOAuth')}</button>`;
            } else {
                actions.innerHTML = `<button class="btn btn-sm" onclick="showEditProviderModal(findProviderIndex('${name}'))">Edit</button> <button class="btn btn-sm btn-primary" onclick="showTokenInput('${name}')">${t('auth.loginToken')}</button>`;
            }
        }
    });


    if (pendingDevice && pendingDevice.status === 'pending') {
        const name = pendingDevice.provider;
        const badge = document.getElementById('badge-' + name);
        const details = document.getElementById('details-' + name);
        const actions = document.getElementById('actions-' + name);
        if (badge) { badge.className = 'provider-badge badge-pending'; badge.textContent = t('auth.authenticating'); }
        if (pendingDevice.device_url && pendingDevice.user_code && details) {
            details.innerHTML = `
            <div class="device-code-box">
              <div class="hint" style="margin-bottom:8px;font-size:12px;">${t('auth.step1')}</div>
              <div class="url"><a href="${pendingDevice.device_url}" target="_blank">${pendingDevice.device_url} &#8599;</a></div>
              <div class="hint" style="margin-top:10px;margin-bottom:4px;font-size:12px;">${t('auth.step2')}</div>
              <div class="code">${pendingDevice.user_code}</div>
              <div class="hint">${t('auth.step3')}</div>
            </div>`;
        }
        if (pendingDevice.error && details) {
            details.innerHTML = `<div class="provider-detail" style="color:var(--error)">${pendingDevice.error}</div>`;
            if (actions) actions.innerHTML = `<button class="btn btn-sm btn-primary" onclick="loginProvider('${name}')">${t('auth.retry')}</button>`;
        } else if (actions) {
            actions.innerHTML = `<button class="btn btn-sm" disabled><span class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle;margin-right:6px;"></span>${t('auth.waiting')}</button>`;
        }
        startAuthPolling();
    } else {
        stopAuthPolling();
    }
}

function startAuthPolling() {
    stopAuthPolling();
    authPollTimer = setInterval(() => loadAuthStatus(), 3000);
}

function stopAuthPolling() {
    if (authPollTimer) { clearInterval(authPollTimer); authPollTimer = null; }
}

async function loginProvider(provider) {
    const actions = document.getElementById('actions-' + provider);
    const original = actions ? actions.innerHTML : '';
    if (actions) actions.querySelectorAll('.btn').forEach(b => { b.disabled = true; b.style.opacity = '0.5'; });

    try {
        const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ provider }),
        });
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json();

        if (data.status === 'redirect' && data.auth_url) {
            showStatus(t('status.openingBrowser'), 'success');
            window.open(data.auth_url, '_blank');
            if (actions) actions.innerHTML = original;
            return;
        }
        if (data.status === 'pending') {
            showStatus(data.message || t('status.loginStarted'), 'success');
            if (data.device_url && data.user_code) {
                const badge = document.getElementById('badge-' + provider);
                const details = document.getElementById('details-' + provider);
                if (badge) { badge.className = 'provider-badge badge-pending'; badge.textContent = t('auth.authenticating'); }
                if (details) {
                    details.innerHTML = `
                    <div class="device-code-box">
                      <div class="hint" style="margin-bottom:8px;font-size:12px;">${t('auth.step1')}</div>
                      <div class="url"><a href="${data.device_url}" target="_blank">${data.device_url} &#8599;</a></div>
                      <div class="hint" style="margin-top:10px;margin-bottom:4px;font-size:12px;">${t('auth.step2')}</div>
                      <div class="code">${data.user_code}</div>
                      <div class="hint">${t('auth.step3')}</div>
                    </div>`;
                }
                if (actions) {
                    actions.innerHTML = `<button class="btn btn-sm" disabled><span class="spinner" style="width:14px;height:14px;border-width:2px;display:inline-block;vertical-align:middle;margin-right:6px;"></span>${t('auth.waiting')}</button>`;
                }
            }
            startAuthPolling();
        } else if (data.status === 'success') {
            showStatus(data.message || t('status.loginSuccess'), 'success');
            loadAuthStatus();
        }
    } catch (e) {
        showStatus(t('status.loginFailed') + ': ' + e.message, 'error');
        if (actions) actions.innerHTML = original;
    }
}

function showTokenInput(provider) {
    const actions = document.getElementById('actions-' + provider);
    actions.innerHTML = `
    <div class="token-input-group">
      <input type="password" id="tokenInput-${provider}" placeholder="${t('auth.pasteKey')}" />
      <button class="btn btn-sm btn-primary" onclick="submitToken('${provider}')">${t('save')}</button>
      <button class="btn btn-sm" onclick="loadAuthStatus()">${t('cancel')}</button>
    </div>`;
    document.getElementById('tokenInput-' + provider).focus();
}

async function submitToken(provider) {
    const input = document.getElementById('tokenInput-' + provider);
    const token = input.value.trim();
    if (!token) { showStatus(t('status.tokenEmpty'), 'error'); return; }
    try {
        const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ provider, token }),
        });
        if (!res.ok) throw new Error(await res.text());
        showStatus(t('status.tokenSaved', { name: provider }), 'success');
        loadAuthStatus();
    } catch (e) {
        showStatus(t('status.loginFailed') + ': ' + e.message, 'error');
    }
}


async function logoutProvider(provider) {
    try {
        const res = await fetch('/api/auth/logout', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ provider }),
        });
        if (!res.ok) throw new Error(await res.text());
        showStatus(t('status.loggedOut', { name: provider }), 'success');
        loadAuthStatus();
    } catch (e) {
        showStatus(t('status.logoutFailed') + ': ' + e.message, 'error');
    }
}

// ── Gateway API (Status/Reload) ─────────────────
// gatewayRunning declared at top of file (always true in embedded mode)

function isModelAvailableGlobal(m) {
    // Resolve provider from model's provider field or protocol prefix
    const provName = m.provider || (m.model ? m.model.split('/')[0] : '');
    const providers = (configData && configData.provider_list) || [];
    const prov = providers.find(p => p.name === provName);

    // Check provider-level API key
    if (prov && prov.api_key) return true;

    // Check provider-level auth method
    const authMethod = prov ? prov.auth_method : '';
    if (authMethod === 'oauth') {
        const authInfo = authProviderMap[provName];
        return !!(authInfo && authInfo.status === 'active');
    }
    if (authMethod) return true;

    // Fallback: check auth store for the protocol
    if (provName) {
        const authInfo = authProviderMap[provName];
        if (authInfo && authInfo.status === 'active') return true;
    }
    return false;
}

function checkStartPrereqs() {
    // Has at least one model configured (with key or auth method)
    const hasModel = !!(configData && configData.model_list &&
        configData.model_list.some(m => isModelAvailableGlobal(m)));
    // Has a model set as default (for onboarding display, not required to start)
    const primaryModelName = configData && configData.agents && configData.agents.defaults
        ? (configData.agents.defaults.model_name || '') : '';
    const hasPrimary = !!(primaryModelName && configData.model_list &&
        configData.model_list.some(m => m.model_name === primaryModelName));
    const hasChannel = configData && configData.channels &&
        Object.keys(channelSchemas).some(k => {
            const cfg = configData.channels[channelSchemas[k].configKey];
            return cfg && cfg.enabled;
        });
    // Can start with just a model — channels are optional (can use web chat or API)
    return { hasModel, hasPrimary, hasChannel, canStart: hasModel };
}

function updateStatusIndicator(state) {
    const dot = document.getElementById('statusDot');
    const text = document.getElementById('statusText');
    if (!dot || !text) return;
    dot.className = 'status-dot ' + (state || 'setup');
    const labels = { setup: t('status.setup'), ready: t('status.ready'), running: t('status.running') };
    text.textContent = labels[state] || labels.setup;
}

function updateOnboarding() {
    const el = document.getElementById('onboarding');
    if (!el) return;
    // Respect manual dismiss
    if (sessionStorage.getItem('ok-onboarding-dismissed')) { el.classList.add('hidden'); return; }
    const prereqs = checkStartPrereqs();
    // Hide if gateway is running or everything is set up
    if (gatewayRunning || (prereqs.hasModel && prereqs.hasChannel)) { el.classList.add('hidden'); return; }
    el.classList.remove('hidden');
    const steps = el.querySelectorAll('.onboarding-step');
    if (steps[1]) {
        steps[1].className = prereqs.hasModel ? 'onboarding-step done' : 'onboarding-step current';
        steps[1].querySelector('.step-check').textContent = prereqs.hasModel ? '\u2713' : '2';
    }
    if (steps[2]) {
        steps[2].className = prereqs.hasChannel ? 'onboarding-step done' : (prereqs.hasModel ? 'onboarding-step current' : 'onboarding-step');
        steps[2].querySelector('.step-check').textContent = prereqs.hasChannel ? '\u2713' : '3';
    }
    if (steps[3]) {
        steps[3].className = prereqs.canStart ? 'onboarding-step current' : 'onboarding-step';
    }
}

function dismissOnboarding() {
    const el = document.getElementById('onboarding');
    if (el) el.classList.add('hidden');
    sessionStorage.setItem('ok-onboarding-dismissed', '1');
}

function updateRunStopButton(running) {
    gatewayRunning = true; // Always running in embedded mode
    const btn = document.getElementById('btnRunStop');
    const icon = document.getElementById('btnRunStopIcon');
    const text = document.getElementById('btnRunStopText');
    const hint = document.getElementById('processHint');
    btn.disabled = false;
    btn.className = 'btn btn-run btn-process';
    icon.innerHTML = '&#8635;';
    text.textContent = 'Reload';
    hint.textContent = '';
    updateStatusIndicator('running');
}

function setButtonLoading(actionType) {
    const btn = document.getElementById('btnRunStop');
    const icon = document.getElementById('btnRunStopIcon');
    const text = document.getElementById('btnRunStopText');
    const hint = document.getElementById('processHint');
    btn.disabled = true;
    btn.className = 'btn btn-process';
    icon.innerHTML = '<span class="btn-spinner"></span>';
    text.textContent = 'Reloading...';
    hint.textContent = '';
}

async function checkGatewayStatus() {
    try {
        const res = await fetch('/api/gateway/status');
        if (res.ok) {
            const data = await res.json();
            updateRunStopButton(true);
        }
    } catch (e) { /* gateway is always running in embedded mode */ }
}

async function reloadConfig() {
    setButtonLoading('reload');
    try {
        const res = await fetch('/api/gateway/reload', { method: 'POST' });
        if (!res.ok) throw new Error(await res.text());
        showStatus(t('status.reloadTriggered'), 'success');
    } catch (e) {
        showStatus('Reload failed: ' + e.message, 'error');
    }
    setTimeout(() => updateRunStopButton(true), 1500);
}

document.getElementById('btnRunStop').addEventListener('click', () => {
    reloadConfig();
});

// Poll status every 10 seconds (just for health)
checkGatewayStatus();
setInterval(checkGatewayStatus, 10000);

// ── Log Panel ───────────────────────────────────────
let logLastTimestamp = '';
let logAutoScrollEnabled = true;
let logHasContent = false;
let logEntries = [];
let debugEnabled = false;
let logCurrentComponent = 'all';

// Log polling via file-based endpoints (gateway runs in-process)
async function pollLogTail() {
    if (logCurrentComponent !== 'all') return; // specific component uses loadComponentFromFile
    try {
        const res = await fetch('/api/logs/tail?component=all&lines=500');
        if (!res.ok) return;
        const data = await res.json();
        const logs = data.logs || [];

        const tableBody = document.getElementById('logTableBody');
        const placeholder = document.getElementById('logPlaceholder');
        if (!tableBody) return;

        // Filter out entries we've already seen (by timestamp)
        const newLogs = logLastTimestamp
            ? logs.filter(e => (e.timestamp || e.ts || '') > logLastTimestamp)
            : logs;

        if (newLogs.length > 0) {
            if (placeholder) placeholder.style.display = 'none';
            logHasContent = true;

            const levelFilter = document.getElementById('logLevelFilter');
            const searchFilter = document.getElementById('logSearchFilter');
            const filterLevel = levelFilter ? levelFilter.value : '';
            const filterText = searchFilter ? searchFilter.value.toLowerCase() : '';

            const frag = document.createDocumentFragment();
            newLogs.forEach(entry => {
                logEntries.push(entry);
                if (logEntries.length > 5000) logEntries.shift();
                const tr = createLogRow(entry);
                if (shouldHideRow(entry, filterLevel, filterText)) {
                    tr.classList.add('log-row-hidden');
                }
                frag.appendChild(tr);
            });
            tableBody.appendChild(frag);

            if (logAutoScrollEnabled) {
                const wrap = document.getElementById('logTableWrap');
                if (wrap) wrap.scrollTop = wrap.scrollHeight;
            }

            const last = newLogs[newLogs.length - 1];
            logLastTimestamp = last.timestamp || last.ts || logLastTimestamp;
        }

        if (!logHasContent && placeholder) {
            placeholder.textContent = t('logs.noLogs');
            placeholder.style.display = '';
        }
    } catch (e) { /* ignore */ }
}

// Poll logs every 3 seconds, but only when the log panel is visible
function startLogPolling() {
    if (logPollTimer) return;
    pollLogTail();
    logPollTimer = setInterval(pollLogTail, 3000);
}

function stopLogPolling() {
    if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null; }
}

function createLogRow(entry) {
    const tr = document.createElement('tr');
    // Time
    const tdTime = document.createElement('td');
    tdTime.className = 'log-col-time';
    const ts = entry.ts || entry.timestamp || '';
    // Show date + time (YYYY-MM-DD HH:MM:SS)
    let timePart = ts;
    if (ts.includes('T')) {
        const [datePart, rest] = ts.split('T');
        timePart = datePart + ' ' + rest.replace('Z', '').substring(0, 8);
    }
    tdTime.textContent = timePart;
    tdTime.title = ts;
    tr.appendChild(tdTime);

    // Level
    const tdLevel = document.createElement('td');
    tdLevel.className = 'log-col-level';
    const level = entry.level || 'OUTPUT';
    tdLevel.innerHTML = '<span class="log-level log-level-' + esc(level) + '">' + esc(level) + '</span>';
    tr.appendChild(tdLevel);

    // Component
    const tdComp = document.createElement('td');
    tdComp.className = 'log-col-component';
    tdComp.textContent = entry.component || '\u2014';
    tr.appendChild(tdComp);

    // Message
    const tdMsg = document.createElement('td');
    tdMsg.className = 'log-col-message';
    tdMsg.textContent = entry.message || '';
    tr.appendChild(tdMsg);

    // Data
    const tdData = document.createElement('td');
    tdData.className = 'log-col-data';
    if (entry.fields && Object.keys(entry.fields).length > 0) {
        const formatter = new JSONFormatter(entry.fields, Infinity, { hoverPreviewEnabled: false, theme: '' });
        tdData.appendChild(formatter.render());
    }
    tr.appendChild(tdData);

    return tr;
}

function shouldHideRow(entry, filterLevel, filterText) {
    // Component filter (when viewing real-time "all" stream)
    if (logCurrentComponent !== 'all') {
        const comp = (entry.component || 'general').replace(/\./g, '_');
        if (comp !== logCurrentComponent) return true;
    }
    if (filterLevel && entry.level !== filterLevel) return true;
    if (filterText) {
        const haystack = ((entry.component || '') + ' ' + (entry.message || '')).toLowerCase();
        if (!haystack.includes(filterText)) return true;
    }
    return false;
}

function clearLogDisplay() {
    const tableBody = document.getElementById('logTableBody');
    const placeholder = document.getElementById('logPlaceholder');
    if (tableBody) tableBody.innerHTML = '';
    logEntries = [];
    logHasContent = false;
    logLastTimestamp = '';
    if (placeholder) {
        placeholder.textContent = t('logs.noLogs');
        placeholder.style.display = '';
    }
}

async function deleteAllLogs() {
    if (!confirm(t('logs.deleteAllConfirm'))) return;
    try {
        const res = await fetch('/api/logs', { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
        clearLogDisplay();
        loadLogComponents();
        showStatus(t('logs.allDeleted'), 'success');
    } catch (e) {
        showStatus('Failed to delete logs: ' + e.message, 'error');
    }
}

function applyLogFilters() {
    const tableBody = document.getElementById('logTableBody');
    if (!tableBody) return;
    const levelFilter = document.getElementById('logLevelFilter');
    const searchFilter = document.getElementById('logSearchFilter');
    const filterLevel = levelFilter ? levelFilter.value : '';
    const filterText = searchFilter ? searchFilter.value.toLowerCase() : '';

    const rows = tableBody.querySelectorAll('tr');
    rows.forEach((tr, i) => {
        if (i < logEntries.length) {
            tr.classList.toggle('log-row-hidden', shouldHideRow(logEntries[i], filterLevel, filterText));
        }
    });
}

async function loadLogComponents() {
    const select = document.getElementById('logComponentFilter');
    if (!select) return;

    // Collect components from file-based logs
    let fileComponents = [];
    try {
        const res = await fetch('/api/logs/components');
        if (res.ok) {
            const data = await res.json();
            fileComponents = data.components || [];
        }
    } catch (e) { /* ignore */ }

    // Also collect components from live real-time entries
    const liveSet = new Set();
    logEntries.forEach(entry => {
        const comp = (entry.component || 'general').replace(/\./g, '_');
        liveSet.add(comp);
    });

    // Merge: file components + live components, deduplicated
    const allSet = new Set(fileComponents.filter(c => c !== 'all'));
    liveSet.forEach(c => allSet.add(c));

    const sorted = Array.from(allSet).sort();
    const components = ['all', ...sorted];

    if (components.length <= 1 && liveSet.size === 0) return;

    // Preserve current selection
    const prev = select.value;
    select.innerHTML = '';
    components.forEach(c => {
        const opt = document.createElement('option');
        opt.value = c;
        opt.textContent = c === 'all' ? t('logs.allComponents') : c;
        select.appendChild(opt);
    });
    // Restore selection if still valid
    if (components.includes(prev)) {
        select.value = prev;
    } else {
        select.value = 'all';
        logCurrentComponent = 'all';
    }
}

async function switchLogComponent() {
    const select = document.getElementById('logComponentFilter');
    if (!select) return;
    logCurrentComponent = select.value;

    // If a specific component is selected, load from file
    if (logCurrentComponent !== 'all') {
        await loadComponentFromFile(logCurrentComponent);
    } else {
        // Re-apply filters to show/hide rows in the existing real-time stream
        applyLogFilters();
    }
}

async function loadComponentFromFile(component) {
    const tableBody = document.getElementById('logTableBody');
    const placeholder = document.getElementById('logPlaceholder');
    if (!tableBody) return;

    try {
        const res = await fetch('/api/logs/tail?component=' + encodeURIComponent(component) + '&lines=500');
        if (!res.ok) return;
        const data = await res.json();
        const logs = data.logs || [];

        // Clear and rebuild display with file-based data
        tableBody.innerHTML = '';
        logEntries = [];
        logHasContent = false;

        if (logs.length === 0) {
            if (placeholder) {
                placeholder.textContent = t('logs.noLogs');
                placeholder.style.display = '';
            }
            return;
        }

        if (placeholder) placeholder.style.display = 'none';
        logHasContent = true;

        const levelFilter = document.getElementById('logLevelFilter');
        const searchFilter = document.getElementById('logSearchFilter');
        const filterLevel = levelFilter ? levelFilter.value : '';
        const filterText = searchFilter ? searchFilter.value.toLowerCase() : '';

        const frag = document.createDocumentFragment();
        logs.forEach(entry => {
            logEntries.push(entry);
            const tr = createLogRow(entry);
            if (shouldHideRow(entry, filterLevel, filterText)) {
                tr.classList.add('log-row-hidden');
            }
            frag.appendChild(tr);
        });
        tableBody.appendChild(frag);

        if (logAutoScrollEnabled) {
            const wrap = document.getElementById('logTableWrap');
            if (wrap) wrap.scrollTop = wrap.scrollHeight;
        }
    } catch (e) { /* ignore */ }
}

function toggleDebug() {
    if (!configData) return;
    configData.debug = !configData.debug;
    debugEnabled = configData.debug;
    const toggle = document.getElementById('debugToggle');
    if (toggle) toggle.classList.toggle('on', debugEnabled);
    saveConfig().then(() => {
        showStatus(debugEnabled ? t('logs.debugOn') : t('logs.debugOff'), 'success');
    });
}

// ── Utilities ───────────────────────────────────────
function esc(s) {
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

function escAttr(s) {
    return String(s).replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"');
}

function maskKey(key) {
    if (!key || key.length < 8) return '****';
    return key.substring(0, 4) + '...' + key.substring(key.length - 4);
}

// ── Chat ────────────────────────────────────────────
let chatWs = null;
let chatMsgCounter = 0;
let chatTypingEl = null;
const chatBotMessages = {};

function chatGetGatewayUrl() {
    if (configData && configData.gateway) {
        const host = configData.gateway.host === '0.0.0.0' ? window.location.hostname : (configData.gateway.host || '127.0.0.1');
        const port = configData.gateway.port || 18790;
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return proto + '//' + host + ':' + port + '/chat/ws';
    }
    return null;
}

function chatConnect() {
    const url = chatGetGatewayUrl();
    const dot = document.getElementById('chatStatusDot');
    const text = document.getElementById('chatStatusText');
    const input = document.getElementById('chatInput');
    const send = document.getElementById('chatSend');

    if (!url || !gatewayRunning) {
        dot.className = 'chat-status-dot';
        text.textContent = 'Offline';
        input.disabled = true;
        send.disabled = true;
        return;
    }

    dot.className = 'chat-status-dot';
    text.textContent = 'Connecting...';

    chatWs = new WebSocket(url);

    chatWs.onopen = () => {
        dot.className = 'chat-status-dot connected';
        text.textContent = 'Connected';
        const msgs = document.getElementById('chatMessages');
        if (msgs) msgs.innerHTML = '';
        input.disabled = false;
        send.disabled = false;
        input.focus();
    };

    chatWs.onclose = () => {
        dot.className = 'chat-status-dot error';
        text.textContent = 'Disconnected';
        input.disabled = true;
        send.disabled = true;
        chatWs = null;
        // Only reconnect if chat panel is active
        const chatPanel = document.getElementById('panelChat');
        if (gatewayRunning && chatPanel && chatPanel.classList.contains('active')) {
            setTimeout(chatConnect, 2000);
        }
    };

    chatWs.onerror = () => {
        dot.className = 'chat-status-dot error';
        text.textContent = 'Connection error';
    };

    chatWs.onmessage = (e) => {
        try { chatHandleMessage(JSON.parse(e.data)); } catch (_) {}
    };
}

function chatDisconnect() {
    if (chatWs) { chatWs.close(); chatWs = null; }
    const dot = document.getElementById('chatStatusDot');
    const text = document.getElementById('chatStatusText');
    if (dot) dot.className = 'chat-status-dot';
    if (text) text.textContent = 'Offline';
    const input = document.getElementById('chatInput');
    const send = document.getElementById('chatSend');
    if (input) input.disabled = true;
    if (send) send.disabled = true;
}

function chatHandleMessage(msg) {
    switch (msg.type) {
        case 'message.create': {
            chatRemoveTyping();
            const content = msg.payload?.content || '';
            const msgId = msg.payload?.message_id || '';
            const role = msg.payload?.role || 'bot';
            const el = chatAddMessage(content, role);
            if (msgId) {
                chatBotMessages[msgId] = el;
                const keys = Object.keys(chatBotMessages);
                if (keys.length > 200) {
                    keys.slice(0, keys.length - 100).forEach(k => delete chatBotMessages[k]);
                }
            }
            break;
        }
        case 'message.update': {
            const content = msg.payload?.content || '';
            const msgId = msg.payload?.message_id || '';
            if (msgId && chatBotMessages[msgId]) {
                const textEl = chatBotMessages[msgId].querySelector('.chat-msg-text');
                if (textEl) textEl.textContent = content;
            }
            break;
        }
        case 'typing.start': chatShowTyping(); break;
        case 'typing.stop': chatRemoveTyping(); break;
    }
}

function chatAddMessage(text, role) {
    const container = document.getElementById('chatMessages');
    const el = document.createElement('div');
    el.className = 'chat-msg ' + role;
    if (role === 'bot') {
        const label = document.createElement('div');
        label.className = 'chat-msg-label';
        label.textContent = 'OK';
        el.appendChild(label);
        const textEl = document.createElement('div');
        textEl.className = 'chat-msg-text';
        textEl.textContent = text;
        el.appendChild(textEl);
    } else {
        el.textContent = text;
    }
    container.appendChild(el);
    container.scrollTop = container.scrollHeight;
    return el;
}

function chatShowTyping() {
    if (chatTypingEl) return;
    const container = document.getElementById('chatMessages');
    chatTypingEl = document.createElement('div');
    chatTypingEl.className = 'chat-msg bot typing';
    chatTypingEl.textContent = 'Thinking...';
    container.appendChild(chatTypingEl);
    container.scrollTop = container.scrollHeight;
}

function chatRemoveTyping() {
    if (chatTypingEl) { chatTypingEl.remove(); chatTypingEl = null; }
}

function chatSendMessage() {
    const input = document.getElementById('chatInput');
    const text = input.value.trim();
    if (!text || !chatWs || chatWs.readyState !== 1) return;

    chatAddMessage(text, 'user');
    chatWs.send(JSON.stringify({
        type: 'message.send',
        id: 'msg-' + (++chatMsgCounter),
        payload: { content: text }
    }));
    input.value = '';
    input.style.height = 'auto';
}

document.getElementById('chatSend').addEventListener('click', chatSendMessage);
document.getElementById('chatInput').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); chatSendMessage(); }
});
document.getElementById('chatInput').addEventListener('input', function() {
    this.style.height = 'auto';
    this.style.height = Math.min(this.scrollHeight, 120) + 'px';
});

// ── Reusable Form Components ────────────────────────

function renderFormField(key, label, type, value, opts = {}) {
    const val = value !== undefined && value !== null ? value : '';
    let h = type === 'textarea' ? '<div class="form-group form-grid-full">' : '<div class="form-group">';
    h += `<label class="form-label">${esc(label)}${opts.required ? ' *' : ''}</label>`;
    if (type === 'textarea') {
        h += `<textarea class="form-input" data-field="${key}" rows="${opts.rows || 3}" placeholder="${opts.placeholder || ''}">${esc(String(val))}</textarea>`;
    } else {
        h += `<input class="form-input ${type === 'number' ? 'form-input-number' : ''}" `;
        h += `type="${type === 'password' ? 'password' : type === 'number' ? 'number' : 'text'}" `;
        h += `data-field="${key}" value="${esc(String(val))}" placeholder="${opts.placeholder || ''}"`;
        if (opts.step) h += ` step="${opts.step}"`;
        if (opts.min !== undefined) h += ` min="${opts.min}"`;
        if (opts.max !== undefined) h += ` max="${opts.max}"`;
        h += '>';
    }
    if (opts.hint) h += `<div class="form-hint">${opts.hint}</div>`;
    h += '</div>';
    return h;
}

function renderToggleField(key, label, value, opts = {}) {
    let h = '<div class="toggle-row">';
    h += `<div class="toggle ${value ? 'on' : ''}" data-field="${key}" onclick="this.classList.toggle('on')"></div>`;
    h += `<span class="toggle-label">${esc(label)}</span>`;
    if (opts.hint) h += `<span class="form-hint" style="margin-left:8px;">${opts.hint}</span>`;
    h += '</div>';
    return h;
}

function renderSelectField(key, label, options, value, opts = {}) {
    let h = '<div class="form-group">';
    h += `<label class="form-label">${esc(label)}</label>`;
    h += `<select class="form-input" data-field="${key}" style="cursor:pointer;">`;
    options.forEach(o => {
        const optVal = typeof o === 'string' ? o : o.value;
        const optLabel = typeof o === 'string' ? o : o.label;
        h += `<option value="${esc(optVal)}" ${optVal === value ? 'selected' : ''}>${esc(optLabel)}</option>`;
    });
    h += '</select></div>';
    return h;
}

function renderArrayField(key, label, items, opts = {}) {
    const arr = items || [];
    let h = '<div class="form-group form-grid-full">';
    h += `<label class="form-label">${esc(label)}</label>`;
    h += `<div class="array-editor" data-field="${key}" data-placeholder="${opts.placeholder || ''}">`;
    arr.forEach(v => {
        h += '<div class="array-row">';
        h += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="${opts.placeholder || ''}">`;
        h += '<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>';
        h += '</div>';
    });
    h += `<div class="array-add" onclick="addArrayRow(this.parentElement)">${t('addItem')}</div>`;
    h += '</div></div>';
    return h;
}

function renderKVEditor(key, label, map, opts = {}) {
    const entries = map ? Object.entries(map) : [];
    let h = '<div class="form-group form-grid-full">';
    h += `<label class="form-label">${esc(label)}</label>`;
    h += `<div class="kv-editor" data-field="${key}">`;
    entries.forEach(([k, v]) => {
        h += '<div class="kv-row">';
        h += `<input class="form-input kv-key" type="text" value="${esc(k)}" placeholder="${opts.keyPlaceholder || 'Key'}">`;
        h += `<input class="form-input kv-value" type="${opts.valueType || 'text'}" value="${esc(String(v))}" placeholder="${opts.valuePlaceholder || 'Value'}">`;
        h += '<button class="btn btn-sm btn-danger" onclick="removeKVRow(this)">&times;</button>';
        h += '</div>';
    });
    h += `<div class="array-add" onclick="addKVRow(this.parentElement)">+ Add</div>`;
    h += '</div></div>';
    return h;
}

function addKVRow(container) {
    const addBtn = container.querySelector('.array-add');
    const row = document.createElement('div');
    row.className = 'kv-row';
    row.innerHTML = '<input class="form-input kv-key" type="text" placeholder="Key">' +
        '<input class="form-input kv-value" type="text" placeholder="Value">' +
        '<button class="btn btn-sm btn-danger" onclick="removeKVRow(this)">&times;</button>';
    container.insertBefore(row, addBtn);
    row.querySelector('.kv-key').focus();
}

function removeKVRow(btn) { btn.parentElement.remove(); }

function collectFormFields(container) {
    const result = {};
    // Text/password/number inputs
    container.querySelectorAll('input[data-field], textarea[data-field]').forEach(el => {
        const key = el.dataset.field;
        let val = el.value.trim();
        if (el.type === 'number' && val) val = parseFloat(val);
        if (val !== '' && val !== 0) result[key] = val;
        else result[key] = el.type === 'number' ? 0 : '';
    });
    // Selects
    container.querySelectorAll('select[data-field]').forEach(el => {
        result[el.dataset.field] = el.value;
    });
    // Toggles
    container.querySelectorAll('.toggle[data-field]').forEach(el => {
        result[el.dataset.field] = el.classList.contains('on');
    });
    // Array editors
    container.querySelectorAll('.array-editor[data-field]').forEach(el => {
        const vals = [];
        el.querySelectorAll('.array-row input').forEach(input => {
            const v = input.value.trim();
            if (v) vals.push(v);
        });
        result[el.dataset.field] = vals;
    });
    // KV editors
    container.querySelectorAll('.kv-editor[data-field]').forEach(el => {
        const map = {};
        el.querySelectorAll('.kv-row').forEach(row => {
            const k = row.querySelector('.kv-key').value.trim();
            const v = row.querySelector('.kv-value').value.trim();
            if (k) map[k] = v;
        });
        result[el.dataset.field] = map;
    });
    return result;
}

// ── MCP Servers Panel ───────────────────────────────

let editingMCPIndex = -1;

function renderMCP() {
    const panel = document.getElementById('panelMCP');
    if (!configData) { panel.innerHTML = ''; return; }

    const servers = configData.mcp_servers || [];

    let html = panelHeader(t('mcp.title'), 'panelMCP');
    html += `<div class="panel-desc">${t('mcp.desc')}</div>`;
    html += `<div style="margin-bottom:14px;"><button class="btn btn-sm btn-primary" onclick="showAddMCPModal()">${t('mcp.addServer')}</button></div>`;

    if (servers.length === 0) {
        html += `<div class="empty-state">
            <div class="empty-state-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="2" y="3" width="12" height="10" rx="2"/><path d="M5 8h6M8 6v4"/></svg></div>
            <div class="empty-state-title">${t('mcp.noServers')}</div>
            <div class="empty-state-desc">${t('mcp.noServersDesc')}</div>
        </div>`;
    } else {
        html += '<div class="model-grid">';
        servers.forEach((s, i) => {
            html += `<div class="model-card ${s.enabled ? '' : 'unavailable'}">`;
            html += '<div class="model-card-head">';
            html += `<div class="model-name">${esc(s.name)} <span class="model-protocol">${esc(s.transport || 'stdio')}</span></div>`;
            if (s.enabled) html += `<span class="badge-primary">${t('enabled')}</span>`;
            else html += `<span class="badge-nokey">${t('disabled')}</span>`;
            html += '</div>';
            if (s.command) html += `<div class="model-detail"><strong>${t('mcp.command')}:</strong> ${esc(s.command)} ${(s.args||[]).map(a => esc(a)).join(' ')}</div>`;
            if (s.url) html += `<div class="model-detail"><strong>URL:</strong> ${esc(s.url)}</div>`;
            if (s.tool_prefix) html += `<div class="model-detail"><strong>${t('mcp.toolPrefix')}:</strong> ${esc(s.tool_prefix)}</div>`;
            html += '<div class="model-actions">';
            html += `<button class="btn btn-sm" onclick="showEditMCPModal(${i})">${t('edit')}</button>`;
            html += `<button class="btn btn-sm btn-success" onclick="testMCPConnection(${i})">${t('mcp.test')}</button>`;
            html += `<button class="btn btn-sm btn-danger" onclick="deleteMCP(${i})">${t('delete')}</button>`;
            html += '</div></div>';
        });
        html += '</div>';
    }

    panel.innerHTML = html;
}

function showAddMCPModal() {
    editingMCPIndex = -1;
    document.getElementById('mcpModalTitle').textContent = t('mcp.addServerTitle');
    renderMCPModalBody({ transport: 'stdio', enabled: true });
    document.getElementById('mcpModal').classList.add('active');
}

function showEditMCPModal(idx) {
    editingMCPIndex = idx;
    const s = configData.mcp_servers[idx];
    document.getElementById('mcpModalTitle').textContent = t('mcp.editServer', { name: s.name });
    renderMCPModalBody(s);
    document.getElementById('mcpModal').classList.add('active');
}

function closeMCPModal() {
    document.getElementById('mcpModal').classList.remove('active');
}

function renderMCPModalBody(data) {
    let html = '';
    html += renderFormField('name', t('mcp.serverName'), 'text', data.name, { required: true, placeholder: 'e.g. filesystem' });
    html += renderToggleField('enabled', t('enabled'), data.enabled !== false);
    html += renderSelectField('transport', t('mcp.transport'), [
        { value: 'stdio', label: t('mcp.transportStdio') },
        { value: 'http', label: t('mcp.transportHttp') },
    ], data.transport || 'stdio');

    // stdio fields
    const isStdio = (data.transport || 'stdio') === 'stdio';
    html += `<div id="mcpStdioFields" style="display:${isStdio ? 'block' : 'none'}">`;
    html += renderFormField('command', t('mcp.command'), 'text', data.command, { placeholder: 'e.g. npx' });
    html += renderArrayField('args', t('mcp.arguments'), data.args, { placeholder: 'e.g. @modelcontextprotocol/server-filesystem' });
    html += renderKVEditor('env', t('mcp.envVars'), data.env, { keyPlaceholder: 'VAR_NAME', valuePlaceholder: 'value' });
    html += '</div>';

    // http fields
    html += `<div id="mcpHttpFields" style="display:${isStdio ? 'none' : 'block'}">`;
    html += renderFormField('url', t('mcp.serverUrl'), 'text', data.url, { placeholder: 'http://localhost:8080/sse' });
    html += renderKVEditor('headers', t('mcp.httpHeaders'), data.headers, { keyPlaceholder: 'Header-Name', valuePlaceholder: 'value' });
    html += '</div>';

    // Common fields
    html += renderFormField('timeout', t('mcp.timeout'), 'number', data.timeout || 30, { min: 1, placeholder: '30' });
    html += renderFormField('tool_prefix', t('mcp.toolPrefix'), 'text', data.tool_prefix, { placeholder: 'Optional prefix for tool names', hint: t('mcp.toolPrefixHint') });

    // Test results area
    html += '<div id="mcpTestResults"></div>';

    document.getElementById('mcpModalBody').innerHTML = html;

    // Add transport change handler
    const transportSelect = document.querySelector('#mcpModalBody select[data-field="transport"]');
    if (transportSelect) {
        transportSelect.addEventListener('change', function() {
            document.getElementById('mcpStdioFields').style.display = this.value === 'stdio' ? 'block' : 'none';
            document.getElementById('mcpHttpFields').style.display = this.value !== 'stdio' ? 'block' : 'none';
        });
    }
}

function saveMCPFromModal() {
    const body = document.getElementById('mcpModalBody');
    const fields = collectFormFields(body);

    if (!fields.name) {
        showStatus(t('mcp.nameRequired'), 'error');
        return;
    }

    const server = {
        name: fields.name,
        enabled: fields.enabled !== false,
        transport: fields.transport || 'stdio',
    };

    if (server.transport === 'stdio') {
        if (fields.command) server.command = fields.command;
        if (fields.args && fields.args.length) server.args = fields.args;
        if (fields.env && Object.keys(fields.env).length) server.env = fields.env;
    } else {
        if (fields.url) server.url = fields.url;
        if (fields.headers && Object.keys(fields.headers).length) server.headers = fields.headers;
    }

    if (fields.timeout) server.timeout = parseInt(fields.timeout) || 30;
    if (fields.tool_prefix) server.tool_prefix = fields.tool_prefix;

    if (!configData.mcp_servers) configData.mcp_servers = [];

    if (editingMCPIndex >= 0) {
        configData.mcp_servers[editingMCPIndex] = server;
    } else {
        configData.mcp_servers.push(server);
    }

    closeMCPModal();
    saveConfig().then(renderMCP);
}

function deleteMCP(idx) {
    if (!confirm(t('mcp.deleteConfirm', { name: configData.mcp_servers[idx].name }))) return;
    configData.mcp_servers.splice(idx, 1);
    saveConfig().then(renderMCP);
}

async function testMCPConnection(idx) {
    const server = configData.mcp_servers[idx];
    showStatus(t('mcp.testing', { name: server.name }), 'success');
    try {
        const res = await fetch('/api/mcp/test', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(server),
        });
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json();
        const tools = data.tools || [];
        showStatus(t('mcp.testResult', { name: server.name, count: tools.length }), 'success');
    } catch (e) {
        showStatus(t('mcp.testFailed', { msg: e.message }), 'error');
    }
}

// ── Agents Panel ────────────────────────────────────

function renderAgents() {
    const panel = document.getElementById('panelAgents');
    if (!configData) return;

    const agents = (configData.agents && configData.agents.list) || [];
    const d = (configData.agents && configData.agents.defaults) || {};

    let html = panelHeader(t('agents.title'), 'panelAgents');
    html += `<div class="panel-desc">${t('agents.desc')}</div>`;

    // ── Defaults section ──
    html += `<div class="form-section-title">${t('agents.defaults')}</div>`;
    html += `<div style="color:var(--text-muted);font-size:12px;margin-bottom:12px;">${t('agents.defaultsDesc')}</div>`;
    html += '<div class="channel-form form-grid" id="agentDefaultsForm">';
    html += renderModelSelect('model_name', t('agents.modelName'), d.model_name || '', { allowEmpty: true, emptyLabel: '\u2014', hint: t('agents.modelNameHint') });
    html += renderFormField('workspace', t('field.workspace'), 'text', d.workspace || '', { placeholder: '~/.ok/workspace' });
    html += renderToggleField('restrict_to_workspace', t('agents.restrictWorkspace'), d.restrict_to_workspace);
    html += renderToggleField('allow_read_outside_workspace', t('agents.allowReadOutside'), d.allow_read_outside_workspace);
    html += renderFormField('max_tokens', t('agents.maxTokens'), 'number', d.max_tokens || 32768, { min: 1 });
    html += renderFormField('temperature', t('agentDefaults.temperature'), 'number', d.temperature !== undefined ? d.temperature : '', { min: 0, max: 2, step: 0.1 });
    html += renderFormField('max_tool_iterations', t('agents.maxToolIter'), 'number', d.max_tool_iterations || 50, { min: 1 });
    html += renderFormField('summarize_message_threshold', t('agents.summarizeThreshold'), 'number', d.summarize_message_threshold || 20, { min: 1, hint: t('agents.summarizeThresholdHint') });
    html += renderFormField('summarize_token_percent', t('agents.summarizeTokenPct'), 'number', d.summarize_token_percent || 75, { min: 1, max: 100 });
    html += renderFormField('max_media_size', t('agents.maxMediaSize'), 'number', d.max_media_size || 0, { hint: t('agents.maxMediaSizeHint') });
    html += `<div style="margin-top:16px;"><button class="btn btn-primary" onclick="saveAgentDefaults()">${t('agents.saveDefaults')}</button></div>`;
    html += '</div>';

    // ── Agent list section ──
    html += `<div class="form-section-title" style="margin-top:28px;">${t('agents.agentList')}</div>`;
    html += `<div style="margin-bottom:14px;"><button class="btn btn-sm btn-primary" onclick="showAddAgentModal()">${t('agents.addAgent')}</button></div>`;
    if (agents.length === 0) {
        html += `<div style="color:var(--text-muted);font-size:13px;margin-bottom:16px;">${t('agents.noAgents')}</div>`;
    } else {
        html += '<div class="model-grid">';
        agents.forEach((a, i) => {
            const modelObj = a.model || {};
            const modelPrimary = typeof modelObj === 'string' ? modelObj : (modelObj.primary || '');
            const modelFallbacks = typeof modelObj === 'string' ? [] : (modelObj.fallbacks || []);
            // Resolve display name from model_list
            const modelDisplay = resolveModelDisplay(modelPrimary);
            const defaultModelDisplay = d.model_name ? resolveModelDisplay(d.model_name) : '';

            html += `<div class="model-card">`;
            html += `<div class="model-card-head"><div class="model-name">${esc(a.name || a.id)}</div>`;
            if (a.default) html += `<span class="badge-primary">${t('agents.defaultAgent')}</span>`;
            html += '</div>';
            html += `<div class="model-detail" style="color:var(--text-muted);font-size:11px;">${esc(a.id)}</div>`;

            // Model info
            if (modelPrimary) {
                html += `<div class="model-detail"><strong>${t('agents.sectionModel')}:</strong> ${esc(modelDisplay)}`;
                html += ` <span style="font-size:10px;color:var(--primary);opacity:0.8;">${t('agents.overridden')}</span>`;
                html += `</div>`;
            } else if (defaultModelDisplay) {
                html += `<div class="model-detail" style="opacity:0.6;"><strong>${t('agents.sectionModel')}:</strong> ${esc(defaultModelDisplay)}`;
                html += ` <span style="font-size:10px;">${t('agents.inheritedFrom')}</span>`;
                html += `</div>`;
            }
            if (modelFallbacks.length > 0) {
                html += `<div class="model-detail"><strong>${t('agents.modelFallbacks')}:</strong> ${modelFallbacks.map(f => esc(resolveModelDisplay(f))).join(', ')}</div>`;
            }
            if (a.skills && a.skills.length) {
                html += `<div class="model-detail"><strong>Skills:</strong> ${a.skills.map(s => esc(s)).join(', ')}</div>`;
            }
            if (a.subagents && a.subagents.allow_agents && a.subagents.allow_agents.length) {
                html += `<div class="model-detail"><strong>${t('agents.subagents')}:</strong> ${a.subagents.allow_agents.map(s => esc(s)).join(', ')}</div>`;
            }
            html += '<div class="model-actions">';
            html += `<button class="btn btn-sm" onclick="showEditAgentModal(${i})">${t('edit')}</button>`;
            html += `<button class="btn btn-sm btn-danger" onclick="deleteAgent(${i})">${t('delete')}</button>`;
            html += '</div></div>';
        });
        html += '</div>';
    }

    panel.innerHTML = html;
}

// Helper: resolve model_name to display string
function resolveModelDisplay(modelName) {
    if (!modelName) return '';
    const models = (configData && configData.model_list) || [];
    const m = models.find(m => m.model_name === modelName);
    if (m && m.model) return modelName + ' (' + m.model + ')';
    return modelName;
}

function saveAgentDefaults() {
    if (!configData) return;
    const form = document.getElementById('agentDefaultsForm');
    const fields = collectFormFields(form);

    if (!configData.agents) configData.agents = {};
    if (!configData.agents.defaults) configData.agents.defaults = {};

    const d = configData.agents.defaults;
    d.workspace = fields.workspace || d.workspace;
    d.restrict_to_workspace = fields.restrict_to_workspace;
    d.allow_read_outside_workspace = fields.allow_read_outside_workspace;
    d.model_name = fields.model_name || '';
    if (fields.max_tokens) d.max_tokens = parseInt(fields.max_tokens) || 32768;
    if (fields.max_tool_iterations) d.max_tool_iterations = parseInt(fields.max_tool_iterations) || 50;
    if (fields.summarize_message_threshold) d.summarize_message_threshold = parseInt(fields.summarize_message_threshold) || 20;
    if (fields.summarize_token_percent) d.summarize_token_percent = parseInt(fields.summarize_token_percent) || 75;
    d.max_media_size = parseInt(fields.max_media_size) || 0;
    if (fields.temperature !== undefined && fields.temperature !== '') {
        d.temperature = parseFloat(fields.temperature);
        if (isNaN(d.temperature)) delete d.temperature;
    } else {
        delete d.temperature;
    }

    saveConfig().then(() => showStatus(t('agents.defaultsSaved'), 'success'));
}

function deleteAgent(idx) {
    const agents = configData.agents.list;
    if (!confirm(t('agents.deleteConfirm', { name: agents[idx].name || agents[idx].id }))) return;
    agents.splice(idx, 1);
    saveConfig().then(renderAgents);
}

// ── Agent Modal ─────────────────────────────────────
let editingAgentIndex = -1;

function showAddAgentModal() {
    editingAgentIndex = -1;
    document.getElementById('agentModalTitle').textContent = t('agents.addAgentTitle');
    renderAgentModalBody({});
    document.getElementById('agentModal').classList.add('active');
}

function showEditAgentModal(idx) {
    editingAgentIndex = idx;
    const a = configData.agents.list[idx];
    document.getElementById('agentModalTitle').textContent = t('agents.editAgent') + ': ' + (a.name || a.id);
    renderAgentModalBody(a);
    document.getElementById('agentModal').classList.add('active');
}

function closeAgentModal() {
    document.getElementById('agentModal').classList.remove('active');
}

async function renderAgentModalBody(data) {
    const modelObj = data.model || {};
    const modelPrimary = typeof modelObj === 'string' ? modelObj : (modelObj.primary || '');
    const modelFallbacks = typeof modelObj === 'string' ? [] : (modelObj.fallbacks || []);
    const agentSkills = data.skills || [];
    const subagents = (data.subagents && data.subagents.allow_agents) || [];
    const allAgents = (configData.agents && configData.agents.list) || [];

    // Ensure skills cache is loaded
    if (!_skillsCache) {
        try {
            const resp = await fetch('/api/skills');
            if (resp.ok) _skillsCache = await resp.json();
        } catch (e) { /* ignore */ }
    }
    const installedSkills = _skillsCache || [];

    let html = '';

    // ── Identity section ──
    html += `<div class="form-section-title" style="margin-top:0;">${t('agents.sectionIdentity')}</div>`;
    html += renderFormField('name', t('agents.sectionIdentity'), 'text', data.name, { required: true, placeholder: t('agents.friendlyName') });
    if (editingAgentIndex >= 0) {
        html += `<div class="form-group"><label class="form-label">ID</label><input class="form-input" type="text" data-field="id" value="${esc(data.id || '')}" readonly style="opacity:0.6;"></div>`;
    } else {
        html += renderFormField('id', 'ID', 'text', data.id, { placeholder: 'auto-generated from name', hint: 'Unique identifier (auto-generated if empty)' });
    }
    html += renderToggleField('default', t('agents.defaultAgent'), !!data.default);

    // ── Model section ──
    html += `<div class="form-section-title">${t('agents.sectionModel')}</div>`;
    html += renderModelSelect('model_primary', t('agents.modelPrimary'), modelPrimary, { allowEmpty: true, emptyLabel: t('agents.inheritsDefaults') });
    html += renderModelSelect('model_fallback_1', t('agents.modelFallbacks'), modelFallbacks[0] || '', { allowEmpty: true, emptyLabel: '\u2014' });

    // ── Skills section ──
    if (installedSkills.length > 0) {
        html += `<div class="form-section-title">Skills</div>`;
        html += `<div class="form-group">`;
        html += `<div style="display:flex;flex-wrap:wrap;gap:8px 16px;">`;
        installedSkills.forEach(s => {
            const checked = agentSkills.includes(s.name) ? ' checked' : '';
            const desc = s.description ? ` title="${esc(s.description)}"` : '';
            const sourceTag = s.source === 'builtin' ? ' <span style="font-size:10px;opacity:0.5;">builtin</span>' : '';
            html += `<label style="display:flex;align-items:center;gap:5px;font-size:13px;cursor:pointer;"${desc}>`;
            html += `<input type="checkbox" class="skill-check" value="${esc(s.name)}"${checked}>`;
            html += `${esc(s.name)}${sourceTag}`;
            html += `</label>`;
        });
        html += `</div>`;
        html += `<span class="form-hint">${t('agents.skillsHint')}</span>`;
        html += `</div>`;
    }

    // ── Advanced section (collapsible) ──
    html += `<details style="margin-top:16px;"><summary class="form-section-title" style="cursor:pointer;user-select:none;">${t('agents.sectionAdvanced')}</summary>`;
    html += `<div style="padding-top:8px;">`;
    html += renderFormField('workspace', t('field.workspace'), 'text', data.workspace, { placeholder: t('agents.inheritsDefaults') });

    // Subagents: checkboxes of other agents
    const otherAgents = allAgents.filter(a => a.id !== data.id);
    if (otherAgents.length > 0) {
        html += `<div class="form-group"><label class="form-label">${t('agents.subagents')}</label>`;
        html += `<div style="display:flex;flex-wrap:wrap;gap:8px;">`;
        otherAgents.forEach(a => {
            const checked = subagents.includes(a.id) ? ' checked' : '';
            html += `<label style="display:flex;align-items:center;gap:4px;font-size:13px;cursor:pointer;">`;
            html += `<input type="checkbox" class="subagent-check" value="${esc(a.id)}"${checked}> ${esc(a.name || a.id)}`;
            html += `</label>`;
        });
        html += `</div>`;
        html += `<span class="form-hint">${t('agents.subagentsHint')}</span></div>`;
    }
    html += `</div></details>`;

    document.getElementById('agentModalBody').innerHTML = html;
}

function saveAgentFromModal() {
    const body = document.getElementById('agentModalBody');
    const fields = collectFormFields(body);

    // Auto-generate ID from name if empty
    let agentId = (fields.id || '').trim();
    const agentName = (fields.name || '').trim();
    if (!agentId && agentName) {
        agentId = agentName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    }
    if (!agentId) {
        showStatus(t('agents.idRequired'), 'error');
        return;
    }

    if (!configData.agents) configData.agents = {};
    if (!configData.agents.list) configData.agents.list = [];

    const agent = editingAgentIndex >= 0 ? { ...configData.agents.list[editingAgentIndex] } : {};
    agent.id = agentId;
    if (agentName) agent.name = agentName; else delete agent.name;
    if (fields.default) agent.default = true; else delete agent.default;
    if (fields.workspace) agent.workspace = fields.workspace; else delete agent.workspace;

    // Model
    const modelPrimary = fields.model_primary || '';
    const modelFallback = fields.model_fallback_1 || '';
    const modelFallbacks = modelFallback ? [modelFallback] : [];
    if (modelPrimary) {
        if (modelFallbacks.length > 0) {
            agent.model = { primary: modelPrimary, fallbacks: modelFallbacks };
        } else {
            agent.model = modelPrimary;
        }
    } else {
        delete agent.model;
    }

    // Skills (from checkboxes)
    const skillChecks = body.querySelectorAll('.skill-check:checked');
    const selectedSkills = Array.from(skillChecks).map(cb => cb.value);
    if (selectedSkills.length > 0) agent.skills = selectedSkills; else delete agent.skills;

    // Subagents
    const subagentChecks = body.querySelectorAll('.subagent-check:checked');
    const allowAgents = Array.from(subagentChecks).map(cb => cb.value);
    if (allowAgents.length > 0) {
        agent.subagents = { allow_agents: allowAgents };
    } else {
        delete agent.subagents;
    }

    if (editingAgentIndex >= 0) {
        configData.agents.list[editingAgentIndex] = agent;
    } else {
        if (configData.agents.list.some(a => a.id === agentId)) {
            showStatus(t('agents.duplicateId', { id: agentId }), 'error');
            return;
        }
        configData.agents.list.push(agent);
    }

    closeAgentModal();
    saveConfig().then(() => {
        renderAgents();
        showStatus(t('agents.agentSaved'), 'success');
    });
}

// ── Tool Settings Panel ─────────────────────────────

function renderToolSettings() {
    const panel = document.getElementById('panelToolSettings');
    if (!configData) return;

    const tools = configData.tools || {};

    let html = panelHeader(t('tools.title'), 'panelToolSettings');
    html += `<div class="panel-desc">${t('tools.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="toolSettingsForm">';

    // Quick toggles
    html += `<div class="form-section-title">${t('tools.toggles')}</div>`;
    const simpleTools = [
        ['append_file', 'Append File'], ['edit_file', 'Edit File'], ['find_skills', 'Find Skills'],
        ['i2c', 'I2C (Hardware)'], ['install_skill', 'Install Skill'], ['list_dir', 'List Directory'],
        ['message', 'Message'], ['read_file', 'Read File'], ['send_file', 'Send File'],
        ['spawn', 'Spawn'], ['spi', 'SPI (Hardware)'], ['subagent', 'Subagent'],
        ['web_fetch', 'Web Fetch'], ['write_file', 'Write File'],
    ];
    html += '<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:4px;grid-column:1/-1;">';
    simpleTools.forEach(([key, label]) => {
        const enabled = tools[key] ? tools[key].enabled : false;
        html += renderToggleField('tool_' + key, label, enabled);
    });
    html += '</div>';

    // Web search config
    html += `<div class="form-section-title">${t('tools.webSearch')}</div>`;
    html += renderToggleField('web_enabled', t('tools.webEnabled'), tools.web ? tools.web.enabled : true);
    html += renderFormField('web_proxy', t('tools.webProxy'), 'text', tools.web ? tools.web.proxy : '', { placeholder: 'http://proxy:port' });
    html += renderFormField('web_fetch_limit', t('tools.webFetchLimit'), 'number', tools.web ? tools.web.fetch_limit_bytes : 10485760);

    // Exec config
    html += `<div class="form-section-title">${t('tools.shellExec')}</div>`;
    html += renderToggleField('exec_enabled', t('tools.execEnabled'), tools.exec ? tools.exec.enabled : true);
    html += renderToggleField('exec_deny_patterns', t('tools.execDenyPatterns'), tools.exec ? tools.exec.enable_deny_patterns : true);
    html += renderFormField('exec_timeout', t('tools.execTimeout'), 'number', tools.exec ? tools.exec.timeout_seconds : 60);
    html += renderArrayField('exec_custom_deny', t('tools.customDenyPatterns'), tools.exec ? tools.exec.custom_deny_patterns : [], { placeholder: 'e.g. rm -rf /' });
    html += renderArrayField('exec_custom_allow', t('tools.customAllowPatterns'), tools.exec ? tools.exec.custom_allow_patterns : [], { placeholder: 'e.g. docker build' });

    // Media Cleanup
    html += `<div class="form-section-title">${t('tools.mediaCleanup')}</div>`;
    html += renderToggleField('media_cleanup_enabled', t('enabled'), tools.media_cleanup ? tools.media_cleanup.enabled : false);
    html += renderFormField('media_cleanup_max_age', t('tools.mediaMaxAge'), 'number', tools.media_cleanup ? tools.media_cleanup.max_age_minutes : 60, { min: 1 });
    html += renderFormField('media_cleanup_interval', t('tools.mediaInterval'), 'number', tools.media_cleanup ? tools.media_cleanup.interval_minutes : 30, { min: 1 });

    // Cron config
    html += `<div class="form-section-title">${t('tools.cron')}</div>`;
    html += renderToggleField('cron_enabled', t('tools.cronEnabled'), tools.cron ? tools.cron.enabled : true);
    html += renderFormField('cron_timeout', t('tools.cronTimeout'), 'number', tools.cron ? tools.cron.exec_timeout_minutes : 5);

    // Paths
    html += `<div class="form-section-title">${t('tools.paths')}</div>`;
    html += renderArrayField('allow_read_paths', t('tools.allowReadPaths'), tools.allow_read_paths, { placeholder: '/path/to/allow' });
    html += renderArrayField('allow_write_paths', t('tools.allowWritePaths'), tools.allow_write_paths, { placeholder: '/path/to/allow' });

    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveToolSettings()">${t('save')}</button></div>`;
    html += '</div>';

    panel.innerHTML = html;
}

function saveToolSettings() {
    if (!configData) return;
    const form = document.getElementById('toolSettingsForm');
    const f = collectFormFields(form);

    if (!configData.tools) configData.tools = {};
    const t_ = configData.tools;

    // Simple tool toggles
    const simpleToolKeys = ['append_file', 'edit_file', 'find_skills', 'i2c', 'install_skill', 'list_dir', 'message', 'read_file', 'send_file', 'spawn', 'spi', 'subagent', 'web_fetch', 'write_file'];
    simpleToolKeys.forEach(key => {
        if (!t_[key]) t_[key] = {};
        t_[key].enabled = !!f['tool_' + key];
    });

    // Web
    if (!t_.web) t_.web = {};
    t_.web.enabled = !!f.web_enabled;
    t_.web.proxy = f.web_proxy || '';
    t_.web.fetch_limit_bytes = parseInt(f.web_fetch_limit) || 10485760;

    // Exec
    if (!t_.exec) t_.exec = {};
    t_.exec.enabled = !!f.exec_enabled;
    t_.exec.enable_deny_patterns = !!f.exec_deny_patterns;
    t_.exec.timeout_seconds = parseInt(f.exec_timeout) || 60;
    t_.exec.custom_deny_patterns = f.exec_custom_deny || [];
    t_.exec.custom_allow_patterns = f.exec_custom_allow || [];

    // Media Cleanup
    if (!t_.media_cleanup) t_.media_cleanup = {};
    t_.media_cleanup.enabled = !!f.media_cleanup_enabled;
    t_.media_cleanup.max_age_minutes = parseInt(f.media_cleanup_max_age) || 60;
    t_.media_cleanup.interval_minutes = parseInt(f.media_cleanup_interval) || 30;

    // Cron
    if (!t_.cron) t_.cron = {};
    t_.cron.enabled = !!f.cron_enabled;
    t_.cron.exec_timeout_minutes = parseInt(f.cron_timeout) || 5;

    // Paths
    t_.allow_read_paths = f.allow_read_paths || [];
    t_.allow_write_paths = f.allow_write_paths || [];

    saveConfig().then(() => showStatus(t('tools.saved'), 'success'));
}

// ── RAG Panel ───────────────────────────────────────

function detectRAGProvider(baseUrl) {
    if (!baseUrl) return '';
    for (const [name, info] of Object.entries(PROVIDER_INFO)) {
        if (info.apiBase && baseUrl.startsWith(info.apiBase.replace(/\/v1$/, ''))) return name;
    }
    return '';
}

function onRAGProviderChange(selectEl) {
    const selected = selectEl.value;
    const info = PROVIDER_INFO[selected];
    const baseUrlInput = document.querySelector('#ragForm input[data-field="base_url"]');
    if (selected && info && baseUrlInput) {
        baseUrlInput.value = info.apiBase;
    }
}

function renderRAG() {
    const panel = document.getElementById('panelRAG');
    if (!configData) return;

    const rag = configData.rag || {};
    const connectedProviders = getAllProviders().filter(p => p.authenticated);
    const detectedProvider = detectRAGProvider(rag.base_url);

    let html = panelHeader(t('rag.title'), 'panelRAG');
    html += `<div class="panel-desc">${t('rag.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="ragForm">';
    html += renderToggleField('enabled', t('enabled'), rag.enabled);

    // Provider select — only connected providers
    if (connectedProviders.length > 0) {
        html += '<div class="form-group">';
        html += `<label class="form-label">${t('field.provider')}</label>`;
        html += '<select class="form-input" data-field="rag_provider" onchange="onRAGProviderChange(this)">';
        connectedProviders.forEach((p, i) => {
            const sel = detectedProvider === p.name || (!detectedProvider && i === 0) ? ' selected' : '';
            html += `<option value="${esc(p.name)}"${sel}>${esc(p.label)}</option>`;
        });
        html += '</select></div>';
    }

    // Hidden field to hold the resolved base_url
    const resolvedBase = detectedProvider ? (PROVIDER_INFO[detectedProvider] || {}).apiBase || '' :
        (connectedProviders.length > 0 ? (PROVIDER_INFO[connectedProviders[0].name] || {}).apiBase || '' : rag.base_url || '');
    html += `<input type="hidden" data-field="base_url" value="${esc(resolvedBase)}">`;

    // Find model_name that matches the RAG model ID
    const ragModelId = rag.model || 'text-embedding-3-small';
    const ragModelList = (configData && configData.model_list) || [];
    const ragMatch = ragModelList.find(m => m.model && (m.model === ragModelId || m.model.endsWith('/' + ragModelId)));
    const ragModelName = ragMatch ? ragMatch.model_name : ragModelId;
    html += renderModelSelect('model', t('rag.embeddingModel'), ragModelName, { allowEmpty: false });
    html += renderFormField('top_k', t('rag.topK'), 'number', rag.top_k || 5, { min: 1, max: 50 });
    html += renderFormField('min_similarity', t('rag.minSimilarity'), 'number', rag.min_similarity || 0.5, { min: 0, max: 1, step: '0.05', hint: t('rag.minSimilarityHint') });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveRAG()">${t('save')}</button></div>`;
    html += '</div>';

    panel.innerHTML = html;
}

function saveRAG() {
    if (!configData) return;
    const form = document.getElementById('ragForm');
    const f = collectFormFields(form);

    // Resolve model ID and base_url from selected model_name
    let modelId = f.model || 'text-embedding-3-small';
    const models = (configData && configData.model_list) || [];
    const providersList = (configData && configData.provider_list) || [];
    const selectedModel = models.find(m => m.model_name === modelId);
    let baseUrl = f.base_url || '';
    if (selectedModel) {
        if (selectedModel.model) {
            modelId = selectedModel.model.replace(/^[^/]+\//, ''); // strip protocol prefix
        }
        // Resolve base_url from provider_list
        const provName = selectedModel.provider || (selectedModel.model ? selectedModel.model.split('/')[0] : '');
        const prov = providersList.find(p => p.name === provName);
        if (prov && prov.api_base) {
            baseUrl = prov.api_base;
        }
    }

    // Fallback: resolve base_url from provider select
    const providerSelect = form.querySelector('select[data-field="rag_provider"]');
    if (!baseUrl && providerSelect && providerSelect.value) {
        const info = PROVIDER_INFO[providerSelect.value];
        if (info) baseUrl = info.apiBase;
    }

    configData.rag = {
        enabled: !!f.enabled,
        base_url: baseUrl,
        model: modelId,
        top_k: parseInt(f.top_k) || 5,
        min_similarity: parseFloat(f.min_similarity) || 0.5,
    };
    saveConfig().then(() => showStatus(t('rag.saved'), 'success'));
}

// ── System Panels ───────────────────────────────────

function renderGateway() {
    const panel = document.getElementById('panelGateway');
    if (!configData) return;
    const gw = configData.gateway || {};
    let html = panelHeader(t('gateway.title'), 'panelGateway');
    html += `<div class="panel-desc">${t('gateway.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="gatewayForm">';
    html += renderFormField('host', t('gateway.host'), 'text', gw.host || '127.0.0.1', { placeholder: '127.0.0.1' });
    html += renderFormField('port', t('gateway.port'), 'number', gw.port || 18790, { min: 1, max: 65535 });
    html += renderFormField('proxy', t('gateway.proxy'), 'text', configData.proxy || '', { placeholder: 'http://proxy:port', hint: t('gateway.proxyHint') });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveGateway()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveGateway() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('gatewayForm'));
    configData.gateway = { host: f.host || '127.0.0.1', port: parseInt(f.port) || 18790 };
    configData.proxy = f.proxy || '';
    if (!configData.proxy) delete configData.proxy;
    saveConfig().then(() => showStatus(t('gateway.saved'), 'success'));
}

function renderSession() {
    const panel = document.getElementById('panelSession');
    if (!configData) return;
    const session = configData.session || {};
    let html = panelHeader(t('session.title'), 'panelSession');
    html += `<div class="panel-desc">${t('session.desc')}</div>`;
    html += '<div class="channel-form" id="sessionForm">';
    html += renderSelectField('dm_scope', t('session.dmScope'), [
        { value: 'per-channel-peer', label: t('session.perChannelPeer') },
        { value: 'per-peer', label: t('session.perPeer') },
        { value: 'per-channel', label: t('session.perChannel') },
        { value: 'global', label: t('session.global') },
    ], session.dm_scope || 'per-channel-peer');
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveSession()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveSession() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('sessionForm'));
    if (!configData.session) configData.session = {};
    configData.session.dm_scope = f.dm_scope || 'per-channel-peer';
    saveConfig().then(() => showStatus(t('session.saved'), 'success'));
}

function renderHeartbeat() {
    const panel = document.getElementById('panelHeartbeat');
    if (!configData) return;
    const hb = configData.heartbeat || {};
    let html = panelHeader(t('heartbeat.title'), 'panelHeartbeat');
    html += `<div class="panel-desc">${t('heartbeat.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="heartbeatForm">';
    html += renderToggleField('enabled', t('enabled'), hb.enabled);
    html += renderFormField('interval', t('heartbeat.interval'), 'number', hb.interval || 30, { min: 5, hint: t('heartbeat.intervalHint') });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveHeartbeat()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveHeartbeat() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('heartbeatForm'));
    configData.heartbeat = {
        enabled: !!f.enabled,
        interval: Math.max(5, parseInt(f.interval) || 30),
    };
    saveConfig().then(() => showStatus(t('heartbeat.saved'), 'success'));
}

function renderDevices() {
    const panel = document.getElementById('panelDevices');
    if (!configData) return;
    const dev = configData.devices || {};
    let html = panelHeader(t('devices.title'), 'panelDevices');
    html += `<div class="panel-desc">${t('devices.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="devicesForm">';
    html += renderToggleField('enabled', t('enabled'), dev.enabled);
    html += renderToggleField('monitor_usb', t('devices.monitorUsb'), dev.monitor_usb);
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveDevices()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveDevices() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('devicesForm'));
    configData.devices = { enabled: !!f.enabled, monitor_usb: !!f.monitor_usb };
    saveConfig().then(() => showStatus(t('devices.saved'), 'success'));
}

function renderDebug() {
    const panel = document.getElementById('panelDebug');
    if (!configData) return;
    let html = panelHeader(t('debug.title'), 'panelDebug');
    html += `<div class="panel-desc">${t('debug.desc')}</div>`;
    html += '<div class="channel-form" id="debugForm">';
    html += renderToggleField('debug', t('debug.debugMode'), configData.debug);
    html += `<div class="form-hint">${t('debug.hint')}</div>`;
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveDebugSetting()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveDebugSetting() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('debugForm'));
    configData.debug = !!f.debug;
    saveConfig().then(() => showStatus(t('debug.saved'), 'success'));
}

// ── New Panels (Routing, Providers, WebSearch, Summarization, AgentDefaults, WebUI) ──

function renderRouting() {
    const panel = document.getElementById('panelRouting');
    if (!configData) return;
    const routing = (configData.agents && configData.agents.defaults && configData.agents.defaults.routing) || {};

    let html = panelHeader(t('routing.title'), 'panelRouting');
    html += `<div class="panel-desc">${t('routing.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="routingForm">';
    html += renderToggleField('enabled', t('enabled'), routing.enabled);
    html += renderModelSelect('light_model', t('routing.lightModel'), routing.light_model || '', { allowEmpty: true, emptyLabel: '—' });
    html += renderFormField('threshold', t('routing.threshold'), 'number', routing.threshold !== undefined ? routing.threshold : 0.3, { min: 0, max: 1, step: 0.01, hint: t('routing.thresholdHint') });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveRouting()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveRouting() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('routingForm'));
    if (!configData.agents) configData.agents = {};
    if (!configData.agents.defaults) configData.agents.defaults = {};
    configData.agents.defaults.routing = {
        enabled: !!f.enabled,
        light_model: f.light_model || '',
        threshold: parseFloat(f.threshold) || 0.3,
    };
    saveConfig().then(() => showStatus(t('routing.saved'), 'success'));
}

function renderWebSearch() {
    const panel = document.getElementById('panelWebSearch');
    if (!configData) return;
    const web = (configData.tools && configData.tools.web) || {};

    let html = panelHeader(t('webSearch.title'), 'panelWebSearch');
    html += `<div class="panel-desc">${t('webSearch.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="webSearchForm">';

    // Global settings
    html += `<div class="form-section-title">${t('webSearch.global')}</div>`;
    html += renderToggleField('enabled', t('enabled'), web.enabled);
    html += renderFormField('proxy', t('field.proxy'), 'text', web.proxy || '');
    html += renderFormField('fetch_limit_bytes', t('tools.webFetchLimit'), 'number', web.fetch_limit_bytes || '', { min: 0 });

    // Search providers
    const searchProviders = [
        { key: 'brave', label: 'Brave', fields: ['api_key', 'max_results'] },
        { key: 'tavily', label: 'Tavily', fields: ['api_key', 'base_url', 'max_results'] },
        { key: 'duckduckgo', label: 'DuckDuckGo', fields: ['max_results'] },
        { key: 'perplexity', label: 'Perplexity', fields: ['api_key', 'max_results'] },
        { key: 'searxng', label: 'SearXNG', fields: ['base_url', 'max_results'] },
    ];

    searchProviders.forEach(sp => {
        const spData = (web.search && web.search[sp.key]) || {};
        html += `<div class="form-section-title">${sp.label}</div>`;
        html += renderToggleField(sp.key + '_enabled', t('enabled'), spData.enabled);
        sp.fields.forEach(f => {
            if (f === 'api_key') html += renderFormField(sp.key + '_api_key', t('field.apiKey'), 'password', spData.api_key || '');
            if (f === 'base_url') html += renderFormField(sp.key + '_base_url', t('field.apiBase'), 'text', spData.base_url || '');
            if (f === 'max_results') html += renderFormField(sp.key + '_max_results', 'Max Results', 'number', spData.max_results || '', { min: 1, max: 50 });
        });
    });

    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveWebSearch()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveWebSearch() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('webSearchForm'));
    if (!configData.tools) configData.tools = {};
    const web = configData.tools.web || {};
    web.enabled = !!f.enabled;
    web.proxy = f.proxy || '';
    web.fetch_limit_bytes = parseInt(f.fetch_limit_bytes) || 0;

    if (!web.search) web.search = {};
    ['brave', 'tavily', 'duckduckgo', 'perplexity', 'searxng'].forEach(key => {
        if (!web.search[key]) web.search[key] = {};
        web.search[key].enabled = !!f[key + '_enabled'];
        if (f[key + '_api_key'] !== undefined) web.search[key].api_key = f[key + '_api_key'] || '';
        if (f[key + '_base_url'] !== undefined) web.search[key].base_url = f[key + '_base_url'] || '';
        if (f[key + '_max_results'] !== undefined) web.search[key].max_results = parseInt(f[key + '_max_results']) || 0;
    });

    configData.tools.web = web;
    saveConfig().then(() => showStatus(t('webSearch.saved'), 'success'));
}

function renderSummarization() {
    const panel = document.getElementById('panelSummarization');
    if (!configData) return;
    const d = (configData.agents && configData.agents.defaults) || {};
    let html = panelHeader(t('summarization.title'), 'panelSummarization');
    html += `<div class="panel-desc">${t('summarization.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="summarizationForm">';
    html += renderFormField('summarize_message_threshold', t('agents.summarizeThreshold'), 'number', d.summarize_message_threshold || 20, { min: 1, hint: t('agents.summarizeThresholdHint') });
    html += renderFormField('summarize_token_percent', t('agents.summarizeTokenPct'), 'number', d.summarize_token_percent || 75, { min: 1, max: 100 });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveSummarization()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveSummarization() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('summarizationForm'));
    if (!configData.agents) configData.agents = {};
    if (!configData.agents.defaults) configData.agents.defaults = {};
    configData.agents.defaults.summarize_message_threshold = parseInt(f.summarize_message_threshold) || 20;
    configData.agents.defaults.summarize_token_percent = parseInt(f.summarize_token_percent) || 75;
    saveConfig().then(() => showStatus(t('summarization.saved'), 'success'));
}

// renderAgentDefaults merged into renderAgents()

function renderWebUI() {
    const panel = document.getElementById('panelWebUI');
    if (!configData) return;
    const ui = configData.web_ui || {};
    let html = panelHeader(t('webui.title'), 'panelWebUI');
    html += `<div class="panel-desc">${t('webui.desc')}</div>`;
    html += '<div class="channel-form form-grid" id="webuiForm">';
    html += renderToggleField('enabled', t('enabled'), ui.enabled);
    html += renderFormField('host', t('gateway.host'), 'text', ui.host || '127.0.0.1', { placeholder: '127.0.0.1' });
    html += renderFormField('port', t('gateway.port'), 'number', ui.port || 18800, { min: 1, max: 65535 });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveWebUI()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveWebUI() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('webuiForm'));
    configData.web_ui = {
        enabled: !!f.enabled,
        host: f.host || '127.0.0.1',
        port: parseInt(f.port) || 18800,
    };
    saveConfig().then(() => showStatus(t('webui.saved'), 'success'));
}

// ── WhatsApp QR SSE ─────────────────────────────────
function connectWhatsAppQR() {
    disconnectWhatsAppQR();
    const statusEl = document.getElementById('waQrStatus');
    const codeEl = document.getElementById('waQrCode');
    if (!statusEl || !codeEl) return;

    if (!gatewayRunning || !configData) {
        statusEl.textContent = 'Start the gateway to pair WhatsApp';
        codeEl.innerHTML = '';
        return;
    }

    const host = (configData.gateway && configData.gateway.host) || '127.0.0.1';
    const port = (configData.gateway && configData.gateway.port) || 18790;
    const url = 'http://' + host + ':' + port + '/whatsapp/qr/stream';

    statusEl.textContent = 'Connecting...';
    waQrSource = new EventSource(url);

    waQrSource.onmessage = function(e) {
        try {
            const data = JSON.parse(e.data);
            if (data.event === 'code' && data.code) {
                var qr = qrcode(0, 'L');
                qr.addData(data.code);
                qr.make();
                codeEl.innerHTML = qr.createSvgTag(5);
                var svg = codeEl.querySelector('svg');
                if (svg) { svg.style.maxWidth = '100%'; svg.style.height = 'auto'; }
                statusEl.textContent = 'Scan with WhatsApp \u203A Linked Devices';
            } else if (data.event === 'success' || data.paired) {
                codeEl.innerHTML = '<div class="badge badge-active" style="display:inline-block;padding:8px 16px;font-size:14px;">Connected</div>';
                statusEl.textContent = 'WhatsApp paired successfully';
            } else if (data.event === 'timeout') {
                statusEl.textContent = 'QR expired, waiting for new code...';
            }
        } catch(_) {}
    };

    waQrSource.onerror = function() {
        if (statusEl) statusEl.textContent = gatewayRunning ? 'Waiting for WhatsApp channel...' : 'Gateway not running';
        if (codeEl) codeEl.innerHTML = '';
    };
}

function disconnectWhatsAppQR() {
    if (waQrSource) {
        waQrSource.close();
        waQrSource = null;
    }
    const statusEl = document.getElementById('waQrStatus');
    const codeEl = document.getElementById('waQrCode');
    if (statusEl) statusEl.textContent = 'Start the gateway to pair WhatsApp';
    if (codeEl) codeEl.innerHTML = '';
}

// ── Skills Panel ─────────────────────────────────────
let _skillsCache = null;

function renderSkills() {
    const panel = document.getElementById('panelSkills');
    if (!panel) return;

    let html = panelHeader(t('skills.title'), 'panelSkills');
    html += `<div class="panel-desc">${t('skills.desc')}</div>`;

    // Installed section
    html += `<h3 style="margin:18px 0 8px;">${t('skills.installed')}</h3>`;
    html += `<div id="skillsInstalledGrid"><div class="empty-state"><div class="empty-state-desc">${t('skills.loading')}</div></div></div>`;

    // Search section
    html += `<h3 style="margin:24px 0 8px;">${t('skills.searchInstall')}</h3>`;
    html += '<div style="display:flex;gap:8px;margin-bottom:14px;">';
    html += `<input type="text" id="skillSearchInput" class="form-input" placeholder="${t('skills.searchPlaceholder')}" style="flex:1" onkeydown="if(event.key==='Enter')searchSkillsUI()">`;
    html += `<button class="btn btn-sm btn-primary" onclick="searchSkillsUI()">${t('skills.search')}</button>`;
    html += '</div>';
    html += '<div id="skillsSearchResults"></div>';

    panel.innerHTML = html;
    fetchInstalledSkills();
}

async function fetchInstalledSkills() {
    const grid = document.getElementById('skillsInstalledGrid');
    if (!grid) return;
    try {
        const resp = await fetch('/api/skills');
        if (!resp.ok) throw new Error(await resp.text());
        const skills = await resp.json();
        _skillsCache = skills;

        if (!skills || skills.length === 0) {
            grid.innerHTML = `<div class="empty-state">
                <div class="empty-state-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M8 2l2 4h4l-3 3 1 4-4-2-4 2 1-4-3-3h4z"/></svg></div>
                <div class="empty-state-title">${t('skills.noSkills')}</div>
                <div class="empty-state-desc">${t('skills.noSkillsDesc')}</div>
            </div>`;
            return;
        }

        let html = '<div class="model-grid">';
        skills.forEach(s => {
            const sourceBadge = s.source === 'builtin' ? '<span class="badge-nokey">builtin</span>'
                : s.source === 'global' ? '<span class="badge-nokey">global</span>'
                : '<span class="badge-primary">workspace</span>';
            html += '<div class="model-card">';
            html += '<div class="model-card-head">';
            html += `<div class="model-name">${esc(s.name)}</div>`;
            html += sourceBadge;
            html += '</div>';
            if (s.description) html += `<div class="model-detail">${esc(s.description)}</div>`;
            html += '<div class="model-actions">';
            html += `<button class="btn btn-sm" onclick="showSkillDetail('${escAttr(s.name)}')">${t('skills.show')}</button>`;
            if (s.source !== 'builtin') {
                html += `<button class="btn btn-sm btn-danger" onclick="removeSkill('${escAttr(s.name)}')">${t('skills.remove')}</button>`;
            }
            html += '</div></div>';
        });
        html += '</div>';
        grid.innerHTML = html;
    } catch (e) {
        grid.innerHTML = `<div class="empty-state"><div class="empty-state-desc">${t('skills.loadFailed', { msg: esc(e.message) })}</div></div>`;
    }
}

async function showSkillDetail(name) {
    try {
        const resp = await fetch('/api/skills/show?name=' + encodeURIComponent(name));
        if (!resp.ok) throw new Error(await resp.text());
        const data = await resp.json();

        document.getElementById('skillDetailTitle').textContent = name;
        document.getElementById('skillDetailBody').innerHTML = `<pre style="white-space:pre-wrap;word-break:break-word;font-size:13px;line-height:1.5;margin:0;">${esc(data.content)}</pre>`;
        document.getElementById('skillDetailModal').classList.add('active');
    } catch (e) {
        showStatus(t('skills.showFailed', { msg: e.message }), 'error');
    }
}

function closeSkillDetailModal() {
    document.getElementById('skillDetailModal').classList.remove('active');
}

async function removeSkill(name) {
    if (!confirm(t('skills.removeConfirm', { name }))) return;
    try {
        const resp = await fetch('/api/skills/remove', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name }),
        });
        if (!resp.ok) throw new Error(await resp.text());
        showStatus(t('skills.removed', { name }), 'success');
        fetchInstalledSkills();
    } catch (e) {
        showStatus(t('skills.removeFailed', { msg: e.message }), 'error');
    }
}

async function searchSkillsUI() {
    const input = document.getElementById('skillSearchInput');
    const container = document.getElementById('skillsSearchResults');
    if (!input || !container) return;

    const query = input.value.trim();
    if (!query) { container.innerHTML = ''; return; }

    container.innerHTML = '<div class="empty-state"><div class="empty-state-desc">' + t('skills.searching') + '</div></div>';

    try {
        const resp = await fetch('/api/skills/search?q=' + encodeURIComponent(query) + '&limit=10');
        if (!resp.ok) throw new Error(await resp.text());
        const results = await resp.json();

        if (!results || results.length === 0) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-desc">' + t('skills.noResults') + '</div></div>';
            return;
        }

        let html = '<div class="model-grid">';
        results.forEach(r => {
            const installed = _skillsCache && _skillsCache.some(s => s.name === r.slug);
            html += '<div class="model-card">';
            html += '<div class="model-card-head">';
            html += `<div class="model-name">${esc(r.display_name || r.slug)}</div>`;
            if (r.version) html += `<span class="model-protocol">${esc(r.version)}</span>`;
            html += '</div>';
            if (r.summary) html += `<div class="model-detail">${esc(r.summary)}</div>`;
            if (r.registry_name) html += `<div class="model-detail" style="font-size:11px;opacity:.6">Registry: ${esc(r.registry_name)}</div>`;
            html += '<div class="model-actions">';
            if (installed) {
                html += '<button class="btn btn-sm" disabled>' + t('skills.alreadyInstalled') + '</button>';
            } else {
                html += `<button class="btn btn-sm btn-primary" onclick="installSkill('${escAttr(r.slug)}', '${escAttr(r.registry_name)}', this)">${t('skills.install')}</button>`;
            }
            html += '</div></div>';
        });
        html += '</div>';
        container.innerHTML = html;
    } catch (e) {
        container.innerHTML = `<div class="empty-state"><div class="empty-state-desc">Search failed: ${esc(e.message)}</div></div>`;
    }
}

async function installSkill(slug, registry, btn) {
    if (btn) { btn.disabled = true; btn.textContent = t('skills.installing'); }
    try {
        const resp = await fetch('/api/skills/install', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ slug, registry }),
        });
        if (!resp.ok) throw new Error(await resp.text());
        const data = await resp.json();

        if (data.is_malware_blocked) {
            showStatus(t('skills.blockedMalware'), 'error');
            return;
        }
        if (data.is_suspicious) {
            showStatus(t('skills.installedSuspicious'), 'error');
        } else {
            showStatus(t('skills.installSuccess', { slug, version: data.version || '?' }), 'success');
        }
        fetchInstalledSkills();
        // Re-run search to update install buttons
        searchSkillsUI();
    } catch (e) {
        showStatus(t('skills.installFailed', { msg: e.message }), 'error');
        if (btn) { btn.disabled = false; btn.textContent = t('skills.install'); }
    }
}

// ── Init ────────────────────────────────────────────
applyI18n();
loadConfig();
loadAuthStatus().then(() => renderModels());

// Hash routing handles #auth and all other panel navigation automatically.
