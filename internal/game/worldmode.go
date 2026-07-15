package game

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/mjmurphy3/DungeonGame/internal/world"
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
	if i := g.world.HealerAt(nx, ny); i >= 0 {
		g.visitHealer(i)
	}
}

// healerCooldown is how long a building rests between treatments.
const healerCooldown = 300.0 // seconds (5 minutes)

// visitHealer treats the player when they step inside a healer building.
func (g *Game) visitHealer(i int) {
	h := g.world.Healers[i]
	if g.healerCD[i] > 0 {
		g.say(fmt.Sprintf("The %s can do no more for you yet (%d:%02d).",
			strings.ToLower(h.Name), int(g.healerCD[i])/60, int(g.healerCD[i])%60))
		return
	}
	heal := min(h.Heal, maxHP-g.hp)
	if heal <= 0 {
		g.say("You are already in perfect health.")
		return
	}
	g.hp += heal
	g.healerCD[i] = healerCooldown
	switch h.Name {
	case "DOCTOR":
		g.say(fmt.Sprintf("The doctor patches you up (+%d HP).", heal))
	case "PUB":
		g.say(fmt.Sprintf("A hearty meal and a pint (+%d HP).", heal))
	default:
		g.say(fmt.Sprintf("A short rest at the inn (+%d HP).", heal))
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
	g.shots = nil
	g.say("You climb down into the dungeon...")
}

// leaveDungeon returns the player to the overworld tile just outside the
// entrance they used.
func (g *Game) leaveDungeon() {
	e := g.world.Entrances[g.curDungeon]
	g.mode = ModeWorld
	g.px, g.py = e.Exit.X, e.Exit.Y
	g.missiles = nil
	g.shots = nil
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
