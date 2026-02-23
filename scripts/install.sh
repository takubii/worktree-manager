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
    return 0
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
    return 0
  fi

  echo "curl or wget is required to download release assets" >&2
  exit 1
}

resolve_release_json() {
  output_file="$1"
  if [ -z "$VERSION" ]; then
    api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
  else
    api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/tags/${VERSION}"
  fi
  download_file "$api_url" "$output_file"
}

extract_tag_name() {
  json_file="$1"
  tag_name="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$json_file" | head -n 1)"
  if [ -z "$tag_name" ]; then
    echo "failed to resolve release tag from GitHub API response" >&2
    exit 1
  fi
  printf '%s' "$tag_name"
}

select_asset_urls() {
  json_file="$1"
  os="$2"
  arch="$3"

  archive_url=""
  checksums_url=""

  urls="$(sed -n 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$json_file")"
  if [ -z "$urls" ]; then
    echo "failed to find release assets in GitHub API response" >&2
    exit 1
  fi

  echo "$urls" | while IFS= read -r url; do
    case "$url" in
      */git-worktree-opener_*_"${os}_${arch}.tar.gz")
        echo "archive=$url"
        ;;
      */checksums.txt)
        echo "checksums=$url"
        ;;
    esac
  done
}

resolve_asset_urls() {
  json_file="$1"
  os="$2"
  arch="$3"

  archive_url=""
  checksums_url=""

  selected="$(select_asset_urls "$json_file" "$os" "$arch")"
  if [ -n "$selected" ]; then
    archive_url="$(echo "$selected" | sed -n 's/^archive=//p' | head -n 1)"
    checksums_url="$(echo "$selected" | sed -n 's/^checksums=//p' | head -n 1)"
  fi

  if [ -z "$archive_url" ]; then
    echo "could not find release archive for ${os}/${arch}" >&2
    exit 1
  fi
  if [ -z "$checksums_url" ]; then
    echo "could not find checksums.txt in release assets" >&2
    exit 1
  fi

  printf '%s\n%s\n' "$archive_url" "$checksums_url"
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

os="$(detect_os)"
arch="$(detect_arch)"

release_json="$tmp_dir/release.json"
resolve_release_json "$release_json"
VERSION="$(extract_tag_name "$release_json")"

asset_urls="$(resolve_asset_urls "$release_json" "$os" "$arch")"
archive_url="$(printf '%s\n' "$asset_urls" | sed -n '1p')"
checksums_url="$(printf '%s\n' "$asset_urls" | sed -n '2p')"
archive_name="$(basename "$archive_url")"

archive_path="$tmp_dir/$archive_name"
checksums_path="$tmp_dir/checksums.txt"

echo "Downloading $archive_name ..."
download_file "$archive_url" "$archive_path"
download_file "$checksums_url" "$checksums_path"

if [ "$SKIP_CHECKSUM" != "1" ]; then
  expected="$(awk -v target="$archive_name" '$2 == target { print $1 }' "$checksums_path" | head -n 1)"
  if [ -z "$expected" ]; then
    echo "failed to find checksum entry for $archive_name" >&2
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
    echo "checksum mismatch for $archive_name" >&2
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
