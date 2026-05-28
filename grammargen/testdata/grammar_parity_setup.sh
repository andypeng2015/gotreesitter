#!/usr/bin/env bash
# Sets up the out-of-tree grammar.json corpus needed by parity_test.go's
# markdown / markdown_inline entries. Idempotent — does nothing if already
# set up. Pins the upstream tree-sitter-markdown to a specific SHA so the
# parity corpus is reproducible across checkouts.

set -euo pipefail

PINNED_SHA="c3570720f7f7bbad22fe96603f106276618e0cf5"
UPSTREAM_URL="https://github.com/tree-sitter-grammars/tree-sitter-markdown"
TARGET_DIR="/tmp/grammar_parity/markdown"

if [ -d "$TARGET_DIR/.git" ]; then
  current_sha=$(git -C "$TARGET_DIR" rev-parse HEAD)
  if [ "$current_sha" = "$PINNED_SHA" ]; then
    exit 0
  fi
  echo "tree-sitter-markdown at $TARGET_DIR is at $current_sha, expected $PINNED_SHA; resyncing..."
  git -C "$TARGET_DIR" fetch origin
  git -C "$TARGET_DIR" checkout "$PINNED_SHA"
  exit 0
fi

mkdir -p "$(dirname "$TARGET_DIR")"
git clone "$UPSTREAM_URL" "$TARGET_DIR"
git -C "$TARGET_DIR" checkout "$PINNED_SHA"
