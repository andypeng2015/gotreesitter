package gotreesitter

func normalizeHCLConfigFileRoot(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "hcl" || root.Type(lang) != "config_file" || len(root.children) == 0 {
		return
	}
	filtered := make([]*Node, 0, len(root.children))
	filteredChanged := false
	for _, child := range root.children {
		if child == nil {
			continue
		}
		if child.Type(lang) == "_whitespace" {
			filteredChanged = true
			continue
		}
		filtered = append(filtered, child)
	}
	if filteredChanged {
		if root.ownerArena != nil {
			buf := root.ownerArena.allocNodeSlice(len(filtered))
			copy(buf, filtered)
			filtered = buf
		}
		root.children = filtered
		root.fieldIDs = nil
		root.fieldSources = nil
	}
	for _, child := range root.children {
		if child == nil || child.Type(lang) != "body" {
			continue
		}
		snapHCLBodyBounds(child)
	}
}

func snapHCLBodyBounds(body *Node) {
	if body == nil || len(body.children) == 0 {
		return
	}
	first, last := firstAndLastNonNilChild(body.children)
	if first == nil || last == nil {
		return
	}
	body.startByte = first.startByte
	body.startPoint = first.startPoint
	body.endByte = last.endByte
	body.endPoint = last.endPoint
}
