# opendoc

A static site generator with an integrated workbench. Single binary, zero dependencies.

Build sites from markdown, preview them locally, edit in the browser, and publish to GitHub Pages.

## Install

**curl** (Linux / macOS):

```sh
curl -fsSL https://raw.githubusercontent.com/cottrellashley/opendoc/main/install.sh | bash
```

**Homebrew**:

```sh
brew install cottrellashley/tap/opendoc
```

**Go**:

```sh
go install github.com/cottrellashley/opendoc/cmd/opendoc@latest
```

**From source**:

```sh
git clone https://github.com/cottrellashley/opendoc.git
cd opendoc
make build
```

## Quick start

```sh
opendoc new my-site
cd my-site
opendoc serve
```

Open `http://localhost:8000`. Edit markdown files in `content/`, the site rebuilds on save.

## Commands

```
opendoc new <name>                  Scaffold a new project
opendoc build [dir]                 Build the static site
opendoc serve [dir] [-p PORT]       Serve with live reload (default: 8000)
opendoc workbench [dir] [-p PORT]   Start the workbench UI (default: 3000)
opendoc publish [dir] [--repo o/r]  Deploy to GitHub Pages
opendoc status [dir]                Show project health
opendoc config show                 Print global config
opendoc config set <key> <value>    Set a config value
opendoc tui                         Interactive terminal UI
```

All commands default to the current directory if `[dir]` is omitted.

## Project structure

```
my-site/
├── opendoc.yml          # Site configuration
├── settings.json        # Workbench settings (auto-generated)
└── content/
    ├── index.md         # Homepage
    ├── about.md         # Pages
    └── posts/           # Collections
        └── first.md
```

## Configuration

`opendoc.yml`:

```yaml
site:
  name: My Site
  author: Jane Doe
  url: https://janedoe.github.io/my-site

content:
  dir: content

build:
  output_dir: dist

theme:
  name: default

nav:
  - Home: index.md
  - About: about.md
  - Posts: posts/
```

### Collections

Directories under `content/` become collections. Configure them:

```yaml
collections:
  posts:
    sort: date_desc
    tags: true
    items_per_page: 10
```

### Private pages

Append `?` to a nav entry to mark it private. Private pages are excluded from `opendoc publish` builds:

```yaml
nav:
  - Drafts: drafts/?
```

## Global config

Stored at `~/.config/opendoc/config.yml`. Set defaults:

```sh
opendoc config set github.default_account myuser
opendoc config set defaults.author "Jane Doe"
opendoc config set defaults.port 4000
```

## Workbench

`opendoc workbench` starts a browser-based environment with:

- File explorer and markdown editor
- Live site preview
- Integrated terminal
- Publishing controls

## Publishing

Requires the [GitHub CLI](https://cli.github.com/) (`gh`):

```sh
opendoc publish --repo owner/repo
```

Builds the site in publish mode and pushes to the `gh-pages` branch.

## Markdown features

- GitHub Flavoured Markdown
- KaTeX equations (`$inline$` and `$$display$$`)
- Margin notes (`{>> note <<}`)
- Tabbed code blocks
- Syntax highlighting (Chroma)

## Docker

```sh
docker compose up -d
# http://localhost:3000
```

## Development

```sh
make build          # Build binary
make test           # Run tests
make lint           # go vet
make serve          # Serve example site
make workbench      # Start workbench for example site
```

## License

MIT
