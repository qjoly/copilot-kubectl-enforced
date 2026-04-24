#!/bin/bash
set -e

# ---------------------------------------------------------------------------
# Install the gh copilot extension at container startup using the runtime
# GH_TOKEN. This avoids any token requirement at image build time.
# ---------------------------------------------------------------------------
if [ -z "$GH_TOKEN" ]; then
  echo "Warning: GH_TOKEN is not set — gh copilot will not be available." >&2
  echo "         Re-run with: docker run -e GH_TOKEN=<token> ..." >&2
else
  echo "Installing gh copilot extension..."
  if gh extension install github/gh-copilot --force 2>/dev/null; then
    echo "  gh copilot extension ready."
  else
    echo "Warning: failed to install gh copilot extension." >&2
  fi
fi

exec /bin/bash --login "$@"
