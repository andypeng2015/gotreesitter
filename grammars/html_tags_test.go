//go:build !grammar_subset || grammar_subset_angular || grammar_subset_astro || grammar_subset_blade || grammar_subset_html || grammar_subset_svelte || grammar_subset_vue

package grammars

import "testing"

func TestHTMLDeserializeTagsIntoReusesBackingArray(t *testing.T) {
	tags := []htmlTag{
		{tagType: htmlTagDiv},
		{tagType: htmlTagCustom, customName: "CUSTOM-ELEMENT"},
	}
	buf := make([]byte, 64)
	n := htmlSerializeTags(tags, buf)
	if n == 0 {
		t.Fatal("htmlSerializeTags returned empty snapshot")
	}

	dst := make([]htmlTag, 1, 4)
	backing := &dst[:cap(dst)][0]
	got := htmlDeserializeTagsInto(dst, buf[:n])
	if len(got) != len(tags) {
		t.Fatalf("htmlDeserializeTagsInto len = %d, want %d", len(got), len(tags))
	}
	if &got[:cap(got)][0] != backing {
		t.Fatal("htmlDeserializeTagsInto did not reuse destination backing array")
	}
	for i := range tags {
		if !htmlTagEq(&got[i], &tags[i]) {
			t.Fatalf("tag %d = %#v, want %#v", i, got[i], tags[i])
		}
	}
}
