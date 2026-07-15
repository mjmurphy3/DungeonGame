// Package raycast renders a Wolfenstein-style first-person view into a
// half-block pixel buffer: DDA wall casting with a per-column z-buffer and
// procedurally drawn billboard sprites.
package raycast

import (
	"math"

	"github.com/gdamore/tcell/v2"

	"github.com/mjmurphy3/DungeonGame/internal/render"
)

// WallKind classifies what a ray can hit.
type WallKind uint8

const (
	WallNone WallKind = iota
	WallStone
	WallDoor
)

// planeLen is tan(FOV/2); 0.66 gives the classic ~66 degree field of view.
const planeLen = 0.66

var (
	colCeilTop  = tcell.NewRGBColor(12, 10, 16)
	colCeilHor  = tcell.NewRGBColor(40, 36, 46)
	colFloorHor = tcell.NewRGBColor(46, 36, 26)
	colFloorBot = tcell.NewRGBColor(95, 75, 52)
	colStone    = tcell.NewRGBColor(155, 150, 158)
	colMortar   = tcell.NewRGBColor(70, 68, 74)
	colDoorWood = tcell.NewRGBColor(140, 92, 44)
	colDoorSeam = tcell.NewRGBColor(72, 46, 22)
)

// Renderer holds reusable buffers between frames.
type Renderer struct {
	zbuf []float64
}

// Render draws one frame: grid reports the wall kind of a map cell; the
// camera sits at (camX, camY) facing ang; sprites are drawn z-buffered.
func (r *Renderer) Render(pb *render.PixelBuf, grid func(x, y int) WallKind, camX, camY, ang float64, sprites []Sprite, tick int) {
	w, h := pb.W, pb.H
	if len(r.zbuf) != w {
		r.zbuf = make([]float64, w)
	}

	// Background: vertical gradients for ceiling and floor, darkest at the
	// horizon so distant corridors fade out.
	half := h / 2
	for y := 0; y < half; y++ {
		t := float64(y) / float64(half)
		c := render.Lerp(colCeilTop, colCeilHor, t)
		for x := 0; x < w; x++ {
			pb.Set(x, y, c)
		}
	}
	for y := half; y < h; y++ {
		t := float64(y-half) / float64(h-half)
		c := render.Lerp(colFloorHor, colFloorBot, t)
		for x := 0; x < w; x++ {
			pb.Set(x, y, c)
		}
	}

	dirX, dirY := math.Cos(ang), math.Sin(ang)
	planeX, planeY := -dirY*planeLen, dirX*planeLen

	for x := 0; x < w; x++ {
		cameraX := 2*float64(x)/float64(w) - 1
		rdX := dirX + planeX*cameraX
		rdY := dirY + planeY*cameraX

		mapX, mapY := int(math.Floor(camX)), int(math.Floor(camY))
		deltaX, deltaY := math.Abs(1/rdX), math.Abs(1/rdY)
		var sideX, sideY float64
		var stepX, stepY int
		if rdX < 0 {
			stepX = -1
			sideX = (camX - float64(mapX)) * deltaX
		} else {
			stepX = 1
			sideX = (float64(mapX) + 1 - camX) * deltaX
		}
		if rdY < 0 {
			stepY = -1
			sideY = (camY - float64(mapY)) * deltaY
		} else {
			stepY = 1
			sideY = (float64(mapY) + 1 - camY) * deltaY
		}

		var hit WallKind
		side := 0
		for i := 0; i < 64; i++ {
			if sideX < sideY {
				sideX += deltaX
				mapX += stepX
				side = 0
			} else {
				sideY += deltaY
				mapY += stepY
				side = 1
			}
			if k := grid(mapX, mapY); k != WallNone {
				hit = k
				break
			}
		}
		if hit == WallNone {
			r.zbuf[x] = math.MaxFloat64
			continue
		}

		var perpDist, wallX float64
		if side == 0 {
			perpDist = sideX - deltaX
			wallX = camY + perpDist*rdY
		} else {
			perpDist = sideY - deltaY
			wallX = camX + perpDist*rdX
		}
		if perpDist < 1e-4 {
			perpDist = 1e-4
		}
		wallX -= math.Floor(wallX)
		r.zbuf[x] = perpDist

		lineH := int(float64(h) / perpDist)
		y0 := half - lineH/2
		y1 := half + lineH/2
		drawY0, drawY1 := y0, y1
		if drawY0 < 0 {
			drawY0 = 0
		}
		if drawY1 > h {
			drawY1 = h
		}

		shade := 1.0 / (1.0 + perpDist*perpDist*0.022)
		if side == 1 {
			shade *= 0.72
		}
		for y := drawY0; y < drawY1; y++ {
			texY := float64(y-y0) / float64(lineH)
			pb.Set(x, y, render.Shade(wallTexel(hit, wallX, texY, mapX, mapY), shade))
		}
	}

	r.drawSprites(pb, camX, camY, dirX, dirY, planeX, planeY, sprites, tick)
}

// wallTexel returns the unshaded surface color at texture coords (u, v).
func wallTexel(kind WallKind, u, v float64, cellX, cellY int) tcell.Color {
	if kind == WallDoor {
		// Vertical planks with a cross beam.
		if math.Mod(u*4, 1) < 0.10 || math.Abs(v-0.5) < 0.035 {
			return colDoorSeam
		}
		return colDoorWood
	}
	// Stone bricks: staggered courses with mortar gaps; per-cell tint
	// variation so long walls don't look uniform.
	row := math.Floor(v * 5)
	off := 0.0
	if int(row)%2 == 1 {
		off = 0.5
	}
	if math.Mod(v*5, 1) < 0.14 || math.Mod(u*2.5+off, 1) < 0.08 {
		return colMortar
	}
	tint := 0.88 + 0.12*float64((cellX*7+cellY*13)%5)/4
	return render.Shade(colStone, tint)
}
