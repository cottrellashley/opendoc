package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig is the global user-level OpenDoc configuration.
type AppConfig struct {
	GitHub   AppGitHub   `yaml:"github"`
	Defaults AppDefaults `yaml:"defaults"`
}

// AppGitHub holds GitHub-related config.
type AppGitHub struct {
	DefaultAccount string `yaml:"default_account"`
}

// AppDefaults holds default values for new projects and commands.
type AppDefaults struct {
	Author string `yaml:"author"`
	Theme  string `yaml:"theme"`
	Port   int    `yaml:"port"`
}

// DefaultAppConfig returns the zero-value config with sensible defaults.
func DefaultAppConfig() AppConfig {
	return AppConfig{
		Defaults: AppDefaults{
			Theme: "default",
			Port:  3000,
		},
	}
}

// AppConfigPath returns the path to the user-level config file.
// Follows XDG_CONFIG_HOME, falls back to ~/.config/opendoc/config.yml.
func AppConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opendoc", "config.yml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".opendoc", "config.yml")
	}
	return filepath.Join(home, ".config", "opendoc", "config.yml")
}

// LoadAppConfig reads the user-level config. Returns defaults if the file doesn't exist.
func LoadAppConfig() AppConfig {
	cfg := DefaultAppConfig()
	path := AppConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)

	// Ensure defaults
	if cfg.Defaults.Theme == "" {
		cfg.Defaults.Theme = "default"
	}
	if cfg.Defaults.Port == 0 {
		cfg.Defaults.Port = 3000
	}

	return cfg
}

// SaveAppConfig writes the user-level config to disk.
func SaveAppConfig(cfg AppConfig) error {
	path := AppConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// SetAppConfigValue sets a dot-separated key to a value (e.g. "github.default_account" = "user").
func SetAppConfigValue(cfg *AppConfig, key, value string) error {
	switch strings.ToLower(key) {
	case "github.default_account":
		cfg.GitHub.DefaultAccount = value
	case "defaults.author":
		cfg.Defaults.Author = value
	case "defaults.theme":
		cfg.Defaults.Theme = value
	case "defaults.port":
		var port int
		if _, err := fmt.Sscanf(value, "%d", &port); err != nil {
			return fmt.Errorf("invalid port: %s", value)
		}
		cfg.Defaults.Port = port
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: github.default_account, defaults.author, defaults.theme, defaults.port", key)
	}
	return nil
}

// FormatAppConfig returns a human-readable string of the config.
func FormatAppConfig(cfg AppConfig) string {
	var b strings.Builder
	b.WriteString("github:\n")
	b.WriteString(fmt.Sprintf("  default_account: %s\n", valueOrNone(cfg.GitHub.DefaultAccount)))
	b.WriteString("defaults:\n")
	b.WriteString(fmt.Sprintf("  author: %s\n", valueOrNone(cfg.Defaults.Author)))
	b.WriteString(fmt.Sprintf("  theme:  %s\n", cfg.Defaults.Theme))
	b.WriteString(fmt.Sprintf("  port:   %d\n", cfg.Defaults.Port))
	return b.String()
}

func valueOrNone(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
