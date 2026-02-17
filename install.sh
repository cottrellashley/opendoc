#!/bin/sh
# OpenDoc installer — curl -fsSL https://raw.githubusercontent.com/cottrellashley/opendoc/main/install.sh | bash
#
# Detects OS/arch, downloads the latest release from GitHub, and installs
# the opendoc binary to /usr/local/bin (or ~/.local/bin if not writable).

set -e

REPO="cottrellashley/opendoc"
BINARY="opendoc"

# ── Helpers ────────────────────────────────────────────────

info()  { printf "\033[0;34m[info]\033[0m  %s\n" "$1"; }
ok()    { printf "\033[0;32m[ok]\033[0m    %s\n" "$1"; }
err()   { printf "\033[0;31m[error]\033[0m %s\n" "$1" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || err "Required tool '$1' not found. Please install it first."
}

# ── Detect platform ───────────────────────────────────────

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) err "Unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *) err "Unsupported architecture: $(uname -m)" ;;
  esac
}

# ── Main ──────────────────────────────────────────────────

need curl
need tar

OS="$(detect_os)"
ARCH="$(detect_arch)"

# Windows arm64 is not supported
if [ "$OS" = "windows" ] && [ "$ARCH" = "arm64" ]; then
  err "Windows arm64 is not supported. Please use amd64."
fi

info "Detected platform: ${OS}/${ARCH}"

# Fetch latest release tag
info "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')

if [ -z "$LATEST" ]; then
  err "Could not determine latest release. Check https://github.com/${REPO}/releases"
fi

VERSION="${LATEST#v}"
info "Latest version: ${VERSION}"

# Build download URL
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
  EXT="zip"
fi

FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

# Download
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${URL}..."
curl -fsSL -o "${TMPDIR}/${FILENAME}" "$URL" || err "Download failed. Check that the release exists at ${URL}"

# Extract
info "Extracting..."
if [ "$EXT" = "zip" ]; then
  need unzip
  unzip -qo "${TMPDIR}/${FILENAME}" -d "${TMPDIR}" || err "Extraction failed"
else
  tar xzf "${TMPDIR}/${FILENAME}" -C "${TMPDIR}" || err "Extraction failed"
fi

# Find the binary
BIN_PATH="${TMPDIR}/${BINARY}"
if [ "$OS" = "windows" ]; then
  BIN_PATH="${TMPDIR}/${BINARY}.exe"
fi

if [ ! -f "$BIN_PATH" ]; then
  err "Binary not found after extraction. Contents: $(ls "${TMPDIR}")"
fi

chmod +x "$BIN_PATH"

# Install
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  # Try with sudo
  if command -v sudo >/dev/null 2>&1; then
    info "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "$BIN_PATH" "${INSTALL_DIR}/${BINARY}"
  else
    # Fallback to ~/.local/bin
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    mv "$BIN_PATH" "${INSTALL_DIR}/${BINARY}"
    info "Installed to ${INSTALL_DIR}/${BINARY}"

    # Check if it's in PATH
    case ":$PATH:" in
      *":${INSTALL_DIR}:"*) ;;
      *) printf "\n\033[0;33m[warn]\033[0m  %s is not in your PATH.\n" "$INSTALL_DIR"
         printf "        Add it with: export PATH=\"%s:\$PATH\"\n\n" "$INSTALL_DIR" ;;
    esac
  fi
else
  info "Installing to ${INSTALL_DIR}..."
  mv "$BIN_PATH" "${INSTALL_DIR}/${BINARY}"
fi

# Verify
if command -v "$BINARY" >/dev/null 2>&1; then
  INSTALLED_VERSION=$("$BINARY" --version 2>&1 | head -1)
  ok "Installed: ${INSTALLED_VERSION}"
else
  ok "Installed ${BINARY} to ${INSTALL_DIR}"
  info "You may need to restart your shell or add ${INSTALL_DIR} to your PATH."
fi

printf "\n"
info "Get started:"
printf "  opendoc new my-site\n"
printf "  cd my-site\n"
printf "  opendoc workbench\n"
printf "\n"
