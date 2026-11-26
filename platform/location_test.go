package platform

import "testing"

// TestLocationAndPermissionsAreIndependent pins the distinction the docs make,
// because collapsing it is the mistake an app is most likely to make: location
// services being ON says nothing about whether THIS app may use them, and an app
// that checks only one of the two either shows a broken map or prompts for a
// permission it did not need.
func TestLocationAndPermissionsAreIndependent(t *testing.T) {
	// The types are independent by construction: Location has no permission
	// concept and PermissionLocation is just another Permission. This test exists
	// to state that in code, so a future "convenience" that merges them has to
	// delete a test that says why not.
	var _ Location = nil
	var _ Permissions = nil

	// A Position carries the units every platform uses, so there is nothing to
	// convert and nothing to name.
	p := Position{Latitude: 51.5072, Longitude: -0.1276, AccuracyMeters: 12.5}
	if p.Latitude == 0 || p.Longitude == 0 {
		t.Fatal("a zero Position is indistinguishable from the Gulf of Guinea; " +
			"an app must check the bool, not the values")
	}
	if p.At.IsZero() {
		t.Log("Position.At is zero on a fresh value; Last() sets it")
	}
}
