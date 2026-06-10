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
