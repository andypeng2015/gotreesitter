package gotreesitter

import "bytes"

func normalizeDoxygenCompatibility(root *Node, source []byte, lang *Language) {
	normalizeDoxygenWholeBlockCommentError(root, source, lang)
}

// doxygenStructuralDelimiterNodeTypes are the node types that can legitimately
// appear in a doxygen comment's ERROR wrapper even when the GLR recovery
// mechanism found no real grammar structure to recover: the comment open/close
// delimiters (and ERROR itself). Anything else — tag, tag_name, identifier,
// description, brief_header, code_block, emphasis, etc. — is evidence that the
// recovery pass actually reconstructed real doxygen structure inside the error
// span, which normalizeDoxygenWholeBlockCommentError must not discard.
var doxygenStructuralDelimiterNodeTypes = map[string]bool{
	"ERROR":             true,
	"_multiline_begin":  true,
	"_multiline_end":    true,
	"_singleline_begin": true,
}

// normalizeDoxygenWholeBlockCommentError mirrors C tree-sitter's shape for
// doxygen comments that produce a single, whole-input ERROR node with no
// recoverable structure (e.g. "/** Adds all words in \a s ... */", which C
// itself parses to a bare `(ERROR)`): it collapses gotreesitter's ERROR node
// down to a childless leaf so downstream consumers see the same shape.
//
// This must NOT fire when the GLR recovery mechanism actually reconstructed
// real doxygen grammar structure (tags, descriptions, etc.) inside the error
// span — that happens for inputs the underlying grammar can parse (verified
// against the upstream tree-sitter-doxygen CLI), where only a narrow lexer
// hiccup around the comment's opening delimiter got wrapped in ERROR before
// recovery kicked in and correctly recognized the rest. Discarding that
// structure would silently zero out highlighting/tags/query results for
// otherwise-parseable doxygen comments.
func normalizeDoxygenWholeBlockCommentError(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "doxygen" || len(source) == 0 || root.Type(lang) != "ERROR" {
		return
	}
	if root.startByte != 0 || root.endByte != uint32(len(source)) || resultChildCount(root) == 0 {
		return
	}
	trimmed := bytes.TrimSpace(source)
	if !(bytes.HasPrefix(trimmed, []byte("/**")) || bytes.HasPrefix(trimmed, []byte("/*!"))) || !bytes.HasSuffix(trimmed, []byte("*/")) {
		return
	}
	if doxygenErrorTreeHasRecoveredStructure(root, lang) {
		return
	}
	replaceNodeChildrenUnfielded(root, nil)
}

// doxygenErrorTreeHasRecoveredStructure reports whether any named descendant
// of root is something other than a bare ERROR wrapper or comment delimiter
// token — i.e. whether GLR recovery reconstructed real doxygen structure.
func doxygenErrorTreeHasRecoveredStructure(root *Node, lang *Language) bool {
	found := false
	walkResultTree(root, func(n *Node) {
		if found || n == nil || !n.IsNamed() {
			return
		}
		if !doxygenStructuralDelimiterNodeTypes[n.Type(lang)] {
			found = true
		}
	})
	return found
}
