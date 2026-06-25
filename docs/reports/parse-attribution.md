# gotreesitter Parse Attribution — where parse time goes, per grammar

> The optimization roadmap. For every measured grammar, parse wall-time is split into disjoint phases (lexer / LR action-loop / GLR merge / GLR cull / result-build / unattributed) from `GOT_PARSE_PHASE_TIMING`. The **dominant phase** tells you what to optimize for each grammar — and the aggregate tells you the highest-leverage parser-core levers.

_Measured grammars: 174. Phases are disjoint (action_dispatch wraps lookup+apply; glr_merge/result-build are separate)._

## Dominant phase distribution

| Dominant phase | Grammars | Lever |
|---|--:|---|
| action_loop | 146 | LR dispatch interpreter loop — lexer codegen, parse-table compaction, dispatch fast-paths (language-agnostic, highest reach) |
| glr_merge | 15 | deep node-equivalence merge — fork-reduction resolvers + merge interning (targeted cluster) |
| result_build | 5 | result-tree build + per-language compat normalization — reduce normalization passes |
| unattributed | 4 | phase-timing gap — instrument further |
| lexer | 4 | DFA scan — lex-state compaction / scanner tuning (big-grammar specific) |

## Lever 1 — LR action-loop (the universal hot path)

**146 of 174 measured grammars** spend most of their parse time in the table-driven LR action loop. This is the language-agnostic interpreter cost and the single highest-reach optimization: every grammar benefits. Levers (from prior parser-core work): lexer codegen (switch-DFA), parse-table compaction, and action-dispatch fast-paths. Representative large-corpus grammars: go (action ~44%), python (~51%), rust, java, typescript.

## Lever 2 — GLR merge (the deep-merge cluster)

**15 grammars** are dominated by `glr_merge` — the O(n²)-ish deep node-equivalence merge. This is the cluster the fork-reduction / merge-interning work targets, and it overlaps the worst full-parse ratios (ledger 609×, authzed 123×). Fixing the merge cost is both a perf win and (for the truncating ones) a measurability unblock.

| grammar | merge % | ms/op | full ×C |
|---|--:|--:|--:|
| ledger | 91% | 148.54 | 3 |
| authzed | 91% | 776.70 | 2 |
| json5 | 80% | 3.45 | 2 |
| gitattributes | 73% | 16.15 | 2 |
| dtd | 72% | 158.30 | 692 |
| org | 72% | 2.78 | 2 |
| nginx | 71% | 4.09 | 33 |
| make | 71% | 227.88 | 1 |
| dockerfile | 55% | 12.46 | 18 |
| ron | 52% | 0.70 | 7 |
| commonlisp | 51% | 0.92 | 2 |
| facility | 49% | 1.22 | 18 |
| yuck | 49% | 4.07 | 1 |
| vimdoc | 48% | 1.15 | 1 |
| just | 45% | 0.40 | 5 |

## Lever 3 — lexer / DFA-bound

Large grammars whose DFA scan dominates. swift's 4.9MB blob makes its lexer the bottleneck.

| grammar | lexer % | ms/op |
|---|--:|--:|
| swift | 57% | 0.140 |
| mojo | 45% | 0.026 |
| thrift | 39% | 0.037 |
| html | 38% | 0.119 |

## Lever 4 — result-build / compat-normalization-bound

Grammars whose post-parse result-build + per-language compatibility normalization dominate. These carry heavy `parser_result_<lang>.go` normalization; the lever is reducing/short-circuiting those passes.

| grammar | result % | ms/op |
|---|--:|--:|
| c | 60% | 20.447 |
| dart | 59% | 0.028 |
| hack | 50% | 0.016 |
| kotlin | 44% | 0.082 |
| php | 42% | 0.994 |

## Full per-grammar attribution

Machine-readable: `attribution.json`. Percentages are of parse wall-time.

| Grammar | ms/op | full ×C | dominant | lexer% | action% | merge% | result% | unattr% |
|---|--:|--:|---|--:|--:|--:|--:|--:|
| authzed | 776.703 | 1.8 | glr_merge | 0 | 8 | 91 | 0 | 1 |
| diff | 438.885 | 365.2 | action_loop | 0 | 75 | 14 | 0 | 10 |
| gomod | 272.049 | 10.9 | action_loop | 2 | 53 | 36 | 0 | 9 |
| make | 227.883 | 0.9 | glr_merge | 1 | 23 | 71 | 0 | 5 |
| desktop | 215.472 | 56.8 | action_loop | 1 | 68 | 18 | 0 | 13 |
| dtd | 158.303 | 691.7 | glr_merge | 1 | 21 | 72 | 0 | 6 |
| ledger | 148.535 | 2.6 | glr_merge | 0 | 7 | 91 | 0 | 1 |
| embedded_template | 90.373 | 15.0 | action_loop | 2 | 61 | 23 | 0 | 14 |
| graphql | 73.869 | 4.6 | action_loop | 5 | 57 | 12 | 3 | 22 |
| go | 51.089 | 1.7 | action_loop | 13 | 44 | 21 | 8 | 14 |
| typescript | 48.355 | 1.7 | action_loop | 19 | 58 | 7 | 22 | 0 |
| kdl | 43.135 | 118.8 | action_loop | 0 | 42 | 15 | 0 | 7 |
| rust | 28.473 | 2.4 | action_loop | 12 | 44 | 25 | 17 | 3 |
| sparql | 23.447 | 1.6 | action_loop | 13 | 50 | 13 | 5 | 18 |
| c | 20.447 | 1.0 | result_build | 20 | 39 | 4 | 60 | 0 |
| python | 20.069 | 1.7 | action_loop | 20 | 51 | 7 | 26 | 0 |
| gitattributes | 16.155 | 1.8 | glr_merge | 1 | 21 | 73 | 0 | 4 |
| http | 14.012 | 20.2 | action_loop | 1 | 65 | 20 | 0 | 14 |
| dockerfile | 12.464 | 17.7 | glr_merge | 2 | 36 | 55 | 0 | 6 |
| eds | 10.625 | 19.7 | action_loop | 5 | 44 | 15 | 1 | 14 |
| json | 6.947 | 1.8 | action_loop | 11 | 55 | 8 | 4 | 23 |
| tmux | 5.983 | 8.6 | action_loop | 12 | 54 | 17 | 0 | 17 |
| hurl | 5.627 | 41.7 | action_loop | 1 | 66 | 20 | 0 | 13 |
| liquid | 5.034 | 9.6 | action_loop | 4 | 44 | 43 | 1 | 9 |
| properties | 4.206 | 26.3 | action_loop | 1 | 65 | 21 | 0 | 13 |
| nginx | 4.094 | 32.8 | glr_merge | 1 | 23 | 71 | 0 | 5 |
| yuck | 4.071 | 1.2 | glr_merge | 2 | 41 | 49 | 1 | 8 |
| json5 | 3.448 | 1.7 | glr_merge | 1 | 15 | 80 | 0 | 3 |
| typst | 3.192 | 12.4 | action_loop | 2 | 64 | 21 | 0 | 13 |
| org | 2.780 | 1.7 | glr_merge | 1 | 22 | 72 | 0 | 5 |
| bitbake | 2.639 | 3.1 | action_loop | 10 | 57 | 17 | 2 | 14 |
| ninja | 2.199 | 12.5 | action_loop | 3 | 62 | 23 | 1 | 12 |
| ssh_config | 1.688 | 29.8 | action_loop | 1 | 67 | 20 | 0 | 12 |
| uxntal | 1.671 | 16.3 | action_loop | 3 | 65 | 18 | 1 | 13 |
| ebnf | 1.630 | 14.5 | action_loop | 3 | 67 | 19 | 1 | 12 |
| corn | 1.583 | 4.7 | action_loop | 4 | 48 | 34 | 2 | 12 |
| djot | 1.544 | 3.5 | action_loop | 14 | 37 | 10 | 0 | 9 |
| elm | 1.503 | 4.0 | action_loop | 6 | 49 | 36 | 1 | 8 |
| facility | 1.223 | 18.0 | glr_merge | 2 | 40 | 49 | 0 | 9 |
| vimdoc | 1.154 | 1.3 | glr_merge | 2 | 39 | 48 | 1 | 11 |
| disassembly | 1.017 | 1.0 | action_loop | 33 | 49 | 1 | 0 | 17 |
| php | 0.994 | 1.1 | result_build | 16 | 31 | 1 | 42 | 11 |
| commonlisp | 0.925 | 1.7 | glr_merge | 2 | 40 | 51 | 1 | 7 |
| markdown | 0.919 | 6.2 | action_loop | 5 | 52 | 23 | 0 | 13 |
| markdown_inline | 0.911 | 9.7 | action_loop | 3 | 53 | 23 | 0 | 16 |
| teal | 0.851 | 3.9 | action_loop | 6 | 55 | 27 | 1 | 11 |
| ada | 0.804 | 3.5 | action_loop | 6 | 60 | 20 | 2 | 12 |
| fsharp | 0.739 | 5.0 | action_loop | 3 | 66 | 17 | 1 | 14 |
| caddy | 0.733 | 1.9 | action_loop | 15 | 44 | 20 | 2 | 19 |
| scheme | 0.717 | 8.2 | action_loop | 3 | 57 | 26 | 1 | 13 |
| ron | 0.701 | 7.3 | glr_merge | 6 | 32 | 52 | 2 | 8 |
| less | 0.676 | 27.3 | action_loop | 1 | 65 | 20 | 0 | 14 |
| hyprlang | 0.657 | 10.3 | action_loop | 2 | 65 | 20 | 0 | 12 |
| sql | 0.645 | 0.9 | unattributed | 13 | 27 | 0 | 21 | 39 |
| jq | 0.553 | 3.7 | action_loop | 9 | 39 | 34 | 3 | 15 |
| yaml | 0.535 | 0.6 | action_loop | 27 | 47 | 0 | 22 | 5 |
| circom | 0.507 | 4.4 | action_loop | 6 | 60 | 16 | 2 | 16 |
| cylc | 0.501 | 5.0 | action_loop | 3 | 63 | 18 | 1 | 15 |
| cpon | 0.456 | 3.7 | action_loop | 9 | 56 | 15 | 4 | 16 |
| git_rebase | 0.421 | 3.5 | action_loop | 6 | 56 | 19 | 4 | 15 |
| just | 0.401 | 4.6 | glr_merge | 6 | 38 | 45 | 1 | 10 |
| nushell | 0.358 | 3.3 | action_loop | 8 | 59 | 16 | 2 | 13 |
| regex | 0.349 | 9.5 | action_loop | 2 | 63 | 21 | 1 | 13 |
| meson | 0.342 | 4.2 | action_loop | 4 | 41 | 41 | 1 | 10 |
| d | 0.338 | 1.6 | action_loop | 20 | 46 | 12 | 12 | 10 |
| capnp | 0.298 | 3.0 | action_loop | 13 | 56 | 13 | 3 | 16 |
| ini | 0.287 | 3.8 | action_loop | 8 | 62 | 14 | 2 | 14 |
| todotxt | 0.282 | 7.6 | action_loop | 3 | 62 | 21 | 0 | 13 |
| textproto | 0.277 | 3.5 | action_loop | 4 | 67 | 14 | 2 | 12 |
| janet | 0.268 | 6.3 | action_loop | 6 | 49 | 33 | 1 | 11 |
| tcl | 0.264 | 2.4 | action_loop | 8 | 58 | 19 | 2 | 13 |
| bass | 0.260 | 3.6 | action_loop | 5 | 62 | 16 | 2 | 14 |
| toml | 0.253 | 1.9 | action_loop | 13 | 53 | 8 | 2 | 24 |
| perl | 0.243 | 1.9 | action_loop | 9 | 64 | 11 | 5 | 11 |
| earthfile | 0.242 | 2.5 | action_loop | 7 | 51 | 25 | 1 | 15 |
| fennel | 0.226 | 2.0 | action_loop | 9 | 67 | 9 | 2 | 12 |
| heex | 0.225 | 3.2 | action_loop | 6 | 56 | 14 | 3 | 20 |
| requirements | 0.221 | 4.1 | action_loop | 4 | 56 | 25 | 1 | 14 |
| twig | 0.220 | 1.9 | action_loop | 8 | 58 | 10 | 4 | 20 |
| eex | 0.208 | 9.4 | action_loop | 3 | 65 | 19 | 1 | 13 |
| editorconfig | 0.200 | 7.3 | action_loop | 4 | 61 | 20 | 1 | 15 |
| hcl | 0.199 | 1.4 | action_loop | 12 | 53 | 11 | 3 | 21 |
| git_config | 0.191 | 7.3 | action_loop | 3 | 65 | 19 | 1 | 13 |
| crystal | 0.190 | 1.6 | action_loop | 14 | 54 | 12 | 1 | 20 |
| dot | 0.190 | 2.6 | action_loop | 8 | 60 | 11 | 3 | 18 |
| forth | 0.171 | 7.4 | action_loop | 2 | 64 | 21 | 1 | 12 |
| verilog | 0.158 | 1.3 | action_loop | 10 | 43 | 25 | 3 | 19 |
| tsx | 0.150 | 1.2 | action_loop | 32 | 47 | 0 | 29 | 0 |
| purescript | 0.145 | 0.9 | action_loop | 15 | 58 | 11 | 3 | 14 |
| starlark | 0.142 | 0.8 | action_loop | 26 | 49 | 1 | 5 | 20 |
| swift | 0.140 | 2.9 | lexer | 57 | 22 | 5 | 8 | 8 |
| r | 0.138 | 1.1 | action_loop | 15 | 60 | 4 | 3 | 17 |
| kconfig | 0.137 | 2.9 | action_loop | 8 | 48 | 26 | 2 | 16 |
| objc | 0.127 | 0.7 | action_loop | 13 | 56 | 9 | 2 | 20 |
| html | 0.119 | 1.6 | lexer | 38 | 33 | 5 | 8 | 15 |
| matlab | 0.113 | 1.2 | action_loop | 18 | 53 | 6 | 4 | 18 |
| angular | 0.110 | 1.5 | action_loop | 22 | 47 | 8 | 5 | 17 |
| haxe | 0.110 | 1.5 | action_loop | 15 | 52 | 9 | 2 | 23 |
| godot_resource | 0.105 | 2.2 | action_loop | 12 | 50 | 15 | 4 | 19 |
| foam | 0.101 | 1.5 | action_loop | 15 | 50 | 16 | 4 | 15 |
| ql | 0.101 | 1.7 | action_loop | 11 | 50 | 22 | 4 | 13 |
| vue | 0.093 | 1.9 | action_loop | 32 | 39 | 3 | 6 | 21 |
| v | 0.092 | 0.9 | action_loop | 14 | 56 | 6 | 5 | 18 |
| wolfram | 0.089 | 4.4 | action_loop | 30 | 43 | 0 | 0 | 26 |
| elixir | 0.087 | 0.6 | action_loop | 22 | 46 | 1 | 11 | 19 |
| glsl | 0.085 | 1.0 | action_loop | 18 | 48 | 8 | 2 | 24 |
| elsa | 0.082 | 1.1 | action_loop | 18 | 52 | 1 | 8 | 21 |
| kotlin | 0.082 | 0.8 | result_build | 22 | 33 | 0 | 44 | 1 |
| gleam | 0.080 | 0.9 | action_loop | 16 | 55 | 4 | 5 | 19 |
| cpp | 0.079 | 1.0 | action_loop | 16 | 47 | 1 | 37 | 0 |
| cuda | 0.077 | 0.9 | action_loop | 36 | 39 | 2 | 5 | 17 |
| firrtl | 0.075 | 1.3 | action_loop | 24 | 42 | 5 | 6 | 23 |
| gdscript | 0.073 | 1.0 | action_loop | 21 | 49 | 4 | 5 | 21 |
| wat | 0.071 | 0.5 | action_loop | 22 | 49 | 0 | 7 | 22 |
| gn | 0.071 | 2.0 | action_loop | 11 | 58 | 9 | 5 | 17 |
| clojure | 0.069 | 0.9 | action_loop | 15 | 63 | 0 | 5 | 17 |
| cairo | 0.069 | 2.4 | action_loop | 10 | 56 | 12 | 4 | 18 |
| pkl | 0.068 | 0.9 | action_loop | 24 | 46 | 2 | 6 | 21 |
| asm | 0.066 | 2.2 | action_loop | 14 | 51 | 9 | 7 | 18 |
| enforce | 0.065 | 1.5 | action_loop | 13 | 48 | 15 | 5 | 18 |
| nickel | 0.063 | 0.6 | action_loop | 26 | 49 | 0 | 11 | 15 |
| lua | 0.062 | 0.6 | action_loop | 14 | 56 | 0 | 8 | 21 |
| rego | 0.060 | 1.1 | action_loop | 19 | 42 | 1 | 14 | 25 |
| pascal | 0.058 | 0.9 | action_loop | 26 | 41 | 3 | 8 | 22 |
| haskell | 0.056 | 0.7 | action_loop | 31 | 41 | 1 | 13 | 14 |
| odin | 0.056 | 1.6 | action_loop | 17 | 51 | 7 | 5 | 20 |
| ocaml | 0.051 | 0.6 | action_loop | 27 | 47 | 0 | 8 | 18 |
| fortran | 0.050 | 1.0 | action_loop | 26 | 42 | 3 | 10 | 19 |
| puppet | 0.049 | 1.0 | action_loop | 21 | 46 | 1 | 7 | 24 |
| devicetree | 0.048 | 1.6 | action_loop | 17 | 47 | 5 | 8 | 24 |
| luau | 0.046 | 0.7 | action_loop | 29 | 42 | 0 | 8 | 21 |
| jsonnet | 0.046 | 0.9 | action_loop | 28 | 41 | 0 | 9 | 22 |
| svelte | 0.045 | 1.0 | action_loop | 32 | 37 | 0 | 8 | 22 |
| xml | 0.044 | 1.4 | action_loop | 19 | 49 | 5 | 3 | 24 |
| linkerscript | 0.043 | 0.5 | action_loop | 26 | 42 | 0 | 1 | 31 |
| tablegen | 0.043 | 1.7 | action_loop | 24 | 38 | 6 | 9 | 22 |
| fidl | 0.042 | 1.1 | action_loop | 22 | 45 | 0 | 9 | 23 |
| jsdoc | 0.041 | 2.2 | action_loop | 13 | 44 | 15 | 6 | 21 |
| java | 0.041 | 1.0 | action_loop | 26 | 45 | 0 | 13 | 15 |
| groovy | 0.040 | 1.0 | action_loop | 14 | 53 | 7 | 4 | 21 |
| proto | 0.039 | 0.9 | action_loop | 17 | 45 | 5 | 8 | 25 |
| wgsl | 0.039 | 1.7 | action_loop | 18 | 47 | 8 | 6 | 21 |
| vhdl | 0.038 | 0.1 | action_loop | 27 | 39 | 0 | 5 | 29 |
| robot | 0.038 | 0.8 | action_loop | 20 | 46 | 3 | 7 | 24 |
| astro | 0.037 | 1.1 | action_loop | 33 | 34 | 2 | 9 | 23 |
| thrift | 0.037 | 1.1 | lexer | 39 | 26 | 0 | 14 | 21 |
| rescript | 0.037 | 0.6 | action_loop | 24 | 46 | 2 | 7 | 21 |
| move | 0.035 | 0.4 | action_loop | 26 | 42 | 0 | 5 | 27 |
| zig | 0.035 | 0.9 | action_loop | 24 | 43 | 0 | 11 | 22 |
| mermaid | 0.034 | 0.9 | action_loop | 16 | 55 | 3 | 4 | 22 |
| hlsl | 0.033 | 0.9 | action_loop | 25 | 42 | 1 | 10 | 22 |
| smithy | 0.033 | 0.7 | action_loop | 21 | 46 | 0 | 11 | 22 |
| solidity | 0.033 | 1.1 | action_loop | 23 | 43 | 1 | 9 | 23 |
| ruby | 0.031 | 0.6 | action_loop | 24 | 45 | 0 | 13 | 18 |
| bicep | 0.030 | 1.1 | action_loop | 25 | 38 | 2 | 11 | 25 |
| julia | 0.030 | 0.8 | action_loop | 26 | 41 | 0 | 9 | 24 |
| llvm | 0.028 | 0.9 | action_loop | 20 | 43 | 1 | 12 | 23 |
| dart | 0.028 | 1.3 | result_build | 22 | 24 | 0 | 59 | 0 |
| brightscript | 0.027 | 0.8 | action_loop | 22 | 42 | 1 | 11 | 24 |
| dhall | 0.026 | 0.9 | action_loop | 19 | 45 | 3 | 10 | 23 |
| mojo | 0.026 | 0.2 | lexer | 45 | 16 | 0 | 17 | 22 |
| gitcommit | 0.022 | 1.3 | action_loop | 19 | 43 | 4 | 8 | 25 |
| templ | 0.022 | 1.1 | action_loop | 28 | 40 | 0 | 9 | 23 |
| prolog | 0.019 | 0.8 | action_loop | 16 | 48 | 0 | 2 | 34 |
| chatito | 0.017 | 2.5 | action_loop | 11 | 52 | 8 | 5 | 24 |
| nim | 0.017 | 0.6 | action_loop | 25 | 49 | 0 | 7 | 18 |
| hack | 0.016 | 1.8 | result_build | 19 | 25 | 0 | 50 | 5 |
| turtle | 0.016 | 1.1 | action_loop | 13 | 54 | 4 | 8 | 22 |
| pem | 0.015 | 2.6 | action_loop | 13 | 52 | 9 | 5 | 21 |
| doxygen | 0.013 | 1.1 | unattributed | 26 | 28 | 0 | 15 | 31 |
| powershell | 0.010 | 0.9 | action_loop | 18 | 38 | 0 | 17 | 27 |
| jinja2 | 0.007 | 0.4 | unattributed | 26 | 33 | 0 | 1 | 40 |
| elisp | 0.006 | 1.4 | action_loop | 24 | 35 | 0 | 12 | 29 |
| blade | 0.002 | 1.5 | unattributed | 7 | 13 | 0 | 28 | 52 |