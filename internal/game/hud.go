package game

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

var (
	hudBG     = tcell.NewRGBColor(25, 25, 35)
	hudText   = tcell.NewRGBColor(200, 200, 210)
	hudMsg    = tcell.NewRGBColor(255, 235, 150)
	hpGood    = tcell.NewRGBColor(70, 200, 70)
	hpWarn    = tcell.NewRGBColor(230, 200, 50)
	hpBad     = tcell.NewRGBColor(230, 60, 50)
	hpEmpty   = tcell.NewRGBColor(70, 40, 40)
	goldColor = tcell.NewRGBColor(255, 210, 70)
)

// drawHUD paints the hint/message bar on the bottom screen row (vitals live
// in the bordered stats box).
func (g *Game) drawHUD(w, h int) {
	y := h - 1
	g.scr.FillRow(y, 0, w, ' ', hudText, hudBG)

	hint := "WASD move  SPACE fire  ESC quit"
	if g.mode == ModeDungeon {
		hint = "W/S walk  A/D turn  SPACE fire  ESC quit"
	}
	g.scr.Text(1, y, hint, hudText, hudBG)

	if g.msgT > 0 && len(g.msg) > 0 {
		x := w - len(g.msg) - 2
		if x < len(hint)+4 {
			x = len(hint) + 4
		}
		g.scr.Text(x, y, g.msg, hudMsg, hudBG)
	}
}

// box draws a bordered rectangle (interior filled) and returns nothing; the
// caller writes its content inside.
func (g *Game) box(x, y, w, h int, border, bg tcell.Color) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			ch := ' '
			switch {
			case (i == 0 || i == w-1) && (j == 0 || j == h-1):
				corners := [2][2]rune{{'╔', '╗'}, {'╚', '╝'}}
				ch = corners[b2i(j == h-1)][b2i(i == w-1)]
			case j == 0 || j == h-1:
				ch = '═'
			case i == 0 || i == w-1:
				ch = '║'
			}
			g.scr.SetCell(x+i, y+j, ch, border, bg)
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// drawStatsBox paints the bordered vitals panel in the top-left corner,
// sized to fit its contents.
func (g *Game) drawStatsBox() {
	const barLen = 10
	hpLine := fmt.Sprintf("HP   %3d/%d ", g.hp, maxHP)
	goldLine := fmt.Sprintf("Gold %d/%d (goal %d)", g.gold, g.totalGold, g.goldGoal)
	orcLine := fmt.Sprintf("Orcs slain %d/%d", g.orcsKilled, g.totalOrcs)

	inner := len(hpLine) + barLen
	for _, s := range []string{goldLine, orcLine} {
		if len(s) > inner {
			inner = len(s)
		}
	}
	bw, bh := inner+4, 5
	x, y := 1, 1
	g.box(x, y, bw, bh, goldColor, hudBG)

	// HP row: number plus a colored bar.
	g.scr.Text(x+2, y+1, hpLine, hudText, hudBG)
	filled := g.hp * barLen / maxHP
	hpCol := hpGood
	if g.hp <= 30 {
		hpCol = hpBad
	} else if g.hp <= 60 {
		hpCol = hpWarn
	}
	for i := 0; i < barLen; i++ {
		c := hpEmpty
		if i < filled {
			c = hpCol
		}
		g.scr.SetCell(x+2+len(hpLine)+i, y+1, '█', c, hudBG)
	}

	g.scr.Text(x+2, y+2, goldLine, goldColor, hudBG)
	g.scr.Text(x+2, y+3, orcLine, hudText, hudBG)
}
