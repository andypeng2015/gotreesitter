#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_ROOT="$REPO_ROOT/harness_out/docker"
LABEL=""

IMAGE_TAG="gotreesitter/cgo-harness:go1.25-local"
MEMORY_LIMIT="8g"
CPUS_LIMIT="4"
# CPUSET_CPUS pins the container to a specific set of physical CPUs via
# docker's --cpuset-cpus. Pinning matters more than CFS quota for benchmark
# stability — without it, the kernel scheduler can move the container
# between cores, blowing cache state and producing 20-30% wall-time
# variance between otherwise identical runs. Empty = no pinning (legacy
# CFS-quota-only behavior).
CPUSET_CPUS=""
PIDS_LIMIT="4096"
GOMEMLIMIT_VALUE="${GOMEMLIMIT:-6GiB}"
GOFLAGS_VALUE="${GOFLAGS:--p=1}"
TEST_PARALLEL="1"
TEST_TIMEOUT="20m"
PARITY_PARALLEL="0"
C_REF_BUILD_JOBS="1"
C_REF_BUILD_JOBS_SET="0"
PARITY_RUN='^TestParityFreshParse$|^TestParityIncrementalParse$|^TestParityHasNoErrors$|^TestParityIssue3Repros$|^TestParityGLRCanaryGo$|^TestParityGLRCanarySet$|^TestParityGLRCapPressureTopLanguages$|^TestParityHighlight$'
STRICT_SCALA=0
BUILD_IMAGE=1

# Ring-matrix scope: the top-50 value languages by default. The parser must
# match tree-sitter C across this set for any parser-core change to merge.
# Override with GTS_PARITY_MODE=smoke for the fast 9-language dev gate, or
# GTS_PARITY_MODE=exhaustive for every curated structural grammar.
: "${GTS_PARITY_MODE:=top50}"
export GTS_PARITY_MODE

usage() {
  cat <<'EOF'
Usage: run_parity_in_docker.sh [options] [-- <custom command>]

Options:
  --image <tag>          Docker image tag (default: gotreesitter/cgo-harness:go1.25-local)
  --repo-root <path>     Repository/worktree root mounted at /workspace
  --out-root <path>      Artifact output root (default: <repo-root>/harness_out/docker)
  --label <name>         Optional run label (used in container/artifact naming)
  --memory <limit>       Container memory limit (default: 8g)
  --cpus <count>         CPU limit passed to Docker (default: 4)
  --cpuset-cpus <list>   Pin container to specific CPUs via --cpuset-cpus
                         (e.g. "18" or "16-19"). Empty = no pinning, but
                         benchmark stability suffers — use this for any
                         perf-comparison run.
  --pids <count>         PID limit passed to Docker (default: 4096)
  --gomemlimit <value>   GOMEMLIMIT inside container (default: 6GiB)
  --goflags <value>      GOFLAGS inside container (default: -p=1)
  --test-parallel <n>    go test -parallel value (default: 1)
  --timeout <duration>   go test -timeout value (default: 20m)
  --parity-parallel <n>  Enable GTS_PARITY_PARALLEL=1 and set -parallel n.
                         This is intentionally container-only.
  --c-ref-build-jobs <n> Concurrent C-reference parser builds (default: 1)
  --run <regex>          go test -run regex for default parity command
  --strict-scala         Also run strict Scala real-world parity probe
  --no-build             Skip docker build step
  -h, --help             Show this help

Ring-matrix scope (GTS_PARITY_MODE, default: top50):
  top50       the 50 top-value languages (default ring matrix)
  smoke       fast 9-language dev gate (bash,c,c_sharp,go,html,js,python,rust,yaml)
  exhaustive  every curated structural grammar

Environment passthrough (if set):
  GOTOOLCHAIN
  GOMAXPROCS
  GOT_GLR_MAX_STACKS
  GOT_GLR_V2_COMPACT_FULL_LEAVES
  GOT_GLR_V2_PENDING_PARENTS
  GOT_PARSE_NODE_LIMIT_SCALE
  GOT_GLR_FORCE_CONFLICT_WIDTH
  GTS_PARITY_MODE
  GTS_PARITY_SKIP_LANGS
  GTS_PARITY_C_REF_BUILD_CACHE

Artifacts are written to <out-root>/<timestamp>[-<label>]/:
  - container.log
  - inspect.json
  - metadata.txt
EOF
}

CUSTOM_CMD=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --image)
      IMAGE_TAG="$2"
      shift 2
      ;;
    --repo-root)
      REPO_ROOT="$2"
      shift 2
      ;;
    --out-root)
      OUT_ROOT="$2"
      shift 2
      ;;
    --label)
      LABEL="$2"
      shift 2
      ;;
    --memory)
      MEMORY_LIMIT="$2"
      shift 2
      ;;
    --cpus)
      CPUS_LIMIT="$2"
      shift 2
      ;;
    --cpuset-cpus)
      CPUSET_CPUS="$2"
      shift 2
      ;;
    --pids)
      PIDS_LIMIT="$2"
      shift 2
      ;;
    --gomemlimit)
      GOMEMLIMIT_VALUE="$2"
      shift 2
      ;;
    --goflags)
      GOFLAGS_VALUE="$2"
      shift 2
      ;;
    --test-parallel)
      TEST_PARALLEL="$2"
      shift 2
      ;;
    --timeout)
      TEST_TIMEOUT="$2"
      shift 2
      ;;
    --parity-parallel)
      PARITY_PARALLEL="1"
      TEST_PARALLEL="$2"
      if [[ "$C_REF_BUILD_JOBS_SET" == "0" ]]; then
        C_REF_BUILD_JOBS="$2"
      fi
      shift 2
      ;;
    --c-ref-build-jobs)
      C_REF_BUILD_JOBS="$2"
      C_REF_BUILD_JOBS_SET="1"
      shift 2
      ;;
    --run)
      PARITY_RUN="$2"
      shift 2
      ;;
    --strict-scala)
      STRICT_SCALA=1
      shift
      ;;
    --no-build)
      BUILD_IMAGE=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      CUSTOM_CMD=("$@")
      break
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

REPO_ROOT="$(cd "$REPO_ROOT" && pwd)"
OUT_ROOT="${OUT_ROOT/#\~/$HOME}"
if [[ ! -d "$REPO_ROOT" ]]; then
  echo "repo root does not exist: $REPO_ROOT" >&2
  exit 2
fi
mkdir -p "$OUT_ROOT"

require_positive_int() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
    echo "$name must be a positive integer, got: $value" >&2
    exit 2
  fi
}

require_positive_int "--test-parallel" "$TEST_PARALLEL"
require_positive_int "--c-ref-build-jobs" "$C_REF_BUILD_JOBS"
require_positive_int "--pids" "$PIDS_LIMIT"

sanitize_label() {
  local in="$1"
  in="${in,,}"
  in="$(echo "$in" | sed -E 's/[^a-z0-9_.-]+/-/g; s/^-+//; s/-+$//; s/-+/-/g')"
  if [[ -z "$in" ]]; then
    in="run"
  fi
  echo "$in"
}

LABEL_SLUG=""
if [[ -n "$LABEL" ]]; then
  LABEL_SLUG="$(sanitize_label "$LABEL")"
fi

if [[ "$BUILD_IMAGE" == "1" ]]; then
  docker build -t "$IMAGE_TAG" "$SCRIPT_DIR"
fi

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="$OUT_ROOT/$STAMP"
if [[ -n "$LABEL_SLUG" ]]; then
  OUT_DIR="${OUT_DIR}-${LABEL_SLUG}"
fi
mkdir -p "$OUT_DIR"

DEFAULT_CMD="cd /workspace/cgo_harness && /usr/bin/time -v go test . -tags treesitter_c_parity -run '$PARITY_RUN' -count=1 -parallel '$TEST_PARALLEL' -timeout '$TEST_TIMEOUT' -v"
if [[ "$STRICT_SCALA" == "1" ]]; then
  DEFAULT_CMD="$DEFAULT_CMD && /usr/bin/time -v env GTS_PARITY_SCALA_REALWORLD_STRICT=1 go test . -tags treesitter_c_parity -run '^TestParityScalaRealWorldCorpus$' -count=1 -parallel '$TEST_PARALLEL' -timeout '$TEST_TIMEOUT' -v"
fi

if [[ ${#CUSTOM_CMD[@]} -gt 0 ]]; then
  INNER_CMD="${CUSTOM_CMD[*]}"
else
  INNER_CMD="$DEFAULT_CMD"
fi
INNER_CMD="export PATH=/usr/local/go/bin:\$PATH; $INNER_CMD"

ENV_ARGS=(
  "-e" "GTS_PARITY_IN_DOCKER=1"
  "-e" "GTS_PARITY_PARALLEL=$PARITY_PARALLEL"
  "-e" "GTS_PARITY_C_REF_BUILD_JOBS=$C_REF_BUILD_JOBS"
  "-e" "GOMEMLIMIT=$GOMEMLIMIT_VALUE"
  "-e" "GOFLAGS=$GOFLAGS_VALUE"
)
for var in GOTOOLCHAIN GOMAXPROCS GOT_GLR_MAX_STACKS GOT_GLR_V2_COMPACT_FULL_LEAVES GOT_GLR_V2_PENDING_PARENTS GOT_PARSE_NODE_LIMIT_SCALE GOT_GLR_FORCE_CONFLICT_WIDTH GTS_PARITY_MODE GTS_PARITY_SKIP_LANGS GTS_PARITY_C_REF_BUILD_CACHE; do
  if [[ -n "${!var:-}" ]]; then
    ENV_ARGS+=("-e" "$var=${!var}")
  fi
done

CONTAINER_NAME="gts-parity-${STAMP,,}"
if [[ -n "$LABEL_SLUG" ]]; then
  CONTAINER_NAME="${CONTAINER_NAME}-${LABEL_SLUG}"
fi
CID=""
cleanup() {
  if [[ -n "$CID" ]]; then
    docker rm -f "$CID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

CPUSET_ARGS=()
if [[ -n "$CPUSET_CPUS" ]]; then
  CPUSET_ARGS+=(--cpuset-cpus "$CPUSET_CPUS")
fi

RUN_START_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
RUN_START_NS="$(date +%s%N)"
CID="$(docker create \
  --name "$CONTAINER_NAME" \
  --init \
  --memory "$MEMORY_LIMIT" \
  --memory-swap "$MEMORY_LIMIT" \
  --cpus "$CPUS_LIMIT" \
  "${CPUSET_ARGS[@]}" \
  --pids-limit "$PIDS_LIMIT" \
  --mount "type=bind,src=$REPO_ROOT,dst=/workspace" \
  --mount "type=volume,src=gotreesitter-go-mod-cache,dst=/go/pkg/mod" \
  --mount "type=volume,src=gotreesitter-go-build-cache,dst=/root/.cache/go-build" \
  "${ENV_ARGS[@]}" \
  "$IMAGE_TAG" \
  bash -c "$INNER_CMD")"

docker start "$CID" >/dev/null
docker logs -f "$CID" 2>&1 | tee "$OUT_DIR/container.log"
EXIT_CODE="$(docker wait "$CID")"
RUN_END_NS="$(date +%s%N)"
RUN_END_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
docker inspect "$CID" >"$OUT_DIR/inspect.json"

OOM_KILLED="$(docker inspect -f '{{.State.OOMKilled}}' "$CID")"
STATE_ERROR="$(docker inspect -f '{{.State.Error}}' "$CID")"

{
  echo "container_name=$CONTAINER_NAME"
  echo "container_id=$CID"
  echo "image=$IMAGE_TAG"
  echo "memory=$MEMORY_LIMIT"
  echo "cpus=$CPUS_LIMIT"
  echo "cpuset_cpus=$CPUSET_CPUS"
  echo "pids=$PIDS_LIMIT"
  echo "gomemlimit=$GOMEMLIMIT_VALUE"
  echo "goflags=$GOFLAGS_VALUE"
  echo "test_parallel=$TEST_PARALLEL"
  echo "test_timeout=$TEST_TIMEOUT"
  echo "parity_parallel=$PARITY_PARALLEL"
  echo "c_ref_build_jobs=$C_REF_BUILD_JOBS"
  echo "strict_scala=$STRICT_SCALA"
  echo "exit_code=$EXIT_CODE"
  echo "oom_killed=$OOM_KILLED"
  echo "state_error=$STATE_ERROR"
  echo "run_start_utc=$RUN_START_UTC"
  echo "run_end_utc=$RUN_END_UTC"
  echo "run_start_ns=$RUN_START_NS"
  echo "run_end_ns=$RUN_END_NS"
  echo "elapsed_ns=$((RUN_END_NS - RUN_START_NS))"
  echo "repo_root=$REPO_ROOT"
  echo "out_root=$OUT_ROOT"
  echo "label=$LABEL_SLUG"
  echo "command=$INNER_CMD"
} >"$OUT_DIR/metadata.txt"

echo "docker parity run complete"
echo "artifacts: $OUT_DIR"
echo "exit_code: $EXIT_CODE"
echo "oom_killed: $OOM_KILLED"
if [[ -n "$STATE_ERROR" ]]; then
  echo "docker_state_error: $STATE_ERROR"
fi

if [[ "$EXIT_CODE" != "0" ]]; then
  exit "$EXIT_CODE"
fi
