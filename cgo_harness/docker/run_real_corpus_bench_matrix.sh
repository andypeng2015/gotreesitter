#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RUNNER="$SCRIPT_DIR/run_parity_in_docker.sh"

LANGS_CSV="go,python,rust,java,javascript,typescript,c"
LANGS_SET=0
LANG_PROFILE=""
LANGS_FILE=""
OUT_ROOT="$REPO_ROOT/harness_out/real_corpus_bench_matrix"
# Default to higher count + longer benchtime than the legacy values
# (5×750ms). The legacy defaults produced 28% wall-time variance between
# back-to-back runs on the same code, which made tracking <5% perf
# improvements impossible. 10×5s gives ~6× more parse iterations per
# bench line, which tightens variance well below 5%.
COUNT="10"
BENCHTIME="5s"
MEMORY_LIMIT="8g"
CPUS_LIMIT="1"
# Pin the container to a specific physical CPU (one for Go bench, the
# adjacent for CGo bench would be ideal but Docker can't switch
# per-invocation, so use a single pinned CPU for both backends in the
# same matrix run). CPU 18 mirrors the host-side ab_pinned.sh default;
# it's near the end of the high-frequency cores so the kernel scheduler
# is less likely to want it for other work. Empty = no pinning.
CPUSET_CPUS="18"
PIDS_LIMIT="4096"
GOMAXPROCS_VALUE="1"
GOMEMLIMIT_VALUE="${GOMEMLIMIT:-6GiB}"
TEST_TIMEOUT="20m"
ALLOW_MISMATCH="0"
SKIP_MISMATCH="0"
PHASE_TIMING="1"
MAX_FILES=""
MAX_BYTES=""
MAX_FILE_BYTES=""
MIN_BYTES=""
SHARDS=""
SHARD=""
ORDER="path"
BENCH_TIMEOUT=""
CORPUS_ROOT=""
CORPUS_SOURCES_ROOT=""
CORPUS_SOURCE_LOCK=""
BUILD_IMAGE=1
EXTRA_ENV=()
KEEP_GOING=1
JOBS="1"
ALLOW_HOST_OVERSUBSCRIBE=0

usage() {
  cat <<'EOF'
Usage: run_real_corpus_bench_matrix.sh [options]

Run real-corpus Go-vs-C parse benchmarks one language per Docker container,
then build a ranked markdown/json report from the benchmark logs.

Options:
  --langs <list>          Comma-separated languages (default: go,python,rust,java,javascript,typescript,c)
  --profile <name>        Language preset: usage-top12, top50-high-value.
                          May not be combined with --langs.
  --langs-file <path>     Newline- or comma-separated language list, such as output from
                          real_corpus_inventory -select ready-to-benchmark -out-langs
  --out-root <path>       Output root (default: harness_out/real_corpus_bench_matrix)
  --count <n>             go test benchmark count (default: 10)
  --benchtime <dur>       go test benchmark benchtime (default: 5s)
  --memory <limit>        Docker memory limit (default: 8g)
  --cpus <count>          Docker CPU limit (default: 1)
  --cpuset-cpus <list>    Pin container to specific CPUs via --cpuset-cpus
                          (e.g. "18" or "16-19"). Default: "18". Empty
                          disables pinning, but ratio comparisons across
                          back-to-back runs become unreliable without it.
  --pids <count>          Docker PID limit (default: 4096)
  --gomaxprocs <n>        GOMAXPROCS inside container (default: 1)
  --gomemlimit <value>    GOMEMLIMIT inside container (default: ${GOMEMLIMIT:-6GiB})
  --timeout <duration>    go test timeout inside container (default: 20m)
  --allow-mismatch        Skip strict fresh parity precheck and time selected files
  --skip-mismatch         Filter out files that fail fresh parity precheck
  --phase-timing <0|1>    Export GOT_PARSE_PHASE_TIMING (default: 1)
  --max-files <n>         Export GTS_REAL_CORPUS_BENCH_MAX_FILES
  --max-bytes <n>         Export GTS_REAL_CORPUS_BENCH_MAX_BYTES
  --max-file-bytes <n>    Export GTS_REAL_CORPUS_BENCH_MAX_FILE_BYTES
  --min-bytes <n>         Export GTS_REAL_CORPUS_BENCH_MIN_BYTES
  --shards <n>            Export GTS_REAL_CORPUS_BENCH_SHARDS for deterministic corpus sharding
  --shard <n>             Export GTS_REAL_CORPUS_BENCH_SHARD, 1-based within --shards
  --order <mode>          path|largest|smallest (default: path)
  --bench-timeout <dur>   Bound each per-language go test inside the container
                          with coreutils timeout, e.g. 2m or 30s. Timed-out
                          languages are reported as benchmark-run-aborted.
  --corpus-root <path>    Export GTS_REAL_CORPUS_BENCH_ROOT. Path is resolved
                          inside /workspace/cgo_harness unless absolute.
  --corpus-sources-root <path>
                          Host path to external per-language corpus source
                          checkouts. Mounted read-only at /corpus-sources and
                          benchmarked as /corpus-sources/<language>.
  --corpus-source-lock <path>
                          Optional lock file for --corpus-sources-root
                          filtering. Mounted read-only at /corpus-source.lock.
  --stop-on-failure       Stop after the first language failure
  --jobs <n>              Concurrent per-language containers (default: 1).
                          Keep at 1 for stable perf attribution; use >1 for
                          coarse screening. Aggregate container memory is
                          guarded against host MemAvailable by default.
  --allow-host-oversubscribe
                          Allow --jobs * --memory to exceed the host memory
                          guard. Intended only for dedicated CI hosts.
  --no-build              Skip Docker image build in underlying runner
  --extra-env <K=V>       Append KEY=VALUE to the env prefix passed to go test
                          inside the container. May be supplied multiple times.
  -h, --help              Show this help

The generated report is written to:
  <out-root>/<timestamp>/REAL_CORPUS_BENCH_REPORT.md
  <out-root>/<timestamp>/real_corpus_bench_report.json
EOF
}

lang_profile_csv() {
  case "$1" in
    usage|usage-top12|top12)
      printf '%s\n' "go,typescript,tsx,javascript,java,python,rust,c,cpp,c_sharp,json,css"
      ;;
    top50-high-value|top50-no-deferred|top50-real)
      # Preserve top-50 priority order while deferring gomod and entries that
      # do not currently have real-corpus inputs in this harness.
      printf '%s\n' "typescript,tsx,javascript,python,java,c_sharp,php,bash,cpp,go,html,css,sql,c,rust,json,ruby,swift,kotlin,dart,lua,yaml,xml,toml,markdown,svelte,scss,powershell,r,scala,hcl,graphql,perl,elixir,haskell,julia,clojure,erlang,ocaml,nix,objc,make,cmake,d,awk,elm"
      ;;
    *)
      echo "unknown --profile: $1" >&2
      echo "known profiles: usage-top12, top50-high-value" >&2
      exit 2
      ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --langs) LANGS_CSV="$2"; LANGS_SET=1; shift 2 ;;
    --profile) LANG_PROFILE="$2"; shift 2 ;;
    --langs-file) LANGS_FILE="$2"; shift 2 ;;
    --out-root) OUT_ROOT="$2"; shift 2 ;;
    --count) COUNT="$2"; shift 2 ;;
    --benchtime) BENCHTIME="$2"; shift 2 ;;
    --memory) MEMORY_LIMIT="$2"; shift 2 ;;
    --cpus) CPUS_LIMIT="$2"; shift 2 ;;
    --cpuset-cpus) CPUSET_CPUS="$2"; shift 2 ;;
    --pids) PIDS_LIMIT="$2"; shift 2 ;;
    --gomaxprocs) GOMAXPROCS_VALUE="$2"; shift 2 ;;
    --gomemlimit) GOMEMLIMIT_VALUE="$2"; shift 2 ;;
    --timeout) TEST_TIMEOUT="$2"; shift 2 ;;
    --allow-mismatch) ALLOW_MISMATCH="1"; shift ;;
    --skip-mismatch) SKIP_MISMATCH="1"; shift ;;
    --phase-timing) PHASE_TIMING="$2"; shift 2 ;;
    --max-files) MAX_FILES="$2"; shift 2 ;;
    --max-bytes) MAX_BYTES="$2"; shift 2 ;;
    --max-file-bytes) MAX_FILE_BYTES="$2"; shift 2 ;;
    --min-bytes) MIN_BYTES="$2"; shift 2 ;;
    --shards) SHARDS="$2"; shift 2 ;;
    --shard) SHARD="$2"; shift 2 ;;
    --order) ORDER="$2"; shift 2 ;;
    --bench-timeout) BENCH_TIMEOUT="$2"; shift 2 ;;
    --corpus-root) CORPUS_ROOT="$2"; shift 2 ;;
    --corpus-sources-root) CORPUS_SOURCES_ROOT="$2"; shift 2 ;;
    --corpus-source-lock) CORPUS_SOURCE_LOCK="$2"; shift 2 ;;
    --stop-on-failure) KEEP_GOING=0; shift ;;
    --jobs) JOBS="$2"; shift 2 ;;
    --allow-host-oversubscribe) ALLOW_HOST_OVERSUBSCRIBE=1; shift ;;
    --no-build) BUILD_IMAGE=0; shift ;;
    --extra-env) EXTRA_ENV+=("$2"); shift 2 ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -n "$LANG_PROFILE" ]]; then
  if [[ "$LANGS_SET" == "1" ]]; then
    echo "--profile may not be combined with --langs" >&2
    exit 2
  fi
  if [[ -n "$LANGS_FILE" ]]; then
    echo "--profile may not be combined with --langs-file" >&2
    exit 2
  fi
  LANGS_CSV="$(lang_profile_csv "$LANG_PROFILE")"
fi

case "$ORDER" in
  path|largest|smallest) ;;
  *)
    echo "invalid --order: $ORDER" >&2
    exit 2
    ;;
esac

require_positive_int() {
  local name="$1"
  local value="$2"
  if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
    echo "$name must be a positive integer, got: $value" >&2
    exit 2
  fi
}

docker_memory_limit_to_bytes() {
  local value="$1"
  local number unit
  if [[ "$value" =~ ^([0-9]+)([bBkKmMgG]?)$ ]]; then
    number="${BASH_REMATCH[1]}"
    unit="${BASH_REMATCH[2],,}"
  else
    return 1
  fi
  case "$unit" in
    ""|b) printf '%s\n' "$number" ;;
    k) printf '%s\n' "$((number * 1024))" ;;
    m) printf '%s\n' "$((number * 1024 * 1024))" ;;
    g) printf '%s\n' "$((number * 1024 * 1024 * 1024))" ;;
    *) return 1 ;;
  esac
}

host_mem_available_bytes() {
  awk '/^MemAvailable:/ { printf "%.0f\n", $2 * 1024 }' /proc/meminfo 2>/dev/null
}

guard_parallel_memory_budget() {
  if [[ "$JOBS" -le 1 || "$ALLOW_HOST_OVERSUBSCRIBE" == "1" ]]; then
    return 0
  fi
  local limit_bytes available_bytes aggregate_bytes guard_bytes
  limit_bytes="$(docker_memory_limit_to_bytes "$MEMORY_LIMIT" || true)"
  available_bytes="$(host_mem_available_bytes || true)"
  if [[ -z "$limit_bytes" || -z "$available_bytes" ]]; then
    echo "warning: could not parse memory guard inputs; proceeding with --jobs=$JOBS memory=$MEMORY_LIMIT" >&2
    return 0
  fi
  aggregate_bytes="$((limit_bytes * JOBS))"
  guard_bytes="$((available_bytes * 80 / 100))"
  if [[ "$aggregate_bytes" -gt "$guard_bytes" ]]; then
    {
      echo "refusing --jobs=$JOBS with --memory=$MEMORY_LIMIT: aggregate container memory exceeds 80% of host MemAvailable"
      echo "aggregate_bytes=$aggregate_bytes memavailable_bytes=$available_bytes guard_bytes=$guard_bytes"
      echo "lower --jobs/--memory or pass --allow-host-oversubscribe on a dedicated host"
    } >&2
    exit 2
  fi
}

require_positive_int "--jobs" "$JOBS"
guard_parallel_memory_budget

OUT_ROOT="${OUT_ROOT/#\~/$HOME}"
if [[ "$OUT_ROOT" != /* ]]; then
  OUT_ROOT="$PWD/$OUT_ROOT"
fi
mkdir -p "$OUT_ROOT"
OUT_ROOT="$(cd "$OUT_ROOT" && pwd -P)"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_DIR="$OUT_ROOT/$STAMP"
DOCKER_OUT="$RUN_DIR/docker"
mkdir -p "$DOCKER_OUT"

RAW_LANGS=()
if [[ -n "$LANGS_FILE" ]]; then
  if [[ ! -f "$LANGS_FILE" ]]; then
    echo "--langs-file not found: $LANGS_FILE" >&2
    exit 2
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    line="${line//,/ }"
    for token in $line; do
      RAW_LANGS+=("$token")
    done
  done <"$LANGS_FILE"
elif [[ "${LANGS_CSV,,}" == "all" || "${LANGS_CSV,,}" == "all206" ]]; then
  while IFS= read -r line; do
    RAW_LANGS+=("$line")
  done < <(
    cd "$REPO_ROOT/cgo_harness"
    go run ./cmd/build_real_corpus -langs all -print-langs
  )
else
  IFS=',' read -r -a RAW_LANGS <<< "$LANGS_CSV"
fi
LANGS=()
for raw in "${RAW_LANGS[@]}"; do
  lang="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  lang="${lang//[[:space:]]/}"
  if [[ -n "$lang" ]]; then
    LANGS+=("$lang")
  fi
done
if [[ ${#LANGS[@]} -eq 0 ]]; then
  echo "no languages selected" >&2
  exit 2
fi
if [[ -n "$CORPUS_SOURCES_ROOT" ]]; then
  if [[ -n "$CORPUS_ROOT" ]]; then
    echo "set only one of --corpus-root or --corpus-sources-root" >&2
    exit 2
  fi
  CORPUS_SOURCES_ROOT="${CORPUS_SOURCES_ROOT/#\~/$HOME}"
  if [[ "$CORPUS_SOURCES_ROOT" != /* ]]; then
    CORPUS_SOURCES_ROOT="$PWD/$CORPUS_SOURCES_ROOT"
  fi
  if [[ ! -d "$CORPUS_SOURCES_ROOT" ]]; then
    echo "--corpus-sources-root does not exist: $CORPUS_SOURCES_ROOT" >&2
    exit 2
  fi
  CORPUS_SOURCES_ROOT="$(cd "$CORPUS_SOURCES_ROOT" && pwd -P)"
  CORPUS_ROOT="/corpus-sources"
fi
if [[ -n "$CORPUS_SOURCE_LOCK" ]]; then
  if [[ -z "$CORPUS_SOURCES_ROOT" ]]; then
    echo "--corpus-source-lock requires --corpus-sources-root" >&2
    exit 2
  fi
  CORPUS_SOURCE_LOCK="${CORPUS_SOURCE_LOCK/#\~/$HOME}"
  if [[ "$CORPUS_SOURCE_LOCK" != /* ]]; then
    CORPUS_SOURCE_LOCK="$PWD/$CORPUS_SOURCE_LOCK"
  fi
  if [[ ! -f "$CORPUS_SOURCE_LOCK" ]]; then
    echo "--corpus-source-lock does not exist: $CORPUS_SOURCE_LOCK" >&2
    exit 2
  fi
  CORPUS_SOURCE_LOCK="$(cd "$(dirname "$CORPUS_SOURCE_LOCK")" && pwd -P)/$(basename "$CORPUS_SOURCE_LOCK")"
fi

bench_env_prefix() {
  local lang="$1"
  local envs=(
    "GOMAXPROCS=$GOMAXPROCS_VALUE"
    "GOT_PARSE_PHASE_TIMING=$PHASE_TIMING"
    "GTS_REAL_CORPUS_BENCH_LANGS=$lang"
    "GTS_REAL_CORPUS_BENCH_ALLOW_MISMATCH=$ALLOW_MISMATCH"
    "GTS_REAL_CORPUS_BENCH_SKIP_MISMATCH=$SKIP_MISMATCH"
    "GTS_REAL_CORPUS_BENCH_ORDER=$ORDER"
  )
  if [[ -n "$CORPUS_ROOT" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_ROOT=$CORPUS_ROOT"); fi
  if [[ -n "$CORPUS_SOURCES_ROOT" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_LOCK_FILTER=1"); fi
  if [[ -n "$CORPUS_SOURCE_LOCK" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_LOCK=/corpus-source.lock"); fi
  if [[ -n "$MAX_FILES" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_MAX_FILES=$MAX_FILES"); fi
  if [[ -n "$MAX_BYTES" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_MAX_BYTES=$MAX_BYTES"); fi
  if [[ -n "$MAX_FILE_BYTES" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_MAX_FILE_BYTES=$MAX_FILE_BYTES"); fi
  if [[ -n "$MIN_BYTES" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_MIN_BYTES=$MIN_BYTES"); fi
  if [[ -n "$SHARDS" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_SHARDS=$SHARDS"); fi
  if [[ -n "$SHARD" ]]; then envs+=("GTS_REAL_CORPUS_BENCH_SHARD=$SHARD"); fi
  for kv in "${EXTRA_ENV[@]}"; do envs+=("$kv"); done
  printf 'env'
  for env_kv in "${envs[@]}"; do
    printf ' %q' "$env_kv"
  done
}

failures=()
build_flag=()
if [[ "$BUILD_IMAGE" == "0" ]]; then
  build_flag=(--no-build)
fi

GIT_REVISION="$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || true)"
GIT_DIRTY="unknown"
if git -C "$REPO_ROOT" diff --quiet --ignore-submodules -- 2>/dev/null && git -C "$REPO_ROOT" diff --cached --quiet --ignore-submodules -- 2>/dev/null; then
  GIT_DIRTY="false"
elif git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  GIT_DIRTY="true"
fi

{
  echo "schema=real-corpus-bench-matrix-v1"
  echo "run_start_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "repo_root=$REPO_ROOT"
  echo "git_revision=$GIT_REVISION"
  echo "git_dirty=$GIT_DIRTY"
  echo "profile=$LANG_PROFILE"
  echo "langs=${LANGS[*]}"
  echo "count=$COUNT"
  echo "benchtime=$BENCHTIME"
  echo "timeout=$TEST_TIMEOUT"
  echo "memory=$MEMORY_LIMIT"
  echo "cpus=$CPUS_LIMIT"
  echo "cpuset_cpus=$CPUSET_CPUS"
  echo "pids=$PIDS_LIMIT"
  echo "gomaxprocs=$GOMAXPROCS_VALUE"
  echo "gomemlimit=$GOMEMLIMIT_VALUE"
  echo "allow_mismatch=$ALLOW_MISMATCH"
  echo "skip_mismatch=$SKIP_MISMATCH"
  echo "phase_timing=$PHASE_TIMING"
  echo "max_files=$MAX_FILES"
  echo "max_bytes=$MAX_BYTES"
  echo "max_file_bytes=$MAX_FILE_BYTES"
  echo "min_bytes=$MIN_BYTES"
  echo "shards=$SHARDS"
  echo "shard=$SHARD"
  echo "order=$ORDER"
  echo "bench_timeout=$BENCH_TIMEOUT"
  echo "corpus_root=$CORPUS_ROOT"
  echo "corpus_sources_root=$CORPUS_SOURCES_ROOT"
  echo "corpus_source_lock=$CORPUS_SOURCE_LOCK"
  echo "keep_going=$KEEP_GOING"
  echo "jobs=$JOBS"
  echo "allow_host_oversubscribe=$ALLOW_HOST_OVERSUBSCRIBE"
  echo "extra_env=${EXTRA_ENV[*]:-}"
} >"$RUN_DIR/matrix_metadata.txt"

run_language() {
  local lang="$1"
  local build_mode="$2"
  echo "==> real-corpus bench: $lang"
  env_prefix="$(bench_env_prefix "$lang")"
  bench_cmd="$env_prefix go test . -tags treesitter_c_parity -run '^$' -bench 'BenchmarkParityRealCorpusParse(Full|IncrementalSingleByteEdit|IncrementalNoEdit)$' -benchmem -count=$COUNT -benchtime=$BENCHTIME -cpu=$GOMAXPROCS_VALUE -timeout=$TEST_TIMEOUT -v"
  if [[ -n "$BENCH_TIMEOUT" ]]; then
    inner_cmd="cd /workspace/cgo_harness && /usr/bin/time -v timeout --foreground --kill-after=30s $BENCH_TIMEOUT $bench_cmd"
  else
    inner_cmd="cd /workspace/cgo_harness && /usr/bin/time -v $bench_cmd"
  fi
  runner_args=(
    --out-root "$DOCKER_OUT"
    --label "real-corpus-bench-$lang"
    --memory "$MEMORY_LIMIT"
    --cpus "$CPUS_LIMIT"
    --gomemlimit "$GOMEMLIMIT_VALUE"
    --timeout "$TEST_TIMEOUT"
    --pids "$PIDS_LIMIT"
  )
  if [[ -n "$CPUSET_CPUS" ]]; then
    runner_args+=(--cpuset-cpus "$CPUSET_CPUS")
  fi
  if [[ "$build_mode" == "no-build" ]]; then
    runner_args+=(--no-build)
  fi
  if [[ -n "$CORPUS_SOURCES_ROOT" ]]; then
    runner_args+=(--mount "$CORPUS_SOURCES_ROOT:/corpus-sources:ro")
  fi
  if [[ -n "$CORPUS_SOURCE_LOCK" ]]; then
    runner_args+=(--mount "$CORPUS_SOURCE_LOCK:/corpus-source.lock:ro")
  fi
  "$RUNNER" "${runner_args[@]}" -- "$inner_cmd" 2>&1 | tee "$RUN_DIR/$lang.runner.log"
}

run_language_serial() {
  local lang="$1"
  local build_mode="$2"
  if run_language "$lang" "$build_mode"; then
    :
  else
    failures+=("$lang")
    if [[ "$KEEP_GOING" == "0" ]]; then
      return 1
    fi
  fi
  return 0
}

if [[ "$JOBS" -eq 1 || ${#LANGS[@]} -eq 1 ]]; then
  for lang in "${LANGS[@]}"; do
    build_mode="build"
    if [[ ${#build_flag[@]} -gt 0 ]]; then
      build_mode="no-build"
    fi
    if ! run_language_serial "$lang" "$build_mode" && [[ "$KEEP_GOING" == "0" ]]; then
      break
    fi
    build_flag=(--no-build)
  done
else
  start_index=0
  if [[ "$BUILD_IMAGE" == "1" ]]; then
    if ! run_language_serial "${LANGS[0]}" "build" && [[ "$KEEP_GOING" == "0" ]]; then
      start_index=${#LANGS[@]}
    else
      start_index=1
    fi
  fi

  declare -a pids=()
  declare -a pid_langs=()
  stop_scheduling=0

  wait_for_one() {
    local pid lang rc
    if [[ ${#pids[@]} -eq 0 ]]; then
      return 0
    fi
    pid="${pids[0]}"
    lang="${pid_langs[0]}"
    if wait "$pid"; then
      rc=0
    else
      rc=$?
    fi
    echo "[done] real-corpus bench: $lang exit=$rc"
    if [[ "$rc" -ne 0 ]]; then
      failures+=("$lang")
      if [[ "$KEEP_GOING" == "0" ]]; then
        stop_scheduling=1
      fi
    fi
    pids=("${pids[@]:1}")
    pid_langs=("${pid_langs[@]:1}")
  }

  for ((i = start_index; i < ${#LANGS[@]}; i++)); do
    while [[ ${#pids[@]} -ge "$JOBS" ]]; do
      wait_for_one
      if [[ "$stop_scheduling" == "1" ]]; then
        break 2
      fi
    done
    lang="${LANGS[$i]}"
    echo "[start] real-corpus bench: $lang"
    run_language "$lang" "no-build" &
    pids+=("$!")
    pid_langs+=("$lang")
  done

  while [[ ${#pids[@]} -gt 0 ]]; do
    wait_for_one
  done
fi

REPORT_FAILED=0
if [[ -n "$(find "$DOCKER_OUT" -name container.log -type f -print -quit)" ]]; then
  if (
    cd "$REPO_ROOT/cgo_harness"
    go run ./cmd/real_corpus_bench_report \
      -input "$DOCKER_OUT" \
      -out-json "$RUN_DIR/real_corpus_bench_report.json" \
      -out-md "$RUN_DIR/REAL_CORPUS_BENCH_REPORT.md"
  ); then
    echo "report: $RUN_DIR/REAL_CORPUS_BENCH_REPORT.md"
  else
    echo "report generation failed; inspect logs under $DOCKER_OUT" >&2
    REPORT_FAILED=1
  fi
else
  echo "no container logs found under $DOCKER_OUT" >&2
  REPORT_FAILED=1
fi

if [[ ${#failures[@]} -gt 0 ]]; then
  printf '%s\n' "${failures[@]}" >"$RUN_DIR/failed_languages.txt"
  {
    echo "run_end_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "failed_languages=${failures[*]}"
    echo "report_failed=$REPORT_FAILED"
  } >>"$RUN_DIR/matrix_metadata.txt"
  echo "failed languages: ${failures[*]}" >&2
  exit 1
fi

if [[ "$REPORT_FAILED" != "0" ]]; then
  {
    echo "run_end_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "failed_languages="
    echo "report_failed=1"
  } >>"$RUN_DIR/matrix_metadata.txt"
  exit 1
fi

{
  echo "run_end_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "failed_languages="
  echo "report_failed=0"
} >>"$RUN_DIR/matrix_metadata.txt"

echo "real-corpus bench matrix complete"
echo "artifacts: $RUN_DIR"
