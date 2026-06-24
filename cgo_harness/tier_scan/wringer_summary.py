#!/usr/bin/env python3
"""Summarize single-grammar wringer artifacts without running parsers."""

from __future__ import annotations

import argparse
from datetime import datetime, timezone
from functools import lru_cache
import json
import re
import shlex
import sys
from pathlib import Path
from typing import Any


DTIER_PREFIX = "MEASURE-DTIER "
PROGRESS_PREFIX = "MEASURE-PROGRESS "
PARSER_PROGRESS_PREFIX = "PARSE-PROGRESS "
COMPARISON_DIAG_PHASE = "comparison_diag"
COMPARISON_DIAG_FIELDS = (
    "diff",
    "firstDiffPath",
    "goType",
    "cType",
    "goSpan",
    "cSpan",
    "goCC",
    "cCC",
    "goRoot",
    "goRootSpan",
    "goRootCC",
    "goRootErr",
    "cRoot",
    "cRootSpan",
    "cRootCC",
    "cRootErr",
    "goErrors",
    "cErrors",
    "goMissing",
    "cMissing",
    "goStop",
    "runtime",
    "goRuntime",
)


def read_json(path: Path, default: Any) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return default


def resolve_recorded_path(recorded: str | Path, wringer_dir: Path, manifest: dict[str, Any]) -> Path:
    """Map container-recorded wringer paths back to the supplied host wringer dir."""
    path = Path(recorded)
    if path.exists():
        return path

    raw = str(recorded)
    host_wringer_dir = wringer_dir.resolve()
    prefixes: list[tuple[str, Path]] = []
    out_dir = str(manifest.get("out_dir") or "")
    baseline_run_dir = str(manifest.get("baseline_run_dir") or "")
    if out_dir:
        prefixes.append((out_dir, host_wringer_dir))
    if baseline_run_dir:
        prefixes.append((baseline_run_dir, host_wringer_dir / "baseline"))

    for prefix, host_prefix in prefixes:
        prefix = prefix.rstrip("/")
        if raw == prefix:
            return host_prefix
        if raw.startswith(prefix + "/"):
            return host_prefix / raw[len(prefix) + 1 :]

    baseline_marker = "/baseline/"
    if baseline_marker in raw:
        candidate = host_wringer_dir / "baseline" / raw.split(baseline_marker, 1)[1]
        if candidate.exists():
            return candidate

    marker = f"/{wringer_dir.name}/"
    if marker in raw:
        return host_wringer_dir / raw.split(marker, 1)[1]
    if raw.endswith(f"/{wringer_dir.name}"):
        return host_wringer_dir
    return path


def host_field(recorded: str | Path, host_path: Path) -> str | None:
    recorded_text = str(recorded)
    host_text = str(host_path)
    if recorded_text == host_text:
        return None
    return host_text


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    if not path.exists():
        return rows
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            continue
    return rows


def iter_jsonl(path: Path):
    if not path.exists():
        return
    with path.open("r", encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                parsed = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(parsed, dict):
                yield parsed


def compact_frame_row(row: dict[str, Any]) -> dict[str, Any]:
    keep = {
        "base",
        "bytes",
        "corpus_kind",
        "duration_ms",
        "elapsed_ms",
        "file",
        "grammar",
        "lifecycle",
        "mode",
        "path",
        "phase",
        "raw_log",
        "result",
        "round",
        "runtime",
        "source_sha256",
        "source_size",
        "stage",
        "telemetry",
        "ts",
    }
    out = {key: row.get(key) for key in keep if key in row}
    if row.get("telemetry") == "parser":
        out["parser_progress"] = compact_parser_progress_from_frame(row)
    return out


def read_active(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    text = path.read_text(encoding="utf-8", errors="replace").strip()
    if not text:
        return {}
    try:
        parsed = json.loads(text)
        if isinstance(parsed, dict):
            return parsed
    except json.JSONDecodeError:
        pass
    fields = text.split("\t", 3)
    if len(fields) >= 4:
        return {
            "ts": fields[0],
            "grammar": fields[1],
            "corpus_kind": fields[2],
            "detail": fields[3],
        }
    return {"raw": text}


def parse_kv(text: str) -> dict[str, str]:
    try:
        parts = shlex.split(text)
    except ValueError:
        parts = text.split()
    out: dict[str, str] = {}
    for part in parts:
        if "=" in part:
            key, value = part.split("=", 1)
            out[key] = value
    return out


def parse_measure_line(line: str) -> dict[str, str]:
    if not line.startswith(DTIER_PREFIX):
        return {}
    parts = line[len(DTIER_PREFIX) :].strip().split(None, 1)
    if len(parts) < 2:
        return {}
    out = parse_kv(parts[1])
    out["grammar"] = parts[0]
    out["raw"] = line
    return out


def parse_runtime_hints(runtime: str | None) -> dict[str, str]:
    if not runtime:
        return {}
    return parse_kv(runtime)


def parse_int(value: str | None) -> int | None:
    if value is None:
        return None
    value = value.split("/", 1)[0]
    try:
        return int(value)
    except ValueError:
        return None


def comparison_diag_row(item: dict[str, str], log_path: Path | None = None) -> dict[str, Any]:
    runtime = item.get("runtime", "")
    go_runtime = item.get("goRuntime", "")
    row = {
        "file": item.get("file", ""),
        "base": item.get("base", ""),
        "path": item.get("path", ""),
        "phase": item.get("phase", ""),
        "result": item.get("result", ""),
        "elapsedMs": item.get("elapsed_ms", ""),
        "diff": item.get("diff", ""),
        "firstDiffPath": item.get("firstDiffPath", item.get("path", "")),
        "goType": item.get("goType", ""),
        "cType": item.get("cType", ""),
        "goSpan": item.get("goSpan", ""),
        "cSpan": item.get("cSpan", ""),
        "goChildCount": parse_int(item.get("goCC")),
        "cChildCount": parse_int(item.get("cCC")),
        "goRoot": item.get("goRoot", ""),
        "goRootSpan": item.get("goRootSpan", ""),
        "goRootChildCount": parse_int(item.get("goRootCC")),
        "goRootErr": item.get("goRootErr", ""),
        "cRoot": item.get("cRoot", ""),
        "cRootSpan": item.get("cRootSpan", ""),
        "cRootChildCount": parse_int(item.get("cRootCC")),
        "cRootErr": item.get("cRootErr", ""),
        "goErrorCount": parse_int(item.get("goErrors")),
        "cErrorCount": parse_int(item.get("cErrors")),
        "goMissingCount": parse_int(item.get("goMissing")),
        "cMissingCount": parse_int(item.get("cMissing")),
        "goStop": item.get("goStop", ""),
        "runtime": runtime,
        "goRuntime": go_runtime,
        "runtime_hints": parse_runtime_hints(runtime or go_runtime),
        "raw": item.get("raw", ""),
    }
    if log_path is not None:
        row["log"] = str(log_path)
    return row


def comparison_diag_evidence(progress: list[dict[str, str]], log_path: Path | None = None) -> dict[str, Any]:
    rows = [
        comparison_diag_row(item, log_path)
        for item in progress
        if item.get("phase") == COMPARISON_DIAG_PHASE
    ]
    return {
        "count": len(rows),
        "diffs": sorted({row["diff"] for row in rows if row.get("diff")}),
        "rootPairs": sorted(
            {
                f"{row.get('goRoot', '')}->{row.get('cRoot', '')}"
                for row in rows
                if row.get("goRoot") or row.get("cRoot")
            }
        ),
        "rows": rows[:5],
    }


def variant_env_assignments(variant: str) -> list[str]:
    return {
        "stack2": ["GOT_GLR_MAX_STACKS=2"],
        "stack8": ["GOT_GLR_MAX_STACKS=8"],
        "stack48": ["GOT_GLR_MAX_STACKS=48"],
        "node3": ["GOT_PARSE_NODE_LIMIT_SCALE=3"],
        "forest_off": ["GOT_GLR_FOREST=0"],
        "forest": ["REPRO_FOREST=1"],
        "merge1": ["GOT_GLR_MAX_MERGE_PER_KEY=1"],
        "merge24": ["GOT_GLR_MAX_MERGE_PER_KEY=24"],
        "faithful": ["GOT_FAITHFUL_CONDENSE=1"],
        "pre_mat": ["GOT_GLR_V2_PRE_MATERIALIZATION_DIAG=1"],
        "mat_off": [
            "GOT_GLR_V2_PENDING_PARENTS=0",
            "GOT_GLR_V2_FINAL_CHILD_REFS=0",
            "GOT_GLR_V2_COMPACT_FULL_LEAVES=0",
        ],
        "crecovery_all": ["GOT_C_RECOVERY=all"],
        "crecovery_off": ["GOT_C_RECOVERY=0"],
    }.get(variant, [])


def parity_counts(value: str | None) -> tuple[int, int] | None:
    if not value:
        return None
    match = re.match(r"^(\d+)/(\d+)", value)
    if not match:
        return None
    return int(match.group(1)), int(match.group(2))


def log_evidence(log_path: Path) -> dict[str, Any]:
    try:
        stat = log_path.stat()
    except OSError:
        return {
            "measure_lines": [],
            "progress_items": [],
            "parser_progress_summary": {"count": 0},
        }
    return _log_evidence_cached(str(log_path), stat.st_mtime_ns, stat.st_size)


@lru_cache(maxsize=256)
def _log_evidence_cached(log_path_text: str, _mtime_ns: int, _size: int) -> dict[str, Any]:
    log_path = Path(log_path_text)
    measure_rows: list[str] = []
    progress: list[dict[str, str]] = []
    parser_count = 0
    last_parser_raw = ""
    with log_path.open("r", encoding="utf-8", errors="replace") as f:
        for line in f:
            if line.startswith(DTIER_PREFIX):
                measure_rows.append(line.strip())
            elif line.startswith(PROGRESS_PREFIX):
                item = parse_kv(line[len(PROGRESS_PREFIX) :])
                item["raw"] = line.strip()
                progress.append(item)
            elif line.startswith(PARSER_PROGRESS_PREFIX):
                parser_count += 1
                last_parser_raw = line.strip()
    parser_summary: dict[str, Any] = {"count": parser_count}
    if parser_count:
        last_parser = parse_kv(last_parser_raw[len(PARSER_PROGRESS_PREFIX) :])
        last_parser["raw"] = last_parser_raw
        parser_summary = {
            "count": parser_count,
            "last": compact_parser_progress(last_parser),
            "last_raw": last_parser.get("raw", ""),
        }
    return {
        "measure_lines": measure_rows,
        "progress_items": progress,
        "parser_progress_summary": parser_summary,
    }


def measure_lines(log_path: Path) -> list[str]:
    if not log_path.exists():
        return []
    return list(log_evidence(log_path).get("measure_lines", []))


def progress_items(log_path: Path) -> list[dict[str, str]]:
    if not log_path.exists():
        return []
    return list(log_evidence(log_path).get("progress_items", []))


def parser_progress_items(log_path: Path) -> list[dict[str, str]]:
    if not log_path.exists():
        return []
    items: list[dict[str, str]] = []
    with log_path.open("r", encoding="utf-8", errors="replace") as f:
        for line in f:
            if line.startswith(PARSER_PROGRESS_PREFIX):
                item = parse_kv(line[len(PARSER_PROGRESS_PREFIX) :])
                item["raw"] = line.strip()
                items.append(item)
    return items


def compact_parser_progress(item: dict[str, str] | None) -> dict[str, Any]:
    if not item:
        return {}
    fields = {
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
    out: dict[str, Any] = {}
    for source, target in fields.items():
        if source not in item:
            continue
        value: Any = item[source]
        if isinstance(value, str) and value.lstrip("-").isdigit():
            value = int(value)
        out[target] = value
    return out


def compact_parser_progress_from_frame(row: dict[str, Any] | None) -> dict[str, Any]:
    if not row:
        return {}
    raw = row.get("parser_progress")
    if isinstance(raw, dict):
        return compact_parser_progress({str(k): str(v) for k, v in raw.items() if v is not None})
    synthetic: dict[str, str] = {}
    for key, value in row.items():
        if key.startswith("parser_") and value is not None:
            synthetic[key.removeprefix("parser_")] = str(value)
    return compact_parser_progress(synthetic)


def parser_progress_summary(log_path: Path) -> dict[str, Any]:
    if not log_path.exists():
        return {"count": 0}
    summary = log_evidence(log_path).get("parser_progress_summary", {"count": 0})
    return dict(summary) if isinstance(summary, dict) else {"count": 0}


def baseline_log_path(baseline_dir: Path, grammar: str, corpus_kind: str, index: int) -> Path:
    return baseline_dir / f"measure-{grammar}-{corpus_kind}-frame-{index:04d}.log"


def load_selected_files(wringer_dir: Path) -> list[dict[str, Any]]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir_recorded = Path(manifest.get("baseline_run_dir", wringer_dir / "baseline"))
    baseline_dir = resolve_recorded_path(baseline_dir_recorded, wringer_dir, manifest)
    aggregate = read_json(baseline_dir / "manifest.json", {})
    selected: list[dict[str, Any]] = []
    for entry in aggregate.get("grammars", []):
        manifest_path = Path(entry.get("manifest", ""))
        host_manifest_path = resolve_recorded_path(manifest_path, wringer_dir, manifest)
        if not host_manifest_path.exists():
            continue
        per_grammar = read_json(host_manifest_path, {})
        corpus_kind = per_grammar.get("corpus_kind", entry.get("corpus_kind", ""))
        corpus_root = per_grammar.get("corpus_root", entry.get("corpus_root", ""))
        grammar = per_grammar.get("grammar", entry.get("grammar", manifest.get("grammar", "")))
        files = per_grammar.get("files", [])
        selected_total = int(per_grammar.get("selected_total") or len(files))
        for selection_ordinal, item in enumerate(files, start=1):
            index = int(item.get("index", len(selected) + 1))
            recorded_baseline_log = baseline_log_path(baseline_dir_recorded, grammar, corpus_kind, index)
            host_baseline_log = baseline_log_path(baseline_dir, grammar, corpus_kind, index)
            row = {
                "ordinal": index,
                "selection_ordinal": selection_ordinal,
                "index": index,
                "total": item.get("total", selected_total),
                "selected_total": selected_total,
                "grammar": grammar,
                "corpus_kind": corpus_kind,
                "corpus_root": corpus_root,
                "path": item.get("path", ""),
                "base": item.get("base", Path(item.get("path", "")).name),
                "size": item.get("size"),
                "sha256": item.get("sha256", ""),
                "baseline_replay_command": item.get("replay_command", ""),
                "baseline_replay_env": item.get("replay_env", {}),
                "manifest": str(manifest_path),
                "baseline_log": str(recorded_baseline_log),
            }
            host_manifest = host_field(manifest_path, host_manifest_path)
            host_log = host_field(recorded_baseline_log, host_baseline_log)
            if host_manifest:
                row["host_manifest"] = host_manifest
            if host_log:
                row["host_baseline_log"] = host_log
            selected.append(row)
    return selected


def frame_events_by_path(baseline_dir: Path) -> dict[str, list[dict[str, Any]]]:
    by_path: dict[str, list[dict[str, Any]]] = {}
    parser_counts: dict[str, int] = {}
    for frame in iter_jsonl(baseline_dir / "frames.jsonl") or ():
        path = frame.get("path") or ""
        if not path:
            continue
        phase = str(frame.get("phase") or "")
        lifecycle = str(frame.get("lifecycle") or "")
        result = str(frame.get("result") or "")
        telemetry = str(frame.get("telemetry") or "")
        bucket = by_path.setdefault(str(path), [])
        if telemetry == "parser":
            parser_counts[str(path)] = parser_counts.get(str(path), 0) + 1
            compact = compact_frame_row(frame)
            compact["parser_progress_count"] = parser_counts[str(path)]
            # Keep only the last parser-progress row per frame. Dense parser
            # telemetry can be hundreds of thousands of rows and is not needed
            # for closed-world control or residual classification.
            replaced = False
            for i in range(len(bucket) - 1, -1, -1):
                if bucket[i].get("telemetry") == "parser":
                    bucket[i] = compact
                    replaced = True
                    break
            if not replaced:
                bucket.append(compact)
            continue
        keep = (
            lifecycle in {"timeout", "fail", "panic"}
            or phase in {"selected_file", "comparison_result", "go_parse_status"}
            or phase.endswith("_panic")
            or (phase and result and result not in {"", "match", "accepted"})
        )
        if keep:
            bucket.append(compact_frame_row(frame))
    status_path = baseline_dir / "status.tsv"
    if status_path.exists():
        for line in status_path.read_text(encoding="utf-8", errors="replace").splitlines():
            fields = line.split("\t", 4)
            if len(fields) < 5:
                continue
            ts, event, grammar, corpus_kind, detail = fields
            if event not in {"END", "FAIL", "TIMEOUT"}:
                continue
            kv = parse_kv(detail)
            path = kv.get("file", "")
            if not path:
                continue
            frame_text = kv.get("frame", "")
            row: dict[str, Any] = {
                "source": "baseline_status",
                "ts": ts,
                "event": event,
                "grammar": grammar,
                "corpus_kind": corpus_kind,
                "path": path,
                "base": Path(path).name,
                "phase": "frame_terminal",
                "lifecycle": "timeout" if event == "TIMEOUT" else ("fail" if event == "FAIL" else "ended"),
                "result": event.lower(),
                "raw_log": kv.get("log", ""),
                "runtime": "",
            }
            ordinal = parse_int(frame_text)
            if ordinal is not None:
                row["ordinal"] = ordinal
            if "sha256" in kv:
                row["source_sha256"] = kv["sha256"]
            if "size" in kv:
                try:
                    row["source_size"] = int(kv["size"])
                except ValueError:
                    row["source_size"] = kv["size"]
            if "rc" in kv:
                try:
                    row["rc"] = int(kv["rc"])
                except ValueError:
                    row["rc"] = kv["rc"]
            by_path.setdefault(path, []).append(row)
    return by_path


def status_success_logs(baseline_dir: Path, wringer_dir: Path, manifest: dict[str, Any]) -> set[str]:
    success: set[str] = set()
    status = baseline_dir / "status.tsv"
    if not status.exists():
        return success
    for line in status.read_text(encoding="utf-8", errors="replace").splitlines():
        fields = line.split("\t", 4)
        if len(fields) < 5 or fields[1] != "END":
            continue
        kv = parse_kv(fields[4])
        log = kv.get("log")
        if log:
            success.add(log)
            success.add(str(resolve_recorded_path(log, wringer_dir, manifest)))
    return success


def parse_timestamp_epoch(ts: str) -> float | None:
    try:
        return datetime.strptime(ts, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc).timestamp()
    except ValueError:
        return None


def baseline_status_rows(wringer_dir: Path) -> list[dict[str, Any]]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    status_path = baseline_dir / "status.tsv"
    if not status_path.exists():
        return []
    rows: list[dict[str, Any]] = []
    for line in status_path.read_text(encoding="utf-8", errors="replace").splitlines():
        fields = line.split("\t", 4)
        if len(fields) < 5:
            continue
        ts, event, grammar, corpus_kind, detail = fields
        kv = parse_kv(detail)
        frame_text = kv.get("frame", "")
        ordinal = parse_int(frame_text)
        row: dict[str, Any] = {
            "ts": ts,
            "event": event,
            "grammar": grammar,
            "corpus_kind": corpus_kind,
            "detail": detail,
            "detail_kv": kv,
        }
        epoch = parse_timestamp_epoch(ts)
        if epoch is not None:
            row["epoch_s"] = epoch
        if ordinal is not None:
            row["ordinal"] = ordinal
            if "/" in frame_text:
                total = parse_int(frame_text.split("/", 1)[1])
                if total is not None:
                    row["total"] = total
        for name in (
            "elapsed_s",
            "duration_s",
            "log_bytes",
            "progress_age_s",
            "parser_elapsed_ms",
            "parser_iter",
            "parser_tokens",
            "parser_stacks",
            "parser_live_stacks",
            "parser_max_stacks",
            "parser_node_count",
            "parser_token_start",
            "parser_token_end",
        ):
            if name in kv:
                try:
                    row[name] = float(kv[name]) if "." in kv[name] else int(kv[name])
                except ValueError:
                    row[name] = kv[name]
        for name in ("worker_pid", "supervisor_pid", "rc"):
            if name in kv:
                try:
                    row[name] = int(kv[name])
                except ValueError:
                    row[name] = kv[name]
        for name in ("file", "size", "sha256", "log", "parser_phase"):
            if name in kv:
                row[name] = kv[name]
        rows.append(row)
    return rows


def compact_status_frame_events(
    wringer_dir: Path,
    selected: list[dict[str, Any]],
    status_rows: list[dict[str, Any]] | None = None,
) -> dict[str, list[dict[str, Any]]] | None:
    """Build bounded per-frame evidence from manifest selections and status.tsv.

    Dense baseline/frames.jsonl can contain one row per parser progress sample.
    When status.tsv has terminal coverage for every selected isolated frame, it
    already carries the closed-world lifecycle evidence needed by summary and
    controls, plus compact last-parser fields from heartbeat/timeout rows.
    """
    if status_rows is None:
        status_rows = baseline_status_rows(wringer_dir)
    if not selected or not status_rows:
        return None

    grouped: dict[int, list[dict[str, Any]]] = {}
    for row in status_rows:
        try:
            ordinal = int(row.get("ordinal") or 0)
        except (TypeError, ValueError):
            continue
        if ordinal <= 0:
            continue
        grouped.setdefault(ordinal, []).append(row)

    selected_ordinals: set[int] = set()
    for item in selected:
        try:
            ordinal = int(item.get("ordinal") or item.get("index") or 0)
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal > 0:
            selected_ordinals.add(ordinal)
    if not selected_ordinals:
        return None

    terminal_events = {"END", "FAIL", "TIMEOUT"}
    terminal_by_ordinal: dict[int, dict[str, Any]] = {}
    for ordinal in selected_ordinals:
        for row in grouped.get(ordinal, []):
            if str(row.get("event") or "") in terminal_events:
                terminal_by_ordinal[ordinal] = row
        if ordinal not in terminal_by_ordinal:
            return None

    events_by_path: dict[str, list[dict[str, Any]]] = {}
    for item in selected:
        try:
            ordinal = int(item.get("ordinal") or item.get("index") or 0)
        except (TypeError, ValueError):
            ordinal = 0
        path = str(item.get("path") or "")
        if ordinal <= 0 or not path:
            continue

        rows = grouped.get(ordinal, [])
        last_parser: dict[str, Any] = {}
        parser_count = 0
        last_heartbeat: dict[str, Any] | None = None
        for row in rows:
            if str(row.get("event") or "") == "HEARTBEAT":
                last_heartbeat = row
                parser_count += 1
            parser = compact_parser_progress(
                {k.removeprefix("parser_"): str(v) for k, v in row.items() if k.startswith("parser_")}
            )
            if parser:
                last_parser = parser

        compact: list[dict[str, Any]] = [
            {
                "source": "baseline_status",
                "phase": "selected_file",
                "grammar": item.get("grammar"),
                "corpus_kind": item.get("corpus_kind"),
                "ordinal": ordinal,
                "selection_ordinal": item.get("selection_ordinal"),
                "total": item.get("total"),
                "selected_total": item.get("selected_total"),
                "path": path,
                "base": item.get("base"),
                "source_sha256": item.get("sha256"),
                "source_size": item.get("size"),
            }
        ]
        if last_parser:
            progress_row: dict[str, Any] = {
                "source": "baseline_status",
                "phase": "parser_progress",
                "telemetry": "parser",
                "grammar": item.get("grammar"),
                "corpus_kind": item.get("corpus_kind"),
                "ordinal": ordinal,
                "path": path,
                "base": item.get("base"),
                "parser_progress": last_parser,
                "parser_progress_count": parser_count,
            }
            if last_heartbeat:
                progress_row["ts"] = last_heartbeat.get("ts")
                progress_row["raw_log"] = last_heartbeat.get("log", "")
            compact.append(progress_row)

        terminal = terminal_by_ordinal[ordinal]
        event_name = str(terminal.get("event") or "")
        terminal_path = str(terminal.get("file") or path)
        terminal_row: dict[str, Any] = {
            "source": "baseline_status",
            "ts": terminal.get("ts"),
            "event": event_name,
            "grammar": terminal.get("grammar", item.get("grammar")),
            "corpus_kind": terminal.get("corpus_kind", item.get("corpus_kind")),
            "path": terminal_path,
            "base": Path(terminal_path).name,
            "ordinal": ordinal,
            "phase": "frame_terminal",
            "lifecycle": "timeout" if event_name == "TIMEOUT" else ("fail" if event_name == "FAIL" else "ended"),
            "result": event_name.lower(),
            "raw_log": terminal.get("log", ""),
            "source_sha256": terminal.get("sha256", item.get("sha256", "")),
            "source_size": terminal.get("size", item.get("size")),
        }
        if "rc" in terminal:
            terminal_row["rc"] = terminal.get("rc")
        compact.append(terminal_row)
        events_by_path.setdefault(path, []).extend(compact)
    return events_by_path


def baseline_frame_telemetry_by_ordinal(wringer_dir: Path) -> dict[int, dict[str, Any]]:
    terminal_events = {"END", "FAIL", "TIMEOUT"}
    grouped: dict[int, dict[str, Any]] = {}
    for row in baseline_status_rows(wringer_dir):
        try:
            ordinal = int(row.get("ordinal") or 0)
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal <= 0:
            continue
        bucket = grouped.setdefault(
            ordinal,
            {
                "event_scope": "baseline_frame",
                "event_lifecycle": [],
                "heartbeat_count": 0,
            },
        )
        event_name = str(row.get("event") or "")
        if event_name:
            bucket["event_lifecycle"].append(event_name)
        detail_kv = row.get("detail_kv") if isinstance(row.get("detail_kv"), dict) else {}
        for name in ("file", "size", "sha256", "log", "total"):
            if row.get(name) not in {None, ""}:
                bucket[name] = row.get(name)
        if event_name == "START":
            bucket["start_event"] = row
            if isinstance(row.get("epoch_s"), (int, float)):
                bucket["started_at_epoch_s"] = float(row["epoch_s"])
                bucket["started_at"] = iso_from_epoch(float(row["epoch_s"]))
        elif event_name in terminal_events:
            bucket["terminal_event"] = row
            bucket["terminal_event_type"] = event_name
            if isinstance(row.get("epoch_s"), (int, float)):
                bucket["finished_at_epoch_s"] = float(row["epoch_s"])
                bucket["finished_at"] = iso_from_epoch(float(row["epoch_s"]))
            if "rc" in row:
                bucket["rc"] = row.get("rc")
            if event_name == "TIMEOUT":
                bucket["timeout"] = True
            parser = compact_parser_progress({k.removeprefix("parser_"): str(v) for k, v in row.items() if k.startswith("parser_")})
            if parser:
                bucket["last_parser_progress"] = parser
                bucket["parser_progress"] = {"count": int(bucket.get("heartbeat_count") or 0), "last": parser}
        elif event_name == "HEARTBEAT":
            bucket["heartbeat_count"] = int(bucket.get("heartbeat_count") or 0) + 1
            bucket["last_heartbeat"] = row
            if isinstance(row.get("epoch_s"), (int, float)):
                bucket["last_heartbeat_at_epoch_s"] = float(row["epoch_s"])
                bucket["last_heartbeat_at"] = iso_from_epoch(float(row["epoch_s"]))
            for name in ("elapsed_s", "worker_pid", "log_bytes", "progress_age_s"):
                if name in row:
                    bucket[f"last_heartbeat_{name}"] = row.get(name)
            parser = compact_parser_progress({k.removeprefix("parser_"): str(v) for k, v in row.items() if k.startswith("parser_")})
            if parser:
                bucket["last_parser_progress"] = parser
                bucket["parser_progress"] = {"count": int(bucket.get("heartbeat_count") or 0), "last": parser}
        if "worker_pid" in row:
            bucket["worker_pid"] = row.get("worker_pid")
            bucket["child_pid"] = row.get("worker_pid")
        elif "worker_pid" in detail_kv:
            bucket["worker_pid"] = detail_kv.get("worker_pid")
            bucket["child_pid"] = detail_kv.get("worker_pid")
    for bucket in grouped.values():
        start = bucket.get("started_at_epoch_s")
        finish = bucket.get("finished_at_epoch_s")
        if "duration_s" not in bucket and isinstance(start, (int, float)) and isinstance(finish, (int, float)):
            bucket["duration_s"] = max(0.0, float(finish) - float(start))
        if "duration_s" not in bucket and isinstance(bucket.get("last_heartbeat_elapsed_s"), (int, float)):
            bucket["elapsed_s"] = bucket.get("last_heartbeat_elapsed_s")
        if "duration_s" in bucket:
            bucket["elapsed_s"] = bucket["duration_s"]
        bucket["has_start_event"] = "start_event" in bucket
        bucket["has_terminal_event"] = "terminal_event" in bucket
        bucket["has_duration"] = "duration_s" in bucket or "elapsed_s" in bucket
        bucket["event_lifecycle"] = list(bucket.get("event_lifecycle") or [])
    return grouped


def baseline_frame_lifecycle_balance(wringer_dir: Path) -> dict[str, Any]:
    terminal = {"END", "FAIL", "TIMEOUT"}
    open_frames: dict[int, dict[str, Any]] = {}
    starts = 0
    terminals = 0
    heartbeats = 0
    for row in baseline_status_rows(wringer_dir):
        try:
            ordinal = int(row.get("ordinal") or 0)
        except (TypeError, ValueError):
            ordinal = 0
        if ordinal <= 0:
            continue
        event_name = str(row.get("event") or "")
        if event_name == "START":
            starts += 1
            open_frames[ordinal] = row
        elif event_name in terminal:
            terminals += 1
            open_frames.pop(ordinal, None)
        elif event_name == "HEARTBEAT":
            heartbeats += 1
            if ordinal in open_frames:
                open_frames[ordinal]["last_heartbeat"] = row
    return {
        "starts": starts,
        "terminals": terminals,
        "heartbeats": heartbeats,
        "open": len(open_frames),
        "open_frames": list(open_frames.values()),
    }


def suspicious_reasons(selected: dict[str, Any], frames: list[dict[str, Any]], success_logs: set[str]) -> list[str]:
    reasons: list[str] = []
    for frame in frames:
        lifecycle = str(frame.get("lifecycle", ""))
        phase = str(frame.get("phase", ""))
        result = str(frame.get("result", ""))
        if lifecycle in {"timeout", "fail"}:
            reasons.append(f"terminal_{lifecycle}")
        if lifecycle == "panic" or phase.endswith("_panic"):
            reasons.append("panic")
        if phase == "comparison_result" and result != "match":
            reasons.append(f"comparison_result_{result or 'missing'}")
        if phase == "go_parse_status" and result != "accepted":
            reasons.append(f"go_parse_status_{result or 'missing'}")
    log_path = Path(str(selected.get("host_baseline_log") or selected.get("baseline_log", "")))
    measures = measure_lines(log_path)
    if not measures:
        reasons.append("missing_measure_dtier")
    else:
        parsed = parse_measure_line(measures[-1])
        counts = parity_counts(parsed.get("parityMatch"))
        if counts != (1, 1):
            reasons.append("measure_dtier_nonmatch")
    if str(selected.get("baseline_log", log_path)) not in success_logs:
        reasons.append("no_successful_measure_frame")
    return sorted(set(reasons))


def load_baseline(wringer_dir: Path) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    return _load_baseline_cached(str(wringer_dir.resolve()))


@lru_cache(maxsize=16)
def _load_baseline_cached(wringer_dir_text: str) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    wringer_dir = Path(wringer_dir_text)
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    selected = load_selected_files(wringer_dir)
    status_rows = baseline_status_rows(wringer_dir)
    frames_by_path = compact_status_frame_events(wringer_dir, selected, status_rows)
    if frames_by_path is None:
        frames_by_path = frame_events_by_path(baseline_dir)
    success_logs = status_success_logs(baseline_dir, wringer_dir, manifest)
    suspicious: list[dict[str, Any]] = []
    for item in selected:
        item_frames = frames_by_path.get(item["path"], [])
        item["_baseline_frame_events"] = item_frames
        reasons = suspicious_reasons(item, item_frames, success_logs)
        item["suspicious_reasons"] = reasons
        if reasons:
            suspicious.append({**item, "reasons": reasons})
    return selected, suspicious


def baseline_terminal_state(selected: dict[str, Any], frames: list[dict[str, Any]]) -> dict[str, Any]:
    comparison = ""
    go_parse_status = ""
    last_phase = ""
    last_lifecycle = ""
    timeout = False
    fail = False
    panic = False
    runtime = ""
    stop_hints: dict[str, str] = {}
    terminal_event: dict[str, Any] = {}
    compact_parser_count = 0
    compact_last_parser: dict[str, Any] = {}
    for frame in frames:
        phase = str(frame.get("phase", ""))
        lifecycle = str(frame.get("lifecycle", ""))
        result = str(frame.get("result", ""))
        if phase:
            last_phase = phase
        if lifecycle:
            last_lifecycle = lifecycle
        if frame.get("telemetry") == "parser":
            try:
                compact_parser_count = max(compact_parser_count, int(frame.get("parser_progress_count") or 0))
            except (TypeError, ValueError):
                compact_parser_count = max(compact_parser_count, 1)
            if isinstance(frame.get("parser_progress"), dict):
                compact_last_parser = frame.get("parser_progress") or {}
        if phase == "comparison_result":
            comparison = result
            runtime = str(frame.get("runtime") or "")
            stop_hints = parse_runtime_hints(runtime)
        elif phase == "go_parse_status":
            go_parse_status = result
            if not runtime:
                runtime = str(frame.get("runtime") or "")
                stop_hints = parse_runtime_hints(runtime)
        if lifecycle == "timeout":
            timeout = True
            terminal_event = frame
        elif lifecycle == "fail":
            fail = True
            terminal_event = frame
        elif lifecycle == "panic" or phase.endswith("_panic"):
            panic = True
            terminal_event = frame
    log_path = Path(str(selected.get("host_baseline_log") or selected.get("baseline_log", "")))
    measures = measure_lines(log_path)
    measure = parse_measure_line(measures[-1]) if measures else {}
    progress = progress_items(log_path)
    parser_progress = parser_progress_summary(log_path)
    if parser_progress.get("count", 0) == 0 and (compact_parser_count or compact_last_parser):
        parser_progress = {"count": compact_parser_count, "last": compact_last_parser}
    comparison_diag = comparison_diag_evidence(progress, log_path)
    if not go_parse_status:
        go_parse_status = str(measure.get("goStatus") or measure.get("goParseStatus") or "")
    if not comparison:
        counts = parity_counts(measure.get("parityMatch"))
        if counts == (1, 1):
            comparison = "match"
        elif counts is not None:
            comparison = "diverge"
    if not runtime:
        runtime = str(measure.get("runtime") or "")
        stop_hints = parse_runtime_hints(runtime)
    out: dict[str, Any] = {
        "comparison_result": comparison,
        "go_parse_status": go_parse_status,
        "last_phase": last_phase,
        "last_lifecycle": last_lifecycle,
        "timeout": timeout,
        "fail": fail,
        "panic": panic,
        "measure": measure,
        "measure_line": measures[-1] if measures else "",
        "runtime": runtime,
        "runtime_hints": stop_hints,
        "comparison_diagnostic": comparison_diag,
        "parser_progress": parser_progress,
    }
    if parser_progress.get("last"):
        out["last_parser_progress"] = parser_progress["last"]
    if terminal_event:
        out["terminal_event"] = terminal_event
    return out


def public_frame_record(item: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in item.items() if not str(key).startswith("_")}


def replay_plan(wringer_dir: Path, item: dict[str, Any]) -> dict[str, Any]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    config = manifest.get("config", {})
    grammar = str(item.get("grammar") or manifest.get("grammar") or "")
    ordinal = int(item.get("ordinal") or 0)
    out_dir = str(manifest.get("out_dir") or wringer_dir)
    corpus_dir = str(item.get("corpus_root") or "")
    path = str(item.get("path") or "")
    variants = config.get("variants") or []
    if isinstance(variants, str):
        variants = variants.split()
    parse_progress_env: list[str] = []
    if config.get("parse_progress") is True:
        parse_progress_env.append("GOT_PARSE_PROGRESS=1")
        interval = str(config.get("parse_progress_interval_ms") or "")
        if interval:
            parse_progress_env.append(f"GOT_PARSE_PROGRESS_INTERVAL_MS={interval}")
    wringer_parse_progress_env: list[str] = []
    if config.get("parse_progress") is True:
        wringer_parse_progress_env.append("GTS_WRINGER_PARSE_PROGRESS=1")
        interval = str(config.get("parse_progress_interval_ms") or "")
        if interval:
            wringer_parse_progress_env.append(f"GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS={interval}")
    variant_filter = str(ordinal) if ordinal else ""
    firstdiff_filter = str(ordinal) if ordinal else ""
    variant_env = [
        "GTS_WRINGER_REUSE_BASELINE=1",
        "GTS_WRINGER_STAGES=variants,summary",
        f"GTS_WRINGER_VARIANT_FRAMES={variant_filter}",
    ]
    firstdiff_env = [
        "GTS_WRINGER_REUSE_BASELINE=1",
        "GTS_WRINGER_STAGES=firstdiff,summary",
        f"GTS_WRINGER_FIRSTDIFF_FRAMES={firstdiff_filter}",
        "GTS_WRINGER_MAX_DIAG_FILES=1",
    ]
    direct_firstdiff_env = [
        "CGO_ENABLED=1",
        f"REPRO_LANG={grammar}",
        f"REPRO_FILE={path}",
    ]
    if config.get("glr_trace") is True:
        firstdiff_env.append("GTS_WRINGER_GLR_TRACE=1")
        direct_firstdiff_env.append("REPRO_GLR_TRACE=1")
    if config.get("debug_dfa") is True:
        firstdiff_env.append("GTS_WRINGER_DEBUG_DFA=1")
        direct_firstdiff_env.append("REPRO_DEBUG_DFA=1")
    direct_firstdiff_env.extend(parse_progress_env)
    variant_env.extend(wringer_parse_progress_env)
    firstdiff_env.extend(wringer_parse_progress_env)
    if variants:
        variant_env.append("GTS_WRINGER_VARIANTS=" + " ".join(str(v) for v in variants))
    if corpus_dir:
        variant_env.append(f"GTS_CORPUS_DIR={corpus_dir}")
        firstdiff_env.append(f"GTS_CORPUS_DIR={corpus_dir}")
    script = "cgo_harness/docker/run_grammar_integrity_wringer.sh"
    baseline_bin = str(Path(str(manifest.get("baseline_run_dir", wringer_dir / "baseline"))) / "measure.test")
    timeout_s = str(config.get("timeout") or 60)
    kill_after = str(config.get("kill_after") or "10s")
    diag_timeout = str(config.get("diag_timeout") or timeout_s)
    rounds = str(config.get("rounds") or 1)
    baseline_log = str(item.get("host_baseline_log") or item.get("baseline_log") or "")
    baseline_env = [
        "GTS_CORPUS_DIR=" + corpus_dir,
        "GTS_WRINGER_STAGES=baseline,summary",
        f"GTS_WRINGER_BASELINE_FRAMES={ordinal}",
    ] if corpus_dir and ordinal else []
    baseline_env.extend(wringer_parse_progress_env)
    direct_baseline = str(item.get("baseline_replay_command") or "")
    if not direct_baseline:
        env_args = [
            "CGO_ENABLED=1",
            f"REPRO_LANG={grammar}",
            f"REPRO_DIR={corpus_dir}",
            f"REPRO_FILE={path}",
            "REPRO_PROGRESS=1",
            "REPRO_N=1",
            f"REPRO_ROUNDS={rounds}",
        ]
        env_args.extend(parse_progress_env)
        direct_baseline = shlex.join(
            [
                "timeout",
                f"--kill-after={kill_after}",
                timeout_s,
                "env",
                *env_args,
                baseline_bin,
                "-test.run",
                "^TestMeasureDtierVsC$",
                "-test.count=1",
            ]
        )
    if baseline_log:
        direct_baseline = f"{direct_baseline} > {shlex.quote(baseline_log)} 2>&1"
    direct_variants: dict[str, str] = {}
    for variant in variants:
        variant_text = str(variant)
        env_args = [
            "CGO_ENABLED=1",
            f"REPRO_LANG={grammar}",
            f"REPRO_DIR={corpus_dir}",
            f"REPRO_FILE={path}",
            "REPRO_PROGRESS=1",
            "REPRO_SIGNATURES=1",
            "REPRO_N=1",
            f"REPRO_ROUNDS={rounds}",
        ]
        env_args.extend(parse_progress_env)
        env_args.extend(variant_env_assignments(variant_text))
        direct_variants[variant_text] = shlex.join(
            [
                "timeout",
                f"--kill-after={kill_after}",
                timeout_s,
                "env",
                *env_args,
                baseline_bin,
                "-test.run",
                "^TestMeasureDtierVsC$",
                "-test.count=1",
            ]
        )
    direct_firstdiff = shlex.join(
        [
            "timeout",
            f"--kill-after={kill_after}",
            diag_timeout,
            "env",
            *direct_firstdiff_env,
            baseline_bin,
            "-test.run",
            "^TestFirstDiffDiag$",
            "-test.count=1",
            "-test.v",
        ]
    )
    return {
        "ordinal": ordinal,
        "baseline_frame_filter": str(ordinal) if ordinal else "",
        "variant_frame_filter": variant_filter,
        "firstdiff_frame_filter": firstdiff_filter,
        "baseline_command": shlex.join(["env", *baseline_env, "bash", script, grammar, out_dir]) if baseline_env else "",
        "baseline_reuse_variant_command": shlex.join(["env", *variant_env, "bash", script, grammar, out_dir]),
        "baseline_reuse_firstdiff_command": shlex.join(["env", *firstdiff_env, "bash", script, grammar, out_dir]),
        "direct_baseline_command": direct_baseline,
        "direct_variant_commands": direct_variants,
        "direct_firstdiff_command": direct_firstdiff,
    }


def build_frame_catalog(wringer_dir: Path) -> list[dict[str, Any]]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    selected, suspicious = load_baseline(wringer_dir)
    suspicious_by_path = {item["path"]: item for item in suspicious}
    rows: list[dict[str, Any]] = []
    selected_total = len(selected)
    for selection_ordinal, item in enumerate(selected, start=1):
        path = str(item.get("path") or "")
        frames = item.get("_baseline_frame_events") if isinstance(item.get("_baseline_frame_events"), list) else []
        terminal = baseline_terminal_state(item, frames)
        reasons = list(item.get("suspicious_reasons", []))
        if path in suspicious_by_path:
            reasons = list(suspicious_by_path[path].get("reasons", reasons))
        ordinal = int(item.get("ordinal") or item.get("index") or selection_ordinal)
        row = {
            "ordinal": ordinal,
            "index": item.get("index", ordinal),
            "total": item.get("total", selected_total),
            "selection_ordinal": item.get("selection_ordinal", selection_ordinal),
            "selected_total": item.get("selected_total", selected_total),
            "grammar": item.get("grammar"),
            "corpus_kind": item.get("corpus_kind"),
            "corpus_root": item.get("corpus_root"),
            "path": path,
            "base": item.get("base"),
            "size": item.get("size"),
            "sha256": item.get("sha256"),
            "manifest": item.get("manifest"),
            "baseline_log": item.get("baseline_log"),
            "baseline_replay_command": item.get("baseline_replay_command", ""),
            "baseline_replay_env": item.get("baseline_replay_env", {}),
            "suspicious": bool(reasons),
            "reasons": reasons,
            "baseline_terminal": terminal,
            "replay_plan": replay_plan(wringer_dir, item),
        }
        for optional in ("host_manifest", "host_baseline_log"):
            if optional in item:
                row[optional] = item[optional]
        rows.append(row)
    return rows


def load_variant_outcomes(wringer_dir: Path) -> dict[str, dict[str, dict[str, Any]]]:
    outcomes: dict[str, dict[str, dict[str, Any]]] = {}
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    variants_dir = wringer_dir / "variants"
    if not variants_dir.exists():
        return outcomes
    for meta_path in sorted(variants_dir.glob("*/*.json")):
        meta = read_json(meta_path, {})
        if not meta.get("variant") or not meta.get("path"):
            continue
        log_path = resolve_recorded_path(Path(meta.get("log", "")), wringer_dir, manifest)
        measures = measure_lines(log_path)
        parsed = parse_measure_line(measures[-1]) if measures else {}
        counts = parity_counts(parsed.get("parityMatch"))
        progress = progress_items(log_path)
        parser_progress = parser_progress_summary(log_path)
        comparison_results = [
            item.get("result", "")
            for item in progress
            if item.get("phase") == "comparison_result"
        ]
        comparison_diag = comparison_diag_evidence(progress, log_path)
        outcome = {
            **meta,
            "measure": parsed,
            "measure_line": measures[-1] if measures else "",
            "parity_match": counts == (1, 1),
            "parity_counts": list(counts) if counts is not None else None,
            "comparison_results": comparison_results,
            "comparison_diagnostic": comparison_diag,
            "parser_progress": parser_progress,
        }
        if parser_progress.get("last"):
            outcome["last_parser_progress"] = parser_progress["last"]
        if int(meta.get("rc") or 0) == 0:
            malformed_reasons = []
            if not measures:
                malformed_reasons.append("missing_measure_dtier")
            if counts is None and not comparison_results:
                malformed_reasons.append("missing_comparison_result")
            if malformed_reasons:
                outcome["malformed"] = True
                outcome["malformed_reason"] = ",".join(malformed_reasons)
        host_log = host_field(meta.get("log", ""), log_path)
        if host_log:
            outcome["host_log"] = host_log
        outcomes.setdefault(str(meta["path"]), {})[str(meta["variant"])] = outcome
    return outcomes


def load_firstdiff(wringer_dir: Path) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    diag_dir = wringer_dir / "firstdiff"
    if not diag_dir.exists():
        return out
    for meta_path in sorted(diag_dir.glob("*.json")):
        meta = read_json(meta_path, {})
        if meta.get("path"):
            log_path = resolve_recorded_path(Path(meta.get("log", "")), wringer_dir, manifest)
            host_log = host_field(meta.get("log", ""), log_path)
            if host_log:
                meta["host_log"] = host_log
            out[str(meta["path"])] = meta
    return out


def load_wringer_events(wringer_dir: Path) -> list[dict[str, Any]]:
    return read_jsonl(wringer_dir / "wringer_events.jsonl")


def event_epoch(event: dict[str, Any]) -> float | None:
    value = event.get("epoch_s")
    try:
        if value is not None:
            return float(value)
    except (TypeError, ValueError):
        pass
    ts = str(event.get("ts") or "")
    if not ts:
        return None
    try:
        return datetime.strptime(ts, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc).timestamp()
    except ValueError:
        return None


def iso_from_epoch(value: float | None) -> str:
    if value is None:
        return ""
    return datetime.fromtimestamp(value, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def wringer_event_key(event: dict[str, Any]) -> tuple[str, str, int, str]:
    stage = str(event.get("stage") or "")
    mode = str(event.get("variant") or event.get("mode") or "")
    try:
        ordinal = int(event.get("ordinal") or 0)
    except (TypeError, ValueError):
        ordinal = 0
    path = str(event.get("path") or "")
    if stage == "baseline" and ordinal == 0:
        mode = "tier_scan"
        path = ""
    return (stage, mode, ordinal, path)


def event_telemetry_by_action(events: list[dict[str, Any]]) -> dict[tuple[str, str, int, str], dict[str, Any]]:
    terminal_events = {"END", "FAIL", "TIMEOUT"}
    grouped: dict[tuple[str, str, int, str], dict[str, Any]] = {}
    for event in events:
        key = wringer_event_key(event)
        bucket = grouped.setdefault(
            key,
            {
                "event_lifecycle": [],
                "heartbeat_count": 0,
            },
        )
        event_name = str(event.get("event") or "")
        if event_name:
            bucket["event_lifecycle"].append(event_name)
        if event_name == "START":
            bucket["start_event"] = event
            epoch = event_epoch(event)
            if epoch is not None:
                bucket["started_at_epoch_s"] = epoch
                bucket["started_at"] = iso_from_epoch(epoch)
        elif event_name in terminal_events:
            bucket["terminal_event"] = event
            bucket["terminal_event_type"] = event_name
            epoch = event_epoch(event)
            if epoch is not None:
                bucket["finished_at_epoch_s"] = epoch
                bucket["finished_at"] = iso_from_epoch(epoch)
            for field in ("duration_s", "elapsed_s"):
                if field in event:
                    try:
                        bucket["duration_s"] = float(event[field])
                    except (TypeError, ValueError):
                        pass
            if "rc" in event:
                bucket["rc"] = event.get("rc")
            if "timeout" in event:
                bucket["timeout"] = bool(event.get("timeout"))
        elif event_name == "HEARTBEAT":
            bucket["heartbeat_count"] = int(bucket.get("heartbeat_count") or 0) + 1
            bucket["last_heartbeat"] = event
            epoch = event_epoch(event)
            if epoch is not None:
                bucket["last_heartbeat_at_epoch_s"] = epoch
                bucket["last_heartbeat_at"] = iso_from_epoch(epoch)
            for field in ("elapsed_s", "child_pid", "log_bytes", "log_age_s", "progress_age_s"):
                if field in event:
                    bucket[f"last_heartbeat_{field}"] = event.get(field)
        if "child_pid" in event:
            bucket["child_pid"] = event.get("child_pid")
    for bucket in grouped.values():
        start = bucket.get("started_at_epoch_s")
        finish = bucket.get("finished_at_epoch_s")
        if "duration_s" not in bucket and isinstance(start, (int, float)) and isinstance(finish, (int, float)):
            bucket["duration_s"] = max(0.0, float(finish) - float(start))
        if "duration_s" in bucket:
            bucket["elapsed_s"] = bucket["duration_s"]
        bucket["has_start_event"] = "start_event" in bucket
        bucket["has_terminal_event"] = "terminal_event" in bucket
        bucket["has_duration"] = "duration_s" in bucket or "elapsed_s" in bucket
        bucket["event_lifecycle"] = list(bucket.get("event_lifecycle") or [])
    return grouped


def lifecycle_balance(events: list[dict[str, Any]]) -> dict[str, Any]:
    terminal = {"END", "FAIL", "TIMEOUT"}
    open_frames: dict[tuple[str, str, int, str], dict[str, Any]] = {}
    terminals = 0
    starts = 0
    for event in events:
        event_name = str(event.get("event") or "")
        key = (
            str(event.get("stage") or ""),
            str(event.get("variant") or event.get("mode") or ""),
            int(event.get("ordinal") or 0),
            str(event.get("path") or ""),
        )
        if event_name == "START":
            starts += 1
            open_frames[key] = event
        elif event_name in terminal:
            terminals += 1
            open_frames.pop(key, None)
    return {
        "starts": starts,
        "terminals": terminals,
        "open": len(open_frames),
        "open_frames": list(open_frames.values()),
    }


def event_counts(events: list[dict[str, Any]]) -> dict[str, Any]:
    by_event: dict[str, int] = {}
    by_stage_event: dict[str, int] = {}
    for event in events:
        event_name = str(event.get("event") or "")
        stage = str(event.get("stage") or "")
        if event_name:
            by_event[event_name] = by_event.get(event_name, 0) + 1
        if stage or event_name:
            key = f"{stage}:{event_name}"
            by_stage_event[key] = by_stage_event.get(key, 0) + 1
    telemetry = event_telemetry_by_action(events)
    durations = [
        float(item["duration_s"])
        for item in telemetry.values()
        if isinstance(item.get("duration_s"), (int, float))
    ]
    return {
        "events": len(events),
        "events_by_type": by_event,
        "events_by_stage_type": by_stage_event,
        "heartbeat_events": by_event.get("HEARTBEAT", 0),
        "event_actions": len(telemetry),
        "event_actions_with_heartbeat": sum(1 for item in telemetry.values() if int(item.get("heartbeat_count") or 0) > 0),
        "event_actions_with_duration": sum(1 for item in telemetry.values() if item.get("has_duration")),
        "event_duration_s_total": round(sum(durations), 6),
        "event_duration_s_max": round(max(durations), 6) if durations else 0,
        "lifecycle_balance": lifecycle_balance(events),
    }


def safe_float(value: Any) -> float | None:
    try:
        if value is not None and value != "":
            return float(value)
    except (TypeError, ValueError):
        return None
    return None


def current_epoch_s() -> float:
    return datetime.now(tz=timezone.utc).timestamp()


def age_s(epoch: Any, now: float | None = None) -> float | None:
    value = safe_float(epoch)
    if value is None:
        return None
    if now is None:
        now = current_epoch_s()
    return round(max(0.0, now - value), 3)


def stale_after_s(manifest: dict[str, Any]) -> float:
    config = manifest.get("config", {}) if isinstance(manifest, dict) else {}
    heartbeat = safe_float(config.get("heartbeat"))
    if heartbeat is None or heartbeat <= 0:
        return 60.0
    return max(30.0, heartbeat * 2.5)


def normalized_variants(config: dict[str, Any]) -> list[str]:
    variants = config.get("variants") or []
    if isinstance(variants, str):
        variants = variants.split()
    return [str(variant) for variant in variants]


def parse_frame_selector(spec: str | None) -> list[tuple[str, Any]] | None:
    if spec is None:
        return None
    spec = str(spec).strip()
    if not spec or spec in {"all", "*"}:
        return None
    selectors: list[tuple[str, Any]] = []
    for raw_part in spec.split(","):
        part = raw_part.strip()
        if not part:
            continue
        if part.startswith("sha256:"):
            prefix = part[len("sha256:") :].strip().lower()
            if not prefix:
                raise ValueError("invalid frame selector sha256 prefix: empty")
            selectors.append(("sha256", prefix))
            continue
        if part.startswith("base:"):
            base = part[len("base:") :].strip()
            if not base:
                raise ValueError("invalid frame selector base: empty")
            selectors.append(("base", base))
            continue
        if part.startswith("path:"):
            path_part = part[len("path:") :].strip()
            if not path_part:
                raise ValueError("invalid frame selector path: empty")
            selectors.append(("path", path_part))
            continue
        if "-" in part:
            if part.count("-") != 1:
                raise ValueError(f"invalid frame selector range: {part!r}")
            start_text, end_text = part.split("-", 1)
            try:
                start = int(start_text)
                end = int(end_text)
            except ValueError as exc:
                raise ValueError(f"invalid frame selector range: {part!r}") from exc
            if start <= 0 or end <= 0 or end < start:
                raise ValueError(f"invalid frame selector range: {part!r}")
            selectors.append(("ordinal_set", set(range(start, end + 1))))
            continue
        try:
            value = int(part)
        except ValueError as exc:
            raise ValueError(
                f"invalid frame selector {part!r}; use ordinals/ranges, sha256:<prefix>, base:<filename>, path:<substring>, all, or *"
            ) from exc
        if value <= 0:
            raise ValueError(f"invalid frame selector ordinal: {part!r}")
        selectors.append(("ordinal", value))
    if not selectors:
        raise ValueError("invalid frame selector: empty frame selection")
    return selectors


def available_frame_identities(rows: list[dict[str, Any]]) -> str:
    parts = []
    for row in rows:
        ordinal = row.get("ordinal", "")
        base = row.get("base", "")
        sha = str(row.get("sha256") or "")
        sha_short = sha[:12] if sha else ""
        path = row.get("path", "")
        parts.append(f"{ordinal}:base:{base}:sha256:{sha_short}:path:{path}")
    return "; ".join(parts) if parts else "none"


def resolve_frame_selector(rows: list[dict[str, Any]], spec: str | None, label: str = "--frames") -> set[int] | None:
    selectors = parse_frame_selector(spec)
    if selectors is None:
        return None
    selected: set[int] = set()
    errors: list[str] = []
    for kind, value in selectors:
        matches: set[int] = set()
        for row in rows:
            try:
                ordinal = int(row.get("ordinal") or 0)
            except (TypeError, ValueError):
                ordinal = 0
            if ordinal <= 0:
                continue
            if kind == "ordinal" and ordinal == int(value):
                matches.add(ordinal)
            elif kind == "ordinal_set" and ordinal in value:
                matches.add(ordinal)
            elif kind == "sha256" and str(row.get("sha256") or "").lower().startswith(str(value)):
                matches.add(ordinal)
            elif kind == "base" and str(row.get("base") or "") == str(value):
                matches.add(ordinal)
            elif kind == "path" and str(value) in str(row.get("path") or ""):
                matches.add(ordinal)
        if not matches:
            if kind == "ordinal_set":
                missing = sorted(int(v) for v in value if int(v) not in {int(r.get("ordinal") or 0) for r in rows})
                rendered = ",".join(str(v) for v in missing) if missing else str(value)
                errors.append(f"{kind}:{rendered}")
            else:
                errors.append(f"{kind}:{value}")
        selected.update(matches)
    if errors:
        raise SystemExit(
            f"{label} selector(s) matched no frames: {', '.join(errors)}; "
            f"available: {available_frame_identities(rows)}"
        )
    return selected


def selector_contains(selector: set[int] | None, ordinal: int) -> bool:
    return selector is None or ordinal in selector


def filter_rows_by_frames(rows: list[dict[str, Any]], spec: str | None) -> list[dict[str, Any]]:
    selector = resolve_frame_selector(rows, spec)
    if selector is None:
        return rows
    return [
        row
        for row in rows
        if int(row.get("ordinal") or 0) in selector
    ]


def incomplete_planned_actions(matrix: list[dict[str, Any]]) -> list[str]:
    incomplete: list[str] = []
    for row in matrix:
        ordinal = row.get("ordinal", "")
        path = row.get("path", "")
        baseline = row.get("baseline") or {}
        if baseline.get("planned") and not baseline.get("completed"):
            incomplete.append(
                f"ordinal={ordinal} action=baseline "
                f"status={baseline.get('status', '')} path={path}"
            )
        for variant, action in sorted((row.get("variants") or {}).items()):
            if action.get("planned") and not action.get("completed"):
                incomplete.append(
                    f"ordinal={ordinal} action=variant:{variant} "
                    f"status={action.get('status', '')} path={path}"
                )
        firstdiff = row.get("firstdiff") or {}
        if firstdiff.get("planned") and not firstdiff.get("completed"):
            incomplete.append(
                f"ordinal={ordinal} action=firstdiff "
                f"status={firstdiff.get('status', '')} path={path}"
            )
    return incomplete


def incomplete_action_records(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    for row in rows:
        if not (row.get("planned") and not row.get("completed")):
            continue
        evidence = row.get("event_evidence") if isinstance(row.get("event_evidence"), dict) else {}
        heartbeat = row.get("last_heartbeat") if isinstance(row.get("last_heartbeat"), dict) else {}
        out.append(
            {
                "action_id": row.get("action_id", ""),
                "stage": row.get("stage", ""),
                "mode": row.get("mode", ""),
                "variant": row.get("variant", ""),
                "ordinal": row.get("ordinal"),
                "path": row.get("path", ""),
                "base": row.get("base", ""),
                "status": row.get("status", ""),
                "reason": row.get("reason", ""),
                "log": row.get("log", ""),
                "host_log": row.get("host_log", ""),
                "heartbeat_count": int(row.get("heartbeat_count") or 0),
                "last_heartbeat": heartbeat,
                "event_evidence": evidence,
                "terminal_missing": not bool(evidence.get("has_terminal_event")) if evidence else True,
                "started_at": row.get("started_at", ""),
                "finished_at": row.get("finished_at", ""),
                "replay_commands": row.get("replay_commands", {}),
            }
        )
    return out


def stage_action_counts(rows: list[dict[str, Any]]) -> dict[str, Any]:
    stages = ("baseline", "variant", "firstdiff")
    by_stage: dict[str, dict[str, int]] = {
        stage: {
            "actions": 0,
            "planned": 0,
            "completed": 0,
            "incomplete": 0,
            "unexpected": 0,
        }
        for stage in stages
    }
    for row in rows:
        stage = str(row.get("stage") or "")
        if stage not in by_stage:
            by_stage[stage] = {
                "actions": 0,
                "planned": 0,
                "completed": 0,
                "incomplete": 0,
                "unexpected": 0,
            }
        bucket = by_stage[stage]
        bucket["actions"] += 1
        if row.get("planned"):
            bucket["planned"] += 1
            if row.get("completed"):
                bucket["completed"] += 1
            else:
                bucket["incomplete"] += 1
        elif row.get("completed") and not stage_observation_skipped(row):
            bucket["unexpected"] += 1
    overall = {
        "actions": sum(bucket["actions"] for bucket in by_stage.values()),
        "planned": sum(bucket["planned"] for bucket in by_stage.values()),
        "completed": sum(bucket["completed"] for bucket in by_stage.values()),
        "incomplete": sum(bucket["incomplete"] for bucket in by_stage.values()),
        "unexpected": sum(bucket["unexpected"] for bucket in by_stage.values()),
    }
    return {"by_stage": by_stage, "overall": overall}


def full_integrity_enabled(manifest: dict[str, Any]) -> bool:
    config = manifest.get("config", {}) if isinstance(manifest, dict) else {}
    return bool(config.get("full_integrity")) or str(config.get("mode") or "") == "full-integrity"


def full_integrity_plan_gaps(matrix: list[dict[str, Any]], manifest: dict[str, Any]) -> list[str]:
    if not full_integrity_enabled(manifest):
        return []
    config = manifest.get("config", {}) if isinstance(manifest, dict) else {}
    stage_enabled = config.get("stage_enabled", {}) if isinstance(config.get("stage_enabled"), dict) else {}
    expected_variants = normalized_variants(config)
    gaps: list[str] = []
    for row in matrix:
        ordinal = row.get("ordinal", "")
        path = row.get("path", "")
        baseline = row.get("baseline") or {}
        if bool(stage_enabled.get("baseline", False)) and not baseline.get("planned"):
            gaps.append(
                f"ordinal={ordinal} action=baseline "
                f"status={baseline.get('status', '')} reason={baseline.get('reason', '')} path={path}"
            )
        variants = row.get("variants") or {}
        if bool(stage_enabled.get("variants", False)):
            for variant in expected_variants:
                action = variants.get(variant) or {}
                if not action.get("planned"):
                    gaps.append(
                        f"ordinal={ordinal} action=variant:{variant} "
                        f"status={action.get('status', 'missing')} reason={action.get('reason', '')} path={path}"
                    )
        firstdiff = row.get("firstdiff") or {}
        if bool(stage_enabled.get("firstdiff", False)) and not firstdiff.get("planned"):
            gaps.append(
                f"ordinal={ordinal} action=firstdiff "
                f"status={firstdiff.get('status', '')} reason={firstdiff.get('reason', '')} path={path}"
            )
    return gaps


def frame_identity(row: dict[str, Any]) -> dict[str, Any]:
    return {
        "ordinal": row.get("ordinal"),
        "index": row.get("index"),
        "selection_ordinal": row.get("selection_ordinal"),
        "selected_total": row.get("selected_total"),
        "grammar": row.get("grammar"),
        "corpus_kind": row.get("corpus_kind"),
        "corpus_root": row.get("corpus_root"),
        "path": row.get("path"),
        "base": row.get("base"),
        "size": row.get("size"),
        "sha256": row.get("sha256"),
    }


def recorded_wringer_path(wringer_dir: Path, manifest: dict[str, Any], *parts: str) -> str:
    root = str(manifest.get("out_dir") or wringer_dir)
    return str(Path(root).joinpath(*parts))


def action_log_path(
    wringer_dir: Path,
    manifest: dict[str, Any],
    row: dict[str, Any],
    stage: str,
    variant: str = "",
    action: dict[str, Any] | None = None,
) -> str:
    action = action or {}
    log = str(action.get("log") or "")
    if log:
        return log
    ordinal = int(row.get("ordinal") or 0)
    frame_label = f"frame-{ordinal:04d}"
    if stage == "baseline":
        return str((row.get("baseline") or {}).get("log") or "")
    if stage == "variant" and variant:
        return recorded_wringer_path(wringer_dir, manifest, "variants", variant, f"{frame_label}.log")
    if stage == "firstdiff":
        return recorded_wringer_path(wringer_dir, manifest, "firstdiff", f"{frame_label}.log")
    return ""


def action_timeout(manifest: dict[str, Any], stage: str) -> str:
    config = manifest.get("config", {})
    if stage == "firstdiff":
        return str(config.get("diag_timeout") or config.get("timeout") or "")
    if stage in {"baseline", "variant"}:
        return str(config.get("timeout") or "")
    return ""


def action_replay_commands(row: dict[str, Any], stage: str, variant: str = "", action: dict[str, Any] | None = None) -> dict[str, str]:
    action = action or {}
    replay_plan = row.get("replay_plan") or {}
    commands: dict[str, str] = {}
    recorded = str(action.get("replay_command") or "")
    if recorded:
        commands["recorded"] = recorded
    if stage == "baseline":
        for name in ("baseline_command", "direct_baseline_command"):
            command = str(replay_plan.get(name) or "")
            if command:
                commands[name] = command
    elif stage == "variant":
        command = str(replay_plan.get("baseline_reuse_variant_command") or "")
        if command:
            commands["baseline_reuse_variant_command"] = command
        direct = replay_plan.get("direct_variant_commands", {})
        if isinstance(direct, dict):
            command = str(direct.get(variant) or "")
            if command:
                commands["direct_variant_command"] = command
    elif stage == "firstdiff":
        for name in ("baseline_reuse_firstdiff_command", "direct_firstdiff_command"):
            command = str(replay_plan.get(name) or "")
            if command:
                commands[name] = command
    return commands


def build_action_row(
    *,
    wringer_dir: Path,
    manifest: dict[str, Any],
    row: dict[str, Any],
    stage: str,
    mode: str,
    action: dict[str, Any],
    action_index: int,
    variant: str = "",
    event_telemetry: dict[str, Any] | None = None,
    event_scope: str = "action",
) -> dict[str, Any]:
    ordinal = int(row.get("ordinal") or 0)
    action_id_parts = [f"{ordinal:04d}", stage, mode]
    action_id = ":".join(part for part in action_id_parts if part)
    out = {
        "action_id": action_id,
        "action_index": action_index,
        "stage": stage,
        "mode": mode,
        "variant": variant,
        "planned": bool(action.get("planned")),
        "completed": bool(action.get("completed")),
        "status": action.get("status", ""),
        "timeout_seconds": action_timeout(manifest, stage),
        "log": action_log_path(wringer_dir, manifest, row, stage, variant, action),
        "host_log": action.get("host_log", ""),
        "replay_commands": action_replay_commands(row, stage, variant, action),
        "frame_matrix_key": {
            "ordinal": row.get("ordinal"),
            "path": row.get("path"),
            "stage": stage,
            "mode": mode,
            "variant": variant,
        },
    }
    reason = str(action.get("reason") or "")
    if reason:
        out["reason"] = reason
    for key, value in frame_identity(row).items():
        out[key] = value
    for key in (
        "rc",
        "timeout",
        "parity_match",
        "comparison_results",
        "comparison_diagnostic",
        "measure",
        "parser_progress",
        "last_parser_progress",
    ):
        if key in action:
            out[key] = action[key]
    if event_telemetry:
        out["event_scope"] = event_scope
        out["started_at"] = event_telemetry.get("started_at", "")
        out["finished_at"] = event_telemetry.get("finished_at", "")
        if "duration_s" in event_telemetry:
            out["duration_s"] = event_telemetry.get("duration_s")
        if "elapsed_s" in event_telemetry:
            out["elapsed_s"] = event_telemetry.get("elapsed_s")
        out["heartbeat_count"] = int(event_telemetry.get("heartbeat_count") or 0)
        if event_telemetry.get("last_heartbeat"):
            out["last_heartbeat"] = event_telemetry.get("last_heartbeat")
        if event_telemetry.get("last_parser_progress"):
            out["last_parser_progress"] = event_telemetry.get("last_parser_progress")
        if event_telemetry.get("parser_progress"):
            out["parser_progress"] = event_telemetry.get("parser_progress")
        out["event_lifecycle"] = event_telemetry.get("event_lifecycle", [])
        out["event_evidence"] = {
            "has_start_event": bool(event_telemetry.get("has_start_event")),
            "has_terminal_event": bool(event_telemetry.get("has_terminal_event")),
            "has_duration": bool(event_telemetry.get("has_duration")),
            "terminal_event_type": event_telemetry.get("terminal_event_type", ""),
            "child_pid": event_telemetry.get("child_pid", ""),
            "worker_pid": event_telemetry.get("worker_pid", event_telemetry.get("child_pid", "")),
        }
    return out


def observed_variant_keys(wringer_dir: Path) -> dict[tuple[str, str, str], dict[str, Any]]:
    observed: dict[tuple[str, str, str], dict[str, Any]] = {}
    for path, by_variant in load_variant_outcomes(wringer_dir).items():
        for variant, outcome in by_variant.items():
            observed[("variant", path, variant)] = outcome
    return observed


def observed_firstdiff_keys(wringer_dir: Path) -> dict[tuple[str, str, str], dict[str, Any]]:
    observed: dict[tuple[str, str, str], dict[str, Any]] = {}
    for path, outcome in load_firstdiff(wringer_dir).items():
        observed[("firstdiff", path, "firstdiff")] = outcome
    return observed


def build_plan(
    wringer_dir: Path,
    matrix: list[dict[str, Any]] | None = None,
) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    config = manifest.get("config", {}) if isinstance(manifest.get("config"), dict) else {}
    if matrix is None:
        matrix = build_frame_matrix(wringer_dir)
    event_telemetry = event_telemetry_by_action(load_wringer_events(wringer_dir))
    baseline_delegated_telemetry = event_telemetry.get(("baseline", "tier_scan", 0, ""))
    baseline_frame_telemetry = baseline_frame_telemetry_by_ordinal(wringer_dir)
    rows: list[dict[str, Any]] = []
    planned_keys: set[tuple[str, str, str]] = set()
    action_index = 0
    for frame in matrix:
        baseline = frame.get("baseline") or {}
        frame_ordinal = int(frame.get("ordinal") or 0)
        baseline_telemetry = baseline_frame_telemetry.get(frame_ordinal) or baseline_delegated_telemetry
        baseline_event_scope = "baseline_frame" if frame_ordinal in baseline_frame_telemetry else "delegated_baseline"
        action_index += 1
        rows.append(
            build_action_row(
                wringer_dir=wringer_dir,
                manifest=manifest,
                row=frame,
                stage="baseline",
                mode="baseline",
                action=baseline,
                action_index=action_index,
                event_telemetry=baseline_telemetry,
                event_scope=baseline_event_scope,
            )
        )
        if baseline.get("planned"):
            planned_keys.add(("baseline", str(frame.get("path") or ""), "baseline"))

        for variant, action in sorted((frame.get("variants") or {}).items()):
            action_index += 1
            action_row = build_action_row(
                wringer_dir=wringer_dir,
                manifest=manifest,
                row=frame,
                stage="variant",
                mode=str(variant),
                variant=str(variant),
                action=action,
                action_index=action_index,
                event_telemetry=event_telemetry.get(("variant", str(variant), int(frame.get("ordinal") or 0), str(frame.get("path") or ""))),
            )
            rows.append(action_row)
            if action.get("planned"):
                planned_keys.add(("variant", str(frame.get("path") or ""), str(variant)))

        firstdiff = frame.get("firstdiff") or {}
        action_index += 1
        rows.append(
            build_action_row(
                wringer_dir=wringer_dir,
                manifest=manifest,
                row=frame,
                stage="firstdiff",
                mode="firstdiff",
                action=firstdiff,
                action_index=action_index,
                event_telemetry=event_telemetry.get(("firstdiff", "firstdiff", int(frame.get("ordinal") or 0), str(frame.get("path") or ""))),
            )
        )
        if firstdiff.get("planned"):
            planned_keys.add(("firstdiff", str(frame.get("path") or ""), "firstdiff"))

    observed = {}
    observed.update(observed_variant_keys(wringer_dir))
    observed.update(observed_firstdiff_keys(wringer_dir))
    unexpected = []
    frames_by_path = {str(frame.get("path") or ""): frame for frame in matrix}
    for key, meta in sorted(observed.items()):
        if key in planned_keys:
            continue
        stage, path, mode = key
        if not stage_enabled_for_observed(config, stage):
            continue
        status = terminal_status(meta) if stage == "firstdiff" else variant_terminal_status(meta)
        unexpected.append(
            {
                "stage": stage,
                "mode": mode,
                "variant": mode if stage == "variant" else "",
                "ordinal": meta.get("ordinal"),
                "grammar": meta.get("grammar"),
                "corpus_kind": meta.get("corpus_kind"),
                "path": path,
                "log": meta.get("log", ""),
                "host_log": meta.get("host_log", ""),
                "status": status,
                "reason": "unplanned_observed_evidence",
                "rc": meta.get("rc"),
                "timeout": bool(meta.get("timeout")),
                "replay_command": meta.get("replay_command", ""),
                "comparison_diagnostic": meta.get("comparison_diagnostic", {}),
                "parser_progress": meta.get("parser_progress", {}),
                "last_parser_progress": meta.get("last_parser_progress", {}),
            }
        )
        action_index += 1
        frame = frames_by_path.get(path, {})
        action_row: dict[str, Any] = {
            "action_id": f"unexpected:{stage}:{int(meta.get('ordinal') or 0):04d}:{mode}",
            "action_index": action_index,
            "stage": stage,
            "mode": mode,
            "variant": mode if stage == "variant" else "",
            "planned": False,
            "completed": True,
            "status": status,
            "reason": "unplanned_observed_evidence",
            "timeout_seconds": action_timeout(manifest, stage),
            "log": meta.get("log", ""),
            "host_log": meta.get("host_log", ""),
            "replay_commands": {"recorded": meta.get("replay_command", "")},
            "frame_matrix_key": {
                "ordinal": meta.get("ordinal"),
                "path": path,
                "stage": stage,
                "mode": mode,
                "variant": mode if stage == "variant" else "",
            },
            "rc": meta.get("rc"),
            "timeout": bool(meta.get("timeout")),
        }
        if meta.get("comparison_diagnostic"):
            action_row["comparison_diagnostic"] = meta.get("comparison_diagnostic", {})
        if meta.get("parser_progress"):
            action_row["parser_progress"] = meta.get("parser_progress", {})
        if meta.get("last_parser_progress"):
            action_row["last_parser_progress"] = meta.get("last_parser_progress", {})
        if frame:
            for identity_key, value in frame_identity(frame).items():
                action_row[identity_key] = value
        else:
            action_row.update(
                {
                    "ordinal": meta.get("ordinal"),
                    "grammar": meta.get("grammar"),
                    "corpus_kind": meta.get("corpus_kind"),
                    "corpus_root": meta.get("corpus_root"),
                    "path": path,
                    "sha256": meta.get("sha256", ""),
                }
            )
        if stage == "variant":
            action_row["parity_match"] = bool(meta.get("parity_match"))
        telemetry = event_telemetry.get((stage, mode, int(meta.get("ordinal") or 0), path))
        if telemetry:
            action_row["started_at"] = telemetry.get("started_at", "")
            action_row["finished_at"] = telemetry.get("finished_at", "")
            if "duration_s" in telemetry:
                action_row["duration_s"] = telemetry.get("duration_s")
            if "elapsed_s" in telemetry:
                action_row["elapsed_s"] = telemetry.get("elapsed_s")
            action_row["heartbeat_count"] = int(telemetry.get("heartbeat_count") or 0)
            if telemetry.get("last_heartbeat"):
                action_row["last_heartbeat"] = telemetry.get("last_heartbeat")
            action_row["event_lifecycle"] = telemetry.get("event_lifecycle", [])
            action_row["event_evidence"] = {
                "has_start_event": bool(telemetry.get("has_start_event")),
                "has_terminal_event": bool(telemetry.get("has_terminal_event")),
                "has_duration": bool(telemetry.get("has_duration")),
                "terminal_event_type": telemetry.get("terminal_event_type", ""),
                "child_pid": telemetry.get("child_pid", ""),
            }
        rows.append(action_row)

    planned_total = sum(1 for row in rows if row.get("planned"))
    completed_planned = sum(1 for row in rows if row.get("planned") and row.get("completed"))
    durations = [
        float(row["duration_s"])
        for row in rows
        if isinstance(row.get("duration_s"), (int, float))
    ]
    action_counts = stage_action_counts(rows)
    incomplete_actions = incomplete_action_records(rows)
    summary = {
        "actions": len(rows),
        "planned_actions": planned_total,
        "completed_planned_actions": completed_planned,
        "incomplete_planned_actions": planned_total - completed_planned,
        "action_counts": action_counts,
        "planned_counts_by_stage": {
            stage: counts["planned"]
            for stage, counts in action_counts.get("by_stage", {}).items()
        },
        "completed_counts_by_stage": {
            stage: counts["completed"]
            for stage, counts in action_counts.get("by_stage", {}).items()
        },
        "incomplete_counts_by_stage": {
            stage: counts["incomplete"]
            for stage, counts in action_counts.get("by_stage", {}).items()
        },
        "incomplete_actions": incomplete_actions,
        "unexpected_observed_actions": len(unexpected),
        "unexpected_actions": unexpected,
        "telemetry": {
            "heartbeat_events": sum(int(item.get("heartbeat_count") or 0) for item in event_telemetry.values()),
            "baseline_frame_heartbeat_events": sum(int(item.get("heartbeat_count") or 0) for item in baseline_frame_telemetry.values()),
            "baseline_frame_actions_with_event_telemetry": len(baseline_frame_telemetry),
            "actions_with_event_telemetry": sum(1 for row in rows if row.get("event_evidence")),
            "actions_with_heartbeat": sum(1 for row in rows if int(row.get("heartbeat_count") or 0) > 0),
            "actions_with_duration": sum(1 for row in rows if isinstance(row.get("duration_s"), (int, float))),
            "duration_s_total": round(sum(durations), 6),
            "duration_s_max": round(max(durations), 6) if durations else 0,
        },
        "artifacts": {
            "wringer_plan_jsonl": str(wringer_dir / "wringer_plan.jsonl"),
            "wringer_plan_json": str(wringer_dir / "wringer_plan.json"),
            "frame_matrix_jsonl": str(wringer_dir / "frame_matrix.jsonl"),
            "wringer_manifest_json": str(wringer_dir / "wringer_manifest.json"),
        },
    }
    return rows, summary


def lifecycle_event_with_age(event: dict[str, Any], now: float, stale_after: float) -> dict[str, Any]:
    heartbeat = event.get("last_heartbeat") if isinstance(event.get("last_heartbeat"), dict) else {}
    heartbeat_epoch = heartbeat.get("epoch_s") if heartbeat else event.get("last_heartbeat_at_epoch_s")
    event_age = age_s(heartbeat_epoch if heartbeat_epoch is not None else event_epoch(event), now)
    out = {
        "stage": event.get("stage", ""),
        "mode": event.get("mode", ""),
        "variant": event.get("variant", ""),
        "ordinal": event.get("ordinal", ""),
        "grammar": event.get("grammar", ""),
        "path": event.get("path", event.get("file", "")),
        "file": event.get("file", ""),
        "log": event.get("log", ""),
        "last_event": event.get("event", ""),
        "last_event_ts": event.get("ts", ""),
        "last_heartbeat_ts": heartbeat.get("ts", event.get("last_heartbeat_at", "")),
        "age_s": event_age,
        "stale": bool(event_age is not None and event_age > stale_after),
    }
    for field in ("elapsed_s", "duration_s", "log_age_s", "progress_age_s", "log_bytes", "child_pid", "worker_pid"):
        if field in event:
            out[field] = event.get(field)
        elif heartbeat and field in heartbeat:
            out[field] = heartbeat.get(field)
    if heartbeat:
        out["last_heartbeat"] = heartbeat
    return out


def baseline_row_contract_errors(matrix: list[dict[str, Any]], plan_rows: list[dict[str, Any]]) -> list[str]:
    errors: list[str] = []
    seen: set[tuple[int, str]] = set()
    for row in matrix:
        ordinal = int(row.get("ordinal") or 0)
        path = str(row.get("path") or "")
        key = (ordinal, path)
        if key in seen:
            errors.append(f"duplicate selected frame ordinal={ordinal} path={path}")
        seen.add(key)
        baseline = row.get("baseline") or {}
        if not baseline.get("planned"):
            errors.append(f"selected frame lacks planned baseline ordinal={ordinal} path={path}")
    baseline_plan: dict[tuple[int, str], int] = {}
    for row in plan_rows:
        if str(row.get("stage") or "") != "baseline":
            continue
        key = (int(row.get("ordinal") or 0), str(row.get("path") or ""))
        baseline_plan[key] = baseline_plan.get(key, 0) + 1
    for key in sorted(seen):
        count = baseline_plan.get(key, 0)
        if count != 1:
            ordinal, path = key
            errors.append(f"expected exactly one baseline plan row ordinal={ordinal} count={count} path={path}")
    for key, count in sorted(baseline_plan.items()):
        if key not in seen:
            ordinal, path = key
            errors.append(f"baseline plan row outside selected matrix ordinal={ordinal} path={path}")
        elif count != 1:
            ordinal, path = key
            errors.append(f"duplicate baseline plan row ordinal={ordinal} count={count} path={path}")
    return errors


def lifecycle_scope(matrix: list[dict[str, Any]]) -> tuple[set[int], set[str]]:
    ordinals = {
        int(row.get("ordinal") or 0)
        for row in matrix
        if int(row.get("ordinal") or 0) > 0
    }
    paths = {str(row.get("path") or "") for row in matrix if str(row.get("path") or "")}
    return ordinals, paths


def lifecycle_key_in_scope(
    stage: str,
    ordinal: int,
    path: str,
    scope_ordinals: set[int],
    scope_paths: set[str],
) -> bool:
    if not scope_ordinals and not scope_paths:
        return False
    if stage == "baseline" and ordinal == 0 and not path:
        return True
    return (ordinal > 0 and ordinal in scope_ordinals) or (bool(path) and path in scope_paths)


def lifecycle_event_in_scope(
    event: dict[str, Any],
    scope_ordinals: set[int],
    scope_paths: set[str],
) -> bool:
    try:
        ordinal = int(event.get("ordinal") or 0)
    except (TypeError, ValueError):
        ordinal = 0
    path = str(event.get("path") or event.get("file") or "")
    stage = str(event.get("stage") or "")
    return lifecycle_key_in_scope(stage, ordinal, path, scope_ordinals, scope_paths)


def unexpected_lifecycle_actions(
    matrix: list[dict[str, Any]],
    events: list[dict[str, Any]],
    manifest: dict[str, Any],
    *,
    scoped: bool = False,
) -> list[dict[str, Any]]:
    config = manifest.get("config", {}) if isinstance(manifest, dict) else {}
    stage_enabled = config.get("stage_enabled", {}) if isinstance(config.get("stage_enabled"), dict) else {}
    scope_ordinals, scope_paths = lifecycle_scope(matrix)
    allowed: set[tuple[str, str, int, str]] = set()
    if stage_enabled.get("baseline"):
        allowed.add(("baseline", "tier_scan", 0, ""))
    for row in matrix:
        ordinal = int(row.get("ordinal") or 0)
        path = str(row.get("path") or "")
        for variant, action in sorted((row.get("variants") or {}).items()):
            if action.get("planned"):
                allowed.add(("variant", str(variant), ordinal, path))
        firstdiff = row.get("firstdiff") or {}
        if firstdiff.get("planned"):
            allowed.add(("firstdiff", "firstdiff", ordinal, path))
    out: list[dict[str, Any]] = []
    for key, telemetry in event_telemetry_by_action(events).items():
        stage, mode, ordinal, path = key
        if stage not in {"variant", "firstdiff", "baseline"}:
            continue
        if not stage_enabled_for_observed(config, stage):
            continue
        if scoped and not lifecycle_key_in_scope(stage, ordinal, path, scope_ordinals, scope_paths):
            continue
        if key in allowed:
            continue
        if stage == "baseline" and mode == "tier_scan" and ordinal == 0 and path == "":
            continue
        out.append(
            {
                "stage": stage,
                "mode": mode,
                "variant": mode if stage == "variant" else "",
                "ordinal": ordinal,
                "path": path,
                "event_lifecycle": telemetry.get("event_lifecycle", []),
                "started_at": telemetry.get("started_at", ""),
                "finished_at": telemetry.get("finished_at", ""),
                "last_heartbeat": telemetry.get("last_heartbeat", {}),
                "terminal_event_type": telemetry.get("terminal_event_type", ""),
                "reason": "lifecycle_action_outside_matrix",
            }
        )
    return out


def build_control_contract(
    wringer_dir: Path,
    frames: str | None = None,
    matrix: list[dict[str, Any]] | None = None,
    plan_rows: list[dict[str, Any]] | None = None,
    plan_summary: dict[str, Any] | None = None,
) -> dict[str, Any]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    config = manifest.get("config", {}) if isinstance(manifest, dict) else {}
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    all_matrix = build_frame_matrix(wringer_dir) if matrix is None else matrix
    filtered_matrix = filter_rows_by_frames(all_matrix, frames)
    if plan_rows is None or plan_summary is None:
        all_plan_rows, built_summary = build_plan(wringer_dir, all_matrix)
        if plan_rows is None:
            plan_rows = all_plan_rows
        if plan_summary is None:
            plan_summary = built_summary
    filtered_plan_rows = filter_rows_by_frames(plan_rows, frames)
    events = load_wringer_events(wringer_dir)
    wringer_balance = lifecycle_balance(events)
    baseline_balance = baseline_frame_lifecycle_balance(wringer_dir)
    frame_scoped = bool(frames and str(frames).strip() not in {"", "all", "*"})
    scope_ordinals, scope_paths = lifecycle_scope(filtered_matrix)
    now = current_epoch_s()
    stale_after = stale_after_s(manifest)
    open_actions = [
        lifecycle_event_with_age(event, now, stale_after)
        for event in wringer_balance.get("open_frames", [])
        if not frame_scoped or lifecycle_event_in_scope(event, scope_ordinals, scope_paths)
    ]
    open_baseline_frames = [
        lifecycle_event_with_age(event, now, stale_after)
        for event in baseline_balance.get("open_frames", [])
        if not frame_scoped or lifecycle_event_in_scope(event, scope_ordinals, scope_paths)
    ]
    stale_actions = [item for item in open_actions if item.get("stale")]
    stale_baseline_frames = [item for item in open_baseline_frames if item.get("stale")]
    action_counts = stage_action_counts(filtered_plan_rows)
    incomplete_actions = incomplete_action_records(filtered_plan_rows)
    baseline_errors = baseline_row_contract_errors(filtered_matrix, filtered_plan_rows)
    unexpected_events = unexpected_lifecycle_actions(
        filtered_matrix,
        events,
        manifest,
        scoped=frame_scoped,
    )
    unexpected_summary_actions = filter_unexpected_actions_by_frames(
        plan_summary.get("unexpected_actions", []) if isinstance(plan_summary, dict) else [],
        all_matrix,
        frames,
    )
    selected_ordinals = [
        int(row.get("ordinal") or 0)
        for row in filtered_matrix
        if int(row.get("ordinal") or 0) > 0
    ]
    active = read_active(wringer_dir / "wringer_active.txt")
    baseline_active = read_active(baseline_dir / "active_grammar.txt")
    artifacts = {
        "wringer_active_txt": str(wringer_dir / "wringer_active.txt"),
        "wringer_active_json": str(wringer_dir / "wringer_active.json"),
        "wringer_events_jsonl": str(wringer_dir / "wringer_events.jsonl"),
        "frame_catalog_jsonl": str(wringer_dir / "frame_catalog.jsonl"),
        "frame_matrix_jsonl": str(wringer_dir / "frame_matrix.jsonl"),
        "wringer_plan_jsonl": str(wringer_dir / "wringer_plan.jsonl"),
        "wringer_plan_json": str(wringer_dir / "wringer_plan.json"),
        "wringer_frames_jsonl": str(wringer_dir / "wringer_frames.jsonl"),
        "wringer_summary_json": str(wringer_dir / "wringer_summary.json"),
        "wringer_summary_md": str(wringer_dir / "wringer_summary.md"),
        "commands_log": str(wringer_dir / "commands.log"),
        "baseline_active_txt": str(baseline_dir / "active_grammar.txt"),
        "baseline_progress": str(baseline_dir / "progress.log"),
        "baseline_status": str(baseline_dir / "status.tsv"),
        "baseline_frames_jsonl": str(baseline_dir / "frames.jsonl"),
    }
    next_files = [
        artifacts["wringer_active_json"],
        artifacts["wringer_events_jsonl"],
        artifacts["wringer_plan_jsonl"],
        artifacts["frame_matrix_jsonl"],
        artifacts["baseline_status"],
        artifacts["baseline_progress"],
        artifacts["commands_log"],
    ]
    for item in open_actions + open_baseline_frames + incomplete_actions:
        log = str(item.get("host_log") or item.get("log") or "")
        if log and log not in next_files:
            next_files.append(log)
    contract_status = "closed" if not (
        action_counts["overall"]["incomplete"]
        or open_actions
        or open_baseline_frames
        or stale_actions
        or stale_baseline_frames
        or baseline_errors
        or unexpected_summary_actions
        or unexpected_events
    ) else "open"
    return {
        "status": contract_status,
        "wringer_dir": str(wringer_dir),
        "grammar": manifest.get("grammar", ""),
        "mode": config.get("mode", ""),
        "profile": config.get("profile", ""),
        "full_integrity": full_integrity_enabled(manifest),
        "active": {
            "wringer": active,
            "baseline": baseline_active,
        },
        "selected_frames": {
            "count": len(filtered_matrix),
            "ordinals": selected_ordinals,
        },
        "action_counts": action_counts,
        "incomplete_actions": incomplete_actions,
        "open_actions": open_actions,
        "stale_actions": stale_actions,
        "unexpected_actions": unexpected_summary_actions,
        "unexpected_lifecycle_actions": unexpected_events,
        "baseline_frame_lifecycle_balance": {
            **baseline_balance,
            "open_frames": open_baseline_frames,
            "stale_frames": stale_baseline_frames,
            "open": len(open_baseline_frames) if frame_scoped else baseline_balance.get("open", 0),
        },
        "wringer_lifecycle_balance": {
            **wringer_balance,
            "open_frames": open_actions,
            "stale_actions": stale_actions,
            "open": len(open_actions) if frame_scoped else wringer_balance.get("open", 0),
        },
        "baseline_row_contract_errors": baseline_errors,
        "stale_after_s": stale_after,
        "artifacts": artifacts,
        "next_files": next_files,
    }


def terminal_status(meta: dict[str, Any] | None) -> str:
    if not meta:
        return "not_run"
    if meta.get("timeout"):
        return "timeout"
    try:
        rc = int(meta.get("rc"))
    except (TypeError, ValueError):
        return "fail"
    return "end" if rc == 0 else "fail"


def variant_terminal_status(outcome: dict[str, Any] | None) -> str:
    status = terminal_status(outcome)
    if status != "end":
        return status
    if outcome and outcome.get("malformed"):
        return "fail"
    return "match" if outcome and outcome.get("parity_match") else "nonmatch"


def baseline_completed(terminal: dict[str, Any]) -> bool:
    if not terminal:
        return False
    if terminal.get("timeout") or terminal.get("fail") or terminal.get("panic"):
        return True
    if str(terminal.get("measure_line") or ""):
        return True
    return bool(str(terminal.get("comparison_result") or ""))


def baseline_terminal_status(terminal: dict[str, Any]) -> str:
    if not baseline_completed(terminal):
        return "not_run"
    if terminal.get("timeout"):
        return "timeout"
    if terminal.get("fail") or terminal.get("panic"):
        return "fail"
    return "match" if terminal.get("comparison_result") == "match" else "nonmatch"


def boolish(value: Any) -> bool:
    if isinstance(value, bool):
        return value
    return str(value).strip().lower() in {"1", "true", "yes", "y"}


def first_int(value: Any) -> int | None:
    if value is None:
        return None
    match = re.search(r"-?\d+", str(value))
    if not match:
        return None
    try:
        return int(match.group(0))
    except ValueError:
        return None


def span_end(value: Any) -> int | None:
    match = re.match(r"^-?\d+:(-?\d+)$", str(value or ""))
    if not match:
        return None
    try:
        return int(match.group(1))
    except ValueError:
        return None


def text_has_scanner_evidence(*values: Any) -> bool:
    text = " ".join(str(value or "") for value in values).lower()
    scanner_needles = (
        "scanner",
        "external_scanner",
        "external scanner",
        "unknown_scanner",
        "unknown scanner",
        "scan_token",
        "scanner_token",
        "token stream",
        "token_stream",
    )
    return any(needle in text for needle in scanner_needles)


NO_EVIDENCE_FAMILY = "not_run_no_evidence"
NO_EVIDENCE_STATUSES = {"", "not_run", "not_planned"}
DIAGNOSTIC_SCANNER_EXCLUDED_FIELDS = {"path", "base", "raw"}


def diagnostic_scanner_values(rows: list[dict[str, Any]]) -> list[Any]:
    values: list[Any] = []
    for row in rows:
        for key, value in row.items():
            if str(key) in DIAGNOSTIC_SCANNER_EXCLUDED_FIELDS:
                continue
            if isinstance(value, (str, int, float, bool)) or value is None:
                values.append(value)
    return values


def stage_observation_skipped(action: dict[str, Any]) -> bool:
    return (
        bool(action.get("completed"))
        and not bool(action.get("planned"))
        and str(action.get("reason") or "") == "stage_disabled"
    )


def stage_enabled_for_observed(config: dict[str, Any], stage: str) -> bool:
    stage_enabled = config.get("stage_enabled", {}) if isinstance(config.get("stage_enabled"), dict) else {}
    if stage == "variant":
        return bool(stage_enabled.get("variants", False))
    if stage in {"baseline", "firstdiff"}:
        return bool(stage_enabled.get(stage, False))
    return True


def diagnostic_rows(terminal: dict[str, Any]) -> list[dict[str, Any]]:
    diag = terminal.get("comparison_diagnostic")
    if not isinstance(diag, dict):
        return []
    rows = diag.get("rows")
    return rows if isinstance(rows, list) else []


def full_span_accepted_no_error(row: dict[str, Any], hints: dict[str, str]) -> bool:
    if str(row.get("goStop") or hints.get("stopReason") or "") != "accepted":
        return False
    if boolish(row.get("goRootErr")) or boolish(row.get("cRootErr")):
        return False
    if (row.get("goErrorCount") or 0) != 0 or (row.get("cErrorCount") or 0) != 0:
        return False
    go_end = span_end(row.get("goSpan"))
    c_end = span_end(row.get("cSpan"))
    expected = first_int(hints.get("expectedEOF"))
    token_end = first_int(hints.get("lastTokenEnd"))
    if go_end is not None and c_end is not None and go_end != c_end:
        return False
    if expected is not None:
        if go_end is not None and go_end != expected:
            return False
        if c_end is not None and c_end != expected:
            return False
        if token_end is not None and token_end != expected:
            return False
    return not boolish(hints.get("truncated")) and not boolish(hints.get("tokenEOFEarly"))


def frontier_loss_evidence(terminal: dict[str, Any], row: dict[str, Any] | None = None) -> list[str]:
    reasons: list[str] = []
    measure = terminal.get("measure") if isinstance(terminal.get("measure"), dict) else {}
    hints = terminal.get("runtime_hints") if isinstance(terminal.get("runtime_hints"), dict) else {}
    if first_int(measure.get("trunc")) and first_int(measure.get("trunc")) > 0:
        reasons.append("measure_trunc")
    if boolish(hints.get("truncated")):
        reasons.append("runtime_truncated")
    if str(hints.get("stopReason") or "") == "no_stacks_alive":
        reasons.append("stopReason_no_stacks_alive")
    if boolish(hints.get("tokenEOFEarly")):
        reasons.append("token_eof_early")
    expected = first_int(hints.get("expectedEOF"))
    token_end = first_int(hints.get("lastTokenEnd"))
    if expected is not None and token_end is not None and 0 <= token_end < expected:
        reasons.append("token_end_before_eof")
    if row:
        go_end = span_end(row.get("goSpan"))
        c_end = span_end(row.get("cSpan"))
        if c_end is not None and go_end is not None and 0 <= go_end < c_end:
            reasons.append("go_span_shorter_than_c")
        if expected is not None and go_end is not None and 0 <= go_end < expected:
            reasons.append("go_span_short_of_eof")
    return sorted(set(reasons))


def timeout_progress_reason(baseline: dict[str, Any]) -> str:
    parser = baseline.get("last_parser_progress")
    if not isinstance(parser, dict):
        terminal = baseline.get("terminal") if isinstance(baseline.get("terminal"), dict) else {}
        parser = terminal.get("last_parser_progress") if isinstance(terminal.get("last_parser_progress"), dict) else {}
    phase = str(parser.get("parser_phase") or "")
    lowered = phase.lower()
    if any(part in lowered for part in ("forest", "material", "fanout", "reduce")):
        return f"parser_phase_{phase}"
    if phase:
        return f"parser_phase_{phase}"
    return "terminal_timeout"


def c_recovery_gate_reason(runtime_hints: dict[str, str]) -> str | None:
    slug = str(runtime_hints.get("cRecoveryGateReason") or "").strip()
    if not slug:
        return None
    return f"c_recovery_gate_{slug}"


def runtime_hints_with_diagnostic_fallback(
    terminal: dict[str, Any],
    first_diag_row: dict[str, Any] | None = None,
) -> dict[str, str]:
    hints: dict[str, str] = {}
    terminal_hints = terminal.get("runtime_hints") if isinstance(terminal.get("runtime_hints"), dict) else {}
    hints.update({str(key): str(value) for key, value in terminal_hints.items()})
    if first_diag_row:
        row_hints = (
            first_diag_row.get("runtime_hints")
            if isinstance(first_diag_row.get("runtime_hints"), dict)
            else parse_runtime_hints(
                str(first_diag_row.get("runtime") or first_diag_row.get("goRuntime") or "")
            )
        )
        for key, value in row_hints.items():
            hints.setdefault(str(key), str(value))
    return hints


def append_c_recovery_gate_reason(family: dict[str, Any], runtime_hints: dict[str, str]) -> dict[str, Any]:
    reason = c_recovery_gate_reason(runtime_hints)
    if not reason:
        return family
    reasons = list(family.get("family_reasons") or [])
    if reason not in reasons:
        reasons.append(reason)
    family["family_reasons"] = reasons
    return family


def classify_frame_family(row: dict[str, Any]) -> dict[str, Any]:
    baseline = row.get("baseline") if isinstance(row.get("baseline"), dict) else {}
    terminal = baseline.get("terminal") if isinstance(baseline.get("terminal"), dict) else {}
    status = str(baseline.get("status") or "")
    reasons: list[str] = []
    diag_rows = diagnostic_rows(terminal)
    first_row = diag_rows[0] if diag_rows else {}
    measure = terminal.get("measure") if isinstance(terminal.get("measure"), dict) else {}
    hints = runtime_hints_with_diagnostic_fallback(terminal, first_row)
    variants = row.get("variants") if isinstance(row.get("variants"), dict) else {}

    if status == "match":
        return {"family": "clean", "family_reasons": ["baseline_match"]}

    if not baseline.get("completed") and status in NO_EVIDENCE_STATUSES:
        return append_c_recovery_gate_reason(
            {"family": NO_EVIDENCE_FAMILY, "family_reasons": ["baseline_not_run_no_terminal_evidence"]},
            hints,
        )

    if baseline.get("timeout") or terminal.get("timeout") or status == "timeout":
        return append_c_recovery_gate_reason(
            {
                "family": "timeout_fanout_perf",
                "family_reasons": [timeout_progress_reason(baseline)],
            },
            hints,
        )

    scanner_text_values: list[Any] = [
        terminal.get("runtime"),
        terminal.get("measure_line"),
        baseline.get("status_detail"),
        baseline.get("reason"),
    ]
    scanner_text_values.extend(diagnostic_scanner_values(diag_rows))
    if text_has_scanner_evidence(*scanner_text_values):
        return append_c_recovery_gate_reason(
            {
                "family": "scanner_token_accounting_or_unknown_scanner",
                "family_reasons": ["scanner_or_token_stream_evidence"],
            },
            hints,
        )

    frontier_reasons = frontier_loss_evidence(terminal, first_row)
    if frontier_reasons:
        return append_c_recovery_gate_reason(
            {
                "family": "truncation_frontier_loss",
                "family_reasons": frontier_reasons,
            },
            hints,
        )

    if first_row and full_span_accepted_no_error(first_row, hints):
        variant_statuses = [
            str(outcome.get("status") or "")
            for outcome in variants.values()
            if isinstance(outcome, dict) and outcome.get("completed")
        ]
        reasons.append(f"accepted_{first_row.get('diff') or 'shape'}_diff")
        reasons.append("full_span_no_error")
        if variant_statuses and "match" not in variant_statuses:
            reasons.append("completed_variants_remain_nonmatch")
        return append_c_recovery_gate_reason(
            {
                "family": "accepted_shape_materialization",
                "family_reasons": sorted(set(reasons)),
            },
            hints,
        )

    go_root = str(first_row.get("goRoot") or first_row.get("goType") or "")
    c_root = str(first_row.get("cRoot") or first_row.get("cType") or "")
    go_errors = first_int(first_row.get("goErrorCount")) or 0
    c_errors = first_int(first_row.get("cErrorCount")) or 0
    err_tree = first_int(measure.get("errTree")) or 0
    go_root_error_excess = go_root == "ERROR" and c_root != "ERROR"
    go_root_err_excess = boolish(first_row.get("goRootErr")) and not boolish(first_row.get("cRootErr"))
    go_error_count_excess = go_errors > c_errors
    if first_row and (go_root_error_excess or go_root_err_excess or go_error_count_excess):
        recovery_reasons = []
        if go_root_error_excess:
            recovery_reasons.append(f"go_error_root_vs_c_{c_root or 'root'}")
        if go_root_err_excess:
            recovery_reasons.append("go_root_error_exceeds_c")
        if err_tree > 0:
            recovery_reasons.append("measure_err_tree")
        if go_error_count_excess:
            recovery_reasons.append("go_error_count_exceeds_c")
        return append_c_recovery_gate_reason(
            {
                "family": "recovery_error_shape",
                "family_reasons": recovery_reasons or ["error_shape_divergence"],
            },
            hints,
        )

    if first_row and boolish(first_row.get("cRootErr")) and c_errors > 0:
        return append_c_recovery_gate_reason(
            {
                "family": "version_or_corpus",
                "family_reasons": ["c_oracle_has_error_tree"],
            },
            hints,
        )

    if terminal.get("fail") or terminal.get("panic") or status == "fail":
        return append_c_recovery_gate_reason(
            {
                "family": "version_or_corpus",
                "family_reasons": ["baseline_fail_or_panic_without_scanner_evidence"],
            },
            hints,
        )

    if first_row:
        return append_c_recovery_gate_reason(
            {
                "family": "version_or_corpus",
                "family_reasons": [f"unclassified_{first_row.get('diff') or 'nonmatch'}_diagnostic"],
            },
            hints,
        )

    return append_c_recovery_gate_reason(
        {
            "family": "version_or_corpus",
            "family_reasons": ["nonmatch_without_comparison_diagnostic"],
        },
        hints,
    )


def planned_action(planned: bool, completed: bool, status: str, reason: str = "") -> dict[str, Any]:
    out = {
        "planned": planned,
        "completed": completed,
        "status": status,
    }
    if reason:
        out["reason"] = reason
    return out


def variant_plan_for_frame(
    *,
    stage_enabled: bool,
    variant_scope: str,
    selector: set[int] | None,
    explicit_selector: bool,
    ordinal: int,
    suspicious: bool,
    suspicious_ordinals_in_cap: set[int],
    max_suspicious: int,
) -> tuple[bool, str]:
    if not stage_enabled:
        return False, "stage_disabled"
    if variant_scope == "suspicious" and not suspicious:
        return False, "no_suspicion"
    if not selector_contains(selector, ordinal):
        return False, "selector"
    if (
        variant_scope == "suspicious"
        and not explicit_selector
        and max_suspicious > 0
        and ordinal not in suspicious_ordinals_in_cap
    ):
        return False, "max_suspicious"
    return True, ""


def firstdiff_plan_for_frame(
    *,
    stage_enabled: bool,
    selector: set[int] | None,
    explicit_selector: bool,
    ordinal: int,
    suspicious: bool,
    suspicious_ordinals_in_cap: set[int],
    max_diag_files: int,
) -> tuple[bool, str]:
    if not stage_enabled:
        return False, "stage_disabled"
    if explicit_selector:
        if not selector_contains(selector, ordinal):
            return False, "selector"
        return True, ""
    if not suspicious:
        return False, "no_suspicion"
    if max_diag_files <= 0 or ordinal not in suspicious_ordinals_in_cap:
        return False, "max_diag_files"
    return True, ""


def build_frame_matrix(wringer_dir: Path, catalog: list[dict[str, Any]] | None = None) -> list[dict[str, Any]]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    config = manifest.get("config", {})
    stage_enabled = config.get("stage_enabled", {})
    variants_stage_enabled = bool(stage_enabled.get("variants", False))
    firstdiff_stage_enabled = bool(stage_enabled.get("firstdiff", False))
    planned_variants = normalized_variants(config)
    variant_scope = str(config.get("variant_scope") or "suspicious")
    variant_selector_text = str(config.get("variant_frames") or "")
    variant_explicit_selector = bool(config.get("variant_frames_explicit")) or bool(variant_selector_text.strip())
    firstdiff_selector_text = str(config.get("firstdiff_frames") or "")
    firstdiff_explicit_selector = bool(config.get("firstdiff_frames_explicit")) or bool(
        firstdiff_selector_text.strip()
    )
    max_diag_files = int(config.get("max_diag_files") or 0)
    max_suspicious = int(config.get("max_suspicious") or 0)

    if catalog is None:
        catalog = build_frame_catalog(wringer_dir)
    variant_selector = resolve_frame_selector(catalog, variant_selector_text, "variant_frames")
    firstdiff_selector = resolve_frame_selector(catalog, firstdiff_selector_text, "firstdiff_frames")
    suspicious_ordinals = [
        int(frame.get("ordinal") or 0)
        for frame in catalog
        if frame.get("suspicious") and int(frame.get("ordinal") or 0) > 0
    ]
    suspicious_ordinals_in_variant_cap = set(suspicious_ordinals[:max_suspicious])
    suspicious_ordinals_in_cap = set(suspicious_ordinals[:max_diag_files])
    variants_by_path = load_variant_outcomes(wringer_dir)
    firstdiff_by_path = load_firstdiff(wringer_dir)
    rows: list[dict[str, Any]] = []
    selected_total = len(catalog)
    for selection_ordinal, frame in enumerate(catalog, start=1):
        path = str(frame.get("path") or "")
        ordinal = int(frame.get("ordinal") or 0)
        suspicious = bool(frame.get("suspicious"))
        variant_planned, variant_not_planned_reason = variant_plan_for_frame(
            stage_enabled=variants_stage_enabled,
            variant_scope=variant_scope,
            selector=variant_selector,
            explicit_selector=variant_explicit_selector,
            ordinal=ordinal,
            suspicious=suspicious,
            suspicious_ordinals_in_cap=suspicious_ordinals_in_variant_cap,
            max_suspicious=max_suspicious,
        )
        variant_outcomes: dict[str, Any] = {}
        planned_variant_actions = 0
        completed_variants = 0
        timeout_variants = 0
        failed_variants = 0
        matched_variants = 0
        nonmatched_variants = 0
        observed_variants = variants_by_path.get(path, {})
        for variant in planned_variants:
            outcome = observed_variants.get(variant)
            if variant_planned:
                planned_variant_actions += 1
            if outcome:
                status = variant_terminal_status(outcome)
                completed = True
                if variant_planned:
                    completed_variants += 1
                if status == "timeout":
                    timeout_variants += 1
                elif status == "fail":
                    failed_variants += 1
                elif status == "match":
                    matched_variants += 1
                elif status == "nonmatch":
                    nonmatched_variants += 1
                variant_outcomes[variant] = planned_action(
                    variant_planned,
                    completed,
                    status,
                    "" if variant_planned else variant_not_planned_reason,
                ) | {
                    "rc": outcome.get("rc"),
                    "timeout": bool(outcome.get("timeout")),
                    "parity_match": bool(outcome.get("parity_match")),
                    "comparison_results": outcome.get("comparison_results", []),
                    "comparison_diagnostic": outcome.get("comparison_diagnostic", {}),
                    "parser_progress": outcome.get("parser_progress", {}),
                    "last_parser_progress": outcome.get("last_parser_progress", {}),
                    "log": outcome.get("log", ""),
                    "host_log": outcome.get("host_log", ""),
                    "replay_command": outcome.get("replay_command", ""),
                    "measure": outcome.get("measure", {}),
                }
                if outcome.get("malformed_reason"):
                    variant_outcomes[variant]["status_detail"] = outcome.get("malformed_reason")
                    variant_outcomes[variant]["reason"] = outcome.get("malformed_reason")
            else:
                status = "not_run" if variant_planned else "not_planned"
                variant_outcomes[variant] = planned_action(
                    variant_planned,
                    False,
                    status,
                    "" if variant_planned else variant_not_planned_reason,
                )

        firstdiff = firstdiff_by_path.get(path)
        firstdiff_planned, firstdiff_not_planned_reason = firstdiff_plan_for_frame(
            stage_enabled=firstdiff_stage_enabled,
            selector=firstdiff_selector,
            explicit_selector=firstdiff_explicit_selector,
            ordinal=ordinal,
            suspicious=suspicious,
            suspicious_ordinals_in_cap=suspicious_ordinals_in_cap,
            max_diag_files=max_diag_files,
        )
        firstdiff_status = terminal_status(firstdiff) if firstdiff else (
            "not_run" if firstdiff_planned else "not_planned"
        )
        firstdiff_outcome: dict[str, Any] = planned_action(
            firstdiff_planned,
            bool(firstdiff),
            firstdiff_status,
            "" if firstdiff_planned else firstdiff_not_planned_reason,
        )
        if firstdiff:
            firstdiff_outcome.update(
                {
                    "rc": firstdiff.get("rc"),
                    "timeout": bool(firstdiff.get("timeout")),
                    "log": firstdiff.get("log", ""),
                    "host_log": firstdiff.get("host_log", ""),
                    "replay_command": firstdiff.get("replay_command", ""),
                }
            )

        baseline_terminal = frame.get("baseline_terminal", {})
        baseline_is_completed = baseline_completed(baseline_terminal)
        baseline_status = baseline_terminal_status(baseline_terminal)
        baseline_outcome = planned_action(True, baseline_is_completed, baseline_status)
        baseline_outcome.update(
            {
                "terminal": baseline_terminal,
                "comparison_diagnostic": baseline_terminal.get("comparison_diagnostic", {}),
                "parser_progress": baseline_terminal.get("parser_progress", {}),
                "last_parser_progress": baseline_terminal.get("last_parser_progress", {}),
                "reasons": frame.get("reasons", []),
                "suspicious": bool(frame.get("suspicious")),
                "log": frame.get("baseline_log", ""),
                "host_log": frame.get("host_baseline_log", ""),
                "replay_command": frame.get("baseline_replay_command", ""),
            }
        )
        matrix_row = {
            "ordinal": frame.get("ordinal"),
            "index": frame.get("index"),
            "selection_ordinal": frame.get("selection_ordinal", selection_ordinal),
            "selected_total": frame.get("selected_total", selected_total),
            "grammar": frame.get("grammar"),
            "corpus_kind": frame.get("corpus_kind"),
            "corpus_root": frame.get("corpus_root"),
            "path": path,
            "base": frame.get("base"),
            "size": frame.get("size"),
            "sha256": frame.get("sha256"),
            "baseline": baseline_outcome,
            "replay_plan": frame.get("replay_plan", {}),
            "variants": variant_outcomes,
            "firstdiff": firstdiff_outcome,
            "stage_completion": {
                "baseline": baseline_is_completed,
                "variants": completed_variants == planned_variant_actions,
                "firstdiff": bool(firstdiff) if firstdiff_planned else True,
            },
            "stage_counts": {
                "baseline_planned": 1,
                "baseline_completed": 1 if baseline_is_completed else 0,
                "configured_variants": len(planned_variants),
                "planned_variants": planned_variant_actions,
                "completed_variants": completed_variants,
                "matched_variants": matched_variants,
                "nonmatched_variants": nonmatched_variants,
                "failed_variants": failed_variants,
                "timeout_variants": timeout_variants,
                "firstdiff_planned": 1 if firstdiff_planned else 0,
                "firstdiff_completed": 1 if firstdiff else 0,
            },
        }
        matrix_row.update(classify_frame_family(matrix_row))
        rows.append(matrix_row)
    return rows


def write_frames(wringer_dir: Path, selected: list[dict[str, Any]], suspicious: list[dict[str, Any]]) -> None:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    suspicious_paths = {item["path"] for item in suspicious}
    rows: list[dict[str, Any]] = []
    used_embedded = False
    for item in selected:
        embedded = item.get("_baseline_frame_events")
        if isinstance(embedded, list):
            rows.extend(embedded)
            used_embedded = True
    if not used_embedded:
        frame_rows = frame_events_by_path(baseline_dir)
        for path in sorted(frame_rows):
            rows.extend(frame_rows[path])
    for item in selected:
        rows.append(
            {
                "source": "wringer",
                "phase": "selected_file",
                "grammar": item.get("grammar"),
                "corpus_kind": item.get("corpus_kind"),
                "ordinal": item.get("ordinal"),
                "selection_ordinal": item.get("selection_ordinal"),
                "total": item.get("total"),
                "selected_total": item.get("selected_total"),
                "path": item.get("path"),
                "base": item.get("base"),
                "suspicious": item.get("path") in suspicious_paths,
                "reasons": item.get("suspicious_reasons", []),
            }
        )
    for meta_by_path in load_variant_outcomes(wringer_dir).values():
        for meta in meta_by_path.values():
            rows.append(
                {
                    "source": "wringer",
                    "phase": "variant",
                    "stage": "variant",
                    "mode": meta.get("variant"),
                    "lifecycle": meta.get("lifecycle", ""),
                    "grammar": meta.get("grammar"),
                    "ordinal": meta.get("ordinal"),
                    "variant": meta.get("variant"),
                    "path": meta.get("path"),
                    "rc": meta.get("rc"),
                    "timeout": meta.get("timeout"),
                    "log": meta.get("log"),
                    "replay_command": meta.get("replay_command", ""),
                    "parity_match": meta.get("parity_match"),
                    "parser_progress": meta.get("parser_progress", {}),
                    "last_parser_progress": meta.get("last_parser_progress", {}),
                }
            )
    for meta in load_firstdiff(wringer_dir).values():
        rows.append(
            {
                "source": "wringer",
                "phase": "firstdiff",
                "stage": "firstdiff",
                "mode": "firstdiff",
                "lifecycle": meta.get("lifecycle", ""),
                "grammar": meta.get("grammar"),
                "ordinal": meta.get("ordinal"),
                "path": meta.get("path"),
                "rc": meta.get("rc"),
                "timeout": meta.get("timeout"),
                "log": meta.get("log"),
                "replay_command": meta.get("replay_command", ""),
            }
        )
    with (wringer_dir / "wringer_frames.jsonl").open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, sort_keys=True) + "\n")


def build_summary(
    wringer_dir: Path,
    *,
    catalog: list[dict[str, Any]] | None = None,
    matrix: list[dict[str, Any]] | None = None,
    plan_rows: list[dict[str, Any]] | None = None,
    plan_summary: dict[str, Any] | None = None,
) -> dict[str, Any]:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    selected, suspicious = load_baseline(wringer_dir)
    variants = load_variant_outcomes(wringer_dir)
    firstdiff = load_firstdiff(wringer_dir)
    events = load_wringer_events(wringer_dir)
    if catalog is None:
        catalog = build_frame_catalog(wringer_dir)
    if matrix is None:
        matrix = build_frame_matrix(wringer_dir, catalog)
    if plan_rows is None or plan_summary is None:
        built_rows, built_summary = build_plan(wringer_dir, matrix)
        if plan_rows is None:
            plan_rows = built_rows
        if plan_summary is None:
            plan_summary = built_summary
    family_counts: dict[str, int] = {}
    family_frames: dict[str, list[dict[str, Any]]] = {}
    for row in matrix:
        family = str(row.get("family") or "version_or_corpus")
        if family == NO_EVIDENCE_FAMILY:
            continue
        family_counts[family] = family_counts.get(family, 0) + 1
        family_frames.setdefault(family, []).append(
            {
                "ordinal": row.get("ordinal"),
                "path": row.get("path"),
                "base": row.get("base"),
                "status": (row.get("baseline") or {}).get("status") if isinstance(row.get("baseline"), dict) else "",
                "family_reasons": row.get("family_reasons", []),
            }
        )
    baseline_dir_recorded = Path(manifest.get("baseline_run_dir", wringer_dir / "baseline"))
    baseline_dir = resolve_recorded_path(baseline_dir_recorded, wringer_dir, manifest)
    counts = {
        "selected_files": len(selected),
        "suspicious_files": len(suspicious),
        "variant_files": len(variants),
        "firstdiff_files": len(firstdiff),
        "families": family_counts,
    }
    counts.update(event_counts(events))
    summary = {
        "grammar": manifest.get("grammar", ""),
        "out_dir": str(wringer_dir),
        "baseline_run_dir": str(baseline_dir_recorded),
        "baseline_exit_status": manifest.get("baseline_exit_status"),
        "infra_status": manifest.get("infra_status", 0),
        "selected_files": [public_frame_record(item) for item in selected],
        "suspicious_files": [public_frame_record(item) for item in suspicious],
        "variant_outcomes": variants,
        "firstdiff": firstdiff,
        "family_counts": family_counts,
        "family_frames": family_frames,
        "plan": plan_summary,
        "control": build_control_contract(
            wringer_dir,
            matrix=matrix,
            plan_rows=plan_rows,
            plan_summary=plan_summary,
        ),
        "counts": counts,
        "artifacts": {
            "frame_catalog_jsonl": str(wringer_dir / "frame_catalog.jsonl"),
            "frame_matrix_jsonl": str(wringer_dir / "frame_matrix.jsonl"),
            "wringer_plan_jsonl": str(wringer_dir / "wringer_plan.jsonl"),
            "wringer_plan_json": str(wringer_dir / "wringer_plan.json"),
            "wringer_manifest_json": str(wringer_dir / "wringer_manifest.json"),
            "wringer_active_txt": str(wringer_dir / "wringer_active.txt"),
            "wringer_active_json": str(wringer_dir / "wringer_active.json"),
            "wringer_events_jsonl": str(wringer_dir / "wringer_events.jsonl"),
            "wringer_frames_jsonl": str(wringer_dir / "wringer_frames.jsonl"),
            "wringer_summary_json": str(wringer_dir / "wringer_summary.json"),
            "wringer_summary_md": str(wringer_dir / "wringer_summary.md"),
            "commands_log": str(wringer_dir / "commands.log"),
            "baseline_manifest_json": str(baseline_dir_recorded / "manifest.json"),
            "baseline_frames_jsonl": str(baseline_dir_recorded / "frames.jsonl"),
        },
    }
    host_baseline_run_dir = host_field(baseline_dir_recorded, baseline_dir)
    if host_baseline_run_dir:
        summary["host_baseline_run_dir"] = host_baseline_run_dir
    return summary


def write_markdown(summary: dict[str, Any], path: Path) -> None:
    lines = [
        f"# Grammar integrity wringer: {summary.get('grammar', '')}",
        "",
        f"- Baseline: `{summary.get('baseline_run_dir', '')}`",
        f"- Selected files: {summary['counts']['selected_files']}",
        f"- Suspicious files: {summary['counts']['suspicious_files']}",
        f"- Variant files: {summary['counts']['variant_files']}",
        f"- First-diff files: {summary['counts']['firstdiff_files']}",
        f"- Family counts: `{json.dumps(summary.get('family_counts', {}), sort_keys=True)}`",
        f"- Frame matrix: `{summary.get('artifacts', {}).get('frame_matrix_jsonl', '')}`",
        f"- Wringer plan: `{summary.get('artifacts', {}).get('wringer_plan_jsonl', '')}`",
        f"- Wringer events: {summary['counts'].get('events', 0)}",
        f"- Infra status: {summary.get('infra_status', 0)}",
        "",
        "## Control contract",
        "",
    ]
    control = summary.get("control", {}) if isinstance(summary.get("control"), dict) else {}
    action_counts = control.get("action_counts", {}) if isinstance(control.get("action_counts"), dict) else {}
    overall = action_counts.get("overall", {}) if isinstance(action_counts.get("overall"), dict) else {}
    selected_frames = control.get("selected_frames", {}) if isinstance(control.get("selected_frames"), dict) else {}
    baseline_balance = (
        control.get("baseline_frame_lifecycle_balance", {})
        if isinstance(control.get("baseline_frame_lifecycle_balance"), dict)
        else {}
    )
    lines.extend(
        [
            f"- Status: `{control.get('status', 'unknown')}`",
            f"- Selected frames: {selected_frames.get('count', 0)}",
            f"- Planned actions: {overall.get('planned', 0)}",
            f"- Completed actions: {overall.get('completed', 0)}",
            f"- Incomplete actions: {overall.get('incomplete', 0)}",
            f"- Open wringer actions: {len(control.get('open_actions', [])) if isinstance(control.get('open_actions'), list) else 0}",
            f"- Open baseline frames: {baseline_balance.get('open', 0)}",
            "",
        ]
    )
    incomplete = control.get("incomplete_actions", []) if isinstance(control.get("incomplete_actions"), list) else []
    if incomplete:
        lines.extend(["### Incomplete planned actions", ""])
        for item in incomplete[:20]:
            lines.append(
                f"- `{item.get('stage', '')}:{item.get('mode', '')}` "
                f"ordinal={item.get('ordinal', '')} status={item.get('status', '')} "
                f"log `{item.get('log', '')}`"
            )
        lines.append("")
    next_files = control.get("next_files", []) if isinstance(control.get("next_files"), list) else []
    if next_files:
        lines.extend(["### Inspect next", ""])
        for item in next_files:
            lines.append(f"- `{item}`")
        lines.append("")
    lines.extend(["## Failure families", ""])
    family_frames = summary.get("family_frames", {}) if isinstance(summary.get("family_frames"), dict) else {}
    if not family_frames:
        lines.append("_None._")
    else:
        for family, frames in sorted(family_frames.items()):
            if not isinstance(frames, list):
                continue
            lines.append(f"- `{family}`: {len(frames)}")
            for item in frames[:10]:
                reasons = ", ".join(str(reason) for reason in item.get("family_reasons", []))
                lines.append(
                    f"  - ordinal={item.get('ordinal', '')} `{item.get('base') or item.get('path', '')}`: {reasons}"
                )
            if len(frames) > 10:
                lines.append(f"  - ... {len(frames) - 10} more")
    lines.append("")
    lines.extend([
        "## Suspicious files",
        "",
    ])
    suspicious = summary.get("suspicious_files", [])
    if not suspicious:
        lines.append("_None._")
    else:
        for item in suspicious:
            lines.append(f"- `{item['path']}`: {', '.join(item.get('reasons', []))}")
    lines.extend(["", "## Variant outcomes", ""])
    variants = summary.get("variant_outcomes", {})
    if not variants:
        lines.append("_None._")
    else:
        for file_path, by_variant in sorted(variants.items()):
            lines.append(f"- `{file_path}`")
            for variant, outcome in sorted(by_variant.items()):
                status = "timeout" if outcome.get("timeout") else f"rc={outcome.get('rc')}"
                parity = "match" if outcome.get("parity_match") else "nonmatch"
                lines.append(f"  - `{variant}`: {status}, {parity}, log `{outcome.get('log', '')}`")
    lines.extend(["", "## First diff", ""])
    firstdiff = summary.get("firstdiff", {})
    if not firstdiff:
        lines.append("_None._")
    else:
        for file_path, outcome in sorted(firstdiff.items()):
            status = "timeout" if outcome.get("timeout") else f"rc={outcome.get('rc')}"
            lines.append(f"- `{file_path}`: {status}, log `{outcome.get('log', '')}`")
    lines.extend(["", "## Artifacts", ""])
    for name, artifact in sorted(summary.get("artifacts", {}).items()):
        lines.append(f"- `{name}`: `{artifact}`")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_summary(wringer_dir: Path) -> None:
    refresh_manifest_artifacts(wringer_dir)
    selected, suspicious = load_baseline(wringer_dir)
    catalog = build_frame_catalog(wringer_dir)
    matrix = build_frame_matrix(wringer_dir, catalog)
    plan_rows, plan_summary = build_plan(wringer_dir, matrix)
    summary = build_summary(
        wringer_dir,
        catalog=catalog,
        matrix=matrix,
        plan_rows=plan_rows,
        plan_summary=plan_summary,
    )
    write_frames(wringer_dir, selected, suspicious)
    write_frame_catalog(wringer_dir, catalog)
    write_frame_matrix(wringer_dir, matrix)
    write_plan(wringer_dir, plan_rows, plan_summary)
    (wringer_dir / "wringer_summary.json").write_text(
        json.dumps(summary, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    write_markdown(summary, wringer_dir / "wringer_summary.md")


def write_frame_catalog(wringer_dir: Path, rows: list[dict[str, Any]] | None = None) -> None:
    if rows is None:
        rows = build_frame_catalog(wringer_dir)
    with (wringer_dir / "frame_catalog.jsonl").open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, sort_keys=True) + "\n")


def write_frame_matrix(wringer_dir: Path, rows: list[dict[str, Any]] | None = None) -> None:
    if rows is None:
        rows = build_frame_matrix(wringer_dir)
    with (wringer_dir / "frame_matrix.jsonl").open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, sort_keys=True) + "\n")


def refresh_manifest_artifacts(wringer_dir: Path) -> None:
    path = wringer_dir / "wringer_manifest.json"
    manifest = read_json(path, {})
    if not isinstance(manifest, dict):
        return
    artifacts = manifest.setdefault("artifacts", {})
    if not isinstance(artifacts, dict):
        artifacts = {}
        manifest["artifacts"] = artifacts
    artifact_paths = {
        "wringer_manifest_json": path,
        "wringer_active_txt": wringer_dir / "wringer_active.txt",
        "wringer_active_json": wringer_dir / "wringer_active.json",
        "wringer_events_jsonl": wringer_dir / "wringer_events.jsonl",
        "frame_catalog_jsonl": wringer_dir / "frame_catalog.jsonl",
        "frame_matrix_jsonl": wringer_dir / "frame_matrix.jsonl",
        "wringer_plan_jsonl": wringer_dir / "wringer_plan.jsonl",
        "wringer_plan_json": wringer_dir / "wringer_plan.json",
        "wringer_frames_jsonl": wringer_dir / "wringer_frames.jsonl",
        "wringer_summary_json": wringer_dir / "wringer_summary.json",
        "wringer_summary_md": wringer_dir / "wringer_summary.md",
        "commands_log": wringer_dir / "commands.log",
    }
    for name, artifact_path in artifact_paths.items():
        artifacts[name] = str(artifact_path)
    path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def write_plan(
    wringer_dir: Path,
    rows: list[dict[str, Any]] | None = None,
    summary: dict[str, Any] | None = None,
) -> None:
    if rows is None or summary is None:
        built_rows, built_summary = build_plan(wringer_dir)
        if rows is None:
            rows = built_rows
        if summary is None:
            summary = built_summary
    with (wringer_dir / "wringer_plan.jsonl").open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, sort_keys=True) + "\n")
    (wringer_dir / "wringer_plan.json").write_text(
        json.dumps(summary, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def emit_frame_catalog_jsonl(wringer_dir: Path, frames: str | None = None) -> None:
    for row in filter_rows_by_frames(build_frame_catalog(wringer_dir), frames):
        print(json.dumps(row, sort_keys=True))


def emit_frame_matrix_jsonl(wringer_dir: Path, frames: str | None = None) -> None:
    for row in filter_rows_by_frames(build_frame_matrix(wringer_dir), frames):
        print(json.dumps(row, sort_keys=True))


def emit_plan_jsonl(wringer_dir: Path, frames: str | None = None) -> None:
    rows, _ = build_plan(wringer_dir)
    for row in filter_rows_by_frames(rows, frames):
        print(json.dumps(row, sort_keys=True))


def print_status(wringer_dir: Path) -> None:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    active = read_active(wringer_dir / "wringer_active.txt")
    baseline_active = read_active(baseline_dir / "active_grammar.txt")
    summary = read_json(wringer_dir / "wringer_summary.json", {})
    counts = summary.get("counts", {}) if isinstance(summary, dict) else {}
    events = load_wringer_events(wringer_dir)
    balance = lifecycle_balance(events)
    baseline_balance = baseline_frame_lifecycle_balance(wringer_dir)
    print(f"wringer_dir={wringer_dir}")
    if active:
        print(
            "wringer_active "
            f"state={active.get('state', '')} "
            f"stage={active.get('stage', '')} "
            f"mode={active.get('mode', '')} "
            f"ordinal={active.get('ordinal', '')} "
            f"path={active.get('path', '')} "
            f"log={active.get('log', '')}"
        )
    else:
        print("wringer_active missing")
    if baseline_active:
        print(
            "baseline_active "
            f"state={baseline_active.get('state', '')} "
            f"grammar={baseline_active.get('grammar', '')} "
            f"corpus={baseline_active.get('corpus_kind', '')} "
            f"detail={baseline_active.get('detail', '')}"
        )
    else:
        print(f"baseline_active missing ({baseline_dir / 'active_grammar.txt'})")
    if counts:
        print(
            "counts "
            f"selected={counts.get('selected_files', 0)} "
            f"suspicious={counts.get('suspicious_files', 0)} "
            f"variant_files={counts.get('variant_files', 0)} "
            f"firstdiff_files={counts.get('firstdiff_files', 0)} "
            f"events={counts.get('events', 0)}"
        )
    print(
        "lifecycle "
        f"starts={balance.get('starts', 0)} "
        f"terminals={balance.get('terminals', 0)} "
        f"open={balance.get('open', 0)}"
    )
    for frame in balance.get("open_frames", [])[:5]:
        print(
            "open_frame "
            f"stage={frame.get('stage', '')} "
            f"variant={frame.get('variant', '')} "
            f"ordinal={frame.get('ordinal', '')} "
            f"path={frame.get('path', '')} "
            f"log={frame.get('log', '')}"
        )
    print(
        "baseline_frame_lifecycle "
        f"starts={baseline_balance.get('starts', 0)} "
        f"terminals={baseline_balance.get('terminals', 0)} "
        f"heartbeats={baseline_balance.get('heartbeats', 0)} "
        f"open={baseline_balance.get('open', 0)}"
    )
    for frame in baseline_balance.get("open_frames", [])[:5]:
        heartbeat = frame.get("last_heartbeat") if isinstance(frame.get("last_heartbeat"), dict) else {}
        print(
            "baseline_open_frame "
            f"ordinal={frame.get('ordinal', '')} "
            f"file={frame.get('file', '')} "
            f"log={frame.get('log', '')} "
            f"last_heartbeat={heartbeat.get('ts', '')}"
        )
    print(f"baseline_progress={baseline_dir / 'progress.log'}")
    print(f"baseline_status={baseline_dir / 'status.tsv'}")


def print_control_status(wringer_dir: Path, frames: str | None = None) -> None:
    control = build_control_contract(wringer_dir, frames)
    active = control.get("active", {}) if isinstance(control.get("active"), dict) else {}
    wringer_active = active.get("wringer", {}) if isinstance(active.get("wringer"), dict) else {}
    baseline_active = active.get("baseline", {}) if isinstance(active.get("baseline"), dict) else {}
    selected = control.get("selected_frames", {}) if isinstance(control.get("selected_frames"), dict) else {}
    action_counts = control.get("action_counts", {}) if isinstance(control.get("action_counts"), dict) else {}
    by_stage = action_counts.get("by_stage", {}) if isinstance(action_counts.get("by_stage"), dict) else {}
    overall = action_counts.get("overall", {}) if isinstance(action_counts.get("overall"), dict) else {}
    wringer_balance = control.get("wringer_lifecycle_balance", {}) if isinstance(control.get("wringer_lifecycle_balance"), dict) else {}
    baseline_balance = (
        control.get("baseline_frame_lifecycle_balance", {})
        if isinstance(control.get("baseline_frame_lifecycle_balance"), dict)
        else {}
    )
    print(f"wringer_dir={wringer_dir}")
    print(
        "control "
        f"status={control.get('status', '')} "
        f"grammar={control.get('grammar', '')} "
        f"mode={control.get('mode', '')} "
        f"profile={control.get('profile', '')} "
        f"full_integrity={str(bool(control.get('full_integrity'))).lower()}"
    )
    print(
        "active "
        f"state={wringer_active.get('state', '')} "
        f"stage={wringer_active.get('stage', '')} "
        f"mode={wringer_active.get('mode', '')} "
        f"variant={wringer_active.get('variant', '')} "
        f"ordinal={wringer_active.get('ordinal', '')} "
        f"action={wringer_active.get('action', '')} "
        f"path={shlex.quote(str(wringer_active.get('path', '') or ''))} "
        f"log={shlex.quote(str(wringer_active.get('log', '') or ''))}"
    )
    print(
        "baseline_active "
        f"state={baseline_active.get('state', '')} "
        f"grammar={baseline_active.get('grammar', '')} "
        f"corpus={baseline_active.get('corpus_kind', '')} "
        f"detail={shlex.quote(str(baseline_active.get('detail', '') or ''))}"
    )
    print(
        "selected_frames "
        f"count={selected.get('count', 0)} "
        f"ordinals={','.join(str(value) for value in selected.get('ordinals', [])) if selected.get('ordinals') else 'none'}"
    )
    print(
        "actions_overall "
        f"planned={overall.get('planned', 0)} "
        f"completed={overall.get('completed', 0)} "
        f"incomplete={overall.get('incomplete', 0)} "
        f"unexpected={overall.get('unexpected', 0)}"
    )
    for stage in ("baseline", "variant", "firstdiff"):
        counts = by_stage.get(stage, {}) if isinstance(by_stage.get(stage), dict) else {}
        print(
            "actions_by_stage "
            f"stage={stage} "
            f"planned={counts.get('planned', 0)} "
            f"completed={counts.get('completed', 0)} "
            f"incomplete={counts.get('incomplete', 0)} "
            f"unexpected={counts.get('unexpected', 0)}"
        )
    print(
        "open_actions "
        f"starts={wringer_balance.get('starts', 0)} "
        f"terminals={wringer_balance.get('terminals', 0)} "
        f"open={wringer_balance.get('open', 0)} "
        f"stale={len(control.get('stale_actions', [])) if isinstance(control.get('stale_actions'), list) else 0}"
    )
    for frame in control.get("open_actions", [])[:5]:
        print(
            "open_action "
            f"stage={frame.get('stage', '')} "
            f"mode={frame.get('mode', '')} "
            f"variant={frame.get('variant', '')} "
            f"ordinal={frame.get('ordinal', '')} "
            f"age_s={frame.get('age_s', '')} "
            f"stale={str(bool(frame.get('stale'))).lower()} "
            f"log_age_s={frame.get('log_age_s', '')} "
            f"progress_age_s={frame.get('progress_age_s', '')} "
            f"path={shlex.quote(str(frame.get('path', '') or ''))} "
            f"log={shlex.quote(str(frame.get('log', '') or ''))}"
        )
    print(
        "baseline_frame_lifecycle "
        f"starts={baseline_balance.get('starts', 0)} "
        f"terminals={baseline_balance.get('terminals', 0)} "
        f"heartbeats={baseline_balance.get('heartbeats', 0)} "
        f"open={baseline_balance.get('open', 0)} "
        f"stale={len(baseline_balance.get('stale_frames', [])) if isinstance(baseline_balance.get('stale_frames'), list) else 0}"
    )
    for frame in baseline_balance.get("open_frames", [])[:5]:
        print(
            "baseline_open_frame "
            f"ordinal={frame.get('ordinal', '')} "
            f"age_s={frame.get('age_s', '')} "
            f"stale={str(bool(frame.get('stale'))).lower()} "
            f"log_age_s={frame.get('log_age_s', '')} "
            f"progress_age_s={frame.get('progress_age_s', '')} "
            f"file={shlex.quote(str(frame.get('file', '') or frame.get('path', '') or ''))} "
            f"log={shlex.quote(str(frame.get('log', '') or ''))}"
        )
    for item in control.get("incomplete_actions", [])[:20]:
        print(
            "incomplete_action "
            f"stage={item.get('stage', '')} "
            f"mode={item.get('mode', '')} "
            f"variant={item.get('variant', '')} "
            f"ordinal={item.get('ordinal', '')} "
            f"status={item.get('status', '')} "
            f"terminal_missing={str(bool(item.get('terminal_missing'))).lower()} "
            f"path={shlex.quote(str(item.get('path', '') or ''))} "
            f"log={shlex.quote(str(item.get('host_log') or item.get('log') or ''))}"
        )
    for item in control.get("baseline_row_contract_errors", []):
        print(f"baseline_contract_gap {item}")
    for action in control.get("unexpected_actions", []):
        print(
            "unexpected_action "
            f"stage={action.get('stage', '')} "
            f"variant={action.get('variant', '')} "
            f"ordinal={action.get('ordinal', '')} "
            f"path={shlex.quote(str(action.get('path', '') or ''))} "
            f"log={shlex.quote(str(action.get('log', '') or ''))}"
        )
    for action in control.get("unexpected_lifecycle_actions", []):
        print(
            "unexpected_lifecycle_action "
            f"stage={action.get('stage', '')} "
            f"mode={action.get('mode', '')} "
            f"ordinal={action.get('ordinal', '')} "
            f"path={shlex.quote(str(action.get('path', '') or ''))}"
        )
    for item in control.get("next_files", []):
        print(f"inspect_next={item}")
    frame_arg = f" --frames {shlex.quote(frames)}" if frames else ""
    print(f"control_json=python3 cgo_harness/tier_scan/wringer_summary.py {wringer_dir}{frame_arg} --emit-control-json")
    print(f"replay_commands=python3 cgo_harness/tier_scan/wringer_summary.py {wringer_dir}{frame_arg} --emit-replay-commands all")


def emit_control_json(wringer_dir: Path, frames: str | None = None) -> None:
    print(json.dumps(build_control_contract(wringer_dir, frames), sort_keys=True))


def print_frame_control(wringer_dir: Path, frames: str | None = None) -> None:
    for row in filter_rows_by_frames(build_frame_matrix(wringer_dir), frames):
        baseline = row.get("baseline") or {}
        variants = row.get("variants") or {}
        firstdiff = row.get("firstdiff") or {}
        counts = row.get("stage_counts") or {}
        plan = row.get("replay_plan") or {}
        direct_variants = plan.get("direct_variant_commands") if isinstance(plan.get("direct_variant_commands"), dict) else {}
        reasons = baseline.get("reasons") or []
        sha = str(row.get("sha256") or "")
        replay_available = [
            name
            for name in (
                "baseline_command",
                "direct_baseline_command",
                "baseline_reuse_variant_command",
                "baseline_reuse_firstdiff_command",
                "direct_firstdiff_command",
            )
            if plan.get(name)
        ]
        if direct_variants:
            replay_available.append("direct_variant_commands")
        variant_status = ",".join(
            f"{variant}:{action.get('status', '')}"
            for variant, action in sorted(variants.items())
        )
        print(
            "frame_control "
            f"ordinal={row.get('ordinal', '')} "
            f"base={shlex.quote(str(row.get('base') or ''))} "
            f"sha256={sha[:12]} "
            f"path={shlex.quote(str(row.get('path') or ''))} "
            f"suspicious={str(bool(baseline.get('suspicious'))).lower()} "
            f"reasons={shlex.quote(','.join(str(reason) for reason in reasons) or '-')} "
            f"baseline={baseline.get('status', '')} "
            f"variants={counts.get('completed_variants', 0)}/{counts.get('planned_variants', 0)} "
            f"variant_status={shlex.quote(variant_status or '-')} "
            f"firstdiff={firstdiff.get('status', '')} "
            f"firstdiff_completed={str(bool(firstdiff.get('completed'))).lower()} "
            f"replay={shlex.quote(','.join(replay_available) or 'none')}"
        )


def print_residual_frames(wringer_dir: Path, frames: str | None = None) -> None:
    for row in filter_rows_by_frames(build_frame_matrix(wringer_dir), frames):
        baseline = row.get("baseline") or {}
        status = str(baseline.get("status") or "")
        if status == "match":
            continue
        terminal = baseline.get("terminal") if isinstance(baseline.get("terminal"), dict) else {}
        diag_rows = diagnostic_rows(terminal)
        first_row = diag_rows[0] if diag_rows else {}
        runtime_hints = runtime_hints_with_diagnostic_fallback(terminal, first_row)
        measure = terminal.get("measure") if isinstance(terminal.get("measure"), dict) else {}
        parser_progress = terminal.get("parser_progress") if isinstance(terminal.get("parser_progress"), dict) else {}
        last_parser = terminal.get("last_parser_progress") if isinstance(terminal.get("last_parser_progress"), dict) else {}
        if not last_parser and isinstance(parser_progress.get("last"), dict):
            last_parser = parser_progress.get("last") or {}
        terminal_event = terminal.get("terminal_event") if isinstance(terminal.get("terminal_event"), dict) else {}
        reasons = list(baseline.get("reasons") or [])
        gate_reason = c_recovery_gate_reason(runtime_hints)
        if gate_reason and gate_reason not in reasons:
            reasons.append(gate_reason)
        sha = str(row.get("sha256") or "")
        go_stop = (
            runtime_hints.get("stopReason")
            or runtime_hints.get("goStop")
            or measure.get("goStop")
            or measure.get("stopReason")
            or ""
        )
        truncation = (
            runtime_hints.get("truncated")
            or runtime_hints.get("goTruncated")
            or measure.get("truncated")
            or measure.get("goTruncated")
            or ""
        )
        last_phase = (
            terminal.get("last_phase")
            or last_parser.get("parser_phase")
            or terminal_event.get("phase")
            or ""
        )
        last_lifecycle = (
            terminal.get("last_lifecycle")
            or terminal_event.get("lifecycle")
            or terminal_event.get("event")
            or ""
        )
        print(
            "residual_frame "
            f"ordinal={row.get('ordinal', '')} "
            f"status={status} "
            f"base={shlex.quote(str(row.get('base') or ''))} "
            f"sha256={sha[:12]} "
            f"path={shlex.quote(str(row.get('path') or ''))} "
            f"reasons={shlex.quote(','.join(str(reason) for reason in reasons) or '-')} "
            f"goStop={shlex.quote(str(go_stop))} "
            f"truncated={shlex.quote(str(truncation))} "
            f"last_phase={shlex.quote(str(last_phase))} "
            f"last_lifecycle={shlex.quote(str(last_lifecycle))} "
            f"log={shlex.quote(str(baseline.get('host_log') or baseline.get('log') or ''))}"
        )


def emit_active_json(wringer_dir: Path) -> None:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    events = load_wringer_events(wringer_dir)
    payload = {
        "wringer_dir": str(wringer_dir),
        "wringer_active": read_active(wringer_dir / "wringer_active.txt"),
        "baseline_active": read_active(baseline_dir / "active_grammar.txt"),
        "lifecycle_balance": lifecycle_balance(events),
        "baseline_frame_lifecycle_balance": baseline_frame_lifecycle_balance(wringer_dir),
        "control": build_control_contract(wringer_dir),
        "artifacts": {
            "wringer_active_txt": str(wringer_dir / "wringer_active.txt"),
            "wringer_active_json": str(wringer_dir / "wringer_active.json"),
            "wringer_events_jsonl": str(wringer_dir / "wringer_events.jsonl"),
            "baseline_active_txt": str(baseline_dir / "active_grammar.txt"),
            "baseline_progress": str(baseline_dir / "progress.log"),
            "baseline_status": str(baseline_dir / "status.tsv"),
        },
    }
    print(json.dumps(payload, sort_keys=True))


def emit_replay_commands(wringer_dir: Path, scope: str, frames: str | None = None) -> None:
    for row in filter_rows_by_frames(build_frame_catalog(wringer_dir), frames):
        ordinal = row.get("ordinal", "")
        path = row.get("path", "")
        plan = row.get("replay_plan", {})
        if scope in {"all", "baseline"}:
            for name in ("baseline_command", "direct_baseline_command"):
                command = str(plan.get(name) or "")
                if command:
                    print(f"{ordinal}\t{name}\t{path}\t{command}")
        if scope in {"all", "variants"}:
            command = str(plan.get("baseline_reuse_variant_command") or "")
            if command:
                print(f"{ordinal}\tbaseline_reuse_variant_command\t{path}\t{command}")
            direct = plan.get("direct_variant_commands", {})
            if isinstance(direct, dict):
                for variant, command in sorted(direct.items()):
                    if command:
                        print(f"{ordinal}\tdirect_variant:{variant}\t{path}\t{command}")
        if scope in {"all", "firstdiff"}:
            for name in ("baseline_reuse_firstdiff_command", "direct_firstdiff_command"):
                command = str(plan.get(name) or "")
                if command:
                    print(f"{ordinal}\t{name}\t{path}\t{command}")


def assert_closed(wringer_dir: Path, frames: str | None = None) -> int:
    control = build_control_contract(wringer_dir, frames)
    balance = control.get("wringer_lifecycle_balance", {})
    open_frames = balance.get("open_frames", []) if isinstance(balance, dict) else []
    baseline_balance = control.get("baseline_frame_lifecycle_balance", {})
    baseline_open = baseline_balance.get("open_frames", []) if isinstance(baseline_balance, dict) else []
    if not open_frames and not baseline_open:
        return 0
    for frame in open_frames:
        print(
            "open child "
            f"stage={frame.get('stage', '')} "
            f"variant={frame.get('variant', '')} "
            f"ordinal={frame.get('ordinal', '')} "
            f"path={frame.get('path', '')} "
            f"log={frame.get('log', '')}",
            file=sys.stderr,
        )
    for frame in baseline_open:
        print(
            "open baseline frame "
            f"ordinal={frame.get('ordinal', '')} "
            f"file={frame.get('file', '')} "
            f"log={frame.get('log', '')}",
            file=sys.stderr,
        )
    return 1


def assert_planned_complete(wringer_dir: Path, frames: str | None = None) -> int:
    matrix = filter_rows_by_frames(build_frame_matrix(wringer_dir), frames)
    incomplete = incomplete_planned_actions(matrix)
    if not incomplete:
        return 0
    for item in incomplete:
        print(f"incomplete planned action {item}", file=sys.stderr)
    return 1


def filter_unexpected_actions_by_frames(
    unexpected: list[dict[str, Any]],
    matrix: list[dict[str, Any]],
    frames: str | None,
) -> list[dict[str, Any]]:
    if not frames:
        return unexpected
    filtered_matrix = filter_rows_by_frames(matrix, frames)
    selected_paths = {str(row.get("path") or "") for row in filtered_matrix}
    selected_ordinals = {
        int(row.get("ordinal") or 0)
        for row in filtered_matrix
        if int(row.get("ordinal") or 0) > 0
    }
    out = []
    for action in unexpected:
        path = str(action.get("path") or "")
        try:
            ordinal = int(action.get("ordinal") or 0)
        except (TypeError, ValueError):
            ordinal = 0
        if path in selected_paths or ordinal in selected_ordinals:
            out.append(action)
    return out


def assert_plan_exact(wringer_dir: Path, frames: str | None = None) -> int:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    matrix = build_frame_matrix(wringer_dir)
    filtered_matrix = filter_rows_by_frames(matrix, frames)
    rows, summary = build_plan(wringer_dir)
    filtered_rows = filter_rows_by_frames(rows, frames)
    incomplete = incomplete_planned_actions(filtered_matrix)
    full_integrity_gaps = full_integrity_plan_gaps(filtered_matrix, manifest)
    baseline_errors = baseline_row_contract_errors(filtered_matrix, filtered_rows)
    unexpected = filter_unexpected_actions_by_frames(
        summary.get("unexpected_actions", []),
        matrix,
        frames,
    )
    unexpected_events = unexpected_lifecycle_actions(
        filtered_matrix,
        load_wringer_events(wringer_dir),
        manifest,
        scoped=bool(frames and str(frames).strip() not in {"", "all", "*"}),
    )
    rc = 0
    for item in baseline_errors:
        print(f"baseline contract failure {item}", file=sys.stderr)
        rc = 1
    for item in full_integrity_gaps:
        print(f"full-integrity missing planned action {item}", file=sys.stderr)
        rc = 1
    for item in incomplete:
        print(f"incomplete planned action {item}", file=sys.stderr)
        rc = 1
    for action in unexpected:
        print(
            "unexpected observed action "
            f"stage={action.get('stage', '')} "
            f"variant={action.get('variant', '')} "
            f"ordinal={action.get('ordinal', '')} "
            f"path={action.get('path', '')} "
            f"log={action.get('log', '')}",
            file=sys.stderr,
        )
        rc = 1
    for action in unexpected_events:
        print(
            "unexpected lifecycle action "
            f"stage={action.get('stage', '')} "
            f"mode={action.get('mode', '')} "
            f"variant={action.get('variant', '')} "
            f"ordinal={action.get('ordinal', '')} "
            f"path={action.get('path', '')}",
            file=sys.stderr,
        )
        rc = 1
    return rc


def telemetry_complete(item: dict[str, Any] | None) -> bool:
    if not item:
        return False
    return bool(item.get("has_start_event") and item.get("has_terminal_event") and item.get("has_duration"))


def assert_telemetry_complete(wringer_dir: Path, frames: str | None = None) -> int:
    manifest = read_json(wringer_dir / "wringer_manifest.json", {})
    stage_enabled = (manifest.get("config", {}).get("stage_enabled") or {})
    telemetry = event_telemetry_by_action(load_wringer_events(wringer_dir))
    baseline_frame_telemetry = baseline_frame_telemetry_by_ordinal(wringer_dir)
    rc = 0

    if bool(stage_enabled.get("baseline", False)):
        if baseline_frame_telemetry:
            rows, _ = build_plan(wringer_dir)
            for row in filter_rows_by_frames(rows, frames):
                if str(row.get("stage") or "") != "baseline":
                    continue
                if not row.get("planned"):
                    continue
                evidence = row.get("event_evidence") if isinstance(row.get("event_evidence"), dict) else {}
                if not telemetry_complete(evidence):
                    print(
                        "incomplete telemetry "
                        "stage=baseline "
                        "mode=baseline "
                        f"ordinal={row.get('ordinal', '')} "
                        f"path={row.get('path', '')} "
                        "event_scope=baseline_frame "
                        "reason=missing_start_terminal_or_duration",
                        file=sys.stderr,
                    )
                    rc = 1
        else:
            baseline = telemetry.get(("baseline", "tier_scan", 0, ""))
            if not telemetry_complete(baseline):
                print(
                    "incomplete telemetry stage=baseline mode=tier_scan ordinal=0 "
                    "event_scope=delegated_baseline "
                    "reason=missing_start_terminal_or_duration",
                    file=sys.stderr,
                )
                rc = 1

    rows, _ = build_plan(wringer_dir)
    for row in filter_rows_by_frames(rows, frames):
        stage = str(row.get("stage") or "")
        if stage not in {"variant", "firstdiff"}:
            continue
        if not row.get("planned"):
            continue
        evidence = row.get("event_evidence") if isinstance(row.get("event_evidence"), dict) else {}
        if not telemetry_complete(evidence):
            print(
                "incomplete telemetry "
                f"stage={stage} "
                f"mode={row.get('mode', '')} "
                f"variant={row.get('variant', '')} "
                f"ordinal={row.get('ordinal', '')} "
                f"path={row.get('path', '')} "
                "reason=missing_start_terminal_or_duration",
                file=sys.stderr,
            )
            rc = 1
    return rc


def strict_load_json(path: Path, label: str) -> tuple[Any, str]:
    if not path.exists() or path.stat().st_size == 0:
        return None, f"missing or empty {label}: {path}"
    try:
        return json.loads(path.read_text(encoding="utf-8")), ""
    except Exception as exc:
        return None, f"corrupt {label}: {path}: {exc}"


def strict_load_jsonl(path: Path, label: str) -> tuple[list[dict[str, Any]], list[str]]:
    errors: list[str] = []
    rows: list[dict[str, Any]] = []
    if not path.exists() or path.stat().st_size == 0:
        return rows, [f"missing or empty {label}: {path}"]
    for lineno, line in enumerate(path.read_text(encoding="utf-8", errors="replace").splitlines(), start=1):
        if not line.strip():
            continue
        try:
            parsed = json.loads(line)
        except json.JSONDecodeError as exc:
            errors.append(f"corrupt {label} line {lineno}: {path}: {exc}")
            continue
        if isinstance(parsed, dict):
            rows.append(parsed)
        else:
            errors.append(f"non-object {label} line {lineno}: {path}")
    if not rows and not errors:
        errors.append(f"empty {label}: {path}")
    return rows, errors


def baseline_status_has_terminal_coverage(
    wringer_dir: Path,
    manifest: dict[str, Any],
    matrix: list[dict[str, Any]],
) -> tuple[bool, list[str]]:
    grammar = str(manifest.get("grammar") or "")
    status_rows = baseline_status_rows(wringer_dir)
    if not status_rows:
        return False, ["baseline status has no parseable rows"]
    selected_ordinals = {
        int(row.get("ordinal") or 0)
        for row in matrix
        if int(row.get("ordinal") or 0) > 0
    }
    if not selected_ordinals:
        return False, ["selected frame matrix has no positive ordinals"]

    terminal_events = {"END", "FAIL", "TIMEOUT"}
    grammar_rows = [
        row
        for row in status_rows
        if str(row.get("grammar") or "") == grammar and int(row.get("ordinal") or 0) > 0
    ]
    terminal_ordinals = {
        int(row.get("ordinal") or 0)
        for row in grammar_rows
        if str(row.get("event") or "") in terminal_events
    }
    missing = sorted(selected_ordinals - terminal_ordinals)
    errors: list[str] = []
    if not grammar_rows:
        errors.append(f"baseline status has no frame rows for grammar={grammar!r}")
    if missing:
        errors.append(
            "baseline status lacks terminal frame coverage for ordinals="
            + ",".join(str(value) for value in missing)
        )
    return not errors, errors


def baseline_dense_frames_have_terminal_evidence(baseline_frames_path: Path, grammar: str) -> tuple[bool, list[str]]:
    grammar_frames = 0
    terminal_frames = 0
    for row in iter_jsonl(baseline_frames_path) or ():
        if str(row.get("grammar") or "") != grammar:
            continue
        grammar_frames += 1
        if (
            row.get("lifecycle") in {"ended", "timeout", "fail", "panic"}
            or row.get("phase") in {"comparison_result", "go_parse_status"}
            or str(row.get("phase") or "").endswith("_panic")
        ):
            terminal_frames += 1
            if terminal_frames > 0:
                break
    errors: list[str] = []
    if not grammar_frames:
        errors.append(f"baseline frames have no rows for grammar={grammar!r}")
    if grammar_frames and terminal_frames == 0:
        errors.append("baseline frames contain only progress/scheduled evidence")
    return not errors, errors


def assert_artifacts_sane(wringer_dir: Path, frames: str | None = None) -> int:
    rc = 0
    errors: list[str] = []
    manifest, error = strict_load_json(wringer_dir / "wringer_manifest.json", "wringer manifest")
    if error:
        print(error, file=sys.stderr)
        return 1
    if not isinstance(manifest, dict):
        print(f"wringer manifest is not an object: {wringer_dir / 'wringer_manifest.json'}", file=sys.stderr)
        return 1

    baseline_dir = resolve_recorded_path(
        Path(manifest.get("baseline_run_dir", wringer_dir / "baseline")),
        wringer_dir,
        manifest,
    )
    required_json = [
        (baseline_dir / "manifest.json", "baseline manifest"),
        (baseline_dir / "summary.json", "baseline summary"),
        (wringer_dir / "wringer_active.json", "wringer active"),
        (wringer_dir / "wringer_plan.json", "wringer plan summary"),
        (wringer_dir / "wringer_summary.json", "wringer summary"),
    ]
    for path, label in required_json:
        _, error = strict_load_json(path, label)
        if error:
            errors.append(error)

    baseline_frames_path = baseline_dir / "frames.jsonl"
    if not baseline_frames_path.exists() or baseline_frames_path.stat().st_size == 0:
        errors.append(f"missing or empty baseline frames: {baseline_frames_path}")
    required_jsonl = [
        (wringer_dir / "wringer_events.jsonl", "wringer events"),
        (wringer_dir / "frame_catalog.jsonl", "frame catalog"),
        (wringer_dir / "frame_matrix.jsonl", "frame matrix"),
        (wringer_dir / "wringer_plan.jsonl", "wringer plan"),
        (wringer_dir / "wringer_frames.jsonl", "wringer frames"),
    ]
    parsed_jsonl: dict[str, list[dict[str, Any]]] = {}
    for path, label in required_jsonl:
        rows, row_errors = strict_load_jsonl(path, label)
        parsed_jsonl[label] = rows
        errors.extend(row_errors)

    try:
        matrix = filter_rows_by_frames(build_frame_matrix(wringer_dir), frames)
    except SystemExit as exc:
        errors.append(str(exc))
        matrix = []
    if not matrix:
        errors.append("selected frame catalog is empty for a normal wringer run")
    else:
        try:
            plan_rows, _ = build_plan(wringer_dir)
            for item in baseline_row_contract_errors(matrix, filter_rows_by_frames(plan_rows, frames)):
                errors.append(f"baseline contract failure {item}")
        except SystemExit as exc:
            errors.append(str(exc))
        for item in full_integrity_plan_gaps(matrix, manifest):
            errors.append(f"full-integrity missing planned action {item}")
        for row in matrix:
            baseline = row.get("baseline") or {}
            terminal = baseline.get("terminal") if isinstance(baseline.get("terminal"), dict) else {}
            if baseline.get("planned") and not baseline.get("completed"):
                phase = terminal.get("last_phase", "")
                lifecycle = terminal.get("last_lifecycle", "")
                detail = "progress-only baseline evidence" if phase or lifecycle else "missing baseline terminal evidence"
                errors.append(
                    f"{detail} ordinal={row.get('ordinal', '')} "
                    f"path={row.get('path', '')} phase={phase} lifecycle={lifecycle}"
                )

    try:
        _, plan_summary = build_plan(wringer_dir)
    except SystemExit as exc:
        errors.append(str(exc))
        plan_summary = {}
    planned_actions = int(plan_summary.get("planned_actions") or 0) if isinstance(plan_summary, dict) else 0
    if planned_actions <= 0:
        errors.append("plan has zero planned actions")

    control = build_control_contract(wringer_dir, frames)
    for action in control.get("incomplete_actions", []):
        errors.append(
            "planned action lacks terminal evidence "
            f"stage={action.get('stage', '')} mode={action.get('mode', '')} "
            f"ordinal={action.get('ordinal', '')} path={action.get('path', '')}"
        )

    grammar = str(manifest.get("grammar") or "")
    status_covered = False
    status_errors: list[str] = []
    if matrix:
        status_covered, status_errors = baseline_status_has_terminal_coverage(wringer_dir, manifest, matrix)
    if not status_covered:
        dense_ok, dense_errors = baseline_dense_frames_have_terminal_evidence(baseline_frames_path, grammar)
        if not dense_ok:
            errors.extend(status_errors)
            errors.extend(dense_errors)

    for action in control.get("unexpected_lifecycle_actions", []):
        errors.append(
            "unexpected lifecycle action "
            f"stage={action.get('stage', '')} mode={action.get('mode', '')} "
            f"ordinal={action.get('ordinal', '')} path={action.get('path', '')}"
        )

    for message in errors:
        print(f"artifact sanity failure: {message}", file=sys.stderr)
        rc = 1
    return rc


def emit_file_list(wringer_dir: Path, scope: str) -> None:
    selected, suspicious = load_baseline(wringer_dir)
    rows = suspicious if scope == "suspicious" else selected
    for item in rows:
        item = public_frame_record(item)
        print(
            "\t".join(
                [
                    str(item.get("ordinal", "")),
                    str(item.get("grammar", "")),
                    str(item.get("corpus_kind", "")),
                    str(item.get("corpus_root", "")),
                    str(item.get("path", "")),
                    ",".join(item.get("reasons", item.get("suspicious_reasons", []))),
                ]
            )
        )


def emit_file_list_jsonl(wringer_dir: Path, scope: str) -> None:
    selected, suspicious = load_baseline(wringer_dir)
    rows = suspicious if scope == "suspicious" else selected
    for item in rows:
        print(json.dumps(public_frame_record(item), sort_keys=True))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("wringer_dir", type=Path)
    parser.add_argument("--write-summary", action="store_true")
    parser.add_argument("--emit-file-list", choices=("selected", "suspicious"))
    parser.add_argument("--emit-file-list-jsonl", choices=("selected", "suspicious"))
    parser.add_argument("--emit-frame-catalog-jsonl", action="store_true")
    parser.add_argument("--emit-frame-matrix-jsonl", action="store_true")
    parser.add_argument("--emit-plan-jsonl", action="store_true")
    parser.add_argument(
        "--frames",
        help="comma-separated frame selector for catalog, matrix, plan, replay commands, and assertions; accepts ordinals/ranges, sha256:<prefix>, base:<filename>, path:<substring>, all, or *",
    )
    parser.add_argument("--print-status", action="store_true")
    parser.add_argument("--print-control-status", action="store_true")
    parser.add_argument("--print-frame-control", action="store_true")
    parser.add_argument("--print-residual-frames", action="store_true")
    parser.add_argument("--emit-active-json", action="store_true")
    parser.add_argument("--emit-control-json", action="store_true")
    parser.add_argument("--emit-replay-commands", choices=("all", "baseline", "variants", "firstdiff"))
    parser.add_argument("--assert-closed", action="store_true")
    parser.add_argument("--assert-planned-complete", action="store_true")
    parser.add_argument("--assert-plan-exact", action="store_true")
    parser.add_argument("--assert-telemetry-complete", action="store_true")
    parser.add_argument("--assert-artifacts-sane", action="store_true")
    args = parser.parse_args()

    try:
        if args.write_summary:
            write_summary(args.wringer_dir)
        if args.emit_file_list:
            emit_file_list(args.wringer_dir, args.emit_file_list)
        if args.emit_file_list_jsonl:
            emit_file_list_jsonl(args.wringer_dir, args.emit_file_list_jsonl)
        if args.emit_frame_catalog_jsonl:
            emit_frame_catalog_jsonl(args.wringer_dir, args.frames)
        if args.emit_frame_matrix_jsonl:
            emit_frame_matrix_jsonl(args.wringer_dir, args.frames)
        if args.emit_plan_jsonl:
            emit_plan_jsonl(args.wringer_dir, args.frames)
        if args.print_status:
            print_status(args.wringer_dir)
        if args.print_control_status:
            print_control_status(args.wringer_dir, args.frames)
        if args.print_frame_control:
            print_frame_control(args.wringer_dir, args.frames)
        if args.print_residual_frames:
            print_residual_frames(args.wringer_dir, args.frames)
        if args.emit_active_json:
            emit_active_json(args.wringer_dir)
        if args.emit_control_json:
            emit_control_json(args.wringer_dir, args.frames)
        if args.emit_replay_commands:
            emit_replay_commands(args.wringer_dir, args.emit_replay_commands, args.frames)
        rc = 0
        if args.assert_closed:
            rc = max(rc, assert_closed(args.wringer_dir, args.frames))
        if args.assert_planned_complete:
            rc = max(rc, assert_planned_complete(args.wringer_dir, args.frames))
        if args.assert_plan_exact:
            rc = max(rc, assert_plan_exact(args.wringer_dir, args.frames))
        if args.assert_telemetry_complete:
            rc = max(rc, assert_telemetry_complete(args.wringer_dir, args.frames))
        if args.assert_artifacts_sane:
            rc = max(rc, assert_artifacts_sane(args.wringer_dir, args.frames))
        return rc
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
