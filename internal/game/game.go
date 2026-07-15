// Package game owns the state machine and main loop tying the overworld,
// the dungeons and the renderer together.
package game

import (
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/mjmurphy3/DungeonGame/internal/dungeon"
	"github.com/mjmurphy3/DungeonGame/internal/raycast"
	"github.com/mjmurphy3/DungeonGame/internal/render"
	"github.com/mjmurphy3/DungeonGame/internal/world"
)

// Mode is the current play state.
type Mode int

const (
	ModeTitle Mode = iota
	ModeWorld
	ModeDungeon
	ModeDead
	ModeVictory
)

const (
	maxHP       = 100
	fps         = 30
	dt          = 1.0 / fps
	regenPeriod = 20 // seconds per healed hit point
	lavaDamage  = 10
)

// Missile is a magic bolt in flight inside a dungeon.
type Missile struct {
	X, Y, DX, DY float64
}

// Game is the full session state.
type Game struct {
	scr *render.Screen
	rng *rand.Rand

	world      *world.World
	dungeons   []*dungeon.Dungeon // generated lazily, state persists per visit
	curDungeon int

	mode Mode
	tick int

	// Overworld position (tile coords).
	px, py int

	// Dungeon position (continuous) and facing.
	fx, fy, ang float64

	hp, gold   int
	regenTicks int

	// Victory bookkeeping: win at 70% of all chest gold, or every foe slain.
	totalGold  int
	goldGoal   int
	totalFoes  int
	foesKilled int

	warnT    float64   // "look around!" indicator time remaining
	healerCD []float64 // per-building cooldown, seconds

	missiles []Missile
	shots    []Shot // enemy projectiles in flight
	rc       raycast.Renderer
	pb       *render.PixelBuf

	msg  string
	msgT float64
}

// New builds a fresh game on a generated world.
func New(scr *render.Screen, seed int64) *Game {
	g := &Game{scr: scr}
	g.reset(seed)
	return g
}

func (g *Game) reset(seed int64) {
	g.rng = rand.New(rand.NewSource(seed))
	g.world = world.Generate(seed)

	// Generate every dungeon up front so the victory goals are known.
	g.dungeons = make([]*dungeon.Dungeon, len(g.world.Entrances))
	g.totalGold, g.totalFoes, g.foesKilled = 0, 0, 0
	for i := range g.dungeons {
		d := dungeon.Generate(seed*131 + int64(i)*7919)
		g.dungeons[i] = d
		for _, c := range d.Chests {
			g.totalGold += c.Gold
		}
		g.totalFoes += len(d.Monsters)
	}
	g.goldGoal = (g.totalGold*7 + 9) / 10 // ceil(70%)

	g.mode = ModeTitle
	g.px, g.py = g.world.Spawn.X, g.world.Spawn.Y
	g.hp, g.gold = maxHP, 0
	g.regenTicks = 0
	g.missiles = nil
	g.shots = nil
	g.warnT = 0
	g.healerCD = make([]float64, len(g.world.Healers))
	g.say("Welcome, wanderer. Dungeons hide in the forests and mountains.")
}

// checkVictory flips to the victory splash once either goal is met.
func (g *Game) checkVictory() {
	if g.mode == ModeDead || g.mode == ModeVictory {
		return
	}
	goldWin := g.totalGold > 0 && g.gold >= g.goldGoal
	foeWin := g.totalFoes > 0 && g.foesKilled >= g.totalFoes
	if goldWin || foeWin {
		g.mode = ModeVictory
	}
}

func (g *Game) say(s string) { g.msg, g.msgT = s, 4.0 }

// Run drives the event/update/draw loop until the player quits with ESC.
func (g *Game) Run() {
	events := make(chan tcell.Event, 16)
	quit := make(chan struct{})
	go func() {
		for {
			ev := g.scr.T.PollEvent()
			if ev == nil {
				return
			}
			select {
			case events <- ev:
			case <-quit:
				return
			}
		}
	}()

	ticker := time.NewTicker(time.Second / fps)
	defer ticker.Stop()
	defer close(quit)

	for {
		select {
		case ev := <-events:
			switch e := ev.(type) {
			case *tcell.EventKey:
				if !g.handleKey(e) {
					return
				}
			case *tcell.EventResize:
				g.scr.T.Sync()
			}
		case <-ticker.C:
			g.update()
			g.draw()
		}
	}
}

// handleKey processes one key event; it returns false to quit the game.
func (g *Game) handleKey(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlC {
		return false
	}
	ch := e.Rune()
	if ch >= 'A' && ch <= 'Z' {
		ch += 'a' - 'A'
	}
	// Arrow keys mirror WASD in every mode.
	switch e.Key() {
	case tcell.KeyUp:
		ch = 'w'
	case tcell.KeyDown:
		ch = 's'
	case tcell.KeyLeft:
		ch = 'a'
	case tcell.KeyRight:
		ch = 'd'
	}
	switch g.mode {
	case ModeTitle:
		if e.Key() == tcell.KeyEnter || ch == ' ' {
			g.mode = ModeWorld
		}
	case ModeWorld:
		g.worldKey(ch)
	case ModeDungeon:
		g.dungeonKey(ch)
	case ModeDead, ModeVictory:
		if ch == 'r' {
			g.reset(time.Now().UnixNano())
		}
	}
	return true
}

// update advances simulation by one frame.
func (g *Game) update() {
	g.tick++
	// The startup resize escape is applied asynchronously by the terminal;
	// re-sync shortly after launch so tcell re-reads the settled window size
	// and the title screen centers without a manual resize.
	if g.scr != nil && (g.tick == 10 || g.tick == 45) {
		g.scr.T.Sync()
	}
	if g.msgT > 0 {
		g.msgT -= dt
	}
	if g.warnT > 0 {
		g.warnT -= dt
	}
	if g.mode == ModeTitle || g.mode == ModeDead || g.mode == ModeVictory {
		return // screens only animate; the world holds its breath
	}

	// Healer cooldowns run on play time, in any mode.
	for i := range g.healerCD {
		if g.healerCD[i] > 0 {
			g.healerCD[i] -= dt
		}
	}

	// Passive healing: +1 HP per minute of play, in every mode.
	g.regenTicks++
	if g.regenTicks >= regenPeriod*fps {
		g.regenTicks = 0
		if g.hp < maxHP {
			g.hp++
		}
	}

	if g.mode == ModeDungeon {
		g.updateDungeon()
	}
}

// damage applies damage to the player and handles death.
func (g *Game) damage(n int, cause string) {
	g.hp -= n
	if g.hp <= 0 {
		g.hp = 0
		g.mode = ModeDead
		g.say(cause)
	}
}

func (g *Game) draw() {
	w, h := g.scr.Size()
	if w < 80 || h < 24 {
		g.scr.Clear()
		g.scr.Text(0, 0, "Terminal too small - need at least 80x24.", tcell.ColorYellow, tcell.ColorBlack)
		g.scr.Show()
		return
	}
	switch g.mode {
	case ModeTitle:
		g.drawTitle(w, h)
	case ModeWorld:
		g.drawWorld(w, h-1)
	case ModeDungeon:
		g.drawDungeon(w, h-1)
	case ModeDead:
		g.drawDeath(w, h)
	case ModeVictory:
		g.drawVictory(w, h)
	}
	if g.mode == ModeWorld || g.mode == ModeDungeon {
		g.drawHUD(w, h)
		g.drawStatsBox()
		g.drawWarning(w)
	}
	g.scr.Show()
}
