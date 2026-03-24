var activeConversationID = null;
var searchTimeout = null;
var isStreaming = false;

var chatInput = document.getElementById('chatInput');
var sendButton = document.getElementById('chatSendButton');
var chatInputArea = document.getElementById('chatInputArea');

// Sidebar toggle
document.getElementById('menuToggle').addEventListener('click', function () {
  document.getElementById('sidebar').classList.add('open');
  document.getElementById('sidebarOverlay').classList.add('visible');
});

document.getElementById('sidebarClose').addEventListener('click', closeSidebar);
document.getElementById('sidebarOverlay').addEventListener('click', closeSidebar);

function closeSidebar() {
  document.getElementById('sidebar').classList.remove('open');
  document.getElementById('sidebarOverlay').classList.remove('visible');
}

// New conversation
document.getElementById('newChatButton').addEventListener('click', function () {
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
      selectConversation(conversation.id, conversation.title || 'Nova conversa');
      loadConversations();
      closeSidebar();
    });
});

// Search with debounce
document.getElementById('searchInput').addEventListener('input', function () {
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

// Logout
document.getElementById('logoutButton').addEventListener('click', function () {
  fetch('/api/auth/logout', { method: 'POST' })
    .then(function () {
      window.location.href = '/login';
    });
});

// Auto-resize textarea
chatInput.addEventListener('input', function () {
  this.style.height = 'auto';
  this.style.height = Math.min(this.scrollHeight, 120) + 'px';
  sendButton.disabled = this.value.trim() === '';
});

// Enter to send, Shift+Enter for newline
chatInput.addEventListener('keydown', function (event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    if (!sendButton.disabled && !isStreaming) {
      document.getElementById('chatInputForm').dispatchEvent(new Event('submit'));
    }
  }
});

// Send message with SSE streaming
document.getElementById('chatInputForm').addEventListener('submit', function (event) {
  event.preventDefault();
  if (isStreaming || !activeConversationID) return;

  var content = chatInput.value.trim();
  if (!content) return;

  chatInput.value = '';
  chatInput.style.height = 'auto';
  sendButton.disabled = true;
  isStreaming = true;

  // Append user message immediately
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
          throw new Error(data.error || 'Erro ao enviar mensagem.');
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
                assistantBubble.textContent = 'Erro: ' + data.error;
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
      assistantBubble.textContent = 'Erro: ' + error.message;
      finishStreaming(assistantBubble);
    });
});

function finishStreaming(bubble) {
  isStreaming = false;
  sendButton.disabled = chatInput.value.trim() === '';
  if (bubble.parentElement) {
    bubble.parentElement.classList.remove('message-streaming');
    // Remove step indicators after completion
    var indicators = bubble.parentElement.querySelectorAll('.step-indicator');
    indicators.forEach(function (el) {
      el.classList.add('step-done');
    });
  }
  loadConversations();
}

function showModeBadge(bubble, mode) {
  if (!bubble.parentElement) return;
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
  if (!bubble.parentElement) return;
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

function appendMessage(role, content) {
  var container = document.getElementById('chatMessages');
  var empty = document.getElementById('chatEmpty');
  empty.style.display = 'none';

  var list = container.querySelector('.message-list');
  if (!list) {
    list = document.createElement('div');
    list.className = 'message-list';
    container.appendChild(list);
  }

  var wrapper = document.createElement('div');
  wrapper.className = 'message message-' + role;

  var roleLabel = document.createElement('div');
  roleLabel.className = 'message-role';
  roleLabel.textContent = role === 'user' ? 'Voce' : 'Assistente';

  var bubble = document.createElement('div');
  bubble.className = 'message-bubble';
  bubble.textContent = content;

  wrapper.appendChild(roleLabel);
  wrapper.appendChild(bubble);
  list.appendChild(wrapper);

  scrollToBottom();
  return bubble;
}

function scrollToBottom() {
  var container = document.getElementById('chatMessages');
  container.scrollTop = container.scrollHeight;
}

// Load conversations
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

// Search conversations
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

// Render sidebar list
function renderConversationList(conversations) {
  var list = document.getElementById('conversationList');
  list.innerHTML = '';

  if (!conversations || conversations.length === 0) {
    var empty = document.createElement('div');
    empty.className = 'conversation-empty';
    empty.textContent = 'Nenhuma conversa';
    list.appendChild(empty);
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
    titleSpan.textContent = conv.title || 'Sem titulo';
    item.appendChild(titleSpan);

    var deleteBtn = document.createElement('button');
    deleteBtn.className = 'conversation-delete';
    deleteBtn.innerHTML = '&times;';
    deleteBtn.setAttribute('aria-label', 'Excluir conversa');
    deleteBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      fetch('/api/conversations/' + conv.id, { method: 'DELETE' })
        .then(function (response) {
          if (!response.ok) throw new Error('failed');
          if (conv.id === activeConversationID) {
            activeConversationID = null;
            document.getElementById('chatTitle').textContent = 'OK';
            chatInputArea.style.display = 'none';
            clearMessageList();
            var empty = document.getElementById('chatEmpty');
            empty.style.display = 'flex';
            empty.querySelector('.chat-empty-text').textContent = 'Selecione uma conversa';
            history.pushState(null, '', '/chat');
          }
          loadConversations();
        });
    });
    item.appendChild(deleteBtn);

    item.addEventListener('click', function () {
      selectConversation(conv.id, conv.title);
    });
    list.appendChild(item);
  });
}

// Select conversation
function selectConversation(id, title, skipPush) {
  activeConversationID = id;
  document.getElementById('chatTitle').textContent = title || 'Sem titulo';
  chatInputArea.style.display = 'block';

  // Update URL
  if (!skipPush) {
    history.pushState({ id: id, title: title }, '', '/chat/' + id);
  }

  // Update active state
  var items = document.querySelectorAll('.conversation-item');
  items.forEach(function (item) {
    item.classList.toggle('active', item.dataset.id == id);
  });

  closeSidebar();

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

// Render messages
function clearMessageList() {
  var container = document.getElementById('chatMessages');
  var existing = container.querySelector('.message-list');
  if (existing) existing.remove();
}

function renderMessages(messages) {
  var container = document.getElementById('chatMessages');
  var empty = document.getElementById('chatEmpty');

  clearMessageList();

  if (!messages || messages.length === 0) {
    empty.style.display = 'flex';
    empty.querySelector('.chat-empty-text').textContent = 'Comece uma conversa';
    return;
  }

  empty.style.display = 'none';

  var list = document.createElement('div');
  list.className = 'message-list';

  messages.forEach(function (msg) {
    var wrapper = document.createElement('div');
    wrapper.className = 'message message-' + msg.role;

    var role = document.createElement('div');
    role.className = 'message-role';
    role.textContent = msg.role === 'user' ? 'Voce' : 'Assistente';

    var bubble = document.createElement('div');
    bubble.className = 'message-bubble';
    bubble.textContent = msg.content;

    wrapper.appendChild(role);
    wrapper.appendChild(bubble);
    list.appendChild(wrapper);
  });

  container.appendChild(list);
  container.scrollTop = container.scrollHeight;
}

// Health check
function checkServices() {
  var report = document.getElementById('healthReport');
  var items = document.getElementById('healthItems');

  fetch('/api/health/services')
    .then(function (response) {
      if (!response.ok) return null;
      return response.json();
    })
    .then(function (data) {
      if (!data) return;

      items.innerHTML = '';
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
          disabledText.textContent = 'nao configurado';
          item.appendChild(disabledText);
        }

        items.appendChild(item);
      });

      report.style.display = 'block';
      sessionStorage.setItem('healthChecked', '1');

      if (!hasError) {
        setTimeout(function () { dismissReport(); }, 3000);
      }
    })
    .catch(function () {});
}

function dismissReport() {
  var report = document.getElementById('healthReport');
  report.classList.add('fade-out');
  setTimeout(function () {
    report.style.display = 'none';
    report.classList.remove('fade-out');
  }, 300);
}

document.getElementById('healthDismiss').addEventListener('click', dismissReport);

// Browser back/forward
window.addEventListener('popstate', function (event) {
  if (event.state && event.state.id) {
    selectConversation(event.state.id, event.state.title, true);
  } else {
    activeConversationID = null;
    document.getElementById('chatTitle').textContent = 'OK';
    chatInputArea.style.display = 'none';
    clearMessageList();
    var empty = document.getElementById('chatEmpty');
    empty.style.display = 'flex';
    empty.querySelector('.chat-empty-text').textContent = 'Selecione uma conversa';
    var items = document.querySelectorAll('.conversation-item');
    items.forEach(function (item) { item.classList.remove('active'); });
  }
});

// Init
loadConversations();

// Open conversation from URL (/chat/123)
var pathMatch = window.location.pathname.match(/^\/chat\/(\d+)$/);
if (pathMatch) {
  var urlConvID = parseInt(pathMatch[1], 10);
  fetch('/api/conversations/' + urlConvID + '/messages')
    .then(function (response) {
      if (!response.ok) throw new Error('not found');
      selectConversation(urlConvID, '', true);
    })
    .catch(function () {
      history.replaceState(null, '', '/chat');
    });
}

if (!sessionStorage.getItem('healthChecked')) {
  checkServices();
}
