// Mail — the fifth web scene: a three-column inbox with list selection, reveal-on-
// select and an app-supplied display face.
//
//	go run ./examples/mail
//	go run ./examples/mail -shot out.png
//
// The subject in the reading pane is set in "Editorial", a family registered from a TTF
// on this machine just before the QML is parsed — `font.family` resolves app-supplied
// faces through graphics.MatchFontFace, and falls back to the preset's face when the
// file is absent. The list is taller than its viewport on purpose: choosing a row
// scrolls it into view.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sixnineone/meui"
	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/slant"
	"github.com/sixnineone/meui/graphics/enums/weight"
	"github.com/sixnineone/meui/web"
	"github.com/sixnineone/meui/widgets"
)

type message struct {
	Sender  string
	Email   string
	When    string
	Subject string
	Snippet string
	Body    string
	Colour  graphics.Color
}

var inboxSeeds = []message{
	{
		Sender: "Mira Chen", Email: "mira@atlas.dev", When: "09:42",
		Subject: "Re: Atlas 1.4 — the migration plan",
		Snippet: "Looks right to me. One thing about the parity suite: the two scenes that drifted are the gradient ones.",
		Body:    "Looks right to me. One thing about the parity suite: the two scenes that drifted are the ones with gradients, and they drifted in the software renderer, not the GL path — so the fix is in the device, not the scene.\n\nI re-ran the nineteen scenes after the change and twelve are byte-identical now. The other seven differ only where the text sits, which is the DPI fix landing.\n\nCan you take the review on Friday?",
		Colour:  0xFF4F46E5,
	},
	{
		Sender: "Tomas Beck", Email: "tomas@atlas.dev", When: "09:12",
		Subject: "Friday's demo — who presents?",
		Snippet: "I can take the renderer section if you want the timeline to breathe a little.",
		Body:    "I can take the renderer section if you want the timeline to breathe. The last run-through was tight on the scroll demo: the reveal lands halfway through the sentence and nobody sees the list move.\n\nIf we cut the parity slides to two, the section fits.",
		Colour:  0xFF0EA5E9,
	},
	{
		Sender: "Priya Nair", Email: "priya@atlas.dev", When: "Yesterday",
		Subject: "Design review notes",
		Snippet: "Three things came out of the review, and one is a blocker for the docs scene.",
		Body:    "Three things came out of the review:\n\n1. The docs sidebar selection is right, but the wash is too strong over the cover image.\n2. The code block needs a horizontal scroll before the long lines land.\n3. The type scale reads well at 1280; at 860 the body is still 15, which is too big.\n\nThe second is the blocker.",
		Colour:  0xFF10B981,
	},
	{
		Sender: "Atlas CI", Email: "ci@atlas.dev", When: "Yesterday",
		Subject: "Build #4821 passed",
		Snippet: "All 19 render scenes matched their baselines. Offscreen parity is still manual.",
		Body:    "All 19 render scenes matched their baselines at tolerance 0.5%.\n\nOffscreen parity is still manual: the software device is checked by the shot tests, but nothing compares it to the GL frame pixel for pixel. That is the next guard worth building.",
		Colour:  0xFF6B7280,
	},
	{
		Sender: "Jonas Weiss", Email: "jonas@atlas.dev", When: "Monday",
		Subject: "Font licensing question",
		Snippet: "The display face we picked ships under an open licence, but only as TTF.",
		Body:    "The display face we picked ships under an open licence, but only as TTF. Our loader refuses OTF, so that is convenient for once.\n\nRegister it under a family name per scene rather than embedding it in the toolkit — the face is art direction, not a keystroke.",
		Colour:  0xFFF59E0B,
	},
	{
		Sender: "Nadia Rahman", Email: "nadia@atlas.dev", When: "Monday",
		Subject: "Q1 roadmap draft",
		Snippet: "Attached the draft. The order is the argument, so please read it in order.",
		Body:    "Attached the draft. The order is the argument, so please read it in order rather than jumping to the list at the end.\n\nThe section on parity is the one I am least sure about; the rest is a restatement of what we already agreed.",
		Colour:  0xFFEF4444,
	},
	{
		Sender: "Mira Chen", Email: "mira@atlas.dev", When: "Sunday",
		Subject: "Re: Scroll performance",
		Snippet: "The list is smooth at 1200 rows now. The fix was the clip, not the layout.",
		Body:    "The list is smooth at 1200 rows now. The fix was the clip, not the layout: the device drew every image whole, so a cover-fit crop spilled its full width across the card and the compositor had to clip it back.\n\nWith the restore path in place, the frame cost dropped by a third.",
		Colour:  0xFF8B5CF6,
	},
	{
		Sender: "Atlas CI", Email: "ci@atlas.dev", When: "Sunday",
		Subject: "Nightly parity report",
		Snippet: "Two scenes drifted by more than the tolerance, both in the software device.",
		Body:    "Two scenes drifted by more than the tolerance, both in the software device: the sky gradient and the rounded card shadows.\n\nThe GL frame is unchanged, so this is the offscreen path only. Baselines have not been rebuilt.",
		Colour:  0xFF6B7280,
	},
}

// inboxCount is deliberately far beyond what a static list can afford: the point of the
// VirtualList is that 240 rows cost twelve widgets.
const inboxCount = 240

// inbox is the data: the eight written messages cycled with a sequence, the way a real
// mailbox repeats shapes with different content.
var inbox = buildInbox(inboxCount)

func buildInbox(n int) []message {
	out := make([]message, 0, n)
	for i := 0; i < n; i++ {
		m := inboxSeeds[i%len(inboxSeeds)]
		if i >= len(inboxSeeds) {
			m.Subject = fmt.Sprintf("%s (%d)", m.Subject, i+1)
			m.When = fmt.Sprintf("%d d", (i/len(inboxSeeds))+1)
		}
		out = append(out, m)
	}
	return out
}

var folders = []string{
	"folderInbox", "folderStarred", "folderSent", "folderDrafts", "folderArchive", "folderTrash",
}

func main() {
	shot := flag.String("shot", "", "render one frame to this PNG and exit (headless, no display)")
	preset := flag.String("preset", "web-light", "theme preset")
	width := flag.Int("w", 1280, "window width")
	height := flag.Int("h", 900, "window height")
	path := flag.String("qml", "examples/mail/mail.qml", "scene file")
	flag.Parse()

	// Before Load: the QML parser resolves `font.family` while it builds the labels.
	registerDisplayFace()

	scene, err := web.Load(*path, *preset, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mail: %v\n", err)
		os.Exit(1)
	}
	wire(scene)

	if *shot != "" {
		if err := scene.Shot(*width, *height, *shot); err != nil {
			fmt.Fprintf(os.Stderr, "mail: shot: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("mail: wrote %s (%dx%d)\n", *shot, *width, *height)
		return
	}

	root, ok := scene.Root.(*meui.Window)
	if !ok {
		fmt.Fprintf(os.Stderr, "mail: root is %T, want *meui.Window\n", scene.Root)
		os.Exit(1)
	}
	root.SetFrameRect(graphics.NewRect(0, 0, float32(*width), float32(*height)))
	app := meui.NewApp(sdl3.NewApp(), meui.AppOptions{Title: "meui · mail"})
	if _, err := app.AddWindow(root, meui.WindowOptions{
		Title: "meui · mail",
		Size:  graphics.NewSize(float32(*width), float32(*height)),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "mail: add window: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mail: run: %v\n", err)
		os.Exit(1)
	}
}

// registerDisplayFace loads a real serif from this machine under the name the scene
// asks for. A missing file is not an error: the scene renders in the preset's face, so
// the demo stays runnable on a machine without DejaVu.
func registerDisplayFace() {
	candidates := []struct {
		path string
		w    weight.Enum
	}{
		{"/usr/share/fonts/truetype/DejaVuSerif-Bold.ttf", weight.Bold},
		{"/usr/share/fonts/truetype/DejaVuSerif.ttf", weight.Regular},
	}
	loaded := 0
	for _, c := range candidates {
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		if err := graphics.RegisterFontFamily("Editorial", c.path, c.w, slant.Upright); err != nil {
			log.Printf("mail: register %s: %v", c.path, err)
			continue
		}
		loaded++
	}
	if loaded == 0 {
		log.Printf("mail: no display face found; the subject falls back to the preset font")
	}
}

func wire(scene *web.Scene) {
	vl, _ := scene.Get("messageView").(*widgets.VirtualList)
	selected := 0

	selectMessage := func(i int) {
		if i < 0 || i >= len(inbox) {
			return
		}
		selected = i
		m := inbox[i]
		scene.SetText("readSubject", m.Subject)
		scene.SetText("readSender", m.Sender)
		scene.SetText("readMeta", m.Email+" · to me · "+m.When)
		scene.SetText("readBody", m.Body)
		if r := []rune(m.Sender); len(r) > 0 {
			scene.SetText("readInitial", string(r[0]))
		}
		if avatar, ok := scene.Get("readAvatar").(*meui.Panel); ok {
			avatar.SetBackground(m.Colour)
		}
		// Restyle the rows that EXIST; rows built later apply `selected` in the builder.
		if vl != nil {
			vl.WalkBuilt(func(index int, w meui.Widget) {
				if row, ok := w.(*meui.Panel); ok {
					row.SetSelected(index == selected)
				}
			})
			vl.EnsureVisible(i)
			vl.MarkForRedraw()
		}
		scene.SetText("statusText", fmt.Sprintf("%s — message %d of %d.", m.Sender, i+1, len(inbox)))
	}

	if vl != nil {
		vl.SetRowCount(len(inbox))
		vl.SetRowBuilder(func(i int) meui.Widget {
			// The closure takes i by parameter, so each built row's handler is its own.
			return buildRow(i, selected, func() { selectMessage(i) })
		})
		vl.Refresh()
		selectMessage(0)
	}

	scene.OnClick("floatCompose", func() {
		scene.SetText("statusText", "New message — the floating button sits above the list by z.")
	})
	scene.OnClick("composeButton", func() {
		scene.SetText("statusText", "Compose would open an editor — not wired in this demo.")
	})
	scene.OnClick("replyButton", func() {
		scene.SetText("statusText", "Reply would focus the composer at the bottom of the thread.")
	})
	scene.OnClick("forwardButton", func() {
		scene.SetText("statusText", "Forward would pick a recipient from your contacts.")
	})
	for _, id := range folders {
		id := id
		scene.OnClick(id, func() {
			for _, other := range folders {
				scene.Select(other, other == id)
			}
			scene.SetText("statusText", "Folders are a selection group; this demo has one inbox.")
		})
	}
}

// buildRow is the row TEMPLATE: a Go function, because QML here is a builder with no
// delegates. The list calls it for an index as the row enters the viewport.
func buildRow(i, selected int, onClick func()) meui.Widget {
	m := inbox[i]
	row := meui.NewPanel()
	row.SetCornerRadius(10)
	row.SetBackground(graphics.Color(0xFFFFFFFF))
	row.SetSelected(i == selected)
	row.SetOnClick(onClick)

	avatar := meui.NewPanel()
	avatar.SetFrameRect(graphics.NewRect(14, 24, 36, 36))
	avatar.SetCornerRadius(18)
	avatar.SetBackground(m.Colour)
	initial := widgets.NewLabel()
	initial.SetText(string([]rune(m.Sender)[0]))
	initial.SetFontSize(11)
	initial.SetFontBold(true)
	initial.SetColor(graphics.Color(0xFFFFFFFF))
	initial.SetHAlign("center")
	initial.SetFrameRect(graphics.NewRect(0, 0, 36, 36))
	avatar.AddChild(initial)
	row.AddChild(avatar)

	sender := widgets.NewLabel()
	sender.SetText(m.Sender)
	sender.SetFontSize(9)
	sender.SetFontBold(true)
	sender.SetColor(graphics.Color(0xFF0F1115))
	sender.SetFrameRect(graphics.NewRect(62, 18, 190, 18))
	row.AddChild(sender)

	when := widgets.NewLabel()
	when.SetText(m.When)
	when.SetFontSize(7)
	when.SetColor(graphics.Color(0xFF9CA3AF))
	when.SetHAlign("right")
	when.SetFrameRect(graphics.NewRect(250, 19, 92, 16))
	row.AddChild(when)

	subject := widgets.NewLabel()
	subject.SetText(m.Subject)
	subject.SetFontSize(9)
	subject.SetColor(graphics.Color(0xFF374151))
	subject.SetWrapMode(widgets.WrapEllipsis)
	subject.SetFrameRect(graphics.NewRect(62, 40, 280, 18))
	row.AddChild(subject)

	snippet := widgets.NewLabel()
	snippet.SetText(m.Snippet)
	snippet.SetFontSize(8)
	snippet.SetColor(graphics.Color(0xFF9CA3AF))
	snippet.SetWrapMode(widgets.WrapWordWrap)
	snippet.SetMaxLines(2)
	snippet.SetFrameRect(graphics.NewRect(62, 62, 280, 36))
	row.AddChild(snippet)

	return row
}
