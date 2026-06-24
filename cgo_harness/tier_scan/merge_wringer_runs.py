#!/usr/bin/env python3
"""Merge compact single-grammar wringer worker artifacts.

This reducer never invokes parser machinery and intentionally ignores dense
baseline artifacts such as baseline/frames.jsonl and baseline/events.jsonl.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import sys
import time
from pathlib import Path
from typing import Any


COMPACT_JSONL = (
    "frame_matrix.jsonl",
    "wringer_plan.jsonl",
    "wringer_frames.jsonl",
    "wringer_events.jsonl",
)
REQUIRED_FINAL_JSONL = frozenset(COMPACT_JSONL)
NO_EVIDENCE_FAMILY = "not_run_no_evidence"


class MergeError(RuntimeError):
    pass


def worker_label(path: Path) -> str:
    return str(path.resolve())


def load_json(path: Path, *, required: bool = False) -> dict[str, Any]:
    if not path.exists():
        if required:
            raise MergeError(f"missing required JSON artifact: {path}")
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise MergeError(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise MergeError(f"expected object JSON in {path}")
    return data


def iter_jsonl(path: Path, *, required: bool = False):
    if not path.exists():
        if required:
            raise MergeError(f"missing required JSONL artifact: {path}")
        return
    with path.open("r", encoding="utf-8", errors="replace") as f:
        for line_no, line in enumerate(f, 1):
            if not line.strip():
                continue
            try:
                item = json.loads(line)
            except json.JSONDecodeError as exc:
                raise MergeError(f"invalid JSONL in {path}:{line_no}: {exc}") from exc
            if not isinstance(item, dict):
                raise MergeError(f"expected object JSONL in {path}:{line_no}")
            yield item


def source_fields(worker_dir: Path, artifact: str, grammar: str = "") -> dict[str, Any]:
    return {
        "source_worker_dir": worker_label(worker_dir),
        "source_worker_name": worker_dir.name,
        "source_worker_grammar": grammar,
        "source_artifact": str(worker_dir / artifact),
    }


def add_source(row: dict[str, Any], worker_dir: Path, artifact: str, grammar: str) -> dict[str, Any]:
    out = dict(row)
    for key, value in source_fields(worker_dir, artifact, grammar).items():
        out.setdefault(key, value)
    return out


def merge_jsonl(stage: Path, run_dirs: list[Path], artifact: str, grammars: dict[Path, str]) -> int:
    count = 0
    with (stage / artifact).open("w", encoding="utf-8") as out:
        for run_dir in run_dirs:
            for row in iter_jsonl(run_dir / artifact):
                out.write(json.dumps(add_source(row, run_dir, artifact, grammars.get(run_dir, "")), sort_keys=True) + "\n")
                count += 1
    return count


def missing_required_compact_artifacts(run_dir: Path, has_summary: bool) -> list[str]:
    if not has_summary:
        return []
    return sorted(artifact for artifact in REQUIRED_FINAL_JSONL if not (run_dir / artifact).exists())


def residual_from_matrix_row(row: dict[str, Any], worker_dir: Path, grammar: str) -> dict[str, Any] | None:
    baseline = row.get("baseline")
    if not isinstance(baseline, dict):
        return None
    status = str(baseline.get("status") or "")
    if status == "match":
        return None
    if str(row.get("family") or "") == NO_EVIDENCE_FAMILY:
        return None
    sha = str(row.get("sha256") or row.get("source_sha256") or "")
    terminal = baseline.get("terminal") if isinstance(baseline.get("terminal"), dict) else {}
    reasons = baseline.get("reasons")
    if reasons is None:
        reasons = row.get("reasons")
    if reasons is None:
        reasons = []
    if not isinstance(reasons, list):
        reasons = [str(reasons)]
    out = {
        "grammar": str(row.get("grammar") or grammar),
        "ordinal": row.get("ordinal") or row.get("index"),
        "base": row.get("base") or Path(str(row.get("path") or "")).name,
        "path": row.get("path") or "",
        "sha256_prefix": sha[:12],
        "sha256": sha,
        "status": status,
        "reasons": reasons,
        "family": row.get("family", ""),
        "family_reasons": row.get("family_reasons", []),
        "log": baseline.get("host_log") or baseline.get("log") or "",
        "source_worker_dir": worker_label(worker_dir),
        "source_worker_name": worker_dir.name,
        "source_artifact": str(worker_dir / "frame_matrix.jsonl"),
    }
    if isinstance(terminal, dict):
        out["terminal"] = {
            key: terminal.get(key)
            for key in (
                "comparison_result",
                "go_parse_status",
                "last_lifecycle",
                "last_phase",
                "measure_line",
                "runtime",
                "timeout",
                "fail",
                "panic",
            )
            if key in terminal
        }
    return out


def manifest_plan_only(manifest: dict[str, Any]) -> bool:
    config = manifest.get("config") if isinstance(manifest.get("config"), dict) else {}
    return bool(config.get("plan_only"))


def write_residuals(
    stage: Path,
    run_dirs: list[Path],
    grammars: dict[Path, str],
    manifests: dict[Path, dict[str, Any]],
) -> int:
    count = 0
    with (stage / "residual_frames.jsonl").open("w", encoding="utf-8") as out:
        for run_dir in run_dirs:
            grammar = grammars.get(run_dir, "")
            plan_only = manifest_plan_only(manifests.get(run_dir, {}))
            for row in iter_jsonl(run_dir / "frame_matrix.jsonl"):
                baseline = row.get("baseline") if isinstance(row.get("baseline"), dict) else {}
                if plan_only and baseline.get("planned") is True and baseline.get("status") == "not_run":
                    continue
                residual = residual_from_matrix_row(row, run_dir, grammar)
                if residual is None:
                    continue
                out.write(json.dumps(residual, sort_keys=True) + "\n")
                count += 1
    return count


def nested_count(summary: dict[str, Any], *keys: str) -> int:
    value: Any = summary
    for key in keys:
        if not isinstance(value, dict):
            return 0
        value = value.get(key)
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    return 0


def compact_list_count(value: Any) -> int:
    if isinstance(value, list):
        return len(value)
    if isinstance(value, dict):
        return len(value)
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        return int(value)
    return 0


def merged_family_counts(frame_matrix_path: Path) -> dict[str, int]:
    counts: dict[str, int] = {}
    for row in iter_jsonl(frame_matrix_path):
        family = str(row.get("family") or "version_or_corpus")
        if family == NO_EVIDENCE_FAMILY:
            continue
        counts[family] = counts.get(family, 0) + 1
    return counts


def read_parent_lifecycle(out_dir: Path) -> dict[str, Any]:
    parent_dir = out_dir.parent
    events_path = parent_dir / "events.jsonl"
    if not events_path.exists():
        return {}
    starts: dict[str, dict[str, Any]] = {}
    terminals: dict[str, dict[str, Any]] = {}
    counts: dict[str, int] = {}
    for line_no, row in enumerate(iter_jsonl(events_path), 1):
        event = str(row.get("event") or "")
        grammar = str(row.get("grammar") or row.get("shard") or "")
        if not grammar:
            grammar = f"line-{line_no}"
        counts[event] = counts.get(event, 0) + 1
        if event == "START":
            starts[grammar] = row
        elif event in {"END", "FAIL", "ABORT", "TIMEOUT"}:
            terminals[grammar] = row
    open_grammars = sorted(set(starts) - set(terminals))
    return {
        "parent_run_dir": str(parent_dir),
        "events_jsonl": str(events_path),
        "active_worker_json": str(parent_dir / "active_worker.json"),
        "active_worker_txt": str(parent_dir / "active_worker.txt"),
        "status_tsv": str(parent_dir / "status.tsv"),
        "progress_log": str(parent_dir / "progress.log"),
        "counts": {
            "starts": len(starts),
            "terminals": len(terminals),
            "open": len(open_grammars),
            "events_by_type": counts,
        },
        "open_workers": [
            {
                "grammar": grammar,
                "pid": starts[grammar].get("pid"),
                "worker_dir": starts[grammar].get("worker_dir", ""),
                "artifacts": starts[grammar].get("artifacts", {}),
            }
            for grammar in open_grammars
        ],
    }


def build_summary(
    out_dir: Path,
    run_dirs: list[Path],
    grammars: dict[Path, str],
    manifests: dict[Path, dict[str, Any]],
    summaries: dict[Path, dict[str, Any]],
    jsonl_counts: dict[str, int],
    residual_frames: int,
    allow_incomplete: bool,
    missing_compact: dict[Path, list[str]],
    merged_frame_matrix_path: Path,
) -> dict[str, Any]:
    worker_rows: list[dict[str, Any]] = []
    complete_controls = 0
    open_controls = 0
    failed_controls = 0
    selected_frames = 0
    planned_actions = 0
    completed_actions = 0
    incomplete_actions = 0

    for run_dir in run_dirs:
        summary = summaries.get(run_dir, {})
        manifest = manifests.get(run_dir, {})
        grammar = grammars.get(run_dir, "")
        control = summary.get("control", {}) if isinstance(summary.get("control"), dict) else {}
        action_counts = control.get("action_counts", {}) if isinstance(control.get("action_counts"), dict) else {}
        overall = action_counts.get("overall", {}) if isinstance(action_counts.get("overall"), dict) else {}
        wringer_lifecycle = (
            control.get("wringer_lifecycle_balance", {})
            if isinstance(control.get("wringer_lifecycle_balance"), dict)
            else {}
        )
        baseline_lifecycle = (
            control.get("baseline_frame_lifecycle_balance", {})
            if isinstance(control.get("baseline_frame_lifecycle_balance"), dict)
            else {}
        )
        control_status = str(control.get("status") or "")
        incomplete_count = nested_count(overall, "incomplete")
        wringer_open_count = nested_count(wringer_lifecycle, "open")
        baseline_open_count = nested_count(baseline_lifecycle, "open")
        open_actions_count = compact_list_count(control.get("open_actions"))
        open_baseline_frames_count = compact_list_count(
            control.get("open_baseline_frames", baseline_lifecycle.get("open_frames"))
        )
        incomplete_actions_count = compact_list_count(control.get("incomplete_actions"))
        missing_artifacts = missing_compact.get(run_dir, [])
        baseline_exit = summary.get("baseline_exit_status", manifest.get("baseline_exit_status"))
        infra_status = manifest.get("infra_status", 0)
        plan_only = manifest_plan_only(manifest)
        has_summary = bool(summary)
        if not has_summary:
            open_controls += 1
            state = "missing-summary"
        elif infra_status not in (0, "0", None, ""):
            failed_controls += 1
            state = "failed"
        elif (
            plan_only
            and wringer_open_count == 0
            and baseline_open_count == 0
            and not open_actions_count
            and not open_baseline_frames_count
            and not missing_artifacts
        ):
            complete_controls += 1
            state = "complete"
        elif (
            control_status == "open"
            or incomplete_count
            or wringer_open_count
            or baseline_open_count
            or missing_artifacts
        ):
            open_controls += 1
            state = "open"
        else:
            complete_controls += 1
            state = "complete"
        effective_control_status = control_status
        if plan_only and state == "complete" and control_status == "open":
            effective_control_status = "plan-only-complete"
        selected_frames += nested_count(summary, "counts", "selected_files")
        planned_actions += nested_count(overall, "planned")
        completed_actions += nested_count(overall, "completed")
        incomplete_actions += nested_count(overall, "incomplete")
        family_counts = summary.get("family_counts", {})
        if not isinstance(family_counts, dict):
            family_counts = {}
        worker_rows.append(
            {
                "worker_dir": worker_label(run_dir),
                "grammar": grammar,
                "state": state,
                "summary_json": str(run_dir / "wringer_summary.json") if has_summary else "",
                "manifest_json": str(run_dir / "wringer_manifest.json") if manifest else "",
                "baseline_exit_status": baseline_exit,
                "infra_status": infra_status,
                "plan_only": plan_only,
                "control_status": effective_control_status,
                "raw_control_status": control_status,
                "missing_compact_artifacts": missing_artifacts,
                "wringer_lifecycle_balance": wringer_lifecycle,
                "baseline_frame_lifecycle_balance": baseline_lifecycle,
                "open_actions": open_actions_count,
                "open_baseline_frames": open_baseline_frames_count,
                "selected_frames": nested_count(summary, "counts", "selected_files"),
                "planned_actions": nested_count(overall, "planned"),
                "completed_actions": nested_count(overall, "completed"),
                "incomplete_actions": incomplete_count,
                "incomplete_action_rows": incomplete_actions_count,
                "family_counts": family_counts,
            }
        )

    grammar_set = {grammar for grammar in grammars.values() if grammar}
    family_counts = merged_family_counts(merged_frame_matrix_path)
    return {
        "out_dir": str(out_dir),
        "merged_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "snapshot_mode": allow_incomplete,
        "complete": open_controls == 0 and failed_controls == 0,
        "run_state": "complete" if open_controls == 0 and failed_controls == 0 else ("snapshot-incomplete" if allow_incomplete else "incomplete"),
        "artifacts": {
            "summary_json": str(out_dir / "summary.json"),
            "frame_matrix_jsonl": str(out_dir / "frame_matrix.jsonl"),
            "wringer_plan_jsonl": str(out_dir / "wringer_plan.jsonl"),
            "residual_frames_jsonl": str(out_dir / "residual_frames.jsonl"),
            "wringer_frames_jsonl": str(out_dir / "wringer_frames.jsonl"),
            "wringer_events_jsonl": str(out_dir / "wringer_events.jsonl"),
        },
        "counts": {
            "worker_count": len(run_dirs),
            "grammar_count": len(grammar_set),
            "complete_controls": complete_controls,
            "open_controls": open_controls,
            "failed_controls": failed_controls,
            "selected_frames": selected_frames,
            "residual_frames": residual_frames,
            "planned_actions": planned_actions,
            "completed_actions": completed_actions,
            "incomplete_actions": incomplete_actions,
            "frame_matrix_rows": jsonl_counts.get("frame_matrix.jsonl", 0),
            "wringer_plan_rows": jsonl_counts.get("wringer_plan.jsonl", 0),
            "wringer_frames_rows": jsonl_counts.get("wringer_frames.jsonl", 0),
            "wringer_events_rows": jsonl_counts.get("wringer_events.jsonl", 0),
            "workers_missing_compact_artifacts": sum(1 for artifacts in missing_compact.values() if artifacts),
            "missing_compact_artifacts": sum(len(artifacts) for artifacts in missing_compact.values()),
            "families": family_counts,
        },
        "family_counts": family_counts,
        "workers": worker_rows,
        "parent_lifecycle": read_parent_lifecycle(out_dir),
    }


def prepare_stage(out_dir: Path) -> Path:
    out_dir.parent.mkdir(parents=True, exist_ok=True)
    stage = out_dir.parent / f".{out_dir.name}.tmp.{os.getpid()}"
    if stage.exists():
        shutil.rmtree(stage)
    stage.mkdir(parents=True)
    return stage


def publish_stage(stage: Path, out_dir: Path) -> None:
    backup = out_dir.parent / f".{out_dir.name}.old.{os.getpid()}"
    if backup.exists():
        shutil.rmtree(backup)
    if out_dir.exists():
        out_dir.replace(backup)
    stage.replace(out_dir)
    if backup.exists():
        shutil.rmtree(backup)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("out_dir", type=Path, help="merged output directory")
    parser.add_argument("worker_dirs", nargs="+", type=Path, help="wringer worker directories")
    parser.add_argument(
        "--allow-incomplete",
        action="store_true",
        help="allow missing live worker summaries while writing a snapshot",
    )
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    out_dir = args.out_dir.resolve()
    run_dirs = [path.resolve() for path in args.worker_dirs]
    missing = [path for path in run_dirs if not path.is_dir()]
    if missing:
        print("missing wringer worker dir(s): " + ", ".join(str(path) for path in missing), file=sys.stderr)
        return 2
    if out_dir in run_dirs:
        print("merged output dir must not be a worker dir", file=sys.stderr)
        return 2

    stage: Path | None = None
    try:
        manifests: dict[Path, dict[str, Any]] = {}
        summaries: dict[Path, dict[str, Any]] = {}
        grammars: dict[Path, str] = {}
        missing_compact: dict[Path, list[str]] = {}
        for run_dir in run_dirs:
            manifest = load_json(run_dir / "wringer_manifest.json")
            summary = load_json(run_dir / "wringer_summary.json", required=not args.allow_incomplete)
            manifests[run_dir] = manifest
            summaries[run_dir] = summary
            grammars[run_dir] = str(summary.get("grammar") or manifest.get("grammar") or run_dir.name)
            if not summary and not args.allow_incomplete:
                raise MergeError(f"missing wringer_summary.json in {run_dir}")
            missing_compact[run_dir] = missing_required_compact_artifacts(run_dir, bool(summary))
            if missing_compact[run_dir] and not args.allow_incomplete:
                raise MergeError(
                    "missing required compact wringer artifact(s) in "
                    f"{run_dir}: {', '.join(missing_compact[run_dir])}"
                )

        stage = prepare_stage(out_dir)
        jsonl_counts = {
            artifact: merge_jsonl(stage, run_dirs, artifact, grammars)
            for artifact in COMPACT_JSONL
        }
        residual_frames = write_residuals(stage, run_dirs, grammars, manifests)
        summary = build_summary(
            out_dir,
            run_dirs,
            grammars,
            manifests,
            summaries,
            jsonl_counts,
            residual_frames,
            args.allow_incomplete,
            missing_compact,
            stage / "frame_matrix.jsonl",
        )
        (stage / "summary.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        publish_stage(stage, out_dir)
        stage = None
        print(f"merged {len(run_dirs)} wringer worker(s) into {out_dir}")
        print(
            "counts: "
            f"workers={summary['counts']['worker_count']} "
            f"grammars={summary['counts']['grammar_count']} "
            f"complete={summary['counts']['complete_controls']} "
            f"open={summary['counts']['open_controls']} "
            f"failed={summary['counts']['failed_controls']} "
            f"selected_frames={summary['counts']['selected_frames']} "
            f"residual_frames={summary['counts']['residual_frames']} "
            f"planned_actions={summary['counts']['planned_actions']} "
            f"completed_actions={summary['counts']['completed_actions']} "
            f"incomplete_actions={summary['counts']['incomplete_actions']}"
        )
        if not summary["complete"] and not args.allow_incomplete:
            return 1
        return 0
    except MergeError as exc:
        print(f"merge failed: {exc}", file=sys.stderr)
        return 1
    finally:
        if stage is not None and stage.exists():
            shutil.rmtree(stage)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
