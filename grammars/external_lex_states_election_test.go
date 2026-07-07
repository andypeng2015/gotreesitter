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
		{name: "arduino", load: ArduinoLanguage},
		{name: "astro", load: AstroLanguage},
		{name: "awk", load: AwkLanguage},
		{name: "bash", load: BashLanguage},
		{name: "beancount", load: BeancountLanguage},
		{name: "bicep", load: BicepLanguage},
		{name: "bitbake", load: BitbakeLanguage},
		{name: "blade", load: BladeLanguage},
		{name: "c_sharp", load: CSharpLanguage},
		{name: "caddy", load: CaddyLanguage},
		{name: "cairo", load: CairoLanguage},
		{name: "cmake", load: CmakeLanguage},
		{name: "cooklang", load: CooklangLanguage},
		{name: "cpp", load: CppLanguage},
		{name: "css", load: CssLanguage},
		{name: "cue", load: CueLanguage},
		{name: "cuda", load: CudaLanguage},
		{name: "d", load: DLanguage},
		{name: "dart", load: DartLanguage},
		{name: "disassembly", load: DisassemblyLanguage},
		{name: "dockerfile", load: DockerfileLanguage},
		{name: "dtd", load: DtdLanguage},
		{name: "earthfile", load: EarthfileLanguage},
		{name: "editorconfig", load: EditorconfigLanguage},
		{name: "elm", load: ElmLanguage},
		{name: "fennel", load: FennelLanguage},
		{name: "firrtl", load: FirrtlLanguage},
		{name: "foam", load: FoamLanguage},
		{name: "fortran", load: FortranLanguage},
		{name: "gdscript", load: GdscriptLanguage},
		{name: "gitcommit", load: GitcommitLanguage},
		{name: "gleam", load: GleamLanguage},
		{name: "gn", load: GnLanguage},
		{name: "hack", load: HackLanguage},
		{name: "haxe", load: HaxeLanguage},
		{name: "hlsl", load: HlslLanguage},
		{name: "html", load: HtmlLanguage},
		{name: "janet", load: JanetLanguage},
		{name: "jsdoc", load: JsdocLanguage},
		{name: "jsonnet", load: JsonnetLanguage},
		{name: "just", load: JustLanguage},
		{name: "kconfig", load: KconfigLanguage},
		{name: "kdl", load: KdlLanguage},
		{name: "kotlin", load: KotlinLanguage},
		{name: "less", load: LessLanguage},
		{name: "liquid", load: LiquidLanguage},
		{name: "lua", load: LuaLanguage},
		{name: "luau", load: LuauLanguage},
		{name: "matlab", load: MatlabLanguage},
		{name: "mojo", load: MojoLanguage},
		{name: "move", load: MoveLanguage},
		{name: "nickel", load: NickelLanguage},
		{name: "nim", load: NimLanguage},
		{name: "nushell", load: NushellLanguage},
		{name: "odin", load: OdinLanguage},
		{name: "org", load: OrgLanguage},
		{name: "php", load: PhpLanguage},
		{name: "pkl", load: PklLanguage},
		{name: "powershell", load: PowershellLanguage},
		{name: "pug", load: PugLanguage},
		{name: "purescript", load: PurescriptLanguage},
		{name: "python", load: PythonLanguage},
		{name: "r", load: RLanguage},
		{name: "rescript", load: RescriptLanguage},
		{name: "ron", load: RonLanguage},
		{name: "ruby", load: RubyLanguage},
		{name: "rust", load: RustLanguage},
		{name: "scala", load: ScalaLanguage},
		{name: "scss", load: ScssLanguage},
		{name: "sql", load: SqlLanguage},
		{name: "squirrel", load: SquirrelLanguage},
		{name: "starlark", load: StarlarkLanguage},
		{name: "svelte", load: SvelteLanguage},
		{name: "tablegen", load: TablegenLanguage},
		{name: "tcl", load: TclLanguage},
		{name: "teal", load: TealLanguage},
		{name: "templ", load: TemplLanguage},
		{name: "tsx", load: TsxLanguage},
		{name: "typst", load: TypstLanguage},
		{name: "uxntal", load: UxntalLanguage},
		{name: "vhdl", load: VhdlLanguage},
		{name: "vue", load: VueLanguage},
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
