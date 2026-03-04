package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"isowebauth/internal/config"
)

func testConfigManager(t *testing.T) *config.Manager {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	m, err := config.NewManagerWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(func(c *config.Config) {
		c.AllowedOrigins = []string{"http://localhost:3000"}
		c.OriginScopes = map[string][]config.OriginScope{
			"http://localhost:3000": {{Namespace: "dev"}},
		}
	})
	return m
}

func TestHostValidationRejectsBadHost(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Host = "evil.attacker.com:7890"
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for bad Host, got %d", w.Code)
	}
}

func TestHostValidationAllowsLocalhost(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Host = "127.0.0.1:7890"
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for localhost Host, got %d", w.Code)
	}
}

func TestStatusEndpoint(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	s.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != true {
		t.Error("expected ok=true")
	}
	if body["version"] != Version {
		t.Errorf("expected version=%s, got %v", Version, body["version"])
	}
}

func TestCORSPreflightAllowed(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	req := httptest.NewRequest(http.MethodOptions, "/sign", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	s.handleSign(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	acao := w.Header().Get("Access-Control-Allow-Origin")
	if acao != "http://localhost:3000" {
		t.Errorf("expected ACAO=http://localhost:3000, got %q", acao)
	}
}

func TestCORSPreflightDenied(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	req := httptest.NewRequest(http.MethodOptions, "/sign", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()
	s.handleSign(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	acao := w.Header().Get("Access-Control-Allow-Origin")
	if acao != "" {
		t.Errorf("expected no ACAO header, got %q", acao)
	}
}

func TestOriginRejection(t *testing.T) {
	m := testConfigManager(t)
	s := New(m, nil)

	body := `{"challenge":"AbCdEfGhIjKlMnOp_1234","namespace":"dev"}`
	req := httptest.NewRequest(http.MethodPost, "/sign", strings.NewReader(body))
	req.Header.Set("Origin", "http://evil.com")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleSign(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ok"] != false {
		t.Error("expected ok=false")
	}
}

func TestSignWithConsentAllowed(t *testing.T) {
	m := testConfigManager(t)
	var gotConsent ConsentRequest
	var s *Server
	s = New(m, func(req ConsentRequest) {
		gotConsent = req
		// Auto-allow consent
		go s.RespondToConsent(req.ID, true)
	})

	body := `{"challenge":"AbCdEfGhIjKlMnOp_1234","namespace":"dev"}`
	req := httptest.NewRequest(http.MethodPost, "/sign", strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleSign(w, req)

	// The sign itself will likely fail (no real ssh-keygen setup) but
	// we verify the consent flow worked correctly
	if gotConsent.Origin != "http://localhost:3000" {
		t.Errorf("expected consent origin http://localhost:3000, got %q", gotConsent.Origin)
	}
	if gotConsent.Namespace != "dev" {
		t.Errorf("expected consent namespace dev, got %q", gotConsent.Namespace)
	}
}

func TestSignWithConsentDenied(t *testing.T) {
	m := testConfigManager(t)
	var s *Server
	s = New(m, func(req ConsentRequest) {
		go s.RespondToConsent(req.ID, false)
	})

	body := `{"challenge":"AbCdEfGhIjKlMnOp_1234","namespace":"dev"}`
	req := httptest.NewRequest(http.MethodPost, "/sign", strings.NewReader(body))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleSign(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "denied") {
		t.Errorf("expected denial message, got: %s", errMsg)
	}
}
