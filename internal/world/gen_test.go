package world

import "testing"

func TestGenerateWorlds(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		w := Generate(seed)

		if !w.At(w.Spawn.X, w.Spawn.Y).Passable() {
			t.Fatalf("seed %d: spawn %v is not passable", seed, w.Spawn)
		}
		if n := len(w.Entrances); n < 3 || n > 5 {
			t.Fatalf("seed %d: got %d entrances, want 3-5", seed, n)
		}

		reach := w.reachableFrom(w.Spawn)
		for i, e := range w.Entrances {
			if w.At(e.Pos.X, e.Pos.Y) != TEntrance {
				t.Errorf("seed %d: entrance %d cell is %v, want TEntrance", seed, i, w.At(e.Pos.X, e.Pos.Y))
			}
			if !w.At(e.Exit.X, e.Exit.Y).Passable() {
				t.Errorf("seed %d: entrance %d exit pad %v not passable", seed, i, e.Exit)
			}
			if !reach[e.Exit.Y*w.W+e.Exit.X] {
				t.Errorf("seed %d: entrance %d exit pad %v unreachable from spawn", seed, i, e.Exit)
			}
			if w.EntranceAt(e.Pos.X, e.Pos.Y) != i {
				t.Errorf("seed %d: EntranceAt does not find entrance %d", seed, i)
			}
		}

		// The town labels exist and sit on wall or brick tiles.
		if len(w.Labels) == 0 {
			t.Fatalf("seed %d: no labels generated", seed)
		}
		for idx := range w.Labels {
			if tt := w.Tiles[idx]; tt != TWall && tt != TBrick {
				t.Errorf("seed %d: label on tile %v, want wall or brick", seed, tt)
			}
		}
	}
}

// TestEntrancesNeverLavaGated verifies every dungeon entrance can be walked
// to from spawn without stepping on a single lava tile, and is never sealed
// inside a mountain range (the exit pad must be a non-lava walkable
// neighbor on the spawn's lava-free flood fill).
func TestEntrancesNeverLavaGated(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		w := Generate(seed)
		reach := w.reachableFrom(w.Spawn) // excludes lava by definition
		for i, e := range w.Entrances {
			if !reach[e.Exit.Y*w.W+e.Exit.X] {
				t.Errorf("seed %d: entrance %d at %v requires crossing lava or is sealed", seed, i, e.Pos)
			}
			if w.At(e.Exit.X, e.Exit.Y) == TLava {
				t.Errorf("seed %d: entrance %d exit pad is lava", seed, i)
			}
			adj := false
			for _, d := range [4]Point{{0, 1}, {0, -1}, {-1, 0}, {1, 0}} {
				if e.Exit.X == e.Pos.X+d.X && e.Exit.Y == e.Pos.Y+d.Y {
					adj = true
				}
			}
			if !adj {
				t.Errorf("seed %d: entrance %d exit pad %v not adjacent to %v", seed, i, e.Exit, e.Pos)
			}
		}
	}
}

// TestHealersScattered verifies the three healer buildings exist, are spread
// out across the map, and all open onto ground reachable from spawn.
func TestHealersScattered(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		w := Generate(seed)
		if len(w.Healers) != 3 {
			t.Fatalf("seed %d: got %d healers, want 3", seed, len(w.Healers))
		}
		want := map[string]int{"DOCTOR": 40, "PUB": 10, "INN": 10}
		reach := w.reachableFrom(w.Spawn)
		for i, h := range w.Healers {
			if want[h.Name] != h.Heal {
				t.Errorf("seed %d: %s heals %d, want %d", seed, h.Name, h.Heal, want[h.Name])
			}
			if !reach[h.Door.Y*w.W+h.Door.X] {
				t.Errorf("seed %d: %s door unreachable from spawn", seed, h.Name)
			}
			for j := 0; j < i; j++ {
				o := w.Healers[j]
				dx := (h.X + h.W/2) - (o.X + o.W/2)
				dy := (h.Y + h.H/2) - (o.Y + o.H/2)
				if dx*dx+dy*dy < 25*25 {
					t.Errorf("seed %d: %s and %s are clustered (%d apart^2)", seed, h.Name, o.Name, dx*dx+dy*dy)
				}
			}
		}
	}
}

// TestContinentFitsInsideBorders verifies the radial falloff: the outer rim
// of the map must be open water on every side, so the coastline is always
// reached before the edge of the world.
func TestContinentFitsInsideBorders(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		w := Generate(seed)
		const rim = 3
		for y := 0; y < w.H; y++ {
			for x := 0; x < w.W; x++ {
				if x >= rim && x < w.W-rim && y >= rim && y < w.H-rim {
					continue
				}
				if tt := w.At(x, y); tt != TDeepWater && tt != TWater {
					t.Fatalf("seed %d: land tile %v at border (%d,%d)", seed, tt, x, y)
				}
			}
		}
	}
}

func TestWaterAnimationSeamless(t *testing.T) {
	w := Generate(3)
	// The animation phase must depend only on world coords and tick, never on
	// where the camera is: calling CellAt twice for the same tile and tick
	// must agree.
	for _, tick := range []int{0, 17, 300} {
		ch1, fg1, bg1 := w.CellAt(10, 10, tick)
		ch2, fg2, bg2 := w.CellAt(10, 10, tick)
		if ch1 != ch2 || fg1 != fg2 || bg1 != bg2 {
			t.Fatalf("CellAt not deterministic at tick %d", tick)
		}
	}
}
