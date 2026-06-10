package game

import (
	"testing"

	"dungeongame/internal/dungeon"
)

// newHeadless builds a game without a terminal; update() never touches the
// screen, so simulation logic is fully testable. The title screen is skipped
// so the simulation runs.
func newHeadless(seed int64) *Game {
	g := &Game{}
	g.reset(seed)
	g.mode = ModeWorld
	return g
}

func TestRegenOneHPPerPeriod(t *testing.T) {
	g := newHeadless(1)
	g.hp = 50
	for i := 0; i < fps*regenPeriod; i++ {
		g.update()
	}
	if g.hp != 51 {
		t.Fatalf("after %ds hp = %d, want 51", regenPeriod, g.hp)
	}
	for i := 0; i < fps*regenPeriod*5; i++ {
		g.update()
	}
	if g.hp != 56 {
		t.Fatalf("after %ds hp = %d, want 56", regenPeriod*6, g.hp)
	}
}

func TestRegenCapsAtMax(t *testing.T) {
	g := newHeadless(1)
	g.hp = maxHP
	for i := 0; i < fps*(regenPeriod+1); i++ {
		g.update()
	}
	if g.hp != maxHP {
		t.Fatalf("hp regenerated past max: %d", g.hp)
	}
}

func TestDamageAndDeath(t *testing.T) {
	g := newHeadless(1)
	g.damage(lavaDamage, "lava")
	if g.hp != maxHP-10 {
		t.Fatalf("hp = %d after lava, want %d", g.hp, maxHP-10)
	}
	g.damage(1000, "doom")
	if g.mode != ModeDead || g.hp != 0 {
		t.Fatalf("expected death at 0 hp, got mode %v hp %d", g.mode, g.hp)
	}
}

func TestMissileDamagesOrc(t *testing.T) {
	g := newHeadless(2)
	d := dungeon.Generate(99)
	g.dungeons = []*dungeon.Dungeon{d}
	g.curDungeon = 0
	g.mode = ModeDungeon

	if len(d.Orcs) == 0 {
		t.Fatal("test dungeon has no orcs")
	}
	o := &d.Orcs[0]
	// Park a slow missile right on top of the orc and step the simulation.
	g.missiles = []Missile{{X: o.X, Y: o.Y, DX: 0.001, DY: 0}}
	g.updateMissiles(d)

	dealt := 20 - o.HP
	if dealt < 1 || dealt > 10 {
		t.Fatalf("missile dealt %d damage, want 1-10", dealt)
	}
	if len(g.missiles) != 0 {
		t.Fatalf("missile should be consumed on hit")
	}
}

func TestVictoryGoalsComputed(t *testing.T) {
	g := newHeadless(5)
	if g.totalGold <= 0 || g.totalOrcs <= 0 {
		t.Fatalf("totals not computed: gold %d orcs %d", g.totalGold, g.totalOrcs)
	}
	want := (g.totalGold*7 + 9) / 10
	if g.goldGoal != want {
		t.Fatalf("gold goal = %d, want ceil(70%%) = %d", g.goldGoal, want)
	}
	// Chest gold must be pre-rolled within the loot bounds.
	for _, d := range g.dungeons {
		for _, c := range d.Chests {
			if c.Gold < 10 || c.Gold > 50 {
				t.Fatalf("chest gold %d outside 10-50", c.Gold)
			}
		}
	}
}

func TestVictoryByGold(t *testing.T) {
	g := newHeadless(5)
	g.gold = g.goldGoal - 1
	g.checkVictory()
	if g.mode == ModeVictory {
		t.Fatal("victory triggered below the gold goal")
	}
	g.gold = g.goldGoal
	g.checkVictory()
	if g.mode != ModeVictory {
		t.Fatal("victory not triggered at the gold goal")
	}
}

func TestVictoryByOrcs(t *testing.T) {
	g := newHeadless(5)
	g.orcsKilled = g.totalOrcs
	g.checkVictory()
	if g.mode != ModeVictory {
		t.Fatal("victory not triggered when all orcs are slain")
	}
}

func TestOrcMeleeDamageBounds(t *testing.T) {
	g := newHeadless(3)
	d := dungeon.Generate(7)
	g.dungeons = []*dungeon.Dungeon{d}
	g.curDungeon = 0
	g.mode = ModeDungeon

	o := &d.Orcs[0]
	for i := 1; i < len(d.Orcs); i++ {
		d.Orcs[i].HP = 0 // only the test orc may swing this frame
	}
	o.AttackCD = 0
	g.fx, g.fy = o.X+0.5, o.Y // inside melee reach
	g.updateOrcs(d)

	dealt := maxHP - g.hp
	if dealt < 1 || dealt > orcMeleeMax {
		t.Fatalf("orc melee dealt %d damage, want 1-%d", dealt, orcMeleeMax)
	}
	if o.AttackCD <= 0 {
		t.Fatal("orc attack cooldown not set after striking")
	}
}
