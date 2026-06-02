package gotreesitter

import (
	"fmt"
	"testing"
)

// scanFunc abstracts the table scan() (a method) and the generated scan<Lang>Gen
// (a func) so one differential/bench loop drives both.
type scanFunc func(l *Lexer, startState uint32, startPos int, startRow, startCol uint32) (Token, bool)

func tableScanFunc(l *Lexer, s uint32, p int, r, c uint32) (Token, bool) { return l.scan(s, p, r, c) }

// diffInputs is a language-agnostic input set: exhaustive single/multi-byte
// sequences (transition coverage from every state) plus mixed snippets.
func diffInputs() [][]byte {
	inputs := [][]byte{
		[]byte(`{"a": 1, "b": [true, false, null], "c": "hi\n"}`),
		[]byte(`[1, 2.5, -3, 4e10, 5.6E-7, 0, -0.0]`),
		[]byte("int main(void){ return 0; } /* c */ // x\n#define A 1\n'a'"),
		[]byte("const x = (a,b) => a+b; `tpl ${y}`; 0x1F; 1_000; /re/g"),
		[]byte("\t \n\r  abc  \n"),
		[]byte(`"é日本語é\t\\"`),
		[]byte(``), []byte(` `), []byte(`}`), []byte(`\`),
	}
	for b := 0; b < 256; b++ {
		inputs = append(inputs,
			[]byte{byte(b)},
			[]byte{byte(b), '1'},
			[]byte{byte(b), '"'},
			[]byte{'a', byte(b), 'z'},
		)
	}
	return inputs
}

// runLexerDifferential asserts gen == table scan() for every state at every
// position across diffInputs. It gives BOTH lexers the language's actual
// immediate/zero-width tables so the table baseline matches the runtime exactly.
func runLexerDifferential(t *testing.T, states []LexState, immediate, zeroWidth []bool, gen scanFunc) {
	t.Helper()
	mismatches := 0
	for _, src := range diffInputs() {
		for state := 0; state < len(states); state++ {
			for pos := 0; pos <= len(src); pos++ {
				la := NewLexer(states, src)
				la.immediateTokens, la.zeroWidthTokens = immediate, zeroWidth
				ta, oka := la.scan(uint32(state), pos, 0, 0)
				lb := NewLexer(states, src)
				lb.immediateTokens, lb.zeroWidthTokens = immediate, zeroWidth
				tb, okb := gen(lb, uint32(state), pos, 0, 0)
				if oka != okb || ta != tb || la.pos != lb.pos {
					mismatches++
					if mismatches <= 12 {
						t.Errorf("DIVERGE state=%d pos=%d src=%q\n  table: ok=%v tok=%+v pos=%d\n  gen:   ok=%v tok=%+v pos=%d",
							state, pos, truncq(src), oka, ta, la.pos, okb, tb, lb.pos)
					}
				}
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("generated scan diverged from table scan() in %d (state,pos,input) cases", mismatches)
	}
}

func truncq(b []byte) string {
	if len(b) > 40 {
		return fmt.Sprintf("%q…", b[:40])
	}
	return fmt.Sprintf("%q", b)
}
