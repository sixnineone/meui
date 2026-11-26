package platform

import "testing"

// TestPermissionStatusesAreDistinct pins the vocabulary an app branches on. The
// four non-granted states mean four different things — ask now, ask again with a
// reason, never ask, and this platform has no such permission — so collapsing
// any two of them would silently change an app's behaviour.
func TestPermissionStatusesAreDistinct(t *testing.T) {
	all := []PermissionStatus{
		PermissionUnknown, PermissionGranted, PermissionDenied,
		PermissionRestricted, PermissionNotDetermined, PermissionUnavailable,
	}
	seen := map[PermissionStatus]string{}
	for _, s := range all {
		if prev, dup := seen[s]; dup {
			t.Errorf("%s and %s share the value %d", s, prev, int(s))
		}
		seen[s] = s.String()
		if s.String() == "unknown" && s != PermissionUnknown {
			t.Errorf("%s prints as \"unknown\", which would hide it in a log", s)
		}
	}
}

// TestPermissionNamesAreStable pins the names, because they appear in logs and
// in the "which permission was refused?" question every support ticket asks.
func TestPermissionNamesAreStable(t *testing.T) {
	cases := map[Permission]string{
		PermissionCamera:             "camera",
		PermissionMicrophone:         "microphone",
		PermissionLocation:           "location",
		PermissionLocationBackground: "location-background",
		PermissionNotifications:      "notifications",
		PermissionBluetooth:          "bluetooth",
		PermissionBluetoothScan:      "bluetooth-scan",
		PermissionContacts:           "contacts",
		PermissionCalendar:           "calendar",
		PermissionPhotos:             "photos",
		PermissionMotion:             "motion",
		PermissionVibrate:            "vibrate",
		PermissionStorageRead:        "storage-read",
	}
	for p, want := range cases {
		if got := p.String(); got != want {
			t.Errorf("Permission(%d).String() = %q, want %q", int(p), got, want)
		}
	}
	if got := Permission(999).String(); got != "unknown" {
		t.Errorf("an out-of-range Permission prints %q, want \"unknown\"", got)
	}
}
