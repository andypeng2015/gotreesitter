#!/usr/bin/env bash
# Forest-vs-C ORACLE gate. Compares the GSS-forest fast path DIRECTLY against
# tree-sitter-c, skipping the production parser — so a language whose production
# parse is too slow to use as the parity baseline (haskell's O(n^2) deep-merge
# blowup times out the production-vs-forest gate, run_forest_corpus_parity.sh)
# can still be vetted for forest promotion. A per-file budget keeps a forest that
# itself blows up (haskell's forestReducer.dfs) from hanging the run; such files
# are reported as fallback reason "timeout".
#
# Heavy (real corpus + CGo) -> runs in Docker per repo testing discipline.
#
# Usage:
#   cgo_harness/docker/run_forest_oracle_parity.sh                       # haskell (default)
#   cgo_harness/docker/run_forest_oracle_parity.sh --langs haskell,d
#   cgo_harness/docker/run_forest_oracle_parity.sh --budget 30s --no-build
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_ROOT="$REPO_ROOT/harness_out/forest_oracle"

IMAGE_TAG="gotreesitter/cgo-harness:go1.25-local"
MEMORY_LIMIT="8g"
CPUS_LIMIT="4"
PIDS_LIMIT="4096"
TIMEOUT="10m"
LANGS="haskell"
BUDGET="10s"
BUILD_IMAGE=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --langs) LANGS="$2"; shift 2 ;;
    --budget) BUDGET="$2"; shift 2 ;;
    --image) IMAGE_TAG="$2"; shift 2 ;;
    --memory) MEMORY_LIMIT="$2"; shift 2 ;;
    --cpus) CPUS_LIMIT="$2"; shift 2 ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    --no-build) BUILD_IMAGE=0; shift ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="$OUT_ROOT/$STAMP"
mkdir -p "$OUT_DIR"

if [[ "$BUILD_IMAGE" == "1" ]]; then
  docker build -f "$SCRIPT_DIR/Dockerfile" -t "$IMAGE_TAG" "$REPO_ROOT" \
    > "$OUT_DIR/build.log" 2>&1 || { echo "image build failed; see $OUT_DIR/build.log" >&2; exit 1; }
fi

set +e
docker run --rm \
  --mount "type=bind,src=$REPO_ROOT,dst=/workspace" \
  --mount "type=volume,src=gotreesitter-go-mod-cache,dst=/go/pkg/mod" \
  --mount "type=volume,src=gotreesitter-go-build-cache,dst=/root/.cache/go-build" \
  -w /workspace/cgo_harness \
  -e GTS_FOREST_ORACLE=1 \
  -e "GTS_FOREST_ORACLE_LANGS=$LANGS" \
  -e "GTS_FOREST_ORACLE_BUDGET=$BUDGET" \
  --memory "$MEMORY_LIMIT" --cpus "$CPUS_LIMIT" --pids-limit "$PIDS_LIMIT" \
  "$IMAGE_TAG" \
  bash -c "go test . -tags treesitter_c_parity -run '^TestForestVsCOracleParity\$' -count=1 -v -timeout $TIMEOUT" \
  2>&1 | tee "$OUT_DIR/container.log"
code=${PIPESTATUS[0]}
set -e

{
  echo "langs=$LANGS"
  echo "budget=$BUDGET"
  echo "image=$IMAGE_TAG"
  echo "exit_code=$code"
} > "$OUT_DIR/summary.txt"

echo "forest oracle parity complete -> $OUT_DIR"
exit "$code"
