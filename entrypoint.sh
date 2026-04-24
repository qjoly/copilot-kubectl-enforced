#!/bin/bash
set -e

if [ -z "$GH_TOKEN" ]; then
  echo "Error: GH_TOKEN is not set." >&2
  echo "       Re-run with: docker run -e GH_TOKEN=<token> ..." >&2
  exit 1
fi

# Install the gh copilot extension if not already present.
# If gh ships copilot as a built-in the install exits non-zero with a
# "built-in" message — that is not an error. Any other failure is fatal.
out=$(gh extension install github/gh-copilot 2>&1); rc=$?
if [ $rc -ne 0 ] && ! echo "$out" | grep -qi "built-in\|alias"; then
  printf 'Error: failed to install gh copilot extension:\n%s\n' "$out" >&2
  exit $rc
fi

exec gh copilot suggest
