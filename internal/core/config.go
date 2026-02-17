// Package core implements the OpenDoc static site generator engine.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Valid options ────────────────────────────────────────────

var ValidLayouts = []string{"timeline", "grid", "minimal"}
var ValidSorts = []string{"newest_first", "oldest_first", "alphabetical"}

func isValidLayout(l string) bool {
	for _, v := range ValidLayouts {
		if v == l {
			return true
		}
	}
	return false
}

func isValidSort(s string) bool {
	for _, v := range ValidSorts {
		if v == s {
			return true
		}
	}
	return false
}

// ── Config types ────────────────────────────────────────────

type SiteConfig struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
}

type ContentConfig struct {
	Dir string `yaml:"dir"`
}

type BuildConfig struct {
	OutputDir string `yaml:"output_dir"`
}

type CollectionConfig struct {
	ItemsPerPage int    `yaml:"items_per_page"`
	DateFormat   string `yaml:"date_format"`
	Sort         string `yaml:"sort"`
	Tags         bool   `yaml:"tags"`
	Archive      bool   `yaml:"archive"`
	Layout       string `yaml:"layout"`
}

type ThemeConfig struct {
	Name string `yaml:"name"`
}

type NavItem struct {
	Label   string
	Path    string
	Private bool
}

type OpenDocConfig struct {
	Site        SiteConfig
	Content     ContentConfig
	Build       BuildConfig
	Collections map[string]CollectionConfig
	Theme       ThemeConfig
	Nav         []NavItem
}

// ── Defaults ────────────────────────────────────────────────

var DefaultSite = SiteConfig{
	Name:        "My Site",
	URL:         "https://example.com",
	Description: "",
	Author:      "",
}

var DefaultContent = ContentConfig{Dir: "content"}
var DefaultBuild = BuildConfig{OutputDir: "dist"}
var DefaultTheme = ThemeConfig{Name: "default"}

var DefaultCollection = CollectionConfig{
	ItemsPerPage: 10,
	DateFormat:   "%B %d, %Y",
	Sort:         "newest_first",
	Tags:         true,
	Archive:      true,
	Layout:       "timeline",
}

// ── Raw YAML structures ─────────────────────────────────────

type rawConfig struct {
	Site        *SiteConfig                   `yaml:"site"`
	Content     *ContentConfig                `yaml:"content"`
	Build       *BuildConfig                  `yaml:"build"`
	Collections map[string]map[string]any     `yaml:"collections"`
	Blog        map[string]any                `yaml:"blog"`
	Theme       *ThemeConfig                  `yaml:"theme"`
	Nav         []map[string]string           `yaml:"nav"`
}

// ── Loader ──────────────────────────────────────────────────

// LoadConfig reads opendoc.yml from projectDir and returns a validated config.
func LoadConfig(projectDir string) (*OpenDocConfig, error) {
	configPath := filepath.Join(projectDir, "opendoc.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no opendoc.yml found in %s", projectDir)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid opendoc.yml: %w", err)
	}

	// Build config with defaults.
	cfg := &OpenDocConfig{
		Site:        DefaultSite,
		Content:     DefaultContent,
		Build:       DefaultBuild,
		Theme:       DefaultTheme,
		Collections: make(map[string]CollectionConfig),
	}

	if raw.Site != nil {
		if raw.Site.Name != "" {
			cfg.Site.Name = raw.Site.Name
		}
		if raw.Site.URL != "" {
			cfg.Site.URL = raw.Site.URL
		}
		if raw.Site.Description != "" {
			cfg.Site.Description = raw.Site.Description
		}
		if raw.Site.Author != "" {
			cfg.Site.Author = raw.Site.Author
		}
	}

	if raw.Content != nil && raw.Content.Dir != "" {
		cfg.Content.Dir = raw.Content.Dir
	}

	if raw.Build != nil && raw.Build.OutputDir != "" {
		cfg.Build.OutputDir = raw.Build.OutputDir
	}

	if raw.Theme != nil && raw.Theme.Name != "" {
		cfg.Theme.Name = raw.Theme.Name
	}

	// Parse nav items — trailing ? marks a page as private.
	for _, item := range raw.Nav {
		for label, rawPath := range item {
			path := rawPath
			isPrivate := strings.HasSuffix(path, "?")
			if isPrivate {
				path = path[:len(path)-1]
			}
			if path == "index.md" {
				path = ""
			} else if strings.HasSuffix(path, ".md") {
				path = strings.TrimSuffix(path, ".md") + "/"
			}
			cfg.Nav = append(cfg.Nav, NavItem{
				Label:   label,
				Path:    path,
				Private: isPrivate,
			})
		}
	}

	// Parse collections.
	if err := parseCollections(&raw, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseCollections(raw *rawConfig, cfg *OpenDocConfig) error {
	if raw.Collections != nil {
		for name, settings := range raw.Collections {
			coll := DefaultCollection

			if v, ok := settings["items_per_page"]; ok {
				if n, ok := toInt(v); ok {
					coll.ItemsPerPage = n
				}
			}
			if v, ok := settings["date_format"]; ok {
				if s, ok := v.(string); ok {
					coll.DateFormat = s
				}
			}
			if v, ok := settings["sort"]; ok {
				if s, ok := v.(string); ok {
					coll.Sort = s
				}
			}
			if v, ok := settings["tags"]; ok {
				if b, ok := v.(bool); ok {
					coll.Tags = b
				}
			}
			if v, ok := settings["archive"]; ok {
				if b, ok := v.(bool); ok {
					coll.Archive = b
				}
			}
			if v, ok := settings["layout"]; ok {
				if s, ok := v.(string); ok {
					coll.Layout = s
				}
			}

			if !isValidLayout(coll.Layout) {
				return fmt.Errorf("invalid layout '%s' for collection '%s'. Must be one of: %s",
					coll.Layout, name, strings.Join(ValidLayouts, ", "))
			}
			if !isValidSort(coll.Sort) {
				return fmt.Errorf("invalid sort '%s' for collection '%s'. Must be one of: %s",
					coll.Sort, name, strings.Join(ValidSorts, ", "))
			}

			cfg.Collections[name] = coll
		}
	} else if raw.Blog != nil {
		// Backward compat: convert old blog: config to a single collection.
		coll := DefaultCollection

		if v, ok := raw.Blog["posts_per_page"]; ok {
			if n, ok := toInt(v); ok {
				coll.ItemsPerPage = n
			}
		}
		if v, ok := raw.Blog["date_format"]; ok {
			if s, ok := v.(string); ok {
				coll.DateFormat = s
			}
		}
		if v, ok := raw.Blog["sort"]; ok {
			if s, ok := v.(string); ok {
				coll.Sort = s
			}
		}
		if v, ok := raw.Blog["tags"]; ok {
			if b, ok := v.(bool); ok {
				coll.Tags = b
			}
		}
		if v, ok := raw.Blog["archive"]; ok {
			if b, ok := v.(bool); ok {
				coll.Archive = b
			}
		}

		postsDir := "posts"
		if raw.Content != nil {
			if v, ok := raw.Blog["posts_dir"]; ok {
				if s, ok := v.(string); ok {
					postsDir = s
				}
			}
		}

		cfg.Collections[postsDir] = coll
	}

	return nil
}

// toInt converts interface{} to int, handling YAML's tendency to produce int or float.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}
