package gotreesitter

import "testing"

func TestParseEagerDefaultReduceDefaultOff(t *testing.T) {
	t.Setenv("GOT_EAGER_DEFAULT_REDUCE", "")
	if parseEagerDefaultReduceEnabled() {
		t.Fatal("parseEagerDefaultReduceEnabled() = true with empty env, want false")
	}
}

func TestParseEagerDefaultReduceExplicitOptIn(t *testing.T) {
	t.Setenv("GOT_EAGER_DEFAULT_REDUCE", "1")
	if !parseEagerDefaultReduceEnabled() {
		t.Fatal("parseEagerDefaultReduceEnabled() = false with env=1, want true")
	}
}
