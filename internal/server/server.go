package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"isowebauth/internal/config"
	"isowebauth/internal/keyutil"
	"isowebauth/internal/policy"
	"isowebauth/internal/signer"
)

const (
	Version        = "1.0.0"
	ConsentTimeout = 30 * time.Second
)

type SignRequest struct {
	Challenge string `json:"challenge"`
	Namespace string `json:"namespace"`
	Company   string `json:"company,omitempty"`
}

type ConsentRequest struct {
	ID        string `json:"id"`
	Origin    string `json:"origin"`
	Namespace string `json:"namespace"`
	Company   string `json:"company,omitempty"`
	Challenge string `json:"challenge"`
}

type ConsentCallback func(req ConsentRequest)

type Server struct {
	configMgr       *config.Manager
	httpServer      *http.Server
	listener        net.Listener
	consentCallback ConsentCallback

	mu             sync.Mutex
	pendingConsent map[string]chan bool
	nextID         int
}

func New(configMgr *config.Manager, onConsent ConsentCallback) *Server {
	s := &Server{
		configMgr:       configMgr,
		consentCallback: onConsent,
		pendingConsent:  make(map[string]chan bool),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/sign", s.handleSign)
	s.httpServer = &http.Server{Handler: mux}
	return s
}

func (s *Server) Start() error {
	cfg := s.configMgr.Get()
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.ServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = ln
	go s.httpServer.Serve(ln)
	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

func (s *Server) RespondToConsent(id string, allowed bool) {
	s.mu.Lock()
	ch, ok := s.pendingConsent[id]
	if ok {
		delete(s.pendingConsent, id)
	}
	s.mu.Unlock()
	if ok {
		ch <- allowed
	}
}

func (s *Server) newConsentID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("consent-%d", s.nextID)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"version": Version,
	})
}

func (s *Server) handleSign(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	if r.Method == http.MethodOptions {
		s.handleCORSPreflight(w, r, origin)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.configMgr.Get()

	// Check origin against allowlist for CORS
	normalizedOrigin := policy.NormalizeOrigin(origin)
	originAllowed := false
	if normalizedOrigin != "" {
		for _, candidate := range policy.EquivalentOrigins(normalizedOrigin) {
			for _, allowed := range cfg.AllowedOrigins {
				if candidate == allowed {
					originAllowed = true
					break
				}
			}
			if originAllowed {
				break
			}
		}
	}

	if !originAllowed {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("Origin not allowed: %s", origin),
		})
		return
	}

	// Set CORS headers for allowed origin
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "Invalid request body",
		})
		return
	}

	// Run policy evaluation
	policyResult := policy.EvaluateSignPolicy(policy.SignPolicyInput{
		Enabled:        cfg.Enabled,
		Challenge:      req.Challenge,
		Namespace:      req.Namespace,
		Company:        req.Company,
		Origin:         origin,
		AllowedOrigins: cfg.AllowedOrigins,
		OriginScopes:   cfg.OriginScopes,
	})

	if !policyResult.OK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": policyResult.Error,
		})
		return
	}

	// Request user consent if callback is configured
	if s.consentCallback != nil {
		consentID := s.newConsentID()
		ch := make(chan bool, 1)
		s.mu.Lock()
		s.pendingConsent[consentID] = ch
		s.mu.Unlock()

		s.consentCallback(ConsentRequest{
			ID:        consentID,
			Origin:    origin,
			Namespace: req.Namespace,
			Company:   req.Company,
			Challenge: req.Challenge,
		})

		select {
		case allowed := <-ch:
			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok":    false,
					"error": "User denied consent",
				})
				return
			}
		case <-time.After(ConsentTimeout):
			s.mu.Lock()
			delete(s.pendingConsent, consentID)
			s.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusGatewayTimeout)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    false,
				"error": "Consent timeout",
			})
			return
		}
	}

	// Validate key
	if err := keyutil.ValidateKeyFile(cfg.KeyPath); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("Key validation failed: %s", err),
		})
		return
	}

	// Sign
	signature, err := signer.Sign(
		policyResult.Challenge,
		policyResult.ExpectedNamespace,
		cfg.KeyPath,
		signer.DefaultTimeout,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("Signing failed: %s", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"signature": signature,
	})
}

func (s *Server) handleCORSPreflight(w http.ResponseWriter, r *http.Request, origin string) {
	cfg := s.configMgr.Get()
	normalizedOrigin := policy.NormalizeOrigin(origin)

	allowed := false
	if normalizedOrigin != "" {
		for _, candidate := range policy.EquivalentOrigins(normalizedOrigin) {
			for _, ao := range cfg.AllowedOrigins {
				if candidate == ao {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}
	}

	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.WriteHeader(http.StatusNoContent)
}
