//go:build javascript_precise_els

// STAGED — inactive by default. Build/enable with: -tags javascript_precise_els
//
// Verbatim ts_external_scanner_states[10][8] from the pinned
// tree-sitter-javascript parser.c. Registering it flips DiagnoseCRecoveryGate
// for javascript (precise ExternalLexStates is the last missing gate input),
// which ELECTS javascript into the faithful C error-recovery cost competition
// by default.
//
// WHY IT IS STAGED (2026-07 cliff campaign measurement)
// Unlike c_sharp — whose stateful interpolation scanner was actively
// corrupted by the union-mask fallback, so the precise table + election fixed
// real misparses (DeclaredTypeManager.cs 31 ERROR nodes -> 0) — javascript's
// worst perf_scan sweep files parse with ZERO error nodes either way, so the
// table has nothing to fix there today, and ELECTION is a measured
// regression on that slice (loaded-box interleaved A/B, elected vs
// GOT_C_RECOVERY=0, live engine with both campaign engine fixes):
//
//	deps/undici/undici.js               2.2-2.4s -> ~5.1s   (accepted both)
//	deps/v8/.../unicode-test.js         ~370ms   -> ~600ms  (median of 4+4)
//	deps/v8/.../box2d.js                memory_budget @49543 either way
//	deps/v8/.../lua_binarytrees.js      memory_budget @6745 either way
//	deps/amaro/dist/index.js            ~100-135ms either way
//
// The elected path's per-token cost (cRecoveryEnabled lexing via
// NextWithErrorRuns + acquire-token bookkeeping) is paid on CLEAN parses,
// and javascript's actual sweep cliffs (asm.js megafunctions dying on
// memory_budget) are unrelated to external-scanner validity. Enable this
// only after the javascript memory-budget cliff class is fixed and a sweep
// re-measurement shows election is at worst neutral. Full test suites
// (go test . ./grammars) PASS with this table active and javascript elected,
// so correctness is not the blocker — perf on the cliff slice is.
//
// Columns are the JavaScript external token indices, in the language's
// ExternalSymbols order (== C enum order):
//
//	0 _automatic_semicolon  4 ||
//	1 _template_chars       5 escape_sequence
//	2 _ternary_qmark        6 regex_pattern
//	3 html_comment          7 jsx_text
//
// Source: https://github.com/tree-sitter/tree-sitter-javascript 58404d8cf191d69f2674a8fd507bd5776f46cb11 src/parser.c
package grammars

// javascriptExternalLexStates mirrors C tree-sitter ts_external_scanner_states.
var javascriptExternalLexStates = [][]bool{
	/* 0 */ {false, false, false, false, false, false, false, false},
	/* 1 */ {true, true, true, true, true, true, false, true},
	/* 2 */ {false, false, false, true, false, false, false, false},
	/* 3 */ {true, false, true, true, true, false, false, false},
	/* 4 */ {false, false, true, true, true, false, false, false},
	/* 5 */ {true, false, false, true, false, false, false, false},
	/* 6 */ {false, false, false, true, false, false, false, true},
	/* 7 */ {false, true, false, true, false, true, false, false},
	/* 8 */ {false, false, false, true, false, true, false, false},
	/* 9 */ {false, false, false, true, false, false, true, false},
}

func init() {
	RegisterExternalLexStates("javascript", javascriptExternalLexStates)
}
