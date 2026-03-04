// Onboarding view — port of onboarding/index.ts (2 functional steps + ready)

function activateStep(stepNum) {
  for (var i = 1; i <= 3; i++) {
    var el = document.getElementById('step-' + i);
    if (!el) continue;
    el.classList.remove('active');
    if (i === stepNum) el.classList.add('active');
  }
}

function markStepDone(stepNum, statusText) {
  var el = document.getElementById('step-' + stepNum);
  var statusEl = document.getElementById('step-' + stepNum + '-status');
  if (el) el.classList.add('done');
  if (statusEl) statusEl.textContent = statusText;
}

function showResult(containerId, pass, text) {
  var el = document.getElementById(containerId);
  if (!el) return;
  el.hidden = false;
  el.className = 'step-result ' + (pass ? 'result-pass' : 'result-fail');
  el.textContent = text;
}

function initOnboarding(cfg) {
  // Step headers toggle
  for (var i = 1; i <= 3; i++) {
    (function(stepNum) {
      var header = document.querySelector('#step-' + stepNum + ' .step-header');
      if (header) {
        header.addEventListener('click', function() { activateStep(stepNum); });
      }
    })(i);
  }

  // Key type tabs (onboarding)
  var keyPathInput = document.getElementById('onboarding-key-path');
  var keyTabs = document.querySelectorAll('[data-key-type][data-onboarding]');
  keyTabs.forEach(function(tab) {
    tab.addEventListener('click', function() {
      var type = tab.dataset.keyType;
      if (!KEY_PATHS[type] || !keyPathInput) return;
      keyPathInput.value = KEY_PATHS[type];
      keyTabs.forEach(function(t) {
        t.setAttribute('aria-pressed', t === tab ? 'true' : 'false');
      });
    });
  });

  // Step 1: Validate key
  var validateBtn = document.getElementById('validate-key-btn');
  if (validateBtn) {
    validateBtn.addEventListener('click', async function() {
      validateBtn.textContent = 'Validating...';
      validateBtn.disabled = true;
      var kp = (keyPathInput ? keyPathInput.value.trim() : '') || '~/.ssh/id_ed25519';
      try {
        var result = await window.go.main.App.ValidateKeyPath(kp);
        var keyOk = result.valid === true;
        showResult('key-result', keyOk, keyOk ? 'Key is valid!' : 'Key error: ' + (result.error || 'validation failed'));
        if (keyOk) {
          markStepDone(1, 'Valid');
          activateStep(2);
        }
      } catch (err) {
        showResult('key-result', false, 'Error: ' + err);
      }
      validateBtn.textContent = 'Validate Key';
      validateBtn.disabled = false;
    });
  }

  // Step 2: Origin rows
  var originRows = document.getElementById('onboarding-origin-rows');
  if (originRows) {
    originRows.appendChild(createOriginRow());
  }
  var addRowBtn = document.getElementById('onboarding-add-origin-row');
  if (addRowBtn && originRows) {
    addRowBtn.addEventListener('click', function() {
      originRows.appendChild(createOriginRow());
    });
  }

  // Save & Finish
  var saveFinishBtn = document.getElementById('save-and-finish');
  if (saveFinishBtn) {
    saveFinishBtn.addEventListener('click', async function() {
      var parsed = collectOriginScopes('onboarding-origin-rows');
      var errorsEl = document.getElementById('onboarding-origin-row-errors');
      if (parsed.invalidLines.length > 0) {
        if (errorsEl) { errorsEl.textContent = parsed.invalidLines.join(' | '); errorsEl.hidden = false; }
        return;
      }
      if (errorsEl) { errorsEl.textContent = ''; errorsEl.hidden = true; }

      var kp = (keyPathInput ? keyPathInput.value.trim() : '') || '~/.ssh/id_ed25519';
      var allowedOrigins = Object.keys(parsed.values);
      var statusEl = document.getElementById('onboarding-status');

      try {
        var currentCfg = await window.go.main.App.GetConfig();
        await window.go.main.App.SetConfig({
          enabled: true,
          keyPath: kp,
          allowedOrigins: allowedOrigins,
          originScopes: parsed.values,
          serverPort: currentCfg.serverPort || 7890
        });

        if (statusEl) {
          statusEl.hidden = false;
          statusEl.textContent = 'Setup complete!';
          statusEl.className = 'status status-success';
        }
        if (allowedOrigins.length > 0) markStepDone(2, 'Configured');
        markStepDone(3, 'Ready');

        // Switch to settings view
        setTimeout(async function() {
          var newCfg = await window.go.main.App.GetConfig();
          showView('view-settings');
          initSettings(newCfg);
        }, 1000);
      } catch (err) {
        if (statusEl) {
          statusEl.hidden = false;
          statusEl.textContent = 'Error: ' + err;
          statusEl.className = 'status status-error';
        }
      }
    });
  }
}
