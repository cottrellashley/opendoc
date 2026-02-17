package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

// FileTreeEntry represents a file or directory in the file tree.
type FileTreeEntry struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Type     string           `json:"type"` // "file" or "directory"
	Ext      string           `json:"ext,omitempty"`
	Children []*FileTreeEntry `json:"children,omitempty"`
}

// RegisterFileRoutes adds file CRUD routes to the router.
func RegisterFileRoutes(r chi.Router, workspace string) {
	// List files as a tree
	r.Get("/api/files", func(w http.ResponseWriter, r *http.Request) {
		tree := buildFileTree(workspace, workspace)
		writeJSON(w, http.StatusOK, tree)
	})

	// Read a file
	r.Get("/api/files/*", func(w http.ResponseWriter, r *http.Request) {
		relPath := chi.URLParam(r, "*")
		absPath := filepath.Join(workspace, relPath)

		if !strings.HasPrefix(absPath, workspace) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
			return
		}

		info, err := os.Stat(absPath)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
			return
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"path":     relPath,
			"content":  string(content),
			"size":     info.Size(),
			"modified": info.ModTime(),
		})
	})

	// Write/update a file
	r.Put("/api/files/*", func(w http.ResponseWriter, r *http.Request) {
		relPath := chi.URLParam(r, "*")
		absPath := filepath.Join(workspace, relPath)

		if !strings.HasPrefix(absPath, workspace) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
			return
		}

		content, err := extractContent(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		dir := filepath.Dir(absPath)
		os.MkdirAll(dir, 0o755)

		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"path": relPath,
			"size": len(content),
		})
	})

	// Create a new file
	r.Post("/api/files/*", func(w http.ResponseWriter, r *http.Request) {
		relPath := chi.URLParam(r, "*")
		absPath := filepath.Join(workspace, relPath)

		if !strings.HasPrefix(absPath, workspace) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
			return
		}

		if _, err := os.Stat(absPath); err == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "File already exists"})
			return
		}

		content, err := extractContent(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		dir := filepath.Dir(absPath)
		os.MkdirAll(dir, 0o755)

		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"path": relPath})
	})

	// Delete a file
	r.Delete("/api/files/*", func(w http.ResponseWriter, r *http.Request) {
		relPath := chi.URLParam(r, "*")
		absPath := filepath.Join(workspace, relPath)

		if !strings.HasPrefix(absPath, workspace) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Access denied"})
			return
		}

		info, err := os.Stat(absPath)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
			return
		}

		if info.IsDir() {
			os.RemoveAll(absPath)
		} else {
			os.Remove(absPath)
		}

		writeJSON(w, http.StatusOK, map[string]string{"deleted": relPath})
	})
}

// ── File tree builder ───────────────────────────────────────

func buildFileTree(dir, root string) []*FileTreeEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Sort: directories first, then alphabetical
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() && !entries[j].IsDir() {
			return true
		}
		if !entries[i].IsDir() && entries[j].IsDir() {
			return false
		}
		return entries[i].Name() < entries[j].Name()
	})

	var result []*FileTreeEntry
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "dist" || name == "dist-publish" ||
			name == "node_modules" || name == "__pycache__" {
			continue
		}

		absPath := filepath.Join(dir, name)
		relPath, _ := filepath.Rel(root, absPath)

		if entry.IsDir() {
			result = append(result, &FileTreeEntry{
				Name:     name,
				Path:     relPath,
				Type:     "directory",
				Children: buildFileTree(absPath, root),
			})
		} else {
			result = append(result, &FileTreeEntry{
				Name: name,
				Path: relPath,
				Type: "file",
				Ext:  filepath.Ext(name),
			})
		}
	}

	return result
}

// ── Helpers ─────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func extractContent(r *http.Request) (string, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/") {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}

	var payload struct {
		Content string `json:"content"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body), nil
	}
	return payload.Content, nil
}
