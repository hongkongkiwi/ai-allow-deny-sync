package sync

import (
	"fmt"
	"os"
	"strings"

	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/config"
	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/format"
)

func Validate(cfg config.Config) error {
	for _, client := range cfg.Clients {
		if err := validateClient(client); err != nil {
			return err
		}
	}
	return nil
}

func validateClient(client config.Client) error {
	switch strings.ToLower(client.Format) {
	case "json-object":
		path := primaryPath(client)
		if path == "" {
			return fmt.Errorf("client %s: json-object requires allow_path or deny_path", client.Name)
		}
		if client.AllowKey == "" && client.DenyKey == "" {
			return fmt.Errorf("client %s: json-object requires allow_key or deny_key", client.Name)
		}
		return validatePathExists(client, path)
	case "json-bool-map":
		path := primaryPath(client)
		if path == "" {
			return fmt.Errorf("client %s: json-bool-map requires allow_path or deny_path", client.Name)
		}
		if client.AllowKey == "" {
			return fmt.Errorf("client %s: json-bool-map requires allow_key", client.Name)
		}
		return validatePathExists(client, path)
	case "codex-rules":
		path := primaryPath(client)
		if path == "" {
			return fmt.Errorf("client %s: codex-rules requires allow_path or deny_path", client.Name)
		}
		return validatePathExists(client, path)
	default:
		if _, err := format.New(client.Format); err != nil {
			return fmt.Errorf("client %s: %w", client.Name, err)
		}
		if client.AllowPath == "" || client.DenyPath == "" {
			return fmt.Errorf("client %s: allow_path and deny_path required", client.Name)
		}
		if err := validatePathExists(client, client.AllowPath); err != nil {
			return err
		}
		if err := validatePathExists(client, client.DenyPath); err != nil {
			return err
		}
	}
	return nil
}

func primaryPath(client config.Client) string {
	if client.AllowPath != "" {
		return client.AllowPath
	}
	return client.DenyPath
}

func validatePathExists(client config.Client, path string) error {
	if client.MissingOK {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("client %s: path not found: %s", client.Name, path)
		}
		return fmt.Errorf("client %s: path error: %v", client.Name, err)
	}
	return nil
}
