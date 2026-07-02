//go:build !grammar_subset

package grammars

func init() {
	registerTokenSourceFactory("authzed", NewAuthzedTokenSourceOrEOF)
	registerTokenSourceFactory("c", NewCTokenSourceOrEOF)
	registerTokenSourceFactory("cpp", NewCTokenSourceOrEOF)
	// Go ships as a grammargen-compiled blob and parses via its baked-in DFA
	// lexer by default. GoTokenSource (wrapping go/scanner) was evaluated as
	// the default here to fix the DFA lexer's context-free ASI approximation
	// (Choice(/\n/, ';', '\0')) — see preferZeroWidthStartAcceptForState —
	// and it does fix that narrow case (the auto-semicolon before a
	// block-closing `}` in TestParityFreshParse/go). But routing all of
	// go's production lexing through GoTokenSource causes a severe
	// performance cliff on real, complex Go sources: parsing this repo's
	// own parser.go (285KB) takes 916ms via the DFA lexer vs. >90s (did not
	// even finish) via GoTokenSource — a >100x regression, most likely GLR
	// stack proliferation from a lexical/ambiguity interaction that the DFA
	// lexer's (buggy but low-fork) zero-width preference happens to avoid.
	// It also did not improve broader corpus parity: N=40 and N=300
	// stdlib-corpus walks produced byte-identical divergence sets and
	// counts (24/40, 234/300) whether or not GoTokenSource was registered,
	// so the corpus-level gap has a different root cause entirely. Given
	// the perf cliff, GoTokenSource stays off the default path here; it
	// remains usable via the public API for callers who explicitly opt in.
	// Recommended follow-up: a stateful external-scanner ASI port for the
	// DFA lexer (mirroring the Lua external-scanner migration below),
	// rather than swapping go's whole lexing backend.
	registerTokenSourceFactory("java", NewJavaTokenSourceOrEOF)
	registerTokenSourceFactory("json", NewJSONTokenSourceOrEOF)
	// Lua now parses via the blob's DFA lexer plus LuaExternalScanner (a
	// line-faithful port of upstream scanner.c), which matches the C oracle
	// where the hand-tuned LuaTokenSource diverged (7/40 corpus parity).
	// LuaTokenSource remains available to downstream callers via the public
	// API.
}
