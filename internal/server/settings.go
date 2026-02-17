package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// WorkbenchSettings holds the workbench configuration.
type WorkbenchSettings struct {
	ProjectName   string `json:"project_name"`
	UserName      string `json:"user_name"`
	GithubAccount string `json:"github_account"`
	GithubRepo    string `json:"github_repo"`
}

var defaultSettings = WorkbenchSettings{
	ProjectName: "OpenDoc",
}

func loadSettings(workspace string) WorkbenchSettings {
	settings := defaultSettings

	// Read project_name from opendoc.yml if available
	configPath := filepath.Join(workspace, "opendoc.yml")
	if data, err := os.ReadFile(configPath); err == nil {
		var raw map[string]any
		if err := yaml.Unmarshal(data, &raw); err == nil {
			if site, ok := raw["site"].(map[string]any); ok {
				if name, ok := site["name"].(string); ok {
					settings.ProjectName = name
				}
			}
		}
	}

	// Override with settings.json values
	settingsPath := filepath.Join(workspace, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var saved WorkbenchSettings
		if err := json.Unmarshal(data, &saved); err == nil {
			if saved.ProjectName != "" {
				settings.ProjectName = saved.ProjectName
			}
			if saved.UserName != "" {
				settings.UserName = saved.UserName
			}
			if saved.GithubAccount != "" {
				settings.GithubAccount = saved.GithubAccount
			}
			if saved.GithubRepo != "" {
				settings.GithubRepo = saved.GithubRepo
			}
		}
	}

	return settings
}

func saveSettings(workspace string, settings WorkbenchSettings) {
	settingsPath := filepath.Join(workspace, "settings.json")
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0o644)

	// Sync project_name into opendoc.yml site.name
	configPath := filepath.Join(workspace, "opendoc.yml")
	if configData, err := os.ReadFile(configPath); err == nil {
		var raw map[string]any
		if err := yaml.Unmarshal(configData, &raw); err == nil {
			if site, ok := raw["site"].(map[string]any); ok {
				if name, ok := site["name"].(string); ok && name != settings.ProjectName {
					site["name"] = settings.ProjectName
					newData, _ := yaml.Marshal(raw)
					os.WriteFile(configPath, newData, 0o644)
				}
			}
		}
	}
}

// RegisterSettingsRoutes adds settings API routes to the router.
func RegisterSettingsRoutes(r chi.Router, workspace string, bm *BuildManager, sse *SSEBroker) {
	r.Get("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, loadSettings(workspace))
	})

	r.Put("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		current := loadSettings(workspace)
		var updates WorkbenchSettings
		if err := json.Unmarshal(body, &updates); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		nameChanged := current.ProjectName != updates.ProjectName

		if updates.ProjectName != "" {
			current.ProjectName = updates.ProjectName
		}
		if updates.UserName != "" {
			current.UserName = updates.UserName
		}
		if updates.GithubAccount != "" {
			current.GithubAccount = updates.GithubAccount
		}
		if updates.GithubRepo != "" {
			current.GithubRepo = updates.GithubRepo
		}

		saveSettings(workspace, current)

		if nameChanged {
			go bm.TriggerBuild()
		}

		sse.Broadcast("settings-changed", current)
		writeJSON(w, http.StatusOK, current)
	})
}
