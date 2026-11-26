package qml

// Every widget a .qml file can place must satisfy meui.EventListener, or the window's
// dispatch silently skips its OnMouseDown: it paints perfectly and takes no input at all.
//
// That is not hypothetical. TabStrip.OnMouseMove returned `bool` while meui.EventListener
// declares `OnMouseMove(x, y float32)` with no return, so the strip did not satisfy the
// interface and every tab was dead to clicks — visible, correct, unclickable, and with
// nothing failing to compile.
//
// This iterates the whole registry, so the next widget with a drifting signature fails here
// rather than in someone's hands.
//
// It applies to INPUT TARGETS, which is the set the window actually dispatches to. A widget
// that is deliberately not a target — InsetSpacer reserves a strip for the IME and takes no
// input — is correctly "dead to clicks", and demanding the interface of it would force it to
// carry no-op handlers that make it LOOK interactive. meui.IsInteractive is the same
// predicate the hit test uses, so this stays honest: a target missing from that registry is
// the touch audit's job to catch, not this test's.

import (
	"testing"

	"github.com/sixnineone/meui"
)

func TestEveryQMLWidgetSatisfiesEventListener(t *testing.T) {
	// The registry is populated at the first parse, not at package init.
	if _, err := ParseWithBindings(`Window { id: root; width: 10; height: 10 }`); err != nil {
		t.Fatalf("priming parse: %s", err)
	}

	names := GetRegisteredWidgets()
	if len(names) == 0 {
		t.Fatal("the widget registry is empty — the priming parse did not run initializeBuiltinWidgets")
	}

	checked, skipped := 0, 0
	for _, name := range names {
		w, err := CreateWidget(name)
		if err != nil {
			t.Fatalf("CreateWidget(%q): %s", name, err)
		}
		if !meui.IsInteractive(w) {
			skipped++
			continue
		}
		checked++
		if _, ok := w.(meui.EventListener); !ok {
			t.Errorf("%s does not satisfy meui.EventListener, so the window will skip its "+
				"OnMouseDown and it will be dead to clicks. OnMouseDown, OnMouseUp and "+
				"OnMouseMove must match the interface EXACTLY — OnMouseMove takes no return "+
				"value. (got %T)", name, w)
		}
	}
	// A vacuous pass would be worse than no test: if the predicate recognised nothing, the
	// assertions above never ran.
	if checked == 0 {
		t.Fatal("no registered widget is an input target, so this proves nothing")
	}
	t.Logf("checked %d input targets, skipped %d non-targets, of %d registered widgets",
		checked, skipped, len(names))
}
