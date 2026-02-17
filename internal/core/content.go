package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ── Types ───────────────────────────────────────────────────

// Page represents a standalone content page.
type Page struct {
	Title           string
	Slug            string
	SourcePath      string
	ContentMarkdown string
	Meta            map[string]any
}

// Entry represents a collection entry (blog post, guide article, etc.).
type Entry struct {
	Title           string
	Slug            string
	SourcePath      string
	ContentMarkdown string
	Date            *time.Time
	Tags            []string
	Description     string
	Draft           bool
	Meta            map[string]any
}

// ── Frontmatter parsing ─────────────────────────────────────

// ParseFrontmatter splits YAML frontmatter from markdown body.
func ParseFrontmatter(text string) (meta map[string]any, body string) {
	meta = make(map[string]any)

	text = strings.TrimLeft(text, "\xef\xbb\xbf") // strip BOM
	if !strings.HasPrefix(text, "---") {
		return meta, strings.TrimSpace(text)
	}

	// Find the closing ---
	rest := text[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, strings.TrimSpace(text)
	}

	yamlBlock := rest[:idx]
	body = strings.TrimSpace(rest[idx+4:])

	_ = yaml.Unmarshal([]byte(yamlBlock), &meta)
	return meta, body
}

// ── Page discovery ──────────────────────────────────────────

// DiscoverPages finds all top-level .md files in contentDir.
func DiscoverPages(contentDir string) []Page {
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil
	}

	var pages []Page
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		filePath := filepath.Join(contentDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		meta, body := ParseFrontmatter(string(data))

		stem := strings.TrimSuffix(entry.Name(), ".md")
		title := stem
		if t, ok := meta["title"].(string); ok && t != "" {
			title = t
		} else {
			title = titleCase(strings.ReplaceAll(stem, "-", " "))
		}

		slug := stem
		if stem == "index" {
			slug = ""
		}

		pages = append(pages, Page{
			Title:           title,
			Slug:            slug,
			SourcePath:      filePath,
			ContentMarkdown: body,
			Meta:            meta,
		})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].SourcePath < pages[j].SourcePath
	})
	return pages
}

// ── Entry discovery ─────────────────────────────────────────

// DiscoverEntries finds collection entries in entriesDir with sorting.
func DiscoverEntries(entriesDir string, sortOrder string, requireDate bool) []Entry {
	dirEntries, err := os.ReadDir(entriesDir)
	if err != nil {
		return nil
	}

	var items []Entry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		if filepath.Ext(de.Name()) != ".md" {
			continue
		}

		filePath := filepath.Join(entriesDir, de.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		meta, body := ParseFrontmatter(string(data))

		stem := strings.TrimSuffix(de.Name(), ".md")
		title := stem
		if t, ok := meta["title"].(string); ok && t != "" {
			title = t
		} else {
			title = titleCase(strings.ReplaceAll(stem, "-", " "))
		}

		var entryDate *time.Time
		if d := meta["date"]; d != nil {
			if t, ok := d.(time.Time); ok {
				entryDate = &t
			} else if s, ok := d.(string); ok {
				if t, err := time.Parse("2006-01-02", s); err == nil {
					entryDate = &t
				} else if t, err := time.Parse(time.RFC3339, s); err == nil {
					entryDate = &t
				}
			}
		}
		if entryDate == nil && requireDate {
			now := time.Now()
			entryDate = &now
		}

		var tags []string
		switch v := meta["tags"].(type) {
		case []any:
			for _, t := range v {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		case string:
			for _, t := range strings.Split(v, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		desc := ""
		if s, ok := meta["description"].(string); ok {
			desc = s
		}

		draft := false
		if b, ok := meta["draft"].(bool); ok {
			draft = b
		}

		items = append(items, Entry{
			Title:           title,
			Slug:            stem,
			SourcePath:      filePath,
			ContentMarkdown: body,
			Date:            entryDate,
			Tags:            tags,
			Description:     desc,
			Draft:           draft,
			Meta:            meta,
		})
	}

	// Sort.
	switch sortOrder {
	case "alphabetical":
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		})
	case "oldest_first":
		sort.Slice(items, func(i, j int) bool {
			ti := timeOrZero(items[i].Date)
			tj := timeOrZero(items[j].Date)
			return ti.Before(tj)
		})
	default: // newest_first
		sort.Slice(items, func(i, j int) bool {
			ti := timeOrZero(items[i].Date)
			tj := timeOrZero(items[j].Date)
			return ti.After(tj)
		})
	}

	return items
}

// ── Helpers ─────────────────────────────────────────────────

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
