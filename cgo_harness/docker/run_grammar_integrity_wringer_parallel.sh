#!/usr/bin/env bash
# Parallel control plane for single-grammar parser integrity wringer runs.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
HARNESS="$REPO_ROOT/cgo_harness"
EXTS_TSV="$HARNESS/tier_scan/exts.tsv"
RUNNER="$SCRIPT_DIR/run_grammar_integrity_wringer.sh"
REDUCER="$HARNESS/tier_scan/merge_wringer_runs.py"

OUT_DIR="$HARNESS/harness_out/grammar_integrity_wringer_parallel/$(date -u +%Y%m%dT%H%M%SZ)"
LANGS_ALLOWLIST="${GTS_WRINGER_LANGS:-}"
PARALLELISM="${GTS_WRINGER_PARALLELISM:-1}"
DRY_RUN="${GTS_WRINGER_PARALLEL_DRY_RUN:-1}"
PARENT_HEARTBEAT_INTERVAL="${GTS_WRINGER_PARENT_HEARTBEAT_INTERVAL:-30}"
NO_MERGE=0

usage() {
  cat <<'EOF'
Usage: run_grammar_integrity_wringer_parallel.sh [options] [out_dir]

Run the single-grammar parser-integrity wringer across selected grammars with
parent-level lifecycle, active-worker, and merge-control artifacts.

Options:
  --langs <list>       Grammar allowlist. Comma or whitespace separated.
                       Defaults to all cgo_harness/tier_scan/exts.tsv rows.
  --parallelism <n>    Max concurrent grammar workers. Default:
                       GTS_WRINGER_PARALLELISM or 1.
  --dry-run            Write manifest and active snapshots without workers.
                       Default: GTS_WRINGER_PARALLEL_DRY_RUN or 1.
  --no-merge           Do not run merge_wringer_runs.py after workers finish.
  -h, --help           Show this help.

Selection also respects:
  GTS_WRINGER_START_AFTER  skip exts.tsv order until after this grammar
  GTS_WRINGER_LIMIT        stop after N selected grammars
  GTS_WRINGER_PARENT_HEARTBEAT_INTERVAL
                           parent heartbeat seconds while workers run;
                           0 disables (default 30)

Workers execute:
  cgo_harness/docker/run_grammar_integrity_wringer.sh <grammar> <worker_dir>

Existing GTS_WRINGER_*, GTS_CORPUS_DIR, and parser-control environment
variables are inherited by each worker.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --langs) LANGS_ALLOWLIST="$2"; shift 2 ;;
    --parallelism) PARALLELISM="$2"; shift 2 ;;
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
if [ "$PARALLELISM" -lt 1 ]; then
  echo "parallelism must be positive" >&2
  exit 2
fi
case "$DRY_RUN" in
  0|1) ;;
  *) echo "GTS_WRINGER_PARALLEL_DRY_RUN must be 0 or 1: $DRY_RUN" >&2; exit 2 ;;
esac
case "$PARENT_HEARTBEAT_INTERVAL" in
  ''|*[!0-9]*) echo "GTS_WRINGER_PARENT_HEARTBEAT_INTERVAL must be a nonnegative integer: $PARENT_HEARTBEAT_INTERVAL" >&2; exit 2 ;;
esac

mkdir -p "$OUT_DIR/workers"

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

prefixes = ("GTS_WRINGER_", "GOT_", "REPRO_")
keys = [
    "GTS_CORPUS_DIR",
    "GOMAXPROCS",
    "GOTOOLCHAIN",
    "CGO_ENABLED",
]
env = {key: os.environ[key] for key in keys if key in os.environ}
for key, value in os.environ.items():
    if key.startswith(prefixes):
        env[key] = value
print(json.dumps(dict(sorted(env.items())), sort_keys=True))
PY
}

COMMAND_SUMMARY="$(printf '%q ' "$RUNNER" '<grammar>' '<worker_dir>' | sed 's/[[:space:]]*$//')"
ENV_SUMMARY="$(env_summary)"

artifact_json() {
  local worker_dir="$1"
  python3 - "$worker_dir" <<'PY'
import json
import sys
from pathlib import Path

worker = Path(sys.argv[1])
baseline = worker / "baseline"
print(json.dumps({
    "wringer_active_json": str(worker / "wringer_active.json"),
    "wringer_summary_json": str(worker / "wringer_summary.json"),
    "frame_matrix_jsonl": str(worker / "frame_matrix.jsonl"),
    "wringer_plan_jsonl": str(worker / "wringer_plan.jsonl"),
    "wringer_frames_jsonl": str(worker / "wringer_frames.jsonl"),
    "wringer_events_jsonl": str(worker / "wringer_events.jsonl"),
    "wringer_manifest_json": str(worker / "wringer_manifest.json"),
    "baseline_status_tsv": str(baseline / "status.tsv"),
    "baseline_progress_log": str(baseline / "progress.log"),
}, sort_keys=True))
PY
}

emit_parent_event() { # $1=event $2=grammar $3=worker_dir $4=pid $5=rc $6=reason
  local ts event grammar worker_dir pid rc reason artifacts
  ts="$(timestamp)"
  event="$1"
  grammar="$2"
  worker_dir="$3"
  pid="${4:-}"
  rc="${5:-}"
  reason="${6:-}"
  artifacts="$(artifact_json "$worker_dir")"
  printf '%s %s grammar=%s pid=%s rc=%s worker_dir=%s %s\n' \
    "$ts" "$event" "$grammar" "$pid" "$rc" "$worker_dir" "$reason" | tee -a "$PARENT_PROGRESS"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$ts" "$event" "$grammar" "$pid" "$rc" "$worker_dir" "$reason" >> "$PARENT_STATUS"
  python3 - "$PARENT_EVENTS" "$ts" "$event" "$grammar" "$worker_dir" "$pid" "$rc" "$reason" "$COMMAND_SUMMARY" "$ENV_SUMMARY" "$artifacts" <<'PY'
import json
import sys

(
    events_path,
    ts,
    event,
    grammar,
    worker_dir,
    pid,
    rc,
    reason,
    command_summary,
    env_summary_raw,
    artifacts_raw,
) = sys.argv[1:]
try:
    env_summary = json.loads(env_summary_raw) if env_summary_raw else {}
except json.JSONDecodeError:
    env_summary = {"raw": env_summary_raw}
try:
    artifacts = json.loads(artifacts_raw) if artifacts_raw else {}
except json.JSONDecodeError:
    artifacts = {"raw": artifacts_raw}
record = {
    "ts": ts,
    "event": event,
    "grammar": grammar,
    "worker_dir": worker_dir,
    "pid": int(pid) if pid.isdigit() else None,
    "rc": int(rc) if rc.lstrip("-").isdigit() else None,
    "reason": reason,
    "command": command_summary,
    "env": env_summary,
    "artifacts": artifacts,
}
with open(events_path, "a", encoding="utf-8") as f:
    f.write(json.dumps(record, sort_keys=True) + "\n")
PY
}

emit_parent_heartbeat() {
  local ts
  ts="$(timestamp)"
  python3 - "$PARENT_ACTIVE_JSON" "$PARENT_PROGRESS" "$PARENT_STATUS" "$PARENT_EVENTS" "$ts" "$COMMAND_SUMMARY" "$ENV_SUMMARY" <<'PY'
import json
import shlex
import sys

active_json, progress_path, status_path, events_path, ts, command_summary, env_summary_raw = sys.argv[1:]
try:
    active = json.load(open(active_json, encoding="utf-8"))
except Exception:
    active = {}
try:
    env_summary = json.loads(env_summary_raw) if env_summary_raw else {}
except json.JSONDecodeError:
    env_summary = {"raw": env_summary_raw}

workers = active.get("workers", [])
counts = {
    "pending": int(active.get("pending_worker_count", 0) or 0),
    "running": int(active.get("running_worker_count", 0) or 0),
    "complete": int(active.get("complete_worker_count", 0) or 0),
    "failed": int(active.get("failed_worker_count", 0) or 0),
}

def text(value):
    if value is None:
        return ""
    return str(value)

def q(value):
    return shlex.quote(text(value))

progress_lines = []
status_lines = []
event_lines = []
for worker in workers:
    if worker.get("state") != "running":
        continue
    last = worker.get("last_wringer_active")
    if not isinstance(last, dict):
        last = {}
    grammar = text(worker.get("grammar"))
    worker_dir = text(worker.get("worker_dir"))
    pid = worker.get("pid")
    fields = {
        "worker_state": text(worker.get("state")),
        "stage": text(last.get("stage")),
        "mode": text(last.get("mode")),
        "variant": text(last.get("variant")),
        "ordinal": text(last.get("ordinal")),
        "path": text(last.get("path")),
        "log": text(last.get("log")),
        "timeout": text(last.get("timeout")),
    }
    counts_text = "pending={pending} running={running} complete={complete} failed={failed}".format(**counts)
    progress_lines.append(
        f"{ts} HEARTBEAT grammar={q(grammar)} pid={q(pid)} rc= worker_dir={q(worker_dir)} "
        f"{counts_text} "
        + " ".join(f"{key}={q(value)}" for key, value in fields.items())
    )
    status_lines.append("\t".join([
        ts,
        "HEARTBEAT",
        grammar,
        text(pid),
        "",
        worker_dir,
        counts_text,
        fields["worker_state"],
        fields["stage"],
        fields["mode"],
        fields["variant"],
        fields["ordinal"],
        fields["path"],
        fields["log"],
        fields["timeout"],
    ]))
    event_lines.append(json.dumps({
        "ts": ts,
        "event": "HEARTBEAT",
        "grammar": grammar,
        "worker_dir": worker_dir,
        "pid": int(pid) if isinstance(pid, int) or text(pid).isdigit() else None,
        "rc": None,
        "reason": "parent_heartbeat",
        "counts": counts,
        "worker_state": text(worker.get("state")),
        "last_wringer_active": last,
        "command": command_summary,
        "env": env_summary,
        "artifacts": worker.get("artifacts", {}),
    }, sort_keys=True))

if progress_lines:
    with open(progress_path, "a", encoding="utf-8") as f:
        f.write("\n".join(progress_lines) + "\n")
    with open(status_path, "a", encoding="utf-8") as f:
        f.write("\n".join(status_lines) + "\n")
    with open(events_path, "a", encoding="utf-8") as f:
        f.write("\n".join(event_lines) + "\n")
PY
}

mapfile -t selected_langs < <(
  python3 - "$EXTS_TSV" "$LANGS_ALLOWLIST" "${GTS_WRINGER_START_AFTER:-}" "${GTS_WRINGER_LIMIT:-}" <<'PY'
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
    if not line.strip() or line.startswith("#"):
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

worker_dirs=()
for grammar in "${selected_langs[@]}"; do
  worker_dirs+=("$OUT_DIR/workers/$grammar")
done

write_parent_manifest() { # $1=state
  python3 - "$PARENT_MANIFEST" "$OUT_DIR" "$PARALLELISM" "$DRY_RUN" "$NO_MERGE" "$1" "$COMMAND_SUMMARY" "$ENV_SUMMARY" "${selected_langs[@]}" -- "${worker_dirs[@]}" <<'PY'
import json
import sys
import time
from pathlib import Path

manifest_path, out_dir, parallelism, dry_run, no_merge, state, command_summary, env_summary_raw = sys.argv[1:9]
rest = sys.argv[9:]
sep = rest.index("--")
grammars = rest[:sep]
worker_dirs = rest[sep + 1:]
try:
    env_summary = json.loads(env_summary_raw) if env_summary_raw else {}
except json.JSONDecodeError:
    env_summary = {"raw": env_summary_raw}
workers = []
for grammar, worker_dir in zip(grammars, worker_dirs):
    worker = Path(worker_dir)
    workers.append({
        "grammar": grammar,
        "worker_dir": worker_dir,
        "artifacts": {
            "wringer_active_json": str(worker / "wringer_active.json"),
            "wringer_summary_json": str(worker / "wringer_summary.json"),
            "frame_matrix_jsonl": str(worker / "frame_matrix.jsonl"),
            "wringer_plan_jsonl": str(worker / "wringer_plan.jsonl"),
            "wringer_frames_jsonl": str(worker / "wringer_frames.jsonl"),
            "wringer_events_jsonl": str(worker / "wringer_events.jsonl"),
            "wringer_manifest_json": str(worker / "wringer_manifest.json"),
            "baseline_status_tsv": str(worker / "baseline" / "status.tsv"),
            "baseline_progress_log": str(worker / "baseline" / "progress.log"),
        },
    })
data = {
    "run_dir": out_dir,
    "kind": "grammar_integrity_wringer_parallel",
    "state": state,
    "updated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "parallelism": int(parallelism),
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
Path(manifest_path).write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

declare -a pids=()
declare -A completed_by_index=()
declare -A failed_by_index=()

write_parent_active() {
  python3 - "$PARENT_ACTIVE_TXT" "$PARENT_ACTIVE_JSON" "${selected_langs[@]}" -- "${worker_dirs[@]}" -- "${pids[@]}" -- "${!completed_by_index[@]}" -- "${!failed_by_index[@]}" <<'PY'
import json
import sys
import time
from pathlib import Path

active_txt, active_json = sys.argv[1:3]
rest = sys.argv[3:]
first_sep = rest.index("--")
second_sep = rest.index("--", first_sep + 1)
third_sep = rest.index("--", second_sep + 1)
fourth_sep = rest.index("--", third_sep + 1)
grammars = rest[:first_sep]
worker_dirs = rest[first_sep + 1:second_sep]
pids = rest[second_sep + 1:third_sep]
completed = {int(item) for item in rest[third_sep + 1:fourth_sep] if item.isdigit()}
failed = {int(item) for item in rest[fourth_sep + 1:] if item.isdigit()}
now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
workers = []
lines = []
for idx, (grammar, worker_dir) in enumerate(zip(grammars, worker_dirs)):
    pid = pids[idx] if idx < len(pids) else ""
    if idx in completed:
        state = "failed" if idx in failed else "complete"
    elif pid:
        state = "running"
    else:
        state = "pending"
    worker = Path(worker_dir)
    wringer_active_json = worker / "wringer_active.json"
    wringer_summary_json = worker / "wringer_summary.json"
    frame_matrix_jsonl = worker / "frame_matrix.jsonl"
    wringer_plan_jsonl = worker / "wringer_plan.jsonl"
    wringer_frames_jsonl = worker / "wringer_frames.jsonl"
    wringer_events_jsonl = worker / "wringer_events.jsonl"
    wringer_manifest_json = worker / "wringer_manifest.json"
    baseline_status = worker / "baseline" / "status.tsv"
    baseline_progress = worker / "baseline" / "progress.log"
    last_active = {}
    if wringer_active_json.exists():
        try:
            parsed = json.loads(wringer_active_json.read_text(encoding="utf-8"))
            if isinstance(parsed, dict):
                last_active = parsed
        except Exception:
            last_active = {"unreadable": str(wringer_active_json)}
    record = {
        "grammar": grammar,
        "worker_dir": worker_dir,
        "pid": int(pid) if pid.isdigit() else None,
        "state": state,
        "last_wringer_active": last_active,
        "wringer_summary_json": str(wringer_summary_json),
        "artifacts": {
            "wringer_active_json": str(wringer_active_json),
            "wringer_summary_json": str(wringer_summary_json),
            "frame_matrix_jsonl": str(frame_matrix_jsonl),
            "wringer_plan_jsonl": str(wringer_plan_jsonl),
            "wringer_frames_jsonl": str(wringer_frames_jsonl),
            "wringer_events_jsonl": str(wringer_events_jsonl),
            "wringer_manifest_json": str(wringer_manifest_json),
            "baseline_status_tsv": str(baseline_status),
            "baseline_progress_log": str(baseline_progress),
        },
    }
    workers.append(record)
    lines.append("\t".join([
        now,
        grammar,
        pid,
        state,
        worker_dir,
        str(wringer_active_json),
        str(wringer_summary_json),
        str(frame_matrix_jsonl),
        str(wringer_plan_jsonl),
        str(baseline_status),
    ]))
Path(active_txt).write_text("\n".join(lines) + "\n", encoding="utf-8")
Path(active_json).write_text(json.dumps({
    "updated_at": now,
    "pending_worker_count": sum(1 for worker in workers if worker["state"] == "pending"),
    "running_worker_count": sum(1 for worker in workers if worker["state"] == "running"),
    "complete_worker_count": sum(1 for worker in workers if worker["state"] == "complete"),
    "failed_worker_count": sum(1 for worker in workers if worker["state"] == "failed"),
    "workers": workers,
}, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

echo "wringer parallel output: $OUT_DIR"
echo "selected grammars: ${#selected_langs[@]}"
echo "parallelism: $PARALLELISM"
echo "parent heartbeat interval: $PARENT_HEARTBEAT_INTERVAL"
printf 'grammars: %s\n' "${selected_langs[*]}"

write_parent_manifest "planned"
write_parent_active

if [ "$DRY_RUN" = "1" ]; then
  write_parent_manifest "dry-run"
  echo "dry run: no workers launched"
  exit 0
fi

parent_finalized=0
worker_status=0

worker_pid_is_running() { # $1=pid
  local pid="$1"
  [ -n "$pid" ] || return 1
  jobs -pr | grep -Fxq "$pid"
}

terminal_event_for_rc() { # $1=rc
  local rc="${1:-}"
  case "$rc" in
    0) printf '%s\n' "END" ;;
    124) printf '%s\n' "FAIL" ;;
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

record_finished_workers() {
  local i rc_file rc event pid
  for i in "${!pids[@]}"; do
    [ "${completed_by_index[$i]:-0}" = "1" ] && continue
    pid="${pids[$i]}"
    rc_file="${worker_dirs[$i]}/.parent_rc"
    if [ -s "$rc_file" ]; then
      rc="$(cat "$rc_file")"
      event="$(terminal_event_for_rc "$rc")"
      [ "$rc" = "0" ] || { worker_status=1; failed_by_index[$i]=1; }
      emit_parent_event "$event" "${selected_langs[$i]}" "${worker_dirs[$i]}" "$pid" "$rc" "worker_exit"
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
    if [ -s "$rc_file" ]; then
      rc="$(cat "$rc_file")"
    fi
    event="$(terminal_event_for_rc "$rc")"
    [ "$rc" = "0" ] || { worker_status=1; failed_by_index[$i]=1; }
    emit_parent_event "$event" "${selected_langs[$i]}" "${worker_dirs[$i]}" "$pid" "$rc" "worker_exit"
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
    if [ -s "$rc_file" ]; then
      rc="$(cat "$rc_file")"
    fi
    event="$(terminal_event_for_rc "${rc:-$parent_rc}")"
    [ "${rc:-$parent_rc}" = "0" ] || failed_by_index[$i]=1
    emit_parent_event "$event" "${selected_langs[$i]}" "${worker_dirs[$i]}" "${pids[$i]}" "$rc" "parent_exit_rc=$parent_rc"
    completed_by_index[$i]=1
  done
  write_parent_active || true
  if [ "$parent_rc" -eq 0 ] && [ "$worker_status" -eq 0 ]; then
    write_parent_manifest "complete" || true
  else
    write_parent_manifest "failed" || true
  fi
}

trap 'rc=$?; finalize_parent "$rc"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

next_worker=0
total_workers="${#worker_dirs[@]}"
last_parent_heartbeat_epoch="$(date -u +%s)"
write_parent_manifest "running"
while [ "$next_worker" -lt "$total_workers" ] || [ "$(running_worker_count)" -gt 0 ]; do
  now_epoch="$(date -u +%s)"
  record_finished_workers
  while [ "$next_worker" -lt "$total_workers" ] && [ "$(running_worker_count)" -lt "$PARALLELISM" ]; do
    grammar="${selected_langs[$next_worker]}"
    worker_dir="${worker_dirs[$next_worker]}"
    mkdir -p "$worker_dir"
    rm -f "$worker_dir/.parent_rc"
    (
      rc=0
      "$RUNNER" "$grammar" "$worker_dir" || rc=$?
      printf '%s\n' "$rc" > "$worker_dir/.parent_rc"
      exit "$rc"
    ) &
    pid="$!"
    pids+=("$pid")
    emit_parent_event "START" "$grammar" "$worker_dir" "$pid" "" "worker_launch"
    next_worker=$((next_worker + 1))
    write_parent_active
  done
  write_parent_active
  if [ "$PARENT_HEARTBEAT_INTERVAL" -gt 0 ] && [ "$(running_worker_count)" -gt 0 ] && [ $((now_epoch - last_parent_heartbeat_epoch)) -ge "$PARENT_HEARTBEAT_INTERVAL" ]; then
    emit_parent_heartbeat
    last_parent_heartbeat_epoch="$now_epoch"
  fi
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
