#!/usr/bin/env bash
# Deterministic single-grammar parser integrity wringer.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  cgo_harness/docker/run_grammar_integrity_wringer.sh <grammar> [out_dir]

Runs one grammar through an isolated baseline tier-scan pass, then replays
suspicious files through controlled parser variants and first-diff diagnostics.

Env:
  GTS_CORPUS_DIR                  corpus root with per-grammar subdirs (required)
  GTS_WRINGER_N                   baseline selected files (default 10)
  GTS_WRINGER_ALL_FILES           set 1 to baseline-select every eligible
                                    corpus file; not implied by full modes
  GTS_WRINGER_ROUNDS              per-file parse rounds (default 1)
  GTS_WRINGER_TIMEOUT             per-file timeout seconds (default 60)
  GTS_WRINGER_KILL_AFTER          timeout SIGKILL grace (default 10s)
  GTS_WRINGER_HEARTBEAT           wringer child heartbeat seconds; 0 disables (default 15)
  GTS_WRINGER_MODE                full-integrity for every selected frame through
                                    baseline, variants, firstdiff, parser progress,
                                    and strict assertions by default
  GTS_WRINGER_FULL                set 1 for full/every-frame replay preset
  GTS_WRINGER_PROFILE             set full for the same full/every-frame replay preset
  GTS_WRINGER_PLAN_ONLY           set 1 to only build the deterministic frame
                                    catalog/control artifacts; no variant or
                                    first-diff parser children run
  GTS_WRINGER_ASSERTS             comma/space list of closed,plan,telemetry,artifacts
                                    after summary; default strict when summary runs, 0 disables
  GTS_WRINGER_REUSE_BASELINE      reuse existing baseline artifacts when set to 1
  GTS_WRINGER_STAGES              all, or comma/space list of baseline variants firstdiff summary (default all)
  GTS_WRINGER_VARIANT_SCOPE       suspicious or all (default suspicious)
  GTS_WRINGER_VARIANTS            variants to run (default "stack2 stack8 node3 forest")
                                    available: stack2 stack8 stack48 node3 forest_off forest merge1 merge24
                                    faithful pre_mat mat_off crecovery_all crecovery_off
  GTS_WRINGER_BASELINE_FRAMES     comma-separated baseline frame selectors, e.g.
                                    1,3-5,sha256:abc123,base:file.json,path:/subdir/
  GTS_WRINGER_TIER_SCAN_MAX_FRAME_ROWS
                                  optional baseline tier-scan dense frame/event
                                    row cap; passed as GTS_TIER_SCAN_MAX_FRAME_ROWS
                                    (tier-scan default is bounded)
  GTS_WRINGER_KEEP_PARSER_PROGRESS_ROWS
                                  optional baseline tier-scan dense parser-progress
                                    row cap; passed as
                                    GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS
  GTS_WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES
                                  optional baseline tier-scan raw measure log
                                    byte cap; passed as
                                    GTS_TIER_SCAN_MAX_RAW_LOG_BYTES
                                    (tier-scan default is 1048576; 0 unlimited)
  GTS_WRINGER_VARIANT_FRAMES      replay only these variant frame selectors
  GTS_WRINGER_FIRSTDIFF_FRAMES    replay only these first-diff frame selectors
  GTS_WRINGER_FRAMES              fallback frame selector for variants and first-diff
  GTS_WRINGER_CLEAR_FRAMES        set 1 to ignore all wringer frame selector env vars
  GTS_WRINGER_MAX_SUSPICIOUS      max suspicious files for default suspicious
                                    variant replay; 0 is unlimited (default 0)
  GTS_WRINGER_MAX_DIAG_FILES      max suspicious files for first-diff (default 10)
  GTS_WRINGER_DIAG_TIMEOUT        first-diff timeout seconds (default GTS_WRINGER_TIMEOUT)
  GTS_WRINGER_GLR_TRACE           pass REPRO_GLR_TRACE=1 to first-diff when set to 1
  GTS_WRINGER_DEBUG_DFA           pass REPRO_DEBUG_DFA=1 to first-diff when set to 1
  GTS_WRINGER_PARSE_PROGRESS      set 1 to pass GOT_PARSE_PROGRESS=1 to baseline
                                    tier-scan and variant measure children
  GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS
                                    optional GOT_PARSE_PROGRESS_INTERVAL_MS value
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  usage >&2
  exit 2
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HARNESS="$REPO_ROOT/cgo_harness"
RUN_TIER_SCAN="$HARNESS/docker/run_tier_scan.sh"
SUMMARY_HELPER="$HARNESS/tier_scan/wringer_summary.py"
GRAMMAR="$1"
CORPUS="${GTS_CORPUS_DIR:?set GTS_CORPUS_DIR to the corpus_sources root}"
N="${GTS_WRINGER_N:-10}"
WRINGER_ALL_FILES="${GTS_WRINGER_ALL_FILES:-0}"
ROUNDS="${GTS_WRINGER_ROUNDS:-1}"
PER_FILE_TIMEOUT="${GTS_WRINGER_TIMEOUT:-60}"
KILL_AFTER="${GTS_WRINGER_KILL_AFTER:-10s}"
WRINGER_HEARTBEAT="${GTS_WRINGER_HEARTBEAT:-15}"
REUSE_BASELINE="${GTS_WRINGER_REUSE_BASELINE:-0}"
STAGES="${GTS_WRINGER_STAGES:-all}"
VARIANT_SCOPE="${GTS_WRINGER_VARIANT_SCOPE:-suspicious}"
VARIANTS="${GTS_WRINGER_VARIANTS:-stack2 stack8 node3 forest}"
BASELINE_FRAMES="${GTS_WRINGER_BASELINE_FRAMES:-}"
WRINGER_TIER_SCAN_MAX_FRAME_ROWS="${GTS_WRINGER_TIER_SCAN_MAX_FRAME_ROWS:-}"
WRINGER_KEEP_PARSER_PROGRESS_ROWS="${GTS_WRINGER_KEEP_PARSER_PROGRESS_ROWS:-}"
WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES="${GTS_WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES:-}"
VARIANT_FRAMES="${GTS_WRINGER_VARIANT_FRAMES:-${GTS_WRINGER_FRAMES:-}}"
FIRSTDIFF_FRAMES="${GTS_WRINGER_FIRSTDIFF_FRAMES:-${GTS_WRINGER_FRAMES:-}}"
VARIANT_FRAMES_EXPLICIT=0
if [ -n "${GTS_WRINGER_VARIANT_FRAMES+x}" ] || [ -n "${GTS_WRINGER_FRAMES+x}" ]; then
  VARIANT_FRAMES_EXPLICIT=1
fi
FIRSTDIFF_FRAMES_EXPLICIT=0
if [ -n "${GTS_WRINGER_FIRSTDIFF_FRAMES+x}" ] || [ -n "${GTS_WRINGER_FRAMES+x}" ]; then
  FIRSTDIFF_FRAMES_EXPLICIT=1
fi
WRINGER_PROFILE="${GTS_WRINGER_PROFILE:-}"
WRINGER_MODE="${GTS_WRINGER_MODE:-}"
WRINGER_PLAN_ONLY="${GTS_WRINGER_PLAN_ONLY:-0}"
PARSE_PROGRESS="${GTS_WRINGER_PARSE_PROGRESS:-0}"
PARSE_PROGRESS_INTERVAL_MS="${GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS:-}"
WRINGER_CLEAR_FRAMES="${GTS_WRINGER_CLEAR_FRAMES:-0}"
if [ "$WRINGER_CLEAR_FRAMES" = "1" ]; then
  BASELINE_FRAMES=""
  VARIANT_FRAMES=""
  FIRSTDIFF_FRAMES=""
  VARIANT_FRAMES_EXPLICIT=0
  FIRSTDIFF_FRAMES_EXPLICIT=0
fi
case "$WRINGER_PROFILE" in
  ""|full) ;;
  *)
    echo "unknown GTS_WRINGER_PROFILE=$WRINGER_PROFILE (want full)" >&2
    exit 2
    ;;
esac
case "$WRINGER_MODE" in
  ""|default|full-integrity) ;;
  *)
    echo "unknown GTS_WRINGER_MODE=$WRINGER_MODE (want full-integrity)" >&2
    exit 2
    ;;
esac
case "$WRINGER_PLAN_ONLY" in
  0|1) ;;
  *)
    echo "unknown GTS_WRINGER_PLAN_ONLY=$WRINGER_PLAN_ONLY (want 0 or 1)" >&2
    exit 2
    ;;
esac
case "$WRINGER_ALL_FILES" in
  0|1) ;;
  *)
    echo "unknown GTS_WRINGER_ALL_FILES=$WRINGER_ALL_FILES (want 0 or 1)" >&2
    exit 2
    ;;
esac
case "$WRINGER_TIER_SCAN_MAX_FRAME_ROWS" in
  ""|*[!0-9]*)
    if [ -n "$WRINGER_TIER_SCAN_MAX_FRAME_ROWS" ]; then
      echo "invalid GTS_WRINGER_TIER_SCAN_MAX_FRAME_ROWS=$WRINGER_TIER_SCAN_MAX_FRAME_ROWS (want nonnegative integer; 0 means unlimited)" >&2
      exit 2
    fi
    ;;
esac
case "$WRINGER_KEEP_PARSER_PROGRESS_ROWS" in
  ""|*[!0-9]*)
    if [ -n "$WRINGER_KEEP_PARSER_PROGRESS_ROWS" ]; then
      echo "invalid GTS_WRINGER_KEEP_PARSER_PROGRESS_ROWS=$WRINGER_KEEP_PARSER_PROGRESS_ROWS (want nonnegative integer)" >&2
      exit 2
    fi
    ;;
esac
case "$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES" in
  ""|*[!0-9]*)
    if [ -n "$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES" ]; then
      echo "invalid GTS_WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES=$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES (want nonnegative integer; 0 means unlimited)" >&2
      exit 2
    fi
    ;;
esac
if [ "$WRINGER_MODE" = "full-integrity" ]; then
  if [ -z "${GTS_WRINGER_VARIANT_SCOPE+x}" ]; then
    VARIANT_SCOPE="all"
  fi
  if [ "$WRINGER_CLEAR_FRAMES" = "1" ] || { [ -z "${GTS_WRINGER_FIRSTDIFF_FRAMES+x}" ] && [ -z "${GTS_WRINGER_FRAMES+x}" ]; }; then
    FIRSTDIFF_FRAMES="all"
  fi
  if [ -z "${GTS_WRINGER_PARSE_PROGRESS+x}" ]; then
    PARSE_PROGRESS="1"
  fi
fi
if [ "${GTS_WRINGER_FULL:-0}" = "1" ] || [ "$WRINGER_PROFILE" = "full" ]; then
  if [ -z "${GTS_WRINGER_VARIANT_SCOPE+x}" ]; then
    VARIANT_SCOPE="all"
  fi
  if [ "$WRINGER_CLEAR_FRAMES" = "1" ] || { [ -z "${GTS_WRINGER_FIRSTDIFF_FRAMES+x}" ] && [ -z "${GTS_WRINGER_FRAMES+x}" ]; }; then
    FIRSTDIFF_FRAMES="all"
  fi
fi
MAX_SUSPICIOUS="${GTS_WRINGER_MAX_SUSPICIOUS:-0}"
case "$MAX_SUSPICIOUS" in
  ''|*[!0-9]*)
    echo "invalid GTS_WRINGER_MAX_SUSPICIOUS=$MAX_SUSPICIOUS (want nonnegative integer; 0 means unlimited)" >&2
    exit 2
    ;;
esac
if [ "$MAX_SUSPICIOUS" -gt 0 ] && [ -z "${GTS_WRINGER_MAX_DIAG_FILES+x}" ]; then
  MAX_DIAG_FILES="$MAX_SUSPICIOUS"
else
  MAX_DIAG_FILES="${GTS_WRINGER_MAX_DIAG_FILES:-10}"
fi
DIAG_TIMEOUT="${GTS_WRINGER_DIAG_TIMEOUT:-$PER_FILE_TIMEOUT}"
OUT_DIR="${2:-$HARNESS/harness_out/grammar_integrity_wringer/${GRAMMAR}-$(date -u +%Y%m%dT%H%M%SZ)}"

case "$WRINGER_HEARTBEAT" in
  ''|*[!0-9]*)
    echo "invalid GTS_WRINGER_HEARTBEAT=$WRINGER_HEARTBEAT (want nonnegative integer seconds)" >&2
    exit 2
    ;;
esac
case "$PARSE_PROGRESS" in
  0|1) ;;
  *)
    echo "unknown GTS_WRINGER_PARSE_PROGRESS=$PARSE_PROGRESS (want 0 or 1)" >&2
    exit 2
    ;;
esac

if [[ "$OUT_DIR" != /* ]]; then
  OUT_DIR="$REPO_ROOT/$OUT_DIR"
fi

case "$VARIANT_SCOPE" in
  suspicious|all) ;;
  *)
    echo "unknown GTS_WRINGER_VARIANT_SCOPE=$VARIANT_SCOPE (want suspicious or all)" >&2
    exit 2
    ;;
esac

case "$REUSE_BASELINE" in
  0|1) ;;
  *)
    echo "unknown GTS_WRINGER_REUSE_BASELINE=$REUSE_BASELINE (want 0 or 1)" >&2
    exit 2
    ;;
esac

stage_baseline=0
stage_variants=0
stage_firstdiff=0
stage_summary=0
if [ "$STAGES" = "all" ]; then
  stage_baseline=1
  stage_variants=1
  stage_firstdiff=1
  stage_summary=1
else
  while IFS= read -r stage; do
    [ -z "$stage" ] && continue
    case "$stage" in
      baseline) stage_baseline=1 ;;
      variants) stage_variants=1 ;;
      firstdiff) stage_firstdiff=1 ;;
      summary) stage_summary=1 ;;
      *)
        echo "unknown GTS_WRINGER_STAGES entry: $stage (want baseline, variants, firstdiff, summary, or all)" >&2
        exit 2
        ;;
    esac
  done < <(printf '%s\n' "$STAGES" | tr ',[:space:]' '\n')
fi
if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
  stage_baseline=1
  stage_variants=0
  stage_firstdiff=0
  stage_summary=1
fi

if [ "$stage_baseline" -eq 0 ] && [ "$stage_variants" -eq 0 ] && [ "$stage_firstdiff" -eq 0 ] && [ "$stage_summary" -eq 0 ]; then
  echo "GTS_WRINGER_STAGES selected no stages" >&2
  exit 2
fi
if [ -z "${GTS_WRINGER_ASSERTS+x}" ]; then
  if [ "$stage_summary" -eq 1 ]; then
    if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
      WRINGER_ASSERTS="0"
    else
      WRINGER_ASSERTS="closed,plan,telemetry,artifacts"
    fi
  else
    WRINGER_ASSERTS="0"
  fi
else
  WRINGER_ASSERTS="$GTS_WRINGER_ASSERTS"
fi

variant_env() { # $1=variant; prints NUL-free env assignment words
  case "$1" in
    stack2) printf '%s\n' 'GOT_GLR_MAX_STACKS=2' ;;
    stack8) printf '%s\n' 'GOT_GLR_MAX_STACKS=8' ;;
    stack48) printf '%s\n' 'GOT_GLR_MAX_STACKS=48' ;;
    node3) printf '%s\n' 'GOT_PARSE_NODE_LIMIT_SCALE=3' ;;
    forest_off) printf '%s\n' 'GOT_GLR_FOREST=0' ;;
    forest) printf '%s\n' 'REPRO_FOREST=1' ;;
    merge1) printf '%s\n' 'GOT_GLR_MAX_MERGE_PER_KEY=1' ;;
    merge24) printf '%s\n' 'GOT_GLR_MAX_MERGE_PER_KEY=24' ;;
    faithful) printf '%s\n' 'GOT_FAITHFUL_CONDENSE=1' ;;
    pre_mat) printf '%s\n' 'GOT_GLR_V2_PRE_MATERIALIZATION_DIAG=1' ;;
    mat_off)
      printf '%s\n' \
        'GOT_GLR_V2_PENDING_PARENTS=0' \
        'GOT_GLR_V2_FINAL_CHILD_REFS=0' \
        'GOT_GLR_V2_COMPACT_FULL_LEAVES=0'
      ;;
    crecovery_all) printf '%s\n' 'GOT_C_RECOVERY=all' ;;
    crecovery_off) printf '%s\n' 'GOT_C_RECOVERY=0' ;;
    *)
      echo "unknown wringer variant: $1" >&2
      return 2
      ;;
  esac
}

for variant in $VARIANTS; do
  variant_env "$variant" >/dev/null
done

mkdir -p "$OUT_DIR"
BASELINE_DIR="$OUT_DIR/baseline"
VARIANTS_DIR="$OUT_DIR/variants"
FIRSTDIFF_DIR="$OUT_DIR/firstdiff"
COMMANDS_LOG="$OUT_DIR/commands.log"
MANIFEST="$OUT_DIR/wringer_manifest.json"
ACTIVE_FILE="$OUT_DIR/wringer_active.txt"
ACTIVE_JSON="$OUT_DIR/wringer_active.json"
EVENTS_LOG="$OUT_DIR/wringer_events.jsonl"
if [ "$stage_baseline" -eq 1 ] && [ "$REUSE_BASELINE" -ne 1 ]; then
  : > "$COMMANDS_LOG"
  rm -rf "$VARIANTS_DIR" "$FIRSTDIFF_DIR"
  rm -f \
    "$ACTIVE_FILE" \
    "$ACTIVE_JSON" \
    "$EVENTS_LOG" \
    "$OUT_DIR/wringer_summary.json" \
    "$OUT_DIR/wringer_summary.md" \
    "$OUT_DIR/wringer_frames.jsonl" \
    "$OUT_DIR/frame_matrix.jsonl" \
    "$OUT_DIR/frame_catalog.jsonl" \
    "$OUT_DIR/wringer_plan.jsonl" \
    "$OUT_DIR/wringer_plan.json" \
    "$OUT_DIR/variant_replay_files.jsonl" \
    "$OUT_DIR/firstdiff_replay_files.jsonl" \
    "$OUT_DIR"/selected_files.* \
    "$OUT_DIR"/suspicious_files.*
else
  touch "$COMMANDS_LOG"
  if [ "$stage_variants" -eq 1 ]; then
    rm -rf "$VARIANTS_DIR"
  fi
  if [ "$stage_firstdiff" -eq 1 ]; then
    rm -rf "$FIRSTDIFF_DIR"
  fi
  if [ "$stage_summary" -eq 1 ] || [ "$stage_variants" -eq 1 ] || [ "$stage_firstdiff" -eq 1 ]; then
    rm -f \
      "$OUT_DIR/wringer_summary.json" \
      "$OUT_DIR/wringer_summary.md" \
      "$OUT_DIR/wringer_frames.jsonl" \
      "$OUT_DIR/frame_matrix.jsonl" \
      "$OUT_DIR/frame_catalog.jsonl" \
      "$OUT_DIR/wringer_plan.jsonl" \
      "$OUT_DIR/wringer_plan.json" \
      "$OUT_DIR/variant_replay_files.jsonl" \
      "$OUT_DIR/firstdiff_replay_files.jsonl"
  fi
  if [ "$stage_variants" -eq 1 ] || [ "$stage_firstdiff" -eq 1 ] || [ "$stage_summary" -eq 1 ]; then
    rm -f \
      "$OUT_DIR"/selected_files.* \
      "$OUT_DIR"/suspicious_files.*
  fi
fi
mkdir -p "$BASELINE_DIR" "$VARIANTS_DIR" "$FIRSTDIFF_DIR"

write_active() {
  local state="$1"
  local stage="${2:-}"
  local mode="${3:-}"
  local variant="${4:-}"
  local ordinal="${5:-0}"
  local corpus_kind="${6:-}"
  local corpus_root="${7:-}"
  local file_path="${8:-}"
  local log_path="${9:-}"
  local timeout_value="${10:-}"
  local replay_command="${11:-}"
  python3 - "$ACTIVE_FILE" "$ACTIVE_JSON" "$state" "$GRAMMAR" "$stage" "$mode" "$variant" "$ordinal" "$corpus_kind" \
    "$corpus_root" "$file_path" "$log_path" "$timeout_value" "$replay_command" <<'PY'
import json
import sys
import time
from pathlib import Path

ordinal_text = sys.argv[8]
try:
    ordinal = int(ordinal_text)
except ValueError:
    ordinal = 0
data = {
    "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "state": sys.argv[3],
    "grammar": sys.argv[4],
    "stage": sys.argv[5],
    "mode": sys.argv[6],
    "variant": sys.argv[7],
    "ordinal": ordinal,
    "corpus_kind": sys.argv[9],
    "corpus_root": sys.argv[10],
    "path": sys.argv[11],
    "log": sys.argv[12],
    "timeout": sys.argv[13],
    "replay_command": sys.argv[14],
}
payload = json.dumps(data, sort_keys=True) + "\n"
Path(sys.argv[1]).write_text(payload, encoding="utf-8")
Path(sys.argv[2]).write_text(payload, encoding="utf-8")
PY
}

append_event() {
  local event="$1"
  local stage="$2"
  local mode="$3"
  local variant="$4"
  local ordinal="$5"
  local corpus_kind="$6"
  local corpus_root="$7"
  local file_path="$8"
  local log_path="$9"
  local timeout_value="${10}"
  local replay_command="${11}"
  local rc="${12:-}"
  local timeout_flag="${13:-0}"
  local started_epoch="${14:-}"
  local child_pid="${15:-}"
  python3 - "$EVENTS_LOG" "$event" "$GRAMMAR" "$stage" "$mode" "$variant" "$ordinal" "$corpus_kind" \
    "$corpus_root" "$file_path" "$log_path" "$timeout_value" "$replay_command" "$rc" "$timeout_flag" \
    "$started_epoch" "$child_pid" <<'PY'
import json
import os
import sys
import time
from pathlib import Path

ordinal_text = sys.argv[7]
try:
    ordinal = int(ordinal_text)
except ValueError:
    ordinal = 0
rc_text = sys.argv[14]
timeout_flag = sys.argv[15] == "1"
started_text = sys.argv[16]
child_pid_text = sys.argv[17]
now = time.time()
now_int = int(now)
started_epoch = None
try:
    if started_text:
        started_epoch = float(started_text)
except ValueError:
    started_epoch = None
record = {
    "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "epoch_s": now,
    "event": sys.argv[2],
    "grammar": sys.argv[3],
    "stage": sys.argv[4],
    "mode": sys.argv[5],
    "variant": sys.argv[6],
    "ordinal": ordinal,
    "corpus_kind": sys.argv[8],
    "corpus_root": sys.argv[9],
    "path": sys.argv[10],
    "log": sys.argv[11],
    "timeout_seconds": sys.argv[12],
    "replay_command": sys.argv[13],
    "timeout": timeout_flag,
}
log_path = Path(sys.argv[11]) if sys.argv[11] else None
if log_path:
    try:
        stat = log_path.stat()
        record["log_bytes"] = stat.st_size
        record["log_age_s"] = max(0.0, now - stat.st_mtime)
        record["progress_age_s"] = record["log_age_s"]
    except OSError:
        record["log_bytes"] = 0
if child_pid_text:
    try:
        record["child_pid"] = int(child_pid_text)
    except ValueError:
        record["child_pid"] = child_pid_text
if started_epoch is not None:
    record["started_at_epoch_s"] = started_epoch
    record["elapsed_s"] = max(0.0, now - started_epoch)
if sys.argv[2] == "START":
    record["started_at_epoch_s"] = started_epoch if started_epoch is not None else now
    record["started_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(record["started_at_epoch_s"]))
elif sys.argv[2] in {"END", "FAIL", "TIMEOUT"}:
    record["finished_at_epoch_s"] = now
    record["finished_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(now_int))
    if started_epoch is not None:
        record["duration_s"] = max(0.0, now - started_epoch)
if rc_text:
    record["rc"] = int(rc_text)
with Path(sys.argv[1]).open("a", encoding="utf-8") as f:
    f.write(json.dumps(record, sort_keys=True) + "\n")
PY
}

terminal_event_name() {
  local rc="$1"
  if is_timeout_status "$rc"; then
    printf 'TIMEOUT'
  elif [ "$rc" -eq 0 ]; then
    printf 'END'
  else
    printf 'FAIL'
  fi
}

current_stage=""
current_mode=""
current_variant=""
current_ordinal=0
current_corpus_kind=""
current_corpus_root=""
current_file_path=""
current_log_path=""
current_timeout=""
current_replay_command=""
current_child_pid=""
current_meta_path=""
current_started_epoch=""
active_child_in_progress=0
active_child_terminal_recorded=0
wringer_exit_finalized=0

remember_active_child() {
  current_stage="$1"
  current_mode="$2"
  current_variant="$3"
  current_ordinal="$4"
  current_corpus_kind="$5"
  current_corpus_root="$6"
  current_file_path="$7"
  current_log_path="$8"
  current_timeout="$9"
  current_replay_command="${10}"
  current_meta_path="${11:-}"
  current_started_epoch="$(date -u +%s)"
  active_child_in_progress=1
  active_child_terminal_recorded=0
}

mark_active_child_terminal_recorded() {
  active_child_terminal_recorded=1
}

clear_active_child() {
  active_child_in_progress=0
  active_child_terminal_recorded=0
  current_child_pid=""
  current_meta_path=""
  current_started_epoch=""
}

record_active_child_terminal() {
  local rc="$1"
  local timeout_flag terminal
  timeout_flag=0
  if is_timeout_status "$rc"; then
    timeout_flag=1
  fi
  terminal="$(terminal_event_name "$rc")"
  if [ "$active_child_terminal_recorded" -eq 0 ]; then
    append_event "$terminal" "$current_stage" "$current_mode" "$current_variant" "$current_ordinal" \
      "$current_corpus_kind" "$current_corpus_root" "$current_file_path" "$current_log_path" \
      "$current_timeout" "$current_replay_command" "$rc" "$timeout_flag" "$current_started_epoch" "$current_child_pid"
    mark_active_child_terminal_recorded
  fi
}

emit_active_heartbeat() {
  append_event HEARTBEAT "$current_stage" "$current_mode" "$current_variant" "$current_ordinal" \
    "$current_corpus_kind" "$current_corpus_root" "$current_file_path" "$current_log_path" \
    "$current_timeout" "$current_replay_command" "" 0 "$current_started_epoch" "$current_child_pid"
}

wait_for_active_child() {
  local pid="$1"
  local interval="$WRINGER_HEARTBEAT"
  local next_heartbeat now tick_pid wait_rc completed
  if [ "$interval" -le 0 ]; then
    wait "$pid"
    return $?
  fi
  next_heartbeat=$(( $(date -u +%s) + interval ))
  while true; do
    sleep 1 &
    tick_pid=$!
    completed=""
    set +e
    wait -n -p completed "$pid" "$tick_pid"
    wait_rc=$?
    if [ "$completed" = "$pid" ]; then
      kill "$tick_pid" 2>/dev/null || true
      wait "$tick_pid" 2>/dev/null || true
      return "$wait_rc"
    fi
    if kill -0 "$pid" 2>/dev/null; then
      now="$(date -u +%s)"
      if [ "$now" -ge "$next_heartbeat" ]; then
        emit_active_heartbeat
        next_heartbeat=$(( now + interval ))
      fi
      continue
    fi
    kill "$tick_pid" 2>/dev/null || true
    wait "$tick_pid" 2>/dev/null || true
    wait "$pid"
    return $?
  done
}

complete_active_child() {
  clear_active_child
  set_idle_active
}

set_idle_active() {
  write_active idle "" "" "" 0 "" "" "" "" "" ""
}

finalize_active_on_exit() {
  local rc state terminal timeout_flag
  rc="$1"
  if [ "$wringer_exit_finalized" -eq 1 ]; then
    return 0
  fi
  wringer_exit_finalized=1
  if [ "$rc" -eq 0 ]; then
    if [ "$active_child_in_progress" -eq 0 ]; then
      set_idle_active
    fi
    return 0
  fi
  state="failed"
  if [ "$rc" -eq 130 ] || [ "$rc" -eq 143 ]; then
    state="aborted"
  fi
  if [ "$active_child_in_progress" -eq 1 ]; then
    if [ -n "$current_child_pid" ]; then
      kill "$current_child_pid" 2>/dev/null || true
    fi
    if [ "$active_child_terminal_recorded" -eq 0 ]; then
      timeout_flag=0
      if is_timeout_status "$rc"; then
        timeout_flag=1
      fi
      terminal="$(terminal_event_name "$rc")"
      append_event "$terminal" "$current_stage" "$current_mode" "$current_variant" "$current_ordinal" \
        "$current_corpus_kind" "$current_corpus_root" "$current_file_path" "$current_log_path" \
        "$current_timeout" "$current_replay_command" "$rc" "$timeout_flag" "$current_started_epoch" "$current_child_pid" || true
      if [ -n "$current_meta_path" ]; then
        write_run_meta "$current_meta_path" "$current_stage" "$current_variant" "$current_ordinal" \
          "$current_corpus_kind" "$current_corpus_root" "$current_file_path" "$current_log_path" \
          "$rc" "$timeout_flag" "$current_replay_command" "$current_started_epoch" || true
      fi
    fi
  fi
  if [ "$active_child_in_progress" -eq 1 ]; then
    write_active "$state" "$current_stage" "$current_mode" "$current_variant" "$current_ordinal" \
      "$current_corpus_kind" "$current_corpus_root" "$current_file_path" "$current_log_path" \
      "$current_timeout" "$current_replay_command" || true
  else
    write_active "$state" "" "" "" 0 "" "" "" "" "" "" || true
  fi
}
trap 'finalize_active_on_exit $?' EXIT
trap 'finalize_active_on_exit 130; exit 130' INT
trap 'finalize_active_on_exit 143; exit 143' TERM
trap 'finalize_active_on_exit 129; exit 129' HUP
set_idle_active

shell_quote() {
  printf '%q' "$1"
}

shell_join() {
  python3 - "$@" <<'PY'
import shlex
import sys

print(shlex.join(sys.argv[1:]), end="")
PY
}

append_parse_progress_env() {
  if [ "$PARSE_PROGRESS" = "1" ]; then
    printf '%s\n' "GOT_PARSE_PROGRESS=1"
    if [ -n "$PARSE_PROGRESS_INTERVAL_MS" ]; then
      printf '%s\n' "GOT_PARSE_PROGRESS_INTERVAL_MS=$PARSE_PROGRESS_INTERVAL_MS"
    fi
  fi
}

console_printf() {
  (trap '' PIPE; printf "$@") 2>/dev/null || true
}

console_line() {
  if [ "$#" -eq 0 ]; then
    console_printf '\n'
  else
    console_printf '%s\n' "$*"
  fi
}

log_command() {
  local label="$1"
  shift
  {
    printf '[%s] %s' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$label"
    for arg in "$@"; do
      printf ' %q' "$arg"
    done
    printf '\n'
  } >> "$COMMANDS_LOG"
}

jsonl_record_fields() {
  python3 - "$1" <<'PY'
import json
import sys

record = json.loads(sys.argv[1])
fields = [
    record.get("ordinal", ""),
    record.get("grammar", ""),
    record.get("corpus_kind", ""),
    record.get("corpus_root", ""),
    record.get("path", ""),
]
sys.stdout.buffer.write(b"\0".join(str(field).encode("utf-8") for field in fields) + b"\0")
PY
}

write_manifest() {
  local baseline_rc="${1:-}"
  local infra_status="${2:-0}"
  python3 - "$MANIFEST" "$GRAMMAR" "$OUT_DIR" "$BASELINE_DIR" "$COMMANDS_LOG" "$baseline_rc" "$infra_status" \
    "$N" "$ROUNDS" "$PER_FILE_TIMEOUT" "$KILL_AFTER" "$VARIANT_SCOPE" "$VARIANTS" "$MAX_DIAG_FILES" "$DIAG_TIMEOUT" \
    "$REUSE_BASELINE" "$STAGES" "$BASELINE_FRAMES" "$VARIANT_FRAMES" "$FIRSTDIFF_FRAMES" "$WRINGER_MODE" "$WRINGER_PROFILE" \
    "$stage_baseline" "$stage_variants" "$stage_firstdiff" "$stage_summary" \
    "${GTS_WRINGER_GLR_TRACE:-0}" "${GTS_WRINGER_DEBUG_DFA:-0}" "$PARSE_PROGRESS" "$PARSE_PROGRESS_INTERVAL_MS" "$WRINGER_PLAN_ONLY" "$WRINGER_HEARTBEAT" "$WRINGER_ALL_FILES" \
    "$MAX_SUSPICIOUS" "$VARIANT_FRAMES_EXPLICIT" "$FIRSTDIFF_FRAMES_EXPLICIT" "$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES" \
    "$WRINGER_TIER_SCAN_MAX_FRAME_ROWS" "$WRINGER_KEEP_PARSER_PROGRESS_ROWS" <<'PY'
import json
import sys
import time
from pathlib import Path

path = Path(sys.argv[1])
baseline_rc = sys.argv[6]
infra_status = sys.argv[7]
manifest = {
    "grammar": sys.argv[2],
    "out_dir": sys.argv[3],
    "baseline_run_dir": sys.argv[4],
    "commands_log": sys.argv[5],
    "baseline_exit_status": int(baseline_rc) if baseline_rc else None,
    "infra_status": int(infra_status),
    "config": {
        "n": int(sys.argv[8]),
        "all_files": sys.argv[33] == "1",
        "rounds": int(sys.argv[9]),
        "timeout": sys.argv[10],
        "kill_after": sys.argv[11],
        "variant_scope": sys.argv[12],
        "variants": sys.argv[13].split(),
        "max_diag_files": int(sys.argv[14]),
        "max_suspicious": int(sys.argv[34]),
        "diag_timeout": sys.argv[15],
        "reuse_baseline": sys.argv[16] == "1",
        "stages": sys.argv[17],
        "baseline_frames": sys.argv[18],
        "variant_frames": sys.argv[19],
        "variant_frames_explicit": sys.argv[35] == "1",
        "firstdiff_frames": sys.argv[20],
        "firstdiff_frames_explicit": sys.argv[36] == "1",
        "tier_scan_max_raw_log_bytes": int(sys.argv[37]) if sys.argv[37] else None,
        "tier_scan_max_frame_rows": int(sys.argv[38]) if sys.argv[38] else None,
        "tier_scan_keep_parser_progress_rows": int(sys.argv[39]) if sys.argv[39] else None,
        "mode": sys.argv[21],
        "profile": sys.argv[22] or ("full" if (sys.argv[12] == "all" and sys.argv[20] in {"all", "*"}) else ""),
        "full_integrity": sys.argv[21] == "full-integrity",
        "glr_trace": sys.argv[27] == "1",
        "debug_dfa": sys.argv[28] == "1",
        "parse_progress": sys.argv[29] == "1",
        "parse_progress_interval_ms": sys.argv[30],
        "plan_only": sys.argv[31] == "1",
        "heartbeat": int(sys.argv[32]),
        "stage_enabled": {
            "baseline": sys.argv[23] == "1",
            "variants": sys.argv[24] == "1",
            "firstdiff": sys.argv[25] == "1",
            "summary": sys.argv[26] == "1",
        },
    },
    "artifacts": {
        "wringer_manifest_json": str(path),
        "wringer_active_txt": str(path.parent / "wringer_active.txt"),
        "wringer_active_json": str(path.parent / "wringer_active.json"),
        "wringer_events_jsonl": str(path.parent / "wringer_events.jsonl"),
        "frame_catalog_jsonl": str(path.parent / "frame_catalog.jsonl"),
        "frame_matrix_jsonl": str(path.parent / "frame_matrix.jsonl"),
        "wringer_plan_jsonl": str(path.parent / "wringer_plan.jsonl"),
        "wringer_plan_json": str(path.parent / "wringer_plan.json"),
        "wringer_frames_jsonl": str(path.parent / "wringer_frames.jsonl"),
        "wringer_summary_json": str(path.parent / "wringer_summary.json"),
        "wringer_summary_md": str(path.parent / "wringer_summary.md"),
        "commands_log": sys.argv[5],
    },
}
path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

write_run_meta() {
  local meta_path="$1"
  local kind="$2"
  local variant="$3"
  local ordinal="$4"
  local corpus_kind="$5"
  local corpus_root="$6"
  local file_path="$7"
  local log_path="$8"
  local rc="$9"
  local timeout_flag="${10}"
  local replay_command="${11}"
  local started_epoch="${12:-}"
  shift 12
  python3 - "$meta_path" "$kind" "$GRAMMAR" "$variant" "$ordinal" "$corpus_kind" "$corpus_root" \
    "$file_path" "$log_path" "$rc" "$timeout_flag" "$replay_command" "$started_epoch" "$@" <<'PY'
import json
import sys
import time
from pathlib import Path

meta_path = Path(sys.argv[1])
env = {}
finished_epoch = time.time()
started_epoch = None
try:
    if sys.argv[13]:
        started_epoch = float(sys.argv[13])
except ValueError:
    started_epoch = None
for item in sys.argv[14:]:
    if "=" in item:
        key, value = item.split("=", 1)
        env[key] = value
data = {
    "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "finished_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(finished_epoch)),
    "finished_at_epoch_s": finished_epoch,
    "kind": sys.argv[2],
    "stage": sys.argv[2],
    "grammar": sys.argv[3],
    "variant": sys.argv[4],
    "mode": sys.argv[4],
    "ordinal": int(sys.argv[5]),
    "corpus_kind": sys.argv[6],
    "corpus_root": sys.argv[7],
    "path": sys.argv[8],
    "log": sys.argv[9],
    "rc": int(sys.argv[10]),
    "timeout": sys.argv[11] == "1",
    "lifecycle": "timeout" if sys.argv[11] == "1" else ("fail" if int(sys.argv[10]) != 0 else "end"),
    "replay_command": sys.argv[12],
    "command": {
        "replay": sys.argv[12],
        "log": sys.argv[9],
    },
    "env": env,
}
if started_epoch is not None:
    data["started_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(started_epoch))
    data["started_at_epoch_s"] = started_epoch
    data["duration_s"] = max(0.0, finished_epoch - started_epoch)
    data["elapsed_s"] = data["duration_s"]
meta_path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

is_timeout_status() {
  [ "$1" -eq 124 ] || [ "$1" -eq 137 ]
}

is_infra_status() {
  [ "$1" -eq 125 ] || [ "$1" -eq 126 ] || [ "$1" -eq 127 ]
}

baseline_artifacts_exist() {
  [ -s "$BASELINE_DIR/manifest.json" ] && [ -s "$BASELINE_DIR/summary.json" ]
}

validate_baseline_artifacts() {
  local context="${1:-baseline}"
  python3 - "$BASELINE_DIR" "$GRAMMAR" "$context" <<'PY'
import json
import sys
from pathlib import Path

baseline_dir = Path(sys.argv[1])
grammar = sys.argv[2]
context = sys.argv[3]

def fail(message: str) -> None:
    raise SystemExit(f"{context}: invalid baseline artifacts: {message}")

def load_json(path: Path):
    if not path.exists() or path.stat().st_size == 0:
        fail(f"missing or empty {path}")
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        fail(f"corrupt JSON {path}: {exc}")

manifest = load_json(baseline_dir / "manifest.json")
summary = load_json(baseline_dir / "summary.json")
if not isinstance(manifest, dict):
    fail("manifest.json is not an object")
if not isinstance(summary, dict):
    fail("summary.json is not an object")

entries = manifest.get("grammars")
if not isinstance(entries, list):
    fail("manifest.json lacks grammars list")
grammar_entries = [entry for entry in entries if isinstance(entry, dict) and entry.get("grammar") == grammar]
if not grammar_entries:
    fail(f"manifest.json has no grammar entry for {grammar!r}")

selected_total = 0
executed_total = 0
missing_per_grammar = []
for entry in grammar_entries:
    try:
        selected_total += int(entry.get("selected_total") or entry.get("files") or 0)
    except (TypeError, ValueError):
        pass
    try:
        executed_total += int(entry.get("executed_files") or entry.get("files") or 0)
    except (TypeError, ValueError):
        pass
    manifest_path = Path(str(entry.get("manifest") or ""))
    if not manifest_path.exists():
        candidate = baseline_dir / manifest_path.name
        manifest_path = candidate if candidate.exists() else manifest_path
    if not manifest_path.exists():
        missing_per_grammar.append(str(entry.get("manifest") or ""))
        continue
    per_grammar = load_json(manifest_path)
    if per_grammar.get("grammar") != grammar:
        fail(f"{manifest_path} grammar={per_grammar.get('grammar')!r}, want {grammar!r}")
    files = per_grammar.get("files")
    if not isinstance(files, list):
        fail(f"{manifest_path} lacks files list")
    try:
        selected_total = max(selected_total, int(per_grammar.get("selected_total") or 0))
    except (TypeError, ValueError):
        pass
    try:
        executed_total += int(per_grammar.get("executed_files") or len(files))
    except (TypeError, ValueError):
        executed_total += len(files)
if missing_per_grammar:
    fail("missing per-grammar manifest(s): " + ", ".join(missing_per_grammar))

if selected_total <= 0:
    fail("selected frame catalog is empty; set a corpus with eligible files or use a zero-file diagnosis path")
if executed_total <= 0:
    fail("executed frame count is empty; normal wringer runs must execute at least one baseline frame")

frames_path = baseline_dir / "frames.jsonl"
if not frames_path.exists() or frames_path.stat().st_size == 0:
    fail(f"missing or empty {frames_path}")
frame_rows = []
for lineno, line in enumerate(frames_path.read_text(encoding="utf-8", errors="replace").splitlines(), start=1):
    if not line.strip():
        continue
    try:
        row = json.loads(line)
    except json.JSONDecodeError as exc:
        fail(f"corrupt frames.jsonl line {lineno}: {exc}")
    if isinstance(row, dict) and row.get("grammar") == grammar:
        frame_rows.append(row)
if not frame_rows:
    fail(f"frames.jsonl has no evidence rows for {grammar!r}")
terminal = [
    row for row in frame_rows
    if row.get("lifecycle") in {"ended", "timeout", "fail", "panic"}
    or row.get("phase") in {"comparison_result", "go_parse_status"}
    or str(row.get("phase") or "").endswith("_panic")
]
if not terminal:
    fail("baseline frames.jsonl contains only progress/scheduled evidence")
PY
}

validate_baseline_plan_artifacts() {
  local context="${1:-plan-only}"
  python3 - "$BASELINE_DIR" "$GRAMMAR" "$context" <<'PY'
import json
import sys
from pathlib import Path

baseline_dir = Path(sys.argv[1])
grammar = sys.argv[2]
context = sys.argv[3]

def fail(message: str) -> None:
    raise SystemExit(f"{context}: invalid plan-only baseline artifacts: {message}")

def load_json(path: Path):
    if not path.exists() or path.stat().st_size == 0:
        fail(f"missing or empty {path}")
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        fail(f"corrupt JSON {path}: {exc}")

manifest = load_json(baseline_dir / "manifest.json")
summary = load_json(baseline_dir / "summary.json")
if not isinstance(manifest, dict):
    fail("manifest.json is not an object")
if not isinstance(summary, dict):
    fail("summary.json is not an object")
entries = [entry for entry in manifest.get("grammars", []) if isinstance(entry, dict) and entry.get("grammar") == grammar]
if not entries:
    fail(f"manifest.json has no grammar entry for {grammar!r}")
selected_total = 0
for entry in entries:
    manifest_path = Path(str(entry.get("manifest") or ""))
    if not manifest_path.exists():
        candidate = baseline_dir / manifest_path.name
        manifest_path = candidate if candidate.exists() else manifest_path
    per_grammar = load_json(manifest_path)
    files = per_grammar.get("files")
    if not isinstance(files, list):
        fail(f"{manifest_path} lacks files list")
    selected_total += len(files)
if selected_total <= 0:
    fail("selected frame catalog is empty")
PY
}

run_summary_assertions() {
  local assertions="$WRINGER_ASSERTS"
  local args=()
  if [ "$stage_summary" -ne 1 ] || [ -z "$assertions" ] || [ "$assertions" = "0" ]; then
    return 0
  fi
  while IFS= read -r assertion; do
    [ -z "$assertion" ] && continue
    case "$assertion" in
      closed) args+=(--assert-closed) ;;
      plan|planned|complete) args+=(--assert-plan-exact) ;;
      telemetry) args+=(--assert-telemetry-complete) ;;
      artifacts|artifact) args+=(--assert-artifacts-sane) ;;
      *)
        echo "unknown GTS_WRINGER_ASSERTS entry: $assertion (want closed, plan, telemetry, artifacts, or 0)" >&2
        return 2
        ;;
    esac
  done < <(printf '%s\n' "$assertions" | tr ',[:space:]' '\n')
  if [ "${#args[@]}" -eq 0 ]; then
    return 0
  fi
  python3 "$SUMMARY_HELPER" "$OUT_DIR" "${args[@]}"
}

filter_file_list() { # $1=input jsonl $2=output jsonl $3=selector spec $4=selector name
  local input="$1"
  local output="$2"
  local spec="$3"
  local selector_name="${4:-frame selector}"
  if [ -z "$spec" ] || [ "$spec" = "all" ] || [ "$spec" = "*" ]; then
    cp "$input" "$output"
    return 0
  fi
  python3 - "$input" "$output" "$spec" "$selector_name" <<'PY'
import json
import sys
from pathlib import Path

input_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])
spec = sys.argv[3]
selector_name = sys.argv[4]

selectors = []
for raw_part in spec.split(","):
    part = raw_part.strip()
    if not part:
        continue
    if part.startswith("sha256:"):
        prefix = part[len("sha256:"):].strip().lower()
        if not prefix:
            raise SystemExit(f"invalid {selector_name} sha256 prefix: empty")
        selectors.append(("sha256", prefix))
        continue
    if part.startswith("base:"):
        base = part[len("base:"):].strip()
        if not base:
            raise SystemExit(f"invalid {selector_name} base: empty")
        selectors.append(("base", base))
        continue
    if part.startswith("path:"):
        path_part = part[len("path:"):].strip()
        if not path_part:
            raise SystemExit(f"invalid {selector_name} path: empty")
        selectors.append(("path", path_part))
        continue
    if "-" in part:
        if part.count("-") != 1:
            raise SystemExit(f"invalid {selector_name} range: {part!r}")
        start_text, end_text = part.split("-", 1)
        try:
            start = int(start_text)
            end = int(end_text)
        except ValueError:
            raise SystemExit(f"invalid {selector_name} range: {part!r}")
        if start <= 0 or end <= 0 or end < start:
            raise SystemExit(f"invalid {selector_name} range: {part!r}")
        selectors.append(("ordinal_set", set(range(start, end + 1))))
    else:
        try:
            value = int(part)
        except ValueError:
            raise SystemExit(
                f"invalid {selector_name} selector: {part!r}; "
                "use ordinals/ranges, sha256:<prefix>, base:<filename>, path:<substring>, all, or *"
            )
        if value <= 0:
            raise SystemExit(f"invalid {selector_name} ordinal: {part!r}")
        selectors.append(("ordinal", value))
if not selectors:
    raise SystemExit(f"invalid {selector_name}: empty frame selection")

seen: set[int] = set()
available: set[int] = set()
identities: list[str] = []
records = []
with input_path.open("r", encoding="utf-8") as src:
    for line in src:
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            continue
        records.append(record)
        try:
            ordinal = int(record.get("ordinal", 0))
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal > 0:
            available.add(ordinal)
        sha = str(record.get("sha256") or "")
        identities.append(
            f"{ordinal}:base:{record.get('base', '')}:sha256:{sha[:12]}:path:{record.get('path', '')}"
        )

missing = []
selected_ordinals: set[int] = set()
for kind, value in selectors:
    matched: set[int] = set()
    for record in records:
        try:
            ordinal = int(record.get("ordinal", 0))
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal <= 0:
            continue
        if kind == "ordinal" and ordinal == int(value):
            matched.add(ordinal)
        elif kind == "ordinal_set" and ordinal in value:
            matched.add(ordinal)
        elif kind == "sha256" and str(record.get("sha256") or "").lower().startswith(str(value)):
            matched.add(ordinal)
        elif kind == "base" and str(record.get("base") or "") == str(value):
            matched.add(ordinal)
        elif kind == "path" and str(value) in str(record.get("path") or ""):
            matched.add(ordinal)
    if not matched:
        missing.append(f"{kind}:{value}")
    selected_ordinals.update(matched)
if missing:
    available_text = "; ".join(identities) if identities else "none"
    raise SystemExit(
        f"{selector_name} selector(s) matched no frames: {', '.join(missing)}; "
        f"available: {available_text}"
    )
with input_path.open("r", encoding="utf-8") as src, output_path.open("w", encoding="utf-8") as dst:
    for line in src:
        if not line.strip():
            continue
        try:
            record = json.loads(line)
        except json.JSONDecodeError:
            continue
        try:
            ordinal = int(record.get("ordinal", 0))
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal in selected_ordinals:
            seen.add(ordinal)
            dst.write(json.dumps(record, sort_keys=True) + "\n")
PY
}

cap_file_list() { # $1=input/output jsonl $2=max rows
  local path="$1"
  local max_rows="$2"
  local tmp_path
  if [ "$max_rows" -le 0 ]; then
    return 0
  fi
  tmp_path="$path.tmp"
  python3 - "$path" "$tmp_path" "$max_rows" <<'PY'
import sys
from pathlib import Path

input_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])
max_rows = int(sys.argv[3])
count = 0
with input_path.open("r", encoding="utf-8") as src, output_path.open("w", encoding="utf-8") as dst:
    for line in src:
        if not line.strip():
            continue
        if count >= max_rows:
            break
        dst.write(line)
        count += 1
PY
  mv "$tmp_path" "$path"
}

infra_status=0
write_manifest ""

baseline_rc=""
if [ "$stage_baseline" -eq 1 ]; then
  if [ "$REUSE_BASELINE" -eq 1 ]; then
    if ! baseline_artifacts_exist; then
      infra_status=1
      write_manifest "" "$infra_status"
      echo "baseline reuse requested but missing baseline manifest/summary artifacts in $BASELINE_DIR" >&2
      exit "$infra_status"
    fi
    if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
      validate_cmd=validate_baseline_plan_artifacts
    else
      validate_cmd=validate_baseline_artifacts
    fi
    if ! "$validate_cmd" baseline-reuse; then
      infra_status=1
      write_manifest "" "$infra_status"
      exit "$infra_status"
    fi
    baseline_rc=0
    log_command baseline-reuse "$BASELINE_DIR"
  else
    baseline_env=(
      "GTS_CORPUS_DIR=$CORPUS"
      "GTS_TIER_SCAN_LANGS=$GRAMMAR"
      "GTS_TIER_SCAN_LIMIT=1"
      "GTS_TIER_SCAN_N=$N"
      "GTS_TIER_SCAN_ALL_FILES=$WRINGER_ALL_FILES"
      "GTS_TIER_SCAN_ROUNDS=$ROUNDS"
      "GTS_TIER_SCAN_TIMEOUT=$PER_FILE_TIMEOUT"
      "GTS_TIER_SCAN_KILL_AFTER=$KILL_AFTER"
      "GTS_TIER_SCAN_HEARTBEAT=$WRINGER_HEARTBEAT"
      "GTS_TIER_SCAN_ISOLATE_FILES=1"
      "GTS_TIER_SCAN_FRAMES=$BASELINE_FRAMES"
      "GTS_TIER_SCAN_PLAN_ONLY=$WRINGER_PLAN_ONLY"
      "GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1"
    )
    if [ -n "$WRINGER_TIER_SCAN_MAX_FRAME_ROWS" ]; then
      baseline_env+=("GTS_TIER_SCAN_MAX_FRAME_ROWS=$WRINGER_TIER_SCAN_MAX_FRAME_ROWS")
    fi
    if [ -n "$WRINGER_KEEP_PARSER_PROGRESS_ROWS" ]; then
      baseline_env+=("GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS=$WRINGER_KEEP_PARSER_PROGRESS_ROWS")
    fi
    if [ -n "$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES" ]; then
      baseline_env+=("GTS_TIER_SCAN_MAX_RAW_LOG_BYTES=$WRINGER_TIER_SCAN_MAX_RAW_LOG_BYTES")
    fi
    mapfile -t parse_progress_env < <(append_parse_progress_env)
    baseline_env+=("${parse_progress_env[@]}")
    log_command baseline env "${baseline_env[@]}" "$RUN_TIER_SCAN" "$BASELINE_DIR"
    baseline_replay_command="$(shell_join env "${baseline_env[@]}" "$RUN_TIER_SCAN" "$BASELINE_DIR")"
    baseline_log_path="$BASELINE_DIR/progress.log"
    write_active active baseline delegated tier_scan 0 "" "$CORPUS" "" "$baseline_log_path" "$PER_FILE_TIMEOUT" "$baseline_replay_command"
    remember_active_child baseline delegated tier_scan 0 "" "$CORPUS" "" "$baseline_log_path" "$PER_FILE_TIMEOUT" "$baseline_replay_command"
    set +e
    env "${baseline_env[@]}" "$RUN_TIER_SCAN" "$BASELINE_DIR" &
    current_child_pid=$!
    append_event START baseline delegated tier_scan 0 "" "$CORPUS" "" "$baseline_log_path" "$PER_FILE_TIMEOUT" "$baseline_replay_command" "" 0 "$current_started_epoch" "$current_child_pid"
    wait_for_active_child "$current_child_pid"
    baseline_rc=$?
    record_active_child_terminal "$baseline_rc"
    current_child_pid=""
    set -e
    complete_active_child
    timeout_flag=0
    if is_timeout_status "$baseline_rc"; then
      timeout_flag=1
    fi
    if [ "$baseline_rc" -ne 0 ]; then
      if ! baseline_artifacts_exist; then
        infra_status="$baseline_rc"
        write_manifest "$baseline_rc" "$infra_status"
        echo "baseline infrastructure failed: rc=$baseline_rc" >&2
        exit "$infra_status"
      fi
      if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
        validate_cmd=validate_baseline_plan_artifacts
      else
        validate_cmd=validate_baseline_artifacts
      fi
      if ! "$validate_cmd" baseline-nonzero; then
        infra_status=1
        write_manifest "$baseline_rc" "$infra_status"
        exit "$infra_status"
      fi
      append_event CONTINUE baseline delegated tier_scan 0 "" "$CORPUS" "" "$baseline_log_path" "$PER_FILE_TIMEOUT" "$baseline_replay_command" "$baseline_rc" "$timeout_flag"
      echo "baseline scan completed with nonzero status: rc=$baseline_rc; continuing with emitted artifacts" >&2
    fi
  fi
else
  if ! baseline_artifacts_exist; then
    infra_status=1
    write_manifest "" "$infra_status"
    echo "baseline stage skipped but missing baseline manifest/summary artifacts in $BASELINE_DIR" >&2
    exit "$infra_status"
  fi
  if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
    validate_cmd=validate_baseline_plan_artifacts
  else
    validate_cmd=validate_baseline_artifacts
  fi
  if ! "$validate_cmd" baseline-skipped; then
    infra_status=1
    write_manifest "" "$infra_status"
    exit "$infra_status"
  fi
  baseline_rc=0
fi
if ! baseline_artifacts_exist; then
  infra_status=1
  write_manifest "$baseline_rc" "$infra_status"
  echo "baseline infrastructure failed: missing baseline manifest/summary artifacts" >&2
  exit "$infra_status"
fi
if [ "$WRINGER_PLAN_ONLY" = "1" ]; then
  validate_cmd=validate_baseline_plan_artifacts
else
  validate_cmd=validate_baseline_artifacts
fi
if ! "$validate_cmd" baseline; then
  infra_status=1
  write_manifest "$baseline_rc" "$infra_status"
  exit "$infra_status"
fi
write_manifest "$baseline_rc" "$infra_status"

if [ "$stage_summary" -eq 1 ]; then
  python3 "$SUMMARY_HELPER" "$OUT_DIR" --write-summary
fi

case "$VARIANT_SCOPE" in
  suspicious) file_scope="suspicious" ;;
  all) file_scope="selected" ;;
esac
FILE_LIST="$OUT_DIR/${file_scope}_files.jsonl"
SELECTED_FILE_LIST="$OUT_DIR/selected_files.jsonl"
SUSPICIOUS_FILE_LIST="$OUT_DIR/suspicious_files.jsonl"
VARIANT_REPLAY_FILE_LIST="$OUT_DIR/variant_replay_files.jsonl"
FIRSTDIFF_REPLAY_FILE_LIST="$OUT_DIR/firstdiff_replay_files.jsonl"
python3 "$SUMMARY_HELPER" "$OUT_DIR" --emit-file-list-jsonl selected > "$SELECTED_FILE_LIST"
python3 "$SUMMARY_HELPER" "$OUT_DIR" --emit-file-list-jsonl "$file_scope" > "$FILE_LIST"
python3 "$SUMMARY_HELPER" "$OUT_DIR" --emit-file-list-jsonl suspicious > "$SUSPICIOUS_FILE_LIST"
filter_file_list "$FILE_LIST" "$VARIANT_REPLAY_FILE_LIST" "$VARIANT_FRAMES" "GTS_WRINGER_VARIANT_FRAMES"
if [ "$MAX_SUSPICIOUS" -gt 0 ] && [ "$VARIANT_FRAMES_EXPLICIT" -eq 0 ] && [ "$VARIANT_SCOPE" = "suspicious" ]; then
  cap_file_list "$VARIANT_REPLAY_FILE_LIST" "$MAX_SUSPICIOUS"
fi
if [ -n "$FIRSTDIFF_FRAMES" ]; then
  filter_file_list "$SELECTED_FILE_LIST" "$FIRSTDIFF_REPLAY_FILE_LIST" "$FIRSTDIFF_FRAMES" "GTS_WRINGER_FIRSTDIFF_FRAMES"
else
  filter_file_list "$SUSPICIOUS_FILE_LIST" "$FIRSTDIFF_REPLAY_FILE_LIST" "$FIRSTDIFF_FRAMES" "GTS_WRINGER_FIRSTDIFF_FRAMES"
fi
if [ "$MAX_DIAG_FILES" -gt 0 ] && [ "$FIRSTDIFF_FRAMES_EXPLICIT" -eq 0 ]; then
  cap_file_list "$FIRSTDIFF_REPLAY_FILE_LIST" "$MAX_DIAG_FILES"
fi

if [ "$stage_variants" -eq 1 ] || [ "$stage_firstdiff" -eq 1 ]; then
  BIN="$BASELINE_DIR/measure.test"
  if [ ! -x "$BIN" ]; then
    BIN="$OUT_DIR/measure.test"
    BUILD_LOG="$OUT_DIR/build-measure.log"
    log_command build env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o "$BIN" . ">" "$BUILD_LOG" "2>&1"
    (cd "$HARNESS" && CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o "$BIN" .) > "$BUILD_LOG" 2>&1
  fi
fi

if [ "$stage_variants" -eq 1 ]; then
  while IFS= read -r file_record; do
    [ -z "${file_record:-}" ] && continue
    mapfile -d '' -t file_fields < <(jsonl_record_fields "$file_record")
    ordinal="${file_fields[0]:-}"
    corpus_kind="${file_fields[2]:-}"
    corpus_root="${file_fields[3]:-}"
    file_path="${file_fields[4]:-}"
    [ -z "${file_path:-}" ] && continue
    frame_label=$(printf '%04d' "$ordinal")
    for variant in $VARIANTS; do
      variant_dir="$VARIANTS_DIR/$variant"
      mkdir -p "$variant_dir"
      log_path="$variant_dir/frame-${frame_label}.log"
      meta_path="$variant_dir/frame-${frame_label}.json"
      variant_assignments=()
      mapfile -t variant_assignments < <(variant_env "$variant")
      run_env=(
        "CGO_ENABLED=1"
        "REPRO_LANG=$GRAMMAR"
        "REPRO_DIR=$corpus_root"
        "REPRO_FILE=$file_path"
        "REPRO_PROGRESS=1"
        "REPRO_SIGNATURES=1"
        "REPRO_N=1"
        "REPRO_ROUNDS=$ROUNDS"
      )
      mapfile -t parse_progress_env < <(append_parse_progress_env)
      run_env+=("${parse_progress_env[@]}")
      run_env+=("${variant_assignments[@]}")
      replay_command="$(shell_join timeout "--kill-after=$KILL_AFTER" "$PER_FILE_TIMEOUT" env "${run_env[@]}" "$BIN" -test.run '^TestMeasureDtierVsC$' -test.count=1) > $(shell_quote "$log_path") 2>&1"
      log_command "variant:$variant" timeout "--kill-after=$KILL_AFTER" "$PER_FILE_TIMEOUT" env "${run_env[@]}" "$BIN" -test.run '^TestMeasureDtierVsC$' -test.count=1 ">" "$log_path" "2>&1"
      write_active active variant "$variant" "$variant" "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$PER_FILE_TIMEOUT" "$replay_command"
      remember_active_child variant "$variant" "$variant" "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$PER_FILE_TIMEOUT" "$replay_command" "$meta_path"
      set +e
      timeout --kill-after="$KILL_AFTER" "$PER_FILE_TIMEOUT" env "${run_env[@]}" \
        "$BIN" -test.run '^TestMeasureDtierVsC$' -test.count=1 > "$log_path" 2>&1 &
      current_child_pid=$!
      append_event START variant "$variant" "$variant" "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$PER_FILE_TIMEOUT" "$replay_command" "" 0 "$current_started_epoch" "$current_child_pid"
      wait_for_active_child "$current_child_pid"
      rc=$?
      set -e
      if is_infra_status "$rc"; then
        infra_status=1
      fi
      timeout_flag=0
      if is_timeout_status "$rc"; then
        timeout_flag=1
      fi
      record_active_child_terminal "$rc"
      current_child_pid=""
      write_run_meta "$meta_path" variant "$variant" "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$rc" "$timeout_flag" "$replay_command" "$current_started_epoch" "${run_env[@]}"
      complete_active_child
    done
  done < "$VARIANT_REPLAY_FILE_LIST"
fi

if [ "$stage_firstdiff" -eq 1 ]; then
  diag_count=0
  while IFS= read -r file_record; do
    [ -z "${file_record:-}" ] && continue
    mapfile -d '' -t file_fields < <(jsonl_record_fields "$file_record")
    ordinal="${file_fields[0]:-}"
    corpus_kind="${file_fields[2]:-}"
    corpus_root="${file_fields[3]:-}"
    file_path="${file_fields[4]:-}"
    [ -z "${file_path:-}" ] && continue
    if [ "$FIRSTDIFF_FRAMES_EXPLICIT" -eq 0 ] && [ -z "$FIRSTDIFF_FRAMES" ] && [ "$diag_count" -ge "$MAX_DIAG_FILES" ]; then
      break
    fi
    diag_count=$((diag_count + 1))
    frame_label=$(printf '%04d' "$ordinal")
    log_path="$FIRSTDIFF_DIR/frame-${frame_label}.log"
    meta_path="$FIRSTDIFF_DIR/frame-${frame_label}.json"
    run_env=(
      "CGO_ENABLED=1"
      "REPRO_LANG=$GRAMMAR"
      "REPRO_FILE=$file_path"
    )
    if [ "${GTS_WRINGER_GLR_TRACE:-0}" = "1" ]; then
      run_env+=("REPRO_GLR_TRACE=1")
    fi
    if [ "${GTS_WRINGER_DEBUG_DFA:-0}" = "1" ]; then
      run_env+=("REPRO_DEBUG_DFA=1")
    fi
    mapfile -t parse_progress_env < <(append_parse_progress_env)
    run_env+=("${parse_progress_env[@]}")
    replay_command="$(shell_join timeout "--kill-after=$KILL_AFTER" "$DIAG_TIMEOUT" env "${run_env[@]}" "$BIN" -test.run '^TestFirstDiffDiag$' -test.count=1 -test.v) > $(shell_quote "$log_path") 2>&1"
    log_command firstdiff timeout "--kill-after=$KILL_AFTER" "$DIAG_TIMEOUT" env "${run_env[@]}" "$BIN" -test.run '^TestFirstDiffDiag$' -test.count=1 -test.v ">" "$log_path" "2>&1"
    write_active active firstdiff firstdiff firstdiff "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$DIAG_TIMEOUT" "$replay_command"
    remember_active_child firstdiff firstdiff firstdiff "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$DIAG_TIMEOUT" "$replay_command" "$meta_path"
    set +e
    timeout --kill-after="$KILL_AFTER" "$DIAG_TIMEOUT" env "${run_env[@]}" \
      "$BIN" -test.run '^TestFirstDiffDiag$' -test.count=1 -test.v > "$log_path" 2>&1 &
    current_child_pid=$!
    append_event START firstdiff firstdiff firstdiff "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$DIAG_TIMEOUT" "$replay_command" "" 0 "$current_started_epoch" "$current_child_pid"
    wait_for_active_child "$current_child_pid"
    rc=$?
    set -e
    if is_infra_status "$rc"; then
      infra_status=1
    fi
    timeout_flag=0
    if is_timeout_status "$rc"; then
      timeout_flag=1
    fi
    record_active_child_terminal "$rc"
    current_child_pid=""
    write_run_meta "$meta_path" firstdiff firstdiff "$ordinal" "$corpus_kind" "$corpus_root" "$file_path" "$log_path" "$rc" "$timeout_flag" "$replay_command" "$current_started_epoch" "${run_env[@]}"
    complete_active_child
  done < "$FIRSTDIFF_REPLAY_FILE_LIST"
fi

set_idle_active
write_manifest "$baseline_rc" "$infra_status"
if [ "$stage_summary" -eq 1 ]; then
  python3 "$SUMMARY_HELPER" "$OUT_DIR" --write-summary
  run_summary_assertions
fi

console_line "wringer out: $OUT_DIR"
console_line "baseline rc: $baseline_rc"
console_line "infra status: $infra_status"
console_line "summary: $OUT_DIR/wringer_summary.json"
exit "$infra_status"
