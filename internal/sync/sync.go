package sync

import (
	"fmt"
	"strings"

	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/config"
	"github.com/hongkongkiwi/codex-claude-allow-deny-sync/internal/format"
)

type Policy struct {
	Allow []string
	Deny  []string
}

type ClientSnapshot struct {
	Client config.Client
	Policy Policy
}

type Options struct {
	DryRun bool
}

func Run(cfg config.Config, opts Options) (Policy, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = "union"
	}
	sortLists := true
	if cfg.Sort != nil {
		sortLists = *cfg.Sort
	}

	snapshots := make([]ClientSnapshot, 0, len(cfg.Clients))
	for _, client := range cfg.Clients {
		var err error
		var allow, deny []string
		switch strings.ToLower(client.Format) {
		case "json-object":
			path := client.AllowPath
			if path == "" {
				path = client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: json-object requires allow_path or deny_path", client.Name)
			}
			if client.AllowKey == "" && client.DenyKey == "" {
				return Policy{}, fmt.Errorf("client %s: json-object requires allow_key or deny_key", client.Name)
			}
			if client.AllowKey != "" {
				allow, err = format.ReadJSONKey(path, client.MissingOK, client.AllowKey)
				if err != nil {
					return Policy{}, fmt.Errorf("client %s allow: %w", client.Name, err)
				}
			}
			if client.DenyKey != "" {
				deny, err = format.ReadJSONKey(path, client.MissingOK, client.DenyKey)
				if err != nil {
					return Policy{}, fmt.Errorf("client %s deny: %w", client.Name, err)
				}
			}
		case "json-bool-map":
			path := client.AllowPath
			if path == "" {
				path = client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: json-bool-map requires allow_path or deny_path", client.Name)
			}
			if client.AllowKey == "" {
				return Policy{}, fmt.Errorf("client %s: json-bool-map requires allow_key", client.Name)
			}
			allow, deny, err = format.ReadJSONBoolMap(path, client.MissingOK, client.AllowKey)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s allow/deny: %w", client.Name, err)
			}
		case "codex-rules":
			path := client.AllowPath
			if path == "" {
				path = client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: codex-rules requires allow_path or deny_path", client.Name)
			}
			allow, deny, err = format.ReadCodexRules(path, client.MissingOK)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s rules: %w", client.Name, err)
			}
		default:
			fmtter, err := format.New(client.Format)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s: %w", client.Name, err)
			}
			allow, err = fmtter.Read(client.AllowPath, client.MissingOK)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s allow: %w", client.Name, err)
			}
			deny, err = fmtter.Read(client.DenyPath, client.MissingOK)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s deny: %w", client.Name, err)
			}
		}
		snapshots = append(snapshots, ClientSnapshot{
			Client: client,
			Policy: Policy{
				Allow: format.Normalize(allow, sortLists),
				Deny:  format.Normalize(deny, sortLists),
			},
		})
	}

	var merged Policy
	switch mode {
	case "union":
		for _, snap := range snapshots {
			merged.Allow = append(merged.Allow, snap.Policy.Allow...)
			merged.Deny = append(merged.Deny, snap.Policy.Deny...)
		}
		merged.Allow = format.Normalize(merged.Allow, sortLists)
		merged.Deny = format.Normalize(merged.Deny, sortLists)
	case "authoritative":
		if cfg.Source == "" {
			return Policy{}, fmt.Errorf("authoritative mode requires source")
		}
		found := false
		for _, snap := range snapshots {
			if snap.Client.Name == cfg.Source {
				merged = snap.Policy
				found = true
				break
			}
		}
		if !found {
			return Policy{}, fmt.Errorf("source %q not found", cfg.Source)
		}
	default:
		return Policy{}, fmt.Errorf("unknown mode %q", mode)
	}

	if opts.DryRun {
		return merged, nil
	}

	for _, snap := range snapshots {
		switch strings.ToLower(snap.Client.Format) {
		case "json-object":
			path := snap.Client.AllowPath
			if path == "" {
				path = snap.Client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: json-object requires allow_path or deny_path", snap.Client.Name)
			}
			if snap.Client.AllowKey != "" {
				if err := format.WriteJSONKey(path, snap.Client.AllowKey, merged.Allow); err != nil {
					return Policy{}, fmt.Errorf("client %s allow write: %w", snap.Client.Name, err)
				}
			}
			if snap.Client.DenyKey != "" {
				if err := format.WriteJSONKey(path, snap.Client.DenyKey, merged.Deny); err != nil {
					return Policy{}, fmt.Errorf("client %s deny write: %w", snap.Client.Name, err)
				}
			}
		case "json-bool-map":
			path := snap.Client.AllowPath
			if path == "" {
				path = snap.Client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: json-bool-map requires allow_path or deny_path", snap.Client.Name)
			}
			if snap.Client.AllowKey == "" {
				return Policy{}, fmt.Errorf("client %s: json-bool-map requires allow_key", snap.Client.Name)
			}
			if err := format.WriteJSONBoolMap(path, snap.Client.AllowKey, merged.Allow, merged.Deny); err != nil {
				return Policy{}, fmt.Errorf("client %s allow/deny write: %w", snap.Client.Name, err)
			}
		case "codex-rules":
			path := snap.Client.AllowPath
			if path == "" {
				path = snap.Client.DenyPath
			}
			if path == "" {
				return Policy{}, fmt.Errorf("client %s: codex-rules requires allow_path or deny_path", snap.Client.Name)
			}
			if err := format.WriteCodexRules(path, merged.Allow, merged.Deny); err != nil {
				return Policy{}, fmt.Errorf("client %s rules write: %w", snap.Client.Name, err)
			}
		default:
			fmtter, err := format.New(snap.Client.Format)
			if err != nil {
				return Policy{}, fmt.Errorf("client %s: %w", snap.Client.Name, err)
			}
			if err := fmtter.Write(snap.Client.AllowPath, merged.Allow); err != nil {
				return Policy{}, fmt.Errorf("client %s allow write: %w", snap.Client.Name, err)
			}
			if err := fmtter.Write(snap.Client.DenyPath, merged.Deny); err != nil {
				return Policy{}, fmt.Errorf("client %s deny write: %w", snap.Client.Name, err)
			}
		}
	}

	return merged, nil
}
