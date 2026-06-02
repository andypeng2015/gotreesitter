package gotreesitter

import (
	"bytes"
	"testing"
)

// jsonAscii is the real runtime ASCII fast-path table, built once. The table
// baseline MUST use it (the runtime does, via Language.LexAsciiTable) or the
// comparison unfairly pits scanJsonGen against the slow linear-scan fallback.
var jsonAscii = buildLexAsciiTable(jsonLexStatesForDiff)

// tokenizeAll lexes the whole buffer from the DFA start state, advancing by each
// token's length (1 byte on a non-accept). Both scanners are differential-clean,
// so they do identical work — a fair head-to-head of the DFA inner loop. The
// lexer is given the real ASCII fast-path table (scanJsonGen ignores it).
func tokenizeAll(src []byte, scan scanFunc) int {
	l := NewLexer(jsonLexStatesForDiff, src)
	l.asciiTable = jsonAscii
	pos, n := 0, 0
	for pos < len(src) {
		_, ok := scan(l, 0, pos, 0, 0)
		if ok && l.pos > pos {
			pos = l.pos
		} else {
			pos++
		}
		n++
	}
	return n
}

func benchJSON() []byte {
	unit := []byte(`{"id":12345,"name":"item-name","active":true,"ratio":3.14159e-2,"tags":["a","b","c"],"nested":{"x":-7,"y":null,"s":"with \"escapes\" and \t"}},`)
	var buf bytes.Buffer
	for buf.Len() < 256*1024 {
		buf.Write(unit)
	}
	return buf.Bytes()
}

func BenchmarkLexJsonTable(b *testing.B) {
	src := benchJSON()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizeAll(src, tableScanFunc)
	}
}

func BenchmarkLexJsonGen(b *testing.B) {
	src := benchJSON()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizeAll(src, scanJsonGen)
	}
}
