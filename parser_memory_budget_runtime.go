package gotreesitter

import "runtime"

const (
	parseRuntimeMemoryPollMask       = 15
	parseRuntimeMemoryMinSourceBytes = 64 * 1024
)

type runtimeMemoryBudgetRestore struct {
	parser      *Parser
	budget      int64
	baseline    uint64
	baselineSys uint64
	poll        uint64
}

func runtimeMemoryBudgetEnabled(p *Parser, bytes int64, sourceLen int) bool {
	return p != nil && bytes > 0 && sourceLen >= parseRuntimeMemoryMinSourceBytes
}

func (p *Parser) enterRuntimeMemoryBudget(bytes int64, sourceLen int) runtimeMemoryBudgetRestore {
	if !runtimeMemoryBudgetEnabled(p, bytes, sourceLen) {
		return runtimeMemoryBudgetRestore{}
	}
	restore := runtimeMemoryBudgetRestore{
		parser:      p,
		budget:      p.parseRuntimeMemoryBudgetBytes,
		baseline:    p.parseRuntimeMemoryBaselineBytes,
		baselineSys: p.parseRuntimeMemoryBaselineSys,
		poll:        p.parseRuntimeMemoryPoll,
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	p.parseRuntimeMemoryBudgetBytes = bytes
	p.parseRuntimeMemoryBaselineBytes = stats.HeapAlloc
	p.parseRuntimeMemoryBaselineSys = stats.Sys
	p.parseRuntimeMemoryPoll = 0

	return restore
}

func (r runtimeMemoryBudgetRestore) restore() {
	if r.parser == nil {
		return
	}
	r.parser.parseRuntimeMemoryBudgetBytes = r.budget
	r.parser.parseRuntimeMemoryBaselineBytes = r.baseline
	r.parser.parseRuntimeMemoryBaselineSys = r.baselineSys
	r.parser.parseRuntimeMemoryPoll = r.poll
}

func (p *Parser) runtimeMemoryBudgetStopReason() ParseStopReason {
	if p == nil || p.parseRuntimeMemoryBudgetBytes <= 0 {
		return ParseStopNone
	}
	p.parseRuntimeMemoryPoll++
	if p.parseRuntimeMemoryPoll&parseRuntimeMemoryPollMask != 0 {
		return ParseStopNone
	}
	return p.runtimeMemoryBudgetStopReasonNow()
}

func (p *Parser) runtimeMemoryBudgetStopReasonNow() ParseStopReason {
	if p == nil || p.parseRuntimeMemoryBudgetBytes <= 0 {
		return ParseStopNone
	}
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	if stats.HeapAlloc <= p.parseRuntimeMemoryBaselineBytes {
		if stats.Sys <= p.parseRuntimeMemoryBaselineSys {
			return ParseStopNone
		}
	} else if int64(stats.HeapAlloc-p.parseRuntimeMemoryBaselineBytes) >= p.parseRuntimeMemoryBudgetBytes {
		return ParseStopMemoryBudget
	}
	if stats.Sys > p.parseRuntimeMemoryBaselineSys &&
		int64(stats.Sys-p.parseRuntimeMemoryBaselineSys) >= p.parseRuntimeMemoryBudgetBytes {
		return ParseStopMemoryBudget
	}
	return ParseStopNone
}
