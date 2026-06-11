package gotreesitter

import "bytes"

func normalizeDoxygenCompatibility(root *Node, source []byte, lang *Language) {
	normalizeDoxygenWholeBlockCommentError(root, source, lang)
}

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
	replaceNodeChildrenUnfielded(root, nil)
}
