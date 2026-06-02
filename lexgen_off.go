//go:build !lexgen

package gotreesitter

// lookupGenScan returns nil in the default build, so Lexer.Next uses the table
// scan(). The generated switch-DFA lexers (cmd/lexgen) and their dispatch
// compile only under -tags lexgen, so the default binary carries no codegen
// bloat.
func lookupGenScan(name string) lexerScanFn { return nil }
