# WEB.md — web-looking scenes on the meui foundation

**Status:** first slice done. The Box, the interaction primitive, the web preset and
the screenshot loop exist; two scenes (`examples/landing`, `examples/dashboard`) are
built and rendered. The gap queue below is what the next scenes will pull from.

This is the record of *why* the foundation changed, so the next person can tell a
deliberate simplification from a missing feature.

---

## 1. The goal

The toolkit is enterprise-first: a dense desktop IDE look with layout managers,
tab panels and a data table. It must also be able to build a *web page* — the kind
of thing in `~/Downloads/example-pics`: a landing page with a display heading and
pill CTAs, a weather dashboard whose cards float on a gradient sky, an art-directed
scene with its own palette.

"Look at Qt QML for the ideas" is the instruction, and it is the right one: not Qt's
engine (no JS, no bindings — this toolkit wires behaviour from Go by id), but Qt's
**vocabulary of surfaces**: `Item`, `Rectangle` with `radius`/`border`/`gradient`/
`clip`, `Text` with weights, elision and line counts, `MouseArea` with hover and
cursor, `anchors`, `Layout`.

The rule that keeps this from becoming a second toolkit:

> **New behaviour → a new widget. New look → a composition of Box + MouseArea + Text,
> or a theme variant.** Nav bars, heroes, cards, badges, chips and charts are
> compositions. DataTable, TextField, Switch and tabs keep their behaviour and get
> web *values* from the preset.

---

## 2. What the foundation had, and what it was missing

Measured before changing anything, by rendering a scene headlessly:

| gap | evidence |
|---|---|
| `Rectangle { radius: 8 }` painted **square** | the builder parsed `radius` and dropped it (`SetCornerRadius removed`); the corner pixel was border ink |
| no shadow, gradient, clip or opacity on a box | `Panel.Paint` was one `DrawRect` + optional stroke; `opacity`/`z` were advertised in the QML registry and never applied |
| a box could not be clicked | `Panel` had Go callbacks but no state model, no cursor, no keyboard activation, no way to say "this box is interactive" |
| nested ids never resolved | the bindings builder built children through the plain builder first, so `getWidgetFromObject` returned them before registration — every id inside a `Rectangle` was dead |
| `x`/`y` were ignored in the bindings path | the fix for the id problem exposed it: special widgets (Rectangle/Text/layouts) skipped `applyCommonProperties` |
| software text was 1.278× too large | draw2d sizes in points at 92 DPI; the seam's `Font.Size` is px |
| software translucent fills were stained | straight colour handed to a painter that expects premultiplied: a 0.8-alpha mint over cream came out (15,29,9) |
| software rendering had no rounded corners | `DrawRoundedRect` delegated to `DrawRect` |
| `border: { … }` and `gradient: { … }` painted nothing | the parser keeps object literals as source TEXT; the handlers only accepted `map[string]interface{}` |

Each of those is now pinned by a test that samples pixels or fires the real input
pipeline — not by a field being set.

---

## 3. What a scene can use now

```
Rectangle {                      // the Box: Panel grown up
  x, y, width, height
  color: "#14243C" | "accent"    // a hex value or a theme token
  radius: 12
  border.width: 1  border.color: "#E5E5E5"   // or border: { color: …, width: … }
  gradient: { from: "#7FC0E8", via: "#B9E0F1", to: "#EAF7FC", angle: 180 }
  shadow: "card" | "popup" | "window" | "none"
  clip: true
  opacity: 0.8
  padding: 16
  clickable: true
  selected: true                 // the theme's SelectedWash, under hover/press
  z: 10                          // sibling paint/hit order (higher is on top)
  blur: true                     // frosted: blur the backdrop, then tint over it
  sticky: true                   // pin to the viewport while scrolled past
  cursor: "pointing"
  hoverColor / pressedColor      // override the theme wash
}

Item { … }                       // a box with nothing painted
Text { text, font.pixelSize, font.weight, lineHeight, maximumLineCount,
       elide, wrapMode, horizontalAlignment, verticalAlignment, color }
MouseArea { anchors.fill: parent; cursor; focusable; onClicked/onPressed/… }
ScrollView { anchors.fill: parent; contentWidth; contentHeight; frameless: true }
Image { source: "…png"; fillMode: "cover" | "contain" | "fill" | "none"; radius }
```

Behaviour is wired from Go:

```go
scene, _ := web.Load("page.qml", "web-light", 1280, 900)
scene.OnClick("ctaPrimary", func() { … })     // a clickable box OR a MouseArea
scene.OnHover("ctaPrimary", func(in bool) { … })
scene.SetText("statusText", "…")
scene.OnPaint("demoPlay", func(c *graphics.Canvas, r graphics.Rect) { … })
scene.SetFrame("article", 0, 0, 900, 700)     // place by id (responsive)
scene.Show("sidebar", false)                  // hide by id
scene.Select("navIntro", true)                // the theme's selection wash
scene.Breakpoints([]web.Breakpoint{           // reflow from the window's resize event
    {MinWidth: 0,    Apply: compact},
    {MinWidth: 1000, Apply: wide},
})
scene.ScrollTo("messageView", 770)            // reveal a row in a ScrollView
scene.Shot(1280, 900, "page.png")             // headless, no display
```

## 3b2. A list of a thousand rows

`VirtualList` builds rows from a Go function as they enter the viewport:

```qml
VirtualList { id: messageView; x: 12; y: 84; width: 356; height: 800
              rowHeight: 110; topInset: 40 }
```
```go
vl := scene.Get("messageView").(*widgets.VirtualList)
vl.SetRowCount(len(inbox))
vl.SetRowBuilder(func(i int) meui.Widget { return buildRow(i) })  // a Go row TEMPLATE
vl.EnsureVisible(selected)                                        // the click reveal
```

It is a ScrollPane underneath (wheel, bars, drag, clipped content), so the interaction it
inherits is the one every other pane has. The visible range plus two rows of overscan are
built; rows that leave the window are dropped. `topInset` is a sticky header's height: the
first row starts below it and later rows scroll under it.

There is deliberately no recycling of a widget for a DIFFERENT index. A row's bindings —
texts, colours, the click closure — are built into it, so rebinding would need a second
lifecycle API every row widget had to implement. Build-on-demand + drop is the honest first
step; a `RowBinder` is the extension point if a heavy row widget ever needs recycling. What
this does buy is the number that matters: the mail scene went from one widget per message
(twelve) to 240 messages and a dozen widgets.

## 3b3. Sticky sections

`Rectangle { sticky: true }` inside a scrolling content pins to the top of the viewport once
the scroll passes it, and stops at its parent section's bottom — `position: sticky`, clamped.
`stickyInset` is for a fixed bar the pane does not know about.

The pane does it by MOVING THE WIDGET'S FRAME, not by translating it while painting. Frames
are what the hit test, hover, the cursor walk and the tooltip chain all read; a paint-time
offset would leave every one of them at the old position, so a pinned header would be
visible where it is not clickable. The pane remembers each sticky box's position in the
content's flow (in the pane, not the widget — the frame is what moves), restores it when the
scroll comes back, and clamps to `parent bottom - height` so a header leaves with its
section.

**Give a pinned header a `z`.** Painting follows declaration order, so a sticky box declared
before the content that scrolls under it is drawn UNDER that content. `sticky: true; z: 10`
is the pair that makes a header pin above the page — the docs scene does exactly that.

The docs scene is the demonstration: `go run ./examples/docs -scroll 760 -shot out.png`
pins "The box is the primitive" under the top bar while the code block passes beneath it.

This work also found a real hit-test defect in `ScrollPane.PanelAt`: it compared the
content-space point against `host.ContentSize`, which is the VIEWPORT (frame minus bars), not
the content's extent — so nothing below the first screen of a scrolled pane could be clicked.
It also treated "x is past the bar's left edge" as "on the bar", which swallowed the whole
viewport while the bars' frames were still zero. Both are point-in-rect now, for the bars and
for the content extent.

## 3b4. Motion

A Box's hover and press washes CROSSFADE on the theme's Motion tokens (Hover, Press)
instead of switching, through `meui.Animation` — evaluated lazily at paint from the frame
clock, so a running transition simply re-renders until it settles. The hover wash fades out
as the press wash fades in, so a press never doubles the ink and a leave animates rather than
snapping.

`Panel.FinishAnimations()` snaps the transitions to their targets: tests use it to measure a
settled state deterministically, and a reduce-motion app would call it after setting state.
The timing is a theme token, so the web preset can be crisper than the desktop one without
any widget knowing.

## 3c. Two seams a scene needs that are not QML

- **A display face.** `graphics.RegisterFontFamily("Editorial", path, weight, slant)`
  loads a TTF and makes `font.family: "Editorial"` resolve through `MatchFontFace`
  (an unregistered family still returns nil, so the embedded-Roboto rule is intact).
  The demo registers DejaVu Serif from the machine if it exists; a scene should ship
  its own TTF rather than depend on one. OTF is still rejected by the parser.
- **A scrolling page's content is hit-tested.** A `ScrollView` opts its content tree
  into the window's hit test (`ScrollPane.SetHitTestContent`), so a message row inside
  a scrolling viewport is a real target. A plain `ScrollPane` keeps the pane-owned
  presses it drag-scrolls with — that is why this is opt-in, and why the enterprise
  list behaves exactly as before.
- **Reveal.** `ScrollPane.ScrollTo(y)` (and `web.ScrollTo`) scrolls so content y is at
  the viewport's top edge, clamped; it establishes the extents itself, because a scene
  that selects a row as it loads runs before the first paint.

`meui.Window.SetOnResize` is the hook underneath: it fires once per ACTUAL size change,
from the resize event — not from the paint path, which calls `SetFrameRect` constantly.

`shadow: "card"` resolves through `theme.Effects.CardShadow`; the web preset is
`web-light` / `web-dark` (`PlatformWeb`): an 8px grid, 40px controls, 8px radii, a
document type scale (9/10/11/14/24), the measured palette and a 2px focus ring. Sizes
are CAP HEIGHTS, not ems: `Font.Size` is the requested cap height and `FontImpl.emPx`
solves for it against the face (the same loop GL's atlas uses), so a 11 is the ~15px
em a browser body uses. The scenes were authored against screenshots taken while the
software device drew at the raw size — 30-40% small, and, because the metrics were not,
with every centred label a few pixels off-centre. The device now draws at the em too,
and the scene type scale was divided by 0.72 to match. A
selected surface uses `SelectedWash` — the accent made translucent — because the
palette's `Selected` is an opaque fill role (selected text, a checked box) and washing a
row with it paints solid accent over the row's own label. The
preset also restyles the CONTROLS through a post-generation tweak: a web primary
Button is accent-filled and borderless, inputs are white with a hairline edge and an
accent focus edge, and the combo matches them. The tweak has to run after `Apply`,
because the generator rebuilds every widget theme from the tokens.

---

## 3b. The scenes

The runnable copies live in [`examples/`](meui/examples/README.md) — six `go run`s plus the
launcher — and `examples/README.md` is the public guide (how to run, every flag, what each
scene shows). This section is the design record behind them.

| scene | what it proves | the shape |
|---|---|---|
| `examples/landing` | display type, boxes with radius/shadow/clip, pill CTAs, a MouseArea region, a painted triangle | a marketing page |
| `examples/dashboard` | a full-bleed gradient, translucent cards over it, a large figure, a bar chart made of boxes, a selectable day strip restyled from Go | a weather dashboard |
| `examples/docs` | a fixed chrome + a SCROLLING article, a real image with `fillMode: "cover"` and `radius`, a sidebar whose selection is a box restyle, and a BREAKPOINT: below 1000px the sidebar goes and the article spans the page | a docs shell |
| `examples/mail` | a three-column inbox: 240 messages in a VIRTUAL list (a dozen widgets exist), list selection moved from Go, reveal-on-select, a FROSTED header the rows scroll under, an app-supplied display face in the reading pane | a mail client |
| `examples/checkout` | a three-step form flow with VALIDATION: an inline error blocks the step and colours the field's focus ring, the indicator tracks done/active/upcoming, and the last step replaces the flow with a success panel | a checkout |
| `examples/settings` | the CONTROLS under the web preset — TextField, ComboBox, Switch, CheckBox, RadioButton, a primary Button — plus a save bar and a toast whose visibility Go flips | a settings form |

## 4. The measured references

Sampled from `~/Downloads/example-pics` with a 4-bit-per-channel histogram, so these
are *cluster* centres, not exact colours; the percentages are area shares of the
top colour. The point is the temperature of each palette, not a byte-exact swatch.

| reference | dominant surfaces | accent / ink | what it proves |
|---|---|---|---|
| `poster.jpg` (landing) | `#FCF4DC` 59%, `#DCECDC` 12% | navy `#14243C`, pink highlight | display type + a thick offset card, outlined pills |
| nimbus (weather) | sky `#B4DCEC`→`#CCE4EC`, white cards | deep blue ink | a full-bleed gradient with translucent cards on it |
| hummock (music) | `#0C3C74` 22%, `#0C4474` 6% | off-white `#F4FCFC` | one saturated field, chunky rounded art cards |
| hi-fi (turntable) | `#24342C`/`#2C3C34` deep green, cream `#E4DCD4` | brass/amber | the palette itself is the art direction |
| soft machine (synth) | clay `#D4D4DC` 36%, `#D4DCDC` 17% | `#e4e8ee` + coral | soft shadows and one surface, no chrome |

The preset is the neutral scaffold; these palettes are **per-scene overrides**. A
scene that wants cream/navy/pink sets those on its boxes (as `landing.qml` does); a
scene that wants a sky sets a gradient (as `dashboard.qml` does). Neither is a new
preset, because the primitives are the same.

`harness-light`'s palette (white / `#F9FAFB` / `#4176E6`, measured in
`LOOK-REFERENCE.md`) is the base `web-light` reuses.

---

## 5. The loop

```
scene → screenshot (1× and 2×, plus states) → gap list: scene · missing capability · workaround
      → fix the PRIMITIVE or the preset → re-shoot the scene
      → regression-shoot the other scenes → delete the workaround
```

Governing rules:

- **A workaround may not appear twice.** Twice means it becomes a primitive.
- A capability is built when a scene needs it, not when it sounds useful.
- Every fix gets a headless test (pixels or the real input pipeline) plus the
  scene's screenshot.
- Scenes are compositions. If a scene needs a *second* copy of the same 20-line
  composition, that is the signal for a widget.

Screenshots:

```sh
cd meui
go run ./cmd/tools/qmlshot -preset web-light -w 1280 -h 900 page.qml   # any scene, headless
go run ./examples/landing -shot out.png                         # a scene WITH its Go wiring
go run ./examples/landing                                       # a real window
```

---

## 6. Where the next scenes would push

Everything the five scenes needed is in: surfaces, interaction, scrolling, images,
responsive layouts, selection, z order, custom fonts, frosted glass, and an
offscreen/GL parity check. What is deliberately NOT there yet, in the order a sixth
scene would reach for it:

2. **Text details** — the engine's `elide` now ends in "…" rather than three periods (the
   text core owns the truncation). There is still no letter-spacing: the seam's Font carries
   a Spacing ENUM (standard/expanded), not a numeric tracking value, so adding tracking means
   extending the font model, not just a widget.

Deliberately NOT planned: a JS/expression engine, a second layout system, a
component library of 40 web widgets. The vocabulary above composes them.

---

## 6a. Frosted glass

`Rectangle { blur: true }` blurs what is ALREADY PAINTED behind the box and then paints the
box's own translucent fill and border on top — a frosted card, or the header a list scrolls
under (the mail scene's inbox title strip).

The strength is a renderer constant, not a per-box radius: the GL path's kernel is a fixed
separable 7-tap Gaussian (`shaders/blur_fs.glsl`, shared with its shadows) and the software
device applies THE SAME 7 taps. That is the point — a radius in one renderer and a constant
in the other is drift by construction.

The GL path blurs on the CPU after reading the region back, then draws the result through the
ordinary image path. The renderer's blur FBO is built for shadows (draw a shape into it,
blur, composite); reusing it to resample the existing framebuffer put its own program and
attribute layout in the middle of the main batch and produced a giant triangle across the
page. The readback is a pipeline stall, so a frosted panel should be a few per page, not many.

The glass is masked to the box's ROUNDED silhouette with one coverage function shared by
both devices (`roundedCoverage`): the software composites through it per pixel, and GL applies
it to the readback's alpha before upload (`maskRoundedCorners`). A rounded frosted card
therefore has the same antialiased corners offscreen and live — the parity scene's card has
radius 18 and the two frames agree within the usual edge noise.

## 6b. GL and the offscreen shot agree — and how to check

`cmd/tools/qmlshot` renders through the software device, `cmd/tools/glshot` through the real
SDL3/GL pipeline, and `cmd/tools/imgdiff` compares them. `examples/parity/parity.qml` is the
scene: SURFACES ONLY (no text — two text rasterizers never agree byte-for-byte), built
from a bordered card, a shadowed card, a gradient, a translucent fill over a solid,
a rounded clip, a rounded cover image, a z pair and a group opacity.

```sh
cd meui
CGO_ENABLED=0 go run ./cmd/tools/qmlshot -preset web-light -w 800 -h 600 -o /tmp/soft.png examples/parity/parity.qml
CGO_ENABLED=0 go run ./cmd/tools/glshot    -preset web-light -w 800 -h 600 -o /tmp/gl.png   examples/parity/parity.qml
CGO_ENABLED=0 go run ./cmd/tools/imgdiff   /tmp/soft.png /tmp/gl.png
```

Measured after the clip fix: **meanΔ 0.129, 0.54% of pixels off by more than 3/channel**
— the residue is antialiased rounded-corner arcs and a little gradient banding, which
two rasterizers will not agree on. `make parity-web` runs the three steps (on a machine
that has `make`).

Two bugs came out of this check and the frosted panel it carries. The first is that the
software device did not clip SHAPES at all:
draw2d has no clip, and the device only culled fully-outside draws, so a `clip: true`
panel's children painted over its edges (a pink block 100px past its rounded clip box in
the parity scene). The device now snapshots the surface when a clip narrows and restores
everything outside it when the scope ends — only the offscreen device pays, because GL
has a real scissor. The second: `ColorDecode` read EIGHT-digit hex as something other than
AARRGGBB (`#CCFFFFFF` became `0x00FFFFCC`, alpha 0), so every translucent colour a scene
wrote was invisible — the frosted box never drew, and the parity scene's translucent overlay
was silently missing. `parseColor` now reads eight digits itself.

## 7. Verification

- `graphics`: soft-device text size and, since the size is a CAP HEIGHT, that the
  device draws at the font's em (the ink of an "H" at size N is N pixels, not 0.7N), rounded corners, gradients, translucent
  compositing, images through the transform, images clipped to the canvas clip.
- `meui`: Box radius/gradient/clip/opacity, the selected wash (and that a hover does
  not replace it), z ordering (paint and hit, with insertion order for equal z), and
  the ClickBehaviour state machine.
- `widgets`: an Image's rounded-corner layer (`radius`) is a mask at display size, and
  a masked image paints no plate behind it.
- `behaviour`: click-through-a-label, release-outside cancels, hover arrival and
  departure, Enter/Space activation, MouseArea as an independent region, ScrollView
  scroll/clamp and clicks inside scrolled content.
- `web`: an id that does not exist is reported; a scene shoots headlessly; breakpoints
  reflow on a real resize event and NOT on a same-size one.
- `examples/mail`: a real click selects exactly one row and updates the reading pane,
  and choosing a row below the fold scrolls the list by the overflow.
- `graphics`: an app-supplied family resolves while an unregistered one still returns
  nil.
- `widgets`: a sticky header pins at the viewport top, clamps at its section's bottom,
  returns to its natural place when scrolled back, and takes the click where it is drawn.
- `widgets`: a virtual list builds only the visible rows (a 300px viewport over 1000 rows
  builds at most twelve), drops them when they leave the window, and reveals a row below
  the fold with the inset respected.
- `widgets`: a frosted box blurs a hard backdrop edge into intermediate grey (the kernel is
  the renderer's, so the test asserts the operation, not a strength).
- `graphics`: the software blur's kernel and the GL readback path.
- `qml`: eight-digit colours decode as AARRGGBB, and z reaches any widget.
- `examples/checkout`: an empty form blocks its step with visible inline errors, a valid
  flow reaches the success panel with the review's data, and the terms gate the order.
- `widgets`: a focused field with an error rings in the error ink, not the accent.
- `widgets`: the label's wrap strategies are the CORE's (word wrap, ellipsis,
  maximumLineCount, rune-safe truncation) — the widget delegates.
- `qml`: nested ids resolve; object-literal properties parse.
- Scenes: `examples/*/main.go -shot`, `cmd/tools/qmlshot`, and the GL/offscreen parity pair.

### The enterprise look: capture, change, compare

`examples/enterprise` is the imported QML demo (the kitchen sink), and it can screenshot itself:

```sh
cd meui/examples/enterprise
MEUI_CAPTURE_SHOT=/tmp/ent_before.png MEUI_CAPTURE_SECS=4 go run .

# … make the change …

MEUI_CAPTURE_SHOT=/tmp/ent_after.png MEUI_CAPTURE_SECS=4 go run .
cd .. && go run ./cmd/tools/imgdiff -diff /tmp/ent_diff.png /tmp/ent_before.png /tmp/ent_after.png
```

The demo is deterministic between runs while nothing touches its window: two captures
of the same build measured **meanΔ 0.000**, full frame. (One session looked noisy — the
theme list appeared at different scroll offsets — and the cause was the OWNER scrolling
that list by mistake while the captures ran. It was input, not rendering.)

The saved baseline in `.shots/enterprise-before.png` was itself taken after that list had
been scrolled, so comparing today's default state against it reads 1.70% of pixels, all
of them inside the theme column, and **0.000 everywhere else**. Either capture a fresh
baseline on a calm machine, or mask the column and expect zero:

```sh
go run ./cmd/tools/imgdiff -ignore 840,80,260,670 /tmp/ent_before.png /tmp/ent_after.png
# 1100x750  meanΔ 0.000  changed 0.00%  big(>3/ch) 0.000%  (ignored 174200 px)
```

With the column masked, the enterprise frame is **pixel-identical to the baseline across
every change** in this document — surfaces, text scale, clip enforcement, blur, sticky,
motion. `.shots/enterprise-clean.png` is an undisturbed capture of the current build.
`-ignore` takes `x,y,w,h` and repeats with `;`.
