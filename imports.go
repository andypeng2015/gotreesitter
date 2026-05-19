package gotreesitter

import (
	"strconv"
	"strings"
)

// ImportRef is a compact language-neutral dependency declaration extracted
// from a syntax tree.
type ImportRef struct {
	Lang      string
	Kind      string
	Path      string
	From      string
	Name      string
	Alias     string
	Static    bool
	Wildcard  bool
	Relative  int
	StartByte uint32
	EndByte   uint32
}

// ExtractImports returns package/import declarations for the languages used by
// Gazelle-style dependency extraction. It is intentionally independent from the
// generic query engine so it can later be backed by compact parser refs.
func ExtractImports(tree *Tree) []ImportRef {
	if tree == nil || tree.RootNode() == nil || tree.Language() == nil {
		return nil
	}
	lang := tree.Language()
	source := tree.Source()
	var refs []ImportRef
	walkImportTree(tree.RootNode(), lang, func(n *Node) bool {
		switch lang.Name {
		case "go":
			return extractGoImportNode(n, lang, source, &refs)
		case "java":
			return extractJavaImportNode(n, lang, source, &refs)
		case "python":
			return extractPythonImportNode(n, lang, source, &refs)
		case "starlark":
			return extractStarlarkImportNode(n, lang, source, &refs)
		default:
			return true
		}
	})
	return refs
}

func walkImportTree(n *Node, lang *Language, visit func(*Node) bool) {
	if n == nil {
		return
	}
	if !visit(n) {
		return
	}
	for _, child := range n.Children() {
		walkImportTree(child, lang, visit)
	}
}

func extractGoImportNode(n *Node, lang *Language, source []byte, refs *[]ImportRef) bool {
	switch n.Type(lang) {
	case "package_clause":
		if name := firstDescendantText(n, lang, source, "package_identifier", "identifier"); name != "" {
			*refs = append(*refs, ImportRef{
				Lang:      lang.Name,
				Kind:      "package",
				Name:      name,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		}
		return false
	case "import_declaration":
		specs := collectDescendantsByType(n, lang, "import_spec")
		if len(specs) == 0 {
			specs = []*Node{n}
		}
		for _, spec := range specs {
			pathNode := firstDescendantByType(spec, lang, "interpreted_string_literal", "raw_string_literal")
			if pathNode == nil {
				continue
			}
			path := importStringLiteralText(pathNode.Text(source))
			if path == "" {
				continue
			}
			ref := ImportRef{
				Lang:      lang.Name,
				Kind:      "import",
				Path:      path,
				Name:      lastDottedName(path),
				Alias:     goImportAlias(spec, pathNode, lang, source),
				StartByte: spec.StartByte(),
				EndByte:   spec.EndByte(),
			}
			*refs = append(*refs, ref)
		}
		return false
	}
	return true
}

func goImportAlias(spec, pathNode *Node, lang *Language, source []byte) string {
	for _, child := range spec.Children() {
		if child == nil || child == pathNode || child.StartByte() >= pathNode.StartByte() {
			break
		}
		switch child.Type(lang) {
		case "package_identifier", "identifier":
			return child.Text(source)
		}
		text := strings.TrimSpace(child.Text(source))
		if text == "." || text == "_" {
			return text
		}
	}
	return ""
}

func extractJavaImportNode(n *Node, lang *Language, source []byte, refs *[]ImportRef) bool {
	switch n.Type(lang) {
	case "package_declaration":
		text := strings.TrimSpace(n.Text(source))
		text = strings.TrimPrefix(text, "package")
		text = strings.TrimSuffix(strings.TrimSpace(text), ";")
		if text != "" {
			*refs = append(*refs, ImportRef{
				Lang:      lang.Name,
				Kind:      "package",
				Path:      text,
				Name:      lastDottedName(text),
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		}
		return false
	case "import_declaration":
		text := strings.TrimSpace(n.Text(source))
		text = strings.TrimPrefix(text, "import")
		text = strings.TrimSuffix(strings.TrimSpace(text), ";")
		ref := ImportRef{
			Lang:      lang.Name,
			Kind:      "import",
			StartByte: n.StartByte(),
			EndByte:   n.EndByte(),
		}
		if strings.HasPrefix(text, "static ") {
			ref.Static = true
			text = strings.TrimSpace(strings.TrimPrefix(text, "static"))
		}
		if strings.HasSuffix(text, ".*") {
			ref.Wildcard = true
			text = strings.TrimSuffix(text, ".*")
		}
		ref.Path = text
		ref.Name = lastDottedName(text)
		*refs = append(*refs, ref)
		return false
	}
	return true
}

func extractPythonImportNode(n *Node, lang *Language, source []byte, refs *[]ImportRef) bool {
	switch n.Type(lang) {
	case "import_statement":
		text := strings.TrimSpace(n.Text(source))
		body := strings.TrimSpace(strings.TrimPrefix(text, "import"))
		for _, part := range splitImportList(body) {
			path, alias := splitImportAlias(part)
			if path == "" {
				continue
			}
			*refs = append(*refs, ImportRef{
				Lang:      lang.Name,
				Kind:      "import",
				Path:      path,
				Name:      lastDottedName(path),
				Alias:     alias,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			})
		}
		return false
	case "import_from_statement":
		text := strings.TrimSpace(n.Text(source))
		body := strings.TrimSpace(strings.TrimPrefix(text, "from"))
		fromPart, importPart, ok := strings.Cut(body, " import ")
		if !ok {
			return false
		}
		relative := countLeadingDots(fromPart)
		from := strings.TrimLeft(fromPart, ".")
		for _, part := range splitImportList(importPart) {
			name, alias := splitImportAlias(part)
			if name == "" {
				continue
			}
			ref := ImportRef{
				Lang:      lang.Name,
				Kind:      "from_import",
				From:      from,
				Name:      name,
				Alias:     alias,
				Relative:  relative,
				StartByte: n.StartByte(),
				EndByte:   n.EndByte(),
			}
			if name == "*" {
				ref.Wildcard = true
				ref.Path = joinPythonImportPath(from, "")
			} else {
				ref.Path = joinPythonImportPath(from, name)
			}
			*refs = append(*refs, ref)
		}
		return false
	}
	return true
}

func extractStarlarkImportNode(n *Node, lang *Language, source []byte, refs *[]ImportRef) bool {
	if n.Type(lang) != "call" {
		return true
	}
	text := strings.TrimSpace(n.Text(source))
	open := strings.IndexByte(text, '(')
	close := strings.LastIndexByte(text, ')')
	if open <= 0 || close <= open || strings.TrimSpace(text[:open]) != "load" {
		return true
	}
	args := splitImportList(text[open+1 : close])
	if len(args) == 0 {
		return false
	}
	from := importStringLiteralText(args[0])
	if from == "" {
		return false
	}
	for _, arg := range args[1:] {
		nameText := arg
		alias := ""
		if left, right, ok := strings.Cut(arg, "="); ok {
			alias = strings.TrimSpace(left)
			nameText = right
		}
		name := importStringLiteralText(nameText)
		if name == "" {
			continue
		}
		*refs = append(*refs, ImportRef{
			Lang:      lang.Name,
			Kind:      "load",
			Path:      from + ":" + name,
			From:      from,
			Name:      name,
			Alias:     alias,
			StartByte: n.StartByte(),
			EndByte:   n.EndByte(),
		})
	}
	return false
}

func splitImportList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "()")
	s = strings.ReplaceAll(s, "\\\n", "")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	s = strings.Join(lines, " ")
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitImportAlias(s string) (path string, alias string) {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 {
		return "", ""
	}
	if len(fields) >= 3 && fields[len(fields)-2] == "as" {
		return strings.Join(fields[:len(fields)-2], " "), fields[len(fields)-1]
	}
	return strings.Join(fields, " "), ""
}

func countLeadingDots(s string) int {
	count := 0
	for count < len(s) && s[count] == '.' {
		count++
	}
	return count
}

func joinPythonImportPath(from, name string) string {
	if from == "" {
		return name
	}
	if name == "" {
		return from
	}
	return from + "." + name
}

func importStringLiteralText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		return unquoted
	}
	return strings.Trim(text, "`\"'")
}

func firstDescendantText(n *Node, lang *Language, source []byte, types ...string) string {
	if child := firstDescendantByType(n, lang, types...); child != nil {
		return child.Text(source)
	}
	return ""
}

func firstDescendantByType(n *Node, lang *Language, types ...string) *Node {
	if n == nil {
		return nil
	}
	typ := n.Type(lang)
	for _, want := range types {
		if typ == want {
			return n
		}
	}
	for _, child := range n.Children() {
		if found := firstDescendantByType(child, lang, types...); found != nil {
			return found
		}
	}
	return nil
}

func collectDescendantsByType(n *Node, lang *Language, typ string) []*Node {
	var out []*Node
	var walk func(*Node)
	walk = func(cur *Node) {
		if cur == nil {
			return
		}
		if cur.Type(lang) == typ {
			out = append(out, cur)
		}
		for _, child := range cur.Children() {
			walk(child)
		}
	}
	walk(n)
	return out
}

func lastDottedName(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, ".*")
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx+1 < len(path) {
		path = path[idx+1:]
	}
	if idx := strings.LastIndex(path, "."); idx >= 0 && idx+1 < len(path) {
		return path[idx+1:]
	}
	return path
}
