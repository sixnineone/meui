package platform

import "testing"

// TestNotificationZeroIDIsValid pins the field an app is most likely to get
// wrong: ID 0 is a real notification ID, not "unset". Posting with the same ID
// replaces the previous notification rather than stacking a second one, so an
// app that treats 0 as "no id" would silently stack.
func TestNotificationZeroIDIsValid(t *testing.T) {
	var n Notification
	if n.ID != 0 {
		t.Fatalf("a zero Notification has ID %d, want 0", n.ID)
	}
	if n.Title != "" || n.Body != "" {
		t.Errorf("a zero Notification has Title=%q Body=%q, want empty", n.Title, n.Body)
	}
}
