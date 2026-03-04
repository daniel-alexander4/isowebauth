// View routing
function showView(id) {
  document.querySelectorAll('.view').forEach(function(v) {
    v.classList.remove('active');
  });
  var el = document.getElementById(id);
  if (el) el.classList.add('active');
}

// Init
async function initApp() {
  try {
    var cfg = await window.go.main.App.GetConfig();
    if (!cfg.allowedOrigins || cfg.allowedOrigins.length === 0) {
      showView('view-onboarding');
      initOnboarding(cfg);
    } else {
      showView('view-settings');
      initSettings(cfg);
    }
  } catch (err) {
    console.error('Failed to init app:', err);
    showView('view-settings');
  }
}

// Listen for Wails events
if (window.runtime) {
  window.runtime.EventsOn('sign-request', function(data) {
    showConsentDialog(data);
  });
}

// About overlay
(function() {
  var overlay = document.getElementById('about-overlay');
  var hamburger = document.getElementById('hamburger-btn');
  var closeBtn = document.getElementById('about-close');

  hamburger.addEventListener('click', function() {
    overlay.hidden = false;
  });
  closeBtn.addEventListener('click', function() {
    overlay.hidden = true;
  });
  overlay.addEventListener('click', function(e) {
    if (e.target === overlay) overlay.hidden = true;
  });
})();

// Fetch version on init
async function fetchVersion() {
  try {
    var ver = await window.go.main.App.GetVersion();
    document.getElementById('about-version').textContent = 'v' + ver;
  } catch (e) {
    console.error('Failed to fetch version:', e);
  }
}

document.addEventListener('DOMContentLoaded', function() {
  initApp();
  fetchVersion();
});
