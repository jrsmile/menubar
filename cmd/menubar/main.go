// Command menubar is a menu-bar terminal multiplexer: a single visible pane
// with a clickable top menu for creating, switching, and closing panes.
package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"

	"menubar/internal/mux"
)

func main() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "menubar:", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "menubar:", err)
		os.Exit(1)
	}
	// Only button presses are needed; skipping motion/drag reporting keeps
	// PuTTY's Shift+drag native selection clean and avoids stray menu toggles.
	screen.EnableMouse(tcell.MouseButtonEvents)

	app := mux.New(screen, shell)
	app.Run()

	screen.DisableMouse()
	screen.Fini()
}
