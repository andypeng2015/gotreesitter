//go:build cobol_precise_els && (!grammar_subset || grammar_subset_cobol)

// STAGED - inactive by default. Build/enable with: -tags cobol_precise_els
//
// The table below is copied verbatim from tree-sitter-cobol parser.c. It is
// not default-elected because Cobol's current real corpus is CICS/embedded-SQL
// input that the pinned grammar does not support; enabling C recovery with this
// table drove BBANK10P.cbl into an unbounded recovery reduction path before any
// oracle comparison frame was reached. Keep this staged until Cobol has either
// a supported corpus slice or a bounded C-recovery fix for the unsupported
// corpus class.
//
// Source: https://github.com/yutaro-sakamoto/tree-sitter-cobol e99dbdc3d800d5fa2796476efd60af91f6b43d93 src/parser.c

package grammars

// cobolExternalLexStates mirrors C tree-sitter ts_external_scanner_states.
var cobolExternalLexStates = [][]bool{
	/* 0 */ {false, false, false, false, false, false},
	/* 1 */ {true, true, true, true, true, true},
	/* 2 */ {true, true, true, true, false, false},
	/* 3 */ {true, true, true, true, false, true},
	/* 4 */ {true, true, true, true, true, false},
}

func init() {
	RegisterExternalLexStates("cobol", cobolExternalLexStates)
}
