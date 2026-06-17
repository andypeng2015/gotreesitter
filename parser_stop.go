package gotreesitter

func (p *Parser) parseStopReasonNow() ParseStopReason {
	return p.activeParseStopReason()
}

func parseStopReasonIsTerminal(reason ParseStopReason) bool {
	return parseStopReasonIsActive(reason)
}
