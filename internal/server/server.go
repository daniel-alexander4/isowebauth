package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"time"

	"isowebauth/internal/config"
	"isowebauth/internal/policy"
	"isowebauth/internal/signer"
)

const maxRequestBody = 4096

const Version = "1.0.0"

type SignRequest struct {
	Challenge string `json:"challenge"`
	Namespace string `json:"namespace"`
	Company   string `json:"company,omitempty"`
}

type Server struct {
	configMgr  *config.Manager
	httpServer *http.Server
	listener   net.Listener
	boundPort  int
}

func New(configMgr *config.Manager, onConsent interface{}) *Server {
	cfg := configMgr.Get()
	s := &Server{
		configMgr: configMgr,
		boundPort: cfg.ServerPort,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/sign", s.handleSign)
	s.httpServer = &http.Server{Handler: s.validateHost(mux)}
	return s
}

// validateHost rejects requests whose Host header is not 127.0.0.1 or localhost
// on the bound port. This prevents DNS rebinding attacks.
func (s *Server) validateHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		allowed := fmt.Sprintf("127.0.0.1:%d", s.boundPort)
		allowedLocal := fmt.Sprintf("localhost:%d", s.boundPort)
		if host != allowed && host != allowedLocal {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	cfg := s.configMgr.Get()
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.ServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = ln
	s.boundPort = cfg.ServerPort
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


func isOriginAllowed(origin string, allowedOrigins []string) bool {
	normalized := policy.NormalizeOrigin(origin)
	if normalized == "" {
		return false
	}
	for _, candidate := range policy.EquivalentOrigins(normalized) {
		for _, allowed := range allowedOrigins {
			if candidate == allowed {
				return true
			}
		}
	}
	return false
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	cfg := s.configMgr.Get()

	if origin != "" && isOriginAllowed(origin, cfg.AllowedOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", policy.NormalizeOrigin(origin))
		w.Header().Set("Vary", "Origin")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
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
	normalizedOrigin := policy.NormalizeOrigin(origin)

	if r.Method == http.MethodOptions {
		s.handleCORSPreflight(w, r, normalizedOrigin)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.configMgr.Get()

	if !isOriginAllowed(origin, cfg.AllowedOrigins) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "Origin not allowed",
		})
		return
	}

	// Set CORS headers using normalized origin
	w.Header().Set("Access-Control-Allow-Origin", normalizedOrigin)
	w.Header().Set("Vary", "Origin")

	// Validate Content-Type
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "Content-Type must be application/json",
		})
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

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
		log.Printf("policy denied sign request: %s", policyResult.Error)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "Request denied by policy",
		})
		return
	}

	// Sign (signer.Sign validates the key internally)
	signature, err := signer.Sign(
		policyResult.Challenge,
		policyResult.ExpectedNamespace,
		policyResult.Origin,
		cfg.KeyPath,
		signer.DefaultTimeout,
	)
	if err != nil {
		log.Printf("signing error: %s", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "Signing failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"signature": signature,
	})
}

func (s *Server) handleCORSPreflight(w http.ResponseWriter, r *http.Request, normalizedOrigin string) {
	cfg := s.configMgr.Get()

	if normalizedOrigin == "" || !isOriginAllowed(normalizedOrigin, cfg.AllowedOrigins) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", normalizedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.Header().Set("Vary", "Origin")
	w.WriteHeader(http.StatusNoContent)
}
