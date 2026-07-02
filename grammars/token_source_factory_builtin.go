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
	//
	// The zero-width ASI bug above is now fixed properly: the grammar's
	// `terminator` rule routes its newline/EOF alternatives through
	// GoExternalScanner (grammars/go_scanner.go, registered below in
	// zzz_scanner_attachments.go), a small stateless external scanner
	// mirroring JavaScriptExternalScanner's `_automatic_semicolon` handling,
	// so the DFA lexer's shared-state tie-break never has to arbitrate
	// between the real newline and the zero-width EOF sentinel. This
	// required a matching, narrow fix in parser_retry.go
	// (goFullParseNeedsBracketComparisonMergeWidth): restructuring the
	// grammar's terminator rule shifts enough of the LALR table that Go's
	// pre-existing, upstream-intentional dynamic-precedence tie between
	// index_expression and generic_type(composite_literal) needed one more
	// merge-per-key survivor on `identifier[identifier] != identifier[identifier] {`
	// shapes; that fix is gated on a cheap source-content probe rather than
	// a global cap change to avoid the >10x parse-time cost widening the cap
	// unconditionally would add on large, unrelated real files.
	registerTokenSourceFactory("java", NewJavaTokenSourceOrEOF)
	registerTokenSourceFactory("json", NewJSONTokenSourceOrEOF)
	// Lua now parses via the blob's DFA lexer plus LuaExternalScanner (a
	// line-faithful port of upstream scanner.c), which matches the C oracle
	// where the hand-tuned LuaTokenSource diverged (7/40 corpus parity).
	// LuaTokenSource remains available to downstream callers via the public
	// API.
}
