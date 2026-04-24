#!/bin/bash
set -e

if [ -z "$GH_TOKEN" ]; then
  echo "Error: GH_TOKEN is not set." >&2
  echo "       Re-run with: docker run -e GH_TOKEN=<token> ..." >&2
  exit 1
fi

# Check whether the gh copilot extension binary is present.
# If the image was built correctly it will already be there; this block is
# only a safety net for ad-hoc runs against an unbuilt image.
EXT_DIR="/root/.local/share/gh/extensions/gh-copilot"
EXT_BIN="${EXT_DIR}/gh-copilot"
if [ ! -x "$EXT_BIN" ]; then
  echo "gh copilot extension binary not found — downloading…" >&2
  # Map uname -m to the asset naming used by github/gh-copilot releases.
  case "$(uname -m)" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)       ARCH="amd64" ;;
  esac
  mkdir -p "$EXT_DIR"
  curl -fsSL \
    "https://github.com/github/gh-copilot/releases/download/v1.2.0/linux-${ARCH}" \
    -o "$EXT_BIN" \
  && chmod +x "$EXT_BIN" \
  || {
    echo "Warning: could not download gh copilot extension — proceeding anyway." >&2
  }
fi

exec gh copilot suggest
