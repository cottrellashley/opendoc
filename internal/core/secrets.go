package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Secrets holds API keys and other sensitive configuration.
// Stored at ~/.config/opendoc/secrets.yml — never in the project directory.
type Secrets struct {
	AnthropicKey string `yaml:"anthropic_api_key"`
	OpenAIKey    string `yaml:"openai_api_key"`
}

// SecretsPath returns the path to the secrets file.
func SecretsPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opendoc", "secrets.yml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".opendoc", "secrets.yml")
	}
	return filepath.Join(home, ".config", "opendoc", "secrets.yml")
}

// LoadSecrets reads the secrets file. Returns empty secrets if file doesn't exist.
func LoadSecrets() Secrets {
	var s Secrets
	data, err := os.ReadFile(SecretsPath())
	if err != nil {
		return s
	}
	_ = yaml.Unmarshal(data, &s)
	return s
}

// SaveSecrets writes the secrets file with restrictive permissions (0600).
func SaveSecrets(s Secrets) error {
	path := SecretsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create secrets dir: %w", err)
	}

	data, err := yaml.Marshal(&s)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	// Write with owner-only read/write permissions
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write secrets: %w", err)
	}

	return nil
}

// ResolveAPIKey returns the API key for the given provider.
// Priority: environment variable > secrets file.
func ResolveAPIKey(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key
		}
		return LoadSecrets().AnthropicKey
	case "openai":
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			return key
		}
		return LoadSecrets().OpenAIKey
	default:
		return ""
	}
}

// MaskKey returns a masked version of an API key for display (e.g. "sk-...abc123").
func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:3] + "•••" + key[len(key)-4:]
}
