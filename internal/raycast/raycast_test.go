package raycast

import (
	"testing"

	"dungeongame/internal/dungeon"
	"dungeongame/internal/render"
)

func frameSetup() (*Renderer, *render.PixelBuf, *dungeon.Dungeon, func(int, int) WallKind, []Sprite) {
	d := dungeon.Generate(42)
	pb := render.NewPixelBuf(256, 63)
	grid := func(x, y int) WallKind {
		switch d.At(x, y) {
		case dungeon.CWall:
			return WallStone
		case dungeon.CDoor:
			return WallDoor
		}
		return WallNone
	}
	sprites := []Sprite{}
	for _, o := range d.Orcs {
		sprites = append(sprites, Sprite{X: o.X, Y: o.Y, Kind: KOrc, Scale: 0.62})
	}
	for _, c := range d.Columns {
		sprites = append(sprites, Sprite{X: c.X, Y: c.Y, Kind: KColumn, Scale: 1.0})
	}
	for _, tc := range d.Torches {
		sprites = append(sprites, Sprite{X: tc.X, Y: tc.Y, Kind: KTorch, Scale: 0.38, Lift: 0.4})
	}
	return &Renderer{}, pb, d, grid, sprites
}

// TestRenderFrame exercises a full frame at target resolution from several
// camera angles; it guards against panics and out-of-range indexing.
func TestRenderFrame(t *testing.T) {
	r, pb, d, grid, sprites := frameSetup()
	for _, ang := range []float64{0, 0.7, 1.6, 3.1, 4.7, 6.0} {
		r.Render(pb, grid, d.StartX, d.StartY, ang, sprites, 12)
	}
}

func BenchmarkRenderFrame(b *testing.B) {
	r, pb, d, grid, sprites := frameSetup()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Render(pb, grid, d.StartX, d.StartY, float64(i)*0.01, sprites, i)
	}
}
