package meui

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
)

//go:embed assets/icons/*.svg
var testSvgEmbedFS embed.FS

func TestLoadAllIconSVGs(t *testing.T) {
	// Verify the embedded filesystem lists the icons we expect.
	entries, err := testSvgEmbedFS.ReadDir("assets/icons")
	if err != nil {
		t.Fatalf("read embed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no icons embedded under assets/icons")
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".svg")

		doc, err := LoadSVGByName(name)
		if err != nil {
			t.Errorf("LoadSVGByName(%q): %v", name, err)
			continue
		}
		if doc.Width <= 0 || doc.Height <= 0 {
			t.Errorf("LoadSVGByName(%q): width/height not parsed (got %v x %v)", name, doc.Width, doc.Height)
		}
		if len(doc.Elements) == 0 {
			t.Errorf("LoadSVGByName(%q): no elements parsed", name)
		}

		// Verify buildSvgPath works on every <path> element so that
		// the parser doesn't crash on smooth-cubic (S) commands.
		for _, el := range doc.Elements {
			if el.Type != "path" {
				continue
			}
			if buildSvgPath(el.Attrs["d"], 0, 0, 1) == nil && el.Attrs["d"] != "" {
				t.Errorf("%s: buildSvgPath returned nil for non-empty d=%q", name, el.Attrs["d"])
			}
		}
	}
}

func TestBuildSvgPathCommands(t *testing.T) {
	cases := []struct {
		name string
		d    string
	}{
		{"MoveLine", "M0 0 L10 10"},
		{"RelativeCubic", "M0 0 c0 1 1 2 2 2"},
		{"SmoothCubicAbsolute", "M0 0 C1 1 2 2 3 3 S4 4 5 5"},
		{"SmoothCubicRelative", "M0 0 c0 1 1 2 2 2 s1 1 2 2"},
		{"SmoothQuadratic", "M0 0 Q1 1 2 2 T4 4"},
		{"Close", "M0 0 L10 0 L10 10 Z"},
		{"HorVer", "M0 0 H10 V10"},
		{"Arc", "M0 0 A2 2 0 0 1 10 0"},
		{"EyeOutline", "M8 3C5.2 3 3 5.2 3 8s2.2 5 5 5 5-2.2 5-5-2.2-5-5-5z"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if buildSvgPath(c.d, 0, 0, 1) == nil {
				t.Fatalf("buildSvgPath(%q) returned nil", c.d)
			}
		})
	}
}

func TestBuildSvgPathEmpty(t *testing.T) {
	if buildSvgPath("", 0, 0, 1) != nil {
		t.Fatal("empty d should return nil path")
	}
}

// TestParsePathValuesNegativeNumbers guards against a regression where
// `0-2.8 2.2-5 5-5` was being read as one number ("0-2.8 2.2-5 5-5")
// which the parser then dropped to 0. The SVG spec allows a sign
// immediately after a digit (as a new number's sign), so this must
// tokenize to [0, -2.8, 2.2, -5, 5, -5].
func TestParsePathValuesNegativeNumbers(t *testing.T) {
	cases := []struct {
		name string
		d    string
		want []float64
	}{
		{
			name: "user_icon_body",
			d:    "M3 14c0-2.8 2.2-5 5-5s5 2.2 5 5",
			want: []float64{3, 14, 0, -2.8, 2.2, -5, 5, -5, 5, 2.2, 5, 5},
		},
		{
			name: "eye_outer_arc",
			d:    "s2.2 5 5 5 5-2.2 5-5-2.2-5-5-5",
			want: []float64{2.2, 5, 5, 5, 5, -2.2, 5, -5, -2.2, -5, -5, -5},
		},
		{
			name: "exponent_sign",
			d:    "M0 0 L1e-5 1e+5",
			want: []float64{0, 0, 1e-5, 1e5},
		},
		{
			name: "leading_sign",
			d:    "M-1 -2 L+3 +4",
			want: []float64{-1, -2, 3, 4},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parsePathValues(c.d)
			gotNums := make([]float64, 0, len(got))
			for _, v := range got {
				if n, ok := v.(float64); ok {
					gotNums = append(gotNums, n)
				}
			}
			if len(gotNums) != len(c.want) {
				t.Fatalf("got %d numbers %v, want %d numbers %v", len(gotNums), gotNums, len(c.want), c.want)
			}
			for i := range c.want {
				if diff := gotNums[i] - c.want[i]; diff > 1e-4 || diff < -1e-4 {
					t.Errorf("[%d] got %v, want %v", i, gotNums[i], c.want[i])
				}
			}
		})
	}
}

func TestLoadSVGNotFound(t *testing.T) {
	// Use a name that definitely isn't in the embed or any search path.
	if _, err := LoadSVGByName("__definitely_not_a_real_icon__"); err == nil {
		t.Fatal("expected error for missing icon")
	}
}

// Sanity check the file-system fallback path used by LoadSVG/LoadSVGByName.
func TestLoadSVGViaFilePath(t *testing.T) {
	wd, _ := os.Getwd()
	dir := filepath.Join(wd, "assets", "icons")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("assets/icons not available at %s: %v", dir, err)
	}
	doc, err := LoadSVG(filepath.Join(dir, "check.svg"))
	if err != nil {
		t.Fatalf("LoadSVG: %v", err)
	}
	if doc.Width <= 0 {
		t.Errorf("expected width > 0, got %v", doc.Width)
	}
}

// DrawSVG must not drop an icon when the backend refuses the supersampled
// offscreen raster (the app runs with MEUI_NO_OFFSCREEN=1). The old behaviour
// skipped the icon and retried - and logged - on every frame; this asserts the
// direct fallback paints pixels and that the decision is cached.
func TestDrawSVGFallsBackWhenOffscreenRefused(t *testing.T) {
	t.Setenv("MEUI_NO_OFFSCREEN", "1")

	doc, err := LoadSVGByName("arrow_up")
	if err != nil || doc == nil || doc.Width <= 0 {
		t.Fatalf("load arrow_up: doc=%v err=%v", doc, err)
	}
	if svgDirect(doc.Elements) {
		t.Fatal("precondition: arrow_up must take the supersampled path, not svgDirect")
	}

	key := svgImgKey{doc: doc, w: 16, h: 16}
	delete(svgNoOffscreen, key)
	delete(svgImageCache, key)

	dev := graphics.NewSoftDevice(16, 16)
	c := graphics.NewCanvas(dev)
	DrawSVG(c, doc, 0, 0, 16, 16)

	drawn := 0
	for _, px := range dev.Pixels() {
		if px != 0 {
			drawn++
		}
	}
	if drawn == 0 {
		t.Fatal("DrawSVG drew nothing when the offscreen raster was refused")
	}
	if !svgNoOffscreen[key] {
		t.Fatal("fallback decision not cached: the offscreen attempt would retry every frame")
	}
}

func TestLoadSVGByNameRefusesPaths(t *testing.T) {
	for _, name := range []string{"../secret", "../../etc/passwd", "a/b", `a\\b`, "..", ".hidden"} {
		if _, err := LoadSVGByName(name); err == nil {
			t.Errorf("LoadSVGByName(%q) accepted a path-like icon name", name)
		}
	}
}
