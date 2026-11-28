package platform

import (
	"strings"
	"testing"
)

// Links reach OpenURL from scene data (Markdown, rich text), so the seam must
// allow-list schemes rather than hand anything to the OS handler.
func TestOpenURLRefusesDangerousSchemes(t *testing.T) {
	var opened []string
	SetOpenURL(func(u string) error {
		opened = append(opened, u)
		return nil
	})
	t.Cleanup(func() { SetOpenURL(nil); SetOpenURLSchemes() })

	for _, raw := range []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
		"https://example.com/\x00",
		"https://example.com/\n",
		"",
		"no-scheme.example.com",
	} {
		if err := OpenURL(raw); err == nil {
			t.Errorf("OpenURL(%q) was allowed", raw)
		}
	}
	if len(opened) != 0 {
		t.Fatalf("backend saw %d URLs that should have been refused: %v", len(opened), opened)
	}
}

func TestOpenURLAllowsTheDefaultSchemes(t *testing.T) {
	var opened []string
	SetOpenURL(func(u string) error {
		opened = append(opened, u)
		return nil
	})
	t.Cleanup(func() { SetOpenURL(nil); SetOpenURLSchemes() })

	for _, raw := range []string{
		"https://example.com/a?b=1",
		"http://example.com",
		"mailto:someone@example.com",
		"tel:+27110000000",
	} {
		if err := OpenURL(raw); err != nil {
			t.Errorf("OpenURL(%q): %v", raw, err)
		}
	}
	if len(opened) != 4 {
		t.Fatalf("backend saw %d URLs, want 4", len(opened))
	}
}

func TestOpenURLSchemeAllowListIsAnAllowList(t *testing.T) {
	var opened string
	SetOpenURL(func(u string) error { opened = u; return nil })
	t.Cleanup(func() { SetOpenURL(nil); SetOpenURLSchemes() })

	SetOpenURLSchemes("myapp")
	if err := OpenURL("myapp://open/1"); err != nil {
		t.Fatalf("custom scheme after SetOpenURLSchemes: %v", err)
	}
	if !strings.HasPrefix(opened, "myapp://") {
		t.Fatalf("opened = %q", opened)
	}
	if err := OpenURL("https://example.com"); err == nil {
		t.Fatal("https should no longer be allowed after replacing the list")
	}
	SetOpenURLSchemes()
	if err := OpenURL("https://example.com"); err != nil {
		t.Fatalf("defaults not restored: %v", err)
	}
}
