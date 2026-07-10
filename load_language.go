package gotreesitter

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
)

// LoadLanguage deserializes a compressed grammar blob into a Language.
// Blobs are produced by EncodeLanguageBlob, grammargen.Generate, or the
// grammar build toolchain. This is the only function needed at runtime to load
// pre-compiled grammars — no grammargen import required. It accepts legacy
// gzip blobs and version-enveloped trailer-bearing blobs.
func LoadLanguage(data []byte) (*Language, error) {
	compressed, expectsTrailer, err := UnwrapLanguageBlobEnvelope(data)
	if err != nil {
		return nil, fmt.Errorf("decode language: %w", err)
	}
	gzr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gzr.Close()

	// Pre-size the decompression buffer using the ISIZE field in the last 4
	// bytes of the gzip trailer. This avoids io.ReadAll's repeated doublings.
	// ISIZE is uncompressed size mod 2^32; for grammar blobs (well under 4 GB)
	// it is exact. Fall back to io.ReadAll if the hint is implausible.
	var raw []byte
	if len(compressed) >= 4 {
		isize := binary.LittleEndian.Uint32(compressed[len(compressed)-4:])
		if isize > 0 && isize < 256*1024*1024 { // sanity cap at 256 MB
			raw = make([]byte, 0, isize)
			var buf [32 * 1024]byte
			for {
				n, readErr := gzr.Read(buf[:])
				if n > 0 {
					raw = append(raw, buf[:n]...)
				}
				if readErr == io.EOF {
					break
				}
				if readErr != nil {
					return nil, fmt.Errorf("read gzip: %w", readErr)
				}
			}
		}
	}
	if raw == nil {
		raw, err = io.ReadAll(gzr)
		if err != nil {
			return nil, fmt.Errorf("read gzip: %w", err)
		}
	}

	var lang Language
	br := bytes.NewReader(raw)
	if err := gob.NewDecoder(br).Decode(&lang); err != nil {
		return nil, fmt.Errorf("decode language: %w", err)
	}
	// LargeStateGotos (when non-empty) is never gob-encoded directly -- see
	// large_state_gotos_trailer.go for why -- so restore it from the trailer
	// appended after the gob message, if the encoder wrote one. This is a
	// cheap no-op (returns immediately) for every blob without a trailer,
	// including all blobs encoded before this trailer mechanism existed.
	trailer, err := DecodeLargeStateGotosTrailer(br)
	if err != nil {
		return nil, fmt.Errorf("decode language: %w", err)
	}
	if expectsTrailer && len(trailer) == 0 {
		return nil, fmt.Errorf("decode language: envelope requires a non-empty large-state-gotos trailer")
	}
	if !expectsTrailer && len(trailer) != 0 {
		return nil, fmt.Errorf("decode language: large-state-gotos trailer requires a versioned envelope")
	}
	if trailer != nil {
		lang.LargeStateGotos = trailer
	}

	InferGeneratedRepeatAuxMetadata(&lang)

	return &lang, nil
}
