package graphics

// Backdrop blur: what a frosted panel is.
//
// The GL path blurs with a fixed separable 7-tap Gaussian (shaders/blur_fs.glsl, the
// same kernel its shadows use), so this device uses THE SAME kernel and tap offsets.
// A radius parameter would have to exist in both to mean anything: with one renderer's
// strength fixed, a variable one elsewhere is drift by construction. The strength of a
// frosted panel is therefore a renderer constant, and the panel only chooses whether it
// is frosted.
//
// The glass is masked to the box's rounded silhouette with the same coverage function on
// both sides (maskRoundedCorners), so a rounded frosted card has the same antialiased
// corners offscreen and live.

// blurKernel is shaders/blur_fs.glsl's weights, centre first.
var blurKernel = [7]float32{0.271, 0.217, 0.217, 0.111, 0.111, 0.037, 0.037}

// BlurBackdrop blurs what is already drawn inside r (device space after the matrix) and
// composites the result back. The panel paints its translucent tint and border on top.
func (d *softDevice) BlurBackdrop(r Rect, cornerRadius float32) bool {
	if d.img == nil {
		return false
	}
	tx, ty, tw, th := d.TransformRect(r.X, r.Y, r.Width, r.Height)
	if tw < 1 || th < 1 {
		return false
	}
	b := d.img.Bounds()
	x0 := imax(int(tx), 0)
	y0 := imax(int(ty), 0)
	x1 := imin(int(tx+tw), b.Dx())
	y1 := imin(int(ty+th), b.Dy())
	if x1 <= x0 || y1 <= y0 {
		return false
	}
	// Sample with a margin so the kernel's edge taps read real pixels, not transparency.
	const margin = 3
	px := imax(x0-margin, 0)
	py := imax(y0-margin, 0)
	px1 := imin(x1+margin, b.Dx())
	py1 := imin(y1+margin, b.Dy())
	pw, ph := px1-px, py1-py

	// Extract, top-down and premultiplied: the surface is bottom-up.
	src := make([]uint32, pw*ph)
	for y := 0; y < ph; y++ {
		row := (b.Dy() - 1 - (py + y)) * d.img.Stride
		for x := 0; x < pw; x++ {
			o := row + (px+x)*4
			src[y*pw+x] = uint32(d.img.Pix[o+3])<<24 | uint32(d.img.Pix[o+2])<<16 |
				uint32(d.img.Pix[o+1])<<8 | uint32(d.img.Pix[o])
		}
	}
	blurH(src, pw, ph)
	blurV(src, pw, ph)

	// Composite back inside the ROUNDED silhouette (and the clip, if one is in force).
	sx, sy := scaleX(d.mat), scaleY(d.mat)
	rc := cornerRadius * (sx + sy) / 2
	hw, hh := tw/2, th/2
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if d.clip != nil && !pointInRect(float32(x)+0.5, float32(y)+0.5, *d.clip) {
				continue
			}
			cov := roundedCoverage(float32(x)+0.5-tx-hw, float32(y)+0.5-ty-hh, hw, hh, rc)
			if cov <= 0 {
				continue
			}
			sp := src[(y-py)*pw+(x-px)]
			sr := float32((sp>>16)&0xFF) * cov
			sg := float32((sp>>8)&0xFF) * cov
			sb := float32(sp&0xFF) * cov
			sa := float32((sp>>24)&0xFF) / 255 * cov
			inv := 1 - sa
			o := (b.Dy()-1-y)*d.img.Stride + x*4
			d.img.Pix[o+0] = uint8(imin(int(sb+float32(d.img.Pix[o+0])*inv+0.5), 255))
			d.img.Pix[o+1] = uint8(imin(int(sg+float32(d.img.Pix[o+1])*inv+0.5), 255))
			d.img.Pix[o+2] = uint8(imin(int(sr+float32(d.img.Pix[o+2])*inv+0.5), 255))
			d.img.Pix[o+3] = uint8(imin(int(sa*255+float32(d.img.Pix[o+3])*inv+0.5), 255))
		}
	}
	return true
}

// blurH applies the 7-tap kernel horizontally, in premultiplied 0xAARRGGBB.
func blurH(pix []uint32, w, h int) {
	tmp := make([]uint32, w)
	for y := 0; y < h; y++ {
		row := pix[y*w : (y+1)*w]
		for x := 0; x < w; x++ {
			var a, r, g, b float32
			for k, wt := range blurKernel {
				off := k / 2
				if k%2 == 1 {
					off = -off
				}
				sx := clampI(x+off, 0, w-1)
				p := row[sx]
				a += float32((p>>24)&0xFF) * wt
				r += float32((p>>16)&0xFF) * wt
				g += float32((p>>8)&0xFF) * wt
				b += float32(p&0xFF) * wt
			}
			tmp[x] = packPremul(a, r, g, b)
		}
		copy(row, tmp)
	}
}

// blurV applies the same kernel vertically.
func blurV(pix []uint32, w, h int) {
	tmp := make([]uint32, h)
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			var a, r, g, b float32
			for k, wt := range blurKernel {
				off := k / 2
				if k%2 == 1 {
					off = -off
				}
				sy := clampI(y+off, 0, h-1)
				p := pix[sy*w+x]
				a += float32((p>>24)&0xFF) * wt
				r += float32((p>>16)&0xFF) * wt
				g += float32((p>>8)&0xFF) * wt
				b += float32(p&0xFF) * wt
			}
			tmp[y] = packPremul(a, r, g, b)
		}
		for y := 0; y < h; y++ {
			pix[y*w+x] = tmp[y]
		}
	}
}

func packPremul(a, r, g, b float32) uint32 {
	cl := func(v float32) uint32 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint32(v + 0.5)
	}
	return cl(a)<<24 | cl(r)<<16 | cl(g)<<8 | cl(b)
}

func pointInRect(x, y float32, r Rect) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// roundedCoverage is the fraction of a pixel at (dx,dy) from the box centre that lies
// inside a rounded rect of half-extents hw,hh and corner radius rc, from the signed
// distance to the rounded box. One pixel of AA at the arc.
func roundedCoverage(dx, dy, hw, hh, rc float32) float32 {
	if rc <= 0 {
		if dx >= -hw && dx < hw && dy >= -hh && dy < hh {
			return 1
		}
		return 0
	}
	if rc > hw {
		rc = hw
	}
	if rc > hh {
		rc = hh
	}
	qx := absF(dx) - hw + rc
	qy := absF(dy) - hh + rc
	outside := float32(0)
	if qx > 0 && qy > 0 {
		outside = sqrtF(qx*qx + qy*qy)
	}
	inside := qx
	if qy > inside {
		inside = qy
	}
	if inside > 0 {
		inside = 0
	}
	return clamp01F(0.5 - (outside + inside - rc))
}

// maskRoundedCorners scales the ALPHA of premultiplied 0xAARRGGBB pixels by the rounded
// coverage of a w×h box. The GL device uses it on the readback before upload, which is
// what makes a rounded frosted corner identical in both renderers.
func maskRoundedCorners(pix []uint32, w, h int, radius float32) {
	if radius <= 0 {
		return
	}
	hw, hh := float32(w)/2, float32(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cov := roundedCoverage(float32(x)+0.5-hw, float32(y)+0.5-hh, hw, hh, radius)
			if cov >= 1 {
				continue
			}
			i := y*w + x
			if cov <= 0 {
				pix[i] = 0
				continue
			}
			p := pix[i]
			a := float32((p>>24)&0xFF) * cov
			r := float32((p>>16)&0xFF) * cov
			g := float32((p>>8)&0xFF) * cov
			b := float32(p&0xFF) * cov
			pix[i] = packPremul(a, r, g, b)
		}
	}
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func sqrtF(v float32) float32 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 6; i++ {
		x = 0.5 * (x + v/x)
	}
	return x
}

func clamp01F(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
