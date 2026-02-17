package core

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/yuin/goldmark"
)

const wordsPerMinute = 200

// ── Build options ────────────────────────────────────────────

// BuildOptions configures the build pipeline.
type BuildOptions struct {
	PublishMode       bool   // When true, exclude private pages/collections
	OutputDirOverride string // Override output directory (e.g. "dist-publish")
	BasePath          string // URL base path override (e.g. "/bark"). Empty = auto from site.url in publish mode.
	NoBasePath        bool   // When true, force empty base path even in publish mode
}

// CollectionContext holds metadata about a collection for templates.
type CollectionContext struct {
	Name      string
	Label     string
	URLPrefix string
	Layout    string
	DateFormat string
}

// ── Main build function ─────────────────────────────────────

// BuildSite runs the full build pipeline.
func BuildSite(config *OpenDocConfig, projectDir string, themesFS fs.FS, options BuildOptions) error {
	outputDirName := config.Build.OutputDir
	if options.OutputDirOverride != "" {
		outputDirName = options.OutputDirOverride
	}
	outputDir := filepath.Join(projectDir, outputDirName)
	contentDir := filepath.Join(projectDir, config.Content.Dir)

	// Step 1: Clean output directory
	if _, err := os.Stat(outputDir); err == nil {
		os.RemoveAll(outputDir)
	}
	os.MkdirAll(outputDir, 0o755)

	// Step 2: Set up renderer
	md := NewMarkdownRenderer()
	env, err := LoadTheme(config.Theme.Name, "", themesFS)
	if err != nil {
		return fmt.Errorf("failed to load theme: %w", err)
	}

	// Step 3: Determine which pages/collections are private
	privatePageSlugs := make(map[string]bool)
	privateCollections := make(map[string]bool)

	if options.PublishMode {
		for _, item := range config.Nav {
			if item.Private {
				slug := strings.TrimSuffix(item.Path, "/")
				if _, ok := config.Collections[slug]; ok {
					privateCollections[slug] = true
				} else {
					privatePageSlugs[slug] = true
				}
			}
		}
	}

	// Build nav for templates — in publish mode, filter out private items
	navForTemplates := config.Nav
	if options.PublishMode {
		var filtered []NavItem
		for _, item := range config.Nav {
			if !item.Private {
				filtered = append(filtered, item)
			}
		}
		navForTemplates = filtered
	}

	// Step 4: Discover and render pages
	pages := DiscoverPages(contentDir)
	if options.PublishMode {
		var filtered []Page
		for _, p := range pages {
			if !privatePageSlugs[p.Slug] {
				filtered = append(filtered, p)
			}
		}
		pages = filtered
	}

	// Compute base_path from site.url for GitHub Pages subpath support
	basePath := ""
	if options.NoBasePath {
		// Explicitly disabled (e.g. workbench publish preview)
	} else if options.BasePath != "" {
		basePath = options.BasePath
	} else if options.PublishMode {
		basePath = extractBasePath(config.Site.URL)
	}

	siteCtx := pongo2.Context{
		"site":      siteToMap(config.Site),
		"nav":       navToList(navForTemplates),
		"config":    configToMap(config),
		"base_path": basePath,
	}

	for _, page := range pages {
		result := RenderMarkdown(md, page.ContentMarkdown)
		ctx := mergePongoCtx(siteCtx, pongo2.Context{
			"page":    pageToMap(page),
			"content": result.HTML,
			"toc":     result.TOC,
		})

		rendered, err := env.RenderTemplate("page.html", ctx)
		if err != nil {
			return fmt.Errorf("render page '%s': %w", page.Slug, err)
		}

		if page.Slug == "" {
			os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(rendered), 0o644)
		} else {
			pageDir := filepath.Join(outputDir, page.Slug)
			os.MkdirAll(pageDir, 0o755)
			os.WriteFile(filepath.Join(pageDir, "index.html"), []byte(rendered), 0o644)
		}
	}

	// Step 5: Process each collection (skip private ones in publish mode)
	for collName, collConfig := range config.Collections {
		if options.PublishMode && privateCollections[collName] {
			continue
		}
		if err := buildCollection(collName, collConfig, contentDir, outputDir, md, env, siteCtx, basePath); err != nil {
			return fmt.Errorf("collection '%s': %w", collName, err)
		}
	}

	// Step 6: Copy static assets from theme
	copyThemeStatic(config.Theme.Name, outputDir, themesFS)

	// Step 7: Write highlight CSS
	cssDir := filepath.Join(outputDir, "static", "css")
	os.MkdirAll(cssDir, 0o755)
	os.WriteFile(filepath.Join(cssDir, "pygments.css"), []byte(GetHighlightCSS()), 0o644)

	// Step 8: Copy user static assets
	userStatic := filepath.Join(projectDir, config.Content.Dir, "static")
	if info, err := os.Stat(userStatic); err == nil && info.IsDir() {
		copyDir(userStatic, filepath.Join(outputDir, "static"))
	}

	// Step 9: Write build ID for live reload
	os.WriteFile(filepath.Join(outputDir, ".opendoc-build-id"), []byte(fmt.Sprintf("%d", time.Now().UnixMilli())), 0o644)

	return nil
}

// ── Collection builder ──────────────────────────────────────

func buildCollection(
	collName string,
	collConfig CollectionConfig,
	contentDir, outputDir string,
	md goldmark.Markdown,
	env *TemplateEnv,
	siteCtx pongo2.Context,
	basePath string,
) error {
	entriesDir := filepath.Join(contentDir, collName)
	isDated := collConfig.Sort != "alphabetical"
	entries := DiscoverEntries(entriesDir, collConfig.Sort, isDated)

	// Filter drafts
	var filtered []Entry
	for _, e := range entries {
		if !e.Draft {
			filtered = append(filtered, e)
		}
	}
	entries = filtered

	collection := CollectionContext{
		Name:       collName,
		Label:      titleCase(strings.ReplaceAll(strings.ReplaceAll(collName, "-", " "), "_", " ")),
		URLPrefix:  basePath + "/" + collName + "/",
		Layout:     collConfig.Layout,
		DateFormat: collConfig.DateFormat,
	}

	allTags := collectTags(entries)

	// Render individual entries
	for _, entry := range entries {
		result := RenderMarkdown(md, entry.ContentMarkdown)
		formattedDate := ""
		if entry.Date != nil {
			formattedDate = Strftime(entry.Date, collConfig.DateFormat)
		}
		readingTime := estimateReadingTime(entry.ContentMarkdown)

		ctx := mergePongoCtx(siteCtx, pongo2.Context{
			"entry":          entryToMap(entry),
			"post":           entryToMap(entry), // backward compat
			"content":        result.HTML,
			"toc":            result.TOC,
			"formatted_date": formattedDate,
			"reading_time":   readingTime,
			"collection":     collectionToMap(collection),
			"all_tags":       allTags,
		})

		rendered, err := env.RenderTemplate("entry.html", ctx)
		if err != nil {
			return fmt.Errorf("render entry '%s': %w", entry.Slug, err)
		}

		entryDir := filepath.Join(outputDir, collName, entry.Slug)
		os.MkdirAll(entryDir, 0o755)
		os.WriteFile(filepath.Join(entryDir, "index.html"), []byte(rendered), 0o644)
	}

	// Render collection index
	if err := renderCollectionIndex(entries, collConfig, collection, allTags, env, siteCtx, outputDir); err != nil {
		return err
	}

	// Render archive (if enabled and dated)
	if collConfig.Archive && isDated {
		renderArchive(entries, collection, env, siteCtx, outputDir)
	}

	// Render tag pages (if enabled)
	if collConfig.Tags && len(allTags) > 0 {
		renderTagPages(allTags, collection, env, siteCtx, outputDir)
	}

	return nil
}

func renderCollectionIndex(
	entries []Entry,
	collConfig CollectionConfig,
	collection CollectionContext,
	allTags map[string][]Entry,
	env *TemplateEnv,
	siteCtx pongo2.Context,
	outputDir string,
) error {
	pageEntries := entries
	if collConfig.ItemsPerPage > 0 && len(entries) > collConfig.ItemsPerPage {
		pageEntries = entries[:collConfig.ItemsPerPage]
	}

	ctx := mergePongoCtx(siteCtx, pongo2.Context{
		"entries":    entriesToListFormatted(pageEntries, collConfig.DateFormat),
		"posts":     entriesToListFormatted(pageEntries, collConfig.DateFormat), // backward compat
		"collection": collectionToMap(collection),
		"all_tags":  allTags,
		"layout":    collConfig.Layout,
	})

	rendered, err := env.RenderTemplate("collection_index.html", ctx)
	if err != nil {
		return err
	}

	collDir := filepath.Join(outputDir, collection.Name)
	os.MkdirAll(collDir, 0o755)
	os.WriteFile(filepath.Join(collDir, "index.html"), []byte(rendered), 0o644)
	return nil
}

func renderArchive(
	entries []Entry,
	collection CollectionContext,
	env *TemplateEnv,
	siteCtx pongo2.Context,
	outputDir string,
) {
	entriesByYear := make(map[int][]Entry)
	for _, entry := range entries {
		if entry.Date != nil {
			year := entry.Date.Year()
			entriesByYear[year] = append(entriesByYear[year], entry)
		}
	}

	// Sort years descending
	var years []int
	for y := range entriesByYear {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	sortedByYear := make(map[string][]map[string]any)
	for _, y := range years {
		sortedByYear[fmt.Sprintf("%d", y)] = entriesToListFormatted(entriesByYear[y], "%b %d")
	}

	ctx := mergePongoCtx(siteCtx, pongo2.Context{
		"entries_by_year": sortedByYear,
		"posts_by_year":  sortedByYear,
		"collection":     collectionToMap(collection),
	})

	rendered, _ := env.RenderTemplate("archive.html", ctx)
	archiveDir := filepath.Join(outputDir, collection.Name, "archive")
	os.MkdirAll(archiveDir, 0o755)
	os.WriteFile(filepath.Join(archiveDir, "index.html"), []byte(rendered), 0o644)
}

func renderTagPages(
	allTags map[string][]Entry,
	collection CollectionContext,
	env *TemplateEnv,
	siteCtx pongo2.Context,
	outputDir string,
) {
	tagsDir := filepath.Join(outputDir, collection.Name, "tags")
	os.MkdirAll(tagsDir, 0o755)

	// Tag index page
	ctx := mergePongoCtx(siteCtx, pongo2.Context{
		"tags":       allTags,
		"collection": collectionToMap(collection),
	})
	rendered, _ := env.RenderTemplate("tags_index.html", ctx)
	os.WriteFile(filepath.Join(tagsDir, "index.html"), []byte(rendered), 0o644)

	// Individual tag pages
	for tag, tagEntries := range allTags {
		tagSlug := strings.ToLower(strings.ReplaceAll(tag, " ", "-"))
		ctx := mergePongoCtx(siteCtx, pongo2.Context{
			"tag":        tag,
			"entries":    entriesToListFormatted(tagEntries, "%b %d, %Y"),
			"posts":     entriesToListFormatted(tagEntries, "%b %d, %Y"),
			"collection": collectionToMap(collection),
		})

		rendered, _ := env.RenderTemplate("tag.html", ctx)
		tagDir := filepath.Join(tagsDir, tagSlug)
		os.MkdirAll(tagDir, 0o755)
		os.WriteFile(filepath.Join(tagDir, "index.html"), []byte(rendered), 0o644)
	}
}

// ── Helpers ─────────────────────────────────────────────────

func estimateReadingTime(text string) int {
	wordCount := len(strings.Fields(text))
	rt := int(math.Ceil(float64(wordCount) / float64(wordsPerMinute)))
	if rt < 1 {
		rt = 1
	}
	return rt
}

func collectTags(entries []Entry) map[string][]Entry {
	tags := make(map[string][]Entry)
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			tags[tag] = append(tags[tag], entry)
		}
	}
	return tags
}

func copyThemeStatic(themeName, outputDir string, themesFS fs.FS) {
	staticDir := filepath.Join("themes", themeName, "static")
	staticDir = filepath.ToSlash(staticDir)

	fs.WalkDir(themesFS, staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		relPath := strings.TrimPrefix(filepath.ToSlash(path), staticDir+"/")
		destPath := filepath.Join(outputDir, "static", relPath)
		os.MkdirAll(filepath.Dir(destPath), 0o755)
		data, _ := fs.ReadFile(themesFS, path)
		os.WriteFile(destPath, data, 0o644)
		return nil
	})
}

func copyDir(src, dst string) {
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dst, relPath)
		os.MkdirAll(filepath.Dir(destPath), 0o755)
		data, _ := os.ReadFile(path)
		os.WriteFile(destPath, data, 0o644)
		return nil
	})
}

// extractBasePath returns the path component from a URL, for GitHub Pages subpath support.
// e.g. "https://user.github.io/repo" → "/repo"
// e.g. "https://example.com" → ""
// e.g. "" → ""
func extractBasePath(siteURL string) string {
	if siteURL == "" {
		return ""
	}
	// Remove protocol
	u := siteURL
	if idx := strings.Index(u, "://"); idx != -1 {
		u = u[idx+3:]
	}
	// Remove host
	if idx := strings.Index(u, "/"); idx != -1 {
		path := u[idx:]
		// Trim trailing slash
		path = strings.TrimRight(path, "/")
		return path
	}
	return ""
}

// ── Pongo2 context converters ───────────────────────────────

func siteToMap(s SiteConfig) map[string]any {
	return map[string]any{
		"name":        s.Name,
		"url":         s.URL,
		"description": s.Description,
		"author":      s.Author,
	}
}

func navToList(nav []NavItem) []map[string]any {
	var list []map[string]any
	for _, item := range nav {
		list = append(list, map[string]any{
			"label":   item.Label,
			"path":    item.Path,
			"private": item.Private,
		})
	}
	return list
}

func configToMap(c *OpenDocConfig) map[string]any {
	return map[string]any{
		"site":  siteToMap(c.Site),
		"theme": map[string]any{"name": c.Theme.Name},
	}
}

func pageToMap(p Page) map[string]any {
	return map[string]any{
		"title": p.Title,
		"slug":  p.Slug,
		"meta":  p.Meta,
	}
}

func entryToMap(e Entry) map[string]any {
	m := map[string]any{
		"title":       e.Title,
		"slug":        e.Slug,
		"tags":        e.Tags,
		"description": e.Description,
		"draft":       e.Draft,
		"meta":        e.Meta,
	}
	if e.Date != nil {
		m["date"] = *e.Date
		m["iso_date"] = Isoformat(e.Date)
	}
	return m
}

// entryToMapFormatted returns an entry map with a pre-formatted date string.
func entryToMapFormatted(e Entry, dateFormat string) map[string]any {
	m := entryToMap(e)
	if e.Date != nil {
		m["formatted_date"] = Strftime(e.Date, dateFormat)
	}
	return m
}

func entriesToList(entries []Entry) []map[string]any {
	var list []map[string]any
	for _, e := range entries {
		list = append(list, entryToMap(e))
	}
	return list
}

func entriesToListFormatted(entries []Entry, dateFormat string) []map[string]any {
	var list []map[string]any
	for _, e := range entries {
		list = append(list, entryToMapFormatted(e, dateFormat))
	}
	return list
}

func collectionToMap(c CollectionContext) map[string]any {
	return map[string]any{
		"name":        c.Name,
		"label":       c.Label,
		"url_prefix":  c.URLPrefix,
		"layout":      c.Layout,
		"date_format": c.DateFormat,
	}
}

func mergePongoCtx(base, overlay pongo2.Context) pongo2.Context {
	merged := pongo2.Context{}
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		merged[k] = v
	}
	return merged
}
