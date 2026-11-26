package meui

// The Box: the properties a web-looking scene depends on and that Rectangle used
// to parse and throw away. These sample PIXELS, because the failure mode here is not
// a wrong field — it is a property that reports itself as set and paints nothing.

import (
	"image"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/theme"
)

func paintPanel(t *testing.T, p *Panel, w, h int) *image.RGBA {
	return paintPanelIn(t, p, w, h, w, h)
}

// paintPanelIn paints a box of boxW×boxH into a larger canvas, which is how a
// clip test can look OUTSIDE the box that is doing the clipping.
func paintPanelIn(t *testing.T, p *Panel, boxW, boxH, canvasW, canvasH int) *image.RGBA {
	t.Helper()
	theme.SetDefaultTheme(theme.DefaultTheme) // ensure a theme is installed
	canvas, dev := graphics.NewSoftwareCanvas(canvasW, canvasH)
	canvas.Clear(0x00000000)
	p.SetFrameRect(graphics.NewRect(0, 0, float32(boxW), float32(boxH)))
	p.Paint(canvas, graphics.NewRect(0, 0, float32(boxW), float32(boxH)))
	return dev.Image()
}

// TestABoxRoundsItsCorners is the one that started this: `Rectangle { radius: 8 }`
// on a web card came out square, because buildRectangle parsed the radius and
// dropped it, and Panel painted a bare rect.
func TestABoxRoundsItsCorners(t *testing.T) {
	p := NewPanel()
	p.SetBackground(graphics.Color(0xFFFFFFFF))
	p.SetCornerRadius(20)
	img := paintPanel(t, p, 100, 100)

	if a := img.RGBAAt(2, 2).A; a != 0 {
		t.Fatalf("corner (2,2) has alpha %d, want 0 — the radius was not applied", a)
	}
	if a := img.RGBAAt(50, 50).A; a != 0xFF {
		t.Fatalf("centre has alpha %d, want the fill", a)
	}
	if a := img.RGBAAt(50, 1).A; a != 0xFF {
		t.Fatalf("top edge midpoint has alpha %d, want the fill", a)
	}
}

// TestABoxRampsAGradient pins `Rectangle { gradient: {...} }`: a landing hero or a
// dashboard sky is a gradient, and a box that silently paints one stop looks like a
// colour choice rather than a bug.
func TestABoxRampsAGradient(t *testing.T) {
	p := NewPanel()
	g := graphics.Gradient{
		Kind:  graphics.GradientLinear,
		Stops: []graphics.GradientStop{{Offset: 0, Color: 0xFFFF0000}, {Offset: 1, Color: 0xFF0000FF}},
		Start: graphics.Point{X: 0.5, Y: 0},
		End:   graphics.Point{X: 0.5, Y: 1},
	}
	p.SetGradientUnit(g)
	img := paintPanel(t, p, 100, 100)

	top, bottom := img.RGBAAt(50, 2), img.RGBAAt(50, 97)
	if top.R < 0xC0 || top.B > 0x40 {
		t.Fatalf("top of the gradient is %v, want red", top)
	}
	if bottom.B < 0xC0 || bottom.R > 0x40 {
		t.Fatalf("bottom of the gradient is %v, want blue", bottom)
	}
}

// TestABoxClipsItsChildren pins `clip: true`: a card with a coloured header has to
// cut that header to its own rounded shape, or the header's square corners paint
// over the card's radius.
func TestABoxClipsItsChildren(t *testing.T) {
	p := NewPanel()
	p.SetBackground(graphics.Color(0xFFFFFFFF))
	p.SetCornerRadius(16)
	p.SetClipChildren(true)

	child := NewPanel()
	child.SetBackground(graphics.Color(0xFF00FF00))
	child.SetFrameRect(graphics.NewRect(50, 50, 200, 200)) // pokes out of the box
	p.AddChild(child)

	img := paintPanelIn(t, p, 100, 100, 200, 200)
	if a := img.RGBAAt(120, 60).A; a != 0 {
		t.Fatalf("a clipped child painted outside the box (alpha %d at 120,60, box ends at 100)", a)
	}
	if g := img.RGBAAt(80, 80).G; g < 0x80 {
		t.Fatalf("the child did not paint inside the box (green %d)", g)
	}
}

// TestABoxAppliesItsOpacity pins `opacity: 0.5` on a box: the blobs behind a landing
// page are translucent boxes, and opacity that only reached a field is invisible.
func TestABoxAppliesItsOpacity(t *testing.T) {
	p := NewPanel()
	p.SetBackground(graphics.Color(0xFF000000))
	p.SetOpacity(0.5)
	img := paintPanel(t, p, 40, 40)
	_, _, _, a := img.RGBAAt(20, 20).RGBA()
	got := int(a >> 8)
	if got < 120 || got > 136 {
		t.Fatalf("a 50%% black box over nothing has alpha %d, want ~128", got)
	}
}

// TestAClickBehaviourCancelsWhenTheReleaseIsOutside is the state machine both a
// clickable box and a MouseArea run.
func TestAClickBehaviourCancelsWhenTheReleaseIsOutside(t *testing.T) {
	var c ClickBehaviour
	clicked := 0
	c.OnClick = func() { clicked++ }

	if !c.Down(true, true, 0) || !c.Pressed() {
		t.Fatal("a press inside did not arm the behaviour")
	}
	if !c.Up(true, false, 0) {
		t.Fatal("a release did not end the press")
	}
	if clicked != 0 {
		t.Fatal("a release outside still clicked")
	}
	if !c.Down(true, true, 0) || !c.Up(true, true, 0) {
		t.Fatal("a press/release inside did not complete")
	}
	if clicked != 1 {
		t.Fatalf("a full click fired %d times, want 1", clicked)
	}
	if c.Down(false, true, 0) {
		t.Fatal("a disabled behaviour armed")
	}
	if c.Down(true, true, 1) {
		t.Fatal("a secondary button armed a click")
	}
}

// TestASelectedBoxPaintsTheSelectionInk pins `selected: true` on the Box: a nav item,
// a list row or a day cell marks itself chosen with the THEME's Selected ink, so a
// scene stops hand-picking a fill per selected control.
func TestASelectedBoxPaintsTheSelectionInk(t *testing.T) {
	plain := NewPanel()
	plain.SetBackground(graphics.Color(0xFFFFFFFF))
	plainImg := paintPanel(t, plain, 80, 40)

	chosen := NewPanel()
	chosen.SetBackground(graphics.Color(0xFFFFFFFF))
	chosen.SetSelected(true)
	chosen.SetSelectedInk(graphics.Color(0x3300FF00))
	chosenImg := paintPanel(t, chosen, 80, 40)

	base := plainImg.RGBAAt(40, 20)
	sel := chosenImg.RGBAAt(40, 20)
	if base.G != 0xFF || base.R != 0xFF {
		t.Fatalf("the plain box is %v, want white", base)
	}
	if sel.G <= sel.R+4 {
		t.Fatalf("the selected box is %v, want a green wash over white", sel)
	}
}

// TestSelectionSurvivesAHover pins the paint ORDER: the selected wash is the surface's
// state and the hover wash layers on top, so hovering a selected row does not unselect
// it visually.
func TestSelectionSurvivesAHover(t *testing.T) {
	p := NewPanel()
	p.SetBackground(graphics.Color(0xFFFFFFFF))
	p.SetSelected(true)
	p.SetSelectedInk(graphics.Color(0x330000FF)) // blue selection
	p.SetClickable(true)
	p.SetHoverInk(graphics.Color(0x22FF0000)) // red hover
	p.SetHoveredState(true)
	p.FinishAnimations() // the hover wash fades in; settle it before measuring
	img := paintPanel(t, p, 60, 30)
	px := img.RGBAAt(30, 15)
	if px.B <= px.R {
		t.Fatalf("hovered selected box is %v; want blue (selection) with a red tint on top", px)
	}
}

// TestAFrostedBoxBlursWhatIsBehindIt pins the backdrop blur: a hard black/white edge
// under the box must come out as intermediate grey. The kernel is the renderer's (the
// same 7 taps GL uses), so this asserts the OPERATION, not a strength.
func TestAFrostedBoxBlursWhatIsBehindIt(t *testing.T) {
	theme.SetDefaultTheme(theme.DefaultTheme)
	canvas, dev := graphics.NewSoftwareCanvas(120, 60)
	canvas.Clear(0xFFFFFFFF)

	// A hard edge at x=60.
	left := graphics.NewPaint()
	left.SetColor(0xFF000000)
	canvas.DrawRect(graphics.NewRect(0, 0, 60, 60), left)
	right := graphics.NewPaint()
	right.SetColor(0xFFFFFFFF)
	canvas.DrawRect(graphics.NewRect(60, 0, 60, 60), right)

	// A frosted box over the edge, with a completely transparent fill so only the blur
	// shows (a tint would hide it).
	box := NewPanel()
	box.SetBackground(graphics.Color(0x00000000))
	box.SetBackdropBlur(true)
	box.SetFrameRect(graphics.NewRect(40, 10, 40, 40))
	canvas.Save()
	canvas.Translate(40, 10)
	box.Paint(canvas, graphics.NewRect(0, 0, 40, 40))
	canvas.Restore()

	img := dev.Image()
	// Well inside the box, far from the edge, the blur leaves the hard colour.
	if px := img.RGBAAt(46, 30); px.R > 0x40 {
		t.Fatalf("left of the edge inside the glass is %v, want near black", px)
	}
	if px := img.RGBAAt(80, 30); px.R < 0xB0 {
		t.Fatalf("right of the edge inside the glass is %v, want near white", px)
	}
	// At the edge the blur must have mixed the two: one pixel inside the black half is
	// no longer black (the kernel reaches 3px past the edge).
	if px := img.RGBAAt(59, 30); px.R < 0x30 || px.R > 0xE0 {
		t.Fatalf("at the edge the glass is %v: the backdrop was not blurred", px)
	}
}

// TestARoundedFrostedBoxKeepsItsCorners pins the MASK. The backdrop is a fine
// checkerboard, so any blur turns it into uniform grey: inside the glass that is what we
// want, and OUTSIDE the rounded silhouette the checker's own black/white must survive. A
// frosted rectangle would grey the corner too, which is the bug this keeps out.
func TestARoundedFrostedBoxKeepsItsCorners(t *testing.T) {
	theme.SetDefaultTheme(theme.DefaultTheme)
	const cw, ch = 120, 60
	canvas, dev := graphics.NewSoftwareCanvas(cw, ch)
	canvas.Clear(0xFFFFFFFF)
	black := graphics.NewPaint()
	black.SetColor(0xFF000000)
	for y := 0; y < ch; y += 2 {
		for x := 0; x < cw; x += 2 {
			if (x/2+y/2)%2 == 0 {
				canvas.DrawRect(graphics.NewRect(float32(x), float32(y), 2, 2), black)
			}
		}
	}

	box := NewPanel()
	box.SetBackground(graphics.Color(0x00000000))
	box.SetBackdropBlur(true)
	box.SetCornerRadius(16)
	box.SetFrameRect(graphics.NewRect(20, 10, 80, 40))
	canvas.Save()
	canvas.Translate(20, 10)
	box.Paint(canvas, graphics.NewRect(0, 0, 80, 40))
	canvas.Restore()

	img := dev.Image()
	// The very corner of the box's RECT is outside the rounded silhouette: the checker
	// must be untouched there.
	if px := img.RGBAAt(21, 11); px.R != 0x00 && px.R != 0xFF {
		t.Fatalf("the frosted box's corner is %v: the blur greyed outside the radius", px)
	}
	// Inside the silhouette the checker is blurred to grey.
	if px := img.RGBAAt(60, 30); px.R < 0x60 || px.R > 0xA0 {
		t.Fatalf("inside the glass is %v, want the blurred (grey) checker", px)
	}
}

// TestTheHoverWashAnimatesIn pins motion on the Box: hovering STARTS a transition (the
// first frame is still the resting colour), and FinishAnimations lands on the theme's
// hover ink. A box that snapped would show the wash on the first frame.
func TestTheHoverWashAnimatesIn(t *testing.T) {
	theme.SetDefaultTheme(theme.DefaultTheme)
	p := NewPanel()
	p.SetBackground(graphics.Color(0xFFFFFFFF))
	p.SetClickable(true)
	p.SetHoverInk(graphics.Color(0x80FF0000))
	p.SetFrameRect(graphics.NewRect(0, 0, 40, 40))

	rest := paintPanel(t, p, 40, 40)

	p.SetHoveredState(true)
	if !p.hoverAnim.Active() {
		t.Fatal("hovering did not start a transition")
	}
	first := paintPanel(t, p, 40, 40)
	if first.RGBAAt(20, 20).R < 0xF0 {
		t.Fatalf("the first hovered frame is already washed (%v): the transition snapped",
			first.RGBAAt(20, 20))
	}

	p.FinishAnimations()
	settled := paintPanel(t, p, 40, 40)
	if !(rest.RGBAAt(20, 20).R > settled.RGBAAt(20, 20).R+20) {
		t.Fatalf("after settling rest=%v settled=%v: the hover ink is not visible",
			rest.RGBAAt(20, 20), settled.RGBAAt(20, 20))
	}

	// Leaving animates too, and settling restores the resting colour.
	p.SetHoveredState(false)
	p.FinishAnimations()
	left := paintPanel(t, p, 40, 40)
	if got, want := left.RGBAAt(20, 20), rest.RGBAAt(20, 20); got != want {
		t.Fatalf("after leaving the wash did not clear: %v, want %v", got, want)
	}
}
