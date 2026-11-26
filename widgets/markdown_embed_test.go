package widgets

import (
	"strings"
	"testing"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/platform"
)

// TestParseFenceInfoSeparatesLanguagesFromWidgets is the discriminator everything rests
// on. Getting it wrong is silent in the dangerous direction: a widget attempt that fell
// through to the code renderer looks like a document that simply contains a code block,
// and nobody notices until the embed stops appearing.
func TestParseFenceInfoSeparatesLanguagesFromWidgets(t *testing.T) {
	cases := []struct {
		info       string
		wantName   string
		wantWidget bool
		wantErr    bool
	}{
		{"go", "go", false, false},
		{"json", "json", false, false},
		{"", "", false, false},
		{"  python  ", "python", false, false},
		{`tool-call {"id":"read"}`, "tool-call", true, false},
		{`tool-call{"id":"read"}`, "tool-call", true, false},
		{`spark {}`, "spark", true, false},
		// A widget attempt with a broken object is still an attempt: it must be an
		// ERROR, not a code block, or the author gets no signal at all.
		{`tool-call {"id":`, "tool-call", true, true},
		// No brace, so not a widget ATTEMPT by syntax — but the resolver may still
		// claim the name (a widget with no parameters is legitimate). Whether it does
		// is the resolver's call, not the parser's, so this is not an error here.
		{`tool-call notjson`, "tool-call", false, false},
		{`{"id":"read"}`, "", true, true}, // parameters with no name
	}
	for _, c := range cases {
		name, _, isWidget, err := ParseFenceInfo(c.info)
		if name != c.wantName {
			t.Errorf("ParseFenceInfo(%q) name = %q, want %q", c.info, name, c.wantName)
		}
		if isWidget != c.wantWidget {
			t.Errorf("ParseFenceInfo(%q) isWidget = %v, want %v", c.info, isWidget, c.wantWidget)
		}
		if (err != nil) != c.wantErr {
			t.Errorf("ParseFenceInfo(%q) err = %v, want an error: %v", c.info, err, c.wantErr)
		}
	}
}

// TestParseFenceInfoKeepsParameterValues: the parameters are the widget's whole
// configuration, and a JSON number must arrive as a number rather than as the string
// "45" — which is the failure a widget ignores silently.
func TestParseFenceInfoKeepsParameterValues(t *testing.T) {
	_, params, isWidget, err := ParseFenceInfo(`tool-call {"id":"read","timeout":45,"deep":true}`)
	if err != nil || !isWidget {
		t.Fatalf("isWidget=%v err=%v", isWidget, err)
	}
	if params["id"] != "read" {
		t.Errorf("id = %v, want read", params["id"])
	}
	if n, ok := params["timeout"].(float64); !ok || n != 45 {
		t.Errorf("timeout = %#v, want the number 45", params["timeout"])
	}
	if b, ok := params["deep"].(bool); !ok || !b {
		t.Errorf("deep = %#v, want the boolean true", params["deep"])
	}
}

// TestParseFenceInfoToleratesTrailingProse: ```tool {"id":"read"} (see below) is a thing a
// person writes, and rejecting it would push authors to drop the parameters rather than
// learn a stricter syntax.
func TestParseFenceInfoToleratesTrailingProse(t *testing.T) {
	name, params, isWidget, err := ParseFenceInfo(`tool {"id":"read"} and then some prose`)
	if err != nil || !isWidget {
		t.Fatalf("isWidget=%v err=%v", isWidget, err)
	}
	if name != "tool" || params["id"] != "read" {
		t.Errorf("name=%q params=%v, want tool/read", name, params)
	}
}

// TestCtrlACopiesTheMarkdownSource is decision 4, and it is a test because a
// convention this unusual is exactly the kind of thing that gets "tidied" back into
// selecting-only by the next person who assumes they know what Ctrl+A does.
//
// The SOURCE is copied rather than the rendered text because it is lossless: an embed
// survives it, since an embed was never anything but a fence. The rendered text flattens a
// widget to whatever it happened to draw.
func TestCtrlACopiesTheMarkdownSource(t *testing.T) {
	saved := platform.GlobalClipboard
	t.Cleanup(func() { platform.GlobalClipboard = saved })
	platform.GlobalClipboard = &platform.Clipboard{}

	const src = "intro\n\n```probe {\"id\":\"read\"}\nbody\n```\n\ntail\n"
	mv := NewMarkdownView()
	mv.SetFrameRect(graphics.NewRect(0, 0, 400, 300))
	mv.SetText(src)
	mv.computeContentH()
	mv.OnFocus()

	if !mv.OnKeyDown(platform.KeyA, platform.ModCtrl) {
		t.Fatal("Ctrl+A was not handled")
	}
	// The selection is made as well, so Ctrl+X or a drag afterwards behave.
	if !mv.HasSelection() {
		t.Error("Ctrl+A copied without selecting; the selection is what makes the copy useful")
	}
	got := platform.GlobalClipboard.GetText()
	if !strings.Contains(got, "```probe") {
		t.Errorf("the clipboard does not contain the embed's fence:\n%q", got)
	}
	if !strings.Contains(got, `{"id":"read"}`) {
		t.Errorf("the clipboard lost the embed's parameters:\n%q", got)
	}
}
