# Grammar tiers — unreleased

Generated 2026-06-12T07:56:13Z at `dc25e8e5`. Parity vs the
tree-sitter C oracle is hard evidence; performance is the clean sub-rank
(rules in `cgo_harness/tier_scan/README.md`).

| tier | count |
| --- | --- |
| I | 38 |
| II | 48 |
| III | 106 |
| unranked | 14 |

## Tier I — parity-clean, fast (38)

`astro`, `bibtex`, `clojure`, `css`, `csv`, `cue`, `dhall`, `elisp`, `faust`, `fidl`, `fish`, `gdscript`, `gitcommit`, `gleam`, `hcl`, `java`, `javascript`, `llvm`, `lua`, `nickel`, `nix`, `php`, `pkl`, `prisma`, `puppet`, `r`, `racket`, `smithy`, `squirrel`, `starlark`, `svelte`, `thrift`, `tsx`, `turtle`, `xml`, `yaml`, `yuck`, `zig`

## Tier II — parity-clean, ok (48)

`arduino`, `bass`, `beancount`, `capnp`, `chatito`, `cmake`, `corn`, `cpon`, `devicetree`, `dot`, `editorconfig`, `elm`, `foam`, `forth`, `fortran`, `git_config`, `git_rebase`, `gitattributes`, `gitignore`, `gn`, `godot_resource`, `hack`, `heex`, `ini`, `janet`, `jq`, `jsdoc`, `json`, `json5`, `just`, `ledger`, `markdown`, `ocaml`, `pem`, `promql`, `python`, `ql`, `requirements`, `ron`, `ruby`, `sparql`, `tablegen`, `textproto`, `todotxt`, `toml`, `twig`, `typescript`, `vue`

## Tier III — parity-clean poor perf or assessed heavy work (106)

### Parity-clean, poor perf

`desktop`, `diff`, `dtd`, `eds`, `eex`, `embedded_template`, `facility`, `gomod`, `http`, `nginx`, `ninja`, `properties`, `ssh_config`

### Assessed non-clean work

| grammar | cause | parity |
| --- | --- | --- |
| `agda` | III-scanner | 2/40 |
| `angular` | III-recovery? | 35/40 |
| `asm` | III-recovery | 0/40 |
| `awk` | III-recovery | 28/29 |
| `bash` | III-recovery? | 30/40 |
| `bitbake` | III-recovery | 35/40 |
| `blade` | III-recovery? | 17/30 |
| `brightscript` | III-recovery? | 0/30 |
| `c` | III-recovery | 21/40 |
| `c_sharp` | III-recovery | 26/40 |
| `caddy` | III-recovery? | 9/30 |
| `cairo` | III-recovery? | 0/30 |
| `circom` | III-shape? | 11/30 |
| `cobol` | III-version | 0/40 |
| `commonlisp` | III-recovery? | 22/30 |
| `cooklang` | III-recovery | 0/3 |
| `cpp` | III-recovery | 10/40 |
| `crystal` | III-perf | 0/0 |
| `cuda` | III-recovery? | 17/30 |
| `cylc` | III-recovery? | 4/30 |
| `d` | III-recovery? | 14/30 |
| `dart` | III-recovery? | 11/30 |
| `disassembly` | III-version | 0/40 |
| `djot` | III-scanner? | 0/40 |
| `dockerfile` | III-recovery? | 0/30 |
| `earthfile` | III-recovery? | 0/30 |
| `ebnf` | III-recovery? | 0/30 |
| `elixir` | III-recovery? | 0/40 |
| `elsa` | III-recovery? | 12/27 |
| `erlang` | III-recovery | 38/40 |
| `fennel` | III-recovery? | 8/30 |
| `firrtl` | III-recovery? | 5/27 |
| `fsharp` | III-perf | 0/8 |
| `glsl` | III-recovery | 11/40 |
| `groovy` | III-recovery | 4/40 |
| `hare` | III-recovery | 20/40 |
| `haskell` | III-scanner | 11/40 |
| `haxe` | III-recovery | 7/40 |
| `html` | III-recovery | 0/40 |
| `hurl` | III-recovery | 13/40 |
| `jinja2` | III-recovery | 3/40 |
| `jsonnet` | III-recovery? | 39/40 |
| `julia` | III-recovery | 28/40 |
| `kconfig` | III-recovery? | 13/30 |
| `kdl` | III-recovery | 12/40 |
| `kotlin` | III-shape? | 17/40 |
| `less` | III-recovery? | 10/40 |
| `linkerscript` | III-recovery | 1/40 |
| `liquid` | III-recovery? | 11/36 |
| `make` | III-recovery? | 19/20 |
| `markdown_inline` | III-scanner | 38/40 |
| `matlab` | III-recovery? | 4/40 |
| `mermaid` | III-recovery? | 0/40 |
| `meson` | III-recovery? | 1/30 |
| `mojo` | III-recovery? | 30/40 |
| `move` | III-recovery? | 14/40 |
| `nim` | III-recovery? | 3/40 |
| `norg` | III-scanner | 0/2 |
| `nushell` | III-recovery? | 5/40 |
| `objc` | III-recovery? | 1/40 |
| `odin` | III-recovery? | 13/40 |
| `org` | III-recovery? | 5/39 |
| `pascal` | III-recovery? | 0/40 |
| `perl` | III-recovery? | 0/40 |
| `powershell` | III-recovery? | 22/40 |
| `prolog` | III-recovery? | 4/40 |
| `proto` | III-recovery? | 25/40 |
| `pug` | III-recovery? | 0/40 |
| `purescript` | III-recovery? | 1/40 |
| `regex` | III-perf | 0/1 |
| `rego` | III-recovery? | 7/40 |
| `rescript` | III-recovery? | 23/40 |
| `robot` | III-recovery? | 28/40 |
| `rst` | III-perf | 1/8 |
| `rust` | III-recovery? | 21/40 |
| `scala` | III-recovery? | 25/40 |
| `scheme` | III-perf | 36/40 |
| `scss` | III-recovery? | 6/40 |
| `sql` | III-recovery? | 8/40 |
| `swift` | III-recovery? | 0/40 |
| `tcl` | III-recovery? | 10/40 |
| `teal` | III-recovery? | 4/40 |
| `templ` | III-recovery? | 24/40 |
| `tlaplus` | III-perf | 14/40 |
| `tmux` | III-recovery? | 0/1 |
| `typst` | III-recovery? | 28/40 |
| `uxntal` | III-recovery? | 0/40 |
| `v` | III-recovery? | 25/40 |
| `verilog` | III-recovery? | 4/40 |
| `vhdl` | III-recovery? | 14/40 |
| `vimdoc` | III-recovery? | 0/30 |
| `wat` | III-recovery? | 4/34 |
| `wgsl` | III-recovery? | 20/40 |

## Unranked — parity-clean, perf measurement pending (14)

`ada`, `apex`, `authzed`, `bicep`, `comment`, `doxygen`, `enforce`, `go`, `graphql`, `hlsl`, `hyprlang`, `luau`, `solidity`, `wolfram`

## Tier IV — unassessed or unknown (0)

**Empty.** Every non-clean grammar has an assessed classification.
