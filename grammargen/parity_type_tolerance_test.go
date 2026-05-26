package grammargen

import "testing"

func TestIsEquivalentListType(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"expression_list", "pattern_list", true},
		{"list_splat", "list_splat_pattern", true},
		{"dictionary_splat", "dictionary_splat_pattern", true},
		{"list_splat", "dictionary_splat_pattern", false},
		{"identifier", "pattern_list", false},
	}

	for _, tc := range tests {
		if got := isEquivalentListType(tc.a, tc.b); got != tc.want {
			t.Fatalf("isEquivalentListType(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := isEquivalentListType(tc.b, tc.a); got != tc.want {
			t.Fatalf("isEquivalentListType(%q, %q) = %v, want %v", tc.b, tc.a, got, tc.want)
		}
	}
}
