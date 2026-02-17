// Package server implements the OpenDoc Workbench HTTP server.
package server

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cottrellashley/opendoc/internal/chat"
	"github.com/cottrellashley/opendoc/internal/web"
)

// WorkbenchConfig holds the server configuration.
type WorkbenchConfig struct {
	Port      int
	Workspace string
	ThemesFS  fs.FS
	PublicFS  fs.FS
}

// StartWorkbench starts the full workbench server.
func StartWorkbench(cfg WorkbenchConfig) error {
	workspace := cfg.Workspace
	configFile := filepath.Join(workspace, "opendoc.yml")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// ── SSE ─────────────────────────────────────────────
	sse := NewSSEBroker()
	r.Get("/api/events", sse.ServeHTTP)

	// ── Build manager ───────────────────────────────────
	bm := NewBuildManager(workspace, cfg.ThemesFS, sse)

	// ── File API ────────────────────────────────────────
	RegisterFileRoutes(r, workspace)

	// ── Build API + Preview ─────────────────────────────
	RegisterBuildRoutes(r, bm, workspace)

	// ── Settings API ────────────────────────────────────
	RegisterSettingsRoutes(r, workspace, bm, sse)

	// ── Integrations API (gh, claude, publish) ──────────
	RegisterIntegrationRoutes(r, workspace, bm, cfg.ThemesFS)

	// ── Chat API ────────────────────────────────────────
	chat.RegisterChatRoutes(r, workspace, func() map[string]any {
		return bm.TriggerBuild()
	})

	// ── Console WebSocket (in-process) ─────────────────
	consoleServer := web.NewServer(web.Config{
		MaxSessions: 10,
		IdleTimeout: 5 * time.Minute,
	})
	r.HandleFunc("/console/ws", consoleServer.HandleWebSocket)

	// ── Serve workbench UI from embedded PublicFS ────────
	publicSub, err := fs.Sub(cfg.PublicFS, "public")
	if err != nil {
		return fmt.Errorf("failed to create public sub-FS: %w", err)
	}
	staticServer := http.FileServer(http.FS(publicSub))

	// SPA catch-all: serve index.html for non-API, non-preview routes
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Don't catch API or preview routes
		if strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/preview") ||
			strings.HasPrefix(path, "/publish-preview") ||
			strings.HasPrefix(path, "/console") {
			http.NotFound(w, r)
			return
		}

		// Try to serve static file
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if _, err := fs.Stat(publicSub, cleanPath); err == nil {
			staticServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback — serve index.html
		indexData, err := fs.ReadFile(publicSub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexData)
	})

	// ── Initial build ───────────────────────────────────
	if _, err := os.Stat(configFile); err == nil {
		log.Println("[workbench] Running initial build...")
		result := bm.TriggerBuild()
		if result["success"] == true {
			log.Println("[workbench] Initial build complete.")
		} else {
			log.Printf("[workbench] Initial build failed: %v", result["error"])
		}
	}

	// ── File watcher ────────────────────────────────────
	contentDir := filepath.Join(workspace, "content")
	if _, err := os.Stat(contentDir); err == nil {
		StartWatcher(workspace, bm, sse)
	} else {
		log.Println("[workbench] No content/ dir yet — watcher will start after scaffold")
	}

	// ── Start server ────────────────────────────────────
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("[workbench] OpenDoc Workbench running on port %d", cfg.Port)
	log.Printf("[workbench] Workspace: %s", workspace)

	return http.ListenAndServe(addr, r)
}
