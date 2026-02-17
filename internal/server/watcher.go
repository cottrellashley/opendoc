package server

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounceDuration = 800 * time.Millisecond

// StartWatcher watches the content directory and config file for changes
// and triggers rebuilds with debouncing.
func StartWatcher(workspace string, bm *BuildManager, sse *SSEBroker) {
	contentDir := filepath.Join(workspace, "content")
	configFile := filepath.Join(workspace, "opendoc.yml")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("[watcher] Failed to create watcher: %v", err)
		return
	}

	// Add content directory (recursively) and config file
	if _, err := os.Stat(contentDir); err == nil {
		addDirRecursive(watcher, contentDir)
	}
	if _, err := os.Stat(configFile); err == nil {
		watcher.Add(configFile)
	}

	log.Println("[watcher] Watching for changes in content/ and opendoc.yml")

	var timer *time.Timer

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Skip hidden files and dist directories
				base := filepath.Base(event.Name)
				if strings.HasPrefix(base, ".") || strings.Contains(event.Name, "/dist") {
					continue
				}

				relPath, _ := filepath.Rel(workspace, event.Name)
				log.Printf("[watcher] Changed: %s", relPath)

				sse.Broadcast("file-changed", map[string]any{
					"path": relPath,
					"time": time.Now().UnixMilli(),
				})

				// If a new directory was created, watch it too
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						watcher.Add(event.Name)
					}
				}

				// Debounced rebuild
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceDuration, func() {
					bm.TriggerBuild()
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[watcher] Error: %v", err)
			}
		}
	}()
}

func addDirRecursive(watcher *fsnotify.Watcher, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			watcher.Add(path)
		}
		return nil
	})
}
