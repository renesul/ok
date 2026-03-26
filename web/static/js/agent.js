// === State ===
var isRunning = false;
var executions = [];
var currentConfirmId = null;
var confirmTimerInterval = null;
var ws = null;
var lastGoal = '';
var currentStreams = {};
var thoughtEl = null;

// === DOM ===
var agentInput = document.getElementById('agentInput');
var agentSend = document.getElementById('agentSend');
var chatPanel = document.getElementById('chatPanel');
var stopBtn = document.getElementById('stopBtn');

// === Resizers ===
setupResizer('resizer2', 'panelLeft', 'panelRight');

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
    if (newLeft > 300 && newRight > 150) {
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
  ws.onmessage = function (e) {
    try {
      var data = JSON.parse(e.data);
      if (data.type === 'hydration') applyHydration(data);
      else handleEvent(data);
    } catch (ex) {}
  };
  ws.onclose = function () { setTimeout(connectWS, 3000); };
}

function applyHydration(state) {
  if (state.running) setRunning(true);
  if (state.phase) addPhaseDrawer(state.phase);
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
document.getElementById('agentForm').addEventListener('submit', function (e) {
  e.preventDefault();
  if (isRunning) return;
  var input = agentInput.value.trim();
  if (!input) return;

  agentInput.value = '';
  agentSend.disabled = true;
  lastGoal = input;
  setRunning(true);
  
  // Clear original greeting only once
  var greet = document.querySelector('.chat-greeting');
  if (greet) greet.style.display = 'none';

  addChatMsg('user', input);

  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'input', content: input }));
  } else {
    fallbackSSE(input);
  }
});

function fallbackSSE(input) {
  fetch('/api/agent/stream', {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ input: input })
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

// === Chat UI Generators ===
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
    wrapper.innerHTML = '<div class="terminal-header"><span class="terminal-dots"><span></span><span></span><span></span></span> bash (' + tool + ')</div><pre class="terminal-body scrollbar-hidden"></pre>';
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
  wrapper.innerHTML = '<div class="diff-header"><span>Git Diff</span> &bull; <span class="diff-file">'+file+'</span></div><div class="diff-body"></div>';
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
      if (bLine) { var del = document.createElement('div'); del.className = 'diff-line del'; del.textContent = '- ' + bLine; diffBody.appendChild(del); }
      if (aLine) { var add = document.createElement('div'); add.className = 'diff-line add'; add.textContent = '+ ' + aLine; diffBody.appendChild(add); }
    } else {
      var same = document.createElement('div'); same.className = 'diff-line'; same.textContent = '  ' + aLine; diffBody.appendChild(same);
    }
  }
  chatPanel.appendChild(wrapper);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function addPhaseDrawer(phase) {
  thoughtEl = null; 
  currentStreams = {}; // Reset streams context
  var pill = document.createElement('div');
  pill.className = 'phase-divider';
  pill.innerHTML = '<span>🚀 Execution Phase:' + phase.toUpperCase() + '</span>';
  chatPanel.appendChild(pill);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

function addStepPill(data) {
  var wrapper = document.createElement('div');
  wrapper.className = 'step-pill-container';
  var pill = document.createElement('span');
  pill.className = 'step-pill ' + data.status;
  pill.innerHTML = '<span class="step-dot"></span> ' + (data.tool || data.name) + ' <span class="step-status">[' + data.status + ']</span>';
  if (data.elapsed_ms) pill.innerHTML += ' <i>(' + data.elapsed_ms + 'ms)</i>';
  wrapper.appendChild(pill);
  chatPanel.appendChild(wrapper);
  chatPanel.scrollTop = chatPanel.scrollHeight;
}

// === Stop & Finish ===
stopBtn.addEventListener('click', function () {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'cancel' }));
  fetch('/api/agent/cancel', { method: 'POST' }).catch(function () {});
});

function setRunning(running) {
  isRunning = running;
  agentSend.disabled = agentInput.value.trim() === '' || running;
  document.querySelector('.status-dot').className = 'status-dot ' + (running ? 'running' : 'idle');
  document.querySelector('.status-text').textContent = running ? 'Processing' : 'Online';
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

// === Sidebar & HIL Modal ===
function renderReplayList() {
  var list = document.getElementById('replayList');
  list.innerHTML = '';
  if (!executions.length) { list.innerHTML = '<div class="section-empty">No recent commands</div>'; return; }
  executions.slice(0, 10).forEach(function (exec) {
    var item = document.createElement('div');
    item.className = 'replay-item';
    item.innerHTML = '<span class="replay-goal">' + esc(exec.goal || '?') + '</span><span class="replay-status ' + exec.status + '">' + exec.status + '</span>';
    list.appendChild(item);
  });
}

function loadJobs() {
  fetch('/api/scheduler/jobs').then(function (r) { return r.json(); }).then(function (jobs) {
    var list = document.getElementById('jobsList');
    list.innerHTML = '';
    if (!jobs || !jobs.length) { list.innerHTML = '<div class="section-empty">No scheduled tasks</div>'; return; }
    jobs.forEach(function (job) {
      var item = document.createElement('div');
      item.className = 'job-item';
      item.innerHTML = '<span class="job-name">' + esc(job.name) + '</span><span class="job-interval">' + job.interval_seconds + 's</span>';
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

function showConfirmModal(id, tool, summary) {
  currentConfirmId = id;
  document.getElementById('confirmTool').textContent = "Tool invoked: " + tool;
  document.getElementById('confirmSummary').textContent = summary;
  document.getElementById('confirmModal').style.display = 'flex';
  agentInput.disabled = true;
  var remaining = 30;
  document.getElementById('confirmTimer').textContent = remaining + 's';
  confirmTimerInterval = setInterval(function () {
    remaining--; document.getElementById('confirmTimer').textContent = remaining + 's';
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
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'confirm', id: currentConfirmId, approved: approved }));
  hideConfirmModal();
}
document.getElementById('confirmApprove').addEventListener('click', function () { respondConfirm(true); });
document.getElementById('confirmReject').addEventListener('click', function () { respondConfirm(false); });
function esc(t) { var d = document.createElement('div'); d.textContent = t; return d.innerHTML; }

// === Init ===
connectWS();
loadJobs();
loadStatus();
