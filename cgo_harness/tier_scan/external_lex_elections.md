# Wave 4 External Lex-State Election Ledger

This ledger covers every grammar in `cgo_harness/tier_scan/exts.tsv`.
It tracks whether each grammar's external-scanner recovery path is
default-elected, staged, blocked on a missing precise ExternalLexStates
table, or not applicable because the grammar has no registered Go external
scanner.

## Summary

| metric | count |
| --- | ---: |
| grammars | 206 |
| registered external scanners | 119 |
| default precise ExternalLexStates tables | 85 |
| staged precise ExternalLexStates tables | 1 |

| status | count |
| --- | ---: |
| default elected | 85 |
| staged precise ELS | 1 |
| blocked: missing precise ELS | 33 |
| not applicable: no external scanner | 87 |

## Verification Receipts

- `default_elected`: Docker: wave4-external-lex-election-inventory-test-v2; TestExternalLexStatesDefaultElectionInventory
- `staged_precise_els`: Docker: wave4-javascript-precise-els-staged-test; TestJavascriptExternalLexStatesRegression (-tags javascript_precise_els); TestJavascriptExternalLexStatesRemainStagedByDefault
- `sample_c_oracle_smoke`: Docker: wave4-external-lex-smoke-20260707T1928; angular/python/yaml clean; scss/wgsl classified recovery/error-shape IV
- `wave3_inventory`: Docker: wave3-tier-plan-206; 206 visited; 202 planned files; 4 planned-empty

## Status Definitions

- `default_elected`: External scanner and precise ExternalLexStates table are registered by default; DiagnoseCRecoveryGate supports the language and C recovery cost competition is default-enabled.
- `staged_precise_els`: A precise ExternalLexStates table exists behind an opt-in build tag or explicit staging policy; default election is intentionally disabled.
- `blocked_missing_precise_els`: External scanner is registered, but no precise ExternalLexStates table is registered or staged yet. Wave 4 cannot default-elect C recovery for this grammar until the table is extracted and certified.
- `not_applicable_no_external_scanner`: No Go external scanner is registered for this grammar, so there is no external-lex-state election to perform. Tier/parity status remains governed by the ordinary C-oracle classification.

## Grammar Ledger

| grammar | status | tier row | parity | scanner | default ELS | staged ELS | extensions |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `angular` | default elected | IV-recovery? | 22/40 | yes | yes | no | `.html` |
| `arduino` | default elected | CLEAN | 40/40 | yes | yes | no | `.ino` |
| `astro` | default elected | CLEAN | 40/40 | yes | yes | no | `.astro` |
| `awk` | default elected | IV-unknown | 25/29 | yes | yes | no | `.auk,.awk,.gawk,.mawk,.nawk` |
| `bash` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.sh` |
| `beancount` | default elected | CLEAN | 3/3 | yes | yes | no | `.bean,.beancount` |
| `bicep` | default elected | IV-unknown | 34/40 | yes | yes | no | `.bicep` |
| `bitbake` | default elected | IV-unknown | 35/40 | yes | yes | no | `.bb,.bbappend,.bbclass` |
| `blade` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.blade.php` |
| `c_sharp` | default elected | IV-recovery | unmeasured | yes | yes | no | `.cs` |
| `caddy` | default elected | IV-recovery? | 11/40 | yes | yes | no | `.caddy,caddyfile` |
| `cairo` | default elected | IV-recovery? | 1/40 | yes | yes | no | `.cairo` |
| `cmake` | default elected | CLEAN | 40/40 | yes | yes | no | `.cmake,.cmake.in` |
| `cooklang` | default elected | IV-recovery | 0/3 | yes | yes | no | `.cook` |
| `cpp` | default elected | IV-recovery | 9/40 | yes | yes | no | `.cc,.cpp,.cxx,.h,.hh,.hpp,.hxx` |
| `css` | default elected | IV-unknown | 37/40 | yes | yes | no | `.css` |
| `cuda` | default elected | IV-recovery? | 21/40 | yes | yes | no | `.cu,.cuh` |
| `cue` | default elected | IV-unknown | 39/40 | yes | yes | no | `.cue` |
| `d` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.d,.di` |
| `dart` | default elected | IV-recovery? | 0/40 | yes | yes | no | `.dart` |
| `disassembly` | default elected | IV-version | 0/40 | yes | yes | no | `.dis,.dump` |
| `dockerfile` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.dockerfile,dockerfile` |
| `dtd` | default elected | IV-unknown | 0/3 | yes | yes | no | `.dtd` |
| `earthfile` | default elected | IV-recovery? | 0/40 | yes | yes | no | `earthfile` |
| `editorconfig` | default elected | CLEAN | 40/40 | yes | yes | no | `.editorconfig` |
| `elm` | default elected | IV-unknown | 0/8 | yes | yes | no | `.elm` |
| `fennel` | default elected | IV-recovery? | 15/40 | yes | yes | no | `.fnl` |
| `firrtl` | default elected | IV-recovery? | 12/27 | yes | yes | no | `.fir` |
| `foam` | default elected | CLEAN | 40/40 | yes | yes | no | `.foam` |
| `fortran` | default elected | CLEAN | 40/40 | yes | yes | no | `.f,.f03,.f08,.f90,.f95` |
| `gdscript` | default elected | IV-unknown | 39/40 | yes | yes | no | `.gd` |
| `gitcommit` | default elected | CLEAN | 13/13 | yes | yes | no | `commit_editmsg` |
| `gleam` | default elected | IV-unknown | 39/40 | yes | yes | no | `.gleam` |
| `gn` | default elected | CLEAN | 40/40 | yes | yes | no | `.gn,.gni` |
| `hack` | default elected | IV-unknown | 38/40 | yes | yes | no | `.hack,.hh` |
| `haxe` | default elected | IV-recovery | 10/40 | yes | yes | no | `.hx` |
| `hlsl` | default elected | IV-unknown | 35/40 | yes | yes | no | `.fx,.hlsl` |
| `html` | default elected | IV-recovery | 0/40 | yes | yes | no | `.htm,.html` |
| `janet` | default elected | CLEAN | 40/40 | yes | yes | no | `.janet` |
| `jsdoc` | default elected | IV-unknown | 13/40 | yes | yes | no | `.jsdoc` |
| `jsonnet` | default elected | IV-recovery? | 39/40 | yes | yes | no | `.jsonnet,.libsonnet` |
| `just` | default elected | IV-unknown | 2/8 | yes | yes | no | `.just,justfile` |
| `kconfig` | default elected | IV-recovery? | 18/40 | yes | yes | no | `kconfig` |
| `kdl` | default elected | IV-recovery | 11/40 | yes | yes | no | `.kdl` |
| `kotlin` | default elected | IV-unknown | 22/40 | yes | yes | no | `.kt,.kts` |
| `less` | default elected | IV-recovery? | 10/40 | yes | yes | no | `.less` |
| `liquid` | default elected | IV-recovery? | 11/36 | yes | yes | no | `.liquid` |
| `lua` | default elected | CLEAN | 40/40 | yes | yes | no | `.lua` |
| `luau` | default elected | IV-perf | unmeasured | yes | yes | no | `.luau` |
| `matlab` | default elected | IV-recovery? | 4/40 | yes | yes | no | `.m,.mat` |
| `mojo` | default elected | IV-recovery? | 29/40 | yes | yes | no | `.mojo,.🔥` |
| `move` | default elected | IV-recovery? | 14/40 | yes | yes | no | `.move` |
| `nickel` | default elected | CLEAN | 40/40 | yes | yes | no | `.ncl` |
| `nim` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.nim,.nims` |
| `nushell` | default elected | IV-recovery? | 7/40 | yes | yes | no | `.nu` |
| `odin` | default elected | IV-recovery? | 1/40 | yes | yes | no | `.odin` |
| `org` | default elected | IV-recovery? | 1/6 | yes | yes | no | `.org` |
| `php` | default elected | IV-unknown | 36/40 | yes | yes | no | `.php` |
| `pkl` | default elected | CLEAN | 40/40 | yes | yes | no | `.pkl` |
| `powershell` | default elected | IV-recovery? | 22/40 | yes | yes | no | `.ps1,.psd1,.psm1` |
| `pug` | default elected | IV-recovery? | 0/40 | yes | yes | no | `.jade,.pug` |
| `purescript` | default elected | IV-recovery? | 1/40 | yes | yes | no | `.purs` |
| `python` | default elected | IV-unknown | 6/40 | yes | yes | no | `.py` |
| `r` | default elected | CLEAN | 40/40 | yes | yes | no | `.r` |
| `rescript` | default elected | IV-recovery? | 24/40 | yes | yes | no | `.res,.resi` |
| `ron` | default elected | CLEAN | 40/40 | yes | yes | no | `.ron` |
| `ruby` | default elected | CLEAN | 40/40 | yes | yes | no | `.rb` |
| `rust` | default elected | IV-recovery? | 11/40 | yes | yes | no | `.rs` |
| `scala` | default elected | IV-recovery? | 5/40 | yes | yes | no | `.scala` |
| `scss` | default elected | IV-recovery? | 6/40 | yes | yes | no | `.scss` |
| `sql` | default elected | IV-recovery? | 8/40 | yes | yes | no | `.sql` |
| `squirrel` | default elected | CLEAN | 40/40 | yes | yes | no | `.nut` |
| `starlark` | default elected | IV-unknown | 34/40 | yes | yes | no | `.bzl,.star` |
| `svelte` | default elected | IV-unknown | 8/40 | yes | yes | no | `.svelte` |
| `tablegen` | default elected | CLEAN | 40/40 | yes | yes | no | `.td` |
| `tcl` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.tcl` |
| `teal` | default elected | IV-recovery? | 8/40 | yes | yes | no | `.tl` |
| `templ` | default elected | IV-recovery? | 31/40 | yes | yes | no | `.templ` |
| `tsx` | default elected | CLEAN | 40/40 | yes | yes | no | `.tsx` |
| `typst` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.typ` |
| `uxntal` | default elected | IV-recovery? | 0/40 | yes | yes | no | `.tal` |
| `vhdl` | default elected | IV-recovery? | unmeasured | yes | yes | no | `.vhd,.vhdl` |
| `vue` | default elected | CLEAN | 40/40 | yes | yes | no | `.vue` |
| `wgsl` | default elected | IV-recovery | 21/40 | yes | yes | no | `.wgsl` |
| `yaml` | default elected | CLEAN | 40/40 | yes | yes | no | `.yaml,.yml` |
| `javascript` | staged precise ELS | CLEAN | 40/40 | yes | no | yes | `.cjs,.js,.mjs` |
| `agda` | blocked: missing precise ELS | IV-scanner | 2/40 | yes | no | no | `.agda` |
| `cobol` | blocked: missing precise ELS | IV-version | 0/40 | yes | no | no | `.cbl,.cob,.cpy` |
| `comment` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.txt` |
| `crystal` | blocked: missing precise ELS | IV-shape? | unmeasured | yes | no | no | `.cr` |
| `dhall` | blocked: missing precise ELS | IV-unknown | 15/40 | yes | no | no | `.dhall` |
| `djot` | blocked: missing precise ELS | IV-scanner? | unmeasured | yes | no | no | `.dj,.djot` |
| `doxygen` | blocked: missing precise ELS | IV-unknown | 20/40 | yes | no | no | `.dox,.doxygen` |
| `elixir` | blocked: missing precise ELS | IV-recovery | unmeasured | yes | no | no | `.ex,.exs` |
| `erlang` | blocked: missing precise ELS | IV-recovery | unmeasured | yes | no | no | `.app,.app.src,.erl,.escript,.hrl,.xrl,.yrl` |
| `fish` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.fish` |
| `fsharp` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.fs,.fsi,.fsx` |
| `go` | blocked: missing precise ELS | IV-unknown | 25/40 | yes | no | no | `.go` |
| `godot_resource` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.tres,.tscn` |
| `haskell` | blocked: missing precise ELS | IV-scanner | unmeasured | yes | no | no | `.hs,.lhs` |
| `hcl` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.hcl,.tf,.tfvars` |
| `julia` | blocked: missing precise ELS | IV-recovery | unmeasured | yes | no | no | `.jl` |
| `markdown` | blocked: missing precise ELS | IV-unknown | 0/40 | yes | no | no | `.md` |
| `markdown_inline` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.md` |
| `nginx` | blocked: missing precise ELS | IV-unknown | 0/1 | yes | no | no | `.nginx,.nginxconf,.vhost,nginx.conf` |
| `nix` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.nix` |
| `norg` | blocked: missing precise ELS | IV-scanner | 0/2 | yes | no | no | `.norg` |
| `ocaml` | blocked: missing precise ELS | IV-unknown | 12/40 | yes | no | no | `.ml,.mli` |
| `perl` | blocked: missing precise ELS | IV-recovery? | unmeasured | yes | no | no | `.pl,.pm` |
| `properties` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.properties` |
| `racket` | blocked: missing precise ELS | IV-unknown | 2/40 | yes | no | no | `.rkt` |
| `rst` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.rst` |
| `swift` | blocked: missing precise ELS | IV-recovery? | 0/40 | yes | no | no | `.swift` |
| `tlaplus` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.tla` |
| `toml` | blocked: missing precise ELS | IV-unknown | 8/11 | yes | no | no | `.toml` |
| `typescript` | blocked: missing precise ELS | IV-perf | unmeasured | yes | no | no | `.ts` |
| `wolfram` | blocked: missing precise ELS | IV-unknown | 0/11 | yes | no | no | `.m,.nb,.wl` |
| `xml` | blocked: missing precise ELS | IV-unknown | 1/40 | yes | no | no | `.xml` |
| `yuck` | blocked: missing precise ELS | IV-unknown | 1/2 | yes | no | no | `.yuck` |
| `ada` | not applicable: no external scanner | IV-unknown | 39/40 | no | no | no | `.adb,.ads` |
| `apex` | not applicable: no external scanner | IV-unknown | 39/40 | no | no | no | `.cls,.trigger` |
| `asm` | not applicable: no external scanner | IV-recovery | 0/40 | no | no | no | `.asm,.s` |
| `authzed` | not applicable: no external scanner | IV-unknown | 36/40 | no | no | no | `.zed` |
| `bass` | not applicable: no external scanner | IV-unknown | 39/40 | no | no | no | `.bass` |
| `bibtex` | not applicable: no external scanner | IV-unknown | 38/40 | no | no | no | `.bib` |
| `brightscript` | not applicable: no external scanner | IV-recovery? | 11/40 | no | no | no | `.brs` |
| `c` | not applicable: no external scanner | IV-recovery | 22/40 | no | no | no | `.c,.h` |
| `capnp` | not applicable: no external scanner | IV-unknown | 14/21 | no | no | no | `.capnp` |
| `chatito` | not applicable: no external scanner | IV-unknown | 1/5 | no | no | no | `.chatito` |
| `circom` | not applicable: no external scanner | IV-shape? | 19/40 | no | no | no | `.circom` |
| `clojure` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.clj,.cljc,.cljs,.edn` |
| `commonlisp` | not applicable: no external scanner | IV-recovery? | unmeasured | no | no | no | `.asd,.cl,.lisp,.lsp` |
| `corn` | not applicable: no external scanner | IV-unknown | 22/23 | no | no | no | `.corn` |
| `cpon` | not applicable: no external scanner | IV-unknown | 9/10 | no | no | no | `.cpon` |
| `csv` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.csv,.tsv` |
| `cylc` | not applicable: no external scanner | IV-recovery? | unmeasured | no | no | no | `.cylc` |
| `desktop` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.desktop,.desktop.in,.service` |
| `devicetree` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.dts,.dtsi` |
| `diff` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.diff,.patch` |
| `dot` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.dot,.gv` |
| `ebnf` | not applicable: no external scanner | IV-recovery? | 0/40 | no | no | no | `.ebnf` |
| `eds` | not applicable: no external scanner | IV-unknown | 0/1 | no | no | no | `.eds` |
| `eex` | not applicable: no external scanner | IV-unknown | 21/40 | no | no | no | `.eex,.heex,.html.eex,.leex` |
| `elisp` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.el` |
| `elsa` | not applicable: no external scanner | IV-recovery? | 0/0 | no | no | no | `.elsa` |
| `embedded_template` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.ejs,.erb` |
| `enforce` | not applicable: no external scanner | IV-unknown | 28/40 | no | no | no | `.c,.enf` |
| `facility` | not applicable: no external scanner | IV-unknown | 1/4 | no | no | no | `.fac,.fsd` |
| `faust` | not applicable: no external scanner | IV-unknown | 39/40 | no | no | no | `.dsp` |
| `fidl` | not applicable: no external scanner | IV-unknown | 37/40 | no | no | no | `.fidl` |
| `forth` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.4th,.fs,.fth` |
| `git_config` | not applicable: no external scanner | CLEAN | 7/7 | no | no | no | `.gitconfig` |
| `git_rebase` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.git-rebase-todo` |
| `gitattributes` | not applicable: no external scanner | IV-unknown | 9/10 | no | no | no | `.gitattributes` |
| `gitignore` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.gitignore` |
| `glsl` | not applicable: no external scanner | IV-recovery | 7/40 | no | no | no | `.frag,.glsl,.vert` |
| `gomod` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.mod` |
| `graphql` | not applicable: no external scanner | IV-unknown | 0/1 | no | no | no | `.gql,.graphql` |
| `groovy` | not applicable: no external scanner | IV-recovery | 6/40 | no | no | no | `.groovy,.gvy` |
| `hare` | not applicable: no external scanner | IV-recovery | 15/40 | no | no | no | `.ha` |
| `heex` | not applicable: no external scanner | IV-unknown | 6/7 | no | no | no | `.heex` |
| `http` | not applicable: no external scanner | IV-unknown | 10/11 | no | no | no | `.http` |
| `hurl` | not applicable: no external scanner | IV-recovery | 14/40 | no | no | no | `.hurl` |
| `hyprlang` | not applicable: no external scanner | IV-unknown | 1/2 | no | no | no | `.conf` |
| `ini` | not applicable: no external scanner | IV-unknown | 9/11 | no | no | no | `.cfg,.conf,.ini` |
| `java` | not applicable: no external scanner | IV-unknown | 22/40 | no | no | no | `.java` |
| `jinja2` | not applicable: no external scanner | IV-recovery | 3/40 | no | no | no | `.j2,.jinja,.jinja2` |
| `jq` | not applicable: no external scanner | IV-unknown | 7/8 | no | no | no | `.jq` |
| `json` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.json` |
| `json5` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.json5` |
| `ledger` | not applicable: no external scanner | IV-unknown | 2/4 | no | no | no | `.journal,.ledger` |
| `linkerscript` | not applicable: no external scanner | IV-recovery | 1/40 | no | no | no | `.ld,.lds` |
| `llvm` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.ll` |
| `make` | not applicable: no external scanner | IV-recovery? | unmeasured | no | no | no | `.mak,.mk,gnumakefile,makefile` |
| `mermaid` | not applicable: no external scanner | IV-recovery? | 0/40 | no | no | no | `.mermaid,.mmd` |
| `meson` | not applicable: no external scanner | IV-recovery? | 2/40 | no | no | no | `meson.build` |
| `ninja` | not applicable: no external scanner | IV-unknown | 3/5 | no | no | no | `.ninja` |
| `objc` | not applicable: no external scanner | IV-recovery | unmeasured | no | no | no | `.m` |
| `pascal` | not applicable: no external scanner | IV-recovery? | 0/40 | no | no | no | `.inc,.pas,.pp` |
| `pem` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.pem` |
| `prisma` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.prisma` |
| `prolog` | not applicable: no external scanner | IV-recovery? | 4/40 | no | no | no | `.pl,.pro` |
| `promql` | not applicable: no external scanner | IV-unknown | 0/4 | no | no | no | `.promql` |
| `proto` | not applicable: no external scanner | IV-recovery? | 24/40 | no | no | no | `.proto` |
| `puppet` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.pp` |
| `ql` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.ql` |
| `regex` | not applicable: no external scanner | CLEAN | 1/1 | no | no | no | `.regex` |
| `rego` | not applicable: no external scanner | IV-recovery? | 7/40 | no | no | no | `.rego` |
| `requirements` | not applicable: no external scanner | IV-unknown | 8/9 | no | no | no | `requirements.txt` |
| `robot` | not applicable: no external scanner | IV-recovery? | 28/40 | no | no | no | `.robot` |
| `scheme` | not applicable: no external scanner | IV-perf | 23/40 | no | no | no | `.scm,.ss` |
| `smithy` | not applicable: no external scanner | IV-unknown | 34/40 | no | no | no | `.smithy` |
| `solidity` | not applicable: no external scanner | IV-unknown | 6/40 | no | no | no | `.sol` |
| `sparql` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.rq,.sparql` |
| `ssh_config` | not applicable: no external scanner | IV-unknown | 1/2 | no | no | no | `config,ssh_config,sshd_config` |
| `textproto` | not applicable: no external scanner | IV-perf | unmeasured | no | no | no | `.pbtxt,.textproto,.txtpb` |
| `thrift` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.thrift` |
| `tmux` | not applicable: no external scanner | IV-recovery? | 0/1 | no | no | no | `.tmux,tmux.conf` |
| `todotxt` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.txt` |
| `turtle` | not applicable: no external scanner | IV-unknown | 21/40 | no | no | no | `.ttl` |
| `twig` | not applicable: no external scanner | CLEAN | 40/40 | no | no | no | `.twig` |
| `v` | not applicable: no external scanner | IV-recovery? | 5/40 | no | no | no | `.v,.vsh` |
| `verilog` | not applicable: no external scanner | IV-recovery? | unmeasured | no | no | no | `.sv,.svh,.v` |
| `vimdoc` | not applicable: no external scanner | IV-recovery? | unmeasured | no | no | no | `.txt` |
| `wat` | not applicable: no external scanner | IV-recovery? | 0/34 | no | no | no | `.wast,.wat` |
| `zig` | not applicable: no external scanner | IV-unknown | 39/40 | no | no | no | `.zig` |
