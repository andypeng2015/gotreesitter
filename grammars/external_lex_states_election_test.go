package grammars

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestExternalLexStatesDefaultElectionInventory(t *testing.T) {
	tests := []struct {
		name string
		load func() *gotreesitter.Language
	}{
		{name: "angular", load: AngularLanguage},
		{name: "awk", load: AwkLanguage},
		{name: "bash", load: BashLanguage},
		{name: "bicep", load: BicepLanguage},
		{name: "bitbake", load: BitbakeLanguage},
		{name: "c_sharp", load: CSharpLanguage},
		{name: "caddy", load: CaddyLanguage},
		{name: "cooklang", load: CooklangLanguage},
		{name: "css", load: CssLanguage},
		{name: "cue", load: CueLanguage},
		{name: "dtd", load: DtdLanguage},
		{name: "elm", load: ElmLanguage},
		{name: "hack", load: HackLanguage},
		{name: "hlsl", load: HlslLanguage},
		{name: "jsdoc", load: JsdocLanguage},
		{name: "jsonnet", load: JsonnetLanguage},
		{name: "just", load: JustLanguage},
		{name: "kconfig", load: KconfigLanguage},
		{name: "lua", load: LuaLanguage},
		{name: "luau", load: LuauLanguage},
		{name: "python", load: PythonLanguage},
		{name: "scss", load: ScssLanguage},
		{name: "svelte", load: SvelteLanguage},
		{name: "wgsl", load: WgslLanguage},
		{name: "yaml", load: YamlLanguage},
	}

	ledgerDefaults := readDefaultElectionLedger(t)
	if got, want := len(ledgerDefaults), len(tests); got != want {
		t.Fatalf("ledger default_elected count = %d, want %d test cases", got, want)
	}
	seen := make(map[string]bool, len(tests))
	for _, tt := range tests {
		if !ledgerDefaults[tt.name] {
			t.Fatalf("%q is in default election test cases but not default_elected in ledger", tt.name)
		}
		seen[tt.name] = true
		t.Run(tt.name, func(t *testing.T) {
			if got := len(LookupExternalLexStates(tt.name)); got == 0 {
				t.Fatalf("LookupExternalLexStates(%q) returned no rows", tt.name)
			}
			lang := tt.load()
			if lang == nil {
				t.Fatal("language loader returned nil")
			}
			if lang.ExternalScanner == nil {
				t.Fatal("ExternalScanner is nil")
			}
			if len(lang.ExternalSymbols) == 0 {
				t.Fatal("ExternalSymbols is empty")
			}
			if len(lang.ExternalLexStates) == 0 {
				t.Fatal("ExternalLexStates is empty")
			}
			diag := gotreesitter.DiagnoseCRecoveryGate(lang)
			if !diag.Supported {
				t.Fatalf("DiagnoseCRecoveryGate rejected language: %s", diag.Reason)
			}
			if !lang.CRecoveryCostCompetitionCapable || !lang.CRecoveryCostCompetitionEnabledByDefault {
				t.Fatalf("C recovery election not default-enabled: capable=%v default=%v",
					lang.CRecoveryCostCompetitionCapable, lang.CRecoveryCostCompetitionEnabledByDefault)
			}
		})
	}
	for name := range ledgerDefaults {
		if !seen[name] {
			t.Fatalf("%q is default_elected in ledger but missing from test cases", name)
		}
	}
}

func readDefaultElectionLedger(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile("../cgo_harness/tier_scan/external_lex_elections.json")
	if err != nil {
		t.Fatalf("read external lex election ledger: %v", err)
	}
	var doc struct {
		Grammars []struct {
			Grammar string `json:"grammar"`
			Status  string `json:"status"`
		} `json:"grammars"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode external lex election ledger: %v", err)
	}
	out := make(map[string]bool)
	for _, row := range doc.Grammars {
		if row.Status == "default_elected" {
			out[row.Grammar] = true
		}
	}
	return out
}
