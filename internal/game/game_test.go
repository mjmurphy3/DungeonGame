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

	if len(d.Monsters) == 0 {
		t.Fatal("test dungeon has no monsters")
	}
	o := &d.Monsters[0]
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
	if g.totalGold <= 0 || g.totalFoes <= 0 {
		t.Fatalf("totals not computed: gold %d foes %d", g.totalGold, g.totalFoes)
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
	g.foesKilled = g.totalFoes
	g.checkVictory()
	if g.mode != ModeVictory {
		t.Fatal("victory not triggered when all foes are slain")
	}
}

// soloOrc returns a dungeon-mode game where only one orc is alive, with the
// player standing right beside it.
func soloOrc(t *testing.T) (*Game, *dungeon.Monster) {
	t.Helper()
	g := newHeadless(3)
	d := dungeon.Generate(7)
	g.dungeons = []*dungeon.Dungeon{d}
	g.curDungeon = 0
	g.mode = ModeDungeon

	var o *dungeon.Monster
	for i := range d.Monsters {
		if o == nil && d.Monsters[i].Kind == dungeon.MOrc {
			o = &d.Monsters[i]
		} else {
			d.Monsters[i].HP = 0 // only the test orc may act
		}
	}
	if o == nil {
		t.Fatal("test dungeon has no orcs")
	}
	o.AttackCD = 0
	g.fx, g.fy = o.X+0.5, o.Y // inside melee reach
	return g, o
}

func TestOrcMeleeDamageBounds(t *testing.T) {
	g, o := soloOrc(t)
	g.ang = 0
	g.updateMonsters(g.dungeons[0])

	dealt := maxHP - g.hp
	if dealt < 1 || dealt > orcMeleeMax {
		t.Fatalf("orc melee dealt %d damage, want 1-%d", dealt, orcMeleeMax)
	}
	if o.AttackCD <= 0 {
		t.Fatal("orc attack cooldown not set after striking")
	}
}

func TestRearAttackWarns(t *testing.T) {
	// The orc stands at -x from the player. Facing +x means it's behind.
	g, o := soloOrc(t)
	g.ang = 0
	g.updateMonsters(g.dungeons[0])
	if g.warnT <= 0 {
		t.Fatal("no warning when struck from behind")
	}

	// Facing the orc (-x direction): no warning.
	g.warnT = 0
	o.AttackCD = 0
	g.ang = 3.14159
	g.updateMonsters(g.dungeons[0])
	if g.warnT > 0 {
		t.Fatal("warning raised for an attacker in plain view")
	}
}

func TestSkeletonArrowStings(t *testing.T) {
	g := newHeadless(4)
	d := dungeon.Generate(7)
	g.dungeons = []*dungeon.Dungeon{d}
	g.curDungeon = 0
	g.mode = ModeDungeon
	g.fx, g.fy = d.StartX, d.StartY

	g.shots = []Shot{{X: g.fx, Y: g.fy, DX: 0.001, DY: 0}}
	g.updateShots(d)

	if got := maxHP - g.hp; got < 1 || got > skelShotMax {
		t.Fatalf("arrow dealt %d damage, want 1-%d", got, skelShotMax)
	}
	if len(g.shots) != 0 {
		t.Fatal("arrow should be consumed on hit")
	}
}

func TestChestHealsFive(t *testing.T) {
	g := newHeadless(5)
	d := dungeon.Generate(99)
	g.dungeons = []*dungeon.Dungeon{d}
	g.curDungeon = 0
	g.mode = ModeDungeon
	if len(d.Chests) == 0 {
		t.Fatal("test dungeon has no chests")
	}

	c := &d.Chests[0]
	g.hp = 50
	g.fx, g.fy = c.X, c.Y
	g.updateChests(d)

	if !c.Opened {
		t.Fatal("chest did not open")
	}
	if g.hp != 50+chestHeal {
		t.Fatalf("hp = %d after chest, want %d", g.hp, 50+chestHeal)
	}
	if g.gold != c.Gold {
		t.Fatalf("gold = %d, want %d", g.gold, c.Gold)
	}
}

func TestDoctorHealsWithCooldown(t *testing.T) {
	g := newHeadless(6)
	h := g.world.Healers[0]
	if h.Name != "DOCTOR" || h.Heal != 40 {
		t.Fatalf("healer 0 = %s/+%d, want DOCTOR/+40", h.Name, h.Heal)
	}

	g.hp = 40
	g.px, g.py = h.Door.X, h.Door.Y
	g.worldKey('w') // step inside
	if g.hp != 80 {
		t.Fatalf("hp = %d after doctor, want 80", g.hp)
	}
	if g.healerCD[0] <= 0 {
		t.Fatal("doctor cooldown not started")
	}

	// Walking back in during the cooldown does nothing.
	g.worldKey('s')
	g.worldKey('w')
	if g.hp != 80 {
		t.Fatalf("hp = %d on cooldown revisit, want 80", g.hp)
	}
}
