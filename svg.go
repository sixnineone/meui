package meui

import (
	"embed"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sixnineone/meui/graphics"
	"github.com/sixnineone/meui/graphics/enums/paintstyle"
)

//go:embed assets/icons/*.svg
var svgEmbedFS embed.FS

// SVGElement represents a parsed SVG element
type SVGElement struct {
	Type     string
	Attrs    map[string]string
	Children []SVGElement
	Text     string
}

// SVGDocument represents a parsed SVG file
type SVGDocument struct {
	Width    float32
	Height   float32
	ViewBox  [4]float32
	Elements []SVGElement
}

// SVGIcon caches loaded SVG documents
var svgCache = make(map[string]*SVGDocument)

// svgSearchPaths contains directories to search for SVG files
var svgSearchPaths = []string{
	"assets/icons/",
	"../assets/icons/",
	"../../assets/icons/",
	"../../../assets/icons/",
}

// LoadSVG loads and parses an SVG file
func LoadSVG(path string) (*SVGDocument, error) {
	if doc, ok := svgCache[path]; ok {
		return doc, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read SVG file: %w", err)
	}

	doc, err := parseSVG(string(data))
	if err != nil {
		return nil, err
	}

	svgCache[path] = doc
	return doc, nil
}

// LoadSVGByName loads an SVG from the embedded assets/icons directory
func LoadSVGByName(name string) (*SVGDocument, error) {
	if !strings.HasSuffix(name, ".svg") {
		name += ".svg"
	}

	// Check cache first
	if doc, ok := svgCache[name]; ok {
		return doc, nil
	}

	// Try embedded filesystem first
	if data, err := svgEmbedFS.ReadFile("assets/icons/" + name); err == nil {
		doc, err := parseSVG(string(data))
		if err == nil {
			svgCache[name] = doc
			return doc, nil
		}
	}

	// Fallback: try each search path
	for _, searchPath := range svgSearchPaths {
		fullPath := filepath.Join(searchPath, name)
		if _, err := os.Stat(fullPath); err == nil {
			return LoadSVG(fullPath)
		}
	}

	// Fallback: try absolute path from module root
	wd, _ := os.Getwd()
	absPath := filepath.Join(wd, "assets/icons", name)
	if _, err := os.Stat(absPath); err == nil {
		return LoadSVG(absPath)
	}

	return nil, fmt.Errorf("SVG icon not found: %s", name)
}

// DrawSVG renders an SVG document onto a canvas at the specified position.
// The vector is rasterized ONCE at 4x, box-filtered to the target size
// (a true 16-tap area average — see boxFilter4x) and drawn 1:1
// pixel-aligned, matching Skia's analytic stroke coverage closely.
func DrawSVG(canvas *graphics.Canvas, doc *SVGDocument, x, y, width, height float32) {
	if doc == nil || width < 1 || height < 1 {
		return
	}

	w, h := int(width+0.5), int(height+0.5)
	key := svgImgKey{doc: doc, w: w, h: h}

	// Icons made purely of primitives (circle, rect, axis-aligned line,
	// text) skip the supersample raster entirely: they draw straight
	// through the analytic SDF/text pipelines at final resolution —
	// exact per-fragment coverage, the same quality as widget shapes.
	if svgDirect(doc.Elements) {
		x = float32(int(x + 0.5))
		y = float32(int(y + 0.5))
		drawSVGDirect(canvas, doc, x, y, float32(w), float32(h))
		return
	}

	// A key the backend refused to rasterise offscreen is remembered, so the
	// attempt and its note happen once per icon+size instead of once per draw.
	if svgNoOffscreen[key] {
		drawSVGVectorAt(canvas, doc, x, y, w, h)
		return
	}

	img, ok := svgImageCache[key]
	if !ok {
		rw, rh := w*4, h*4
		big, err := graphics.NewImageFromDrawing(canvas.Device(), rw, rh, 72, func(c *graphics.Canvas) {
			drawSVGVector(c, doc, 0, 0, float32(rw), float32(rh))
		})
		if err != nil {
			// The backend refused the offscreen pass - e.g. the app runs with
			// MEUI_NO_OFFSCREEN=1 because an offscreen render mutates renderer
			// state shared by every window. Skipping the icon (the old
			// behaviour) lost it silently AND retried every frame, printing a
			// line per draw. Draw it directly through the same analytic
			// pipelines every other shape uses: softer than the 4x box-filtered
			// raster, but the icon is drawn, the decision is cached, and the
			// note is one line per icon.
			fmt.Printf("[drawsvg] offscreen unavailable, drawing directly: %v\n", err)
			svgNoOffscreen[key] = true
			drawSVGVectorAt(canvas, doc, x, y, w, h)
			return
		}
		img = boxFilter4x(big, w, h)
		svgImageCache[key] = img
		if os.Getenv("MEUI_LAYOUT_DEBUG") != "" {
			fmt.Printf("[drawsvg] rasterized %dx%d -> box %dx%d -> draw at (%.1f,%.1f %vx%v)\n", rw, rh, w, h, x, y, width, height)
		}
	}
	// Pixel-align: a 1:1 texture quad only maps texel centers to pixel
	// centers when the destination is integer-aligned; otherwise LINEAR
	// blurs every edge.
	x = float32(int(x + 0.5))
	y = float32(int(y + 0.5))
	canvas.DrawImageInRect(img, graphics.NewRect(x, y, float32(w), float32(h)), nil, nil)
}

// drawSVGVectorAt draws the raw vector at final resolution, pixel-aligned -
// the fallback when the supersampled offscreen raster is unavailable.
func drawSVGVectorAt(canvas *graphics.Canvas, doc *SVGDocument, x, y float32, w, h int) {
	drawSVGVector(canvas, doc, float32(int(x+0.5)), float32(int(y+0.5)), float32(w), float32(h))
}

// boxFilter4x area-averages a 4x-supersampled bottom-up ARGB image
// down to w*h. Bilinear minification would sample 4 of the 16 texels
// per output pixel (ragged thin strokes); the box filter averages all
// 16 — near-analytic coverage. Channels are premultiplied, so linear
// averaging composites correctly.
func boxFilter4x(big *graphics.Image, w, h int) *graphics.Image {
	src := big.Pixels()
	rw := int(big.Width())
	out := make([]uint32, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var a, r, g, b int
			for dy := 0; dy < 4; dy++ {
				row := (y*4 + dy) * rw
				for dx := 0; dx < 4; dx++ {
					px := src[row+x*4+dx]
					a += int((px >> 24) & 0xFF)
					r += int((px >> 16) & 0xFF)
					g += int((px >> 8) & 0xFF)
					b += int(px & 0xFF)
				}
			}
			out[y*w+x] = uint32(a/16)<<24 | uint32(r/16)<<16 | uint32(g/16)<<8 | uint32(b/16)
		}
	}
	return graphics.NewImage(out, int32(w), int32(h))
}

type svgImgKey struct {
	doc *SVGDocument
	w   int
	h   int
}

var svgImageCache = make(map[svgImgKey]*graphics.Image)

// svgNoOffscreen records icon+size keys whose supersampled offscreen raster the
// backend refused, so the direct-draw fallback is chosen once and neither the
// attempt nor its note repeats on every frame. Same single-thread (UI) use as
// svgImageCache.
var svgNoOffscreen = make(map[svgImgKey]bool)

// drawSVGVector draws the raw vector elements scaled to fit (w,h) at (x,y).
func drawSVGVector(canvas *graphics.Canvas, doc *SVGDocument, x, y, width, height float32) {
	scaleX := width / doc.Width
	scaleY := height / doc.Height
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	offsetX := x + (width-doc.Width*scale)/2
	offsetY := y + (height-doc.Height*scale)/2

	canvas.Save()
	canvas.Translate(offsetX, offsetY)
	canvas.Scale(scale, scale)

	for _, el := range doc.Elements {
		drawSVGElement(canvas, el, scale)
	}

	canvas.Restore()
}

// svgDirect reports whether every element can be drawn analytically at
// final resolution: FILLED circles/rects (no strokes — a 1.5px SDF
// stroke band reads bolder and dirtier than the box-filtered raster,
// which matches Skia's partial-coverage softness) and text. Anything
// else — stroked shapes, paths, polylines, polygons, lines — uses the
// supersample raster.
func svgDirect(els []SVGElement) bool {
	if len(els) == 0 {
		return false
	}
	for _, el := range els {
		switch el.Type {
		case "circle", "rect":
			fill := el.Attrs["fill"]
			if fill == "" || fill == "none" {
				return false
			}
			if s := el.Attrs["stroke"]; s != "" && s != "none" {
				return false
			}
		case "text":
			continue
		default:
			return false
		}
	}
	return true
}

// drawSVGDirect draws primitive-only icon elements at 1x through the
// analytic pipelines.
func drawSVGDirect(canvas *graphics.Canvas, doc *SVGDocument, x, y, width, height float32) {
	scaleX := width / doc.Width
	scaleY := height / doc.Height
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	offsetX := x + (width-doc.Width*scale)/2
	offsetY := y + (height-doc.Height*scale)/2

	canvas.Save()
	canvas.Translate(offsetX, offsetY)
	canvas.Scale(scale, scale)

	for _, el := range doc.Elements {
		drawSVGElement(canvas, el, scale)
	}

	canvas.Restore()
}

func parseSVG(data string) (*SVGDocument, error) {
	decoder := xml.NewDecoder(strings.NewReader(data))
	doc := &SVGDocument{Width: 16, Height: 16}

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch el := token.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "svg":
				for _, attr := range el.Attr {
					switch attr.Name.Local {
					case "width":
						doc.Width = parseFloat(attr.Value)
					case "height":
						doc.Height = parseFloat(attr.Value)
					case "viewBox":
						parts := strings.Fields(attr.Value)
						if len(parts) == 4 {
							doc.ViewBox[0] = parseFloat(parts[0])
							doc.ViewBox[1] = parseFloat(parts[1])
							doc.ViewBox[2] = parseFloat(parts[2])
							doc.ViewBox[3] = parseFloat(parts[3])
						}
					}
				}
			case "circle":
				doc.Elements = append(doc.Elements, parseShape(el, "circle"))
			case "rect":
				doc.Elements = append(doc.Elements, parseShape(el, "rect"))
			case "line":
				doc.Elements = append(doc.Elements, parseShape(el, "line"))
			case "polyline":
				doc.Elements = append(doc.Elements, parseShape(el, "polyline"))
			case "polygon":
				doc.Elements = append(doc.Elements, parseShape(el, "polygon"))
			case "path":
				doc.Elements = append(doc.Elements, parseShape(el, "path"))
			case "text":
				doc.Elements = append(doc.Elements, parseText(el))
			}
		}
	}

	return doc, nil
}

func parseShape(el xml.StartElement, shapeType string) SVGElement {
	elem := SVGElement{Type: shapeType, Attrs: make(map[string]string)}
	for _, attr := range el.Attr {
		elem.Attrs[attr.Name.Local] = attr.Value
	}
	return elem
}

func parseText(el xml.StartElement) SVGElement {
	elem := SVGElement{Type: "text", Attrs: make(map[string]string)}
	for _, attr := range el.Attr {
		elem.Attrs[attr.Name.Local] = attr.Value
	}
	return elem
}

// effectiveStroke converts an SVG stroke-width (in SVG units) to the
// device-pixel width for the current DrawSVG scale (Skia scales strokes
// by the CTM the same way).
func effectiveStroke(el SVGElement, scale float32) float32 {
	sw := float32(1)
	if v := el.Attrs["stroke-width"]; v != "" {
		sw = parseFloat(v)
	}
	return sw * scale
}

func drawSVGElement(canvas *graphics.Canvas, el SVGElement, scale float32) {
	switch el.Type {
	case "circle":
		drawCircle(canvas, el, scale)
	case "rect":
		drawRect(canvas, el, scale)
	case "line":
		drawLine(canvas, el, scale)
	case "polyline":
		drawPolyline(canvas, el, scale)
	case "polygon":
		drawPolygon(canvas, el)
	case "path":
		drawPath(canvas, el, scale)
	case "text":
		drawText(canvas, el)
	}
}

// buildSvgPath parses an SVG path `d` attribute into a Path, offset by
// (x,y) and scaled by scale. Supports M/m, L/l, H/h, V/v, C/c, S/s, Q/q,
// T/t, A (flattened to segments), and Z/z.
func buildSvgPath(d string, x, y, scale float32) *graphics.Path {
	if strings.TrimSpace(d) == "" {
		return nil
	}
	path := graphics.NewPath()
	var cx, cy, startX, startY float32
	// tokenise into numbers + command letters
	var cmd byte
	var nums []float32
	i := 0
	nextNum := func() (float32, bool) {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\n' || d[i] == '\r' || d[i] == '\t') {
			i++
		}
		if i >= len(d) {
			return 0, false
		}
		start := i
		if d[i] == '-' || d[i] == '+' {
			i++
		}
		for i < len(d) && (d[i] >= '0' && d[i] <= '9' || d[i] == '.') {
			i++
		}
		if start == i {
			return 0, false
		}
		var v float32
		fmt.Sscanf(d[start:i], "%g", &v)
		return v, true
	}
	applyCmd := func(rel bool) {
		switch cmd {
		case 'M':
			if len(nums) >= 2 {
				cx, cy = nums[0], nums[1]
				if rel {
					cx, cy = x+cx*scale, y+cy*scale
				} else {
					cx, cy = x+cx*scale, y+cy*scale
				}
				startX, startY = cx, cy
				path.MoveTo(graphics.Point{X: cx, Y: cy})
				for j := 2; j+1 < len(nums); j += 2 {
					cx, cy = x+nums[j]*scale, y+nums[j+1]*scale
					path.LineTo(graphics.Point{X: cx, Y: cy})
				}
			}
		case 'L':
			for j := 0; j+1 < len(nums); j += 2 {
				nx, ny := nums[j], nums[j+1]
				if rel {
					cx += nx * scale
					cy += ny * scale
				} else {
					cx, cy = x+nx*scale, y+ny*scale
				}
				path.LineTo(graphics.Point{X: cx, Y: cy})
			}
		case 'H':
			for _, nx := range nums {
				if rel {
					cx += nx * scale
				} else {
					cx = x + nx*scale
				}
				path.LineTo(graphics.Point{X: cx, Y: cy})
			}
		case 'V':
			for _, ny := range nums {
				if rel {
					cy += ny * scale
				} else {
					cy = y + ny*scale
				}
				path.LineTo(graphics.Point{X: cx, Y: cy})
			}
		case 'C', 'S':
			for j := 0; j+5 < len(nums); j += 6 {
				x1, y1, x2, y2, nx, ny := nums[j], nums[j+1], nums[j+2], nums[j+3], nums[j+4], nums[j+5]
				if rel {
					x1, y1, x2, y2, nx, ny = x1*scale, y1*scale, x2*scale, y2*scale, nx*scale, ny*scale
					x1, y1 = cx+x1, cy+y1
					x2, y2 = cx+x2, cy+y2
					cx, cy = cx+nx, cy+ny
				} else {
					x1, y1, x2, y2 = x+x1*scale, y+y1*scale, x+x2*scale, y+y2*scale
					cx, cy = x+nx*scale, y+ny*scale
				}
				// mobilegles.Path has no CubicTo — approximate the cubic
				// with two quadratic curves through the midpoint.
				mx, my := (x1+2*x2+cx)/4, (y1+2*y2+cy)/4
				path.QuadTo(
					graphics.Point{X: (x1 + 2*x2) / 3, Y: (y1 + 2*y2) / 3},
					graphics.Point{X: mx, Y: my},
				)
				path.QuadTo(
					graphics.Point{X: (x2 + 2*cx) / 3, Y: (y2 + 2*cy) / 3},
					graphics.Point{X: cx, Y: cy},
				)
			}
		case 'Q', 'T':
			for j := 0; j+3 < len(nums); j += 4 {
				qx, qy, nx, ny := nums[j], nums[j+1], nums[j+2], nums[j+3]
				if rel {
					qx, qy, nx, ny = qx*scale, qy*scale, nx*scale, ny*scale
					qx, qy = cx+qx, cy+qy
					cx, cy = cx+nx, cy+ny
				} else {
					qx, qy = x+qx*scale, y+qy*scale
					cx, cy = x+nx*scale, y+ny*scale
				}
				path.QuadTo(
					graphics.Point{X: qx, Y: qy},
					graphics.Point{X: cx, Y: cy},
				)
			}
		case 'A':
			// Flatten arcs into line segments (7 params each).
			for j := 0; j+6 < len(nums); j += 7 {
				nx, ny := nums[j+5], nums[j+6]
				if rel {
					cx += nx * scale
					cy += ny * scale
				} else {
					cx, cy = x+nx*scale, y+ny*scale
				}
				path.LineTo(graphics.Point{X: cx, Y: cy})
			}
		}
	}
	for i < len(d) {
		ch := d[i]
		switch {
		case ch == ' ' || ch == ',' || ch == '\n' || ch == '\r' || ch == '\t':
			i++
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z'):
			cmd = ch
			i++
			nums = nums[:0]
			for {
				v, ok := nextNum()
				if !ok {
					break
				}
				nums = append(nums, v)
			}
			rel := cmd >= 'a'
			c := cmd
			if rel {
				c = cmd - 32
			}
			if c == 'Z' {
				path.Close()
				cx, cy = startX, startY
				continue
			}
			applyCmd(rel)
		default:
			i++
		}
	}
	return path
}

// parsePathValues extracts all numeric values from an SVG path `d` string,
// returned as float64 values. Handles signs, decimals, and exponents.
func parsePathValues(d string) []interface{} {
	var out []interface{}
	i := 0
	for i < len(d) {
		ch := d[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			i++
			continue
		}
		if ch == ' ' || ch == ',' || ch == '\n' || ch == '\r' || ch == '\t' {
			i++
			continue
		}
		start := i
		if ch == '-' || ch == '+' {
			i++
		}
		for i < len(d) && ((d[i] >= '0' && d[i] <= '9') || d[i] == '.') {
			i++
		}
		if i < len(d) && (d[i] == 'e' || d[i] == 'E') {
			j := i + 1
			if j < len(d) && (d[j] == '-' || d[j] == '+') {
				j++
			}
			for j < len(d) && d[j] >= '0' && d[j] <= '9' {
				j++
			}
			i = j
		}
		if i == start {
			i++
			continue
		}
		var v float64
		fmt.Sscanf(d[start:i], "%g", &v)
		out = append(out, v)
	}
	return out
}

func drawPath(canvas *graphics.Canvas, el SVGElement, scale float32) {
	d := el.Attrs["d"]
	if d == "" {
		return
	}
	path := buildSvgPath(d, 0, 0, 1)
	if path == nil {
		return
	}
	paint := graphics.NewPaint()
	fill := el.Attrs["fill"]
	if fill == "" {
		fill = "#000000"
	}
	if fill == "none" {
		paint.SetStyle(paintstyle.Stroke)
		// Strokes are device-space in the renderer (the CTM scales
		// coordinates but not widths), so scale the width ourselves.
		paint.SetStrokeWidth(effectiveStroke(el, scale))
		paint.SetColor(parseSVGColor(el.Attrs["stroke"]))
		canvas.DrawPath(path, paint)
		return
	}
	paint.SetColor(parseSVGColor(fill))
	canvas.DrawPath(path, paint)
}

func drawCircle(canvas *graphics.Canvas, el SVGElement, scale float32) {
	cx := parseFloat(el.Attrs["cx"])
	cy := parseFloat(el.Attrs["cy"])
	r := parseFloat(el.Attrs["r"])

	paint := graphics.NewPaint()

	if fill := el.Attrs["fill"]; fill != "" && fill != "none" {
		paint.SetColor(parseSVGColor(fill))
		canvas.DrawCircle(cx, cy, r, paint)
	}

	if stroke := el.Attrs["stroke"]; stroke != "" && stroke != "none" {
		paint.SetColor(parseSVGColor(stroke))
		paint.SetStyle(paintstyle.Stroke)
		paint.SetStrokeWidth(effectiveStroke(el, scale))
		canvas.DrawCircle(cx, cy, r, paint)
	}
}

func drawRect(canvas *graphics.Canvas, el SVGElement, scale float32) {
	x := parseFloat(el.Attrs["x"])
	y := parseFloat(el.Attrs["y"])
	w := parseFloat(el.Attrs["width"])
	h := parseFloat(el.Attrs["height"])
	rx := parseFloat(el.Attrs["rx"])

	paint := graphics.NewPaint()

	if fill := el.Attrs["fill"]; fill != "" && fill != "none" {
		paint.SetColor(parseSVGColor(fill))
		if rx > 0 {
			canvas.DrawRoundedRect(graphics.NewRect(x, y, w, h), rx, rx, paint)
		} else {
			canvas.DrawRect(graphics.NewRect(x, y, w, h), paint)
		}
	}

	if stroke := el.Attrs["stroke"]; stroke != "" && stroke != "none" {
		paint.SetColor(parseSVGColor(stroke))
		paint.SetStyle(paintstyle.Stroke)
		paint.SetStrokeWidth(effectiveStroke(el, scale))
		if rx > 0 {
			canvas.DrawRoundedRect(graphics.NewRect(x, y, w, h), rx, rx, paint)
		} else {
			canvas.DrawRect(graphics.NewRect(x, y, w, h), paint)
		}
	}
}

func drawLine(canvas *graphics.Canvas, el SVGElement, scale float32) {
	x1 := parseFloat(el.Attrs["x1"])
	y1 := parseFloat(el.Attrs["y1"])
	x2 := parseFloat(el.Attrs["x2"])
	y2 := parseFloat(el.Attrs["y2"])

	paint := graphics.NewPaint()
	if stroke := el.Attrs["stroke"]; stroke != "" {
		paint.SetColor(parseSVGColor(stroke))
	} else {
		paint.SetColor(graphics.Color(0xFF000000))
	}
	paint.SetStyle(paintstyle.Stroke)
	paint.SetStrokeWidth(effectiveStroke(el, scale))

	canvas.DrawLine(x1, y1, x2, y2, paint)
}

func drawPolyline(canvas *graphics.Canvas, el SVGElement, scale float32) {
	points := parsePoints(el.Attrs["points"])
	if len(points) < 4 {
		return
	}

	paint := graphics.NewPaint()
	if stroke := el.Attrs["stroke"]; stroke != "" {
		paint.SetColor(parseSVGColor(stroke))
	} else {
		paint.SetColor(graphics.Color(0xFF000000))
	}
	paint.SetStyle(paintstyle.Stroke)
	paint.SetStrokeWidth(effectiveStroke(el, scale))

	for i := 0; i < len(points)-2; i += 2 {
		canvas.DrawLine(points[i], points[i+1], points[i+2], points[i+3], paint)
	}
}

func drawPolygon(canvas *graphics.Canvas, el SVGElement) {
	points := parsePoints(el.Attrs["points"])
	if len(points) < 6 {
		return
	}

	paint := graphics.NewPaint()

	if fill := el.Attrs["fill"]; fill != "" && fill != "none" {
		paint.SetColor(parseSVGColor(fill))
		path := graphics.NewPath()
		path.MoveTo(graphics.NewPoint(points[0], points[1]))
		for i := 2; i < len(points); i += 2 {
			path.LineTo(graphics.NewPoint(points[i], points[i+1]))
		}
		path.Close()
		canvas.DrawPath(path, paint)
	}

	if stroke := el.Attrs["stroke"]; stroke != "" && stroke != "none" {
		paint.SetColor(parseSVGColor(stroke))
		paint.SetStyle(paintstyle.Stroke)
		if sw := el.Attrs["stroke-width"]; sw != "" {
			paint.SetStrokeWidth(parseFloat(sw))
		} else {
			paint.SetStrokeWidth(1)
		}
		path := graphics.NewPath()
		path.MoveTo(graphics.NewPoint(points[0], points[1]))
		for i := 2; i < len(points); i += 2 {
			path.LineTo(graphics.NewPoint(points[i], points[i+1]))
		}
		path.Close()
		canvas.DrawPath(path, paint)
	}
}

func drawText(canvas *graphics.Canvas, el SVGElement) {
	x := parseFloat(el.Attrs["x"])
	y := parseFloat(el.Attrs["y"])
	fontSize := parseFloat(el.Attrs["font-size"])
	if fontSize == 0 {
		fontSize = 12
	}

	paint := graphics.NewPaint()
	if fill := el.Attrs["fill"]; fill != "" {
		paint.SetColor(parseSVGColor(fill))
	} else {
		paint.SetColor(graphics.Color(0xFF000000))
	}

	font := graphics.SystemFont
	if font != nil {
		canvas.DrawSimpleString(el.Text, x, y, font, paint)
	}
}

func parsePoints(s string) []float32 {
	s = strings.TrimSpace(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	result := make([]float32, 0, len(parts))
	for _, p := range parts {
		if v := parseFloat(p); v != 0 || p == "0" {
			result = append(result, v)
		}
	}
	return result
}

func parseFloat(s string) float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0
	}
	return float32(v)
}

func parseSVGColor(s string) graphics.Color {
	s = strings.TrimSpace(s)

	// Named colors
	colorMap := map[string]graphics.Color{
		"white":  0xFFFFFFFF,
		"black":  0xFF000000,
		"red":    0xFFFF0000,
		"green":  0xFF00FF00,
		"blue":   0xFF0000FF,
		"yellow": 0xFFFFFF00,
		"gray":   0xFF888888,
		"grey":   0xFF888888,
	}
	if c, ok := colorMap[strings.ToLower(s)]; ok {
		return c
	}

	// Hex colors
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		if len(hex) == 3 {
			hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
		}
		if len(hex) == 6 {
			hex = "FF" + hex
		}
		var val uint32
		fmt.Sscanf(hex, "%x", &val)
		return graphics.Color(val)
	}

	return graphics.Color(0xFF000000)
}
