package gotreesitter

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// parser_recover_c.go is the stage-1 faithful port of tree-sitter C's error
// recovery loop into the pure-Go GLR engine, gated per grammar via parser.c
// capability metadata, explicit default certification, and conservative
// runtime capability validation.
//
// THE C CODE IS THE SPEC (tree-sitter v0.25 lib/src):
//   - parser.c  ts_parser__handle_error / ts_parser__recover /
//     ts_parser__do_all_potential_reductions / ts_parser__recover_to_state /
//     ts_parser__compare_versions / ts_parser__version_status /
//     ts_parser__better_version_exists / ts_parser__condense_stack
//   - stack.c   pause/resume/summary/error-cost bookkeeping
//   - subtree.c ts_subtree_error_cost / summarize_children
//   - error_costs.h cost constants
//
// Mapping notes (Go GLR engine vs C stack versions):
//   - C keeps one Stack with multi-link versions; this engine keeps separate
//     glrStack copies. C's handle_error merges all do_all_potential_reductions
//     versions into ONE multi-path version before recording the stack summary;
//     here each forked version becomes its own absorbing stack with its own
//     summary. The union of per-stack summaries covers the same recovery
//     candidates; the C cost competition (ported below) prunes the set.
//   - C marks an erroring version "paused", lets other versions advance, and
//     resumes the best paused version in condense_stack. This engine processes
//     stacks in lockstep per token, so handle_error runs immediately at the
//     no-action point; cCondenseStacks applies the same cost competition after
//     each dispatch pass.
//   - C's ERROR_STATE (state 0) is real in the generated Go tables too (the
//     recover row). An absorbing stack pushes a node-less stackEntry{state: 0}
//     as the C "NULL subtree discontinuity"; with state 0 on top, the DFA token
//     source lexes with LexModes[0] — exactly C's error-mode lexing.
//   - C's error_repeat chain is flattened: the open error region is a single
//     ERROR node whose children are the absorbed tokens.
//   - C memoizes error cost per subtree; stage 1 recomputes it by walking the
//     stack (gated grammars parse small files). Stage 2 should make it
//     incremental if wider grammars need it.

// C error_costs.h. NOTE: ERROR_COST_PER_SKIPPED_LINE is 30 in the C header
// (the recovery-cost-competition.md table said 2; the header wins).
const (
	cErrCostPerRecovery    = 500
	cErrCostPerMissingTree = 110
	cErrCostPerSkippedTree = 100
	cErrCostPerSkippedLine = 30
	cErrCostPerSkippedChar = 1
)

const (
	// C parser.c MAX_VERSION_COUNT / MAX_SUMMARY_DEPTH / MAX_COST_DIFFERENCE.
	cRecoverMaxVersionCount     = 6
	cRecoverMaxSummaryDepth     = 16
	cRecoverMaxCostDifference   = 18 * cErrCostPerSkippedTree
	cRecoverMaxReduceIterations = 1024
	// cErrorState is the C ERROR_STATE: the generated tables' recover row.
	cErrorState = StateID(0)
)

// errorCostCompetitionLanguage reports whether the faithful C error-recovery
// port is enabled for the active grammar. By default the gate requires
// parser.c-backed capability metadata, explicit parity certification, and
// conservative runtime table validation.
// GOT_C_RECOVERY=0 force-disables the gate (baseline A/B measurement);
// GOT_C_RECOVERY=all/1 or a comma-separated grammar list force-enables it for
// diagnostic sweeps when runtime table validation passes.
func errorCostCompetitionLanguage(lang *Language) bool {
	if lang == nil {
		return false
	}
	switch v := os.Getenv("GOT_C_RECOVERY"); v {
	case "":
	case "0":
		return false
	case "all", "1":
		return languageSupportsCRecoveryCostCompetition(lang)
	default:
		forced := false
		for _, n := range strings.Split(v, ",") {
			if strings.TrimSpace(n) == lang.Name {
				forced = true
				break
			}
		}
		if !forced {
			return false
		}
		return languageSupportsCRecoveryCostCompetition(lang)
	}
	if !lang.CRecoveryCostCompetitionCapable || !lang.CRecoveryCostCompetitionEnabledByDefault {
		return false
	}
	return languageSupportsCRecoveryCostCompetition(lang)
}

func cRecoveryGateReasonSlug(reason string) string {
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" {
		return ""
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range reason {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := b.String()
	return strings.Trim(out, "_")
}

func cRecoveryGateReason(lang *Language) string {
	if os.Getenv("GOT_C_RECOVERY") == "0" {
		return "disabled_by_got_c_recovery_0"
	}
	diag := DiagnoseCRecoveryGate(lang)
	if !diag.Supported {
		return cRecoveryGateReasonSlug(diag.Reason)
	}
	switch v := os.Getenv("GOT_C_RECOVERY"); v {
	case "all", "1":
		return ""
	case "":
		if lang == nil {
			return "nil_language"
		}
		if !lang.CRecoveryCostCompetitionCapable {
			return "not_c_recovery_capable"
		}
		if !lang.CRecoveryCostCompetitionEnabledByDefault {
			return "not_enabled_by_default"
		}
		return ""
	default:
		if lang != nil {
			for _, n := range strings.Split(v, ",") {
				if strings.TrimSpace(n) == lang.Name {
					return ""
				}
			}
		}
		return "not_enabled_by_got_c_recovery"
	}
}

func (p *Parser) cRecoveryGateReason() string {
	if p == nil {
		return "nil_parser"
	}
	if p.errorCostCompetition {
		if p.noTreeBenchmarkOnly {
			return "disabled_for_no_tree_benchmark"
		}
		if p.noTreeCheckpointBenchmarkOnly {
			return "disabled_for_no_tree_checkpoint_benchmark"
		}
		return ""
	}
	return cRecoveryGateReason(p.language)
}

// CRecoveryGateDiagnostics describes the runtime validation result for the
// C recovery-cost competition gate. Reason is empty when Supported is true;
// otherwise it names the first failed validation check.
type CRecoveryGateDiagnostics struct {
	Supported bool
	Reason    string

	StateCount       int
	SymbolCount      int
	TokenCount       int
	LexModeCount     int
	LexStateCount    int
	ParseActionCount int

	HasExternalScanner     bool
	ExternalSymbolCount    int
	ExternalTokenCount     int
	ExternalLexStateRows   int
	ExternalLexStateMinLen int
}

// DiagnoseCRecoveryGate validates the runtime table surface required by the
// faithful C recovery-cost competition path and returns the first failure.
func DiagnoseCRecoveryGate(lang *Language) CRecoveryGateDiagnostics {
	if lang == nil {
		return CRecoveryGateDiagnostics{Reason: "nil language"}
	}
	d := CRecoveryGateDiagnostics{
		StateCount:             int(lang.StateCount),
		SymbolCount:            int(lang.SymbolCount),
		TokenCount:             int(lang.TokenCount),
		LexModeCount:           len(lang.LexModes),
		LexStateCount:          len(lang.LexStates),
		ParseActionCount:       len(lang.ParseActions),
		HasExternalScanner:     lang.ExternalScanner != nil,
		ExternalSymbolCount:    len(lang.ExternalSymbols),
		ExternalTokenCount:     int(lang.ExternalTokenCount),
		ExternalLexStateRows:   len(lang.ExternalLexStates),
		ExternalLexStateMinLen: externalLexStateMinLen(lang),
	}
	fail := func(reason string) CRecoveryGateDiagnostics {
		d.Reason = reason
		return d
	}
	if lang.InitialState != 1 {
		return fail("initial state is not 1")
	}
	if lang.StateCount == 0 {
		return fail("state count is zero")
	}
	if lang.SymbolCount == 0 {
		return fail("symbol count is zero")
	}
	if lang.TokenCount == 0 {
		return fail("token count is zero")
	}
	if len(lang.SymbolMetadata) < int(lang.SymbolCount) {
		return fail("symbol metadata is shorter than SymbolCount")
	}
	if len(lang.SymbolNames) < int(lang.SymbolCount) {
		return fail("symbol names are shorter than SymbolCount")
	}
	if len(lang.ParseActions) == 0 {
		return fail("parse actions are empty")
	}
	if len(lang.LexModes) == 0 {
		return fail("lex modes are empty")
	}
	if len(lang.LexStates) == 0 {
		return fail("lex states are empty")
	}
	ls := lang.LexModes[0].LexStateIndex()
	if ls == noLookaheadLexState {
		return fail("error-state lex mode has no lookahead lex state")
	}
	if int(ls) >= len(lang.LexStates) {
		return fail("error-state lex mode references missing lex state")
	}
	if int(lang.StateCount) > len(lang.LexModes) {
		return fail("state count exceeds lex mode count")
	}
	if langHasExternalRecoverySurface(lang) {
		if reason := externalLexStatesCRecoveryFailure(lang); reason != "" {
			return fail(reason)
		}
	}
	if reason := parseTablesCRecoveryFailure(lang); reason != "" {
		return fail(reason)
	}
	if reason := parseActionsCRecoveryFailure(lang); reason != "" {
		return fail(reason)
	}
	d.Supported = true
	return d
}

func languageSupportsCRecoveryCostCompetition(lang *Language) bool {
	return DiagnoseCRecoveryGate(lang).Supported
}

// CertifyCRecoveryCostCompetition validates a language's runtime recovery
// surface and updates the C recovery metadata.
//
// Capability means the tables satisfy the runtime gate. Default enablement is
// narrower: grammars with external recovery metadata are enabled only after an
// actual external scanner is attached and precise ExternalLexStates are
// available.
func CertifyCRecoveryCostCompetition(lang *Language) CRecoveryGateDiagnostics {
	diag := DiagnoseCRecoveryGate(lang)
	if lang == nil {
		return diag
	}
	lang.CRecoveryCostCompetitionEnabledByDefault = false
	if !diag.Supported {
		return diag
	}
	lang.CRecoveryCostCompetitionCapable = true
	lang.CRecoveryCostCompetitionEnabledByDefault = generatedCRecoveryDefaultSafe(lang)
	return diag
}

// CertifyGeneratedCRecoveryCostCompetition is kept for existing generated
// grammar call sites. New callers should use CertifyCRecoveryCostCompetition.
func CertifyGeneratedCRecoveryCostCompetition(lang *Language) CRecoveryGateDiagnostics {
	return CertifyCRecoveryCostCompetition(lang)
}

func generatedCRecoveryDefaultSafe(lang *Language) bool {
	if lang == nil {
		return false
	}
	if !langHasExternalRecoverySurface(lang) {
		return true
	}
	return lang.ExternalScanner != nil && externalLexStatesCRecoveryFailure(lang) == ""
}

func languageParseTablesValidForCRecovery(lang *Language) bool {
	return parseTablesCRecoveryFailure(lang) == ""
}

func langHasExternalRecoverySurface(lang *Language) bool {
	return lang != nil && (lang.ExternalScanner != nil || len(lang.ExternalSymbols) > 0 || lang.ExternalTokenCount > 0)
}

func externalLexStateMinLen(lang *Language) int {
	if lang == nil || len(lang.ExternalLexStates) == 0 {
		return 0
	}
	min := len(lang.ExternalLexStates[0])
	for _, row := range lang.ExternalLexStates[1:] {
		if len(row) < min {
			min = len(row)
		}
	}
	return min
}

func externalLexStatesCRecoveryFailure(lang *Language) string {
	if lang == nil {
		return "nil language"
	}
	if len(lang.ExternalSymbols) == 0 {
		return "external scanner surface has no ExternalSymbols"
	}
	if len(lang.ExternalLexStates) == 0 {
		return "external scanner requires precise ExternalLexStates"
	}
	for _, sym := range lang.ExternalSymbols {
		if sym >= Symbol(lang.SymbolCount) {
			return "external symbol is outside SymbolCount"
		}
	}
	for _, row := range lang.ExternalLexStates {
		if len(row) < len(lang.ExternalSymbols) {
			return "ExternalLexStates row is shorter than ExternalSymbols"
		}
	}
	for state := 0; state < int(lang.StateCount) && state < len(lang.LexModes); state++ {
		if int(lang.LexModes[state].ExternalLexState) >= len(lang.ExternalLexStates) {
			return "lex mode references missing ExternalLexStates row"
		}
	}
	return ""
}

func parseTablesCRecoveryFailure(lang *Language) string {
	if lang == nil {
		return "nil language"
	}
	tokenCount := int(lang.TokenCount)
	symbolCount := int(lang.SymbolCount)
	stateCount := int(lang.StateCount)
	if tokenCount <= 0 {
		return "token count is zero"
	}
	if symbolCount <= 0 {
		return "symbol count is zero"
	}
	if stateCount <= 0 {
		return "state count is zero"
	}
	validateValue := func(sym int, val uint16) string {
		if val == 0 {
			return ""
		}
		if sym < 0 || sym >= symbolCount {
			return "parse table symbol is outside SymbolCount"
		}
		if sym < tokenCount {
			if int(val) >= len(lang.ParseActions) {
				return "parse table terminal action index is outside ParseActions"
			}
			return ""
		}
		if int(val) >= stateCount {
			return "parse table goto state is outside StateCount"
		}
		return ""
	}
	for _, row := range lang.ParseTable {
		for sym, val := range row {
			if reason := validateValue(sym, val); reason != "" {
				return reason
			}
		}
	}
	if len(lang.SmallParseTableMap) == 0 {
		return ""
	}
	table := lang.SmallParseTable
	if len(table) == 0 {
		return "small parse table map exists but table is empty"
	}
	for _, offset := range lang.SmallParseTableMap {
		pos := int(offset)
		if pos < 0 || pos >= len(table) {
			return "small parse table offset is outside table"
		}
		groupCount := int(table[pos])
		pos++
		for i := 0; i < groupCount; i++ {
			if pos+1 >= len(table) {
				return "small parse table group header is truncated"
			}
			val := table[pos]
			n := int(table[pos+1])
			pos += 2
			if n < 0 || pos+n > len(table) {
				return "small parse table group symbols are truncated"
			}
			for j := 0; j < n; j++ {
				if reason := validateValue(int(table[pos+j]), val); reason != "" {
					return reason
				}
			}
			pos += n
		}
	}
	return ""
}

func parseActionsCRecoveryFailure(lang *Language) string {
	if lang == nil {
		return "nil language"
	}
	for _, entry := range lang.ParseActions {
		for _, action := range entry.Actions {
			switch action.Type {
			case ParseActionShift, ParseActionRecover:
				if action.State >= StateID(lang.StateCount) {
					return "parse action target state is outside StateCount"
				}
			case ParseActionReduce:
				if action.Symbol >= Symbol(lang.SymbolCount) ||
					int(action.Symbol) >= len(lang.SymbolMetadata) ||
					int(action.Symbol) >= len(lang.SymbolNames) {
					return "reduce action symbol is outside symbol metadata"
				}
			case ParseActionAccept:
			default:
				return "parse action type is unsupported"
			}
		}
	}
	return ""
}

func languageHasPreciseExternalLexStates(lang *Language) bool {
	return externalLexStatesCRecoveryFailure(lang) == ""
}

func (p *Parser) errorCostCompetitionEnabled() bool {
	return p != nil && p.errorCostCompetition && !p.noTreeBenchmarkOnly && !p.noTreeCheckpointBenchmarkOnly
}

// cStackSummaryEntry mirrors C StackSummaryEntry (stack.h): a (depth, state)
// pair with the stack position at that depth, recorded when entering the
// error state and consulted by ts_parser__recover strategy 1.
type cStackSummaryEntry struct {
	depth    int
	state    StateID
	posBytes uint32
	posRow   uint32
}

// cRecGroup coordinates the absorbing stacks that map to C's ONE merged
// error-state version. C ts_parser__handle_error pushes the NULL discontinuity
// onto every do_all_potential_reductions result and merges them into a single
// multi-link version; this engine keeps one linear stack per path, so the
// group makes them act like the single C version: the per-token strategy-1
// summary scan (ts_parser__recover) is performed ONCE across the union of the
// members' summaries, in C's breadth-first record order.
type cRecGroup struct {
	// electionTokenStart/electionTokenSymbol identify the token for which the
	// group's strategy-1 election has already run.
	electionTokenStart  uint32
	electionTokenSymbol Symbol
	electionDone        bool
}

// cRecoverState marks a glrStack as being in the C error state (head at
// ERROR_STATE absorbing skipped tokens). nil == not in error.
type cRecoverState struct {
	summary []cStackSummaryEntry
	// group ties the absorbing stacks that represent the same C version.
	group *cRecGroup
	// groupOrder preserves the path order inside the C merged error-state
	// version. Later Go stack ordering can move members around, but C's
	// strategy-1 summary scan still walks the merged paths in record order.
	groupOrder int
	// openErr is the open error region node on the stack top (the C
	// error_repeat being accumulated). nil right after entering the error
	// state — the C "ERROR_STATE head with NULL subtree" shape, which costs an
	// extra ERROR_COST_PER_RECOVERY in ts_stack_error_cost.
	openErr *Node
}

func (r *cRecoverState) clone() *cRecoverState {
	if r == nil {
		return nil
	}
	cp := &cRecoverState{openErr: r.openErr, group: r.group, groupOrder: r.groupOrder}
	if len(r.summary) > 0 {
		cp.summary = append([]cStackSummaryEntry(nil), r.summary...)
	}
	return cp
}

// cRecoverOutcome describes what the gated recovery did with the current
// stack for the current token.
type cRecoverOutcome int

const (
	// cRecFallthrough: not handled — caller continues normal dispatch.
	cRecFallthrough cRecoverOutcome = iota
	// cRecConsumed: token absorbed into the error region (or recover_eof
	// accepted); the stack is done with this token.
	cRecConsumed
	// cRecHalted: the stack was halted (clearly worse than another version).
	cRecHalted
)

// ---------------------------------------------------------------------------
// Error cost (subtree.c ts_subtree_error_cost / summarize_children port)
// ---------------------------------------------------------------------------

func cSymbolVisibleLang(lang *Language, sym Symbol) bool {
	if sym == errorSymbol {
		return true
	}
	if lang == nil || int(sym) >= len(lang.SymbolMetadata) {
		return false
	}
	return lang.SymbolMetadata[sym].Visible
}

func (p *Parser) cSymbolVisible(sym Symbol) bool {
	if p == nil {
		return false
	}
	return cSymbolVisibleLang(p.language, sym)
}

// cNodeVisibleChildCount mirrors SubtreeHeapData.visible_child_count:
// direct children that are visible, plus the visible children of invisible
// internal children.
func cNodeVisibleChildCountLang(lang *Language, n *Node) int {
	if n == nil {
		return 0
	}
	count := 0
	for _, c := range n.children {
		if c == nil {
			continue
		}
		if cSymbolVisibleLang(lang, c.symbol) {
			count++
		} else if len(c.children) > 0 {
			count += cNodeVisibleChildCountLang(lang, c)
		}
	}
	return count
}

// cNodeVisibleSubtreeCount mirrors stack.c stack__subtree_node_count: the
// visible descendant count (plus the node itself when visible), used for
// node-count-since-error bookkeeping. Memoized per (node, equivVersion) when
// the gate is on — C keeps this in the subtree header too.
func (p *Parser) cNodeVisibleSubtreeCount(n *Node) int {
	if n == nil {
		return 0
	}
	if p != nil && p.cNodeMemo != nil {
		if e, ok := p.cNodeMemo[n]; ok && e.hasVis && e.ver == n.equivVersion {
			return e.visCount
		}
	}
	count := 0
	if p.cSymbolVisible(n.symbol) {
		count++
	}
	for _, c := range n.children {
		count += p.cNodeVisibleSubtreeCount(c)
	}
	if p != nil && p.cNodeMemo != nil {
		e := p.cNodeMemo[n]
		if e.ver != n.equivVersion {
			e = cNodeMemoEntry{ver: n.equivVersion}
		}
		e.visCount = count
		e.hasVis = true
		p.cNodeMemo[n] = e
	}
	return count
}

// cNodeErrorCost ports ts_subtree_error_cost + the error-cost part of
// ts_subtree_summarize_children. Go has no error_repeat chain: ERROR nodes
// hold absorbed children directly, so the per-ERROR recovery cost is charged
// once per ERROR node. Go nodes carry no padding either; an ERROR node's span
// already starts at its first real token, matching the C "size excludes
// padding" rule for the common case.
func cNodeErrorCostLang(lang *Language, n *Node) uint32 {
	if n == nil {
		return 0
	}
	if n.isMissing() && len(n.children) == 0 {
		return cErrCostPerMissingTree + cErrCostPerRecovery
	}
	var cost uint32
	for _, c := range n.children {
		cost += cNodeErrorCostLang(lang, c)
	}
	if n.symbol == errorSymbol {
		for _, c := range n.children {
			if c == nil || c.isExtra() {
				continue
			}
			if c.symbol == errorSymbol && len(c.children) == 0 {
				continue
			}
			if cSymbolVisibleLang(lang, c.symbol) {
				cost += cErrCostPerSkippedTree
			} else if len(c.children) > 0 {
				cost += cErrCostPerSkippedTree * uint32(cNodeVisibleChildCountLang(lang, c))
			}
		}
		bytes := uint32(0)
		rows := uint32(0)
		if n.endByte > n.startByte {
			bytes = n.endByte - n.startByte
		}
		if n.endPoint.Row > n.startPoint.Row {
			rows = n.endPoint.Row - n.startPoint.Row
		}
		cost += cErrCostPerRecovery + cErrCostPerSkippedChar*bytes + cErrCostPerSkippedLine*rows
	}
	return cost
}

func cNodeErrorCostLangWithScratch(scratch *glrMergeScratch, lang *Language, n *Node) uint32 {
	if n == nil {
		return 0
	}
	if scratch == nil {
		return cNodeErrorCostLang(lang, n)
	}
	if cached, ok := scratch.cErrorCost[n]; ok && cached.ver == n.equivVersion {
		return cached.cost
	}
	if scratch.cErrorCost == nil {
		scratch.cErrorCost = make(map[*Node]glrCErrorCostEntry)
	}
	if n.isMissing() && len(n.children) == 0 {
		cost := uint32(cErrCostPerMissingTree + cErrCostPerRecovery)
		scratch.cErrorCost[n] = glrCErrorCostEntry{ver: n.equivVersion, cost: cost}
		return cost
	}
	var cost uint32
	for _, c := range n.children {
		cost += cNodeErrorCostLangWithScratch(scratch, lang, c)
	}
	if n.symbol == errorSymbol {
		for _, c := range n.children {
			if c == nil || c.isExtra() {
				continue
			}
			if c.symbol == errorSymbol && len(c.children) == 0 {
				continue
			}
			if cSymbolVisibleLang(lang, c.symbol) {
				cost += cErrCostPerSkippedTree
			} else if len(c.children) > 0 {
				cost += cErrCostPerSkippedTree * uint32(cNodeVisibleChildCountLang(lang, c))
			}
		}
		bytes := uint32(0)
		rows := uint32(0)
		if n.endByte > n.startByte {
			bytes = n.endByte - n.startByte
		}
		if n.endPoint.Row > n.startPoint.Row {
			rows = n.endPoint.Row - n.startPoint.Row
		}
		cost += cErrCostPerRecovery + cErrCostPerSkippedChar*bytes + cErrCostPerSkippedLine*rows
	}
	scratch.cErrorCost[n] = glrCErrorCostEntry{ver: n.equivVersion, cost: cost}
	return cost
}

// cNodeMemoEntry caches the gated recovery's per-subtree aggregates, the
// engine analogue of C SubtreeHeapData.error_cost / node counts computed once
// per subtree in ts_subtree_summarize_children. Finished subtrees never
// mutate during a parse; the open error region is bumped (equivVersion) on
// every absorb, invalidating its entry. Without the memo every
// condense/competition step rewalks whole accumulated subtrees per token,
// which is O(n^2) on large gated files.
type cNodeMemoEntry struct {
	ver      uint32
	cost     uint32
	visCount int
	hasCost  bool
	hasVis   bool
}

func (p *Parser) cNodeErrorCost(n *Node) uint32 {
	if p == nil {
		return 0
	}
	if n == nil {
		return 0
	}
	if p.cNodeMemo == nil {
		return cNodeErrorCostLang(p.language, n)
	}
	if e, ok := p.cNodeMemo[n]; ok && e.hasCost && e.ver == n.equivVersion {
		return e.cost
	}
	if n.isMissing() && len(n.children) == 0 {
		return cErrCostPerMissingTree + cErrCostPerRecovery
	}
	var cost uint32
	for _, c := range n.children {
		cost += p.cNodeErrorCost(c)
	}
	if n.symbol == errorSymbol {
		lang := p.language
		for _, c := range n.children {
			if c == nil || c.isExtra() {
				continue
			}
			if c.symbol == errorSymbol && len(c.children) == 0 {
				continue
			}
			if cSymbolVisibleLang(lang, c.symbol) {
				cost += cErrCostPerSkippedTree
			} else if len(c.children) > 0 {
				cost += cErrCostPerSkippedTree * uint32(cNodeVisibleChildCountLang(lang, c))
			}
		}
		bytes := uint32(0)
		rows := uint32(0)
		if n.endByte > n.startByte {
			bytes = n.endByte - n.startByte
		}
		if n.endPoint.Row > n.startPoint.Row {
			rows = n.endPoint.Row - n.startPoint.Row
		}
		cost += cErrCostPerRecovery + cErrCostPerSkippedChar*bytes + cErrCostPerSkippedLine*rows
	}
	e := p.cNodeMemo[n]
	if e.ver != n.equivVersion {
		e = cNodeMemoEntry{ver: n.equivVersion}
	}
	e.cost = cost
	e.hasCost = true
	p.cNodeMemo[n] = e
	return cost
}

// cStackErrorCost ports ts_stack_error_cost: the accumulated error cost of
// every subtree on the stack, plus one open recovery when the version just
// entered the error state and has not absorbed anything yet (the C
// "ERROR_STATE head with NULL subtree" case).
func (p *Parser) cStackErrorCost(s *glrStack) uint32 {
	if s == nil {
		return 0
	}
	var cost uint32
	walk := func(n *Node) {
		if n != nil {
			cost += p.cNodeErrorCost(n)
		}
	}
	if len(s.entries) > 0 {
		for i := range s.entries {
			walk(stackEntryNode(s.entries[i]))
		}
	} else {
		for gn := s.gss.head; gn != nil; gn = gn.prev {
			walk(stackEntryNode(gn.entry))
		}
	}
	if s.cPaused || (s.cRec != nil && s.cRec.openErr == nil) {
		cost += cErrCostPerRecovery
	}
	return cost
}

func cStackErrorCostForMerge(lang *Language, s *glrStack) uint32 {
	return cStackErrorCostForMergeWithScratch(nil, lang, s)
}

func cStackErrorCostForMergeWithScratch(scratch *glrMergeScratch, lang *Language, s *glrStack) uint32 {
	if s == nil {
		return 0
	}
	var cost uint32
	walk := func(n *Node) {
		if n != nil {
			cost += cNodeErrorCostLangWithScratch(scratch, lang, n)
		}
	}
	if len(s.entries) > 0 {
		for i := range s.entries {
			walk(stackEntryNode(s.entries[i]))
		}
	} else {
		for gn := s.gss.head; gn != nil; gn = gn.prev {
			walk(stackEntryNode(gn.entry))
		}
	}
	if s.cPaused || (s.cRec != nil && s.cRec.openErr == nil) {
		cost += cErrCostPerRecovery
	}
	return cost
}

// cStackCumulativeNodeCount mirrors C StackNode.node_count at the stack head:
// the sum of stack__subtree_node_count over every subtree on the stack. The
// engine's open ERROR region node plays the role of the C error_repeat chain
// (its own visible +1 matches the chain's single error_repeat bonus).
func (p *Parser) cStackCumulativeNodeCount(s *glrStack) int {
	if s == nil {
		return 0
	}
	count := 0
	walk := func(e stackEntry) {
		if n := stackEntryNode(e); n != nil {
			count += p.cNodeVisibleSubtreeCount(n)
		}
	}
	if len(s.entries) > 0 {
		for i := range s.entries {
			walk(s.entries[i])
		}
		return count
	}
	for gn := s.gss.head; gn != nil; gn = gn.prev {
		walk(gn.entry)
	}
	return count
}

func (p *Parser) cApplyMergedErrorGroupBaseline(versions []glrStack) int {
	groupBaseline := 0
	for vi := range versions {
		if count := p.cStackCumulativeNodeCount(&versions[vi]); count > groupBaseline {
			groupBaseline = count
		}
	}
	for vi := range versions {
		versions[vi].cNodeBaseline = groupBaseline
	}
	return groupBaseline
}

// cNodeCountSinceError ports ts_stack_node_count_since_error: the cumulative
// node count minus the count recorded when the error discontinuity was pushed
// (glrStack.cNodeBaseline; zero for stacks that never errored, which matches
// C's node_count_at_last_error starting at zero).
func (p *Parser) cNodeCountSinceError(s *glrStack) int {
	if s == nil {
		return 0
	}
	count := p.cStackCumulativeNodeCount(s) - s.cNodeBaseline
	if count < 0 {
		// C clamps (and writes back) when the stack popped below the baseline.
		s.cNodeBaseline = p.cStackCumulativeNodeCount(s)
		return 0
	}
	return count
}

// ---------------------------------------------------------------------------
// Version status + comparison (parser.c ErrorStatus / ErrorComparison port)
// ---------------------------------------------------------------------------

type cErrorStatus struct {
	cost      uint32
	nodeCount int
	dynPrec   int
	isInError bool
}

type cErrorComparison int

const (
	cErrorComparisonTakeLeft cErrorComparison = iota
	cErrorComparisonPreferLeft
	cErrorComparisonNone
	cErrorComparisonPreferRight
	cErrorComparisonTakeRight
)

func (p *Parser) cVersionStatus(s *glrStack) cErrorStatus {
	cost := p.cStackErrorCost(s)
	if s.cPaused {
		cost += cErrCostPerSkippedTree
	}
	return cErrorStatus{
		cost:      cost,
		nodeCount: p.cNodeCountSinceError(s),
		dynPrec:   s.score,
		// C ts_parser__version_status: in error when paused or at ERROR_STATE.
		isInError: s.cPaused || s.cRec != nil,
	}
}

// A condense-step drop of a recovery-owning version in favor of a marker-free
// sibling was evaluated as a second trigger for cRecoveryUnvalidatedMarker
// (tagging the surviving stack) but rejected: measured against this repo's
// own valid, compiling Go source, cCondenseAndResume's cost competition
// drops/re-forks recovery-owning-vs-clean candidates as routine, frequent
// (thousands of times per large file) GLR disambiguation on the Go grammar's
// LALR table, not evidence of a real syntax error — a "surviving stack"
// tag propagates through every subsequent clone() of that (winning, and
// therefore usually eventually-selected) lineage for the rest of the parse,
// so it cannot be scoped away by the same-position sibling check the way
// cRecoverToState's single-stack-gated marker can (see below). Only
// cRecoverToState marks cRecoveryUnvalidatedMarker; see its doc comment for
// why that one narrower trigger is sufficient for the confirmed defect class
// (java/php/gomod) without this one.

// cCompareVersions is a literal port of ts_parser__compare_versions.
func cCompareVersions(a, b cErrorStatus) cErrorComparison {
	if !a.isInError && b.isInError {
		if a.cost < b.cost {
			return cErrorComparisonTakeLeft
		}
		return cErrorComparisonPreferLeft
	}
	if a.isInError && !b.isInError {
		if b.cost < a.cost {
			return cErrorComparisonTakeRight
		}
		return cErrorComparisonPreferRight
	}
	if a.cost < b.cost {
		if (b.cost-a.cost)*uint32(1+a.nodeCount) > cRecoverMaxCostDifference {
			return cErrorComparisonTakeLeft
		}
		return cErrorComparisonPreferLeft
	}
	if b.cost < a.cost {
		if (a.cost-b.cost)*uint32(1+b.nodeCount) > cRecoverMaxCostDifference {
			return cErrorComparisonTakeRight
		}
		return cErrorComparisonPreferRight
	}
	if a.dynPrec > b.dynPrec {
		return cErrorComparisonPreferLeft
	}
	if b.dynPrec > a.dynPrec {
		return cErrorComparisonPreferRight
	}
	return cErrorComparisonNone
}

// cBetterVersionExists ports ts_parser__better_version_exists: would the
// candidate (self with hypothetical cost) clearly lose to an existing live
// stack at the same or later position? Stacks in the same absorbing group are
// excluded — they are paths of the same C version, not competitors.
func (p *Parser) cBetterVersionExists(stacks []glrStack, self int, isInError bool, cost uint32) bool {
	pos := stacks[self].byteOffset
	group := (*cRecGroup)(nil)
	if stacks[self].cRec != nil {
		group = stacks[self].cRec.group
	}
	status := cErrorStatus{
		cost:      cost,
		isInError: isInError,
		dynPrec:   stacks[self].score,
		nodeCount: p.cNodeCountSinceError(&stacks[self]),
	}
	for i := range stacks {
		if i == self || stacks[i].dead || stacks[i].byteOffset < pos {
			continue
		}
		// C removes accepted versions from the pool (their tree is stashed
		// for select_tree); they are not competitors here.
		if stacks[i].accepted {
			continue
		}
		if group != nil && stacks[i].cRec != nil && stacks[i].cRec.group == group {
			continue
		}
		if group != nil && stacks[i].cRecoverMissingGroup == group {
			continue
		}
		st := p.cVersionStatus(&stacks[i])
		switch cCompareVersions(status, st) {
		case cErrorComparisonTakeRight:
			return true
		case cErrorComparisonPreferRight:
			// C: only when the two versions could merge (ts_stack_can_merge:
			// same state, position, and error cost).
			if stacksHeaderEquivalent(stacks[self], stacks[i]) &&
				p.cStackErrorCost(&stacks[self]) == p.cStackErrorCost(&stacks[i]) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Stack walking helpers
// ---------------------------------------------------------------------------

// cStackEntriesTopFirst materializes the stack spine top-first.
func cStackEntriesTopFirst(s *glrStack, gssScratch *gssScratch) []stackEntry {
	s.ensureGSS(gssScratch)
	depth := s.depth()
	if depth == 0 {
		return nil
	}
	entries := make([]stackEntry, 0, depth)
	for n := s.gss.head; n != nil; n = n.prev {
		entries = append(entries, n.entry)
	}
	return entries
}

// cEntryCountsTowardDepth mirrors stack__iter subtree counting: non-extra
// subtrees count, NULL discontinuities count, extras do not.
func cEntryCountsTowardDepth(e stackEntry) bool {
	if !stackEntryHasNode(e) {
		// Node-less entries: the stack base does not count (C's base node has
		// no link); the error discontinuity (state 0) counts like C's NULL
		// subtree link. Both are node-less; distinguish by state — only the
		// discontinuity carries cErrorState above a non-empty prefix, and the
		// base is never crossed by bounded pops anyway.
		return e.state == cErrorState
	}
	return !stackEntryNodeIsExtra(e)
}

// cRecordSummary ports ts_stack_record_summary over this engine's linear
// spine: entries top-first, depth = crossings of depth-counting links,
// deduped on (depth, state), bounded by MAX_SUMMARY_DEPTH.
func (p *Parser) cRecordSummary(entries []stackEntry) []cStackSummaryEntry {
	summary := make([]cStackSummaryEntry, 0, 8)
	depth := 0
	// Position of the node at-or-below each entry: C node positions are the
	// cumulative input position at that stack node.
	posBytesAt := make([]uint32, len(entries))
	posRowAt := make([]uint32, len(entries))
	var pb uint32
	var pr uint32
	for i := len(entries) - 1; i >= 0; i-- {
		if stackEntryHasNode(entries[i]) {
			pb = stackEntryNodeEndByte(entries[i])
			pr = stackEntryNodeEndPoint(entries[i]).Row
		}
		posBytesAt[i] = pb
		posRowAt[i] = pr
	}
	record := func(d int, st StateID, posBytes, posRow uint32) {
		for j := len(summary) - 1; j >= 0; j-- {
			if summary[j].depth < d {
				break
			}
			if summary[j].depth == d && summary[j].state == st {
				return
			}
		}
		summary = append(summary, cStackSummaryEntry{depth: d, state: st, posBytes: posBytes, posRow: posRow})
	}
	for i := 0; i < len(entries); i++ {
		record(depth, entries[i].state, posBytesAt[i], posRowAt[i])
		if cEntryCountsTowardDepth(entries[i]) {
			depth++
			if depth > cRecoverMaxSummaryDepth {
				break
			}
		}
	}
	return summary
}

// ---------------------------------------------------------------------------
// do_all_potential_reductions port
// ---------------------------------------------------------------------------

type cReduceActionKey struct {
	symbol Symbol
	count  uint8
}

// cCollectPotentialReductions gathers the deduped reduce-action set for the
// state over the symbol range, and whether any non-extra shift exists
// (parser.c lines 1121-1157).
func (p *Parser) cCollectPotentialReductions(state StateID, lookaheadSym Symbol, anyLookahead bool, reduces *[]ParseAction) bool {
	*reduces = (*reduces)[:0]
	hasShift := false
	seen := make(map[cReduceActionKey]bool, 4)
	scan := func(sym Symbol) {
		idx := p.lookupActionIndex(state, sym)
		if idx == 0 || int(idx) >= len(p.language.ParseActions) {
			return
		}
		for _, act := range p.language.ParseActions[idx].Actions {
			switch act.Type {
			case ParseActionShift, ParseActionRecover:
				if !act.Extra && !act.Repetition {
					hasShift = true
				}
			case ParseActionAccept:
				hasShift = true
			case ParseActionReduce:
				if act.ChildCount > 0 {
					key := cReduceActionKey{symbol: act.Symbol, count: act.ChildCount}
					if !seen[key] {
						seen[key] = true
						*reduces = append(*reduces, act)
					}
				}
			}
		}
	}
	if anyLookahead {
		tokenCount := Symbol(p.language.TokenCount)
		for sym := Symbol(1); sym < tokenCount; sym++ {
			scan(sym)
		}
	} else {
		scan(lookaheadSym)
	}
	return hasShift
}

// cDoAllPotentialReductions ports ts_parser__do_all_potential_reductions for
// one starting stack. It returns the resulting version set (what the starting
// version became, plus surviving forks) and whether some version can shift
// the lookahead. With anyLookahead true the reductions reachable on ANY
// symbol are applied (the "close in-progress productions" step); versions
// that dead-end keep their pre-reduction shape (C leaves them in place).
// With anyLookahead false, dead-end versions are dropped (C removes them).
// EOF is symbol 0, so callers must pass anyLookahead explicitly instead of
// overloading lookaheadSym == 0.
func (p *Parser) cDoAllPotentialReductions(source []byte, start glrStack, lookaheadSym Symbol, anyLookahead bool, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) ([]glrStack, bool) {
	oldDisablePostReduceForkMerge := p.disablePostReduceForkMerge
	p.disablePostReduceForkMerge = true
	defer func() {
		p.disablePostReduceForkMerge = oldDisablePostReduceForkMerge
	}()

	versions := []glrStack{start}
	canShift := false
	var reduces []ParseAction
	v := 0
	for iter := 0; ; iter++ {
		if v >= len(versions) {
			break
		}
		// Merge check against earlier versions created in this call.
		merged := false
		for j := 0; j < v; j++ {
			if p.cTryMergeReductionVersion(&versions[j], &versions[v]) {
				versions = append(versions[:v], versions[v+1:]...)
				merged = true
				break
			}
		}
		if merged {
			continue
		}
		state := versions[v].top().state
		hasShift := p.cCollectPotentialReductions(state, lookaheadSym, anyLookahead, &reduces)
		lastReduction := -1
		for _, act := range reduces {
			actionCandidates := p.cReductionCandidatesForAction(source, versions[v], act, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
			actionReductionVersion := -1
			if len(actionCandidates) > 0 {
				versions, actionReductionVersion = p.cAppendActionReductionVersions(versions, actionCandidates, v, arena)
			}
			// C overwrites reduction_version for every reduce action, including
			// STACK_VERSION_NONE when the action only merges into an existing
			// version or produces no surviving version.
			lastReduction = actionReductionVersion
		}
		if anyLookahead && lastReduction >= 0 && iter < cRecoverMaxReduceIterations {
			if hasShift {
				canShift = true
			}
			// During handle_error's close-in-progress pass, continue chasing
			// reduction results even when this state can shift some token. The
			// shift is not necessarily the current lookahead, and retaining every
			// intermediate shiftable version inflates the recovery summary depth.
			versions[v] = versions[lastReduction]
			versions = append(versions[:lastReduction], versions[lastReduction+1:]...)
			continue
		}
		if hasShift {
			canShift = true
		} else if lastReduction >= 0 && iter < cRecoverMaxVersionCount {
			// C renumbers the LAST reduction version onto the current version
			// (reduction_version is overwritten per reduce action) and
			// reprocesses it in place.
			versions[v] = versions[lastReduction]
			versions = append(versions[:lastReduction], versions[lastReduction+1:]...)
			continue
		} else if !anyLookahead {
			versions = append(versions[:v], versions[v+1:]...)
			continue
		}
		if v == 0 {
			v = 1
		} else {
			v++
		}
		if len(versions) > cRecoverMaxVersionCount+1 {
			break
		}
	}
	return versions, canShift
}

func (p *Parser) cReductionCandidatesForAction(source []byte, start glrStack, act ParseAction, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) []glrStack {
	p.pendingForkStacks = p.pendingForkStacks[:0]
	fork := start.cloneWithScratch(gssScratch)
	fork.cRec = start.cRec.clone()
	var dummy bool
	p.applyAction(source, &fork, act, tok, &dummy, nodeCount, arena, entryScratch, gssScratch, nil, false, trackChildErrors)
	candidates := make([]glrStack, 0, 1+len(p.pendingForkStacks))
	if !fork.dead {
		candidates = append(candidates, fork)
	}
	for i := range p.pendingForkStacks {
		if !p.pendingForkStacks[i].dead {
			candidates = append(candidates, p.pendingForkStacks[i])
		}
	}
	p.pendingForkStacks = p.pendingForkStacks[:0]
	return candidates
}

func (p *Parser) cAppendActionReductionVersions(versions []glrStack, candidates []glrStack, originalVersion int, arena *nodeArena) ([]glrStack, int) {
	candidates = p.cCollapseSamePopReductionCandidates(candidates, arena)
	firstAppended := -1
	for i := range candidates {
		var appended bool
		versions, appended = p.cAppendReductionVersion(versions, candidates[i], originalVersion)
		if appended && firstAppended < 0 {
			firstAppended = len(versions) - 1
		}
	}
	return versions, firstAppended
}

func (p *Parser) cCollapseSamePopReductionCandidates(candidates []glrStack, arena *nodeArena) []glrStack {
	if len(candidates) < 2 {
		return candidates
	}
	out := candidates[:0]
	for i := range candidates {
		keep := true
		for j := 0; j < len(out); j++ {
			if p.cTryCollapseSamePopReductionVersion(&out[j], &candidates[i], arena) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, candidates[i])
		}
	}
	return out
}

func (p *Parser) cAppendReductionVersion(versions []glrStack, candidate glrStack, originalVersion int) ([]glrStack, bool) {
	for i := range versions {
		if i == originalVersion {
			continue
		}
		if p.cTryMergeReductionVersion(&versions[i], &candidate) {
			return versions, false
		}
	}
	versions = append(versions, candidate)
	return versions, true
}

func (p *Parser) cTryMergeReductionVersion(target, candidate *glrStack) bool {
	if target == nil || candidate == nil || target.dead || candidate.dead || target.accepted || candidate.accepted {
		return false
	}
	if target.entries != nil || candidate.entries != nil || target.gss.head == nil || candidate.gss.head == nil {
		return false
	}
	if !stacksHeaderEquivalent(*target, *candidate) {
		return false
	}
	return tryGSSMainMergeForParser(p, target, candidate)
}

func (p *Parser) cTryCollapseSamePopReductionVersion(target, candidate *glrStack, arena *nodeArena) bool {
	if target == nil || candidate == nil || target.dead || candidate.dead || target.accepted || candidate.accepted {
		return false
	}
	targetParent, targetPopTo, ok := cReductionParentAndPopTarget(target)
	if !ok {
		return false
	}
	candidateParent, candidatePopTo, ok := cReductionParentAndPopTarget(candidate)
	if !ok || targetPopTo != candidatePopTo {
		return false
	}
	if targetPopTo == nil || !stacksHeaderEquivalent(*target, *candidate) {
		return false
	}
	if p.cSelectReplacementParentEntry(arena, targetParent.entry, candidateParent.entry) {
		*target = *candidate
	}
	return true
}

func cReductionParentAndPopTarget(s *glrStack) (*gssNode, *gssNode, bool) {
	if s == nil || s.gss.head == nil || s.entries != nil {
		return nil, nil, false
	}
	n := s.gss.head
	for n != nil {
		entryNode := stackEntryNode(n.entry)
		if entryNode == nil || !entryNode.isExtra() {
			break
		}
		n = n.prev
	}
	if n == nil || !stackEntryHasNode(n.entry) {
		return nil, nil, false
	}
	return n, n.prev, true
}

func (p *Parser) cSelectReplacementParentEntry(arena *nodeArena, existing, candidate stackEntry) bool {
	existingCost := p.rawStackEntryErrorCost(arena, existing)
	candidateCost := p.rawStackEntryErrorCost(arena, candidate)
	if candidateCost < existingCost {
		return true
	}
	if existingCost < candidateCost {
		return false
	}
	existingDyn := stackEntryDynamicPrecedence(existing)
	candidateDyn := stackEntryDynamicPrecedence(candidate)
	if candidateDyn > existingDyn {
		return true
	}
	if existingDyn > candidateDyn {
		return false
	}
	if existingCost > 0 {
		return true
	}
	return p.compareRawStackEntries(arena, candidate, existing) < 0
}

// ---------------------------------------------------------------------------
// handle_error port
// ---------------------------------------------------------------------------

// cTerminalNextState ports ts_language_next_state for terminals: the shift
// target of the last action (extra shifts keep the state).
func (p *Parser) cTerminalNextState(state StateID, sym Symbol) (StateID, ParseAction, bool) {
	idx := p.lookupActionIndex(state, sym)
	if idx == 0 || int(idx) >= len(p.language.ParseActions) {
		return 0, ParseAction{}, false
	}
	actions := p.language.ParseActions[idx].Actions
	if len(actions) == 0 {
		return 0, ParseAction{}, false
	}
	act := actions[len(actions)-1]
	if act.Type != ParseActionShift {
		return 0, ParseAction{}, false
	}
	if act.Extra {
		return state, act, true
	}
	return act.State, act, true
}

// cHandleError ports ts_parser__handle_error for the stack at index si: run
// do_all_potential_reductions on ANY symbol, attempt one missing-token
// insertion across the version set, push the error discontinuity on every
// version, record summaries, then run ts_parser__recover for the current
// lookahead. C merges all discontinuity versions into ONE multi-link version;
// this engine keeps one linear stack per path and ties them into a cRecGroup
// so the strategy-1 election runs once per token across the group.
//
// Returns the outcome for the original stack (which stacks[si] now reflects)
// and whether any new version still needs to act on the current token (the
// missing-token versions and strategy-1 recoveries) — the caller must force a
// re-dispatch pass for the same token.
func (p *Parser) cHandleError(stacks *[]glrStack, si int, source []byte, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) (cRecoverOutcome, bool) {
	s := &(*stacks)[si]
	s.cPaused = false
	// cHandleError running is NOT proof the input is malformed — LALR table
	// limitations routinely drive well-formed input into a momentary
	// no-action point that step 1 below (cDoAllPotentialReductions) resolves
	// losslessly. Recording that we ran here only lets
	// resolveCRecoverySwallowedError use this as a cheap pre-filter; the
	// actual suspicion signal is cRecoveryDroppedErrorForClean, scoped to the
	// selected result's own lineage in buildResultFromGLR.
	p.crecoveryEnteredErrorState = true
	// s is re-entering recovery, so whatever marker content it may already
	// carry from an earlier cRecoverToState fork or condense-drop win (see
	// cRecoveryUnvalidatedMarker) is about to be re-accounted for by this
	// call's own cost bookkeeping (and, if it competes again,
	// by cCondenseAndResume's comparison loop). Clear the "unvalidated" flag
	// so a lineage that keeps recovering normally doesn't look suspicious at
	// ACCEPT — only a lineage that creates/inherits a marker and then reaches
	// ACCEPT WITHOUT ever coming back through here does.
	s.cRecoveryUnvalidatedMarker = false
	// The swallowed-error defect class (see cRecoveryUnvalidatedMarker) is
	// specifically about a single-stack no-action dead end — the exact
	// scenario the parser.go ~4708 gate is about. Highly ambiguous grammars
	// (kotlin's generic-vs-comparison forks, for example) can drive multiple
	// unrelated GLR candidates into cHandleError while genuinely disambiguating
	// valid input; recovery forks born there routinely settle back to a
	// legitimately clean tree without ever being "re-validated", so they must
	// not be treated as suspicious. Track this per call so cRecoverToState
	// only marks a fork unvalidated when recovery owned the whole parse at
	// that moment.
	p.crecoveryHandleErrorSingleStack = len(*stacks) == 1

	// 1. Close in-progress productions: reductions reachable on any symbol.
	versions, _ := p.cDoAllPotentialReductions(source, s.clone(), 0, true, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
	group := &cRecGroup{}

	// 2. Missing-token insertion (once across the version set, in order).
	// C keeps every version that survives do_all_potential_reductions on the
	// lookahead (the copied version plus its reduction forks).
	var missingVersions []glrStack
	if !p.isGraphQLRecoveryTripleQuote(tok.Symbol) {
		for vi := range versions {
			state := versions[vi].top().state
			tokenCount := Symbol(p.language.TokenCount)
			for ms := Symbol(1); ms < tokenCount; ms++ {
				nextState, shiftAct, ok := p.cTerminalNextState(state, ms)
				if !ok || nextState == 0 || nextState == state {
					continue
				}
				if !p.stateHasLeadingReduceAction(nextState, tok.Symbol) {
					continue
				}
				cand := versions[vi].cloneWithScratch(gssScratch)
				cand.cRec = nil
				cand.cRecoverMissingGroup = nil
				missingTok := Token{
					Symbol:     ms,
					StartByte:  tok.StartByte,
					EndByte:    tok.StartByte,
					StartPoint: tok.StartPoint,
					EndPoint:   tok.StartPoint,
					Missing:    true,
				}
				if top := cand.top(); stackEntryHasNode(top) && stackEntryNodeEndByte(top) <= tok.StartByte {
					missingTok.StartByte = stackEntryNodeEndByte(top)
					missingTok.EndByte = stackEntryNodeEndByte(top)
					missingTok.StartPoint = stackEntryNodeEndPoint(top)
					missingTok.EndPoint = stackEntryNodeEndPoint(top)
				}
				var dummy bool
				p.applyAction(source, &cand, shiftAct, missingTok, &dummy, nodeCount, arena, entryScratch, gssScratch, nil, false, trackChildErrors)
				if p.rejectUndrainedPendingForkStacks(&cand) {
					continue
				}
				cand.shifted = false
				if cand.dead {
					continue
				}
				reduced, canShift := p.cDoAllPotentialReductions(source, cand, tok.Symbol, false, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
				if !canShift || len(reduced) == 0 {
					continue
				}
				missingVersions = reduced
				break
			}
			if missingVersions != nil {
				break
			}
		}
	}

	// 3. Enter the error state on every version: push the discontinuity
	// (C NULL subtree at ERROR_STATE), apply the merged node-count baseline
	// (C: ts_stack_merge resets node_count_at_last_error to the merged head's
	// max node count), and record the stack summary. All versions share one
	// cRecGroup — the engine equivalent of C merging them into one version
	// before ts_stack_record_summary.
	for vi := range versions {
		v := &versions[vi]
		v.pushEntry(stackEntry{state: cErrorState}, entryScratch, gssScratch)
		v.shifted = false
	}
	p.cApplyMergedErrorGroupBaseline(versions)
	for vi := range versions {
		v := &versions[vi]
		entries := cStackEntriesTopFirst(v, gssScratch)
		v.cRec = &cRecoverState{summary: p.cRecordSummary(entries), group: group, groupOrder: vi}
		v.cRecoverMissingGroup = nil
	}

	// The original stack becomes the first absorbing version.
	*s = versions[0]
	for vi := 1; vi < len(versions); vi++ {
		*stacks = append(*stacks, versions[vi])
	}

	// 4. Run recover for the current lookahead across the absorbing group.
	// Recover may fork one strategy-1 candidate (which must act on this
	// token), absorb the token on each member, or halt members.
	needsRedispatch := false
	outcome := cRecoverOutcome(cRecConsumed)
	first := true
	for i := 0; i < len(*stacks); i++ {
		v := &(*stacks)[i]
		if v.dead || v.cRec == nil || v.cRec.group != group {
			continue
		}
		res, forked := p.cRecover(stacks, v, source, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
		v = &(*stacks)[i]
		if forked {
			needsRedispatch = true
		}
		if first {
			outcome = res
			first = false
		} else if res == cRecHalted {
			v.dead = true
		}
	}

	for vi := range missingVersions {
		missingVersions[vi].branchOrder = (*stacks)[si].branchOrder
		missingVersions[vi].cRecoverMissingGroup = group
		*stacks = append(*stacks, missingVersions[vi])
		needsRedispatch = true
	}
	return outcome, needsRedispatch
}

// ---------------------------------------------------------------------------
// recover port
// ---------------------------------------------------------------------------

// cRecover ports ts_parser__recover for one absorbing group member. The
// strategy-1 summary scan runs ONCE per token across the whole group (C runs
// it once on its single merged version); the skip-token tail runs per member.
// It may append one strategy-1 recovered fork to *stacks (returned
// forked=true: the fork must act on the current token), absorb the token into
// the open error region (cRecConsumed), accept at EOF (cRecConsumed), or halt
// the member (cRecHalted).
func (p *Parser) cRecover(stacks *[]glrStack, v *glrStack, source []byte, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) (cRecoverOutcome, bool) {
	rec := v.cRec
	if rec == nil {
		return cRecFallthrough, false
	}
	vIndex := -1
	for i := range *stacks {
		if &(*stacks)[i] == v {
			vIndex = i
			break
		}
	}

	// Strategy 1: recover to a previous state from the group summary in which
	// the lookahead is valid. Elected once per token across the group.
	didRecover := false
	forked := false
	if g := rec.group; g != nil {
		if !g.electionDone || g.electionTokenStart != tok.StartByte || g.electionTokenSymbol != tok.Symbol {
			g.electionDone = true
			g.electionTokenStart = tok.StartByte
			g.electionTokenSymbol = tok.Symbol
			didRecover, forked = p.cRecoverStrategy1Election(stacks, g, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
			// Re-resolve v: the fork append may have reallocated the slice.
			if vIndex >= 0 {
				v = &(*stacks)[vIndex]
				rec = v.cRec
			}
		}
	}

	// C: if strategy 1 succeeded and there are already too many versions,
	// drop the absorbing version. Count the group as ONE version (C keeps the
	// absorbing paths inside a single merged version).
	if didRecover && p.cEffectiveVersionCount(*stacks, rec.group) > cRecoverMaxVersionCount {
		v.dead = true
		return cRecHalted, forked
	}

	// EOF: wrap everything and accept (ts_parser__recover recover_eof).
	if tok.Symbol == 0 && tok.StartByte == tok.EndByte {
		p.cRecoverEOFAccept(v, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
		return cRecConsumed, forked
	}

	// Do not skip the token if doing so would clearly be worse than some
	// existing version.
	tokBytes := uint32(0)
	if tok.EndByte > tok.StartByte {
		tokBytes = tok.EndByte - tok.StartByte
	}
	tokRows := uint32(0)
	if tok.EndPoint.Row > tok.StartPoint.Row {
		tokRows = tok.EndPoint.Row - tok.StartPoint.Row
	}
	newCost := p.cStackErrorCost(v) + cErrCostPerSkippedTree +
		tokBytes*cErrCostPerSkippedChar + tokRows*cErrCostPerSkippedLine
	if vIndex >= 0 && p.cBetterVersionExists(*stacks, vIndex, true, newCost) {
		v.dead = true
		return cRecHalted, forked
	}

	// Wrap the lookahead into the open error region (strategy 2).
	if !p.guardRealTokenAttachmentGap(source, v, tok, "c-recover") {
		return cRecHalted, forked
	}
	p.cAbsorbTokenIntoError(v, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
	v.shifted = true
	return cRecConsumed, forked
}

// cEffectiveVersionCount counts live stacks with the absorbing group folded
// into a single version, mirroring C's version accounting.
func (p *Parser) cEffectiveVersionCount(stacks []glrStack, group *cRecGroup) int {
	count := 0
	members := 0
	for i := range stacks {
		if stacks[i].dead || stacks[i].accepted {
			continue
		}
		if group != nil && stacks[i].cRec != nil && stacks[i].cRec.group == group {
			members++
			continue
		}
		count++
	}
	if members > 0 {
		count++
	}
	return count
}

// cRecoverStrategy1Election runs the C summary scan once per token across all
// absorbing group members, in C's merged-summary order: depth-major, member
// order minor (ts_stack_record_summary's breadth-first traversal of the
// merged version's paths), deduped on (depth, state). At most one fork is
// created, owned by the member whose path carried the elected entry.
func (p *Parser) cRecoverStrategy1Election(stacks *[]glrStack, group *cRecGroup, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) (didRecover, forked bool) {
	// C runs the summary scan for every non-error lookahead INCLUDING the EOF
	// token (parser.c ts_parser__recover: `summary && !ts_subtree_is_error`).
	// EOF election is what lets C pop the open error region to a state where
	// the end symbol is valid and finish with a named root + contained ERROR
	// instead of recover_eof's whole-file ERROR wrap. A wide symbol-0 token is
	// this engine's unlexable run (C: error-subtree lookahead) — C skips
	// strategy 1 for those.
	if tok.Symbol == errorSymbol {
		return false, false
	}
	if p.isGraphQLRecoveryTripleQuote(tok.Symbol) {
		return false, false
	}
	if tok.Symbol == 0 && tok.StartByte != tok.EndByte {
		return false, false
	}
	var members []int
	for i := range *stacks {
		if !(*stacks)[i].dead && (*stacks)[i].cRec != nil && (*stacks)[i].cRec.group == group {
			members = append(members, i)
		}
	}
	if len(members) == 0 {
		return false, false
	}
	cSortRecoverMembersByGroupOrder((*stacks), members)
	type seenKey struct {
		depth int
		state StateID
	}
	seen := make(map[seenKey]bool, 16)
	for d := 0; d <= cRecoverMaxSummaryDepth+1; d++ {
		for _, mi := range members {
			for _, entry := range (*stacks)[mi].cRec.summary {
				if entry.depth != d {
					continue
				}
				if entry.state == cErrorState {
					continue
				}
				key := seenKey{depth: entry.depth, state: entry.state}
				if seen[key] {
					continue
				}
				pos := (*stacks)[mi].byteOffset
				if entry.posBytes == pos {
					continue
				}
				depthBump := 0
				if p.cNodeCountSinceError(&(*stacks)[mi]) > 0 {
					// C: the open error region occupies one extra non-extra slot above
					// the recorded summary.
					depthBump = 1
				}
				depth := entry.depth + depthBump
				// Do not recover in ways that create redundant stack versions.
				wouldMerge := false
				for i := range *stacks {
					if (*stacks)[i].dead || (*stacks)[i].accepted {
						continue
					}
					if (*stacks)[i].top().state == entry.state && (*stacks)[i].byteOffset == pos {
						wouldMerge = true
						break
					}
				}
				if wouldMerge {
					continue
				}
				if p.lookupActionIndex(entry.state, tok.Symbol) == 0 {
					continue
				}
				curCost := p.cStackErrorCost(&(*stacks)[mi])
				curRow := cStackPosRow(&(*stacks)[mi])
				newCost := curCost +
					uint32(entry.depth)*cErrCostPerSkippedTree +
					(pos-entry.posBytes)*cErrCostPerSkippedChar +
					(curRow-entry.posRow)*cErrCostPerSkippedLine
				if p.cBetterVersionExists(*stacks, mi, false, newCost) {
					return false, false
				}
				if fork, ok := p.cRecoverToState(&(*stacks)[mi], depth, entry.state, arena, entryScratch, gssScratch, trackChildErrors); ok {
					seen[key] = true
					fork.branchOrder = (*stacks)[mi].branchOrder
					*stacks = append(*stacks, fork)
					if nodeCount != nil {
						*nodeCount = *nodeCount + 1
					}
					if p.glrTrace {
						traceCRecoverToState(entry.state, depth)
					}
					return true, true
				}
			}
		}
	}
	return false, false
}

func cSortRecoverMembersByGroupOrder(stacks []glrStack, members []int) {
	for i := 1; i < len(members); i++ {
		cur := members[i]
		curOrder := stacks[cur].cRec.groupOrder
		j := i - 1
		for ; j >= 0; j-- {
			prev := members[j]
			if stacks[prev].cRec.groupOrder <= curOrder {
				break
			}
			members[j+1] = prev
		}
		members[j+1] = cur
	}
}

// cRecoverEOFAccept ports the recover_eof tail of ts_parser__recover combined
// with ts_parser__accept's root construction: wrap every subtree on the stack
// (with the open error region's children spliced, mirroring the invisible
// error_repeat flattening) into one ERROR root, and accept.
func (p *Parser) cRecoverEOFAccept(v *glrStack, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) {
	entries := cStackEntriesTopFirst(v, gssScratch)
	children := make([]*Node, 0, len(entries))
	openErr := (*Node)(nil)
	if v.cRec != nil {
		openErr = v.cRec.openErr
	}
	var rawFirst, rawLast *Node
	for i := len(entries) - 1; i >= 0; i-- {
		if !stackEntryHasNode(entries[i]) {
			continue // the stack base and the error discontinuity
		}
		n, _ := materializeStackEntryPayloadEntryWithParser(p, arena, entries[i], materializeForRecovery, materializeForRecovery)
		if n == nil {
			continue
		}
		if rawFirst == nil {
			rawFirst = n
		}
		rawLast = n
		if n == openErr {
			// Open-region children were visible-spliced at absorb time.
			children = append(children, n.children...)
			continue
		}
		children = p.cAppendRecoveryVisibleSplice(children, n, arena)
	}
	root := newParentNodeInArena(arena, errorSymbol, true, children, nil, 0)
	if rawFirst != nil {
		cSetNodeSpan(root, rawFirst.startByte, rawLast.endByte, rawFirst.startPoint, rawLast.endPoint)
	} else {
		cSetNodeSpan(root, tok.StartByte, tok.EndByte, tok.StartPoint, tok.EndPoint)
	}
	root.setHasError(true)
	nodeBumpEquivVersion(root)
	if perfCountersEnabled {
		perfRecordErrorNode()
	}
	if trackChildErrors != nil {
		*trackChildErrors = true
	}
	if nodeCount != nil {
		*nodeCount = *nodeCount + 1
	}
	v.truncate(1)
	v.cRec = nil
	v.cRecoverMissingGroup = nil
	p.pushStackNode(v, 1, root, entryScratch, gssScratch)
	v.accepted = true
	v.shifted = true
}

func cStackPosRow(s *glrStack) uint32 {
	if s == nil {
		return 0
	}
	if len(s.entries) > 0 {
		for i := len(s.entries) - 1; i >= 0; i-- {
			if stackEntryHasNode(s.entries[i]) {
				return stackEntryNodeEndPoint(s.entries[i]).Row
			}
		}
		return 0
	}
	for gn := s.gss.head; gn != nil; gn = gn.prev {
		if stackEntryHasNode(gn.entry) {
			return stackEntryNodeEndPoint(gn.entry).Row
		}
	}
	return 0
}

// cAppendVisibleSplice appends n's visible projection to dst the way the
// engine's reduce does: invisible nodes are spliced to their children at
// build time (C keeps them and hides them at query time instead — same
// visible tree). ERROR and missing nodes always stay.
func (p *Parser) cAppendVisibleSplice(dst []*Node, n *Node) []*Node {
	if n == nil {
		return dst
	}
	if n.symbol == errorSymbol || n.isMissing() || p.cSymbolVisible(n.symbol) {
		return append(dst, n)
	}
	for _, c := range n.children {
		dst = p.cAppendVisibleSplice(dst, c)
	}
	return dst
}

func (p *Parser) cRecoveryVisibleSpliceCandidate(n *Node) bool {
	if !p.errorCostCompetitionEnabled() ||
		n == nil ||
		n.symbol != errorSymbol ||
		!n.isExtra() ||
		n.isMissing() ||
		len(n.children) == 0 {
		return false
	}
	for _, child := range n.children {
		if child == nil ||
			child.symbol == errorSymbol ||
			child.isExtra() ||
			child.isMissing() ||
			!p.cSymbolVisible(child.symbol) {
			return false
		}
	}
	return true
}

func (p *Parser) cRecoveryVisibleSpliceSignature(n *Node) (ProductionSignature, bool) {
	if p == nil || p.language == nil || len(p.language.ProductionSignatures) == 0 {
		return ProductionSignature{}, false
	}
	match := -1
	for i := range p.language.ProductionSignatures {
		sig := p.language.ProductionSignatures[i]
		if !p.cSymbolVisible(sig.LHS) || len(sig.RHS) != len(n.children) {
			continue
		}
		matches := true
		for j, rhs := range sig.RHS {
			if !p.cRecoverySignatureChildMatches(rhs, n.children[j]) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		if match >= 0 {
			return ProductionSignature{}, false
		}
		match = i
	}
	if match < 0 {
		return ProductionSignature{}, false
	}
	if !p.cRecoverySignatureProductionSafe(p.language.ProductionSignatures[match].ProductionID) {
		return ProductionSignature{}, false
	}
	return p.language.ProductionSignatures[match], true
}

func (p *Parser) cRecoverySignatureChildMatches(rhs Symbol, child *Node) bool {
	if p == nil || p.language == nil || child == nil {
		return false
	}
	if p.cRecoverySignatureChildMatchesDirect(rhs, child.symbol) {
		return true
	}
	if int(rhs) >= len(p.language.HiddenChoicePassthroughSymbols) ||
		!p.language.HiddenChoicePassthroughSymbols[rhs] {
		return false
	}
	for i := range p.language.ProductionSignatures {
		sig := p.language.ProductionSignatures[i]
		if sig.LHS != rhs || len(sig.RHS) != 1 {
			continue
		}
		if p.cRecoverySignatureChildMatchesDirect(sig.RHS[0], child.symbol) {
			return true
		}
	}
	return false
}

func (p *Parser) cRecoverySignatureChildMatchesDirect(rhs, child Symbol) bool {
	if rhs == child {
		return true
	}
	if p == nil || p.language == nil {
		return false
	}
	for _, subtype := range p.language.SupertypeChildren(rhs) {
		if subtype == child {
			return true
		}
	}
	return false
}

func (p *Parser) cRecoverySignatureProductionSafe(productionID uint16) bool {
	if p == nil || p.language == nil {
		return false
	}
	idx := int(productionID)
	if idx < len(p.language.FieldMapSlices) && p.language.FieldMapSlices[idx][1] != 0 {
		return false
	}
	if idx < len(p.language.AliasSequences) && len(p.language.AliasSequences[idx]) != 0 {
		return false
	}
	return true
}

func (p *Parser) cRecoveryVisibleSpliceCount(n *Node) (int, bool) {
	if !p.cRecoveryVisibleSpliceCandidate(n) {
		return 0, false
	}
	if _, ok := p.cRecoveryVisibleSpliceSignature(n); ok {
		return 1, true
	}
	return len(n.children), true
}

func (p *Parser) cRecoveryVisibleSpliceSignatureNode(n *Node, arena *nodeArena) *Node {
	if arena == nil {
		return nil
	}
	sig, ok := p.cRecoveryVisibleSpliceSignature(n)
	if !ok {
		return nil
	}
	children := arena.allocNodeSliceNoClear(len(n.children))
	copy(children, n.children)
	parent := newParentNodeInArena(arena, sig.LHS, p.isNamedSymbol(sig.LHS), children, nil, sig.ProductionID)
	parent.preGotoState = n.preGotoState
	parent.parseState = n.parseState
	if n.preGotoState != 0 {
		if gotoState := p.lookupGoto(n.preGotoState, sig.LHS); gotoState != 0 {
			parent.parseState = gotoState
		}
	}
	parent.rawShape = captureRawShapeForNodeSlice(arena, sig.LHS, sig.ProductionID, children)
	nodeBumpEquivVersion(parent)
	return parent
}

func (p *Parser) cRecoveryVisibleSpliceChildren(n *Node, arena *nodeArena) ([]*Node, bool) {
	if !p.cRecoveryVisibleSpliceCandidate(n) {
		return nil, false
	}
	if wrapped := p.cRecoveryVisibleSpliceSignatureNode(n, arena); wrapped != nil {
		return []*Node{wrapped}, true
	}
	return n.children, true
}

func (p *Parser) cAppendRecoveryVisibleSplice(dst []*Node, n *Node, arena *nodeArena) []*Node {
	if children, ok := p.cRecoveryVisibleSpliceChildren(n, arena); ok {
		return append(dst, children...)
	}
	return p.cAppendVisibleSplice(dst, n)
}

// cSetNodeSpan pins a recovery node's span explicitly: C error regions span
// every absorbed subtree, including invisible ones the engine splices away.
func cSetNodeSpan(n *Node, startByte, endByte uint32, startPoint, endPoint Point) {
	n.startByte = startByte
	n.endByte = endByte
	n.startPoint = startPoint
	n.endPoint = endPoint
}

// cRecoverToState ports ts_parser__recover_to_state: pop `depth`
// depth-counting links off a copy of v, splice in any open error region
// children, wrap the popped subtrees (minus trailing extras) into an extra
// ERROR node pushed at the goal state, and re-push the trailing extras.
func (p *Parser) cRecoverToState(v *glrStack, depth int, goal StateID, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) (glrStack, bool) {
	entries := cStackEntriesTopFirst(v, gssScratch)
	if len(entries) == 0 {
		return glrStack{}, false
	}
	// Find the cut index: cross `depth` depth-counting links from the top.
	crossed := 0
	cut := -1
	for i := 0; i < len(entries); i++ {
		if crossed == depth {
			cut = i
			break
		}
		if cEntryCountsTowardDepth(entries[i]) {
			crossed++
		}
	}
	if cut < 0 {
		if crossed == depth {
			cut = len(entries)
		} else {
			return glrStack{}, false
		}
	}
	if cut >= len(entries) || entries[cut].state != goal {
		return glrStack{}, false
	}

	// Materialize popped payloads in stack order (base-most first).
	popped := entries[:cut]
	nodes := make([]*Node, 0, len(popped))
	for i := len(popped) - 1; i >= 0; i-- {
		if !stackEntryHasNode(popped[i]) {
			continue // the error discontinuity
		}
		n, _ := materializeStackEntryPayloadEntryWithParser(p, arena, popped[i], materializeForRecovery, materializeForRecovery)
		if n == nil {
			return glrStack{}, false
		}
		nodes = append(nodes, n)
	}

	// Split trailing extras (re-pushed after the ERROR per C).
	end := len(nodes)
	for end > 0 && nodes[end-1].isExtra() {
		end--
	}
	wrapped := nodes[:end]
	trailing := nodes[end:]

	// Flatten the open error region (C splices popped error subtrees /
	// keeps error_repeat chains invisible; the Go equivalent is splicing the
	// open ERROR node's children) and splice invisible nodes the way the
	// engine's reduce does. The raw popped extent pins the ERROR span (C
	// error regions cover invisible subtrees too).
	children := make([]*Node, 0, len(wrapped)+2)
	openErr := (*cRecoverState)(nil)
	if v.cRec != nil {
		openErr = v.cRec
	}
	var rawFirst, rawLast *Node
	for _, n := range wrapped {
		if rawFirst == nil {
			rawFirst = n
		}
		rawLast = n
		if openErr != nil && n == openErr.openErr {
			// Open-region children were visible-spliced at absorb time.
			children = append(children, n.children...)
			continue
		}
		children = p.cAppendRecoveryVisibleSplice(children, n, arena)
	}

	fork := v.cloneWithScratch(gssScratch)
	fork.cRec = nil
	fork.cRecoverMissingGroup = nil
	fork.dead = false
	fork.shifted = false
	keepDepth := len(entries) - cut
	if !fork.truncate(keepDepth) {
		return glrStack{}, false
	}
	// C also pops a directly-preceding closed ERROR subtree and splices its
	// children in front (ts_stack_pop_error).
	if top := stackEntryNode(fork.top()); top != nil && top.symbol == errorSymbol && !top.isMissing() && fork.depth() > 1 {
		prev := top
		if fork.truncate(fork.depth() - 1) {
			children = append(append(make([]*Node, 0, len(prev.children)+len(children)), prev.children...), children...)
			if rawFirst == nil {
				rawLast = prev
			}
			rawFirst = prev
		}
	}

	if rawFirst != nil {
		errNode := newParentNodeInArena(arena, errorSymbol, true, children, nil, 0)
		cSetNodeSpan(errNode, rawFirst.startByte, rawLast.endByte, rawFirst.startPoint, rawLast.endPoint)
		errNode.setHasError(true)
		errNode.setExtra(true)
		errNode.preGotoState = goal
		errNode.parseState = goal
		nodeBumpEquivVersion(errNode)
		if perfCountersEnabled {
			perfRecordErrorNode()
		}
		if trackChildErrors != nil {
			*trackChildErrors = true
		}
		// This fork now genuinely carries an ERROR node (real C's
		// recover_to_state always wraps something). If it reaches ACCEPT
		// without ever cycling back through cHandleError to be
		// re-validated by another cost competition, cStackErrorCost cannot
		// legitimately have dropped to zero — see the ACCEPT-time check in
		// buildResultFromGLR. Track this only for:
		//   - single-stack dead ends (see crecoveryHandleErrorSingleStack): a
		//     fork born while several unrelated GLR ambiguity candidates were
		//     already live is normal disambiguation, not the swallowed-error
		//     defect class (e.g. kotlin's generic-vs-comparison forks).
		//   - a small absolute recovered span (see
		//     crecoverySwallowedErrorMaxFallbackErrorBytes, reused here for
		//     the same "local, not whole-construct, recovery" rationale): an
		//     adversarial review found cRecoverToState firing on ordinary,
		//     syntactically valid Go source — a real repo file's own
		//     composite-literal/statement disambiguation can drive a
		//     hundreds-of-bytes single-stack strategy-1 recovery that the
		//     rest of the (entirely valid) parse absorbs losslessly, the
		//     same "LALR table gap resolved without a real error" pattern as
		//     eds's legitimate empty-value production. The confirmed
		//     defect-class fixtures recover a single malformed token or
		//     directive (java 6 bytes, gomod 5 bytes) — nowhere near that
		//     scale.
		if p.crecoveryHandleErrorSingleStack &&
			errNode.endByte-errNode.startByte <= crecoverySwallowedErrorMaxFallbackErrorBytes {
			fork.cRecoveryUnvalidatedMarker = true
		}
		p.pushStackNode(&fork, goal, errNode, entryScratch, gssScratch)
	}
	for _, ex := range trailing {
		p.pushStackNode(&fork, goal, ex, entryScratch, gssScratch)
	}
	return fork, true
}

// cAbsorbTokenIntoError ports the strategy-2 tail of ts_parser__recover:
// mark extra-shiftable tokens extra (excluded from error cost), then fold the
// token into the open error region at ERROR_STATE.
func (p *Parser) cAbsorbTokenIntoError(v *glrStack, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) {
	// Invisible tokens stay out of the visible children (the engine splices
	// invisibles at build time; C hides them at query time) but still extend
	// the error region's span.
	leafVisible := p.cSymbolVisible(tok.Symbol)
	var leaf *Node
	if leafVisible {
		leaf = newLeafNodeInArena(arena, tok.Symbol, p.isNamedSymbol(tok.Symbol),
			tok.StartByte, tok.EndByte, tok.StartPoint, tok.EndPoint)
		leaf.setHasError(true)
		// C: if the token shifts as extra in state 1, mark it extra so it is
		// not counted in error cost calculations.
		if idx := p.lookupActionIndex(1, tok.Symbol); idx != 0 && int(idx) < len(p.language.ParseActions) {
			if actions := p.language.ParseActions[idx].Actions; len(actions) > 0 {
				if last := actions[len(actions)-1]; last.Type == ParseActionShift && last.Extra {
					leaf.setExtra(true)
				}
			}
		}
	}
	if trackChildErrors != nil {
		*trackChildErrors = true
	}

	appendLeaf := func(dst []*Node) []*Node {
		if leaf != nil {
			return append(dst, leaf)
		}
		return dst
	}

	rec := v.cRec
	if rec != nil && rec.openErr != nil {
		top := stackEntryNode(v.top())
		if top == rec.openErr {
			rec.openErr.children = appendLeaf(rec.openErr.children)
			rec.openErr.endByte = tok.EndByte
			rec.openErr.endPoint = tok.EndPoint
			nodeBumpEquivVersion(rec.openErr)
			if v.byteOffset < tok.EndByte {
				v.byteOffset = tok.EndByte
			}
			if nodeCount != nil {
				*nodeCount = *nodeCount + 1
			}
			return
		}
		// Extras were pushed above the open error region (C pops the previous
		// error_repeat plus trailing extras and re-wraps them together).
		entries := cStackEntriesTopFirst(v, gssScratch)
		above := 0
		found := false
		for i := 0; i < len(entries); i++ {
			if stackEntryNode(entries[i]) == rec.openErr {
				above = i
				found = true
				break
			}
		}
		if found {
			extras := make([]*Node, 0, above)
			for i := above - 1; i >= 0; i-- {
				n, _ := materializeStackEntryPayloadEntryWithParser(p, arena, entries[i], materializeForRecovery, materializeForRecovery)
				if n == nil {
					found = false
					break
				}
				extras = p.cAppendVisibleSplice(extras, n)
			}
			if found && v.truncate(len(entries)-above) {
				rec.openErr.children = append(rec.openErr.children, extras...)
				rec.openErr.children = appendLeaf(rec.openErr.children)
				rec.openErr.endByte = tok.EndByte
				rec.openErr.endPoint = tok.EndPoint
				nodeBumpEquivVersion(rec.openErr)
				if v.byteOffset < tok.EndByte {
					v.byteOffset = tok.EndByte
				}
				if nodeCount != nil {
					*nodeCount = *nodeCount + 1
				}
				return
			}
		}
	}

	var errChildren []*Node
	if leaf != nil {
		errChildren = []*Node{leaf}
	}
	errNode := newParentNodeInArena(arena, errorSymbol, true, errChildren, nil, 0)
	cSetNodeSpan(errNode, tok.StartByte, tok.EndByte, tok.StartPoint, tok.EndPoint)
	errNode.setHasError(true)
	errNode.parseState = cErrorState
	nodeBumpEquivVersion(errNode)
	if perfCountersEnabled {
		perfRecordErrorNode()
	}
	p.pushStackNode(v, cErrorState, errNode, entryScratch, gssScratch)
	if rec != nil {
		rec.openErr = errNode
	}
	if nodeCount != nil {
		*nodeCount = *nodeCount + 2
	}
}

// ---------------------------------------------------------------------------
// Dispatch hooks
// ---------------------------------------------------------------------------

// cRecoverDispatchInError intercepts dispatch for a stack already in the
// error state (C: the ERROR_STATE table row). Shiftable tokens fall through
// to the normal dispatch — extras shift in ERROR_STATE without extending the
// error, and non-terminal extras (e.g. requirements' linebreak) enter their
// sub-parse via a real shift in the state-0 row. Everything else goes through
// ts_parser__recover.
func (p *Parser) cRecoverDispatchInError(stacks *[]glrStack, si int, source []byte, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) (cRecoverOutcome, bool) {
	s := &(*stacks)[si]
	if s.top().state != cErrorState {
		// Mid non-terminal-extra parse: the version temporarily left
		// ERROR_STATE; C dispatches it through the normal table rows.
		return cRecFallthrough, false
	}
	if tok.Symbol != 0 {
		if p.isGraphQLRecoveryTripleQuote(tok.Symbol) {
			return p.cRecover(stacks, s, source, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
		}
		if idx := p.lookupActionIndex(cErrorState, tok.Symbol); idx != 0 && int(idx) < len(p.language.ParseActions) {
			if actions := p.language.ParseActions[idx].Actions; len(actions) > 0 &&
				actions[0].Type == ParseActionShift {
				return cRecFallthrough, false
			}
		}
		// Zero-width non-EOF tokens are skipped (C's error-mode lexer never
		// returns empty internal tokens; the Go DFA source can).
		if tok.StartByte == tok.EndByte {
			s.shifted = true
			return cRecConsumed, false
		}
	}
	return p.cRecover(stacks, s, source, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
}

func (p *Parser) isGraphQLRecoveryTripleQuote(sym Symbol) bool {
	return p != nil &&
		p.language != nil &&
		p.language.Name == "graphql" &&
		int(sym) < len(p.language.SymbolNames) &&
		p.language.SymbolNames[sym] == "\"\"\""
}

// cCondenseAndResume ports ts_parser__condense_stack for the gated grammar:
// remove halted versions, remove versions that clearly lose the error-cost
// competition, order survivors most-promising-first, enforce
// MAX_VERSION_COUNT, and resume the best paused version (ts_parser__handle_error)
// when no unpaused version outranks it. Merging identical stacks remains the
// job of the regular mergeStacks pass. Only runs when some stack is paused or
// in the error state, so clean parses keep today's behavior exactly.
//
// Returns the condensed slice and whether new versions need to re-dispatch
// the current token (strategy-1 forks / missing-token versions created by a
// resumed handle_error).
func (p *Parser) cCondenseAndResume(stacks []glrStack, source []byte, tok Token, nodeCount *int, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch, trackChildErrors *bool) ([]glrStack, bool) {
	relevant := false
	for i := range stacks {
		if stacks[i].cPaused || stacks[i].cRec != nil || stacks[i].cRecoverMissingGroup != nil {
			relevant = true
			break
		}
	}
	if !relevant {
		return stacks, false
	}
	// Drop dead versions first (C removes halted versions in condense).
	// Accepted versions have left the pool in C (ts_parser__accept stashes
	// the tree and removes the version): they sit out the cost competition,
	// the ordering, and the version cap, and rejoin only for final result
	// selection.
	var acceptedStacks []glrStack
	alive := stacks[:0]
	for i := range stacks {
		if stacks[i].dead {
			continue
		}
		if stacks[i].accepted {
			acceptedStacks = append(acceptedStacks, stacks[i])
			continue
		}
		alive = append(alive, stacks[i])
	}
	stacks = alive
	for i := 1; i < len(stacks); i++ {
		statusI := p.cVersionStatus(&stacks[i])
		for j := 0; j < i; j++ {
			if cRecoverVersionsSameGroup(stacks[j], stacks[i]) {
				continue
			}
			if cRecoverVersionShouldStayBefore(stacks[j], stacks[i]) {
				continue
			}
			if cRecoverVersionShouldStayBefore(stacks[i], stacks[j]) {
				stacks[i], stacks[j] = stacks[j], stacks[i]
				statusI = p.cVersionStatus(&stacks[i])
				continue
			}
			statusJ := p.cVersionStatus(&stacks[j])
			switch cCompareVersions(statusJ, statusI) {
			case cErrorComparisonTakeLeft:
				if p.glrTrace {
					p.traceCCondenseDrop("take-left", i, j, stacks[i], stacks[j], statusI, statusJ)
				}
				stacks = append(stacks[:i], stacks[i+1:]...)
				i--
				j = i
			case cErrorComparisonPreferRight:
				if p.glrTrace {
					p.traceCCondenseSwap("prefer-right", i, j, stacks[i], stacks[j], statusI, statusJ)
				}
				stacks[i], stacks[j] = stacks[j], stacks[i]
				statusI = p.cVersionStatus(&stacks[i])
			case cErrorComparisonTakeRight:
				if p.glrTrace {
					p.traceCCondenseDrop("take-right", j, i, stacks[j], stacks[i], statusJ, statusI)
				}
				stacks = append(stacks[:j], stacks[j+1:]...)
				i--
				j--
				statusI = p.cVersionStatus(&stacks[i])
			}
			if i < 1 {
				break
			}
		}
	}
	if len(stacks) > cRecoverMaxVersionCount {
		if p.glrTrace {
			for i := cRecoverMaxVersionCount; i < len(stacks); i++ {
				p.traceCCondenseTrim(i, stacks[i])
			}
		}
		stacks = stacks[:cRecoverMaxVersionCount]
	}

	// Resume the best paused version; remove the rest (C condense tail).
	//
	// C's single dispatch loop keeps every version at the same token position,
	// so at most one paused version ever reaches condense out of sync with its
	// siblings. This engine's per-token settle step can occasionally let two
	// versions arrive at condense both paused (e.g. a reduce-only version that
	// still needs to recheck a not-yet-advanced token alongside a sibling that
	// already shifted past it and paused one token later). When that
	// happens, the higher-priority (post-sort index 0) candidate can be
	// "stale": its position no longer matches the current token, so
	// guardRealTokenAttachmentGap halts it immediately. Faithfully resuming
	// only that one and dropping every other paused version (as plain
	// ts_parser__condense_stack does when it only ever sees one paused
	// version) then discards a still-viable sibling and the parse dies with
	// no recovery at all. Fall through to the next paused candidate whenever
	// a resume halts, instead of unconditionally dropping the remaining
	// paused versions after the first attempt — this only ever recovers
	// MORE input (a halt previously meant giving up outright), so it cannot
	// regress an already-successful single-candidate resume.
	needsRedispatch := false
	hasUnpaused := false
	for i := 0; i < len(stacks); i++ {
		if !stacks[i].cPaused {
			hasUnpaused = true
			continue
		}
		if !hasUnpaused {
			if p.glrTrace {
				fmt.Printf("      -> C-RESUME stack=%d state=%d byte=%d\n", i, stacks[i].top().state, stacks[i].byteOffset)
			}
			outcome, redispatch := p.cHandleError(&stacks, i, source, tok, nodeCount, arena, entryScratch, gssScratch, trackChildErrors)
			if redispatch {
				needsRedispatch = true
			}
			if outcome == cRecHalted {
				stacks[i].dead = true
				// Keep hasUnpaused false: give the next still-paused
				// candidate (if any) a chance to resume instead of dropping
				// it unattempted.
				continue
			}
			hasUnpaused = true
			continue
		}
		stacks = append(stacks[:i], stacks[i+1:]...)
		i--
	}
	stacks = append(stacks, acceptedStacks...)
	return stacks, needsRedispatch
}

func (p *Parser) traceCCondenseDrop(reason string, dropIndex, keepIndex int, drop, keep glrStack, dropStatus, keepStatus cErrorStatus) {
	fmt.Printf("      -> C-CONDENSE-DROP reason=%s drop=%d %s %s keep=%d %s %s\n",
		reason,
		dropIndex,
		cRecoverStackTraceKind(drop),
		cCondenseStackTraceSummary(drop, dropStatus),
		keepIndex,
		cRecoverStackTraceKind(keep),
		cCondenseStackTraceSummary(keep, keepStatus),
	)
}

func (p *Parser) traceCCondenseSwap(reason string, i, j int, left, right glrStack, leftStatus, rightStatus cErrorStatus) {
	fmt.Printf("      -> C-CONDENSE-SWAP reason=%s i=%d %s %s j=%d %s %s\n",
		reason,
		i,
		cRecoverStackTraceKind(left),
		cCondenseStackTraceSummary(left, leftStatus),
		j,
		cRecoverStackTraceKind(right),
		cCondenseStackTraceSummary(right, rightStatus),
	)
}

func (p *Parser) traceCCondenseTrim(index int, stack glrStack) {
	fmt.Printf("      -> C-CONDENSE-TRIM index=%d %s state=%d byte=%d depth=%d score=%d\n",
		index,
		cRecoverStackTraceKind(stack),
		stack.top().state,
		stack.byteOffset,
		stack.depth(),
		stack.score,
	)
}

func cCondenseStackTraceSummary(stack glrStack, status cErrorStatus) string {
	return fmt.Sprintf("{state:%d byte:%d depth:%d score:%d cost:%d inErr:%v dyn:%d nodes:%d}",
		stack.top().state,
		stack.byteOffset,
		stack.depth(),
		stack.score,
		status.cost,
		status.isInError,
		status.dynPrec,
		status.nodeCount,
	)
}

func cRecoverVersionsSameGroup(a, b glrStack) bool {
	return a.cRec != nil &&
		b.cRec != nil &&
		a.cRec.group != nil &&
		a.cRec.group == b.cRec.group
}

func cRecoverVersionShouldStayBefore(a, b glrStack) bool {
	if a.dead || b.dead || a.accepted || b.accepted {
		return false
	}
	if a.cPaused || b.cPaused {
		return false
	}
	return a.cRec != nil && a.cRec.group != nil && b.cRec == nil && b.cRecoverMissingGroup == a.cRec.group
}

// cAcceptRootRebuild ports the root construction half of ts_parser__accept:
// C pops the whole accepted stack, finds the TOPMOST non-extra subtree, and
// rebuilds a node with that subtree's symbol whose children are every other
// popped subtree spliced around its own children — so trailing extras (a
// comment after the last rule of a stylesheet, the EOF padding) become
// children OF the root instead of siblings. Without this, the engine's root
// builder sees multiple top-level nodes on an accepted stack and wraps them
// in a synthetic ERROR root (the css "collapse" shape: stylesheet{ERROR{...}}
// where C has stylesheet{...}).
//
// The stack becomes [base, rebuiltRoot]. Stacks that already hold a single
// payload node are left untouched (the no-error fast path and
// cRecoverEOFAccept results).
func (p *Parser) cAcceptRootRebuild(s *glrStack, arena *nodeArena, entryScratch *glrEntryScratch, gssScratch *gssScratch) {
	if p == nil || s == nil || !s.accepted {
		return
	}
	entries := cStackEntriesTopFirst(s, gssScratch)
	payloads := 0
	for i := range entries {
		if stackEntryHasNode(entries[i]) {
			payloads++
		}
	}
	if payloads <= 1 {
		return
	}
	// Materialize base-most first, mirroring C's popped subtree order.
	nodes := make([]*Node, 0, payloads)
	for i := len(entries) - 1; i >= 0; i-- {
		if !stackEntryHasNode(entries[i]) {
			continue // stack base / error discontinuity
		}
		n, _ := materializeStackEntryPayloadEntryWithParser(p, arena, entries[i], materializeForRecovery, materializeForRecovery)
		if n == nil {
			return
		}
		nodes = append(nodes, n)
	}
	// C: the topmost non-extra subtree names the root.
	rootIdx := -1
	for j := len(nodes) - 1; j >= 0; j-- {
		if !nodes[j].isExtra() {
			rootIdx = j
			break
		}
	}
	if rootIdx < 0 {
		return
	}
	cand := nodes[rootIdx]
	children := make([]*Node, 0, len(nodes)-1+len(cand.children))
	for _, n := range nodes[:rootIdx] {
		children = p.cAppendVisibleSplice(children, n)
	}
	children = append(children, cand.children...)
	for _, n := range nodes[rootIdx+1:] {
		children = p.cAppendVisibleSplice(children, n)
	}
	root := newParentNodeInArena(arena, cand.symbol, p.isNamedSymbol(cand.symbol), children, nil, 0)
	root.rawShape = captureRawShapeForNodeSlice(arena, cand.symbol, cand.productionID, children)
	root.dynamicPrecedence = nodeSliceDynamicPrecedence(children)
	first, last := nodes[0], nodes[len(nodes)-1]
	cSetNodeSpan(root, first.startByte, last.endByte, first.startPoint, last.endPoint)
	hasErr := false
	for _, c := range children {
		if c != nil && c.hasError() {
			hasErr = true
			break
		}
	}
	if hasErr || cand.hasError() {
		root.setHasError(true)
	}
	nodeBumpEquivVersion(root)
	if !s.truncate(1) {
		return
	}
	p.pushStackNode(s, 1, root, entryScratch, gssScratch)
}

func nodeSliceDynamicPrecedence(children []*Node) int32 {
	var dyn int32
	for _, child := range children {
		if child != nil {
			dyn += child.dynamicPrecedence
		}
	}
	return dyn
}

func captureRawShapeForNodeSlice(arena *nodeArena, symbol Symbol, productionID uint16, children []*Node) rawShapeRef {
	if arena == nil || len(children) == 0 {
		return 0
	}
	entries := make([]stackEntry, 0, len(children))
	for _, child := range children {
		if child != nil {
			entries = append(entries, newStackEntryNode(child.parseState, child))
		}
	}
	return captureRawShapeForEntries(arena, symbol, productionID, entries)
}

func captureRawShapeForEntries(arena *nodeArena, symbol Symbol, productionID uint16, entries []stackEntry) rawShapeRef {
	if arena == nil || len(entries) == 0 {
		return 0
	}
	var p Parser
	return p.captureRawShape(arena, symbol, productionID, entries, 0, len(entries))
}

// cStackResultErrorCost is the result-selection cost: the error cost of the
// stack's would-be tree (requirement 4 of the spec: fold error cost into
// stackCompareForResultSelection).
func (p *Parser) cStackResultErrorCost(s *glrStack) uint32 {
	return p.cStackErrorCost(s)
}

// cTreeErrorCost computes the C error cost over a finished tree, for
// retry-selection integration (preferRetryTree).
func (p *Parser) cTreeErrorCost(t *Tree) uint32 {
	if t == nil || t.root == nil {
		return 0
	}
	return p.cNodeErrorCost(t.root)
}

func traceCRecoverToState(state StateID, depth int) {
	fmt.Printf("      -> C-RECOVER-TO-STATE state=%d depth=%d\n", state, depth)
}
