#!/usr/bin/env bash
# Parallel wrapper for tier scan worker runs.
#
# Each shard writes to its own run directory. No clean/tier_iv/unmeasured
# artifact is shared while workers execute; cgo_harness/tier_scan/merge_runs.py
# reduces worker outputs after completion.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HARNESS="$REPO_ROOT/cgo_harness"
EXTS_TSV="$HARNESS/tier_scan/exts.tsv"
RUNNER="$SCRIPT_DIR/run_tier_scan.sh"
REDUCER="$HARNESS/tier_scan/merge_runs.py"

OUT_DIR="$HARNESS/harness_out/tier_scan_parallel/$(date -u +%Y%m%dT%H%M%SZ)"
PARALLELISM="${GTS_TIER_SCAN_PARALLELISM:-1}"
SHARDS="${GTS_TIER_SCAN_SHARDS:-$PARALLELISM}"
LANGS_ALLOWLIST="${GTS_TIER_SCAN_LANGS:-}"
DRY_RUN="${GTS_TIER_SCAN_PARALLEL_DRY_RUN:-0}"
NO_MERGE=0

usage() {
  cat <<'EOF'
Usage: run_tier_scan_parallel.sh [options] [out_dir]

Split selected tier-scan grammars into disjoint worker run directories, run
run_tier_scan.sh for each shard, then merge worker-local artifacts.

Options:
  --langs <list>       Grammar allowlist. Comma or whitespace separated.
                       Defaults to GTS_TIER_SCAN_LANGS or all exts.tsv rows.
  --parallelism <n>    Max concurrent workers. Default: GTS_TIER_SCAN_PARALLELISM or 1.
  --shards <n>         Number of allowlist shards. Default: GTS_TIER_SCAN_SHARDS
                       or the parallelism value.
  --dry-run            Print selected shards and worker dirs without running.
  --no-merge           Do not run merge_runs.py after workers finish.
  -h, --help           Show this help.

Important env forwarded to workers:
  GTS_CORPUS_DIR, GTS_TIER_SCAN_N, GTS_TIER_SCAN_ROUNDS,
  GTS_TIER_SCAN_TIMEOUT, GTS_TIER_SCAN_ISOLATE_FILES,
  GTS_TIER_SCAN_HEARTBEAT, GOT_PARSE_NODE_LIMIT_SCALE,
  GOT_GLR_MAX_STACKS.

Workers default GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 so parallel runs do not
race while rewriting docs/reports/tiers.{md,json}. Explicit 0 is rejected.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --langs) LANGS_ALLOWLIST="$2"; shift 2 ;;
    --parallelism) PARALLELISM="$2"; shift 2 ;;
    --shards) SHARDS="$2"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    --no-merge) NO_MERGE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    --*) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
    *) OUT_DIR="$1"; shift ;;
  esac
done

if [[ "$OUT_DIR" != /* ]]; then
  OUT_DIR="$REPO_ROOT/$OUT_DIR"
fi

case "$PARALLELISM" in
  ''|*[!0-9]*) echo "parallelism must be a positive integer: $PARALLELISM" >&2; exit 2 ;;
esac
case "$SHARDS" in
  ''|*[!0-9]*) echo "shards must be a positive integer: $SHARDS" >&2; exit 2 ;;
esac
if [ "$PARALLELISM" -lt 1 ] || [ "$SHARDS" -lt 1 ]; then
  echo "parallelism and shards must be positive" >&2
  exit 2
fi
if [ "${GTS_TIER_SCAN_SKIP_TIER_PUBLISH:-1}" != "1" ]; then
  echo "parallel tier scan requires GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1; docs tier publication cannot run concurrently" >&2
  exit 2
fi

rm -rf "$OUT_DIR/shards" "$OUT_DIR/workers"
mkdir -p "$OUT_DIR/shards" "$OUT_DIR/workers"

PARENT_PROGRESS="$OUT_DIR/progress.log"
PARENT_STATUS="$OUT_DIR/status.tsv"
PARENT_EVENTS="$OUT_DIR/events.jsonl"
PARENT_ACTIVE_TXT="$OUT_DIR/active_worker.txt"
PARENT_ACTIVE_JSON="$OUT_DIR/active_worker.json"
PARENT_MANIFEST="$OUT_DIR/manifest.json"
: > "$PARENT_PROGRESS"
: > "$PARENT_STATUS"
: > "$PARENT_EVENTS"

timestamp() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

env_summary() {
  python3 - <<'PY'
import json
import os

keys = [
    "GTS_CORPUS_DIR",
    "GTS_TIER_SCAN_N",
    "GTS_TIER_SCAN_ROUNDS",
    "GTS_TIER_SCAN_TIMEOUT",
    "GTS_TIER_SCAN_KILL_AFTER",
    "GTS_TIER_SCAN_ISOLATE_FILES",
    "GTS_TIER_SCAN_FRAMES",
    "GTS_TIER_SCAN_HEARTBEAT",
    "GOT_PARSE_NODE_LIMIT_SCALE",
    "GOT_GLR_MAX_STACKS",
    "GTS_TIER_SCAN_SKIP_TIER_PUBLISH",
]
print(json.dumps({key: os.environ.get(key, "") for key in keys if key in os.environ}, sort_keys=True))
PY
}

COMMAND_SUMMARY="$(printf '%q ' "$RUNNER" '<worker_dir>' | sed 's/[[:space:]]*$//')"
ENV_SUMMARY="$(env_summary)"

emit_parent_event() { # $1=event $2=shard $3=worker_dir $4=langs $5=pid $6=rc $7=reason
  local ts event shard worker_dir langs pid rc reason
  ts="$(timestamp)"
  event="$1"
  shard="$2"
  worker_dir="$3"
  langs="$4"
  pid="${5:-}"
  rc="${6:-}"
  reason="${7:-}"
  printf '%s %s shard=%s pid=%s rc=%s worker_dir=%s %s\n' \
    "$ts" "$event" "$shard" "$pid" "$rc" "$worker_dir" "$reason" | tee -a "$PARENT_PROGRESS"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$ts" "$event" "$shard" "$pid" "$rc" "$worker_dir" "$reason" >> "$PARENT_STATUS"
  python3 - "$PARENT_EVENTS" "$ts" "$event" "$shard" "$worker_dir" "$langs" "$pid" "$rc" "$reason" "$COMMAND_SUMMARY" "$ENV_SUMMARY" <<'PY'
import json
import sys
from pathlib import Path

(
    events_path,
    ts,
    event,
    shard,
    worker_dir,
    langs,
    pid,
    rc,
    reason,
    command_summary,
    env_summary_raw,
) = sys.argv[1:]
worker = Path(worker_dir)
try:
    env_summary = json.loads(env_summary_raw) if env_summary_raw else {}
except json.JSONDecodeError:
    env_summary = {"raw": env_summary_raw}
record = {
    "ts": ts,
    "event": event,
    "shard": shard,
    "worker_dir": worker_dir,
    "languages": [item for item in langs.split() if item],
    "pid": int(pid) if pid.isdigit() else None,
    "rc": int(rc) if rc.lstrip("-").isdigit() else None,
    "reason": reason,
    "command": command_summary,
    "env": env_summary,
    "artifacts": {
        "active_grammar": str(worker / "active_grammar.txt"),
        "progress_log": str(worker / "progress.log"),
        "status_tsv": str(worker / "status.tsv"),
        "summary_json": str(worker / "summary.json"),
    },
}
with open(events_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(record, sort_keys=True) + "\n")
PY
}

mapfile -t selected_langs < <(
  python3 - "$EXTS_TSV" "$LANGS_ALLOWLIST" "${GTS_TIER_SCAN_START_AFTER:-}" "${GTS_TIER_SCAN_LIMIT:-}" <<'PY'
import sys
from pathlib import Path

exts_tsv = Path(sys.argv[1])
allow_raw = sys.argv[2]
start_after = sys.argv[3]
limit_raw = sys.argv[4]
allow = {item for item in allow_raw.replace(",", " ").split() if item}
limit = int(limit_raw) if limit_raw else None
seen_start = not bool(start_after)
selected = []
for line in exts_tsv.read_text(encoding="utf-8").splitlines():
    if not line.strip():
        continue
    grammar = line.split("\t", 1)[0]
    if not seen_start:
        if grammar == start_after:
            seen_start = True
        continue
    if allow and grammar not in allow:
        continue
    selected.append(grammar)
    if limit is not None and len(selected) >= limit:
        break
for grammar in selected:
    print(grammar)
PY
)

if [ "${#selected_langs[@]}" -eq 0 ]; then
  echo "no grammars selected" >&2
  exit 2
fi

if [ "$SHARDS" -gt "${#selected_langs[@]}" ]; then
  SHARDS="${#selected_langs[@]}"
fi

for ((i = 0; i < SHARDS; i++)); do
  : > "$OUT_DIR/shards/shard-$(printf '%03d' "$i").txt"
done

for i in "${!selected_langs[@]}"; do
  shard=$((i % SHARDS))
  printf '%s\n' "${selected_langs[$i]}" >> "$OUT_DIR/shards/shard-$(printf '%03d' "$shard").txt"
done

echo "tier scan parallel output: $OUT_DIR"
echo "selected grammars: ${#selected_langs[@]}"
echo "shards: $SHARDS"
echo "parallelism: $PARALLELISM"

worker_dirs=()
shard_names=()
shard_langs=()
for shard_file in "$OUT_DIR"/shards/shard-*.txt; do
  shard_name="$(basename "$shard_file" .txt)"
  worker_dir="$OUT_DIR/workers/$shard_name"
  langs="$(tr '\n' ' ' < "$shard_file" | sed 's/[[:space:]]*$//')"
  [ -z "$langs" ] && continue
  worker_dirs+=("$worker_dir")
  shard_names+=("$shard_name")
  shard_langs+=("$langs")
  echo "$shard_name: $langs"
done

write_parent_manifest() { # $1=state
  python3 - "$PARENT_MANIFEST" "$OUT_DIR" "$PARALLELISM" "$SHARDS" "$DRY_RUN" "$NO_MERGE" "$1" "$COMMAND_SUMMARY" "$ENV_SUMMARY" "${shard_names[@]}" -- "${worker_dirs[@]}" -- "${shard_langs[@]}" <<'PY'
import json
import sys
import time

manifest_path, out_dir, parallelism, shards, dry_run, no_merge, state, command_summary, env_summary_raw = sys.argv[1:10]
rest = sys.argv[10:]
first_sep = rest.index("--")
second_sep = rest.index("--", first_sep + 1)
shard_names = rest[:first_sep]
worker_dirs = rest[first_sep + 1:second_sep]
shard_langs = rest[second_sep + 1:]
try:
    env_summary = json.loads(env_summary_raw) if env_summary_raw else {}
except json.JSONDecodeError:
    env_summary = {"raw": env_summary_raw}
workers = []
for shard, worker_dir, langs in zip(shard_names, worker_dirs, shard_langs):
    workers.append({
        "shard": shard,
        "worker_dir": worker_dir,
        "languages": [item for item in langs.split() if item],
        "artifacts": {
            "active_grammar": f"{worker_dir}/active_grammar.txt",
            "progress_log": f"{worker_dir}/progress.log",
            "status_tsv": f"{worker_dir}/status.tsv",
            "summary_json": f"{worker_dir}/summary.json",
        },
    })
data = {
    "run_dir": out_dir,
    "kind": "tier_scan_parallel",
    "state": state,
    "updated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "parallelism": int(parallelism),
    "shards": int(shards),
    "dry_run": dry_run == "1",
    "no_merge": no_merge == "1",
    "command": command_summary,
    "env": env_summary,
    "artifacts": {
        "active_worker_txt": f"{out_dir}/active_worker.txt",
        "active_worker_json": f"{out_dir}/active_worker.json",
        "events_jsonl": f"{out_dir}/events.jsonl",
        "status_tsv": f"{out_dir}/status.tsv",
        "progress_log": f"{out_dir}/progress.log",
        "merged_summary_json": f"{out_dir}/merged/summary.json",
    },
    "workers": workers,
}
with open(manifest_path, "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2, sort_keys=True)
    f.write("\n")
PY
}

write_parent_manifest "planned"

if [ "$DRY_RUN" = "1" ]; then
  write_parent_manifest "dry-run"
  printf '%s\tidle\tnone\t\t\t\tno workers launched\n' "$(timestamp)" > "$PARENT_ACTIVE_TXT"
  python3 - "$PARENT_ACTIVE_JSON" <<'PY'
import json
import sys
from pathlib import Path

Path(sys.argv[1]).write_text(json.dumps({"state": "dry-run", "workers": []}, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
  echo "dry run: no workers launched"
  exit 0
fi

declare -a pids=()
declare -A completed_by_index=()
parent_finalized=0

worker_pid_is_running() { # $1=pid
  local live_pids pid
  pid="${1:-}"
  [ -n "$pid" ] || return 1
  live_pids="$(jobs -pr || true)"
  grep -Fxq "$pid" <<<"$live_pids"
}

missing_parent_rc_event_for_rc() { # $1=wait_rc
  local rc
  rc="${1:-}"
  case "$rc" in
    124) printf '%s\n' "TIMEOUT" ;;
    ''|*[!0-9]*) printf '%s\n' "FAIL" ;;
    *)
      if [ "$rc" -ge 128 ]; then
        printf '%s\n' "ABORT"
      else
        printf '%s\n' "FAIL"
      fi
      ;;
  esac
}

write_parent_active() {
  python3 - "$PARENT_ACTIVE_TXT" "$PARENT_ACTIVE_JSON" "${shard_names[@]}" -- "${worker_dirs[@]}" -- "${shard_langs[@]}" -- "${pids[@]}" -- "${!completed_by_index[@]}" <<'PY'
import json
import os
import sys
import time
from pathlib import Path

active_txt, active_json = sys.argv[1:3]
rest = sys.argv[3:]
first_sep = rest.index("--")
second_sep = rest.index("--", first_sep + 1)
third_sep = rest.index("--", second_sep + 1)
fourth_sep = rest.index("--", third_sep + 1)
shard_names = rest[:first_sep]
worker_dirs = rest[first_sep + 1:second_sep]
shard_langs = rest[second_sep + 1:third_sep]
pids = rest[third_sep + 1:fourth_sep]
completed = {int(item) for item in rest[fourth_sep + 1:] if item.isdigit()}
now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
workers = []
lines = []
for idx, (shard, worker_dir, langs) in enumerate(zip(shard_names, worker_dirs, shard_langs)):
    pid = pids[idx] if idx < len(pids) else ""
    state = "complete" if idx in completed else ("running" if pid else "pending")
    worker = Path(worker_dir)
    active_grammar = worker / "active_grammar.txt"
    progress_log = worker / "progress.log"
    status_tsv = worker / "status.tsv"
    summary_json = worker / "summary.json"
    current = ""
    if active_grammar.exists():
        current = active_grammar.read_text(encoding="utf-8", errors="replace").splitlines()[-1:] or [""]
        current = current[0]
    progress_mtime = ""
    if progress_log.exists():
        progress_mtime = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(progress_log.stat().st_mtime))
    record = {
        "shard": shard,
        "worker_dir": worker_dir,
        "languages": [item for item in langs.split() if item],
        "pid": int(pid) if pid.isdigit() else None,
        "state": state,
        "current_active_grammar": current,
        "artifacts": {
            "active_grammar": str(active_grammar),
            "progress_log": str(progress_log),
            "status_tsv": str(status_tsv),
            "summary_json": str(summary_json),
        },
        "progress_log_mtime": progress_mtime,
    }
    workers.append(record)
    if state == "running":
        lines.append("\t".join([
            now,
            shard,
            pid,
            state,
            worker_dir,
            str(active_grammar),
            str(progress_log),
            str(status_tsv),
            str(summary_json),
            current,
        ]))
if not lines:
    lines.append("\t".join([now, "idle", "", "idle", "", "", "", "", "", ""]))
Path(active_txt).write_text("\n".join(lines) + "\n", encoding="utf-8")
Path(active_json).write_text(json.dumps({
    "updated_at": now,
    "running_worker_count": sum(1 for worker in workers if worker["state"] == "running"),
    "pending_worker_count": sum(1 for worker in workers if worker["state"] == "pending"),
    "complete_worker_count": sum(1 for worker in workers if worker["state"] == "complete"),
    "workers": workers,
}, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

record_finished_workers() {
  local i rc_file rc event pid
  for i in "${!pids[@]}"; do
    [ "${completed_by_index[$i]:-0}" = "1" ] && continue
    rc_file="${worker_dirs[$i]}/.parent_rc"
    pid="${pids[$i]}"
    if [ -s "$rc_file" ]; then
      rc="$(cat "$rc_file")"
      event="END"
      if [ "$rc" != "0" ]; then
        event="FAIL"
        worker_status=1
      fi
      emit_parent_event "$event" "${shard_names[$i]}" "${worker_dirs[$i]}" "${shard_langs[$i]}" "$pid" "$rc" "worker_exit"
      completed_by_index[$i]=1
      continue
    fi
    if worker_pid_is_running "$pid"; then
      continue
    fi
    if wait "$pid"; then
      rc=0
    else
      rc=$?
    fi
    event="$(missing_parent_rc_event_for_rc "$rc")"
    if [ -s "$rc_file" ]; then
      # The child may have finished writing between the first check and wait.
      rc="$(cat "$rc_file")"
      event="END"
      if [ "$rc" != "0" ]; then
        event="FAIL"
        worker_status=1
      fi
      emit_parent_event "$event" "${shard_names[$i]}" "${worker_dirs[$i]}" "${shard_langs[$i]}" "$pid" "$rc" "worker_exit"
    else
      worker_status=1
      emit_parent_event "$event" "${shard_names[$i]}" "${worker_dirs[$i]}" "${shard_langs[$i]}" "$pid" "$rc" "missing_parent_rc"
    fi
    completed_by_index[$i]=1
  done
}

running_worker_count() {
  local i count
  count=0
  for i in "${!pids[@]}"; do
    [ "${completed_by_index[$i]:-0}" = "1" ] && continue
    count=$((count + 1))
  done
  printf '%s\n' "$count"
}

finalize_parent() { # $1=parent_rc
  local parent_rc i rc_file rc event
  parent_rc="${1:-0}"
  [ "$parent_finalized" -eq 1 ] && return 0
  parent_finalized=1
  record_finished_workers || true
  for i in "${!pids[@]}"; do
    [ "${completed_by_index[$i]:-0}" = "1" ] && continue
    rc_file="${worker_dirs[$i]}/.parent_rc"
    rc=""
    event="FAIL"
    if [ -s "$rc_file" ]; then
      rc="$(cat "$rc_file")"
      [ "$rc" = "0" ] && event="END"
    fi
    emit_parent_event "$event" "${shard_names[$i]}" "${worker_dirs[$i]}" "${shard_langs[$i]}" "${pids[$i]}" "$rc" "parent_exit_rc=$parent_rc"
    if [ "$parent_rc" -eq 0 ]; then
      completed_by_index[$i]=1
    fi
  done
  write_parent_active || true
  if [ "$parent_rc" -eq 0 ]; then
    write_parent_manifest "complete" || true
  else
    write_parent_manifest "failed" || true
  fi
}

trap 'rc=$?; finalize_parent "$rc"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

worker_status=0
next_worker=0
total_workers="${#worker_dirs[@]}"
write_parent_manifest "running"
write_parent_active
while [ "$next_worker" -lt "$total_workers" ] || [ "$(running_worker_count)" -gt 0 ]; do
  record_finished_workers
  while [ "$next_worker" -lt "$total_workers" ] && [ "$(running_worker_count)" -lt "$PARALLELISM" ]; do
    shard_name="${shard_names[$next_worker]}"
    worker_dir="${worker_dirs[$next_worker]}"
    langs="${shard_langs[$next_worker]}"
    mkdir -p "$worker_dir"
    rm -f "$worker_dir/.parent_rc"
    (
      rc=0
      export GTS_TIER_SCAN_LANGS="$langs"
      export GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1
      "$RUNNER" "$worker_dir" || rc=$?
      printf '%s\n' "$rc" > "$worker_dir/.parent_rc"
      exit "$rc"
    ) &
    pid="$!"
    pids+=("$pid")
    emit_parent_event "START" "$shard_name" "$worker_dir" "$langs" "$pid" "" "worker_launch"
    next_worker=$((next_worker + 1))
    write_parent_active
  done
  write_parent_active
  if [ "$next_worker" -lt "$total_workers" ] || [ "$(running_worker_count)" -gt 0 ]; then
    sleep 1
  fi
done
record_finished_workers

for pid in "${pids[@]}"; do
  wait "$pid" || true
done

merge_status=0
if [ "$NO_MERGE" = "0" ]; then
  if ! python3 "$REDUCER" "$OUT_DIR/merged" "${worker_dirs[@]}"; then
    merge_status=1
  fi
fi

if [ "$worker_status" -ne 0 ] || [ "$merge_status" -ne 0 ]; then
  exit 1
fi
finalize_parent 0
