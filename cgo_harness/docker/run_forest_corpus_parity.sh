#!/usr/bin/env bash
# Forest correctness gate: parse the real corpus with the production parser and
# with the GSS-forest fast path and assert byte-identical trees on every file
# the forest would dispatch (clean + complete). Divergences block promoting a
# language onto the forest default-on path (languageWantsForest). Also reports
# dispatch rate and wall speedup — the "wall" half of parity-wall-and-correctness.
#
# Heavy (real corpus) → runs in Docker per repo testing discipline.
#
# Usage:
#   cgo_harness/docker/run_forest_corpus_parity.sh                 # bash (default)
#   cgo_harness/docker/run_forest_corpus_parity.sh --langs bash,swift
#   cgo_harness/docker/run_forest_corpus_parity.sh --no-build
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_ROOT="$REPO_ROOT/harness_out/forest_corpus"

IMAGE_TAG="gotreesitter/cgo-harness:go1.25-local"
MEMORY_LIMIT="12g"
CPUS_LIMIT="4"
PIDS_LIMIT="4096"
TIMEOUT="20m"
LANGS="bash"
BUILD_IMAGE=1

usage() {
  cat <<'EOF'
Usage: run_forest_corpus_parity.sh [options]
  --langs <csv>      comma-separated languages to gate (default: bash)
  --image <tag>      docker image tag (default: gotreesitter/cgo-harness:go1.25-local)
  --memory <limit>   container memory limit (default: 12g)
  --cpus <count>     cpu limit (default: 4)
  --timeout <dur>    go test timeout (default: 20m)
  --no-build         skip docker build step
  -h, --help         show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --langs) LANGS="$2"; shift 2 ;;
    --image) IMAGE_TAG="$2"; shift 2 ;;
    --memory) MEMORY_LIMIT="$2"; shift 2 ;;
    --cpus) CPUS_LIMIT="$2"; shift 2 ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    --no-build) BUILD_IMAGE=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="$OUT_ROOT/$STAMP"
mkdir -p "$OUT_DIR"

if [[ "$BUILD_IMAGE" == "1" ]]; then
  docker build -t "$IMAGE_TAG" "$SCRIPT_DIR"
fi

CONTAINER_NAME="gts-forest-corpus-$STAMP"
INNER_CMD="cd /workspace/cgo_harness && GTS_FOREST_CORPUS=1 GTS_FOREST_LANGS='$LANGS' /usr/bin/time -v go test ./ -run '^TestForestCorpusParity$' -count=1 -timeout $TIMEOUT -v"

CID="$(docker create \
  --name "$CONTAINER_NAME" \
  --init \
  --memory "$MEMORY_LIMIT" \
  --memory-swap "$MEMORY_LIMIT" \
  --cpus "$CPUS_LIMIT" \
  --pids-limit "$PIDS_LIMIT" \
  --mount "type=bind,src=$REPO_ROOT,dst=/workspace" \
  --mount "type=volume,src=gotreesitter-go-mod-cache,dst=/go/pkg/mod" \
  --mount "type=volume,src=gotreesitter-go-build-cache,dst=/root/.cache/go-build" \
  "$IMAGE_TAG" \
  bash -c "$INNER_CMD")"

docker start "$CID" >/dev/null
docker logs -f "$CID" 2>&1 | tee "$OUT_DIR/container.log"
EXIT_CODE="$(docker wait "$CID")"
OOM_KILLED="$(docker inspect -f '{{.State.OOMKilled}}' "$CID")"
docker rm "$CID" >/dev/null 2>&1 || true

{
  echo "langs=$LANGS"
  echo "image=$IMAGE_TAG"
  echo "memory=$MEMORY_LIMIT"
  echo "exit_code=$EXIT_CODE"
  echo "oom_killed=$OOM_KILLED"
} | tee "$OUT_DIR/summary.txt"

echo "artifacts: $OUT_DIR"
exit "$EXIT_CODE"
