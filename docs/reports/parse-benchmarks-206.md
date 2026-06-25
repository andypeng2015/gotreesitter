# gotreesitter Parse Benchmark Report — all 206 grammars

> **A consumer planning guide, not a leaderboard.** The goal is to tell you what each bundled grammar *costs in practice* — distribution size, cold-load time, warm parse throughput, allocation behavior, and workload fit — so you can decide whether to embed all grammars, the core set, a narrow subset, or load blobs externally. gotreesitter's promise is pure-Go, no-CGo parsing with broad grammar coverage and flexible packaging.

| | |
|---|---|
| Commit | `0f7b1dc2` |
| Go version | go1.25.1 |
| Machine | Intel Core Ultra 9 285, linux (WSL2) |
| Date | 2026-06-08 |
| Grammars | 206 |
| Warm benchmark | `BenchmarkParityRealCorpusParse*`, GOMAXPROCS=1, count=3, 500ms (warm sweep); cold decode median of 5 |
| Corpus policy | real-world OSS files, lock-pinned (gotreesitter-corpora/corpus_sources); --order smallest --max-files 3; warm parse vs C tree-sitter |

## Summary

- **Workload-fit tiers (parity-gated):** I (clean + fast) **40** · II (clean, normal) **14** · III (clean, heavier) **3** · **IV (poisoned: diverges from C, or unmeasured) 149**
- **Trustworthy (parity-clean vs C, tiers I–III): 57 / 206.** The rest diverge from the C oracle — their trees are not yet reliable.
- **Parse status:** parity-clean **57** · timing-only (measured but parity diverges) **149** · unmeasured **0**
- **Median warm throughput:** 4.11 MB/s · **p95:** 14.9 MB/s
- **Median cold decode:** 0.78 ms
- **Heaviest cold load (decode):** swift 1253ms/5.0MB, verilog 34ms/653KB, cobol 26ms/629KB, fsharp 23ms/472KB, cpp 22ms/416KB
- **Biggest blobs (distribution cost):** swift 5.0MB, nim 661KB, verilog 653KB, cobol 629KB, sql 581KB
- **Slowest warm parse (×C):** dtd 692×, diff 365×, cooklang 166×, rst 139×, kdl 119×
- **Lowest allocations/op:** chatito 7, elisp 7, turtle 7, mermaid 8, blade 9, groovy 12

> ⚠️ The single biggest distribution outlier is **swift** (4.9MB blob, ~1.25s cold decode). Do not embed it in latency-sensitive or size-sensitive builds unless you need Swift.

## How to read this report

- **Distribution cost** = `blob_bytes`: how much the grammar adds to your binary if embedded. Don't ship all 206 unless you want all 206 — use a subset/core build for a productized CLI.
- **Cold-load cost** = `cold_decode_ms`: time to decode the blob into a usable language on first touch. Dominates for short-lived processes (pre-commit/git hooks, one-shot CLI/CI analyzers).
- **Warm (steady-state) cost** = `throughput_mb_s`, `warm_ns_op`, `allocs_op`, `bytes_op`: the per-parse cost once the language is loaded. Dominates for daemons/indexers/LSP.
- **`full_x_vs_c`** = full-parse time ÷ C tree-sitter on the same files (lower is faster; 1.0 = parity with C). **`edit_x_vs_c`** is the single-byte incremental edit lane — gotreesitter beats C (<1.0) on incremental for most grammars.
- **`external_scanner`**: hand-written scanner involvement; often explains tail/edge behavior.
- **`parse_status`**: `parity-clean` = byte-for-byte match with C on the corpus; `timing-only` = timed via allow-mismatch but the tree still diverges from C somewhere (speed is real, fidelity is not yet guaranteed); `unmeasured` = no number yet (see notes).
- **Numbers are real-corpus oracle ratios, not universal truth. Benchmark your own corpus** — small config files inflate ratios via fixed per-parse overhead; large-corpus grammars (go, c, java, python, rust, typescript) are the trustworthy steady-state signal.

## Consumer recommendations

**CLI authors** — Use subset builds; avoid embedding all 206 unless you truly need universal coverage. **Only tiers I–III are parity-clean (trustworthy vs the C oracle); tier IV diverges from C — its trees are poisoned, do not ship them without verifying on your own corpus.** Among the trustworthy set, optimize for blob size + cold decode; prefer tier I/II.

**Repo indexers / AI code-context tools** — Warm throughput, `allocs_op`, and tail behavior matter more than cold load. Reuse parsers, cache loaded languages, batch your work. Tier III is fine here.

**Editor / LSP integrations** — Watch p95 latency and the incremental (`edit_x_vs_c`) lane, not just median full-parse throughput. Cold decode matters at startup; warm/incremental matters per keystroke.

**Security / static-analysis tools** — Correctness, query compatibility, and recovery matter as much as speed. Prefer `parity-clean` grammars; for `timing-only`/`unmeasured`, validate against your representative files before trusting structure.

**WASM / cross-compiled users** — The no-CGo story is the headline: one clean static binary, no toolchain. Expect grammar-specific variance, but vastly simpler builds.

## The three costs, separated

### 1. Distribution (binary size)

Choose a packaging mode deliberately: all grammars · core set · explicit subset (build tags) · external blobs (runtime directory). The biggest blobs to be aware of: **swift** 5.0MB, **nim** 661KB, **verilog** 653KB, **cobol** 629KB, **sql** 581KB, **scala** 473KB, **fsharp** 472KB, **cpp** 416KB. A dedicated `packaging-costs.md` (binary size per mode) is the planned companion to this report.

### 2. Cold start (first touch)

For hooks/one-shot CLIs, decode time is the cost. Most grammars decode in well under a millisecond (median 0.78 ms), but a long tail is expensive: **swift** 1253ms, **verilog** 34ms, **cobol** 26ms, **fsharp** 23ms, **cpp** 22ms, **nim** 21ms.

### 3. Steady state (warm parse)

For daemons/indexers, throughput + allocations dominate. Median warm throughput 4.11 MB/s, p95 14.9 MB/s. The incremental edit lane beats C for the large majority of grammars — gotreesitter's strength for editor-style workloads.

## Workload-fit tiers

Tiers describe *fit*, not quality. Every grammar here parses; the tier tells you which workloads it suits best.

| Tier | Meaning | Rough rule |
|---|---|---|
> **Parity is a hard gate.** A grammar whose tree diverges from the C oracle is *poisoned* — untrustworthy regardless of speed — and is **tier IV**, full stop. Only parity-clean grammars (they match tree-sitter C on full parse) are ranked I/II/III by performance.

| **I** | Parity-clean AND latency-friendly | parity-clean, ≤1.5× C, cold ≤5ms, blob ≤150KB |
| **II** | Parity-clean, normal | parity-clean, ≤8× C, cold ≤20ms |
| **III** | Parity-clean, heavier | parity-clean, slow (>8× C) or cold >20ms or blob >400KB |
| **IV** | **Poisoned or unmeasured — do not trust the tree** | diverges from C (any parity failure) OR unmeasured |

**Tier I — parity-clean + latency-friendly** (40):
> awk, brightscript, c, clojure, css, dhall, disassembly, erlang, gdscript, gleam, groovy, java, jinja2, jsonnet, linkerscript, llvm, lua, luau, mermaid, mojo, move, nickel, nix, pascal, pkl, powershell, prisma, prolog, proto, puppet, rescript, robot, smithy, squirrel, starlark, vhdl, wat, yaml, yuck, zig

**Tier III — parity-clean, heavier** (3):
> cpp, nim, sql

**Tier IV — poisoned (diverges from C) or unmeasured** (149):
> ada, agda, angular, apex, arduino, asm, astro, authzed, bass, beancount, bibtex, bicep, bitbake, blade, caddy, cairo, capnp, chatito, circom, cmake, cobol, comment, commonlisp, cooklang, corn, cpon, crystal, csv, cue, cylc, d, dart, desktop, devicetree, diff, djot, dockerfile, dot, doxygen, dtd, earthfile, ebnf, editorconfig, eds, eex, elisp, elm, elsa, embedded_template, enforce, facility, faust, fennel, fidl, firrtl, fish, foam, forth, fortran, fsharp, git_config, git_rebase, gitcommit, gitignore, glsl, gn, go, godot_resource, gomod, graphql, hack, hare, haxe, hcl, heex, html, http, hurl, hyprlang, ini, janet, javascript, jq, jsdoc, json, just, kconfig, kdl, kotlin, ledger, less, liquid, make, markdown, markdown_inline, matlab, meson, nginx, ninja, norg, nushell, odin, org, pem, perl, php, promql, properties, pug, python, ql, r, racket, regex, rego, requirements, ron, rst, rust, scala, scheme, scss, solidity, sparql, ssh_config, svelte, swift, tablegen, tcl, teal, templ, textproto, thrift, tlaplus, tmux, todotxt, toml, tsx, turtle, twig, typescript, typst, uxntal, verilog, vimdoc, vue, wgsl, wolfram, xml

## Full table (206 grammars)

Sortable/filterable copies: `parse-benchmarks-206.csv`, `parse-benchmarks-206.json`. blob = distribution cost; cold = decode ms; MB/s & ×C = warm parse; allocs/B per op = memory.

| Grammar | Tier | Status | Blob | Cold ms | MB/s | Full ×C | Edit ×C | allocs/op | B/op | Ext.scan | Notes |
|---|:--:|---|--:|--:|--:|--:|--:|--:|--:|:--:|---|
| awk | I | parity-clean | 63KB | 2.26 | 6.5 | 0.93 | 5.53 | 44 | 8552 | yes | GSS-forest fast path (default-on); external scanner |
| brightscript | I | parity-clean | 35KB | 1.00 | 12.6 | 0.82 | 0.48 | 32 | 2320 |  |  |
| c | I | parity-clean | 66KB | 3.37 | 10.9 | 0.95 | 0.04 | 7305 | 1228285 |  |  |
| clojure | I | parity-clean | 16KB | 0.50 | 6.1 | 0.93 | 0.33 | 21 | 2184 |  |  |
| css | I | parity-clean | 14KB | 0.53 | 9.9 | 0.95 | 0.26 | 1443 | 312281 | yes | GSS-forest fast path (default-on); external scanner |
| dhall | I | parity-clean | 27KB | 0.91 | 12.3 | 0.94 | 0.32 | 25 | 2232 | yes | external scanner |
| disassembly | I | parity-clean | 3KB | 0.12 | 8.4 | 0.96 | 1.05 | 253 | 17410 | yes | external scanner |
| erlang | I | parity-clean | 35KB | 1.15 | 6.8 | 0.92 | 0.01 | 237 | 81207 | yes | GSS-forest fast path (default-on); external scanner |
| gdscript | I | parity-clean | 35KB | 1.02 | 5.0 | 0.99 | 2.07 | 31 | 2384 | yes | external scanner |
| gleam | I | parity-clean | 49KB | 1.34 | 4.7 | 0.94 | 4.05 | 27 | 3280 | yes | external scanner |
| groovy | I | parity-clean | 84KB | 2.70 | 2.9 | 0.98 | 0.10 | 12 | 1336 |  |  |
| java | I | parity-clean | 47KB | 2.14 | 14.4 | 0.95 | 0.85 | 12 | 1352 |  |  |
| jinja2 | I | parity-clean | 3KB | 0.12 | 15.3 | 0.41 | 0.41 | 17 | 864 |  |  |
| jsonnet | I | parity-clean | 10KB | 0.33 | 8.1 | 0.92 | 5.90 | 31 | 2352 | yes | external scanner |
| linkerscript | I | parity-clean | 15KB | 0.42 | 14.9 | 0.46 | 0.03 | 59 | 3104 |  |  |
| llvm | I | parity-clean | 86KB | 4.27 | 11.8 | 0.95 | 1.11 | 35 | 2424 |  |  |
| lua | I | parity-clean | 10KB | 0.31 | 7.1 | 0.61 | 0.09 | 25 | 3016 |  |  |
| luau | I | parity-clean | 17KB | 0.47 | 7.8 | 0.75 | 0.69 | 29 | 2288 | yes | external scanner |
| mermaid | I | parity-clean | 28KB | 0.96 | 3.6 | 0.90 | 0.50 | 8 | 1240 |  |  |
| mojo | I | parity-clean | 51KB | 2.60 | 59.1 | 0.24 | 0.18 | 81 | 5790 | yes | external scanner |
| move | I | parity-clean | 12KB | 0.41 | 9.8 | 0.37 | 0.49 | 28 | 2264 |  |  |
| nickel | I | parity-clean | 24KB | 0.74 | 5.6 | 0.61 | 0.31 | 39 | 2504 | yes | external scanner |
| nix | I | parity-clean | 12KB | 0.39 | 6.1 | 0.73 | 2.30 | 108 | 22328 | yes | GSS-forest fast path (default-on); external scanner |
| pascal | I | parity-clean | 76KB | 2.26 | 4.3 | 0.87 | 0.33 | 62 | 4280 |  |  |
| pkl | I | parity-clean | 31KB | 0.86 | 5.2 | 0.91 | 0.38 | 21 | 2184 | yes | external scanner |
| powershell | I | parity-clean | 68KB | 3.07 | 11.1 | 0.89 | 0.43 | 14 | 1456 | yes | external scanner |
| prisma | I | parity-clean | 7KB | 0.21 | 14.3 | 0.78 | 0.59 | 82 | 9296 |  | GSS-forest fast path (default-on) |
| prolog | I | parity-clean | 7KB | 0.23 | 3.5 | 0.83 | 0.06 | 19 | 976 |  |  |
| proto | I | parity-clean | 9KB | 0.33 | 9.0 | 0.93 | 0.46 | 32 | 2504 |  |  |
| puppet | I | parity-clean | 15KB | 0.48 | 7.4 | 0.98 | 0.11 | 21 | 2184 |  |  |
| rescript | I | parity-clean | 82KB | 2.49 | 3.2 | 0.64 | 0.29 | 20 | 1520 | yes | external scanner |
| robot | I | parity-clean | 13KB | 0.44 | 9.1 | 0.85 | 0.65 | 21 | 2184 |  |  |
| smithy | I | parity-clean | 11KB | 0.32 | 10.2 | 0.70 | 0.17 | 31 | 2352 |  |  |
| squirrel | I | parity-clean | 41KB | 1.15 | 5.5 | 0.80 | 2.53 | 110 | 22864 | yes | GSS-forest fast path (default-on); external scanner |
| starlark | I | parity-clean | 48KB | 1.56 | 3.8 | 0.82 | 0.19 | 42 | 2704 | yes | external scanner |
| vhdl | I | parity-clean | 150KB | 3.66 | 8.8 | 0.06 | 0.04 | 59 | 3016 | yes | external scanner |
| wat | I | parity-clean | 24KB | 0.75 | 7.9 | 0.51 | 0.69 | 34 | 2352 |  |  |
| yaml | I | parity-clean | 25KB | 0.65 | 9.9 | 0.58 | 2.09 | 511 | 10104 | yes | external scanner |
| yuck | I | parity-clean | 5KB | 0.19 | 0.8 | 1.25 | 0.07 | 25 | 2723 | yes | external scanner |
| zig | I | parity-clean | 45KB | 1.93 | 9.8 | 0.90 | 0.17 | 42 | 2632 |  |  |
| bash | II | parity-clean | 153KB | 4.71 | 6.2 | 0.50 | 1.14 | 1347 | 65872 | yes | GSS-forest fast path (default-on); external scanner |
| c_sharp | II | parity-clean | 291KB | 11.89 | 12.1 | 0.98 | 13.59 | 561 | 112794 | yes | GSS-forest fast path (default-on); external scanner |
| cuda | II | parity-clean | 312KB | 18.73 | 54.5 | 0.90 | 0.42 | 36 | 2456 | yes | external scanner |
| elixir | II | parity-clean | 213KB | 12.13 | 4.7 | 0.65 | 1.08 | 34 | 2432 | yes | external scanner |
| gitattributes | II | parity-clean | 5KB | 0.21 | 0.1 | 1.82 | 0.08 | 37 | 4084 |  | GSS-forest fast path (default-on) |
| haskell | II | parity-clean | 291KB | 10.38 | 6.2 | 0.69 | 0.45 | 60 | 4208 | yes | external scanner |
| hlsl | II | parity-clean | 304KB | 18.64 | 10.1 | 0.90 | 0.38 | 43 | 2616 | yes | external scanner |
| json5 | II | parity-clean | 8KB | 1.11 | 0.6 | 1.66 | 0.09 | 17 | 1473 |  |  |
| julia | II | parity-clean | 242KB | 14.49 | 11.3 | 0.77 | 1.87 | 29 | 2280 | yes | external scanner |
| objc | II | parity-clean | 268KB | 14.14 | 3.3 | 0.72 | 0.24 | 47 | 4432 |  |  |
| ocaml | II | parity-clean | 255KB | 12.89 | 7.0 | 0.58 | 0.72 | 31 | 2392 | yes | external scanner |
| purescript | II | parity-clean | 245KB | 19.77 | 6.6 | 0.94 | 1.68 | 34 | 3432 | yes | external scanner |
| ruby | II | parity-clean | 148KB | 5.93 | 10.7 | 0.65 | 0.69 | 55 | 3808 | yes | external scanner |
| v | II | parity-clean | 165KB | 4.77 | 3.9 | 0.94 | 0.09 | 22 | 2696 |  |  |
| cpp | III | parity-clean | 416KB | 22.22 | 4.7 | 0.97 | 0.19 | 72 | 12200 | yes | external scanner |
| nim | III | parity-clean | 661KB | 21.14 | 3.5 | 0.62 | 0.66 | 18 | 1360 | yes | external scanner |
| sql | III | parity-clean | 581KB | 20.35 | 5.6 | 0.92 | 1.67 | 189 | 46904 | yes | external scanner |
| ada | IV | timing-only | 45KB | 1.79 | 1.9 | 3.48 | 0.04 | 39 | 4152 |  | parity-blocked (timing-only via allow-mismatch) |
| agda | IV | timing-only | 223KB | 7.66 | — | 1.06 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| angular | IV | timing-only | 20KB | 0.61 | 4.0 | 1.46 | 0.07 | 235 | 5520 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| apex | IV | timing-only | 144KB | 13.74 | — | 2.14 | — | — | — |  | parity-blocked (timing-only via allow-mismatch) |
| arduino | IV | timing-only | 277KB | 18.53 | 24.7 | 0.81 | 6.53 | 88 | 8976 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| asm | IV | timing-only | 6KB | 0.48 | 3.5 | 2.21 | 0.08 | 93 | 7416 |  | parity-blocked (timing-only via allow-mismatch) |
| astro | IV | timing-only | 5KB | 0.16 | 9.1 | 1.07 | 0.31 | 123 | 3216 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| authzed | IV | timing-only | 28KB | 3.77 | 0.1 | 1.84 | 0.87 | 5772 | 143192232 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| bass | IV | timing-only | 3KB | 0.14 | 1.5 | 3.61 | 0.12 | 41 | 4160 |  | parity-blocked (timing-only via allow-mismatch) |
| beancount | IV | timing-only | 47KB | 4.82 | 12.0 | 1.87 | 25.35 | 108 | 21186 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| bibtex | IV | timing-only | 3KB | 0.13 | 9.4 | 0.95 | 1.65 | 100 | 9952 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| bicep | IV | timing-only | 19KB | 0.70 | 11.1 | 1.10 | 0.98 | 40 | 2651 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| bitbake | IV | timing-only | 88KB | 13.23 | 1.9 | 3.06 | 1.33 | 431 | 42148 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| blade | IV | timing-only | 197KB | 6.70 | — | 1.54 | — | 9 | 760 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| caddy | IV | timing-only | 16KB | 0.98 | 3.8 | 1.91 | 0.17 | 153 | 14240 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| cairo | IV | timing-only | 42KB | 1.28 | 5.2 | 2.40 | 0.33 | 30 | 2864 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| capnp | IV | timing-only | 13KB | 0.47 | 12.6 | 3.02 | 2.24 | 74 | 8640 |  | parity-blocked (timing-only via allow-mismatch) |
| chatito | IV | timing-only | 4KB | 0.15 | 3.3 | 2.51 | 0.28 | 7 | 728 |  | parity-blocked (timing-only via allow-mismatch) |
| circom | IV | timing-only | 9KB | 0.33 | 2.0 | 4.36 | 0.22 | 38 | 4128 |  | parity-blocked (timing-only via allow-mismatch) |
| cmake | IV | timing-only | 13KB | 0.38 | 0.5 | 4.75 | 0.27 | 79 | 33291 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| cobol | IV | timing-only | 629KB | 25.96 | — | 1.46 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| comment | IV | timing-only | 2KB | 0.09 | — | 29.93 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| commonlisp | IV | timing-only | 89KB | 2.67 | 0.4 | 1.66 | 0.36 | 37 | 4072 |  | parity-blocked (timing-only via allow-mismatch) |
| cooklang | IV | timing-only | 11KB | 0.52 | — | 166.16 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| corn | IV | timing-only | 4KB | 0.12 | 0.9 | 4.66 | 0.36 | 215 | 25484 |  | parity-blocked (timing-only via allow-mismatch) |
| cpon | IV | timing-only | 4KB | 0.13 | 2.2 | 3.68 | 0.13 | 29 | 3400 |  | parity-blocked (timing-only via allow-mismatch) |
| crystal | IV | timing-only | 268KB | 8.05 | 0.5 | 1.56 | — | 281 | 24409 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| csv | IV | timing-only | 2KB | 0.09 | 2.9 | 1.09 | 3.59 | 60 | 7072 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| cue | IV | timing-only | 33KB | 2.41 | — | 1.03 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| cylc | IV | timing-only | 22KB | 1.45 | 0.8 | 5.02 | 0.49 | 33 | 3464 |  | parity-blocked (timing-only via allow-mismatch) |
| d | IV | timing-only | 237KB | 9.88 | 6.6 | 1.60 | 7.06 | 66 | 6432 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| dart | IV | timing-only | 92KB | 3.86 | 16.1 | 1.27 | 1.80 | 51 | 2912 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| desktop | IV | timing-only | 3KB | 0.11 | 0.1 | 56.83 | 0.01 | 93 | 271994898 |  | parity-blocked (timing-only via allow-mismatch) |
| devicetree | IV | timing-only | 21KB | 0.60 | 7.2 | 1.61 | 0.36 | 33 | 2384 |  | parity-blocked (timing-only via allow-mismatch) |
| diff | IV | timing-only | 5KB | 0.19 | 0.0 | 365.22 | 0.08 | 536 | 981485344 |  | parity-blocked (timing-only via allow-mismatch) |
| djot | IV | timing-only | 49KB | 1.51 | 0.2 | 3.52 | 9.08 | 187 | 13170 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| dockerfile | IV | timing-only | 12KB | 0.38 | 1.2 | 17.68 | 31.35 | 154 | 16814 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| dot | IV | timing-only | 9KB | 0.26 | 1.9 | 2.60 | 0.40 | 36 | 4032 |  | parity-blocked (timing-only via allow-mismatch) |
| doxygen | IV | timing-only | 9KB | 0.39 | 21.9 | 1.06 | 0.91 | 21 | 2184 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| dtd | IV | timing-only | 8KB | 0.32 | 0.0 | 691.73 | 0.05 | 561 | 587192 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| earthfile | IV | timing-only | 67KB | 2.36 | 1.6 | 2.48 | 0.07 | 43 | 4208 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| ebnf | IV | timing-only | 2KB | 0.07 | 1.9 | 14.49 | 0.04 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| editorconfig | IV | timing-only | 3KB | 0.12 | 0.9 | 7.32 | 0.33 | 13 | 1368 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| eds | IV | timing-only | 2KB | 0.07 | 0.5 | 19.72 | 7.42 | 57 | 6424 |  | parity-blocked (timing-only via allow-mismatch) |
| eex | IV | timing-only | 2KB | 0.09 | 1.0 | 9.39 | 0.40 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| elisp | IV | timing-only | 5KB | 0.31 | 13.9 | 1.40 | 0.66 | 7 | 728 |  | parity-blocked (timing-only via allow-mismatch) |
| elm | IV | timing-only | 54KB | 2.13 | 1.1 | 4.04 | 1.04 | 32 | 2882 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| elsa | IV | timing-only | 2KB | 0.10 | 5.0 | 1.13 | — | 21 | 2184 |  | parity-blocked (timing-only via allow-mismatch) |
| embedded_template | IV | timing-only | 2KB | 0.09 | 1.1 | 15.04 | 0.54 | 4680 | 519785 |  | parity-blocked (timing-only via allow-mismatch) |
| enforce | IV | timing-only | 22KB | 0.65 | 5.5 | 1.46 | 0.07 | 31 | 2872 |  | parity-blocked (timing-only via allow-mismatch) |
| facility | IV | timing-only | 5KB | 0.22 | 1.1 | 17.97 | 0.06 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| faust | IV | timing-only | 21KB | 0.83 | 6.1 | 0.74 | 2.15 | 90 | 9640 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| fennel | IV | timing-only | 33KB | 1.21 | 2.0 | 1.96 | 5.41 | 32 | 3408 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| fidl | IV | timing-only | 9KB | 0.33 | 9.0 | 1.06 | 0.51 | 21 | 2184 |  | parity-blocked (timing-only via allow-mismatch) |
| firrtl | IV | timing-only | 13KB | 0.40 | 6.2 | 1.27 | 0.46 | 42 | 2448 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| fish | IV | timing-only | 12KB | 0.41 | 6.0 | 1.04 | 1.23 | 80 | 15712 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| foam | IV | timing-only | 9KB | 0.29 | 18.9 | 1.52 | 2.74 | 48 | 4248 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| forth | IV | timing-only | 8KB | 0.38 | 1.4 | 7.44 | 0.11 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| fortran | IV | timing-only | 299KB | 13.21 | 8.3 | 1.01 | — | 29 | 5432 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| fsharp | IV | timing-only | 472KB | 23.41 | 0.5 | 5.03 | 0.05 | 44 | 3144 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| git_config | IV | timing-only | 3KB | 0.13 | 2.0 | 7.26 | 0.39 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| git_rebase | IV | timing-only | 2KB | 0.08 | 2.6 | 3.54 | 0.29 | 118 | 12696 |  | parity-blocked (timing-only via allow-mismatch) |
| gitcommit | IV | timing-only | 70KB | 4.59 | 14.3 | 1.30 | 2.24 | 22 | 2696 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| gitignore | IV | timing-only | 3KB | 0.13 | 1.1 | 1.78 | 2.02 | 30 | 3776 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| glsl | IV | timing-only | 87KB | 3.70 | 2.9 | 1.04 | 0.23 | 63 | 4992 |  | parity-blocked (timing-only via allow-mismatch) |
| gn | IV | timing-only | 6KB | 0.17 | 5.2 | 2.02 | 0.31 | 40 | 3112 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| go | IV | timing-only | 371KB | 13.71 | 3.9 | 1.70 | 0.02 | 110 | 35377 |  | parity-blocked (timing-only via allow-mismatch) |
| godot_resource | IV | timing-only | 3KB | 0.12 | 4.2 | 2.23 | 0.39 | 22 | 2696 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| gomod | IV | timing-only | 5KB | 0.17 | 1.3 | 10.95 | 1.06 | 2684 | 286925 |  | parity-blocked (timing-only via allow-mismatch) |
| graphql | IV | timing-only | 11KB | 0.37 | 4.8 | 4.65 | 4.42 | 49 | 2470 |  | parity-blocked (timing-only via allow-mismatch) |
| hack | IV | timing-only | 136KB | 4.79 | 6.7 | 1.81 | — | 20 | 1568 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| hare | IV | timing-only | 16KB | 0.50 | — | 4.23 | — | — | — |  | parity-blocked (timing-only via allow-mismatch) |
| haxe | IV | timing-only | 75KB | 2.01 | 3.3 | 1.53 | 1.14 | 85 | 4000 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| hcl | IV | timing-only | 20KB | 1.29 | 1.9 | 1.43 | 0.22 | 40 | 4056 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| heex | IV | timing-only | 5KB | 0.18 | 3.3 | 3.23 | 0.23 | 103 | 11104 |  | parity-blocked (timing-only via allow-mismatch) |
| html | IV | timing-only | 3KB | 0.14 | 4.8 | 1.63 | 0.32 | 264 | 5424 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| http | IV | timing-only | 24KB | 1.17 | 0.1 | 20.23 | 0.05 | 94 | 9928 |  | parity-blocked (timing-only via allow-mismatch) |
| hurl | IV | timing-only | 25KB | 0.88 | 0.1 | 41.75 | 0.17 | 39 | 4108 |  | parity-blocked (timing-only via allow-mismatch) |
| hyprlang | IV | timing-only | 6KB | 0.25 | 3.0 | 10.34 | 0.23 | 23 | 2448 |  | parity-blocked (timing-only via allow-mismatch) |
| ini | IV | timing-only | 2KB | 0.08 | 4.9 | 3.78 | 0.15 | 47 | 5392 |  | parity-blocked (timing-only via allow-mismatch) |
| janet | IV | timing-only | 4KB | 0.15 | 1.6 | 6.27 | 0.39 | 38 | 4112 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| javascript | IV | timing-only | 41KB | 1.40 | 5.7 | 1.22 | 0.13 | 852 | 44024036 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| jq | IV | timing-only | 8KB | 0.35 | 2.1 | 3.72 | 0.26 | 132 | 13760 |  | parity-blocked (timing-only via allow-mismatch) |
| jsdoc | IV | timing-only | 5KB | 0.31 | 8.4 | 2.18 | 3.33 | 21 | 2184 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| json | IV | timing-only | 2KB | 0.12 | 9.9 | 1.79 | 4.39 | 7557 | 494842 |  | parity-blocked (timing-only via allow-mismatch) |
| just | IV | timing-only | 10KB | 0.31 | 6.3 | 4.56 | 0.17 | 53 | 4320 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| kconfig | IV | timing-only | 10KB | 0.35 | 2.8 | 2.94 | 2.02 | 34 | 4024 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| kdl | IV | timing-only | 16KB | 0.61 | 0.0 | 118.77 | 142.12 | 1021 | 50052 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| kotlin | IV | timing-only | 337KB | 20.98 | 4.6 | 0.77 | — | 128 | 17960 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| ledger | IV | timing-only | 45KB | 8.41 | 0.0 | 2.62 | 0.05 | 13 | 1417 |  | parity-blocked (timing-only via allow-mismatch) |
| less | IV | timing-only | 14KB | 0.55 | 0.2 | 27.29 | 83.66 | 25 | 2152 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| liquid | IV | timing-only | 22KB | 0.65 | 1.8 | 9.59 | 50.43 | 134 | 14411 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| make | IV | timing-only | 25KB | 0.75 | 0.2 | 0.89 | 3.06 | 470 | 48216 |  | GSS-forest fast path (default-on); parity-blocked (timing-only via allow-mismatch) |
| markdown | IV | timing-only | 36KB | 1.38 | 0.7 | 6.22 | 9.59 | 28 | 3824 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| markdown_inline | IV | timing-only | 41KB | 1.66 | 0.4 | 9.68 | 15.13 | 27 | 3732 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| matlab | IV | timing-only | 52KB | 1.77 | 2.5 | 1.20 | 7.23 | 156 | 7192 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| meson | IV | timing-only | 22KB | 0.62 | 0.9 | 4.23 | 0.07 | 50 | 7912 |  | parity-blocked (timing-only via allow-mismatch) |
| nginx | IV | timing-only | 45KB | 11.96 | 0.1 | 32.78 | 0.16 | 14 | 1378 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| ninja | IV | timing-only | 7KB | 0.23 | 0.6 | 12.55 | 0.21 | 59 | 5449 |  | parity-blocked (timing-only via allow-mismatch) |
| norg | IV | timing-only | 227KB | 8.58 | — | 8.80 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| nushell | IV | timing-only | 204KB | 12.54 | 0.9 | 3.25 | 6.33 | 53 | 5232 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| odin | IV | timing-only | 181KB | 6.05 | 6.3 | 1.65 | 2.16 | 33 | 2904 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| org | IV | timing-only | 105KB | 3.99 | 0.2 | 1.67 | 2.04 | 53 | 4418 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| pem | IV | timing-only | 2KB | 0.10 | 11.5 | 2.63 | 0.20 | 12 | 1336 |  | parity-blocked (timing-only via allow-mismatch) |
| perl | IV | timing-only | 206KB | 16.22 | 1.6 | 1.86 | 0.29 | 46 | 3936 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| php | IV | timing-only | 95KB | 3.04 | 6.4 | 1.06 | 0.01 | 219 | 23512 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| promql | IV | timing-only | 6KB | 0.19 | — | 6.28 | — | — | — |  | parity-blocked (timing-only via allow-mismatch) |
| properties | IV | timing-only | 3KB | 0.11 | 0.3 | 26.33 | 56.13 | 28 | 2754 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| pug | IV | timing-only | 30KB | 1.01 | — | 0.91 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| python | IV | timing-only | 60KB | 2.19 | 2.6 | 1.69 | 0.17 | 166 | 10454 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| ql | IV | timing-only | 32KB | 1.27 | 3.6 | 1.66 | 0.57 | 27 | 2840 |  | parity-blocked (timing-only via allow-mismatch) |
| r | IV | timing-only | 48KB | 2.11 | 4.0 | 1.06 | 2.00 | 27 | 2280 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| racket | IV | timing-only | 13KB | 0.55 | 5.4 | 1.08 | 0.61 | 62 | 6946 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| regex | IV | timing-only | 5KB | 0.17 | 0.2 | 9.46 | 0.08 | 15 | 1432 |  | parity-blocked (timing-only via allow-mismatch) |
| rego | IV | timing-only | 19KB | 0.55 | 6.5 | 1.06 | 0.25 | 63 | 6552 |  | parity-blocked (timing-only via allow-mismatch) |
| requirements | IV | timing-only | 8KB | 0.24 | 3.9 | 4.14 | 0.08 | 27 | 2840 |  | parity-blocked (timing-only via allow-mismatch) |
| ron | IV | timing-only | 10KB | 0.89 | 1.1 | 7.25 | 0.31 | 41 | 4248 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| rst | IV | timing-only | 8KB | 0.25 | — | 138.86 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| rust | IV | timing-only | 113KB | 4.80 | 2.5 | 2.42 | 0.02 | 702 | 1159698 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| scala | IV | timing-only | 473KB | 12.85 | — | 1.69 | — | — | — | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| scheme | IV | timing-only | 10KB | 0.60 | 1.0 | 8.17 | 0.08 | 39 | 4152 |  | parity-blocked (timing-only via allow-mismatch) |
| scss | IV | timing-only | 19KB | 0.87 | 8.7 | 1.30 | 0.01 | 141 | 64105 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| solidity | IV | timing-only | 37KB | 1.40 | 12.5 | 1.13 | 0.18 | 36 | 2424 |  | parity-blocked (timing-only via allow-mismatch) |
| sparql | IV | timing-only | 31KB | 1.16 | 4.8 | 1.60 | 0.53 | 6916 | 670190 |  | parity-blocked (timing-only via allow-mismatch) |
| ssh_config | IV | timing-only | 33KB | 1.19 | 0.9 | 29.84 | 0.05 | 13 | 1384 |  | parity-blocked (timing-only via allow-mismatch) |
| svelte | IV | timing-only | 8KB | 0.24 | 7.7 | 1.02 | 1.90 | 142 | 3336 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| swift | IV | timing-only | 5.0MB | 1252.80 | 12.1 | 2.90 | 0.73 | 97 | 6668 | yes | external scanner; parity-blocked (timing-only via allow-mismatch); very large blob (5.0MB) — exclude from latency-sensitive builds; slow cold decode (1253ms) |
| tablegen | IV | timing-only | 13KB | 0.69 | 29.3 | 1.65 | 0.50 | 63 | 5352 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| tcl | IV | timing-only | 15KB | 0.76 | 2.1 | 2.42 | 0.09 | 39 | 4152 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| teal | IV | timing-only | 18KB | 0.82 | 1.4 | 3.90 | 6.22 | 52 | 5464 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| templ | IV | timing-only | 62KB | 5.56 | 5.4 | 1.13 | — | 25 | 1544 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| textproto | IV | timing-only | 5KB | 0.27 | 2.6 | 3.46 | 0.38 | 39 | 4152 |  | parity-blocked (timing-only via allow-mismatch) |
| thrift | IV | timing-only | 20KB | 1.11 | 56.7 | 1.13 | 0.32 | 63 | 6200 |  | parity-blocked (timing-only via allow-mismatch) |
| tlaplus | IV | timing-only | 229KB | 17.15 | 8.0 | 0.66 | — | 117 | 15504 | yes | GSS-forest fast path (default-on); external scanner; parity-blocked (timing-only via allow-mismatch) |
| tmux | IV | timing-only | 277KB | 12.61 | 3.3 | 8.58 | — | 228 | 182776 |  | parity-blocked (timing-only via allow-mismatch) |
| todotxt | IV | timing-only | 2KB | 0.08 | 1.2 | 7.60 | 0.14 | 13 | 1392 |  | parity-blocked (timing-only via allow-mismatch) |
| toml | IV | timing-only | 5KB | 0.19 | 4.1 | 1.87 | 0.20 | 19 | 1592 |  | parity-blocked (timing-only via allow-mismatch) |
| tsx | IV | timing-only | 124KB | 4.36 | 5.9 | 1.18 | — | 75 | 40432 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| turtle | IV | timing-only | 6KB | 0.24 | 3.6 | 1.07 | 0.07 | 7 | 728 |  | parity-blocked (timing-only via allow-mismatch) |
| twig | IV | timing-only | 13KB | 0.43 | 3.0 | 1.92 | 0.63 | 116 | 13504 |  | parity-blocked (timing-only via allow-mismatch) |
| typescript | IV | timing-only | 121KB | 4.08 | 6.7 | 1.70 | 0.03 | 71 | 79406 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| typst | IV | timing-only | 90KB | 3.20 | 0.1 | 12.38 | 23.69 | 74 | 4490 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| uxntal | IV | timing-only | 7KB | 0.22 | 0.4 | 16.28 | 52.99 | 117 | 10427 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| verilog | IV | timing-only | 653KB | 33.60 | 2.1 | 1.33 | 0.07 | 73 | 5264 |  | parity-blocked (timing-only via allow-mismatch) |
| vimdoc | IV | timing-only | 13KB | 0.79 | 0.2 | 1.27 | 0.20 | 197 | 17860 |  | parity-blocked (timing-only via allow-mismatch) |
| vue | IV | timing-only | 5KB | 0.17 | 5.6 | 1.91 | 1.92 | 418 | 8536 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| wgsl | IV | timing-only | 12KB | 0.38 | 3.7 | 1.73 | 0.55 | 16 | 2480 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| wolfram | IV | timing-only | 4KB | 0.13 | 3.5 | 4.44 | 0.21 | 17 | 992 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |
| xml | IV | timing-only | 11KB | 0.32 | 10.4 | 1.43 | 0.13 | 13 | 800 | yes | external scanner; parity-blocked (timing-only via allow-mismatch) |

## Caveats

- **No benchmark is universal.** Small config-file grammars show inflated ×C ratios because fixed per-parse overhead dominates tiny inputs — not a steady-state regression. Benchmark your own corpus.
- **Correctness > speed.** `timing-only` grammars are measured via allow-mismatch: the speed is real but the tree still diverges from C somewhere. Don't make speed-only claims for them.
- **Pathological inputs exist.** The `unmeasured` tier-D grammars hit resource budgets or diverge on real files (see notes); they need per-corpus validation.
- **External scanners vary.** Grammars with hand-written scanners can have different tail behavior.
- **Methodology matters.** Warm = best-effort steady state, count=3 short runs (a planning signal, not a tuned micro-benchmark); cold = median of 5 decodes. For optimization work, re-run with higher count and benchstat.
