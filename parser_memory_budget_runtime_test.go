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
