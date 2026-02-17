package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateProject scaffolds a new OpenDoc project, pre-filling from app config.
func CreateProject(name string) (string, error) {
	projectDir := name
	if _, err := os.Stat(projectDir); err == nil {
		return "", fmt.Errorf("directory '%s' already exists", name)
	}

	// Load app config for defaults
	appCfg := LoadAppConfig()
	author := appCfg.Defaults.Author
	theme := appCfg.Defaults.Theme
	if theme == "" {
		theme = "default"
	}

	contentDir := filepath.Join(projectDir, "content")
	writingDir := filepath.Join(contentDir, "writing")
	os.MkdirAll(writingDir, 0o755)

	os.WriteFile(filepath.Join(projectDir, "opendoc.yml"), []byte(opendocYML(name, author, theme)), 0o644)
	os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(indexMD(name)), 0o644)
	os.WriteFile(filepath.Join(contentDir, "about.md"), []byte(aboutMD), 0o644)
	os.WriteFile(filepath.Join(writingDir, "hello-world.md"), []byte(helloWorldMD), 0o644)
	os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(gitignoreContent), 0o644)

	return projectDir, nil
}

// ── Template strings ────────────────────────────────────────

func opendocYML(name, author, theme string) string {
	return fmt.Sprintf(`site:
  name: "%s"
  url: "https://example.com"
  description: "A new site built with OpenDoc"
  author: "%s"

content:
  dir: "content"

build:
  output_dir: "dist"

collections:
  writing:
    sort: "newest_first"
    date_format: "%%B %%d, %%Y"
    items_per_page: 10
    tags: true
    archive: true
    layout: "timeline"

nav:
  - Home: index.md
  - About: about.md
  - Writing: writing/

theme:
  name: "%s"
`, name, author, theme)
}

func indexMD(name string) string {
	return fmt.Sprintf(`---
title: "Home"
---

# Welcome to %s

This is your new site, built with [OpenDoc](https://github.com/cottrellashley/bark).

Check out the [writing](/writing/) or read [about](/about/) this site.
`, name)
}

const aboutMD = `---
title: "About"
---

# About

This site is powered by **OpenDoc**, a static site generator.

Write your content in markdown, configure with YAML, and build elegant static sites.
`

const helloWorldMD = `---
title: "Hello World"
date: 2026-02-14
tags: [getting-started, opendoc]
description: "Your first post with OpenDoc."
---

# Hello World

Welcome to your first OpenDoc post!

## Writing Content

Posts are just markdown files with YAML frontmatter. Put them in ` + "`content/writing/`" + ` and OpenDoc
will take care of the rest.

### Code highlighting

` + "```python\ndef hello():\n    print(\"Hello from OpenDoc!\")\n```" + `

### Lists

- Write in markdown
- Configure with YAML
- Build with ` + "`opendoc build`" + `
- Serve with ` + "`opendoc serve`" + `

Happy writing!
`

const gitignoreContent = `dist/
node_modules/
.DS_Store
`
