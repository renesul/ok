// === State ===
var isRunning = false;
var executions = [];
var currentConfirmId = null;
var confirmTimerInterval = null;
var ws = null;
var lastGoal = '';

// === DOM ===
var agentInput = document.getElementById('agentInput');
var agentSend = document.getElementById('agentSend');
var chatPanel = document.getElementById('chatPanel');
var stopBtn = document.getElementById('stopBtn');

// === Terminal (xterm.js) ===
var term = new Terminal({
  theme: { background: '#000', foreground: '#ccc', cursor: '#FFD700' },
  fontSize: 13, fontFamily: "'SF Mono', 'Fira Code', monospace",
  cursorBlink: false, disableStdin: true, convertEol: true
});
term.open(document.getElementById('terminalContainer'));
term.writeln('\x1b[90m[OK Terminal]\x1b[0m');

// === Tabs ===
document.querySelectorAll('.tab').forEach(function (tab) {
  tab.addEventListener('click', function () {
    document.querySelectorAll('.tab').forEach(function (t) { t.classList.remove('active'); });
    document.querySelectorAll('.tab-content').forEach(function (c) { c.classList.remove('active'); });
    tab.classList.add('active');
    document.getElementById('tab' + capitalize(tab.dataset.tab)).classList.add('active');
  });
});
function capitalize(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

// === Resizers ===
setupResizer('resizer1', 'panelLeft', 'panelCenter');
setupResizer('resizer2', 'panelCenter', 'panelRight');

function setupResizer(resizerId, leftId, rightId) {
  var resizer = document.getElementById(resizerId);
  var left = document.getElementById(leftId);
  var right = document.getElementById(rightId);
  var startX, startLeftW, startRightW;

  resizer.addEventListener('mousedown', function (e) {
    e.preventDefault();
    startX = e.clientX;
    startLeftW = left.offsetWidth;
    startRightW = right.offsetWidth;
    resizer.classList.add('active');
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  });

  function onMove(e) {
    var dx = e.clientX - startX;
    var newLeft = startLeftW + dx;
    var newRight = startRightW - dx;
    if (newLeft > 150 && newRight > 150) {
      left.style.flexBasis = newLeft + 'px';
      right.style.flexBasis = newRight + 'px';
    }
  }
  function onUp() {
    resizer.classList.remove('active');
    document.removeEventListener('mousemove', onMove);
    document.removeEventListener('mouseup', onUp);
  }
}

// === WebSocket ===
function connectWS() {
  var protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(protocol + '//' + location.host + '/ws/agent');

  ws.onopen = function () {
    term.writeln('\x1b[32m[WS connected]\x1b[0m');
  };

  ws.onmessage = function (e) {
    try {
      var data = JSON.parse(e.data);
      if (data.type === 'hydration') {
        applyHydration(data);
      } else {
        handleEvent(data);
      }
    } catch (ex) {}
  };

  ws.onclose = function () {
    term.writeln('\x1b[31m[WS disconnected]\x1b[0m');
    setTimeout(connectWS, 3000);
  };

  ws.onerror = function () {};
}

function applyHydration(state) {
  if (state.running) {
    setRunning(true);
  }
  if (state.terminal_history && state.terminal_history.length) {
    state.terminal_history.forEach(function (line) {
      term.write(line);
    });
  }
  if (state.phase) {
    addPhaseDrawer(state.phase);
  }
}

// === Input ===
agentInput.addEventListener('input', function () {
  agentSend.disabled = this.value.trim() === '' || isRunning;
});

agentInput.addEventListener('keydown', function (e) {
  if (e.key === 'Enter' && !e.shiftKey && !agentSend.disabled) {
    e.preventDefault();
    document.getElementById('agentForm').dispatchEvent(new Event('submit'));
  }
});

// === Submit ===
document.getElementById('agentForm').addEventListener('submit', function (e) {
  e.preventDefault();
  if (isRunning) return;

  var input = agentInput.value.trim();
  if (!input) return;

  agentInput.value = '';
  agentSend.disabled = true;
  lastGoal = input;
  setRunning(true);
  addChatMsg('user', input);
  term.writeln('\x1b[90m> ' + input + '\x1b[0m');

  // Enviar via WebSocket se conectado, senao fallback SSE
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'input', content: input }));
  } else {
    fallbackSSE(input);
  }
});

// === SSE Fallback ===
function fallbackSSE(input) {
  fetch('/api/agent/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input: input })
  }).then(function (response) {
    if (!response.ok) {
      return response.json().then(function (d) { throw new Error(d.error || 'Erro'); });
    }
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

// === Event Handler ===
function handleEvent(data) {
  switch (data.type) {
    case 'phase':
      addPhaseDrawer(data.content);
      break;
    case 'step':
      addStepPill(data);
      break;
    case 'stream':
      if (data.tool === 'shell' || data.tool === 'repl') {
        term.write(data.content);
        activateTab('terminal');
      } else if (data.tool === 'thought') {
        appendThought(data.content);
      }
      break;
    case 'terminal':
      term.write(data.content);
      activateTab('terminal');
      break;
    case 'message':
      addChatMsg('assistant', data.content);
      break;
    case 'diff':
      showDiff(data.name, data.content);
      activateTab('diff');
      break;
    case 'confirm':
      showConfirmModal(data.name, data.tool, data.content);
      break;
    case 'done':
      finish(lastGoal);
      break;
  }
}

function activateTab(name) {
  document.querySelectorAll('.tab').forEach(function (t) { t.classList.remove('active'); });
  document.querySelectorAll('.tab-content').forEach(function (c) { c.classList.remove('active'); });
  document.querySelector('.tab[data-tab="' + name + '"]').classList.add('active');
  document.getElementById('tab' + capitalize(name)).classList.add('active');
}

// === Chat Messages ===
function addChatMsg(type, content) {
  var msg = document.createElement('div');
  msg.className = 'chat-msg ' + type;
  msg.textContent = content;
  chatPanel.appendChild(msg);
  chatPanel.scrollTop = chatPanel.scrollHeight;
  return msg;
}

// === Thought Streaming ===
var thoughtEl = null;
function appendThought(token) {
  if (!thoughtEl) {
    thoughtEl = document.createElement('div');
    thoughtEl.className = 'chat-msg thought';
    chatPanel.appendChild(thoughtEl);
  }
  thoughtEl.textContent += token;
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

// === Phase Drawers ===
function addPhaseDrawer(phase) {
  thoughtEl = null; // reset thought accumulator on new phase
  var drawer = document.createElement('div');
  drawer.className = 'msg-drawer';
  var toggle = document.createElement('button');
  toggle.className = 'drawer-toggle';
  toggle.textContent = '\u25B6 ' + phase;
  drawer.appendChild(toggle);
  var body = document.createElement('div');
  body.className = 'drawer-body';
  drawer.appendChild(body);
  toggle.addEventListener('click', function () {
    body.classList.toggle('open');
    toggle.textContent = (body.classList.contains('open') ? '\u25BC ' : '\u25B6 ') + phase;
  });
  chatPanel.appendChild(drawer);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

// === Step Pills ===
function addStepPill(data) {
  var pill = document.createElement('span');
  pill.className = 'step-pill ' + data.status;
  pill.innerHTML = '<span class="step-dot"></span>' + (data.tool || data.name) + ' \u2192 ' + data.status;
  if (data.elapsed_ms) pill.innerHTML += ' <small>(' + data.elapsed_ms + 'ms)</small>';
  chatPanel.appendChild(pill);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

// === Diff Viewer ===
function showDiff(file, content) {
  var parts = content.split('\n---SEPARATOR---\n');
  if (parts.length < 2) return;
  var before = parts[0].split('\n');
  var after = parts[1].split('\n');
  document.getElementById('diffEmpty').style.display = 'none';
  document.getElementById('diffViewer').style.display = 'block';
  document.getElementById('diffHeader').textContent = file;
  var diffEl = document.getElementById('diffContent');
  diffEl.innerHTML = '';
  var maxLen = Math.max(before.length, after.length);
  for (var i = 0; i < maxLen; i++) {
    var bLine = before[i] || '';
    var aLine = after[i] || '';
    if (bLine !== aLine) {
      if (bLine) { var del = document.createElement('div'); del.className = 'diff-del'; del.textContent = '- ' + bLine; diffEl.appendChild(del); }
      if (aLine) { var add = document.createElement('div'); add.className = 'diff-add'; add.textContent = '+ ' + aLine; diffEl.appendChild(add); }
    } else {
      var same = document.createElement('div'); same.textContent = '  ' + aLine; diffEl.appendChild(same);
    }
  }
}

// === HIL Confirm Modal ===
function showConfirmModal(id, tool, summary) {
  currentConfirmId = id;
  document.getElementById('confirmTool').textContent = tool;
  document.getElementById('confirmSummary').textContent = summary;
  document.getElementById('confirmModal').style.display = 'flex';
  agentInput.disabled = true;
  var remaining = 30;
  document.getElementById('confirmTimer').textContent = remaining + 's';
  confirmTimerInterval = setInterval(function () {
    remaining--;
    document.getElementById('confirmTimer').textContent = remaining + 's';
    if (remaining <= 0) respondConfirm(false);
  }, 1000);
}

function hideConfirmModal() {
  document.getElementById('confirmModal').style.display = 'none';
  agentInput.disabled = false;
  currentConfirmId = null;
  if (confirmTimerInterval) { clearInterval(confirmTimerInterval); confirmTimerInterval = null; }
}

function respondConfirm(approved) {
  if (!currentConfirmId) return;
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'confirm', id: currentConfirmId, approved: approved }));
  } else {
    fetch('/api/agent/confirm/' + currentConfirmId, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approved: approved })
    }).catch(function () {});
  }
  hideConfirmModal();
}

document.getElementById('confirmApprove').addEventListener('click', function () { respondConfirm(true); });
document.getElementById('confirmReject').addEventListener('click', function () { respondConfirm(false); });

// === Stop Button ===
stopBtn.addEventListener('click', function () {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'cancel' }));
  }
  fetch('/api/agent/cancel', { method: 'POST' }).catch(function () {});
});

// === Running State ===
function setRunning(running) {
  isRunning = running;
  agentSend.disabled = agentInput.value.trim() === '' || running;
  document.querySelector('.status-dot').className = 'status-dot ' + (running ? 'running' : 'idle');
  document.querySelector('.status-text').textContent = running ? 'running' : 'idle';
  stopBtn.style.display = running ? 'inline-block' : 'none';
}

function finish(goal) {
  thoughtEl = null;
  setRunning(false);
  executions.unshift({ goal: goal, status: 'done', time: new Date().toLocaleTimeString() });
  renderReplayList();
  loadJobs();
}

// === Replay List ===
function renderReplayList() {
  var list = document.getElementById('replayList');
  list.innerHTML = '';
  if (!executions.length) { list.innerHTML = '<div class="section-empty">Sem historico</div>'; return; }
  executions.slice(0, 10).forEach(function (exec) {
    var item = document.createElement('div');
    item.className = 'replay-item';
    item.innerHTML = '<span class="replay-goal">' + esc(exec.goal || '?') + '</span>' +
      '<span class="replay-status ' + exec.status + '">' + exec.status + '</span>';
    list.appendChild(item);
  });
}

// === Load Data ===
function loadJobs() {
  fetch('/api/scheduler/jobs').then(function (r) { return r.json(); }).then(function (jobs) {
    var list = document.getElementById('jobsList');
    list.innerHTML = '';
    if (!jobs || !jobs.length) { list.innerHTML = '<div class="section-empty">Sem jobs</div>'; return; }
    jobs.forEach(function (job) {
      var item = document.createElement('div');
      item.className = 'job-item';
      item.innerHTML = '<span class="job-name">' + esc(job.name) + '</span><span class="job-interval">' + job.interval_seconds + 's</span>';
      var toggle = document.createElement('button');
      toggle.className = 'job-toggle ' + (job.enabled ? 'on' : 'off');
      toggle.addEventListener('click', function () {
        fetch('/api/scheduler/jobs/' + job.id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled: !job.enabled }) }).then(function () { loadJobs(); });
      });
      item.appendChild(toggle);
      list.appendChild(item);
    });
  }).catch(function () {});
}

function loadStatus() {
  fetch('/api/agent/status').then(function (r) { return r.json(); }).then(function (data) {
    var list = document.getElementById('channelsList');
    list.innerHTML = '';
    [{ name: 'Web', on: true }, { name: 'WhatsApp', on: data.whatsapp_enabled }, { name: 'Telegram', on: data.telegram_enabled }, { name: 'Discord', on: data.discord_enabled }].forEach(function (ch) {
      var item = document.createElement('div');
      item.className = 'channel-item';
      item.innerHTML = '<span class="channel-dot ' + (ch.on ? 'on' : 'off') + '"></span><span>' + ch.name + '</span>';
      list.appendChild(item);
    });
  }).catch(function () {});
}

function esc(text) { var div = document.createElement('div'); div.textContent = text; return div.innerHTML; }

// === Thought message style ===
var style = document.createElement('style');
style.textContent = '.chat-msg.thought { font-size: 12px; color: #888; font-style: italic; background: transparent; border: 1px dashed var(--color-border); align-self: flex-start; max-width: 90%; }';
document.head.appendChild(style);

// === Init ===
connectWS();
loadJobs();
loadStatus();
