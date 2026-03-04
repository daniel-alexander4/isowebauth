// Settings view — port of popup/index.ts

var KEY_PATHS = {
  ed25519: '~/.ssh/id_ed25519',
  rsa: '~/.ssh/id_rsa',
  ecdsa: '~/.ssh/id_ecdsa'
};
var NAMESPACE_REGEX = /^[A-Za-z0-9._:-]{1,64}$/;
var MAX_ORIGIN_ROWS = 100;

function setStatus(text, level) {
  var el = document.getElementById('status');
  if (!el) return;
  el.textContent = text;
  el.classList.remove('status-success', 'status-error');
  if (level) el.classList.add('status-' + level);
  el.hidden = !text;
}

function renderToggle(enabled) {
  var btn = document.getElementById('toggle-enabled');
  if (!btn) return;
  btn.setAttribute('aria-pressed', enabled ? 'true' : 'false');
  btn.textContent = enabled ? 'On' : 'Off';
}

function detectKeyType(keyPath) {
  var v = (keyPath || '').trim();
  if (v.endsWith('id_ed25519')) return 'ed25519';
  if (v.endsWith('id_rsa')) return 'rsa';
  if (v.endsWith('id_ecdsa')) return 'ecdsa';
  return null;
}

function renderKeyTypeTabs(active) {
  var tabs = document.querySelectorAll('#view-settings [data-key-type]:not([data-onboarding])');
  tabs.forEach(function(tab) {
    tab.setAttribute('aria-pressed', tab.dataset.keyType === active ? 'true' : 'false');
  });
}

function splitOriginParts(origin) {
  try {
    var url = new URL(origin);
    var port = url.port || '';
    var host = url.protocol + '//' + url.hostname;
    return { host: host, port: port };
  } catch (e) {
    return { host: origin, port: '' };
  }
}

function createOriginRow(origin, namespace, port) {
  origin = origin || '';
  namespace = namespace || '';
  port = port || '';

  var row = document.createElement('div');
  row.className = 'origin-row';

  var originInput = document.createElement('input');
  originInput.type = 'text';
  originInput.placeholder = 'https://example.com';
  originInput.value = origin;
  originInput.className = 'origin-input';
  originInput.addEventListener('input', function() { validateOriginRow(row); });

  var portInput = document.createElement('input');
  portInput.type = 'text';
  portInput.placeholder = 'port';
  portInput.value = port;
  portInput.className = 'port-input';
  portInput.addEventListener('input', function() { validateOriginRow(row); });

  var nsInput = document.createElement('input');
  nsInput.type = 'text';
  nsInput.placeholder = 'namespace';
  nsInput.value = namespace;
  nsInput.className = 'namespace-input';
  nsInput.addEventListener('input', function() { validateOriginRow(row); });

  var removeBtn = document.createElement('span');
  removeBtn.className = 'remove-row-btn';
  removeBtn.textContent = '\uD83D\uDDD1';
  removeBtn.setAttribute('role', 'button');
  removeBtn.tabIndex = 0;
  removeBtn.addEventListener('click', function() { row.remove(); });

  row.appendChild(originInput);
  row.appendChild(portInput);
  row.appendChild(nsInput);
  row.appendChild(removeBtn);
  return row;
}

function validateOriginRow(row) {
  var originInput = row.querySelector('.origin-input');
  var portInput = row.querySelector('.port-input');
  var nsInput = row.querySelector('.namespace-input');
  var valid = true;

  var rawOrigin = originInput.value.trim();
  if (rawOrigin) {
    try { new URL(rawOrigin); originInput.classList.remove('invalid'); }
    catch (e) { originInput.classList.add('invalid'); valid = false; }
  } else {
    originInput.classList.remove('invalid');
  }

  var portVal = portInput.value.trim();
  if (portVal) {
    var portNum = Number(portVal);
    if (!Number.isInteger(portNum) || portNum < 1 || portNum > 65535) {
      portInput.classList.add('invalid'); valid = false;
    } else {
      portInput.classList.remove('invalid');
    }
  } else {
    portInput.classList.remove('invalid');
  }

  var nsVal = nsInput.value.trim();
  if (nsVal && !NAMESPACE_REGEX.test(nsVal)) {
    nsInput.classList.add('invalid'); valid = false;
  } else {
    nsInput.classList.remove('invalid');
  }

  return valid;
}

function collectOriginScopes(containerId) {
  containerId = containerId || 'origin-rows';
  var container = document.getElementById(containerId);
  var out = {};
  var invalidLines = [];
  if (!container) return { values: out, invalidLines: invalidLines };

  var rows = container.querySelectorAll('.origin-row');
  if (rows.length > MAX_ORIGIN_ROWS) {
    return { values: out, invalidLines: ['Too many target systems (max ' + MAX_ORIGIN_ROWS + ')'] };
  }

  rows.forEach(function(row) {
    var originInput = row.querySelector('.origin-input');
    var portInput = row.querySelector('.port-input');
    var nsInput = row.querySelector('.namespace-input');
    var rawHost = originInput.value.trim();
    var rawPort = portInput.value.trim();
    var nsVal = nsInput.value.trim();

    if (!rawHost && !nsVal && !rawPort) return;

    if (!rawHost || !nsVal) {
      invalidLines.push((rawHost || '(empty host)') + ' \u2014 host and namespace are required');
      return;
    }

    if (rawPort) {
      var portNum = Number(rawPort);
      if (!Number.isInteger(portNum) || portNum < 1 || portNum > 65535) {
        invalidLines.push(rawHost + ' \u2014 invalid port "' + rawPort + '"');
        return;
      }
    }

    var fullUrl = rawPort ? rawHost + ':' + rawPort : rawHost;
    var origin = '';
    try { origin = new URL(fullUrl).origin; }
    catch (e) { invalidLines.push(fullUrl + ' \u2014 invalid URL'); return; }

    if (!NAMESPACE_REGEX.test(nsVal)) {
      invalidLines.push(rawHost + ' \u2014 invalid namespace "' + nsVal + '"');
      return;
    }

    if (!out[origin]) out[origin] = [];
    var exists = out[origin].some(function(e) { return e.namespace === nsVal; });
    if (!exists) out[origin].push({ namespace: nsVal });
  });

  return { values: out, invalidLines: invalidLines };
}

function renderOriginRows(scopes, containerId) {
  containerId = containerId || 'origin-rows';
  var container = document.getElementById(containerId);
  if (!container) return;
  container.innerHTML = '';

  var entries = Object.entries(scopes || {}).sort(function(a, b) { return a[0].localeCompare(b[0]); });
  entries.forEach(function(entry) {
    var origin = entry[0];
    var scopeList = entry[1];
    var parts = splitOriginParts(origin);
    (Array.isArray(scopeList) ? scopeList : []).forEach(function(scope) {
      container.appendChild(createOriginRow(parts.host, scope.namespace, parts.port));
    });
  });

  if (container.children.length === 0) {
    container.appendChild(createOriginRow());
  }
}

function setCheckItem(id, status, label) {
  var iconEl = document.getElementById(id + '-icon');
  var labelEl = document.getElementById(id + '-label');
  if (iconEl) {
    iconEl.className = 'check-icon ' + status;
    iconEl.textContent = status === 'pass' ? '\u2713' : status === 'fail' ? '\u2717' : '\u2022';
  }
  if (labelEl) labelEl.textContent = label;
}

async function verifySetup() {
  var checklistEl = document.getElementById('setup-checklist');
  if (checklistEl) checklistEl.hidden = false;

  setCheckItem('check-key', 'pending', 'Checking SSH key...');
  setCheckItem('check-origins', 'pending', 'Checking target systems...');
  setCheckItem('check-server', 'pending', 'Checking HTTP server...');

  // Check SSH key
  var keyOk = false;
  try {
    var keyResult = await window.go.main.App.ValidateKey();
    keyOk = keyResult.valid === true;
    setCheckItem('check-key', keyOk ? 'pass' : 'fail',
      keyOk ? 'SSH key valid' : 'SSH key: ' + (keyResult.error || 'validation failed'));
  } catch (err) {
    setCheckItem('check-key', 'fail', 'SSH key: ' + err);
  }

  // Check target systems
  var collected = collectOriginScopes();
  var hasOrigins = Object.keys(collected.values).length > 0;
  setCheckItem('check-origins', hasOrigins ? 'pass' : 'fail',
    hasOrigins ? Object.keys(collected.values).length + ' target system(s) configured' : 'No target systems configured');

  // Check HTTP server
  var serverOk = false;
  try {
    var serverStatus = await window.go.main.App.GetServerStatus();
    serverOk = serverStatus.running === true;
    setCheckItem('check-server', serverOk ? 'pass' : 'fail',
      serverOk ? 'HTTP server running on ' + serverStatus.address : 'HTTP server not running');
  } catch (err) {
    setCheckItem('check-server', 'fail', 'HTTP server: ' + err);
  }

  var allOk = keyOk && hasOrigins && serverOk;
  setStatus(allOk ? 'All checks passed.' : 'Some checks failed.', allOk ? 'success' : 'error');

  setTimeout(function() {
    if (checklistEl) checklistEl.hidden = true;
  }, 4000);
}

function renderSettingsConfig(cfg) {
  renderToggle(cfg.enabled !== false);
  var keyPathInput = document.getElementById('key-path');
  if (keyPathInput) {
    keyPathInput.value = cfg.keyPath || '';
    renderKeyTypeTabs(detectKeyType(cfg.keyPath || ''));
  }
  var serverPortInput = document.getElementById('server-port');
  if (serverPortInput) {
    serverPortInput.value = cfg.serverPort || 7890;
  }
  renderOriginRows(cfg.originScopes || {});
}

async function saveSettingsConfig() {
  var keyPathInput = document.getElementById('key-path');
  var serverPortInput = document.getElementById('server-port');
  var keyPath = (keyPathInput ? keyPathInput.value : '').trim();
  var serverPort = serverPortInput ? parseInt(serverPortInput.value, 10) : 7890;
  if (isNaN(serverPort) || serverPort < 1024 || serverPort > 65535) serverPort = 7890;

  var parsed = collectOriginScopes();
  var errorsEl = document.getElementById('origin-row-errors');
  if (parsed.invalidLines.length > 0) {
    if (errorsEl) { errorsEl.textContent = parsed.invalidLines.join(' | '); errorsEl.hidden = false; }
    setStatus('Error: invalid target system(s)', 'error');
    return;
  }
  if (errorsEl) { errorsEl.textContent = ''; errorsEl.hidden = true; }

  var allowedOrigins = Object.keys(parsed.values);
  try {
    await window.go.main.App.SetConfig({
      enabled: document.getElementById('toggle-enabled').getAttribute('aria-pressed') === 'true',
      keyPath: keyPath,
      allowedOrigins: allowedOrigins,
      originScopes: parsed.values,
      serverPort: serverPort
    });
    var cfg = await window.go.main.App.GetConfig();
    renderSettingsConfig(cfg);
    setStatus('Config saved.', 'success');
  } catch (err) {
    setStatus('Error: ' + err, 'error');
  }
}

var settingsListenersBound = false;

function initSettings(cfg) {
  renderSettingsConfig(cfg);

  if (settingsListenersBound) return;
  settingsListenersBound = true;

  // Toggle
  var toggleBtn = document.getElementById('toggle-enabled');
  if (toggleBtn) {
    toggleBtn.addEventListener('click', async function() {
      var currentlyEnabled = toggleBtn.getAttribute('aria-pressed') === 'true';
      try {
        var newCfg = await window.go.main.App.SetEnabled(!currentlyEnabled);
        renderSettingsConfig(newCfg);
      } catch (err) {
        setStatus('Error: ' + err, 'error');
      }
    });
  }

  // Save
  var saveBtn = document.getElementById('save-config');
  if (saveBtn) {
    saveBtn.addEventListener('click', function() { saveSettingsConfig(); });
  }

  // Verify
  var verifyBtn = document.getElementById('verify-setup');
  if (verifyBtn) {
    verifyBtn.addEventListener('click', function() { verifySetup(); });
  }

  // Add origin row
  var addBtn = document.getElementById('add-origin-row');
  var container = document.getElementById('origin-rows');
  if (addBtn && container) {
    addBtn.addEventListener('click', function() {
      container.appendChild(createOriginRow());
    });
  }

  // Key type tabs
  var tabs = document.querySelectorAll('#view-settings [data-key-type]:not([data-onboarding])');
  var keyPathInput = document.getElementById('key-path');
  tabs.forEach(function(tab) {
    tab.addEventListener('click', function() {
      var type = tab.dataset.keyType;
      var nextPath = KEY_PATHS[type];
      if (!nextPath || !keyPathInput) return;
      keyPathInput.value = nextPath;
      renderKeyTypeTabs(type);
    });
  });
}
