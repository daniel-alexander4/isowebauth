// Consent dialog for sign requests

var currentConsentId = null;

function showConsentDialog(data) {
  currentConsentId = data.id;

  var overlay = document.getElementById('consent-overlay');
  document.getElementById('consent-origin').textContent = data.origin || '';
  document.getElementById('consent-namespace').textContent = data.namespace || '';

  var companyRow = document.getElementById('consent-company-row');
  var companyEl = document.getElementById('consent-company');
  if (data.company) {
    companyEl.textContent = data.company;
    companyRow.hidden = false;
  } else {
    companyRow.hidden = true;
  }

  var challengeText = data.challenge || '';
  if (challengeText.length > 64) {
    challengeText = challengeText.substring(0, 64) + '...';
  }
  document.getElementById('consent-challenge').textContent = challengeText;

  overlay.hidden = false;
}

function hideConsentDialog() {
  var overlay = document.getElementById('consent-overlay');
  overlay.hidden = true;
  currentConsentId = null;
}

document.addEventListener('DOMContentLoaded', function() {
  var allowBtn = document.getElementById('consent-allow');
  var denyBtn = document.getElementById('consent-deny');

  if (allowBtn) {
    allowBtn.addEventListener('click', function() {
      if (currentConsentId) {
        window.go.main.App.RespondToSignRequest(currentConsentId, true);
      }
      hideConsentDialog();
    });
  }

  if (denyBtn) {
    denyBtn.addEventListener('click', function() {
      if (currentConsentId) {
        window.go.main.App.RespondToSignRequest(currentConsentId, false);
      }
      hideConsentDialog();
    });
  }
});
