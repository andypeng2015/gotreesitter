#!/usr/bin/env python3
"""Perf-tier ratchet gate: fail if any grammar regressed below its floor tier.

Current state is read from the canonical published artifacts, NOT a perf rerun:

  * parity  — ``cgo_harness/tier_scan/clean_grammars.txt`` (the hard clean
    ratchet: a grammar listed there parses byte-identical to the C oracle on
    its full measured corpus).
  * tier    — ``docs/reports/tiers.json`` (the published per-release tier
    table). The current tier of grammar ``g`` is tiers.json's tier for ``g``
    (``IV`` if absent). The parity gate is re-asserted defensively here: a
    non-clean grammar may not publish as I/II/unranked, and may publish as III
    only when it carries an assessed cause.

The committed floor lives in ``tier_floors.json`` (one ``{"tier": ...}`` entry
per grammar). The gate exits non-zero if any grammar's current tier is BELOW
its floor. ``--bump`` ratchets the floor up to the current tiers (use only
after a verified, published lift).

Canonical: ``docs/reports/tier-ratchet.md`` and
``cgo_harness/tier_scan/README.md``.
"""
import json, os, sys

ROOT = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(ROOT, "..", ".."))
FLOOR = os.path.join(ROOT, "tier_floors.json")
TIERS = os.path.join(ROOT, "tiers.json")
CLEAN = os.path.join(REPO, "cgo_harness", "tier_scan", "clean_grammars.txt")
CLASS_TSV = os.path.join(REPO, "cgo_harness", "tier_scan", "tier_classification.tsv")
UNKNOWN_CAUSES = {"unknown", "unassessed", "unclassified"}

# Tiers are Roman numerals (I best .. IV worst) so "III" never collides with the
# C language. `unranked` is parity-clean with perf pending: better than IV, but
# not yet eligible for a perf tier.
RANK = {"I": 4, "II": 3, "III": 2, "unranked": 1, "IV": 0}  # higher is better


def clean_set():
    """Grammars that passed full-corpus parity vs the C oracle (the hard gate)."""
    with open(CLEAN) as f:
        return {ln.strip() for ln in f if ln.strip()}


def classification_tiers():
    """Classification TSV tier by grammar, used to reject contradictory sources."""
    rows = {}
    with open(CLASS_TSV) as f:
        for i, ln in enumerate(f):
            parts = ln.rstrip("\n").split("\t")
            if i == 0 and parts and parts[0] == "grammar":
                continue
            if len(parts) >= 2 and parts[0]:
                rows[parts[0]] = parts[1]
    return rows


def is_assessed_non_clean_cause(cause):
    if not cause or cause == "CLEAN" or "-" not in cause:
        return False
    prefix, suffix = cause.split("-", 1)
    if suffix.rstrip("?") in UNKNOWN_CAUSES:
        return False
    return prefix in ("III", "IV")


def check_clean_classification_consistency(clean, classification):
    clean_with_non_clean_row = sorted(
        (g, classification[g]) for g in clean
        if g in classification and classification[g] != "CLEAN"
    )
    clean_rows_absent_from_ratchet = sorted(
        g for g, t in classification.items() if t == "CLEAN" and g not in clean
    )
    if not clean_with_non_clean_row and not clean_rows_absent_from_ratchet:
        return

    print("TIER DATA HYGIENE VIOLATION:")
    if clean_with_non_clean_row:
        print("  clean_grammars.txt entries with non-CLEAN classification rows:")
        for g, t in clean_with_non_clean_row:
            print(f"    {g}: {t}")
    if clean_rows_absent_from_ratchet:
        print("  tier_classification.tsv CLEAN rows absent from clean_grammars.txt:")
        for g in clean_rows_absent_from_ratchet:
            print(f"    {g}")
    sys.exit(1)


def current_tiers():
    """Current published tier per grammar: tiers.json tier, IV if absent.

    PARITY IS STILL THE HARD CLEAN EVIDENCE: I/II/unranked require
    clean_grammars.txt membership. Tier III also allows non-clean grammars
    whose remaining work is assessed and cause-coded (III-* preferred; old
    assessed IV-* accepted during the taxonomy migration). Tier IV is reserved
    for unknown, unassessed, or unclassified work.
    """
    clean = clean_set()
    classification = classification_tiers()
    tiers = json.load(open(TIERS))
    out = {}
    violations = []
    for x in tiers["grammars"]:
        n = x["grammar"]
        t = x.get("tier", "IV")
        if n not in clean:
            cause = x.get("cause") or x.get("iv_cause") or classification.get(n, "")
            if t in ("I", "II", "unranked"):
                violations.append((n, t, cause, "non-clean grammar published as clean-only tier"))
            elif t == "III" and not is_assessed_non_clean_cause(cause):
                violations.append((n, t, cause, "non-clean Tier III lacks assessed cause"))
        out[n] = t if t in RANK else "IV"
    if violations:
        print("TIER DATA HYGIENE VIOLATION:")
        for n, t, cause, msg in sorted(violations):
            print(f"  {n}: tier={t} cause={cause or '<missing>'} — {msg}")
        sys.exit(1)
    return out


def main():
    bump = "--bump" in sys.argv
    clean = clean_set()
    check_clean_classification_consistency(clean, classification_tiers())
    floor = {k: v["tier"] for k, v in json.load(open(FLOOR)).items()}
    cur = current_tiers()
    regressions, lifts = [], []
    for n, ft in floor.items():
        ct = cur.get(n, "IV")
        if RANK[ct] < RANK[ft]:
            regressions.append((n, ft, ct))
        elif RANK[ct] > RANK[ft]:
            lifts.append((n, ft, ct))
    if lifts:
        print(f"LIFTS ({len(lifts)}): " + ", ".join(f"{n} {a}->{b}" for n, a, b in sorted(lifts)))
    if regressions:
        print(f"\nRATCHET VIOLATION — {len(regressions)} grammar(s) below floor:")
        for n, ft, ct in sorted(regressions):
            print(f"  {n}: floor={ft} current={ct}")
        if not bump:
            sys.exit(1)
    if bump:
        full = json.load(open(FLOOR))
        for n, ct in cur.items():
            if n in full and RANK[ct] > RANK[full[n]["tier"]]:
                full[n]["tier"] = ct
        with open(FLOOR, "w") as f:
            json.dump(full, f, indent=2)
            f.write("\n")
        print(f"\nfloor ratcheted up ({len(lifts)} lifts applied)")
    else:
        print("\nratchet OK — no grammar below floor" if not regressions else "")


if __name__ == "__main__":
    main()
