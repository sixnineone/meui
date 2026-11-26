package main

import (
	"fmt"
	"math"

	"github.com/sixnineone/meui-gl/window"
	"github.com/sixnineone/mobilegles"
)

func main() {
	fmt.Println("MeUi-Mobile Kitchen Sink Demo")
	fmt.Println("=============================")

	fontPath := "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	if _, err := mobilegles.LoadSystemFont(fontPath); err != nil {
		fmt.Printf("Warning: could not load font %s: %v\n", fontPath, err)
	} else {
		fmt.Println("Font loaded")
	}

	sys := mobilegles.SystemTTF()
	titleFont := mobilegles.NewFont("DejaVu", 24, sys)
	bodyFont := mobilegles.NewFont("DejaVu", 16, sys)
	smallFont := mobilegles.NewFont("DejaVu", 12, sys)
	sizeFonts := make([]*mobilegles.Font, 4)
	for i, s := range []float32{10, 14, 18, 22} {
		sizeFonts[i] = mobilegles.NewFont("DejaVu", s, sys)
	}

	win, err := window.New(
		window.Title("MeUi-Mobile Kitchen Sink"),
		window.WindowSize(1280, 720),
		window.FPSCounter(true),
		window.BackgroundColor(mobilegles.RGBA(26, 26, 46, 255)),
	)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}

	win.OnKeyDown = func(key mobilegles.Keycode, _ mobilegles.KeyMod, _ bool) bool {
		if key == mobilegles.KeyEscape {
			win.Close()
			return true
		}
		return false
	}
	win.OnAllowClose = func() bool { return true }
	win.OnResized = func(w, h int32) {
		fmt.Printf("Window resized: %dx%d\n", w, h)
	}

	win.OnDraw = func(canvas *mobilegles.Canvas, w, h int32) {
		labelPaint := mobilegles.NewPaint()
		labelPaint.SetStyle(int32(mobilegles.Fill))
		labelPaint.SetColor(mobilegles.RGBA(80, 80, 120, 255))

		canvas.DrawRect(10, 45, 180, 25, labelPaint)
		canvas.DrawRect(250, 45, 180, 25, labelPaint)
		canvas.DrawRect(520, 45, 100, 25, labelPaint)
		canvas.DrawRect(700, 45, 120, 25, labelPaint)
		canvas.DrawRect(520, 160, 90, 25, labelPaint)
		canvas.DrawRect(520, 180, 160, 25, labelPaint)
		canvas.DrawRect(700, 130, 90, 25, labelPaint)
		canvas.DrawRect(520, 290, 140, 25, labelPaint)
		canvas.DrawRect(900, 45, 90, 25, labelPaint)

		demoBasicShapes(canvas)
		demoPaths(canvas)
		demoTextKitchen(canvas, titleFont, bodyFont, smallFont, sizeFonts)
		demoTransforms(canvas)
		demoClipping(canvas)
		demoBlendModes(canvas)
		demoEffects(canvas)
		demoMatrixStack(canvas)
		demoImages(canvas)
	}

	if err := win.Run(); err != nil {
		fmt.Printf("Run error: %v\n", err)
	}
}

func demoTextKitchen(canvas *mobilegles.Canvas, titleFont, bodyFont, smallFont *mobilegles.Font, sizeFonts []*mobilegles.Font) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Fill))

	paint.SetColor(mobilegles.RGBA(255, 220, 100, 255))
	canvas.DrawText("GPU Text Rendering", 530, 70, titleFont, paint)

	paint.SetColor(mobilegles.RGBA(180, 220, 255, 255))
	canvas.DrawText("Glyphs -> Paths -> GPU", 530, 105, bodyFont, paint)

	paint.SetColor(mobilegles.RGBA(180, 255, 180, 220))
	canvas.DrawText("GPU-offloaded via earcut", 530, 130, smallFont, paint)

	sizes := []float32{10, 14, 18, 22}
	colors := []mobilegles.Color{
		mobilegles.RGBA(255, 150, 150, 255),
		mobilegles.RGBA(150, 255, 150, 255),
		mobilegles.RGBA(150, 150, 255, 255),
		mobilegles.RGBA(255, 255, 150, 255),
	}
	for i, size := range sizes {
		paint.SetColor(colors[i])
		canvas.DrawText(fmt.Sprintf("Size %d px", int(size)), 530, 155+float32(i)*22, sizeFonts[i], paint)
	}
}

func demoBasicShapes(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetColor(mobilegles.RGBA(255, 255, 255, 255))
	paint.SetStyle(int32(mobilegles.Fill))
	canvas.DrawRect(20, 60, 80, 50, paint)

	paint.SetStyle(int32(mobilegles.Stroke))
	paint.SetStrokeWidth(3)
	canvas.DrawRect(120, 60, 80, 50, paint)

	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(100, 200, 255, 255))
	canvas.DrawCircle(60, 150, 30, paint)

	paint.SetColor(mobilegles.RGBA(255, 200, 100, 255))
	canvas.DrawOval(20, 190, 80, 50, paint)

	paint.SetStyle(int32(mobilegles.Stroke))
	paint.SetStrokeWidth(2)
	paint.SetColor(mobilegles.RGBA(200, 255, 100, 255))
	canvas.DrawLine(20, 260, 120, 300, paint)

	paint.SetColor(mobilegles.RGBA(255, 150, 200, 255))
	canvas.DrawArc(70, 270, 80, 60, 0, 270, paint)
}

func demoPaths(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Stroke))
	paint.SetStrokeWidth(2)
	paint.SetColor(mobilegles.RGBA(255, 255, 100, 255))

	path := mobilegles.NewPath()
	path.MoveTo(mobilegles.Point{X: 250, Y: 100})
	path.QuadTo(mobilegles.Point{X: 200, Y: 60}, mobilegles.Point{X: 210, Y: 80})
	path.QuadTo(mobilegles.Point{X: 220, Y: 60}, mobilegles.Point{X: 250, Y: 70})
	path.QuadTo(mobilegles.Point{X: 280, Y: 60}, mobilegles.Point{X: 290, Y: 80})
	path.QuadTo(mobilegles.Point{X: 300, Y: 60}, mobilegles.Point{X: 250, Y: 100})
	path.Close()
	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(255, 80, 80, 200))
	canvas.DrawPath(path, paint)

	paint.SetColor(mobilegles.RGBA(100, 255, 200, 255))
	paint.SetStyle(int32(mobilegles.Fill))
	star := mobilegles.NewPath()
	cx, cy, outerR, innerR := float64(330), float64(155), float64(45), float64(20)
	for i := 0; i < 5; i++ {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		outerX := cx + outerR*math.Cos(angle)
		outerY := cy + outerR*math.Sin(angle)
		innerAngle := angle + math.Pi/5
		innerX := cx + innerR*math.Cos(innerAngle)
		innerY := cy + innerR*math.Sin(innerAngle)
		if i == 0 {
			star.MoveTo(mobilegles.Point{X: float32(outerX), Y: float32(outerY)})
			star.LineTo(mobilegles.Point{X: float32(innerX), Y: float32(innerY)})
		} else {
			star.LineTo(mobilegles.Point{X: float32(outerX), Y: float32(outerY)})
			star.LineTo(mobilegles.Point{X: float32(innerX), Y: float32(innerY)})
		}
	}
	star.Close()
	canvas.DrawPath(star, paint)

	paint.SetStyle(int32(mobilegles.Stroke))
	paint.SetStrokeWidth(1.5)
	paint.SetColor(mobilegles.RGBA(255, 200, 100, 220))
	spiral := mobilegles.NewPath()
	spiral.MoveTo(mobilegles.Point{X: 400, Y: 100})
	segments := 60
	for i := 1; i <= segments; i++ {
		t := float64(i) / float64(segments)
		angle := t * 6 * math.Pi
		radius := t * 40
		x := 400 + radius*math.Cos(angle)
		y := 130 + radius*math.Sin(angle)
		spiral.LineTo(mobilegles.Point{X: float32(x), Y: float32(y)})
	}
	canvas.DrawPath(spiral, paint)

	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(200, 100, 255, 200))
	arrow := mobilegles.NewPath()
	arrow.MoveTo(mobilegles.Point{X: 260, Y: 190})
	arrow.LineTo(mobilegles.Point{X: 310, Y: 190})
	arrow.LineTo(mobilegles.Point{X: 310, Y: 170})
	arrow.LineTo(mobilegles.Point{X: 340, Y: 200})
	arrow.LineTo(mobilegles.Point{X: 310, Y: 230})
	arrow.LineTo(mobilegles.Point{X: 310, Y: 210})
	arrow.LineTo(mobilegles.Point{X: 260, Y: 210})
	arrow.Close()
	canvas.DrawPath(arrow, paint)
}

func demoTransforms(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(255, 100, 150, 220))

	saveCount := canvas.Save()
	canvas.Translate(400, 100)
	canvas.Scale(1.5, 1.5)
	canvas.DrawRect(0, 0, 60, 40, paint)
	canvas.RestoreToCount(saveCount)

	saveCount = canvas.Save()
	canvas.Translate(400, 180)
	canvas.Rotate(45)
	paint.SetColor(mobilegles.RGBA(150, 255, 100, 220))
	canvas.DrawRect(-30, -20, 60, 40, paint)
	canvas.RestoreToCount(saveCount)

	saveCount = canvas.Save()
	canvas.Translate(400, 260)
	canvas.Skew(30, 0)
	paint.SetColor(mobilegles.RGBA(255, 200, 100, 220))
	canvas.DrawRect(0, 0, 60, 40, paint)
	canvas.RestoreToCount(saveCount)
}

func demoClipping(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()

	saveCount := canvas.Save()
	canvas.ClipRect(mobilegles.Rect{X: 520, Y: 60, Width: 150, Height: 100})

	paint.SetStyle(int32(mobilegles.Fill))
	for i := 0; i < 5; i++ {
		paint.SetColor(mobilegles.RGBA(255, uint8(100+i*30), 150, 200))
		canvas.DrawRect(float32(520), float32(60+i*20), float32(150), float32(15), paint)
	}
	canvas.RestoreToCount(saveCount)
}

func demoBlendModes(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Fill))

	paint.SetColor(mobilegles.RGBA(50, 50, 80, 255))
	canvas.DrawRect(520, 180, 150, 100, paint)

	blendModes := []mobilegles.BlendMode{
		mobilegles.BlendModeNormal,
		mobilegles.BlendModeMultiply,
		mobilegles.BlendModeScreen,
		mobilegles.BlendModeOverlay,
	}

	for i, mode := range blendModes {
		paint.SetColor(mobilegles.RGBA(200, 100, 255, 180))
		paint.SetBlendMode(mode)
		canvas.DrawRect(float32(530+i*35), 200, 30, 30, paint)
	}
}

func demoEffects(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Fill))

	paint.SetColor(mobilegles.RGBA(100, 200, 255, 255))
	canvas.DropShadow(mobilegles.Rect{X: 700, Y: 60, Width: 80, Height: 50}, paint, 5, 5, 3)

	paint.SetStyle(int32(mobilegles.Fill))
	canvas.DrawRect(700, 60, 80, 50, paint)

	paint.SetColor(mobilegles.RGBA(255, 200, 100, 255))
	canvas.DrawRoundedRect(mobilegles.Rect{X: 700, Y: 130, Width: 80, Height: 50}, 10, 10, paint)
}

func demoMatrixStack(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()
	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(255, 100, 200, 220))

	saveCount := canvas.Save()
	canvas.Translate(520, 300)
	canvas.Scale(0.7, 0.7)
	canvas.DrawRect(0, 0, 60, 40, paint)

	canvas.Save()
	canvas.Translate(80, 0)
	canvas.Rotate(30)
	paint.SetColor(mobilegles.RGBA(100, 200, 255, 220))
	canvas.DrawRect(0, 0, 60, 40, paint)
	canvas.RestoreToCount(saveCount)
}

func demoImages(canvas *mobilegles.Canvas) {
	paint := mobilegles.NewPaint()

	paint.SetStyle(int32(mobilegles.Stroke))
	paint.SetStrokeWidth(2)
	paint.SetColor(mobilegles.RGBA(200, 200, 255, 200))
	canvas.DrawRect(900, 60, 100, 100, paint)

	paint.SetStyle(int32(mobilegles.Fill))
	paint.SetColor(mobilegles.RGBA(150, 150, 200, 150))
	canvas.DrawImageInRect(nil, mobilegles.Rect{X: 900, Y: 60, Width: 100, Height: 100}, paint)
}
