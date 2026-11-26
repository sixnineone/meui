package graphics

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/golang/freetype/truetype"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/spacing"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/mobilegles"
)

// FontImpl is our own Font type. It holds font metadata and a rendering
// backend (a *mobilegles.Font) that is used for actual drawing.
type FontImpl struct {
	family  string
	size    float32 // requested size in cap-height logical px (unison semantics)
	weight  weight.Enum
	spacing spacing.Enum
	slant   slant.Enum
	face    *truetype.Font
	gl      *mobilegles.Font
	em      float32 // fitted rasterization size (em px), lazy

	// syntheticBold/SyntheticItalic: the face lacks a real bold/italic
	// companion, so the atlas emboldens/shears the glyphs at raster time.
	syntheticBold   bool
	syntheticItalic bool
}

// fontFitCache caches em fitting per face/size (unison fontSizeCache parity).
var fontFitCache sync.Map

// emPx returns the rasterization em size for this font: the unison
// FontFace.Font fitting loop, where the requested size is the CAP HEIGHT
// in logical pixels. Unison iterates the underlying size until the font's
// cap height matches the request; we solve the same loop against the
// truetype metrics.
func (f *FontImpl) emPx() float32 {
	if f == nil {
		return 0
	}
	if f.em > 0 {
		return f.em
	}
	tf := f.face
	if tf == nil {
		tf = mobilegles.SystemTTF()
	}
	requested := f.size
	if requested <= 0 {
		requested = 1
	}
	key := fmt.Sprintf("%p:%.3f", tf, requested)
	if v, ok := fontFitCache.Load(key); ok {
		f.em = v.(float32)
		return f.em
	}
	em := requested
	if tf != nil {
		cap0 := mobilegles.CapHeightPx(tf, requested)
		if cap0 > 0 {
			em = float32(math.Floor(float64(requested) * float64(requested) / float64(cap0)))
			for mobilegles.CapHeightPx(tf, em) < requested {
				em++
			}
			for em >= 1 && mobilegles.CapHeightPx(tf, em) > requested {
				em -= 0.5
			}
		}
	}
	f.em = em
	fontFitCache.Store(key, em)
	return em
}

// descentPx returns the descent in px at the fitted em size.
func (f *FontImpl) descentPx() float32 {
	tf := f.face
	if tf == nil {
		tf = mobilegles.SystemTTF()
	}
	return mobilegles.DescentPx(tf, f.emPx())
}

// Font is an alias for *FontImpl. It is shaped as a pointer so a zero
// value of Font is a nil pointer (matching unison's Font interface
// zero-value semantics) and `Font: nil` works in struct literals.
type Font = *FontImpl

func (f *FontImpl) Size() float32 {
	if f == nil {
		return 0
	}
	return f.size
}

func (f *FontImpl) Family() string {
	if f == nil {
		return ""
	}
	return f.family
}

// Font returns the underlying *truetype.Font. Matches the
// unison.Font interface signature so meui code that does
// `meuishim.MonospacedFont.Font.Face()` works as if it were the
// truetype face. nil if no truetype face is available.
func (f *FontImpl) Font() *truetype.Font {
	if f == nil {
		return nil
	}
	return f.face
}

// Face returns a FontFace descriptor for this font. Used to re-create
// the font at a different size. Carries the resolved truetype face so
// re-created fonts rasterize with the same family (unison parity —
// dropping it silently swapped the family for the system font).
func (f *FontImpl) Face() *FontFace {
	if f == nil {
		return nil
	}
	return &FontFace{
		family: f.family, Weight: f.weight, Spacing: f.spacing, Slant: f.slant, face: f.face,
		SyntheticBold: f.syntheticBold, SyntheticItalic: f.syntheticItalic,
	}
}

// TruetypeFace returns the underlying truetype font face, or nil.
func (f *FontImpl) TruetypeFace() *truetype.Font {
	if f == nil {
		return nil
	}
	return f.face
}

// FontFace returns a descriptor for this font. Used for re-creating
// the font at a different size.
func (f *FontImpl) FontFace() *FontFace {
	if f == nil {
		return nil
	}
	return &FontFace{family: f.family}
}

func (f *FontImpl) Baseline() float32 {
	if f == nil {
		return 0
	}
	// Unison parity: baseline = descent (real font) + requested size.
	if d := f.descentPx(); d > 0 {
		return d + f.size
	}
	return f.size * 0.96
}

func (f *FontImpl) LineHeight() float32 {
	if f == nil {
		return 0
	}
	// Unison parity: line height = requested size + descent*2.
	if d := f.descentPx(); d > 0 {
		return f.size + d*2
	}
	return f.size * 1.2
}

func (f *FontImpl) TextWidth(text string) float32 {
	if f == nil {
		return 0
	}
	if f.gl == nil {
		// Lazily build the engine font — measuring through the SAME
		// atlas the draw uses (the len*size*0.6 fallback here was the
		// span-overlap bug: styled runs measured ~20% short).
		f.renderFont()
	}
	if f.gl != nil {
		return f.gl.TextWidth(text)
	}
	return float32(len(text)) * f.size * 0.6
}

// SimpleWidth is an alias for TextWidth.
func (f *FontImpl) SimpleWidth(text string) float32 {
	return f.TextWidth(text)
}

func (f *FontImpl) Descriptor() FontDescriptor {
	if f == nil {
		return FontDescriptor{}
	}
	return FontDescriptor{
		Family:  f.family,
		Weight:  f.weight,
		Spacing: f.spacing,
		Slant:   f.slant,
		Size:    f.size,
		Face:    f.face,
	}
}

// renderFont returns the underlying mobilegles.Font for use by
// Canvas drawing methods. Creates it lazily.
func (f *FontImpl) renderFont() *mobilegles.Font {
	if f == nil {
		return nil
	}
	if f.gl == nil {
		// Always ensure the system font is loaded: it doubles as the
		// atlas symbol-fallback face (✓ ☑ ⚠ live in DejaVu, not Roboto).
		ensureSystemTTF()
		if f.face == nil {
			f.face = mobilegles.SystemTTF()
		}
		f.gl = mobilegles.NewFontStyled(f.family, f.emPx(), f.face, f.syntheticBold, f.syntheticItalic)
	}
	return f.gl
}

// FontDescriptor describes a font by family, weight, spacing, slant,
// and size. Calling .Font() produces a Font at the given size, with
// the requested weight/spacing/slant baked in.
type FontDescriptor struct {
	Family  string
	Weight  weight.Enum
	Spacing spacing.Enum
	Slant   slant.Enum
	Size    float32
	Face    *truetype.Font // concrete face; nil = resolve by family
}

// Font returns a Font of the given size based on this descriptor.
func (fd *FontDescriptor) Font() Font {
	if fd == nil {
		return nil
	}
	f := &FontImpl{
		family:  fd.Family,
		size:    fd.Size,
		weight:  fd.Weight,
		spacing: fd.Spacing,
		slant:   fd.Slant,
		face:    fd.Face,
	}
	// Faces that can't really provide the requested style (custom
	// loaded fonts have no bold/italic companions) render SYNTHETIC
	// bold/italic in the atlas — same as Skia's fallback synthesis.
	if fd.Face != nil {
		sub := fd.Face.Name(truetype.NameIDFontSubfamily)
		wantBold := fd.Weight >= weight.SemiBold
		wantItalic := fd.Slant == slant.Italic
		f.syntheticBold = wantBold && !strings.Contains(sub, "Bold")
		f.syntheticItalic = wantItalic && !strings.Contains(sub, "Italic")
	}
	return f
}

// Synthetic reports whether the face lacks real bold/italic and the
// atlas synthesizes them (raster embolden/shear).
func (f *FontImpl) Synthetic() (bold, italic bool) {
	if f == nil {
		return false, false
	}
	return f.syntheticBold, f.syntheticItalic
}

// FontFace is our own FontFace type. It carries a descriptor; calling
// Font(size) produces a Font at the given size. Family is exposed
// via the Family() method, not a field, to match unison's FontFace
// API. face, when non-nil, is the concrete truetype font the family
// resolved to (embedded Roboto faces); nil means "use the system font".
type FontFace struct {
	family  string
	Weight  weight.Enum
	Spacing spacing.Enum
	Slant   slant.Enum
	face    *truetype.Font

	// Synthetic styles for faces without real bold/italic companions
	// (rasterized emboldened/sheared — Skia-style synthesis).
	SyntheticBold   bool
	SyntheticItalic bool
}

// Font returns a Font of the given size based on this descriptor.
func (ff *FontFace) Font(size float32) Font {
	if ff == nil {
		return nil
	}
	return &FontImpl{
		family: ff.family, size: size, face: ff.face, weight: ff.Weight, slant: ff.Slant,
		syntheticBold: ff.SyntheticBold, syntheticItalic: ff.SyntheticItalic,
	}
}

// Family returns the family name. Method form (mirrors unison.FontFace.Family()).
func (ff *FontFace) Family() string {
	if ff == nil {
		return ""
	}
	return ff.family
}

// Style returns the (weight, spacing, slant) triple. Method form
// (mirrors unison.FontFace.Style()).
func (ff *FontFace) Style() (weight.Enum, spacing.Enum, slant.Enum) {
	if ff == nil {
		return 0, 0, 0
	}
	return ff.Weight, ff.Spacing, ff.Slant
}

// Face returns this descriptor as a *FontFace. Used in code that
// iterates over `[]FontFaceDescriptor` and rebuilds fonts at a new
// size: `for _, fd := range all { f := fd.Face().Font(size) }`.
func (ff *FontFace) Face() *FontFace {
	if ff == nil {
		return nil
	}
	return ff
}

// robotoStyleName maps weight/slant enums to the embedded file style
// suffix.
func robotoStyleName(w weight.Enum, sl slant.Enum) string {
	base := "Regular"
	switch w {
	case weight.Thin:
		base = "Thin"
	case weight.ExtraLight:
		base = "ExtraLight"
	case weight.Light:
		base = "Light"
	case weight.Medium:
		base = "Medium"
	case weight.SemiBold:
		base = "SemiBold"
	case weight.Bold, weight.ExtraBold, weight.Black:
		// Black/ExtraBold fall back to Bold (closest embedded face).
		base = "Bold"
	}
	if sl == slant.Italic {
		return base + " Italic"
	}
	return base
}

// LoadFontFace loads a truetype/otf font from path and returns a
// FontFace for it — lets applications ship their own UI fonts
// (e.g. fonts/InterDisplay-Regular.otf) instead of falling through
// the system-font candidate list.
func LoadFontFace(path string) (*FontFace, error) {
	f, err := mobilegles.LoadSystemFont(path)
	if err != nil {
		return nil, err
	}
	family := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return &FontFace{family: family, face: f}, nil
}

// WeightEnum maps a CSS-style numeric weight (100..900) to the face-weight enum.
// It is the ONE mapping: the QML `font.weight` handler and the theme presets both
// call it, so a 600 cannot mean "semibold" in a scene and "bold" in a preset.
func WeightEnum(w int) weight.Enum {
	switch {
	case w <= 250:
		return weight.Thin
	case w <= 350:
		return weight.Light
	case w <= 450:
		return weight.Regular
	case w <= 550:
		return weight.Medium
	case w <= 650:
		return weight.SemiBold
	default:
		return weight.Bold
	}
}

// registeredFonts holds app-supplied families: a face per weight and slant. It exists
// so a SCENE can ship a display face (`font.family: "Editorial"`) without the toolkit
// embedding it, and without MatchFontFace's "only Roboto resolves" rule changing for
// families nobody registered.
var (
	fontFamiliesMu sync.RWMutex
	fontFamilies   = map[string]map[fontFaceKey]*truetype.Font{}
)

type fontFaceKey struct {
	w  weight.Enum
	sl slant.Enum
}

// RegisterFontFamily parses a TTF and registers it under family at a weight and slant.
// An empty family uses the font's own name table. OTF is not accepted: the parser this
// tree uses cannot read it (that is a separate gap, recorded in WEB.md).
func RegisterFontFamily(family, path string, w weight.Enum, sl slant.Enum) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	face, err := truetype.Parse(data)
	if err != nil {
		return err
	}
	if family == "" {
		family = face.Name(truetype.NameIDFontFamily)
	}
	if family == "" {
		return fmt.Errorf("graphics: %s has no family name", path)
	}
	fontFamiliesMu.Lock()
	defer fontFamiliesMu.Unlock()
	if fontFamilies[family] == nil {
		fontFamilies[family] = map[fontFaceKey]*truetype.Font{}
	}
	fontFamilies[family][fontFaceKey{w, sl}] = face
	return nil
}

// RegisteredFamilies lists the app-supplied families, sorted.
func RegisteredFamilies() []string {
	fontFamiliesMu.RLock()
	defer fontFamiliesMu.RUnlock()
	names := make([]string, 0, len(fontFamilies))
	for n := range fontFamilies {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// lookupRegisteredFamily resolves a registered face, preferring an exact match, then
// the same slant, then upright, then whatever the family has.
func lookupRegisteredFamily(family string, w weight.Enum, sl slant.Enum) *truetype.Font {
	fontFamiliesMu.RLock()
	defer fontFamiliesMu.RUnlock()
	faces := fontFamilies[family]
	if len(faces) == 0 {
		return nil
	}
	if f := faces[fontFaceKey{w, sl}]; f != nil {
		return f
	}
	if f := faces[fontFaceKey{weight.Regular, sl}]; f != nil {
		return f
	}
	if f := faces[fontFaceKey{w, slant.Upright}]; f != nil {
		return f
	}
	if f := faces[fontFaceKey{weight.Regular, slant.Upright}]; f != nil {
		return f
	}
	for _, f := range faces {
		return f
	}
	return nil
}

// MatchFontFace resolves a family/style to a concrete font face.
// Unison parity: only the embedded Roboto family resolves (unison
// embeds Roboto and, unlike skia's font manager here, we do not fall
// back to arbitrary system fonts); unknown families return nil so
// callers fall back exactly like production.
func MatchFontFace(family string, w weight.Enum, sp spacing.Enum, sl slant.Enum) *FontFace {
	// Unison parity: the embedded Roboto family resolves; an APP-SUPPLIED family
	// resolves if it was registered. Unknown families return nil so callers fall back
	// exactly like production.
	if family == "Roboto" {
		return &FontFace{family: family, Weight: w, Spacing: sp, Slant: sl, face: mobilegles.EmbeddedRobotoFace(robotoStyleName(w, sl))}
	}
	if face := lookupRegisteredFamily(family, w, sl); face != nil {
		return &FontFace{family: family, Weight: w, Spacing: sp, Slant: sl, face: face}
	}
	return nil
}

func AllFontFaces() (all, monospaced []FontFaceDescriptor) {
	return nil, nil
}

// FontFaceDescriptor is a placeholder for compatibility.
type FontFaceDescriptor = FontFace

// SystemFont is pre-allocated to a Font wrapping mobilegles.SystemFont().
// It may be nil if no system font is loaded.
var SystemFont Font = wrapSystemFont()

// ReloadSystemFont re-resolves SystemFont (and MonospacedFont) from the platform's
// CURRENT system font.
//
// SystemFont is resolved once, at package init, and a widget captures the face it
// was built with — so installing a font later changed the underlying face without
// changing anything an app could see. That made LoadFontFace, which is documented
// as the way to ship your own UI font, silently ineffective: text already laid out
// kept the old glyphs and the theme kept pointing at them, with no error anywhere.
//
// Changing a font at runtime needs both halves: this re-resolves the font, and
// theme.UseSystemFont makes the theme use it. Neither package can do the other's
// half, because theme imports graphics and not the reverse.
func ReloadSystemFont() Font {
	SystemFont = wrapSystemFont()
	MonospacedFont = SystemFont
	return SystemFont
}

// MonospacedFont falls back to SystemFont; mobilegles has no
// monospace distinction.
var MonospacedFont = SystemFont

// systemFontCandidates are probed in order when the app has not loaded
// a font explicitly. Mobile backends replace this with an embedded font.
var systemFontCandidates = []string{
	"fonts/InterDisplay-Regular.otf",
	// Android: /system/fonts is the platform's own font directory, and
	// Roboto-Regular.ttf is what its fonts.xml calls the sans-serif default.
	// Without an entry here the toolkit finds no font on a device and every
	// glyph is invisible — a silent failure, because a nil font still lays out.
	"/system/fonts/Roboto-Regular.ttf",
	"/system/fonts/RobotoStatic-Regular.ttf",
	"/system/fonts/DroidSans.ttf",
	"/system/fonts/NotoSans-Regular.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular_0.ttf",
	"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
	"/usr/share/fonts/TTF/DejaVuSans.ttf",
	"/Library/Fonts/Arial.ttf",
	"C:/Windows/Fonts/arial.ttf",
}

var systemFontOnce sync.Once

// preferredSystemFamilies are the families a UI base font should come from,
// in order. They are the default or metric-compatible sans faces of the common
// distributions; taking whichever family sorts first would hand the app a
// monospace (Adwaita Mono is first on openSUSE) or a math/T X face, which is a
// worse default than any of these.
var preferredSystemFamilies = []string{
	"DejaVu Sans",     // Debian, Ubuntu, openSUSE, Fedora
	"Carlito",         // metric-compatible with Calibri
	"Liberation Sans", // metric-compatible with Arial
	"Noto Sans",
	"Ubuntu",
	"Adwaita Sans",
}

// ensureSystemTTF lazily loads the first available system font so meui
// apps get text without explicit font setup (parity with unison, which
// resolves system fonts internally).
func ensureSystemTTF() {
	systemFontOnce.Do(func() {
		if mobilegles.SystemTTF() != nil {
			return
		}
		for _, path := range systemFontCandidates {
			if _, err := os.Stat(path); err != nil {
				continue
			}
			// LoadSystemFont (not LoadTTFFromFile) installs the global
			// that SystemFont()/SystemTTF() read.
			if _, err := mobilegles.LoadSystemFont(path); err == nil {
				return
			}
		}
		// Nothing in that list matched. The list is one distribution's layout
		// (Debian's per-family directories, Android's /system/fonts, macOS,
		// Windows), and its first entry is an OTF that this parser cannot read
		// at all. On any other layout — openSUSE and Fedora put DejaVuSans.ttf
		// directly in /usr/share/fonts/truetype — the process was left with NO
		// face, and painting a label then dereferenced a nil *truetype.Font.
		discoverSystemFont()
	})
}

// discoverSystemFont loads a face found by scanning the standard font
// directories, for a layout the candidate list does not know. Returns whether a
// face was installed.
func discoverSystemFont() bool {
	info, ok := pickSystemFont(ListSystemFonts())
	if !ok {
		return false
	}
	_, err := mobilegles.LoadSystemFont(info.Path)
	return err == nil
}

// systemFontFace returns the face the toolkit draws text with: the system font
// it discovered, or the face embedded in this binary when none could be found.
// It is never nil for a normal build, which is what the software device needs —
// draw2d falls back to a folder lookup that exists only in the module source tree
// and logs a miss for every string.
func systemFontFace() *truetype.Font {
	ensureSystemTTF()
	if f := mobilegles.SystemTTF(); f != nil {
		return f
	}
	return mobilegles.EmbeddedRobotoFace("Regular")
}

// pickSystemFont chooses the face discovery should load: a preferred family in
// preference order, otherwise the first family the scan returned.
func pickSystemFont(fonts []SystemFontInfo) (SystemFontInfo, bool) {
	if len(fonts) == 0 {
		return SystemFontInfo{}, false
	}
	for _, want := range preferredSystemFamilies {
		for _, f := range fonts {
			if strings.EqualFold(f.Family, want) {
				return f, true
			}
		}
	}
	return fonts[0], true
}

func wrapSystemFont() *FontImpl {
	ensureSystemTTF()
	gl := mobilegles.SystemFont()
	if gl == nil {
		return &FontImpl{family: "system", size: 12}
	}
	return &FontImpl{family: gl.Family(), size: gl.Size(), face: gl.Face(), gl: gl}
}

// SystemFontInfo describes one discovered outline font family.
type SystemFontInfo struct {
	Family string // family name from the font's name table
	Path   string // font file to load with LoadFontFace
}

var (
	listFontsOnce  sync.Once
	listFontsCache []SystemFontInfo
)

// ListSystemFonts scans the standard font directories (plus any extra
// dirs, e.g. an app's bundled fonts/) and returns one entry per family.
// When several faces share a family the "Regular" file wins. Family
// names come from each font's name table; unparseable files are
// skipped. The result is scanned once and cached — the first call
// reads every font file, so call it before the UI is shown.
func ListSystemFonts(extraDirs ...string) []SystemFontInfo {
	listFontsOnce.Do(func() {
		dirs := append([]string{}, extraDirs...)
		dirs = append(dirs,
			"/usr/share/fonts",
			"/usr/local/share/fonts",
		)
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs,
				filepath.Join(home, ".fonts"),
				filepath.Join(home, ".local", "share", "fonts"),
			)
		}
		byFamily := map[string]SystemFontInfo{}
		rank := map[string]int{}
		for _, dir := range dirs {
			_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d == nil || d.IsDir() {
					return nil
				}
				ext := strings.ToLower(filepath.Ext(path))
				if ext != ".ttf" && ext != ".otf" {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				f, err := truetype.Parse(data)
				if err != nil {
					return nil
				}
				family := f.Name(truetype.NameIDFontFamily)
				if family == "" {
					family = strings.TrimSuffix(filepath.Base(path), ext)
				}
				s := 1 // non-regular face
				if strings.Contains(strings.ToLower(path), "regular") {
					s = 0
				}
				if _, ok := byFamily[family]; !ok || s < rank[family] {
					byFamily[family] = SystemFontInfo{Family: family, Path: path}
					rank[family] = s
				}
				return nil
			})
		}
		out := make([]SystemFontInfo, 0, len(byFamily))
		for _, info := range byFamily {
			out = append(out, info)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Family < out[j].Family })
		listFontsCache = out
	})
	return listFontsCache
}
