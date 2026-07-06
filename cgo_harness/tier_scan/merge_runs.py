#!/usr/bin/env python3
"""Merge disjoint tier-scan worker artifact directories.

This reducer does not invoke parser machinery. It combines worker-local
artifacts produced by cgo_harness/docker/run_tier_scan.sh into one inventory
view for follow-up diagnosis and reporting.
"""

from __future__ import annotations

import argparse
import json
import os
import shlex
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Any


TEXT_ARTIFACTS = (
    "tier_scan.txt",
    "clean.txt",
    "tier_iv.txt",
    "unmeasured.txt",
    "visited_grammars.txt",
)
JSONL_ARTIFACTS = ("events.jsonl", "frames.jsonl")
OPTIONAL_APPEND_ARTIFACTS = ("progress.log", "status.tsv")


class MergeError(RuntimeError):
    pass


def read_nonempty_lines(path: Path) -> list[str]:
    if not path.exists():
        return []
    return [line.rstrip("\n") for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def grammar_from_line(filename: str, line: str) -> str:
    fields = line.split()
    if not fields:
        return ""
    if filename == "tier_scan.txt":
        if len(fields) < 2 or fields[0] != "MEASURE-DTIER":
            raise MergeError(f"{filename} has an unexpected line: {line!r}")
        return fields[1]
    return fields[0]


def worker_label(run_dir: Path) -> str:
    return str(run_dir.resolve())


def load_json(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise MergeError(f"invalid JSON in {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise MergeError(f"expected object JSON in {path}")
    return data


def parse_jsonl_line(path: Path, line_no: int, line: str) -> dict[str, Any]:
    try:
        item = json.loads(line)
    except json.JSONDecodeError as exc:
        raise MergeError(f"invalid JSONL in {path}:{line_no}: {exc}") from exc
    if not isinstance(item, dict):
        raise MergeError(f"expected object JSONL in {path}:{line_no}")
    return item


def ensure_unique_grammar(
    owners: dict[str, tuple[Path, str]],
    grammar: str,
    run_dir: Path,
    artifact: str,
) -> None:
    if not grammar:
        return
    previous = owners.get(grammar)
    if previous is None:
        owners[grammar] = (run_dir, artifact)
        return
    prev_dir, prev_artifact = previous
    if prev_dir.resolve() == run_dir.resolve() and prev_artifact == artifact:
        return
    raise MergeError(
        "duplicate grammar in worker artifacts: "
        f"{grammar} appears in {prev_dir}/{prev_artifact} and {run_dir}/{artifact}"
    )


def merge_text_artifacts(out_dir: Path, run_dirs: list[Path], allow_incomplete: bool = False) -> dict[str, set[str]]:
    merged: dict[str, list[str]] = {name: [] for name in TEXT_ARTIFACTS}
    owners_by_artifact: dict[str, dict[str, tuple[Path, str]]] = {
        artifact: {} for artifact in TEXT_ARTIFACTS
    }

    for run_dir in run_dirs:
        for artifact in TEXT_ARTIFACTS:
            path = run_dir / artifact
            lines = read_nonempty_lines(path)
            seen_in_file: set[str] = set()
            for line in lines:
                grammar = grammar_from_line(artifact, line)
                if grammar in seen_in_file:
                    raise MergeError(f"duplicate grammar {grammar} within {path}")
                seen_in_file.add(grammar)
                ensure_unique_grammar(owners_by_artifact[artifact], grammar, run_dir, artifact)
                merged[artifact].append(line)

    for artifact, lines in merged.items():
        if artifact in {"clean.txt", "visited_grammars.txt"}:
            lines = sorted(lines)
        (out_dir / artifact).write_text("\n".join(lines) + ("\n" if lines else ""), encoding="utf-8")

    clean = {grammar_from_line("clean.txt", line) for line in merged["clean.txt"]}
    tier_iv = {grammar_from_line("tier_iv.txt", line) for line in merged["tier_iv.txt"]}
    unmeasured = {grammar_from_line("unmeasured.txt", line) for line in merged["unmeasured.txt"]}
    measured = {grammar_from_line("tier_scan.txt", line) for line in merged["tier_scan.txt"]}
    visited = {grammar_from_line("visited_grammars.txt", line) for line in merged["visited_grammars.txt"]}
    overlap = (clean & tier_iv) | (clean & unmeasured) | (tier_iv & unmeasured)
    if overlap:
        raise MergeError(f"grammar appears in multiple final classifications: {', '.join(sorted(overlap))}")
    classified = clean | tier_iv | unmeasured
    missing_classification = visited - classified
    if missing_classification and not allow_incomplete:
        raise MergeError(
            "visited grammar missing final classification: "
            + ", ".join(sorted(missing_classification))
        )
    extra_classification = classified - visited
    if extra_classification:
        raise MergeError(
            "classification exists outside visited grammars: "
            + ", ".join(sorted(extra_classification))
        )
    expected_measured = clean | tier_iv
    if measured != expected_measured:
        missing_measured = expected_measured - measured
        extra_measured = measured - expected_measured
        details = []
        if missing_measured:
            details.append("missing measured rows for " + ", ".join(sorted(missing_measured)))
        if extra_measured:
            details.append("unexpected measured rows for " + ", ".join(sorted(extra_measured)))
        if not allow_incomplete:
            raise MergeError("tier_scan.txt does not equal clean union tier_iv: " + "; ".join(details))
    return {
        "clean": clean,
        "tier_iv": tier_iv,
        "unmeasured": unmeasured,
        "measured": measured,
        "visited": visited,
        "classified": classified,
        "missing_classification": missing_classification,
    }


def merge_jsonl_artifact(out_dir: Path, run_dirs: list[Path], artifact: str, allow_incomplete: bool = False) -> int:
    count = 0
    with (out_dir / artifact).open("w", encoding="utf-8") as out:
        for run_dir in run_dirs:
            path = run_dir / artifact
            if not path.exists():
                continue
            lines = path.read_text(encoding="utf-8").splitlines()
            last_nonempty_line_no = 0
            for i, raw_line in enumerate(lines, 1):
                if raw_line.strip():
                    last_nonempty_line_no = i
            for line_no, line in enumerate(lines, 1):
                if not line.strip():
                    continue
                try:
                    item = parse_jsonl_line(path, line_no, line)
                except MergeError:
                    if allow_incomplete and line_no == last_nonempty_line_no:
                        continue
                    raise
                item.setdefault("worker_run_dir", worker_label(run_dir))
                item.setdefault("source_artifact", str(path))
                out.write(json.dumps(item, sort_keys=True) + "\n")
                count += 1
    return count


def merge_append_artifact(out_dir: Path, run_dirs: list[Path], artifact: str) -> int:
    lines_written = 0
    with (out_dir / artifact).open("w", encoding="utf-8") as out:
        for run_dir in run_dirs:
            path = run_dir / artifact
            if not path.exists():
                continue
            for line in path.read_text(encoding="utf-8").splitlines():
                out.write(line.rstrip("\n") + "\n")
                lines_written += 1
    return lines_written


def merge_manifest(out_dir: Path, run_dirs: list[Path], published_out_dir: Path | None = None) -> int:
    entries: list[dict[str, Any]] = []
    seen: dict[tuple[str, str], Path] = {}
    for run_dir in run_dirs:
        manifest_path = run_dir / "manifest.json"
        manifest = load_json(manifest_path)
        for raw_entry in manifest.get("grammars", []):
            if not isinstance(raw_entry, dict):
                raise MergeError(f"manifest entry in {manifest_path} is not an object")
            grammar = str(raw_entry.get("grammar", ""))
            corpus_kind = str(raw_entry.get("corpus_kind", ""))
            key = (grammar, corpus_kind)
            if not grammar or not corpus_kind:
                raise MergeError(f"manifest entry in {manifest_path} is missing grammar/corpus_kind")
            previous = seen.get(key)
            if previous is not None:
                raise MergeError(
                    "duplicate grammar/corpus manifest entry: "
                    f"{grammar}/{corpus_kind} appears in {previous} and {manifest_path}"
                )
            seen[key] = manifest_path
            entry = dict(raw_entry)
            entry["worker_run_dir"] = worker_label(run_dir)
            entry["source_manifest"] = str(manifest_path)
            entries.append(entry)

    merged = {
        "run_dir": str(published_out_dir or out_dir),
        "merged_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "worker_run_dirs": [worker_label(run_dir) for run_dir in run_dirs],
        "grammars": entries,
    }
    (out_dir / "manifest.json").write_text(json.dumps(merged, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return len(entries)


def read_active_grammar(run_dir: Path) -> dict[str, str]:
    path = run_dir / "active_grammar.txt"
    lines = read_nonempty_lines(path)
    if not lines:
        return {}
    fields = lines[-1].split("\t")
    active = {"raw": lines[-1], "path": str(path)}
    if len(fields) >= 1:
        active["timestamp"] = fields[0]
    if len(fields) >= 2:
        active["grammar"] = fields[1]
    if len(fields) >= 3:
        active["corpus_kind"] = fields[2]
    if len(fields) >= 4:
        active["detail"] = fields[3]
    return active


def read_checkpoint(run_dir: Path) -> dict[str, str]:
    path = run_dir / "checkpoint.env"
    if not path.exists():
        return {}
    checkpoint: dict[str, str] = {"path": str(path)}
    for line_no, raw_line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        try:
            parsed = shlex.split(line, posix=True)
        except ValueError as exc:
            raise MergeError(f"invalid checkpoint.env line in {path}:{line_no}: {exc}") from exc
        if len(parsed) != 1 or "=" not in parsed[0]:
            raise MergeError(f"invalid checkpoint.env line in {path}:{line_no}: {raw_line!r}")
        key, value = parsed[0].split("=", 1)
        checkpoint[key] = value
    return checkpoint


def read_parent_worker_lifecycle(out_dir: Path) -> dict[str, Any]:
    parent_dir = out_dir.parent
    events_path = parent_dir / "events.jsonl"
    if not events_path.exists():
        return {}

    starts: dict[str, dict[str, Any]] = {}
    terminals: dict[str, dict[str, Any]] = {}
    counts: dict[str, int] = {}
    for line_no, line in enumerate(events_path.read_text(encoding="utf-8").splitlines(), 1):
        if not line.strip():
            continue
        item = parse_jsonl_line(events_path, line_no, line)
        event = str(item.get("event", ""))
        shard = str(item.get("shard", ""))
        if not shard:
            continue
        counts[event] = counts.get(event, 0) + 1
        if event == "START":
            starts[shard] = item
        elif event in {"END", "FAIL", "TIMEOUT", "ABORT"}:
            terminals[shard] = item

    open_shards = sorted(set(starts) - set(terminals))
    active_snapshot = load_json(parent_dir / "active_worker.json")
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
            "open": len(open_shards),
            "events_by_type": counts,
        },
        "open_shards": open_shards,
        "open_workers": [
            {
                "shard": shard,
                "pid": starts[shard].get("pid"),
                "worker_dir": starts[shard].get("worker_dir", ""),
                "languages": starts[shard].get("languages", []),
                "artifacts": starts[shard].get("artifacts", {}),
            }
            for shard in open_shards
        ],
        "active_snapshot": active_snapshot,
    }


def build_summary(
    out_dir: Path,
    run_dirs: list[Path],
    grammar_sets: dict[str, set[str]],
    jsonl_counts: dict[str, int],
    manifest_entries: int,
    allow_incomplete: bool = False,
) -> dict[str, Any]:
    worker_summaries = []
    exit_status = 0
    complete = True
    for run_dir in run_dirs:
        summary_path = run_dir / "summary.json"
        summary = load_json(summary_path)
        state = summary.get("run_state", "missing-summary")
        active = read_active_grammar(run_dir)
        checkpoint = read_checkpoint(run_dir)
        worker_exit = int(summary.get("exit_status", 0 if allow_incomplete else (1 if not summary else 0)))
        if worker_exit != 0 and not allow_incomplete:
            exit_status = 1
        if state != "complete":
            complete = False
        worker_summaries.append(
            {
                "run_dir": worker_label(run_dir),
                "summary": str(summary_path) if summary_path.exists() else "",
                "run_state": state,
                "exit_status": worker_exit,
                "counts": summary.get("counts", {}),
                "active": active,
                "checkpoint": checkpoint,
            }
        )

    missing_classification = grammar_sets["missing_classification"]
    if allow_incomplete and (missing_classification or not complete):
        run_state = "snapshot-incomplete"
    else:
        run_state = "complete" if complete and exit_status == 0 else ("failed" if exit_status else "incomplete")
    return {
        "out_dir": str(out_dir),
        "artifacts": {
            "events_jsonl": str(out_dir / "events.jsonl"),
            "frames_jsonl": str(out_dir / "frames.jsonl"),
            "manifest_json": str(out_dir / "manifest.json"),
            "resume_env": str(out_dir / "resume.env"),
        },
        "snapshot_mode": allow_incomplete,
        "complete": run_state == "complete",
        "run_state": run_state,
        "counts": {
            "clean": len(grammar_sets["clean"]),
            "tier_iv": len(grammar_sets["tier_iv"]),
            "unmeasured": len(grammar_sets["unmeasured"]),
            "classified": len(grammar_sets["classified"]),
            "in_progress": len(missing_classification),
            "missing_classification": len(missing_classification),
            "visited": len(grammar_sets["visited"]),
        },
        "selected_grammar_count": len(grammar_sets["visited"]),
        "visited_grammar_count": len(grammar_sets["visited"]),
        "in_progress": sorted(missing_classification),
        "missing_classification": sorted(missing_classification),
        "events": {
            "jsonl": jsonl_counts.get("events.jsonl", 0),
            "frames": jsonl_counts.get("frames.jsonl", 0),
        },
        "final_active": {
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "grammar": "idle" if run_state == "complete" else "merged",
            "corpus_kind": "none",
            "detail": f"merged_workers={len(run_dirs)}",
        },
        "resume_hint": "",
        "checkpoint": {},
        "exit_status": exit_status,
        "merge": {
            "worker_count": len(run_dirs),
            "worker_runs": worker_summaries,
            "manifest_entries": manifest_entries,
            "allow_incomplete": allow_incomplete,
            "parent_lifecycle": read_parent_worker_lifecycle(out_dir),
        },
    }


def write_resume(out_dir: Path, summary: dict[str, Any], published_out_dir: Path | None = None) -> None:
    artifact_dir = published_out_dir or out_dir
    lines = [
        f"updated_at={summary['final_active']['timestamp']!r}",
        "last_completed_grammar=''",
        f"status={summary['run_state']!r}",
        f"detail={'merged_workers=' + str(summary['merge']['worker_count'])!r}",
        "resume_hint=''",
        f"active_file={str(artifact_dir / 'active_grammar.txt')!r}",
        "checkpoint_file=''",
    ]
    (out_dir / "resume.env").write_text("\n".join(lines) + "\n", encoding="utf-8")
    (out_dir / "active_grammar.txt").write_text(
        "\t".join(
            [
                summary["final_active"]["timestamp"],
                summary["final_active"]["grammar"],
                summary["final_active"]["corpus_kind"],
                summary["final_active"]["detail"],
            ]
        )
        + "\n",
        encoding="utf-8",
    )


def run_summarizer(out_dir: Path, summarizer: Path) -> int:
    if not summarizer.exists():
        return 0
    return subprocess.run([sys.executable, str(summarizer), str(out_dir)], check=False).returncode


def remove_diagnostic_summaries(out_dir: Path) -> None:
    for path in out_dir.glob("diagnostic_summary.*"):
        if path.is_file() or path.is_symlink():
            path.unlink()


def prepare_stage(out_dir: Path) -> Path:
    parent = out_dir.parent
    parent.mkdir(parents=True, exist_ok=True)
    stage = parent / f".{out_dir.name}.tmp.{os.getpid()}"
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
    parser.add_argument("run_dirs", nargs="+", type=Path, help="worker run directories to merge")
    parser.add_argument(
        "--allow-incomplete",
        "--snapshot",
        dest="allow_incomplete",
        action="store_true",
        help="write a live snapshot without failing on incomplete worker summaries or unclassified visited grammars",
    )
    parser.add_argument("--no-summarize", action="store_true", help="do not invoke summarize_scan.py after merging")
    parser.add_argument(
        "--summarizer",
        type=Path,
        default=Path(__file__).with_name("summarize_scan.py"),
        help="diagnostic summarizer to invoke when present",
    )
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    out_dir = args.out_dir.resolve()
    run_dirs = [path.resolve() for path in args.run_dirs]
    missing = [path for path in run_dirs if not path.is_dir()]
    if missing:
        print("missing worker run dir(s): " + ", ".join(str(path) for path in missing), file=sys.stderr)
        return 2
    if out_dir in run_dirs:
        print("merged output dir must not be one of the worker run dirs", file=sys.stderr)
        return 2

    if out_dir.exists():
        remove_diagnostic_summaries(out_dir)
    stage: Path | None = None
    try:
        stage = prepare_stage(out_dir)
        grammar_sets = merge_text_artifacts(stage, run_dirs, allow_incomplete=args.allow_incomplete)
        jsonl_counts = {
            artifact: merge_jsonl_artifact(
                stage,
                run_dirs,
                artifact,
                allow_incomplete=args.allow_incomplete,
            )
            for artifact in JSONL_ARTIFACTS
        }
        for artifact in OPTIONAL_APPEND_ARTIFACTS:
            merge_append_artifact(stage, run_dirs, artifact)
        manifest_entries = merge_manifest(stage, run_dirs, out_dir)
        summary = build_summary(
            out_dir,
            run_dirs,
            grammar_sets,
            jsonl_counts,
            manifest_entries,
            allow_incomplete=args.allow_incomplete,
        )
        (stage / "summary.json").write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        write_resume(stage, summary, out_dir)
        summarizer_status = 0
        if not args.no_summarize:
            summarizer_status = run_summarizer(stage, args.summarizer.resolve())
            if summarizer_status != 0:
                remove_diagnostic_summaries(stage)
                print(f"warning: diagnostic summarizer exited {summarizer_status}", file=sys.stderr)
        publish_stage(stage, out_dir)
        stage = None
        print(f"merged {len(run_dirs)} worker run(s) into {out_dir}")
        print(
            "counts: "
            f"clean={len(grammar_sets['clean'])} "
            f"tier_iv={len(grammar_sets['tier_iv'])} "
            f"unmeasured={len(grammar_sets['unmeasured'])} "
            f"classified={len(grammar_sets['classified'])} "
            f"in_progress={len(grammar_sets['missing_classification'])} "
            f"visited={len(grammar_sets['visited'])}"
        )
        if summary["exit_status"] != 0 or summary["run_state"] != "complete":
            print(
                f"merged summary is not complete: run_state={summary['run_state']} "
                f"exit_status={summary['exit_status']}",
                file=sys.stderr,
            )
            if not args.allow_incomplete:
                return 1
        if summarizer_status != 0 and not args.allow_incomplete:
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
