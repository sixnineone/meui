package theme

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

var exportTestSeq int

// uniqueExportID keeps registrations distinct even under -count>1, since
// RegisterPreset rejects duplicate ids.
func uniqueExportID(base string) string {
	exportTestSeq++
	return fmt.Sprintf("%s-export-%d", base, exportTestSeq)
}

// Every registered preset must survive export -> load unchanged. This is
// the core promise of the format: what SaveThemeFile writes is what the
// loader reads back.
func TestThemeFileExportRoundTrip(t *testing.T) {
	for _, preset := range PresetIDs() {
		t.Run(preset, func(t *testing.T) {
			if err := LoadPreset(preset); err != nil {
				t.Fatalf("LoadPreset(%s): %v", preset, err)
			}
			src := DefaultTheme

			id := uniqueExportID(preset)
			src.Meta.ID = id
			src.Name = id

			data, err := ToThemeFile(src)
			if err != nil {
				t.Fatalf("ToThemeFile: %v", err)
			}
			gotID, err := LoadThemeFileData(data)
			if err != nil {
				t.Fatalf("LoadThemeFileData: %v", err)
			}
			if err := LoadPreset(gotID); err != nil {
				t.Fatalf("LoadPreset(%s): %v", gotID, err)
			}
			if diff := DiffThemes(src, DefaultTheme); len(diff) > 0 {
				t.Fatalf("round-trip differs on %d field(s):\n  %s", len(diff), strings.Join(diff, "\n  "))
			}
		})
	}
}

// The exported document must carry every palette key and every token
// group, not just the ones the older hand-written sections covered.
func TestThemeFileExportCoversPaletteAndTokens(t *testing.T) {
	if err := LoadPreset("windows11-dark"); err != nil {
		t.Fatal(err)
	}
	data, err := ToThemeFile(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		ID      string            `json:"id"`
		Base    string            `json:"base"`
		Palette map[string]string `json:"palette"`
		Tokens  map[string]any    `json:"tokens"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("exported JSON does not parse: %v", err)
	}
	if doc.ID == "" || doc.Base == "" {
		t.Fatalf("id/base missing (id=%q base=%q)", doc.ID, doc.Base)
	}

	for _, target := range paletteColorTargets(DefaultTheme) {
		if _, ok := doc.Palette[target.key]; !ok {
			t.Errorf("palette missing %q", target.key)
		}
	}

	for _, section := range []string{"meta", "metrics", "motion", "typography", "effects", "chrome", "inputs", "controls"} {
		group, ok := doc.Tokens[section].(map[string]any)
		if !ok || len(group) == 0 {
			t.Errorf("tokens.%s missing or empty", section)
		}
	}
	metrics, _ := doc.Tokens["metrics"].(map[string]any)
	if metrics["controlHeight"] == nil {
		t.Errorf("tokens.metrics.controlHeight missing: %#v", doc.Tokens["metrics"])
	}
	chrome, _ := doc.Tokens["chrome"].(map[string]any)
	if chrome["style"] == nil {
		t.Errorf("tokens.chrome.style missing: %#v", doc.Tokens["chrome"])
	}
	// Nested colours inside token groups are hex-encoded like palette ones.
	effects, _ := doc.Tokens["effects"].(map[string]any)
	shadow, _ := effects["windowShadow"].(map[string]any)
	if s, ok := shadow["color"].(string); !ok || !strings.HasPrefix(s, "#") {
		t.Errorf("tokens.effects.windowShadow.color = %#v, want a hex string", shadow["color"])
	}
}

// An explicit zero colour must survive: the lenient parser exists because
// parseHexColor rejects black/transparent, which are valid palette values.
func TestThemeColorRoundTripEdges(t *testing.T) {
	for _, c := range []graphics.Color{
		0x00000000, 0xFF000000, 0x80FF0000, 0xFFFFFFFF, 0x12AB34CD, 0x00000001,
	} {
		s := formatThemeColor(c)
		got, ok := parseThemeColor(s)
		if !ok || got != c {
			t.Errorf("color %08X -> %q -> %08X (ok=%v)", uint32(c), s, uint32(got), ok)
		}
	}
}

func TestDiffThemesDetectsChange(t *testing.T) {
	if err := LoadPreset("meui-dark"); err != nil {
		t.Fatal(err)
	}
	a := DefaultTheme
	if err := LoadPreset("windows11-dark"); err != nil {
		t.Fatal(err)
	}
	b := DefaultTheme

	d := DiffThemes(a, b)
	if len(d) == 0 {
		t.Fatal("expected meui-dark and windows11-dark to differ")
	}
	for i := 1; i < len(d); i++ {
		if d[i-1] > d[i] {
			t.Fatalf("diff not sorted at %d:\n  %q\n  %q", i, d[i-1], d[i])
		}
	}
	if self := DiffThemes(a, a); len(self) != 0 {
		t.Fatalf("self-diff should be empty, got %v", self)
	}
	// Palette and token paths both appear, so the editor can group them.
	var sawPalette, sawToken bool
	for _, line := range d {
		if strings.HasPrefix(line, "palette.") {
			sawPalette = true
		}
		if strings.HasPrefix(line, "tokens.") {
			sawToken = true
		}
	}
	if !sawPalette || !sawToken {
		t.Fatalf("expected both palette and token diffs (palette=%v token=%v)", sawPalette, sawToken)
	}
}

// A hand-authored delta file — no tokens section — must keep working
// exactly as before.
func TestThemeFileTokensAbsentStillAppliesDeltas(t *testing.T) {
	tf := &themeFile{
		ID:      uniqueExportID("delta-only"),
		Base:    "meui-dark",
		Palette: map[string]string{"accent": "#E62D42"},
		Metrics: &themeFileMetrics{ControlHeight: f32(40)},
	}
	id, err := registerThemeFile(tf)
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadPreset(id); err != nil {
		t.Fatal(err)
	}
	if DefaultTheme.Accent != 0xFFE62D42 {
		t.Errorf("accent = %08X, want FFE62D42", uint32(DefaultTheme.Accent))
	}
	if DefaultTheme.Metrics.ControlHeight != 40 {
		t.Errorf("controlHeight = %v, want 40", DefaultTheme.Metrics.ControlHeight)
	}
}

// ParseThemeFile must not touch the preset registry, so a tool can load
// the same file twice — or two revisions sharing an id — to compare them.
func TestParseThemeFileDoesNotRegister(t *testing.T) {
	if err := LoadPreset("macos-dark"); err != nil {
		t.Fatal(err)
	}
	data, err := ToThemeFile(DefaultTheme)
	if err != nil {
		t.Fatal(err)
	}

	first, err := ParseThemeFile(data)
	if err != nil {
		t.Fatalf("ParseThemeFile: %v", err)
	}
	second, err := ParseThemeFile(data)
	if err != nil {
		t.Fatalf("second ParseThemeFile (should not collide): %v", err)
	}
	if diff := DiffThemes(first, second); len(diff) != 0 {
		t.Fatalf("two parses of the same bytes differ: %v", diff)
	}
	if diff := DiffThemes(DefaultTheme, first); len(diff) != 0 {
		t.Fatalf("parsed theme differs from its source: %v", diff)
	}
	if _, err := ParseThemeFile([]byte(`{"id":"x","base":"no-such-base"}`)); err == nil {
		t.Error("unknown base accepted")
	}
}

// Widget themes are generated rather than authored, but the read-only
// snapshot must still cover every widget theme and encode colours/fonts usefully. The
// count moved from 29 to 30 when the Markdown renderer's palette became a theme — it used
// to carry VS Code's colours as literals, so a document ignored every preset.
func TestResolvedWidgetThemes(t *testing.T) {
	if err := LoadPreset("macos-dark"); err != nil {
		t.Fatal(err)
	}

	names := WidgetThemeNames()
	if len(names) != 30 {
		t.Fatalf("WidgetThemeNames returned %d entries, want 30: %v", len(names), names)
	}
	// The markdown palette is the one that had to be ADDED, so pin it by name: a count
	// alone would pass if a different theme were registered instead.
	found := false
	for _, n := range names {
		if n == "markdown" {
			found = true
		}
	}
	if !found {
		t.Error("the markdown theme is not exported; documents would ignore every preset")
	}
	all := ResolvedWidgetThemes()
	if len(all) != 30 {
		t.Fatalf("ResolvedWidgetThemes returned %d entries, want 30", len(all))
	}

	btn, ok := all["button"]
	if !ok {
		t.Fatal("no button entry")
	}
	if got, ok := btn["cornerRadius"].(float64); !ok || float32(got) != DefaultButtonTheme.CornerRadius {
		t.Errorf("button.cornerRadius = %#v, want %v", btn["cornerRadius"], DefaultButtonTheme.CornerRadius)
	}
	if ink, ok := btn["backgroundInk"].(string); !ok || !strings.HasPrefix(ink, "#") {
		t.Errorf("button.backgroundInk = %#v, want a hex colour", btn["backgroundInk"])
	}
	// The generator leaves widget Font fields nil — widgets fall back to
	// Theme.BaseFont at paint time — so the snapshot renders null here.
	if btn["font"] != nil {
		t.Errorf("button.font = %#v, want nil (the generator leaves it unset)", btn["font"])
	}

	// When a widget theme DOES carry a font it must encode as a
	// descriptor: FontImpl has no exported fields, so naive reflection
	// would render an empty object.
	if DefaultTheme.BaseFont == nil {
		t.Log("no BaseFont resolved in this environment; font descriptor path not exercised")
		return
	}
	saved := DefaultButtonTheme.Font
	defer func() { DefaultButtonTheme.Font = saved }()
	DefaultButtonTheme.Font = DefaultTheme.BaseFont
	btn = ResolvedWidgetThemes()["button"]
	font, ok := btn["font"].(map[string]any)
	if !ok || font["size"] == nil {
		t.Errorf("set button.font = %#v, want a {family,size} descriptor", btn["font"])
	}
}

func TestDiffWidgetThemes(t *testing.T) {
	if err := LoadPreset("meui-dark"); err != nil {
		t.Fatal(err)
	}
	a := ResolvedWidgetThemes()
	if err := LoadPreset("windows11-dark"); err != nil {
		t.Fatal(err)
	}
	b := ResolvedWidgetThemes()

	d := DiffWidgetThemes(a, b)
	if len(d) == 0 {
		t.Fatal("expected meui-dark and windows11-dark widget themes to differ")
	}
	for i := 1; i < len(d); i++ {
		if d[i-1] > d[i] {
			t.Fatalf("diff not sorted at %d: %q > %q", i, d[i-1], d[i])
		}
	}
	for _, line := range d {
		if !strings.HasPrefix(line, "widgets.") {
			t.Errorf("unexpected diff path %q", line)
		}
	}
	if self := DiffWidgetThemes(a, a); len(self) != 0 {
		t.Fatalf("self-diff should be empty, got %v", self)
	}
}
