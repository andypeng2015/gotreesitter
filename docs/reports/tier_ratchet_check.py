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

The committed floor lives in ``tier_floors.json`` (one ``{"tier": ...,
"state": ...}`` entry per grammar). The gate exits non-zero if any grammar's
current state is BELOW its floor. ``--bump`` ratchets the floor up to the
current tiers/states (use only after a verified, published lift, or to backfill
floor state during taxonomy migrations).

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
# C language. In the mixed taxonomy, a flat tier rank is not enough:
# - clean unranked is progress from assessed non-clean III;
# - clean unranked is regression from clean perf-measured III.
VALID_TIERS = {"I", "II", "III", "unranked", "IV"}
CLEAN_PERF_RANK = {"I": 5, "II": 4, "III": 3, "unranked": 2}
ASSESSED_NON_CLEAN_RANK = 1
UNKNOWN_RANK = 0


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


def tier_state(tier, is_clean, cause=""):
    """Classify a tier into the state that the ratchet actually compares."""
    if tier not in VALID_TIERS:
        return "unknown"
    if is_clean:
        if tier in CLEAN_PERF_RANK:
            return "clean_perf" if tier != "unranked" else "clean_unranked"
        return "unknown"
    if tier == "III" and is_assessed_non_clean_cause(cause):
        return "assessed_non_clean"
    return "unknown"


def state_rank(tier, state):
    if state in ("clean_perf", "clean_unranked"):
        return CLEAN_PERF_RANK.get(tier, UNKNOWN_RANK)
    if state == "assessed_non_clean":
        return ASSESSED_NON_CLEAN_RANK
    return UNKNOWN_RANK


def compare_floor_current(floor_tier, floor_state, current_tier, current_state):
    """Return -1 for regression, 0 for equal floor, 1 for lift/progress."""
    floor_rank = state_rank(floor_tier, floor_state)
    current_rank = state_rank(current_tier, current_state)
    if current_rank < floor_rank:
        return -1
    if current_rank > floor_rank:
        return 1
    return 0


def run_self_tests():
    cases = [
        # Assessed non-clean III becoming clean perf-pending is progress.
        ("III", "assessed_non_clean", "unranked", "clean_unranked", 1),
        # Clean perf-measured III losing perf evidence is still a regression.
        ("III", "clean_perf", "unranked", "clean_unranked", -1),
        # Non-clean cannot satisfy a clean unranked floor.
        ("unranked", "clean_unranked", "III", "assessed_non_clean", -1),
        # Ordinary clean perf ratchet remains monotonic.
        ("II", "clean_perf", "I", "clean_perf", 1),
        ("II", "clean_perf", "III", "clean_perf", -1),
    ]
    failures = []
    for case in cases:
        got = compare_floor_current(*case[:4])
        if got != case[4]:
            failures.append((case, got))
    if failures:
        print("SELF-TEST FAILURES:")
        for case, got in failures:
            print(f"  {case[:4]}: got {got}, want {case[4]}")
        sys.exit(1)
    print("self-test OK")


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
    """Current published tier state per grammar: tiers.json tier, IV if absent.

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
        if t not in VALID_TIERS:
            t = "IV"
        cause = x.get("cause") or x.get("iv_cause") or classification.get(n, "")
        is_clean = n in clean
        if n not in clean:
            if t in ("I", "II", "unranked"):
                violations.append((n, t, cause, "non-clean grammar published as clean-only tier"))
            elif t == "III" and not is_assessed_non_clean_cause(cause):
                violations.append((n, t, cause, "non-clean Tier III lacks assessed cause"))
        out[n] = {
            "tier": t,
            "state": tier_state(t, is_clean, cause),
            "cause": cause,
        }
    if violations:
        print("TIER DATA HYGIENE VIOLATION:")
        for n, t, cause, msg in sorted(violations):
            print(f"  {n}: tier={t} cause={cause or '<missing>'} — {msg}")
        sys.exit(1)
    return out


def main():
    if "--self-test" in sys.argv:
        run_self_tests()
        return
    bump = "--bump" in sys.argv
    clean = clean_set()
    classification = classification_tiers()
    check_clean_classification_consistency(clean, classification)
    floor_full = json.load(open(FLOOR))
    cur = current_tiers()
    regressions, lifts = [], []
    for n, floor_entry in floor_full.items():
        ft = floor_entry["tier"]
        fs = floor_entry.get("state") or tier_state(ft, n in clean, classification.get(n, ""))
        current = cur.get(n, {"tier": "IV", "state": "unknown"})
        ct, cs = current["tier"], current["state"]
        cmp = compare_floor_current(ft, fs, ct, cs)
        if cmp < 0:
            regressions.append((n, ft, ct))
        elif cmp > 0:
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
        full = floor_full
        for n, current in cur.items():
            if n not in full:
                continue
            ft = full[n]["tier"]
            fs = full[n].get("state") or tier_state(ft, n in clean, classification.get(n, ""))
            if compare_floor_current(ft, fs, current["tier"], current["state"]) > 0:
                full[n]["tier"] = current["tier"]
                full[n]["state"] = current["state"]
            elif "state" not in full[n]:
                full[n]["state"] = fs
        with open(FLOOR, "w") as f:
            json.dump(full, f, indent=2)
            f.write("\n")
        print(f"\nfloor ratcheted up ({len(lifts)} lifts applied)")
    else:
        print("\nratchet OK — no grammar below floor" if not regressions else "")


if __name__ == "__main__":
    main()
