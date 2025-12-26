package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "syncd.yaml")
	input := []byte("clients:\n  - name: test\n    format: newline\n    allow_path: ~/allow.txt\n    deny_path: ~/deny.txt\n")
	if err := os.WriteFile(cfgPath, input, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(cfg.Clients))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if cfg.Clients[0].AllowPath != filepath.Join(home, "allow.txt") {
		t.Fatalf("allow_path not expanded: %s", cfg.Clients[0].AllowPath)
	}
	if cfg.Clients[0].DenyPath != filepath.Join(home, "deny.txt") {
		t.Fatalf("deny_path not expanded: %s", cfg.Clients[0].DenyPath)
	}
}
