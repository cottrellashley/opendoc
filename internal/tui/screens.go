package tui

import (
	"fmt"
	"strings"
)

// ── Screen identifiers ──────────────────────────────────────

type screen int

const (
	screenHome screen = iota
	screenDocs
	screenCommands
)

// ── Home screen ─────────────────────────────────────────────

func viewHome(width int) string {
	banner := accentStyle.Render(`
   ██████╗ ██████╗ ███████╗███╗   ██╗██████╗  ██████╗  ██████╗
  ██╔═══██╗██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝
  ██║   ██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║██║
  ██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║██║
  ╚██████╔╝██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝╚██████╗
   ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝  ╚═════╝`)

	version := mutedStyle.Render("  v0.1.0 — Static Site Generator")

	desc := normalItemStyle.Render(`
  Welcome to the OpenDoc interactive console.
  Use this terminal to manage your site, browse
  documentation, and run build commands.`)

	nav := "\n" + mutedStyle.Render("  Navigate:") + "\n" +
		accentStyle.Render("  [1]") + normalItemStyle.Render(" Home   ") +
		accentStyle.Render("[2]") + normalItemStyle.Render(" Docs   ") +
		accentStyle.Render("[3]") + normalItemStyle.Render(" Commands") + "\n" +
		mutedStyle.Render("  Press q to quit")

	return banner + "\n" + version + "\n" + desc + "\n" + nav
}

// ── Docs screen ─────────────────────────────────────────────

// DocItem represents a documentation page.
type DocItem struct {
	Title   string
	Path    string
	Summary string
}

// DefaultDocs returns the hardcoded list of doc pages.
func DefaultDocs() []DocItem {
	return []DocItem{
		{"Getting Started", "/getting-started/", "Installation and first site setup"},
		{"Configuration", "/guide/configuration/", "Full opendoc.yml reference"},
		{"Collections", "/guide/collections/", "Organise content into named groups"},
		{"Layouts", "/guide/layouts/", "Timeline, grid, and minimal index pages"},
		{"Code Blocks", "/guide/code-blocks/", "Syntax highlighting, tabs, copy buttons"},
		{"Equations & Math", "/guide/equations/", "LaTeX math, theorem environments"},
		{"Margin Notes", "/guide/margin-notes/", "Tufte-style sidenotes and widgets"},
		{"Themes", "/guide/themes/", "Customise the default theme"},
		{"CLI Reference", "/guide/cli-reference/", "opendoc new, build, serve commands"},
	}
}

func viewDocs(docs []DocItem, cursor int, showDetail bool) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Documentation") + "\n")
	b.WriteString(mutedStyle.Render("  Browse the OpenDoc guide. ↑/↓ to move, Enter for details.\n"))
	b.WriteString("\n")

	for i, d := range docs {
		prefix := "  "
		if i == cursor {
			prefix = accentStyle.Render("▸ ")
			b.WriteString(prefix + selectedStyle.Render(d.Title) + "\n")
			if showDetail {
				b.WriteString(mutedStyle.Render("    "+d.Path) + "\n")
				b.WriteString(normalItemStyle.Render("    "+d.Summary) + "\n")
			}
		} else {
			b.WriteString(prefix + normalItemStyle.Render(d.Title) + "\n")
		}
	}

	b.WriteString("\n" + mutedStyle.Render("  Esc to collapse · q to quit"))
	return b.String()
}

// ── Commands screen ─────────────────────────────────────────

// Command represents an executable OpenDoc command.
type Command struct {
	Label string
	Cmd   string
	Desc  string
}

// DefaultCommands returns the whitelisted commands.
func DefaultCommands(allowed []string) []Command {
	all := []Command{
		{"Build Site", "opendoc build /workspace", "Rebuild the static site from content"},
		{"Build Help", "opendoc build --help", "Show build command options"},
		{"Serve Help", "opendoc serve --help", "Show serve command options"},
		{"New Help", "opendoc new --help", "Show scaffold command options"},
	}

	if len(allowed) == 0 {
		return all
	}

	// Filter to only allowed commands.
	allowSet := make(map[string]bool)
	for _, a := range allowed {
		allowSet[strings.TrimSpace(a)] = true
	}

	var filtered []Command
	for _, c := range all {
		if allowSet[c.Cmd] {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return all
	}
	return filtered
}

func viewCommands(cmds []Command, cursor int, output string, running bool) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Commands") + "\n")
	b.WriteString(mutedStyle.Render("  Run OpenDoc commands. ↑/↓ to select, Enter to execute.\n"))
	b.WriteString("\n")

	for i, c := range cmds {
		prefix := "  "
		if i == cursor {
			prefix = accentStyle.Render("▸ ")
			b.WriteString(prefix + selectedStyle.Render(c.Label) + "\n")
			b.WriteString(mutedStyle.Render(fmt.Sprintf("    $ %s", c.Cmd)) + "\n")
			b.WriteString(mutedStyle.Render("    "+c.Desc) + "\n")
		} else {
			b.WriteString(prefix + normalItemStyle.Render(c.Label) + "\n")
		}
	}

	if running {
		b.WriteString("\n" + warningStyle.Render("  ⟳ Running..."))
	} else if output != "" {
		b.WriteString("\n" + mutedStyle.Render("  ── Output ──────────────────") + "\n")
		// Indent and limit output.
		lines := strings.Split(output, "\n")
		maxLines := 15
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		for _, l := range lines {
			b.WriteString("  " + normalItemStyle.Render(l) + "\n")
		}
	}

	b.WriteString("\n" + mutedStyle.Render("  q to quit"))
	return b.String()
}
