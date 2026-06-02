package gotreesitter

import (
	"fmt"
	"testing"
)

// TestScanJsonGenMatchesTable is the differential gate for the generated json
// lexer: scanJsonGen must return byte-for-byte the same token (and advance the
// lexer to the same position) as the table-driven scan() for EVERY lex state at
// EVERY position across a diverse input set. A codegen bug shows up here as a
// concrete divergence before scanJsonGen is ever wired into a real parse.
func TestScanJsonGenMatchesTable(t *testing.T) {
	states := jsonLexStatesForDiff

	inputs := [][]byte{
		[]byte(`{"a": 1, "b": [true, false, null], "c": "hi\n"}`),
		[]byte(`[1, 2.5, -3, 4e10, 5.6E-7, 0, -0.0]`),
		[]byte(`"string with \"escapes\", \\, \/, é, \t"`),
		[]byte(`{ "nested": { "deep": [ [ [ 1 ] ] ] } }`),
		[]byte("\t \n\r  123  \n"),
		[]byte(`truefalsenull`),
		[]byte(`12345678901234567890.12345e+99`),
		[]byte(`"é日本語�lang"`),
		[]byte(``),
		[]byte(` `),
		[]byte(`}`),
		[]byte(`\`),
	}
	// Synthetic: every byte 0-255 as a lone char and as a 2-char lead.
	for b := 0; b < 256; b++ {
		inputs = append(inputs, []byte{byte(b)}, []byte{byte(b), '1'}, []byte{byte(b), '"'})
	}

	mismatches := 0
	for _, src := range inputs {
		for state := 0; state < len(states); state++ {
			for pos := 0; pos <= len(src); pos++ {
				la := NewLexer(states, src)
				ta, oka := la.scan(uint32(state), pos, 0, 0)
				lb := NewLexer(states, src)
				tb, okb := scanJsonGen(lb, uint32(state), pos, 0, 0)
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
		t.Fatalf("scanJsonGen diverged from table scan() in %d (state,pos,input) cases", mismatches)
	}
}

func truncq(b []byte) string {
	if len(b) > 32 {
		return fmt.Sprintf("%q…", b[:32])
	}
	return fmt.Sprintf("%q", b)
}
