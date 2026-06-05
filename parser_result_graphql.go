package gotreesitter

func normalizeGraphQLCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "graphql" || root.Type(lang) != "source_file" {
		return
	}
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "operation_type", "query", "mutation", "subscription")
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "boolean_value", "true", "false")
}
