//go:build !grammar_subset || grammar_subset_sql

package grammars

import gotreesitter "github.com/odvcencio/gotreesitter"

// External token indexes for the SQL grammar (DerekStride/tree-sitter-sql).
const (
	sqlTokDollarTagStart = 0 // "_dollar_quoted_string_tag" — opening $tag$
	sqlTokContent        = 1 // "content" — body between tags
	sqlTokDollarTagEnd   = 2 // "_dollar_quoted_string_end_tag" — closing $tag$
)

// sqlScannerState stores the dollar-quote tag for matching the closing delimiter.
type sqlScannerState struct {
	tag string // empty when not inside a dollar-quoted string
}

// SqlExternalScanner implements gotreesitter.ExternalScanner for tree-sitter-sql.
//
// This is a Go port of the C external scanner from DerekStride/tree-sitter-sql.
// The scanner handles PostgreSQL dollar-quoted strings: $tag$..content..$tag$.
// It uses state to remember the opening tag so the closing tag can be matched.
type SqlExternalScanner struct{}

func (SqlExternalScanner) Create() any {
	return &sqlScannerState{}
}

func (SqlExternalScanner) Destroy(payload any) {}

func (SqlExternalScanner) Serialize(payload any, buf []byte) int {
	s := payload.(*sqlScannerState)
	if s.tag == "" {
		return 0
	}
	// Store tag + null terminator.
	tagLen := len(s.tag) + 1
	if tagLen > len(buf) {
		return 0
	}
	copy(buf, s.tag)
	buf[len(s.tag)] = 0
	return tagLen
}

func (SqlExternalScanner) Deserialize(payload any, buf []byte) {
	s := payload.(*sqlScannerState)
	s.tag = ""
	if len(buf) > 1 {
		// Find the null terminator.
		for i, b := range buf {
			if b == 0 {
				s.tag = string(buf[:i])
				return
			}
		}
		// No null terminator found — use the whole buffer.
		s.tag = string(buf)
	}
}

func (SqlExternalScanner) Scan(payload any, lexer *gotreesitter.ExternalLexer, validSymbols []bool) bool {
	s := payload.(*sqlScannerState)
	lang := SqlLanguage()
	tagStartSym := lang.ExternalSymbols[sqlTokDollarTagStart]
	contentSym := lang.ExternalSymbols[sqlTokContent]
	tagEndSym := lang.ExternalSymbols[sqlTokDollarTagEnd]

	// Start tag: scan a $..$ tag and store it.
	if sqlValid(validSymbols, sqlTokDollarTagStart) && s.tag == "" {
		tag, ok := scanSqlDollarTag(lexer, true)
		if !ok {
			return false
		}
		s.tag = tag
		lexer.MarkEnd()
		lexer.SetResultSymbol(tagStartSym)
		return true
	}

	// Content: scan the body between dollar-quote tags, stopping immediately
	// before the matching closing tag. Non-matching '$' sequences are content.
	if sqlValid(validSymbols, sqlTokContent) && s.tag != "" {
		return scanSqlDollarContent(lexer, s.tag, contentSym)
	}

	// End tag: scan for a matching $..$ tag.
	if sqlValid(validSymbols, sqlTokDollarTagEnd) && s.tag != "" {
		tag, ok := scanSqlDollarTag(lexer, false)
		if !ok || tag != s.tag {
			return false
		}
		s.tag = ""
		lexer.MarkEnd()
		lexer.SetResultSymbol(tagEndSym)
		return true
	}

	return false
}

// scanSqlDollarTag scans a $identifier$ or $$ tag and returns the full tag
// string (including the $ delimiters). Returns ("", false) if the current
// position doesn't start a valid dollar tag.
func scanSqlDollarTag(lexer *gotreesitter.ExternalLexer, skipWhitespace bool) (string, bool) {
	if skipWhitespace {
		for sqlIsScannerWhitespace(lexer.Lookahead()) {
			lexer.Advance(true)
		}
	}
	if lexer.Lookahead() != '$' {
		return "", false
	}
	lexer.Advance(false)

	var tag []byte
	tag = append(tag, '$')

	// Tag identifier: [a-zA-Z_][a-zA-Z0-9_]* or empty (for $$).
	ch := lexer.Lookahead()
	if ch == '$' {
		// Empty tag: $$
		lexer.Advance(false)
		tag = append(tag, '$')
		return string(tag), true
	}

	if !isSqlTagStart(ch) {
		return "", false
	}

	for isSqlTagChar(lexer.Lookahead()) {
		tag = append(tag, byte(lexer.Lookahead()))
		lexer.Advance(false)
	}

	if lexer.Lookahead() != '$' {
		return "", false
	}
	lexer.Advance(false)
	tag = append(tag, '$')

	return string(tag), true
}

// scanSqlDollarContent scans the body of a dollar-quoted string. It mirrors
// upstream tree-sitter-sql: content ends only before the full matching tag.
func scanSqlDollarContent(lexer *gotreesitter.ExternalLexer, tag string, contentSym gotreesitter.Symbol) bool {
	if tag == "" {
		return false
	}
	pos := 0
	lexer.SetResultSymbol(contentSym)
	lexer.MarkEnd()
	for {
		ch := lexer.Lookahead()
		if ch == 0 {
			return false
		}
		if pos < len(tag) && ch == rune(tag[pos]) {
			if pos == len(tag)-1 {
				return true
			}
			if pos == 0 {
				lexer.SetResultSymbol(contentSym)
				lexer.MarkEnd()
			}
			pos++
			lexer.Advance(false)
			continue
		}
		if pos != 0 {
			pos = 0
			continue
		}
		lexer.Advance(false)
	}
}

func isSqlTagStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isSqlTagChar(ch rune) bool {
	return isSqlTagStart(ch) || (ch >= '0' && ch <= '9')
}

func sqlIsScannerWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
}

func sqlValid(validSymbols []bool, idx int) bool {
	return idx >= 0 && idx < len(validSymbols) && validSymbols[idx]
}
