package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PublishOptions configures the publish operation.
type PublishOptions struct {
	ProjectDir string
	Repo       string // "owner/repo" override
	ThemesFS   fs.FS
}

// PublishResult holds the outcome of a publish.
type PublishResult struct {
	Repo      string
	OutputDir string
	URL       string
}

// Publish builds in publish mode and deploys to GitHub Pages via gh.
func Publish(opts PublishOptions) (*PublishResult, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	projectDir, _ = filepath.Abs(projectDir)

	// 1. Resolve repo
	repo := opts.Repo
	if repo == "" {
		repo = resolveRepo(projectDir)
	}
	if repo == "" {
		return nil, fmt.Errorf("no GitHub repository configured\n\nSet it with one of:\n  opendoc publish --repo owner/repo\n  opendoc config set github.default_account <account>\n  Edit settings.json in your project with github_repo")
	}

	// 2. Check gh is installed
	if err := checkGH(); err != nil {
		return nil, err
	}

	// 3. Check gh is authenticated
	if err := checkGHAuth(); err != nil {
		return nil, err
	}

	// 4. Load config and build
	config, err := LoadConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	outputDir := filepath.Join(projectDir, "dist-publish")
	err = BuildSite(config, projectDir, opts.ThemesFS, BuildOptions{
		PublishMode:       true,
		OutputDirOverride: "dist-publish",
	})
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	// 5. Deploy by pushing to gh-pages branch
	if err := deployToGHPages(outputDir, repo); err != nil {
		return nil, err
	}

	// 6. Build result
	parts := strings.SplitN(repo, "/", 2)
	var url string
	if len(parts) == 2 {
		url = fmt.Sprintf("https://%s.github.io/%s/", parts[0], parts[1])
	}

	return &PublishResult{
		Repo:      repo,
		OutputDir: outputDir,
		URL:       url,
	}, nil
}

// deployToGHPages pushes the contents of outputDir to the gh-pages branch of the repo.
func deployToGHPages(outputDir, repo string) error {
	// Get the authenticated HTTPS remote URL via gh
	remoteURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// Use a temporary directory for the git operations
	tmpDir, err := os.MkdirTemp("", "opendoc-deploy-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git in tmpDir
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	// Init a new repo
	if _, err := git("init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Configure git user for the commit
	git("config", "user.email", "opendoc@deploy")
	git("config", "user.name", "OpenDoc Deploy")

	// Copy dist-publish contents into the tmp repo
	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(outputDir, path)
		destPath := filepath.Join(tmpDir, relPath)
		os.MkdirAll(filepath.Dir(destPath), 0o755)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("copy files: %w", err)
	}

	// Add a .nojekyll file so GitHub serves raw HTML
	os.WriteFile(filepath.Join(tmpDir, ".nojekyll"), []byte(""), 0o644)

	// Stage all files
	if _, err := git("add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Commit
	if _, err := git("commit", "-m", "Deploy via OpenDoc"); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Force push to gh-pages branch
	pushOut, err := git("push", "--force", remoteURL, "HEAD:gh-pages")
	if err != nil {
		return fmt.Errorf("git push failed: %s\n\nMake sure:\n  1. The repository %s exists\n  2. You have push access\n  3. Enable GitHub Pages (source: gh-pages branch) at:\n     https://github.com/%s/settings/pages", pushOut, repo, repo)
	}

	return nil
}

// ── Helpers ─────────────────────────────────────────────────

// resolveRepo finds the GitHub repo from settings.json, app config, or git remote.
func resolveRepo(projectDir string) string {
	// Try settings.json first
	settingsPath := filepath.Join(projectDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var s struct {
			GithubRepo string `json:"github_repo"`
		}
		if json.Unmarshal(data, &s) == nil && s.GithubRepo != "" {
			return s.GithubRepo
		}
	}

	// Try app config
	appCfg := LoadAppConfig()
	if appCfg.GitHub.DefaultAccount != "" {
		// Infer repo name from directory name
		base := filepath.Base(projectDir)
		return appCfg.GitHub.DefaultAccount + "/" + base
	}

	// Try git remote
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err == nil {
		remote := strings.TrimSpace(string(out))
		return extractRepoFromRemote(remote)
	}

	return ""
}

// extractRepoFromRemote parses owner/repo from a git remote URL.
func extractRepoFromRemote(remote string) string {
	remote = strings.TrimSuffix(remote, ".git")

	// SSH: git@github.com:owner/repo
	if strings.HasPrefix(remote, "git@github.com:") {
		return strings.TrimPrefix(remote, "git@github.com:")
	}

	// HTTPS: https://github.com/owner/repo
	if strings.Contains(remote, "github.com/") {
		idx := strings.Index(remote, "github.com/")
		return remote[idx+len("github.com/"):]
	}

	return ""
}

// checkGH verifies that the gh CLI is installed.
func checkGH() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found\n\nInstall it from: https://cli.github.com/\n  brew install gh        (macOS)\n  sudo apt install gh    (Ubuntu/Debian)")
	}
	return nil
}

// checkGHAuth verifies that gh is authenticated.
func checkGHAuth() error {
	cmd := exec.Command("gh", "auth", "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh is not authenticated\n\nRun: gh auth login\n\nOutput: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// CheckGHAvailable returns true if gh is installed and authenticated.
func CheckGHAvailable() (installed bool, authenticated bool) {
	if checkGH() != nil {
		return false, false
	}
	if checkGHAuth() != nil {
		return true, false
	}
	return true, true
}
