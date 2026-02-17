package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/cottrellashley/opendoc/internal/core"
)

// ── Integration status types ────────────────────────────────

// ToolStatus represents the install/auth state of a CLI tool.
type ToolStatus struct {
	Installed     bool   `json:"installed"`
	Authenticated bool   `json:"authenticated"`
	Version       string `json:"version,omitempty"`
	Account       string `json:"account,omitempty"`
	Error         string `json:"error,omitempty"`
}

// IntegrationsStatus holds the status of all integrations.
type IntegrationsStatus struct {
	GitHub ToolStatus `json:"github"`
	Claude ToolStatus `json:"claude"`
}

// ── Check functions ─────────────────────────────────────────

func checkGHStatus() ToolStatus {
	status := ToolStatus{}

	// Check installed
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return status
	}
	status.Installed = true
	_ = ghPath

	// Get version
	out, err := exec.Command("gh", "--version").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			status.Version = strings.TrimSpace(lines[0])
		}
	}

	// Check auth
	out, err = exec.Command("gh", "auth", "status").CombinedOutput()
	if err == nil {
		status.Authenticated = true
		// Try to extract account name
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "Logged in to") {
				status.Account = extractAccount(line)
			}
			if strings.Contains(line, "account") {
				// e.g. "✓ Logged in to github.com account cottrellashley"
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "account" && i+1 < len(parts) {
						status.Account = parts[i+1]
					}
				}
			}
		}
	}

	return status
}

func extractAccount(line string) string {
	// Various gh output formats
	parts := strings.Fields(line)
	for i, p := range parts {
		if p == "as" && i+1 < len(parts) {
			return strings.TrimRight(parts[i+1], "()")
		}
	}
	return ""
}

func checkClaudeStatus() ToolStatus {
	status := ToolStatus{}

	// Check installed
	_, err := exec.LookPath("claude")
	if err != nil {
		return status
	}
	status.Installed = true

	// Get version
	out, err := exec.Command("claude", "--version").CombinedOutput()
	if err == nil {
		status.Version = strings.TrimSpace(string(out))
	}

	// Claude doesn't have a simple auth check; we consider it "authenticated"
	// if it's installed (auth is handled by the tool itself on first use)
	status.Authenticated = true

	return status
}

// ── Publish deployment ──────────────────────────────────────

// PublishRequest holds the parameters for a publish deployment.
type PublishRequest struct {
	Repo string `json:"repo"`
}

// PublishDeployResult holds the outcome of a publish deployment.
type PublishDeployResult struct {
	Success bool   `json:"success"`
	Repo    string `json:"repo,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
	Log     string `json:"log,omitempty"`
}

// ── Register routes ─────────────────────────────────────────

// RegisterIntegrationRoutes adds integration-related API routes.
func RegisterIntegrationRoutes(r chi.Router, workspace string, bm *BuildManager, themesFS fs.FS) {
	// GET /api/integrations/status — check all tools
	r.Get("/api/integrations/status", func(w http.ResponseWriter, r *http.Request) {
		status := IntegrationsStatus{
			GitHub: checkGHStatus(),
			Claude: checkClaudeStatus(),
		}
		writeJSON(w, http.StatusOK, status)
	})

	// GET /api/integrations/api-keys — get masked key status
	r.Get("/api/integrations/api-keys", func(w http.ResponseWriter, r *http.Request) {
		anthropicKey := core.ResolveAPIKey("anthropic")
		openaiKey := core.ResolveAPIKey("openai")
		writeJSON(w, http.StatusOK, map[string]any{
			"anthropic": map[string]any{
				"configured": anthropicKey != "",
				"masked":     core.MaskKey(anthropicKey),
				"source":     keySource("anthropic"),
			},
			"openai": map[string]any{
				"configured": openaiKey != "",
				"masked":     core.MaskKey(openaiKey),
				"source":     keySource("openai"),
			},
		})
	})

	// PUT /api/integrations/api-keys — save API keys
	r.Put("/api/integrations/api-keys", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AnthropicKey string `json:"anthropic_key"`
			OpenAIKey    string `json:"openai_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		secrets := core.LoadSecrets()

		// Only update keys that were provided (non-empty)
		// A special value of "__REMOVE__" clears the key
		if req.AnthropicKey == "__REMOVE__" {
			secrets.AnthropicKey = ""
		} else if req.AnthropicKey != "" {
			secrets.AnthropicKey = req.AnthropicKey
		}

		if req.OpenAIKey == "__REMOVE__" {
			secrets.OpenAIKey = ""
		} else if req.OpenAIKey != "" {
			secrets.OpenAIKey = req.OpenAIKey
		}

		if err := core.SaveSecrets(secrets); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Return updated status
		anthropicKey := core.ResolveAPIKey("anthropic")
		openaiKey := core.ResolveAPIKey("openai")
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"anthropic": map[string]any{
				"configured": anthropicKey != "",
				"masked":     core.MaskKey(anthropicKey),
				"source":     keySource("anthropic"),
			},
			"openai": map[string]any{
				"configured": openaiKey != "",
				"masked":     core.MaskKey(openaiKey),
				"source":     keySource("openai"),
			},
		})
	})

	// GET /api/integrations/gh-login — start gh auth login (SSE stream)
	r.Get("/api/integrations/gh-login", func(w http.ResponseWriter, r *http.Request) {
		startGHLogin(w, r)
	})

	// POST /api/integrations/publish-deploy — build + deploy to GitHub Pages
	r.Post("/api/integrations/publish-deploy", func(w http.ResponseWriter, r *http.Request) {
		handlePublishDeploy(w, r, workspace, bm, themesFS)
	})
}

// ── gh auth login (streaming output) ────────────────────────

func startGHLogin(w http.ResponseWriter, r *http.Request) {
	// Check gh is installed first
	if _, err := exec.LookPath("gh"); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   "gh CLI not installed. Install from https://cli.github.com/",
		})
		return
	}

	// Set up SSE for streaming output
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	// Run gh auth login with web flow
	cmd := exec.Command("gh", "auth", "login", "--hostname", "github.com", "--web", "--git-protocol", "https")
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1")

	// Capture stdout and stderr
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		sendSSEEvent(w, flusher, "error", map[string]string{"message": fmt.Sprintf("Failed to start: %v", err)})
		return
	}

	// Stream output
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamOutput(w, flusher, stdout, "output")
	}()

	go func() {
		defer wg.Done()
		streamOutput(w, flusher, stderr, "output")
	}()

	wg.Wait()
	err := cmd.Wait()

	if err != nil {
		sendSSEEvent(w, flusher, "error", map[string]string{"message": fmt.Sprintf("Login failed: %v", err)})
	} else {
		// Recheck status
		status := checkGHStatus()
		sendSSEEvent(w, flusher, "complete", map[string]any{
			"success": true,
			"status":  status,
		})
	}
}

func streamOutput(w http.ResponseWriter, flusher http.Flusher, reader io.Reader, eventType string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		sendSSEEvent(w, flusher, eventType, map[string]string{"message": line})
	}
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
	flusher.Flush()
}

// ── Publish deploy handler ──────────────────────────────────

// deployToGHPagesBranch pushes outputDir contents to the gh-pages branch.
func deployToGHPagesBranch(outputDir, repo string) error {
	remoteURL := fmt.Sprintf("https://github.com/%s.git", repo)

	tmpDir, err := os.MkdirTemp("", "opendoc-deploy-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	if _, err := git("init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	git("config", "user.email", "opendoc@deploy")
	git("config", "user.name", "OpenDoc Deploy")

	// Copy files from outputDir to tmpDir
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(outputDir, path)
		destPath := filepath.Join(tmpDir, relPath)
		os.MkdirAll(filepath.Dir(destPath), 0o755)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copy files: %w", err)
	}

	// Add .nojekyll
	os.WriteFile(filepath.Join(tmpDir, ".nojekyll"), []byte(""), 0o644)

	if _, err := git("add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if _, err := git("commit", "-m", "Deploy via OpenDoc"); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	pushOut, err := git("push", "--force", remoteURL, "HEAD:gh-pages")
	if err != nil {
		return fmt.Errorf("push failed: %s\n\nMake sure the repo %s exists and you have push access.\nEnable GitHub Pages at: https://github.com/%s/settings/pages", pushOut, repo, repo)
	}
	return nil
}

func handlePublishDeploy(w http.ResponseWriter, r *http.Request, workspace string, bm *BuildManager, themesFS fs.FS) {
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Try to get repo from settings
		req.Repo = ""
	}

	// Resolve repo
	repo := req.Repo
	if repo == "" {
		settings := loadSettings(workspace)
		if settings.GithubRepo != "" {
			repo = settings.GithubRepo
		}
	}
	if repo == "" {
		appCfg := core.LoadAppConfig()
		if appCfg.GitHub.DefaultAccount != "" {
			repo = appCfg.GitHub.DefaultAccount + "/" + filepath.Base(workspace)
		}
	}
	if repo == "" {
		writeJSON(w, http.StatusBadRequest, PublishDeployResult{
			Success: false,
			Error:   "No GitHub repository configured. Set it in Settings or provide --repo.",
		})
		return
	}

	// Check gh
	if _, err := exec.LookPath("gh"); err != nil {
		writeJSON(w, http.StatusBadRequest, PublishDeployResult{
			Success: false,
			Error:   "gh CLI not installed.",
		})
		return
	}

	// Check auth
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		writeJSON(w, http.StatusBadRequest, PublishDeployResult{
			Success: false,
			Error:   "gh is not authenticated. Connect GitHub in Settings first.",
		})
		return
	}

	// Build in publish mode
	config, err := core.LoadConfig(workspace)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, PublishDeployResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to load config: %v", err),
		})
		return
	}

	outputDir := filepath.Join(workspace, "dist-publish")
	err = core.BuildSite(config, workspace, themesFS, core.BuildOptions{
		PublishMode:       true,
		OutputDirOverride: "dist-publish",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, PublishDeployResult{
			Success: false,
			Error:   fmt.Sprintf("Build failed: %v", err),
		})
		return
	}

	// Deploy by pushing to gh-pages branch
	deployErr := deployToGHPagesBranch(outputDir, repo)
	if deployErr != nil {
		writeJSON(w, http.StatusInternalServerError, PublishDeployResult{
			Success: false,
			Repo:    repo,
			Error:   fmt.Sprintf("Deploy failed: %v", deployErr),
			Log:     deployErr.Error(),
		})
		return
	}

	// Build the URL
	parts := strings.SplitN(repo, "/", 2)
	url := ""
	if len(parts) == 2 {
		url = fmt.Sprintf("https://%s.github.io/%s/", parts[0], parts[1])
	}

	writeJSON(w, http.StatusOK, PublishDeployResult{
		Success: true,
		Repo:    repo,
		URL:     url,
		Log:     "Deployed to gh-pages branch",
	})
}

// keySource returns where an API key is coming from.
func keySource(provider string) string {
	envVar := ""
	switch provider {
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	case "openai":
		envVar = "OPENAI_API_KEY"
	}
	if envVar != "" && os.Getenv(envVar) != "" {
		return "environment"
	}
	secrets := core.LoadSecrets()
	switch provider {
	case "anthropic":
		if secrets.AnthropicKey != "" {
			return "settings"
		}
	case "openai":
		if secrets.OpenAIKey != "" {
			return "settings"
		}
	}
	return "none"
}
