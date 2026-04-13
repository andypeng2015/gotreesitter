package grammars

import (
	"errors"

	"github.com/odvcencio/gotreesitter"
)

// DecodeLanguageBlob decodes a gzip+gob encoded grammar blob (as produced by
// the ts2go pipeline and stored under grammars/grammar_blobs/*.bin) into a
// ready-to-use *gotreesitter.Language.
//
// This is the public entry point for consumer packages that ship their own
// grammar blobs via go:embed rather than having the grammar absorbed into
// gotreesitter's built-in registry. The expected usage pattern is to pair
// DecodeLanguageBlob with [Register] or [RegisterExternalScanner] from an
// init() function in the consumer package, so the grammar becomes available
// through [DetectLanguage], [DetectLanguageByName], and [AllLanguages] without
// modifying gotreesitter itself.
//
// The input bytes are consumed once; the returned Language is safe to cache
// and reuse across parsers. Returns an error if the input is empty or cannot
// be decoded (wrong format, truncated data, incompatible schema).
//
// See the package example for an end-to-end consumer integration.
func DecodeLanguageBlob(data []byte) (*gotreesitter.Language, error) {
	if len(data) == 0 {
		return nil, errors.New("gotreesitter: DecodeLanguageBlob called with empty data")
	}
	return decodeLanguageBlobData("<external>", data)
}
