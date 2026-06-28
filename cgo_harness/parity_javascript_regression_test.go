//go:build cgo && treesitter_c_parity

package cgoharness

import "testing"

func TestParityJavaScriptStandaloneBlockBeforeSimpleAssignment(t *testing.T) {
	src := []byte("{a}b=c")
	runParityCase(t, parityCase{name: "javascript", source: string(src)}, "issue111", src)
}
