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

	// Ask the terminal to grow to 256x64 cells (xterm window-resize escape;
	// honored by Windows Terminal and most modern emulators, ignored
	// harmlessly elsewhere). Sent before tcell takes over the screen.
	fmt.Print("\x1b[8;64;256t")
	time.Sleep(150 * time.Millisecond) // give the emulator a beat to apply it

	scr, err := render.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not open terminal:", err)
		os.Exit(1)
	}
	defer scr.Fini()

	game.New(scr, *seed).Run()
}
