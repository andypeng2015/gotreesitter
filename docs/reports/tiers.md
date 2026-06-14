# Grammar tiers — unreleased

Generated 2026-06-14T05:53:05Z at `e895278e`. Parity vs the
tree-sitter C oracle is the hard gate; performance is the sub-rank
(rules in `cgo_harness/tier_scan/README.md`).

| tier | count |
| --- | --- |
| I | 38 |
| II | 48 |
| III | 13 |
| unranked | 18 |
| IV | 89 |

## Tier I — parity-clean, fast (38)

`astro`, `bibtex`, `clojure`, `css`, `csv`, `cue`, `dhall`, `elisp`, `faust`, `fidl`, `fish`, `gdscript`, `gitcommit`, `gleam`, `hcl`, `java`, `javascript`, `llvm`, `lua`, `nickel`, `nix`, `php`, `pkl`, `prisma`, `puppet`, `r`, `racket`, `smithy`, `squirrel`, `starlark`, `svelte`, `thrift`, `tsx`, `turtle`, `xml`, `yaml`, `yuck`, `zig`

## Tier II — parity-clean, ok (48)

`arduino`, `bass`, `beancount`, `capnp`, `chatito`, `cmake`, `corn`, `cpon`, `devicetree`, `dot`, `editorconfig`, `elm`, `foam`, `forth`, `fortran`, `git_config`, `git_rebase`, `gitattributes`, `gitignore`, `gn`, `godot_resource`, `hack`, `heex`, `ini`, `janet`, `jq`, `jsdoc`, `json`, `json5`, `just`, `ledger`, `markdown`, `ocaml`, `pem`, `promql`, `python`, `ql`, `requirements`, `ron`, `ruby`, `sparql`, `tablegen`, `textproto`, `todotxt`, `toml`, `twig`, `typescript`, `vue`

## Tier III — parity-clean, poor perf (13)

`desktop`, `diff`, `dtd`, `eds`, `eex`, `embedded_template`, `facility`, `gomod`, `http`, `nginx`, `ninja`, `properties`, `ssh_config`

## Unranked — parity-clean, perf measurement pending (18)

`ada`, `apex`, `authzed`, `awk`, `bicep`, `bitbake`, `comment`, `doxygen`, `enforce`, `go`, `graphql`, `hlsl`, `hyprlang`, `kotlin`, `luau`, `markdown_inline`, `solidity`, `wolfram`

## Tier IV — not parity-clean (89)

| grammar | cause | parity |
| --- | --- | --- |
| `agda` | IV-scanner | 2/40 |
| `angular` | IV-recovery? | 37/40 |
| `asm` | IV-recovery | 0/40 |
| `bash` | IV-recovery? | 31/40 |
| `blade` | IV-recovery? | 17/30 |
| `brightscript` | IV-recovery? | 0/30 |
| `c` | IV-recovery | 21/40 |
| `c_sharp` | IV-recovery | 30/40 |
| `caddy` | IV-recovery? | 9/30 |
| `cairo` | IV-recovery? | 0/30 |
| `circom` | IV-shape? | 11/30 |
| `cobol` | IV-version | 0/40 |
| `commonlisp` | IV-recovery? | 22/30 |
| `cooklang` | IV-recovery | 1/3 |
| `cpp` | IV-recovery | 10/40 |
| `crystal` | IV-shape? | 19/40 |
| `cuda` | IV-recovery? | 17/30 |
| `cylc` | IV-recovery? | 4/30 |
| `d` | IV-recovery? | 14/30 |
| `dart` | IV-recovery? | 11/30 |
| `disassembly` | IV-version | 0/40 |
| `djot` | IV-scanner? | 0/40 |
| `dockerfile` | IV-recovery? | 0/30 |
| `earthfile` | IV-recovery? | 0/30 |
| `ebnf` | IV-recovery? | 0/30 |
| `elixir` | IV-recovery | 9/40 |
| `elsa` | IV-recovery? | 12/27 |
| `erlang` | IV-recovery | 39/40 |
| `fennel` | IV-recovery? | 8/30 |
| `firrtl` | IV-recovery? | 5/27 |
| `fsharp` | IV-perf | 0/8 |
| `glsl` | IV-recovery | 11/40 |
| `groovy` | IV-recovery | 4/40 |
| `hare` | IV-recovery | 20/40 |
| `haskell` | IV-scanner | 11/40 |
| `haxe` | IV-recovery | 7/40 |
| `html` | IV-recovery | 0/40 |
| `hurl` | IV-recovery | 14/40 |
| `jinja2` | IV-recovery | 3/40 |
| `jsonnet` | IV-recovery? | 39/40 |
| `julia` | IV-recovery | 32/40 |
| `kconfig` | IV-recovery? | 13/30 |
| `kdl` | IV-recovery | 12/40 |
| `less` | IV-recovery? | 10/40 |
| `linkerscript` | IV-recovery | 21/40 |
| `liquid` | IV-recovery? | 11/36 |
| `make` | IV-recovery? | 19/20 |
| `matlab` | IV-recovery? | 4/40 |
| `mermaid` | IV-recovery? | 0/40 |
| `meson` | IV-recovery? | 1/30 |
| `mojo` | IV-recovery? | 30/40 |
| `move` | IV-recovery? | 14/40 |
| `nim` | IV-recovery? | 3/40 |
| `norg` | IV-scanner | 0/2 |
| `nushell` | IV-recovery? | 5/40 |
| `objc` | IV-recovery | 25/40 |
| `odin` | IV-recovery? | 13/40 |
| `org` | IV-recovery? | 5/39 |
| `pascal` | IV-recovery? | 0/40 |
| `perl` | IV-recovery? | 0/40 |
| `powershell` | IV-recovery? | 25/40 |
| `prolog` | IV-recovery? | 4/40 |
| `proto` | IV-recovery? | 26/40 |
| `pug` | IV-recovery? | 0/40 |
| `purescript` | IV-recovery? | 1/40 |
| `regex` | IV-perf | 0/1 |
| `rego` | IV-recovery? | 7/40 |
| `rescript` | IV-recovery? | 29/40 |
| `robot` | IV-recovery? | 29/40 |
| `rst` | IV-perf | 1/8 |
| `rust` | IV-recovery? | 21/40 |
| `scala` | IV-recovery? | 25/40 |
| `scheme` | IV-perf | 28/28 |
| `scss` | IV-recovery? | 6/40 |
| `sql` | IV-recovery? | 8/40 |
| `swift` | IV-recovery? | 0/40 |
| `tcl` | IV-recovery? | 10/40 |
| `teal` | IV-recovery? | 4/40 |
| `templ` | IV-recovery? | 35/40 |
| `tlaplus` | IV-perf | 14/40 |
| `tmux` | IV-recovery? | 0/1 |
| `typst` | IV-recovery? | 35/40 |
| `uxntal` | IV-recovery? | 0/40 |
| `v` | IV-recovery? | 25/40 |
| `verilog` | IV-recovery? | 4/40 |
| `vhdl` | IV-recovery? | 14/40 |
| `vimdoc` | IV-recovery? | 0/30 |
| `wat` | IV-recovery? | 4/34 |
| `wgsl` | IV-recovery | 37/40 |
