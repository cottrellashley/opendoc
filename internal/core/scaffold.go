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
	os.WriteFile(filepath.Join(contentDir, "features.md"), []byte(featuresMD), 0o644)
	os.WriteFile(filepath.Join(contentDir, "settings-guide.md"), []byte(settingsGuideMD), 0o644)
	os.WriteFile(filepath.Join(writingDir, "getting-started.md"), []byte(gettingStartedMD), 0o644)
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
  - Features: features.md
  - Settings Guide: settings-guide.md
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

Your site is ready. Everything you see here is a **markdown file** — and you have an **AI assistant** built right in to help you build it.

---

<div style="background: linear-gradient(135deg, #1a1a2e 0%%, #16213e 50%%, #0f3460 100%%); padding: 2rem 2.5rem; border-radius: 16px; margin: 2rem 0; border: 1px solid rgba(121, 165, 242, 0.3);">
  <h2 style="margin: 0 0 0.75rem; color: #79a5f2; font-size: 1.4rem;">Try the AI Chatbot</h2>
  <p style="color: #e94560; margin: 0 0 0.75rem; font-size: 0.9rem; font-weight: 500;">First, configure a provider in <a href="/settings-guide/" style="color: #79a5f2; text-decoration: underline;">Settings</a> (gear icon, top-right) — connect GitHub Copilot or add an API key.</p>
  <p style="color: #ccc; margin: 0 0 1.25rem; line-height: 1.6;">
    Then open the <strong style="color: #fff;">chat panel</strong> (top-right ` + "`" + `&lt;` + "`" + ` button). The AI can create pages, write content, edit files, update your site config, and rebuild — all through conversation.
  </p>
  <div style="display: flex; flex-direction: column; gap: 0.5rem;">
    <code style="background: rgba(255,255,255,0.08); padding: 0.5rem 0.75rem; border-radius: 8px; color: #e2e8f0; font-size: 0.9rem; border: 1px solid rgba(255,255,255,0.1);">&#x1f4ac; "Add a new page called Projects"</code>
    <code style="background: rgba(255,255,255,0.08); padding: 0.5rem 0.75rem; border-radius: 8px; color: #e2e8f0; font-size: 0.9rem; border: 1px solid rgba(255,255,255,0.1);">&#x1f4ac; "Write a blog post about quantum physics with equations"</code>
    <code style="background: rgba(255,255,255,0.08); padding: 0.5rem 0.75rem; border-radius: 8px; color: #e2e8f0; font-size: 0.9rem; border: 1px solid rgba(255,255,255,0.1);">&#x1f4ac; "Create a recipe collection and add my pasta recipe"</code>
    <code style="background: rgba(255,255,255,0.08); padding: 0.5rem 0.75rem; border-radius: 8px; color: #e2e8f0; font-size: 0.9rem; border: 1px solid rgba(255,255,255,0.1);">&#x1f4ac; "What pages does my site have?"</code>
  </div>
  <p style="color: #888; margin: 1rem 0 0; font-size: 0.85rem;">
    The chatbot reads and writes your files, updates <code style="background: rgba(255,255,255,0.08); padding: 2px 6px; border-radius: 4px; color: #aaa;">opendoc.yml</code>, and triggers rebuilds automatically.
  </p>
</div>

---

## How it works

| Step | What you do | What happens |
|------|------------|--------------|
| 1 | Write markdown files in ` + "`" + `content/` + "`" + ` | Each ` + "`" + `.md` + "`" + ` file becomes a page |
| 2 | Configure ` + "`" + `opendoc.yml` + "`" + ` | Controls navigation, collections, and site settings |
| 3 | Save | The preview updates instantly |

Or skip all that and just **tell the chatbot what you want**. It handles the files, config, and builds for you.

## You can also edit files directly

Every page is a markdown file. This homepage is ` + "`" + `content/index.md` + "`" + `. Open it in any text editor, make changes, and the preview updates. The chatbot and manual editing work together — use whichever is faster for the task.

## Explore

- [**Features**](/features/) — equations, code blocks, embedded HTML/JS, tabs, theorems, and margin notes
- [**Settings Guide**](/settings-guide/) — how to connect GitHub Copilot, add API keys, and configure providers
- [**Writing**](/writing/) — your first collection, ready for blog posts, notes, or anything

Happy building!
`, name)
}

const featuresMD = `---
title: "Features"
---

# What your markdown can do

OpenDoc renders standard markdown, but it also supports **equations**, **syntax-highlighted code**, **embedded HTML and JavaScript**, **tabbed content**, **theorem blocks**, and **margin notes**. This page is a live demo of all of them.

---

## Equations and LaTeX

Write inline math like $E = mc^2$ or $\nabla \times \mathbf{B} = \mu_0 \mathbf{J}$ directly in your text.

For display equations, use $$ on their own lines:

$$
\int_0^\infty e^{-x^2}\, dx = \frac{\sqrt{\pi}}{2}
$$

LaTeX environments work too:

\begin{align}
\nabla \cdot \mathbf{E} &= \frac{\rho}{\epsilon_0} \\
\nabla \cdot \mathbf{B} &= 0 \\
\nabla \times \mathbf{E} &= -\frac{\partial \mathbf{B}}{\partial t} \\
\nabla \times \mathbf{B} &= \mu_0 \mathbf{J} + \mu_0 \epsilon_0 \frac{\partial \mathbf{E}}{\partial t}
\end{align}

You can even number equations for reference:

$$
F = ma
$${#eq:newton}

---

## Theorem blocks

:::theorem Pythagorean Theorem
For a right triangle with legs $a$, $b$ and hypotenuse $c$:

$$a^2 + b^2 = c^2$$
:::

:::definition Continuous Function
A function $f: \mathbb{R} \to \mathbb{R}$ is **continuous** at a point $a$ if:

$$\lim_{x \to a} f(x) = f(a)$$
:::

:::proof
Let $\epsilon > 0$. Choose $\delta = \epsilon / M$ where $M$ is the Lipschitz constant. Then for $|x - a| < \delta$ we have $|f(x) - f(a)| \leq M|x - a| < \epsilon$. $\square$
:::

Supported types: **theorem**, **definition**, **lemma**, **proposition**, **corollary**, **remark**, **proof**. Each type is auto-numbered independently.

---

## Code with syntax highlighting

` + "```python" + `
def fibonacci(n: int) -> list[int]:
    """Generate the first n Fibonacci numbers."""
    seq = [0, 1]
    for _ in range(n - 2):
        seq.append(seq[-1] + seq[-2])
    return seq[:n]

print(fibonacci(10))  # [0, 1, 1, 2, 3, 5, 8, 13, 21, 34]
` + "```" + `

` + "```javascript" + `
const debounce = (fn, ms) => {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), ms);
  };
};
` + "```" + `

` + "```sql" + `
SELECT users.name, COUNT(posts.id) AS post_count
FROM users
LEFT JOIN posts ON posts.author_id = users.id
GROUP BY users.id
ORDER BY post_count DESC
LIMIT 10;
` + "```" + `

All common languages are supported: Python, JavaScript, TypeScript, Go, Rust, SQL, HTML, CSS, YAML, and many more.

---

## Tabs

:::tabs
=== Python
` + "```python" + `
for i in range(5):
    print(f"Hello {i}")
` + "```" + `
=== JavaScript
` + "```javascript" + `
for (let i = 0; i < 5; i++) {
  console.log(` + "`" + `Hello ${i}` + "`" + `);
}
` + "```" + `
=== Go
` + "```go" + `
for i := 0; i < 5; i++ {
    fmt.Printf("Hello %d\n", i)
}
` + "```" + `
:::

Content inside each tab is full markdown — code blocks, lists, images, everything works.

---

## Margin notes

:::sidenote Why margin notes?
Margin notes keep supplementary information visible without interrupting the main flow. They are inspired by Edward Tufte's book designs.
:::

Margin notes appear alongside your text. They are perfect for definitions, historical context, or tangential thoughts that would clutter the main body.

:::deepdive How OpenDoc renders this
The ` + "`" + `:::sidenote` + "`" + ` syntax is preprocessed before the markdown is converted to HTML. The inner content is rendered as full markdown, so you can include **bold**, *italic*, code, and even math like $e^{i\pi} + 1 = 0$ inside a note.
:::

Variants: **sidenote** (general), **deepdive** (extended detail), **aside** (tangential), **widget** (interactive).

---

## Embedded HTML and JavaScript

Raw HTML passes straight through. Build anything you can build on the web:

<div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 2rem; border-radius: 12px; color: white; text-align: center; margin: 1.5rem 0;">
  <h3 style="margin: 0 0 0.5rem; color: white;">Custom styled section</h3>
  <p style="margin: 0; opacity: 0.9;">This is raw HTML with inline CSS — gradients, rounded corners, anything goes.</p>
</div>

You can embed interactive JavaScript too:

<div id="counter-demo" style="display: flex; align-items: center; gap: 1rem; padding: 1rem; border: 1px solid #333; border-radius: 8px; margin: 1.5rem 0;">
  <button onclick="document.getElementById('count').textContent = parseInt(document.getElementById('count').textContent) - 1" style="padding: 0.5rem 1rem; border-radius: 6px; border: 1px solid #555; background: #2a2a3a; color: #fff; cursor: pointer; font-size: 1.1rem;">−</button>
  <span id="count" style="font-size: 1.5rem; font-weight: bold; min-width: 3rem; text-align: center;">0</span>
  <button onclick="document.getElementById('count').textContent = parseInt(document.getElementById('count').textContent) + 1" style="padding: 0.5rem 1rem; border-radius: 6px; border: 1px solid #555; background: #2a2a3a; color: #fff; cursor: pointer; font-size: 1.1rem;">+</button>
  <span style="opacity: 0.6; font-size: 0.85rem;">← try clicking these</span>
</div>

This means you can embed **charts**, **calculators**, **interactive diagrams**, **videos**, **iframes** — anything that works in HTML.

---

## Tables, task lists, and more

| Feature         | Syntax                  | Supported |
|-----------------|-------------------------|-----------|
| Bold            | ` + "`" + `**text**` + "`" + `             | Yes       |
| Italic          | ` + "`" + `*text*` + "`" + `               | Yes       |
| Strikethrough   | ` + "`" + `~~text~~` + "`" + `             | Yes       |
| Task lists      | ` + "`" + `- [x] done` + "`" + `           | Yes       |
| Auto-link URLs  | just paste the URL      | Yes       |
| Heading anchors | auto-generated          | Yes       |
| Smart quotes    | ` + "`" + `"text"` + "`" + ` → "text"     | Yes       |

- [x] Markdown basics
- [x] Equations and LaTeX
- [x] Syntax-highlighted code
- [x] Embedded HTML and JavaScript
- [x] Tabs, theorems, and margin notes
- [ ] Your content here!

---

## Edit this page

This file lives at ` + "`" + `content/features.md` + "`" + `. Open it in your editor, or ask the chatbot:

> "Edit the features page and add a section about my project"

Every page on your site works the same way — it is just a markdown file.
`

const settingsGuideMD = `---
title: "Settings Guide"
---

# Settings Guide

The workbench settings panel lets you connect AI providers, manage API keys, and configure the chatbot. Click the **gear icon** in the top-right corner of the workbench to open it.

---

## AI Providers

The chatbot supports three AI providers. You can switch between them at any time using the provider buttons above the chat input.

| Provider | How to connect | Models available |
|----------|---------------|-----------------|
| **GitHub Copilot** | Sign in with GitHub (SSO) | GPT-4.1, Claude Sonnet 4, GPT-4o, Gemini 2.5 Pro |
| **Anthropic** | Paste API key | Claude Sonnet 4, Claude Haiku 3.5 |
| **OpenAI** | Paste API key | GPT-4o, GPT-4o Mini, o3 Mini |

---

## GitHub Copilot (recommended)

If you have a **GitHub Copilot subscription** (individual, business, or enterprise), you can use it directly — no API key needed.

### How to connect

1. Open **Settings** (gear icon, top-right)
2. Find the **GitHub Copilot** card
3. Click **Sign in with GitHub**
4. You will see a **device code** and a link to GitHub
5. Click the link (or go to [github.com/login/device](https://github.com/login/device))
6. Paste the code and authorize
7. The workbench will detect the authorization automatically

Once connected, select **GitHub Copilot** from the provider buttons in the chat panel. You can choose from multiple models including GPT-4.1, Claude Sonnet 4, GPT-4o, and Gemini 2.5 Pro.

:::sidenote What is the device code flow?
This is the same secure OAuth flow used by GitHub CLI, VS Code, and other developer tools. OpenDoc never sees your GitHub password — it only receives a scoped token for Copilot access.
:::

### Disconnecting

To disconnect Copilot, go to Settings and click **Disconnect** on the Copilot card. You can also revoke access from your [GitHub settings](https://github.com/settings/apps/authorizations).

---

## Anthropic (Claude)

To use Claude models directly:

1. Go to [console.anthropic.com](https://console.anthropic.com/) and create an account
2. Navigate to **API Keys** and create a new key
3. Open **Settings** in the workbench
4. Paste your API key in the **Anthropic** field and click **Save**

Alternatively, set the environment variable before starting the workbench:

` + "```bash" + `
export ANTHROPIC_API_KEY="sk-ant-..."
opendoc workbench
` + "```" + `

---

## OpenAI (GPT)

To use GPT models directly:

1. Go to [platform.openai.com](https://platform.openai.com/) and create an account
2. Navigate to **API Keys** and create a new key
3. Open **Settings** in the workbench
4. Paste your API key in the **OpenAI** field and click **Save**

Or set the environment variable:

` + "```bash" + `
export OPENAI_API_KEY="sk-..."
opendoc workbench
` + "```" + `

---

## Switching providers and models

Once you have at least one provider connected, you will see **provider buttons** above the chat input:

- Click a provider name to switch (e.g. **GitHub Copilot**, **Claude**, **GPT**)
- Use the **model dropdown** next to the buttons to pick a specific model
- Switching providers starts a new chat session

:::sidenote Which provider should I use?
**GitHub Copilot** is the easiest to set up if you already have a subscription — no API keys, no billing. It also gives you access to models from multiple vendors (OpenAI, Anthropic, Google) through a single connection. If you do not have Copilot, Anthropic's Claude and OpenAI's GPT are both excellent choices.
:::

---

## API key storage

API keys are stored locally in ` + "`" + `~/.config/opendoc/secrets.yml` + "`" + ` on your machine. They are **never sent anywhere** except to the provider's own API endpoint.

| Method | Location | Priority |
|--------|----------|----------|
| Environment variable | ` + "`" + `ANTHROPIC_API_KEY` + "`" + `, ` + "`" + `OPENAI_API_KEY` + "`" + ` | Highest |
| Settings file | ` + "`" + `~/.config/opendoc/secrets.yml` + "`" + ` | Fallback |
| GitHub Copilot | OAuth token (auto-managed) | Separate flow |

Environment variables take priority over the settings file, so you can override keys per-session without changing your saved config.

---

## Troubleshooting

**"API key not set" error when chatting**
- Open Settings and check that your key is saved, or that Copilot shows as connected
- If you just connected Copilot, try refreshing the page

**Copilot sign-in stuck on "Waiting for authorization"**
- Make sure you completed the device code flow on GitHub
- Check that your GitHub account has an active Copilot subscription
- Try disconnecting and signing in again

**Want to use a different model?**
- Use the model dropdown next to the provider buttons in the chat panel
- Or set an environment variable: ` + "`" + `ANTHROPIC_MODEL` + "`" + `, ` + "`" + `OPENAI_MODEL` + "`" + `, or ` + "`" + `COPILOT_MODEL` + "`" + `
`

const gettingStartedMD = `---
title: "Getting Started with OpenDoc"
date: 2026-02-18
tags: [guide, opendoc, getting-started]
description: "Learn how to create content, use the AI chatbot, and configure your site."
---

# Getting Started with OpenDoc

Everything on your site is a **markdown file**. You can edit them directly in any text editor, or use the **AI chatbot** built into the workbench to create and modify content for you.

## Two ways to edit

### 1. Edit files directly

Your content lives in the ` + "`" + `content/` + "`" + ` folder. Each ` + "`" + `.md` + "`" + ` file becomes a page:

- ` + "`" + `content/index.md` + "`" + ` → your homepage
- ` + "`" + `content/features.md` + "`" + ` → the features page
- ` + "`" + `content/writing/this-post.md` + "`" + ` → this post

Open any file, make changes, and the preview updates automatically.

### 2. Ask the chatbot

The AI chatbot (bottom-right panel in the workbench) can do anything you can do with files — but faster. Try:

- *"Create a new page called Projects with a grid of my work"*
- *"Add a recipe collection to my site"*
- *"Write a blog post about quantum mechanics with equations"*
- *"What files are in my content folder?"*
- *"Change the site name to My Portfolio"*

The chatbot reads and writes files, updates ` + "`" + `opendoc.yml` + "`" + `, and triggers rebuilds — all through natural conversation.

## How files become pages

Every markdown file starts with **frontmatter** — a YAML header between ` + "`" + `---` + "`" + ` markers:

` + "```yaml" + `
---
title: "My Page Title"
date: 2026-02-18
tags: [guide, tutorial]
description: "A short summary shown in listings."
---

Your content starts here...
` + "```" + `

**Frontmatter fields:**

| Field         | Required? | Purpose                                |
|---------------|-----------|----------------------------------------|
| ` + "`" + `title` + "`" + `       | Optional  | Page title (defaults to filename)       |
| ` + "`" + `date` + "`" + `        | For dated collections | Sort order (YYYY-MM-DD)   |
| ` + "`" + `tags` + "`" + `        | Optional  | Tags for categorisation                 |
| ` + "`" + `description` + "`" + ` | Optional  | Summary shown in collection listings    |
| ` + "`" + `draft` + "`" + `       | Optional  | Set ` + "`" + `true` + "`" + ` to hide from builds          |

## Collections

A **collection** is any folder inside ` + "`" + `content/` + "`" + ` that holds related entries. This "writing" section is a collection. You can have as many as you want:

- ` + "`" + `content/writing/` + "`" + ` → blog posts, essays, notes
- ` + "`" + `content/recipes/` + "`" + ` → your recipe book
- ` + "`" + `content/projects/` + "`" + ` → project showcases
- ` + "`" + `content/photos/` + "`" + ` → photo journal

To create a new collection, just:

1. Create a folder: ` + "`" + `content/recipes/` + "`" + `
2. Add it to ` + "`" + `opendoc.yml` + "`" + ` under ` + "`" + `collections:` + "`" + ` and ` + "`" + `nav:` + "`" + `
3. Add markdown files inside it

Or just ask the chatbot: *"Add a recipes collection to my site"* — it handles all three steps.

## Configuration

Your site is configured in ` + "`" + `opendoc.yml` + "`" + ` at the project root:

` + "```yaml" + `
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
  - Settings Guide: settings-guide.md
  - Writing: writing/
` + "```" + `

The nav section controls what appears in your site's navigation bar. Pages are referenced by their filename, and collections by their directory with a trailing ` + "`" + `/` + "`" + `.

## What's next?

- **Explore the [features page](/features/)** to see equations, code, embedded HTML, tabs, and more
- **Read the [settings guide](/settings-guide/)** to connect GitHub Copilot or add API keys
- **Ask the chatbot** to create your first custom page
- **Edit this post** — it is at ` + "`" + `content/writing/getting-started.md` + "`" + `

Your site can become anything: a blog, a wiki, a portfolio, a recipe book, a research notebook, a personal knowledge base. Start building.
`

const gitignoreContent = `dist/
node_modules/
.DS_Store
`
