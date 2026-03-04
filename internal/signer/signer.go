package signer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"isowebauth/internal/keyutil"
	"isowebauth/internal/policy"
)

const DefaultTimeout = 10 * time.Second

// sshKeygenPath is the path to the ssh-keygen binary. Tests can override this.
var sshKeygenPath string

// validateKeyPathFunc validates the key path. Tests can override this.
var validateKeyPathFunc = keyutil.ValidateKeyPath

func init() {
	if p, err := exec.LookPath("ssh-keygen"); err == nil {
		sshKeygenPath = p
	} else {
		sshKeygenPath = "ssh-keygen"
	}
}

func Sign(challenge, namespace, origin, keyPath string, timeout time.Duration) (string, error) {
	challenge = strings.TrimSpace(challenge)
	namespace = strings.TrimSpace(namespace)

	if challenge == "" {
		return "", fmt.Errorf("challenge is empty")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace is empty")
	}
	if !policy.ChallengeRegex.MatchString(challenge) {
		return "", fmt.Errorf("invalid challenge format")
	}
	if !policy.NamespaceRegex.MatchString(namespace) {
		return "", fmt.Errorf("invalid namespace format")
	}
	if origin == "" {
		return "", fmt.Errorf("origin is empty")
	}

	if err := validateKeyPathFunc(keyPath); err != nil {
		return "", err
	}
	resolvedKey, err := keyutil.ResolveKeyPath(keyPath)
	if err != nil {
		return "", err
	}
	if err := keyutil.ValidateKeyFile(resolvedKey); err != nil {
		return "", err
	}

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	keygen := sshKeygenPath

	// Use a private base dir under user's home to avoid world-readable /tmp
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	privateTmpBase := filepath.Join(homeDir, ".cache", "isowebauth", "tmp")
	if err := os.MkdirAll(privateTmpBase, 0700); err != nil {
		return "", fmt.Errorf("failed to create private temp base: %w", err)
	}
	tmpDir, err := os.MkdirTemp(privateTmpBase, "sign-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	challengeFile := filepath.Join(tmpDir, "challenge")
	sigFile := filepath.Join(tmpDir, "challenge.sig")

	if err := os.WriteFile(challengeFile, []byte(challenge), 0600); err != nil {
		return "", fmt.Errorf("failed to write challenge file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, keygen,
		"-Y", "sign",
		"-f", resolvedKey,
		"-n", namespace,
		challengeFile,
	)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("ssh-keygen sign timed out after %s", timeout)
	}
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("ssh-keygen sign failed (code %d): %s", exitErr.ExitCode(), detail)
		}
		return "", fmt.Errorf("ssh-keygen sign failed: %s", detail)
	}

	sigData, err := os.ReadFile(sigFile)
	if err != nil {
		return "", fmt.Errorf("signature file was not created by ssh-keygen")
	}

	signature := strings.TrimSpace(string(sigData))
	if signature == "" {
		return "", fmt.Errorf("signature file is empty")
	}

	return signature, nil
}
