package gotreesitter

import "runtime"

const parseRuntimeMemoryPollMask = 15

func (p *Parser) enterRuntimeMemoryBudget(bytes int64) func() {
	if p == nil || bytes <= 0 {
		return func() {}
	}
	prevBudget := p.parseRuntimeMemoryBudgetBytes
	prevBaseline := p.parseRuntimeMemoryBaselineBytes
	prevBaselineSys := p.parseRuntimeMemoryBaselineSys
	prevPoll := p.parseRuntimeMemoryPoll

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	p.parseRuntimeMemoryBudgetBytes = bytes
	p.parseRuntimeMemoryBaselineBytes = stats.HeapAlloc
	p.parseRuntimeMemoryBaselineSys = stats.Sys
	p.parseRuntimeMemoryPoll = 0

	return func() {
		p.parseRuntimeMemoryBudgetBytes = prevBudget
		p.parseRuntimeMemoryBaselineBytes = prevBaseline
		p.parseRuntimeMemoryBaselineSys = prevBaselineSys
		p.parseRuntimeMemoryPoll = prevPoll
	}
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
