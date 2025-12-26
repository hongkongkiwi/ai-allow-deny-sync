package format

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestJSONKeyReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	input := []byte(`{"permissions":{"allow":["A","B"],"deny":["X"]}}`)
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatal(err)
	}

	allow, err := ReadJSONKey(path, false, "permissions.allow")
	if err != nil {
		t.Fatalf("read allow: %v", err)
	}
	deny, err := ReadJSONKey(path, false, "permissions.deny")
	if err != nil {
		t.Fatalf("read deny: %v", err)
	}

	if !reflect.DeepEqual(allow, []string{"A", "B"}) {
		t.Fatalf("allow mismatch: %v", allow)
	}
	if !reflect.DeepEqual(deny, []string{"X"}) {
		t.Fatalf("deny mismatch: %v", deny)
	}

	if err := WriteJSONKey(path, "permissions.allow", []string{"C"}); err != nil {
		t.Fatalf("write allow: %v", err)
	}
	if err := WriteJSONKey(path, "permissions.deny", []string{"Y", "Z"}); err != nil {
		t.Fatalf("write deny: %v", err)
	}

	allow, err = ReadJSONKey(path, false, "permissions.allow")
	if err != nil {
		t.Fatalf("read allow after write: %v", err)
	}
	deny, err = ReadJSONKey(path, false, "permissions.deny")
	if err != nil {
		t.Fatalf("read deny after write: %v", err)
	}

	if !reflect.DeepEqual(allow, []string{"C"}) {
		t.Fatalf("allow mismatch after write: %v", allow)
	}
	if !reflect.DeepEqual(deny, []string{"Y", "Z"}) {
		t.Fatalf("deny mismatch after write: %v", deny)
	}
}

func TestJSONBoolMapReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	input := []byte(`{"chat":{"tools":{"terminal":{"autoApprove":{"mkdir":true,"del":false}}}}}`)
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatal(err)
	}

	allow, deny, err := ReadJSONBoolMap(path, false, "chat.tools.terminal.autoApprove")
	if err != nil {
		t.Fatalf("read map: %v", err)
	}
	if !reflect.DeepEqual(Normalize(allow, true), []string{"mkdir"}) {
		t.Fatalf("allow mismatch: %v", allow)
	}
	if !reflect.DeepEqual(Normalize(deny, true), []string{"del"}) {
		t.Fatalf("deny mismatch: %v", deny)
	}

	if err := WriteJSONBoolMap(path, "chat.tools.terminal.autoApprove", []string{"git status"}, []string{"rm"}); err != nil {
		t.Fatalf("write map: %v", err)
	}
	allow, deny, err = ReadJSONBoolMap(path, false, "chat.tools.terminal.autoApprove")
	if err != nil {
		t.Fatalf("read map after write: %v", err)
	}
	if !reflect.DeepEqual(Normalize(allow, true), []string{"git status"}) {
		t.Fatalf("allow mismatch after write: %v", allow)
	}
	if !reflect.DeepEqual(Normalize(deny, true), []string{"rm"}) {
		t.Fatalf("deny mismatch after write: %v", deny)
	}
}

func TestCodexRulesReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default.rules")
	input := []byte(
		codexManagedMarker + "\n" +
			"prefix_rule(pattern=[\"git\"], decision=\"allow\")\n" +
			codexManagedMarker + "\n" +
			"prefix_rule(pattern=[\"rm\"], decision=\"forbidden\")\n" +
			"prefix_rule(pattern=[\"foo\"], decision=\"prompt\")\n",
	)
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatal(err)
	}

	allow, deny, err := ReadCodexRules(path, false)
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	if !reflect.DeepEqual(allow, []string{"git"}) {
		t.Fatalf("allow mismatch: %v", allow)
	}
	if !reflect.DeepEqual(deny, []string{"rm"}) {
		t.Fatalf("deny mismatch: %v", deny)
	}

	if err := WriteCodexRules(path, []string{"ls", "git status"}, []string{"rm -rf"}); err != nil {
		t.Fatalf("write rules: %v", err)
	}
	allow, deny, err = ReadCodexRules(path, false)
	if err != nil {
		t.Fatalf("read rules after write: %v", err)
	}

	if !reflect.DeepEqual(Normalize(allow, true), []string{"git status", "ls"}) {
		t.Fatalf("allow mismatch after write: %v", allow)
	}
	if !reflect.DeepEqual(deny, []string{"rm -rf"}) {
		t.Fatalf("deny mismatch after write: %v", deny)
	}
}

func TestCodexRuleSplitQuoted(t *testing.T) {
	got := splitCommandPattern(`git commit -m "hello world"`)
	want := []string{"git", "commit", "-m", "hello world"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("split mismatch: %v", got)
	}
}
