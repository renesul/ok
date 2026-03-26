// Sidebar navigation
document.querySelectorAll('.nav-item[data-section]').forEach(function (item) {
  item.addEventListener('click', function () {
    document.querySelectorAll('.nav-item').forEach(function (i) { i.classList.remove('active'); });
    document.querySelectorAll('.config-panel').forEach(function (p) { p.classList.remove('active'); });
    this.classList.add('active');
    document.getElementById('panel-' + this.dataset.section).classList.add('active');
    // Auto-resize textareas na secao que ficou visivel
    document.querySelectorAll('#panel-' + this.dataset.section + ' .config-textarea').forEach(function (ta) {
      autoResize(ta);
    });
    // Atualizar URL
    history.pushState(null, '', '/profile/' + this.dataset.section);
    // Close mobile sidebar
    document.getElementById('sidebar').classList.remove('open');
  });
});

// Detectar secao pela URL (ex: /profile/scheduler)
(function () {
  var parts = window.location.pathname.split('/');
  // /profile/scheduler → parts = ['', 'profile', 'scheduler']
  var section = parts.length >= 3 ? parts[2] : '';
  if (section) {
    var target = document.querySelector('.nav-item[data-section="' + section + '"]');
    if (target) target.click();
  }
})();

// Auto-expand textareas
function autoResize(el) {
  el.style.height = 'auto';
  el.style.height = el.scrollHeight + 'px';
}
document.querySelectorAll('.config-textarea').forEach(function (ta) {
  ta.addEventListener('input', function () { autoResize(this); });
});

// Mobile sidebar toggle
document.getElementById('menuToggle').addEventListener('click', function () {
  document.getElementById('sidebar').classList.toggle('open');
});

// Soul
fetch('/api/agent/config/soul')
  .then(function (r) { return r.ok ? r.json() : null; })
  .then(function (data) {
    if (data && data.value) {
      var el = document.getElementById('soulText');
      el.value = data.value;
      autoResize(el);
    }
  })
  .catch(function () {});

document.getElementById('saveSoul').addEventListener('click', function () {
  var text = document.getElementById('soulText').value.trim();
  var status = document.getElementById('soulStatus');
  if (!text) return;

  fetch('/api/agent/config/soul', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ value: text })
  })
    .then(function (r) {
      if (!r.ok) throw new Error('Erro');
      status.textContent = 'Salvo!';
      status.className = 'config-status success';
    })
    .catch(function (e) {
      status.textContent = e.message;
      status.className = 'config-status error';
    });
});

// Generic template loader/saver
function loadTemplate(key, textareaId) {
  fetch('/api/agent/config/' + key)
    .then(function (r) { return r.ok ? r.json() : null; })
    .then(function (data) { if (data && data.value) { var el = document.getElementById(textareaId); el.value = data.value; autoResize(el); } })
    .catch(function () {});
}

function saveTemplate(key, textareaId, statusId) {
  var text = document.getElementById(textareaId).value.trim();
  var status = document.getElementById(statusId);
  if (!text) return;
  fetch('/api/agent/config/' + key, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ value: text })
  })
    .then(function (r) { if (!r.ok) throw new Error('Erro'); status.textContent = 'Salvo!'; status.className = 'config-status success'; })
    .catch(function (e) { status.textContent = e.message; status.className = 'config-status error'; });
}

// Identity
loadTemplate('identity', 'identityText');
document.getElementById('saveIdentity').addEventListener('click', function () { saveTemplate('identity', 'identityText', 'identityStatus'); });

// User
loadTemplate('user_profile', 'userText');
document.getElementById('saveUser').addEventListener('click', function () { saveTemplate('user_profile', 'userText', 'userStatus'); });

// Environment
loadTemplate('environment_notes', 'environmentText');
document.getElementById('saveEnvironment').addEventListener('click', function () { saveTemplate('environment_notes', 'environmentText', 'environmentStatus'); });

// Config publica + LLM/Embed/Sistema
fetch('/api/config')
  .then(function (r) { return r.json(); })
  .then(function (cfg) {
    document.getElementById('llmBaseUrl').textContent = cfg.llm_base_url || '—';
    document.getElementById('llmModel').textContent = cfg.llm_model || '—';
    document.getElementById('embedProvider').textContent = cfg.embed_provider || '—';
    document.getElementById('embedModel').textContent = cfg.embed_model || '—';
    document.getElementById('agentSandbox').textContent = cfg.agent_sandbox || '—';
    document.getElementById('serverPort').textContent = cfg.server_port || '—';
    document.getElementById('debugMode').textContent = cfg.debug ? 'ON' : 'OFF';
  })
  .catch(function () {});

// Health services (LLM + Embed status)
fetch('/api/health/services')
  .then(function (r) { return r.json(); })
  .then(function (data) {
    if (data.llm) {
      document.getElementById('llmStatus').textContent = data.llm.status + (data.llm.latency_ms ? ' (' + data.llm.latency_ms + 'ms)' : '');
      document.getElementById('llmStatus').style.color = data.llm.status === 'ok' ? 'var(--color-success)' : 'var(--color-error)';
    }
    if (data.embedding) {
      document.getElementById('embedStatus').textContent = data.embedding.status + (data.embedding.latency_ms ? ' (' + data.embedding.latency_ms + 'ms)' : '');
      document.getElementById('embedStatus').style.color = data.embedding.status === 'ok' ? 'var(--color-success)' : 'var(--color-error)';
    }
  })
  .catch(function () {});

// Channels
fetch('/api/agent/status')
  .then(function (r) { return r.json(); })
  .then(function (data) {
    var list = document.getElementById('channelsList');
    [
      { name: 'Web', on: true },
      { name: 'WhatsApp', on: data.whatsapp_enabled },
      { name: 'Telegram', on: data.telegram_enabled },
      { name: 'Discord', on: data.discord_enabled }
    ].forEach(function (ch) {
      var card = document.createElement('div');
      card.className = 'channel-card';
      card.innerHTML = '<span class="channel-dot ' + (ch.on ? 'on' : 'off') + '"></span>' +
        '<span class="channel-card-name">' + ch.name + '</span>' +
        '<span class="channel-card-status">' + (ch.on ? 'ativo' : 'inativo') + '</span>';
      list.appendChild(card);
    });
  })
  .catch(function () {});

// ============ Agent Limits ============

fetch('/api/agent/limits')
  .then(function (r) { return r.ok ? r.json() : null; })
  .then(function (limits) {
    if (!limits) return;
    document.getElementById('limit-steps').value = limits.max_steps;
    document.getElementById('limit-attempts').value = limits.max_attempts;
    document.getElementById('limit-timeout').value = limits.timeout_ms;
  })
  .catch(function () {});

document.getElementById('saveAgentConfig').addEventListener('click', function () {
  var status = document.getElementById('agentConfigStatus');
  var limits = {
    max_steps: parseInt(document.getElementById('limit-steps').value) || 6,
    max_attempts: parseInt(document.getElementById('limit-attempts').value) || 4,
    timeout_ms: parseInt(document.getElementById('limit-timeout').value) || 120000
  };

  if (limits.max_steps <= 0 || limits.max_attempts <= 0 || limits.timeout_ms <= 0) {
    status.textContent = 'Todos os valores devem ser > 0';
    status.className = 'config-status error';
    return;
  }

  fetch('/api/agent/limits', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(limits) })
    .then(function (r) { if (!r.ok) return r.json().then(function (d) { throw new Error(d.error || 'Erro'); }); return r.json(); })
    .then(function () { status.textContent = 'Salvo!'; status.className = 'config-status success'; })
    .catch(function (e) { status.textContent = e.message; status.className = 'config-status error'; });
});

// Jobs
function loadJobs() {
  fetch('/api/scheduler/jobs')
    .then(function (r) { return r.json(); })
    .then(function (jobs) {
      var list = document.getElementById('jobsList');
      list.innerHTML = '';
      if (!jobs || !jobs.length) {
        list.innerHTML = '<div style="font-size:12px;color:#999">Sem jobs</div>';
        return;
      }
      jobs.forEach(function (job) {
        var row = document.createElement('div');
        row.className = 'job-row';
        row.innerHTML = '<span class="job-row-name">' + job.name + '</span>' +
          '<span class="job-row-interval">' + job.interval_seconds + 's</span>';

        var toggle = document.createElement('button');
        toggle.className = 'job-row-toggle ' + (job.enabled ? 'on' : 'off');
        toggle.addEventListener('click', function () {
          fetch('/api/scheduler/jobs/' + job.id, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ enabled: !job.enabled })
          }).then(function () { loadJobs(); });
        });
        row.appendChild(toggle);

        var del = document.createElement('button');
        del.className = 'job-row-delete';
        del.textContent = 'x';
        del.addEventListener('click', function () {
          fetch('/api/scheduler/jobs/' + job.id, { method: 'DELETE' }).then(function () { loadJobs(); });
        });
        row.appendChild(del);

        list.appendChild(row);
      });
    })
    .catch(function () {});
}
loadJobs();

// New job form
document.getElementById('newJobForm').addEventListener('submit', function (e) {
  e.preventDefault();
  var form = this;
  var status = document.getElementById('jobStatus');

  fetch('/api/scheduler/jobs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: form.name.value,
      task_type: form.task_type.value,
      input: form.input.value,
      interval_seconds: parseInt(form.interval_seconds.value) || 120
    })
  })
    .then(function (r) {
      if (!r.ok) return r.json().then(function (d) { throw new Error(d.error); });
      return r.json();
    })
    .then(function () {
      status.textContent = 'Job criado!';
      status.className = 'config-status success';
      form.reset();
      loadJobs();
    })
    .catch(function (err) {
      status.textContent = err.message;
      status.className = 'config-status error';
    });
});

// Import ChatGPT
var importFile = document.getElementById('importFile');
var importLabel = document.getElementById('importLabel');
var importText = document.getElementById('importText');
var importButton = document.getElementById('importButton');
var importStatus = document.getElementById('importStatus');
var importProgress = document.getElementById('importProgress');
var importProgressFill = document.getElementById('importProgressFill');
var importProgressText = document.getElementById('importProgressText');

importFile.addEventListener('change', function () {
  if (this.files.length > 0) {
    importText.textContent = this.files[0].name;
    importLabel.classList.add('has-file');
    importButton.disabled = false;
    importStatus.textContent = '';
    importStatus.className = 'import-status';
  } else {
    importText.textContent = 'Select .zip file';
    importLabel.classList.remove('has-file');
    importButton.disabled = true;
  }
});

document.getElementById('importForm').addEventListener('submit', function (e) {
  e.preventDefault();
  var file = importFile.files[0];
  if (!file) return;

  importButton.disabled = true;
  importStatus.textContent = '';
  importStatus.className = 'import-status';
  importProgress.classList.add('active');
  importProgressFill.style.width = '0%';
  importProgressText.textContent = 'Uploading... 0%';

  var formData = new FormData();
  formData.append('file', file);

  var xhr = new XMLHttpRequest();
  xhr.open('POST', '/api/import/chatgpt');

  xhr.upload.addEventListener('progress', function (evt) {
    if (evt.lengthComputable) {
      var pct = Math.round((evt.loaded / evt.total) * 100);
      importProgressFill.style.width = pct + '%';
      importProgressText.textContent = 'Uploading... ' + pct + '%';
      if (pct === 100) {
        importProgressText.textContent = 'Processing conversations...';
      }
    }
  });

  xhr.addEventListener('load', function () {
    importProgress.classList.remove('active');
    try {
      var data = JSON.parse(xhr.responseText);
      if (xhr.status >= 200 && xhr.status < 300) {
        importProgressFill.style.width = '100%';
        importStatus.textContent = data.message;
        importStatus.className = 'import-status success';
        importFile.value = '';
        importText.textContent = 'Select .zip file';
        importLabel.classList.remove('has-file');
      } else {
        importStatus.textContent = data.error || 'Import failed';
        importStatus.className = 'import-status error';
        importButton.disabled = false;
      }
    } catch (err) {
      importStatus.textContent = 'Unexpected response';
      importStatus.className = 'import-status error';
      importButton.disabled = false;
    }
  });

  xhr.addEventListener('error', function () {
    importProgress.classList.remove('active');
    importStatus.textContent = 'Connection failed';
    importStatus.className = 'import-status error';
    importButton.disabled = false;
  });

  xhr.send(formData);
});
