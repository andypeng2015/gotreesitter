//go:build cgo && treesitter_c_parity

package cgoharness

// top50ParityLanguages is the lock-step top-50 correctness surface used by
// grammars/update_tier1_top50.txt and cgo_harness/testdata/top50_manifest.json.
// The order is user-impact priority, not alphabetical.
var top50ParityLanguages = []string{
	"typescript",
	"tsx",
	"javascript",
	"python",
	"java",
	"c_sharp",
	"php",
	"bash",
	"cpp",
	"go",
	"html",
	"css",
	"sql",
	"c",
	"rust",
	"json",
	"ruby",
	"swift",
	"kotlin",
	"dart",
	"lua",
	"yaml",
	"xml",
	"toml",
	"markdown",
	"svelte",
	"scss",
	"powershell",
	"r",
	"scala",
	"hcl",
	"graphql",
	"perl",
	"elixir",
	"haskell",
	"julia",
	"clojure",
	"erlang",
	"ocaml",
	"nix",
	"objc",
	"gomod",
	"json5",
	"ini",
	"zig",
	"make",
	"cmake",
	"d",
	"awk",
	"elm",
}

var top50ParityLanguageSet = func() map[string]bool {
	out := make(map[string]bool, len(top50ParityLanguages))
	for _, name := range top50ParityLanguages {
		out[name] = true
	}
	return out
}()
