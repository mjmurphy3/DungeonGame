package dungeon

import "testing"

func TestGenerateDungeons(t *testing.T) {
	for seed := int64(1); seed <= 12; seed++ {
		d := Generate(seed)

		if n := len(d.Rooms); n < 6 || n > 12 {
			t.Fatalf("seed %d: got %d rooms, want 6-12", seed, n)
		}
		if d.At(d.Entry[0], d.Entry[1]) != CEntry {
			t.Errorf("seed %d: entry cell is not CEntry", seed)
		}
		if d.At(d.Exit[0], d.Exit[1]) != CExit {
			t.Errorf("seed %d: exit cell is not CExit", seed)
		}
		if d.Entry == d.Exit {
			t.Errorf("seed %d: entry and exit share a cell", seed)
		}
		if !d.PathExists() {
			t.Fatalf("seed %d: no path from entry to exit", seed)
		}
		if !d.Walkable(d.StartX, d.StartY) {
			t.Errorf("seed %d: player start (%v,%v) not walkable", seed, d.StartX, d.StartY)
		}

		for i, o := range d.Orcs {
			if o.HP != 20 {
				t.Errorf("seed %d: orc %d spawned with %d HP, want 20", seed, i, o.HP)
			}
			if !d.At(int(o.X), int(o.Y)).Passable() {
				t.Errorf("seed %d: orc %d spawned inside a wall", seed, i)
			}
		}
		if len(d.Orcs) == 0 {
			t.Errorf("seed %d: dungeon has no orcs", seed)
		}
		if len(d.Torches) == 0 {
			t.Errorf("seed %d: dungeon has no torches", seed)
		}

		// Doors must be passages: framed by walls on one axis, open on the other.
		for y := 0; y < d.H; y++ {
			for x := 0; x < d.W; x++ {
				if d.At(x, y) != CDoor {
					continue
				}
				horiz := d.At(x-1, y).Passable() && d.At(x+1, y).Passable()
				vert := d.At(x, y-1).Passable() && d.At(x, y+1).Passable()
				if !horiz && !vert {
					t.Errorf("seed %d: door at (%d,%d) connects nothing", seed, x, y)
				}
			}
		}
	}
}

func TestOpenDoor(t *testing.T) {
	d := Generate(1)
	for y := 0; y < d.H; y++ {
		for x := 0; x < d.W; x++ {
			if d.At(x, y) == CDoor {
				d.OpenDoor(x, y)
				if d.At(x, y) != CDoorOpen {
					t.Fatalf("door at (%d,%d) did not open", x, y)
				}
				return
			}
		}
	}
	t.Skip("no door generated for seed 1")
}
