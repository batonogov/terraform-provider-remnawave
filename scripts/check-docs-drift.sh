#!/bin/sh

set -eu

docs_status=$(git status --porcelain=v1 --untracked-files=all -- docs)

if [ -z "$docs_status" ]; then
  exit 0
fi

echo "Generated documentation is out of date:" >&2
printf '%s\n' "$docs_status" >&2

# Show tracked changes when available. Untracked files are listed above by
# git status; neither command modifies the caller's index or working tree.
git --no-pager diff -- docs >&2 || true
git --no-pager diff --cached -- docs >&2 || true

echo "Run 'task docs' and commit the generated files." >&2
exit 1
