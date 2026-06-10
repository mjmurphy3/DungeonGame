// Package render wraps tcell with cell drawing helpers and a half-block
// pixel framebuffer used by the dungeon raycaster.
package render

import (
	"github.com/gdamore/tcell/v2"
)

// Screen owns the tcell screen and exposes simple drawing primitives.
type Screen struct {
	T tcell.Screen
}

// New initializes the terminal screen.
func New() (*Screen, error) {
	t, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := t.Init(); err != nil {
		return nil, err
	}
	t.HideCursor()
	t.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	return &Screen{T: t}, nil
}

func (s *Screen) Fini()            { s.T.Fini() }
func (s *Screen) Size() (int, int) { return s.T.Size() }
func (s *Screen) Show()            { s.T.Show() }
func (s *Screen) Clear()           { s.T.Clear() }

// SetCell draws one character cell.
func (s *Screen) SetCell(x, y int, ch rune, fg, bg tcell.Color) {
	s.T.SetContent(x, y, ch, nil, tcell.StyleDefault.Foreground(fg).Background(bg))
}

// Text draws a string starting at (x, y).
func (s *Screen) Text(x, y int, str string, fg, bg tcell.Color) {
	for i, ch := range []rune(str) {
		s.SetCell(x+i, y, ch, fg, bg)
	}
}

// FillRow fills a full row of cells with one rune/style.
func (s *Screen) FillRow(y, x0, x1 int, ch rune, fg, bg tcell.Color) {
	for x := x0; x < x1; x++ {
		s.SetCell(x, y, ch, fg, bg)
	}
}

// Shade scales an RGB color toward black by factor f in [0,1] (1 = unchanged).
func Shade(c tcell.Color, f float64) tcell.Color {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	r, g, b := c.TrueColor().RGB()
	return tcell.NewRGBColor(int32(float64(r)*f), int32(float64(g)*f), int32(float64(b)*f))
}

// Lerp blends color a toward b by t in [0,1].
func Lerp(a, b tcell.Color, t float64) tcell.Color {
	ar, ag, ab := a.TrueColor().RGB()
	br, bg, bb := b.TrueColor().RGB()
	return tcell.NewRGBColor(
		ar+int32(float64(br-ar)*t),
		ag+int32(float64(bg-ag)*t),
		ab+int32(float64(bb-ab)*t),
	)
}

// PixelBuf is a framebuffer with two vertical "pixels" per terminal cell.
// Row pairs are emitted as '▀' with fg = top pixel and bg = bottom pixel.
type PixelBuf struct {
	W, H int // pixel dimensions; H is 2x the cell rows it will occupy
	pix  []tcell.Color
}

// NewPixelBuf allocates a buffer covering cols x rows terminal cells.
func NewPixelBuf(cols, rows int) *PixelBuf {
	return &PixelBuf{W: cols, H: rows * 2, pix: make([]tcell.Color, cols*rows*2)}
}

// Resize reallocates if the cell dimensions changed.
func (p *PixelBuf) Resize(cols, rows int) {
	if p.W == cols && p.H == rows*2 {
		return
	}
	p.W, p.H = cols, rows*2
	p.pix = make([]tcell.Color, cols*rows*2)
}

// Set writes one pixel; out-of-bounds writes are ignored.
func (p *PixelBuf) Set(x, y int, c tcell.Color) {
	if x < 0 || y < 0 || x >= p.W || y >= p.H {
		return
	}
	p.pix[y*p.W+x] = c
}

// Get reads one pixel (black when out of bounds).
func (p *PixelBuf) Get(x, y int) tcell.Color {
	if x < 0 || y < 0 || x >= p.W || y >= p.H {
		return tcell.ColorBlack
	}
	return p.pix[y*p.W+x]
}

// Fill sets every pixel to c.
func (p *PixelBuf) Fill(c tcell.Color) {
	for i := range p.pix {
		p.pix[i] = c
	}
}

// VLine draws a vertical pixel run [y0, y1) at column x.
func (p *PixelBuf) VLine(x, y0, y1 int, c tcell.Color) {
	for y := y0; y < y1; y++ {
		p.Set(x, y, c)
	}
}

// Blit writes the buffer to the screen with its top-left at cell (ox, oy).
func (p *PixelBuf) Blit(s *Screen, ox, oy int) {
	for cy := 0; cy < p.H/2; cy++ {
		row0 := cy * 2 * p.W
		row1 := row0 + p.W
		for cx := 0; cx < p.W; cx++ {
			s.SetCell(ox+cx, oy+cy, '▀', p.pix[row0+cx], p.pix[row1+cx])
		}
	}
}
