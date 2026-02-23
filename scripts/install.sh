#!/bin/sh
set -eu

REPO_OWNER="takubii"
REPO_NAME="git-worktree-opener"
BINARY_NAME="wto"

VERSION="${WTO_VERSION:-}"
INSTALL_DIR="${WTO_INSTALL_DIR:-${HOME}/.local/bin}"
SKIP_CHECKSUM="${WTO_SKIP_CHECKSUM:-0}"

usage() {
  cat <<'EOF'
Install wto from GitHub Releases.

Options:
  --version <tag>      Install a specific tag (for example: v0.1.0)
  --install-dir <dir>  Install directory (default: $HOME/.local/bin)
  --skip-checksum      Skip SHA256 verification
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      if [ "$#" -lt 2 ]; then
        echo "missing value for --version" >&2
        exit 1
      fi
      VERSION="$2"
      shift 2
      ;;
    --install-dir)
      if [ "$#" -lt 2 ]; then
        echo "missing value for --install-dir" >&2
        exit 1
      fi
      INSTALL_DIR="$2"
      shift 2
      ;;
    --skip-checksum)
      SKIP_CHECKSUM="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

download_file() {
  url="$1"
  dest="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
    return
  fi

  echo "curl or wget is required to download release assets" >&2
  exit 1
}

resolve_latest_version() {
  api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
  response_file="$1"
  download_file "$api_url" "$response_file"

  latest="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$response_file" | head -n 1)"
  if [ -z "$latest" ]; then
    echo "failed to resolve latest release version from GitHub API" >&2
    exit 1
  fi
  printf '%s' "$latest"
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf '%s' "linux" ;;
    Darwin) printf '%s' "darwin" ;;
    *)
      echo "unsupported OS: $(uname -s). This script supports Linux and macOS." >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf '%s' "amd64" ;;
    aarch64|arm64) printf '%s' "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m). Supported: amd64, arm64." >&2
      exit 1
      ;;
  esac
}

tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t wto-install)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT HUP INT TERM

if [ -z "$VERSION" ]; then
  VERSION="$(resolve_latest_version "$tmp_dir/latest-release.json")"
fi

os="$(detect_os)"
arch="$(detect_arch)"
archive="git-worktree-opener_${VERSION}_${os}_${arch}.tar.gz"
archive_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${archive}"
checksums_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/checksums.txt"

archive_path="$tmp_dir/$archive"
checksums_path="$tmp_dir/checksums.txt"

echo "Downloading $archive ..."
download_file "$archive_url" "$archive_path"
download_file "$checksums_url" "$checksums_path"

if [ "$SKIP_CHECKSUM" != "1" ]; then
  expected="$(awk -v target="$archive" '$2 == target { print $1 }' "$checksums_path" | head -n 1)"
  if [ -z "$expected" ]; then
    echo "failed to find checksum entry for $archive" >&2
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$archive_path" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
  else
    echo "warning: sha256sum/shasum not found; skipping checksum verification" >&2
    actual="$expected"
  fi

  if [ "$actual" != "$expected" ]; then
    echo "checksum mismatch for $archive" >&2
    exit 1
  fi
fi

tar -xzf "$archive_path" -C "$tmp_dir"
binary_path="$tmp_dir/$BINARY_NAME"
if [ ! -f "$binary_path" ]; then
  echo "downloaded archive does not contain $BINARY_NAME" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo "Installed $BINARY_NAME $VERSION to $INSTALL_DIR"
case ":${PATH:-}:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "Add this directory to PATH to run '$BINARY_NAME' directly:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
