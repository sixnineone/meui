// Checkout — the sixth web scene: a three-step flow with validation.
//
//	go run ./examples/checkout
//	go run ./examples/checkout -shot step1.png
//	go run ./examples/checkout -fill -step 3 -shot review.png
//
// The point of the scene is the FORM: a bad field gets the error ink, an inline message,
// and the step does not advance; fixing it clears both. The step indicator, the footer's
// labels and the success panel are all restyled from Go by id.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

type step int

const (
	stepDetails step = iota
	stepPayment
	stepReview
	stepDone
)

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "examples/checkout/checkout.qml", "scene file")
	wantStep := flag.Int("step", 0, "advance to this step with valid data before shooting (0-3)")
	fill := flag.Bool("fill", false, "prefill the form with valid data")
	invalid := flag.Bool("invalid", false, "press Continue once with empty fields, to shoot the error state")
	flag.Parse()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkout: %v\n", err)
		os.Exit(1)
	}
	wizard := wire(scene)
	if *fill {
		wizard.fillValid()
	}
	if *invalid {
		wizard.onNext()
	}
	if *wantStep > 0 {
		wizard.advanceTo(step(*wantStep), scene)
	}

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "checkout: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("checkout: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "checkout: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · checkout"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · checkout",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "checkout: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "checkout: run: %v\n", err)
		os.Exit(1)
	}
}

// wizard owns the flow's state. Every widget it touches is looked up once, so a missing id
// is reported at wire time rather than silently doing nothing on a click.
type wizard struct {
	scene *web.Scene
	step  step

	fieldEmail, fieldName, fieldCompany *widgets.TextField
	fieldCard, fieldExpiry, fieldCvc    *widgets.TextField
	errEmail, errName, errCard          *widgets.Label
	errExpiry, errCvc, errTerms         *widgets.Label
	stepPanel                           []*meui.Panel
	dot, num                            []*meui.Panel
	numLabel                            []*widgets.Label
	line                                []*meui.Panel
	stepLabel                           []*widgets.Label
	next                                *widgets.Button
	back, flowCard, successPanel        *meui.Panel
	sumEmail, sumCard, orderNumber      *widgets.Label
	agreeTerms                          *widgets.CheckBox
	statusText                          *widgets.Label
	nextStep                            func()
}

func wire(scene *web.Scene) *wizard {
	w := &wizard{scene: scene}
	get := func(id string) meui.Widget {
		wd := scene.Get(id)
		if wd == nil {
			panic("checkout: missing id " + id)
		}
		return wd
	}
	w.fieldEmail = get("fieldEmail").(*widgets.TextField)
	w.fieldName = get("fieldName").(*widgets.TextField)
	w.fieldCompany = get("fieldCompany").(*widgets.TextField)
	w.fieldCard = get("fieldCard").(*widgets.TextField)
	w.fieldExpiry = get("fieldExpiry").(*widgets.TextField)
	w.fieldCvc = get("fieldCvc").(*widgets.TextField)
	w.errEmail = get("errEmail").(*widgets.Label)
	w.errName = get("errName").(*widgets.Label)
	w.errCard = get("errCard").(*widgets.Label)
	w.errExpiry = get("errExpiry").(*widgets.Label)
	w.errCvc = get("errCvc").(*widgets.Label)
	w.errTerms = get("errTerms").(*widgets.Label)
	w.stepPanel = []*meui.Panel{
		get("step1").(*meui.Panel),
		get("step2").(*meui.Panel),
		get("step3").(*meui.Panel),
	}
	for _, id := range []string{"step1Dot", "step2Dot", "step3Dot"} {
		w.dot = append(w.dot, get(id).(*meui.Panel))
	}
	for _, id := range []string{"step1Num", "step2Num", "step3Num"} {
		w.numLabel = append(w.numLabel, get(id).(*widgets.Label))
	}
	for _, id := range []string{"line1", "line2"} {
		w.line = append(w.line, get(id).(*meui.Panel))
	}
	for _, id := range []string{"step1Label", "step2Label", "step3Label"} {
		w.stepLabel = append(w.stepLabel, get(id).(*widgets.Label))
	}
	w.next = get("nextButton").(*widgets.Button)
	w.back = get("backButton").(*meui.Panel)
	w.flowCard = get("flowCard").(*meui.Panel)
	w.successPanel = get("successPanel").(*meui.Panel)
	w.sumEmail = get("sumEmail").(*widgets.Label)
	w.sumCard = get("sumCard").(*widgets.Label)
	w.orderNumber = get("orderNumber").(*widgets.Label)
	w.agreeTerms = get("agreeTerms").(*widgets.CheckBox)
	w.statusText = get("statusText").(*widgets.Label)

	// Checks are DRAWN: the toolkit's face has no ✓, and a tofu box is not a tick.
	for i, id := range []string{"step1Dot", "step2Dot", "step3Dot"} {
		i := i
		scene.OnPaint(id, func(c *graphics.Canvas, r graphics.Rect) {
			if step(i) < w.step {
				drawCheck(c, r, 36)
			}
		})
	}
	scene.OnPaint("successIcon", func(c *graphics.Canvas, r graphics.Rect) {
		drawCheck(c, r, 44)
	})

	w.next.SetOnClicked(func() { w.onNext() })
	scene.OnClick("backButton", func() { w.onBack() })
	scene.OnClick("backToStore", func() {
		scene.SetText("statusText", "Back to the store — this demo has no store, so the flow resets.")
		w.reset()
	})
	w.apply()
	return w
}

// fillValid fills every field with data that passes, for the -fill/-step shots and tests.
func (w *wizard) fillValid() {
	w.fieldEmail.SetText("port@atlas.dev")
	w.fieldName.SetText("Port Aurelia")
	w.fieldCompany.SetText("Atlas Studio")
	w.fieldCard.SetText("4242 4242 4242 4242")
	w.fieldExpiry.SetText("08/29")
	w.fieldCvc.SetText("123")
	w.agreeTerms.SetChecked(true)
}

// advanceTo walks the flow with valid data until it reaches target. It is how a shot of a
// later step is produced without fake state.
func (w *wizard) advanceTo(target step, scene *web.Scene) {
	// Stopping at stepDone is what keeps a target past the last step from looping: onNext
	// has nowhere to go once the order is placed.
	for w.step < target && w.step != stepDone {
		w.onNext()
	}
}

// reset returns to the first step with the form cleared.
func (w *wizard) reset() {
	for _, f := range []*widgets.TextField{w.fieldEmail, w.fieldName, w.fieldCompany, w.fieldCard, w.fieldExpiry, w.fieldCvc} {
		f.SetText("")
		f.SetValidationState("")
	}
	w.agreeTerms.SetChecked(false)
	for _, e := range []*widgets.Label{w.errEmail, w.errName, w.errCard, w.errExpiry, w.errCvc, w.errTerms} {
		e.SetVisible(false)
	}
	w.step = stepDetails
	w.successPanel.SetVisible(false)
	w.flowCard.SetVisible(true)
	w.apply()
}

func (w *wizard) onNext() {
	if w.step == stepDone {
		return
	}
	switch w.step {
	case stepDetails:
		if !w.validateDetails() {
			return
		}
		w.step = stepPayment
	case stepPayment:
		if !w.validatePayment() {
			return
		}
		w.step = stepReview
		w.sumEmail.SetText(w.fieldEmail.Text)
		digits := digitsOnly(w.fieldCard.Text)
		w.sumCard.SetText("Card ending " + digits[len(digits)-4:])
	case stepReview:
		if !w.agreeTerms.Checked {
			w.errTerms.SetVisible(true)
			w.scene.SetText("statusText", "Accept the terms to place the order.")
			return
		}
		w.errTerms.SetVisible(false)
		w.step = stepDone
		w.successPanel.SetVisible(true)
		w.flowCard.SetVisible(false)
		w.orderNumber.SetText("Order A-10428 · licence for " + w.fieldEmail.Text)
		w.scene.SetText("statusText", "Order placed. Nothing is charged — this is a demo.")
		return
	}
	w.apply()
}

func (w *wizard) onBack() {
	if w.step == stepDetails || w.step == stepDone {
		return
	}
	w.step--
	w.apply()
}

// apply repaints the chrome for the current step: which panel shows, the indicator's
// done/active/upcoming states, the footer labels and the status line.
func (w *wizard) apply() {
	for i, p := range w.stepPanel {
		p.SetVisible(step(i) == w.step)
	}
	for i := range w.dot {
		done := step(i) < w.step
		active := step(i) == w.step
		switch {
		case done || active:
			w.dot[i].SetBackground(graphics.Color(0xFF14243C))
			w.dot[i].SetBorderWidth(0)
			if done {
				w.numLabel[i].SetText("")
			} else {
				w.numLabel[i].SetText(fmt.Sprintf("%d", i+1))
			}
			w.numLabel[i].SetColor(graphics.Color(0xFFFFFFFF))
			w.stepLabel[i].SetColor(graphics.Color(0xFF0F1115))
			w.stepLabel[i].SetFontBold(true)
		default:
			w.dot[i].SetBackground(graphics.Color(0xFFFFFFFF))
			w.dot[i].SetBorderWidth(1)
			w.dot[i].SetBorderColor(graphics.Color(0xFFD9D4C8))
			w.numLabel[i].SetText(fmt.Sprintf("%d", i+1))
			w.numLabel[i].SetColor(graphics.Color(0xFF9CA3AF))
			w.stepLabel[i].SetColor(graphics.Color(0xFF9CA3AF))
			w.stepLabel[i].SetFontBold(false)
		}
	}
	for i, line := range w.line {
		if step(i) < w.step {
			line.SetBackground(graphics.Color(0xFF14243C))
		} else {
			line.SetBackground(graphics.Color(0xFFD9D4C8))
		}
	}
	w.back.SetVisible(w.step != stepDetails && w.step != stepDone)
	switch w.step {
	case stepDetails:
		w.next.SetText("Continue")
		w.scene.SetText("statusText", "Your details are only used for the receipt.")
	case stepPayment:
		w.next.SetText("Continue")
		w.scene.SetText("statusText", "This demo never contacts a payment provider.")
	case stepReview:
		w.next.SetText("Place order")
		w.scene.SetText("statusText", "Check the order, accept the terms, and place it.")
	}
}

// ── validation ──────────────────────────────────────────────────────

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
var expiryRE = regexp.MustCompile(`^(0[1-9]|1[0-2])/\d{2}$`)

// validateDetails marks each bad field, reveals its message, focuses the first one, and
// reports whether the step may advance.
func (w *wizard) validateDetails() bool {
	ok := true
	badEmail := !emailRE.MatchString(strings.TrimSpace(w.fieldEmail.Text))
	w.fieldEmail.SetValidationState(errState(badEmail))
	w.errEmail.SetVisible(badEmail)
	if badEmail {
		ok = false
		w.focus(w.fieldEmail)
	}
	badName := strings.TrimSpace(w.fieldName.Text) == ""
	w.fieldName.SetValidationState(errState(badName))
	w.errName.SetVisible(badName)
	if badName {
		ok = false
		if !badEmail {
			w.focus(w.fieldName)
		}
	}
	if !ok {
		w.scene.SetText("statusText", "Fix the highlighted field and continue.")
	}
	return ok
}

func (w *wizard) validatePayment() bool {
	ok := true
	card := digitsOnly(w.fieldCard.Text)
	badCard := len(card) != 16
	w.fieldCard.SetValidationState(errState(badCard))
	w.errCard.SetVisible(badCard)
	if badCard {
		ok = false
		w.focus(w.fieldCard)
	}
	badExpiry := !expiryRE.MatchString(strings.TrimSpace(w.fieldExpiry.Text))
	w.fieldExpiry.SetValidationState(errState(badExpiry))
	w.errExpiry.SetVisible(badExpiry)
	if badExpiry {
		ok = false
		if !badCard {
			w.focus(w.fieldExpiry)
		}
	}
	cvc := digitsOnly(w.fieldCvc.Text)
	badCvc := len(cvc) < 3 || len(cvc) > 4
	w.fieldCvc.SetValidationState(errState(badCvc))
	w.errCvc.SetVisible(badCvc)
	if badCvc {
		ok = false
		if !badCard && !badExpiry {
			w.focus(w.fieldCvc)
		}
	}
	if !ok {
		w.scene.SetText("statusText", "Check the payment details and continue.")
	}
	return ok
}

func errState(bad bool) string {
	if bad {
		return "error"
	}
	return ""
}

// focus gives a failed field the keyboard so the user can type over the bad value.
func (w *wizard) focus(f *widgets.TextField) {
	if win, ok := w.scene.Root.(*meui.Window); ok {
		win.SetFocusedChild(f)
	}
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// drawCheck paints a tick inside a square of the given design size: two strokes, because
// the scene cannot rely on a check mark existing in the resolved face.
func drawCheck(c *graphics.Canvas, r graphics.Rect, design float32) {
	s := r.Width / design
	p := graphics.NewPaint()
	p.SetStyle(paintstyle.Stroke)
	p.SetStrokeWidth(2.4 * s)
	p.SetColor(graphics.Color(0xFFFFFFFF))
	x, y := r.X, r.Y
	c.Save()
	c.Translate(x, y)
	c.DrawLine(12*s, 19*s, 17*s, 24*s, p)
	c.DrawLine(17*s, 24*s, 27*s, 12*s, p)
	c.Restore()
}
