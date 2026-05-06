package gotreesitter

import "testing"

func TestResultCompatibilityStrutInventoryResolves(t *testing.T) {
	seen := make(map[string]struct{}, len(resultCompatibilityStrutLanguageNames))
	for _, name := range resultCompatibilityStrutLanguageNames {
		if _, ok := seen[name]; ok {
			t.Fatalf("duplicate result compatibility strut language %q", name)
		}
		seen[name] = struct{}{}
		if got := resultCompatibilityStrutForLanguage(name); got == nil {
			t.Fatalf("resultCompatibilityStrutForLanguage(%q) = nil, want strut", name)
		}
	}
	if got := resultCompatibilityStrutForLanguage("definitely_not_a_language"); got != nil {
		t.Fatal("resultCompatibilityStrutForLanguage returned a strut for unknown language")
	}
}
