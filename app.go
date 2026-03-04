package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"isowebauth/internal/config"
	"isowebauth/internal/keyutil"
	"isowebauth/internal/server"
)

type App struct {
	ctx       context.Context
	configMgr *config.Manager
	server    *server.Server
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	mgr, err := config.NewManager()
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to load config: %s", err)
		return
	}
	a.configMgr = mgr

	a.server = server.New(mgr, func(req server.ConsentRequest) {
		runtime.EventsEmit(ctx, "sign-request", map[string]interface{}{
			"id":        req.ID,
			"origin":    req.Origin,
			"namespace": req.Namespace,
			"company":   req.Company,
			"challenge": req.Challenge,
		})
		runtime.WindowShow(ctx)
	})

	if err := a.server.Start(); err != nil {
		runtime.LogErrorf(ctx, "Failed to start HTTP server: %s", err)
	}

	// Show onboarding if no origins configured
	cfg := mgr.Get()
	if len(cfg.AllowedOrigins) == 0 {
		runtime.EventsEmit(ctx, "show-onboarding")
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		a.server.Stop()
	}
}

func (a *App) GetConfig() config.Config {
	if a.configMgr == nil {
		return config.DefaultConfig()
	}
	return a.configMgr.Get()
}

func (a *App) SetConfig(cfg config.Config) error {
	if a.configMgr == nil {
		return fmt.Errorf("config not initialized")
	}
	return a.configMgr.Update(func(c *config.Config) {
		c.Enabled = cfg.Enabled
		c.KeyPath = cfg.KeyPath
		c.AllowedOrigins = cfg.AllowedOrigins
		c.OriginScopes = cfg.OriginScopes
		if cfg.ServerPort > 0 {
			c.ServerPort = cfg.ServerPort
		}
	})
}

func (a *App) SetEnabled(enabled bool) config.Config {
	if a.configMgr != nil {
		a.configMgr.Update(func(c *config.Config) {
			c.Enabled = enabled
		})
	}
	return a.GetConfig()
}

func (a *App) ValidateKey() map[string]interface{} {
	if a.configMgr == nil {
		return map[string]interface{}{"valid": false, "error": "config not initialized"}
	}
	cfg := a.configMgr.Get()
	return a.ValidateKeyPath(cfg.KeyPath)
}

func (a *App) ValidateKeyPath(keyPath string) map[string]interface{} {
	err := keyutil.ValidateKeyFile(keyPath)
	if err != nil {
		return map[string]interface{}{"valid": false, "error": err.Error()}
	}
	return map[string]interface{}{"valid": true, "error": ""}
}

func (a *App) GetServerStatus() map[string]interface{} {
	if a.server == nil {
		return map[string]interface{}{
			"running": false,
			"address": "",
		}
	}
	addr := a.server.Addr()
	return map[string]interface{}{
		"running": addr != "",
		"address": addr,
	}
}

func (a *App) RespondToSignRequest(id string, allowed bool) {
	if a.server != nil {
		a.server.RespondToConsent(id, allowed)
	}
}
