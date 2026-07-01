package gotreesitter

func normalizeHackCompatibility(root *Node, source []byte, lang *Language) {
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "true", "true")
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "false", "false")
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "null", "null")
}
