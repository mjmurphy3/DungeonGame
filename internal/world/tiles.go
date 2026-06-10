// Package world generates and renders the 256x256 overworld.
package world

import (
	"github.com/gdamore/tcell/v2"
)

// Tile is one overworld map cell.
type Tile uint8

const (
	TDeepWater Tile = iota
	TWater
	TSand
	TGrass
	TForest
	TMountain
	TLava
	TBrick
	TWall
	TEntrance
)

// Passable reports whether the player can step onto this tile.
func (t Tile) Passable() bool {
	switch t {
	case TSand, TGrass, TForest, TLava, TEntrance:
		return true
	}
	return false
}

var (
	colDeepWater  = tcell.NewRGBColor(8, 28, 110)
	colDeepWater2 = tcell.NewRGBColor(12, 40, 140)
	colWater      = tcell.NewRGBColor(20, 70, 190)
	colWater2     = tcell.NewRGBColor(30, 95, 220)
	colWave       = tcell.NewRGBColor(120, 180, 255)
	colSand       = tcell.NewRGBColor(210, 190, 120)
	colSandDot    = tcell.NewRGBColor(170, 150, 90)
	colGrass      = tcell.NewRGBColor(40, 120, 40)
	colGrassDot   = tcell.NewRGBColor(60, 150, 50)
	colForestBG   = tcell.NewRGBColor(25, 85, 30)
	colForestFG   = tcell.NewRGBColor(10, 55, 15)
	colMountBG    = tcell.NewRGBColor(95, 95, 100)
	colMountFG    = tcell.NewRGBColor(190, 190, 200)
	colLava       = tcell.NewRGBColor(140, 20, 0)
	colLava2      = tcell.NewRGBColor(190, 45, 0)
	colLavaGlow   = tcell.NewRGBColor(255, 200, 40)
	colBrickBG    = tcell.NewRGBColor(120, 55, 35)
	colBrickFG    = tcell.NewRGBColor(160, 80, 50)
	colWallBG     = tcell.NewRGBColor(105, 105, 115)
	colWallFG     = tcell.NewRGBColor(225, 225, 235)
	colLabel      = tcell.NewRGBColor(255, 245, 180)
	colEntrance   = tcell.NewRGBColor(20, 16, 12)
)

// waterPat / lavaPat are 1-D ripple patterns sampled along a diagonal of the
// world coordinates, so adjacent tiles always line up seamlessly.
var waterPat = []rune{'~', ' ', ' ', '≈', ' ', ' ', ' ', '~', ' ', ' ', '≈', ' '}
var lavaPat = []rune{'▒', '~', ' ', '∙', ' ', '▒', ' ', '~', ' ', ' '}

// mod returns a non-negative remainder.
func mod(a, n int) int {
	m := a % n
	if m < 0 {
		m += n
	}
	return m
}

// CellAt returns the glyph and colors to draw for world position (x, y) at
// animation tick t (one tick per frame, ~30/s). Water and lava phases derive
// from absolute world coordinates plus the clock, so the slow scroll is
// continuous across every tile on screen.
func (w *World) CellAt(x, y, tick int) (rune, tcell.Color, tcell.Color) {
	t := TDeepWater // outside the map reads as open sea
	idx := -1
	if x >= 0 && y >= 0 && x < w.W && y < w.H {
		idx = y*w.W + x
		t = w.Tiles[idx]
	}

	// Wall-inset label letters (PUB, KEEP, warnings...).
	if idx >= 0 {
		if ch, ok := w.Labels[idx]; ok {
			bg := colWallBG
			if t == TBrick {
				bg = colBrickBG
			}
			return ch, colLabel, bg
		}
	}

	switch t {
	case TDeepWater, TWater:
		base, base2 := colWater, colWater2
		if t == TDeepWater {
			base, base2 = colDeepWater, colDeepWater2
		}
		// Ripples drift slowly westward; shimmer phase drifts independently.
		ph := mod(x+y*3+tick/6, len(waterPat))
		bg := base
		if mod(x*2+y*5+tick/9, 7) < 2 {
			bg = base2
		}
		ch := waterPat[ph]
		if ch == ' ' {
			return ' ', bg, bg
		}
		return ch, colWave, bg
	case TLava:
		ph := mod(x+y*2+tick/10, len(lavaPat))
		bg := colLava
		if mod(x*3+y*4+tick/12, 5) < 2 {
			bg = colLava2
		}
		ch := lavaPat[ph]
		if ch == ' ' {
			return ' ', bg, bg
		}
		return ch, colLavaGlow, bg
	case TSand:
		if mod(x*7+y*13, 9) == 0 {
			return '·', colSandDot, colSand
		}
		return ' ', colSand, colSand
	case TGrass:
		switch mod(x*7+y*13, 11) {
		case 0:
			return ',', colGrassDot, colGrass
		case 5:
			return '\'', colGrassDot, colGrass
		}
		return ' ', colGrass, colGrass
	case TForest:
		if mod(x*5+y*11, 4) != 0 {
			return '♣', colForestFG, colForestBG
		}
		return ' ', colForestBG, colForestBG
	case TMountain:
		if mod(x*3+y*7, 3) != 0 {
			return '▲', colMountFG, colMountBG
		}
		return '^', colMountFG, colMountBG
	case TBrick:
		return '▒', colBrickFG, colBrickBG
	case TWall:
		return '█', colWallFG, colWallBG
	case TEntrance:
		return '∩', colLabel, colEntrance
	}
	return '?', tcell.ColorRed, tcell.ColorBlack
}
