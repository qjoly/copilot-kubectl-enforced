#!/bin/bash
set -e

if [ -z "$GH_TOKEN" ]; then
  echo "Error: GH_TOKEN is not set." >&2
  echo "       Re-run with: docker run -e GH_TOKEN=<token> ..." >&2
  exit 1
fi

# Install the gh copilot extension if it is not already present.
# GH_TOKEN is already exported so gh uses it for authentication automatically.
if ! gh extension list 2>/dev/null | grep -q 'copilot'; then
  echo "Installing gh copilot extension…" >&2
  gh extension install github/gh-copilot
fi

exec gh copilot suggest
