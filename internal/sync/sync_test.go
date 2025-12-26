package sync

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/config"
	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/format"
)

func TestRunUnionDryRun(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")

	if err := os.WriteFile(pathA, []byte(`{"permissions":{"allow":["A"],"deny":["X"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(`{"permissions":{"allow":["B"],"deny":["Y"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Mode: "union",
		Sort: boolPtr(true),
		Clients: []config.Client{
			{
				Name:      "a",
				Format:    "json-object",
				AllowPath: pathA,
				DenyPath:  pathA,
				AllowKey:  "permissions.allow",
				DenyKey:   "permissions.deny",
			},
			{
				Name:      "b",
				Format:    "json-object",
				AllowPath: pathB,
				DenyPath:  pathB,
				AllowKey:  "permissions.allow",
				DenyKey:   "permissions.deny",
			},
		},
	}

	policy, err := Run(cfg, Options{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !reflect.DeepEqual(policy.Allow, []string{"A", "B"}) {
		t.Fatalf("allow mismatch: %v", policy.Allow)
	}
	if !reflect.DeepEqual(policy.Deny, []string{"X", "Y"}) {
		t.Fatalf("deny mismatch: %v", policy.Deny)
	}
}

func TestRunAuthoritative(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")

	if err := os.WriteFile(pathA, []byte(`{"permissions":{"allow":["A"],"deny":["X"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(`{"permissions":{"allow":["B"],"deny":["Y"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Mode:   "authoritative",
		Source: "a",
		Sort:   boolPtr(true),
		Clients: []config.Client{
			{
				Name:      "a",
				Format:    "json-object",
				AllowPath: pathA,
				DenyPath:  pathA,
				AllowKey:  "permissions.allow",
				DenyKey:   "permissions.deny",
			},
			{
				Name:      "b",
				Format:    "json-object",
				AllowPath: pathB,
				DenyPath:  pathB,
				AllowKey:  "permissions.allow",
				DenyKey:   "permissions.deny",
			},
		},
	}

	policy, err := Run(cfg, Options{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !reflect.DeepEqual(policy.Allow, []string{"A"}) {
		t.Fatalf("allow mismatch: %v", policy.Allow)
	}
	if !reflect.DeepEqual(policy.Deny, []string{"X"}) {
		t.Fatalf("deny mismatch: %v", policy.Deny)
	}
}

func TestRunWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := os.WriteFile(path, []byte(`{"permissions":{"allow":["A"],"deny":["X"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Mode: "union",
		Sort: boolPtr(true),
		Clients: []config.Client{
			{
				Name:      "a",
				Format:    "json-object",
				AllowPath: path,
				DenyPath:  path,
				AllowKey:  "permissions.allow",
				DenyKey:   "permissions.deny",
			},
		},
	}

	if _, err := Run(cfg, Options{DryRun: false}); err != nil {
		t.Fatalf("run: %v", err)
	}

	allow, err := format.ReadJSONKey(path, false, "permissions.allow")
	if err != nil {
		t.Fatalf("read allow: %v", err)
	}
	deny, err := format.ReadJSONKey(path, false, "permissions.deny")
	if err != nil {
		t.Fatalf("read deny: %v", err)
	}
	if !reflect.DeepEqual(allow, []string{"A"}) {
		t.Fatalf("allow mismatch: %v", allow)
	}
	if !reflect.DeepEqual(deny, []string{"X"}) {
		t.Fatalf("deny mismatch: %v", deny)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
