package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"isowebauth/internal/config"
	"isowebauth/internal/keyutil"
	"isowebauth/internal/policy"
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

	a.server = server.New(mgr, nil)

	if err := a.server.Start(); err != nil {
		runtime.LogErrorf(ctx, "Failed to start HTTP server: %s", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		if err := a.server.Stop(); err != nil {
			runtime.LogErrorf(ctx, "Failed to stop HTTP server: %s", err)
		}
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
	if cfg.KeyPath != "" {
		if err := keyutil.ValidateKeyPath(cfg.KeyPath); err != nil {
			return err
		}
	}
	if cfg.ServerPort > 0 && (cfg.ServerPort < 1024 || cfg.ServerPort > 65535) {
		return fmt.Errorf("server port must be between 1024 and 65535")
	}
	if len(cfg.AllowedOrigins) > 100 {
		return fmt.Errorf("too many allowed origins (max 100)")
	}
	for _, origin := range cfg.AllowedOrigins {
		if policy.NormalizeOrigin(origin) == "" {
			return fmt.Errorf("invalid origin: %s", origin)
		}
	}
	for origin, scopes := range cfg.OriginScopes {
		if policy.NormalizeOrigin(origin) == "" {
			return fmt.Errorf("invalid origin in scopes: %s", origin)
		}
		for _, scope := range scopes {
			if policy.NormalizeNamespace(scope.Namespace) == "" {
				return fmt.Errorf("invalid namespace for origin %s: %s", origin, scope.Namespace)
			}
		}
	}
	return a.configMgr.Update(func(c *config.Config) {
		c.Enabled = cfg.Enabled
		c.KeyPath = cfg.KeyPath
		c.AllowedOrigins = cfg.AllowedOrigins
		c.OriginScopes = cfg.OriginScopes
		if cfg.ServerPort >= 1024 {
			c.ServerPort = cfg.ServerPort
		}
	})
}

func (a *App) SetEnabled(enabled bool) config.Config {
	if a.configMgr != nil {
		if err := a.configMgr.Update(func(c *config.Config) {
			c.Enabled = enabled
		}); err != nil {
			runtime.LogErrorf(a.ctx, "Failed to save config: %s", err)
		}
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

func (a *App) GetVersion() string {
	return server.Version
}
