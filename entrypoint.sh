#!/bin/bash
set -e

# ---------------------------------------------------------------------------
# Install the gh copilot extension at container startup using the runtime
# GH_TOKEN, then launch the Copilot CLI directly.
# ---------------------------------------------------------------------------
if [ -z "$GH_TOKEN" ]; then
  echo "Error: GH_TOKEN is not set." >&2
  echo "       Re-run with: docker run -e GH_TOKEN=<token> ..." >&2
  exit 1
fi

echo "Installing gh copilot extension..."
if ! gh extension install github/gh-copilot --force; then
  echo "" >&2
  echo "Error: failed to install gh copilot extension." >&2
  echo "       Make sure GH_TOKEN has the 'copilot_requests: write' permission." >&2
  echo "       See docs/github-pat.md for how to create a scoped token." >&2
  exit 1
fi

echo ""
exec gh copilot suggest
