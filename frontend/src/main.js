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

document.addEventListener('DOMContentLoaded', initApp);
