package gotreesitter

import "testing"

func TestResultCompatibilityStrutInventoryResolves(t *testing.T) {
	seen := make(map[string]struct{}, len(resultCompatibilityStrutRules))
	for _, rule := range resultCompatibilityStrutRules {
		if _, ok := seen[rule.languageName]; ok {
			t.Fatalf("duplicate result compatibility strut language %q", rule.languageName)
		}
		seen[rule.languageName] = struct{}{}
		if rule.strut == resultCompatibilityStrutNone {
			t.Fatalf("result compatibility strut rule for %q has no strut", rule.languageName)
		}
		if got := resultCompatibilityStrutForLanguage(rule.languageName); got == nil {
			t.Fatalf("resultCompatibilityStrutForLanguage(%q) = nil, want strut", rule.languageName)
		}
	}
	if got := resultCompatibilityStrutForLanguage("definitely_not_a_language"); got != nil {
		t.Fatal("resultCompatibilityStrutForLanguage returned a strut for unknown language")
	}
}
