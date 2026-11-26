// cliptest probes Save/ClipRect/Restore pixel behavior through the same
// meuishim path TabPanel uses: translate to a local origin, clip a tab-bar
// rect, draw inside the clip, restore, draw after. Runs through the real
// window capture path (RenderOffscreen needs a live GL context).
package main

import (
	"fmt"
	"os"

	"github.com/sixnineone/meui-gl/sdl3"
	"github.com/sixnineone/meui/graphics"
)

const W, H = 300, 160

// probeContent satisfies Drawable; the shim's reflection finds DrawCallback.
type probeContent struct {
	DrawCallback func(*graphics.Canvas, graphics.Rect)
}

func (p *probeContent) Paint(c *graphics.Canvas, r graphics.Rect) {}

func main() {
	wnd, err := sdl3.NewWindowSized("cliptest", W, H)
	if err != nil {
		fmt.Fprintf(os.Stderr, "window: %v\n", err)
		os.Exit(1)
	}
	content := &probeContent{}
	content.DrawCallback = func(c *graphics.Canvas, rect graphics.Rect) {
		bg := graphics.NewPaint()
		bg.SetColor(graphics.RGB(0, 0, 0))
		c.DrawRect(graphics.NewRect(0, 0, W, H), bg) // before translate

		// Simulate TabPanel: Save, translate to panel origin (10, 20),
		// bar bg before clip, tab button inside clip, rect after restore.
		c.Save()
		c.Translate(10, 20)

		barBg := graphics.NewPaint()
		barBg.SetColor(graphics.RGB(40, 40, 40))
		c.DrawRect(graphics.NewRect(0, 0, 280, 28), barBg) // before clip

		c.Save()
		c.ClipRect(graphics.NewRect(0, 0, 280, 28), 0, false)

		tab := graphics.NewPaint()
		tab.SetColor(graphics.RGB(200, 200, 200))
		c.DrawRect(graphics.NewRect(2, 2, 60, 26), tab) // inside clip

		c.Restore()

		after := graphics.NewPaint()
		after.SetColor(graphics.RGB(255, 0, 0))
		c.DrawRect(graphics.NewRect(100, 5, 40, 20), after) // after restore
		c.Restore()
	}
	wnd.SetContentRect(graphics.NewRect(0, 0, W, H))
	wnd.SetContent(content)

	wnd.CaptureAfterFrame(3, func(pix []uint32, pw, ph int32) {
		if int(pw) != W || int(ph) != H {
			fmt.Printf("FATAL: capture %dx%d, want %dx%d\n", pw, ph, W, H)
			os.Exit(1)
		}
		probe := func(x, y int) uint32 { // y in top-down screen coords
			return pix[(int(ph)-1-y)*int(pw)+x]
		}
		r := probe(200, 35) // barBg-only area: outside tab (12..72) and after (110..150)
		g := probe(30, 35)  // inClip device rect: (12,22)-(72,48)
		b := probe(120, 35) // after device rect: (110,25)-(150,45)
		fmt.Printf("barBg   @(50,135)  = %08X (want ~FF282828)\n", r)
		fmt.Printf("inClip  @(30,110)  = %08X (want ~FFC8C8C8)\n", g)
		fmt.Printf("after   @(120,130) = %08X (want ~FFFF0000)\n", b)
		fail := false
		if r&0x00FFFFFF != 0x282828 {
			fmt.Println("FAIL: bar bg (before clip) missing")
			fail = true
		}
		if g&0x00FFFFFF != 0xC8C8C8 {
			fmt.Println("FAIL: rect inside ClipRect missing — clip path broken")
			fail = true
		}
		if b&0x00FFFFFF != 0xFF0000 {
			fmt.Println("FAIL: rect after Restore missing — restore broken")
			fail = true
		}
		if fail {
			os.Exit(1)
		}
		fmt.Println("PASS: all three probes correct")
	})

	wnd.Show()
	sdl3.Start(nil)
}
