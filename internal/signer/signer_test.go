package signer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func skipKeyPathValidation(t *testing.T) {
	t.Helper()
	old := validateKeyPathFunc
	validateKeyPathFunc = func(string) error { return nil }
	t.Cleanup(func() { validateKeyPathFunc = old })
}

func createKeyFile(t *testing.T, dir string) string {
	t.Helper()
	keyFile := filepath.Join(dir, "id_test")
	if err := os.WriteFile(keyFile, []byte("dummy-private-key"), 0600); err != nil {
		t.Fatal(err)
	}
	return keyFile
}

func createFakeKeygen(t *testing.T, dir, content string) string {
	t.Helper()
	script := filepath.Join(dir, "fake-keygen.sh")
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestSignRejectsInvalidChallengeFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows")
	}
	dir := t.TempDir()
	keyFile := createKeyFile(t, dir)

	_, err := Sign("bad!!", "myapp", "http://localhost:3000", keyFile, DefaultTimeout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid challenge format") {
		t.Errorf("expected 'invalid challenge format', got: %s", err)
	}
}

func TestSignRejectsInvalidNamespaceFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows")
	}
	dir := t.TempDir()
	keyFile := createKeyFile(t, dir)

	_, err := Sign("AbCdEfGhIjKlMnOp_1234", "bad namespace", "http://localhost:3000", keyFile, DefaultTimeout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid namespace format") {
		t.Errorf("expected 'invalid namespace format', got: %s", err)
	}
}

func TestSignReturnsErrorWhenKeygenFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows")
	}
	skipKeyPathValidation(t)
	dir := t.TempDir()
	keyFile := createKeyFile(t, dir)
	fakeKeygen := createFakeKeygen(t, dir,
		"#!/usr/bin/env bash\necho 'simulated sign failure' >&2\nexit 17\n",
	)

	old := sshKeygenPath
	sshKeygenPath = fakeKeygen
	defer func() { sshKeygenPath = old }()

	_, err := Sign("AbCdEfGhIjKlMnOp_1234", "myapp", "http://localhost:3000", keyFile, DefaultTimeout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ssh-keygen sign failed") {
		t.Errorf("expected 'ssh-keygen sign failed', got: %s", err)
	}
	if !strings.Contains(err.Error(), "code 17") {
		t.Errorf("expected 'code 17' in error, got: %s", err)
	}
}

func TestSignReturnsErrorWhenKeygenTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows")
	}
	skipKeyPathValidation(t)
	dir := t.TempDir()
	keyFile := createKeyFile(t, dir)
	fakeKeygen := createFakeKeygen(t, dir,
		"#!/usr/bin/env bash\nsleep 2\nexit 0\n",
	)

	old := sshKeygenPath
	sshKeygenPath = fakeKeygen
	defer func() { sshKeygenPath = old }()

	_, err := Sign("AbCdEfGhIjKlMnOp_1234", "myapp", "http://localhost:3000", keyFile, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got: %s", err)
	}
}

func TestSignReturnsErrorForMissingKeyFile(t *testing.T) {
	skipKeyPathValidation(t)
	_, err := Sign("AbCdEfGhIjKlMnOp_1234", "myapp", "http://localhost:3000", "/nonexistent/key", DefaultTimeout)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err)
	}
}
