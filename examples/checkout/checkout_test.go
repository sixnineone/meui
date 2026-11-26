package main

// The checkout flow's contract: a bad field blocks the step and shows its message, a good
// one advances, and the last step turns the whole flow into the success panel. Clicks go
// through the window, so the real hit test is exercised.

import (
	"testing"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

func loadCheckout(t *testing.T) (*web.Scene, *meui.Window, *wizard) {
	t.Helper()
	scene, err := web.Load("checkout.qml", "web-light", 1280, 900)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	w := wire(scene)
	win, ok := scene.Root.(*meui.Window)
	if !ok {
		t.Fatalf("root is %T, want *meui.Window", scene.Root)
	}
	return scene, win, w
}

// clickContinue presses the footer button: the card is at (280,240), the button at
// local (548,396) size 140x44, so its centre is (898, 658) in window space.
func clickContinue(t *testing.T, win *meui.Window) {
	t.Helper()
	pt := graphics.NewPoint(898, 658)
	win.HandleEvent(&platform.Event{Type: platform.EventMouseDown, Pos: pt, Button: 0})
	win.HandleEvent(&platform.Event{Type: platform.EventMouseUp, Pos: pt, Button: 0})
}

func TestAnInvalidFieldBlocksTheStepAndExplainsItself(t *testing.T) {
	scene, win, w := loadCheckout(t)

	clickContinue(t, win)

	if w.step != stepDetails {
		t.Fatalf("the step advanced to %d with an empty form", w.step)
	}
	if !w.errEmail.Visible() || !w.errName.Visible() {
		t.Fatal("the inline errors are not visible")
	}
	if got := w.fieldEmail.ValidationState; got != "error" {
		t.Fatalf("email validationState = %q, want error", got)
	}
	if w.stepPanel[0].Visible() == false {
		t.Fatal("step 1 should still be the visible panel")
	}
	if label, ok := scene.Get("statusText").(*widgets.Label); ok {
		if label.Text == "" {
			t.Fatal("the status line did not explain the failure")
		}
	}
}

func TestAValidFlowReachesTheSuccessPanel(t *testing.T) {
	scene, win, w := loadCheckout(t)
	w.fillValid()

	clickContinue(t, win) // details → payment
	if w.step != stepPayment {
		t.Fatalf("after valid details the step is %d, want payment", w.step)
	}
	if w.errEmail.Visible() {
		t.Fatal("the email error stayed visible after being fixed")
	}

	clickContinue(t, win) // payment → review
	if w.step != stepReview {
		t.Fatalf("after valid payment the step is %d, want review", w.step)
	}
	if label := scene.Get("sumEmail").(*widgets.Label); label.Text != "port@atlas.dev" {
		t.Fatalf("review email = %q", label.Text)
	}
	if label := scene.Get("sumCard").(*widgets.Label); label.Text != "Card ending 4242" {
		t.Fatalf("review card = %q", label.Text)
	}

	clickContinue(t, win) // place order
	if w.step != stepDone {
		t.Fatalf("placing the order left the step at %d", w.step)
	}
	if !w.successPanel.Visible() {
		t.Fatal("the success panel is not visible")
	}
	if w.flowCard.Visible() {
		t.Fatal("the form is still visible behind the success panel")
	}
	if label := scene.Get("orderNumber").(*widgets.Label); label.Text == "" || label.Text == "—" {
		t.Fatalf("no order number was written: %q", label.Text)
	}
}

func TestTheTermsGateTheOrder(t *testing.T) {
	_, win, w := loadCheckout(t)
	w.fillValid()
	w.agreeTerms.SetChecked(false)

	clickContinue(t, win) // details → payment
	clickContinue(t, win) // payment → review
	clickContinue(t, win) // place order, without the terms

	if w.step != stepReview {
		t.Fatalf("the order was placed without the terms (step %d)", w.step)
	}
	if !w.errTerms.Visible() {
		t.Fatal("the terms error is not visible")
	}
	w.agreeTerms.SetChecked(true)
	clickContinue(t, win)
	if w.step != stepDone {
		t.Fatalf("accepting the terms did not place the order (step %d)", w.step)
	}
}
