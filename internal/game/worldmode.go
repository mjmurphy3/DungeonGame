package game

import (
	"github.com/gdamore/tcell/v2"

	"dungeongame/internal/world"
)

// worldKey handles one keypress in overworld mode (one tile per press;
// terminal autorepeat provides held-key walking).
func (g *Game) worldKey(ch rune) {
	dx, dy := 0, 0
	switch ch {
	case 'w':
		dy = -1
	case 's':
		dy = 1
	case 'a':
		dx = -1
	case 'd':
		dx = 1
	case ' ':
		g.say("Your magic fizzles in the open air.")
		return
	default:
		return
	}
	nx, ny := g.px+dx, g.py+dy
	t := g.world.At(nx, ny)
	if !t.Passable() {
		return
	}
	g.px, g.py = nx, ny

	switch t {
	case world.TLava:
		g.damage(lavaDamage, "The lava consumes you.")
		if g.mode != ModeDead {
			g.say("The lava sears you for 10!")
		}
	case world.TEntrance:
		if i := g.world.EntranceAt(nx, ny); i >= 0 {
			g.enterDungeon(i)
		}
	}
}

// enterDungeon drops the player at the level's entry ladder. All dungeons are
// generated with the world, and their state persists between visits.
func (g *Game) enterDungeon(i int) {
	d := g.dungeons[i]
	g.curDungeon = i
	g.mode = ModeDungeon
	g.fx, g.fy, g.ang = d.StartX, d.StartY, d.StartA
	g.missiles = nil
	g.say("You climb down into the dungeon...")
}

// leaveDungeon returns the player to the overworld tile just outside the
// entrance they used.
func (g *Game) leaveDungeon() {
	e := g.world.Entrances[g.curDungeon]
	g.mode = ModeWorld
	g.px, g.py = e.Exit.X, e.Exit.Y
	g.missiles = nil
	g.say("You climb back out into the daylight.")
}

// drawWorld renders the scrolling tile viewport centered on the player.
func (g *Game) drawWorld(w, h int) {
	ox := g.px - w/2
	oy := g.py - h/2
	for sy := 0; sy < h; sy++ {
		for sx := 0; sx < w; sx++ {
			ch, fg, bg := g.world.CellAt(ox+sx, oy+sy, g.tick)
			g.scr.SetCell(sx, sy, ch, fg, bg)
		}
	}
	// Player on top, keeping the tile's background.
	_, _, bg := g.world.CellAt(g.px, g.py, g.tick)
	g.scr.SetCell(w/2, h/2, '@', tcell.ColorWhite, bg)
}
