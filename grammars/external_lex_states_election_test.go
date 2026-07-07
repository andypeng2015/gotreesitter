package grammars

import (
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
		{name: "janet", load: JanetLanguage},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(LookupExternalLexStates(tt.name)); got == 0 {
				t.Fatalf("LookupExternalLexStates(%q) returned no rows", tt.name)
			}
			lang := tt.load()
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
}
