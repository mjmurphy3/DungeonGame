package raycast

import (
	"math"
	"sort"

	"github.com/gdamore/tcell/v2"

	"dungeongame/internal/render"
)

// SpriteKind selects which procedural billboard to draw.
type SpriteKind uint8

const (
	KOrc SpriteKind = iota
	KOrcDead
	KChest
	KChestOpen
	KColumn
	KLadder
	KTorch
	KMissile
)

// Sprite is a world-positioned billboard.
type Sprite struct {
	X, Y  float64
	Kind  SpriteKind
	Frame int     // animation frame (walk cycle, flicker...)
	Scale float64 // height relative to a full wall (1.0 = floor to ceiling)
	Lift  float64 // bottom offset above the floor, in wall heights
}

type projected struct {
	s     Sprite
	depth float64
}

func (r *Renderer) drawSprites(pb *render.PixelBuf, camX, camY, dirX, dirY, planeX, planeY float64, sprites []Sprite, tick int) {
	w, h := pb.W, pb.H
	order := make([]projected, 0, len(sprites))
	for _, s := range sprites {
		dx, dy := s.X-camX, s.Y-camY
		order = append(order, projected{s, dx*dx + dy*dy})
	}
	sort.Slice(order, func(i, j int) bool { return order[i].depth > order[j].depth })

	invDet := 1.0 / (planeX*dirY - dirX*planeY)
	for _, p := range order {
		s := p.s
		relX, relY := s.X-camX, s.Y-camY
		transX := invDet * (dirY*relX - dirX*relY)
		transY := invDet * (-planeY*relX + planeX*relY) // depth into the screen
		if transY < 0.15 {
			continue
		}
		screenX := int(float64(w) / 2 * (1 + transX/transY))

		wallH := float64(h) / transY // apparent height of a full wall here
		sprH := int(wallH * s.Scale)
		sprW := sprH // square aspect in pixel space
		if sprH < 1 || sprW < 1 {
			continue
		}
		bottom := h/2 + int(wallH/2) - int(s.Lift*wallH)
		top := bottom - sprH
		x0, x1 := screenX-sprW/2, screenX+sprW/2
		shade := 1.0 / (1.0 + transY*transY*0.022)

		for x := x0; x < x1; x++ {
			if x < 0 || x >= w || transY >= r.zbuf[x] {
				continue
			}
			u := float64(x-x0) / float64(x1-x0)
			for y := top; y < bottom; y++ {
				if y < 0 || y >= h {
					continue
				}
				v := float64(y-top) / float64(sprH)
				if c, ok := texel(s.Kind, s.Frame, u, v); ok {
					pb.Set(x, y, render.Shade(c, shade))
				}
			}
		}
	}
}

// inEllipse tests (u,v) against an ellipse at (cx,cy) with radii (rx,ry).
func inEllipse(u, v, cx, cy, rx, ry float64) bool {
	du, dv := (u-cx)/rx, (v-cy)/ry
	return du*du+dv*dv <= 1
}

var (
	orcSkin   = tcell.NewRGBColor(70, 145, 55)
	orcDark   = tcell.NewRGBColor(45, 95, 35)
	orcCloth  = tcell.NewRGBColor(95, 75, 40)
	orcEye    = tcell.NewRGBColor(255, 60, 40)
	orcTusk   = tcell.NewRGBColor(235, 230, 200)
	chestWood = tcell.NewRGBColor(135, 85, 35)
	chestDark = tcell.NewRGBColor(80, 50, 20)
	chestBand = tcell.NewRGBColor(90, 90, 100)
	chestGold = tcell.NewRGBColor(255, 210, 70)
	chestVoid = tcell.NewRGBColor(18, 12, 8)
	colStoneC = tcell.NewRGBColor(165, 160, 168)
	ladderW   = tcell.NewRGBColor(150, 100, 45)
	torchWood = tcell.NewRGBColor(110, 70, 35)
	flameOut  = tcell.NewRGBColor(255, 120, 10)
	flameIn   = tcell.NewRGBColor(255, 230, 90)
	boltCore  = tcell.NewRGBColor(245, 250, 255)
	boltGlow  = tcell.NewRGBColor(110, 190, 255)
)

// texel returns the sprite surface color at (u, v) in [0,1]^2, or ok=false
// where the billboard is transparent.
func texel(kind SpriteKind, frame int, u, v float64) (tcell.Color, bool) {
	switch kind {
	case KOrc:
		return orcTexel(frame, u, v)
	case KOrcDead:
		// A slumped remains pile on the floor.
		if inEllipse(u, v, 0.5, 0.88, 0.38, 0.12) {
			if inEllipse(u, v, 0.62, 0.84, 0.10, 0.06) {
				return orcDark, true
			}
			return orcSkin, true
		}
		return 0, false
	case KChest, KChestOpen:
		return chestTexel(kind == KChestOpen, u, v)
	case KColumn:
		return columnTexel(u, v)
	case KLadder:
		return ladderTexel(u, v)
	case KTorch:
		return torchTexel(frame, u, v)
	case KMissile:
		d := math.Hypot((u-0.5)*2, (v-0.5)*2)
		switch {
		case d < 0.45:
			return boltCore, true
		case d < 0.95:
			return boltGlow, true
		}
		return 0, false
	}
	return 0, false
}

func orcTexel(frame int, u, v float64) (tcell.Color, bool) {
	swing := 0.0
	if frame%2 == 1 {
		swing = 0.05
	}
	// Head with ears, eyes and tusks.
	if inEllipse(u, v, 0.5, 0.17, 0.17, 0.14) {
		if inEllipse(u, v, 0.43, 0.15, 0.035, 0.030) || inEllipse(u, v, 0.57, 0.15, 0.035, 0.030) {
			return orcEye, true
		}
		if inEllipse(u, v, 0.44, 0.26, 0.025, 0.045) || inEllipse(u, v, 0.56, 0.26, 0.025, 0.045) {
			return orcTusk, true
		}
		return orcSkin, true
	}
	if inEllipse(u, v, 0.31, 0.13, 0.05, 0.07) || inEllipse(u, v, 0.69, 0.13, 0.05, 0.07) {
		return orcSkin, true // pointed ears
	}
	// Torso: broad shoulders tapering to the waist, leather across the chest.
	if v >= 0.30 && v < 0.66 {
		halfW := 0.26 - 0.12*(v-0.30)/0.36
		if math.Abs(u-0.5) < halfW {
			if v > 0.40 && v < 0.52 {
				return orcCloth, true
			}
			return orcSkin, true
		}
		// Arms swinging in opposite phase.
		armL := 0.30 + swing
		armR := 0.30 - swing
		if math.Abs(u-(0.5-halfW-0.05)) < 0.05 && v > armL && v < armL+0.30 {
			return orcDark, true
		}
		if math.Abs(u-(0.5+halfW+0.05)) < 0.05 && v > armR && v < armR+0.30 {
			return orcDark, true
		}
	}
	// Legs: one lifts a little each walk frame.
	if v >= 0.66 {
		lEnd, rEnd := 1.0, 0.92
		if frame%2 == 1 {
			lEnd, rEnd = 0.92, 1.0
		}
		if math.Abs(u-0.40) < 0.065 && v < lEnd {
			return orcDark, true
		}
		if math.Abs(u-0.60) < 0.065 && v < rEnd {
			return orcDark, true
		}
	}
	return 0, false
}

func chestTexel(open bool, u, v float64) (tcell.Color, bool) {
	if u < 0.08 || u > 0.92 || v < 0.30 || v > 0.97 {
		return 0, false
	}
	if open {
		if v < 0.42 {
			return chestVoid, true // raised lid interior
		}
		if v < 0.55 && u > 0.2 && u < 0.8 {
			return chestGold, true // treasure glinting inside
		}
	} else {
		if v < 0.52 && math.Abs(v-0.50) < 0.025 {
			return chestDark, true // lid seam
		}
		if inEllipse(u, v, 0.5, 0.56, 0.06, 0.07) {
			return chestGold, true // lock plate
		}
	}
	if math.Abs(u-0.28) < 0.035 || math.Abs(u-0.72) < 0.035 {
		return chestBand, true
	}
	if v > 0.93 {
		return chestDark, true
	}
	return chestWood, true
}

func columnTexel(u, v float64) (tcell.Color, bool) {
	// Capital and base are wider than the shaft.
	if v < 0.07 || v > 0.93 {
		if u > 0.22 && u < 0.78 {
			return render.Shade(colStoneC, 0.85), true
		}
		return 0, false
	}
	if u < 0.32 || u > 0.68 {
		return 0, false
	}
	// Cylindrical shading across the shaft, with fluting grooves.
	t := (u - 0.5) / 0.18
	bright := math.Sqrt(math.Max(0, 1-t*t))*0.55 + 0.45
	if math.Mod(u*14, 1) < 0.18 {
		bright *= 0.78
	}
	return render.Shade(colStoneC, bright), true
}

func ladderTexel(u, v float64) (tcell.Color, bool) {
	rail := math.Abs(u-0.32) < 0.05 || math.Abs(u-0.68) < 0.05
	rung := u > 0.27 && u < 0.73 && math.Mod(v*6, 1) < 0.22
	if rail || rung {
		if rung && !rail {
			return render.Shade(ladderW, 0.8), true
		}
		return ladderW, true
	}
	return 0, false
}

func torchTexel(frame int, u, v float64) (tcell.Color, bool) {
	// Handle.
	if math.Abs(u-0.5) < 0.06 && v > 0.45 && v < 0.95 {
		return torchWood, true
	}
	// Flame leans with the flicker frame.
	lean := 0.04
	ry := 0.30
	if frame%2 == 1 {
		lean, ry = -0.04, 0.34
	}
	if inEllipse(u, v, 0.5+lean, 0.28, 0.16, ry) {
		if inEllipse(u, v, 0.5+lean/2, 0.34, 0.08, 0.16) {
			return flameIn, true
		}
		return flameOut, true
	}
	return 0, false
}
