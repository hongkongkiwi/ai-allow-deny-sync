package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode    string   `yaml:"mode"`
	Source  string   `yaml:"source"`
	Sort    *bool    `yaml:"sort"`
	Clients []Client `yaml:"clients"`
}

type Client struct {
	Name      string `yaml:"name"`
	AllowPath string `yaml:"allow_path"`
	DenyPath  string `yaml:"deny_path"`
	Format    string `yaml:"format"`
	AllowKey  string `yaml:"allow_key"`
	DenyKey   string `yaml:"deny_key"`
	MissingOK bool   `yaml:"missing_ok"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Clients) == 0 {
		return Config{}, fmt.Errorf("config has no clients")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolve home dir: %w", err)
	}
	for i := range cfg.Clients {
		cfg.Clients[i].AllowPath = expandHome(cfg.Clients[i].AllowPath, home)
		cfg.Clients[i].DenyPath = expandHome(cfg.Clients[i].DenyPath, home)
	}
	return cfg, nil
}

func expandHome(path string, home string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return strings.Replace(path, "~", home, 1)
	}
	return path
}
