package keyutil

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"isowebauth/internal/config"
)

func ResolveKeyPath(override string) (string, error) {
	p := strings.TrimSpace(override)
	if p == "" {
		p = config.DefaultKeyPath
	}
	if strings.HasPrefix(p, "~/") {
		u, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		p = filepath.Join(u.HomeDir, p[2:])
	}
	return p, nil
}

// ValidateKeyPath checks that the resolved key path is within ~/.ssh/.
func ValidateKeyPath(path string) error {
	resolved, err := ResolveKeyPath(path)
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot determine home directory for path validation: %w", err)
	}
	sshDir := filepath.Join(u.HomeDir, ".ssh")
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}
	// Resolve symlinks to get canonical path
	absResolved, err = filepath.EvalSymlinks(absResolved)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot resolve symlinks: %w", err)
	}
	absSshDir, err := filepath.Abs(sshDir)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}
	if !strings.HasPrefix(absResolved, absSshDir+string(filepath.Separator)) {
		return fmt.Errorf("key path must be within ~/.ssh/: %s", resolved)
	}
	return nil
}

func ValidateKeyFile(path string) error {
	resolved, err := ResolveKeyPath(path)
	if err != nil {
		return err
	}

	info, err := os.Lstat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key file not found: %s", resolved)
		}
		return fmt.Errorf("cannot stat key file: %s: %w", resolved, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("key path is not a regular file: %s", resolved)
	}

	if runtime.GOOS != "windows" {
		// Check ownership
		uid := os.Getuid()
		if uid >= 0 {
			stat := info.Sys()
			if stat != nil {
				if sysUID := fileUID(stat); sysUID >= 0 && sysUID != uid {
					return fmt.Errorf("key file is not owned by current user: %s", resolved)
				}
			}
		}

		// Check permissions
		mode := info.Mode().Perm()
		if mode&0o077 != 0 {
			return fmt.Errorf("key file permissions are too open (%#o), expected 0o600 or stricter", mode)
		}
	}

	return nil
}
