// ============================================================================
// workspace.js — Unified Agent + Chat controller
// Two modes:
//   Agent mode  (activeConversationID === null) — WebSocket to /ws/agent
//   Chat mode   (activeConversationID !== null) — SSE via /api/conversations/:id/messages
// ============================================================================

// === State ===
var activeConversationID = null;
var isRunning = false;
var isStreaming = false;
var ws = null;
var lastGoal = '';
var currentStreams = {};
var thoughtEl = null;
var currentConfirmId = null;
var confirmTimerInterval = null;
var executions = [];
var searchTimeout = null;

// === 1. DOM References ===
var agentInput = document.getElementById('agentInput');
var agentSend = document.getElementById('agentSend');
var agentForm = document.getElementById('agentForm');
var chatPanel = document.getElementById('chatPanel');
var chatMessages = document.getElementById('chatMessages');
var chatEmpty = document.getElementById('chatEmpty');
var chatTitle = document.getElementById('chatTitle');
var chatInputArea = document.getElementById('chatInputArea');
var stopBtn = document.getElementById('stopBtn');
var agentState = document.getElementById('agentState');
var sidebar = document.getElementById('sidebar');
var sidebarOverlay = document.getElementById('sidebarOverlay');
var sidebarClose = document.getElementById('sidebarClose');
var menuToggle = document.getElementById('menuToggle');
var searchInput = document.getElementById('searchInput');
var conversationList = document.getElementById('conversationList');
var newChatBtn = document.getElementById('newChatButton');
var logoutBtn = document.getElementById('logoutButton');
var healthReport = document.getElementById('healthReport');
var healthItems = document.getElementById('healthItems');
var healthDismiss = document.getElementById('healthDismiss');
var replayList = document.getElementById('replayList');
var jobsList = document.getElementById('jobsList');
var channelsList = document.getElementById('channelsList');
var confirmModal = document.getElementById('confirmModal');
var confirmTool = document.getElementById('confirmTool');
var confirmSummary = document.getElementById('confirmSummary');
var confirmTimer = document.getElementById('confirmTimer');
var confirmApprove = document.getElementById('confirmApprove');
var confirmReject = document.getElementById('confirmReject');
var resizer = document.getElementById('resizer');

// === Helper ===
function esc(text) {
  var d = document.createElement('div');
  d.textContent = text;
  return d.innerHTML;
}

// === 2. Sidebar Management ===
if (menuToggle) {
  menuToggle.addEventListener('click', function () {
    sidebar.classList.add('open');
    sidebarOverlay.classList.add('visible');
  });
}

if (sidebarClose) sidebarClose.addEventListener('click', closeSidebar);
if (sidebarOverlay) sidebarOverlay.addEventListener('click', closeSidebar);

function closeSidebar() {
  sidebar.classList.remove('open');
  sidebarOverlay.classList.remove('visible');
}

// New conversation
if (newChatBtn) {
  newChatBtn.addEventListener('click', function () {
    fetch('/api/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: '' })
    })
      .then(function (response) {
        if (!response.ok) throw new Error('failed');
        return response.json();
      })
      .then(function (conversation) {
        selectConversation(conversation.id, conversation.title || 'New conversation');
        loadConversations();
        closeSidebar();
      });
  });
}

// Search with debounce
if (searchInput) {
  searchInput.addEventListener('input', function () {
    var query = this.value.trim();
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(function () {
      if (query === '') {
        loadConversations();
      } else {
        searchConversations(query);
      }
    }, 300);
  });
}

// Logout
if (logoutBtn) {
  logoutBtn.addEventListener('click', function () {
    fetch('/api/auth/logout', { method: 'POST' })
      .then(function () {
        window.location.href = '/login';
      });
  });
}

function loadConversations() {
  fetch('/api/conversations')
    .then(function (response) {
      if (!response.ok) throw new Error('failed');
      return response.json();
    })
    .then(renderConversationList)
    .catch(function () {
      renderConversationList([]);
    });
}

function searchConversations(query) {
  fetch('/api/conversations/search?q=' + encodeURIComponent(query))
    .then(function (response) {
      if (!response.ok) throw new Error('failed');
      return response.json();
    })
    .then(renderConversationList)
    .catch(function () {
      renderConversationList([]);
    });
}

function renderConversationList(conversations) {
  conversationList.innerHTML = '';

  if (!conversations || conversations.length === 0) {
    var empty = document.createElement('div');
    empty.className = 'conversation-empty';
    empty.textContent = 'No conversations';
    conversationList.appendChild(empty);
    return;
  }

  conversations.forEach(function (conv) {
    var item = document.createElement('button');
    item.className = 'conversation-item';
    item.dataset.id = conv.id;
    if (conv.id === activeConversationID) {
      item.classList.add('active');
    }

    var channel = conv.channel || 'web';
    if (channel !== 'web') {
      var badge = document.createElement('span');
      badge.className = 'channel-badge channel-' + channel;
      var channelLabels = { whatsapp: 'WA', telegram: 'TG', discord: 'DC' };
      badge.textContent = channelLabels[channel] || channel.substring(0, 2).toUpperCase();
      item.appendChild(badge);
    }

    var titleSpan = document.createElement('span');
    titleSpan.className = 'conversation-title';
    titleSpan.textContent = conv.title || 'Untitled';
    item.appendChild(titleSpan);

    var deleteBtn = document.createElement('button');
    deleteBtn.className = 'conversation-delete';
    deleteBtn.innerHTML = '&times;';
    deleteBtn.setAttribute('aria-label', 'Delete conversation');
    deleteBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      fetch('/api/conversations/' + conv.id, { method: 'DELETE' })
        .then(function (response) {
          if (!response.ok) throw new Error('failed');
          if (conv.id === activeConversationID) {
            deselectConversation();
          }
          loadConversations();
        });
    });
    item.appendChild(deleteBtn);

    item.addEventListener('click', function () {
      selectConversation(conv.id, conv.title);
    });
    conversationList.appendChild(item);
  });
}

// === 3. Conversation Selection ===
function selectConversation(id, title, skipPush) {
  activeConversationID = id;
  chatTitle.textContent = title || 'Untitled';

  // Switch to chat mode: show chatMessages, hide agentFeed
  chatMessages.style.display = '';
  chatPanel.style.display = 'none';
  chatInputArea.style.display = 'block';

  // Update URL
  if (!skipPush) {
    history.pushState({ id: id, title: title }, '', '/?c=' + id);
  }

  // Update active state in sidebar
  var items = document.querySelectorAll('.conversation-item');
  items.forEach(function (item) {
    item.classList.toggle('active', item.dataset.id == id);
  });

  closeSidebar();

  // Load historical messages
  fetch('/api/conversations/' + id + '/messages')
    .then(function (response) {
      if (!response.ok) throw new Error('failed');
      return response.json();
    })
    .then(renderMessages)
    .catch(function () {
      renderMessages([]);
    });
}

function deselectConversation() {
  activeConversationID = null;
  chatTitle.textContent = 'OK';

  // Switch to agent mode: hide chatMessages, show agentFeed
  chatMessages.style.display = 'none';
  chatPanel.style.display = '';
  chatInputArea.style.display = 'block';

  // Show greeting if no agent content
  if (!chatPanel.hasChildNodes() || chatPanel.children.length === 0) {
    showAgentGreeting();
  }

  // Clear chat messages
  clearMessageList();
  chatEmpty.style.display = 'flex';
  if (chatEmpty.querySelector('.chat-empty-text')) {
    chatEmpty.querySelector('.chat-empty-text').textContent = 'Send a message to get started';
  }

  history.pushState(null, '', '/');

  var items = document.querySelectorAll('.conversation-item');
  items.forEach(function (item) { item.classList.remove('active'); });
}

function showAgentGreeting() {
  var existing = chatPanel.querySelector('.chat-greeting');
  if (existing) {
    existing.style.display = '';
    return;
  }
  var greet = document.createElement('div');
  greet.className = 'chat-greeting';
  greet.innerHTML = '<div class="chat-empty-logo">OK</div><p class="chat-empty-text">Send a command to get started</p>';
  chatPanel.appendChild(greet);
}

// === 4. WebSocket Connection (Agent Mode) ===
function connectWS() {
  var protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(protocol + '//' + location.host + '/ws/agent');
  ws.onmessage = function (e) {
    try {
      var data = JSON.parse(e.data);
      if (data.type === 'hydration') applyHydration(data);
      else handleEvent(data);
    } catch (ex) {}
  };
  ws.onclose = function () {
    setTimeout(connectWS, 3000);
  };
}

function applyHydration(state) {
  if (state.running) {
    setRunning(true);
    if (state.phase) addPhaseDrawer(state.phase);
  }
}

// === 5. Input Handling ===
agentInput.addEventListener('input', function () {
  this.style.height = 'auto';
  this.style.height = Math.min(this.scrollHeight, 120) + 'px';
  agentSend.disabled = this.value.trim() === '' || isRunning || isStreaming;
});

agentInput.addEventListener('keydown', function (e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    if (!agentSend.disabled && !isRunning && !isStreaming) {
      agentForm.dispatchEvent(new Event('submit'));
    }
  }
});

agentForm.addEventListener('submit', function (e) {
  e.preventDefault();
  if (isRunning || isStreaming) return;
  var input = agentInput.value.trim();
  if (!input) return;

  agentInput.value = '';
  agentInput.style.height = 'auto';
  agentSend.disabled = true;

  if (activeConversationID) {
    sendChatMessage(input);
  } else {
    sendAgentCommand(input);
  }
});

// === 6. Agent Command (WebSocket path) ===
function sendAgentCommand(input) {
  lastGoal = input;
  setRunning(true);

  // Ensure agent feed is visible
  chatMessages.style.display = 'none';
  chatPanel.style.display = '';

  // Hide greeting
  var greet = chatPanel.querySelector('.chat-greeting');
  if (greet) greet.style.display = 'none';

  addChatMsg('user', input);

  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'input', content: input }));
  } else {
    fallbackSSE(input);
  }
}

function fallbackSSE(input) {
  fetch('/api/agent/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input: input })
  }).then(function (response) {
    if (!response.ok) return response.json().then(function (d) { throw new Error(d.error || 'Error'); });
    var reader = response.body.getReader();
    var decoder = new TextDecoder();
    var buffer = '';
    function readChunk() {
      return reader.read().then(function (result) {
        if (result.done) { finish(input); return; }
        buffer += decoder.decode(result.value, { stream: true });
        var lines = buffer.split('\n');
        buffer = lines.pop();
        lines.forEach(function (line) {
          if (!line.startsWith('data: ')) return;
          try { handleEvent(JSON.parse(line.substring(6))); } catch (ex) {}
        });
        return readChunk();
      });
    }
    return readChunk();
  }).catch(function (error) {
    addChatMsg('error', error.message);
    finish(input);
  });
}

// === 7. Chat Message (SSE path) ===
function sendChatMessage(content) {
  isStreaming = true;

  // Ensure chat view is visible
  chatMessages.style.display = '';
  chatPanel.style.display = 'none';

  appendMessage('user', content);

  // Create streaming assistant placeholder
  var assistantBubble = appendMessage('assistant', '');
  assistantBubble.parentElement.classList.add('message-streaming');

  fetch('/api/conversations/' + activeConversationID + '/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content: content })
  })
    .then(function (response) {
      if (!response.ok && response.headers.get('content-type') && response.headers.get('content-type').indexOf('application/json') !== -1) {
        return response.json().then(function (data) {
          throw new Error(data.error || 'Error sending message.');
        });
      }

      var reader = response.body.getReader();
      var decoder = new TextDecoder();
      var buffer = '';

      function readChunk() {
        return reader.read().then(function (result) {
          if (result.done) {
            finishStreaming(assistantBubble);
            return;
          }

          buffer += decoder.decode(result.value, { stream: true });
          var lines = buffer.split('\n');
          buffer = lines.pop();

          lines.forEach(function (line) {
            if (!line.startsWith('data: ')) return;
            var jsonStr = line.substring(6);
            try {
              var data = JSON.parse(jsonStr);
              if (data.error) {
                assistantBubble.textContent = 'Error: ' + data.error;
                finishStreaming(assistantBubble);
                return;
              }
              if (data.done === true && !data.type) {
                finishStreaming(assistantBubble);
                return;
              }
              // Structured events
              if (data.type === 'token') {
                assistantBubble.textContent += data.content;
                scrollToBottom();
              } else if (data.type === 'message') {
                assistantBubble.textContent = data.content;
                scrollToBottom();
              } else if (data.type === 'intent') {
                showModeBadge(assistantBubble, data.mode);
              } else if (data.type === 'step') {
                showStepIndicator(assistantBubble, data);
              } else if (data.type === 'done') {
                finishStreaming(assistantBubble);
                return;
              }
              // Legacy: token field for backward compat
              if (data.token && !data.type) {
                assistantBubble.textContent += data.token;
                scrollToBottom();
              }
            } catch (e) {
              // Skip unparseable lines
            }
          });

          return readChunk();
        });
      }

      return readChunk();
    })
    .catch(function (error) {
      assistantBubble.textContent = 'Error: ' + error.message;
      finishStreaming(assistantBubble);
    });
}

function finishStreaming(bubble) {
  isStreaming = false;
  agentSend.disabled = agentInput.value.trim() === '';
  if (bubble && bubble.parentElement) {
    bubble.parentElement.classList.remove('message-streaming');
    var indicators = bubble.parentElement.querySelectorAll('.step-indicator');
    indicators.forEach(function (el) {
      el.classList.add('step-done');
    });
  }
  loadConversations();
}

// === 8. Agent Event Handler ===
function handleEvent(data) {
  switch (data.type) {
    case 'phase':
      addPhaseDrawer(data.content);
      break;
    case 'step':
      addStepPill(data);
      break;
    case 'stream':
      if (data.tool === 'thought') appendThought(data.content);
      else appendStream(data.tool, data.content);
      break;
    case 'terminal':
      appendStream('Internal Logs', data.content);
      break;
    case 'message':
      addChatMsg('assistant', data.content);
      break;
    case 'diff':
      appendDiffMessage(data.name, data.content);
      break;
    case 'confirm':
      showConfirmModal(data.name, data.tool, data.content);
      break;
    case 'done':
      finish(lastGoal);
      break;
  }
}

// === 9. UI Renderers (Agent Mode) ===
function addChatMsg(type, content) {
  var msg = document.createElement('div');
  msg.className = 'chat-msg ' + type + '-bubble';
  if (type === 'assistant') {
    var botName = document.createElement('div');
    botName.className = 'msg-bot-tag';
    botName.textContent = 'AI Agent';
    msg.appendChild(botName);
  }
  var txt = document.createElement('div');
  txt.textContent = content;
  msg.appendChild(txt);
  chatPanel.appendChild(msg);
  chatPanel.scrollTop = chatPanel.scrollHeight;
  return msg;
}

function appendThought(token) {
  if (!thoughtEl) {
    thoughtEl = document.createElement('div');
    thoughtEl.className = 'chat-msg thought-bubble';
    chatPanel.appendChild(thoughtEl);
  }
  thoughtEl.textContent += token;
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function appendStream(tool, text) {
  if (!currentStreams[tool]) {
    var wrapper = document.createElement('div');
    wrapper.className = 'chat-msg terminal-bubble';
    wrapper.innerHTML = '<div class="terminal-header"><span class="terminal-dots"><span></span><span></span><span></span></span> bash (' + esc(tool) + ')</div><pre class="terminal-body scrollbar-hidden"></pre>';
    chatPanel.appendChild(wrapper);
    currentStreams[tool] = wrapper.querySelector('.terminal-body');
  }
  // Remove ANSI escape codes for pure HTML rendering
  var cleanText = text.replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');
  currentStreams[tool].textContent += cleanText;
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function appendDiffMessage(file, content) {
  var wrapper = document.createElement('div');
  wrapper.className = 'chat-msg diff-bubble';
  wrapper.innerHTML = '<div class="diff-header"><span>Git Diff</span> &bull; <span class="diff-file">' + esc(file) + '</span></div><div class="diff-body"></div>';
  var diffBody = wrapper.querySelector('.diff-body');

  var parts = content.split('\n---SEPARATOR---\n');
  if (parts.length < 2) return;
  var before = parts[0].split('\n');
  var after = parts[1].split('\n');

  var maxLen = Math.max(before.length, after.length);
  for (var i = 0; i < maxLen; i++) {
    var bLine = before[i] || '';
    var aLine = after[i] || '';
    if (bLine !== aLine) {
      if (bLine) {
        var del = document.createElement('div');
        del.className = 'diff-line del';
        del.textContent = '- ' + bLine;
        diffBody.appendChild(del);
      }
      if (aLine) {
        var add = document.createElement('div');
        add.className = 'diff-line add';
        add.textContent = '+ ' + aLine;
        diffBody.appendChild(add);
      }
    } else {
      var same = document.createElement('div');
      same.className = 'diff-line';
      same.textContent = '  ' + aLine;
      diffBody.appendChild(same);
    }
  }
  chatPanel.appendChild(wrapper);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function addPhaseDrawer(phase) {
  thoughtEl = null;
  currentStreams = {};
  var pill = document.createElement('div');
  pill.className = 'phase-divider';
  pill.innerHTML = '<span>Execution Phase: ' + esc(phase).toUpperCase() + '</span>';
  chatPanel.appendChild(pill);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function addStepPill(data) {
  var wrapper = document.createElement('div');
  wrapper.className = 'step-pill-container';
  var pill = document.createElement('span');
  pill.className = 'step-pill ' + (data.status || '');
  pill.innerHTML = '<span class="step-dot"></span> ' + esc(data.tool || data.name || '') + ' <span class="step-status">[' + esc(data.status || '') + ']</span>';
  if (data.elapsed_ms) pill.innerHTML += ' <i>(' + data.elapsed_ms + 'ms)</i>';
  wrapper.appendChild(pill);
  chatPanel.appendChild(wrapper);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

// === 10. UI Renderers (Chat Mode) ===
function appendMessage(role, content) {
  chatEmpty.style.display = 'none';

  var list = chatMessages.querySelector('.message-list');
  if (!list) {
    list = document.createElement('div');
    list.className = 'message-list';
    chatMessages.appendChild(list);
  }

  var wrapper = document.createElement('div');
  wrapper.className = 'message message-' + role;

  var roleLabel = document.createElement('div');
  roleLabel.className = 'message-role';
  roleLabel.textContent = role === 'user' ? 'You' : 'Assistant';

  var bubble = document.createElement('div');
  bubble.className = 'message-bubble';
  bubble.textContent = content;

  wrapper.appendChild(roleLabel);
  wrapper.appendChild(bubble);
  list.appendChild(wrapper);

  scrollToBottom();
  return bubble;
}

function renderMessages(messages) {
  clearMessageList();

  if (!messages || messages.length === 0) {
    chatEmpty.style.display = 'flex';
    if (chatEmpty.querySelector('.chat-empty-text')) {
      chatEmpty.querySelector('.chat-empty-text').textContent = 'Start a conversation';
    }
    return;
  }

  chatEmpty.style.display = 'none';

  var list = document.createElement('div');
  list.className = 'message-list';

  messages.forEach(function (msg) {
    var wrapper = document.createElement('div');
    wrapper.className = 'message message-' + msg.role;

    var role = document.createElement('div');
    role.className = 'message-role';
    role.textContent = msg.role === 'user' ? 'You' : 'Assistant';

    var bubble = document.createElement('div');
    bubble.className = 'message-bubble';
    bubble.textContent = msg.content;

    wrapper.appendChild(role);
    wrapper.appendChild(bubble);
    list.appendChild(wrapper);
  });

  chatMessages.appendChild(list);
  scrollToBottom();
}

function clearMessageList() {
  var existing = chatMessages.querySelector('.message-list');
  if (existing) existing.remove();
}

function clearFeed() {
  chatPanel.innerHTML = '';
  thoughtEl = null;
  currentStreams = {};
}

function showModeBadge(bubble, mode) {
  if (!bubble || !bubble.parentElement) return;
  var roleEl = bubble.parentElement.querySelector('.message-role');
  if (!roleEl) return;
  var existing = roleEl.querySelector('.mode-badge');
  if (existing) existing.remove();
  if (mode && mode !== 'direct') {
    var badge = document.createElement('span');
    badge.className = 'mode-badge mode-' + mode;
    var labels = { task: 'Task', agent: 'Agent' };
    badge.textContent = labels[mode] || mode;
    roleEl.appendChild(badge);
  }
}

function showStepIndicator(bubble, data) {
  if (!bubble || !bubble.parentElement) return;
  var indicator = bubble.parentElement.querySelector('.step-indicator[data-tool="' + data.tool + '"]');
  if (!indicator) {
    indicator = document.createElement('div');
    indicator.className = 'step-indicator';
    indicator.dataset.tool = data.tool;
    bubble.parentElement.insertBefore(indicator, bubble);
  }
  indicator.className = 'step-indicator step-' + (data.status || 'running');
  var label = data.name || data.tool;
  if (data.elapsed_ms) {
    label += ' (' + data.elapsed_ms + 'ms)';
  }
  indicator.textContent = label;
  scrollToBottom();
}

function scrollToBottom() {
  chatMessages.scrollTop = chatMessages.scrollHeight;
}

// === 11. Confirmation Modal (HIL) ===
function showConfirmModal(id, tool, summary) {
  currentConfirmId = id;
  confirmTool.textContent = 'Tool invoked: ' + tool;
  confirmSummary.textContent = summary;
  confirmModal.style.display = 'flex';
  agentInput.disabled = true;
  var remaining = 30;
  confirmTimer.textContent = remaining + 's';
  confirmTimerInterval = setInterval(function () {
    remaining--;
    confirmTimer.textContent = remaining + 's';
    if (remaining <= 0) respondConfirm(false);
  }, 1000);
}

function hideConfirmModal() {
  confirmModal.style.display = 'none';
  agentInput.disabled = false;
  currentConfirmId = null;
  if (confirmTimerInterval) {
    clearInterval(confirmTimerInterval);
    confirmTimerInterval = null;
  }
}

function respondConfirm(approved) {
  if (!currentConfirmId) return;
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'confirm', id: currentConfirmId, approved: approved }));
  }
  hideConfirmModal();
}

confirmApprove.addEventListener('click', function () { respondConfirm(true); });
confirmReject.addEventListener('click', function () { respondConfirm(false); });

// === 12. Status & Telemetry ===
stopBtn.addEventListener('click', function () {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'cancel' }));
  fetch('/api/agent/cancel', { method: 'POST' }).catch(function () {});
});

function setRunning(running) {
  isRunning = running;
  agentSend.disabled = agentInput.value.trim() === '' || running || isStreaming;
  var dot = document.querySelector('.status-dot');
  var text = document.querySelector('.status-text');
  if (dot) dot.className = 'status-dot ' + (running ? 'running' : 'idle');
  if (text) text.textContent = running ? 'Processing' : 'Online';
  stopBtn.style.display = running ? 'inline-flex' : 'none';
}

function finish(goal) {
  thoughtEl = null;
  currentStreams = {};
  setRunning(false);
  executions.unshift({ goal: goal, status: 'done', time: new Date().toLocaleTimeString() });
  renderReplayList();
  loadJobs();
}

function renderReplayList() {
  replayList.innerHTML = '';
  if (!executions.length) {
    replayList.innerHTML = '<div class="section-empty">No recent commands</div>';
    return;
  }
  executions.slice(0, 10).forEach(function (exec) {
    var item = document.createElement('div');
    item.className = 'replay-item';
    item.innerHTML = '<span class="replay-goal">' + esc(exec.goal || '?') + '</span><span class="replay-status ' + exec.status + '">' + exec.status + '</span>';
    replayList.appendChild(item);
  });
}

function loadJobs() {
  fetch('/api/scheduler/jobs').then(function (r) { return r.json(); }).then(function (jobs) {
    jobsList.innerHTML = '';
    if (!jobs || !jobs.length) {
      jobsList.innerHTML = '<div class="section-empty">No scheduled tasks</div>';
      return;
    }
    jobs.forEach(function (job) {
      var item = document.createElement('div');
      item.className = 'job-item';
      item.innerHTML = '<span class="job-name">' + esc(job.name) + '</span><span class="job-interval">' + job.interval_seconds + 's</span>';
      jobsList.appendChild(item);
    });
  }).catch(function () {});
}

function loadStatus() {
  fetch('/api/agent/status').then(function (r) { return r.json(); }).then(function (data) {
    channelsList.innerHTML = '';
    [
      { name: 'Web', on: true },
      { name: 'WhatsApp', on: data.whatsapp_enabled },
      { name: 'Telegram', on: data.telegram_enabled },
      { name: 'Discord', on: data.discord_enabled }
    ].forEach(function (ch) {
      var item = document.createElement('div');
      item.className = 'channel-item';
      item.innerHTML = '<span class="channel-dot ' + (ch.on ? 'on' : 'off') + '"></span><span>' + ch.name + '</span>';
      channelsList.appendChild(item);
    });
  }).catch(function () {});
}

// === 13. Health Check ===
function checkServices() {
  fetch('/api/health/services')
    .then(function (response) {
      if (!response.ok) return null;
      return response.json();
    })
    .then(function (data) {
      if (!data) return;

      healthItems.innerHTML = '';
      var hasError = false;

      ['llm', 'embedding'].forEach(function (key) {
        var service = data[key];
        if (!service) return;

        var label = key === 'llm' ? 'LLM' : 'Embedding';
        var item = document.createElement('div');
        item.className = 'health-item';

        var dot = document.createElement('span');
        dot.className = 'health-dot ' + service.status;

        var model = document.createElement('span');
        model.className = 'health-model';
        model.textContent = label + (service.model ? ' — ' + service.model : '');

        item.appendChild(dot);
        item.appendChild(model);

        if (service.status === 'ok' && service.latency_ms) {
          var latency = document.createElement('span');
          latency.className = 'health-latency';
          latency.textContent = service.latency_ms + 'ms';
          item.appendChild(latency);
        }

        if (service.status === 'error' && service.error) {
          hasError = true;
          var errText = document.createElement('div');
          errText.className = 'health-error-text';
          errText.textContent = service.error.substring(0, 80);
          item.appendChild(errText);
        }

        if (service.status === 'disabled') {
          var disabledText = document.createElement('span');
          disabledText.className = 'health-latency';
          disabledText.textContent = 'not configured';
          item.appendChild(disabledText);
        }

        healthItems.appendChild(item);
      });

      healthReport.style.display = 'block';
      sessionStorage.setItem('healthChecked', '1');

      if (!hasError) {
        setTimeout(function () { dismissReport(); }, 3000);
      }
    })
    .catch(function () {});
}

function dismissReport() {
  healthReport.classList.add('fade-out');
  setTimeout(function () {
    healthReport.style.display = 'none';
    healthReport.classList.remove('fade-out');
  }, 300);
}

healthDismiss.addEventListener('click', dismissReport);

// === 14. Resizer ===
function setupResizer(resizerId, leftId, rightId) {
  var resizerEl = document.getElementById(resizerId);
  var left = document.getElementById(leftId);
  var right = document.getElementById(rightId);
  if (!resizerEl || !left || !right) return;

  var startX, startLeftW, startRightW;

  resizerEl.addEventListener('mousedown', function (e) {
    e.preventDefault();
    startX = e.clientX;
    startLeftW = left.offsetWidth;
    startRightW = right.offsetWidth;
    resizerEl.classList.add('active');
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  });

  function onMove(e) {
    var dx = e.clientX - startX;
    var newLeft = startLeftW + dx;
    var newRight = startRightW - dx;
    if (newLeft > 300 && newRight > 150) {
      left.style.flexBasis = newLeft + 'px';
      right.style.flexBasis = newRight + 'px';
    }
  }

  function onUp() {
    resizerEl.classList.remove('active');
    document.removeEventListener('mousemove', onMove);
    document.removeEventListener('mouseup', onUp);
  }
}

// === 15. Browser Navigation ===
window.addEventListener('popstate', function (event) {
  if (event.state && event.state.id) {
    selectConversation(event.state.id, event.state.title, true);
  } else {
    deselectConversation();
  }
});

// === 16. Init ===
(function init() {
  // Connect WebSocket for agent mode
  connectWS();

  // Load sidebar data
  loadConversations();
  loadJobs();
  loadStatus();

  // Setup resizer if right panel exists
  setupResizer('resizer', 'chatPanel', 'panelRight');

  // Determine initial mode from URL
  var params = new URLSearchParams(window.location.search);
  var convId = params.get('c');
  if (convId) {
    var numId = parseInt(convId, 10);
    if (numId) {
      fetch('/api/conversations/' + numId + '/messages')
        .then(function (response) {
          if (!response.ok) throw new Error('not found');
          selectConversation(numId, '', true);
        })
        .catch(function () {
          history.replaceState(null, '', '/');
        });
    }
  } else {
    // Agent mode: show agent feed, hide chat messages
    chatMessages.style.display = 'none';
    chatPanel.style.display = '';
    showAgentGreeting();
  }

  // Health check on first visit
  if (!sessionStorage.getItem('healthChecked')) {
    checkServices();
  }
})();
