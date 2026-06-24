#!/usr/bin/env python3
"""Perf-tier ratchet gate: fail if any grammar regressed below its floor tier.

Current state is read from the canonical published artifacts, NOT a perf rerun:

  * current — ``cgo_harness/tier_scan/tier_classification.tsv``. A grammar is
    currently eligible for tiers I/II/III iff its row is ``CLEAN``. An assessed
    ``IV-*`` row is current non-clean scan truth, even when the grammar is still
    listed in the historical clean ratchet.
  * floor   — ``cgo_harness/tier_scan/clean_grammars.txt`` is the parity-clean
    ratchet/floor. It can make a current ``IV-*`` row a regression, but it does
    not make that row contradictory.
  * tier    — ``docs/reports/tiers.json`` (the published per-release tier
    table). The current performance tier of a CLEAN grammar is tiers.json's
    tier for ``g`` (``unranked`` if absent or not ranked). The current tier of
    any assessed ``IV-*`` grammar is forced to ``IV`` regardless of what
    tiers.json says — parity-vs-C is the hard gate, full stop.

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

# Tiers are Roman numerals (I best .. IV worst) so "III" never collides with the
# C language. `unranked` is parity-clean with perf pending: better than IV, but
# not yet eligible for a perf tier.
RANK = {"I": 4, "II": 3, "III": 2, "unranked": 1, "IV": 0}  # higher is better


def clean_set():
    """Grammars that passed full-corpus parity vs the C oracle (the hard gate)."""
    with open(CLEAN) as f:
        return {ln.strip() for ln in f if ln.strip()}


def classification_rows():
    """Classification TSV tier by grammar plus duplicate row diagnostics."""
    rows = {}
    duplicates = set()
    with open(CLASS_TSV) as f:
        for i, ln in enumerate(f):
            parts = ln.rstrip("\n").split("\t")
            if i == 0 and parts and parts[0] == "grammar":
                continue
            if len(parts) >= 2 and parts[0]:
                if parts[0] in rows:
                    duplicates.add(parts[0])
                rows[parts[0]] = parts[1]
    return rows, sorted(duplicates)


def check_classification_hygiene(classification, clean, duplicates):
    invalid = sorted(
        (g, t) for g, t in classification.items()
        if t != "CLEAN" and (not t.startswith("IV-") or t.startswith("IV-unassessed"))
    )
    clean_without_ratchet = sorted(
        g for g, t in classification.items() if t == "CLEAN" and g not in clean
    )
    if not duplicates and not invalid and not clean_without_ratchet:
        return

    print("TIER DATA HYGIENE VIOLATION:")
    if duplicates:
        print("  duplicate tier_classification.tsv rows:")
        for g in duplicates:
            print(f"    {g}")
    if invalid:
        print("  tier_classification.tsv rows must be CLEAN or an assessed IV-* cause:")
        for g, t in invalid:
            print(f"    {g}: {t or 'missing'}")
    if clean_without_ratchet:
        print("  tier_classification.tsv CLEAN rows absent from clean_grammars.txt:")
        for g in clean_without_ratchet:
            print(f"    {g}")
    sys.exit(1)


def raise_clean_floor(floor, clean):
    """Apply the parity-clean ratchet as an effective minimum floor."""
    for g in clean:
        if RANK.get(floor.get(g, "IV"), 0) < RANK["unranked"]:
            floor[g] = "unranked"


def current_tiers(classification):
    """Current published tier per grammar: tiers.json tier, IV if absent.

    PARITY IS A HARD GATE (2026-06-08): a grammar whose tree diverges from the C
    oracle is POISONED — untrustworthy regardless of speed — and is tier IV,
    full stop. Only grammars classified CLEAN in tier_classification.tsv are
    ranked I/II/III. We therefore clamp any current IV-* grammar to IV even if
    tiers.json happens to list it higher.
    """
    tiers = json.load(open(TIERS))
    published = {x["grammar"]: x.get("tier", "IV") for x in tiers["grammars"]}
    out = {g: "IV" for g, t in classification.items() if t.startswith("IV-")}
    for g, t in classification.items():
        if t != "CLEAN":
            continue
        current = published.get(g, "unranked")
        out[g] = current if current in RANK and current != "IV" else "unranked"
    for x in tiers["grammars"]:
        n = x["grammar"]
        if n not in out:
            out[n] = "IV"
    return out


def main():
    bump = "--bump" in sys.argv
    clean = clean_set()
    classification, duplicates = classification_rows()
    check_classification_hygiene(classification, clean, duplicates)
    floor = {k: v["tier"] for k, v in json.load(open(FLOOR)).items()}
    raise_clean_floor(floor, clean)
    cur = current_tiers(classification)
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
        for g in clean:
            row = full.setdefault(g, {"tier": "IV"})
            if RANK.get(row.get("tier", "IV"), 0) < RANK["unranked"]:
                row["tier"] = "unranked"
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
