# gotreesitter Packaging Costs — binary size & cold start by mode

> Companion to `parse-benchmarks-206.md`. After "how fast is grammar X?" the next
> question every consumer asks is **"how big will my binary get, and how do I avoid
> shipping all 206 grammars?"** This doc answers it with measured numbers.

| | |
|---|---|
| Commit | `0f7b1dc2` |
| Go version | go1.25.1 |
| Machine | Intel Core Ultra 9 285, linux (WSL2) |
| Date | 2026-06-08 |
| Link flags | `-s -w` (stripped, as a real release CLI ships) |
| Probe | a minimal `main` that calls `grammars.GoLanguage()` + parses one file |
| Embedded blob weight | all 206 = **20MB**, core 100 = **14MB** |

## TL;DR — pick a packaging mode deliberately

| Mode | Build tags | Binary | Grammars in binary | Runtime requirement |
|---|---|--:|--:|---|
| **All** (default) | _(none)_ | **27 MB** | 206 | none |
| **Core set** | `grammar_set_core` | **21 MB** | 100 | none |
| **Subset** | `grammar_subset grammar_subset_<lang> …` | **5.5 MB** (1 lang) | only selected | none |
| **External blobs** | `grammar_blobs_external` | **7.4 MB** | 0 embedded | blob dir at runtime |
| _(reference)_ empty Go binary | — | 1.5 MB | — | — |

**The headline: do not ship all 206 grammars unless you actually want all 206.** A
productized CLI that parses one or a few languages should use a **subset build** (5.5 MB)
— an ~80% smaller binary than the default all-grammars build (27 MB).

## What each mode is for

- **All (default, 27 MB)** — convenience. Good for internal tools, dev environments, or
  apps that genuinely need universal language coverage. The 20 MB of embedded blobs
  dominate the binary.
- **Core set (`grammar_set_core`, 21 MB)** — the 100 most common grammars embedded
  (14 MB of blobs). A middle ground when you want broad-but-not-exhaustive coverage
  without hand-picking.
- **Subset (`grammar_subset` + per-language tags, 5.5 MB)** — embed only what you ship.
  ```sh
  go build -tags 'grammar_subset grammar_subset_go grammar_subset_python' ./your/cmd
  ```
  embeds only `go.bin` + `python.bin`. The build also dead-code-eliminates the unselected
  grammars' Go (scanners/loaders), so a 1-language subset (5.5 MB) is even smaller than the
  external binary (7.4 MB, which links all grammar code). **This is the recommended mode for
  productized CLIs.**
- **External blobs (`grammar_blobs_external`, 7.4 MB binary)** — embed nothing; load
  `*.bin` from a directory at runtime via `GOTREESITTER_GRAMMAR_BLOB_DIR`. Ship only the
  blobs you need as files (or all 20 MB). Lets you update/add grammars without recompiling.
  Blobs are mmap'd on unix, so cold start is barely affected (see below).

## Distribution cost is dominated by blobs — and by a few outliers

The 20 MB of embedded blobs is not evenly distributed. The biggest single grammars:

| Grammar | Blob | Note |
|---|--:|---|
| swift | **4.9 MB** | ~25% of the entire all-grammars blob weight, and ~1.25s cold decode |
| nim | 645 KB | |
| verilog | 637 KB | |
| cobol | 614 KB | |
| sql | 568 KB | |

**Excluding `swift` alone drops the all-grammars blob weight by ~25%.** If you use the
all/core mode but don't need Swift, a subset/external build that omits it is a large win.

## Cold start: embedded vs external

Process wall time for the probe (`main()` start → `GoLanguage()` decode → parse one file),
best of several runs:

| Mode | Cold start (go grammar) |
|---|--:|
| Embedded (all) | ~22 ms |
| External (mmap blob dir) | ~24 ms |

Breakdown: Go runtime startup ~8 ms + blob decode ~13.7 ms (for `go`; see
`parse-benchmarks-206` cold-decode column for per-grammar). **External mmap adds only ~2 ms** —
the trade-off for external is operational (you must ship + locate the blob dir), not
performance. Decode time scales with grammar complexity: most are <1 ms, but the tail is
real (swift ~1.25 s, verilog ~34 ms, cobol ~26 ms).

## Recommendations by consumer

- **Productized CLI / editor helper binary** → **subset build**. Embed only your languages.
  5.5 MB for one language vs 27 MB for all. Watch cold-decode for any heavy grammar you include.
- **Pre-commit / git hook / one-shot CI analyzer** → subset + mind cold start. Avoid heavy-decode
  grammars (swift especially) unless required; each process pays decode on first touch.
- **Repo indexer / LSP daemon / agent** → mode is less critical (long-lived, decode amortized).
  Core or all is fine; reuse loaded languages across files.
- **Plugin-style / user-extensible tool** → **external blobs**. Small binary, drop-in new
  grammars without recompiling.
- **WASM / cross-compiled** → any embedded mode (no runtime file deps); subset keeps the
  artifact small. The no-CGo story means a single clean module with no toolchain.

## How to reproduce

```sh
go build -ldflags '-s -w' -o bin_all       ./your/cmd                                   # 27 MB
go build -ldflags '-s -w' -o bin_core      -tags grammar_set_core ./your/cmd            # 21 MB
go build -ldflags '-s -w' -o bin_subset    -tags 'grammar_subset grammar_subset_go' ./your/cmd  # 5.5 MB
go build -ldflags '-s -w' -o bin_external  -tags grammar_blobs_external ./your/cmd      # 7.4 MB
GOTREESITTER_GRAMMAR_BLOB_DIR=/path/to/blobs ./bin_external
```

## Caveats

- Sizes are stripped (`-s -w`); unstripped binaries are larger (debug/symbol tables).
- Binary size beyond blobs is the shared gotreesitter + Go runtime floor (~7 MB of code,
  ~1.5 MB empty-binary baseline); it does not scale with grammar count except via DCE in
  subset mode.
- Subset DCE depends on the linker pruning unreferenced grammar code; verify your own build
  if you reference grammars dynamically (e.g. via `AllLanguages()`), which retains all.
