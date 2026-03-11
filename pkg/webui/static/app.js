// ── Early declarations (needed before switchTab init) ────
let gatewayRunning = true;
let waQrSource = null;
let logPollTimer = null;

const panelHelp = {
    en: {
        panelModels: '<strong>Models</strong> are LLM configurations. Each model entry defines a provider endpoint (OpenAI, Anthropic, Ollama, etc.) with its credentials.<ul><li>Use <code>vendor/model-id</code> format (e.g. <code>openai/gpt-4o</code>)</li><li>Models without an API key appear grayed out</li><li>The <strong>Primary</strong> model is used by default for all agents</li><li>Multiple entries with same name are load-balanced</li></ul>',
        panelAuth: '<strong>Provider Auth</strong> manages OAuth and API token credentials for cloud providers.<ul><li><strong>OpenAI</strong>: Device code flow (no browser redirect needed)</li><li><strong>Anthropic</strong>: Paste your API key directly</li><li><strong>Google</strong>: Browser OAuth flow</li></ul>Credentials are stored in <code>~/.ok/auth.json</code>.',
        panelAgents: '<strong>Agents</strong> are autonomous AI assistants. Each agent has its own model, workspace, tools, and conversation context.<ul><li><strong>Defaults</strong> apply to all agents unless overridden</li><li><strong>Workspace</strong>: directory where the agent operates</li><li><strong>Max tokens</strong>: context window limit per request</li><li><strong>Summarize threshold</strong>: messages before auto-summarization</li></ul>',
        panelBindings: '<strong>Bindings</strong> route messages from channels to specific agents. Without bindings, all messages go to the default agent.<ul><li>Match by <strong>channel</strong>, <strong>peer</strong> (user), <strong>guild</strong> (server), or <strong>team</strong></li><li>Priority: peer > guild > team > account > channel wildcard > default</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> channel connects your bot to Telegram.<ul><li>Get your bot token from <strong>@BotFather</strong></li><li>Enable the channel and paste the token</li><li>Supports groups, DMs, inline commands, and media</li></ul>',
        panelCh_discord: '<strong>Discord</strong> channel connects to a Discord bot.<ul><li>Create a bot in the <strong>Discord Developer Portal</strong></li><li><strong>Mention Only</strong>: bot responds only when @mentioned in channels</li><li>Supports threads, reactions, and file attachments</li></ul>',
        panelCh_slack: '<strong>Slack</strong> channel connects via Socket Mode.<ul><li>You need both a <strong>Bot Token</strong> (xoxb-) and <strong>App Token</strong> (xapp-)</li><li>Create a Slack App with Socket Mode enabled</li><li>Supports threads, DMs, and channel messages</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> connects directly via whatsmeow.<ul><li>Start the gateway, then scan the QR code below with WhatsApp &gt; Linked Devices</li><li>QR refreshes automatically every ~20 seconds</li><li>Once paired, the session persists across restarts</li></ul>',
        panelSkills: '<strong>Skills</strong> are reusable prompt modules that extend agent capabilities.<ul><li><strong>Workspace</strong> skills are project-level (highest priority)</li><li><strong>Global</strong> skills are in <code>~/.ok/skills/</code></li><li><strong>Builtin</strong> skills ship with OK</li><li>Search and install skills from the ClawHub registry</li></ul>',
        panelMCP: '<strong>MCP Servers</strong> extend agent capabilities with external tools via the Model Context Protocol.<ul><li><strong>stdio</strong>: runs a local command (e.g. <code>npx</code>)</li><li><strong>http</strong>: connects to a remote SSE endpoint</li><li>Tools discovered from MCP servers become available to all agents</li></ul>',
        panelToolSettings: '<strong>Tool Settings</strong> control which tools agents can use and their limits.<ul><li>Toggle individual tools on/off</li><li><strong>Shell Exec</strong>: configure timeout and deny patterns for safety</li><li><strong>Path Restrictions</strong>: limit file read/write to specific directories</li><li><strong>Cron</strong>: scheduled task execution</li></ul>',
        panelRAG: '<strong>RAG</strong> (Retrieval-Augmented Generation) gives agents long-term semantic memory.<ul><li>Requires an embeddings endpoint (OpenAI, Ollama, vLLM)</li><li>Past conversations are indexed and retrieved based on similarity</li><li><strong>Top K</strong>: number of relevant memories to retrieve</li><li><strong>Min Similarity</strong>: threshold to filter weak matches</li></ul>',
        panelGateway: '<strong>Gateway</strong> is the main server process that connects channels to agents.<ul><li><strong>Host</strong>: bind address (127.0.0.1 for local, 0.0.0.0 for network)</li><li><strong>Port</strong>: HTTP port for API and web UI</li></ul>',
        panelSession: '<strong>Session</strong> controls how conversation history is scoped and linked.<ul><li><strong>Per Channel Peer</strong>: each user per channel gets their own history</li><li><strong>Per Peer</strong>: same user across channels shares history</li><li><strong>Global</strong>: all conversations share one history</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> sends periodic health check messages to a configured channel.<ul><li>Useful for monitoring that your gateway is alive</li><li>Minimum interval: 5 minutes</li></ul>',
        panelDevices: '<strong>Devices</strong> enables hardware device monitoring on Linux.<ul><li><strong>Monitor USB</strong>: detect USB device plug/unplug events</li><li>Agents can react to hardware changes</li></ul>',
        panelDebug: '<strong>Debug Mode</strong> enables verbose logging for troubleshooting.<ul><li>Shows detailed request/response data</li><li>Logs tool call inputs and outputs</li><li>Takes effect on next gateway restart</li></ul>',
        panelChat: '<strong>Chat</strong> lets you test your agents directly from the web UI.<ul><li>Connects via WebSocket to the running gateway</li><li>Messages go to the default agent</li><li>Status dot shows connection state</li></ul>',
        panelLogs: '<strong>Logs</strong> show real-time gateway output.<ul><li>Filter by <strong>level</strong>, <strong>component</strong>, or text search</li><li><strong>Debug toggle</strong>: enable verbose logging live</li><li>Data column shows structured JSON fields</li></ul>',
        panelRawJson: '<strong>Raw JSON</strong> editor gives direct access to the configuration file.<ul><li>All changes from other panels are saved to this file</li><li>Use <strong>Format</strong> to pretty-print the JSON</li><li>Invalid JSON is highlighted with a red border</li><li>File location shown in the header</li></ul>',
    },
    'pt-BR': {
        panelModels: '<strong>Modelos</strong> são configurações de LLM. Cada entrada define um endpoint de provedor (OpenAI, Anthropic, Ollama, etc.) com suas credenciais.<ul><li>Use o formato <code>vendor/model-id</code> (ex: <code>openai/gpt-4o</code>)</li><li>Modelos sem chave API aparecem esmaecidos</li><li>O modelo <strong>Primário</strong> é usado por padrão em todos os agentes</li><li>Múltiplas entradas com mesmo nome fazem balanceamento de carga</li></ul>',
        panelAuth: '<strong>Autenticação</strong> gerencia credenciais OAuth e tokens de API para provedores cloud.<ul><li><strong>OpenAI</strong>: Fluxo de código de dispositivo (sem redirecionamento do navegador)</li><li><strong>Anthropic</strong>: Cole sua chave API diretamente</li><li><strong>Google</strong>: Fluxo OAuth via navegador</li></ul>Credenciais são armazenadas em <code>~/.ok/auth.json</code>.',
        panelAgents: '<strong>Agentes</strong> são assistentes de IA autônomos. Cada agente tem seu próprio modelo, workspace, ferramentas e contexto.<ul><li><strong>Padrões</strong> se aplicam a todos os agentes, exceto quando sobrescritos</li><li><strong>Workspace</strong>: diretório onde o agente opera</li><li><strong>Máx. tokens</strong>: limite da janela de contexto por requisição</li><li><strong>Limite de resumo</strong>: mensagens antes do resumo automático</li></ul>',
        panelBindings: '<strong>Vínculos</strong> direcionam mensagens de canais para agentes específicos. Sem vínculos, todas as mensagens vão para o agente padrão.<ul><li>Filtre por <strong>canal</strong>, <strong>peer</strong> (usuário), <strong>guild</strong> (servidor), ou <strong>team</strong></li><li>Prioridade: peer > guild > team > conta > canal wildcard > padrão</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> conecta seu bot ao Telegram.<ul><li>Obtenha seu token do bot no <strong>@BotFather</strong></li><li>Ative o canal e cole o token</li><li>Suporta grupos, DMs, comandos inline e mídia</li></ul>',
        panelCh_discord: '<strong>Discord</strong> conecta a um bot Discord.<ul><li>Crie um bot no <strong>Discord Developer Portal</strong></li><li><strong>Apenas Menções</strong>: bot responde apenas quando @mencionado</li><li>Suporta threads, reações e anexos</li></ul>',
        panelCh_slack: '<strong>Slack</strong> conecta via Socket Mode.<ul><li>Você precisa de um <strong>Bot Token</strong> (xoxb-) e um <strong>App Token</strong> (xapp-)</li><li>Crie um Slack App com Socket Mode ativado</li><li>Suporta threads, DMs e mensagens de canal</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> conecta diretamente via whatsmeow.<ul><li>Inicie o gateway e escaneie o QR code com WhatsApp &gt; Aparelhos Conectados</li><li>O QR atualiza automaticamente a cada ~20 segundos</li><li>Uma vez pareado, a sessão persiste entre reinicializações</li></ul>',
        panelSkills: '<strong>Skills</strong> são módulos de prompt reutilizáveis que estendem as capacidades dos agentes.<ul><li>Skills de <strong>Workspace</strong> têm prioridade máxima</li><li>Skills <strong>Globais</strong> ficam em <code>~/.ok/skills/</code></li><li>Skills <strong>Builtin</strong> vêm com o OK</li><li>Busque e instale skills do registro ClawHub</li></ul>',
        panelMCP: '<strong>Servidores MCP</strong> estendem as capacidades dos agentes com ferramentas externas via Model Context Protocol.<ul><li><strong>stdio</strong>: executa um comando local (ex: <code>npx</code>)</li><li><strong>http</strong>: conecta a um endpoint SSE remoto</li><li>Ferramentas descobertas ficam disponíveis para todos os agentes</li></ul>',
        panelToolSettings: '<strong>Config. Ferramentas</strong> controla quais ferramentas os agentes podem usar e seus limites.<ul><li>Ative/desative ferramentas individuais</li><li><strong>Shell Exec</strong>: configure timeout e padrões de negação</li><li><strong>Restrições de Caminho</strong>: limite leitura/escrita a diretórios específicos</li><li><strong>Cron</strong>: execução de tarefas agendadas</li></ul>',
        panelRAG: '<strong>RAG</strong> (Geração Aumentada por Recuperação) dá aos agentes memória semântica de longo prazo.<ul><li>Requer um endpoint de embeddings (OpenAI, Ollama, vLLM)</li><li>Conversas passadas são indexadas e recuperadas por similaridade</li><li><strong>Top K</strong>: número de memórias relevantes a recuperar</li><li><strong>Similaridade Mín.</strong>: limiar para filtrar correspondências fracas</li></ul>',
        panelGateway: '<strong>Gateway</strong> é o processo principal que conecta canais a agentes.<ul><li><strong>Host</strong>: endereço de bind (127.0.0.1 para local, 0.0.0.0 para rede)</li><li><strong>Porta</strong>: porta HTTP para API e web UI</li></ul>',
        panelSession: '<strong>Sessão</strong> controla como o histórico de conversas é organizado.<ul><li><strong>Por Canal e Peer</strong>: cada usuário por canal tem seu próprio histórico</li><li><strong>Por Peer</strong>: mesmo usuário entre canais compartilha histórico</li><li><strong>Global</strong>: todas as conversas compartilham um histórico</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> envia mensagens periódicas de verificação de saúde.<ul><li>Útil para monitorar se seu gateway está ativo</li><li>Intervalo mínimo: 5 minutos</li></ul>',
        panelDevices: '<strong>Dispositivos</strong> habilita monitoramento de hardware no Linux.<ul><li><strong>Monitorar USB</strong>: detecta eventos de conexão/desconexão de dispositivos USB</li><li>Agentes podem reagir a mudanças de hardware</li></ul>',
        panelDebug: '<strong>Modo Debug</strong> habilita logs detalhados para diagnóstico.<ul><li>Mostra dados detalhados de requisição/resposta</li><li>Registra entradas e saídas de chamadas de ferramentas</li><li>Entra em vigor na próxima reinicialização do gateway</li></ul>',
        panelChat: '<strong>Chat</strong> permite testar seus agentes diretamente da web UI.<ul><li>Conecta via WebSocket ao gateway em execução</li><li>Mensagens vão para o agente padrão</li><li>O ponto de status mostra o estado da conexão</li></ul>',
        panelLogs: '<strong>Logs</strong> mostram a saída do gateway em tempo real.<ul><li>Filtre por <strong>nível</strong>, <strong>componente</strong>, ou texto</li><li><strong>Toggle Debug</strong>: ative logs detalhados ao vivo</li><li>Coluna Data mostra campos JSON estruturados</li></ul>',
        panelRawJson: '<strong>JSON Bruto</strong> dá acesso direto ao arquivo de configuração.<ul><li>Todas as alterações dos outros painéis são salvas neste arquivo</li><li>Use <strong>Formatar</strong> para indentar o JSON</li><li>JSON inválido é destacado com borda vermelha</li><li>Localização do arquivo mostrada no cabeçalho</li></ul>',
    },
    es: {
        panelModels: '<strong>Modelos</strong> son configuraciones de LLM. Cada entrada define un endpoint de proveedor (OpenAI, Anthropic, Ollama, etc.) con sus credenciales.<ul><li>Use el formato <code>vendor/model-id</code> (ej: <code>openai/gpt-4o</code>)</li><li>Los modelos sin clave API aparecen atenuados</li><li>El modelo <strong>Primario</strong> se usa por defecto en todos los agentes</li><li>Múltiples entradas con el mismo nombre hacen balanceo de carga</li></ul>',
        panelAuth: '<strong>Autenticación</strong> administra credenciales OAuth y tokens de API para proveedores cloud.<ul><li><strong>OpenAI</strong>: Flujo de código de dispositivo (sin redirección del navegador)</li><li><strong>Anthropic</strong>: Pegue su clave API directamente</li><li><strong>Google</strong>: Flujo OAuth en navegador</li></ul>Las credenciales se almacenan en <code>~/.ok/auth.json</code>.',
        panelAgents: '<strong>Agentes</strong> son asistentes de IA autónomos. Cada agente tiene su propio modelo, workspace, herramientas y contexto.<ul><li><strong>Predeterminados</strong> se aplican a todos los agentes, excepto cuando se sobreescriben</li><li><strong>Workspace</strong>: directorio donde opera el agente</li><li><strong>Máx. tokens</strong>: límite de ventana de contexto por solicitud</li><li><strong>Umbral de resumen</strong>: mensajes antes del resumen automático</li></ul>',
        panelBindings: '<strong>Vínculos</strong> dirigen mensajes de canales a agentes específicos. Sin vínculos, todos los mensajes van al agente predeterminado.<ul><li>Filtre por <strong>canal</strong>, <strong>peer</strong> (usuario), <strong>guild</strong> (servidor), o <strong>team</strong></li><li>Prioridad: peer > guild > team > cuenta > canal wildcard > predeterminado</li></ul>',
        panelCh_telegram: '<strong>Telegram</strong> conecta su bot a Telegram.<ul><li>Obtenga su token del bot en <strong>@BotFather</strong></li><li>Active el canal y pegue el token</li><li>Soporta grupos, DMs, comandos inline y media</li></ul>',
        panelCh_discord: '<strong>Discord</strong> conecta a un bot de Discord.<ul><li>Cree un bot en el <strong>Discord Developer Portal</strong></li><li><strong>Solo Menciones</strong>: el bot responde solo cuando es @mencionado</li><li>Soporta threads, reacciones y archivos adjuntos</li></ul>',
        panelCh_slack: '<strong>Slack</strong> conecta via Socket Mode.<ul><li>Necesita un <strong>Bot Token</strong> (xoxb-) y un <strong>App Token</strong> (xapp-)</li><li>Cree una Slack App con Socket Mode activado</li><li>Soporta threads, DMs y mensajes de canal</li></ul>',
        panelCh_whatsapp: '<strong>WhatsApp</strong> conecta directamente via whatsmeow.<ul><li>Inicie el gateway y escanee el código QR con WhatsApp &gt; Dispositivos Vinculados</li><li>El QR se actualiza automáticamente cada ~20 segundos</li><li>Una vez emparejado, la sesión persiste entre reinicios</li></ul>',
        panelSkills: '<strong>Skills</strong> son módulos de prompt reutilizables que extienden las capacidades de los agentes.<ul><li>Skills de <strong>Workspace</strong> tienen prioridad máxima</li><li>Skills <strong>Globales</strong> están en <code>~/.ok/skills/</code></li><li>Skills <strong>Builtin</strong> vienen con OK</li><li>Busque e instale skills del registro ClawHub</li></ul>',
        panelMCP: '<strong>Servidores MCP</strong> extienden las capacidades de los agentes con herramientas externas via Model Context Protocol.<ul><li><strong>stdio</strong>: ejecuta un comando local (ej: <code>npx</code>)</li><li><strong>http</strong>: conecta a un endpoint SSE remoto</li><li>Las herramientas descubiertas quedan disponibles para todos los agentes</li></ul>',
        panelToolSettings: '<strong>Config. Herramientas</strong> controla qué herramientas pueden usar los agentes y sus límites.<ul><li>Active/desactive herramientas individuales</li><li><strong>Shell Exec</strong>: configure timeout y patrones de denegación</li><li><strong>Restricciones de Ruta</strong>: limite lectura/escritura a directorios específicos</li><li><strong>Cron</strong>: ejecución de tareas programadas</li></ul>',
        panelRAG: '<strong>RAG</strong> (Generación Aumentada por Recuperación) da a los agentes memoria semántica a largo plazo.<ul><li>Requiere un endpoint de embeddings (OpenAI, Ollama, vLLM)</li><li>Conversaciones pasadas son indexadas y recuperadas por similitud</li><li><strong>Top K</strong>: número de memorias relevantes a recuperar</li><li><strong>Similitud Mín.</strong>: umbral para filtrar coincidencias débiles</li></ul>',
        panelGateway: '<strong>Gateway</strong> es el proceso principal que conecta canales a agentes.<ul><li><strong>Host</strong>: dirección de bind (127.0.0.1 para local, 0.0.0.0 para red)</li><li><strong>Puerto</strong>: puerto HTTP para API y web UI</li></ul>',
        panelSession: '<strong>Sesión</strong> controla cómo se organiza el historial de conversaciones.<ul><li><strong>Por Canal y Peer</strong>: cada usuario por canal tiene su propio historial</li><li><strong>Por Peer</strong>: mismo usuario entre canales comparte historial</li><li><strong>Global</strong>: todas las conversaciones comparten un historial</li></ul>',
        panelHeartbeat: '<strong>Heartbeat</strong> envía mensajes periódicos de verificación de salud.<ul><li>Útil para monitorear que su gateway está activo</li><li>Intervalo mínimo: 5 minutos</li></ul>',
        panelDevices: '<strong>Dispositivos</strong> habilita monitoreo de hardware en Linux.<ul><li><strong>Monitorear USB</strong>: detecta eventos de conexión/desconexión de dispositivos USB</li><li>Los agentes pueden reaccionar a cambios de hardware</li></ul>',
        panelDebug: '<strong>Modo Debug</strong> habilita logs detallados para diagnóstico.<ul><li>Muestra datos detallados de solicitud/respuesta</li><li>Registra entradas y salidas de llamadas de herramientas</li><li>Aplica en el próximo reinicio del gateway</li></ul>',
        panelChat: '<strong>Chat</strong> permite probar sus agentes directamente desde la web UI.<ul><li>Conecta via WebSocket al gateway en ejecución</li><li>Los mensajes van al agente predeterminado</li><li>El punto de estado muestra el estado de la conexión</li></ul>',
        panelLogs: '<strong>Logs</strong> muestran la salida del gateway en tiempo real.<ul><li>Filtre por <strong>nivel</strong>, <strong>componente</strong>, o texto</li><li><strong>Toggle Debug</strong>: active logs detallados en vivo</li><li>La columna Data muestra campos JSON estructurados</li></ul>',
        panelRawJson: '<strong>JSON Crudo</strong> da acceso directo al archivo de configuración.<ul><li>Todos los cambios de otros paneles se guardan en este archivo</li><li>Use <strong>Formatear</strong> para indentar el JSON</li><li>JSON inválido se resalta con borde rojo</li><li>Ubicación del archivo mostrada en el encabezado</li></ul>',
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
        // Model fields
        'field.modelName': 'Model Name',
        'field.modelId': 'Model ID',
        'field.modelIdHint': 'Format: protocol/model-id',
        'field.apiKey': 'API Key',
        'field.apiBase': 'API Base',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Auth Method',
        'field.connectMode': 'Connect Mode',
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
        // Channel
        'ch.configure': 'Configure {name} channel settings.',
        'ch.docLink': 'Configuration Guide',
        'ch.accessControl': 'Access Control',
        'ch.allowFrom': 'Allow From (User IDs)',
        'ch.allowedGroups': 'Allowed Groups',
        'ch.allowedContacts': 'Allowed Contacts',
        'ch.addItem': '+ Add',
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
        'agents.desc': 'Configure agent defaults and individual agent overrides.',
        'agents.defaults': 'Default Settings',
        'agents.agentList': 'Agent List',
        'agents.addAgent': '+ Add Agent',
        'agents.editAgent': 'Edit Agent',
        'agents.addAgentTitle': 'Add Agent',
        'agents.noAgents': 'No custom agents. The default agent handles all messages.',
        'agents.saveDefaults': 'Save Defaults',
        'agents.defaultsSaved': 'Agent defaults saved',
        'agents.deleteConfirm': 'Delete agent "{name}"?',
        'agents.idRequired': 'Agent ID is required',
        'agents.duplicateId': 'Agent with ID "{id}" already exists',
        'agents.restrictWorkspace': 'Restrict to Workspace',
        'agents.allowReadOutside': 'Allow Read Outside',
        'agents.modelName': 'Model Name',
        'agents.modelNameHint': 'Must match a model_name from Models panel',
        'agents.maxTokens': 'Max Tokens',
        'agents.maxToolIter': 'Max Tool Iterations',
        'agents.summarizeThreshold': 'Summarize Threshold',
        'agents.summarizeThresholdHint': 'Messages before summarization triggers',
        'agents.summarizeTokenPct': 'Summarize Token %',
        'agents.maxMediaSize': 'Max Media Size (bytes)',
        'agents.maxMediaSizeHint': '0 = default (20MB)',
        'agents.defaultAgent': 'Default Agent',
        'agents.friendlyName': 'Friendly name',
        'agents.inheritsDefaults': 'Inherits from defaults',
        'agents.modelHint': 'Model name (must match a model_name from Models)',
        // Bindings
        'bindings.title': 'Bindings',
        'bindings.desc': 'Map agents to channels. Bindings control which agent handles messages from specific channels or chat IDs.',
        'bindings.addBinding': '+ Add Binding',
        'bindings.addBindingTitle': 'Add Binding',
        'bindings.editBinding': 'Edit Binding',
        'bindings.noBindings': 'No bindings configured. The default agent handles all messages.',
        'bindings.deleteConfirm': 'Delete this binding?',
        'bindings.required': 'Agent ID and Channel are required',
        'bindings.agentId': 'Agent ID',
        'bindings.channel': 'Channel',
        'bindings.accountId': 'Account ID',
        'bindings.peerKind': 'Peer Kind',
        'bindings.peerId': 'Peer ID',
        'bindings.guildId': 'Guild ID',
        'bindings.teamId': 'Team ID',
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
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Offline',
        'chat.placeholder': 'Type a message...',
        'chat.send': 'Send',
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
        // Model fields
        'field.modelName': 'Nome do Modelo',
        'field.modelId': 'ID do Modelo',
        'field.modelIdHint': 'Formato: protocolo/id-modelo',
        'field.apiKey': 'Chave API',
        'field.apiBase': 'URL Base da API',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Método de Autenticação',
        'field.connectMode': 'Modo de Conexão',
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
        // Channel
        'ch.configure': 'Configurar canal {name}.',
        'ch.docLink': 'Guia de Configuração',
        'ch.accessControl': 'Controle de Acesso',
        'ch.allowFrom': 'Permitir de (IDs de Usuário)',
        'ch.allowedGroups': 'Grupos Permitidos',
        'ch.allowedContacts': 'Contatos Permitidos',
        'ch.addItem': '+ Adicionar',
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
        'agents.desc': 'Configure padrões de agentes e sobreposições individuais.',
        'agents.defaults': 'Configurações Padrão',
        'agents.agentList': 'Lista de Agentes',
        'agents.addAgent': '+ Adicionar Agente',
        'agents.editAgent': 'Editar Agente',
        'agents.addAgentTitle': 'Adicionar Agente',
        'agents.noAgents': 'Nenhum agente personalizado. O agente padrão trata todas as mensagens.',
        'agents.saveDefaults': 'Salvar Padrões',
        'agents.defaultsSaved': 'Padrões de agente salvos',
        'agents.deleteConfirm': 'Excluir agente "{name}"?',
        'agents.idRequired': 'ID do agente é obrigatório',
        'agents.duplicateId': 'Agente com ID "{id}" já existe',
        'agents.restrictWorkspace': 'Restringir ao Workspace',
        'agents.allowReadOutside': 'Permitir Leitura Externa',
        'agents.modelName': 'Nome do Modelo',
        'agents.modelNameHint': 'Deve corresponder a um model_name do painel Modelos',
        'agents.maxTokens': 'Máx. Tokens',
        'agents.maxToolIter': 'Máx. Iterações de Ferramentas',
        'agents.summarizeThreshold': 'Limite de Resumo',
        'agents.summarizeThresholdHint': 'Mensagens antes do resumo automático',
        'agents.summarizeTokenPct': '% Tokens para Resumo',
        'agents.maxMediaSize': 'Tamanho Máx. de Mídia (bytes)',
        'agents.maxMediaSizeHint': '0 = padrão (20MB)',
        'agents.defaultAgent': 'Agente Padrão',
        'agents.friendlyName': 'Nome amigável',
        'agents.inheritsDefaults': 'Herda dos padrões',
        'agents.modelHint': 'Nome do modelo (deve corresponder a um model_name de Modelos)',
        // Bindings
        'bindings.title': 'Vínculos',
        'bindings.desc': 'Mapeie agentes para canais. Vínculos controlam qual agente responde mensagens de canais ou chats específicos.',
        'bindings.addBinding': '+ Adicionar Vínculo',
        'bindings.addBindingTitle': 'Adicionar Vínculo',
        'bindings.editBinding': 'Editar Vínculo',
        'bindings.noBindings': 'Nenhum vínculo configurado. O agente padrão trata todas as mensagens.',
        'bindings.deleteConfirm': 'Excluir este vínculo?',
        'bindings.required': 'ID do Agente e Canal são obrigatórios',
        'bindings.agentId': 'ID do Agente',
        'bindings.channel': 'Canal',
        'bindings.accountId': 'ID da Conta',
        'bindings.peerKind': 'Tipo de Peer',
        'bindings.peerId': 'ID do Peer',
        'bindings.guildId': 'ID do Guild',
        'bindings.teamId': 'ID do Team',
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
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Desconectado',
        'chat.placeholder': 'Digite uma mensagem...',
        'chat.send': 'Enviar',
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
        // Model fields
        'field.modelName': 'Nombre del Modelo',
        'field.modelId': 'ID del Modelo',
        'field.modelIdHint': 'Formato: protocolo/id-modelo',
        'field.apiKey': 'Clave API',
        'field.apiBase': 'URL Base de API',
        'field.proxy': 'Proxy',
        'field.authMethod': 'Método de Autenticación',
        'field.connectMode': 'Modo de Conexión',
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
        // Channel
        'ch.configure': 'Configurar canal {name}.',
        'ch.docLink': 'Guía de Configuración',
        'ch.accessControl': 'Control de Acceso',
        'ch.allowFrom': 'Permitir Desde (IDs de Usuario)',
        'ch.allowedGroups': 'Grupos Permitidos',
        'ch.allowedContacts': 'Contactos Permitidos',
        'ch.addItem': '+ Agregar',
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
        'agents.desc': 'Configure valores predeterminados de agentes y sobrecargas individuales.',
        'agents.defaults': 'Configuración Predeterminada',
        'agents.agentList': 'Lista de Agentes',
        'agents.addAgent': '+ Agregar Agente',
        'agents.editAgent': 'Editar Agente',
        'agents.addAgentTitle': 'Agregar Agente',
        'agents.noAgents': 'Sin agentes personalizados. El agente predeterminado maneja todos los mensajes.',
        'agents.saveDefaults': 'Guardar Predeterminados',
        'agents.defaultsSaved': 'Valores predeterminados de agente guardados',
        'agents.deleteConfirm': 'Eliminar agente "{name}"?',
        'agents.idRequired': 'ID del agente es requerido',
        'agents.duplicateId': 'Ya existe un agente con ID "{id}"',
        'agents.restrictWorkspace': 'Restringir al Workspace',
        'agents.allowReadOutside': 'Permitir Lectura Externa',
        'agents.modelName': 'Nombre del Modelo',
        'agents.modelNameHint': 'Debe coincidir con un model_name del panel Modelos',
        'agents.maxTokens': 'Máx. Tokens',
        'agents.maxToolIter': 'Máx. Iteraciones de Herramientas',
        'agents.summarizeThreshold': 'Umbral de Resumen',
        'agents.summarizeThresholdHint': 'Mensajes antes del resumen automático',
        'agents.summarizeTokenPct': '% Tokens para Resumen',
        'agents.maxMediaSize': 'Tamaño Máx. de Media (bytes)',
        'agents.maxMediaSizeHint': '0 = predeterminado (20MB)',
        'agents.defaultAgent': 'Agente Predeterminado',
        'agents.friendlyName': 'Nombre amigable',
        'agents.inheritsDefaults': 'Hereda de los predeterminados',
        'agents.modelHint': 'Nombre del modelo (debe coincidir con un model_name de Modelos)',
        // Bindings
        'bindings.title': 'Vínculos',
        'bindings.desc': 'Mapee agentes a canales. Los vínculos controlan qué agente maneja mensajes de canales o chats específicos.',
        'bindings.addBinding': '+ Agregar Vínculo',
        'bindings.addBindingTitle': 'Agregar Vínculo',
        'bindings.editBinding': 'Editar Vínculo',
        'bindings.noBindings': 'Sin vínculos configurados. El agente predeterminado maneja todos los mensajes.',
        'bindings.deleteConfirm': '¿Eliminar este vínculo?',
        'bindings.required': 'ID del Agente y Canal son requeridos',
        'bindings.agentId': 'ID del Agente',
        'bindings.channel': 'Canal',
        'bindings.accountId': 'ID de Cuenta',
        'bindings.peerKind': 'Tipo de Peer',
        'bindings.peerId': 'ID del Peer',
        'bindings.guildId': 'ID del Guild',
        'bindings.teamId': 'ID del Team',
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
        // Chat
        'chat.title': 'Chat',
        'chat.offline': 'Desconectado',
        'chat.placeholder': 'Escriba un mensaje...',
        'chat.send': 'Enviar',
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
        renderModels(); renderAgents(); renderBindings();
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
    document.getElementById('authDesc').innerHTML = t('auth.desc');
    // Sync selector
    const sel = document.getElementById('langSelect');
    if (sel) sel.value = currentLang;
}

// ── State ───────────────────────────────────────────
let configData = null;
let configPath = '';
let authPollTimer = null;
let editingModelIndex = -1;

// ── Channel schemas ─────────────────────────────────
const channelSchemas = {
    telegram: {
        title: 'Telegram', configKey: 'telegram', docSlug: 'telegram',
        fields: [
            { key: 'token', label: 'Bot Token', type: 'password', placeholder: 'Telegram bot token from @BotFather' },
            { key: 'proxy', label: 'Proxy', type: 'text', placeholder: 'http://proxy:port' },
        ]
    },
    discord: {
        title: 'Discord', configKey: 'discord', docSlug: 'discord',
        fields: [
            { key: 'token', label: 'Bot Token', type: 'password', placeholder: 'Discord bot token' },
            { key: 'mention_only', label: 'ch.mentionOnly', type: 'toggle', hint: 'ch.mentionOnlyHint', i18nLabel: true },
        ]
    },
    slack: {
        title: 'Slack', configKey: 'slack', docSlug: 'slack',
        fields: [
            { key: 'bot_token', label: 'Bot Token', type: 'password', placeholder: 'xoxb-...' },
            { key: 'app_token', label: 'App Token', type: 'password', placeholder: 'xapp-...' },
        ]
    },
    whatsapp: {
        title: 'WhatsApp', configKey: 'whatsapp', docSlug: null,
        fields: [
            { key: 'session_store_path', label: 'Session Store Path', type: 'text', placeholder: '~/.ok/workspace/whatsapp' },
            { key: 'allow_self', label: 'Allow Self Chat', type: 'toggle', hint: 'Respond to messages from the connected number' },
        ]
    },
};

// ── Tab Navigation ──────────────────────────────────
const tabDefs = {
    core: [
        { panel: 'panelModels', label: 'Models', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M8 2L2 5l6 3 6-3Zm0 6L2 11l6 3 6-3Z"/></svg>' },
        { panel: 'panelAuth', label: 'Auth', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="3" y="7" width="10" height="7" rx="2"/><path d="M5.5 7V5a2.5 2.5 0 015 0v2"/></svg>' },
        { panel: 'panelAgents', label: 'Agents', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="5" r="3"/><path d="M3 14c0-2.8 2.2-5 5-5s5 2.2 5 5"/></svg>' },
        { panel: 'panelBindings', label: 'Bindings', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M6 4h7M6 8h7M6 12h7M3 4h.01M3 8h.01M3 12h.01"/></svg>' },
        { panel: 'panelSkills', label: 'Skills', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M8 2l2 4h4l-3 3 1 4-4-2-4 2 1-4-3-3h4z"/></svg>' },
    ],
    channels: [
        { panel: 'panelCh_telegram', label: 'Telegram', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"><path d="M14 2L7 9m7-7l-4 12-3-6L1 6z"/></svg>' },
        { panel: 'panelCh_discord', label: 'Discord', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5.5 3S7 2 8 2s2.5 1 2.5 1M3.5 5.5C3 7 2.5 9 3 11c.3 1 1.5 2 2 2.5l1-1.5m5-6.5c.5 1.5 1 3.5.5 5.5-.3 1-1.5 2-2 2.5l-1-1.5"/><circle cx="6" cy="8.5" r=".8" fill="currentColor" stroke="none"/><circle cx="10" cy="8.5" r=".8" fill="currentColor" stroke="none"/></svg>' },
        { panel: 'panelCh_slack', label: 'Slack', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"><path d="M3 6h10M3 10h10M6 3v10M10 3v10"/></svg>' },
        { panel: 'panelCh_whatsapp', label: 'WhatsApp', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M6.5 6c0-.5.5-.8.8-.5l.5.5c.3.5 0 1.2 0 1.2s.8.8 1.2 1.2c0 0 .7-.3 1.2 0l.5.5c.3.3 0 .8-.5.8C8.5 10.5 5.5 7.5 6.5 6z"/></svg>' },
    ],
    system: [
        { panel: 'panelMCP', label: 'MCP', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="3" width="12" height="10" rx="2"/><path d="M5 8h6M8 6v4"/></svg>' },
        { panel: 'panelToolSettings', label: 'Tools', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M6.5 2v3M6.5 9v5M11.5 2v7M11.5 13v1"/><circle cx="6.5" cy="7" r="2"/><circle cx="11.5" cy="11" r="2"/></svg>' },
        { panel: 'panelRAG', label: 'RAG', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 4h12M2 8h8M2 12h10"/><circle cx="13" cy="10" r="2"/></svg>' },
        { panel: 'panelGateway', label: 'Gateway', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="2" width="12" height="5" rx="1.5"/><rect x="2" y="9" width="12" height="5" rx="1.5"/><circle cx="5" cy="4.5" r="1" fill="currentColor" stroke="none"/><circle cx="5" cy="11.5" r="1" fill="currentColor" stroke="none"/></svg>' },
        { panel: 'panelSession', label: 'Session', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M8 2a6 6 0 110 12A6 6 0 018 2zM8 5v3l2 2"/></svg>' },
        { panel: 'panelHeartbeat', label: 'Heartbeat', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 8h2l2-4 3 8 2-4h3"/></svg>' },
        { panel: 'panelDevices', label: 'Devices', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="3" y="2" width="10" height="8" rx="1.5"/><path d="M6 13h4M8 10v3"/></svg>' },
        { panel: 'panelDebug', label: 'Debug', icon: '<svg class="si" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="3"/><path d="M8 2v2M8 12v2M2 8h2M12 8h2M3.8 3.8l1.4 1.4M10.8 10.8l1.4 1.4M3.8 12.2l1.4-1.4M10.8 5.2l1.4-1.4"/></svg>' },
    ],
    // Utility tabs map directly to panels (no sub-tabs)
    chat: [{ panel: 'panelChat' }],
    logs: [{ panel: 'panelLogs' }],
    json: [{ panel: 'panelRawJson' }],
};

// Track which sub-tab was last active per group
const lastSubTab = { core: 'panelModels', channels: 'panelCh_telegram', system: 'panelMCP' };

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
    if (panelId === 'panelBindings') renderBindings();
    if (panelId === 'panelToolSettings') renderToolSettings();
    if (panelId === 'panelRAG') renderRAG();
    if (panelId === 'panelGateway') renderGateway();
    if (panelId === 'panelSession') renderSession();
    if (panelId === 'panelHeartbeat') renderHeartbeat();
    if (panelId === 'panelDevices') renderDevices();
    if (panelId === 'panelSkills') renderSkills();
    if (panelId === 'panelDebug') renderDebug();
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
if (!navigateToHash()) switchTab('core');

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
        if (m.api_base) html += `<div class="model-detail"><strong>API Base:</strong> ${esc(m.api_base)}</div>`;
        if (m.api_key) html += `<div class="model-detail"><strong>API Key:</strong> ${maskKey(m.api_key)}</div>`;
        if (m.auth_method) html += `<div class="model-detail"><strong>Auth:</strong> ${esc(m.auth_method)}</div>`;
        if (m.proxy) html += `<div class="model-detail"><strong>Proxy:</strong> ${esc(m.proxy)}</div>`;

        html += `<div class="model-actions">`;
        html += `<button class="btn btn-sm" onclick="showEditModelModal(${idx})">${t('edit')}</button>`;
        if (available && !isPrimary) {
            html += `<button class="btn btn-sm btn-success" onclick="setPrimaryModel(${idx})">${t('models.setPrimary')}</button>`;
        }
        html += `<button class="btn btn-sm btn-danger" onclick="deleteModel(${idx})">${t('delete')}</button>`;
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

// ── Model Modal ─────────────────────────────────────
const modelFieldsRequired = [
    { key: 'model_name', labelKey: 'field.modelName', type: 'text', placeholder: 'e.g. gpt-4o', required: true },
    { key: 'model', labelKey: 'field.modelId', type: 'text', placeholder: 'e.g. openai/gpt-4o', required: true, hintKey: 'field.modelIdHint' },
    { key: 'api_key', labelKey: 'field.apiKey', type: 'password', placeholder: 'API key' },
    { key: 'api_base', labelKey: 'field.apiBase', type: 'text', placeholder: 'https://api.openai.com/v1' },
];
const modelFieldsOptional = [
    { key: 'proxy', labelKey: 'field.proxy', type: 'text', placeholder: 'http://proxy:port' },
    { key: 'auth_method', labelKey: 'field.authMethod', type: 'text', placeholder: 'oauth / token' },
    { key: 'connect_mode', labelKey: 'field.connectMode', type: 'text', placeholder: 'stdio / grpc' },
    { key: 'workspace', labelKey: 'field.workspace', type: 'text', placeholder: 'Workspace path' },
    { key: 'rpm', labelKey: 'field.rpm', type: 'number', placeholder: 'RPM' },
    { key: 'request_timeout', labelKey: 'field.requestTimeout', type: 'number', placeholder: 'Seconds' },
];
const modelFields = [...modelFieldsRequired, ...modelFieldsOptional];

function showEditModelModal(idx) {
    editingModelIndex = idx;
    const m = configData.model_list[idx];
    document.getElementById('modalTitle').textContent = t('models.editModel') + ': ' + m.model_name;
    renderModalBody(m);
    document.getElementById('modelModal').classList.add('active');
}

function showAddModelModal() {
    editingModelIndex = -1;
    document.getElementById('modalTitle').textContent = t('models.addModel');
    renderModalBody({});
    document.getElementById('modelModal').classList.add('active');
}

function closeModelModal() {
    document.getElementById('modelModal').classList.remove('active');
}

function renderModalBody(data) {
    const hasOptionalValues = modelFieldsOptional.some(f => {
        const v = data[f.key];
        return v !== undefined && v !== null && v !== '' && v !== 0;
    });

    function renderField(f) {
        const val = data[f.key] !== undefined && data[f.key] !== null ? data[f.key] : '';
        const label = t(f.labelKey);
        let h = `<div class="form-group">`;
        h += `<label class="form-label">${label}${f.required ? ' *' : ''}</label>`;
        h += `<input class="form-input ${f.type === 'number' ? 'form-input-number' : ''}" `;
        h += `type="${f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'}" `;
        h += `data-field="${f.key}" value="${esc(String(val))}" placeholder="${f.placeholder || ''}">`;
        if (f.hintKey) h += `<div class="form-hint">${t(f.hintKey)}</div>`;
        h += `</div>`;
        return h;
    }

    let html = '';
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
    const inputs = document.querySelectorAll('#modalBody input[data-field]');
    const obj = {};
    inputs.forEach(input => {
        const key = input.dataset.field;
        let val = input.value.trim();
        if (input.type === 'number' && val) val = parseInt(val, 10) || 0;
        if (val !== '' && val !== 0) obj[key] = val;
        else if (key === 'model_name' || key === 'model') obj[key] = val;
    });

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



// ── Channel Forms ───────────────────────────────────
function renderChannelForm(chKey) {
    const schema = channelSchemas[chKey];
    if (!schema) return;
    const panel = document.getElementById('panelCh_' + chKey);
    const chData = (configData && configData.channels && configData.channels[schema.configKey]) || {};

    let html = panelHeader(schema.title, 'panelCh_' + chKey);
    html += `<div class="panel-desc">${t('ch.configure', { name: schema.title })}`;
    if (schema.docSlug) {
        const docBase = 'https://docs.ok.io/docs/channels/';
        html += ` <a class="doc-link" href="${docBase}${schema.docSlug}" target="_blank" rel="noopener noreferrer">\u{1F4D6} ${t('ch.docLink')}</a>`;
    }
    html += `</div>`;

    // WhatsApp: side-by-side layout (form left, QR right)
    if (chKey === 'whatsapp') {
        html += `<div style="display:flex; flex-wrap:wrap; gap:24px 32px; align-items:flex-start;">`;
        html += `<div style="flex:1 1 300px; min-width:0;">`;
    }

    html += `<div class="channel-form" id="chForm_${chKey}">`;

    // Enabled toggle
    html += `<div class="toggle-row">`;
    html += `<div class="toggle ${chData.enabled ? 'on' : ''}" id="chToggle_${chKey}" onclick="toggleChannelEnabled('${chKey}', this)"></div>`;
    html += `<span class="toggle-label">${t('enabled')}</span>`;
    html += `</div>`;

    schema.fields.forEach(f => {
        const label = f.i18nLabel ? t(f.label) : f.label;
        if (f.type === 'toggle') {
            const hint = f.i18nLabel && f.hint ? t(f.hint) : (f.hint || '');
            html += `<div class="toggle-row">`;
            html += `<div class="toggle ${chData[f.key] ? 'on' : ''}" data-chfield="${f.key}" onclick="this.classList.toggle('on')"></div>`;
            html += `<span class="toggle-label">${label}</span>`;
            if (hint) html += `<span class="form-hint" style="margin-left:8px;">${hint}</span>`;
            html += `</div>`;
        } else if (f.type === 'array') {
            const arr = chData[f.key] || [];
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
            const val = chData[f.key] !== undefined && chData[f.key] !== null ? chData[f.key] : '';
            html += `<div class="form-group">`;
            html += `<label class="form-label">${label}</label>`;
            html += `<input class="form-input ${f.type === 'number' ? 'form-input-number' : ''}" `;
            html += `type="${f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'}" `;
            html += `data-chfield="${f.key}" value="${esc(String(val))}" placeholder="${f.placeholder || ''}">`;
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
        html += `<div id="waQrCode" style="margin-bottom:8px;"></div>`;
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

function toggleChannelEnabled(chKey, el) { el.classList.toggle('on'); }

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
            if (el) chObj[f.key] = el.classList.contains('on');
        } else if (f.type === 'array') {
            const container = form.querySelector(`.array-editor[data-chfield="${f.key}"]`);
            if (container) {
                const vals = [];
                container.querySelectorAll('.array-row input').forEach(input => {
                    const v = input.value.trim();
                    if (v) vals.push(v);
                });
                chObj[f.key] = vals;
            }
        } else {
            const input = form.querySelector(`input[data-chfield="${f.key}"]`);
            if (input) {
                let val = input.value.trim();
                if (f.type === 'number' && val) {
                    val = parseInt(val, 10);
                    if (isNaN(val)) val = 0;
                }
                chObj[f.key] = val === '' ? (f.type === 'number' ? 0 : '') : val;
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

    ['openai', 'anthropic', 'google-antigravity'].forEach(name => {
        const badge = document.getElementById('badge-' + name);
        const details = document.getElementById('details-' + name);
        const actions = document.getElementById('actions-' + name);
        const p = providerMap[name];

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
            details.innerHTML = dh;
            actions.innerHTML = `<button class="btn btn-sm btn-danger" onclick="logoutProvider('${name}')">${t('auth.logout')}</button>`;
        } else {
            badge.className = 'provider-badge badge-none';
            badge.textContent = t('auth.notLoggedIn');
            details.innerHTML = '';
            if (name === 'openai') {
                actions.innerHTML = `<button class="btn btn-sm btn-primary" onclick="loginProvider('openai')">${t('auth.loginDevice')}</button>`;
            } else if (name === 'anthropic') {
                actions.innerHTML = `<button class="btn btn-sm btn-primary" onclick="showTokenInput('anthropic')">${t('auth.loginToken')}</button>`;
            } else {
                actions.innerHTML = `<button class="btn btn-sm btn-primary" onclick="loginProvider('google-antigravity')">${t('auth.loginOAuth')}</button>`;
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
    if (m.api_key) return true;
    if (m.auth_method === 'oauth') {
        const protocol = m.model ? m.model.split('/')[0] : '';
        const providerName = protocol === 'google-antigravity' ? 'google-antigravity' : protocol;
        const authInfo = authProviderMap[providerName];
        return !!(authInfo && authInfo.status === 'active');
    }
    if (m.auth_method) return true;
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
        showStatus('Config reload triggered', 'success');
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
let logComponentsLoaded = false;

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
    // Show HH:MM:SS, full in title
    const timePart = ts.includes('T') ? ts.split('T')[1].replace('Z', '').substring(0, 8) : ts;
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
    if (!confirm('Delete all log files? This cannot be undone.')) return;
    try {
        const res = await fetch('/api/logs', { method: 'DELETE' });
        if (!res.ok) throw new Error(await res.text());
        clearLogDisplay();
        loadLogComponents();
        showStatus('All logs deleted', 'success');
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
    logComponentsLoaded = true;
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
            if (msgId) chatBotMessages[msgId] = el;
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
    let h = '<div class="form-group">';
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
    let h = '<div class="form-group">';
    h += `<label class="form-label">${esc(label)}</label>`;
    h += `<div class="array-editor" data-field="${key}" data-placeholder="${opts.placeholder || ''}">`;
    arr.forEach(v => {
        h += '<div class="array-row">';
        h += `<input class="form-input" type="text" value="${esc(String(v))}" placeholder="${opts.placeholder || ''}">`;
        h += '<button class="btn btn-sm btn-danger" onclick="removeArrayRow(this)">&times;</button>';
        h += '</div>';
    });
    h += `<div class="array-add" onclick="addArrayRow(this.parentElement)">+ Add</div>`;
    h += '</div></div>';
    return h;
}

function renderKVEditor(key, label, map, opts = {}) {
    const entries = map ? Object.entries(map) : [];
    let h = '<div class="form-group">';
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

    const defaults = (configData.agents && configData.agents.defaults) || {};
    const agents = (configData.agents && configData.agents.list) || [];

    let html = panelHeader(t('agents.title'), 'panelAgents');
    html += `<div class="panel-desc">${t('agents.desc')}</div>`;

    // Defaults section
    html += `<div class="form-section-title">${t('agents.defaults')}</div>`;
    html += '<div class="channel-form" id="agentDefaultsForm">';
    html += renderFormField('workspace', t('field.workspace'), 'text', defaults.workspace, { placeholder: '~/.ok/workspace' });
    html += renderToggleField('restrict_to_workspace', t('agents.restrictWorkspace'), defaults.restrict_to_workspace !== false);
    html += renderToggleField('allow_read_outside_workspace', t('agents.allowReadOutside'), defaults.allow_read_outside_workspace);
    html += renderFormField('model_name', t('agents.modelName'), 'text', defaults.model_name || defaults.model, { placeholder: 'Default model name', hint: t('agents.modelNameHint') });
    html += renderFormField('max_tokens', t('agents.maxTokens'), 'number', defaults.max_tokens || 32768, { min: 1 });
    html += renderFormField('max_tool_iterations', t('agents.maxToolIter'), 'number', defaults.max_tool_iterations || 50, { min: 1 });
    html += renderFormField('summarize_message_threshold', t('agents.summarizeThreshold'), 'number', defaults.summarize_message_threshold || 20, { hint: t('agents.summarizeThresholdHint') });
    html += renderFormField('summarize_token_percent', t('agents.summarizeTokenPct'), 'number', defaults.summarize_token_percent || 75, { min: 1, max: 100 });
    html += renderFormField('max_media_size', t('agents.maxMediaSize'), 'number', defaults.max_media_size || 0, { hint: t('agents.maxMediaSizeHint') });
    html += `<div style="margin-top:16px;"><button class="btn btn-primary" onclick="saveAgentDefaults()">${t('agents.saveDefaults')}</button></div>`;
    html += '</div>';

    // Agent list
    html += `<div class="form-section-title" style="margin-top:32px;">${t('agents.agentList')}</div>`;
    html += `<div style="margin-bottom:14px;"><button class="btn btn-sm btn-primary" onclick="showAddAgentModal()">${t('agents.addAgent')}</button></div>`;
    if (agents.length === 0) {
        html += `<div style="color:var(--text-muted);font-size:13px;margin-bottom:16px;">${t('agents.noAgents')}</div>`;
    } else {
        html += '<div class="model-grid">';
        agents.forEach((a, i) => {
            html += `<div class="model-card">`;
            html += `<div class="model-card-head"><div class="model-name">${esc(a.name || a.id)}</div>`;
            if (a.default) html += '<span class="badge-primary">Default</span>';
            html += '</div>';
            html += `<div class="model-detail"><strong>ID:</strong> ${esc(a.id)}</div>`;
            if (a.workspace) html += `<div class="model-detail"><strong>Workspace:</strong> ${esc(a.workspace)}</div>`;
            if (a.model) {
                const modelStr = typeof a.model === 'string' ? a.model : (a.model.primary || '');
                html += `<div class="model-detail"><strong>Model:</strong> ${esc(modelStr)}</div>`;
            }
            if (a.skills && a.skills.length) html += `<div class="model-detail"><strong>Skills:</strong> ${a.skills.map(s => esc(s)).join(', ')}</div>`;
            html += '<div class="model-actions">';
            html += `<button class="btn btn-sm" onclick="showEditAgentModal(${i})">${t('edit')}</button>`;
            html += `<button class="btn btn-sm btn-danger" onclick="deleteAgent(${i})">${t('delete')}</button>`;
            html += '</div></div>';
        });
        html += '</div>';
    }

    panel.innerHTML = html;
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
    if (fields.model_name) d.model_name = fields.model_name;
    if (fields.max_tokens) d.max_tokens = parseInt(fields.max_tokens) || 32768;
    if (fields.max_tool_iterations) d.max_tool_iterations = parseInt(fields.max_tool_iterations) || 50;
    if (fields.summarize_message_threshold) d.summarize_message_threshold = parseInt(fields.summarize_message_threshold) || 20;
    if (fields.summarize_token_percent) d.summarize_token_percent = parseInt(fields.summarize_token_percent) || 75;
    d.max_media_size = parseInt(fields.max_media_size) || 0;

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

function renderAgentModalBody(data) {
    const modelStr = data.model ? (typeof data.model === 'string' ? data.model : (data.model.primary || '')) : '';
    let html = '';
    html += renderFormField('id', 'ID', 'text', data.id, { required: true, placeholder: 'e.g. research-agent' });
    if (editingAgentIndex >= 0) {
        // Make ID read-only on edit by replacing the input after render
    }
    html += renderFormField('name', 'Name', 'text', data.name, { placeholder: t('agents.friendlyName') });
    html += renderToggleField('default', t('agents.defaultAgent'), !!data.default);
    html += renderFormField('workspace', t('field.workspace'), 'text', data.workspace, { placeholder: t('agents.inheritsDefaults') });
    html += renderFormField('model', t('agents.modelName'), 'text', modelStr, { placeholder: 'e.g. gpt-4.1-mini', hint: t('agents.modelHint') });
    html += renderArrayField('skills', 'Skills', data.skills, { placeholder: 'Skill name' });
    document.getElementById('agentModalBody').innerHTML = html;
    if (editingAgentIndex >= 0) {
        const idInput = document.querySelector('#agentModalBody input[data-field="id"]');
        if (idInput) { idInput.readOnly = true; idInput.style.opacity = '0.6'; }
    }
}

function saveAgentFromModal() {
    const body = document.getElementById('agentModalBody');
    const fields = collectFormFields(body);

    if (!fields.id) {
        showStatus(t('agents.idRequired'), 'error');
        return;
    }

    if (!configData.agents) configData.agents = {};
    if (!configData.agents.list) configData.agents.list = [];

    const agent = editingAgentIndex >= 0 ? { ...configData.agents.list[editingAgentIndex] } : {};
    agent.id = fields.id;
    if (fields.name) agent.name = fields.name; else delete agent.name;
    if (fields.default) agent.default = true; else delete agent.default;
    if (fields.workspace) agent.workspace = fields.workspace; else delete agent.workspace;
    if (fields.model) agent.model = fields.model; else delete agent.model;
    if (fields.skills && fields.skills.length) agent.skills = fields.skills; else delete agent.skills;

    if (editingAgentIndex >= 0) {
        configData.agents.list[editingAgentIndex] = agent;
    } else {
        // Check for duplicate ID
        if (configData.agents.list.some(a => a.id === agent.id)) {
            showStatus(t('agents.duplicateId', { id: agent.id }), 'error');
            return;
        }
        configData.agents.list.push(agent);
    }

    closeAgentModal();
    saveConfig().then(renderAgents);
}

// ── Bindings Panel ──────────────────────────────────

function renderBindings() {
    const panel = document.getElementById('panelBindings');
    if (!configData) return;

    const bindings = configData.bindings || [];

    let html = panelHeader(t('bindings.title'), 'panelBindings');
    html += `<div class="panel-desc">${t('bindings.desc')}</div>`;
    html += `<div style="margin-bottom:14px;"><button class="btn btn-sm btn-primary" onclick="showAddBindingModal()">${t('bindings.addBinding')}</button></div>`;

    if (bindings.length === 0) {
        html += `<div style="color:var(--text-muted);font-size:13px;">${t('bindings.noBindings')}</div>`;
    } else {
        html += '<div class="model-grid">';
        bindings.forEach((b, i) => {
            html += `<div class="model-card">`;
            html += `<div class="model-card-head"><div class="model-name">${esc(b.agent_id)}</div></div>`;
            html += `<div class="model-detail"><strong>${t('bindings.channel')}:</strong> ${esc(b.match.channel)}</div>`;
            if (b.match.account_id) html += `<div class="model-detail"><strong>${t('bindings.accountId')}:</strong> ${esc(b.match.account_id)}</div>`;
            if (b.match.peer) html += `<div class="model-detail"><strong>Peer:</strong> ${esc(b.match.peer.kind)}:${esc(b.match.peer.id)}</div>`;
            if (b.match.guild_id) html += `<div class="model-detail"><strong>Guild:</strong> ${esc(b.match.guild_id)}</div>`;
            if (b.match.team_id) html += `<div class="model-detail"><strong>Team:</strong> ${esc(b.match.team_id)}</div>`;
            html += '<div class="model-actions">';
            html += `<button class="btn btn-sm" onclick="showEditBindingModal(${i})">${t('edit')}</button>`;
            html += `<button class="btn btn-sm btn-danger" onclick="deleteBinding(${i})">${t('delete')}</button>`;
            html += '</div></div>';
        });
        html += '</div>';
    }

    panel.innerHTML = html;
}

function deleteBinding(idx) {
    if (!confirm(t('bindings.deleteConfirm'))) return;
    configData.bindings.splice(idx, 1);
    saveConfig().then(renderBindings);
}

// ── Binding Modal ───────────────────────────────────
let editingBindingIndex = -1;
const bindingChannelOptions = ['telegram', 'discord', 'slack', 'whatsapp'];
const bindingPeerKindOptions = [{ value: '', label: '(none)' }, { value: 'direct', label: 'direct' }, { value: 'group', label: 'group' }];

function getAgentIdOptions() {
    const opts = [];
    const agents = (configData.agents && configData.agents.list) || [];
    agents.forEach(a => opts.push(a.id));
    if (opts.length === 0) opts.push('default');
    return opts;
}

function showAddBindingModal() {
    editingBindingIndex = -1;
    document.getElementById('bindingModalTitle').textContent = t('bindings.addBindingTitle');
    renderBindingModalBody({});
    document.getElementById('bindingModal').classList.add('active');
}

function showEditBindingModal(idx) {
    editingBindingIndex = idx;
    const b = configData.bindings[idx];
    document.getElementById('bindingModalTitle').textContent = t('bindings.editBinding');
    renderBindingModalBody(b);
    document.getElementById('bindingModal').classList.add('active');
}

function closeBindingModal() {
    document.getElementById('bindingModal').classList.remove('active');
}

function renderBindingModalBody(data) {
    const match = data.match || {};
    const agentIds = getAgentIdOptions();
    let html = '';
    html += renderSelectField('agent_id', t('bindings.agentId'), agentIds, data.agent_id || agentIds[0]);
    html += renderSelectField('channel', t('bindings.channel'), bindingChannelOptions, match.channel || 'telegram');
    html += renderFormField('account_id', t('bindings.accountId'), 'text', match.account_id, { placeholder: 'Optional' });
    html += renderSelectField('peer_kind', t('bindings.peerKind'), bindingPeerKindOptions, (match.peer && match.peer.kind) || '');
    html += renderFormField('peer_id', t('bindings.peerId'), 'text', (match.peer && match.peer.id) || '', { placeholder: 'Optional — user or group ID' });
    html += renderFormField('guild_id', t('bindings.guildId'), 'text', match.guild_id, { placeholder: 'Optional — Discord server ID' });
    html += renderFormField('team_id', t('bindings.teamId'), 'text', match.team_id, { placeholder: 'Optional — Slack workspace ID' });
    document.getElementById('bindingModalBody').innerHTML = html;
}

function saveBindingFromModal() {
    const body = document.getElementById('bindingModalBody');
    const fields = collectFormFields(body);

    if (!fields.agent_id || !fields.channel) {
        showStatus(t('bindings.required'), 'error');
        return;
    }

    if (!configData.bindings) configData.bindings = [];

    const binding = { agent_id: fields.agent_id, match: { channel: fields.channel } };
    if (fields.account_id) binding.match.account_id = fields.account_id;
    if (fields.peer_kind && fields.peer_id) {
        binding.match.peer = { kind: fields.peer_kind, id: fields.peer_id };
    }
    if (fields.guild_id) binding.match.guild_id = fields.guild_id;
    if (fields.team_id) binding.match.team_id = fields.team_id;

    if (editingBindingIndex >= 0) {
        configData.bindings[editingBindingIndex] = binding;
    } else {
        configData.bindings.push(binding);
    }

    closeBindingModal();
    saveConfig().then(renderBindings);
}

// ── Tool Settings Panel ─────────────────────────────

function renderToolSettings() {
    const panel = document.getElementById('panelToolSettings');
    if (!configData) return;

    const tools = configData.tools || {};

    let html = panelHeader(t('tools.title'), 'panelToolSettings');
    html += `<div class="panel-desc">${t('tools.desc')}</div>`;
    html += '<div class="channel-form" id="toolSettingsForm">';

    // Quick toggles
    html += `<div class="form-section-title">${t('tools.toggles')}</div>`;
    const simpleTools = [
        ['append_file', 'Append File'], ['edit_file', 'Edit File'], ['find_skills', 'Find Skills'],
        ['i2c', 'I2C (Hardware)'], ['install_skill', 'Install Skill'], ['list_dir', 'List Directory'],
        ['message', 'Message'], ['read_file', 'Read File'], ['send_file', 'Send File'],
        ['spawn', 'Spawn'], ['spi', 'SPI (Hardware)'], ['subagent', 'Subagent'],
        ['web_fetch', 'Web Fetch'], ['write_file', 'Write File'],
    ];
    html += '<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:4px;">';
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

function renderRAG() {
    const panel = document.getElementById('panelRAG');
    if (!configData) return;

    const rag = configData.rag || {};

    let html = panelHeader(t('rag.title'), 'panelRAG');
    html += `<div class="panel-desc">${t('rag.desc')}</div>`;
    html += '<div class="channel-form" id="ragForm">';
    html += renderToggleField('enabled', t('enabled'), rag.enabled);
    const primaryBase = (configData.model_list && configData.model_list.length > 0) ? configData.model_list[0].api_base : '';
    const baseHint = primaryBase ? 'Leave empty to use: ' + primaryBase : 'OpenAI-compatible /v1/embeddings endpoint';
    html += renderFormField('base_url', t('rag.embeddingsUrl'), 'text', rag.base_url, { placeholder: primaryBase || 'http://localhost:11434/v1', hint: baseHint });
    html += renderFormField('api_key', t('rag.apiKey'), 'password', rag.api_key, { placeholder: 'API key for embeddings service' });
    html += renderFormField('model', t('rag.embeddingModel'), 'text', rag.model || 'text-embedding-3-small', { placeholder: 'text-embedding-3-small' });
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
    configData.rag = {
        enabled: !!f.enabled,
        base_url: f.base_url || '',
        api_key: f.api_key || '',
        model: f.model || 'text-embedding-3-small',
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
    html += '<div class="channel-form" id="gatewayForm">';
    html += renderFormField('host', t('gateway.host'), 'text', gw.host || '127.0.0.1', { placeholder: '127.0.0.1' });
    html += renderFormField('port', t('gateway.port'), 'number', gw.port || 18790, { min: 1, max: 65535 });
    html += `<div style="margin-top:20px;"><button class="btn btn-primary" onclick="saveGateway()">${t('save')}</button></div>`;
    html += '</div>';
    panel.innerHTML = html;
}

function saveGateway() {
    if (!configData) return;
    const f = collectFormFields(document.getElementById('gatewayForm'));
    configData.gateway = { host: f.host || '127.0.0.1', port: parseInt(f.port) || 18790 };
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
    html += '<div class="channel-form" id="heartbeatForm">';
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
    html += '<div class="channel-form" id="devicesForm">';
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
            html += `<button class="btn btn-sm" onclick="showSkillDetail('${esc(s.name)}')">${t('skills.show')}</button>`;
            if (s.source !== 'builtin') {
                html += `<button class="btn btn-sm btn-danger" onclick="removeSkill('${esc(s.name)}')">${t('skills.remove')}</button>`;
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
        showStatus('Failed to load skill: ' + e.message, 'error');
    }
}

function closeSkillDetailModal() {
    document.getElementById('skillDetailModal').classList.remove('active');
}

async function removeSkill(name) {
    if (!confirm('Remove skill "' + name + '"? This cannot be undone.')) return;
    try {
        const resp = await fetch('/api/skills/remove', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name }),
        });
        if (!resp.ok) throw new Error(await resp.text());
        showStatus('Skill "' + name + '" removed', 'success');
        fetchInstalledSkills();
    } catch (e) {
        showStatus('Failed to remove skill: ' + e.message, 'error');
    }
}

async function searchSkillsUI() {
    const input = document.getElementById('skillSearchInput');
    const container = document.getElementById('skillsSearchResults');
    if (!input || !container) return;

    const query = input.value.trim();
    if (!query) { container.innerHTML = ''; return; }

    container.innerHTML = '<div class="empty-state"><div class="empty-state-desc">Searching...</div></div>';

    try {
        const resp = await fetch('/api/skills/search?q=' + encodeURIComponent(query) + '&limit=10');
        if (!resp.ok) throw new Error(await resp.text());
        const results = await resp.json();

        if (!results || results.length === 0) {
            container.innerHTML = '<div class="empty-state"><div class="empty-state-desc">No results found.</div></div>';
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
                html += '<button class="btn btn-sm" disabled>Installed</button>';
            } else {
                html += `<button class="btn btn-sm btn-primary" onclick="installSkill('${esc(r.slug)}', '${esc(r.registry_name)}', this)">Install</button>`;
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
    if (btn) { btn.disabled = true; btn.textContent = 'Installing...'; }
    try {
        const resp = await fetch('/api/skills/install', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ slug, registry }),
        });
        if (!resp.ok) throw new Error(await resp.text());
        const data = await resp.json();

        if (data.is_malware_blocked) {
            showStatus('Skill blocked: flagged as malware', 'error');
            return;
        }
        if (data.is_suspicious) {
            showStatus('Skill installed but flagged as suspicious', 'error');
        } else {
            showStatus('Skill "' + slug + '" installed (v' + (data.version || '?') + ')', 'success');
        }
        fetchInstalledSkills();
        // Re-run search to update install buttons
        searchSkillsUI();
    } catch (e) {
        showStatus('Install failed: ' + e.message, 'error');
        if (btn) { btn.disabled = false; btn.textContent = 'Install'; }
    }
}

// ── Init ────────────────────────────────────────────
applyI18n();
loadConfig();
loadAuthStatus().then(() => renderModels());

// Hash routing handles #auth and all other panel navigation automatically.
