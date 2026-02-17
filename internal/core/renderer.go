package core

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/cottrellashley/opendoc/internal/core/extensions"
)

// ── Date formatting (Python strftime compat) ────────────────

var monthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}
var monthAbbr = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

// Strftime formats a date using Python-style format codes.
func Strftime(t *time.Time, format string) string {
	if t == nil {
		return ""
	}
	d := *t
	s := format
	s = strings.ReplaceAll(s, "%B", monthNames[d.Month()-1])
	s = strings.ReplaceAll(s, "%b", monthAbbr[d.Month()-1])
	s = strings.ReplaceAll(s, "%d", fmt.Sprintf("%02d", d.Day()))
	s = strings.ReplaceAll(s, "%m", fmt.Sprintf("%02d", int(d.Month())))
	s = strings.ReplaceAll(s, "%Y", fmt.Sprintf("%d", d.Year()))
	s = strings.ReplaceAll(s, "%y", fmt.Sprintf("%02d", d.Year()%100))
	return s
}

// Isoformat returns the ISO 8601 date string.
func Isoformat(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// ── Markdown renderer ───────────────────────────────────────

// NewMarkdownRenderer creates a configured goldmark markdown renderer.
func NewMarkdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify,
			extension.Typographer,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML passthrough
		),
	)
}

// RenderResult holds the rendered HTML and TOC.
type RenderResult struct {
	HTML string
	TOC  string
}

// RenderMarkdown preprocesses and renders markdown to HTML.
func RenderMarkdown(md goldmark.Markdown, source string) RenderResult {
	// Apply preprocessors in order: math → tabs → sidenotes
	processed := source
	processed = extensions.PreprocessMath(processed)
	processed = extensions.PreprocessTabs(processed)
	processed = extensions.PreprocessSidenotes(processed)

	// Render to HTML
	var buf bytes.Buffer
	if err := md.Convert([]byte(processed), &buf); err != nil {
		return RenderResult{HTML: processed}
	}
	htmlStr := buf.String()

	// Generate TOC
	toc := generateTOC(htmlStr)

	return RenderResult{HTML: htmlStr, TOC: toc}
}

// ── TOC generator ───────────────────────────────────────────

var headingRe = regexp.MustCompile(`<h([23])\s*(?:id="([^"]*)")?\s*[^>]*>(.*?)</h[23]>`)
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func generateTOC(htmlStr string) string {
	matches := headingRe.FindAllStringSubmatch(htmlStr, -1)
	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("<div class=\"toc\">\n<ul>\n")

	for _, m := range matches {
		level := m[1]
		id := m[2]
		text := htmlTagRe.ReplaceAllString(m[3], "")
		text = strings.TrimSpace(text)

		if id == "" {
			// Generate ID from text
			id = strings.ToLower(text)
			id = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(id, "")
			id = regexp.MustCompile(`\s+`).ReplaceAllString(id, "-")
		}

		if level == "2" {
			b.WriteString(fmt.Sprintf("<li><a href=\"#%s\">%s</a></li>\n", id, text))
		} else {
			b.WriteString(fmt.Sprintf("<li class=\"toc-h3\"><a href=\"#%s\">%s</a></li>\n", id, text))
		}
	}

	b.WriteString("</ul>\n</div>")
	return b.String()
}

// ── Template environment ────────────────────────────────────

// TemplateEnv wraps a pongo2 template set for rendering.
type TemplateEnv struct {
	set *pongo2.TemplateSet
}

// LoadTheme creates a template environment from the given theme name.
// It checks for a custom theme directory first, then falls back to the embedded themes.
func LoadTheme(themeName string, customThemeDir string, themesFS fs.FS) (*TemplateEnv, error) {
	// Try custom theme directory first
	if customThemeDir != "" {
		if info, err := os.Stat(customThemeDir); err == nil && info.IsDir() {
			loader := pongo2.MustNewLocalFileSystemLoader(customThemeDir)
			set := pongo2.NewSet("custom", loader)
			registerFilters(set)
			return &TemplateEnv{set: set}, nil
		}
	}

	// Use embedded themes FS
	themeDir := filepath.Join("themes", themeName)

	// Create a loader from the embedded FS
	loader, err := newEmbedLoader(themesFS, themeDir)
	if err != nil {
		return nil, fmt.Errorf("theme '%s' not found", themeName)
	}

	set := pongo2.NewSet("embedded", loader)
	registerFilters(set)
	return &TemplateEnv{set: set}, nil
}

// RenderTemplate renders a named template with the given context.
func (env *TemplateEnv) RenderTemplate(name string, ctx pongo2.Context) (string, error) {
	tpl, err := env.set.FromFile(name)
	if err != nil {
		return "", fmt.Errorf("template '%s': %w", name, err)
	}
	return tpl.Execute(ctx)
}

// ── Pongo2 custom filters ───────────────────────────────────

var filtersRegistered bool

func registerFilters(set *pongo2.TemplateSet) {
	if filtersRegistered {
		return
	}
	filtersRegistered = true

	// Register global filters
	pongo2.RegisterFilter("strftime", filterStrftime)
	pongo2.RegisterFilter("isoformat", filterIsoformat)
	pongo2.RegisterFilter("replace", filterReplace)
	pongo2.RegisterFilter("slugify", filterSlugify)
}

func filterStrftime(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	format := param.String()
	if format == "" {
		format = "%B %d, %Y"
	}

	var t *time.Time
	switch v := in.Interface().(type) {
	case time.Time:
		t = &v
	case *time.Time:
		t = v
	default:
		return pongo2.AsValue(""), nil
	}

	return pongo2.AsValue(Strftime(t, format)), nil
}

func filterIsoformat(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	var t *time.Time
	switch v := in.Interface().(type) {
	case time.Time:
		t = &v
	case *time.Time:
		t = v
	default:
		return pongo2.AsValue(""), nil
	}

	return pongo2.AsValue(Isoformat(t)), nil
}

func filterSlugify(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := strings.ToLower(in.String())
	s = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return pongo2.AsValue(s), nil
}

func filterReplace(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	// param format: "old,new" — split on first comma
	paramStr := param.String()
	parts := strings.SplitN(paramStr, ",", 2)
	if len(parts) != 2 {
		return in, nil
	}
	old := strings.Trim(parts[0], "' \"")
	new := strings.Trim(parts[1], "' \"")
	return pongo2.AsValue(strings.ReplaceAll(in.String(), old, new)), nil
}

// ── Embedded FS template loader ─────────────────────────────

type embedLoader struct {
	fs      fs.FS
	baseDir string
}

func newEmbedLoader(fsys fs.FS, baseDir string) (*embedLoader, error) {
	// Verify the directory exists in the FS
	_, err := fs.Stat(fsys, baseDir)
	if err != nil {
		return nil, err
	}
	return &embedLoader{fs: fsys, baseDir: baseDir}, nil
}

func (l *embedLoader) Abs(base, name string) string {
	if filepath.IsAbs(name) || base == "" {
		return name
	}
	return name
}

func (l *embedLoader) Get(path string) (io.Reader, error) {
	fullPath := filepath.Join(l.baseDir, path)
	fullPath = filepath.ToSlash(fullPath) // Ensure forward slashes for embed.FS
	f, err := l.fs.Open(fullPath)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ── Highlight CSS ───────────────────────────────────────────

// GetHighlightCSS returns the monokai-inspired CSS for syntax highlighting.
func GetHighlightCSS() string {
	return `/* highlight.js — Monokai theme */
.highlight pre { margin: 0; padding: 0; }
.highlight code.hljs { display: block; overflow-x: auto; padding: 1em; }
.hljs { color: #f8f8f2; background: #272822; }
.hljs-tag, .hljs-subst { color: #f8f8f2; }
.hljs-emphasis { font-style: italic; }
.hljs-strong { font-weight: bold; }
.hljs-bullet, .hljs-quote, .hljs-regexp, .hljs-literal, .hljs-link { color: #ae81ff; }
.hljs-number, .hljs-doctag { color: #ae81ff; }
.hljs-code, .hljs-title, .hljs-section, .hljs-selector-class { color: #a6e22e; }
.hljs-title.class_.inherited__ { color: #a6e22e; }
.hljs-keyword, .hljs-selector-tag, .hljs-name { color: #f92672; }
.hljs-attr, .hljs-attribute { color: #a6e22e; }
.hljs-symbol, .hljs-variable, .hljs-template-variable, .hljs-template-tag { color: #66d9ef; }
.hljs-params { color: #f8f8f2; }
.hljs-string, .hljs-type, .hljs-built_in, .hljs-selector-id, .hljs-selector-attr, .hljs-selector-pseudo, .hljs-addition { color: #e6db74; }
.hljs-comment, .hljs-deletion, .hljs-meta { color: #75715e; }
`
}
