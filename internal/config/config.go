package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	DefaultKeyPath    = "~/.ssh/id_ed25519"
	DefaultServerPort = 7890
	configDirName     = "sshkey-web-auth"
	configFileName    = "config.json"
)

type OriginScope struct {
	Namespace string `json:"namespace"`
	Company   string `json:"company,omitempty"`
}

type Config struct {
	Enabled      bool                    `json:"enabled"`
	KeyPath      string                  `json:"keyPath"`
	AllowedOrigins []string              `json:"allowedOrigins"`
	OriginScopes map[string][]OriginScope `json:"originScopes"`
	ServerPort   int                     `json:"serverPort"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		KeyPath:        DefaultKeyPath,
		AllowedOrigins: []string{},
		OriginScopes:   map[string][]OriginScope{},
		ServerPort:     DefaultServerPort,
	}
}

type Manager struct {
	mu       sync.RWMutex
	path     string
	config   Config
}

func NewManager() (*Manager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(configDir, configDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	m := &Manager{
		path:   filepath.Join(dir, configFileName),
		config: DefaultConfig(),
	}
	if err := m.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return m, nil
}

func NewManagerWithPath(path string) (*Manager, error) {
	m := &Manager{
		path:   path,
		config: DefaultConfig(),
	}
	if err := m.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return m, nil
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify config file permissions on Unix
	if runtime.GOOS != "windows" {
		info, err := os.Stat(m.path)
		if err == nil {
			mode := info.Mode().Perm()
			if mode&0o077 != 0 {
				return fmt.Errorf("config file permissions are too open (%#o), expected 0600 or stricter: %s", mode, m.path)
			}
		}
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if cfg.OriginScopes == nil {
		cfg.OriginScopes = map[string][]OriginScope{}
	}
	if cfg.AllowedOrigins == nil {
		cfg.AllowedOrigins = []string{}
	}
	if cfg.KeyPath == "" {
		cfg.KeyPath = DefaultKeyPath
	}
	if cfg.ServerPort == 0 {
		cfg.ServerPort = DefaultServerPort
	}
	m.config = cfg
	return nil
}

func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

// saveLocked writes the current config to disk. Caller must hold mu.
func (m *Manager) saveLocked() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.path)
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if err := os.Chmod(tmpName, 0600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, m.path)
}

func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg := m.config
	origins := make([]string, len(cfg.AllowedOrigins))
	copy(origins, cfg.AllowedOrigins)
	cfg.AllowedOrigins = origins
	scopes := make(map[string][]OriginScope, len(cfg.OriginScopes))
	for k, v := range cfg.OriginScopes {
		s := make([]OriginScope, len(v))
		copy(s, v)
		scopes[k] = s
	}
	cfg.OriginScopes = scopes
	return cfg
}

func (m *Manager) Update(fn func(*Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	fn(&m.config)
	return m.saveLocked()
}
