// DungeonGame: a terminal adventure with an Ultima-style overworld and a
// raycast first-person dungeon mode.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"dungeongame/internal/game"
	"dungeongame/internal/render"
)

func main() {
	seed := flag.Int64("seed", 0, "world seed (0 = random)")
	flag.Parse()
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	// Deliberately send no terminal-resize escapes. Windows Terminal accepts
	// a CSI 8 request for a grid wider than the monitor, silently clips the
	// right edge, and keeps that oversized grid stuck to the tab until the
	// window is resized by hand - which made everything (title screen,
	// missiles, crosshair) render off-center while looking fine to the
	// program. The game adapts live to whatever size the window really is;
	// maximize the terminal for the biggest view.

	scr, err := render.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not open terminal:", err)
		os.Exit(1)
	}
	defer scr.Fini()

	game.New(scr, *seed).Run()
}
