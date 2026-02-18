---
title: "Getting Started with OpenDoc"
date: 2026-02-18
tags: [guide, opendoc, getting-started]
description: "Learn how to create content, use the AI chatbot, and configure your site."
---

# Getting Started with OpenDoc

Everything on your site is a **markdown file**. You can edit them directly in any text editor, or use the **AI chatbot** built into the workbench to create and modify content for you.

## Two ways to edit

### 1. Edit files directly

Your content lives in the `content/` folder. Each `.md` file becomes a page:

- `content/index.md` → your homepage
- `content/features.md` → the features page
- `content/writing/this-post.md` → this post

Open any file, make changes, and the preview updates automatically.

### 2. Ask the chatbot

The AI chatbot (bottom-right panel in the workbench) can do anything you can do with files — but faster. Try:

- *"Create a new page called Projects with a grid of my work"*
- *"Add a recipe collection to my site"*
- *"Write a blog post about quantum mechanics with equations"*
- *"What files are in my content folder?"*
- *"Change the site name to My Portfolio"*

The chatbot reads and writes files, updates `opendoc.yml`, and triggers rebuilds — all through natural conversation.

## How files become pages

Every markdown file starts with **frontmatter** — a YAML header between `---` markers:

```yaml
---
title: "My Page Title"
date: 2026-02-18
tags: [guide, tutorial]
description: "A short summary shown in listings."
---

Your content starts here...
```

**Frontmatter fields:**

| Field         | Required? | Purpose                                |
|---------------|-----------|----------------------------------------|
| `title`       | Optional  | Page title (defaults to filename)       |
| `date`        | For dated collections | Sort order (YYYY-MM-DD)   |
| `tags`        | Optional  | Tags for categorisation                 |
| `description` | Optional  | Summary shown in collection listings    |
| `draft`       | Optional  | Set `true` to hide from builds          |

## Collections

A **collection** is any folder inside `content/` that holds related entries. This "writing" section is a collection. You can have as many as you want:

- `content/writing/` → blog posts, essays, notes
- `content/recipes/` → your recipe book
- `content/projects/` → project showcases
- `content/photos/` → photo journal

To create a new collection, just:

1. Create a folder: `content/recipes/`
2. Add it to `opendoc.yml` under `collections:` and `nav:`
3. Add markdown files inside it

Or just ask the chatbot: *"Add a recipes collection to my site"* — it handles all three steps.

## Configuration

Your site is configured in `opendoc.yml` at the project root:

```yaml
site:
  name: "My Site"
  description: "A personal site"
  author: "Your Name"

collections:
  writing:
    sort: "newest_first"
    layout: "timeline"      # timeline, grid, or minimal
    tags: true

nav:
  - Home: index.md
  - Features: features.md
  - Writing: writing/
```

The nav section controls what appears in your site's navigation bar. Pages are referenced by their filename, and collections by their directory with a trailing `/`.

## What's next?

- **Explore the [features page](/features/)** to see equations, code, embedded HTML, tabs, and more
- **Ask the chatbot** to create your first custom page
- **Edit this post** — it is at `content/writing/getting-started.md`

Your site can become anything: a blog, a wiki, a portfolio, a recipe book, a research notebook, a personal knowledge base. Start building.
