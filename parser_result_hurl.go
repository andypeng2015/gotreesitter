package gotreesitter

func normalizeHurlCompatibility(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "hurl" || root.Type(lang) != "ERROR" {
		return
	}
	if !hurlRootHasTrailingFileDelimiterError(root, lang) {
		return
	}
	sym, ok := symbolByName(lang, "hurl_file")
	if !ok {
		return
	}
	retagResultRootAndRefreshError(root, sym, symbolIsNamed(lang, sym))
}

func hurlRootHasTrailingFileDelimiterError(root *Node, lang *Language) bool {
	if root == nil || lang == nil || resultChildCount(root) != 2 {
		return false
	}
	if entry := resultChildAt(root, 0); entry == nil || entry.Type(lang) != "entry" {
		return false
	}
	errNode := resultChildAt(root, 1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || resultChildCount(errNode) != 1 {
		return false
	}
	child := resultChildAt(errNode, 0)
	return child != nil && child.Type(lang) == "file,"
}
