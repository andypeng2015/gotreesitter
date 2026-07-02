#!/usr/bin/env bash
# Tier classification scan — run on every release.
#
# Measures every grammar listed in cgo_harness/tier_scan/exts.tsv against the
# tree-sitter C oracle over a real source corpus (REPRO_N first-sorted files,
# 32B..200KB) and classifies it:
#
#   CLEAN    parityMatch == measured files (100%, files > 0)
#   TIER-IV  anything below 100% (incorrect parse vs the C oracle)
#   UNMEASURED  no corpus dir / zero eligible files / timeout
#
# The committed ratchet (cgo_harness/tier_scan/clean_grammars.txt) makes tier
# IV strictly transitory: any previously-clean grammar that drops below 100%
# FAILS the scan (exit 1). Newly-clean grammars are reported so the ratchet
# can be advanced in the same release PR.
#
# Usage:
#   GTS_CORPUS_DIR=/path/to/corpus_sources cgo_harness/docker/run_tier_scan.sh [out_dir]
#
# Env:
#   GTS_CORPUS_DIR        corpus root with per-grammar subdirs (required)
#   GTS_TIER_SCAN_N       files per grammar (default 40)
#   GTS_TIER_SCAN_ALL_FILES
#                         set 1 to select all eligible files after sorting;
#                         trumps GTS_TIER_SCAN_N for manifest selection
#   GTS_TIER_SCAN_ROUNDS  timing parse rounds per measured file (default 3)
#   GTS_TIER_SCAN_TIMEOUT per-grammar timeout seconds (default 600)
#   GTS_TIER_SCAN_KILL_AFTER
#                         SIGKILL grace after timeout SIGTERM (default 30s)
#   GTS_TIER_SCAN_LANGS   optional comma/whitespace grammar allowlist
#   GTS_TIER_SCAN_START_AFTER
#                         skip grammars until after this grammar name
#   GTS_TIER_SCAN_LIMIT   stop after N selected grammars
#   GTS_TIER_SCAN_HEARTBEAT
#                         per-grammar heartbeat seconds; 0 disables (default 60).
#                         Heartbeats include last MEASURE-PROGRESS phase/file
#                         seen in the raw per-grammar log.
#   GTS_TIER_SCAN_MAX_FRAME_ROWS
#                         max dense MEASURE-PROGRESS frame rows to copy into
#                         frames/events JSONL (default 20000; 0 is unlimited).
#                         Terminal timeout/fail and compact parser summary rows
#                         are retained outside this dense-row cap.
#   GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS
#                         max dense PARSE-PROGRESS rows to copy into frames/events
#                         per raw log (default 0). A compact parser summary row
#                         with count and last parser progress is still retained.
#   GTS_TIER_SCAN_MAX_RAW_LOG_BYTES
#                         max bytes retained in each raw measure-*.log after
#                         frame ingestion (default 1048576; 0 is unlimited).
#                         Over-cap logs are deterministically compacted as
#                         head + marker + tail so terminal context remains.
#   GTS_TIER_SCAN_ISOLATE_FILES
#                         set 1 to run one timeout-bounded measure child per
#                         manifest-selected file and aggregate the rows
#   GTS_TIER_SCAN_FRAMES comma-separated frame selector for isolated per-file
#                         execution, e.g. 1,3-5. Also accepts sha256:<prefix>,
#                         base:<filename>, and path:<substring>. Applied after
#                         first-N manifest selection and preserves original
#                         index/total values.
#   GTS_TIER_SCAN_PLAN_ONLY
#                         set 1 to write deterministic manifests/control
#                         artifacts without running parser measure frames
#   GTS_TIER_SCAN_SKIP_TIER_PUBLISH
#                         set 1 to skip cgo_harness/tier_scan/tiers.{md,json}
#                         rewrite
#   GOT_PARSE_PROGRESS    when set by a caller, threads parser-loop progress
#                         into measure children and replay commands
#   GOT_PARSE_PROGRESS_INTERVAL_MS
#                         optional parser-loop progress interval in ms
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HARNESS="$REPO_ROOT/cgo_harness"
EXTS_TSV="$HARNESS/tier_scan/exts.tsv"
RATCHET="$HARNESS/tier_scan/clean_grammars.txt"
CORPUS="${GTS_CORPUS_DIR:?set GTS_CORPUS_DIR to the corpus_sources root}"
N="${GTS_TIER_SCAN_N:-40}"
ALL_FILES="${GTS_TIER_SCAN_ALL_FILES:-0}"
ROUNDS="${GTS_TIER_SCAN_ROUNDS:-3}"
PER_GRAMMAR_TIMEOUT="${GTS_TIER_SCAN_TIMEOUT:-600}"
KILL_AFTER="${GTS_TIER_SCAN_KILL_AFTER:-30s}"
LANGS_ALLOWLIST="${GTS_TIER_SCAN_LANGS:-}"
START_AFTER="${GTS_TIER_SCAN_START_AFTER:-}"
LIMIT="${GTS_TIER_SCAN_LIMIT:-}"
HEARTBEAT="${GTS_TIER_SCAN_HEARTBEAT:-60}"
MAX_FRAME_ROWS="${GTS_TIER_SCAN_MAX_FRAME_ROWS:-20000}"
KEEP_PARSER_PROGRESS_ROWS="${GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS:-0}"
MAX_RAW_LOG_BYTES="${GTS_TIER_SCAN_MAX_RAW_LOG_BYTES:-1048576}"
ISOLATE_FILES="${GTS_TIER_SCAN_ISOLATE_FILES:-0}"
FRAME_FILTER="${GTS_TIER_SCAN_FRAMES:-}"
PLAN_ONLY="${GTS_TIER_SCAN_PLAN_ONLY:-0}"
if [ "$PLAN_ONLY" = "1" ]; then
  ISOLATE_FILES=1
fi
if [ "$FRAME_FILTER" = "all" ] || [ "$FRAME_FILTER" = "*" ]; then
  FRAME_FILTER=""
fi
OUT_DIR="${1:-$HARNESS/harness_out/tier_scan/$(date -u +%Y%m%dT%H%M%SZ)}"
PARSE_PROGRESS_ENV=()
if [ -n "${GOT_PARSE_PROGRESS:-}" ]; then
  PARSE_PROGRESS_ENV+=("GOT_PARSE_PROGRESS=$GOT_PARSE_PROGRESS")
fi
if [ -n "${GOT_PARSE_PROGRESS_INTERVAL_MS:-}" ]; then
  PARSE_PROGRESS_ENV+=("GOT_PARSE_PROGRESS_INTERVAL_MS=$GOT_PARSE_PROGRESS_INTERVAL_MS")
fi
if [[ "$OUT_DIR" != /* ]]; then
  OUT_DIR="$REPO_ROOT/$OUT_DIR"
fi
if [ -n "$FRAME_FILTER" ] && [ "$ISOLATE_FILES" != "1" ]; then
  echo "GTS_TIER_SCAN_FRAMES requires GTS_TIER_SCAN_ISOLATE_FILES=1" >&2
  exit 2
fi
case "$PLAN_ONLY" in
  0|1) ;;
  *)
    echo "unknown GTS_TIER_SCAN_PLAN_ONLY=$PLAN_ONLY (want 0 or 1)" >&2
    exit 2
    ;;
esac
case "$ALL_FILES" in
  0|1) ;;
  *)
    echo "unknown GTS_TIER_SCAN_ALL_FILES=$ALL_FILES (want 0 or 1)" >&2
    exit 2
    ;;
esac
case "$MAX_FRAME_ROWS" in
  ''|*[!0-9]*)
    echo "invalid GTS_TIER_SCAN_MAX_FRAME_ROWS=$MAX_FRAME_ROWS (want nonnegative integer; 0 means unlimited)" >&2
    exit 2
    ;;
esac
case "$KEEP_PARSER_PROGRESS_ROWS" in
  ''|*[!0-9]*)
    echo "invalid GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS=$KEEP_PARSER_PROGRESS_ROWS (want nonnegative integer)" >&2
    exit 2
    ;;
esac
case "$MAX_RAW_LOG_BYTES" in
  ''|*[!0-9]*)
    echo "invalid GTS_TIER_SCAN_MAX_RAW_LOG_BYTES=$MAX_RAW_LOG_BYTES (want nonnegative integer; 0 means unlimited)" >&2
    exit 2
    ;;
esac
export GTS_TIER_SCAN_MAX_FRAME_ROWS="$MAX_FRAME_ROWS"
export GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS="$KEEP_PARSER_PROGRESS_ROWS"
export GTS_TIER_SCAN_MAX_RAW_LOG_BYTES="$MAX_RAW_LOG_BYTES"
mkdir -p "$OUT_DIR"
rm -f "$OUT_DIR"/measure-*.log "$OUT_DIR"/diagnostic_summary.json "$OUT_DIR"/diagnostic_summary.md
REPORT="$OUT_DIR/tier_scan.txt"
CLEAN_OUT="$OUT_DIR/clean.txt"
TIER_IV_OUT="$OUT_DIR/tier_iv.txt"
UNMEASURED_OUT="$OUT_DIR/unmeasured.txt"
PROGRESS_OUT="$OUT_DIR/progress.log"
STATUS_OUT="$OUT_DIR/status.tsv"
ACTIVE_OUT="$OUT_DIR/active_grammar.txt"
VISITED_OUT="$OUT_DIR/visited_grammars.txt"
SUMMARY_OUT="$OUT_DIR/summary.json"
CHECKPOINT_OUT="$OUT_DIR/checkpoint.env"
EVENTS_OUT="$OUT_DIR/events.jsonl"
FRAMES_OUT="$OUT_DIR/frames.jsonl"
MANIFEST_OUT="$OUT_DIR/manifest.json"
RESUME_OUT="$OUT_DIR/resume.env"
DIAGNOSTIC_SUMMARY="$HARNESS/tier_scan/summarize_scan.py"
: > "$REPORT"; : > "$CLEAN_OUT"; : > "$TIER_IV_OUT"; : > "$UNMEASURED_OUT"
: > "$PROGRESS_OUT"; : > "$STATUS_OUT"; : > "$ACTIVE_OUT"; : > "$VISITED_OUT"
: > "$EVENTS_OUT"; : > "$FRAMES_OUT"
python3 - "$MANIFEST_OUT" "$OUT_DIR" <<'PY'
import json
import sys
from pathlib import Path

Path(sys.argv[1]).write_text(json.dumps({"run_dir": sys.argv[2], "grammars": []}, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
if [ -n "$FRAME_FILTER" ]; then
  python3 - "$FRAME_FILTER" <<'PY'
import sys

spec = sys.argv[1]
selectors = []

def invalid(message):
    print(message, file=sys.stderr)
    raise SystemExit(2)

for raw_part in spec.split(","):
    part = raw_part.strip()
    if not part:
        continue
    if part.startswith(("sha256:", "base:", "path:")):
        key, value = part.split(":", 1)
        if not value.strip():
            invalid(f"invalid GTS_TIER_SCAN_FRAMES {key}: empty")
        selectors.append((key, value.strip()))
        continue
    if "-" in part:
        if part.count("-") != 1:
            invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
        start_text, end_text = part.split("-", 1)
        try:
            start = int(start_text)
            end = int(end_text)
        except ValueError:
            invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
        if start <= 0 or end <= 0 or end < start:
            invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
        selectors.append(("ordinal_range", (start, end)))
    else:
        try:
            value = int(part)
        except ValueError:
            invalid(f"invalid GTS_TIER_SCAN_FRAMES ordinal: {part!r}")
        if value <= 0:
            invalid(f"invalid GTS_TIER_SCAN_FRAMES ordinal: {part!r}")
        selectors.append(("ordinal", value))
if not selectors:
    invalid("invalid GTS_TIER_SCAN_FRAMES: empty frame selection")
PY
fi

timestamp() {
  date -u +%Y-%m-%dT%H:%M:%SZ
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

console_prefixed_lines() { # $1=prefix $2=text
  local prefix text
  prefix="$1"
  text="$2"
  (
    trap '' PIPE
    while IFS= read -r line; do
      printf '%s%s\n' "$prefix" "$line" || exit 0
    done <<< "$text"
  ) 2>/dev/null || true
}

emit_status() { # $1=event $2=grammar $3=corpus_kind $4=detail
  local ts event grammar corpus_kind detail line
  ts="$(timestamp)"
  event="$1"
  grammar="$2"
  corpus_kind="$3"
  detail="${4:-}"
  line="$ts $event grammar=$grammar corpus=$corpus_kind $detail"
  printf '%s\n' "$line" >> "$PROGRESS_OUT"
  printf '%s\t%s\t%s\t%s\t%s\n' "$ts" "$event" "$grammar" "$corpus_kind" "$detail" >> "$STATUS_OUT"
  python3 - "$EVENTS_OUT" "$ts" "$event" "$grammar" "$corpus_kind" "$detail" <<'PY'
import json
import os
import sys

path, ts, event, grammar, corpus_kind, detail = sys.argv[1:]
with open(path, "a", encoding="utf-8") as f:
    f.write(json.dumps({
        "ts": ts,
        "event": event,
        "grammar": grammar,
        "corpus_kind": corpus_kind,
        "detail": detail,
        "supervisor_pid": os.getppid(),
    }, sort_keys=True) + "\n")
PY
  # Console progress is useful interactively, but harness artifacts are the
  # source of truth. Do not let a closed stdout pipe abort status finalization.
  console_printf '%s\n' "$line"
}

set_active() { # $1=grammar $2=corpus_kind $3=detail
  printf '%s\t%s\t%s\t%s\n' "$(timestamp)" "$1" "$2" "${3:-}" > "$ACTIVE_OUT"
}

is_timeout_status() { # $1=exit_status
  [ "$1" -eq 124 ] || [ "$1" -eq 137 ]
}

timeout_status_detail() { # $1=exit_status
  if [ "$1" -eq 137 ]; then
    printf 'timeout_exit=%s killed=true' "$1"
  else
    printf 'rc=%s' "$1"
  fi
}

lang_allowed() { # $1=grammar
  local grammar token
  grammar="$1"
  [ -z "$LANGS_ALLOWLIST" ] && return 0
  for token in ${LANGS_ALLOWLIST//,/ }; do
    [ "$token" = "$grammar" ] && return 0
  done
  return 1
}

count_selected_grammars() {
  local seen_start_after count grammar exts
  seen_start_after=0
  if [ -z "$START_AFTER" ]; then
    seen_start_after=1
  fi
  count=0
  while IFS=$'\t' read -r grammar exts; do
    [ -z "$grammar" ] && continue
    if [ "$seen_start_after" -eq 0 ]; then
      if [ "$grammar" = "$START_AFTER" ]; then
        seen_start_after=1
      fi
      continue
    fi
    if ! lang_allowed "$grammar"; then
      continue
    fi
    if [ -n "$LIMIT" ] && [ "$count" -ge "$LIMIT" ]; then
      break
    fi
    count=$((count + 1))
  done < "$EXTS_TSV"
  printf '%s\n' "$count"
}

write_checkpoint() { # $1=last_completed_grammar $2=status $3=detail
  local grammar checkpoint_status detail clean_count tier_iv_count unmeasured_count
  grammar="$1"
  checkpoint_status="$2"
  detail="${3:-}"
  clean_count=$(awk 'NF {n++} END {print n+0}' "$CLEAN_OUT" 2>/dev/null || printf '0')
  tier_iv_count=$(awk 'NF {n++} END {print n+0}' "$TIER_IV_OUT" 2>/dev/null || printf '0')
  unmeasured_count=$(awk 'NF {n++} END {print n+0}' "$UNMEASURED_OUT" 2>/dev/null || printf '0')
  {
    printf 'updated_at=%q\n' "$(timestamp)"
    printf 'last_completed_grammar=%q\n' "$grammar"
    printf 'status=%q\n' "$checkpoint_status"
    printf 'detail=%q\n' "$detail"
    printf 'selected_completed=%q\n' "${selected_count:-0}"
    printf 'selected_total=%q\n' "${selected_total:-0}"
    printf 'clean_count=%q\n' "$clean_count"
    printf 'tier_iv_count=%q\n' "$tier_iv_count"
    printf 'unmeasured_count=%q\n' "$unmeasured_count"
    if [ -n "$grammar" ] && [ "$grammar" != "none" ]; then
      printf 'resume_hint=%q\n' "GTS_TIER_SCAN_START_AFTER=$grammar"
    else
      printf 'resume_hint=\n'
    fi
  } > "$CHECKPOINT_OUT"
  {
    printf 'updated_at=%q\n' "$(timestamp)"
    printf 'last_completed_grammar=%q\n' "$grammar"
    printf 'status=%q\n' "$checkpoint_status"
    printf 'detail=%q\n' "$detail"
    printf 'resume_hint=%q\n' "$([ -n "$grammar" ] && [ "$grammar" != "none" ] && printf 'GTS_TIER_SCAN_START_AFTER=%s' "$grammar" || true)"
    printf 'active_file=%q\n' "$ACTIVE_OUT"
    printf 'checkpoint_file=%q\n' "$CHECKPOINT_OUT"
  } > "$RESUME_OUT"
}

last_measure_progress_detail() { # $1=raw_log
  local raw_log
  raw_log="$1"
  python3 - "$raw_log" <<'PY'
import shlex
import sys
from pathlib import Path

raw_log = Path(sys.argv[1])
try:
    with raw_log.open("r", encoding="utf-8", errors="replace") as f:
        lines = f
        last_measure = None
        last_parser = None
        parser_progress_count = 0
        for line in lines:
            line = line.rstrip("\n")
            if line.startswith("MEASURE-PROGRESS "):
                last_measure = line
            elif line.startswith("PARSE-PROGRESS "):
                last_parser = line
                parser_progress_count += 1
except FileNotFoundError:
    last_measure = None
    last_parser = None
    parser_progress_count = 0

if not last_measure and not last_parser:
    print("progress=none", end="")
    raise SystemExit

item = {}
if last_measure:
    for part in shlex.split(last_measure[len("MEASURE-PROGRESS "):]):
        if "=" in part:
            key, value = part.split("=", 1)
            item[key] = value

parts = []
if item:
    parts.append(
        "progress_phase={phase} progress_file={file} progress_path={path} "
        "progress_base={base} progress_elapsed_ms={elapsed} progress_duration_ms={duration}".format(
        phase=item.get("phase", ""),
        file=item.get("file", ""),
        path=item.get("path", ""),
        base=item.get("base", ""),
        elapsed=item.get("elapsed_ms", ""),
        duration=item.get("duration_ms", ""),
        )
    )
parser = {}
if last_parser:
    for part in shlex.split(last_parser[len("PARSE-PROGRESS "):]):
        if "=" in part:
            key, value = part.split("=", 1)
            parser[key] = value
if parser:
    parser_fields = {
        "phase": "parser_phase",
        "elapsed_ms": "parser_elapsed_ms",
        "iter": "parser_iter",
        "tokens": "parser_tokens",
        "stacks": "parser_stacks",
        "live_stacks": "parser_live_stacks",
        "max_stacks": "parser_max_stacks",
        "node_count": "parser_node_count",
        "token_start": "parser_token_start",
        "token_end": "parser_token_end",
    }
    parts.append(" ".join(f"{out_key}={parser.get(in_key, '')}" for in_key, out_key in parser_fields.items()))
    parts.append(f"parser_progress_count={parser_progress_count}")
print(" ".join(parts), end="")
PY
}

compact_raw_measure_log() { # $1=raw_log
  local raw_log
  raw_log="$1"
  if [ "$MAX_RAW_LOG_BYTES" -eq 0 ] 2>/dev/null; then
    return 0
  fi
  python3 - "$raw_log" "$MAX_RAW_LOG_BYTES" <<'PY'
import os
import sys
import tempfile
from pathlib import Path

path = Path(sys.argv[1])
cap = int(sys.argv[2])
if cap <= 0:
    raise SystemExit
try:
    size = path.stat().st_size
except FileNotFoundError:
    raise SystemExit
if size <= cap:
    raise SystemExit

marker = (
    f"\n[GTS_TIER_SCAN_RAW_LOG_COMPACTED original_bytes={size} "
    f"retained_bytes_limit={cap} omitted_middle_bytes={{omitted}}]\n"
).encode("utf-8")
if cap <= len(marker) + 2:
    head_len = 0
    tail_len = cap
    marker_bytes = b""
else:
    remaining = cap - len(marker)
    head_len = remaining // 2
    tail_len = remaining - head_len
    omitted = max(size - head_len - tail_len, 0)
    marker_bytes = (
        f"\n[GTS_TIER_SCAN_RAW_LOG_COMPACTED original_bytes={size} "
        f"retained_bytes_limit={cap} omitted_middle_bytes={omitted}]\n"
    ).encode("utf-8")
    # Keep the promised hard cap even if decimal expansion made the marker longer.
    while head_len + len(marker_bytes) + tail_len > cap and head_len > 0:
        head_len -= 1
    while head_len + len(marker_bytes) + tail_len > cap and tail_len > 0:
        tail_len -= 1

with path.open("rb") as src:
    head = src.read(head_len) if head_len else b""
    if tail_len:
        src.seek(max(size - tail_len, 0))
        tail = src.read(tail_len)
    else:
        tail = b""

fd, tmp_name = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=str(path.parent))
try:
    with os.fdopen(fd, "wb") as dst:
        dst.write(head)
        dst.write(marker_bytes)
        dst.write(tail)
    os.replace(tmp_name, path)
finally:
    try:
        os.unlink(tmp_name)
    except FileNotFoundError:
        pass
PY
}

write_measure_manifest() { # $1=grammar $2=exts $3=corpus_root $4=corpus_kind
  local grammar exts corpus_root corpus_kind manifest_tmp
  grammar="$1"
  exts="$2"
  corpus_root="$3"
  corpus_kind="$4"
  manifest_tmp="$OUT_DIR/manifest-${grammar}-${corpus_kind}.json"
  python3 - "$MANIFEST_OUT" "$manifest_tmp" "$grammar" "$exts" "$corpus_root" "$corpus_kind" "$N" "$ALL_FILES" "$ROUNDS" "$ISOLATE_FILES" "$FRAME_FILTER" \
    "$BIN" "$PER_GRAMMAR_TIMEOUT" "$KILL_AFTER" "${GOT_PARSE_PROGRESS:-}" "${GOT_PARSE_PROGRESS_INTERVAL_MS:-}" <<'PY'
import hashlib
import json
import os
import shlex
import sys
from pathlib import Path

aggregate_path = Path(sys.argv[1])
manifest_path = Path(sys.argv[2])
grammar, exts_raw, corpus_root, corpus_kind = sys.argv[3:7]
n = int(sys.argv[7])
all_files = sys.argv[8] == "1"
rounds = int(sys.argv[9])
isolate_files = sys.argv[10] == "1"
frame_filter = sys.argv[11]
measure_bin = sys.argv[12]
per_grammar_timeout = sys.argv[13]
kill_after = sys.argv[14]
got_parse_progress = sys.argv[15]
got_parse_progress_interval_ms = sys.argv[16]
grammar_dir = Path(corpus_root) / grammar
exts = [e.strip().lower() for e in exts_raw.split(",") if e.strip()]

def parse_frame_filter(spec):
    if not spec:
        return []
    selectors = []
    def invalid(message):
        print(message, file=sys.stderr)
        raise SystemExit(2)
    for raw_part in spec.split(","):
        part = raw_part.strip()
        if not part:
            continue
        if part.startswith("sha256:"):
            prefix = part[len("sha256:"):].strip().lower()
            if not prefix:
                invalid("invalid GTS_TIER_SCAN_FRAMES sha256 prefix: empty")
            selectors.append(("sha256", prefix))
            continue
        if part.startswith("base:"):
            base = part[len("base:"):].strip()
            if not base:
                invalid("invalid GTS_TIER_SCAN_FRAMES base: empty")
            selectors.append(("base", base))
            continue
        if part.startswith("path:"):
            path_part = part[len("path:"):].strip()
            if not path_part:
                invalid("invalid GTS_TIER_SCAN_FRAMES path: empty")
            selectors.append(("path", path_part))
            continue
        if "-" in part:
            if part.count("-") != 1:
                invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
            start_text, end_text = part.split("-", 1)
            try:
                start = int(start_text)
                end = int(end_text)
            except ValueError:
                invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
            if start <= 0 or end <= 0 or end < start:
                invalid(f"invalid GTS_TIER_SCAN_FRAMES range: {part!r}")
            selectors.append(("ordinal_set", set(range(start, end + 1))))
        else:
            try:
                value = int(part)
            except ValueError:
                invalid(
                    f"invalid GTS_TIER_SCAN_FRAMES selector: {part!r}; "
                    "use ordinals/ranges, sha256:<prefix>, base:<filename>, path:<substring>, all, or *"
                )
            if value <= 0:
                invalid(f"invalid GTS_TIER_SCAN_FRAMES ordinal: {part!r}")
            selectors.append(("ordinal", value))
    if not selectors:
        invalid("invalid GTS_TIER_SCAN_FRAMES: empty frame selection")
    return selectors

def frame_identities(items):
    out = []
    for item in items:
        sha = item.get("sha256", "")
        out.append(
            f"{item.get('index', '')}:base:{item.get('base', '')}:"
            f"sha256:{sha[:12]}:path:{item.get('path', '')}"
        )
    return "; ".join(out) if out else "none"

def ext_match(base):
    lower = base.lower()
    for ext in exts:
        if ext.startswith(".") and lower.endswith(ext):
            return True
        if lower == ext or lower.endswith("." + ext):
            return True
    return False

files = []
if grammar_dir.is_dir():
    for root, dirs, names in os.walk(grammar_dir):
        dirs[:] = [d for d in dirs if d != ".git"]
        for name in names:
            p = Path(root) / name
            if "/.git/" in str(p):
                continue
            try:
                st = p.stat()
            except OSError:
                continue
            if not (32 <= st.st_size <= 200_000) or not ext_match(p.name):
                continue
            files.append({
                "path": str(p),
                "base": p.name,
                "size": st.st_size,
            })
files.sort(key=lambda item: item["path"])
eligible_total = len(files)
if not all_files:
    files = files[:n]
selected_total = len(files)
for item in files:
    p = Path(item["path"])
    h = hashlib.sha256()
    try:
        with p.open("rb") as f:
            for chunk in iter(lambda: f.read(1024 * 1024), b""):
                h.update(chunk)
    except OSError:
        item["sha256"] = ""
    else:
        item["sha256"] = h.hexdigest()
for i, item in enumerate(files, 1):
    item["index"] = i
    item["total"] = selected_total
    item["selected_total"] = selected_total
    item["replay_env"] = {
        "CGO_ENABLED": "1",
        "REPRO_LANG": grammar,
        "REPRO_DIR": corpus_root,
        "REPRO_EXTS": exts_raw,
        "REPRO_N": "1",
        "REPRO_ROUNDS": str(rounds),
        "REPRO_FILE": item["path"],
        "REPRO_PROGRESS": "1",
    }
    if got_parse_progress:
        item["replay_env"]["GOT_PARSE_PROGRESS"] = got_parse_progress
    if got_parse_progress_interval_ms:
        item["replay_env"]["GOT_PARSE_PROGRESS_INTERVAL_MS"] = got_parse_progress_interval_ms
    env_args = [f"{key}={value}" for key, value in item["replay_env"].items()]
    item["replay_command"] = shlex.join([
        "timeout",
        f"--kill-after={kill_after}",
        per_grammar_timeout,
        "env",
        *env_args,
        measure_bin,
        "-test.run",
        "^TestMeasureDtierVsC$",
        "-test.count=1",
    ])

if isolate_files and frame_filter:
    selectors = parse_frame_filter(frame_filter)
    selected_ordinals = set()
    missing = []
    for kind, value in selectors:
        matches = set()
        for item in files:
            ordinal = int(item["index"])
            if kind == "ordinal" and ordinal == int(value):
                matches.add(ordinal)
            elif kind == "ordinal_set" and ordinal in value:
                matches.add(ordinal)
            elif kind == "sha256" and str(item.get("sha256", "")).lower().startswith(str(value)):
                matches.add(ordinal)
            elif kind == "base" and str(item.get("base", "")) == str(value):
                matches.add(ordinal)
            elif kind == "path" and str(value) in str(item.get("path", "")):
                matches.add(ordinal)
        if not matches:
            missing.append(f"{kind}:{value}")
        selected_ordinals.update(matches)
    if missing:
        print(
            f"invalid GTS_TIER_SCAN_FRAMES for {grammar}/{corpus_kind}: "
            f"selector(s) matched no frames: {', '.join(missing)}; "
            f"available: {frame_identities(files)}",
            file=sys.stderr,
        )
        raise SystemExit(2)
    files = [item for item in files if int(item["index"]) in selected_ordinals]

entry = {
    "grammar": grammar,
    "corpus_kind": corpus_kind,
    "corpus_root": corpus_root,
    "extensions": exts,
    "limit": n,
    "all_files": all_files,
    "eligible_total": eligible_total,
    "rounds": rounds,
    "frame_filter": frame_filter if isolate_files else "",
    "selected_total": selected_total,
    "executed_files": len(files),
    "files": files,
    "identifiers": {
        "go_language": grammar,
        "c_oracle": grammar,
    },
}
manifest_path.write_text(json.dumps(entry, indent=2, sort_keys=True) + "\n", encoding="utf-8")

try:
    aggregate = json.loads(aggregate_path.read_text(encoding="utf-8"))
except Exception:
    aggregate = {"grammars": []}
aggregate["grammars"] = [g for g in aggregate.get("grammars", []) if not (g.get("grammar") == grammar and g.get("corpus_kind") == corpus_kind)]
aggregate["grammars"].append({
    "grammar": grammar,
    "corpus_kind": corpus_kind,
    "manifest": str(manifest_path),
    "files": len(files),
    "all_files": all_files,
    "eligible_total": eligible_total,
    "selected_total": selected_total,
    "executed_files": len(files),
    "frame_filter": frame_filter if isolate_files else "",
    "corpus_root": corpus_root,
    "extensions": exts,
})
aggregate_path.write_text(json.dumps(aggregate, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

ingest_measure_frames() { # $1=grammar $2=corpus_kind $3=raw_log $4=rc $5=terminal_event [$6=fallback_file $7=fallback_path $8=fallback_size $9=fallback_sha]
  local grammar corpus_kind raw_log rc terminal_event fallback_file fallback_path fallback_size fallback_sha
  grammar="$1"
  corpus_kind="$2"
  raw_log="$3"
  rc="$4"
  terminal_event="$5"
  fallback_file="${6:-}"
  fallback_path="${7:-}"
  fallback_size="${8:-}"
  fallback_sha="${9:-}"
  python3 - "$EVENTS_OUT" "$FRAMES_OUT" "$grammar" "$corpus_kind" "$raw_log" "$rc" "$terminal_event" "$fallback_file" "$fallback_path" "$fallback_size" "$fallback_sha" <<'PY'
import json
import os
import shlex
import sys
import time
from pathlib import Path

events_path = Path(sys.argv[1])
frames_path = Path(sys.argv[2])
grammar, corpus_kind, raw_log, rc, terminal_event = sys.argv[3:8]
fallback_file = sys.argv[8] if len(sys.argv) > 8 else ""
fallback_path = sys.argv[9] if len(sys.argv) > 9 else ""
fallback_size = sys.argv[10] if len(sys.argv) > 10 else ""
fallback_sha = sys.argv[11] if len(sys.argv) > 11 else ""
raw_path = Path(raw_log)
max_frame_rows = int(os.environ.get("GTS_TIER_SCAN_MAX_FRAME_ROWS", "20000") or "0")
keep_parser_progress_rows = int(os.environ.get("GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS", "0") or "0")

def parse_progress(line):
    if not line.startswith("MEASURE-PROGRESS "):
        return None
    out = {}
    for part in shlex.split(line[len("MEASURE-PROGRESS "):]):
        if "=" in part:
            k, v = part.split("=", 1)
            out[k] = v
    return out

def parse_parser_progress(line):
    if not line.startswith("PARSE-PROGRESS "):
        return None
    out = {}
    for part in shlex.split(line[len("PARSE-PROGRESS "):]):
        if "=" in part:
            k, v = part.split("=", 1)
            out[k] = v
    return out

def maybe_int(value):
    if isinstance(value, str) and value.lstrip("-").isdigit():
        return int(value)
    return value if value != "" else None

def parser_fields(item):
    return {
        "parser_phase": item.get("phase", ""),
        "parser_elapsed_ms": maybe_int(item.get("elapsed_ms", "")),
        "parser_iter": maybe_int(item.get("iter", "")),
        "parser_tokens": maybe_int(item.get("tokens", "")),
        "parser_stacks": maybe_int(item.get("stacks", "")),
        "parser_live_stacks": maybe_int(item.get("live_stacks", "")),
        "parser_max_stacks": maybe_int(item.get("max_stacks", "")),
        "parser_node_count": maybe_int(item.get("node_count", "")),
        "parser_token_start": maybe_int(item.get("token_start", "")),
        "parser_token_end": maybe_int(item.get("token_end", "")),
    }

def lifecycle_for(phase):
    if phase.endswith("_start") or phase in {"selected_files", "selected_file"}:
        return "started" if phase.endswith("_start") else "scheduled"
    if phase.endswith("_end") or phase in {"file_complete", "comparison_result", "read_skip", "go_parse_status"}:
        return "ended"
    if phase.endswith("_panic"):
        return "panic"
    return "observed"

last = None
last_parser = None
parser_progress_count = 0
parser_progress_written = 0
now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())

def existing_frame_rows():
    if max_frame_rows <= 0 or not frames_path.exists():
        return 0
    with frames_path.open("r", encoding="utf-8", errors="replace") as f:
        return sum(1 for line in f if line.strip())

frame_rows_remaining = None if max_frame_rows <= 0 else max(max_frame_rows - existing_frame_rows(), 0)

def can_write_dense_frame():
    return frame_rows_remaining is None or frame_rows_remaining > 0

def note_dense_frame_written():
    global frame_rows_remaining
    if frame_rows_remaining is not None:
        frame_rows_remaining -= 1

def write_frame_and_event(frames, events, frame):
    frames.write(json.dumps(frame, sort_keys=True) + "\n")
    events.write(json.dumps({
        "ts": now,
        "event": "FRAME",
        "grammar": grammar,
        "corpus_kind": corpus_kind,
        "frame": frame,
    }, sort_keys=True) + "\n")

def iter_lines(path):
    try:
        with path.open("r", encoding="utf-8", errors="replace") as f:
            for line in f:
                yield line.rstrip("\n")
    except FileNotFoundError:
        return

with events_path.open("a", encoding="utf-8") as events, frames_path.open("a", encoding="utf-8") as frames:
    for line in iter_lines(raw_path):
        item = parse_progress(line)
        if item:
            last = item
            if not can_write_dense_frame():
                continue
            phase = item.get("phase", "")
            frame = {
                "ts": now,
                "grammar": grammar,
                "corpus_kind": corpus_kind,
                "stage": "baseline",
                "mode": "isolated" if "-frame-" in Path(raw_log).name else "aggregate",
                "file": item.get("file", ""),
                "path": item.get("path", ""),
                "base": item.get("base", ""),
                "bytes": int(item.get("bytes", "-1")) if item.get("bytes", "-1").lstrip("-").isdigit() else item.get("bytes", ""),
                "source_size": int(fallback_size) if fallback_size.isdigit() else None,
                "source_sha256": fallback_sha,
                "phase": phase,
                "lifecycle": lifecycle_for(phase),
                "elapsed_ms": int(item["elapsed_ms"]) if item.get("elapsed_ms", "").isdigit() else None,
                "duration_ms": int(item["duration_ms"]) if item.get("duration_ms", "").isdigit() else None,
                "round": int(item["round"]) if item.get("round", "").isdigit() else None,
                "result": item.get("result", ""),
                "runtime": item.get("runtime", ""),
                "raw_log": raw_log,
            }
            write_frame_and_event(frames, events, frame)
            note_dense_frame_written()
            continue
        parser = parse_parser_progress(line)
        if parser:
            last_parser = parser
            parser_progress_count += 1
            if parser_progress_written >= keep_parser_progress_rows or not can_write_dense_frame():
                continue
            frame_path = fallback_path or (last.get("path", "") if last else "")
            frame = {
                "ts": now,
                "grammar": grammar,
                "corpus_kind": corpus_kind,
                "stage": "baseline",
                "mode": "isolated" if "-frame-" in Path(raw_log).name else "aggregate",
                "telemetry": "parser",
                "file": fallback_file or (last.get("file", "") if last else ""),
                "path": frame_path,
                "base": Path(frame_path).name if frame_path else (last.get("base", "") if last else ""),
                "source_size": int(fallback_size) if fallback_size.isdigit() else None,
                "source_sha256": fallback_sha,
                "phase": parser.get("phase", ""),
                "lifecycle": "observed",
                "raw_log": raw_log,
                "parser_progress": parser,
                "parser_progress_count": parser_progress_count,
            }
            frame.update(parser_fields(parser))
            write_frame_and_event(frames, events, frame)
            parser_progress_written += 1
            note_dense_frame_written()
    if parser_progress_count > 0:
        frame_path = fallback_path or (last.get("path", "") if last else "")
        parser_summary = {
            "ts": now,
            "grammar": grammar,
            "corpus_kind": corpus_kind,
            "stage": "baseline",
            "mode": "isolated" if "-frame-" in Path(raw_log).name else "aggregate",
            "telemetry": "parser",
            "file": fallback_file or (last.get("file", "") if last else ""),
            "path": frame_path,
            "base": Path(frame_path).name if frame_path else (last.get("base", "") if last else ""),
            "source_size": int(fallback_size) if fallback_size.isdigit() else None,
            "source_sha256": fallback_sha,
            "phase": "parser_progress_summary",
            "lifecycle": "observed",
            "raw_log": raw_log,
            "parser_progress": last_parser or {},
            "parser_progress_count": parser_progress_count,
            "parser_progress_rows_copied": parser_progress_written,
        }
        if last_parser:
            parser_summary.update(parser_fields(last_parser))
        write_frame_and_event(frames, events, parser_summary)
    if terminal_event in {"TIMEOUT", "FAIL"}:
        terminal_path = (last.get("path", "") if last else "") or fallback_path
        terminal_base = (last.get("base", "") if last else "") or (Path(fallback_path).name if fallback_path else "")
        terminal = {
            "ts": now,
            "grammar": grammar,
            "corpus_kind": corpus_kind,
            "stage": "baseline",
            "mode": "isolated" if "-frame-" in Path(raw_log).name else "aggregate",
            "phase": last.get("phase", "") if last else "",
            "file": last.get("file", "") if last else fallback_file,
            "path": terminal_path,
            "base": terminal_base,
            "source_size": int(fallback_size) if fallback_size.isdigit() else None,
            "source_sha256": fallback_sha,
            "lifecycle": terminal_event.lower(),
            "rc": int(rc),
            "raw_log": raw_log,
        }
        if last_parser:
            terminal["last_parser_progress"] = parser_fields(last_parser)
            terminal.update(parser_fields(last_parser))
        if parser_progress_count:
            terminal["parser_progress"] = {"count": parser_progress_count, "last": parser_fields(last_parser) if last_parser else {}}
            terminal["parser_progress_count"] = parser_progress_count
        frames.write(json.dumps(terminal, sort_keys=True) + "\n")
PY
}

write_summary() { # $1=exit_status
  local exit_status
  exit_status="$1"
  python3 - "$SUMMARY_OUT" "$OUT_DIR" "$CLEAN_OUT" "$TIER_IV_OUT" "$UNMEASURED_OUT" \
    "$VISITED_OUT" "$STATUS_OUT" "$ACTIVE_OUT" "$CHECKPOINT_OUT" "$exit_status" <<'PY'
import json
import sys
from pathlib import Path

summary_path = Path(sys.argv[1])
out_dir = sys.argv[2]
clean_path = Path(sys.argv[3])
tier_iv_path = Path(sys.argv[4])
unmeasured_path = Path(sys.argv[5])
visited_path = Path(sys.argv[6])
status_path = Path(sys.argv[7])
active_path = Path(sys.argv[8])
checkpoint_path = Path(sys.argv[9])
exit_status = int(sys.argv[10])

def line_count(path):
    if not path.exists():
        return 0
    with path.open("r", encoding="utf-8") as f:
        return sum(1 for line in f if line.strip())

event_counts = {}
if status_path.exists():
    with status_path.open("r", encoding="utf-8") as f:
        for line in f:
            fields = line.rstrip("\n").split("\t", 4)
            if len(fields) >= 2:
                event_counts[fields[1]] = event_counts.get(fields[1], 0) + 1

active_fields = []
if active_path.exists():
    active_fields = active_path.read_text(encoding="utf-8").rstrip("\n").split("\t")
while len(active_fields) < 4:
    active_fields.append("")

checkpoint = {}
if checkpoint_path.exists():
    with checkpoint_path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.rstrip("\n")
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            checkpoint[key] = value

summary = {
    "out_dir": out_dir,
    "artifacts": {
        "events_jsonl": str(Path(out_dir) / "events.jsonl"),
        "frames_jsonl": str(Path(out_dir) / "frames.jsonl"),
        "manifest_json": str(Path(out_dir) / "manifest.json"),
        "resume_env": str(Path(out_dir) / "resume.env"),
    },
    "run_state": "failed" if exit_status != 0 else ("complete" if active_fields[1] == "idle" else "incomplete"),
    "counts": {
        "clean": line_count(clean_path),
        "tier_iv": line_count(tier_iv_path),
        "unmeasured": line_count(unmeasured_path),
    },
    "selected_grammar_count": line_count(visited_path),
    "visited_grammar_count": line_count(visited_path),
    "events": {
        "timeout": event_counts.get("TIMEOUT", 0),
        "fail": event_counts.get("FAIL", 0),
        "heartbeat": event_counts.get("HEARTBEAT", 0),
    },
    "final_active": {
        "timestamp": active_fields[0],
        "grammar": active_fields[1],
        "corpus_kind": active_fields[2],
        "detail": active_fields[3],
    },
    "resume_hint": checkpoint.get("resume_hint", ""),
    "checkpoint": checkpoint,
    "exit_status": exit_status,
}
summary_path.write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY
}

summary_written=0
finalize_summary() {
  local rc
  rc=$?
  if [ "$summary_written" -eq 0 ]; then
    write_summary "$rc" || true
  fi
}
trap finalize_summary EXIT

selected_count=0
selected_total=$(count_selected_grammars)
write_checkpoint "none" "initialized" "selected_total=$selected_total"

BIN="$OUT_DIR/measure.test"
BUILD_LOG="$OUT_DIR/build-measure.log"
if [ "$PLAN_ONLY" = "1" ]; then
  console_line "plan-only: skipping measure binary build"
  : > "$BUILD_LOG"
  emit_status BUILD_SKIP "measure.test" "build" "plan_only=1 log=$BUILD_LOG"
else
  console_line "building measure binary..."
  set_active "build" "binary" "go test -c log=$BUILD_LOG"
  emit_status BUILD_START "measure.test" "build" "log=$BUILD_LOG"
  set +e
  (cd "$HARNESS" && CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o "$BIN" .) > "$BUILD_LOG" 2>&1
  build_rc=$?
  set -e
  if [ "$build_rc" -ne 0 ]; then
    emit_status BUILD_FAIL "measure.test" "build" "rc=$build_rc log=$BUILD_LOG"
    set_active "idle" "none" "build-failed rc=$build_rc log=$BUILD_LOG"
    write_summary "$build_rc"
    summary_written=1
    exit "$build_rc"
  fi
  emit_status BUILD_END "measure.test" "build" "rc=0 log=$BUILD_LOG"
fi

# Curated fallback corpus: repo-root corpus_real/<grammar> holds hand-curated
# real-world files for grammars whose external corpus checkout contains no
# matching sources (e.g. toml's checkout is the spec repo, gitcommit's is the
# grammar repo). The external corpus stays authoritative — the curated dir is
# only consulted when the external checkout yields zero eligible files.
CURATED="$REPO_ROOT/corpus_real"

MEASURE_LINE=""
MEASURE_ZERO_FILES=0
MEASURE_PLANNED_FILES=0
manifest_file_rows() { # $1=manifest_json
  local manifest_json
  manifest_json="$1"
  python3 - "$manifest_json" <<'PY'
import json
import sys
from pathlib import Path

manifest = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
files = manifest.get("files", [])
total = len(files)
for item in files:
    print(
        f"{item.get('index', 0)}\t"
        f"{item.get('total', total)}\t"
        f"{item.get('path', '')}\t"
        f"{item.get('size', '')}\t"
        f"{item.get('sha256', '')}"
    )
PY
}

manifest_repro_n() { # $1=manifest_json
  local manifest_json
  manifest_json="$1"
  python3 - "$manifest_json" "$N" <<'PY'
import json
import sys
from pathlib import Path

manifest = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
fallback = sys.argv[2]
if manifest.get("all_files"):
    selected_total = int(manifest.get("selected_total") or len(manifest.get("files", [])))
    print(selected_total if selected_total > 0 else fallback)
else:
    print(fallback)
PY
}

aggregate_isolated_measure_line() { # $1=grammar $2=status_jsonl $3...=raw_logs
  local grammar status_jsonl
  grammar="$1"
  status_jsonl="$2"
  shift 2
  python3 - "$grammar" "$status_jsonl" "$@" <<'PY'
import json
import re
import shlex
import statistics
import sys
from pathlib import Path

grammar = sys.argv[1]
status_path = Path(sys.argv[2])
logs = [Path(p) for p in sys.argv[3:]]
status_by_log = {}
if status_path.exists():
    for line in status_path.read_text(encoding="utf-8", errors="replace").splitlines():
        if not line.strip():
            continue
        try:
            item = json.loads(line)
        except json.JSONDecodeError:
            continue
        log_text = str(item.get("log") or "")
        if log_text:
            status_by_log[log_text] = item

mode = "prod"
matched = 0
total = len(logs)
diverge = 0
failed = 0
trunc = 0
err_tree = 0
panics = 0
go_ns = 0
c_ns = 0
ratios = []
missing_exact_timing = False

def parse_pair(value):
    if not value:
        return 0, 0
    m = re.match(r"^(\d+)/(\d+)", value)
    if not m:
        return 0, 0
    return int(m.group(1)), int(m.group(2))

def int_value(value):
    try:
        return int(value)
    except Exception:
        return 0

for log in logs:
    status = status_by_log.get(str(log), {})
    frame_failed = bool(status.get("failed") or status.get("timeout"))
    if frame_failed:
        failed += 1
    try:
        lines = log.read_text(encoding="utf-8", errors="replace").splitlines()
    except FileNotFoundError:
        diverge += 1
        continue
    measure = str(status.get("measure_line") or "")
    if not measure:
        for line in lines:
            if line.startswith("MEASURE-DTIER "):
                measure = line
    if not measure:
        diverge += 1
        continue
    parts = shlex.split(measure)
    kv = {}
    if len(parts) > 2:
        for part in parts[2:]:
            if "=" in part:
                key, value = part.split("=", 1)
                kv[key] = value
    mode = kv.get("mode", mode)
    m, t = parse_pair(kv.get("parityMatch"))
    if frame_failed:
        diverge += 1
    elif m == 1 and t == 1:
        matched += 1
    else:
        diverge += 1
    trunc += int_value(kv.get("trunc"))
    err_tree += int_value(kv.get("errTree"))
    panics += int_value(kv.get("panics"))
    g = int_value(kv.get("goNS"))
    c = int_value(kv.get("cNS"))
    if g > 0 and c > 0:
        go_ns += g
        c_ns += c
        ratios.append(float(g) / float(c))
    elif kv.get("medianRatio"):
        missing_exact_timing = True
        try:
            ratios.append(float(kv["medianRatio"].rstrip("x")))
        except Exception:
            pass

median = statistics.median(ratios) if ratios else 0.0
agg = (float(go_ns) / float(c_ns)) if c_ns > 0 else 0.0
files = total
pct = (100.0 * float(matched) / float(total)) if total > 0 else 0.0
extra = f" isolatedFiles={len(logs)} failedFiles={failed} nonMatchFrames={diverge}"
if missing_exact_timing:
    extra += " timingApprox=median_only"
print(
    f"MEASURE-DTIER {grammar} mode={mode} files={files} "
    f"medianRatio={median:.2f}x aggRatio={agg:.2f}x "
    f"parityMatch={matched}/{total}({pct:.0f}%) diverge={diverge} "
    f"trunc={trunc} errTree={err_tree} panics={panics} "
    f"goNS={go_ns} cNS={c_ns}{extra}"
)
PY
}

record_frame_status() { # $1=status_jsonl $2=raw_log $3=rc $4=terminal_event [$5=measure_line]
  local status_jsonl raw_log rc terminal_event measure_line
  status_jsonl="$1"
  raw_log="$2"
  rc="$3"
  terminal_event="$4"
  measure_line="${5:-}"
  python3 - "$status_jsonl" "$raw_log" "$rc" "$terminal_event" "$measure_line" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
rc = int(sys.argv[3])
terminal = sys.argv[4]
measure_line = sys.argv[5]
with path.open("a", encoding="utf-8") as f:
    f.write(json.dumps({
        "log": sys.argv[2],
        "rc": rc,
        "terminal_event": terminal,
        "failed": terminal != "END" or rc != 0,
        "timeout": terminal == "TIMEOUT",
        "measure_line": measure_line,
    }, sort_keys=True) + "\n")
PY
}

measure_grammar_isolated() { # $1=grammar $2=exts $3=corpus_root $4=corpus_kind
  local grammar exts corpus_root corpus_kind detail manifest_json frame_index frame_total frame_path frame_size frame_sha
  local raw_log rc failed_count line logs frame_label frame_pid heartbeat_pid status_jsonl
  grammar="$1"
  exts="$2"
  corpus_root="$3"
  corpus_kind="$4"
  manifest_json="$OUT_DIR/manifest-${grammar}-${corpus_kind}.json"
  status_jsonl="$OUT_DIR/measure-${grammar}-${corpus_kind}-frame-status.jsonl"
  detail="dir=$corpus_root exts=$exts n=$N all_files=$ALL_FILES rounds=$ROUNDS timeout=${PER_GRAMMAR_TIMEOUT}s kill_after=$KILL_AFTER isolate_files=1"

  MEASURE_LINE=""
  MEASURE_ZERO_FILES=0
  MEASURE_PLANNED_FILES=0
  write_measure_manifest "$grammar" "$exts" "$corpus_root" "$corpus_kind"
  if [ "$PLAN_ONLY" = "1" ]; then
    planned_count=$(python3 - "$manifest_json" <<'PY'
import json
import sys
from pathlib import Path
manifest = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(len(manifest.get("files", [])))
PY
)
    set_active "$grammar" "$corpus_kind" "$detail plan_only=1 planned_files=$planned_count"
    emit_status PLAN "$grammar" "$corpus_kind" "$detail plan_only=1 planned_files=$planned_count manifest=$manifest_json"
    emit_status PLAN_DONE "$grammar" "$corpus_kind" "plan_only=1 planned_files=$planned_count"
    MEASURE_PLANNED_FILES="$planned_count"
    if [ "$planned_count" -eq 0 ]; then
      MEASURE_ZERO_FILES=1
    fi
    return 0
  fi
  set_active "$grammar" "$corpus_kind" "$detail"
  emit_status START "$grammar" "$corpus_kind" "$detail"

  failed_count=0
  : > "$status_jsonl"
  logs=()
  while IFS=$'\t' read -r frame_index frame_total frame_path frame_size frame_sha; do
    [ -z "$frame_path" ] && continue
    frame_label=$(printf '%04d' "$frame_index")
    raw_log="$OUT_DIR/measure-${grammar}-${corpus_kind}-frame-${frame_label}.log"
    logs+=("$raw_log")
    set_active "$grammar" "$corpus_kind" "$detail frame=$frame_index/$frame_total file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log"
    emit_status START "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log"
    set +e
    timeout --kill-after="$KILL_AFTER" "$PER_GRAMMAR_TIMEOUT" env CGO_ENABLED=1 "${PARSE_PROGRESS_ENV[@]}" \
      REPRO_LANG="$grammar" REPRO_DIR="$corpus_root" REPRO_EXTS="$exts" REPRO_N=1 \
      REPRO_ROUNDS="$ROUNDS" \
      REPRO_FILE="$frame_path" \
      REPRO_PROGRESS=1 \
      "$BIN" -test.run '^TestMeasureDtierVsC$' -test.count=1 > "$raw_log" 2>&1 &
    frame_pid=$!
    heartbeat_pid=""
    if [ "${HEARTBEAT:-0}" -gt 0 ] 2>/dev/null; then
      (
        start_epoch=$(date -u +%s)
        while kill -0 "$frame_pid" 2>/dev/null; do
          sleep "$HEARTBEAT"
          if kill -0 "$frame_pid" 2>/dev/null; then
            now_epoch=$(date -u +%s)
            elapsed=$((now_epoch - start_epoch))
            log_bytes=$(wc -c < "$raw_log" 2>/dev/null || printf '0')
            if [ -e "$raw_log" ]; then
              progress_age_s=$((now_epoch - $(stat -c %Y "$raw_log" 2>/dev/null || printf '%s' "$now_epoch")))
            else
              progress_age_s=
            fi
            progress_detail=$(last_measure_progress_detail "$raw_log")
            emit_status HEARTBEAT "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log elapsed_s=${elapsed} supervisor_pid=$$ worker_pid=$frame_pid log_bytes=$log_bytes progress_age_s=${progress_age_s:-} $progress_detail"
          fi
        done
      ) &
      heartbeat_pid=$!
    fi
    wait "$frame_pid"
    rc=$?
    if [ -n "$heartbeat_pid" ]; then
      kill "$heartbeat_pid" 2>/dev/null || true
      wait "$heartbeat_pid" 2>/dev/null || true
    fi
    set -e
    line=$(grep -E '^MEASURE-DTIER' "$raw_log" | tail -n 1 || true)
    if [ "$rc" -eq 0 ] && [ -n "$line" ]; then
      ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "END" "$frame_index/$frame_total" "$frame_path" "$frame_size" "$frame_sha"
      record_frame_status "$status_jsonl" "$raw_log" "$rc" "END" "$line"
      emit_status END "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total rc=$rc file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log"
      compact_raw_measure_log "$raw_log"
      continue
    fi
    failed_count=$((failed_count + 1))
    if is_timeout_status "$rc"; then
      ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "TIMEOUT" "$frame_index/$frame_total" "$frame_path" "$frame_size" "$frame_sha"
      record_frame_status "$status_jsonl" "$raw_log" "$rc" "TIMEOUT"
      emit_status TIMEOUT "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total $(timeout_status_detail "$rc") file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log $(last_measure_progress_detail "$raw_log")"
      compact_raw_measure_log "$raw_log"
    else
      ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "FAIL" "$frame_index/$frame_total" "$frame_path" "$frame_size" "$frame_sha"
      record_frame_status "$status_jsonl" "$raw_log" "$rc" "FAIL"
      if [ "$rc" -eq 0 ]; then
        emit_status FAIL "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total rc=$rc no-measure-line file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log $(last_measure_progress_detail "$raw_log")"
      else
        emit_status FAIL "$grammar" "$corpus_kind" "frame=$frame_index/$frame_total rc=$rc file=$frame_path size=$frame_size sha256=$frame_sha log=$raw_log $(last_measure_progress_detail "$raw_log")"
      fi
      compact_raw_measure_log "$raw_log"
    fi
  done < <(manifest_file_rows "$manifest_json")

  if [ "${#logs[@]}" -eq 0 ]; then
    MEASURE_LINE=""
    MEASURE_ZERO_FILES=1
  else
    MEASURE_LINE="$(aggregate_isolated_measure_line "$grammar" "$status_jsonl" "${logs[@]}")"
  fi
  emit_status END "$grammar" "$corpus_kind" "rc=0 aggregated isolate_files=1 failed_files=$failed_count logs=${#logs[@]}"
  return 0
}

measure_grammar() { # $1=grammar $2=exts $3=corpus_root $4=corpus_kind
  local grammar exts corpus_root corpus_kind raw_log rc line detail measure_pid heartbeat_pid manifest_json repro_n
  grammar="$1"
  exts="$2"
  corpus_root="$3"
  corpus_kind="$4"
  raw_log="$OUT_DIR/measure-${grammar}-${corpus_kind}.log"
  manifest_json="$OUT_DIR/manifest-${grammar}-${corpus_kind}.json"
  detail="dir=$corpus_root exts=$exts n=$N all_files=$ALL_FILES rounds=$ROUNDS timeout=${PER_GRAMMAR_TIMEOUT}s kill_after=$KILL_AFTER"

  MEASURE_LINE=""
  MEASURE_ZERO_FILES=0
  MEASURE_PLANNED_FILES=0
  if [ "$ISOLATE_FILES" = "1" ]; then
    measure_grammar_isolated "$grammar" "$exts" "$corpus_root" "$corpus_kind"
    return $?
  fi
  write_measure_manifest "$grammar" "$exts" "$corpus_root" "$corpus_kind"
  repro_n="$(manifest_repro_n "$manifest_json")"
  set_active "$grammar" "$corpus_kind" "$detail"
  emit_status START "$grammar" "$corpus_kind" "$detail"

  set +e
  timeout --kill-after="$KILL_AFTER" "$PER_GRAMMAR_TIMEOUT" env CGO_ENABLED=1 "${PARSE_PROGRESS_ENV[@]}" \
    REPRO_LANG="$grammar" REPRO_DIR="$corpus_root" REPRO_EXTS="$exts" REPRO_N="$repro_n" \
    REPRO_ROUNDS="$ROUNDS" \
    REPRO_PROGRESS=1 \
    "$BIN" -test.run '^TestMeasureDtierVsC$' -test.count=1 > "$raw_log" 2>&1 &
  measure_pid=$!
  heartbeat_pid=""
  if [ "${HEARTBEAT:-0}" -gt 0 ] 2>/dev/null; then
    (
      start_epoch=$(date -u +%s)
      while kill -0 "$measure_pid" 2>/dev/null; do
        sleep "$HEARTBEAT"
        if kill -0 "$measure_pid" 2>/dev/null; then
          now_epoch=$(date -u +%s)
          elapsed=$((now_epoch - start_epoch))
          log_bytes=$(wc -c < "$raw_log" 2>/dev/null || printf '0')
          if [ -e "$raw_log" ]; then
            progress_age_s=$((now_epoch - $(stat -c %Y "$raw_log" 2>/dev/null || printf '%s' "$now_epoch")))
          else
            progress_age_s=
          fi
          progress_detail=$(last_measure_progress_detail "$raw_log")
          emit_status HEARTBEAT "$grammar" "$corpus_kind" "elapsed_s=${elapsed} supervisor_pid=$$ worker_pid=$measure_pid log=$raw_log log_bytes=$log_bytes progress_age_s=${progress_age_s:-} $progress_detail"
        fi
      done
    ) &
    heartbeat_pid=$!
  fi
  wait "$measure_pid"
  rc=$?
  if [ -n "$heartbeat_pid" ]; then
    kill "$heartbeat_pid" 2>/dev/null || true
    wait "$heartbeat_pid" 2>/dev/null || true
  fi
  set -e

  line=$(grep -E '^MEASURE-DTIER' "$raw_log" | tail -n 1 || true)
  if [ "$rc" -eq 0 ] && [ -n "$line" ]; then
    MEASURE_LINE="$line"
    ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "END"
    emit_status END "$grammar" "$corpus_kind" "rc=$rc log=$raw_log"
    compact_raw_measure_log "$raw_log"
    return 0
  fi
  if is_timeout_status "$rc"; then
    ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "TIMEOUT"
    emit_status TIMEOUT "$grammar" "$corpus_kind" "$(timeout_status_detail "$rc") log=$raw_log $(last_measure_progress_detail "$raw_log")"
    compact_raw_measure_log "$raw_log"
    return "$rc"
  fi
  ingest_measure_frames "$grammar" "$corpus_kind" "$raw_log" "$rc" "FAIL"
  if [ "$rc" -eq 0 ]; then
    emit_status FAIL "$grammar" "$corpus_kind" "rc=$rc no-measure-line log=$raw_log $(last_measure_progress_detail "$raw_log")"
  else
    emit_status FAIL "$grammar" "$corpus_kind" "rc=$rc log=$raw_log $(last_measure_progress_detail "$raw_log")"
  fi
  compact_raw_measure_log "$raw_log"
  return "$rc"
}

seen_start_after=0
if [ -z "$START_AFTER" ]; then
  seen_start_after=1
fi

while IFS=$'\t' read -r grammar exts; do
  [ -z "$grammar" ] && continue
  if [ "$seen_start_after" -eq 0 ]; then
    if [ "$grammar" = "$START_AFTER" ]; then
      seen_start_after=1
    fi
    continue
  fi
  if ! lang_allowed "$grammar"; then
    continue
  fi
  if [ -n "$LIMIT" ] && [ "$selected_count" -ge "$LIMIT" ]; then
    break
  fi
  selected_count=$((selected_count + 1))
  echo "$grammar" >> "$VISITED_OUT"
  emit_status START "$grammar" "grammar" "index=$selected_count selected_total=$selected_total limit=${LIMIT:-all} start_after=${START_AFTER:-none} exts=$exts"
  line=""
  external_status=0
  external_attempted=0
  external_had_zero_files=0
  external_planned_files=0
  curated_status=0
  curated_had_zero_files=0
  curated_planned_files=0
  if [ -d "$CORPUS/$grammar" ]; then
    external_attempted=1
    measure_grammar "$grammar" "$exts" "$CORPUS" "external" || external_status=$?
    external_planned_files="${MEASURE_PLANNED_FILES:-0}"
    line="$MEASURE_LINE"
    files=""
    if [ -n "$line" ]; then
      files=$(awk -F= '{print $2}' <<<"$(awk '{print $4}' <<<"$line")")
    fi
    if [ -n "$line" ] && [ "${files:-0}" = "0" ]; then
      external_had_zero_files=1
    fi
    if [ "${MEASURE_ZERO_FILES:-0}" = "1" ]; then
      external_had_zero_files=1
    fi
  else
    emit_status SKIP "$grammar" "external" "no-corpus dir=$CORPUS/$grammar"
  fi
  if [ "$PLAN_ONLY" = "1" ]; then
    if [ "${external_planned_files:-0}" -le 0 ]; then
      if [ -d "$CURATED/$grammar" ]; then
        measure_grammar "$grammar" "$exts" "$CURATED" "curated" || curated_status=$?
        curated_planned_files="${MEASURE_PLANNED_FILES:-0}"
      else
        emit_status SKIP "$grammar" "curated" "no-corpus dir=$CURATED/$grammar"
      fi
    fi
  else
    if { [ -z "$line" ] || [ "${files:-0}" = "0" ]; } && [ -d "$CURATED/$grammar" ]; then
      measure_grammar "$grammar" "$exts" "$CURATED" "curated" || curated_status=$?
      curated_line="$MEASURE_LINE"
      if [ "${MEASURE_ZERO_FILES:-0}" = "1" ]; then
        curated_had_zero_files=1
      fi
      if [ -n "$curated_line" ]; then
        line="$curated_line corpus=curated"
        files=$(awk -F= '{print $2}' <<<"$(awk '{print $4}' <<<"$curated_line")")
      fi
    elif [ -z "$line" ] || [ "${files:-0}" = "0" ]; then
      emit_status SKIP "$grammar" "curated" "no-corpus dir=$CURATED/$grammar"
    fi
  fi
  if [ "$PLAN_ONLY" = "1" ]; then
    planned_files=$(( ${external_planned_files:-0} + ${curated_planned_files:-0} ))
    if [ "$planned_files" -gt 0 ]; then
      emit_status END "$grammar" "grammar" "classification=planned plan_only=1 planned_files=$planned_files external_planned_files=${external_planned_files:-0} curated_planned_files=${curated_planned_files:-0}"
      write_checkpoint "$grammar" "planned" "index=$selected_count selected_total=$selected_total planned_files=$planned_files external_planned_files=${external_planned_files:-0} curated_planned_files=${curated_planned_files:-0}"
    else
      emit_status END "$grammar" "grammar" "classification=planned-empty plan_only=1 planned_files=0 external_status=$external_status curated_status=$curated_status"
      write_checkpoint "$grammar" "planned-empty" "index=$selected_count selected_total=$selected_total planned_files=0 external_status=$external_status curated_status=$curated_status"
    fi
    continue
  fi
  if [ -z "$line" ]; then
    if is_timeout_status "$external_status"; then
      echo "$grammar timeout external" >> "$UNMEASURED_OUT"
      emit_status TIMEOUT "$grammar" "external" "unmeasured $(timeout_status_detail "$external_status")"
    elif is_timeout_status "${curated_status:-0}"; then
      echo "$grammar timeout curated" >> "$UNMEASURED_OUT"
      emit_status TIMEOUT "$grammar" "curated" "unmeasured $(timeout_status_detail "$curated_status")"
    elif [ "${curated_status:-0}" -ne 0 ]; then
      echo "$grammar fail curated" >> "$UNMEASURED_OUT"
      emit_status FAIL "$grammar" "curated" "unmeasured"
    elif [ "$curated_had_zero_files" -eq 1 ]; then
      echo "$grammar zero-files curated" >> "$UNMEASURED_OUT"
      emit_status SKIP "$grammar" "curated" "zero-files"
    elif [ "$external_attempted" -eq 1 ] && [ "$external_had_zero_files" -eq 1 ] && [ ! -d "$CURATED/$grammar" ]; then
      echo "$grammar zero-files external" >> "$UNMEASURED_OUT"
      emit_status SKIP "$grammar" "external" "zero-files"
    elif [ -d "$CORPUS/$grammar" ]; then
      echo "$grammar fail external" >> "$UNMEASURED_OUT"
      emit_status FAIL "$grammar" "external" "unmeasured"
    else
      echo "$grammar no-corpus" >> "$UNMEASURED_OUT"
      emit_status SKIP "$grammar" "grammar" "no-corpus"
    fi
    write_checkpoint "$grammar" "unmeasured" "index=$selected_count selected_total=$selected_total"
    continue
  fi
  echo "$line" >> "$REPORT"
  # parityMatch=A/B(P%) is field 7; files=N is field 4.
  parity=$(awk '{print $7}' <<<"$line")
  files=$(awk -F= '{print $2}' <<<"$(awk '{print $4}' <<<"$line")")
  matched="${parity#parityMatch=}"; matched="${matched%%/*}"
  total="${parity#*/}"; total="${total%%(*}"
  if [ "$files" -gt 0 ] && [ "$matched" = "$total" ]; then
    echo "$grammar" >> "$CLEAN_OUT"
    emit_status END "$grammar" "grammar" "classification=clean files=$files parity=$parity"
    write_checkpoint "$grammar" "clean" "index=$selected_count selected_total=$selected_total files=$files parity=$parity"
  else
    echo "$grammar $parity" >> "$TIER_IV_OUT"
    emit_status END "$grammar" "grammar" "classification=tier_iv files=$files parity=$parity"
    write_checkpoint "$grammar" "tier_iv" "index=$selected_count selected_total=$selected_total files=$files parity=$parity"
  fi
done < "$EXTS_TSV"

set_active "idle" "none" "scan-loop-complete"

# From here on, correctness artifacts are already written and the script
# manages the final gate through the explicit status variable. In delegated
# wringer runs stdout may be closed by the parent; do not let informational
# console writes convert completed evidence into rc=141 infrastructure failure.
set +e

sort -o "$CLEAN_OUT" "$CLEAN_OUT"
sort -o "$VISITED_OUT" "$VISITED_OUT"
console_line
console_line "=== tier scan summary ($OUT_DIR)"
console_line "clean:      $(wc -l < "$CLEAN_OUT")"
console_line "tier IV:    $(wc -l < "$TIER_IV_OUT")"
console_line "unmeasured: $(wc -l < "$UNMEASURED_OUT")"
console_line "visited:    $(wc -l < "$VISITED_OUT")"

if [ "$PLAN_ONLY" = "1" ]; then
  console_line
  console_line "=== plan-only complete (parser measure frames skipped)"
  write_summary 0
  summary_written=1
  console_line "summary: $SUMMARY_OUT"
  exit 0
fi

status=0
regressions=$(comm -23 <(comm -12 <(sort "$RATCHET") "$VISITED_OUT") "$CLEAN_OUT")
if [ -n "$regressions" ]; then
  console_line
  console_line "RATCHET REGRESSIONS (previously clean, now tier IV/unmeasured):"
  console_prefixed_lines "  " "$regressions"
  status=1
fi
newly_clean=$(comm -13 <(sort "$RATCHET") "$CLEAN_OUT")
if [ -n "$newly_clean" ]; then
  console_line
  console_line "NEWLY CLEAN (advance the ratchet in tier_scan/clean_grammars.txt):"
  console_prefixed_lines "  " "$newly_clean"
fi

# Tier-IV characterization gate. Tier IV = not parity-clean, full stop
# (tiers I-III are perf ranks reserved for parity-clean grammars). Every
# non-clean grammar, including unmeasured timeout/no-corpus cases, must carry a
# named assessed IV sub-cause in tier_classification.tsv (IV-recovery /
# IV-scanner / IV-shape / IV-version / IV-stackcap / IV-extmap / IV-perf /
# IV-unknown). An UNCHARACTERIZED tier-IV grammar (no row, tier
# 'IV-unassessed', or any stale non-IV row such as CLEAN) is the state we drive
# to zero.
CLASS_TSV="$HARNESS/tier_scan/tier_classification.tsv"
if [ -f "$CLASS_TSV" ]; then
  console_line
  console_line "=== tier characterization (vs tier_classification.tsv)"
  uncharacterized=""
  duplicate_classifications=$(awk -F'\t' 'NR>1 && $1 != "" {seen[$1]++} END {for (g in seen) if (seen[g] > 1) print g}' "$CLASS_TSV" | sort)
  if [ -n "$duplicate_classifications" ]; then
    console_line "DUPLICATE tier_classification.tsv rows (must be unique):"
    console_prefixed_lines "  " "$duplicate_classifications"
    status=1
  fi

  check_tier_characterization() { # $1=grammar $2=source
    local grammar source tier row_count
    grammar="$1"
    source="$2"
    [ -z "$grammar" ] && return 0
    tier=$(awk -F'\t' -v g="$grammar" '$1==g{print $2}' "$CLASS_TSV")
    row_count=$(awk -F'\t' -v g="$grammar" '$1==g{n++} END{print n+0}' "$CLASS_TSV")
    if [ "$row_count" -ne 1 ]; then
      uncharacterized="$uncharacterized $grammar:$source:duplicate-row-count-$row_count"
    elif [ -z "$tier" ] || [[ "$tier" == IV-unassessed* ]] || [[ "$tier" != IV-* ]]; then
      uncharacterized="$uncharacterized $grammar:$source:${tier:-missing}"
    fi
  }

  while read -r grammar rest; do
    [ -z "$grammar" ] && continue
    check_tier_characterization "$grammar" "tier_iv"
  done < "$TIER_IV_OUT"
  while read -r grammar rest; do
    [ -z "$grammar" ] && continue
    check_tier_characterization "$grammar" "unmeasured"
  done < "$UNMEASURED_OUT"
  if [ -n "$uncharacterized" ]; then
    console_line "UNCHARACTERIZED/STALE TIER IV (must be triaged into an IV-* sub-cause):"
    for g in $uncharacterized; do
      IFS=: read -r grammar source tier <<<"$g"
      console_line "  $grammar ($source, tier_classification.tsv tier=$tier)"
    done
    status=1
  else
    console_line "all non-clean grammars characterized (0 uncharacterized/stale) ✓"
  fi
fi

# Per-release tier publication: regenerate cgo_harness/tier_scan/tiers.{md,json}
# from the committed ratchet + classification (+ local perf evidence when
# present) and commit the refreshed artifact in the release PR. With
# GTS_TIERS_REQUIRE_ZERO_IV=1 any tier-IV grammar is release-blocking — the
# first tier-publishing release and every one after it requires IV=0.
console_line
if [ "${GTS_TIER_SCAN_SKIP_TIER_PUBLISH:-0}" = "1" ]; then
  console_line "=== tier publication skipped (GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1)"
else
  console_line "=== tier publication (cgo_harness/tier_scan/tiers.md)"
  TIERS_FLAGS=""
  if [ "${GTS_TIERS_REQUIRE_ZERO_IV:-0}" = "1" ]; then
    TIERS_FLAGS="--require-zero-iv"
  fi
  if ! python3 "$REPO_ROOT/cgo_harness/tier_scan/gen_tiers.py" \
      --version "${GTS_RELEASE_VERSION:-unreleased}" $TIERS_FLAGS; then
    status=1
  fi
fi

write_summary "$status"
summary_written=1
console_line "summary: $SUMMARY_OUT"
if ! python3 "$DIAGNOSTIC_SUMMARY" "$OUT_DIR"; then
  echo "warning: diagnostic tier scan summary failed" >&2
fi

exit "$status"
