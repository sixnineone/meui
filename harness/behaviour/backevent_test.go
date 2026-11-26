package behaviour

import (
	"testing"

	"github.com/sixnineone/meui/platform"
)

// TestBackIsOfferedToTheAppFirst is Android's back contract, asserted: the gesture goes
// to the app, and only a DECLINE closes the window. The distinction is the whole point
// of the event — before it existed, the system closed the app and an app could not
// participate in its own navigation at all.
func TestBackIsOfferedToTheAppFirst(t *testing.T) {
	h := Load(t, `Window { width: 300; height: 200; VStack { anchors.fill: parent } }`)

	calls := 0
	h.Win.SetOnBack(func() bool {
		calls++
		return true // handled: the app popped its own navigation
	})

	if consumed := h.Win.HandleEvent(&platform.Event{Type: platform.EventBack}); !consumed {
		t.Error("a handled back request reported itself unconsumed, so the window would close anyway")
	}
	if calls != 1 {
		t.Errorf("the handler ran %d times, want 1", calls)
	}
}

// TestBackAtTheRootIsDeclined covers the other half: with no handler, or one that
// declines, the event is NOT consumed — which is what lets the adapter close the window
// at the root of the stack, as the platform expects.
func TestBackAtTheRootIsDeclined(t *testing.T) {
	h := Load(t, `Window { width: 300; height: 200; VStack { anchors.fill: parent } }`)

	// No handler installed.
	if consumed := h.Win.HandleEvent(&platform.Event{Type: platform.EventBack}); consumed {
		t.Error("an unhandled back request was consumed, so the app could never be closed by the gesture")
	}

	// A handler that declines, which is how an app says "I am at the root".
	h.Win.SetOnBack(func() bool { return false })
	if consumed := h.Win.HandleEvent(&platform.Event{Type: platform.EventBack}); consumed {
		t.Error("a DECLINED back request was consumed, so the window would never close")
	}
}
