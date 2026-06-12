package gotreesitter

func normalizeRescriptCompatibility(root *Node, lang *Language) {
	normalizeRescriptValueIdentifierPaths(root, lang)
}

func normalizeRescriptValueIdentifierPaths(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "rescript" || root.HasError() {
		return
	}
	valuePathSym, valuePathNamed, ok := symbolMeta(lang, "value_identifier_path")
	if !ok {
		return
	}
	moduleIdentSym, moduleIdentNamed, ok := symbolMeta(lang, "module_identifier")
	if !ok {
		return
	}
	valueIdentSym, valueIdentNamed, ok := symbolMeta(lang, "value_identifier")
	if !ok {
		return
	}

	walkResultTree(root, func(n *Node) {
		if n == nil || n.Type(lang) != "member_expression" || n.HasError() {
			return
		}
		if resultChildCount(n) != 3 {
			return
		}
		object := resultChildAt(n, 0)
		dot := resultChildAt(n, 1)
		property := resultChildAt(n, 2)
		if object == nil || dot == nil || property == nil {
			return
		}
		if object.Type(lang) != "variant" || dot.Type(lang) != "." || property.Type(lang) != "property_identifier" {
			return
		}
		if object.startByte >= object.endByte || property.startByte >= property.endByte {
			return
		}

		n.symbol = valuePathSym
		n.setNamed(valuePathNamed)
		object.symbol = moduleIdentSym
		object.setNamed(moduleIdentNamed)
		replaceNodeChildrenUnfielded(object, nil)
		property.symbol = valueIdentSym
		property.setNamed(valueIdentNamed)
		replaceNodeChildrenUnfielded(property, nil)
	})
}
