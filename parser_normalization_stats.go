package gotreesitter

type normalizationPassCounters struct {
	nodesVisited   uint64
	nodesRewritten uint64
}

type normalizationStats struct {
	passesChecked  uint64
	passesRun      uint64
	nodesVisited   uint64
	nodesRewritten uint64
	nanos          int64
}

func (p *Parser) resetNormalizationStats() {
	if p == nil {
		return
	}
	p.normalizationStats = normalizationStats{}
}

func (p *Parser) runNormalizationPass(enabled func() bool, fn func() normalizationPassCounters) {
	if enabled == nil {
		return
	}
	if p != nil {
		p.normalizationStats.passesChecked++
	}
	if p == nil {
		if enabled() {
			fn()
		}
		return
	}
	run := enabled()
	var counters normalizationPassCounters
	if run {
		p.normalizationStats.passesRun++
		counters = fn()
	}
	p.normalizationStats.nodesVisited += counters.nodesVisited
	p.normalizationStats.nodesRewritten += counters.nodesRewritten
}

func (p *Parser) copyNormalizationStats(rt *ParseRuntime) {
	if p == nil || rt == nil {
		return
	}
	rt.NormalizationPassesChecked = p.normalizationStats.passesChecked
	rt.NormalizationPassesRun = p.normalizationStats.passesRun
	rt.NormalizationNodesVisited = p.normalizationStats.nodesVisited
	rt.NormalizationNodesRewritten = p.normalizationStats.nodesRewritten
	rt.NormalizationNanos = p.normalizationStats.nanos
}
