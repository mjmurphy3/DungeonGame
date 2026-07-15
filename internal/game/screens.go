package game

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"

	"github.com/mjmurphy3/DungeonGame/internal/render"
)

// bannerFont is a 5-row block font for the title and victory banners.
var bannerFont = map[rune][]string{
	'D': {"#### ", "#   #", "#   #", "#   #", "#### "},
	'U': {"#   #", "#   #", "#   #", "#   #", " ### "},
	'N': {"#   #", "##  #", "# # #", "#  ##", "#   #"},
	'G': {" ####", "#    ", "#  ##", "#   #", " ### "},
	'E': {"#####", "#    ", "###  ", "#    ", "#####"},
	'O': {" ### ", "#   #", "#   #", "#   #", " ### "},
	'A': {" ### ", "#   #", "#####", "#   #", "#   #"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'V': {"#   #", "#   #", "#   #", " # # ", "  #  "},
	'I': {"###", " # ", " # ", " # ", "###"},
	'C': {" ####", "#    ", "#    ", "#    ", " ####"},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'R': {"#### ", "#   #", "#### ", "#  # ", "#   #"},
	'Y': {"#   #", " # # ", "  #  ", "  #  ", "  #  "},
	' ': {"  ", "  ", "  ", "  ", "  "},
}

// bannerWidth measures the rendered width of text at a pixel scale.
func bannerWidth(text string, scale int) int {
	w := 0
	for _, ch := range text {
		if glyph, ok := bannerFont[ch]; ok {
			w += (len(glyph[0]) + 1) * scale
		}
	}
	return w
}

// drawBanner renders block text centered on screen width w at row y, with a
// per-row color, doubling pixel width when the screen is wide enough.
func (g *Game) drawBanner(text string, w, y int, rowColor func(row int) tcell.Color) int {
	scale := 2
	if bannerWidth(text, scale) > w-4 {
		scale = 1
	}
	x := (w - bannerWidth(text, scale)) / 2
	for _, ch := range text {
		glyph, ok := bannerFont[ch]
		if !ok {
			continue
		}
		for row, line := range glyph {
			for col, c := range line {
				if c != '#' {
					continue
				}
				for s := 0; s < scale; s++ {
					g.scr.SetCell(x+col*scale+s, y+row, '█', rowColor(row), tcell.ColorBlack)
				}
			}
		}
		x += (len(glyph[0]) + 1) * scale
	}
	return y + 5
}

var (
	titleGold  = tcell.NewRGBColor(255, 215, 110)
	titleEmber = tcell.NewRGBColor(200, 80, 20)
	starDim    = tcell.NewRGBColor(90, 90, 120)
	flameHot   = tcell.NewRGBColor(255, 230, 90)
	flameCool  = tcell.NewRGBColor(255, 130, 20)
	stickCol   = tcell.NewRGBColor(120, 80, 40)
)

// drawTitle paints the opening screen: starfield, flaming banner, help box.
func (g *Game) drawTitle(w, h int) {
	g.scr.Clear()

	// Sparse starfield over the upper half.
	for y := 0; y < h/2; y++ {
		for x := 0; x < w; x++ {
			if (x*31+y*57)%197 == 0 {
				g.scr.SetCell(x, y, '·', starDim, tcell.ColorBlack)
			}
		}
	}

	by := h / 8
	rowCol := func(row int) tcell.Color {
		return render.Lerp(titleGold, titleEmber, float64(row)/4)
	}
	bottom := g.drawBanner("DUNGEON GAME", w, by, rowCol)

	// Torches flanking the banner, flickering on alternate ticks.
	bw := bannerWidth("DUNGEON GAME", 2)
	if bw > w-4 {
		bw = bannerWidth("DUNGEON GAME", 1)
	}
	for _, tx := range []int{(w-bw)/2 - 5, (w+bw)/2 + 4} {
		flame := flameHot
		if (g.tick/4+tx)%2 == 0 {
			flame = flameCool
		}
		g.scr.SetCell(tx, by+1, '▲', flame, tcell.ColorBlack)
		g.scr.SetCell(tx, by+2, '█', stickCol, tcell.ColorBlack)
		g.scr.SetCell(tx, by+3, '█', stickCol, tcell.ColorBlack)
	}

	sub := "an adventure of tiles and torchlight"
	g.scr.Text((w-len(sub))/2, bottom+1, sub, starDim, tcell.ColorBlack)

	// Help box.
	help := []string{
		"WASD/arrows .. move (overworld) / walk and turn (dungeon)",
		"Q / E ........ strafe left / right (dungeon)",
		"SPACE ........ fire a magic missile (1-10 damage)",
		"Orcs claw up close; skeleton arrows hit hard from afar -",
		"if you can't see your attacker, LOOK AROUND!",
		"Chests give gold and +5 HP; ladders climb to the surface",
		"The DOCTOR heals +40, the PUB and INN +10 (5 min rest)",
		"Lava burns 10 HP a step - you heal 1 HP every 20 seconds",
		"",
		"WIN: claim 70% of all dungeon gold, or slay every foe",
	}
	boxW := 0
	for _, s := range help {
		if len(s) > boxW {
			boxW = len(s)
		}
	}
	boxW += 6
	boxH := len(help) + 4
	bx, byy := (w-boxW)/2, bottom+3
	g.box(bx, byy, boxW, boxH, titleGold, tcell.ColorBlack)
	caption := " HOW TO PLAY "
	g.scr.Text(bx+(boxW-len(caption))/2, byy, caption, titleGold, tcell.ColorBlack)
	for i, s := range help {
		g.scr.Text(bx+3, byy+2+i, s, hudText, tcell.ColorBlack)
	}

	pulse := 0.55 + 0.45*math.Sin(float64(g.tick)/7)
	prompt := ">> Press ENTER to begin - ESC to quit <<"
	g.scr.Text((w-len(prompt))/2, byy+boxH+2, prompt, render.Shade(titleGold, pulse), tcell.ColorBlack)

	world := fmt.Sprintf("This world hides %d dungeons, %d gold pieces and %d monsters.",
		len(g.dungeons), g.totalGold, g.totalFoes)
	g.scr.Text((w-len(world))/2, byy+boxH+4, world, starDim, tcell.ColorBlack)
}

var (
	skyTop    = tcell.NewRGBColor(30, 25, 80)
	skyHor    = tcell.NewRGBColor(255, 150, 60)
	sunCore   = tcell.NewRGBColor(255, 240, 150)
	sunRim    = tcell.NewRGBColor(255, 180, 60)
	mountSil  = tcell.NewRGBColor(60, 40, 80)
	grassNear = tcell.NewRGBColor(30, 90, 35)
	grassFar  = tcell.NewRGBColor(70, 150, 60)
	castleCol = tcell.NewRGBColor(160, 160, 175)
	castleDk  = tcell.NewRGBColor(95, 95, 110)
	windowLit = tcell.NewRGBColor(255, 220, 90)
	birdCol   = tcell.NewRGBColor(40, 35, 60)
	flower1   = tcell.NewRGBColor(255, 120, 160)
	flower2   = tcell.NewRGBColor(250, 240, 120)
)

// drawVictory paints the win splash: sunrise over mountains, a castle on a
// flowered meadow, and the tally of the player's deeds.
func (g *Game) drawVictory(w, h int) {
	horizon := h * 3 / 5
	sunRy := float64(h) / 5
	sunRx := sunRy * 2.2 // terminal cells are ~half as wide as tall

	// Sky with the sun rising at the horizon's center.
	for y := 0; y < horizon; y++ {
		t := float64(y) / float64(horizon)
		sky := render.Lerp(skyTop, skyHor, t*t)
		for x := 0; x < w; x++ {
			c := sky
			dx := (float64(x) - float64(w)/2) / sunRx
			dy := (float64(y) - float64(horizon)) / sunRy
			if d := dx*dx + dy*dy; d <= 1 {
				c = sunRim
				if d <= 0.55 {
					c = sunCore
				}
			}
			g.scr.SetCell(x, y, ' ', c, c)
		}
	}

	// Mountain silhouettes in front of the sky.
	peaks := [][3]float64{ // {center fraction, height fraction, slope}
		{0.18, 0.45, 1.6},
		{0.40, 0.28, 1.9},
		{0.80, 0.38, 1.7},
	}
	for x := 0; x < w; x++ {
		mh := 0.0
		for _, p := range peaks {
			ph := p[1] * float64(horizon)
			d := math.Abs(float64(x)-p[0]*float64(w)) * p[2] * float64(horizon) / float64(w) * 2
			if v := ph - d; v > mh {
				mh = v
			}
		}
		for y := horizon - int(mh); y < horizon; y++ {
			if y >= 0 {
				g.scr.SetCell(x, y, ' ', mountSil, mountSil)
			}
		}
	}

	// Meadow with scattered flowers.
	for y := horizon; y < h; y++ {
		t := float64(y-horizon) / float64(h-horizon)
		c := render.Lerp(grassFar, grassNear, t)
		for x := 0; x < w; x++ {
			ch, fg := ' ', c
			if (x*53+y*101)%89 == 0 {
				ch = '*'
				fg = flower1
				if (x+y)%2 == 0 {
					fg = flower2
				}
			}
			g.scr.SetCell(x, y, ch, fg, c)
		}
	}

	g.drawCastle(w*3/4, horizon)

	// Gliding birds.
	for i := 0; i < 3; i++ {
		bx := (g.tick/6 + i*37) % (w + 10)
		by := h/6 + (i*5)%7 + int(math.Sin(float64(g.tick)/20+float64(i))*1.5)
		if bx < w && by >= 0 && by < horizon {
			g.scr.Text(bx-1, by, "~v~", birdCol, render.Lerp(skyTop, skyHor, float64(by)/float64(horizon)))
		}
	}

	// Banner and tally.
	rowCol := func(row int) tcell.Color {
		return render.Lerp(sunCore, tcell.NewRGBColor(255, 140, 40), float64(row)/4)
	}
	bottom := g.drawBanner("VICTORY", w, h/12, rowCol)

	tally := fmt.Sprintf("You claimed %d of %d gold pieces and felled %d of %d foes.",
		g.gold, g.totalGold, g.foesKilled, g.totalFoes)
	g.scr.Text((w-len(tally))/2, bottom+2, tally, sunCore, skyTop)

	pulse := 0.55 + 0.45*math.Sin(float64(g.tick)/7)
	prompt := " Press R for a new adventure - ESC to rest at last "
	g.scr.Text((w-len(prompt))/2, h-2, prompt, render.Shade(sunCore, pulse), grassNear)
}

var (
	deathGlow  = tcell.NewRGBColor(70, 10, 8)
	boneCol    = tcell.NewRGBColor(225, 218, 200)
	socketDark = tcell.NewRGBColor(15, 8, 8)
	sockDim    = tcell.NewRGBColor(120, 20, 10)
	sockHot    = tcell.NewRGBColor(255, 60, 30)
	emberHot   = tcell.NewRGBColor(255, 180, 60)
	emberDim   = tcell.NewRGBColor(110, 45, 25)
	bloodHi    = tcell.NewRGBColor(255, 80, 60)
	bloodLo    = tcell.NewRGBColor(110, 10, 10)
)

// inE tests a point against an ellipse, for the procedural skull.
func inE(u, v, cx, cy, rx, ry float64) bool {
	du, dv := (u-cx)/rx, (v-cy)/ry
	return du*du+dv*dv <= 1
}

// drawDeath paints the game-over scene: ember-lit darkness, a glowering
// skull with pulsing eyes, ash rising from the ground, and the epitaph.
func (g *Game) drawDeath(w, h int) {
	// Darkness fading down into ember-glow.
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		bg := render.Lerp(tcell.ColorBlack, deathGlow, t*t)
		g.scr.FillRow(y, 0, w, ' ', bg, bg)
	}

	ry := h / 6
	if ry < 4 {
		ry = 4
	}
	cy := h * 30 / 100
	g.drawSkull(w/2, cy, ry)

	// Embers drifting up from the ground, cooling as they climb.
	for i := 0; i < 40; i++ {
		px := (i*89)%w + int(3*math.Sin(float64(g.tick)/12+float64(i)))
		py := h - 1 - (g.tick/3+i*31)%(h+6)
		if px < 0 || px >= w || py < 0 || py >= h {
			continue
		}
		t := float64(py) / float64(h)
		ch := '·'
		if t > 0.7 {
			ch = '*'
		} else if t > 0.4 {
			ch = '•'
		}
		bt := t * t
		bg := render.Lerp(tcell.ColorBlack, deathGlow, bt)
		g.scr.SetCell(px, py, ch, render.Lerp(emberDim, emberHot, t), bg)
	}

	rowCol := func(row int) tcell.Color {
		return render.Lerp(bloodHi, bloodLo, float64(row)/4)
	}
	bottom := g.drawBanner("YOU DIED", w, cy+ry*5/4+2, rowCol)

	spacing := 1
	if h >= 40 {
		spacing = 2
	}
	center := func(y int, s string, fg tcell.Color) {
		g.scr.Text((w-len(s))/2, y, s, fg, tcell.ColorBlack)
	}
	center(bottom+spacing, g.msg, hudText)
	center(bottom+spacing*2, fmt.Sprintf("You perished with %d gold, %d foes avenged you not.", g.gold, g.totalFoes-g.foesKilled), goldColor)
	pulse := 0.55 + 0.45*math.Sin(float64(g.tick)/7)
	center(bottom+spacing*3, ">> Press R to rise again in a new world - ESC to surrender <<", render.Shade(bloodHi, pulse))
}

// drawSkull renders a block-art skull centered at (cx, cy) with vertical
// radius ry cells; the eye sockets smolder in time with the clock.
func (g *Game) drawSkull(cx, cy, ry int) {
	rx := float64(ry) * 2.1 // cells are tall; stretch horizontally to look round
	pulse := 0.5 + 0.5*math.Sin(float64(g.tick)/6)
	glow := render.Lerp(sockDim, sockHot, pulse)

	for dy := -ry; dy <= ry*3/2; dy++ {
		v := float64(dy) / float64(ry)
		for dx := -int(rx) - 1; dx <= int(rx)+1; dx++ {
			u := float64(dx) / rx
			head := u*u+v*v <= 1 && v <= 0.45
			jaw := v > 0.45 && v <= 1.18 && math.Abs(u) <= 0.60-0.25*(v-0.45)
			if !head && !jaw {
				continue
			}
			// Rounded shading so the dome reads as bone, not a flat blob.
			c := render.Shade(boneCol, 0.78+0.22*(1-(u*u+v*v)*0.55))
			switch {
			case inE(u, v, -0.40, -0.08, 0.23, 0.30) || inE(u, v, 0.40, -0.08, 0.23, 0.30):
				c = socketDark
				if inE(u, v, -0.40, -0.02, 0.10, 0.13) || inE(u, v, 0.40, -0.02, 0.10, 0.13) {
					c = glow
				}
			case v > 0.10 && v <= 0.42 && math.Abs(u) < (v-0.10)*0.30:
				c = socketDark // nasal cavity
			case v > 0.68: // teeth with seams
				if math.Mod((u+1)*5, 1) < 0.16 || math.Abs(v-0.90) < 0.05 {
					c = render.Shade(boneCol, 0.40)
				}
			}
			g.scr.SetCell(cx+dx, cy+dy, ' ', c, c)
		}
	}
}

// drawCastle places a small keep with battlements, a gate, lit windows and a
// pennant, its base on the horizon line.
func (g *Game) drawCastle(cx, baseY int) {
	const cw, towerH, wallH = 19, 7, 4
	x0 := cx - cw/2
	put := func(x, y int, ch rune, fg, bg tcell.Color) {
		g.scr.SetCell(x, y, ch, fg, bg)
	}
	// Towers at each end, wall between.
	for i := 0; i < cw; i++ {
		isTower := i < 3 || i >= cw-3
		hgt := wallH
		if isTower {
			hgt = towerH
		}
		for j := 1; j <= hgt; j++ {
			put(x0+i, baseY-j, '█', castleCol, castleDk)
		}
		// Battlement merlons on top.
		if i%2 == 0 {
			put(x0+i, baseY-hgt-1, '█', castleCol, tcell.ColorBlack)
		}
	}
	// Gate arch.
	put(cx-1, baseY-1, '█', castleDk, castleDk)
	put(cx, baseY-1, '█', castleDk, castleDk)
	put(cx+1, baseY-1, '█', castleDk, castleDk)
	put(cx, baseY-2, '█', castleDk, castleDk)
	// Lit windows in the towers.
	put(x0+1, baseY-towerH+2, '▪', windowLit, castleDk)
	put(x0+cw-2, baseY-towerH+2, '▪', windowLit, castleDk)
	// Pennant on the left tower.
	put(x0+1, baseY-towerH-2, '│', castleDk, tcell.ColorBlack)
	put(x0+2, baseY-towerH-2, '▶', tcell.NewRGBColor(220, 40, 50), tcell.ColorBlack)
}
