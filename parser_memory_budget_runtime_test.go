package gotreesitter

import "testing"

func TestRuntimeMemoryBudgetStopReasonSamplesHeapGrowth(t *testing.T) {
	parser := &Parser{
		parseRuntimeMemoryBudgetBytes:   1,
		parseRuntimeMemoryBaselineBytes: 0,
		parseRuntimeMemoryPoll:          parseRuntimeMemoryPollMask,
	}
	if got := parser.runtimeMemoryBudgetStopReason(); got != ParseStopMemoryBudget {
		t.Fatalf("runtimeMemoryBudgetStopReason() = %q, want %q", got, ParseStopMemoryBudget)
	}
}

func TestRuntimeMemoryBudgetDisabledForSmallSource(t *testing.T) {
	parser := &Parser{}
	restore := parser.enterRuntimeMemoryBudget(1, parseRuntimeMemoryMinSourceBytes-1)
	if restore.parser != nil {
		t.Fatal("enterRuntimeMemoryBudget enabled for small source")
	}
	if parser.parseRuntimeMemoryBudgetBytes != 0 {
		t.Fatalf("parseRuntimeMemoryBudgetBytes = %d, want 0", parser.parseRuntimeMemoryBudgetBytes)
	}
}

func TestRuntimeMemoryBudgetEnabledForLargeSource(t *testing.T) {
	parser := &Parser{}
	restore := parser.enterRuntimeMemoryBudget(1, parseRuntimeMemoryMinSourceBytes)
	if restore.parser != parser {
		t.Fatal("enterRuntimeMemoryBudget did not return parser restore state")
	}
	defer restore.restore()
	if parser.parseRuntimeMemoryBudgetBytes != 1 {
		t.Fatalf("parseRuntimeMemoryBudgetBytes = %d, want 1", parser.parseRuntimeMemoryBudgetBytes)
	}
}
