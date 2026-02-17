package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	opendoc "github.com/cottrellashley/opendoc"
	"github.com/cottrellashley/opendoc/internal/core"
	"github.com/cottrellashley/opendoc/internal/server"
	"github.com/cottrellashley/opendoc/internal/tui"
)

// ── Root ────────────────────────────────────────────────────

var rootCmd = &cobra.Command{
	Use:   "opendoc",
	Short: "OpenDoc — a static site generator",
	Long: `OpenDoc is a static site generator with an integrated workbench and AI chat.

Build beautiful static sites from markdown, publish to GitHub Pages,
and manage everything from the CLI or the browser-based workbench.

Get started:
  opendoc new my-site     Create a new project
  opendoc build           Build the static site
  opendoc serve           Serve locally with live reload
  opendoc publish         Deploy to GitHub Pages
  opendoc workbench       Start the full workbench UI`,
}

// ── opendoc build ───────────────────────────────────────────

var buildCmd = &cobra.Command{
	Use:   "build [project-dir]",
	Short: "Build the static site",
	Long:  "Build the static site from markdown content. Defaults to the current directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := resolveProjectDir(args)
		config, err := core.LoadConfig(projectDir)
		if err != nil {
			return err
		}

		core.InfoMsg(fmt.Sprintf("Building %s...", core.CLIBold.Render(config.Site.Name)))
		start := time.Now()

		if err := core.BuildSite(config, projectDir, opendoc.ThemesFS, core.BuildOptions{}); err != nil {
			core.ErrMsg(fmt.Sprintf("Build failed: %v", err))
			return err
		}

		elapsed := time.Since(start)
		core.OkMsg(fmt.Sprintf("Built to %s/ in %dms", config.Build.OutputDir, elapsed.Milliseconds()))
		return nil
	},
}

// ── opendoc serve ───────────────────────────────────────────

var servePort string

var serveCmd = &cobra.Command{
	Use:   "serve [project-dir]",
	Short: "Serve the site locally with live reload",
	Long:  "Build the site and start a local HTTP server. Rebuilds on file changes.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := resolveProjectDir(args)
		config, err := core.LoadConfig(projectDir)
		if err != nil {
			return err
		}

		// Resolve port: flag > app config > default
		port := servePort
		if port == "" || port == "8000" {
			appCfg := core.LoadAppConfig()
			if appCfg.Defaults.Port != 0 && appCfg.Defaults.Port != 3000 {
				port = strconv.Itoa(appCfg.Defaults.Port)
			}
		}

		core.InfoMsg(fmt.Sprintf("Building %s...", core.CLIBold.Render(config.Site.Name)))
		start := time.Now()
		if err := core.BuildSite(config, projectDir, opendoc.ThemesFS, core.BuildOptions{}); err != nil {
			core.ErrMsg(fmt.Sprintf("Build failed: %v", err))
			return err
		}
		core.OkMsg(fmt.Sprintf("Built in %dms", time.Since(start).Milliseconds()))

		outputDir := filepath.Join(projectDir, config.Build.OutputDir)
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(outputDir)))

		addr := ":" + port
		fmt.Println()
		core.InfoMsg(fmt.Sprintf("Serving at %s", core.CLIAccent.Render("http://localhost"+addr)))
		core.StepMsg("Press Ctrl+C to stop")
		fmt.Println()

		srv := &http.Server{Addr: addr, Handler: mux}

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-stop
			fmt.Println()
			core.InfoMsg("Stopping server...")
			srv.Close()
		}()

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	},
}

// ── opendoc new ─────────────────────────────────────────────

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new project",
	Long:  "Create a new OpenDoc project with sample content and configuration.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Pre-fill author from app config
		appCfg := core.LoadAppConfig()
		if appCfg.Defaults.Author != "" {
			core.StepMsg(fmt.Sprintf("Using default author: %s", appCfg.Defaults.Author))
		}

		if _, err := core.CreateProject(name); err != nil {
			return err
		}

		fmt.Println()
		core.OkMsg(fmt.Sprintf("Created project %s", core.CLIBold.Render(name)))
		fmt.Println()
		core.StepMsg("Get started:")
		fmt.Printf("    cd %s\n", name)
		fmt.Println("    opendoc build")
		fmt.Println("    opendoc serve")
		fmt.Println()
		return nil
	},
}

// ── opendoc publish ─────────────────────────────────────────

var publishRepo string

var publishCmd = &cobra.Command{
	Use:   "publish [project-dir]",
	Short: "Build and deploy to GitHub Pages",
	Long: `Build the site in publish mode (excluding private pages) and deploy
to GitHub Pages using the gh CLI.

The target repository is resolved from:
  1. --repo flag
  2. settings.json github_repo field
  3. App config github.default_account + directory name
  4. Git remote origin`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := resolveProjectDir(args)

		core.Banner()
		core.InfoMsg("Starting publish...")
		fmt.Println()

		start := time.Now()
		result, err := core.Publish(core.PublishOptions{
			ProjectDir: projectDir,
			Repo:       publishRepo,
			ThemesFS:   opendoc.ThemesFS,
		})
		if err != nil {
			core.ErrMsg(err.Error())
			return fmt.Errorf("publish failed")
		}

		elapsed := time.Since(start)
		fmt.Println()
		core.DoneMsg(fmt.Sprintf("Published to %s in %ds", core.CLIBold.Render(result.Repo), int(elapsed.Seconds())))
		if result.URL != "" {
			core.StepMsg(fmt.Sprintf("URL: %s", core.CLIAccent.Render(result.URL)))
		}
		fmt.Println()
		return nil
	},
}

// ── opendoc status ──────────────────────────────────────────

var statusCmd = &cobra.Command{
	Use:   "status [project-dir]",
	Short: "Show project status and health",
	Long:  "Display project info, build status, and GitHub configuration.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := resolveProjectDir(args)

		core.Banner()

		// Project config
		config, err := core.LoadConfig(projectDir)
		if err != nil {
			core.ErrMsg("No opendoc.yml found in " + projectDir)
			return nil
		}

		fmt.Println(core.StatusLine("project", core.CLIBold.Render(config.Site.Name)))
		fmt.Println(core.StatusLine("directory", projectDir))
		fmt.Println(core.StatusLine("content", config.Content.Dir+"/"))
		fmt.Println(core.StatusLine("output", config.Build.OutputDir+"/"))
		fmt.Println(core.StatusLine("theme", config.Theme.Name))
		fmt.Println()

		// Count pages
		contentDir := filepath.Join(projectDir, config.Content.Dir)
		pages := core.DiscoverPages(contentDir)
		fmt.Println(core.StatusLine("pages", strconv.Itoa(len(pages))))

		// Count collection entries
		totalEntries := 0
		for collName := range config.Collections {
			entries := core.DiscoverEntries(filepath.Join(contentDir, collName), "newest_first", false)
			totalEntries += len(entries)
		}
		fmt.Println(core.StatusLine("entries", strconv.Itoa(totalEntries)))
		fmt.Println(core.StatusLine("collections", strconv.Itoa(len(config.Collections))))

		// Nav items
		privCount := 0
		for _, n := range config.Nav {
			if n.Private {
				privCount++
			}
		}
		navInfo := strconv.Itoa(len(config.Nav))
		if privCount > 0 {
			navInfo += fmt.Sprintf(" (%d private)", privCount)
		}
		fmt.Println(core.StatusLine("nav items", navInfo))
		fmt.Println()

		// Last build
		buildIDPath := filepath.Join(projectDir, config.Build.OutputDir, ".opendoc-build-id")
		if data, err := os.ReadFile(buildIDPath); err == nil {
			ms, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if ms > 0 {
				fmt.Println(core.StatusLine("last build", core.TimeSince(ms)))
			}
		} else {
			fmt.Println(core.StatusLine("last build", core.CLIWarn.Render("never")))
		}

		// GitHub
		ghInstalled, ghAuthed := core.CheckGHAvailable()
		if ghInstalled && ghAuthed {
			fmt.Println(core.StatusLine("gh cli", core.CLISuccess.Render("authenticated")))
		} else if ghInstalled {
			fmt.Println(core.StatusLine("gh cli", core.CLIWarn.Render("installed but not authenticated")))
		} else {
			fmt.Println(core.StatusLine("gh cli", core.CLIMuted.Render("not installed")))
		}

		// Repo
		repo := ""
		settingsPath := filepath.Join(projectDir, "settings.json")
		if data, err := os.ReadFile(settingsPath); err == nil {
			var s struct {
				GithubRepo string `json:"github_repo"`
			}
			if err := parseJSON(data, &s); err == nil && s.GithubRepo != "" {
				repo = s.GithubRepo
			}
		}
		if repo != "" {
			fmt.Println(core.StatusLine("github repo", repo))
		} else {
			fmt.Println(core.StatusLine("github repo", core.CLIMuted.Render("not configured")))
		}
		fmt.Println()

		return nil
	},
}

// ── opendoc config ──────────────────────────────────────────

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage app-level configuration",
	Long: `View and modify the global OpenDoc configuration.

Config file location: ` + core.AppConfigPath() + `

Available keys:
  github.default_account   Your GitHub username
  defaults.author          Default author for new projects
  defaults.theme           Default theme (default: "default")
  defaults.port            Default serve port (default: 3000)`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := core.LoadAppConfig()
		core.Banner()
		fmt.Println(core.StatusLine("config file", core.AppConfigPath()))
		fmt.Println()
		fmt.Print(core.FormatAppConfig(cfg))
		fmt.Println()
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a global configuration value.

Examples:
  opendoc config set github.default_account cottrellashley
  opendoc config set defaults.author "Ashley Cottrell"
  opendoc config set defaults.theme default
  opendoc config set defaults.port 8000`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg := core.LoadAppConfig()
		if err := core.SetAppConfigValue(&cfg, key, value); err != nil {
			return err
		}
		if err := core.SaveAppConfig(cfg); err != nil {
			return err
		}

		core.OkMsg(fmt.Sprintf("Set %s = %s", core.CLIAccent.Render(key), value))
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print config file location",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(core.AppConfigPath())
	},
}

// ── opendoc workbench ───────────────────────────────────────

var workbenchPort string

var workbenchCmd = &cobra.Command{
	Use:   "workbench [project-dir]",
	Short: "Start the IDE-like workbench server",
	Long:  "Start the full workbench with editor, preview, chat, and terminal.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := resolveProjectDir(args)

		port := 3000
		if workbenchPort != "" {
			fmt.Sscanf(workbenchPort, "%d", &port)
		}

		return server.StartWorkbench(server.WorkbenchConfig{
			Port:      port,
			Workspace: projectDir,
			ThemesFS:  opendoc.ThemesFS,
			PublicFS:  opendoc.PublicFS,
		})
	},
}

// ── opendoc tui ─────────────────────────────────────────────

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Run the interactive terminal UI",
	Long:  "Launch the Bubble Tea TUI for browsing docs and running commands.",
	RunE: func(cmd *cobra.Command, args []string) error {
		allowedStr := os.Getenv("ALLOWED_COMMANDS")
		var allowed []string
		if allowedStr != "" {
			allowed = strings.Split(allowedStr, ",")
		}
		model := tui.NewModel(allowed)
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

// ── init ────────────────────────────────────────────────────

func init() {
	// Flags
	serveCmd.Flags().StringVarP(&servePort, "port", "p", "8000", "Port to serve on")
	workbenchCmd.Flags().StringVarP(&workbenchPort, "port", "p", "3000", "Port for the workbench")
	publishCmd.Flags().StringVar(&publishRepo, "repo", "", "GitHub repo (owner/repo) to deploy to")

	// Config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)

	// Root commands
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(publishCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(workbenchCmd)
	rootCmd.AddCommand(tuiCmd)

	rootCmd.SilenceUsage = true
}

// ── Helpers ─────────────────────────────────────────────────

func resolveProjectDir(args []string) string {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, _ := filepath.Abs(dir)
	return abs
}

// parseJSON is a small helper to unmarshal JSON into a struct.
func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
