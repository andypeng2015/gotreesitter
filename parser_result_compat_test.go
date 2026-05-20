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
		if got := resultCompatibilityStrutIDForLanguage(rule.languageName); got != rule.strut {
			t.Fatalf("resultCompatibilityStrutIDForLanguage(%q) = %d, want %d", rule.languageName, got, rule.strut)
		}
	}
	if got := resultCompatibilityStrutIDForLanguage("definitely_not_a_language"); got != resultCompatibilityStrutNone {
		t.Fatalf("resultCompatibilityStrutIDForLanguage returned %d for unknown language", got)
	}
}
