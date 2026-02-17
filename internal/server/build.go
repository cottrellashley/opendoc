package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cottrellashley/opendoc/internal/core"
)

// BuildManager handles build operations.
type BuildManager struct {
	mu               sync.Mutex
	building         bool
	publishBuilding  bool
	workspace        string
	themesFS         fs.FS
	sse              *SSEBroker
}

// NewBuildManager creates a new build manager.
func NewBuildManager(workspace string, themesFS fs.FS, sse *SSEBroker) *BuildManager {
	return &BuildManager{
		workspace: workspace,
		themesFS:  themesFS,
		sse:       sse,
	}
}

// TriggerBuild runs a full site build.
func (bm *BuildManager) TriggerBuild() map[string]any {
	bm.mu.Lock()
	if bm.building {
		bm.mu.Unlock()
		return map[string]any{"success": false, "error": "Build in progress", "time": time.Now().UnixMilli()}
	}
	bm.building = true
	bm.mu.Unlock()

	bm.sse.Broadcast("build-start", map[string]any{"time": time.Now().UnixMilli()})

	config, err := core.LoadConfig(bm.workspace)
	if err != nil {
		bm.mu.Lock()
		bm.building = false
		bm.mu.Unlock()
		result := map[string]any{"success": false, "error": err.Error(), "time": time.Now().UnixMilli()}
		bm.sse.Broadcast("build-complete", result)
		return result
	}

	err = core.BuildSite(config, bm.workspace, bm.themesFS, core.BuildOptions{})
	bm.mu.Lock()
	bm.building = false
	bm.mu.Unlock()

	if err != nil {
		result := map[string]any{"success": false, "error": err.Error(), "time": time.Now().UnixMilli()}
		bm.sse.Broadcast("build-complete", result)
		return result
	}

	result := map[string]any{"success": true, "time": time.Now().UnixMilli()}
	bm.sse.Broadcast("build-complete", result)
	return result
}

// TriggerPublishBuild runs a publish-mode build.
func (bm *BuildManager) TriggerPublishBuild() map[string]any {
	bm.mu.Lock()
	if bm.publishBuilding {
		bm.mu.Unlock()
		return map[string]any{"success": false, "error": "Build in progress", "time": time.Now().UnixMilli()}
	}
	bm.publishBuilding = true
	bm.mu.Unlock()

	bm.sse.Broadcast("publish-build-start", map[string]any{"time": time.Now().UnixMilli()})

	config, err := core.LoadConfig(bm.workspace)
	if err != nil {
		bm.mu.Lock()
		bm.publishBuilding = false
		bm.mu.Unlock()
		result := map[string]any{"success": false, "error": err.Error(), "time": time.Now().UnixMilli()}
		bm.sse.Broadcast("publish-build-complete", result)
		return result
	}

	err = core.BuildSite(config, bm.workspace, bm.themesFS, core.BuildOptions{
		PublishMode:       true,
		OutputDirOverride: "dist-publish",
		NoBasePath:        true, // Workbench preview uses its own path rewriting
	})

	bm.mu.Lock()
	bm.publishBuilding = false
	bm.mu.Unlock()

	if err != nil {
		result := map[string]any{"success": false, "error": err.Error(), "time": time.Now().UnixMilli()}
		bm.sse.Broadcast("publish-build-complete", result)
		return result
	}

	result := map[string]any{"success": true, "time": time.Now().UnixMilli()}
	bm.sse.Broadcast("publish-build-complete", result)
	return result
}

// RegisterBuildRoutes adds build API routes to the router.
func RegisterBuildRoutes(r chi.Router, bm *BuildManager, workspace string) {
	configFile := filepath.Join(workspace, "opendoc.yml")
	distDir := filepath.Join(workspace, "dist")
	distPublishDir := filepath.Join(workspace, "dist-publish")

	// Build API
	r.Post("/api/opendoc/build", func(w http.ResponseWriter, r *http.Request) {
		result := bm.TriggerBuild()
		if result["success"] == true {
			writeJSON(w, http.StatusOK, result)
		} else {
			writeJSON(w, http.StatusInternalServerError, result)
		}
	})

	r.Get("/api/opendoc/status", func(w http.ResponseWriter, r *http.Request) {
		bm.mu.Lock()
		building := bm.building
		bm.mu.Unlock()

		_, configExists := os.Stat(configFile)
		_, distExists := os.Stat(distDir)

		writeJSON(w, http.StatusOK, map[string]any{
			"building":  building,
			"workspace": workspace,
			"hasConfig": configExists == nil,
			"hasDist":   distExists == nil,
		})
	})

	// Publish build API
	r.Post("/api/opendoc/publish-build", func(w http.ResponseWriter, r *http.Request) {
		result := bm.TriggerPublishBuild()
		if result["success"] == true {
			writeJSON(w, http.StatusOK, result)
		} else {
			writeJSON(w, http.StatusInternalServerError, result)
		}
	})

	// Preview serving
	r.Handle("/preview/static/*", http.StripPrefix("/preview/static/",
		http.FileServer(http.Dir(filepath.Join(distDir, "static")))))

	r.Get("/preview", func(w http.ResponseWriter, r *http.Request) {
		servePreviewHTML(filepath.Join(distDir, "index.html"), "/preview/", w)
	})

	r.Get("/preview/*", func(w http.ResponseWriter, r *http.Request) {
		subPath := chi.URLParam(r, "*")
		servePreviewPath(distDir, subPath, "/preview/", w)
	})

	// Publish preview serving
	r.Handle("/publish-preview/static/*", http.StripPrefix("/publish-preview/static/",
		http.FileServer(http.Dir(filepath.Join(distPublishDir, "static")))))

	r.Get("/publish-preview", func(w http.ResponseWriter, r *http.Request) {
		servePreviewHTML(filepath.Join(distPublishDir, "index.html"), "/publish-preview/", w)
	})

	r.Get("/publish-preview/*", func(w http.ResponseWriter, r *http.Request) {
		subPath := chi.URLParam(r, "*")
		servePreviewPath(distPublishDir, subPath, "/publish-preview/", w)
	})
}

// ── Preview HTML rewriting ──────────────────────────────────

func servePreviewHTML(htmlPath, prefix string, w http.ResponseWriter) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	html := string(data)

	// Rewrite absolute href/src/action paths to include prefix
	rewriter := strings.NewReplacer(
		`href="/`, fmt.Sprintf(`href="%s`, prefix),
		`src="/`, fmt.Sprintf(`src="%s`, prefix),
		`action="/`, fmt.Sprintf(`action="%s`, prefix),
	)
	html = rewriter.Replace(html)

	// Inject: hide scrollbar + external links in new tab + theme sync
	inject := `<style>
html { scrollbar-width: none; }
::-webkit-scrollbar { display: none; }
</style>
<script>
document.addEventListener('click', function(e) {
  var a = e.target.closest('a');
  if (!a) return;
  var href = a.getAttribute('href') || '';
  if (href.startsWith('http://') || href.startsWith('https://')) {
    e.preventDefault();
    window.open(href, '_blank');
  }
});
window.addEventListener('message', function(e) {
  if (e.data && e.data.type === 'opendoc-theme-sync') {
    document.documentElement.setAttribute('data-theme', e.data.theme);
    try { localStorage.setItem('opendoc-theme', e.data.theme); } catch(ex) {}
  }
});
(function() {
  var origToggle = document.getElementById('theme-toggle');
  if (!origToggle) return;
  origToggle.addEventListener('click', function() {
    var t = document.documentElement.getAttribute('data-theme') || 'light';
    if (window.parent && window.parent !== window) {
      window.parent.postMessage({ type: 'opendoc-theme-sync', theme: t }, '*');
    }
  });
})();
</script>`

	if strings.Contains(html, "</head>") {
		html = strings.Replace(html, "</head>", inject+"</head>", 1)
	} else {
		html = inject + html
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func servePreviewPath(distDir, subPath, prefix string, w http.ResponseWriter) {
	direct := filepath.Join(distDir, subPath)

	// If it's a file that exists, serve directly
	if info, err := os.Stat(direct); err == nil && !info.IsDir() {
		http.ServeFile(w, &http.Request{}, direct)
		return
	}

	// Try index.html for clean URLs
	tryIndex := filepath.Join(distDir, subPath, "index.html")
	if _, err := os.Stat(tryIndex); err == nil {
		servePreviewHTML(tryIndex, prefix, w)
		return
	}

	// Try direct .html
	tryHTML := direct
	if !strings.HasSuffix(tryHTML, ".html") {
		tryHTML += ".html"
	}
	if _, err := os.Stat(tryHTML); err == nil {
		servePreviewHTML(tryHTML, prefix, w)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
