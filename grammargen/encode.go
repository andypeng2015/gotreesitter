package grammargen

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"reflect"

	"github.com/odvcencio/gotreesitter"
)

// Generate compiles a Grammar definition into a binary blob that
// gotreesitter can load via DecodeLanguageBlob / loadEmbeddedLanguage.
// LR(1) state splitting is always attempted; a rollback guard reverts to the
// plain LALR table if splitting does not reduce GLR conflicts.
func Generate(g *Grammar) ([]byte, error) {
	report, err := generateWithReport(g, reportBuildOptions{includeLanguage: true, includeBlob: true})
	if err != nil {
		return nil, err
	}
	return report.Blob, nil
}

// GenerateLanguage compiles a Grammar into a Language struct without encoding.
// LR(1) state splitting is always attempted; a rollback guard reverts to the
// plain LALR table if splitting does not reduce GLR conflicts.
func GenerateLanguage(g *Grammar) (*gotreesitter.Language, error) {
	return GenerateLanguageWithContext(context.Background(), g)
}

// GenerateLanguageAndBlob compiles a Grammar into both a Language and its
// serialized blob representation in a single generation pass.
func GenerateLanguageAndBlob(g *Grammar) (*gotreesitter.Language, []byte, error) {
	return GenerateLanguageAndBlobWithContext(context.Background(), g)
}

// GenerateLanguageWithContext is like GenerateLanguage but accepts a context
// for cancellation. When the context is cancelled, LR table construction and
// DFA building abort promptly, allowing the caller to reclaim memory that
// would otherwise be held by an orphaned goroutine.
func GenerateLanguageWithContext(ctx context.Context, g *Grammar) (*gotreesitter.Language, error) {
	report, err := generateWithReportCtx(ctx, g, reportBuildOptions{includeLanguage: true})
	if err != nil {
		return nil, err
	}
	return report.Language, nil
}

// GenerateLanguageAndBlobWithContext is like GenerateLanguageAndBlob but
// accepts a context for cancellation.
func GenerateLanguageAndBlobWithContext(ctx context.Context, g *Grammar) (*gotreesitter.Language, []byte, error) {
	report, err := generateWithReportCtx(ctx, g, reportBuildOptions{includeLanguage: true, includeBlob: true})
	if err != nil {
		return nil, nil, err
	}
	return report.Language, report.Blob, nil
}

// allSymbolsSet returns a set containing all symbol IDs from the patterns.
func allSymbolsSet(patterns []TerminalPattern) map[int]bool {
	s := make(map[int]bool, len(patterns))
	for _, p := range patterns {
		s[p.SymbolID] = true
	}
	return s
}

// computeSkipExtras returns the set of extra symbol IDs that should be
// silently consumed (Skip=true in the DFA). Only invisible/anonymous extras
// are skipped. Visible extras like `comment` produce tree nodes.
func computeSkipExtras(ng *NormalizedGrammar) map[int]bool {
	skip := make(map[int]bool)
	for _, e := range ng.ExtraSymbols {
		if e > 0 && e < len(ng.Symbols) && !ng.Symbols[e].Visible {
			skip[e] = true
		}
	}
	return skip
}

// encodeLanguageBlob serializes a Language using gob+gzip.
//
// Language.LargeStateGotos is a map, and gob's map codec iterates via
// reflect's randomized MapRange, so gob-encoding it directly would make two
// encodes of the same Language produce different bytes (see
// large_state_gotos_trailer.go in the root package for the full mechanism
// and why simpler fixes -- GobEncoder on the field, or a new Language field
// -- don't preserve blob compatibility). When lang.LargeStateGotos is
// non-empty, this instead gob-encodes a shallow copy with the map cleared
// (lang itself is never mutated) and appends a deterministic sorted-pairs
// trailer after the gob payload, inside the same compressed stream. That gzip
// payload is wrapped in a versioned outer envelope so older gzip-first runtimes
// reject it instead of silently losing the trailer.
// decodeLanguageBlob reads that trailer back and repopulates LargeStateGotos
// after the gob decode. When LargeStateGotos is empty -- every language but
// c_sharp today -- this is byte-identical to a plain gob.Encode: no clone,
// no trailer.
func encodeLanguageBlob(lang *gotreesitter.Language) ([]byte, error) {
	toEncode := lang
	var trailer []byte
	if lang != nil && len(lang.LargeStateGotos) > 0 {
		var err error
		trailer, err = gotreesitter.EncodeLargeStateGotosTrailer(lang.LargeStateGotos)
		if err != nil {
			return nil, fmt.Errorf("encode language blob: %w", err)
		}
		clone := shallowCopyExportedLanguageFields(lang)
		clone.LargeStateGotos = nil
		toEncode = clone
	}

	var out bytes.Buffer
	gzw := gzip.NewWriter(&out)
	if err := gob.NewEncoder(gzw).Encode(toEncode); err != nil {
		_ = gzw.Close()
		return nil, fmt.Errorf("encode language blob: %w", err)
	}
	if len(trailer) > 0 {
		if _, err := gzw.Write(trailer); err != nil {
			_ = gzw.Close()
			return nil, fmt.Errorf("encode language blob: %w", err)
		}
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("finalize language blob: %w", err)
	}
	if len(trailer) == 0 {
		return out.Bytes(), nil
	}
	enveloped, err := gotreesitter.WrapLanguageBlobEnvelope(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("finalize language blob: %w", err)
	}
	return enveloped, nil
}

// shallowCopyExportedLanguageFields returns a new *gotreesitter.Language with
// every exported field copied by value from lang, for callers (like
// encodeLanguageBlob) that need to hand gob a modified-but-independent copy
// without disturbing the original.
//
// This deliberately does not use `clone := *lang`. Language embeds several
// sync.Once and atomic.Pointer hot-path caches (symbolMapOnce,
// cRecoveryGateCache, etc.) as unexported fields, so a plain struct copy (a)
// trips `go vet`'s copylocks check, and (b) is not what we want anyway: gob
// only ever transmits exported fields, so the unexported caches have no
// business being duplicated.
//
// This also deliberately does not mutate lang in place (e.g. temporarily
// nil-ing LargeStateGotos and restoring it after encoding) even though
// encodeLanguageBlob's current callers happen not to share lang with another
// goroutine while it runs: Language.LargeStateGotos is read on the parser's
// hot path (parser_tables.go), and a future caller that re-encodes a
// Language that's already in concurrent use elsewhere would otherwise hit an
// unsynchronized "concurrent map read and map write". Copying only the
// exported fields sidesteps both problems: the copy is independent (so
// mutating its LargeStateGotos can never race with a reader of lang's), and
// it never touches the lock-bearing fields at all.
func shallowCopyExportedLanguageFields(lang *gotreesitter.Language) *gotreesitter.Language {
	src := reflect.ValueOf(lang).Elem()
	t := src.Type()
	dstPtr := reflect.New(t)
	dst := dstPtr.Elem()
	for i := 0; i < t.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}
		dst.Field(i).Set(src.Field(i))
	}
	return dstPtr.Interface().(*gotreesitter.Language)
}

// decodeLanguageBlob deserializes a legacy gob+gzip Language blob or a
// version-enveloped blob containing the LargeStateGotos trailer written by
// encodeLanguageBlob. Envelope detection is a cheap no-op for every legacy
// blob that doesn't have one.
func decodeLanguageBlob(data []byte) (*gotreesitter.Language, error) {
	compressed, expectsTrailer, err := gotreesitter.UnwrapLanguageBlobEnvelope(data)
	if err != nil {
		return nil, fmt.Errorf("decode language blob: %w", err)
	}
	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gzr.Close()

	// Decode from a fully-buffered *bytes.Reader, not gzr directly: gob's
	// Decoder can read ahead past its own message boundary on a streaming
	// reader, which would silently swallow the trailer bytes below before we
	// get a chance to read them.
	raw, err := io.ReadAll(gzr)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}

	var lang gotreesitter.Language
	br := bytes.NewReader(raw)
	if err := gob.NewDecoder(br).Decode(&lang); err != nil {
		return nil, fmt.Errorf("decode language blob: %w", err)
	}
	trailer, err := gotreesitter.DecodeLargeStateGotosTrailer(br)
	if err != nil {
		return nil, fmt.Errorf("decode language blob: %w", err)
	}
	if expectsTrailer && len(trailer) == 0 {
		return nil, fmt.Errorf("decode language blob: envelope requires a non-empty large-state-gotos trailer")
	}
	if !expectsTrailer && len(trailer) != 0 {
		return nil, fmt.Errorf("decode language blob: large-state-gotos trailer requires a versioned envelope")
	}
	if trailer != nil {
		lang.LargeStateGotos = trailer
	}
	return &lang, nil
}

func repairGeneratedCompatibilitySymbols(lang *gotreesitter.Language) {
	if lang == nil {
		return
	}
	switch lang.Name {
	case "dart":
		repairGeneratedCollapsedLeafTokenSymbol(lang, "nullable_type", "?")
		repairGeneratedCollapsedLeafTokenSymbol(lang, "null_literal", "null")
	}
}

func repairGeneratedCollapsedLeafTokenSymbol(lang *gotreesitter.Language, parentName, childName string) {
	if !generatedLanguageHasSymbolName(lang, parentName) || generatedLanguageHasSymbolName(lang, childName) {
		return
	}
	for len(lang.SymbolMetadata) < len(lang.SymbolNames) {
		lang.SymbolMetadata = append(lang.SymbolMetadata, gotreesitter.SymbolMetadata{})
	}
	lang.SymbolNames = append(lang.SymbolNames, childName)
	lang.SymbolMetadata = append(lang.SymbolMetadata, gotreesitter.SymbolMetadata{
		Name:    childName,
		Visible: true,
		Named:   false,
	})
	lang.SymbolCount = uint32(len(lang.SymbolNames))
}

func generatedLanguageHasSymbolName(lang *gotreesitter.Language, name string) bool {
	for _, symbolName := range lang.SymbolNames {
		if symbolName == name {
			return true
		}
	}
	return false
}
